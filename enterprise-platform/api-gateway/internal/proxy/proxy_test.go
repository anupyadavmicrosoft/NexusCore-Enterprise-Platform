package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nexuscore/api-gateway/internal/config"
	"github.com/nexuscore/api-gateway/internal/middleware"
)

func TestCircuitBreaker_AllowAndRecord(t *testing.T) {
	cb := NewCircuitBreaker("test-service", 0.5, 100*time.Millisecond)

	// In closed state, should allow
	if !cb.Allow() {
		t.Fatal("expected closed breaker to allow requests")
	}

	// Trigger failures to trip the breaker
	for i := 0; i < 15; i++ {
		cb.RecordResult(false)
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected breaker to be open, got state %v", cb.State())
	}

	if cb.Allow() {
		t.Fatal("expected open breaker to deny requests")
	}

	// Wait for cooldown
	time.Sleep(150 * time.Millisecond)

	if !cb.Allow() {
		t.Fatal("expected breaker to transition to half-open and allow request")
	}

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected state to be half-open, got %v", cb.State())
	}

	// Fail in half-open state should immediately reopen
	cb.RecordResult(false)
	if cb.State() != StateOpen {
		t.Fatalf("expected failure in half-open state to trip breaker back to open, got %v", cb.State())
	}
}

func TestRateLimiter_Enforcement(t *testing.T) {
	rl := NewRateLimiter(10.0, 3) // 10 tokens/sec, max 3 burst

	// Consume 3 burst tokens
	for i := 0; i < 3; i++ {
		allowed, _, _ := rl.Allow("client-1")
		if !allowed {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}

	// 4th request should be rate-limited
	allowed, _, _ := rl.Allow("client-1")
	if allowed {
		t.Fatal("expected 4th request to be denied by rate limiter")
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)
	allowed, _, _ = rl.Allow("client-1")
	if !allowed {
		t.Fatal("expected rate limiter to refill and allow request after wait")
	}
}

func TestGatewayCache(t *testing.T) {
	cache := NewGatewayCache()
	key := "GET:/api/v1/data"

	_, hit := cache.Get(key)
	if hit {
		t.Fatal("expected cache miss for un-cached key")
	}

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	entry := CacheEntry{
		Status:    200,
		Body:      []byte(`{"msg":"cached"}`),
		Headers:   headers,
		ExpiresAt: time.Now().Add(100 * time.Millisecond),
	}

	cache.Set(key, entry)

	cachedEntry, hit := cache.Get(key)
	if !hit {
		t.Fatal("expected cache hit after setting entry")
	}
	if string(cachedEntry.Body) != `{"msg":"cached"}` {
		t.Fatalf("unexpected cached payload: %s", string(cachedEntry.Body))
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)
	_, hit = cache.Get(key)
	if hit {
		t.Fatal("expected cache miss after TTL expired")
	}
}

func TestRoundRobinLoadBalancing(t *testing.T) {
	cfg := config.LoadConfig()
	cache := NewGatewayCache()

	targets := []string{"http://10.0.0.1:8080", "http://10.0.0.2:8080", "http://10.0.0.3:8080"}
	rp, err := NewReverseProxy("LoadBalancedService", targets, cfg, cache)
	if err != nil {
		t.Fatalf("failed to create reverse proxy: %v", err)
	}

	expectedPaths := []string{
		"http://10.0.0.2:8080",
		"http://10.0.0.3:8080",
		"http://10.0.0.1:8080",
	}

	var mu sync.Mutex
	var actualPaths []string

	for i := 0; i < 3; i++ {
		tgt := rp.NextTarget()
		mu.Lock()
		actualPaths = append(actualPaths, tgt.String())
		mu.Unlock()
	}

	for i, path := range expectedPaths {
		if actualPaths[i] != path {
			t.Fatalf("expected index %d to map to %s, got %s", i, path, actualPaths[i])
		}
	}
}

func TestRBACMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/protected", func(c *gin.Context) {
		// simulate role setting
		c.Set("role", "Guest")
		c.Next()
	}, middleware.RBACMiddleware("Admin", "Operator"), func(c *gin.Context) {
		c.String(http.StatusOK, "authorized")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected HTTP 403 Forbidden for Guest role on Admin/Operator route, got %d", w.Code)
	}
}

func TestAuthMiddleware_MockOIDC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := config.LoadConfig()

	r.GET("/api", middleware.AuthMiddleware(cfg), func(c *gin.Context) {
		user, _ := c.Get("user")
		role, _ := c.Get("role")
		c.JSON(http.StatusOK, gin.H{"user": user, "role": role})
	})

	// 1. Unauthenticated -> Guest
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/api", nil)
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w1.Code)
	}

	// 2. Mock OIDC token -> Operator/Admin
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api", nil)
	req2.Header.Set("Authorization", "Bearer mock_oidc_admin_token")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w2.Code)
	}
}

func TestProxy_PathRewrite(t *testing.T) {
	// Start a local test server to act as a mock microservice backend
	serverCalled := false
	var receivedURL *url.URL

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		receivedURL = r.URL
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("backend-success"))
	}))
	defer ts.Close()

	cfg := config.LoadConfig()
	cfg.MaxRetryAttempts = 0
	cfg.RequestTimeout = 1 * time.Second
	cache := NewGatewayCache()

	rp, err := NewReverseProxy("TestProxyRewrite", []string{ts.URL}, cfg, cache)
	if err != nil {
		t.Fatalf("failed to build reverse proxy: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Handle and rewrite: strip prefix "/api/v1/compute" and add "/transformed"
	r.Any("/api/v1/compute/*any", rp.Handle("/api/v1/compute", "/transformed"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/compute/transactions?id=45", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	if !serverCalled {
		t.Fatal("expected mock backend server to be called")
	}

	if receivedURL == nil {
		t.Fatal("expected received URL to be recorded")
	}

	expectedPath := "/transformed/transactions"
	if receivedURL.Path != expectedPath {
		t.Fatalf("expected rewritten path to be %s, got %s", expectedPath, receivedURL.Path)
	}

	if receivedURL.RawQuery != "id=45" {
		t.Fatalf("expected query param to persist, got %s", receivedURL.RawQuery)
	}
}

package proxy

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nexuscore/api-gateway/internal/config"
	"github.com/nexuscore/api-gateway/internal/metrics"
)

type CacheEntry struct {
	Status    int
	Body      []byte
	Headers   http.Header
	ExpiresAt time.Time
}

type GatewayCache struct {
	mu    sync.RWMutex
	store map[string]CacheEntry
}

func NewGatewayCache() *GatewayCache {
	return &GatewayCache{store: make(map[string]CacheEntry)}
}

func (gc *GatewayCache) Get(key string) (CacheEntry, bool) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	entry, found := gc.store[key]
	if !found {
		return CacheEntry{}, false
	}
	if time.Now().After(entry.ExpiresAt) {
		return CacheEntry{}, false
	}
	return entry, true
}

func (gc *GatewayCache) Set(key string, entry CacheEntry) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.store[key] = entry
}

type ReverseProxy struct {
	serviceName     string
	targets         []*url.URL
	index           uint64
	circuitBreaker  *CircuitBreaker
	client          *http.Client
	config          *config.Config
	cache           *GatewayCache
}

func NewReverseProxy(serviceName string, targetURLs []string, cfg *config.Config, cache *GatewayCache) (*ReverseProxy, error) {
	var parsedTargets []*url.URL
	for _, raw := range targetURLs {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, err
		}
		parsedTargets = append(parsedTargets, u)
	}

	cb := NewCircuitBreaker(serviceName, cfg.CircuitFailureRate, cfg.CircuitCooldown)

	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
	}

	return &ReverseProxy{
		serviceName:    serviceName,
		targets:        parsedTargets,
		circuitBreaker: cb,
		client:         client,
		config:         cfg,
		cache:          cache,
	}, nil
}

// NextTarget picks a target using Round-Robin load balancing
func (rp *ReverseProxy) NextTarget() *url.URL {
	n := atomic.AddUint64(&rp.index, 1)
	return rp.targets[n%uint64(len(rp.targets))]
}

func (rp *ReverseProxy) Handle(stripPrefix string, replacePrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Caching Layer Check (Only for safe idempotent GET operations)
		cacheKey := c.Request.Method + ":" + c.Request.URL.Path + "?" + c.Request.URL.RawQuery
		if rp.config.EnableCaching && c.Request.Method == http.MethodGet {
			if entry, hit := rp.cache.Get(cacheKey); hit {
				// Cache HIT
				for k, vals := range entry.Headers {
					for _, v := range vals {
						c.Writer.Header().Add(k, v)
					}
				}
				c.Writer.Header().Set("X-Gateway-Cache", "HIT")
				c.Writer.WriteHeader(entry.Status)
				_, _ = c.Writer.Write(entry.Body)
				c.Abort()
				return
			}
		}

		// 2. Circuit Breaker Enforcement
		if !rp.circuitBreaker.Allow() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":        "The target microservice is currently unavailable due to system overload.",
				"code":         "CIRCUIT_BREAKER_OPEN",
				"service_name": rp.serviceName,
			})
			c.Abort()
			return
		}

		// 3. Round-Robin Node Selection
		target := rp.NextTarget()

		// 4. Transform Path / URI Rewrite
		newPath := c.Request.URL.Path
		if stripPrefix != "" {
			if len(newPath) >= len(stripPrefix) && newPath[:len(stripPrefix)] == stripPrefix {
				newPath = newPath[len(stripPrefix):]
			}
		}
		if replacePrefix != "" {
			if len(newPath) == 0 || newPath[0] != '/' {
				newPath = replacePrefix + "/" + newPath
			} else {
				newPath = replacePrefix + newPath
			}
		}
		if newPath == "" {
			newPath = "/"
		}

		// 5. Create Request and Inject Trace Headers
		reqURL := *target
		reqURL.Path = newPath
		reqURL.RawQuery = c.Request.URL.RawQuery

		// Read original body
		var bodyBytes []byte
		if c.Request.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(c.Request.Body)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body", "code": "MALFORMED_BODY"})
				c.Abort()
				return
			}
		}

		var lastErr error
		var resp *http.Response
		backoff := 50 * time.Millisecond

		// 6. Resilient Transport Dispatch Loop (Retries + Timeout)
		for attempt := 0; attempt <= rp.config.MaxRetryAttempts; attempt++ {
			reqCtx, cancelCtx := context.WithTimeout(c.Request.Context(), rp.config.RequestTimeout)
			defer cancelCtx()

			outReq, err := http.NewRequestWithContext(reqCtx, c.Request.Method, reqURL.String(), bytes.NewReader(bodyBytes))
			if err != nil {
				lastErr = err
				break
			}

			// Header copy and enhancement (Transformation)
			for k, vv := range c.Request.Header {
				outReq.Header[k] = vv
			}
			outReq.Header.Set("X-Forwarded-For", c.ClientIP())
			outReq.Header.Set("X-Gateway-Processed-By", "NexusCore-Gateway-1.0")
			if correlation, exists := c.Get("correlationID"); exists {
				outReq.Header.Set("X-Correlation-ID", correlation.(string))
			}

			resp, err = rp.client.Do(outReq)
			if err == nil && resp.StatusCode < 500 {
				// Succeeded (or returned normal client error, circuit breaker treats 4xx as healthy API response)
				lastErr = nil
				break
			}

			// If we failed with network issue or >= 500 server error, count and retry
			lastErr = err
			if err != nil {
				metrics.BackendFailureCount.WithLabelValues(rp.serviceName, "network_error").Inc()
			} else {
				metrics.BackendFailureCount.WithLabelValues(rp.serviceName, "status_5xx").Inc()
				resp.Body.Close() // close stale response body
			}

			if attempt < rp.config.MaxRetryAttempts {
				slog.Warn("Downstream service request failed, initiating retry back-off", "attempt", attempt+1, "service", rp.serviceName, "error", err)
				time.Sleep(backoff)
				backoff *= 2
			}
		}

		// 7. Record Circuit Breaker Result
		if lastErr != nil {
			rp.circuitBreaker.RecordResult(false)
			c.JSON(http.StatusBadGateway, gin.H{
				"error":        "Service request failed after successive delivery attempts.",
				"code":         "DOWNSTREAM_DISPATCH_FAULT",
				"service_name": rp.serviceName,
			})
			c.Abort()
			return
		}

		rp.circuitBreaker.RecordResult(true)
		defer resp.Body.Close()

		// Read response body
		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read response from downstream microservice", "code": "MALFORMED_DOWNSTREAM_RESPONSE"})
			c.Abort()
			return
		}

		// Write headers back
		for k, vv := range resp.Header {
			for _, v := range vv {
				c.Writer.Header().Add(k, v)
			}
		}
		c.Writer.Header().Set("X-Gateway-Cache", "MISS")
		c.Writer.WriteHeader(resp.StatusCode)
		_, _ = c.Writer.Write(respBytes)

		// 8. Cache response if enabled and candidate is safe
		if rp.config.EnableCaching && c.Request.Method == http.MethodGet && resp.StatusCode == http.StatusOK {
			copiedHeaders := make(http.Header)
			for k, vv := range resp.Header {
				copiedHeaders[k] = vv
			}
			rp.cache.Set(cacheKey, CacheEntry{
				Status:    resp.StatusCode,
				Body:      respBytes,
				Headers:   copiedHeaders,
				ExpiresAt: time.Now().Add(rp.config.CacheTTL),
			})
		}
	}
}

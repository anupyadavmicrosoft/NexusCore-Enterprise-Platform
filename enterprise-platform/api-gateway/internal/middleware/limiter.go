package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nexuscore/api-gateway/internal/config"
	"github.com/nexuscore/api-gateway/internal/metrics"
)

type ClientBucket struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu           sync.Mutex
	buckets      map[string]*ClientBucket
	refillRate   float64 // tokens per second
	burstCapacity float64
}

func NewRateLimiter(refillRate float64, burstCapacity int) *RateLimiter {
	return &RateLimiter{
		buckets:       make(map[string]*ClientBucket),
		refillRate:    refillRate,
		burstCapacity: float64(burstCapacity),
	}
}

func (rl *RateLimiter) Allow(clientKey string) (bool, int, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[clientKey]
	now := time.Now()

	if !exists {
		bucket = &ClientBucket{
			tokens:     rl.burstCapacity,
			lastRefill: now,
		}
		rl.buckets[clientKey] = bucket
	}

	// Calculate token replenishment
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * rl.refillRate
	if bucket.tokens > rl.burstCapacity {
		bucket.tokens = rl.burstCapacity
	}
	bucket.lastRefill = now

	// Calculate retry after time
	var retryAfter time.Duration
	if bucket.tokens < 1.0 {
		requiredTokens := 1.0 - bucket.tokens
		retryAfter = time.Duration(requiredTokens/rl.refillRate*float64(time.Second))
		metrics.RateLimitTrippedTotal.WithLabelValues(clientKey, "rate_limit").Inc()
		return false, 0, retryAfter
	}

	bucket.tokens -= 1.0
	return true, int(bucket.tokens), 0
}

func RateLimitMiddleware(cfg *config.Config) gin.HandlerFunc {
	limiter := NewRateLimiter(cfg.RateLimitRefill, cfg.RateLimitBurst)

	// Clean up stale client buckets in the background
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			limiter.mu.Lock()
			now := time.Now()
			for k, v := range limiter.buckets {
				if now.Sub(v.lastRefill) > 10*time.Minute {
					delete(limiter.buckets, k)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		// Identify client key (API Key role, or client IP as fallback)
		clientKey := c.ClientIP()
		if key, exists := c.Get("apiKey"); exists {
			clientKey = key.(string)
		} else if user, exists := c.Get("user"); exists {
			clientKey = user.(string)
		}

		allowed, remaining, retryAfter := limiter.Allow(clientKey)

		// Set response rate limit headers
		c.Writer.Header().Set("X-RateLimit-Limit", getIntString(int(limiter.burstCapacity)))
		c.Writer.Header().Set("X-RateLimit-Remaining", getIntString(remaining))
		c.Writer.Header().Set("X-RateLimit-Reset", "1")

		if !allowed {
			c.Writer.Header().Set("Retry-After", getIntString(int(retryAfter.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":          "API rate limit exceeded. Please apply exponential back-off.",
				"code":           "RATE_LIMIT_EXCEEDED",
				"retry_after_s":  retryAfter.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func getIntString(n int) string {
	if n <= 0 {
		return "0"
	}
	switch n {
	case 1:
		return "1"
	case 2:
		return "2"
	case 5:
		return "5"
	default:
		return strconv.Itoa(n)
	}
}

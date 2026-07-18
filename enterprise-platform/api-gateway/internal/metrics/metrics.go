package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_http_requests_total",
			Help: "Total number of HTTP requests processed by NexusCore API Gateway",
		},
		[]string{"path", "method", "status_code"},
	)

	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_http_request_duration_seconds",
			Help:    "Latency histogram of HTTP requests routed through the gateway",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"path", "method", "status_code"},
	)

	RateLimitTrippedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_rate_limit_tripped_total",
			Help: "Total number of requests dropped due to rate limiting or quota exhaustion",
		},
		[]string{"client_ip", "quota_type"},
	)

	CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_circuit_breaker_state",
			Help: "Current state of the backend service circuit breakers (0=Closed, 1=Half-Open, 2=Open)",
		},
		[]string{"service_name"},
	)

	BackendFailureCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_backend_failure_total",
			Help: "Total count of network retry trigger events to downstream nodes",
		},
		[]string{"service_name", "error_type"},
	)
)

func init() {
	prometheus.MustRegister(HttpRequestsTotal)
	prometheus.MustRegister(HttpRequestDuration)
	prometheus.MustRegister(RateLimitTrippedTotal)
	prometheus.MustRegister(CircuitBreakerState)
	prometheus.MustRegister(BackendFailureCount)
}

// Gin Middleware to record Prometheus Metrics
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		c.Next()

		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		HttpRequestsTotal.WithLabelValues(path, method, statusCode).Inc()
		HttpRequestDuration.WithLabelValues(path, method, statusCode).Observe(duration)
	}
}

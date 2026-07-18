package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/nexuscore/api-gateway/internal/config"
	"github.com/nexuscore/api-gateway/internal/docs"
	"github.com/nexuscore/api-gateway/internal/metrics"
	"github.com/nexuscore/api-gateway/internal/middleware"
	"github.com/nexuscore/api-gateway/internal/proxy"
)

func main() {
	// 1. Load System Configuration
	cfg := config.LoadConfig()

	// 2. Initialize Structured JSON Logger with appropriate level
	var programLevel slog.Level
	switch cfg.LogLevel {
	case "DEBUG":
		programLevel = slog.LevelDebug
	case "WARN":
		programLevel = slog.LevelWarn
	case "ERROR":
		programLevel = slog.LevelError
	default:
		programLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel}))
	slog.SetDefault(logger)

	logger.Info("Starting NexusCore Enterprise API Gateway (Sprint 11)...",
		"port", cfg.Port,
		"log_level", cfg.LogLevel,
		"env", "production",
	)

	// 3. Configure OpenTelemetry Tracer Provider
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error("Failed to gracefully shutdown OpenTelemetry tracer provider", "error", err)
		}
	}()

	// 4. Set up router configuration
	gin.SetMode(gin.ReleaseMode)
	// Create a new gin router without Default middlewares to avoid duplicates and slow logger formats
	r := gin.New()

	// 5. Global Middlewares
	r.Use(gin.Recovery())
	r.Use(gzip.Gzip(gzip.DefaultCompression)) // Compression Policy
	r.Use(metrics.PrometheusMiddleware())      // Metrics Collection
	r.Use(middleware.LoggingAndTracingMiddleware()) // Tracing & Structured Logging

	// 6. Prometheus Scraper Metrics Route
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 7. Swagger Documentation Portal
	docs.RegisterSwaggerPortal(r)

	// 8. Health and Readiness Endpoints
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "HEALTHY",
			"service":   "api-gateway",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Global Cache Store for proxy reads
	globalCache := proxy.NewGatewayCache()

	// Create Auth Service Proxy
	authProxy, err := proxy.NewReverseProxy("AuthService", cfg.AuthServiceURLs, cfg, globalCache)
	if err != nil {
		logger.Error("Initialization fault on Auth Service reverse proxy", "error", err)
		os.Exit(1)
	}

	// Create Compute Service Proxy
	computeProxy, err := proxy.NewReverseProxy("ComputeService", cfg.ComputeServiceURLs, cfg, globalCache)
	if err != nil {
		logger.Error("Initialization fault on Compute Service reverse proxy", "error", err)
		os.Exit(1)
	}

	// Readiness endpoint checks downstream dependencies
	r.GET("/readyz", func(c *gin.Context) {
		// Verify backends status (check closed vs open breaker states)
		if authProxy.State() == proxy.StateOpen {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "UNREADY", "reason": "AuthService circuit breaker is open"})
			return
		}
		if computeProxy.State() == proxy.StateOpen {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "UNREADY", "reason": "ComputeService circuit breaker is open"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "READY"})
	})

	// 9. Core Dispatch Routing Proxies
	// Unprotected Auth routes
	publicAuth := r.Group("/api/v1/auth")
	publicAuth.Use(middleware.RateLimitMiddleware(cfg))
	{
		publicAuth.Any("/*any", authProxy.Handle("", ""))
	}

	// Rate Limiting and Authenticated Group
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg))
	protected.Use(middleware.RateLimitMiddleware(cfg))

	// Protected routes matching Admin or Operator clearance
	rbacRoutes := protected.Group("/")
	rbacRoutes.Use(middleware.RBACMiddleware("Admin", "Operator"))
	{
		rbacRoutes.Any("/api/v1/compute/*any", computeProxy.Handle("", ""))
		rbacRoutes.Any("/transactions/*any", computeProxy.Handle("", ""))
	}

	// Define Server Configuration
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 10. Start Server with Graceful Shutdown Controller
	shutdownChan := make(chan error, 1)
	go func() {
		logger.Info("NexusCore Gateway is fully active and accepting requests", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownChan <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-shutdownChan:
		logger.Error("Ingress listener failed, starting emergency shutdown", "error", err)
	case sig := <-quit:
		logger.Info("System received exit code termination command", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Graceful listener cancellation triggered failures", "error", err)
		os.Exit(1)
	}

	logger.Info("NexusCore API Gateway shut down clean. Process finished.")
}

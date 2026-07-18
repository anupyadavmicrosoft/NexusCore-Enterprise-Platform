package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// Initialize Structured Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	logger.Info("Starting NexusCore API Gateway...", "port", "8080", "version", "1.0.0")

	// Setup OpenTelemetry Tracer Provider
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error("Error shutting down tracer provider", "error", err)
		}
	}()

	// Get target microservice URLs from Environment
	authServiceURLStr := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURLStr == "" {
		authServiceURLStr = "http://localhost:8081"
	}
	computeServiceURLStr := os.Getenv("COMPUTE_SERVICE_URL")
	if computeServiceURLStr == "" {
		computeServiceURLStr = "http://localhost:8082"
	}

	authServiceURL, err := url.Parse(authServiceURLStr)
	if err != nil {
		logger.Error("Malformed AUTH_SERVICE_URL", "url", authServiceURLStr, "error", err)
		os.Exit(1)
	}

	computeServiceURL, err := url.Parse(computeServiceURLStr)
	if err != nil {
		logger.Error("Malformed COMPUTE_SERVICE_URL", "url", computeServiceURLStr, "error", err)
		os.Exit(1)
	}

	// Dynamic Reverse Proxies
	authProxy := httputil.NewSingleHostReverseProxy(authServiceURL)
	computeProxy := httputil.NewSingleHostReverseProxy(computeServiceURL)

	mux := http.NewServeMux()

	// 1. Health Probe Endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"HEALTHY","service":"api-gateway","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
	})

	// 2. Gateway Proxy routing handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path

		// Rate Limiting simulation/logging
		logger.Info("Incoming request received by Gateway Ingress", "method", r.Method, "path", path, "ip", r.RemoteAddr)

		// Routing resolution rules
		if strings.HasPrefix(path, "/api/v1/auth") {
			logger.Debug("Routing to Auth Service...", "target", authServiceURLStr)
			authProxy.ServeHTTP(w, r)
		} else if strings.HasPrefix(path, "/transactions") || strings.HasPrefix(path, "/api/v1/compute") {
			logger.Debug("Routing to Compute & Transaction Engine...", "target", computeServiceURLStr)
			computeProxy.ServeHTTP(w, r)
		} else {
			logger.Warn("Unmatched gateway route request path", "path", path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"requested route does not exist in gateway namespace","code":404}`))
		}

		logger.Info("Ingress gateway resolved dispatch pipeline", "path", path, "duration_ms", time.Since(start).Milliseconds())
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Server Graceful Shutdown Controller
	shutdownChan := make(chan error, 1)
	go func() {
		logger.Info("API Gateway fully operational, listening at TCP :8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownChan <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-shutdownChan:
		logger.Error("Ingress listener failed, initiating shutdown", "error", err)
	case sig := <-quit:
		logger.Info("Platform received OS terminal termination signal", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Gateway shutdown finished with core execution faults", "error", err)
		os.Exit(1)
	}

	logger.Info("API Gateway clean container shutdown finalized. Exiting.")
}

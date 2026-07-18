package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpDel "github.com/nexuscore/compute-engine/internal/delivery/http"
	"github.com/nexuscore/compute-engine/internal/repository"
	"github.com/nexuscore/compute-engine/internal/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// Initialize Structured Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	logger.Info("Starting Compute & Transaction Engine node...", "version", "1.0.0")

	// Setup OpenTelemetry Tracing
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	// Wire high-performance event-driven components
	eventStore := repository.NewMemoryEventStore()
	schemaRegistry := repository.NewMemorySchemaRegistry()
	kafkaCluster := repository.NewMemoryKafkaCluster(schemaRegistry)
	queryStore := usecase.NewMemoryQueryStore()

	service := usecase.NewComputeService(eventStore, kafkaCluster, queryStore)
	handler := httpDel.NewHttpComputeHandler(service)

	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"UP","service":"compute-engine"}`))
	}))
	mux.Handle("/", handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	shutdownChan := make(chan error, 1)
	go func() {
		logger.Info("Compute Engine running and accepting HTTP transport commands", "port", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownChan <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-shutdownChan:
		logger.Error("Compute Engine thread crashed", "error", err)
	case sig := <-quit:
		logger.Info("Received stop command, graceful shutdown active", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.Shutdown(ctx)
	logger.Info("Compute engine thread successfully shut down.")
}

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

	"github.com/gin-gonic/gin"
	grpcDelivery "github.com/nexuscore/auth-service/internal/delivery/grpc"
	httpDelivery "github.com/nexuscore/auth-service/internal/delivery/http"
	"github.com/nexuscore/auth-service/internal/repository"
	"github.com/nexuscore/auth-service/internal/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func initTracer() *sdktrace.TracerProvider {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("nexuscore-identity-service"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp
}

func main() {
	// 1. Initialize Standard structured JSON logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting NexusCore Identity Platform Service...")

	// 2. Initialize OpenTelemetry Distributed Tracing Provider
	tp := initTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Error("Failed to gracefully shutdown OpenTelemetry Provider", "error", err)
		}
	}()

	// 3. Read Environments Configs with strict fallbacks
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8081"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	kafkaBrokers := []string{os.Getenv("KAFKA_BROKERS")}
	if len(kafkaBrokers) == 0 || kafkaBrokers[0] == "" {
		kafkaBrokers = []string{"kafka-cluster-kafka-bootstrap.platform.svc:9092"}
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "nexuscore_master_cryptographic_signing_key_secret_2026"
	}

	// 4. Initialize Repository Database Adapters
	postgresDb := repository.NewPostgresRepository()
	redisCache := repository.NewRedisCacheRepository()
	kafkaBroker := repository.NewKafkaPublisher(kafkaBrokers)

	// 5. Initialize Identity Logic Orchestrator Usecase
	identityUsecase := usecase.NewIdentityUsecase(postgresDb, redisCache, kafkaBroker, jwtSecret)

	// 6. Initialize Delivery Channels (gRPC & HTTP)
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Attach logger middleware
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("HTTP Request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", time.Since(start).String(),
			"client_ip", c.ClientIP(),
		)
	})

	// Register REST Handlers
	httpHandler := httpDelivery.NewGinIdentityHandler(identityUsecase)
	httpHandler.RegisterRoutes(router)

	// Register and Boot internal gRPC Core Server
	grpcServer := grpcDelivery.NewGrpcIdentityServer(identityUsecase)
	err := grpcServer.StartGrpcServer("0.0.0.0:" + grpcPort)
	if err != nil {
		slog.Error("CRITICAL: Failed to bind gRPC server, terminating...", "error", err)
		os.Exit(1)
	}

	// Start Gin Server asynchronously
	srv := &http.Server{
		Addr:    "0.0.0.0:" + httpPort,
		Handler: router,
	}

	go func() {
		slog.Info("Gin HTTP Multiplexer Server online and serving REST API", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("REST API Server crashed unexpectedly", "error", err)
		}
	}()

	// 7. Establish Signal Trap for graceful microservice teardowns
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Teardown signal caught. Shutting down servers gracefully...")

	// Define shutdown timeouts
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("REST HTTP Server forced to shutdown", "error", err)
	}

	slog.Info("NexusCore Identity Platform Service successfully offlined.")
}

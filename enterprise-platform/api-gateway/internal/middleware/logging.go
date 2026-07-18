package middleware

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func LoggingAndTracingMiddleware() gin.HandlerFunc {
	tracer := otel.Tracer("nexuscore-api-gateway")

	return func(c *gin.Context) {
		start := time.Now()

		// Generate or extract Correlation ID
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = generateRandomID()
		}
		c.Writer.Header().Set("X-Correlation-ID", correlationID)
		c.Set("correlationID", correlationID)

		// Start OpenTelemetry Span
		ctx, span := tracer.Start(c.Request.Context(), fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.correlation_id", correlationID),
				attribute.String("client.ip", c.ClientIP()),
			),
		)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		// Set request size validation check
		if c.Request.ContentLength > 10*1024*1024 { // 10MB Limit
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body exceeds maximum operational allowance of 10MB",
				"code":  "PAYLOAD_TOO_LARGE",
			})
			c.Abort()
			return
		}

		// Buffer request body for inspection if log level is debug/verbose
		var requestBody []byte
		if c.Request.Body != nil {
			var err error
			requestBody, err = io.ReadAll(c.Request.Body)
			if err == nil {
				// Restore original body reader so downstream handlers can parse it
				c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}
		}

		// Intercept response body using custom writer
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Dispatch request pipeline
		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Log trace status
		span.SetAttributes(attribute.Int("http.status_code", statusCode))
		if statusCode >= 500 {
			span.SetStatus(trace.StatusError, "Internal Server Failure")
		} else {
			span.SetStatus(trace.StatusOK, "Processed")
		}

		// Structure logging values
		logger := slog.With(
			slog.String("correlation_id", correlationID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", statusCode),
			slog.Int64("latency_ms", latency.Milliseconds()),
			slog.Int("bytes_written", c.Writer.Size()),
			slog.String("client_ip", c.ClientIP()),
		)

		if statusCode >= 400 {
			respSnippet := blw.body.String()
			if len(respSnippet) > 1000 {
				respSnippet = respSnippet[:1000] + "... [truncated]"
			}
			logger.Error("API Gateway returned failure status to client",
				"response_body", respSnippet,
				"request_payload_size", len(requestBody),
			)
		} else {
			logger.Info("Ingress gateway resolved dispatch pipeline")
		}
	}
}

func generateRandomID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "static-correlation-id-fallback-0000"
	}
	return hex.EncodeToString(bytes)
}

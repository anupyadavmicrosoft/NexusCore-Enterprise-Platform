# ==============================================================================
# STAGE 1: Compilation Builder Environment
# ==============================================================================
FROM golang:1.22-alpine AS builder

WORKDIR /workspace

# Copy global shared modules and dependencies
COPY shared-jwt-library/ /workspace/shared-jwt-library/
COPY shared-oauth-library/ /workspace/shared-oauth-library/
COPY shared-security-library/ /workspace/shared-security-library/
COPY identity-service/ /workspace/identity-service/

WORKDIR /workspace/identity-service

# Force static linkage, disabling external CGO bindings for maximal execution speed
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -ldflags="-s -w" -o /bin/identity-service ./cmd/server/main.go

# ==============================================================================
# STAGE 2: Distroless Minimal Security Runtime
# ==============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS final

# Strict nonroot container bindings
USER 65532:65532

COPY --from=builder /bin/identity-service /app/identity-service

# Expose microservice standard network ingress interfaces
EXPOSE 8080

ENTRYPOINT ["/app/identity-service"]

# Compile stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o compute-engine cmd/server/main.go

# Multi-stage secure static debian execution context
FROM gcr.io/distroless/static-debian12:latest

COPY --from=builder /app/compute-engine /compute-engine

EXPOSE 8082
USER nonroot:nonroot

ENTRYPOINT ["/compute-engine"]

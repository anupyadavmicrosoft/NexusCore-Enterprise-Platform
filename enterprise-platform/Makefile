# Makefile for NexusCore Enterprise Go Workspace
# Handles multi-service Go module targets and local infrastructure orchestration

.PHONY: all build test clean run up down restart logs status bootstrap lint

# Default target
all: build test

# Build all Go binaries locally
build:
	@echo "==> Building api-gateway..."
	cd api-gateway && go build -v -o ./bin/api-gateway cmd/server/main.go
	@echo "==> Building auth-service..."
	cd auth-service && go build -v -o ./bin/auth-service cmd/server/main.go
	@echo "==> Building compute-engine..."
	cd compute-engine && go build -v -o ./bin/compute-engine cmd/server/main.go
	@echo "==> All builds completed successfully."

# Execute unit and integration tests across the workspace
test:
	@echo "==> Running unit tests for Go Workspace..."
	go test -v ./api-gateway/... ./auth-service/... ./compute-engine/...

# Formats all Go code inside the workspace
fmt:
	@echo "==> Formatting source code..."
	go fmt ./api-gateway/... ./auth-service/... ./compute-engine/...

# Lint all services using golangci-lint (if installed)
lint:
	@echo "==> Linting Go Workspace..."
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint is not installed. Skipping."; \
	fi

# Clean binary builds and temporary cache files
clean:
	@echo "==> Cleaning built assets..."
	rm -rf api-gateway/bin auth-service/bin compute-engine/bin
	go clean -cache -testcache

# Launch the complete Docker Compose workspace (infrastructure + microservices)
up:
	@echo "==> Spinning up local enterprise cluster..."
	docker compose up -d --build

# Bring down the compose services and purge anonymous database volumes
down:
	@echo "==> Stopping local enterprise cluster and clearing state..."
	docker compose down -v

# Restart all services in the compose setup
restart: down up

# Show logs from Docker Compose containers
logs:
	docker compose logs -f

# View running container status
status:
	docker compose ps

# Bootstrap script (seeds database, verifies configs, runs custom checks)
bootstrap:
	@echo "==> Bootstrapping workspace configurations..."
	chmod +x ./scripts/bootstrap.sh
	./scripts/bootstrap.sh

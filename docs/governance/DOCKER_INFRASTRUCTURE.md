# NexusCore Docker Infrastructure Governance Specification

## 1. Executive Mandate & Scope

This document establishes the official **NexusCore Docker and Containerization Infrastructure Standards**. 

Containers form the core execution runtime of our microservices across development, staging, and production Kubernetes clusters. To maintain absolute security, performance, and deterministic operational behavior, all container images must adhere to strict, automated standard structures.

All Dockerfiles, docker-compose configurations, and container-related automation templates must comply with the guidelines defined herein.

---

## 2. Image Architecture & Classification

To balance performance during continuous integration (CI) compile times with extreme security postures in production, NexusCore defines four distinct container image classes:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       CLASS 1: BUILDER IMAGES (CI/CD)                       │
│  - Heavy environment: Contains compilers, libraries, Git, and packaging.    │
│  - Base: alpine:latest or golang:1.22-alpine.                               │
│  - Never deployed to production environments.                               │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    CLASS 2: PRODUCTION RUNTIME IMAGES (Distroless)          │
│  - Ultra-lightweight: Contains only compiled static binaries.               │
│  - Base: gcr.io/distroless/static-debian12:latest.                          │
│  - No shell, no package manager, no root capability.                        │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      CLASS 3: DEVELOPMENT CONTAINER IMAGES                   │
│  - Feature-rich: Includes diagnostic tools (gdb, curl, netcat), shells.     │
│  - Base: mcr.microsoft.com/devcontainers/go:1-1.22-bookworm.               │
│  - Configured strictly for local debugging, VS Code, and sandboxing.       │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Multi-Stage Builds Specification

All microservices written in compiled languages (Go, C++) must use **Multi-Stage Builds** to completely separate the build-time environment from the runtime environment.

### 3.1 Design Template (Go Example)
The following template must be used as the mandatory baseline for all Go microservices:

```dockerfile
# Stage 1: Build & Compilation (Class 1 Builder Image)
FROM golang:1.22-alpine AS builder

# Enforce secure compiler constraints
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

# Cache dependency layers efficiently
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree and compile optimized static binary
COPY . .
RUN go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o service-binary \
    cmd/server/main.go

# Stage 2: Hardened Secure Execution (Class 2 Runtime Image)
FROM gcr.io/distroless/static-debian12:latest

# Copy statically linked binary from Builder stage
COPY --from=builder /app/service-binary /service-binary

# Expose target application port
EXPOSE 8080

# Enforce nonroot execution context (UID 65532)
USER nonroot:nonroot

# Execute application binary directly without shell interpreter wrapper
ENTRYPOINT ["/service-binary"]
```

---

## 4. Image Optimization and Size Mitigation

To ensure high-speed deployments and scaling capabilities, container images must undergo extreme size optimization:

1.  **Linker Flag Pruning (`-ldflags="-s -w"`)**:
    *   **`-s`**: Removes all symbol tables and debug references from the compiled binary.
    *   **`-w`**: Removes DWARF debugging information.
    *   *Outcome*: Reduces static Go binary footprints by **30% to 45%** with zero impact on runtime performance.
2.  **Layer Minimization**:
    *   Consolidate related RUN instructions inside builder stages to minimize file-system layers.
    *   Utilize multi-stage patterns to ensure build-time tools (like `apk`, compilers, git) never leak into final runtime layers.
3.  **Caching Optimization**:
    *   Place less volatile instructions (such as copying and downloading `go.mod` dependencies) *before* volatile instructions (such as copying the active code repository) to leverage the Docker cache daemon.

---

## 5. Security Hardening & Zero-Trust Policies

Production container execution contexts must operate under strict **least-privilege, zero-trust policies**:

### 5.1 Distroless Runtime Baseline
*   The default base image for production containers is **Google's Distroless static image (`gcr.io/distroless/static-debian12`)**.
*   Distroless images contain *only* the application and its runtime dependencies. They lack shells (`sh`, `bash`), package managers (`apt`, `apk`), and standard GNU core utilities, reducing the container's attack surface area.

### 5.2 Non-Root Execution (`nonroot:nonroot`)
*   **Absolutely no container is allowed to execute as `root` (UID 0).**
*   All Dockerfiles must declare a non-privileged user context:
    ```dockerfile
    USER nonroot:nonroot
    ```
*   Kubernetes Security Contexts must enforce `runAsNonRoot: true` and `allowPrivilegeEscalation: false`.

### 5.3 Read-Only Container Filesystems
*   Final runtime filesystems should be mounted as read-only. Microservices must never write to their local container file-system directly.
*   For ephemeral operations (such as compiling local logs or caches), configure explicit, localized `tmpfs` mounts or memory volumes.

---

## 6. Networking & Volume Segregation Architectures

Docker Compose configurations must segregate operations to prevent lateral network traversal by an attacker in the event of a container compromise.

### 6.1 Network Segregation (Bridges)
NexusCore defines isolated network bridges:
*   `frontend-network`: Bridges external public edge routers and proxies to the API Gateway.
*   `internal-network`: Connects the API Gateway, Auth Service, and Compute Engine microservices.
*   `data-network`: Connects back-end microservices directly to PostgreSQL, Redis, and Kafka. The edge API Gateway must never belong to the `data-network`.

```
                    [ External Public Client ]
                                │
                                ▼
                       { frontend-network }
                                │
                                ▼
                        [ API Gateway ]
                                │
                                ▼
                       { internal-network }
                                │
             ┌──────────────────┴──────────────────┐
             ▼                                     ▼
      [ Auth Service ]                     [ Compute Engine ]
             │                                     │
             └──────────────────┬──────────────────┘
                                │
                                ▼
                        { data-network }
                                │
         ┌──────────────────────┼──────────────────────┐
         ▼                      ▼                      ▼
    [ Database ]             [ Cache ]              [ Broker ]
```

### 6.2 Volume Protocols
*   All persistent container storage (such as database directories or cache filesystems) must use **named volumes** (e.g., `pgdata`, `redisdata`).
*   Host bind mounts (e.g., `./data:/var/lib/data`) are strictly prohibited in production and staging environments because they can expose the host file system to the container.

---

## 7. Container Health Auditing Standards

To support automated healing, progressive rollouts, and high availability, every active container in our configurations must define a robust, deterministic **Health Check**.

### 7.1 Database & Cache Health Checks
*   **PostgreSQL**:
    ```yaml
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5
    ```
*   **Redis**:
    ```yaml
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "redis_secure_pass_77", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5
    ```

### 7.2 Microservice Liveness & Readiness Checks
Microservices must expose dedicated `/health` endpoints on independent HTTP handlers.
*   **Liveness Check**: Returns `200 OK` if the process is active.
*   **Readiness Check**: Returns `200 OK` *only* if downstream connections (PostgreSQL, Kafka) are verified active. If any dependency is offline, returns `503 Service Unavailable`.

---

## 8. Continuous Vulnerability Scanning (Image Scanning)

To proactively identify and remediate security issues, container images must undergo continuous scanning in the deployment pipeline.

### 8.1 Automated CI Scan Gate
Our CI/CD pipelines run **Trivy Container Scans** on built images before they are pushed to Google Artifact Registry:
*   The build step is flagged as a failure if Trivy detects any **HIGH** or **CRITICAL** severity vulnerabilities with an available patch.
*   Weekly cron checks scan active production containers to intercept newly discovered CVEs.

# Enterprise Developer Guide (NexusCore Workspace)

This document provides onboarding steps, coding guidelines, testing requirements, and debugging instructions for engineers working within the Go workspace.

## 1. Local Environment Onboarding Setup

To begin active development, ensure your local workstation satisfies these requirements:

### 1.1 Prerequisites Installation
```bash
# MacOS Installation
brew install go docker docker-compose make golangci-lint
```

### 1.2 Initialize Workspace Environment
```bash
# Clone the repository
git clone https://github.com/enterprise/nexus-core.git
cd nexus-core/enterprise-platform

# Verify multi-module workspace recognition
go work init ./api-gateway ./auth-service ./compute-engine

# Pull infrastructure dependencies (PostgreSQL, Kafka, Redis)
make up
```

---

## 2. Multi-Module Go Project Structure

The codebase is organized as a unified Go Multi-Module Workspace, facilitating strict modular isolation while easing local imports:

```
. (enterprise-platform root)
├── go.work                 # Multi-module workspace coordinator
├── Makefile                # Automation commands list
├── api-gateway/            # Isolated Go module
│   ├── go.mod
│   └── main.go
├── auth-service/           # Isolated Go module
│   ├── go.mod
│   └── main.go
└── compute-engine/         # Isolated Go module
    ├── go.mod
    └── main.go
```

---

## 3. Strict Coding Standards

Engineers must follow these syntactic guidelines to ensure maintainability:

*   **Explicit Error Handling**: Do not ignore returned `error` properties. Wrap error chains with informative context using `%w`:
    ```go
    if err != nil {
        return nil, fmt.Errorf("failed to verify hmac key validation parameters: %w", err)
    }
    ```
*   **Struct Tags**: Ensure serialization formats are declared explicitly:
    ```go
    type TokenClaims struct {
        Subject   string `json:"subject" db:"subject_id"`
        Role      string `json:"role" db:"auth_role"`
        ExpiresAt int64  `json:"expires_at" db:"exp_timestamp"`
    }
    ```
*   **Linting Constraints**: Code must pass `golangci-lint` without exceptions. Configured linters include `gofmt`, `govet`, `errcheck`, `staticcheck`, and `gosec`.

---

## 4. Testing & Coverage Requirements

Every PR submitted to the mainline branch must adhere to the **Continuous Quality Thresholds**:
*   **Statement Coverage**: Minimal code line test coverage of **95%** across core security and financial libraries.
*   **Unit Tests**: Created alongside logic inside `*_test.go` files, using standard Go test commands:
    ```bash
    go test -v -cover ./...
    ```
*   **Integration Tests**: Execute integration scenarios targeting multi-module endpoints using docker-compose:
    ```bash
    make test-integration
    ```

---

## 5. Local Debugging & Scaffolding a New Microservice

### 5.1 Real-Time API Logs Inspection
Use `docker logs` to stream real-time JSON log outputs from individual services during development:
```bash
docker logs -f compute-engine
```

### 5.2 Scaffolding a New Module (e.g., `analytics-service`)
Follow this structural pattern when creating a new microservice in the workspace:

1. Create a new directory at the root:
    ```bash
    mkdir analytics-service
    cd analytics-service
    ```
2. Initialize Go Module:
    ```bash
    go mod init github.com/enterprise/nexuscore/analytics-service
    ```
3. Create main entrypoint file `main.go`.
4. Register the new module into the root Go workspace coordinator:
    ```bash
    cd ../
    go work use ./analytics-service
    ```
5. Update `docker-compose.yml` to include the container orchestrator configuration.

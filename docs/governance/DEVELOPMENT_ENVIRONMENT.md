# NexusCore Development Environment Manual

This document provides complete instructions for bootstrapping, executing, debugging, and testing the NexusCore Enterprise Platform workspace. It is fully certified for use on **Linux, macOS (Intel & Apple Silicon), and Windows (WSL2 / Git Bash)**.

---

## 1. Local Workstation Prerequisite Matrix

Before starting, guarantee your system has the following binary components installed:

| Component | Minimum Version | Installation / Management Command |
| :--- | :--- | :--- |
| **Go** | 1.22+ | `brew install go` (macOS), `sudo apt install golang` (Ubuntu/Debian) |
| **Docker Engine** | 24.0+ | [Docker Desktop](https://www.docker.com/products/docker-desktop/) |
| **kubectl** | 1.28+ | `brew install kubernetes-cli` |
| **Helm** | 3.12+ | `brew install helm` |
| **golangci-lint** | 1.57+ | `go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@v1.57.2` |

---

## 2. Dev Containers (Standardized Sandboxing)

To isolate dependencies and eliminate "works on my machine" issues, we supply a fully featured **VS Code Dev Container** setup.

### How to use:
1.  Ensure you have **VS Code** and the **Remote - Containers** extension installed.
2.  Open the repository directory in VS Code.
3.  A notification will prompt: `"Folder contains a Dev Container configuration file. Reopen in Container"`.
4.  Click **Reopen in Container**.
5.  VS Code will compile the container image based on `.devcontainer/Dockerfile`, mapping in Go tools, Kubernetes tools (`kind`, `kubectl`, `helm`), databases, caches, and utilities automatically.
6.  Upon loading, `scripts/bootstrap.sh` executes automatically, preparing the multi-module workspace.

---

## 3. VS Code Workspace Customization

Our workspace settings (`.vscode/`) configure language servers, auto-formatting, key diagnostics, and advanced microservice debugging.

### settings.json (Formating & Vetting File Watchers)
Our settings guarantee standard formatting (`goimports` + `gofumpt`) and run workspace-wide `golangci-lint` analyses in the background on every file save:
*   Automatically organizes and imports package headers.
*   Excludes high-write and temporary directories (`/node_modules`, `/dist`, `/tmp`) from active VS Code file watchers, reducing CPU loads.

### launch.json (Service-Level & Compound Debuggers)
We provide native Go launch profiles for our individual services, as well as a **compound configuration** to debug the complete platform concurrently:
*   **Debug: API Gateway**: Launches edge service binding on port `8080` targeting local dependencies.
*   **Debug: Auth Service**: Launches identity microservice on port `8081` pointing to database pools.
*   **Debug: Compute Engine**: Launches compute service on port `8082` listening to Kafka cluster endpoints.
*   **Debug All Services (Compound)**: Spawns all three debugging runtimes inside VS Code's Call Stack console simultaneously, routing stdout logs to separate Debug consoles.

---

## 4. Docker-Compose Sandbox Execution

To execute the platform locally with high-performance containers, leverage the workspace `docker-compose.yml` file.

### Spin up dependencies only:
This runs PostgreSQL, Redis, Kafka, ZooKeeper, Jaeger, and Prometheus in the background, allowing you to debug your Go binaries locally on your host OS:
```bash
make deps
```

### Spin up the entire platform (with services):
This compiles the microservice Dockerfiles and launches the entire network:
```bash
docker-compose up --build
```

### Stop all processes and wipe temporary data:
```bash
make down
```

---

## 5. Local Kubernetes (Kind / Minikube Setup)

For advanced operators who want to test helm layouts, horizontal autoscaling, and network policies locally, we provide a bootstrap script:
`/enterprise-platform/scripts/local-k8s-setup.sh`.

### Features:
*   **Cross-Platform support**: Automatically detects and leverages `kind` or `minikube` depending on host capabilities.
*   **Network Ingress**: Maps host port `8080` to the cluster ingress interface so API traffic can flow directly.
*   **Local Secrets**: Standardizes configuration keys and JWT tokens inside the cluster.
*   **Database & Cache Subcharts**: Automatically provisions Bitnami `postgresql` and `redis` instances with precise connection credentials.
*   **Image Sideloading**: Compiles local Docker images and loads them into the cluster without requiring a public container registry.

### Execution:
```bash
bash scripts/local-k8s-setup.sh
```

---

## 6. Pre-Commit Verification Hook

To protect the shared repository against syntax faults, styling regressions, or test failures, we enforce an ironclad **Pre-Commit Hook**.

The hook checks three key gates:
1.  **Format Gate**: Audits Go files via `gofmt -l`. If files are unformatted, the commit is safely blocked.
2.  **Lint Gate**: Vets files using `golangci-lint` (or `go vet` fallback).
3.  **Test Gate**: Executes all unit tests with the Go race-detector (`go test -race -short`).

### Manual Hook Installation:
Symlink the pre-commit script into your Git configuration folder:
```bash
ln -sf ../../scripts/pre-commit-hook.sh .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```
On Windows platforms utilizing Git Bash or WSL2, copying the script directly to `.git/hooks/pre-commit` achieves identical outcomes.

---

## 7. Dynamic Hot-Reloading & File Watchers

To expedite active development, we recommend using `air` for hot-reloading Go binaries in the background:

### Installation:
```bash
go install github.com/air-verse/air@latest
```

### Configuration (air.toml):
Each microservice module can be run with an `air.toml` template to watch files, rebuild upon saves, and dynamically hot-swap binary execution in under 2 seconds.
For example, to execute `air` inside `auth-service/`:
```bash
cd auth-service
air
```
This is fully configured inside our workspace environment.

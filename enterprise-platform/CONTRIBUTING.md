# Contributing to NexusCore Enterprise Platform

First off, thank you for taking the time to contribute! This document defines the development workflows, codebase structure, coding guidelines, and validation rules for the NexusCore monorepo workspace.

Adhering to these guidelines ensures that the system remains robust, performant, secure, and easy to maintain across multiple distributed service components.

---

## 1. Monorepo Structure & Go Workspaces

NexusCore is structured as a **Go Multi-Module Workspace** (Go 1.22+). It separates services into individual folders, each containing its own `go.mod` file, allowing isolated dependency management while coordinating local developments with `go.work`.

```
.
├── api-gateway/            # Go Module: Ingress API Gateway Service
├── auth-service/           # Go Module: Zero-Trust Identity Platform (AuthN/AuthZ)
├── compute-engine/         # Go Module: High-performance Transaction & Compute Engine
├── config/                 # Shared Static Configurations & Telemetry Layouts
├── scripts/                # Setup, Database Seeding, & Migration Shell Scripts
├── go.work                 # Root Go Workspace File (DO NOT commit temporary changes)
└── Makefile                # Root Build Automation Controller
```

### Module Guidelines:
*   Never introduce circular dependencies between modules.
*   Keep internal packages inside `/internal` subdirectories of each module to prevent unauthorized external imports.
*   Publish shared libraries or models meant for cross-module usage within a distinct package or designated module to maintain clean boundaries.

---

## 2. Coding Standards & Best Practices

All backend development must adhere to standard Go idiomatic programming principles and performance targets:

### 2.1 Code Layout & Formatting
*   Always format files utilizing `go fmt` or `goimports`.
*   Maintain clear naming conventions: variables and functions use `camelCase`, exported types and functions use `PascalCase`.
*   Avoid naked returns in non-trivial functions.

### 2.2 Error Handling
*   Handle errors explicitly. Do not ignore errors with `_` unless there is an documented, deliberate reason.
*   Wrap errors with contextual details when bubble-up propagation is necessary (e.g., `fmt.Errorf("retrieving user: %w", err)`).
*   Avoid panics in production logic. Panic recovery should only be a final safety net in root-level routing or daemon executors.

### 2.3 Performance & Allocation Optimization
*   Pre-allocate slices and maps using `make(T, capacity)` whenever the target size is known beforehand to reduce Garbage Collection overhead.
*   Prefer struct pass-by-pointer (`*MyStruct`) for large data payloads to avoid copying structures across call stacks, but use pass-by-value for immutable primitives.

---

## 3. Testing Requirements & Quality Gates

Quality is a non-negotiable metric at NexusCore. All incoming Pull Requests must meet the following gates:

*   **Core Logic Coverage**: Core components (including Cryptography, Identity, Ledger, Security packages) must maintain **>= 95% unit test statement coverage**.
*   **General Logic Coverage**: General routing and auxiliary utility packages must maintain **>= 80% unit test statement coverage**.
*   **No Flaky Tests**: Tests must execute deterministically. Use proper mock providers or synchronized context clocks rather than arbitrary sleep windows.

To execute the test suite locally:
```bash
make test
```

---

## 4. Git Branching & Pull Request Workflow

We utilize a structured Git Branching model aligned with a bi-weekly release train:

1.  **Branch Name Conventions**:
    *   Features: `feature/username/description-jira-id`
    *   Bug Fixes: `bugfix/username/description-jira-id`
    *   Hotfixes: `hotfix/username/description-jira-id`
2.  **Pull Request Guidelines**:
    *   Target the `main` branch.
    *   Ensure all commits are squash-merged to maintain a clean git history.
    *   Every PR must reference a specific backlog item or tracking issue.
    *   CI workflows (linting, compiling, testing) must pass cleanly before merge authorization.
    *   Requires approval from at least one **Principal Architect** or designated **Service Owner**.
3.  **Semantic Versioning**:
    *   Releases must be tagged strictly using SemVer (`vMAJOR.MINOR.PATCH`).

---

## 5. Automated Linters

We enforce uniform syntax checks via `golangci-lint`. Make sure your code is compliant before committing:
```bash
make lint
```

Refer to the `.golangci.yml` file for rules, which include standard static analysis, security vetting (`gosec`), code complexity checks, and dependency audits.

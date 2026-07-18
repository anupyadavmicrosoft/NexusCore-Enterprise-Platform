# NexusCore Repository Governance

## 1. Executive Summary

This document establishes the mandatory **NexusCore Repository Governance and Structural Standards**.

As our systems evolve under a distributed architecture, we must enforce strict organization, clear service boundaries, explicit dependency controls, and deterministic release flows. This standard guarantees that our repository scales efficiently across dozens of autonomous engineering squads without code duplication, circular dependencies, or architectural drift.

These standards are defined, audited, and co-enforced by the **Principal Enterprise Architect**, the **Principal Golang Architect**, and the **Senior Platform Engineers**.

---

## 2. Directory Structure and Repository Layout

NexusCore utilizes a standardized monorepo layout (with strict encapsulation) or standard modular polyrepo templates. The workspace directory layout is specified below:

```
/ (Repository Root)
├── .github/                       # Central CI/CD workflow configurations
│   └── workflows/                 # Automated compilation, testing, and security gates
├── charts/                        # Kubernetes Helm manifests and deployment configurations
│   └── helm/
│       ├── api-gateway/           # API Gateway deployment configuration
│       ├── auth-service/          # Authentication microservice configurations
│       └── compute-engine/        # Compute workloads and horizontal scaling profiles
├── docs/                          # Architecture documentation and governance specifications
│   ├── adr/                       # Historical Architecture Decision Records (0001-XXXX)
│   ├── governance/                # Regulatory handbook, standards, and compliance gates
│   └── runbook/                   # Incident management instructions and on-call guides
├── enterprise-platform/           # Main backend domain services directory
│   ├── api-gateway/               # Ingress gateway proxy (Go/Gin)
│   ├── auth-service/              # Cryptographic identity validation service (Go)
│   ├── compute-engine/            # Transaction processors and math engines (Go)
│   └── go.work                    # Multi-module Go workspace file (Go 1.22+)
├── src/                           # Central portal front-end dashboard (React/Vite)
│   ├── components/                # Modular, targetable, stateful component modules
│   ├── types.ts                   # Unified front-end type declarations and enums
│   └── App.tsx                    # Ingress router and container frame
├── .env.example                   # Reference environment configuration template
├── package.json                   # Front-end configuration and script managers
└── metadata.json                  # Application metadata and runtime configurations
```

---

## 3. Codebase Ownership & Service Boundaries

To promote autonomous execution while maintaining clear accountability, every subdirectory within the repository must have a designated owner.

### 3.1 Folder Ownership Matrix

| Folder Path | Primary Domain Owner | Secondary Domain Owner | Authorization Scope |
| :--- | :--- | :--- | :--- |
| `/enterprise-platform/api-gateway/` | Principal Golang Architect | Senior Platform Engineers | Ingress, rate limiting, routing proxy, cache layer. |
| `/enterprise-platform/auth-service/` | Chief Information Security Officer (CISO) | Senior Security Engineers | Identity, JWT signing, password validation, API key generation. |
| `/enterprise-platform/compute-engine/` | Principal Enterprise Architect | Senior Backend Engineers | Mathematical engines, ledger balance audits, calculation workers. |
| `/charts/` | Principal DevOps Architect | Senior Kubernetes Engineers | Helm parameters, resource quotas, container limits. |
| `/src/` | Senior Frontend Engineers | Product Managers | User interface, state management, visualization. |
| `/docs/` | Senior Documentation Engineers | Domain Technical Leads | System runbooks, ADRs, compliance handbooks, release logs. |

### 3.2 Service Boundaries
*   **Data Encapsulation**: A microservice has absolute, exclusive ownership of its datastores (PostgreSQL databases, Redis instances, Kafka topic writers). Direct cross-database queries or writes from external microservices are strictly prohibited.
*   **Interaction Limits**: Interaction between microservices must occur through official, typed interface endpoints (gRPC/Protobuf) or asynchronous broker logs (Kafka). No internal logic components or non-exported files may be directly imported across boundary services.

---

## 4. Shared Libraries & Dependency Rules

Shared code speeds up development but introduces coupling risks. We enforce strict criteria to prevent shared packages from becoming a dumping ground for technical debt.

### 4.1 Shared Library Criteria
Shared libraries must consist **strictly** of non-business-specific, generic helper logic:
1.  *Cryptographic Wrappers*: AES-256-GCM functions, Argon2id password hashers, JWT parsers.
2.  *Middleware Frames*: Unified logging encoders, rate limit interfaces, OpenTelemetry tracers.
3.  *Utility Handlers*: Standard calculation formulas, slice filter tools, string transformations.

### 4.2 Dependency Rules
*   **Unbound Imports Prohibited**: Microservices are prohibited from importing code blocks or local files from other active microservice directories (e.g., `auth-service` must never import a file located inside `/enterprise-platform/compute-engine/`).
*   **Third-Party Library Minimization**: To prevent supply-chain attacks and binary size bloat, any addition of a third-party package must be evaluated against standard criteria:
    1.  *Is it maintained?* Must show active commits within the last 6 months.
    2.  *Is it secure?* Must contain zero known High or Critical CVE disclosures.
    3.  *Is it light?* The library should perform one function with minimal transitive dependency overhead.
*   **Version Pinning**: All dependencies in `package.json` and `go.mod` must be pinned to **exact, deterministic version numbers**. Caret (`^`) and tilde (`~`) symbols are strictly prohibited in product package sheets to prevent compile-time drift.

---

## 5. Import and Package Structure

```
                  ┌─────────────────────────────────────────┐
                  │              main package               │
                  └────────────────────┬────────────────────┘
                                       │ (bootstrapping only)
                                       ▼
                  ┌─────────────────────────────────────────┐
                  │             internal/config             │
                  └────────────────────┬────────────────────┘
                                       │
         ┌─────────────────────────────┴─────────────────────────────┐
         ▼                                                           ▼
┌──────────────────┐                                        ┌──────────────────┐
│internal/middleware│                                        │  internal/proxy  │
└────────┬─────────┘                                        └────────┬─────────┘
         │                                                           │
         └─────────────────────────────┬─────────────────────────────┘
                                       │
                                       ▼
                            ┌─────────────────────┐
                            │  internal/metrics   │
                            └─────────────────────┘
```

### 5.1 Go Package Guidelines
*   **cmd/**: Reserved for entrypoint files that bootstrap and launch the service. No business or routing logic is permitted in this directory. The `main.go` file must compile into a single executable binary.
*   **internal/**: Contains code specific to this microservice. Go's runtime prevents other modules from importing files within an `internal/` directory, naturally enforcing architectural encapsulation.
*   **pkg/**: Reserved for utilities and helpers that are safe for external modules to import (e.g., auto-generated Protobuf client files).

### 5.2 Circular Dependency Prevention
*   **No Circular Imports**: Package `A` cannot import package `B` if `B` also imports `A` (directly or transitively). This is enforced by `golangci-lint` and the Go compiler itself.
*   **Resolution Strategy**: If circular dependencies occur, engineers must extract the common shared interfaces or variables into a separate, lower-level package (e.g., `internal/types`), allowing both packages to import it without direct cycle paths.

---

## 6. Code Generation Standards

To minimize manual boilerplate and maximize consistency, all serialization frameworks, service APIs, and test doubles must utilize automated generation tools.

### 6.1 Generated Code Protection
*   **Read-Only Annotation**: All generated source files must begin with a clear, standard comment warning developers that manual changes will be lost:
    ```go
    // Code generated by protoc-gen-go. DO NOT EDIT.
    ```
*   **Manual Edits Prohibited**: Under no circumstances may an engineer manually edit a generated code file. Any changes to the API structure must be made in the source contract file (e.g., `.proto`, `.yaml`) and compiled again.

### 6.2 Generation Toolchain
*   **gRPC/Protobuf Compilation**: Protobuf compilation is managed via the **Buf CLI**. No manual raw `protoc` execution commands are allowed.
*   **Mock Generation**: Mock objects must be automatically compiled using standard tools (such as `mockery` for Go or `ts-jest` templates).

---

## 7. API Versioning Standards

Client and server-to-server endpoints must evolve smoothly without breaking active users.

### 7.1 REST URI Versioning
*   REST endpoints must embed the API version directly into the path prefix:
    *   *Format*: `/api/v[MAJOR]/[domain_context]/[resource]`
    *   *Example*: `/api/v1/auth/login`, `/api/v2/compute/transactions`
*   **Header Versioning Fallback**: For micro-variations where paths must persist, clients may request a specific schema via headers:
    ```http
    Accept: application/vnd.nexuscore.v1.1+json
    ```

### 7.2 Protobuf Package Isolation
*   gRPC packages must maintain separate folders for every version iteration to prevent naming conflicts and runtime compilation failures:
    ```protobuf
    package nexuscore.compute.v1;
    ```

---

## 8. Database Migration Standards

To ensure zero-downtime production deployments, database migrations are strictly regulated.

### 8.1 Migration Script Design Rules
*   **Two-Way Scripts**: Every schema change must contain both an `.up.sql` (to apply modifications) and a `.down.sql` (to revert modifications) script file.
*   **Idempotency**: Migrations must be written defensively to ensure they can be executed multiple times without throwing errors:
    ```sql
    CREATE TABLE IF NOT EXISTS users (...);
    ALTER TABLE transactions ADD COLUMN IF NOT EXISTS audit_id VARCHAR(36);
    ```
*   **No Locking Schema Upgrades**: Migrations that lock tables (e.g., adding a default value to an existing massive column without null protection) must be executed progressively to preserve service availability.

---

## 9. Deprecation and Compatibility Policies

As the platform evolves, older code blocks, endpoints, packages, and variables must be retired safely.

### 9.1 The 3-Sprint Deprecation Policy
Decommissioning an API endpoint or package requires three distinct sprints:

```
┌──────────────────────────────────────────────┐
│ SPRINT N: Mark Deprecated                    │
│ - Add @deprecated annotations to contracts.  │
│ - Log warning with trace correlations.       │
└──────────────────────┬───────────────────────┘
                       ▼
┌──────────────────────────────────────────────┐
│ SPRINT N+1: Active Migration                 │
│ - Migrate all internal clients to new API.  │
│ - Block new users via API Gateway.           │
└──────────────────────┬───────────────────────┘
                       ▼
┌──────────────────────────────────────────────┐
│ SPRINT N+2: Hard Decommission                │
│ - Safely delete source code and structures.  │
│ - Release clean binary versions.             │
└──────────────────────────────────────────────┘
```

1.  **Sprint N (Deprecation Notice)**:
    *   Add `@deprecated` JSDoc annotations to front-end APIs.
    *   Add Go comment deprecation tags to fields: `// Deprecated: use NewMethod instead.`
    *   The API Gateway must return a standard **`Sunset`** or **`Deprecation`** HTTP response header on affected routes.
2.  **Sprint N+1 (Internal Migration)**:
    *   All internal, automated, and federated applications must rewrite active calls to target the new versions.
    *   Examine trace logs to verify that zero production calls still query the deprecated endpoints.
3.  **Sprint N+2 (Hard Decommission)**:
    *   Safely delete the source files, routes, tables, and variables from the active codebase.

### 9.2 Compatibility Policy
All schema structures, API parameters, and message formats must remain backward-compatible with the active, preceding version of the microservice. 
*   **No Breaking Changes**: Breaking changes are strictly prohibited in minor (`MINOR`) releases.
*   **Major Revisions**: If a breaking change is mandatory, a new `MAJOR` version of the API or package must be introduced, allowing older services to continue running concurrently during the migration period.

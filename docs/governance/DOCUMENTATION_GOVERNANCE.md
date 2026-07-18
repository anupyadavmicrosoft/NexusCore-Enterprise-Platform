# NexusCore Documentation Governance Specification

## 1. Executive Mandate & Scope

This document establishes the official **NexusCore Documentation Governance and Standards Framework**. 

To maintain the operational integrity of our enterprise systems, documentation must be treated with the same engineering rigor as code. Outdated, inaccurate, or unstructured documentation represents an operational vulnerability, increases onboarding friction, and extends recovery windows during live incidents. 

All documentation within the NexusCore ecosystem must be version-controlled, automatically validated, and reviewed in lockstep with codebase mutations. This specification establishes the standards for ten critical documentation classes:

1.  **README Standards**
2.  **API Documentation Standards**
3.  **Architecture Documentation Standards**
4.  **Runbooks**
5.  **Operations Manuals**
6.  **Incident Response Guides**
7.  **Deployment Guides**
8.  **Troubleshooting Guides**
9.  **Developer Guides**
10. **Contributor Guides**

---

## 2. Documentation Architecture

We organize our documentation into three separate layers based on audience, volatility, and location:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            LAYER 1: SERVICE-LEVEL                           │
│  Located in service subdirectories. Highly volatile.                        │
│  Examples: README.md, OpenAPI JSON, Service Protobuf, Local Runbooks        │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            LAYER 2: REPOSITORY-LEVEL                        │
│  Located in /docs/ or repository root. Medium volatility.                   │
│  Examples: Developer Guides, Troubleshooting, ADRs, Deployment Manuals      │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            LAYER 3: ORGANIZATIONAL                          │
│  Located in central wikis or core governance paths. Low volatility.          │
│  Examples: Coding Standards, SDLC Policies, Security Standards, CLA         │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Class 1: README Standards

Every microservice and distinct directory within the NexusCore repository must feature a standard, comprehensive `README.md` file.

### 3.1 Standard Structure Template
Every service README must follow this exact Markdown structure:
*   **Header Section**: Title of the service, followed by standardized badges indicating build status, code coverage, linter health, and active SemVer release.
*   **Overview**: A concise (1-2 paragraph) business and technical explanation of the service's purpose. What does it solve?
*   **Quick Start**: Step-by-step commands required to run the service locally (including dependency setup).
*   **Architecture & Design**: Brief overview of the data model, dependencies (databases, brokers), and structural patterns used.
*   **Exposed APIs**: List of primary HTTP, gRPC, or async subscription topics.
*   **Configuration**: Table of all required environment variables, default values, and description strings.
*   **Runbook Link**: Direct relative link to the corresponding service runbook.

### 3.2 Badging Requirements
Standard badges must be rendered dynamically via build pipeline badges. No hardcoded status badges are allowed:
```markdown
![Build Status](https://github.com/nexuscore/platform/actions/workflows/api-gateway-ci.yml/badge.svg)
![Coverage](https://img.shields.io/badge/coverage-95%25-green.svg)
![SemVer](https://img.shields.io/badge/semver-1.0.0-blue.svg)
```

---

## 4. Class 2: API Documentation Standards

API contracts are the integration boundaries between teams. They must remain accurate, complete, and machine-readable.

### 4.1 REST API Contracts (OpenAPI v3)
*   All public REST endpoints must be defined via standard **OpenAPI 3.0.3** spec sheets under `/docs/openapi.json`.
*   Every property, parameter, and response payload must declare:
    *   `type` (e.g., `string`, `integer`, `boolean`).
    *   `required` status.
    *   Valid values or constraints (e.g., minimum size, formatting expressions).
    *   Example payloads representing both successful and unsuccessful states.
*   **Route Synclink**: OpenAPI specifications must be compiled dynamically in non-production environments to serve an interactive Swagger UI portal under `/swagger`.

### 4.2 gRPC / Protobuf API Contracts
*   Internal backend APIs must be defined using strict **Protocol Buffers v3** syntax (.proto files).
*   **Comment Block Requirements**: Every gRPC message field and service RPC definition must feature inline documentation comments:
    ```protobuf
    // CreateSession issues a secure cryptographic bearer token.
    rpc CreateSession (SessionRequest) returns (SessionResponse) {
        // ...
    }
    ```
*   **Buf Linter Integration**: Any protobuf changes must pass compatibility tests to prevent breaking clients during rolling upgrades.

---

## 5. Class 3: Architecture Documentation Standards (ADR)

Significant system changes, structural transformations, database selections, or design patterns must be preceded by an **Architecture Decision Record (ADR)**.

### 5.1 ADR File Conventions
*   ADR files are written in Markdown and checked in under `/docs/adr/`.
*   File names must follow the index pattern: `[INDEX]-[short-lowercase-description].md` (e.g., `/docs/adr/0011-enterprise-api-gateway.md`).

### 5.2 ADR Markdown Template
```markdown
# ADR [INDEX]: [TITLE]

## Status
[Draft | Proposed | Approved | Superceded]

## Context
[Describe the business challenge, technical context, and alternative options considered.]

## Decision
[Detail the exact technical selection, library, or architectural pattern chosen. Use active voice "We decided to...".]

## Consequences
### Positive (Outcomes)
*   [Benefit 1]
*   [Benefit 2]

### Negative (Trade-offs / Technical Debt)
*   [Cost 1]
*   [Cost 2]
```

---

## 6. Class 4: Runbooks (Incident Mitigation)

Runbooks are operational checklists designed for on-call engineers to diagnose and mitigate service failures rapidly.

### 6.1 Runbook Structuring Requirements
*   Every microservice must feature a matching runbook file under `/docs/runbook/[service-name]-runbook.md`.
*   **Metrics-to-Action Mapping**: Runbooks must contain direct action steps tied to Prometheus alerts.
*   **Structure Template**:
    *   **Service Overview**: Brief, operational summary of the system components.
    *   **Alert Escalation Path**: On-call pager targets and fallback leads.
    *   **Diagnostics Section**: Exact curl commands, SQL queries, or log filters to inspect current cluster health.
    *   **Common Failure Scenarios**:
        *   *Scenario A: High DB Connection Consumption*: Step-by-step instructions to scale read replicas or kill locking queries.
        *   *Scenario B: Circuit Breaker Open*: Steps to override, increase timeouts, or redirect calls to alternate availability zones.
        *   *Scenario C: Memory Leak / OOM*: Steps to capture heap profiles via `pprof` and perform a safe rolling restart.

---

## 7. Class 5: Operations Manuals (Routine Tasks)

Operations Manuals provide instructions for non-incident, routine systems maintenance.

### 7.1 Manual Contents
*   **Data Backup & Restore**: Exact commands and verification steps for database dump and restore cycles.
*   **Resource Scaling Rules**: Configuration procedures for modifying Horizontal Pod Autoscaler (HPA) parameters in Helm templates.
*   **Data Purging & Retention**: Schedules and maintenance routines for archiving or purging expired transactions, logs, and historical indices.
*   **Credential Rotation**: Step-by-step instructions to rotate API keys, SSL certificates, and database master passwords with zero downtime.

---

## 8. Class 6: Incident Response Guides (IRG)

The Incident Response Guide governs organizational collaboration, role-assignments, and communication channels during SEV-1 and SEV-2 critical platform incidents.

### 8.1 Incident Command Roles
*   **Incident Commander (IC)**: Holds absolute authority over triage, routing, and hotfix approval during the incident. Coordinates tasks and delegates technical debugging.
*   **Communications Lead (CL)**: Responsible for drafting client-facing status page updates, internal executive notifications, and support team briefings.
*   **Lead Engineers / SREs**: Technical solvers focused on diagnostics, rollbacks, or hotfixes.

### 8.2 Live Incident War Room Procedure
```
                        [ Incident Detected ]
                                  │
                                  ▼
                     [ PagerDuty Alert Triggered ]
                                  │
                                  ▼
                [ IC Launches War Room & Bridge Line ]
                                  │
         ┌────────────────────────┴────────────────────────┐
         ▼                                                 ▼
[ Solvers: Triage & Mitigate ]             [ CL: Status Page & Updates ]
         │                                                 │
         └────────────────────────┬────────────────────────┘
                                  ▼
                    [ SLA Latency / Error Restored ]
                                  │
                                  ▼
                 [ Post-Incident Postmortem Session ]
```

1.  **War Room Activation**: The IC opens a dedicated conference bridge and invites all active domain solvers.
2.  **Mitigation Focus**: The priority is **restoration of service (reducing latency/errors)**, not investigating the core bug. If possible, execute an immediate rollback of the last deployed version.
3.  **Communication Intervals**: CL must update the public status page every **15 minutes** for SEV-1 incidents, even if there is no change in status.

---

## 9. Class 7: Deployment Guides

Deployment Guides explain how to move stable code versions securely into staging and production clusters.

### 9.1 GitOps Integration
*   Deployments are managed via **ArgoCD** reading from Helm configurations in the `/charts/` folder.
*   Manual `kubectl apply` commands in production are strictly prohibited.

### 9.2 Canary Rollout Verification Rules
Deployments must follow a progressive rollout structure:
1.  **Initial Canary**: Direct 5% of target cluster traffic to the new version pods.
2.  **Bake Period**: Maintain the 5% split for **1 hour**.
3.  **Validation Checkpoints**:
    *   *Check 1*: Verify that error rates (HTTP 5xx, gRPC failures) did not spike on Canary pods.
    *   *Check 2*: Verify that p95 response latencies are within SLA limits (< 100ms).
    *   *Check 3*: Verify that container CPU and Memory usage are stable.
4.  **Promotion**: If all checkpoints pass, scale incrementally (25% -> 50% -> 100%). If any checkpoint fails, execute an automated rollback.

---

## 10. Class 8: Troubleshooting Guides

Troubleshooting Guides help engineers isolate common, recurring software anomalies and client integration errors.

### 10.1 Diagnostic Query Matrices
The guide must maintain an updated matrix mapping error codes to structural root causes and diagnostic commands:

| Error Code | Potential Root Cause | Diagnostic Step | Mitigation Action |
| :--- | :--- | :--- | :--- |
| `DOWNSTREAM_DISPATCH_FAULT` | Network timeout, DNS failure, backend container crash. | `kubectl logs -n nexus-core -l app=api-gateway` | Verify pod health; check downstream service status. |
| `CIRCUIT_BREAKER_OPEN` | Downstream service is failing consecutive requests. | Check Prometheus `circuit_breaker_state` gauge. | Run downstream service diagnostics; scale replicas if overloaded. |
| `PAYLOAD_TOO_LARGE` | Client request body exceeds maximum operational limits (10MB). | Inspect `Content-Length` header in logs. | Reject request; instruct client to compress or chunk payload. |

---

## 11. Class 9: Developer Guides

The Developer Guide provides everything a new engineer needs to set up their workstation, download the code, and run tests.

### 11.1 Workstation Setup Instructions
*   **Required Dependencies**: Go 1.22+, Node.js v20+, Docker Desktop.
*   **Repository Cloning**: Standard git clone parameters.
*   **Local Launch Procedure**:
    ```bash
    # Run the multi-module Go backend environment
    go work init
    go work use ./enterprise-platform/api-gateway
    go run ./enterprise-platform/api-gateway/cmd/server/main.go
    ```
*   **Test Suite Verification**: Instructions to verify compilation and run the full suite of unit and integration tests locally before submitting pull requests.

---

## 12. Class 10: Contributor Guides

The Contributor Guide defines the rules and policies for contributing code to the NexusCore codebase.

### 12.1 Contributor License Agreement (CLA)
All external contributors must sign the NexusCore CLA before a pull request can be accepted. The CLA ensures that all contributions are licensed under permissive enterprise terms.

### 12.2 Pull Request Submission Checklist
Before requesting review, developers must satisfy this automated check list:
*   [ ] Branch name conforms to conventional patterns (e.g., `feat/auth-rate-limit`).
*   [ ] Commits conform to Conventional Commit standards (e.g., `feat(api): add rate limits`).
*   [ ] Local compile passes cleanly without formatting warnings or linter flags.
*   [ ] Code coverage metrics are maintained or exceeded.
*   [ ] Documentation (READMEs, Swagger APIs, ADRs) is updated to reflect all code changes.

---

## 13. Automated Documentation Maintenance (DaC)

To prevent technical documentation from rotting, we utilize **Documentation-as-Code (DaC)** automation inside our continuous integration (CI) pipelines.

```
                  ┌─────────────────────────────────────────┐
                  │          Pull Request Opened            │
                  └────────────────────┬────────────────────┘
                                       │
         ┌─────────────────────────────┼─────────────────────────────┐
         ▼                             ▼                             ▼
┌──────────────────┐          ┌──────────────────┐          ┌──────────────────┐
│  Markdown Linter │          │   Link Checker   │          │ OpenAPI Sync Val │
│  Verify syntax.  │          │ Find dead links. │          │ Spec validation. │
└────────┬─────────┘          └────────┬─────────┘          └────────┬─────────┘
         │                             │                             │
         └──────────────────────┬──────┘                             │
                                │                                    ▼
                                ▼                           ┌──────────────────┐
                     [ Quality Gate Passed ] ──────────────►│ PR Merge Allowed │
                                                            └──────────────────┘
```

### 13.1 CI Validation Gates
Every Pull Request triggers automated documentation validation pipelines:
1.  **Markdown Syntax Linter**: `markdownlint` validates that all files follow standard markdown styling, heading structures, and whitespace configurations.
2.  **Broken Link Checker**: A custom script scans all markdown files inside `/docs/` and service READMEs to ensure that all relative paths and external URLs are active and resolve correctly.
3.  **OpenAPI Spec Schema Validator**: Evaluates OpenAPI spec sheets (`/docs/openapi.json`) to confirm schema compliance against the official specification.

### 13.2 Automated Doc Generation
*   **Protobuf Documenter**: Whenever `.proto` files are modified, a protobuf compilation task (`protoc-gen-doc`) automatically generates Markdown document listings detailing all messages, fields, and services.
*   **Swagger Sync**: Swagger portals dynamically load the OpenAPI spec sheet on demand, guaranteeing that client-facing documentation is always synchronized with the deployed API version.

# NexusCore Release & Versioning Standards

## 1. Release Philosophy

At NexusCore, code changes are integrated continuously and released safely. We minimize risk by releasing software in small, incremental, and highly automated cycles.

This document defines the deployment mechanics, versioning rules, Git policies, branching models, and release pipelines enforced across all NexusCore project repositories.

---

## 2. Versioning Standards (SemVer)

All software artifacts—including microservice binaries, library modules, Docker images, Protobuf APIs, and Helm charts—must adhere strictly to **Semantic Versioning 2.0.0 (SemVer)**.

### 2.1 SemVer Breakdown
Versions are defined in the format: **`MAJOR.MINOR.PATCH`**

*   **`MAJOR`**: Incremented when backward-incompatible API changes or schema modifications are introduced. Incrementing `MAJOR` requires approval from the **CTO** and **Principal Enterprise Architect**.
*   **`MINOR`**: Incremented when new, backward-compatible functionality, endpoints, or features are added.
*   **`PATCH`**: Incremented when backward-compatible bug fixes, optimizations, or security patches are applied.

---

## 3. Git Branching Strategy & Conventional Commits

We utilize a structured **Trunk-Based Development** model with short-lived feature branches. Long-lived, divergent release branches are strictly prohibited.

### 3.1 Branching Strategy
*   **`main` (Trunk)**: The single source of truth for all active production-ready code. Commits may never be pushed directly to `main`. All additions must go through short-lived feature branches merged via approved Pull Requests.
*   **Feature Branches (`feat/*`, `fix/*`, `chore/*`)**: Short-lived branches spawned off `main`. Feature branches must be merged back into `main` within **48 hours** to prevent complex merge conflicts.

### 3.2 Conventional Commits
All commit messages must adhere to the Conventional Commits specification. This allows automated tools to generate changelogs and compute next SemVer numbers during releases.

*   *Format*: `<type>(<scope>): <description>`
*   *Allowed Types*:
    *   **`feat`**: A new user-facing feature (maps to SemVer `MINOR`).
    *   **`fix`**: A bug fix (maps to SemVer `PATCH`).
    *   **`chore`**: Maintenance, package dependencies, tool adjustments.
    *   **`docs`**: Documentation changes only.
    *   **`refactor`**: Code changes that do not fix bugs or add features.
    *   **`perf`**: A code change that improves performance.
*   *Breaking Changes*: Must include a `BREAKING CHANGE:` block in the footer or append `!` after the type (maps to SemVer `MAJOR`).
*   *Example*:
    ```
    feat(api-gateway): add secure rate limiting middleware to protected paths
    ```

---

## 4. Pull Requests & Code Review Rules

No code may enter the `main` branch without satisfying the following automated gates and peer reviews:

### 4.1 Pull Request (PR) Requirements
To submit a PR, the author must guarantee:
1.  **Green CI Pipeline**: Code must compile, pass all linters (`golangci-lint`, `eslint`), and run unit and integration tests successfully.
2.  **Coverage Gate**: Statement coverage metrics must satisfy domain-specific targets (e.g., minimum 95% for security packages, 80% elsewhere).
3.  **Documentation Sync**: If an API contract is changed, the Protobuf declarations, OpenAPI definitions, ADRs, and READMEs must be updated in the same PR.

### 4.2 Code Review Rules
*   **Minimum Reviewers**: Every PR requires a minimum of **two independent positive approvals** from senior engineers before merge permissions are unlocked.
*   **Scope & Depth**: Reviewers must actively check for:
    *   Error handling adequacy.
    *   SQL injection vulnerabilities, race conditions, or unbuffered channel leaks.
    *   Sufficient unique HTML `id` targeting properties in React components.
*   **Constructive Tone**: Reviews must remain professional, respectful, and blameless. If rejecting, a reviewer must offer a clear alternative or explain *why* the code fails to satisfy coding standards.

---

## 5. Deployment Control & Progressive Releases (Helm & Kubernetes)

Deployments are coordinated using GitOps pipelines (ArgoCD) and structured Kubernetes objects.

### 5.1 Helm Chart Management
*   All microservices must contain a standard Helm chart under `/charts/helm/`.
*   All configurable environment variables, replica allocations, and resource limits must be declared in `values.yaml`. Hardcoding parameters inside Kubernetes templates is prohibited.

### 5.2 Progressive Deployment Strategy
Production releases must utilize **Canary Releases** or **Blue-Green Deployments**:
1.  **Canary Phase**: Deploy the new container image to a small, isolated subset of pods (e.g., 5% of traffic).
2.  **Observation**: Monitor Prometheus error counters, system latencies, and container logs for **1 hour**.
3.  **Rollout**: If no alerts are triggered, scale traffic to 100% and decommission the older containers. If anomalies are detected, execute an automated rollback.

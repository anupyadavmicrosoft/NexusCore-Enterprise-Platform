# Corporate Contributing Guidelines (NexusCore)

This document establishes the guidelines, workflows, review processes, and release standards for internal engineering contributions.

## 1. Branching Strategy & Git Flow

NexusCore enforces a structured **Trunk-Based Development** Git workflow. Direct commits to the `main` branch are strictly blocked by branch protection rules.

```
       [ main branch ] (Stable, ready for Production GKE)
             ^
             | (PR Approval, Green CI, Squashed Merge)
       [ feature/nexus-120-add-mfa ] (Short-lived developer branches)
```

### 1.1 Developer Branch Naming Standard
Branches must follow a strict, scannable naming convention referencing the target task or Jira ticket:
*   `feature/nexus-<ticket-id>-<short-description>` (e.g., `feature/nexus-124-add-cors-validation`)
*   `bugfix/nexus-<ticket-id>-<short-description>` (e.g., `bugfix/nexus-982-leak-in-socket-pool`)
*   `hotfix/<short-description>` (for direct patch deployment to production under SRE runbook commands)

---

## 2. Conventional Commit Standards

Commit messages must be clear and structured to enable automated CHANGELOG generation and semantic release updates:

### 2.1 Commit Structure
```
<type>(<scope>): <short description>

[Optional longer body detail]
[Optional footer referencing ticket IDs]
```

### 2.2 Accepted Commit Types
*   **`feat`**: A new feature implementation. (Increments minor version).
*   **`fix`**: A bug fix. (Increments patch version).
*   **`docs`**: Documentation alterations only.
*   **`style`**: Changes that do not affect code logic (formatting, spacing, etc.).
*   **`refactor`**: Code changes that neither fix a bug nor add a feature.
*   **`test`**: Restructuring or adding test cases.
*   **`chore`**: Maintenance, build system, or library dependency updates.

### 2.3 Commit Examples
```
feat(auth): support JWT algorithmic validation exclusions for alg none

Avoids JWT bypass attempts by throwing HTTP 401 Unauthorized exceptions 
when an unverified none signature payload is detected in headers.

Refs: NEXUS-124
```

---

## 3. Pull Request Submission & Review Process

Before submitting a Pull Request (PR) for review, complete the following checklist:

### 3.1 Pre-Submission Checklist
1. **Linting Verification**: Ensure code passes local linting constraints:
    ```bash
    golangci-lint run
    ```
2. **Local Tests Check**: Confirm that all unit tests execute successfully:
    ```bash
    go test -v -cover ./...
    ```
3. **Commit Sign-off**: All commits must be signed-off (`git commit -s`) to verify developer ownership compliance.

### 3.2 Code Review SLA
*   Every PR requires a minimum of **2 approved code reviews** from senior platform developers before merging.
*   Reviewers must evaluate architectural patterns, error propagation, memory allocation efficiency, and security exposures.
*   All automated CI/CD checks (unit tests, security scans, compilation tests) must report **green** before the merge block can release.

---

## 4. Semantic Versioning (SemVer) Release Policy

Releases are tagged automatically via the GitOps pipeline according to the Semantic Versioning 2.0.0 guidelines:
*   **MAJOR** version: Significant structural refactoring or API-breaking changes (e.g., `v2.0.0`).
*   **MINOR** version: Backwards-compatible functionality releases (e.g., `v1.4.0`).
*   **PATCH** version: Backwards-compatible bug fixes or minor vulnerability remediations (e.g., `v1.3.1`).

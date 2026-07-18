# NexusCore Testing Standards

## 1. Quality Assurance Philosophy

At NexusCore, tests are not written to fulfill a metric; they are written to guarantee that our production systems remain stable, resilient, and correct under continuous rapid changes.

This document establishes the testing methodologies, structure, coverage targets, and test automation rules mandated for all NexusCore repositories.

---

## 2. Testing Pyramid & Categories

We organize our testing suites in a traditional pyramid structure, prioritizing fast, deterministic unit tests, backed by targeted integration and end-to-end regression pipelines.

```
                  / \
                 /   \
                / E2E \  <-- 5% (Slow, high coverage)
               /-------\
              /  Integ  \  <-- 20% (Medium speed, docker dependencies)
             /-----------\
            /    Unit     \  <-- 75% (Sub-millisecond, highly isolated)
           /───────────────\
```

### 2.1 Unit Tests (`*_test.go`)
*   **Definition**: Tests that execute a single function, method, or struct in absolute isolation.
*   **Isolation Rules**: Unit tests **MUST NOT** make real network calls, connect to live databases, read external files, or rely on active downstream containers. All external dependencies must be mocked or stubbed.
*   **Performance Target**: Individual unit tests must execute in **less than 10 milliseconds**. The entire unit test suite for a microservice must compile and run in under **15 seconds**.

### 2.2 Integration Tests
*   **Definition**: Tests that validate the integration between a microservice and its immediate external dependencies (such as PostgreSQL, Redis, or Kafka).
*   **Infrastructure**: We utilize local Docker Compose stacks to spin up ephemeral database and broker instances during integration test execution.
*   **Command Pattern**:
    ```bash
    # Execute Go workspace integration scenarios
    go test -v -tags=integration ./...
    ```

### 2.3 End-to-End (E2E) Tests
*   **Definition**: Comprehensive regression scenarios that test a complete business flow (e.g., calling the API Gateway, authenticating via Auth Service, executing calculations on Compute Engine, and auditing logs).
*   **Execution**: E2E tests run automatically in CI pipelines targeting ephemeral Kubernetes namespaces or a staging environment prior to a production release.

---

## 3. Coverage Thresholds & CI Gates

To guarantee quality over time, CI/CD deployment pipelines enforce strict coverage limits.

### 3.1 Statement Coverage Requirements
*   **General Backend Logic**: Minimum **80%** statement coverage.
*   **Security & Auth Libraries**: Minimum **95%** statement coverage.
*   **Core Transaction Ledger**: Minimum **95%** statement coverage.

### 3.2 Gate Enforcement
Any Pull Request that causes the statement coverage of a modified package to fall below these thresholds will be automatically rejected by the CI system and cannot be merged.

---

## 4. Mocking & Test Double Policies

To write fast, deterministic unit tests, we utilize structured mocking conventions:

### 4.1 Interface-Driven Design
*   Always code against interfaces rather than concrete implementations for external resources (repositories, HTTP clients, message producers).
*   *Example*:
    ```go
    type UserRepository interface {
        GetUser(ctx context.Context, id string) (*User, error)
    }
    ```

### 4.2 Automated Mock Generation
*   Do not write custom, verbose, manual mock structures. Use standardized mock generators (such as `mockery` for Go or `jest.mock` for TypeScript) to generate test doubles automatically.
*   Keep mocks updated and checked into git or generated on the fly during the compile step.

---

## 5. Test Data Isolation & Idempotency

*   **No Inter-test State Pollution**: Every test must set up and tear down its own datasets. Relying on the outcome or ordering of a preceding test is strictly prohibited.
*   **Unique IDs**: Generate randomized strings or UUIDs for all test records to prevent primary key conflicts in shared integration databases.
*   **Database Cleansing**: After integration runs, execute a teardown script that truncates or cleanses the database tables rather than dropping the schemas completely.

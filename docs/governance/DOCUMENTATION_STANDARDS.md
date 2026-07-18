# NexusCore Documentation Standards

## 1. Documentation as Code (DaC)

At NexusCore, system documentation is treated with the same rigor, versioning, and code review rules as software source files. Documentation that is outdated or incorrect is a system bug.

This document establishes the structural standards, file templates, and review gates mandated for all corporate documentation by the **Senior Documentation Engineers** and the **CTO**.

---

## 2. Architecture Decision Records (ADRs)

All significant architectural modifications, framework migrations, database design shifts, or third-party integrations must be preceded by an **Architecture Decision Record (ADR)**.

### 2.1 ADR Lifecycle & Statuses
ADRs must be written in Markdown and checked into the repository under `/docs/adr/`. Every ADR must transition through these statuses:
*   **Draft**: The proposal is under active writing and internal discussion.
*   **Proposed**: Ready for review by the Architectural Board.
*   **Approved**: Authorized by the Principal Architects and CTO. Execution may proceed.
*   **Superceded**: A later decision has modified or overwritten this decision. Point to the new ADR.

### 2.2 ADR Structural Template
Every ADR must follow this exact Markdown structure:
*   **Title**: `# ADR [Index] — [Topic Name]`
*   **Status**: Active status (e.g., `Approved`) and date.
*   **Context**: The business problem, technical constraints, and choices considered.
*   **Decision**: The exact technology, framework, design pattern, or architecture selected. Explicitly state "We decided to...".
*   **Consequences**: The trade-offs of the choice. Explicitly list both **Positive** outcomes and **Negative** consequences.

---

## 3. API Contract Documentation

To support distributed development teams without coordination bottlenecks, all service APIs must be documented using machine-readable contracts.

### 3.1 REST APIs (OpenAPI / Swagger)
*   All client-facing REST APIs must maintain a corresponding **OpenAPI v3.0** specification file in JSON or YAML.
*   **Dynamic Exposing**: The API Gateway exposes Swagger UI directly in non-production environments under the `/swagger` path, compiling and reading from `/docs/openapi.json`.
*   **Validation**: Every parameter, header, request body, and return payload must explicitly declare its type, required status, and error status codes (`400 Bad Request`, `401 Unauthorized`, `403 Forbidden`, `500 Internal Error`).

### 3.2 Internal Services (gRPC / Protobuf)
*   gRPC APIs must be defined inside `.proto` files with explicit parameter naming, parameter types, package namespaces, and option configurations.
*   Every protobuf parameter and message must feature inline block comments explaining its operational purpose.

---

## 4. Code Documentation & Comment Policies

Code comments must explain **why** something is done, not **what** the code does. The code itself should clearly convey "what" it is doing.

### 4.1 Go Comment Standards
*   Exported fields, structs, functions, and packages must feature standard Go comment blocks ending in a period. The comment must begin with the entity's name:
    ```go
    // ReverseProxy handles load-balancing, timeouts, and downstream forwarding.
    type ReverseProxy struct {
        // ...
    }
    ```
*   Avoid inline, repetitive comments that state the obvious:
    ```go
    // Bad
    i++ // increment i
    ```

### 4.2 TypeScript & React JSDoc
*   Complex utility functions and exported React components must feature standard JSDoc markup declaring parameters, return structures, and component states.
    ```typescript
    /**
     * Calculates the exponential backoff delay with random jitter.
     * @param attempt - The current retry iteration (0-indexed).
     * @param baseDelayMs - The initial sleep delay in milliseconds.
     * @returns The duration to pause before executing the next retry.
     */
    export function calculateJitter(attempt: number, baseDelayMs: number): number {
        // ...
    }
    ```

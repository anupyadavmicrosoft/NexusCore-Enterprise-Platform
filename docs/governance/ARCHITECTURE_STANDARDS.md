# NexusCore Architecture Standards

## 1. Vision & Architecture Philosophy

NexusCore employs a highly available, event-driven, decoupled microservices architecture designed to support enterprise-scale high-throughput workloads.

This document outlines the architectural boundaries, microservice blueprints, communication standards, and resiliency practices mandated by the **Principal Enterprise Architect** and the **CTO**.

---

## 2. Microservice Boundaries & Communication

To ensure high cohesion and loose coupling, microservices are separated strictly by Domain-Driven Design (DDD) bounded contexts.

```
                  ┌─────────────────────────────────────────┐
                  │          Ingress / API Gateway          │
                  └────────────────────┬────────────────────┘
                                       │ (gRPC-Gateway / HTTP)
         ┌─────────────────────────────┴─────────────────────────────┐
         ▼ (gRPC / HTTP)                                             ▼ (gRPC / HTTP)
┌─────────────────┐                                         ┌─────────────────┐
│  Auth Service   │                                         │ Compute Engine  │
└────────┬────────┘                                         └────────┬────────┘
         │                                                           │
         ▼                                                           ▼
┌─────────────────┐                                         ┌─────────────────┐
│ PostgreSQL (DB) │                                         │ PostgreSQL (DB) │
└────────┬────────┘                                         └────────┬────────┘
         │                                                           │
         └─────────────────────────────┬─────────────────────────────┘
                                       │ (Async Pub/Sub)
                                       ▼
                            ┌─────────────────────┐
                            │    Kafka Cluster    │
                            └─────────────────────┘
```

### 2.1 Communication Protocol Selection
Microservices communicate through three distinct channels depending on latency, delivery guarantees, and interaction type:

1.  **gRPC (Primary Service-to-Service)**: All synchronous, internal backend communication must utilize gRPC. Protobuf definitions are the single source of truth for interfaces and contracts.
2.  **HTTP/REST (Gateway-to-Client)**: Exposed via the API Gateway using the **gRPC-Gateway** compilation layer or native high-performance routers (e.g., Gin). All client-facing APIs must follow RESTful standards and return JSON payloads.
3.  **Kafka (Asynchronous Pub/Sub)**: Used for state mutations, transaction auditing, and cross-domain events. Direct synchronous cross-service DB writes are strictly forbidden; changes must propagate asynchronously via Kafka.

### 2.2 Protobuf & API Contract Evolution
API contracts must remain backward-compatible to prevent deployment locks.
*   **No Field Re-numbering**: Once a field number is assigned in a `.proto` file, it can **never** be changed, re-assigned, or reused.
*   **Reserved Identifiers**: If a field is deprecated, mark it as `reserved` to prevent future engineers from using the same field number or tag name.
*   **Compatibility Checks**: Protobuf changes must be validated in CI pipelines using backward-compatibility linter tools before code generation is triggered.

---

## 3. Resilience & Traffic Controls

Backend microservices must protect themselves defensively. The edge gateway and internal proxies utilize four primary traffic control mechanisms:

### 3.1 Rate Limiting & Quota Management
*   **Token Bucket Algorithm**: All endpoints are guarded by a thread-safe token bucket rate limiter based on client metadata (e.g., authenticated API Keys or client IPs).
*   **Quota Enforcement**: Global, long-term quotas (e.g., maximum API queries per user per calendar month) are cached in highly available Redis instances and validated during gateway ingress.

### 3.2 Circuit Breakers
*   To prevent cascading failures across services, all outgoing calls to downstream services must be wrapped in a stateful circuit breaker.
*   **Breaker States**:
    *   **Closed**: Requests flow normally. Failures are counted.
    *   **Open**: Downstream service is failing. Requests are rejected immediately at the gateway, returning a `503 Service Unavailable` error to protect the downstream target.
    *   **Half-Open**: Periodic, limited requests are permitted to test downstream recovery. Any failure transitions the breaker immediately back to `Open`. Consistent success transitions it back to `Closed`.

### 3.3 Retry and Timeout Policies
*   **Strict Timeouts**: Every HTTP and gRPC request must feature an explicit, context-driven timeout. Undefined timeout configurations are a blocker for production deployment.
    *   *Standard Gateway-to-Backend Timeout*: **10 Seconds**.
    *   *Standard Backend-to-Database Timeout*: **3 Seconds**.
*   **Exponential Backoff Retries**: Failed idempotent requests (e.g., `GET` operations, structured transactions with unique idempotency keys) must be retried automatically using exponential backoff with random jitter to prevent "thundering herd" patterns.

---

## 4. Observability and Tracing

Distributed architectures are impossible to debug without tracing. 

### 4.1 Distributed Tracing (OpenTelemetry)
*   All ingress points must generate or propagate an **`X-Correlation-ID`** and standard OpenTelemetry tracing headers.
*   Spans must be injected into the context of every synchronous network request (HTTP headers or gRPC metadata) and asynchronous message payload (Kafka header bytes).

### 4.2 Structured Logging
*   All logs must be emitted to `stdout` in structured JSON formatting utilizing standard logging libraries (such as Go's `slog`).
*   **Mandatory Log Keys**: Every log line must include:
    *   `correlation_id`: Used to map trace context across microservices.
    *   `level`: `DEBUG`, `INFO`, `WARN`, or `ERROR`.
    *   `timestamp`: In RFC3339 format.
    *   `service`: The name of the originating microservice.

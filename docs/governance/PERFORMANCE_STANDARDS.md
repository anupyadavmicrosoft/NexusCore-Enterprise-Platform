# NexusCore Performance Standards

## 1. Objective & Performance-First Philosophy

To satisfy enterprise customer expectations, NexusCore software must remain performant, responsive, and resource-efficient. High latency is treated as a service degradation.

This document establishes the performance SLAs, resource usage budgets, profiling guidelines, and concurrency rules enforced by the **Principal Cloud Architect** and the **SRE team**.

---

## 2. Response Time Service Level Agreements (SLAs)

Every microservice endpoint must satisfy these latency limits under a normal peak workload (defined as 1.5x of historical standard volume):

| Metric | API Gateway Edge | Internal gRPC | DB Queries | Kafka Delivery |
| :--- | :--- | :--- | :--- | :--- |
| **p50 Latency** | **< 50ms** | **< 10ms** | **< 5ms** | **< 15ms** |
| **p95 Latency** | **< 100ms** | **< 25ms** | **< 15ms** | **< 50ms** |
| **p99 Latency** | **< 250ms** | **< 50ms** | **< 30ms** | **< 100ms** |

---

## 3. Resource Allocation & Container Budgets

To minimize cloud compute costs and prevent Out-Of-Memory (OOM) crashes inside Kubernetes clusters:

### 3.1 Kubernetes Container Limits
All microservices must specify CPU and Memory requests and limits in Helm templates.

*   **Standard Microservice Budget**:
    *   `resources.requests.cpu`: `100m` (0.1 CPU core)
    *   `resources.requests.memory`: `128Mi` (Megabytes)
    *   `resources.limits.cpu`: `500m` (0.5 CPU core)
    *   `resources.limits.memory`: `256Mi` (Megabytes)
*   **High-Compute (e.g., Compute Engine) Budget**:
    *   `resources.requests.cpu`: `500m`
    *   `resources.requests.memory`: `512Mi`
    *   `resources.limits.cpu`: `2000m` (2 CPU cores)
    *   `resources.limits.memory`: `1024Mi` (1 Gigabyte)

### 3.2 Memory Allocation Optimization
*   **Zero Dynamic Allocations inside Loops**: Avoid creating heap-allocated variables or repeating string concatenations inside deep iterative execution loops. Utilize `strings.Builder` or buffer pools (`sync.Pool`) to recycle memory allocations.
*   **Garbage Collection Tuning**:
    *   For memory-sensitive Go services, tune the runtime garbage collection threshold using `GOGC` (e.g., `GOGC=100` by default, drop to `GOGC=50` to reduce memory spikes, or adjust `GOMEMLIMIT` to avoid OOM crashes).

---

## 4. Performance Profiling & Optimization Guidelines

When an endpoint violates the latency SLAs:

### 4.1 CPU and Memory Profiling (Go pprof)
*   All Go microservices must compile-in the `net/http/pprof` endpoints (safeguarded behind internal Admin-only networks or gateway paths).
*   Engineers must analyze performance bottlenecks using pprof files:
    ```bash
    # Capture a 30-second CPU profile from a running container
    go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
    ```

### 4.2 Database Access Patterns (Avoid N+1 Queries)
*   **No N+1 Queries**: When retrieving a list of entities and their sub-items, execute a single JOIN query or collect primary keys and use an `IN` query. Never iterate through primary records and execute a separate database read query for every single row.
*   **Read Replicas**: Route heavy, non-transactional analytical read queries to PostgreSQL read replica instances to preserve the primary database's CPU cycles for transactional mutations.

---

## 5. Concurrency Controls & Pooling

To protect systems from resource exhaustion under load spikes:

### 5.1 Connection Pooling
*   Never open and close database connections on a per-request basis. Utilize connection pools with strict limits.
*   The API Gateway maintains an idle-connection pool for routing, reusing TCP sockets to avoid TLS handshake penalties on every client request.

### 5.2 Concurrency Limits
*   Avoid unbound goroutine spawning. Use standard worker pools to cap parallel thread counts.
*   Use contexts with cancellation signals (`context.WithCancel`, `context.WithTimeout`) to terminate active goroutines immediately if a client closes the HTTP connection early. This halts unneeded backend CPU cycles.

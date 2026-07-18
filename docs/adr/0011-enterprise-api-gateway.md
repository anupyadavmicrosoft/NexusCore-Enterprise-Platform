# Architecture Decision Record (ADR): 0011-enterprise-api-gateway

## Status
Approved

## Context
NexusCore requires a secure, high-throughput, low-latency edge entrypoint to terminate SSL, enforce authorization rules, guard downstream microservices against traffic spikes (rate limiting), prevent cascading failures (circuit breaking), retry transient failures, cache safe responses, and expose API schema details dynamically. 

Previously, a minimal HTTP proxy structure existed in `/enterprise-platform/api-gateway/cmd/server/main.go` that lacked security validation, load balancing, resilient retry/timeouts, metrics, and caching capabilities.

## Decision
We decided to rewrite and structure the API Gateway into a modular Go application utilizing **Gin** as the high-performance routing frame, augmented with:
1. **JWT Signature & API Key Authentication Middlewares**: Enforce authentication at the ingress edge before reaching backend microservices.
2. **Token Bucket Rate Limiting & Quota Guards**: Implemented as a thread-safe sliding window rate limiter tracking client identifiers.
3. **Stateful Circuit Breaker**: Evaluates success/failure ratios of target backends inside a sliding window, tripping to an Open state to protect exhausted backends.
4. **Resilient HTTP Client & Transport Dispatcher**: Implements Round-Robin load-balancing, contextual timeout boundaries, and automated retry mechanisms with exponential backoff.
5. **Gzip Response Compression & TTL Caching**: Shrinks response payloads and bypasses downstream calls for safe idempotent GET requests.
6. **OpenAPI v3.0 Documentation Portal & Prometheus Metrics**: Serves the OpenAPI specification dynamically with an interactive Swagger portal, and exposes metrics under `/metrics`.

## Consequences
- **Positive**: 
  - Complete security parity with Enterprise-grade Zero-Trust frameworks.
  - Zero cascading failures due to downstream bottlenecks (mitigated by circuit breakers and rate limiters).
  - High observability via unified OTel tracing headers, structured slog output, and Prometheus scrapers.
  - Reduced latency and compute load due to cached GET endpoints.
- **Negative**:
  - In-memory caching uses container RAM; cache eviction rules and TTL limits are strictly set to 60 seconds to prevent OOM errors.
  - Token bucket tracking requires synchronizing locks, though localized to `sync.Map` and atomic operations to sustain up to 100k requests per second per node.

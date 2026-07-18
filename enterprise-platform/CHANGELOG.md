# Changelog

All notable changes to the NexusCore platform will be documented in this file.

## [Sprint 11] - 2026-07-18
### Added
- **Enterprise API Ingress Gateway**: Completely refactored `api-gateway` in Go and Gin.
- **Security Middleware Group**: Added API Key authentication and cryptographic JWT signature checks.
- **Resiliency Protections**: Implemented thread-safe sliding window rate limiters, stateful circuit breakers, and exponential backoff retry loops.
- **Round-Robin Load Balancing**: Added automatic distribution of incoming calls across downstream backend microservice clusters.
- **Dynamic Swagger UI Portal**: Integrated embedded OpenAPI 3.0 specification under `/docs/openapi.json` and a portal at `/swagger`.
- **Prometheus Observability**: Registered request histograms, failure counters, and state gauges under `/metrics`.
- **Performance Optimizations**: Enabled response Gzip compression and in-memory TTL caching for safe read requests.
- **Helm Configuration**: Declared target production deployment baselines in `charts/helm/values.yaml`.
- **Architecture Documentation**: Formulated `0011-enterprise-api-gateway.md` Architecture Decision Record.

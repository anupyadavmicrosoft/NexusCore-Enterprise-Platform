# NexusCore Enterprise Istio Service Mesh Governance Specification

## 1. Executive Mandate & Zero-Trust Mesh Architecture

This document establishes the official **NexusCore Enterprise Service Mesh Standards**.

To protect lateral traffic, implement deep observability, and enable advanced traffic steering, all production deployments must utilize **Istio Service Mesh**. Security is governed by a **Zero-Trust Mutual TLS (mTLS)** policy enforced at the proxy layer, while application routing is decoupled from service definitions using Envoy-backed custom resources.

Manual mesh modifications or sidecar bypass configuration are strictly prohibited.

---

## 2. Service Mesh Architecture & Data Flow

The Service Mesh bifurcates communication into a logical Control Plane (Istiod) and a highly performant Data Plane composed of sidecar Envoy proxies running alongside every microservice container:

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                              EDGE / INGRESS GATEWAY BOUNDARY                           │
│  - Public Port 80/443 (TLS terminated via Let's Encrypt / Google-Managed Cert)        │
│  - Gateway Resource binds to host name api.nexuscore-enterprise.com                    │
└──────────────────────────────────────────┬─────────────────────────────────────────────┘
                                           ▼ (mTLS Encrypted Mutual TLS)
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                              API-GATEWAY PROXY (Sidecar)                              │
│  - AuthorizationPolicy enforces ingress rules from Ingress Gateway ONLY                 │
│  - VirtualService steers traffic based on path rules (/api/v1/auth, /api/v1/compute)   │
└──────────────┬───────────────────────────────────────────────────┬─────────────────────┘
               ▼ (mTLS Strict)                                     ▼ (mTLS Strict)
┌──────────────────────────────┐                   ┌──────────────────────────────┐
│     AUTH-SERVICE (Sidecar)   │                   │   COMPUTE-ENGINE (Sidecar)   │
│  - Enforces JWT context      │                   │  - VirtualService Retries    │
│  - AuthPolicy: Gateway only  │                   │  - DestinationRule Break     │
└──────────────┬───────────────┘                   └──────────────┬───────────────┘
               │                                                  │
               ▼ (mTLS to State)                                  ▼ (Egress Gateway)
┌──────────────────────────────┐                   ┌──────────────────────────────┐
│      DATABASE / CACHE        │                   │    EGRESS GATEWAY (Proxy)    │
│   (PostgreSQL / Redis HA)    │                   │  - Restricts external APIs   │
└──────────────────────────────┘                   └──────────────┬───────────────┘
                                                                  ▼
                                                   ┌──────────────────────────────┐
                                                   │    AUTHORIZED THIRD-PARTY    │
                                                   │      (External API Keys)     │
                                                   └──────────────────────────────┘
```

---

## 3. Mutual TLS & Identity Security (mTLS)

### 3.1 PeerAuthentication
*   **mTLS Posture**: We enforce **STRICT** mutual TLS mesh-wide via a global `PeerAuthentication` resource inside the root namespace (`istio-system` or `nexuscore-prod`).
*   **Permissive Mode Prohibition**: Permissive mode is strictly prohibited in production, except during a 24-hour phased brownfield migration window.
*   **SPIFFE Identity**: Workloads discover each other via SPIFFE-compatible client certificates formatted as:
    `spiffe://cluster.local/ns/<namespace>/sa/<service-account-name>`

### 3.2 AuthorizationPolicies (Lateral Microsegmentation)
*   No service is allowed to talk to another unless explicitly whitelisted via an `AuthorizationPolicy`.
*   **Ingress Gateway**: Only traffic from the Istio Ingress Gateway is allowed to reach `api-gateway`.
*   **Backends**: Only the `api-gateway` identity is authorized to contact `auth-service` or `compute-engine`.

---

## 4. Traffic Management, Canaries, and Resiliency

To prevent cascading failures and guarantee seamless user experiences, we implement advanced Envoy-backed traffic routing:

### 4.1 Traffic Splitting (Canaries & Blue-Green)
*   **VirtualService Weighting**: Application deployments utilize weight-based traffic shifting (e.g., 90% production, 10% canary) to validate new releases before full rollout.
*   **Header-Based Routing**: Internal testers can trigger canary pathways by passing specific request headers (e.g., `x-release-tier: canary`).

### 4.2 Resiliency & Fault Tolerance
*   **Retries**: In the event of transient failures, Envoy proxies automatically attempt **3 retries with exponential backoffs** (configured with a 2-second timeout per retry).
*   **Timeouts**: Maximum duration for a request before failure is capped to enforce system latency bounds (e.g., 5 seconds for normal APIs, 600 seconds for heavy compute).
*   **Circuit Breaking**: Under persistent failure, Envoy's circuit breaker trips to isolate bad replicas. We configure:
    *   `consecutive5xxErrors: 3` (3 sequential failures leads to ejection)
    *   `baseEjectionTime: 30s` (Node ejected for 30 seconds)
    *   `maxEjectionPercent: 100` (Allows ejecting all broken instances if necessary)

---

## 5. Egress Contraction & External Whitelisting

To mitigate data exfiltration risks:
*   **Mesh Configuration**: The mesh registry blocks direct egress by setting `outboundTrafficPolicy.mode = REGISTRY_ONLY`.
*   **Egress Gateway**: All external third-party traffic must be routed through a dedicated, monitored **Egress Gateway** cluster. This forces traffic auditing and prevents lateral nodes from issuing arbitrary external outbound calls.

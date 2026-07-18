# NexusCore Helm Chart Infrastructure Governance Specification

## 1. Executive Mandate & DRY Paradigm

This document establishes the official **NexusCore Helm Chart Infrastructure Standards**. 

To maintain unified, secure, and easily maintainable deployments across GKE clusters, we strictly prohibit duplicate Kubernetes YAML definitions. All stateless microservices within the NexusCore Enterprise platform must utilize the **unified `nexuscore-service` base Helm chart**, specialized entirely through environment-specific and microservice-specific `values.yaml` files.

---

## 2. Reusable Chart Architecture

The `nexuscore-service` chart implements a highly dynamic, parameterizable engine capable of auto-generating all core and edge Kubernetes resources based on declaration alone.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       REUSABLE HELM CHART ENGINE (nexuscore-service)       │
│  Declares templates for:                                                    │
│  - Deployment (Rolling update, Non-root, Read-only file-system)             │
│  - Service (ClusterIP, exposed on configurable port)                         │
│  - Ingress (Custom class, TLS termination, SSL redirect, rate-limiting)      │
│  - ConfigMaps & Secrets (Specialized and dynamically injected)             │
│  - Autoscaler (HPA v2 matching CPU and Memory targets)                      │
│  - NetworkPolicies (Zero-Trust isolation rules)                             │
│  - ServiceMonitors & PrometheusRules (Metrics and alerts)                  │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
            ┌──────────────────────────┼──────────────────────────┐
            ▼                          ▼                          ▼
┌───────────────────────┐  ┌───────────────────────┐  ┌───────────────────────┐
│  api-gateway.values   │  │  auth-service.values  │  │ compute-engine.values │
│ - Ingress enabled     │  │ - Secret DB injection │  │ - Dedicated HPA scale │
│ - TLS termination     │  │ - DB/Cache egress NPV │  │ - Kafka connectivity  │
│ - Rates: 100 rps      │  │ - Ingress: Gateway    │  │ - Ingress: Gateway    │
└───────────────────────┘  └───────────────────────┘  └───────────────────────┘
```

---

## 3. Deployment & Validation Workflow

Every modification to the core Helm templates or specialized values files must undergo automated quality verification in the continuous delivery pipeline.

### 3.1 Lint & Structure Verification
Ensure syntactical correctness and alignment with standard best-practices:
```bash
helm lint /enterprise-platform/charts/nexuscore-service
```

### 3.2 Manifest Generation Audits (Dry-Run / Template)
To audit the compiled Kubernetes manifests before pushing to target namespaces:
```bash
# Render the API Gateway manifest
helm template api-gateway /enterprise-platform/charts/nexuscore-service -f /enterprise-platform/charts/api-gateway.values.yaml

# Render the Auth Service manifest
helm template auth-service /enterprise-platform/charts/nexuscore-service -f /enterprise-platform/charts/auth-service.values.yaml

# Render the Compute Engine manifest
helm template compute-engine /enterprise-platform/charts/nexuscore-service -f /enterprise-platform/charts/compute-engine.values.yaml
```

---

## 4. Production Security & Quality Enforcements

To comply with our **Zero-Trust Infrastructure** directives, the unified Helm chart templates enforce the following rules:

1.  **Immutability**: Containers run with `readOnlyRootFilesystem: true`. Write-capable ephemeral folders (such as `/tmp`) must utilize dynamic `emptyDir` or `tmpfs` volume configurations.
2.  **Least Privilege**: Root contexts are stripped entirely (`runAsNonRoot: true`, `runAsUser: 65532`, `allowPrivilegeEscalation: false`).
3.  **Strict Lateral Limits**: Individual NetworkPolicies limit access strictly to required routes (e.g. `auth-service` cannot contact Kafka, and `compute-engine` cannot speak to Redis).
4.  **Prometheus Integration**: Metrics scraping is decoupled via dedicated `ServiceMonitor` structures, allowing target scrapers to auto-discover workloads using labels.
5.  **Autonomous Alerting**: `PrometheusRules` are registered alongside the application to raise alarms on latency spikes, high error ratios, or message queues lag.

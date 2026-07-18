# Enterprise Security Architecture & Compliance (NexusCore)

This document specifies the security controls, cryptographical mechanisms, network partition policies, and SAST/DAST verification rules.

## 1. Zero-Trust Mesh Architecture & PEP/PDP Pattern

NexusCore operates on a Zero-Trust Network Architecture (ZTNA). Every request must be authenticated, authorized, and cryptographically verified at both the entry edge boundary and between microservice nodes.

```
 [ External Client ] ---> | PEP: api-gateway (Port 8080) |
                                  |
                                  | (gRPC mTLS)
                                  v
                         | PDP: auth-service (Port 8081) |
```

*   **Policy Enforcement Point (PEP)**: Managed at the `api-gateway` layer. It acts as the gatekeeper, terminating external TLS, sanitizing query inputs, and assessing rate limits.
*   **Policy Decision Point (PDP)**: Located at the `auth-service` layer. Inspects token scopes, RBAC permissions, and makes execution allowance determinations.

---

## 2. Cryptographic Security Standards

### 2.1 JSON Web Tokens (JWT) Validation Policy
*   **Algorithm**: HMAC SHA-256 (for token payloads integrity verification) or RS256 (asymmetric keys).
*   **Token Expiry**: Strict timeout ceiling of `3600 seconds` (1 hour) for access tokens, and `14 days` for refresh tokens.
*   **Algorithm 'None' Safeguards**: API Ingress explicitly blocks and rejects header payloads referencing `"alg": "none"`, preventing signature bypass attacks.

### 2.2 Mutual TLS (mTLS) Mesh Communication
Every microservice-to-microservice gRPC connection requires mutual TLS authentication using **Istio** or **Linkerd** SPIFFE/SPIRE certificates. Cleartext internal TCP traffic is blocked.

---

## 3. Kubernetes Network Policies

Network isolation is enforced declaratively. Default firewall rules deny all cross-namespace traffic. Pods can only communicate with approved dependencies.

### 3.1 Network Policy: Restrict DB Ingress to Auth & Compute Only
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: database-access-policy
  namespace: nexus-core
spec:
  podSelector:
    matchLabels:
      app: postgres-db
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: auth-service
    - podSelector:
        matchLabels:
          app: compute-engine
    ports:
    - protocol: TCP
      port: 5432
```

---

## 4. Secret Management Protocol

*   **No Hardcoded Credentials**: Source code files are strictly forbidden from checking in database passwords, JWT secrets, or cloud service keys.
*   **GCP Secret Manager Integration**: Credentials are provisioned inside Google Cloud Secret Manager and mounted dynamically into pods as memory-only volumes via the Kubernetes **External Secrets Operator (ESO)**:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: postgres-db-secret
  namespace: nexus-core
spec:
  refreshInterval: "1h"
  secretStoreRef:
    name: gcp-store
    kind: ClusterSecretStore
  target:
    name: k8s-db-credentials
  data:
  - secretKey: db_password
    remoteRef:
      key: prod_postgres_password
```

---

## 5. DevSecOps: SAST & DAST Automated Pipeline

To preserve enterprise compliance standard levels, automated validation runs on every commit:

*   **SAST (Static Application Security Testing)**:
    *   Tool: `gosec` for Go codebases.
    *   Command: `gosec -fmt=json -out=sast_results.json ./...`
    *   Scan criteria: Evaluates memory leakage vulnerabilities, unsafe math, weak hashes, or hardcoded strings.
*   **DAST (Dynamic Application Security Testing)**:
    *   Tool: `OWASP ZAP` & `security_test.py` custom suites.
    *   Scan criteria: Actively injects SQL Injection strings, CORS header parameters, and JWT algorithmic bypasses against edge gateways to verify correct rejection handling.

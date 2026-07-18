# NexusCore Security Standards

## 1. Zero-Trust Security Paradigm

NexusCore is constructed on a **Zero-Trust Network Architecture (ZTNA)**. We operate under the structural assumption that the internal network boundary is compromised. 

This document defines the mandatory cryptography, authentication, authorization, and secret governance rules established by the **Chief Information Security Officer (CISO)**.

---

## 2. Authentication (AuthN) & Credentials

Every API request reaching a NexusCore microservice must be authenticated. An unauthenticated request must be rejected at the API Gateway or edge proxy before consuming internal microservice compute resources.

### 2.1 Identity Providers & OIDC / OAuth2
*   **Federated Identity**: Customer and system-to-system identities are managed via OpenID Connect (OIDC) or OAuth2 identity servers.
*   **Bearer Tokens**: All client-facing API requests must feature a standard cryptographic bearer JSON Web Token (JWT) inside the `Authorization` header:
    ```http
    Authorization: Bearer <JWT_TOKEN_BODY>
    ```

### 2.2 JWT Verification Parameters
Internal microservices must verify JWT signatures using cryptographic public keys (RS256/ECDSA) or HMAC symmetric secrets (HS256) loaded securely on startup.
*   **Mandatory Claim Validation**:
    *   **Expiration Time (`exp`)**: Tokens must feature a short expiration TTL (typically **1 Hour** or less). Expired tokens must be rejected instantly.
    *   **Issuer (`iss`)**: Must match the authorized NexusCore Identity Server URL exactly.
    *   **Audience (`aud`)**: Must validate that the token was intended for the target service namespace.
*   **Mock Verification for Local Environments**: In local, air-gapped test environments, services may support mock prefix validation (e.g., `mock_oidc_`) to ease local debugging without connecting to the cloud identity server. This mock configuration must be completely compiled-out or disabled in production environments.

### 2.3 API Key Governance
For machine-to-machine, server-to-server, and automated CI/CD integrations:
*   API keys must be securely generated using high-entropy cryptographically secure random number generators (minimum 32 bytes hex-encoded).
*   Keys must be provided inside the **`X-API-Key`** header.
*   API keys must be stored in the database in a one-way hashed format (e.g., using SHA-256). Storing raw, plain-text API keys in a database or config file is strictly prohibited.

---

## 3. Authorization (AuthZ) & RBAC

Authentication verifies *who* the client is; authorization determines *what* they are permitted to do.

### 3.1 Role-Based Access Control (RBAC)
We enforce strict role-based access checks at the API Gateway and microservice level. Active roles are defined below:

| Role | Permissions | Typical Use Case |
| :--- | :--- | :--- |
| **Admin** | Full operational control, database schema mutations, security overrides, billing modifications. | High-clearance DevOps, System Administrators. |
| **Operator** | Read-write access to transaction datasets, initiating compute jobs, reading metrics. | Support engineering, automation scripts. |
| **Guest / Anonymous** | Read-only access to public endpoints (healthz, docs), initiating signup requests. | Public traffic, unauthenticated clients. |

### 3.2 Verification Middleware
All microservice entrypoints must inject a validation middleware that asserts the client's role against the endpoint's requirements.
```go
// Example of Go middleware enforcement
func RequireAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        role, exists := c.Get("role")
        if !exists || role.(string) != "Admin" {
            c.JSON(http.StatusForbidden, gin.H{"error": "Access denied. Admin role required."})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

---

## 4. Cryptographic Standards & Secrets Management

To protect data in transit and at rest:

### 4.1 Encryption Rules
*   **Transit (TLS)**: All public and internal service-to-service communications must be encrypted using TLS 1.3 (with fallback to TLS 1.2 using secure cipher suites only). SSL, TLS 1.0, and TLS 1.1 are strictly deprecated.
*   **At Rest**: Sensitive personal data (PII) and credentials must be encrypted in PostgreSQL using industry-standard AES-256-GCM algorithms before serialization.
*   **Hashing**: Password structures and credential validations must utilize Argon2id or bcrypt (with a minimum cost factor of 12) for hash storage. Never use MD5, SHA-1, or plain SHA-256 for password hashing.

### 4.2 Secrets Governance
*   **No Hardcoded Secrets**: Under no circumstances may API keys, cryptographic secrets, database passwords, or certificates be committed to the Git repository.
*   **Environment Injection**: Secrets must be injected into the application context at runtime using secure container environment variables loaded from cloud managers (e.g., Google Secret Manager or HashiCorp Vault).
*   **Local Templates**: Every project must maintain a `.env.example` file documenting all required environment variables with blank placeholders, serving as a template for local developers.

# NexusCore Enterprise Platform - Identity Platform Architecture
## Sprint 3 Architectural Design & Cryptographic Specification

This document presents the complete architectural blueprint and security specification for the **NexusCore Identity Platform**. Designed as a cloud-native, highly available, multi-tenant Identity and Access Management (IAM) engine, it serves as the foundation for authentication (AuthN), authorization (AuthZ), and governance across the entire NexusCore ecosystem.

---

## 1. Identity Architecture

The Identity Platform is architected as a decoupled, multi-tenant microservice (`auth-service`) interacting with high-performance storage and cache layers. It supports physical and logical isolation boundaries suitable for global enterprise SaaS deployments.

### 1.1 Architectural Topology
The diagram below illustrates the decoupled structure of the Identity Platform, demonstrating how incoming requests pass through the API Gateway, interface with the secure `auth-service`, and access private caching and persistence zones.

```
                           +--------------------------+
                           |   Enterprise Client IP   |
                           +------------+-------------+
                                        |
                                        | HTTPS (TLS 1.3)
                                        ▼
                           +--------------------------+
                           |       API Gateway        |
                           +------------+-------------+
                                        |
                 +----------------------+----------------------+
                 | gRPC (mTLS)                                 | gRPC (mTLS)
                 ▼                                             ▼
+------------------------------+               +------------------------------+
|     auth-service (Node A)    |               |     auth-service (Node B)    |
|   OIDC / JWT / MFA Engine    |               |   OIDC / JWT / MFA Engine    |
+--------+--------------+------+               +--------+--------------+------+
         |              |                               |              |
         | Redis Cache  | PostgreSQL (Primary)          | Redis Cache  | PostgreSQL (Primary)
         | Protocol     | Connection Pool               | Protocol     | Connection Pool
         ▼              ▼                               ▼              ▼
  +--------------+  +--------------------------------------------------+
  | Redis Cache  |  |                 PostgreSQL Cluster               |
  | Cluster (HA) |  |   - Multi-tenant schemas with RLS enabled        |
  | Sessions /   |  |   - Partitioned tables for audit trails          |
  | Rate Limits  |  |   - Encrypted data-at-rest (AES-256-GCM)         |
  +--------------+  +--------------------------------------------------+
```

### 1.2 Data Schema Strategy
The database uses **PostgreSQL** with Row-Level Security (RLS) policies to enforce tenant isolation. Table names are partitioned by Tenant ID for extreme throughput, and database files are protected using AES-256-GCM at-rest encryption.

```sql
-- Core Tenants Table
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Core Organizations Table (Multi-level Hierarchy)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    path LTREE NOT NULL, -- PostgreSQL ltree for hierarchical queries
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Core Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    org_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    mfa_secret VARCHAR(128),
    mfa_enabled BOOLEAN DEFAULT FALSE,
    failed_login_attempts INT DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    password_changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_tenant_email UNIQUE(tenant_id, email)
);

-- Enable RLS on Tenant-bound tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON users
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

---

## 2. Authentication Flow (AuthN)

The authentication engine operates on a zero-trust model. Passwords are never stored in plaintext, and brute-force attacks are mitigated via adaptive rate-limiting and account lockout mechanisms.

### 2.1 Cryptographic Hashing Standards
All user passwords are encrypted using **Argon2id** (the winner of the Password Hashing Competition) with the following enterprise parameters (complying with RFC 9106):
-   **Memory Cost ($m$):** 65,536 KB (64 MiB)
-   **Time Cost ($t$):** 3 iterations
-   **Parallelism ($p$):** 4 threads
-   **Salt Length:** 16 bytes (cryptographically secure random generator)
-   **Key Length:** 32 bytes

### 2.2 Interactive Authentication Workflow
The sequence diagram below displays the interaction during an interactive login request including primary validation and multi-factor challenge verification.

```
Client App                   API Gateway                 auth-service                   Redis Cache             Database (Postgres)
    |                             |                           |                             |                            |
    |--- POST /auth/login ------->|                           |                             |                            |
    |    (email, password, etc)   |--- Validate Rate Limit -->|                             |                            |
    |                             |    & Forward Request      |                             |                            |
    |                             |                           |--- Check Lockout State ---->|                            |
    |                             |                           |    (Tenant:Email)           |                            |
    |                             |                           |<-- State Returned (Clear) --|                            |
    |                             |                           |                                                          |
    |                             |                           |--- Fetch user metadata & password_hash ----------------->|
    |                             |                           |<-- User Entity Returned ---------------------------------|
    |                             |                           |                                                          |
    |                             |                           |--- Compute Argon2id Hashing & Verify --------------------> [Internal Crypto]
    |                             |                           |                                                          |
    |                             |                           |--- Is MFA Enabled? (Yes)                                 |
    |                             |                           |                                                          |
    |                             |                           |--- Generate ephemeral MFA ticket in Cache ------------->|
    |                             |                           |<-- Ticket ID Mapped -------------------------------------|
    |                             |<-- Return MFA Challenge --|                                                          |
    |<-- MFA Required (Ticket ID)-|                                                                                      |
    |                                                                                                                    |
    |--- POST /auth/mfa/verify ----------------------------------------------------------------------------------------->|
    |    (Ticket ID, TOTP Code)                                                                                          |
    |                             |                                                                                      |
    |                             |---------------------------|--- Fetch Ephemeral MFA Ticket & TOTP Secret ------------>|
    |                             |                           |<-- Ticket & Secret Returned -----------------------------|
    |                             |                           |                                                          |
    |                             |                           |--- Cryptographically Validate TOTP Code -----------------> [Internal Crypto]
    |                             |                           |                                                          |
    |                             |                           |--- Generate JWT & Refresh Token pairs ------------------>|
    |                             |                           |--- Write Active Session Descriptor to Cache ------------>|
    |                             |<-- Return Auth Tokens ----|                                                          |
    |<-- Auth Success (JWTs) -----|                                                                                      |
```

---

## 3. Authorization Flow (AuthZ)

NexusCore couples **Role-Based Access Control (RBAC)** for coarse-grained resource access with **Attribute-Based Access Control (ABAC)** for fine-grained, contextual execution boundaries.

### 3.1 Policy Evaluation Model
The platform adopts an Envoy-compatible Open Policy Agent (OPA) architecture. The Gatekeeper intercepts traffic, extracts the client's token identity, maps permissions, and evaluates policy decisions based on incoming variables:
-   **Subject ($S$):** User UUID, active role list, identity tenant, organization scope.
-   **Resource ($R$):** System module, action URI, unique resource owner.
-   **Action ($A$):** HTTP verb mapping (`GET` -> Read, `POST` -> Create, `PUT`/`PATCH` -> Update, `DELETE` -> Delete).
-   **Context ($C$):** Request IP, Geo-location, device score, current timestamp.

```
       +------------------+
       |   HTTP Request   |
       +--------+---------+
                |
                ▼
       +------------------+
       |   API Gateway    |
       +--------+---------+
                |
                | Forward to PDP
                ▼
       +------------------+          Fetch Roles / Context
       | Policy Decision  | <-------------------------------------+
       |   Point (PDP)    |                                       |
       +--------+---------+                                       ▼
                |                                      +--------------------+
                | Evaluate Policy                      |   auth-service     |
                | (Rego / ABAC Rules)                  |   (Claims Store)   |
                ▼                                      +--------------------+
       +------------------+
       |   Allow / Deny   |
       +------------------+
```

### 3.2 OPA Rego Authorization Rule Example
Below is the standard Rego layout loaded inside sidecar Gatekeepers to filter administrative traffic by IP CIDR block and authorization scope.

```rego
package nexuscore.authz

default allow = false

# Retrieve token metadata
import input.jwt.claims as user_claims
import input.request as http_request

# Allow administrator access if inside private CIDR range
allow {
    user_claims.role == "SYSTEM_ADMIN"
    net.cidr_contains("10.0.0.0/8", http_request.client_ip)
}

# Allow standard operators matching path patterns
allow {
    user_claims.role == "ORGANIZATION_OPERATOR"
    http_request.method == "GET"
    re_match("^/api/v1/organizations/[a-f0-9-]+/metrics", http_request.path)
    user_claims.org_id == http_request.org_id
}
```

---

## 4. Lifecycle Management

To prevent data drift and secure accounts, every dynamic element (users, roles, permissions, organizations, tenants) is bound to strict state machine lifecycles.

### 4.1 User Lifecycle
A user account transitions through a secure DAG state machine. State adjustments are permanently recorded in the immutable audit trail.

```
    [ Provisioned ]
           │
           │ (Invitation Sent)
           ▼
     [ Pending ] <──────────────┐ (Self-Service Reset)
           │                    │
           │ (Verify Email /    │
           │  Set Password)     │
           ▼                    │
      [ Active ] ───────────────┼─────────────┐
        │    ▲                  │             │
        │    │                  │             │
        │    └─ (Unlock Account)│             │
        │                       │             │
        │ (Failed Logins > 5)   │             │
        ▼                       │             │ (Deprovision)
      [ Locked ] ───────────────┘             │
        │                                     │
        │ (Administrative Action)             │
        ▼                                     ▼
  [ Suspended ] ────────────────────────> [ Archival ] (Soft Deleted)
```

-   **Provisioned:** Account created via administrative API or automated onboarding. Password fields are empty.
-   **Pending:** Verification email transmitted. Account is non-functional; login calls will be rejected.
-   **Active:** Hashing executed, verification confirmed, MFA initiated. System access permitted.
-   **Locked:** Triggered by five consecutive failed credentials. Remains locked for a cool-down block of 30 minutes.
-   **Suspended:** Administrative override to instantly lock access. Sessions invalidated across the Redis store.
-   **Archival:** Soft-deleted record. Anonymizes PII data fields to comply with GDPR "Right to be Forgotten" mandates.

### 4.2 Role Lifecycle
Roles define semantic access boundaries. Organization admins can instantiate custom roles or assign standard system roles:
-   **Draft:** Role structure is being designed; cannot be assigned to users.
-   **Active:** Bound to users and actively evaluated by OPA gates.
-   **Deprecated:** Warning emitted when edited or used; blocks any new user associations.
-   **Deactivated:** Fully disabled; any evaluation attempts fall back to "Guest" or "Deny".

### 4.3 Permission Lifecycle
Permissions represent discrete action scopes formatted as colon-separated wildcards (e.g., `tenant:org:billing:write`).
-   **Registration:** Permissions are hardcoded inside microservice schemas and registered with the database on migration.
-   **Association:** Mapped to roles via junction tables.
-   **Deprecation:** Removed from the system automatically when services are upgraded and modules decommissioned.

### 4.4 Organization Lifecycle
Organizations are logical subdivisions of a tenant, managing business units:
-   **Created:** Instantiated beneath a specific parent node.
-   **Active:** Full read/write microsegmentation boundaries enabled.
-   **Suspended:** All internal operations blocked, though tenant administrators can view metadata.
-   **Terminated:** Hard delete or purge of all workspace documents linked to the organization hierarchy.

### 4.5 Tenant Lifecycle
Tenants represent the ultimate data separation boundary in SaaS environments:
-   **Onboarding:** Dedicated PostgreSQL schemas, S3 storage buckets, and Secret vaults are provisioned via Terraform.
-   **Active:** Live tenant traffic routed dynamically based on subdomain or custom domains.
-   **Suspended:** Financial or compliance lockout. Complete traffic rejection at the ingress network boundary.
-   **Offboarding:** Immutable audit export generated, followed by cryptographic erasure ("Crypto-shredding") of tenant encryption keys, rendering all stored fields instantly unreadable.

---

## 5. OAuth2 & OpenID Connect (OIDC) Flows

NexusCore provides a fully compliant RFC 6749 OAuth 2.0 authorization server and an OpenID Connect 1.0 identity provider to manage internal microservice bindings and support third-party integrations.

### 5.1 Authorization Code Grant with PKCE (Proof Key for Code Exchange)
To protect native mobile and Single Page Applications (SPAs) against interception attacks, the platform mandates **PKCE** (RFC 7636).

```
Client App (SPA)             User Browser              API Gateway (OIDC)           auth-service
    |                             |                           |                            |
    |--- 1. Click Login --------->|                           |                            |
    |    Generate Verifier &      |                           |                            |
    |    Challenge (SHA-256)      |                           |                            |
    |                             |                           |                            |
    |--- 2. Redirect with Challenge ------------------------->|                            |
    |    (client_id, code_challenge, code_challenge_method=S256)                           |
    |                             |                           |--- 3. Prompt Credentials ->|
    |                             |<-- Render Login UI -------|                            |
    |                             |                           |                            |
    |                             |--- 4. Authenticate ------>|                            |
    |                             |    (MFA Verified)         |                            |
    |                             |                           |--- 5. Issue Temp Code ---->|
    |                             |<-- 6. Redirect with Code -|                            |
    |<-- Get Code ----------------|                           |                            |
    |                             |                           |                            |
    |--- 7. Exchange Code ----------------------------------->|                            |
    |    (code, code_verifier)    |                           |                            |
    |                             |                           |--- 8. Validate Verifier -->|
    |                             |                           |    SHA256(verifier) ==     |
    |                             |                           |    stored challenge        |
    |                             |                           |                            |
    |                             |                           |--- 9. Issue Tokens ------->|
    |                             |<-- 10. Return ID & JWT ---|                            |
    |<-- Store Tokens Securely ---|                           |                            |
```

### 5.2 OIDC Discovery Document (`.well-known/openid-configuration`)
The platform exposes metadata automatically, allowing automated OIDC integrations to instantly map parameters:

```json
{
  "issuer": "https://identity.nexuscore.com",
  "authorization_endpoint": "https://identity.nexuscore.com/oauth2/v1/authorize",
  "token_endpoint": "https://identity.nexuscore.com/oauth2/v1/token",
  "userinfo_endpoint": "https://identity.nexuscore.com/oauth2/v1/userinfo",
  "jwks_uri": "https://identity.nexuscore.com/oauth2/v1/certs",
  "response_types_supported": ["code"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"],
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "claims_supported": ["sub", "iss", "aud", "exp", "iat", "email", "tenant_id", "org_id", "role"]
}
```

---

## 6. Token & Key Management Strategies

Cryptographic key lifecycles, signature operations, and session persistence must withstand high concurrency while maintaining sub-millisecond response times.

### 6.1 JWT Strategy (AuthN Token)
JWT tokens are transient credentials signed with an asymmetric **RS256 (RSA with SHA-256)** algorithm using 4096-bit private keys.
-   **Validity Duration:** 15 minutes.
-   **Token Schema:**
```json
{
  "protected_header": {
    "alg": "RS256",
    "typ": "JWT",
    "kid": "key_rotation_v2_2026_q3"
  },
  "payload": {
    "iss": "https://identity.nexuscore.com",
    "sub": "usr_99a8b7c6_e5f4_3d2c_1b0a_998877665544",
    "aud": "https://api.nexuscore.com",
    "exp": 1784400000,
    "nbf": 1784399100,
    "iat": 1784399100,
    "jti": "jti_ff99aa33_bc77_ef88_a122",
    "tenant_id": "ten_11223344_5566_7788_9900",
    "org_id": "org_aabbccdd_eeff_1122_3344",
    "role": "SYSTEM_ADMIN",
    "permissions": ["billing:write", "users:read", "infra:configure"]
  }
}
```

### 6.2 Refresh Token Strategy
Refresh tokens allow clients to silently renew expired access tokens without re-entering credentials:
-   **Validity Duration:** 30 days.
-   **Sliding Window:** Every token refresh operation returns a brand-new Refresh Token, invalidating the old one.
-   **Token Reuse Protection (Replay Prevention):** Refresh tokens are stored with their cryptographic hashes in Redis. If a revoked or already-used refresh token is submitted, the system assumes a replay attack has occurred, immediately marks the user's entire token tree as compromised, and invalidates all active sessions for that user.

### 6.3 API Key Strategy
External automated services interface via API keys:
-   **Format:** `nx_live_[cryptographically_secure_random_64_characters]`
-   **Storage:** Keys are stored as SHA-256 hashes in PostgreSQL to prevent internal key leak exploitation.
-   **Verification:** The gateway parses the key, computes `SHA256(submitted_key)`, matches it with the database, and loads the mapped client configurations into the context headers.

### 6.4 Session Strategy
For stateful, high-security operations (like admin control panels), state is stored inside a highly available **Redis cluster**:
-   **Format:** Mapped UUID session records storing user metadata and active device fingerprint variables.
-   **Maximum Concurrent Sessions:** Enforced at five concurrent active sessions per user account. Subsequent logins trigger either a termination of the oldest active session or block the login flow based on user security configuration.

---

## 7. Security Policies & Hardening

Security policies are constructed defensively, assuming that physical infrastructure is subject to penetration and compromised credentials.

### 7.1 Enterprise Password Policy
-   **Minimum Length:** 14 characters.
-   **Complexity Rules:** Must contain at least one uppercase letter, one lowercase letter, one numeric digit, and one complex special character (`!@#$%^&*()_+-=[]{}|;':",./<>?`).
-   **Password History Limit:** Blocks reuse of any of the last twelve (12) previously used hashes.
-   **Breached Password Check:** Integrates with local HaveIBeenPwned API lookups during mutation requests to block compromised values.
-   **Cool-Down Lockout:** Five failed login attempts lock the account for 30 minutes, dynamically compounding to 24 hours on subsequent breaches.

### 7.2 MFA Flow (Multi-Factor Authentication)
MFA is highly recommended for all users and mandatory for administrators.
-   **TOTP (Time-Based One-Time Password):** Conforms to **RFC 6238** using standard SHA-1 with a 30-second step and a 6-digit output length.
-   **Backup Codes:** Creates twelve (12) single-use cryptographically random recovery codes (8-character alphanumeric string) on registration.
-   **Hardware Keys (FIDO2 / WebAuthn):** Employs hardware-bound key tokens validating digital signatures directly inside client browsers.

### 7.3 Device Trust
-   **Fingerprinting:** Combines client platform characteristics (browser headers, screen width, WebGL properties) into a persistent fingerprint hash.
-   **Location Anomaly Detection:** Triggers elevated MFA verification if the geographic distance between consecutive logins implies physical travel speeds exceeding Mach 1 ($1234.8$ km/h).
-   **Device Pinning:** Users receive notification emails whenever a session initializes from a previously unmapped platform environment.

---

## 8. Audit & Compliance Flow

All authorization checks, cryptographic updates, status alterations, and login requests must produce structural, immutable logging records to comply with SOC2, ISO 27001, and HIPAA protocols.

### 8.1 Audit Trail Event Schema
Audit logs are compiled using structural JSON formatting and transmitted directly to downstream security scanning services (SIEM / Elastic):

```json
{
  "timestamp": "2026-07-18T23:59:59.123Z",
  "event_id": "evt_aa00bb11_cc22_dd33_ee44_ff5566778899",
  "event_type": "USER_AUTHENTICATION_SUCCESS",
  "tenant_id": "ten_11223344_5566_7788_9900",
  "actor": {
    "user_id": "usr_99a8b7c6_e5f4_3d2c_1b0a_998877665544",
    "email": "operator@nexuscore.com",
    "role": "SYSTEM_ADMIN"
  },
  "action": "LOGIN",
  "status": "SUCCESS",
  "request_metadata": {
    "client_ip": "198.51.100.42",
    "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
    "device_fingerprint": "fp_ff33aa99cc77"
  },
  "compliance_tags": ["SOC2", "GDPR", "HIPAA"],
  "cryptographic_signature": "sig_00eeff443322aa..."
}
```

### 8.2 Tamper-Detection Chains
To guarantee that records cannot be altered or removed by bad actors with root access to the database, audit table files utilize a **cryptographic hash chain**. Every new log entry maps $H_n = \text{SHA-256}(H_{n-1} \mathbin{\Vert} \text{Record}_n)$. A scheduled daily job validates the integrity of the chain against independent verification anchors stored in secure write-once-read-many (WORM) vaults.

---

### 🏆 Sprint 3 Architectural Review Board (ARB) Sign-Off

The **Identity Platform Architecture** has been designed to meet and exceed enterprise-grade security mandates. The architecture enforces multi-tenancy separation, integrates OPA authorization models, defines comprehensive lifecycles, and implements secure authentication protocols.

*NexusCore Architecture Board*  
*Status: Approved and Logged*  
*Awaiting Prompt 32 for Implementation phase.*

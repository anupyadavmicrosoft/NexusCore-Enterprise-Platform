# NexusCore Security Policy

We take the security of our enterprise distributed platform and data privacy seriously. This document outlines the security procedures, reporting protocols, cryptographic guidelines, and defensive practices applied across the NexusCore codebase.

---

## 1. Supported Versions

Security updates are actively applied to the following platform releases:

| Version | Supported | Notes |
| :--- | :---: | :--- |
| **v1.X.X** | Yes | Active Development / Production |
| **v0.Y.Y** | No | Ephemeral Sandbox / Prototypes |

---

## 2. Reporting a Vulnerability

If you discover a security vulnerability, **do not open a public GitHub issue**. Instead, report it through our private security channels to allow for coordinated disclosure and hotfix deployment.

### 2.1 Reporting Steps:
1.  Send a detailed email containing the exploit vector, reproducer script, and affected service components to: **security@nexuscore-enterprise.com**.
2.  Provide any relevant details about your environment, OS, and hardware.
3.  We will acknowledge your report within **24 hours** and supply a tracking ID.
4.  A private hotfix branch will be created to resolve the vulnerability.
5.  A coordinated disclosure patch and release note will be announced once the hotfix is successfully deployed into all production clusters.

---

## 3. Core Security Rules & Defensive Coding

Every engineer contributing to NexusCore must abide by the following secure development guidelines:

### 3.1 Zero Hardcoded Secrets
*   **Absolutely no credentials, API keys, certificates, or tokens may be hardcoded into the source code.**
*   Secrets must be fetched at runtime from secure environment variables or integrated key managers (such as Google Secret Manager).
*   Always define required local variables inside `.env.example` as a placeholder without providing active production secrets.

### 3.2 Password Hashing
*   All user credentials must be stored using the **Argon2id** key derivation function.
*   Bcrypt is permitted only for older auxiliary systems under strict isolated legacy constraints, but Argon2id (m=65536, t=3, p=4) remains the mandatory baseline for primary identities.

### 3.3 Data Encryption (PII Protection)
*   Any Personally Identifiable Information (PII) written to persistent databases (PostgreSQL) must be encrypted at rest.
*   Use authenticated symmetric encryption schemes: **AES-256-GCM** is the standard.
*   Generate unique cryptographic nonces for every single encryption operation; never reuse nonces across records.

### 3.4 API Security (AuthN & AuthZ)
*   All non-public routes on the API Gateway must validate asymmetric signed tokens (**RS256 JWT** bearer tokens).
*   Role-Based Access Control (RBAC) middleware must actively assert required role claims (e.g., `Admin`, `Operator`) at the handler entrance.
*   API gateways must actively enforce token-bucket rate limits and sliding-window limits to prevent distributed denial of service (DDoS) and brute-force attempts.

---

## 4. Continuous Security Audits (DevSecOps)

Our CI/CD pipelines automate multiple defensive scans on every Pull Request:
*   **Static Application Security Testing (SAST)**: We execute `gosec` and `semgrep` on all Go modules to identify common coding pitfalls, SQL injection vectors, and weak cipher definitions.
*   **Software Composition Analysis (SCA)**: We verify third-party libraries against the National Vulnerability Database (NVD) to intercept known vulnerability imports.
*   **Container Vulnerability Scanning**: Production scratch and alpine images are dynamically audited via `trivy` before being pushed to container registries.

# shared-jwt-library
## Enterprise Cryptographic JWT Signing & RS256 Verification Library

A zero-dependency, compiled Go 1.22 library designed for asymmetric token generation, signing, and verification.

---

## 📦 Features

- **Asymmetric Signature Enforcer:** Restricts and verifies signatures using standard **RS256 (RSA with SHA-256)** 4096-bit parameters.
- **Strict Claims Validation:** Automatically enforces standard OIDC and custom tenant claims (`tenant_id`, `org_id`, `role`, `permissions`).
- **Cryptographic Expiration Guards:** Full protection against expired tokens and premature evaluation requests.

## ⚙️ Quick Start

```go
import "github.com/nexuscore/identity-platform/shared-jwt-library"

// 1. Generate keys
privKey, pubKey, err := jwt.GenerateRSAKeyPair()

// 2. Issue Token
claims := jwt.Claims{
    Issuer: "nexuscore",
    Subject: "user_123",
    Expiry: time.Now().Add(15 * time.Minute).Unix(),
    Role: "SYSTEM_ADMIN",
}
token, err := jwt.SignTokenRS256(claims, privKey, "key_v1")

// 3. Verify Token
verifiedClaims, err := jwt.VerifyTokenRS256(token, pubKey)
```

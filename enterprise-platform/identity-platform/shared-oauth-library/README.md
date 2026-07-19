# shared-oauth-library
## RFC 7636 PKCE & OAuth2 Grant State Orchestrator Library

A high-performance library providing cryptographic primitives for PKCE validation and OAuth 2.0 structures.

---

## 📦 Features

- **Standard PKCE Generation:** Automated high-entropy verifiers and SHA-256 S256 challenge generation (RFC 7636).
- **Constant-Time Verification:** Protects against timing attacks during code exchange operations.
- **Unified Struct Layouts:** Standardizes serialization schemas for Token and Code Exchange payloads.

## ⚙️ Quick Start

```go
import "github.com/nexuscore/identity-platform/shared-oauth-library"

// 1. Generate verifier and challenge
pair, err := oauth.GeneratePKCEPair()

// 2. Validate incoming verifier against stored challenge
ok, err := oauth.VerifyPKCE(pair.Verifier, pair.Challenge)
```

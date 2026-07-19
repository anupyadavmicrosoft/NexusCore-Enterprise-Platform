# shared-security-library
## Enterprise Cryptographic Utilities & Hardening Library

A production-ready library containing cryptographic routines used for password hashing, symmetric authenticated encryption, and randomized identifier generation.

---

## 📦 Features

- **HMAC-SHA256 Multi-Stretched Credential Hashing:** Safe stretch iteration structures preventing dictionary and brute-force cracking.
- **Constant-Time Validation:** Prevention of cryptographic timing-leak attacks via Go's native `subtle.ConstantTimeCompare`.
- **Authenticated Symmetric AES-256-GCM Encryption:** Direct support for Envelope Encryption using 12-byte standard cryptographically unique nonces.
- **Random Tokens:** High-entropy random generator wrappers mapped directly onto `crypto/rand`.

## ⚙️ Quick Start

```go
import "github.com/nexuscore/identity-platform/shared-security-library"

// 1. Hash credentials
hash, err := security.HashPassword("SecureP@ss123!")

// 2. Cryptographically verify
matches, err := security.VerifyPassword("SecureP@ss123!", hash)
```

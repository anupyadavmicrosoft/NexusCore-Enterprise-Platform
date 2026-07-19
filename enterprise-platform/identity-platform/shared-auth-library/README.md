# shared-auth-library
## Context Security Injection, Interceptors & Middleware Library

A highly performant package supporting consistent context parsing, metadata retrieval, and authorization checking across HTTP REST routers and gRPC stream listeners.

---

## 📦 Features

- **Context Scanners:** Safely injects and extracts typed identification claims onto Go’s native request context variables.
- **Micro-Authorization Checks:** Direct support for wildcard permissions comparisons (`*` or prefix match like `tenant:org:*`).
- **Standard Security Errors:** Emits predictable canonical HTTP/gRPC security exception blocks.

## ⚙️ Quick Start

```go
import "github.com/nexuscore/identity-platform/shared-auth-library"

// 1. Map context state
actx := &auth.AuthContext{
    TenantID: "tenant_abc",
    UserID: "user_777",
    Permissions: []string{"billing:*"},
}
ctx := auth.InjectIntoContext(context.Background(), actx)

// 2. Evaluate check
hasAccess := auth.CheckPermission(actx.Permissions, "billing:write") // Returns true
```

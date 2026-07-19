# NexusCore Identity Platform Monorepo
## Production-Ready Enterprise Identity & Access Management (IAM) Suite

This directory contains the complete, enterprise-grade, highly decoupled repository structure of the **NexusCore Identity Platform**. 

Built with Go 1.22, Docker Multi-Stage Distroless images, Kubernetes Pod Security Standards (Restricted), Helm templates, and zero-trust security designs, this ecosystem is the cryptographic security foundation of the entire NexusCore architecture.

---

## 🗂️ Monorepo Structure

The platform is structured as an advanced, decoupled microservices monorepo with dedicated domain services and optimized shared libraries:

```
identity-platform/
├── go.work                         # Go workspace orchestrating all sub-modules
├── README.md                       # Comprehensive monorepo governance guide
│
├── [Services]
│   ├── identity-service/          # OIDC Identity Provider & centralized console
│   ├── auth-service/              # Core authentication & session orchestrator
│   ├── authorization-service/     # Open Policy Agent (OPA) integration & ABAC/RBAC engine
│   ├── user-service/              # Enterprise profile, credential, and lifecycle manager
│   ├── tenant-service/            # SaaS multi-tenant isolation & onboarding manager
│   └── organization-service/      # Hierarchical business unit (ltree) organizer
│
└── [Shared Libraries]
    ├── shared-auth-library/       # gRPC & HTTP security interceptors & middlewares
    ├── shared-security-library/   # Cryptographic helper (Argon2id, AES-256-GCM, rate limits)
    ├── shared-jwt-library/        # RS256 sign/verify engine & JWKS keystore manager
    └── shared-oauth-library/      # RFC 7636 PKCE & Oauth2 grant state orchestrators
```

---

## 🚀 Architectural Principles

1. **Decoupled Architecture:** Each service owns its database bounds, separating concerns between profile management, policy evaluation, tenant lifecycle, and active authentication.
2. **Zero Trust & Compliance:** Complete end-to-end mTLS bindings, secure Argon2id credential hashing, strict OPA Rego sidecars, and immutable tamper-evident audit trails.
3. **Resiliency:** Guaranteed QoS class (requests equal limits), PodDisruptionBudgets (PDB) to preserve active quorum, and Horizontal Pod Autoscaling (HPA) to dynamically scale with demand spikes.

---

## 🛠️ Global Development Guide

Initialize the workspace on your localized terminal:
```bash
go work init
go work use ./identity-service ./auth-service ./authorization-service ./user-service ./tenant-service ./organization-service ./shared-auth-library ./shared-security-library ./shared-jwt-library ./shared-oauth-library
```

*NexusCore Platform Engineering Team - UTC Coordinated Clock*

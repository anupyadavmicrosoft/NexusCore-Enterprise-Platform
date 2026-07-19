# tenant-service
## NexusCore SaaS Tenant Isolation & Onboarding Manager

This microservice acts as the orchestrator of SaaS tenant isolation, provisioning separate Postgres schemas, dedicated storage, and cryptographic erasure operations ("crypto-shredding").

---

## ⚙️ Configuration Variables

| Env Var | Default | Description |
| --- | --- | --- |
| `PORT` | `8084` | Inbound REST port for tenant management operations |

---

## 🛣️ API Routes Map

- `POST /tenants/provision` - Registers multi-tenant metadata and constructs separate schemas.
- `POST /tenants/shred` - Permanently erases tenant keys rendering all stored fields instantly unreadable.
- `GET /healthz` - Service health diagnostic.

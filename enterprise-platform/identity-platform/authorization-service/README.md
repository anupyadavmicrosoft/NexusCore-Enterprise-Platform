# authorization-service
## NexusCore RBAC & ABAC Engine

This microservice acts as the policy engine checking permission scopes, wildcards, and situational access contexts (ABAC/RBAC).

---

## ⚙️ Configuration Variables

| Env Var | Default | Description |
| --- | --- | --- |
| `PORT` | `8082` | Inbound REST port for active authorization checks |

---

## 🛣️ API Routes Map

- `POST /authz/evaluate` - Evaluates authorization parameters against RBAC/ABAC mappings.
- `GET /healthz` - Service health diagnostic.

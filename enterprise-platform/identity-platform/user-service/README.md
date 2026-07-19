# user-service
## NexusCore User Profile & Lifecycle Service

This microservice acts as the single source of truth for user profiles, status lifecycles, account locks, and credentials parameters matching enterprise password complexity targets.

---

## ⚙️ Configuration Variables

| Env Var | Default | Description |
| --- | --- | --- |
| `PORT` | `8083` | Inbound REST port for user profiles management |

---

## 🛣️ API Routes Map

- `POST /users/create` - Creates pending accounts running strong credentials stretching validations.
- `POST /users/lifecycle` - Orchestrates state transition overrides (e.g. locks, suspensions, gdpr archivals).
- `GET /healthz` - Service health diagnostic.

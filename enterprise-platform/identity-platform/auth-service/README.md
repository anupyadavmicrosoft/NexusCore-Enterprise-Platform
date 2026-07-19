# auth-service
## NexusCore Authentication & Session Orchestrator

This microservice handles active authentication flows, session state tracking, credentials verification, and MFA checks.

---

## ⚙️ Configuration Variables

| Env Var | Default | Description |
| --- | --- | --- |
| `PORT` | `8081` | Inbound REST port for active authentication |

---

## 🛣️ API Routes Map

- `POST /auth/login` - Primary authentication route verifying stretched hashes.
- `POST /auth/mfa/verify` - Verifies second-factor challenges.
- `GET /healthz` - SRE probe endpoint.

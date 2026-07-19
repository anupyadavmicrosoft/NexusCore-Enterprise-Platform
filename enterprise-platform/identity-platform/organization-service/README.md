# organization-service
## NexusCore Hierarchical Organization Service

This microservice organizes business units dynamically, managing hierarchical tree structures inside a single tenant using PostgreSQL ltree data maps.

---

## ⚙️ Configuration Variables

| Env Var | Default | Description |
| --- | --- | --- |
| `PORT` | `8085` | Inbound REST port for active organization structures |

---

## 🛣️ API Routes Map

- `POST /orgs/create` - Appends a hierarchical node to the tenant organization ltree.
- `GET /orgs/tree` - Queries and returns the full nested organization tree.
- `GET /healthz` - Service health diagnostic.

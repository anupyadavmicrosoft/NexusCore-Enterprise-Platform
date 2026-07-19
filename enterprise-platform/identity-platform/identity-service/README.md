# identity-service
## NexusCore Central OIDC Identity Provider

This microservice acts as the centralized OpenID Connect (OIDC) identity provider and OAuth 2.0 authorization server.

---

## ⚙️ Configuration Variables

| Env Var | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | Network port for standard HTTP REST ingress traffic |

---

## 🛣️ API Routes Map

- `GET /.well-known/openid-configuration` - Exposes structural configuration metadata.
- `GET /oauth2/v1/certs` - Exposes public keys (JWKS format).
- `GET /oauth2/v1/authorize` - Intercepts authorizations with S256 PKCE challenges.
- `POST /oauth2/v1/token` - Authenticates authorization codes returning signed RS256 access tokens.
- `GET /healthz` - Liveness/Readiness SRE endpoint.

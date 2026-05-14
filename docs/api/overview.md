# API Overview

Lower-level gateway API reference. Application SDK examples should use `issue`, `pay`, and `verify` where available: issue a scoped capability, invoke upstream through `pay` with a max budget, then verify the receipt.


SatGate exposes these OSS/gateway compatibility API groups on port **8080** (same port as the proxy). They are useful for local gateways and lower-level integrations; app teams should prefer issue/pay/verify when available:

## Developer-first paths

Most application integrations should start with the developer voice rather than the low-level gateway API:

- [Quickstart](../getting-started/quickstart.md)
- [Raw HTTP issue/pay/verify](../guides/raw-http.md)
- [MCP integration](../guides/mcp-gateway.md)
- [Receipt schema](../reference/receipt-schema.md)
- [Capability schema](../reference/capability-schema.md)

## Capability APIs (`/api/capability/`)

Token management and validation endpoints. Require `ADMIN_TOKEN` for minting.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/capability/mint` | Mint a new capability token (admin only) |
| `POST` | `/api/capability/validate` | Validate a capability token |
| `POST` | `/api/capability/delegate` | Delegate a token with additional caveats (reduced permissions, tighter budget) |
| `GET` | `/api/capability/ping` | Validate token and return success (health check with auth) |
| `GET` | `/api/capability/admin` | Admin-scoped endpoint (requires admin scope) |

## Governance APIs (`/api/governance/`)

Token lifecycle and observability endpoints.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/governance/ban` | Ban/revoke a token |
| `GET` | `/api/governance/graph` | Token lineage graph (for dashboard visualization) |
| `POST` | `/api/governance/reset` | Reset dashboard/governance data |
| `GET` | `/api/governance/evidence-pack` | Export a canonical Policy-to-Proof Evidence Pack v1 |

## Authentication

- **Admin endpoints** (mint, ban, reset): Require `Authorization: Bearer <ADMIN_TOKEN>`
- **Evidence Pack export**: Require tenant context (`X-Tenant-ID`) and a tenant API key or admin credential on protected deployments
- **Token endpoints** (validate, delegate, ping): Require a valid capability token (macaroon)
- **Proxied routes**: Authentication depends on the route's policy (public, capability, or L402)

## Example: Mint a Token

```bash
curl -X POST http://localhost:8080/api/capability/mint \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope": "read", "budget": 1000}'
```

## Example: Delegate a Token

```bash
curl -X POST http://localhost:8080/api/capability/delegate \
  -H "Authorization: Bearer <parent-token>" \
  -H "Content-Type: application/json" \
  -d '{"caveats": [{"type": "budget", "limit": 100}, {"type": "expiry", "ttl": "24h"}]}'
```

The delegated token has strictly fewer permissions than its parent — this is enforced cryptographically by the macaroon structure.

# API Reference

SatGate Gateway exposes two APIs:

- **Data Plane** (`:8080`) — Public API traffic
- **Admin Plane** (`:9090`) — Management and configuration

## Authentication

### Admin Token

For admin API access, use the `X-Admin-Token` header:

```bash
curl -H "X-Admin-Token: your-admin-token" \
  http://localhost:9090/api/v1/tokens
```

### JWT Sessions

For user-based access, obtain a JWT via login:

```bash
# Login
curl -X POST http://localhost:9090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "..."}'

# Use JWT
curl -H "Authorization: Bearer eyJ..." \
  http://localhost:9090/api/v1/tokens
```

## Admin API Endpoints

### Health & System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe |
| GET | `/metrics` | Prometheus metrics |
| GET | `/api/v1/system/preflight` | Enterprise readiness checks |

### Token Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/tokens` | Mint new token |
| GET | `/api/v1/tokens` | List all tokens |
| GET | `/api/v1/tokens/{signature}` | Get token details |
| DELETE | `/api/v1/tokens/{signature}` | Revoke token |
| POST | `/api/v1/tokens/{signature}/delegate` | Delegate token |

### Governance

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/governance/ban` | Ban token (kill switch) |
| DELETE | `/api/v1/governance/ban/{signature}` | Unban token |
| GET | `/api/v1/governance/banlist` | List banned tokens |
| GET | `/api/v1/governance/graph` | Token lineage graph |

### Tenant Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/tenants` | Create tenant (idempotent) |
| GET | `/api/v1/tenants` | List tenants |
| GET | `/api/v1/tenants/{id}` | Get tenant |
| PUT | `/api/v1/tenants/{id}` | Update tenant |
| DELETE | `/api/v1/tenants/{id}` | Delete tenant |
| POST | `/api/v1/tenants/{id}/domains` | Add domain |
| DELETE | `/api/v1/tenants/{id}/domains/{domain}` | Remove domain |

### Configuration

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/config` | Get current config |
| PUT | `/api/v1/config` | Update config |
| POST | `/api/v1/config/validate` | Validate config |

### Statistics

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/stats` | Gateway statistics |
| GET | `/api/v1/stats/routes` | Per-route statistics |
| GET | `/api/v1/stats/tokens` | Token statistics |

## Common Requests

### Mint Token

```bash
curl -X POST http://localhost:9090/api/v1/tokens \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "api:read",
    "expiresIn": 3600,
    "metadata": {
      "user": "agent-1"
    }
  }'
```

Response:
```json
{
  "token": "eyJ2IjoxLCJsIjoiaHR0cHM6Ly9zYXRnYXRlLmlvIi...",
  "signature": "abc123def456",
  "scope": "api:read",
  "expiresAt": "2024-01-01T12:00:00Z"
}
```

### Delegate Token

```bash
curl -X POST http://localhost:9090/api/v1/tokens/abc123def456/delegate \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "caveats": [
      {"type": "expires", "value": "1h"},
      {"type": "rate_limit", "value": "10/minute"}
    ]
  }'
```

### Ban Token

```bash
curl -X POST http://localhost:9090/api/v1/governance/ban \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "signature": "abc123def456",
    "reason": "Compromised"
  }'
```

### Create Tenant

```bash
curl -X POST http://localhost:9090/api/v1/tenants \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "acme-corp",
    "name": "ACME Corporation",
    "quotas": {
      "requestsPerDay": 100000,
      "tokensMax": 1000
    }
  }'
```

## Error Responses

All errors follow this format:

```json
{
  "error": "Description of the error",
  "code": "ERROR_CODE"
}
```

### Common Error Codes

| Code | Status | Description |
|------|--------|-------------|
| `UNAUTHORIZED` | 401 | Missing or invalid token |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `RATE_LIMITED` | 429 | Rate limit exceeded |
| `INTERNAL_ERROR` | 500 | Server error |

## Rate Limiting

Admin API is rate limited (default: 60 req/min per IP).

Headers:
```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1704067200
```

---

## Enterprise API Endpoints (Cloud/Enterprise Only)

> ⚠️ The following endpoints are only available in SatGate Cloud or Enterprise deployments.
> They require multi-tenant context, admin tokens, or cloud sessions.

### Delegation v2 (Economic Firewall)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/cloud/delegation-v2/tokens` | Create root or child token |
| GET | `/cloud/delegation-v2/tokens` | List delegation tokens |
| GET | `/cloud/delegation-v2/tokens/{id}` | Get token details + budget |
| POST | `/cloud/delegation-v2/tokens/{id}/revoke` | Cascade revoke token + descendants |
| GET | `/cloud/delegation-v2/tree` | Get delegation tree hierarchy |
| GET | `/cloud/delegation-v2/tokens/{id}/spend` | Get spend history for token |

### Budget Enforcement

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/cloud/delegation-v2/alerts` | List budget alerts |
| POST | `/cloud/delegation-v2/alerts/{id}/acknowledge` | Acknowledge single alert |
| POST | `/cloud/delegation-v2/tokens/{id}/alerts/acknowledge-all` | Acknowledge all token alerts |
| GET | `/cloud/delegation-v2/export` | Export spend data (JSON/CSV) |

### Settlement (Admin-Only)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/cloud/delegation-v2/tokens/{id}/settle` | Add credits to token (admin) |
| GET | `/cloud/delegation-v2/tokens/{id}/settlements` | Get settlement history |

### Team Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/cloud/team` | List team members |
| GET | `/cloud/team/members/{userId}` | Get member details |
| PUT | `/cloud/team/members/{userId}/role` | Change member role |
| DELETE | `/cloud/team/members/{userId}` | Remove team member |
| GET | `/cloud/team/invites` | List pending invites |
| POST | `/cloud/team/invites` | Create team invite |
| DELETE | `/cloud/team/invites/{inviteId}` | Cancel invite |
| POST | `/cloud/team/accept-invite` | Accept invite (invitee) |

### Support System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/cloud/support` | List customer tickets |
| POST | `/api/cloud/support` | Create support ticket |
| GET | `/api/cloud/support/{ticketId}` | Get ticket details |
| PUT | `/api/cloud/support/{ticketId}` | Update ticket status |
| DELETE | `/api/cloud/support/{ticketId}` | Delete/close ticket |
| GET | `/api/cloud/support/{ticketId}/messages` | Get message thread |
| POST | `/api/cloud/support/{ticketId}/messages` | Add message to ticket |

### Admin Impersonation

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/admin/impersonate` | Start impersonation session |
| DELETE | `/api/admin/impersonate` | End impersonation session |
| GET | `/api/admin/impersonate` | Get impersonation status |

### Security & Audit (Admin)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/cloud/audit` | Get audit events (with filters) |
| GET | `/admin/security` | Security dashboard data |

---

## Enterprise Request Headers

| Header | Description | Required |
|--------|-------------|----------|
| `X-Tenant-ID` | Target tenant (admin cross-tenant ops) | Admin only |
| `X-Request-ID` | Idempotency key for budget spend | Recommended |
| `Idempotency-Key` | Alternative idempotency key | Recommended |
| `X-Credit-Token` | Delegation token for L402→Credits | L402 settlement |
| `X-Budget-Remaining` | (Response) Remaining budget | Read-only |
| `X-Credit-Settlement` | (Response) Settlement status | Read-only |
| `X-Credits-Added` | (Response) Credits added | Read-only |

---

## OpenAPI Specification

Full OpenAPI spec available at:

```
GET /api/docs/openapi.yaml
```

Swagger UI:
```
GET /api/docs
```




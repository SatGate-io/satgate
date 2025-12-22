# SatGate Security Model

This document describes the security architecture and access control model for SatGate.

## Overview

SatGate implements a **Zero Trust Policy Enforcement Point (PEP)** for API traffic. It enforces per-request authorization at the edge with no implicit network trust.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          TRUST BOUNDARY                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   [Internet]                                                            │
│       │                                                                 │
│       ▼                                                                 │
│   ┌─────────┐    ┌───────────┐    ┌─────────────┐    ┌──────────────┐  │
│   │ CDN/WAF │───▶│ Aperture  │───▶│  SatGate    │───▶│ Your API     │  │
│   │         │    │  (L402)   │    │  (Backend)  │    │  (Origin)    │  │
│   └─────────┘    └───────────┘    └─────────────┘    └──────────────┘  │
│                        │                │                              │
│                        │                │                              │
│              Lightning Payment    Capability Token                     │
│              Verification         Verification                         │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## Endpoint Classification

### PUBLIC Endpoints (No Authentication Required)

These endpoints are intentionally public:

| Endpoint | Purpose | Rate Limited |
|----------|---------|--------------|
| `GET /health` | Health check for load balancers | No |
| `GET /ready` | Readiness probe for k8s | No |
| `GET /api/free/*` | Free tier API endpoints | Yes (300/min) |
| `GET /api/governance/graph` | Public telemetry (read-only) | Yes (300/min) |
| `GET /api/governance/stats` | Public stats summary | Yes (300/min) |
| `GET /dashboard` | Governance dashboard UI | Optional |

**Note:** The dashboard can be protected by setting `DASHBOARD_PUBLIC=false`.

### ADMIN Endpoints (Authentication Required)

These endpoints require the `X-Admin-Token` header matching `PRICING_ADMIN_TOKEN`:

| Endpoint | Purpose | Rate Limited | Audit Logged |
|----------|---------|--------------|--------------|
| `POST /api/governance/ban` | Ban a token (kill switch) | Yes (30/min) | ✅ |
| `POST /api/governance/unban` | Unban a token | Yes (30/min) | ✅ |
| `GET /api/governance/banned` | List banned tokens | Yes (30/min) | ✅ |
| `GET /api/governance/audit` | View audit log | Yes (30/min) | No |
| `POST /api/free/pricing` | Update display pricing | Yes (30/min) | No |
| `PUT /api/free/pricing` | Bulk update pricing | Yes (30/min) | No |

**Authentication:**
```bash
curl -X POST https://your-api.com/api/governance/ban \
  -H "X-Admin-Token: your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"tokenSignature": "abc123...", "reason": "Compromised"}'
```

### L402 PROTECTED Endpoints (Lightning Payment Required)

These endpoints are protected by Aperture and require L402 payment:

| Endpoint Pattern | Price | Timeout |
|------------------|-------|---------|
| `/api/micro/*` | 1 sat | 24 hours |
| `/api/basic/*` | 10 sats | 24 hours |
| `/api/standard/*` | 100 sats | 24 hours |
| `/api/premium/*` | 1000 sats | 24 hours |

### CAPABILITY Endpoints (Macaroon Required)

These endpoints require a valid capability macaroon (no payment):

| Endpoint Pattern | Auth |
|------------------|------|
| `/api/capability/*` | Macaroon signature verification |

## Security Controls

### 1. Rate Limiting

All endpoints have rate limits to prevent abuse:

- **Admin endpoints:** 30 requests/minute per IP (configurable via `ADMIN_RATE_LIMIT`)
- **API endpoints:** 300 requests/minute per IP
- **L402 endpoints:** Implicitly rate-limited by payment cost (economic firewall)

Rate limit headers returned:
```
X-RateLimit-Limit: 30
X-RateLimit-Remaining: 29
X-RateLimit-Reset: 1703289600
```

### 2. Audit Logging

All admin actions are logged with:
- Timestamp (ISO 8601)
- Action type (BAN_SUCCESS, UNBAN_SUCCESS, etc.)
- Details (token signature, reason)
- Client IP
- User-Agent

Logs are stored in Redis (if available) and in-memory (last 1000 entries).

### 3. Input Validation

Admin endpoints validate:
- Token signatures must be valid hex strings
- Reasons are sanitized and logged
- Missing required fields return 400 errors

### 4. Token Revocation (Kill Switch)

Stateless tokens + stateful blocklist:

```
┌─────────────────────────────────────────────────────────────┐
│                    TOKEN VALIDATION                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Cryptographic Validation (Stateless)                    │
│     - Verify signature                                       │
│     - Check expiry caveat                                    │
│     - Validate scope constraints                             │
│                                                             │
│  2. Blocklist Check (Stateful)                              │
│     - Check if signature in bannedTokens set                 │
│     - O(1) lookup via Redis SET                              │
│                                                             │
│  Result: ALLOW | DENY                                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

This provides:
- **Speed:** Cryptographic validation is fast (no DB lookup for valid tokens)
- **Revocation:** Compromised tokens can be instantly banned
- **Scalability:** Blocklist is small (only banned tokens, not all tokens)

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `PRICING_ADMIN_TOKEN` | Secret for admin endpoints | Yes (prod) | None |
| `DASHBOARD_PUBLIC` | Allow public dashboard access | No | true |
| `ADMIN_RATE_LIMIT` | Admin requests per minute | No | 30 |
| `REDIS_URL` | Redis connection for persistence | No | In-memory |

## Deployment Security

### Recommended Architecture

```
                    Internet
                       │
                       ▼
                 ┌───────────┐
                 │  CDN/WAF  │  ← DDoS protection, TLS termination
                 └─────┬─────┘
                       │
                       ▼
                 ┌───────────┐
                 │ Aperture  │  ← L402 payment verification
                 └─────┬─────┘
                       │
                       ▼
                 ┌───────────┐
                 │ SatGate   │  ← Capability validation, governance
                 └─────┬─────┘
                       │
                       ▼
                 ┌───────────┐
                 │ Your API  │  ← Business logic
                 └───────────┘
```

### Security Checklist

- [ ] Set `PRICING_ADMIN_TOKEN` to a strong random value
- [ ] Use HTTPS in production (TLS at CDN/edge)
- [ ] Configure `DASHBOARD_PUBLIC=false` if dashboard should be admin-only
- [ ] Set up Redis for persistent blocklist across restarts
- [ ] Monitor `/api/governance/audit` for suspicious activity
- [ ] Rotate admin token periodically

### Compliance Notes

**SOC2 / ISO 27001:**
- ✅ Per-request authorization (no implicit trust)
- ✅ Audit logging for admin actions
- ✅ Token revocation capability
- ✅ Cryptographic chain of custody (macaroon lineage)

**For Auditors:**
Use the Governance Inspector to demonstrate token provenance:
```bash
node cli/inspect.js <token>
```

## Threat Model

| Threat | Mitigation |
|--------|------------|
| DDoS attacks | Economic firewall (L402 payment required) |
| Credential stuffing | Micropayment per attempt bankrupts attacker |
| Stolen tokens | Kill switch + short expiry caveats |
| Admin endpoint abuse | Auth + rate limiting + audit logging |
| Replay attacks | Macaroon expiry caveats |
| Token forgery | Cryptographic signatures |

## Contact

For security issues, contact: [security@satgate.io]


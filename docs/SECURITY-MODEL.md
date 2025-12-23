# SatGate Security Model

This document describes the security architecture and access control model for SatGate.

## Overview

SatGate implements a **Zero Trust Policy Enforcement Point (PEP)** for API traffic:

- **Per-request authorization** at the edge
- **No implicit trust** from network location
- **Stateless capability verification** (macaroons) with least-privilege caveats
- **Optional L402 payment challenges** (HTTP 402) to add economic friction against abuse

## Trust Boundary

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

Policy enforcement occurs at SatGate (capabilities) and/or Aperture (payments). Your API origin should assume requests are already authenticated/authorized per the chosen policy.

## Roles and Access Model

SatGate separates the **data plane** (API traffic) from the **control plane** (governance/admin):

| Plane | Purpose | Default Access |
|-------|---------|----------------|
| Data Plane | API traffic handling | Per-endpoint policy |
| Control Plane | Governance, telemetry, admin | **Admin-only** |

---

## Endpoint Classification

### Public Endpoints (Minimal Data, Rate Limited)

These endpoints return minimal information and are safe to expose behind CDN/WAF.

| Endpoint | Purpose | Rate Limited |
|----------|---------|--------------|
| `GET /health` | Health check for load balancers | ✅ 600/min |
| `GET /ready` | Readiness probe for orchestrators | ✅ 600/min |
| `GET /api/free/*` | Free tier endpoints | ✅ 300/min |

> **Security Note:** Health endpoints return only status and uptime—no version, environment, or build details exposed.

### Admin Endpoints (Authentication Required)

Admin endpoints require the `X-Admin-Token` header. They should **never** be publicly accessible.

| Endpoint | Purpose | Rate Limited | Audit Logged |
|----------|---------|--------------|--------------|
| `POST /api/governance/ban` | Ban a token (kill switch) | ✅ 30/min | ✅ |
| `POST /api/governance/unban` | Remove ban | ✅ 30/min | ✅ |
| `GET /api/governance/banned` | List banned token signatures | ✅ 30/min | ✅ |
| `POST /api/governance/reset` | Reset telemetry/demo view | ✅ 30/min | ✅ |
| `GET /api/governance/audit` | View admin audit log | ✅ 30/min | ✅ |
| `POST /api/free/pricing` | Update display pricing | ✅ 30/min | ✅ |

### Governance Telemetry & Dashboard

**By default, these are admin-only.**

| Endpoint | Purpose | Default Access |
|----------|---------|----------------|
| `GET /api/governance/graph` | Telemetry graph data | Admin-only |
| `GET /api/governance/stats` | Summary metrics | Admin-only |
| `GET /dashboard` | Governance dashboard UI | Admin-only |

#### Demo Mode (Optional)

To support public demos, SatGate can enable demo mode, which:

- ✅ Redacts token identifiers (hash/truncate)
- ✅ Removes IP / user-agent from views
- ✅ Aggregates metrics (no per-request detail)
- ✅ Prevents sensitive admin actions where possible

**Usage:**
```bash
# For public demo environments only
MODE=demo
DASHBOARD_PUBLIC=true
```

> 🔒 **Guardrail:** In `MODE=prod` (the default), `DASHBOARD_PUBLIC=true` is **ignored** — the dashboard always requires admin authentication. This prevents accidental exposure if someone misconfigures environment variables. A startup warning is logged if this misconfiguration is detected.

---

## Protected Data Plane Endpoints

### L402-Protected Endpoints (Lightning Payment Required)

These routes are protected by Aperture using L402 payment challenges:

| Pattern | Price | Timeout |
|---------|-------|---------|
| `/api/micro/*` | 1 sat | 24 hours |
| `/api/basic/*` | 10 sats | 24 hours |
| `/api/standard/*` | 100 sats | 24 hours |
| `/api/premium/*` | 1000 sats | 24 hours |

### Capability-Protected Endpoints (Macaroon Required)

These routes require a valid capability macaroon (no payment):

| Pattern | Auth |
|---------|------|
| `/api/capability/*` | Macaroon signature verification + caveat enforcement |

---

## Admin Authentication

### Current (Demo / Early Production)

Admin endpoints require an `X-Admin-Token` header matching `PRICING_ADMIN_TOKEN`.

```bash
curl -X POST https://your-api.com/api/governance/ban \
  -H "X-Admin-Token: your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"tokenSignature":"abc123...","reason":"Compromised"}'
```

### Roadmap (Enterprise)

| Phase | Authentication Method |
|-------|----------------------|
| **Now** | `X-Admin-Token` (single shared secret) |
| **Next** | OIDC/OAuth login for dashboard + short-lived admin JWT |
| **Then** | RBAC roles (Viewer / Operator / Admin), MFA via IdP |
| **Later** | Audit export to SIEM (Splunk/Datadog/etc.) |

---

## Data Handling and Retention

SatGate is designed to **avoid centralized user identity**:

- ✅ No user accounts required for capability verification or L402 payments
- ✅ No PII is required for data plane authorization

**Operational metadata** may be collected for security and audit:

| Data Type | Contents | Redacted in Demo Mode |
|-----------|----------|----------------------|
| Admin audit logs | timestamp, action, reason, client IP, user-agent | ✅ IP/UA redacted |
| Telemetry | aggregated counters (paid, blocked, banned) | ✅ Token IDs truncated |

**Retention:**
- In-memory: last N events (implementation-defined)
- Redis (optional): persisted across restarts; apply TTL policies per environment

---

## Security Controls

### 1) Rate Limiting

Default limits (configurable via environment variables):

| Endpoint Type | Default Limit | Config Variable |
|---------------|---------------|-----------------|
| Admin endpoints | 30/min per IP | `ADMIN_RATE_LIMIT` |
| Health probes | 600/min per IP | `HEALTH_RATE_LIMIT` |
| Free/API endpoints | 300/min per IP | — |
| L402 endpoints | Payment provides economic friction | — |

Rate limit headers returned:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

### 2) Audit Logging

Admin actions are logged with:
- ISO timestamp
- Action type (`BAN_SUCCESS`, `UNBAN_SUCCESS`, `RESET`, etc.)
- Token signature (redacted in demo mode)
- Reason
- Client IP + user-agent (removed in demo mode)

### 3) Input Validation

Admin endpoints validate:
- Token signatures are valid hex strings
- Required fields present
- Reason sanitized/logged
- Malformed requests return `400`

### 4) Token Revocation (Kill Switch)

SatGate uses **stateless verification** for valid tokens plus a **stateful ban list** for compromised tokens:

```
┌─────────────────────────────────────────────────────────────┐
│                    TOKEN VALIDATION                          │
├─────────────────────────────────────────────────────────────┤
│  1) Cryptographic Validation (Stateless)                     │
│     - Verify signature against root key                      │
│     - Check expiry caveat                                    │
│     - Enforce scope/time/budget caveats                      │
│                                                             │
│  2) Blocklist Check (Stateful)                               │
│     - Check token signature against banned set               │
│     - O(1) lookup via Redis SET (if enabled)                 │
│                                                             │
│  Result: ALLOW | DENY                                        │
└─────────────────────────────────────────────────────────────┘
```

### 5) Replay and Token Theft Considerations

Macaroons are **bearer credentials**; if stolen, they can be replayed until they expire or are revoked.

**Mitigations:**
- Short TTL caveats
- Narrow scope caveats
- Budget/max-use caveats (where applicable)
- Kill switch / ban list
- TLS everywhere (CDN/edge termination and origin protection)

### 6) L402 / 402 Abuse Protections

Attackers may try to DoS invoice creation or create churn.

**Recommended protections:**
- Rate-limit 402/invoice issuance per IP and per route
- Enforce invoice expiry and minimum pricing
- Track unpaid challenges per IP and throttle noisy sources
- Keep payment verification behind CDN/WAF

---

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `MODE` | Operating mode: `prod` (secure) or `demo` (public demos) | No | `prod` |
| `PRICING_ADMIN_TOKEN` | Secret for admin endpoints (min 32 chars) | ✅ (prod) | None |
| `ADMIN_TOKEN_NEXT` | Secondary admin token for rotation | No | None |
| `DASHBOARD_PUBLIC` | Allow public dashboard access (only in `MODE=demo`) | No | `false` |
| `ADMIN_RATE_LIMIT` | Admin req/min per IP | No | `30` |
| `HEALTH_RATE_LIMIT` | Health check req/min per IP | No | `600` |
| `REDIS_URL` | Redis connection for persistence | No | in-memory |

> **Note:** `DASHBOARD_PUBLIC=true` is ignored when `MODE=prod` — see guardrail above.

---

## Deployment Security

### Recommended Architecture

```
Internet
  │
  ▼
CDN/WAF        ← DDoS protection, TLS termination, request filtering
  │
  ▼
Aperture       ← L402 payment verification (for paid routes)
  │
  ▼
SatGate        ← Capability validation, governance, kill switch
  │
  ▼
Your API       ← Business logic
```

### Recommendations

- ✅ TLS at the edge; protect origin access (allowlist only CDN/WAF IPs)
- ✅ Make governance/dashboard admin-only in production
- ✅ Enable Redis for durable ban list and audit logs
- ✅ Set strong, random `PRICING_ADMIN_TOKEN`
- ✅ Rotate admin credentials periodically

---

## Security Checklist

- [ ] Set `PRICING_ADMIN_TOKEN` to a strong random value
- [ ] Set `DASHBOARD_PUBLIC=false` (production)
- [ ] Set `DEMO_MODE=false` (production)
- [ ] Enable Redis for persistent ban list + audit logs
- [ ] Rate limit `/health` and `/ready` (or allowlist probes)
- [ ] Monitor governance audit log for suspicious activity
- [ ] Rotate admin credentials periodically
- [ ] Ensure TLS and origin protection are configured

---

## Compliance Notes (SOC2 / ISO 27001)

SatGate supports common controls:

| Control | Implementation |
|---------|----------------|
| Per-request authorization | ✅ No implicit trust |
| Admin audit logging | ✅ All admin actions logged |
| Token revocation | ✅ Kill switch with O(1) lookup |
| Least privilege | ✅ Scope/time/budget caveats |
| Non-repudiation | ✅ Cryptographic signatures |

For auditors: use the **Governance Inspector** to demonstrate token constraints and provenance:

```bash
node cli/inspect.js <token>
```

---

## Threat Model

| Threat | Mitigation |
|--------|------------|
| DDoS / L7 floods | CDN/WAF + rate limits + (optional) economic friction via L402 |
| Scraping / automation abuse | L402 micropayments + throttling + telemetry |
| Credential stuffing | Price `/login` attempts; kill switch; short TTL |
| Stolen tokens | Short TTL + narrow scope + ban list |
| Admin endpoint abuse | Admin auth + rate limit + audit log + (roadmap) OIDC/RBAC |
| Replay attacks | Caveats + TTL + revocation |
| Token forgery | Cryptographic signatures |

---

## Responsible Disclosure

For security issues, contact: **security@satgate.io**

See [`SECURITY.md`](../SECURITY.md) in the repository root for disclosure process and expected response timelines.

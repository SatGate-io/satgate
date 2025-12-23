# SatGate Security Model

This document describes the security architecture and access control model for SatGate.

## Overview

SatGate implements a **Zero Trust Policy Enforcement Point (PEP)** for API traffic:

- **Per-request authorization** at the edge
- **No implicit trust** from network location
- **Stateless capability verification** (macaroons) with least-privilege caveats
- **Optional L402 payment challenges** (HTTP 402) to add economic friction against abuse

## Trust Boundary

SatGate supports two L402 deployment modes:

### L402 Native Mode (Recommended)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    TRUST BOUNDARY (L402 Native)                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   [Internet]                                                            │
│       │                                                                 │
│       ▼                                                                 │
│   ┌─────────┐    ┌─────────────────────────────────┐    ┌───────────┐  │
│   │ CDN/WAF │───▶│           SatGate               │───▶│ Your API  │  │
│   │         │    │  (L402 Authority + Governance)  │    │ (Origin)  │  │
│   └─────────┘    └─────────────────────────────────┘    └───────────┘  │
│                        │              │                                │
│                        │              │                                │
│              Lightning Backend   Capability Token                      │
│              (phoenixd/LND)      Verification                          │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

In **L402 Native Mode** (`L402_MODE=native`), SatGate is the L402 authority:
- Issues 402 challenges with `WWW-Authenticate: L402`
- Validates `Authorization: LSAT <macaroon>:<preimage>`
- Enforces per-request metering via `max_calls` and `budget_sats` caveats
- Re-challenges when budget/calls exhausted

### Aperture Sidecar Mode (Legacy)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    TRUST BOUNDARY (Aperture Sidecar)                    │
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

In **Aperture Sidecar Mode** (`L402_MODE=aperture`, default), Aperture handles L402 challenges and SatGate tracks/validates.

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

These routes are protected using L402 payment challenges:

| Pattern | Price | Default TTL | Default Max Calls |
|---------|-------|-------------|-------------------|
| `/api/micro/*` | 1 sat | 1 hour | 100 |
| `/api/basic/*` | 10 sats | 1 hour | 100 |
| `/api/standard/*` | 100 sats | 1 hour | 100 |
| `/api/premium/*` | 1000 sats | 1 hour | 100 |

#### Per-Request Metering (L402 Native Mode)

When `L402_MODE=native`, SatGate enforces **per-request-within-window** metering:

| Caveat | Purpose | Exhaustion Behavior |
|--------|---------|---------------------|
| `max_calls = N` | Limits requests within TTL | HTTP 429, then 402 re-challenge |
| `budget_sats = M` | Limits economic spend within TTL | HTTP 402 re-challenge |

The SDK flow is:
1. Client calls paid endpoint
2. SatGate returns `402` with `WWW-Authenticate: L402 macaroon="...", invoice="..."`
3. Client pays invoice, retries with `Authorization: LSAT <macaroon>:<preimage>`
4. SatGate validates + decrements counters atomically (Redis)
5. When exhausted → SatGate returns new 402 challenge (SDK pays again)

This ensures **"per request, per time window"** is technically enforceable.

### Capability-Protected Endpoints (Macaroon Required)

These routes require a valid capability macaroon (no payment):

| Pattern | Auth |
|---------|------|
| `/api/capability/*` | Macaroon signature verification + caveat enforcement |

Capability tokens support stateful metering via:
- `max_calls = N` caveat (HTTP 429 on exhaustion)
- `budget_sats = M` caveat (HTTP 402 on exhaustion)

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

### Core Configuration

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `MODE` | Operating mode: `prod` (secure) or `demo` (public demos) | No | `prod` |
| `PRICING_ADMIN_TOKEN` | Secret for admin endpoints (min 32 chars) | ✅ (prod) | None |
| `ADMIN_TOKEN_NEXT` | Secondary admin token for rotation | No | None |
| `DASHBOARD_PUBLIC` | Allow public dashboard access (only in `MODE=demo`) | No | `false` |
| `ADMIN_RATE_LIMIT` | Admin req/min per IP | No | `30` |
| `HEALTH_RATE_LIMIT` | Health check req/min per IP | No | `600` |
| `REDIS_URL` | Redis connection for persistence | No | in-memory |
| `CORS_ORIGINS` | Comma-separated allowed origins | No | localhost only |

> **Note:** `DASHBOARD_PUBLIC=true` is ignored when `MODE=prod` — see guardrail above.

### L402 Native Mode Configuration

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `L402_MODE` | L402 mode: `native` or `aperture` | No | `aperture` |
| `LIGHTNING_BACKEND` | Backend: `phoenixd`, `lnd`, `opennode`, `mock` | If native | None |
| `PHOENIXD_URL` | phoenixd REST API URL | If phoenixd | None |
| `PHOENIXD_PASSWORD` | phoenixd API password | If phoenixd | None |
| `LND_REST_URL` | LND REST API URL | If lnd | None |
| `LND_MACAROON` | LND admin macaroon (hex) | If lnd | None |
| `OPENNODE_API_KEY` | OpenNode API key | If opennode | None |
| `L402_DEFAULT_TTL` | Default token TTL (seconds) | No | `3600` |
| `L402_DEFAULT_MAX_CALLS` | Default max calls per token | No | `100` |

> **Warning:** OpenNode may not expose the Lightning payment hash required for LSAT preimage verification. Use phoenixd or LND for production.

---

## Deployment Security

### Recommended Architecture (L402 Native Mode)

```
Internet
  │
  ▼
CDN/WAF              ← DDoS protection, TLS termination, request filtering
  │
  ▼
SatGate              ← L402 authority, capability validation, governance
  │
  ├───▶ Lightning    ← phoenixd / LND (invoice creation, preimage verification)
  │     Backend
  │
  ├───▶ Redis        ← Metering counters, ban list, audit logs
  │
  ▼
Your API             ← Business logic
```

### Alternative: Aperture Sidecar Mode

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
- ✅ Enable Redis for durable ban list, audit logs, and metering counters
- ✅ Set strong, random `PRICING_ADMIN_TOKEN`
- ✅ Rotate admin credentials periodically
- ✅ Use `L402_MODE=native` with phoenixd or LND for enforceable per-request metering

---

## Security Checklist

### Core Security
- [ ] Set `PRICING_ADMIN_TOKEN` to a strong random value (min 32 chars)
- [ ] Set `MODE=prod` in production
- [ ] Set `DASHBOARD_PUBLIC=false` (production)
- [ ] Enable Redis for persistent ban list, audit logs, and metering
- [ ] Rate limit `/health` and `/ready` (or allowlist probes)
- [ ] Monitor governance audit log for suspicious activity
- [ ] Rotate admin credentials periodically
- [ ] Ensure TLS and origin protection are configured

### L402 Native Mode (if enabled)
- [ ] Set `L402_MODE=native`
- [ ] Configure Lightning backend (`LIGHTNING_BACKEND`, credentials)
- [ ] Use phoenixd or LND (OpenNode may not support preimage verification)
- [ ] Enable Redis for atomic metering counters
- [ ] Set appropriate `L402_DEFAULT_TTL` and `L402_DEFAULT_MAX_CALLS`
- [ ] Verify `/api/lightning/status` returns healthy

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
| DDoS / L7 floods | CDN/WAF + rate limits + economic friction via L402 |
| Scraping / automation abuse | L402 micropayments + throttling + telemetry |
| Credential stuffing | Price `/login` attempts; kill switch; short TTL |
| Stolen tokens | Short TTL + narrow scope + ban list + `max_calls` limit |
| Admin endpoint abuse | Admin auth + rate limit + audit log + (roadmap) OIDC/RBAC |
| Replay attacks | Caveats + TTL + revocation + stateful metering (`max_calls`, `budget_sats`) |
| Token forgery | Cryptographic signatures (macaroon HMAC chain) |
| Invoice-spam DoS | Rate limit challenge issuance per IP |
| Token exhaustion gaming | Atomic Redis counters; re-challenge on exhaustion |
| Lightning backend failure | Multi-backend support; health monitoring |

---

## Responsible Disclosure

For security issues, contact: **security@satgate.io**

See [`SECURITY.md`](../SECURITY.md) in the repository root for disclosure process and expected response timelines.

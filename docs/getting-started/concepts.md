# Core Concepts

Understanding SatGate Gateway's architecture and terminology.

## Capability Tokens

SatGate uses **capability tokens** (based on Google's Macaroons) for authentication.

### What Makes Them Special?

| Feature | Traditional API Keys | Capability Tokens |
|---------|---------------------|-------------------|
| Delegation | Requires server call | Offline (cryptographic) |
| Attenuation | Not possible | Add restrictions without server |
| Revocation | Per-key | Hierarchical (revoke parent = revoke children) |
| Audit | Limited | Full lineage tracking |

### Token Structure

```
┌─────────────────────────────────────────────────────────┐
│                   Capability Token                       │
├─────────────────────────────────────────────────────────┤
│  Location:  https://satgate.io                          │
│  Identifier: token-id-abc123                            │
│  Caveats:                                               │
│    - scope = api:read                                   │
│    - expires = 2024-01-01T12:00:00Z                     │
│    - rate_limit = 100/minute                            │
│  Signature: cryptographic-signature                     │
└─────────────────────────────────────────────────────────┘
```

### Offline Delegation

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Admin     │     │   Agent     │     │   Worker    │
│   Token     │────▶│   Token     │────▶│   Token     │
│ (all perms) │     │ (api:read)  │     │ (api:read,  │
│             │     │             │     │  1 hour)    │
└─────────────┘     └─────────────┘     └─────────────┘
       │                  │                   │
       │   No server      │   No server       │
       │   call needed    │   call needed     │
       ▼                  ▼                   ▼
    Mint              Delegate            Use API
```

## Routes and Upstreams

### Upstreams

An **upstream** is a backend API that the gateway proxies to:

```yaml
upstreams:
  my_api:
    url: "https://api.example.com"
    timeout: 30s
    healthCheck:
      path: "/health"
      interval: 10s
```

### Routes

A **route** matches requests and applies policies:

```yaml
routes:
  - name: protected-api
    path: /api/*
    upstream: my_api
    policy:
      kind: observe    # or: control, charge, public
      scope: api:read
```

### Policy Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `public` | No authentication | Health checks, public APIs |
| `observe` | Authentication + metering | Monitor before enforcing |
| `control` | Authentication + budget enforcement | Enterprise FinOps, quotas |
| `charge` | Authentication + payment required | API monetization |
| `deny` | Always reject | Deprecated endpoints |

> **"Protection by default. Payments optional."** — Start with `observe` to monitor, graduate to `control` for budgets, add `charge` when ready to monetize.

## Tenant Isolation

For multi-tenant deployments, SatGate enforces isolation:

```
┌─────────────────────────────────────────────────────────┐
│                    SatGate Gateway                       │
├─────────────────────────────────────────────────────────┤
│  Tenant A                    │  Tenant B                │
│  ─────────                   │  ─────────               │
│  Quota: 10,000 req/day       │  Quota: 50,000 req/day   │
│  Domains: api.tenant-a.com   │  Domains: api.tenant-b.com│
│  Tokens: isolated            │  Tokens: isolated        │
│  Audit: tenant-scoped        │  Audit: tenant-scoped    │
└─────────────────────────────────────────────────────────┘
```

### Single-Tenant Default

For enterprise self-hosted deployments, set a default tenant:

```yaml
server:
  defaultTenantId: "my-company"
```

All requests are attributed to this tenant without client changes.

## Governance

### Token Lifecycle

```
Mint → Active → [Delegated] → Revoked/Expired
  │       │          │             │
  │       │          │             └── Kill Switch (instant)
  │       │          └── Child tokens inherit parent state
  │       └── Used for API access
  └── Created by admin
```

### Kill Switch

Revoke a token and all its descendants instantly:

```bash
curl -X POST /api/v1/governance/ban \
  -d '{"signature": "token-to-ban"}'
```

### Audit Trail

Every action is logged with tamper-evident hash chain:

```json
{
  "event": "token.mint",
  "timestamp": "2024-01-01T12:00:00Z",
  "userId": "admin@example.com",
  "resourceId": "token-abc123",
  "previousHash": "sha256:...",
  "eventHash": "sha256:..."
}
```

## Data Plane vs Admin Plane

| Plane | Port | Purpose | Access |
|-------|------|---------|--------|
| **Data** | 8080 | API traffic | Public (via ingress) |
| **Admin** | 9090 | Management | Internal only |

**Security**: The admin plane should NEVER be exposed publicly.

## Policy Modes Deep Dive

### Observe Mode (Default Starting Point)

- Capability tokens for access control
- Metering without enforcement
- No Lightning/Bitcoin required

```yaml
routes:
  - name: observed-api
    policy:
      kind: observe
      scope: api:read
```

### Control Mode (Budget Enforcement)

Meter usage and enforce budgets without 402 challenges:

- Works with capability tokens (no client changes)
- No 402 challenges — enforcement is via 429
- Budget enforcement with soft/hard limits
- Export to finance systems (SAP, NetSuite)

```yaml
routes:
  - name: controlled-api
    policy:
      kind: control
      scope: api:read
      budget:
        default: 10000
        period: monthly
        hardLimit: true
```

### Charge Mode (Payment Required)

SatGate supports two payment rails:

| Rail | Description | 402 Challenge? | License |
|------|-------------|----------------|---------|
| **L402** | Lightning micropayments | Yes | OSS |
| **Fiat402** | Fiat invoice gating | Yes | Enterprise |

#### L402 (Lightning Payments)

Pay-per-request using the Lightning Network:

```yaml
lightning:
  provider: "phoenixd"
  l402RootKey: "${L402_ROOT_KEY}"

routes:
  - name: premium
    policy:
      kind: charge
      rail: l402
      priceSats: 50
```

Use cases:
- Public API monetization
- Micropayments for AI/ML inference
- Anonymous paid access

#### Fiat402 Mode (Enterprise)

402-style gating with fiat invoices:

```yaml
routes:
  - name: paid
    policy:
      kind: charge
      rail: fiat402
      priceUsd: 0.10
```

Use cases:
- Pre-paid credits
- Stripe/invoice integration
- Enterprise service agreements

See [Policy Modes](../POLICY_MODES.md) for complete documentation.

## Next Steps

- [Your First Route](first-route.md) — Protect your API
- [Token Management](../guides/tokens.md) — Mint and delegate
- [Production Deployment](../guides/kubernetes.md) — Kubernetes setup

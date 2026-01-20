<p align="center">
  <img src="logo_blue_transparent.png" alt="SatGate Logo" width="200">
</p>

# SatGate™ Gateway vs Aperture

> Enterprise-grade API protection and payments for the Agent Economy
>
> **SatGate™ Gateway Documentation**

## Executive Summary

| Dimension | **Aperture** | **SatGate Gateway** |
|-----------|-------------|---------------------|
| **Primary Focus** | L402 payments only | Protection + Payments ("Protect & Pay") |
| **Target User** | Developers, hobbyists | Enterprise, AI/Agent platforms |
| **Maintained** | Community/stale | Active development |

---

## Feature Comparison

| Feature | Aperture | SatGate Gateway |
|---------|:--------:|:---------------:|
| **L402 Lightning Payments** | ✅ | ✅ |
| **Capability Tokens (no payment)** | ❌ | ✅ |
| **Chargeback/Showback Mode** | ❌ | ✅ |
| **Fiat402 (JWT receipts)** | ❌ | ✅ |
| **Multi-tenant Isolation** | ❌ | ✅ |
| **Token Governance (ban/lineage)** | ❌ | ✅ |
| **Admin Dashboard UI** | ❌ | ✅ |
| **SCIM 2.0 Provisioning** | ❌ | ✅ |
| **GitOps Signed Configs** | ❌ | ✅ |
| **Audit Logging (WORM)** | ❌ | ✅ |
| **Budget Management** | ❌ | ✅ |
| **Rate Limiting (Redis HA)** | Basic | ✅ Per-route, Redis-backed |
| **mTLS to Upstreams** | ❌ | ✅ |
| **Circuit Breaker** | ❌ | ✅ |
| **Prometheus Metrics** | Basic | ✅ Full observability |
| **Helm Chart** | Community | ✅ Official |
| **Client SDKs** | Go only | ✅ Go, Node.js, Python |
| **Supply Chain Security** | ❌ | ✅ Cosign, SBOM, attestations |

---

## Architectural Differences

### Aperture

```
┌─────────────────────────────────┐
│          APERTURE               │
├─────────────────────────────────┤
│  ┌──────────────┐               │
│  │   L402 Proxy │               │
│  │  (pay only)  │               │
│  └──────────────┘               │
│         │                       │
│  ┌──────┴──────┐                │
│  │  LND only   │                │
│  │  (coupled)  │                │
│  └─────────────┘                │
└─────────────────────────────────┘
```

### SatGate Gateway

```
┌─────────────────────────────────┐
│       SATGATE GATEWAY           │
├─────────────────────────────────┤
│  Protection │ Payments │ Gov    │
│  (Capability)│(L402/Fiat)│(Audit)│
├─────────────────────────────────┤
│  Pluggable Lightning Providers  │
│  Phoenixd│LND│CLN│Alby│Strike   │
├─────────────────────────────────┤
│  Enterprise Infrastructure      │
│  Postgres│Redis│Prometheus│SCIM │
└─────────────────────────────────┘
```

---

## Key Benefits of SatGate Gateway

### 1. "Protection by default. Payments optional."

- Start with capability tokens (API security) → no procurement friction
- Add payments later when business model is proven
- Aperture forces the payment decision upfront

### 2. Enterprise-Grade Security

| Security Feature | Aperture | SatGate |
|-----------------|:--------:|:-------:|
| Tamper-evident audit logs | ❌ | ✅ |
| Token ban/revocation | ❌ | ✅ |
| Token lineage tracking | ❌ | ✅ |
| SCIM user provisioning | ❌ | ✅ |
| Signed config enforcement | ❌ | ✅ |
| WORM export for compliance | ❌ | ✅ |

### 3. Flexible Payment Modes

```yaml
# Aperture: One mode
policy: l402  # Must pay or rejected

# SatGate: Three modes per route
routes:
  - match: { pathPrefix: /api/v1 }
    policy:
      kind: capability  # Free, but authenticated
      
  - match: { pathPrefix: /api/premium }
    policy:
      kind: pay
      pay:
        mode: chargeback  # Metered, billed internally
        
  - match: { pathPrefix: /api/paid }
    policy:
      kind: pay
      pay:
        mode: l402  # Pay-per-request with Lightning
```

### 4. Multi-Tenant by Design

- **Aperture:** Single tenant only
- **SatGate:** Full tenant isolation, per-tenant budgets, quotas

### 5. Operational Excellence

| Ops Feature | Aperture | SatGate |
|-------------|:--------:|:-------:|
| Admin Dashboard | ❌ | ✅ Full UI |
| Config generator | ❌ | ✅ YAML/Helm/Env |
| Preflight checks | ❌ | ✅ Enterprise readiness |
| Support bundles | ❌ | ✅ One-click diagnostics |
| Backup/restore runbook | ❌ | ✅ Documented |

### 6. Lightning Provider Flexibility

**Aperture (LND only):**
```yaml
lnd:
  host: localhost:10009
  macaroon: /path/to/admin.macaroon
```

**SatGate (Any provider):**
```yaml
lightning:
  provider: phoenixd  # or lnd, cln, alby, lnbits, strike
  endpoint: http://phoenixd:9740
  # Provider-specific config...
```

---

## When to Use Each

| Use Case | Recommended |
|----------|-------------|
| Simple hobby project with LND | Either |
| Quick L402 demo | Either |
| Enterprise API monetization | **SatGate Gateway** |
| Multi-tenant SaaS platform | **SatGate Gateway** |
| AI Agent authentication | **SatGate Gateway** |
| FinOps / chargeback tracking | **SatGate Gateway** |
| Compliance requirements (SOC2, audit) | **SatGate Gateway** |
| Production deployment with SLAs | **SatGate Gateway** |

---

## Migration from Aperture

SatGate maintains L402 wire compatibility—existing macaroon clients work unchanged.

**Aperture config:**
```yaml
services:
  - name: myapi
    price: 100
```

**SatGate equivalent:**
```yaml
routes:
  - name: myapi
    match:
      pathPrefix: /
    upstream: http://myapi:8080
    policy:
      kind: l402
      priceSats: 100
```

> **Wire Compatibility:** SatGate maintains L402 wire compatibility—existing macaroon clients work unchanged. You can migrate incrementally, route by route.

---

## Payment Modes Explained

### Capability (Protection Only)

- No payment required
- Token-based authentication
- Scope-based authorization
- Use case: Internal APIs, free tier, agent auth

### L402 (Lightning Payments)

- Pay-per-request with Lightning
- Macaroon + preimage verification
- Use case: Public API monetization

### Chargeback (Enterprise FinOps)

- Metered usage, no real payment
- Internal cost allocation
- Budget enforcement
- Use case: Enterprise showback/chargeback

### Fiat402 (JWT Receipts)

- Prepaid receipts via external billing
- JWT token verification
- Use case: Enterprise invoicing, credits

---

## Resources

- [SatGate Gateway Documentation](https://satgate.io/docs)
- [GitHub Repository](https://github.com/SatGate-io/satgate-gateway)
- [Aperture (Lightning Labs)](https://github.com/lightninglabs/aperture)

---

<p align="center">
  <img src="logo_blue_transparent.png" alt="SatGate Logo" width="120">
</p>

<p align="center">
  <strong>SatGate™ Gateway</strong> — EZ-Pass for the Agent Economy ⚡
</p>

<p align="center">
  <em>"Protection by default. Payments optional."</em>
</p>

<p align="center">
  <sub>SatGate™ is a trademark of SatGate, Inc. All rights reserved.</sub>
</p>


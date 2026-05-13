# SatGate Gateway Documentation

**The Economic Firewall — Protection and Payments for the Enterprise**  
*"Protection by default. Payments optional."*

---

## What is SatGate Gateway?

SatGate Gateway is an enterprise API gateway that provides:

- **Observe Mode** — Track who's calling your APIs (authentication + metering)
- **Control Mode** — Enforce budget limits (authentication + enforcement)
- **Charge Mode** — Require payment per request (L402/Fiat402 monetization)
- **Governance** — receipt-backed Evidence Packs, token revocation, compliance exports

## Quick Links

| I want to... | Go to... |
|--------------|----------|
| Understand the big picture | [Overview](OVERVIEW.md) |
| Try it in 5 minutes | [Quick Start](getting-started/quickstart.md) |
| Learn the concepts | [Core Concepts](getting-started/concepts.md) |
| Protect my first API | [Your First Route](getting-started/first-route.md) |
| See the architecture | [Architecture](ARCHITECTURE.md) |
| Understand policy modes | [Policy Modes](POLICY_MODES.md) |
| Learn about tokens | [Capability Tokens](CAPABILITY_TOKENS.md) |
| Use with lnget (Lightning Labs) | [lnget Integration](guides/lnget-integration.md) |
| Deploy to Kubernetes | [Helm Installation](guides/kubernetes.md) |
| See the API reference | [API Reference](api/overview.md) |
| Export Policy-to-Proof evidence | [Evidence Pack v1](reference/evidence-pack.md) |
| Discover SatGate trust metadata | [SatGate Trust Metadata](reference/satgate-trust-metadata.md) |
| Sketch upstream acceptor metadata | [Acceptor Metadata Draft](reference/acceptor.md) |
| Run in production | [Production Checklist](operations/production-checklist.md) |
| **Enterprise: Economic Firewall** | [Economic Firewall Quickstart](getting-started/economic-firewall-quickstart.md) |
| **Enterprise: Team Management** | [Team Management Guide](enterprise/TEAM_MANAGEMENT.md) |
| **Enterprise: Admin Impersonation** | [Admin Impersonation Guide](enterprise/ADMIN_IMPERSONATION.md) |
| **Enterprise: Feature Matrix** | [OSS vs Enterprise](enterprise/FEATURE_MATRIX.md) |

## Documentation Hierarchy

```
📚 SatGate Documentation
│
├── OVERVIEW.md            ← "What is it? Why should I care?"
│   └── Executive summary, business value
│
├── ARCHITECTURE.md        ← "How is it built? Can I trust it?"
│   └── Technical deep dive, deployment patterns
│
├── CAPABILITY_TOKENS.md   ← "How do credentials work?"
│   └── Macaroons, SatGate Mint, delegation
│
├── POLICY_MODES.md        ← "How do policies work?"
│   └── Observe, Control, Charge modes
│
└── /getting-started/      ← "How do I get started?"
    └── Tutorials and guides
```

## Deployment Options

| Environment | Method | Time | Use Case |
|-------------|--------|------|----------|
| **Local/POC** | [Docker Compose](../deploy/docker-compose/) | 5 min | Evaluation, demos |
| **Production** | [Helm (Kubernetes)](guides/kubernetes.md) | 30 min | Enterprise deployment |
| **AWS** | [Terraform](../deploy/terraform/aws/) | 45 min | EKS + RDS + ElastiCache |
| **GCP** | [Terraform](../deploy/terraform/gcp/) | 45 min | GKE + Cloud SQL + Memorystore |

## Client SDKs

| Language | Package | Documentation |
|----------|---------|---------------|
| Go | `github.com/satgate-io/satgate-go` | [Go SDK](sdks/go.md) |
| Python | `satgate` | [Python SDK](sdks/python.md) |
| Node.js | `@satgate/sdk` | [Node.js SDK](sdks/nodejs.md) |

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Ingress                                 │
│                    (TLS termination)                            │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    SatGate Gateway                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌────────────────┐  │
│  │  Data Plane     │  │  Admin Plane    │  │  Governance    │  │
│  │  (Public API)   │  │  (Internal)     │  │  (Ban/Audit)   │  │
│  │  :8080          │  │  :9090          │  │                │  │
│  └────────┬────────┘  └────────┬────────┘  └───────┬────────┘  │
│           │                    │                   │            │
│           ▼                    ▼                   ▼            │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  PostgreSQL (config, audit) │ Redis (rate limit, sessions) ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Upstream APIs                              │
│              (Your protected backend services)                  │
└─────────────────────────────────────────────────────────────────┘
```

## Policy Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `public` | No authentication | Health checks, public APIs |
| `observe` | Authentication + metering | Monitor before enforcing |
| `control` | Authentication + budget enforcement | Enterprise FinOps, quotas |
| `charge` | Authentication + payment required | API monetization |

> **"Protection by default. Payments optional."** — Start with `observe` to monitor, graduate to `control` for budgets, add `charge` when ready to monetize.

## OSS vs Enterprise

| Feature | OSS (Community) | Enterprise/Cloud |
|---------|:---------------:|:----------------:|
| Protection Mode (tokens, delegation) | ✅ | ✅ |
| L402 Charge Mode (Lightning) | ✅ | ✅ |
| Control Mode (budgets) | ❌ | ✅ |
| Fiat402 Mode | ❌ | ✅ |
| Economic Firewall (Delegation v2) | ❌ | ✅ |
| Budget Enforcement (HTTP 402) | ❌ | ✅ |
| Multi-Rail Settlement | ❌ | ✅ |
| Team Management (RBAC) | ❌ | ✅ |
| Admin Impersonation | ❌ | ✅ |
| Support System | ❌ | ✅ |
| Audit Dashboard | ❌ | ✅ |

> **"Protection by default. Payments optional."** — OSS provides core security and monetization. Enterprise adds governance, budgets, and multi-tenant management.

## Key Features

### Observe Mode (Start Here) — OSS ✅

- **Capability Tokens** — Google-style macaroons for granular access control
- **Offline Delegation** — Attenuate tokens without network calls
- **Kill Switch** — Instant token revocation across all instances
- **Usage Metering** — Track who's using what

### Charge Mode (Monetization) — OSS ✅

- **L402 Protocol** — Pay-per-request via Lightning Network
- **402 Challenges** — Standard HTTP payment required flow

### Control Mode (Budget Enforcement) — Enterprise

- **Budget Limits** — Set per-token or per-tenant limits
- **Hard Stop** — HTTP 402 when budget exhausted
- **Delegation v2** — Hierarchical token tree with scope/budget constraints
- **Alerts** — Threshold notifications at 80%, 90%, 100%
- **Spend Export** — JSON/CSV reports with filters

### Fiat402 Mode — Enterprise

- **Fiat Payments** — Stripe integration
- **Receipt Tokens** — JWT-based payment proofs
- **Settlement Coupling** — L402/Stripe → Credits bridging

### Multi-Tenant Governance — Enterprise

- **Team Management** — RBAC with Owner/Admin/Member/Viewer roles
- **Admin Impersonation** — "View as Customer" for support
- **Audit Logging** — Tamper-evident receipt chain
- **Security Dashboard** — Filterable receipt and Evidence Pack viewer
- **SSO** — OIDC/SAML integration
- **Fleet Management** — Multi-gateway deployments

## Getting Started

1. **[Overview](OVERVIEW.md)** — Understand what SatGate is
2. **[Quick Start](getting-started/quickstart.md)** — Run locally in 5 minutes
3. **[Core Concepts](getting-started/concepts.md)** — Learn the terminology
4. **[Your First Route](getting-started/first-route.md)** — Protect an API endpoint

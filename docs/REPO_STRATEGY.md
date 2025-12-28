# Repository Strategy

SatGate follows an **OSS core + managed Cloud** model, similar to GitLab, Supabase, and PostHog.

## Two Repositories

### 1. `satgate/satgate` (Public, MIT)

The open-source gateway, SDKs, and documentation.

```
satgate/
├── proxy/              # L402 gateway (Node.js + Aperture)
├── sdk/
│   ├── python/         # pip install satgate
│   ├── js/             # npm install @satgate/sdk
│   └── go/             # go get github.com/satgate/satgate/sdk/go
├── docker/             # One-click deployment
├── deploy/             # Cloud deployment configs
├── docs/               # Architecture, security, guides
├── examples/           # Demo scripts
├── cli/                # Token inspection tools
└── satgate-landing/    # satgate.io marketing site
```

**License:** MIT

### 2. `satgate/satgate-cloud` (Private, Proprietary)

The managed Cloud platform (multi-tenant SaaS).

```
satgate-cloud/
├── apps/
│   ├── control-plane/  # Auth, projects, billing, earnings
│   ├── data-plane/     # Multi-tenant L402 gateway
│   └── dashboard/      # Customer UI (cloud.satgate.io)
├── db/                 # Schema + migrations
├── ops/                # Operator tools
├── packages/           # Shared TypeScript packages
└── docs/               # Internal documentation
```

**License:** Proprietary (All Rights Reserved)

## Why This Split?

| Aspect | Public Repo | Private Repo |
|--------|-------------|--------------|
| **Purpose** | Trust, distribution, community | Velocity, competitive moat |
| **Contributions** | Welcome (PRs, issues) | Internal only |
| **Deployment** | Self-hosted anywhere | Fly.io managed |
| **Monetization** | N/A (free OSS) | SaaS subscriptions |

### Benefits

1. **Clear contribution boundaries** — OSS contributors know what's open
2. **No accidental IP leakage** — Private repo stays private
3. **Simpler CI/CD** — Each repo deploys independently
4. **Better optics** — Enterprise customers see clean separation

## What Stays Open

- **L402 protocol implementation** — Trust requires transparency
- **Gateway proxy** — The core product
- **SDKs** — Developer adoption
- **Documentation** — Community growth
- **Self-hosted deployment** — Alternative to Cloud

## What Stays Private

- **Multi-tenant control plane** — Billing, auth, earnings
- **Payout/ledger system** — Financial operations
- **Operator dashboard** — Admin tools
- **Fee configuration** — Business logic
- **GTM materials** — Sales, strategy

## Contributing

- **OSS gateway/SDKs:** [github.com/satgate/satgate](https://github.com/satgate/satgate)
- **Cloud platform:** Internal only (contact us for enterprise features)

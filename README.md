<p align="center">
  <img src="docs/assets/logo.png" alt="SatGate" width="120" />
</p>

<h1 align="center">SatGate</h1>

<p align="center">
  <strong>The Open Source Economic Firewall for APIs</strong>
</p>

<p align="center">
  <a href="https://github.com/satgate-io/satgate/actions"><img src="https://github.com/satgate-io/satgate/workflows/CI/badge.svg" alt="CI Status"></a>
  <a href="https://goreportcard.com/report/github.com/satgate-io/satgate"><img src="https://goreportcard.com/badge/github.com/satgate-io/satgate" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/satgate-io/satgate"><img src="https://pkg.go.dev/badge/github.com/satgate-io/satgate.svg" alt="Go Reference"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#documentation">Docs</a> •
  <a href="#architecture">Architecture</a> •
  <a href="https://satgate.io">Website</a>
</p>

---

## What is SatGate?

SatGate is an **Economic Firewall**—an API gateway that adds cryptographic capability verification and payment protocols to any backend. It's designed for the agentic web, where AI agents and automated systems need secure, verifiable access to APIs.

**Protection by default. Economics optional.**

## Features

- 🛡️ **Capability Tokens (Macaroons)** — Cryptographic API keys with built-in caveats, delegation, and instant revocation
- ⚡ **L402 Protocol** — Native Bitcoin Lightning payments for API monetization
- 🔒 **Default Protection** — All routes require valid credentials unless explicitly public
- 🚀 **<50ms Latency** — Lightweight proxy with minimal overhead
- 📦 **Self-Hosted** — Run on your infrastructure with full control
- 🔌 **Drop-in** — Works with any HTTP backend, no code changes required

## Quick Start

### Using Docker (Recommended)

```bash
# Clone the repo
git clone https://github.com/satgate-io/satgate.git
cd satgate

# Start with Docker Compose
docker compose up -d

# Gateway is now running on http://localhost:8080
```

### Using Binary

```bash
# Download latest release
curl -L https://github.com/satgate-io/satgate/releases/latest/download/satgate-linux-amd64 -o satgate
chmod +x satgate

# Run with config
./satgate --config gateway.yaml
```

### Using Go

```bash
go install github.com/satgate-io/satgate/cmd/satgate@latest
satgate --config gateway.yaml
```

## Configuration

Create a `gateway.yaml`:

```yaml
# SatGate Configuration
listen: ":8080"

# Root key for Macaroon signing (generate with: openssl rand -hex 32)
capability:
  rootKey: "your-64-char-hex-key-here"

# Upstreams (your backend services)
upstreams:
  api:
    url: "http://localhost:3000"
    healthPath: "/health"

# Routes
routes:
  # Public routes (no auth required)
  - name: health
    path: /health
    upstream: api
    policy:
      kind: public

  # Protected routes (require valid Macaroon)
  - name: api
    path: /api/*
    upstream: api
    policy:
      kind: capability

  # Paid routes (require L402 payment)
  - name: premium
    path: /premium/*
    upstream: api
    policy:
      kind: l402
      priceSats: 100
```

## Policy Types

| Policy | Description | Use Case |
|--------|-------------|----------|
| `public` | No authentication | Health checks, docs, webhooks |
| `capability` | Requires valid Macaroon | Protected API endpoints |
| `l402` | Requires Lightning payment | Monetized endpoints |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        SatGate                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    Request Flow                       │   │
│  │                                                       │   │
│  │   Request → Policy Check → Credential Verify → Proxy │   │
│  │                    ↓                                  │   │
│  │              [public: pass]                           │   │
│  │              [capability: verify macaroon]            │   │
│  │              [l402: verify payment proof]             │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

**Key Concepts:**

- **Macaroons**: Cryptographic bearer tokens with embedded caveats (expiry, scope, IP restrictions)
- **Delegation**: Create child tokens with reduced permissions without contacting the server
- **Revocation**: Instantly ban tokens via governance endpoints
- **L402**: HTTP 402 Payment Required + Lightning invoice for machine-to-machine payments

## SDKs

Official SDKs for interacting with SatGate-protected APIs:

| Language | Package | Docs |
|----------|---------|------|
| Python | `pip install satgate` | [README](sdk/python/README.md) |
| JavaScript | `npm install @satgate/client` | [README](sdk/js/README.md) |
| Go | `go get github.com/satgate-io/satgate/sdk/go` | [README](sdk/go/README.md) |

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [Security Model](docs/SECURITY-MODEL.md)
- [L402 Response Schema](docs/L402-RESPONSE-SCHEMA.md)
- [Self-Hosted Deployment](docs/SELF_HOSTED.md)
- [API Reference](docs/API.md)

## SatGate Enterprise

Need more? [SatGate Enterprise](https://cloud.satgate.io) adds:

- 🏢 **Multi-tenant Control Plane** — Managed dashboard for teams
- 📊 **Observe Policy** — Usage metering and FinOps visibility
- 🎚️ **Control Policy** — Budget enforcement with Fiat402
- 💰 **Charge Policy** — Stripe billing integration
- 🔐 **Delegation v2** — Hierarchical tokens with budgets
- 🤖 **SatGate Mint** — Zero-touch agent provisioning (K8s, AWS, OIDC)
- 🌐 **Hybrid Mode** — Self-hosted gateway with cloud control plane
- 📝 **Tamper-Evident Audit** — Compliance-ready logging
- 🎫 **Support & SLA** — Enterprise support

[Contact Sales](mailto:contact@satgate.io) | [Start Free Trial](https://cloud.satgate.io)

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
# Clone
git clone https://github.com/satgate-io/satgate.git
cd satgate

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o satgate ./cmd/satgate
```

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.

## Community

- 🐦 [Twitter](https://twitter.com/satgate_io)
- 💬 [Discord](https://discord.gg/satgate)
- 📧 [Email](mailto:hello@satgate.io)

---

<p align="center">
  <sub>Built with ⚡ by <a href="https://satgate.io">SatGate</a></sub>
</p>

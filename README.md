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

### 60-Second Demo

```bash
# Download the binary (macOS Apple Silicon shown — see docs for other platforms)
curl -L https://github.com/satgate-io/satgate/releases/latest/download/satgate-darwin-arm64 -o satgate
chmod +x satgate

# Start with the example config (mock Lightning, auto-generated keys)
export ADMIN_TOKEN=my-secret-token
export LIGHTNING_BACKEND=mock
./satgate --config examples/gateway.yaml
```

**In another terminal, try the three policies:**

```bash
# 1. Public route — no auth needed
curl http://localhost:8080/health

# 2. Protected route — mint a token, then use it
curl -X POST http://localhost:8080/api/capability/mint \
  -H "X-Admin-Token: my-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"scope": "api:read", "duration": "1h"}'

# Save the token from the response, then:
curl -H "Authorization: Bearer <your-token>" \
  http://localhost:8080/api/capability/ping

# 3. Paid route — get an L402 challenge
curl http://localhost:8080/api/micro
# Returns HTTP 402 with a Lightning invoice
```

That's it. Protection → Capability → Payment. Three commands.

📖 **[Full Quick Start Guide →](docs/getting-started/quickstart.md)**

### Other Install Methods

**Docker:**

```bash
git clone https://github.com/satgate-io/satgate.git
cd satgate && docker compose up -d
```

**Build from source:**

```bash
git clone https://github.com/satgate-io/satgate.git
cd satgate && go build -o satgate ./cmd/satgate
```

## Configuration

```yaml
version: 1

server:
  listen: ":8080"

admin:
  capabilityRootKey: "${CAPABILITY_ROOT_KEY}"

lightning:
  provider: "${LIGHTNING_BACKEND}"
  config:
    connectionString: "${NWC_CONNECTION_STRING}"

upstreams:
  api:
    url: "http://localhost:3000"

routes:
  - name: public-health
    match:
      pathPrefix: /health
    upstream: api
    policy:
      kind: public

  - name: protected-api
    match:
      pathPrefix: /api/
    upstream: api
    policy:
      kind: capability
      scope: "api:read"

  - name: premium-api
    match:
      pathPrefix: /premium/
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

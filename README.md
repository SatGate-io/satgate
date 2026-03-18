<p align="center">
  <img src="docs/assets/logo.png" alt="SatGate" width="120" />
</p>

<h1 align="center">SatGate</h1>

<p align="center">
  <strong>The Economic Firewall for AI Agents</strong><br/>
  <em>Hard budget enforcement · Per-tool cost attribution · L402 micropayments</em>
</p>

<p align="center">
  <a href="https://github.com/satgate-io/satgate/actions"><img src="https://github.com/satgate-io/satgate/workflows/CI/badge.svg" alt="CI Status"></a>
  <a href="https://goreportcard.com/report/github.com/satgate-io/satgate"><img src="https://goreportcard.com/badge/github.com/satgate-io/satgate" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/satgate-io/satgate"><img src="https://pkg.go.dev/badge/github.com/satgate-io/satgate.svg" alt="Go Reference"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
</p>

<p align="center">
  <a href="#the-problem">Why</a> •
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#documentation">Docs</a> •
  <a href="https://satgate.io">Website</a> •
  <a href="https://satgate.io/blog/why-routing-isnt-governance">Blog</a>
</p>

---

<div align="center">

### 🎬 See SatGate in Action

<a href="https://satgate.io#explainer"><img src="https://img.shields.io/badge/▶_Explainer-30s-purple?style=for-the-badge" alt="Watch Explainer"></a>&nbsp;&nbsp;<a href="https://satgate.io#delegation"><img src="https://img.shields.io/badge/▶_Token_Delegation-45s-blue?style=for-the-badge" alt="Watch Delegation"></a>

</div>

---

<div align="center">

### ☁️ Don't want to self-host? Try SatGate Cloud

**Managed SaaS — zero setup, multi-tenant isolation, enterprise dashboard.**<br/>
Free Observe tier. No credit card required.

<a href="https://cloud.satgate.io"><img src="https://img.shields.io/badge/🚀_Try_SatGate_Cloud-Free-blue?style=for-the-badge" alt="Try SatGate Cloud"></a>

</div>

---

## The Problem

AI agents are making API calls autonomously. They spawn sub-agents, call MCP tools, and run overnight while you sleep.

Your existing stack answers: *"Is this request authenticated?"*

Nobody answers: **"Should this agent spend this?"**

```
✓ Network Firewall    → "Can this packet enter?"
✓ Application Firewall → "Is this request safe?"
? Economic Firewall    → "Should this agent spend this?"
```

That's the gap. SatGate fills it.

## What is SatGate?

SatGate is an **Economic Firewall** that enforces **Economic Access Control** for AI agent requests. Drop it in front of your APIs — it handles authentication, budget enforcement, cost attribution, and optional micropayments.

**Not another routing layer.** Routing gateways (Bifrost, LiteLLM, Portkey) optimize *which provider* handles a call. SatGate governs *whether the call should happen at all* based on budgets, policies, and cost.

Use them together:

```
Agent → SatGate (economic governance) → Routing Gateway → LLM Providers
```

## Features

- 🛡️ **Capability Tokens (Macaroons)** — Cryptographic credentials with built-in caveats, delegation, and instant revocation. Not API keys — tokens that agents can safely sub-delegate.
- 🎯 **MCP-Aware** — Parses MCP JSON-RPC tool calls. Know that Agent X spent $47 on `search_database` and $12 on `send_email` — not just "1,000 requests."
- 💰 **Budget Enforcement** — Hard stops per agent, team, or API. When the budget hits zero, requests are *blocked*. Not logged. Not alerted. Blocked.
- ⚡ **L402 Protocol** — Native Bitcoin Lightning micropayments for API monetization. Sub-cent pricing that's uneconomical on card rails.
- 🔒 **Default-Deny** — All routes require valid credentials unless explicitly public. Zero Trust by design.
- 🚀 **<50ms Overhead** — Lightweight Go proxy. Adds governance without adding latency.
- 📦 **Self-Hosted** — Your infrastructure, your rules. Single binary, Docker, or Kubernetes.
- 🔌 **Drop-in** — Works with any HTTP backend. REST, GraphQL, MCP servers. No code changes.

## Quick Start

### 60-Second Demo

```bash
# Download the binary (macOS Apple Silicon — see Releases for other platforms)
curl -L https://github.com/satgate-io/satgate/releases/latest/download/satgate-darwin-arm64 -o satgate
chmod +x satgate

# Start with example config (mock Lightning, auto-generated keys)
export ADMIN_TOKEN=my-secret-token
export LIGHTNING_BACKEND=mock
./satgate --config examples/gateway.yaml
```

**Try the three policies:**

```bash
# 1. Public — no auth needed
curl http://localhost:8080/health

# 2. Protected — mint a capability token, then use it
curl -X POST http://localhost:8080/api/capability/mint \
  -H "X-Admin-Token: my-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"scope": "api:read", "duration": "1h"}'

# Use the token:
curl -H "Authorization: Bearer <your-token>" \
  http://localhost:8080/api/capability/ping

# 3. Paid — get an L402 challenge (HTTP 402 + Lightning invoice)
curl http://localhost:8080/api/micro
```

Public → Protected → Paid. Three policies, one gateway.

📖 **[Full Quick Start Guide →](docs/getting-started/quickstart.md)**

### Other Install Methods

```bash
# Docker
docker run -v $(pwd)/gateway.yaml:/etc/satgate/gateway.yaml \
  -e ADMIN_TOKEN=my-secret-token -e LIGHTNING_BACKEND=mock \
  -p 8080:8080 ghcr.io/satgate-io/satgate:latest

# Build from source
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

## How It's Different

| | SatGate | Routing Gateways | Traditional API Gateways |
|---|---|---|---|
| **Primary concern** | Economic governance | Provider routing | Traffic management |
| **Budget enforcement** | Hard caps (blocked at limit) | Soft alerts only | ❌ |
| **MCP cost attribution** | Per-tool granularity | ❌ | ❌ |
| **Credential model** | Macaroons (delegatable) | API keys | API keys / OAuth |
| **Agent delegation** | Sub-tokens with reduced budgets | ❌ | ❌ |
| **Micropayments** | L402 Lightning-native | ❌ | ❌ |
| **Works alongside** | — | ✅ Use together | ✅ Use together |

## Architecture

```
┌──────────────────────────────────────────────────┐
│                    SatGate                        │
│                                                   │
│  Request → Route Match → Policy Check → Proxy    │
│                             │                     │
│              ┌──────────────┼──────────────┐     │
│              │              │              │      │
│          [public]    [capability]      [l402]     │
│          pass        verify token     verify     │
│                      check budget     payment    │
│                      log MCP tool     + token    │
└──────────────────────────────────────────────────┘
```

**Key Concepts:**

- **Macaroons**: Bearer tokens with embedded caveats (expiry, scope, budget, IP). Not API keys — they support *delegation* without server roundtrips.
- **Delegation**: Agent A gives Agent B a sub-token with reduced permissions and a $50 budget cap. B can't escalate.
- **MCP Parsing**: SatGate reads MCP JSON-RPC payloads to attribute costs to specific tool calls, not just HTTP endpoints.
- **L402**: HTTP 402 + Lightning invoice for machine-to-machine payments. The protocol for the agent economy.

## SDKs

| Language | Package | Docs |
|----------|---------|------|
| Python | `pip install satgate` | [README](sdk/python/README.md) |
| JavaScript | `npm install satgate-sdk` | [README](sdk/nodejs/README.md) |

## MCP Proxy (NEW)

SatGate now includes a native **MCP proxy** that governs tool calls for any MCP-compatible agent:

```bash
# Run MCP proxy with 1000-credit budget
satgate-mcp --config satgate-mcp.yaml
```

- **Budget enforcement**: Hard 402 when agents exhaust their allocation
- **Delegation**: Parent agents mint sub-agent tokens with carved budgets
- **Per-tool costs**: `web_search: 5`, `dalle_generate: 50` (wildcard patterns supported)
- **Two transports**: stdio (local sidecar) or SSE/HTTP (remote multi-agent)
- **Three auth modes**: none, static token, macaroon (HMAC chain)

See [`pkg/mcpserver/README.md`](pkg/mcpserver/README.md) for full documentation.

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [Quick Start Guide](docs/getting-started/quickstart.md)
- [Configuration Reference](docs/reference/configuration.md)
- [Production Checklist](docs/operations/production-checklist.md)
- [Kubernetes Deployment](docs/guides/kubernetes.md)
- [LangChain Integration](docs/guides/langchain-integration.md)
- [MCP Gateway Guide](docs/guides/mcp-gateway.md)

## ☁️ SatGate Cloud & Enterprise

> **Self-hosting not your thing?** [SatGate Cloud](https://cloud.satgate.io) is the fully managed version — same gateway, zero ops.

The open-source gateway handles protection and payments. SatGate Cloud adds the control plane:

- 📊 **Observe** — Real-time dashboards, usage attribution, cost center tagging
- 🎚️ **Control** — Budget enforcement with Fiat402 (enterprise credits)
- 🤖 **SatGate Mint** — Zero-touch agent provisioning (K8s, AWS, OIDC)
- 🏢 **Multi-tenant** — Team isolation, RBAC, SSO/SCIM
- 📝 **Audit** — Tamper-evident logging, compliance exports

<a href="https://cloud.satgate.io"><strong>Start Free →</strong></a> (Observe mode is free, unlimited, forever)

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
git clone https://github.com/satgate-io/satgate.git
cd satgate
go mod download
go test ./...
go build -o satgate ./cmd/satgate
```

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.

## Links

- 🌐 [satgate.io](https://satgate.io) — Website
- 📝 [Blog](https://satgate.io/blog) — Technical articles
- 🏢 [Enterprise Governance](https://satgate.io/govern) — CISO/CFO/CTO use cases
- 💰 [Pricing](https://satgate.io/pricing) — Free Observe tier, Pro for enforcement
- 🔒 [Security Model](https://satgate.io/security) — Architecture & compliance
- 🧪 [Sandbox](https://satgate.io/sandbox) — Try without signup
- 📊 [ROI Calculator](https://satgate.io/roi-calculator) — Estimate savings
- ⚖️ [Compare](https://satgate.io/compare) — SatGate vs Zuplo, Bifrost, cloud-native
- 📧 [contact@satgate.io](mailto:contact@satgate.io)

---

<p align="center">
  <sub>Built with ⚡ by <a href="https://satgate.io">SatGate</a> — The Economic Firewall</sub>
</p>

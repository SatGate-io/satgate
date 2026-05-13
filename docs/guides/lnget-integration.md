# SatGate + lnget: Agent Payments End-to-End

[lnget](https://github.com/lightninglabs/lightning-agent-tools) is Lightning Labs' L402-aware HTTP client — like `curl`, but it automatically pays Lightning invoices. SatGate is natively compatible with `lnget`, making it the fastest path to monetizing your APIs for AI agents.

**lnget handles the client side** (agents paying for API access).
**SatGate handles the server side** (enforcing budgets, attributing costs, collecting payments).

Together, they form the complete agent commerce stack.

## How It Works

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   AI Agent   │         │   SatGate   │         │  Your API   │
│  (+ lnget)   │────────▶│   Gateway   │────────▶│  (upstream) │
│              │◀────────│             │◀────────│             │
└─────────────┘         └─────────────┘         └─────────────┘
       │                       │
       │  1. GET /api/data     │
       │─────────────────────▶│
       │                       │
       │  2. 402 Payment       │
       │     Required          │
       │  + Lightning invoice  │
       │  + Macaroon           │
       │◀─────────────────────│
       │                       │
       │  3. (lnget pays       │
       │   invoice via LN)     │
       │                       │
       │  4. GET /api/data     │
       │  + Authorization:     │
       │    L402 <macaroon>    │
       │    :<preimage>        │
       │─────────────────────▶│
       │                       │
       │  5. 200 OK + data     │
       │◀─────────────────────│
```

1. Agent requests a resource through SatGate
2. SatGate returns `402 Payment Required` with a Lightning invoice and macaroon
3. `lnget` automatically pays the invoice via the agent's Lightning backend
4. `lnget` retries the request with the L402 authorization header (macaroon + preimage)
5. SatGate returns access with a paid-call receipt and preserves it for Evidence Pack export
5. SatGate verifies payment, proxies request to upstream, returns response

The agent code doesn't change — `lnget` handles steps 2-4 transparently.

## Quick Start

### Prerequisites

- SatGate gateway running with L402 (Charge) mode enabled
- A Lightning backend (LND, Phoenixd, or mock for testing)
- `lnget` installed on the agent's machine

### 1. Install lnget

```bash
# Via Lightning Agent Kit (Claude Code)
claude plugin marketplace add lightninglabs/lightning-agent-tools
claude plugin install lightning-agent-tools@lightninglabs

# Via npx
npx -y @lightninglabs/lnget https://your-api.example.com/resource

# From source
git clone https://github.com/lightninglabs/lightning-agent-tools.git
cd lightning-agent-tools
go install ./skills/lnget/...
```

### 2. Configure SatGate for L402 (Charge Mode)

Add an L402-gated route to your `gateway.yaml`:

```yaml
# gateway.yaml
lightning:
  provider: lnd
  lnd:
    host: localhost:10009
    macaroonPath: /path/to/admin.macaroon
    tlsCertPath: /path/to/tls.cert

upstreams:
  - name: paid-api
    url: http://localhost:8080
    paths:
      - path: /api/v1/*
    l402:
      enabled: true
      priceSats: 100          # 100 sats per request
      description: "Premium API access"
```

Start the gateway:

```bash
satgate-gateway -config gateway.yaml
```

### 3. Make a Paid Request with lnget

```bash
# lnget handles the 402 → pay → retry flow automatically
lnget https://your-gateway.example.com/api/v1/data

# Set a per-request spending ceiling
lnget --max-cost 500 https://your-gateway.example.com/api/v1/expensive-query

# Verbose output to see the L402 negotiation
lnget -v https://your-gateway.example.com/api/v1/data
```

That's it. The agent gets data, the API gets paid.

## SatGate Adds What lnget Can't

`lnget` gives agents the ability to pay. SatGate gives API operators the ability to **govern**.

| Capability | lnget | SatGate |
|-----------|-------|---------|
| Pay L402 invoices | ✅ | — |
| Per-request cost ceiling | ✅ (`--max-cost`) | ✅ (per-route pricing) |
| **Cumulative budget enforcement** | ❌ | ✅ |
| **Per-agent spend tracking** | ❌ | ✅ |
| **Per-tool cost attribution (MCP)** | ❌ | ✅ |
| **Token delegation hierarchy** | ❌ | ✅ |
| **Instant token revocation** | ❌ | ✅ |
| **Cost center rollups** | ❌ | ✅ |
| **Multi-tenant isolation** | ❌ | ✅ |
| **Observe → Control → Charge modes** | ❌ | ✅ |

`lnget --max-cost` is a seatbelt. SatGate is the traffic system — lights, lanes, speed limits, and cameras.

## Advanced: Progressive Policy Modes

The real power is SatGate's three-mode progression. Start with visibility, add controls when you're ready, monetize when it makes sense.

### Observe Mode (Free)

See all agent traffic — including `lnget` requests — without blocking anything:

```yaml
upstreams:
  - name: my-api
    url: http://localhost:8080
    paths:
      - path: /api/*
    policy: observe    # Log everything, block nothing
```

Agents using `lnget` or plain `curl` both work. You get full paid-call receipts and Evidence Pack export in the dashboard.

### Control Mode (Fiat402)

Enforce budgets with fiat-denominated credits — no Lightning required on the client side:

```yaml
upstreams:
  - name: my-api
    url: http://localhost:8080
    paths:
      - path: /api/*
    policy: control
    fiat402:
      costPerRequest: 0.003   # $0.003 per call
```

Agents authenticate with macaroon tokens that have baked-in budget ceilings. When the budget is exhausted, SatGate returns `402` — but with a "budget exceeded" message, not a Lightning invoice.

### Charge Mode (L402)

Monetize with real Lightning micropayments — this is where `lnget` shines:

```yaml
upstreams:
  - name: my-api
    url: http://localhost:8080
    paths:
      - path: /api/*
    policy: charge
    l402:
      enabled: true
      priceSats: 100
```

Agents with `lnget` pay automatically. Agents without it get a standard `402` response they can handle however they want.

## Dynamic Pricing with MCP

When SatGate sits in front of an [MCP server](./mcp-gateway.md), it can price individual tools differently:

```yaml
mcp:
  costProfiles:
    - tool: "image_generate"
      costPerCall: 0.05        # $0.05 — expensive GPU operation
    - tool: "search_database"
      costPerCall: 0.001       # $0.001 — cheap lookup
    - tool: "*"
      costPerCall: 0.003       # Default for unlisted tools
```

An agent using `lnget` against an MCP endpoint pays per-tool prices automatically. SatGate tracks which agent called which tool, how much it cost, and whether it's within budget.

## Macaroon Compatibility

SatGate emits standard [libmacaroon V2 binary format](https://github.com/go-macaroon/macaroon) macaroons in L402 challenges, ensuring compatibility with `lnget` and any L402-compliant client. Both V2 binary and JSON formats are accepted for verification.

The L402 challenge header follows the standard format:

```
WWW-Authenticate: L402 macaroon="<base64-encoded-macaroon>", invoice="<lightning-invoice>"
```

## Example: Full Agent Commerce Loop

Two agents, one API, zero human intervention:

```bash
# Agent A: Hosts a paid sentiment analysis API behind SatGate
# (SatGate config with L402 enabled on /api/sentiment)

# Agent B: Consumes the API using lnget
lnget https://agent-a-gateway.example.com/api/sentiment \
  -d '{"text": "SatGate is the economic firewall for the agent economy"}'

# Response: {"sentiment": "positive", "confidence": 0.94}
# Payment: 100 sats, settled in <1 second
```

Agent B paid Agent A. No signup, no API key exchange, no invoice processing. Lightning settled it in milliseconds. SatGate issued a paid-call receipt, attributed the cost, and enforced Agent A's pricing policy.

This is machine-to-machine commerce.

## Comparison: SatGate vs Aperture

Lightning Labs also offers [Aperture](https://github.com/lightninglabs/aperture), an L402 reverse proxy. See our [detailed comparison](../SATGATE_VS_APERTURE.md) for the full breakdown, but the short version:

| Feature | Aperture | SatGate |
|---------|----------|---------|
| L402 gating | ✅ | ✅ |
| Receipts & Evidence Packs | ❌ | ✅ |
| Budget enforcement | ❌ | ✅ |
| Per-agent attribution | ❌ | ✅ |
| MCP cost profiles | ❌ | ✅ |
| Multi-tenant | ❌ | ✅ |
| Fiat402 (enterprise) | ❌ | ✅ |
| Token delegation | ❌ | ✅ |

Aperture is a toll booth. SatGate is the economic firewall.

## Next Steps

- [Quickstart Guide](../getting-started/quickstart.md) — Get SatGate running in 5 minutes
- [MCP Gateway Guide](./mcp-gateway.md) — Gate MCP servers with per-tool pricing
- [Lightning Provider Config](../configuration/lightning-providers.md) — Set up LND, Phoenixd, or mock backends
- [SatGate vs Aperture](../SATGATE_VS_APERTURE.md) — Detailed competitive comparison

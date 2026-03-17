# Core Concepts

## The Three Layers

SatGate operates in three layers:

### Layer 0 — Protection (Default)
Every request through SatGate can be verified using **macaroon tokens** — cryptographic bearer tokens that are verified locally in sub-millisecond time. No external auth service required.

### Layer 1 — Economic Policy (Per Route)
Each route gets one of three economic policies:

| Policy | What It Does | Use Case |
|--------|-------------|----------|
| **Observe** (`chargeback`) | Verify → allow → meter/log | "Show me what agents are spending" |
| **Control** (`fiat402`) | Verify → check budget → allow/deny | "Cap each team at $500/month" |
| **Charge** (`l402`) | Verify → require payment → allow | "Charge 10 sats per API call" |

### Layer 2 — MCP Awareness
SatGate can parse MCP (Model Context Protocol) JSON-RPC payloads to attribute costs at the **tool level**, not just the endpoint level. See the [MCP Gateway Guide](../guides/mcp-gateway.md).

## Macaroon Tokens

Unlike API keys, macaroon tokens are:
- **Cryptographically verifiable** — no database lookup needed
- **Delegatable** — Agent A can give Agent B a token with reduced permissions
- **Caveat-bearing** — embed expiry, scope, budget, IP binding directly in the token
- **Unforgeable** — built on HMAC chains; can't escalate permissions

## Upstreams

Upstreams are the backend services SatGate proxies to. Each upstream is a named HTTP(S) endpoint:

```yaml
upstreams:
  my-api:
    url: "https://api.example.com"
    timeout: 30s
```

## Routes

Routes match incoming requests and apply policies. Routes are evaluated in order — first match wins.

```yaml
routes:
  - name: my-route
    match:
      pathPrefix: /api/
    upstream: my-api
    policy:
      kind: capability
```

## API Endpoints

SatGate exposes management APIs on the same port as the proxy:

- `/api/capability/mint` — Create tokens (admin)
- `/api/capability/validate` — Validate tokens
- `/api/capability/delegate` — Delegate tokens with reduced permissions
- `/api/governance/ban` — Revoke tokens
- `/api/governance/graph` — Token lineage visualization

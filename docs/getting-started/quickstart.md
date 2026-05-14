# Quick Start

Get SatGate running in under 2 minutes.

## Developer primitive

For application code, prefer the three-call primitive: `issue` a scoped capability, `pay` or invoke upstream with a max budget, then `verify` the receipt. The local gateway quickstart below preserves OSS token endpoints for compatibility and local testing.

## 1. Download

```bash
# macOS (Apple Silicon)
curl -L https://github.com/SatGate-io/satgate/releases/latest/download/satgate-darwin-arm64 -o satgate

# macOS (Intel)
curl -L https://github.com/SatGate-io/satgate/releases/latest/download/satgate-darwin-amd64 -o satgate

# Linux (x86_64)
curl -L https://github.com/SatGate-io/satgate/releases/latest/download/satgate-linux-amd64 -o satgate

chmod +x satgate
```

Or build from source:
```bash
git clone https://github.com/SatGate-io/satgate.git
cd satgate
go build -o satgate ./cmd/satgate/
```

## 2. Configure

Create `gateway.yaml`:

```yaml
version: 1

server:
  listen: ":8080"

admin:
  token: "my-admin-token"

# Optional: enable mock settlement for local paid-route tests.
lightning:
  provider: mock

upstreams:
  httpbin:
    url: "https://httpbin.org"

routes:
  - name: public-health
    match:
      pathPrefix: /health
    upstream: httpbin
    policy:
      kind: public

  - name: protected-api
    match:
      pathPrefix: /api
    upstream: httpbin
    policy:
      kind: capability

  # Optional paid route. Use this only when testing settlement adapters;
  # the primary SatGate contract remains govern/enforce/prove.
  - name: paid-endpoint
    match:
      pathPrefix: /premium
    upstream: httpbin
    policy:
      kind: l402
      priceSats: 10
```

## 3. Run

```bash
export ADMIN_TOKEN=my-admin-token
./satgate --config gateway.yaml
```

SatGate starts on port **8080**.

## 4. Test

```bash
# Public route — no auth needed
curl http://localhost:8080/health

# Mint a capability token through the OSS gateway compatibility endpoint
TOKEN=$(curl -s -X POST http://localhost:8080/api/capability/mint \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' | jq -r '.token')

# Use the token on a protected route
curl http://localhost:8080/api/anything \
  -H "Authorization: Bearer $TOKEN"

# Optional paid route — returns 402 Payment Required with a mock settlement challenge; paid retry returns a paid-call receipt for the Evidence Pack
curl -i http://localhost:8080/premium/anything
```

## Next Steps

- [Route Configuration](../configuration/routes.md)
- [Policy & Scope](../configuration/policy-scope.md)
- [MCP Gateway Guide](../guides/mcp-gateway.md)
- [Lightning Providers](../configuration/lightning-providers.md) — optional paid-route settlement
- [Raw HTTP issue/pay/verify](../guides/raw-http.md)
- [Capability Schema](../reference/capability-schema.md)
- [Receipt Schema](../reference/receipt-schema.md)
- [API Overview](../api/overview.md)
- [SatGate Cloud](https://cloud.satgate.io) — managed version, free Observe tier

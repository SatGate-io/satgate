# Your First Route

This guide walks through configuring your first SatGate route step by step.

## 1. Start Simple — Public Route

```yaml
version: 1

server:
  listen: ":8080"

admin:
  token: "my-admin-token"

lightning:
  provider: mock

upstreams:
  backend:
    url: "https://httpbin.org"

routes:
  - name: public-api
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
```

```bash
./satgate --config gateway.yaml
curl http://localhost:8080/anything  # Proxied to httpbin, no auth
```

## 2. Add Protection — Capability Route

Add a protected route above the public one (routes match in order):

```yaml
routes:
  - name: protected-api
    match:
      pathPrefix: /api/
    upstream: backend
    policy:
      kind: capability

  - name: public-fallback
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
```

Now `/api/*` requires a valid macaroon token:

```bash
# This fails with 401
curl http://localhost:8080/api/anything

# Mint a token
TOKEN=$(curl -s -X POST http://localhost:8080/api/capability/mint \
  -H "Authorization: Bearer my-admin-token" \
  -H "Content-Type: application/json" \
  -d '{}' | jq -r '.token')

# This works
curl http://localhost:8080/api/anything \
  -H "Authorization: Bearer $TOKEN"
```

## 3. Add Payments — L402 Route

```yaml
routes:
  - name: premium-api
    match:
      pathPrefix: /premium/
    upstream: backend
    policy:
      kind: l402
      priceSats: 10

  - name: protected-api
    match:
      pathPrefix: /api/
    upstream: backend
    policy:
      kind: capability

  - name: public-fallback
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
```

Now `/premium/*` returns HTTP 402 with a Lightning invoice. Pay the invoice to get access.

## 4. Add Budget Enforcement — Fiat402 Route

```yaml
  - name: budget-api
    match:
      pathPrefix: /controlled/
    upstream: backend
    policy:
      kind: fiat402
      costCredits: 5
      pay:
        mode: fiat402
        price: 0.05
        unit: USD
        enforceBudget: true
        costCenterHeader: X-Cost-Center
```

Agents are stopped when their budget is spent. No alerts — blocked.

## Next Steps

- [Route Configuration](../configuration/routes.md) — full route matching options
- [Policy & Scope](../configuration/policy-scope.md) — policy details
- [Lightning Providers](../configuration/lightning-providers.md) — configure real payments

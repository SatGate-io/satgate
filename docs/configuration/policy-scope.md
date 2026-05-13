# Policy & Scope Configuration

## Policy Kinds

SatGate supports six policy kinds organized in two layers:

### Layer 0 — Protection (No Economics)
- **`public`** — No auth required. Use for health checks, docs, public endpoints.
- **`deny`** — Block all requests. Use for deprecated or disabled routes.
- **`capability`** (aliases: `protected`, `protect`) — Requires a valid macaroon token. Verifies cryptographically, sub-millisecond, no external auth calls.

### Layer 1 — Economic Policies
- **`chargeback`** (aliases: `observe`, `audit`) — Verify token → allow request → emit receipts and meter usage. Non-blocking. Good for initial rollout.
- **`fiat402`** (aliases: `control`, `budget`) — Verify token → check budget → allow or block. Hard enforcement.
- **`control`** requires a `pay` block in the route config.
- **`l402`** (aliases: `charge`, `monetize`) — Verify token → require Lightning payment proof → allow. Requires `priceSats` or `pay` configuration.

## Scopes

Macaroon tokens can carry scope caveats. When a route specifies `scope`, the token must include that scope:

```yaml
routes:
  - name: read-api
    match:
      pathPrefix: /api/read
    upstream: backend
    policy:
      kind: capability
      scope: read

  - name: admin-api
    match:
      pathPrefix: /api/admin
    upstream: backend
    policy:
      kind: capability
      scope: admin
```

Scopes are enforced via macaroon caveats — agents cannot escalate their own scope.

## Cost Credits

For `fiat402`/`control` mode, you can assign different credit costs per route:

```yaml
routes:
  - name: expensive-endpoint
    match:
      pathPrefix: /api/generate
    upstream: backend
    policy:
      kind: fiat402
      costCredits: 50
      pay:
        mode: fiat402
        price: 0.50
        unit: USD
        enforceBudget: true
        costCenterHeader: X-Cost-Center

  - name: cheap-endpoint
    match:
      pathPrefix: /api/search
    upstream: backend
    policy:
      kind: fiat402
      costCredits: 1
      pay:
        mode: fiat402
        price: 0.01
        unit: USD
        enforceBudget: true
```

## Pay Policy

Routes with economic policies (`chargeback`, `fiat402`, `l402`) can include a `pay` block:

```yaml
policy:
  kind: l402
  pay:
    mode: l402              # chargeback, fiat402, or l402
    price: 10.0             # Price per request
    unit: sats              # sats, USD, or credits
    scope: read             # Required scope
    costCenterHeader: X-Cost-Center  # Header for cost attribution
    enforceBudget: true     # Enforce tenant budgets
    creditsPerSat: 1        # L402 → credits conversion rate
```

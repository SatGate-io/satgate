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

Macaroon tokens can carry scope caveats. When a route specifies `scope`, every scope caveat on the token must grant that scope:

```yaml
routes:
  - name: read-api
    match:
      pathPrefix: /api/read
    upstream: backend
    policy:
      kind: capability
      scope: api:read

  - name: admin-api
    match:
      pathPrefix: /api/admin
    upstream: backend
    policy:
      kind: capability
      scope: api:admin
```

Scopes are enforced via macaroon caveats — agents cannot escalate their own scope.

### How a scope is matched

A scope is one or more segments joined by `:`, such as `api:read` or
`api:users:read`. The leading segments are its namespace. A required scope is
granted by a token scope that is:

| Token scope | Grants | Does not grant |
|-------------|--------|----------------|
| the same scope, e.g. `api:read` | `api:read` | `api:write`, `api:read:users` |
| `<namespace>:*`, e.g. `api:*` | every scope that starts with `api:` (`api:read`, `api:users:read`) | `api` itself, `mcp:search`, `l402`, `read` |
| `api:users:*` | `api:users:read` | `api:orders:read` |
| `*` | every scope | |

A wildcard never leaves its namespace: `api:*` grants nothing outside `api:`,
and `mcp:*` grants nothing outside `mcp:`. Comparison is exact and
case-sensitive. A scope that is empty, contains whitespace or a comma, has an
empty segment (`api::read`, `api:`), or has `*` anywhere other than as the
whole last segment, grants nothing.

A token with no scope caveat is not limited by scope. When a token carries
several scope caveats (for example after delegation), each one must grant the
required scope, so delegation can only narrow. On the HTTP gateway each scope
caveat is one scope; a comma-separated list in one caveat grants nothing.

The built-in `/api/capability/admin` endpoint requires `api:capability:admin`
and a token with at least one scope caveat.

For MCP tools, see [MCP Gateway](../guides/mcp-gateway.md#tool-scopes).

### Token expiry

Every `expires` caveat must hold an absolute time, in one of two forms:

- Unix time in milliseconds, digits only, e.g. `expires = 1793491200000`.
  This is the form the gateway writes when it mints a token.
- An RFC 3339 time with a zone, e.g. `expires = 2026-11-01T00:00:00Z`.

The token is refused once any of its `expires` caveats is in the past. Any
other value refuses the token, including a duration such as `24h` or `5m`, a
date without a time, a signed or fractional number, and an empty value. The
delegation endpoint refuses to add such a caveat. A legacy `time < N` caveat
is also accepted, where `N` is Unix time in seconds, digits only.

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
    scope: api:read         # Required scope
    costCenterHeader: X-Cost-Center  # Header for cost attribution
    enforceBudget: true     # Enforce tenant budgets
    creditsPerSat: 1        # L402 → credits conversion rate
```

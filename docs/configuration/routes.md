# Route Configuration

Routes define how SatGate matches incoming requests and which policy to apply.

## Route Structure

```yaml
routes:
  - name: my-api              # Unique route name (required; duplicates refused at startup)
    match:
      pathPrefix: /api/        # Path starts with this value
      # pathExact: /api/health # Path equals this value
      # pathRegex: "^/api/v[0-9]+" # Go regexp matches the path
      methods: [GET, POST]     # Optional: method is one of these (exact, upper case)
      headers:                 # Optional: each header sent once with this value
        X-Version: "2"         # Exact value
        X-Custom: "~^prod-"   # Go regexp (prefix with ~)
    upstream: my-backend       # Target upstream name
    rewrite: /v2               # Optional: replace the path sent upstream (takes precedence over stripPrefix)
    stripPrefix: true          # Optional: strip matched prefix (no effect when rewrite is set)
    policy:
      kind: capability         # Policy type (see below)
      scope: api:read          # Optional: required macaroon scope (see policy-scope.md)
      costCredits: 10          # Optional: credits per request (for budget enforcement)
    mcp:                       # Optional: MCP JSON-RPC parsing
      enabled: true
      maxBodySize: 1048576     # Max body to buffer (default 1MB)
```

## Policy Types

SatGate uses a layered policy model:

### No Verification
| Kind | Description |
|------|-------------|
| `public` | No authentication required |
| `deny` | Block all requests |

### Layer 0 — Default Protection (verify only, no economics)
| Kind | Aliases | Description |
|------|---------|-------------|
| `capability` | `protected`, `protect` | Macaroon verification — cryptographic proof of authorization |

### Layer 1 — Economic Policies
| Kind | Aliases | Description |
|------|---------|-------------|
| `chargeback` | `observe`, `audit` | Verify → allow → emit receipts and meter usage |
| `fiat402` | `control`, `budget` | Verify → enforce budget → allow |
| `l402` | `charge`, `monetize` | Verify → require Lightning payment → return paid-call receipt → allow |

**Example: Three-tier pricing**

```yaml
routes:
  - name: free-api
    match:
      pathPrefix: /api/free
    upstream: backend
    policy:
      kind: public

  - name: authenticated-api
    match:
      pathPrefix: /api/v1
    upstream: backend
    policy:
      kind: capability
      scope: api:read

  - name: paid-api
    match:
      pathPrefix: /api/premium
    upstream: backend
    policy:
      kind: l402
      priceSats: 10
```

## Route Matching

Routes are evaluated **in order** — the first match wins. Put more specific routes before general ones.

A route matches a request only when the request meets **every** matcher the
route sets; a request that fails one is tested against the next route.

- `pathPrefix` — the request path starts with this value (a trailing `*` is ignored)
- `pathExact` — the request path equals this value
- `pathRegex` — the Go regular expression matches the request path (anywhere in it, unless anchored with `^`/`$`)
- `methods` — the request method equals one entry exactly (entries must be non-empty method tokens with no lower-case letter)
- `headers` — each named header is sent exactly once with exactly this value, or, for a value starting with `~`, a value the Go regular expression after the `~` matches. `Host` and the framing and hop-by-hop headers (`Content-Length`, `Transfer-Encoding`, `Connection`, `Keep-Alive`, `Proxy-Connection`, `TE`, `Trailer`, `Upgrade`) cannot be matched and are refused at startup

A route that sets `rateLimit` is refused at startup: the gateway does not
enforce per-route rate limits.

Request paths are checked before any route is matched, and before an
`OPTIONS` request is answered: a path with a `.` or `..` segment, an
encoded slash or backslash, a backslash, a NUL or a malformed percent
sequence (as sent, or after one percent-decoding) is refused with `400`.
The path sent upstream, after `rewrite` or `stripPrefix` and the upstream
URL's own path, is checked the same way before the route's policy runs,
and a `rewrite` or upstream URL path that fails on its own is refused at
startup. See [Route matching and request paths](../reference/configuration.md#route-matching)
for the exact rules.

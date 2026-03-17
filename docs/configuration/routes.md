# Route Configuration

Routes define how SatGate matches incoming requests and which policy to apply.

## Route Structure

```yaml
routes:
  - name: my-api              # Unique route name (required)
    match:
      pathPrefix: /api/        # Match by path prefix
      # pathExact: /api/health # Or exact path match
      # pathRegex: "^/api/v[0-9]+" # Or regex match
      methods: [GET, POST]     # Optional: restrict to HTTP methods
      headers:                 # Optional: match by headers
        X-Version: "2"         # Exact match
        X-Custom: "~^prod-"   # Regex match (prefix with ~)
    upstream: my-backend       # Target upstream name
    rewrite: /v2               # Optional: rewrite path before proxying
    stripPrefix: true          # Optional: strip matched prefix
    policy:
      kind: capability         # Policy type (see below)
      scope: read              # Optional: required macaroon scope
      costCredits: 10          # Optional: credits per request (for budget enforcement)
    rateLimit:                 # Optional: per-route rate limiting
      requestsPerMinute: 60
      burstSize: 10
      key: ip                  # ip, token, or header:X-Custom
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
| `chargeback` | `observe`, `audit` | Verify → allow → meter and log usage |
| `fiat402` | `control`, `budget` | Verify → enforce budget → allow |
| `l402` | `charge`, `monetize` | Verify → require Lightning payment → allow |

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
      scope: read

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

Matching supports:
- `pathPrefix` — matches if request path starts with this value
- `pathExact` — matches if request path equals this value exactly  
- `pathRegex` — matches against a Go regex pattern
- `methods` — restricts to specific HTTP methods
- `headers` — matches specific header values (supports regex with `~` prefix)

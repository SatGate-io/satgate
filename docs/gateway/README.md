# SatGate Gateway Mode

SatGate v2.0 introduces **Gateway Mode** - a full reverse proxy with L402 enforcement, designed to protect any upstream API without code changes.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         SatGate Gateway                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────┐         ┌─────────────────────┐           │
│  │    Data Plane       │         │    Admin Plane      │           │
│  │    (Port 8080)      │         │    (Port 9090)      │           │
│  │                     │         │                     │           │
│  │  • Route matching   │         │  • /auth/decide     │           │
│  │  • Policy engine    │         │  • /api/governance  │           │
│  │  • L402 challenges  │         │  • Token minting    │           │
│  │  • Streaming proxy  │         │  • Kill switch      │           │
│  │  • Metering         │         │  • Telemetry        │           │
│  └─────────┬───────────┘         └─────────────────────┘           │
│            │                                                        │
│            ▼                                                        │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Upstream Pool                             │   │
│  │  (Config-defined only - SSRF prevention)                     │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Two Deployment Modes

### 1. Gateway Mode (SMB/Startups)

SatGate acts as a full reverse proxy, handling all traffic:

```yaml
# satgate.gateway.yaml
version: 1

upstreams:
  customer_api:
    url: "http://internal-api:8080"

routes:
  - name: "premium"
    match:
      pathPrefix: "/v1/premium/"
    upstream: "customer_api"
    policy:
      kind: "l402"
      tier: "premium"
      priceSats: 1000
      scope: "api:premium:*"
```

Start with:
```bash
SATGATE_RUNTIME=gateway node proxy/server.js
```

### 2. Ingress Integration (Enterprise)

Use your existing NGINX/Envoy with SatGate's Auth Decision API:

```
Client → NGINX/Envoy → /auth/decide → SatGate
              ↓                          ↓
        (if 200 OK)              L402 validation
              ↓                   + metering
         Upstream
```

See:
- `nginx-openresty.conf` - OpenResty/Lua integration
- `envoy-ext-authz.yaml` - Envoy ext_authz integration

## Auth Decision API

**Endpoint:** `POST /auth/decide`

**Headers Required:**
```
X-Original-Method: GET
X-Original-URI: /v1/premium/insights
X-Original-Host: api.example.com
X-Original-Proto: https
Authorization: L402 <macaroon>:<preimage>  (optional)
```

**Responses:**

| Status | Meaning | Headers |
|--------|---------|---------|
| 200 | Allow | `X-SatGate-Route`, `X-SatGate-Tier`, etc. |
| 402 | Payment Required | `WWW-Authenticate`, `X-L402-Price`, etc. |
| 403 | Forbidden | `X-SatGate-Reason` |
| 429 | Exhausted | `X-Calls-Remaining: 0` |
| 500 | Error | Fail-closed |

## Configuration Reference

See `satgate.gateway.yaml` in the project root for a complete example.

### Key Sections

**Upstreams:** Named backend targets (SSRF prevention - only these can be reached)
```yaml
upstreams:
  my_api:
    url: "http://10.0.2.10:8080"
    passHostHeader: false
    addHeaders:
      X-Internal-Key: "secret"
```

**Routes:** Path matching with policies
```yaml
routes:
  - name: "my-route"
    match:
      pathPrefix: "/v1/"
      methods: ["GET", "POST"]
    upstream: "my_api"
    policy:
      kind: "l402"
      tier: "standard"
      priceSats: 100
      scope: "api:standard:*"
```

**Policies:**
- `public`: No auth, pass through
- `deny`: Block with status code
- `l402`: Lightning payment required
- `capability`: Macaroon token required (no payment)

## Security Features

- **SSRF Prevention**: Upstreams defined in config only
- **Streaming Proxy**: No body buffering (memory-safe)
- **Header Sanitization**: Hop-by-hop headers stripped
- **Fail-Closed**: Auth errors = 503, not 200
- **Two Listeners**: Admin plane separate from data plane

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SATGATE_RUNTIME` | `gateway` or `embedded` | Auto-detect |
| `SATGATE_GATEWAY_CONFIG` | Config file path | `./satgate.gateway.yaml` |
| `L402_ROOT_KEY` | L402 macaroon key | Required in prod |
| `REDIS_URL` | Redis for metering | Optional |
| `GATEWAY_DEBUG` | Enable debug logging | `false` |

## Comparison

| Feature | Gateway Mode | Ingress Integration |
|---------|--------------|---------------------|
| Deployment | Single binary | Sidecar to NGINX/Envoy |
| Complexity | Lower | Higher |
| Performance | Good | Best (native) |
| Customization | Config-based | Full control |
| Best for | SMBs, startups | Enterprise, high-scale |


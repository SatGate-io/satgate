# Configuration Reference

Full reference for `gateway.yaml`. All fields match the Go config structs.

## Top-Level Structure

```yaml
version: 1

server:
  listen: ":8080"
  readTimeout: 30s
  writeTimeout: 30s
  maxRequestBody: 1048576
  trustedProxies: []
  allowedHosts: []
  tenantIsolationEnabled: false
  tenantIsolationRequired: false
  tenantQuotasEnabled: false
  defaultTenantId: ""

admin:
  token: "${ADMIN_TOKEN}"
  capabilityRootKey: "${CAPABILITY_ROOT_KEY}"
  separateListener: ""            # e.g., "127.0.0.1:9090" for separate admin port
  allowedIps: []
  corsAllowedOrigins: []
  rateLimitPerMinute: 0
  rateLimitKeyType: "ip"          # ip, token, global
  rateLimitBackend: "memory"      # memory, redis
  enableDevLogin: false           # NEVER enable in production
  enableSwaggerUI: false
  keyManagement:
    provider: "env"               # env, file, memory, aws-kms, gcp-kms, vault
    algorithm: "RS256"
    rotationDays: 90
    retainPrevious: 2
    requireSigningKey: false

lightning:
  provider: "mock"                # phoenixd, lnd, nwc, alby, lnbits, mock, disabled
  config: {}
  l402RootKey: "${L402_ROOT_KEY}"
  requireInvoiceRecord: true
  verifyWithNode: false

upstreams:
  my-backend:
    url: "https://api.example.com"
    timeout: 30s
    headers: {}                   # Set on every forwarded request; see "What an upstream receives"
    healthCheck:
      path: /health
      interval: 30s
      timeout: 5s
    circuitBreaker:
      maxFailures: 5
      resetTimeout: 30s
      halfOpenMaxReqs: 3
    tls:
      insecureSkipVerify: false
      caCertFile: ""
      clientCertFile: ""
      clientKeyFile: ""

routes:
  - name: my-route
    match:
      pathPrefix: /api/
      methods: [GET, POST]
      headers: {}
    upstream: my-backend
    rewrite: ""
    stripPrefix: false
    policy:
      kind: capability
      scope: ""
      costCredits: 1
      priceSats: 0
      pay:
        mode: ""
        price: 0
        unit: sats
        costCenterHeader: ""
        enforceBudget: false
    transform:
      stripPrefix: ""
      addHeaders: {}
    rateLimit:
      requestsPerMinute: 0
      burstSize: 0
      key: ip
    mcp:
      enabled: false
      maxBodySize: 1048576
```

### Route names

Every route needs a `name`, and route names must be unique (compared
exactly, so `orders` and `Orders` are different names). The name is the
route's identity: an L402 payment challenge is issued for one route name. A
configuration in which two routes share a name is refused at startup with
`route name "<name>" is used by routes <i> and <j>: route names must be
unique`. An agent policy promoted at run time (`/api/agent/promote`) whose
name is already used by a route is refused with `409 route_name_in_use`, and
a saved agent route whose name is in use is not restored at startup.

### What an upstream receives

The gateway forwards a copy of each request it lets through. Before
forwarding, it removes the credentials it consumed or owns:

| Removed | When | Why |
|---|---|---|
| `Authorization` (every value) | The route's policy is `capability` (also the default when no policy kind is set), `l402` or `identity` | The gateway checked this credential to let the request through. |
| `Authorization` values that carry a token this gateway issued | Any other route (`public`, `chargeback`) | A macaroon signed under this gateway's root key, sent as `Bearer <token>`, as the bare token, or as `L402`/`LSAT <macaroon>:<preimage>`, is a SatGate credential, not the upstream's. |
| `X-Admin-Token`, `X-Tenant-ID`, every `X-SatGate-*` header | Always | SatGate reads these itself (admin authentication, tenant selection) or reserves the names; a client's value must not reach the upstream as if SatGate had set it. |
| SatGate's session cookies (`cloud_session`, `__Host-cloud_session`, `__Host-admin_session`, `impersonation`, `__Host-impersonation`, and any cookie whose name starts with `satgate`) | Always | They are SatGate sign-in sessions. Other cookies are passed on exactly as sent. |

On a `public` or `chargeback` route, an `Authorization` value that is not a
token this gateway issued (an upstream API key, a JWT, `Basic` credentials,
a macaroon from another issuer) is passed on unchanged: the gateway did not
check it, and on a route the gateway does not guard it belongs to the
upstream. Every header not listed above is passed on unchanged.

**Giving an upstream its own credentials.** Set them in the upstream's
`headers`. They are set on every request to that upstream after the removal
above, so a configured header replaces a client header of the same name
(the configured value wins):

```yaml
upstreams:
  my-backend:
    url: "https://api.example.com"
    headers:
      Authorization: "Bearer ${MY_BACKEND_TOKEN}"
```

A configuration is refused at load if an upstream's `headers` sets a header
SatGate owns (`X-Admin-Token`, `X-Tenant-ID`, `X-SatGate-*`), has a name that
is not a valid HTTP header name, a value with a line break or other control
character, or two names that differ only in case.

**Compatibility.** Earlier versions forwarded the client's `Authorization`,
`X-Admin-Token`, `X-Tenant-ID` and cookies as sent, and ignored `headers`.
An upstream that relied on receiving the client's SatGate token must now be
given its own credential in `headers`. A configuration whose `headers` names
a header SatGate owns no longer loads.

## Optional Sections

### Redis (HA)
```yaml
redis:
  enabled: false
  addr: "localhost:6379"
  password: ""
  db: 0
```

In the open-source gateway, an enabled Redis holds the L402 consumed-proof
store, shared by every instance that uses it. Without Redis that store is in
process memory and correct for a single instance only. See
[L402 Consumed-Proof Store](../configuration/lightning-providers.md#l402-consumed-proof-store).

### PostgreSQL
```yaml
postgres:
  enabled: false
  url: "postgres://user:pass@localhost:5432/satgate?sslmode=disable"
  maxConnections: 25
  autoMigrate: true
```

### Billing (Enterprise)
```yaml
billing:
  enabled: false
  defaultMode: chargeback       # chargeback, l402, fiat402
  defaultUnit: USD              # sats, USD, credits
```

### Notifications
```yaml
notifications:
  enabled: false
  channels:
    - name: slack-alerts
      type: slack               # slack, discord, http, email
      enabled: true
      url: "https://hooks.slack.com/..."
      alertTypes: [budget_exceeded, token_revoked]
      minSeverity: warning
```

### OpenTelemetry Tracing
```yaml
tracing:
  enabled: false
  serviceName: satgate-gateway
  exporter:
    type: otlp                  # otlp, otlp-http, stdout, none
    endpoint: "localhost:4317"
    insecure: true
  sampling:
    type: parent                # always, never, ratio, parent
    ratio: 1.0
```

### GitOps (Signed Config)
```yaml
gitops:
  enabled: false
  requireSignature: false
  trustedKeysDir: /etc/satgate/keys
  pollInterval: 30s
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `ADMIN_TOKEN` | Admin API bearer token |
| `CAPABILITY_ROOT_KEY` | Macaroon signing key (hex, 32+ bytes) |
| `L402_ROOT_KEY` | L402 macaroon signing key (base64, 32+ bytes) |
| `LIGHTNING_BACKEND` | Lightning provider name |
| `NWC_CONNECTION_STRING` | Nostr Wallet Connect URI |
| `LND_REST_URL` | LND REST API URL |
| `LND_MACAROON` | LND admin macaroon (hex) |

## Defaults

- Listen: `:8080`
- Read/Write timeout: `30s`
- Lightning provider: `mock` (if no env var set)
- SSRF protection: blocks private IPs in upstream URLs
- Policy kinds are aliased (e.g., `observe` → `chargeback`, `protect` → `capability`)

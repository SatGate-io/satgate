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
    headers: {}
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

## Optional Sections

### Redis (HA)
```yaml
redis:
  enabled: false
  addr: "localhost:6379"
  password: ""
  db: 0
```

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

# Configuration Reference

Complete reference for `gateway.yaml` configuration.

## File Structure

```yaml
version: 1

server:
  # Server settings

admin:
  # Admin API settings

lightning:
  # Lightning provider settings

redis:
  # Redis connection

postgres:
  # PostgreSQL connection

upstreams:
  # Backend APIs

routes:
  # Route definitions
```

## Server Settings

```yaml
server:
  listen: ":8080"                # Data plane listen address
  readTimeout: 30s               # HTTP read timeout
  writeTimeout: 30s              # HTTP write timeout
  maxRequestBody: 10485760       # Max request size (10MB)
  
  # Trusted proxies (can set X-Tenant-ID)
  trustedProxies:
    - "10.0.0.0/8"
    - "172.16.0.0/12"
  
  # Tenant isolation
  tenantIsolationEnabled: true   # Enable tenant context extraction
  tenantIsolationRequired: false # Require valid tenant (strict mode)
  tenantQuotasEnabled: true      # Enforce per-tenant quotas
  defaultTenantId: "my-company"  # Default for requests without context
  
  # Host allowlist (empty = allow all)
  allowedHosts:
    - "api.example.com"
    - "api.staging.example.com"
```

## Admin Settings

```yaml
admin:
  token: ""                      # Admin token (set via env ADMIN_TOKEN)
  jwtSecret: ""                  # JWT signing secret (set via env JWT_SECRET)
  jwtExpiry: "15m"               # Access token expiry
  refreshExpiry: "7d"            # Refresh token expiry
  
  # Separate listener for admin API (recommended)
  separateListener: ":9090"
  
  # Rate limiting
  rateLimitPerMinute: 60         # Max requests per minute
  rateLimitBackend: "redis"      # memory | redis
  rateLimitKeyType: "ip"         # ip | token | global
  
  # IP allowlist (empty = allow all)
  allowedIps:
    - "10.0.0.0/8"
    - "192.168.1.0/24"
  
  # CORS settings (required for Dashboard access)
  # SECURITY: Do NOT use wildcard "*" - explicitly list Dashboard URL(s)
  corsAllowedOrigins:
    - "https://dashboard.example.com"
  corsAllowCredentials: true
  
  # Audit hash chain
  auditHashChainKey: ""          # Set via env AUDIT_HASH_CHAIN_KEY
  requireAuditHashChainKey: true # Fail startup if not set
  
  # Security headers
  security:
    enableHSTS: true
    hstsMaxAge: 31536000
    contentSecurityPolicy: "default-src 'self'"
```

## Lightning Settings

### Mock Provider (Protection Mode Only)

```yaml
lightning:
  provider: "mock"
```

### Phoenixd (Production)

```yaml
lightning:
  provider: "phoenixd"
  l402RootKey: "${L402_ROOT_KEY}"      # REQUIRED: 32+ byte secret for macaroon signing
  requireInvoiceRecord: true            # Require invoice in store (fail-closed)
  verifyWithNode: true                  # Verify payment status with Lightning node
  config:
    apiUrl: "http://phoenixd:9740"
    apiPassword: "${PHOENIXD_PASSWORD}"
    tlsVerify: true
```

### LND

```yaml
lightning:
  provider: "lnd"
  l402RootKey: "${L402_ROOT_KEY}"
  requireInvoiceRecord: true
  verifyWithNode: true
  config:
    rpcHost: "lnd:10009"
    macaroonPath: "/certs/admin.macaroon"
    tlsCertPath: "/certs/tls.cert"
```

### L402 Security Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `l402RootKey` | (required) | Secret key for signing L402 macaroons |
| `requireInvoiceRecord` | `true` | Fail-closed if invoice not in store |
| `verifyWithNode` | `true` | Verify payment with Lightning node |

## Redis Settings

```yaml
redis:
  enabled: true
  addr: "redis:6379"
  password: "${REDIS_PASSWORD}"
  db: 0
  poolSize: 10
  maxRetries: 3
```

## PostgreSQL Settings

```yaml
postgres:
  enabled: true
  url: "postgres://user:pass@host:5432/dbname?sslmode=require"
  maxConnections: 20
  autoMigrate: true
  migrationsPath: "/app/migrations"
```

## Billing/Payments Settings

Configure API monetization and cost management. See [Payments Configuration](../configuration/payments.md) for detailed usage.

```yaml
billing:
  enabled: true
  defaultMode: control           # observe | control | l402 | fiat402
  defaultUnit: USD               # USD | sats | credits
  failOpenControl: false         # Allow requests when billing unavailable (control mode only)

  # L402 display settings
  l402:
    displayUnit: USD             # For UI display conversion
    exchangeRate: 0.00001        # sats per USD

  # Control mode settings (Enterprise)
  control:
    budgets:
      enabled: true
    export:
      enabled: true

  # Fiat402 settings (Enterprise)
  fiat402:
    receiptTTL: "1h"             # Receipt token validity
    signingKeyRef: "jwt-key"     # Reference to KeyManager
    replayProtection: both       # expiry | jti | both

  # Alerting (budget notifications)
  alerting:
    enabled: true
    webhooks:
      - url: ""                  # Webhook URL for budget alerts
        secret: ""               # HMAC secret for webhook signatures
        alertTypes: [budget.threshold, budget.exceeded]
```

> **Note:** L402 security settings (`requireInvoiceRecord`, `verifyWithNode`, `l402RootKey`) 
> are configured under `lightning:`, not `billing:`. See [Lightning Settings](#lightning-settings).

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `L402_ROOT_KEY` | Macaroon signing key (32+ bytes) | For L402 |
| `JWT_SIGNING_KEY` | Base64 PEM signing key for Fiat402 receipts | For Fiat402 |
| `DATABASE_URL` | PostgreSQL connection URL (enables Postgres) | Recommended |
| `REDIS_URL` | Redis connection URL (enables Redis) | Recommended |
| `AUDIT_HASH_CHAIN_KEY` | Tamper-evident audit hash-chain key | Recommended/Required (enterprise) |
| `CAPABILITY_ROOT_KEY` | Capability (macaroon) root key | Recommended |

> **Note:** Webhook secrets are configured in `billing.alerting.webhooks[].secret` and can be injected with `${ENV_VAR}` if you prefer.

## Upstreams

```yaml
upstreams:
  my_api:
    url: "https://api.example.com"
    timeout: 30s
    maxIdleConns: 100
    idleConnTimeout: 90s
    
    # Health check
    healthCheck:
      path: "/health"
      interval: 10s
      timeout: 5s
      unhealthyThreshold: 3
    
    # Circuit breaker
    circuitBreaker:
      enabled: true
      threshold: 5
      timeout: 30s
    
    # Headers to add
    headers:
      X-Forwarded-For: "${remote_addr}"
      X-Request-ID: "${request_id}"
```

## Routes

```yaml
routes:
  # Public route (no auth) - exact match
  - name: health
    match:
      pathExact: /health
    upstream: my_api
    policy:
      kind: public
  
  # Protected route (capability token required) - prefix match
  - name: api
    match:
      pathPrefix: /api/
    upstream: my_api
    policy:
      kind: observe
      scope: api:read
    
    # Optional rate limiting
    rateLimit:
      requestsPerMinute: 100
      burstSize: 20
  
  # L402 route (Lightning payment required)
  - name: premium
    match:
      pathPrefix: /premium/
    upstream: my_api
    policy:
      kind: l402
      priceSats: 10          # Satoshis per request
      scope: premium:access

  # Pay route (Chargeback or Fiat402) - Enterprise
  - name: metered-api
    match:
      pathPrefix: /api/metered/
    upstream: my_api
    policy:
      kind: pay
      pay:
        mode: control               # control | fiat402
        unit: USD
        price: 0.01                 # Price per request
        scope: api:metered
        costCenterHeader: "X-Cost-Center"  # Header to extract cost center
        enforceBudget: true         # Enforce tenant budgets
  
  # Deny route (always reject) - regex match
  - name: deprecated
    match:
      pathRegex: "^/v1/old/.*"
    policy:
      kind: deny
      message: "This endpoint is deprecated. Use /v2/ instead."
```

## Route Matching

Routes are matched in order. First match wins.

```yaml
routes:
  # More specific paths first
  - name: api-admin
    match:
      pathPrefix: /api/admin/
    policy:
      kind: observe
      scope: admin:*
  
  # Less specific paths last
  - name: api-read
    match:
      pathPrefix: /api/
    policy:
      kind: observe
      scope: api:read
```

### Match Types

| Type | Syntax | Example | Matches |
|------|--------|---------|---------|
| `pathExact` | Exact string | `/health` | Only `/health` |
| `pathPrefix` | Prefix string | `/api/` | `/api/`, `/api/users`, `/api/v1/items` |
| `pathRegex` | Go regex | `^/v[0-9]+/.*` | `/v1/foo`, `/v2/bar` |

**Note:** Use `pathPrefix` for most cases. Use `pathRegex` only when prefix matching is insufficient.

## Environment Variable Substitution

Use `${VAR}` syntax to substitute environment variables:

```yaml
admin:
  token: "${ADMIN_TOKEN}"
  
postgres:
  url: "${DATABASE_URL}"
```

## Full Example

```yaml
version: 1

server:
  listen: ":8080"
  readTimeout: 30s
  writeTimeout: 30s
  maxRequestBody: 10485760
  tenantIsolationEnabled: true
  tenantIsolationRequired: true
  defaultTenantId: "acme-corp"
  allowedHosts:
    - "api.acme.com"

admin:
  token: "${ADMIN_TOKEN}"
  jwtExpiry: "15m"
  refreshExpiry: "7d"
  separateListener: ":9090"
  rateLimitPerMinute: 60
  rateLimitBackend: redis
  requireAuditHashChainKey: true
  corsAllowedOrigins:
    - "https://dashboard.acme.com"

lightning:
  provider: phoenixd
  config:
    apiUrl: "http://phoenixd:9740"
    apiPassword: "${PHOENIXD_PASSWORD}"

billing:
  enabled: true
  defaultMode: control
  defaultUnit: USD
  failOpenControl: false
  control:
    budgets:
      enabled: true
    export:
      enabled: true
  alerting:
    enabled: true
    webhooks:
      - url: "https://slack.com/api/webhooks/..."
        secret: "${BUDGET_ALERT_SECRET}"
        alertTypes: [budget.threshold, budget.exceeded]

redis:
  enabled: true
  addr: "redis:6379"
  password: "${REDIS_PASSWORD}"

postgres:
  enabled: true
  url: "${DATABASE_URL}"
  autoMigrate: true

upstreams:
  backend:
    url: "http://backend-service:8080"
    timeout: 30s
    healthCheck:
      path: /health
      interval: 10s

routes:
  # Public health check
  - name: public-health
    match:
      pathExact: /health
    upstream: backend
    policy:
      kind: public

  # Capability-protected API
  - name: protected-api
    match:
      pathPrefix: /api/
    upstream: backend
    policy:
      kind: observe
      scope: api:read
    rateLimit:
      requestsPerMinute: 100

  # L402 paid API (Lightning)
  - name: premium-api
    match:
      pathPrefix: /api/premium/
    upstream: backend
    policy:
      kind: l402
      priceSats: 50
      scope: premium:access

  # Control mode metered API (Enterprise)
  - name: metered-api
    match:
      pathPrefix: /api/metered/
    upstream: backend
    policy:
      kind: pay
      pay:
        mode: control
        unit: USD
        price: 0.001
```

## Related Documentation

- [Dashboard Guide](../guides/dashboard.md) — Web UI for management and monitoring
- [Payments Configuration](../configuration/payments.md) — Detailed payments setup
- [Enterprise Deployment](../enterprise/DEPLOYMENT_GUIDE.md) — Production deployment
- [Feature Matrix](../enterprise/FEATURE_MATRIX.md) — OSS vs Enterprise features


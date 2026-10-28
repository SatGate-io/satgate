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
forwarding, it removes the credentials it consumed or owns, from the header
block and from request trailers alike:

| Removed | When | Why |
|---|---|---|
| `Authorization` (every value) | The route's policy is `capability` (also the default when no policy kind is set), `l402` or `identity` | The gateway checked this credential to let the request through. |
| Each credential in an `Authorization` value that carries a token this gateway issued | Any other route (`public`, `chargeback`) | A macaroon signed under this gateway's root key is a SatGate credential, not the upstream's, in whatever form it is sent (see below). |
| `X-Admin-Token`, `X-Tenant-ID`, every `X-SatGate-*` header | Always | SatGate reads these itself (admin authentication, tenant selection) or reserves the names; a client's value must not reach the upstream as if SatGate had set it. |
| SatGate's session cookies (`cloud_session`, `__Host-cloud_session`, `__Host-admin_session`, `impersonation`, `__Host-impersonation`, and any cookie whose name starts with `satgate`) | Always | They are SatGate sign-in sessions. Other cookies are passed on exactly as sent. |

**Forms of a gateway token that are removed.** On `public` and `chargeback`
routes the gateway finds a token it issued anywhere in an `Authorization`
value: under any scheme or none (`Bearer`, `Basic`, `Digest`, `L402`, `LSAT`
or another, in any letter case, or the bare token), after any whitespace
(spaces, tabs, other Unicode spaces), quoted or as a parameter value
(`token="..."`), as the macaroon of `L402`/`LSAT <macaroon>:<preimage>`, in
the JSON or the binary macaroon format, in standard or URL-safe base64,
padded or not, percent-encoded, or inside base64-encoded `Basic`
credentials. Every `Authorization` line is checked. A value that joins
several credentials with commas is split into them (a comma inside a quoted
string, or between one credential's parameters, does not split); each
credential that carries a gateway token is removed, the others are passed on
joined with `, `, and a value left with no credential is dropped. A value
that carries no gateway token is passed on byte for byte. An
`Authorization` value that is not a token this gateway issued (an upstream
API key, a JWT, `Basic` credentials, a macaroon from another issuer) is
passed on unchanged: the gateway did not check it, and on a route the
gateway does not guard it belongs to the upstream.

**Request trailers.** Trailers lose what the table above removes, and also
every hop-by-hop name and every name the upstream has a configured header
for, so a trailer can neither carry a SatGate credential nor contradict a
configured header. They are cleaned as the request leaves the gateway and
again once its body has been read, just before the trailers are written.
Other trailers are passed on as before: when the gateway reads a request's
body before forwarding it (an MCP-enabled route), its trailers are
forwarded; otherwise trailer values arrive only while the body is being
forwarded, and they are not forwarded.

Every header not listed above is passed on unchanged, except that the
reverse proxy removes hop-by-hop headers and adds the client's address to
`X-Forwarded-For`, as it always has.

**Giving an upstream its own credentials.** Set them in the upstream's
`headers`. They are set on every request to that upstream as the request
leaves the gateway, after every other change, so each arrives with exactly
its configured value and as the only value of that header: a configured
header replaces a client header of the same name, even one the client named
in its `Connection` header. Names are sent in canonical capitalization
(`x-api-key` is sent as `X-Api-Key`); HTTP header names are not
case-sensitive.

```yaml
upstreams:
  my-backend:
    url: "https://api.example.com"
    headers:
      Authorization: "Bearer ${MY_BACKEND_TOKEN}"
```

Accepted: any valid header name not refused below, for example
`Authorization`, `Cookie`, `User-Agent`, `Accept`, `Accept-Encoding`,
`Expect`, `Via` or a custom `X-...` name. `Host` is accepted and is sent as
the request's Host (instead of the upstream URL's host), provided its value
is a plain ASCII host or `host:port`: letters, digits and
``-._~!$&'()*+,;=:[]``, not empty. A non-ASCII name or an IPv6 zone (`%`) is
refused, because the HTTP client would rewrite it.

Refused when the configuration is loaded (and when a gateway is built from a
configuration in code), with an error that names the upstream and the
entry, never the value:

<!-- upstream-headers-refused:start -->
| Name | Why it is refused |
|---|---|
| `X-Admin-Token`, `X-Tenant-ID`, `X-SatGate-*` | Owned by SatGate (see the table above). |
| `Connection`, `Keep-Alive`, `Proxy-Authenticate`, `Proxy-Authorization`, `Proxy-Connection`, `Te`, `Trailer`, `Transfer-Encoding`, `Upgrade` | Hop-by-hop: they apply to one connection. The reverse proxy removes them and the HTTP client writes its own, so a configured value would not arrive. |
| `X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`, `Forwarded` | Managed by the proxy for each request: they describe the client's connection (the proxy adds the client's address to `X-Forwarded-For` on every request), so a fixed value would misstate it. |
| `Content-Length` | Computed by the HTTP client from the request body; a configured value is ignored. |
<!-- upstream-headers-refused:end -->

Also refused:

- a name that is empty or is not a valid HTTP header name (an RFC 9110
  token);
- two names that differ only in case (for example `X-Key` and `x-key`);
- a value with a control character: bytes 0x00 to 0x08, 0x0A to 0x1F
  (including line breaks) and 0x7F. A horizontal tab (0x09) is accepted
  between other characters;
- a value that starts or ends with a space or a horizontal tab, because HTTP
  does not deliver leading or trailing whitespace. An empty value is
  accepted and is sent as an empty header.

**Compatibility.** Earlier versions forwarded the client's `Authorization`,
`X-Admin-Token`, `X-Tenant-ID`, cookies and trailers as sent, and never
applied `headers`. An upstream that relied on receiving the client's SatGate
token must now be given its own credential in `headers`. Because `headers`
had no effect before, this version refuses at load an upstream `headers`
entry that is owned by SatGate, malformed, contains a control character,
duplicates another name by case, or cannot be delivered as configured (the
names and values refused above); the error names the upstream and the
entry. Remove or correct the entry to load the configuration. No other
configuration that loaded before is refused.

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

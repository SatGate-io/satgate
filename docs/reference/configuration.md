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
| The whole `Authorization` field (every value) when any value carries a token this gateway issued | Any other route (`public`, `chargeback`) | A macaroon signed under this gateway's root key is a SatGate credential, not the upstream's. The rule and its boundary are below. |
| `X-Admin-Token`, `X-Tenant-ID`, every `X-SatGate-*` header | Always | SatGate reads these itself (admin authentication, tenant selection) or reserves the names; a client's value must not reach the upstream as if SatGate had set it. |
| SatGate's session cookies (`cloud_session`, `__Host-cloud_session`, `__Host-admin_session`, `impersonation`, `__Host-impersonation`, and any cookie whose name starts with `satgate`) | Always | They are SatGate sign-in sessions. Other cookies are passed on exactly as sent. |

**The rule on `public` and `chargeback` routes.** If a token this gateway
issued is found anywhere in an `Authorization` value, as written or inside
one wrapper, the whole `Authorization` field is not forwarded: every value
of it, including a foreign credential (an upstream API key, `Basic`
credentials, another issuer's token) sent in the same field, which is
dropped with it. If no value carries such a token, every value is forwarded
exactly as sent. Nothing is split, rewritten or partly removed. The header
block and the request trailers are each judged on their own `Authorization`
lines.

A token is found by character class, not by parsing: the gateway does not
read quotes, commas, parameters or schemes, so a broken quote, a malformed
parameter or percent sequence, invalid base64, empty elements or a very
long value elsewhere in the value cannot hide a token. It takes the runs of
base64 and base64url characters (padded or not) in:

- the value as written, and with its backslash escapes (as in a quoted
  string) resolved, wherever they are;
- the value with its percent-encoding decoded once (a `%` that does not
  start a valid `%XX` sequence is kept as written);
- each run of the value as written, decoded from base64 once, in either
  alphabet, padded or not (`Basic` credentials `user:<token>` or
  `<token>:`, or any other single base64 layer).

A run is the gateway's token when it decodes, the way the gateway's own
token reader decodes a token, to a macaroon signed under this gateway's
root key, in either format (JSON or binary); the macaroon of an
`L402`/`LSAT` value is the run before the `:`. Every spelling the gateway
accepts is recognised, including the line breaks its base64 decoding skips.
This covers every form the gateway accepts a token in (`Bearer <token>`,
the bare token, `Bearer Bearer <token>` at the MCP server,
`L402 <macaroon>:<preimage>` and `LSAT <macaroon>:<preimage>`), and the same
token under any other scheme or none, in any letter case, after any
separator, quoted or as a parameter value. Expired and attenuated tokens
the gateway issued count; a macaroon from another root key does not.

Out of scope, and forwarded like any other client data: a token the client
wraps in two or more encodings (base64 of base64, percent-encoding of
percent-encoding, base64 of percent-encoded text, percent-encoded `Basic`
credentials and the like), and a token in any header other than
`Authorization`, in the query string or in the body. The gateway never
accepts its token in those forms.

The work is linear in the length of the field: each run yields at most one
decoded result, and that result is searched for runs but never decoded as
a wrapper again.

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
`Via` or a custom `X-...` name (`User-Agent` and `Accept-Encoding` only with
a value that is not empty). `Host` is accepted and is sent as
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
| `Expect` | Managed by the HTTP client: for `100-continue` it waits for an interim response before sending the body, and a server answers any other expectation with 417, so a fixed value changes how the request is sent or whether it is served. |
<!-- upstream-headers-refused:end -->

Also refused:

- a name that is empty or is not a valid HTTP header name (an RFC 9110
  token);
- two names that differ only in case (for example `X-Key` and `x-key`);
- a value with a control character: bytes 0x00 to 0x08, 0x0A to 0x1F
  (including line breaks) and 0x7F. A horizontal tab (0x09) is accepted
  between other characters;
- a value that starts or ends with a space or a horizontal tab, because HTTP
  does not deliver leading or trailing whitespace;
- an empty value for `User-Agent` or `Accept-Encoding`, because the HTTP
  client leaves out an empty `User-Agent` and adds its own
  `Accept-Encoding` when that header is empty. For every other accepted
  name an empty value is accepted and is sent as one empty header.

**Compatibility.** Earlier versions forwarded the client's `Authorization`,
`X-Admin-Token`, `X-Tenant-ID`, cookies and trailers as sent, and never
applied `headers`. An upstream that relied on receiving the client's SatGate
token must now be given its own credential in `headers`. On `public` and
`chargeback` routes, a request that carries this gateway's token in
`Authorization` reaches the upstream with no `Authorization` at all, even
when a foreign credential was sent beside it; configure the upstream's
credential in `headers` rather than relying on the client to send both.
Because `headers`
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

# Route Configuration

Routes define how the SatGate Gateway handles incoming requests.

## Route Structure

```yaml
routes:
  - name: "route-name"          # Unique identifier
    match:                      # Request matching rules
      pathPrefix: "/api/"       # Path prefix match
      pathExact: "/health"      # Exact path match
      pathRegex: "/users/\\d+"  # Regex path match
      methods: ["GET", "POST"]  # Allowed HTTP methods
      headers:                  # Required headers
        X-API-Version: "2"
    upstream: "upstream-name"   # Target upstream
    policy:                     # Access policy
      kind: "l402"             # l402, capability, public, deny
      priceSats: 100           # Price for L402 requests
      scope: "api:read"        # Required scope for capability
    transform:                  # Request transformations
      stripPrefix: "/api"      # Remove prefix from path
      addHeaders:              # Add headers to upstream request
        X-Gateway: "SatGate"
    rateLimit:                  # Rate limiting
      requestsPerMinute: 100
      burstSize: 10
      key: "token"             # Rate limit key: ip, token, header:X-Custom
```

## Policy Types

### Public

No authentication required. Anyone can access.

```yaml
policy:
  kind: "public"
```

### Deny

Block all requests. Useful for maintenance or deprecated endpoints.

```yaml
policy:
  kind: "deny"
```

### Capability

Requires a valid capability token (macaroon) with matching scope.

```yaml
policy:
  kind: "capability"
  scope: "api:users:read"
```

Token scope matching:
- `api:*` → matches everything under `api:`
- `api:users:*` → matches `api:users:read`, `api:users:write`
- `api:users:read` → exact match only

### L402 (Paid Access)

Requires payment via Lightning Network.

```yaml
policy:
  kind: "l402"
  priceSats: 100
  scope: "premium"
```

## Path Matching

Routes are evaluated in order. First match wins.

### Prefix Match

Matches any path starting with the prefix.

```yaml
match:
  pathPrefix: "/api/v1/"
# Matches: /api/v1/users, /api/v1/orders/123
```

### Exact Match

Matches only the exact path.

```yaml
match:
  pathExact: "/health"
# Matches: /health (only)
```

### Regex Match

Matches paths using regular expressions.

```yaml
match:
  pathRegex: "^/users/[0-9]+$"
# Matches: /users/123, /users/456
```

## Method Filtering

Restrict which HTTP methods are allowed.

```yaml
match:
  pathPrefix: "/api/"
  methods: ["GET", "POST"]
# Only GET and POST requests allowed
```

## Header Matching

Require specific headers to be present.

```yaml
match:
  pathPrefix: "/api/"
  headers:
    X-API-Version: "2"
    Accept: "application/json"
```

## Transformations

### Strip Prefix

Remove the matched prefix before forwarding to upstream.

```yaml
match:
  pathPrefix: "/api/v1/"
upstream: "backend"
transform:
  stripPrefix: "/api/v1"
# /api/v1/users → /users (to backend)
```

### Add Headers

Add headers to the upstream request.

```yaml
transform:
  addHeaders:
    X-Gateway: "SatGate"
    X-Request-ID: "${request_id}"
```

## Rate Limiting

### Per-IP Rate Limiting

```yaml
rateLimit:
  requestsPerMinute: 60
  burstSize: 10
  key: "ip"
```

### Per-Token Rate Limiting

```yaml
rateLimit:
  requestsPerMinute: 1000
  burstSize: 100
  key: "token"
```

### Custom Header Rate Limiting

```yaml
rateLimit:
  requestsPerMinute: 100
  burstSize: 20
  key: "header:X-Customer-ID"
```

## Complete Examples

### E-commerce API

```yaml
routes:
  # Public product catalog
  - name: "products-public"
    match:
      pathPrefix: "/api/products"
      methods: ["GET"]
    upstream: "catalog"
    policy:
      kind: "public"
    rateLimit:
      requestsPerMinute: 100
      key: "ip"

  # Authenticated cart operations
  - name: "cart"
    match:
      pathPrefix: "/api/cart"
    upstream: "orders"
    policy:
      kind: "capability"
      scope: "api:cart"
    rateLimit:
      requestsPerMinute: 60
      key: "token"

  # Premium recommendations (paid)
  - name: "recommendations"
    match:
      pathPrefix: "/api/recommendations"
    upstream: "ml-service"
    policy:
      kind: "l402"
      priceSats: 50
```

### SaaS API

```yaml
routes:
  # Free tier
  - name: "basic-api"
    match:
      pathPrefix: "/api/v1/basic/"
    upstream: "api"
    policy:
      kind: "capability"
      scope: "tier:basic"
    rateLimit:
      requestsPerMinute: 100
      key: "token"

  # Pro tier (higher limits)
  - name: "pro-api"
    match:
      pathPrefix: "/api/v1/pro/"
    upstream: "api"
    policy:
      kind: "capability"
      scope: "tier:pro"
    rateLimit:
      requestsPerMinute: 1000
      key: "token"

  # Enterprise tier (custom)
  - name: "enterprise-api"
    match:
      pathPrefix: "/api/v1/enterprise/"
    upstream: "api"
    policy:
      kind: "capability"
      scope: "tier:enterprise"
```

## Best Practices

1. **Order routes from most specific to least specific**
2. **Use capability tokens for internal/trusted access**
3. **Use L402 for public monetization**
4. **Set appropriate rate limits for each tier**
5. **Add a catch-all deny route at the end**

```yaml
routes:
  # Specific routes first...
  
  # Catch-all deny at the end
  - name: "default-deny"
    match:
      pathPrefix: "/"
    policy:
      kind: "deny"
```




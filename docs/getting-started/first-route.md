# Your First Protected Route

Add SatGate protection to your existing API.

## Prerequisites

- SatGate Gateway running (see [Quick Start](quickstart.md))
- Your backend API accessible from the gateway

## Step 1: Add Your Upstream

Edit `gateway.yaml`:

```yaml
upstreams:
  my_backend:
    url: "http://your-api.internal:8080"
    timeout: 30s
    healthCheck:
      path: "/health"
      interval: 10s
```

## Step 2: Add a Protected Route

```yaml
routes:
  # Public health check (optional)
  - name: my-health
    path: /my-api/health
    upstream: my_backend
    policy:
      kind: public

  # Protected API endpoints
  - name: my-api
    path: /my-api/*
    upstream: my_backend
    policy:
      kind: observe
      scope: myapi:access
```

## Step 3: Restart Gateway

```bash
# Docker Compose
docker compose restart gateway

# Kubernetes
kubectl rollout restart deployment/satgate-gateway -n satgate
```

## Step 4: Mint a Token

```bash
curl -X POST http://localhost:9090/api/v1/tokens \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "myapi:access",
    "expiresIn": 86400,
    "metadata": {
      "client": "my-first-client"
    }
  }'
```

Save the token from the response.

## Step 5: Test Access

**Without token:**
```bash
curl http://localhost:8080/my-api/users
# → 401 Unauthorized
```

**With token:**
```bash
curl http://localhost:8080/my-api/users \
  -H "Authorization: Bearer YOUR_TOKEN"
# → Your API response
```

## Step 6: Distribute Token to Clients

### Option A: Direct Token

Give the token directly to trusted clients:

```bash
# Client uses the token
curl https://api.example.com/my-api/users \
  -H "Authorization: Bearer eyJ..."
```

### Option B: Delegated Token

Create a restricted token for less-trusted clients:

```bash
# Delegate with restrictions
curl -X POST http://localhost:9090/api/v1/tokens/SIGNATURE/delegate \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "caveats": [
      {"type": "expires", "value": "1h"},
      {"type": "rate_limit", "value": "100/minute"}
    ]
  }'
```

## Adding Rate Limiting

Protect against abuse:

```yaml
routes:
  - name: my-api
    path: /my-api/*
    upstream: my_backend
    policy:
      kind: observe
      scope: myapi:access
    rateLimit:
      requestsPerMinute: 100
      burstSize: 20
```

## Monitoring Your Route

### Check Stats

```bash
curl http://localhost:9090/api/v1/stats/routes \
  -H "X-Admin-Token: $ADMIN_TOKEN" | jq '.routes["my-api"]'
```

### View Metrics

```bash
curl http://localhost:8080/metrics | grep 'satgate_route_requests.*my-api'
```

## Common Patterns

### Read-Only vs Write Access

```yaml
routes:
  # Read-only access
  - name: my-api-read
    path: /my-api/*
    methods: [GET, HEAD]
    upstream: my_backend
    policy:
      kind: observe
      scope: myapi:read

  # Write access
  - name: my-api-write
    path: /my-api/*
    methods: [POST, PUT, DELETE]
    upstream: my_backend
    policy:
      kind: observe
      scope: myapi:write
```

### Internal vs External

```yaml
routes:
  # External (public internet)
  - name: public-api
    path: /api/v1/*
    upstream: my_backend
    policy:
      kind: observe
      scope: api:external

  # Internal (service-to-service)
  - name: internal-api
    path: /internal/*
    upstream: my_backend
    policy:
      kind: observe
      scope: api:internal
    # Only from trusted proxies
```

## Next Steps

- [Token Management](../guides/tokens.md) — Lifecycle and delegation
- [Rate Limiting](../guides/rate-limiting.md) — Protect against abuse
- [Production Deployment](kubernetes.md) — Go to production




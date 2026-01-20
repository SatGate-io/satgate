# Quick Start

Get SatGate Gateway running locally in 5 minutes.

## Prerequisites

- Docker and Docker Compose
- curl (for testing)

## Step 1: Clone and Start

```bash
git clone https://github.com/SatGate-io/satgate-gateway.git
cd satgate-gateway/deploy/docker-compose

docker compose up -d
```

## Step 2: Verify Health

```bash
curl http://localhost:8080/healthz
```

Expected response:
```json
{"status":"ok"}
```

## Step 3: Mint a Capability Token

```bash
curl -X POST http://localhost:9090/api/v1/tokens \
  -H "X-Admin-Token: quickstart-admin-token-change-me" \
  -H "Content-Type: application/json" \
  -d '{"scope": "api:read", "expiresIn": 3600}'
```

Response:
```json
{
  "token": "eyJ2IjoxLCJsIjoiaHR0cHM6Ly9zYXRnYXRlLmlvIi...",
  "signature": "abc123...",
  "expiresAt": "2024-01-01T12:00:00Z"
}
```

Save the token:
```bash
export TOKEN="eyJ2IjoxLCJsIjoiaHR0cHM6Ly9zYXRnYXRlLmlvIi..."
```

## Step 4: Test Protected Route

**Without token (rejected):**
```bash
curl http://localhost:8080/api/get
```

Response:
```json
{"error":"Authorization required"}
```

**With token (success):**
```bash
curl http://localhost:8080/api/get \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "args": {},
  "headers": {
    "Authorization": "Bearer eyJ2IjoxLCJsIjoiaHR0cHM6Ly9zYXRnYXRlLmlvIi...",
    "Host": "httpbin.org",
    ...
  },
  "url": "https://httpbin.org/get"
}
```

## Step 5: Test Public Route

Public routes don't require tokens:

```bash
curl http://localhost:8080/demo/posts/1
```

## What Just Happened?

1. **Gateway started** with PostgreSQL and Redis
2. **Token minted** with `api:read` scope
3. **Protected route** validated the token before proxying
4. **Public route** proxied without authentication

## Next Steps

- [Core Concepts](concepts.md) — Understand capability tokens
- [Add Your API](first-route.md) — Protect your own endpoints
- [Kubernetes Deployment](../guides/kubernetes.md) — Production setup

## Clean Up

```bash
docker compose down -v
```

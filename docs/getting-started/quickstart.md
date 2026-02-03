# Quick Start

Get SatGate running and see the Economic Firewall in action. Three policies, three curl commands, sixty seconds.

## Prerequisites

- **macOS/Linux** (or Windows WSL)
- **curl** (for testing)

## Step 1: Download and Run

**Option A: Pre-built Binary (fastest)**

```bash
# macOS (Apple Silicon)
curl -L https://github.com/satgate-io/satgate/releases/latest/download/satgate-darwin-arm64 -o satgate

# macOS (Intel)
curl -L https://github.com/satgate-io/satgate/releases/latest/download/satgate-darwin-amd64 -o satgate

# Linux (x86_64)
curl -L https://github.com/satgate-io/satgate/releases/latest/download/satgate-linux-amd64 -o satgate

chmod +x satgate
```

**Option B: Build from Source**

```bash
git clone https://github.com/satgate-io/satgate.git
cd satgate
go build -o satgate ./cmd/satgate
```

**Option C: Docker**

```bash
git clone https://github.com/satgate-io/satgate.git
cd satgate
docker compose up -d
# Gateway is now running on http://localhost:8080
```

## Step 2: Start the Gateway

```bash
# Set an admin token (or one will be auto-generated)
export ADMIN_TOKEN=my-secret-admin-token

# Use the mock Lightning provider for this demo
export LIGHTNING_BACKEND=mock

# Start with the example config
./satgate --config examples/gateway.yaml
```

You should see:

```
INF Starting SatGate OSS Gateway version=dev
WRN CAPABILITY_ROOT_KEY not set - using auto-generated key (demo mode)
INF Macaroon service initialized
INF Governance service initialized (in-memory)
INF Lightning provider initialized provider=mock
INF Routes configured routes=8
INF Gateway listening listen=:8080
INF Supported policies: public, capability, l402
```

> **Note:** Demo mode auto-generates a root key. Tokens won't persist across restarts. Set `CAPABILITY_ROOT_KEY` for production (see [Configuration](../configuration/)).

## Step 3: See the Three Policies

Open a **new terminal** and try these:

### Policy 1: Public (open access)

```bash
curl http://localhost:8080/health
```

```json
{"status":"healthy","service":"satgate-oss","routes":8}
```

No token needed. Public routes are explicitly opted out of protection.

---

### Policy 2: Capability (protected)

**Without a token — blocked:**

```bash
curl http://localhost:8080/protected/get
```

```
Authorization required
```

**Mint a capability token:**

```bash
curl -s -X POST http://localhost:8080/api/capability/mint \
  -H "X-Admin-Token: my-secret-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"scope": "api:read", "duration": "1h"}'
```

```json
{
  "token": "eyJ2IjoxLCJs...",
  "scope": "api:read",
  "expiresAt": "2026-02-03T13:00:00Z",
  "signature": "e05e049c..."
}
```

Save the token:

```bash
export TOKEN="eyJ2IjoxLCJs..."
```

**With a token — access granted:**

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/capability/ping
```

```json
{
  "status": "ok",
  "message": "Token validated successfully",
  "caveats": ["expires = ...", "scope = api:read"],
  "validated": "2026-02-03T12:00:00Z"
}
```

Your request was cryptographically verified using a [Macaroon](../ARCHITECTURE.md) bearer token with embedded caveats.

---

### Policy 3: L402 (paid)

```bash
curl http://localhost:8080/api/micro
```

```json
{
  "error": "payment_required",
  "invoice": "lnbc1u1p...",
  "amount_sats": 1,
  "payment_hash": "5348..."
}
```

HTTP 402 — Payment Required. The gateway issued a Lightning invoice. Pay it, submit the preimage, and access is granted. That's the [L402 protocol](../ARCHITECTURE.md#l402-protocol).

---

## Step 4: Delegate a Token

One of the most powerful features — create a child token with *fewer* permissions, without contacting the server:

```bash
curl -s -X POST http://localhost:8080/api/capability/delegate \
  -H "Content-Type: application/json" \
  -d "{\"parentToken\": \"$TOKEN\", \"caveats\": [\"scope = api:read:limited\"]}"
```

```json
{
  "token": "eyJ2IjoxLCJs...",
  "caveats": ["expires = ...", "scope = api:read", "scope = api:read:limited"],
  "signature": "ec91c4d8..."
}
```

The child token inherits the parent's caveats *plus* the new restriction. Caveats can only be added, never removed. This is how agent swarms work — a master agent delegates restricted tokens to workers.

## Step 5: Revoke a Token

Instant kill switch — ban a token by its signature:

```bash
curl -s -X POST http://localhost:8080/api/governance/ban \
  -H "X-Admin-Token: my-secret-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"tokenSignature": "e05e049c...", "reason": "Compromised"}'
```

```json
{
  "status": "banned",
  "tokenSignature": "e05e049c...",
  "reason": "Compromised",
  "bannedAt": "2026-02-03T12:05:00Z"
}
```

The token is now dead. Any request using it — or any child token delegated from it — will be rejected.

## Step 6: View the Token Graph

```bash
curl http://localhost:8080/api/governance/graph
```

Returns a full lineage graph of all minted and delegated tokens, their status (active/banned), usage counts, and relationships.

---

## What Just Happened?

In under a minute, you:

1. **Started** an API gateway with zero dependencies
2. **Protected** routes with cryptographic capability tokens
3. **Monetized** routes with Lightning micropayments (L402)
4. **Delegated** tokens with restricted permissions
5. **Revoked** a compromised token instantly
6. **Inspected** the full token lineage graph

No database. No Redis. No Docker. Just a single binary.

---

## What's Next?

| Guide | Description |
|-------|-------------|
| [Core Concepts](concepts.md) | Understand Macaroons, caveats, and delegation |
| [Add Your API](first-route.md) | Protect your own backend endpoints |
| [Configuration Reference](../configuration/) | Full YAML reference |
| [Architecture](../ARCHITECTURE.md) | How SatGate works under the hood |
| [Self-Hosted Deployment](../operations/) | Production deployment guide |

## Configuration Used

The example config (`examples/gateway.yaml`) sets up:

| Route | Path | Policy | Strip Prefix | Description |
|-------|------|--------|:---:|-------------|
| health | `/health` | Built-in | — | Always available |
| public-demo | `/public/*` | `public` | ✓ | No auth required |
| protected-demo | `/protected/*` | `capability` | ✓ | Requires valid Macaroon |
| premium-demo | `/premium/*` | `l402` (100 sats) | ✓ | Lightning payment |
| api-micro | `/api/micro` | `l402` (1 sat) | — | Lightning micropayment |
| api-basic | `/api/basic` | `l402` (10 sats) | — | Lightning payment |
| api-standard | `/api/standard` | `l402` (100 sats) | — | Lightning payment |
| api-premium | `/api/premium` | `l402` (1000 sats) | — | Lightning payment |

> **Tip:** `stripPrefix: true` removes the matched path prefix before proxying. So `/public/get` → upstream receives `/get`. Use `rewrite` for static path replacement instead.

---

## Clean Up

Just kill the process (`Ctrl+C`). No state to clean up — everything is in-memory in demo mode.

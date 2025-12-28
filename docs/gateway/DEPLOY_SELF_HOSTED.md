<p align="center">
  <img src="../assets/brand/logo_blue_transparent.png" alt="SatGate™" width="140" />
</p>

# Deploy SatGate™ Gateway (Self-Hosted)

**Time to deploy:** ~15 minutes  
**Prerequisites:** Your API is publicly reachable via HTTPS

This guide covers deploying SatGate Gateway to **Railway**, **Fly.io**, or **Docker**. Choose your platform and follow the steps.

---

## Quick Start (All Platforms)

### 1. Get Your Config Ready

```bash
# Clone the repo
git clone https://github.com/SatGate-io/satgate.git
cd satgate

# Copy the SMB starter config
cp satgate.gateway.smb.yaml satgate.gateway.yaml

# Edit with your upstream URL
# Replace "https://api.example.com" with your actual API
```

### 2. Validate Your Config (Recommended)

```bash
# Validate before deploying
node cli/validate-gateway-config.js satgate.gateway.yaml
```

Expected output:
```
✓ Config valid
  Upstreams: my_api
  Routes: premium, basic, micro, default-deny
  L402 mode: native
```

### 3. Choose Your Platform

- [Railway](#railway-recommended-for-smbs) — Fastest, zero-config deploys
- [Fly.io](#flyio) — Edge deployment, good free tier
- [Docker](#docker) — Full control, any infrastructure

---

## Railway (Recommended for SMBs)

### Step 1: Create Railway Project

1. Go to [railway.app](https://railway.app)
2. Click **New Project** → **Deploy from GitHub repo**
3. Select your forked `satgate` repo (or use the template)

### Step 2: Set Environment Variables

In Railway dashboard → **Variables**, add:

```bash
# Required: Runtime mode
SATGATE_RUNTIME=gateway
MODE=prod

# Required: Security keys (generate with: openssl rand -hex 16)
L402_ROOT_KEY=<your-32-char-random-key>
PRICING_ADMIN_TOKEN=<your-32-char-random-key>

# Lightning Backend (choose ONE)
# ─────────────────────────────────────────────────────────────────
# Option A: phoenixd (recommended for SMB - simplest setup)
LIGHTNING_BACKEND=phoenixd
PHOENIXD_URL=http://your-phoenixd-host:9740
PHOENIXD_PASSWORD=<your-phoenixd-password>

# Option B: LND (more control, requires node management)
# LIGHTNING_BACKEND=lnd
# LND_REST_URL=https://your-lnd-node:8080
# LND_MACAROON=<base64-encoded-invoice-macaroon>
# ─────────────────────────────────────────────────────────────────

# Optional: Redis for multi-instance metering
# REDIS_URL=redis://user:pass@host:6379
```

### Step 3: Configure Build Settings

In Railway dashboard → **Settings**:

| Setting | Value |
|---------|-------|
| **Dockerfile Path** | `deploy/Dockerfile.native` |
| **Health Check Path** | `/healthz` |
| **Health Check Timeout** | 30 |

### Step 4: Deploy

Railway auto-deploys on push. Or click **Deploy** manually.

### Step 5: Verify

```bash
# Get your Railway URL from the dashboard
GATEWAY_URL="https://your-project.up.railway.app"

# Health check
curl $GATEWAY_URL/healthz
# Expected: {"status":"ok","plane":"data",...}

# L402 challenge (should return 402)
curl -i $GATEWAY_URL/v1/basic/test
# Expected: HTTP 402 + WWW-Authenticate header
```

---

## Fly.io

### Step 1: Install Fly CLI

```bash
# macOS
brew install flyctl

# Or see: https://fly.io/docs/hands-on/install-flyctl/
```

### Step 2: Create fly.toml

Create `fly.toml` in your project root:

```toml
app = "satgate-gateway"
primary_region = "iad"

[build]
  dockerfile = "deploy/Dockerfile.native"

[env]
  SATGATE_RUNTIME = "gateway"
  MODE = "prod"
  # Don't put secrets here - use fly secrets set

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 1

[[http_service.checks]]
  path = "/healthz"
  interval = "10s"
  timeout = "5s"
```

### Step 3: Set Secrets

```bash
# Required secrets
fly secrets set L402_ROOT_KEY="$(openssl rand -hex 16)"
fly secrets set PRICING_ADMIN_TOKEN="$(openssl rand -hex 16)"

# Lightning backend (phoenixd)
fly secrets set LIGHTNING_BACKEND=phoenixd
fly secrets set PHOENIXD_URL=http://your-phoenixd-host:9740
fly secrets set PHOENIXD_PASSWORD=<your-password>

# Or Lightning backend (LND)
# fly secrets set LIGHTNING_BACKEND=lnd
# fly secrets set LND_REST_URL=https://your-lnd-node:8080
# fly secrets set LND_MACAROON=<base64-macaroon>
```

### Step 4: Deploy

```bash
fly launch --no-deploy  # First time only
fly deploy
```

### Step 5: Verify

```bash
GATEWAY_URL="https://satgate-gateway.fly.dev"

curl $GATEWAY_URL/healthz
curl -i $GATEWAY_URL/v1/basic/test
```

---

## Docker

### Step 1: Build the Image

```bash
docker build -f deploy/Dockerfile.native -t satgate-gateway .
```

### Step 2: Create Environment File

Create `.env.gateway`:

```bash
# Runtime
SATGATE_RUNTIME=gateway
MODE=prod

# Security keys
L402_ROOT_KEY=your-32-char-random-key-here
PRICING_ADMIN_TOKEN=your-32-char-admin-token

# Lightning backend (phoenixd - recommended)
LIGHTNING_BACKEND=phoenixd
PHOENIXD_URL=http://phoenixd:9740
PHOENIXD_PASSWORD=your-phoenixd-password

# Or LND
# LIGHTNING_BACKEND=lnd
# LND_REST_URL=https://lnd:8080
# LND_MACAROON=base64-macaroon-here

# Optional: Redis
# REDIS_URL=redis://redis:6379
```

### Step 3: Run the Container

```bash
docker run -d \
  --name satgate-gateway \
  --env-file .env.gateway \
  -p 8080:8080 \
  satgate-gateway
```

### Step 4: Verify

```bash
curl http://localhost:8080/healthz
curl -i http://localhost:8080/v1/basic/test
```

### Docker Compose (with phoenixd)

```yaml
# docker-compose.gateway.yaml
version: '3.8'

services:
  gateway:
    build:
      context: .
      dockerfile: deploy/Dockerfile.native
    ports:
      - "8080:8080"
    environment:
      SATGATE_RUNTIME: gateway
      MODE: prod
      L402_ROOT_KEY: ${L402_ROOT_KEY}
      PRICING_ADMIN_TOKEN: ${PRICING_ADMIN_TOKEN}
      LIGHTNING_BACKEND: phoenixd
      PHOENIXD_URL: http://phoenixd:9740
      PHOENIXD_PASSWORD: ${PHOENIXD_PASSWORD}
    depends_on:
      - phoenixd
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 10s
      timeout: 5s
      retries: 3

  phoenixd:
    image: phoenixd/phoenixd:latest
    ports:
      - "9740:9740"
    volumes:
      - phoenixd_data:/root/.phoenix
    # See phoenixd docs for full configuration

volumes:
  phoenixd_data:
```

Run:
```bash
export L402_ROOT_KEY=$(openssl rand -hex 16)
export PRICING_ADMIN_TOKEN=$(openssl rand -hex 16)
export PHOENIXD_PASSWORD=your-password

docker-compose -f docker-compose.gateway.yaml up -d
```

---

## Environment Variables Reference

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `SATGATE_RUNTIME` | Must be `gateway` | `gateway` |
| `MODE` | Environment posture | `prod` |
| `L402_ROOT_KEY` | L402 macaroon signing key (32+ chars) | `a7b3c9d4e5f6...` |
| `PRICING_ADMIN_TOKEN` | Admin API token (32+ chars) | `1f2e3d4c5b6a...` |

### Lightning Backend (choose one)

#### phoenixd (Recommended for SMB)

| Variable | Description | Example |
|----------|-------------|---------|
| `LIGHTNING_BACKEND` | Set to `phoenixd` | `phoenixd` |
| `PHOENIXD_URL` | phoenixd HTTP endpoint | `http://localhost:9740` |
| `PHOENIXD_PASSWORD` | phoenixd password | `your-password` |

**Why phoenixd?**
- No channel management
- No liquidity concerns
- Simple HTTP API
- Perfect for SMB/startup use case

#### LND

| Variable | Description | Example |
|----------|-------------|---------|
| `LIGHTNING_BACKEND` | Set to `lnd` | `lnd` |
| `LND_REST_URL` | LND REST endpoint | `https://lnd.example.com:8080` |
| `LND_MACAROON` | Base64-encoded invoice macaroon | `AgEDbG5kAv...` |
| `LND_INSECURE_TLS` | Allow self-signed certs (localhost only) | `false` |

### Optional

| Variable | Description | Default |
|----------|-------------|---------|
| `SATGATE_GATEWAY_CONFIG` | Config file path | `./satgate.gateway.yaml` |
| `REDIS_URL` | Redis for distributed metering | In-memory |
| `CAPABILITY_ROOT_KEY` | For capability token routes | — |
| `CORS_ORIGINS` | Allowed CORS origins | — |

---

## Health Checks

### Data Plane Health

```bash
curl https://<your-gateway>/healthz
```

Expected response:
```json
{
  "status": "ok",
  "plane": "data",
  "timestamp": "2025-01-15T10:30:00.000Z"
}
```

### Lightning Backend Health

Check your gateway logs for:
```
[Gateway] L402 service initialized (Lightning: phoenixd)
```

Or test by triggering a 402:
```bash
curl -i https://<your-gateway>/v1/basic/test
# Look for: WWW-Authenticate: L402 macaroon="...", invoice="lnbc..."
```

If you see an invoice starting with `lnbc...`, Lightning is connected.

---

## Common Issues

### "Gateway config file not found"

**Cause:** `satgate.gateway.yaml` not in container or wrong path.

**Fix:**
1. Ensure `COPY satgate.gateway.yaml ./` is in your Dockerfile
2. Or set `SATGATE_GATEWAY_CONFIG` to the correct path

### "L402 invoice creation failed"

**Cause:** Lightning backend unreachable or misconfigured.

**Fixes:**
- **phoenixd:** Check `PHOENIXD_URL` and `PHOENIXD_PASSWORD`
- **LND:** Check `LND_REST_URL` and `LND_MACAROON` (must be base64, not hex)
- **Network:** Ensure gateway can reach Lightning backend

### "502 Bad Gateway"

**Cause:** Upstream unreachable from gateway.

**Fixes:**
1. Verify upstream URL is publicly accessible
2. Check for HTTPS/TLS issues
3. Verify upstream is not blocking gateway IP

### Health check fails on Railway

**Cause:** Health check path mismatch.

**Fix:** Set health check path to `/healthz` (not `/health`) in Railway settings.

---

## Next Steps

1. **Verify L402 flow:** See [First Success Demo](./FIRST_SUCCESS_DEMO.md)
2. **Add upstream auth:** See [SMB Onboarding](./SMB_ONBOARDING.md#upstream-authentication)
3. **Monitor:** Check logs for request counts and errors

---

## Generate Secure Keys

```bash
# Generate all required keys at once
echo "L402_ROOT_KEY=$(openssl rand -hex 16)"
echo "PRICING_ADMIN_TOKEN=$(openssl rand -hex 16)"
echo "CAPABILITY_ROOT_KEY=$(openssl rand -hex 16)"
```

Copy the output into your environment variables.


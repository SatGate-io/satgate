# SatGate Cloud - Fly.io Deployment Guide

## Overview

SatGate Cloud consists of 3 apps:

| App | Purpose | Domain |
|-----|---------|--------|
| **control-plane** | Auth, Projects, Config API | `satgate-cloud-control.fly.dev` (or custom) |
| **data-plane** | Multi-tenant L402 Gateway | `*.satgate.cloud` (wildcard) |
| **dashboard** | Next.js Frontend | `cloud.satgate.io` (or custom) |

## Prerequisites

1. [Fly CLI](https://fly.io/docs/hands-on/install-flyctl/) installed
2. Postgres database (Fly Postgres or Supabase)
3. SMTP provider for magic link emails (e.g., Resend, SendGrid)

## Step 1: Provision Postgres

```bash
# Create Fly Postgres (or use Supabase)
fly postgres create --name satgate-cloud-db --region ord

# Get connection string
fly postgres connect -a satgate-cloud-db
```

Apply the schema:
```bash
psql $DATABASE_URL < satgate-cloud/db/schema.sql
```

## Step 2: Generate Secrets

```bash
# Generate encryption key (32+ chars)
openssl rand -base64 32

# Generate internal token
openssl rand -hex 32

# Generate L402 root key
openssl rand -hex 32
```

## Step 3: Deploy Control Plane

```bash
cd satgate-cloud/apps/control-plane

# Create app
fly apps create satgate-cloud-control

# Set secrets
fly secrets set \
  DATABASE_URL="postgres://..." \
  SECRETS_ENCRYPTION_KEY="..." \
  DATA_PLANE_URL="https://satgate-cloud-data.fly.dev" \
  DATA_PLANE_INTERNAL_TOKEN="..." \
  APP_URL="https://cloud.satgate.io" \
  SMTP_HOST="smtp.resend.com" \
  SMTP_PORT="465" \
  SMTP_USER="resend" \
  SMTP_PASS="re_..." \
  EMAIL_FROM="noreply@satgate.io"

# Deploy
fly deploy
```

## Step 4: Deploy Data Plane

```bash
cd satgate-cloud/apps/data-plane

# Create app
fly apps create satgate-cloud-data

# Set secrets
fly secrets set \
  DATABASE_URL="postgres://..." \
  SECRETS_ENCRYPTION_KEY="..." \
  INTERNAL_AUTH_TOKEN="..." \
  L402_ROOT_KEY="..."

# Deploy
fly deploy
```

## Step 5: Configure Wildcard Domain

```bash
# Add wildcard certificate
fly certs create "*.satgate.cloud" -a satgate-cloud-data

# Follow DNS instructions to verify
```

DNS Records:
```
*.satgate.cloud  CNAME  satgate-cloud-data.fly.dev
```

## Step 6: Deploy Dashboard

```bash
cd satgate-cloud/apps/dashboard

# Create app
fly apps create satgate-cloud-dashboard

# Deploy with API URL
fly deploy --build-arg NEXT_PUBLIC_API_URL=https://satgate-cloud-control.fly.dev

# Add custom domain
fly certs create cloud.satgate.io
```

## Step 6b: Configure Same-Origin Cookies (Recommended)

For session cookies to work reliably, the dashboard and control plane should share a domain:

**Option A: Same-origin (recommended)**
```
Dashboard:     cloud.satgate.io
Control Plane: cloud.satgate.io/api/* (via Fly internal routing or Caddy)
```
No additional cookie config needed.

**Option B: Same-site subdomain**
```
Dashboard:     cloud.satgate.io  
Control Plane: api.satgate.io
```
Set on control plane:
```bash
fly secrets set COOKIE_DOMAIN=".satgate.io" -a satgate-cloud-control
```

**Option C: Cross-site (not recommended)**
```
Dashboard:     cloud.satgate.io
Control Plane: satgate-cloud-control.fly.dev
```
Set on control plane:
```bash
fly secrets set COOKIE_CROSS_SITE=true -a satgate-cloud-control
```
Note: Cross-site cookies are increasingly blocked by browsers.

## Step 7: Smoke Test

```bash
# 1. Login flow
open https://cloud.satgate.io
# Request magic link → check email → click → should land on /dashboard

# 2. Create project
# In dashboard, create "test-api" project
# Note the host: test-api-abc123.satgate.cloud

# 3. Upload config
# Paste starter config, click Save & Deploy

# 4. Test data plane
curl -i https://test-api-abc123.satgate.cloud/api/test
# Should return 403 (default-deny) or 402 (if route is priced)
```

## Environment Variables Reference

### Control Plane
| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | ✓ | Postgres connection string |
| `SECRETS_ENCRYPTION_KEY` | ✓ | Base64-encoded 32 bytes, or 32+ char string |
| `DATA_PLANE_URL` | ✓ | Data plane base URL |
| `DATA_PLANE_INTERNAL_TOKEN` | ✓ | Token for cache invalidation |
| `APP_URL` | ✓ | Dashboard URL (for magic links) |
| `SMTP_HOST` | ✓ | SMTP server host |
| `SMTP_PORT` | ✓ | SMTP server port |
| `SMTP_USER` | ✓ | SMTP username |
| `SMTP_PASS` | ✓ | SMTP password |
| `EMAIL_FROM` | | From address (default: noreply@satgate.io) |
| `CORS_ORIGINS` | | Allowed origins (default: APP_URL) |
| `COOKIE_DOMAIN` | | Shared cookie domain (e.g., `.satgate.io`) |
| `COOKIE_CROSS_SITE` | | Set to `true` for cross-site cookies (SameSite=None) |

### Data Plane
| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | ✓ | Postgres connection string |
| `SECRETS_ENCRYPTION_KEY` | ✓ | Same as control plane |
| `INTERNAL_AUTH_TOKEN` | ✓ | Token for /_internal endpoints |
| `L402_ROOT_KEY` | ✓ | Root key for macaroon signing |
| `LIGHTNING_ENABLED` | | Set to "true" for real Lightning |
| `LIGHTNING_BACKEND` | | `phoenixd` or `mock` (default: mock) |
| `PHOENIXD_URL` | | phoenixd REST URL (e.g., http://localhost:9740) |
| `PHOENIXD_PASSWORD` | | phoenixd HTTP auth password |

### Dashboard
| Variable | Required | Description |
|----------|----------|-------------|
| `NEXT_PUBLIC_API_URL` | ✓ | Control plane API URL |

## Lightning Configuration

SatGate Cloud supports multiple Lightning providers:

### Mock (default)
For testing without real payments:
```bash
# No additional config needed (default)
LIGHTNING_ENABLED=false
```

### phoenixd (recommended for production)
[phoenixd](https://phoenix.acinq.co/server) is a lightweight Lightning node by ACINQ:

```bash
# Install phoenixd
curl -sSL https://phoenix.acinq.co/server/install.sh | sh

# Start phoenixd
phoenixd

# Configure data plane
fly secrets set \
  LIGHTNING_ENABLED=true \
  LIGHTNING_BACKEND=phoenixd \
  PHOENIXD_URL=http://your-phoenixd-host:9740 \
  PHOENIXD_PASSWORD=your-password \
  -a satgate-cloud-data
```

phoenixd creates invoices at `/createinvoice` and reports payment status at `/getincomingpayment`.

### Testing Payments

With mock provider, you can test the L402 flow:
1. Request → 402 Payment Required (with fake invoice)
2. Client "pays" by computing preimage for the payment hash
3. Client retries with `Authorization: L402 <macaroon>:<preimage>`
4. Request succeeds

With real Lightning:
1. Request → 402 Payment Required (with real BOLT11 invoice)
2. Client pays via Lightning wallet
3. Client retries with the preimage from payment
4. Request succeeds

## Scaling

```bash
# Scale data plane for more traffic
fly scale count 3 -a satgate-cloud-data

# Scale control plane
fly scale count 2 -a satgate-cloud-control
```

## Monitoring

```bash
# View logs
fly logs -a satgate-cloud-data
fly logs -a satgate-cloud-control

# Check health
curl https://satgate-cloud-control.fly.dev/healthz
curl https://satgate-cloud-data.fly.dev/healthz
```


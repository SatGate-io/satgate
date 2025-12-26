# SatGate Cloud (SMB/Startup) — Bring Your Own Upstream

**The fastest way to monetize and protect your API.**

This guide describes the **SMB/Startup-first** onboarding path: connect your existing API as an **upstream**, attach an L402 policy, and SatGate becomes your paid/protected gateway in minutes.

---

## What You're Setting Up

```
┌─────────────┐     ┌─────────────────┐     ┌──────────────┐
│   Client    │────▶│  SatGate Cloud  │────▶│  Your API    │
│  (pays)     │     │  (L402 + proxy) │     │  (upstream)  │
└─────────────┘     └─────────────────┘     └──────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  Lightning  │
                    │  (managed)  │
                    └─────────────┘
```

**What SatGate handles:**
- L402 invoice issuance + validation
- Per-request metering (`maxCalls`, `budgetSats`)
- Lightning payment processing (managed node)
- Request proxying to your upstream
- Analytics + revenue dashboard

**What you provide:**
- A publicly reachable API endpoint (your upstream)
- Route definitions + pricing (YAML config)

---

## Prerequisites

- [ ] Your API is reachable via HTTPS (e.g., `https://api.yourcompany.com`)
- [ ] You know which endpoints to monetize
- [ ] You have pricing in mind (sats per request per tier)

> 💡 **No Lightning node required.** SatGate Cloud manages the payment infrastructure.

---

## Step 1: Create Your Config

Start with the SMB starter template:

```bash
# Download the SMB-optimized config
curl -O https://raw.githubusercontent.com/SatGate-io/satgate/main/satgate.gateway.smb.yaml

# Or copy from the repo
cp satgate.gateway.smb.yaml my-gateway.yaml
```

Edit the config with your upstream and routes:

```yaml
# my-gateway.yaml
version: 1

upstreams:
  my_api:
    url: "https://api.yourcompany.com"  # ← Your API

routes:
  # Premium endpoints - 100 sats
  - name: "premium"
    match:
      pathPrefix: "/v1/premium/"
    upstream: "my_api"
    policy:
      kind: "l402"
      tier: "premium"
      priceSats: 100
      scope: "api:premium:*"

  # Basic endpoints - 10 sats
  - name: "basic"
    match:
      pathPrefix: "/v1/"
    upstream: "my_api"
    policy:
      kind: "l402"
      tier: "basic"
      priceSats: 10
      scope: "api:basic:*"

  # Everything else - blocked (fail-closed)
  - name: "default-deny"
    match:
      pathPrefix: "/"
    policy:
      kind: "deny"
      status: 403
```

---

## Step 2: Test Locally (Optional)

Before deploying to SatGate Cloud, you can test locally:

```bash
# Set environment
export SATGATE_RUNTIME=gateway
export MODE=demo
export LIGHTNING_BACKEND=mock  # No real payments in test
export L402_ROOT_KEY=test-key-for-local-dev-only

# Run gateway
node proxy/server.js
```

Test the flow:

```bash
# Health check
curl http://localhost:8080/healthz

# L402 challenge (should return 402)
curl -i http://localhost:8080/v1/premium/test

# Fail-closed (should return 403)
curl -i http://localhost:8080/unknown
```

---

## Step 3: Deploy to SatGate Cloud

### Option A: Hosted Gateway (Recommended for SMB)

1. **Sign up** at `https://cloud.satgate.io` (coming soon)
2. **Upload your config** (or paste YAML in dashboard)
3. **Get your gateway URL**: `https://your-project.satgate.cloud`
4. **Update your DNS** (optional): CNAME `api.yourcompany.com` → `your-project.satgate.cloud`

### Option B: Self-Hosted (Railway/Fly/Docker)

Deploy to Railway with one click:

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/satgate)

Or manually:

```bash
# Set production environment
SATGATE_RUNTIME=gateway
MODE=prod
LIGHTNING_BACKEND=lnd
LND_REST_URL=https://your-lnd-node:8080
LND_MACAROON=<base64-macaroon>
L402_ROOT_KEY=<32-char-random-key>
PRICING_ADMIN_TOKEN=<32-char-random-key>
```

---

## Step 4: Verify Your Gateway

### Test 1: Health Check

```bash
curl https://your-gateway.satgate.cloud/healthz
# Expected: {"status":"ok","plane":"data",...}
```

### Test 2: L402 Challenge

```bash
curl -i https://your-gateway.satgate.cloud/v1/premium/test
# Expected: HTTP 402 + WWW-Authenticate: L402 macaroon="...", invoice="lnbc..."
```

### Test 3: Fail-Closed

```bash
curl -i https://your-gateway.satgate.cloud/unknown
# Expected: HTTP 403 + {"error":"Forbidden","route":"default-deny"}
```

### Test 4: Full Payment Flow

```bash
# Get the invoice from the 402 response
INVOICE="lnbc..."

# Pay with any Lightning wallet, get preimage
PREIMAGE="abc123..."

# Retry with LSAT token
MACAROON="eyJ..."
curl -H "Authorization: LSAT ${MACAROON}:${PREIMAGE}" \
  https://your-gateway.satgate.cloud/v1/premium/test
# Expected: HTTP 200 + response from your upstream API
```

---

## Step 5: Go Live

### DNS Setup (Optional but Recommended)

Point your API subdomain to the SatGate gateway:

```
api.yourcompany.com  CNAME  your-project.satgate.cloud
```

### Announce to Your Users

Your API now requires Lightning payment. Provide users with:

1. **Endpoint URL**: `https://api.yourcompany.com/v1/...`
2. **Pricing**: micro (1 sat), basic (10 sats), premium (100 sats)
3. **How to pay**: Any Lightning wallet, or use WebLN-enabled browser

---

## Pricing Your API

### Suggested Tiers

| Tier | Price | Use Case |
|------|-------|----------|
| **Micro** | 1 sat (~$0.001) | High-volume, low-value calls |
| **Basic** | 10 sats (~$0.01) | Standard API access |
| **Premium** | 100 sats (~$0.10) | Expensive operations |
| **Enterprise** | 1000+ sats | Heavy compute, bulk data |

### Pricing Strategy

- **Start low**: 1-10 sats removes bots without friction for real users
- **Price by value**: expensive operations (AI inference, data export) can be higher
- **Economic firewall first**: even 1 sat stops most abuse

---

## SMB Guardrails

### What's Included (v1)

- ✅ Hosted gateway endpoint
- ✅ Managed Lightning (no node setup)
- ✅ Metering + basic analytics
- ✅ Route/pricing configuration
- ✅ API keys + rotation

### What's Not Included (v1)

- ❌ VPC peering / private upstreams (require public URL)
- ❌ Enterprise SSO / RBAC
- ❌ Custom domains (use CNAME)
- ❌ Multi-region (single region in v1)

> 💡 Need enterprise features? Contact sales@satgate.io for "SatGate Operate" (managed deployment in your infra).

---

## Troubleshooting

### "502 Bad Gateway"

Your upstream is unreachable from SatGate Cloud.

**Fix:** Ensure your API is publicly accessible via HTTPS.

### "L402 invoice creation failed"

Lightning backend issue.

**Fix (self-hosted):** Check `LND_REST_URL` and `LND_MACAROON` environment variables.

### "403 on valid routes"

Route matching isn't working as expected.

**Fix:** Routes are evaluated in order—ensure more specific routes come before general ones.

---

## Next Steps

1. **Monitor revenue**: Check the SatGate dashboard for payment analytics
2. **Adjust pricing**: Use data to optimize (lower friction vs. higher revenue)
3. **Add routes**: Expand L402 protection to more endpoints
4. **Upgrade**: Move to enterprise tier for advanced features

---

## Support

- **Docs**: https://docs.satgate.io
- **Discord**: https://discord.gg/satgate
- **Email**: support@satgate.io

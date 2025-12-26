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
- Lightning payment processing (SatGate Cloud: managed node; self-hosted: your node)
- Request proxying to your upstream
- Analytics + revenue dashboard (SatGate Cloud)

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
# Copy from this repo
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

### Upstream Authentication (If Your API Requires It)

Most APIs require an API key or Bearer token. Add it to `addHeaders`:

```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    addHeaders:
      # Option 1: API Key header
      X-API-Key: "${API_KEY}"
      
      # Option 2: Bearer token
      # Authorization: "Bearer ${UPSTREAM_TOKEN}"
```

> 💡 **Tip:** Use environment variable references (`${VAR}`) for secrets instead of hardcoding them in the config file.

---

## Step 2: Test Locally (Optional)

Before deploying, you can test locally:

```bash
# Set environment
export SATGATE_RUNTIME=gateway
export MODE=demo
export LIGHTNING_BACKEND=mock  # No real payments in test
export L402_ROOT_KEY=test-key-for-local-dev-only

# Point SatGate at your config (choose ONE option)
export SATGATE_GATEWAY_CONFIG=./my-gateway.yaml
# OR: cp my-gateway.yaml satgate.gateway.yaml

# Run the gateway
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

> **Coming soon:** SatGate Cloud (hosted gateway + managed Lightning).  
> Until then, use **Option B: Self-Hosted** with the same config model.

When SatGate Cloud is available:
1. **Upload your config** (or paste YAML in dashboard)
2. **Get your gateway URL** (example): `https://your-project.satgate.cloud`
3. **Update your DNS** (optional): CNAME `api.yourcompany.com` → `your-project.satgate.cloud`

### Option B: Self-Hosted (Railway/Fly/Docker)

Deploy on your platform of choice (Railway/Fly/Docker) using the same config + env vars.

```bash
# Set production environment
SATGATE_RUNTIME=gateway
MODE=prod
# Lightning backend (recommended for SMB self-host): phoenixd
LIGHTNING_BACKEND=phoenixd
PHOENIXD_URL=http://localhost:9740
PHOENIXD_PASSWORD=<phoenixd-password>

# Or Lightning backend: LND
# LIGHTNING_BACKEND=lnd
# LND_REST_URL=https://your-lnd-node:8080
# LND_MACAROON=<base64-macaroon>
L402_ROOT_KEY=<32-char-random-key>
PRICING_ADMIN_TOKEN=<32-char-random-key>
```

---

## Step 4: Verify Your Gateway

> Replace `<your-gateway-host>` with your actual gateway URL:
> - **Self-hosted:** Your Railway/Fly/Docker URL (e.g., `my-gateway.up.railway.app`)
> - **SatGate Cloud:** Provided when Cloud launches (e.g., `your-project.satgate.cloud`)

### Test 1: Health Check

```bash
curl https://<your-gateway-host>/healthz
# Expected: {"status":"ok","plane":"data",...}
```

### Test 2: L402 Challenge

```bash
curl -i https://<your-gateway-host>/v1/premium/test
# Expected: HTTP 402 + WWW-Authenticate: L402 macaroon="...", invoice="lnbc..."
```

### Test 3: Fail-Closed

```bash
curl -i https://<your-gateway-host>/unknown
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
  https://<your-gateway-host>/v1/premium/test
# Expected: HTTP 200 + response from your upstream API
```

---

## Step 5: Go Live

### DNS Setup (Optional but Recommended)

Point your API subdomain to your SatGate gateway:

```
api.yourcompany.com  CNAME  <your-gateway-host>
```

> **Example:** `api.yourcompany.com CNAME my-gateway.up.railway.app`

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

- ✅ Hosted gateway endpoint (SatGate Cloud)
- ✅ Managed Lightning (SatGate Cloud)
- ✅ Metering + basic analytics (SatGate Cloud)
- ✅ Route/pricing configuration
- ✅ API keys + rotation

### What's Not Included (v1)

- ❌ VPC peering / private upstreams (require public URL)
- ❌ Enterprise SSO / RBAC
- ❌ Custom domains (use CNAME)
- ❌ Multi-region (single region in v1)

> 💡 Need enterprise features? Contact sales@satgate.io for **SatGate Operate** (managed deployment in your infra).

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

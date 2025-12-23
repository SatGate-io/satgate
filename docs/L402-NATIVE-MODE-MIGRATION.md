# L402 Native Mode Migration Guide

> **Migrate from Aperture Sidecar to SatGate as the L402 Authority**

This guide explains how to switch from the current Aperture-based L402 setup to SatGate Native Mode, where SatGate handles all L402 challenges directly.

---

## When to Migrate

| Use Case | Recommended Mode |
|----------|------------------|
| Quick demo with existing LND/Voltage | **Aperture Sidecar** (current) |
| Production with per-request metering | **L402 Native Mode** |
| Using phoenixd (self-custodial) | **L402 Native Mode** |
| Simplify deployment (one service) | **L402 Native Mode** |
| Need `max_calls` / `budget_sats` enforcement | **L402 Native Mode** |

---

## Architecture Comparison

### Current: Aperture Sidecar Mode

```
Internet → Aperture (L402) → SatGate → Your API
              ↓
           Voltage LND
```

- Aperture handles 402 challenges and payment verification
- SatGate handles capabilities, governance, dashboard
- Requires LNC credentials for Aperture

### Target: L402 Native Mode

```
Internet → SatGate (L402 + Governance) → Your API
              ↓
         Lightning Backend
         (phoenixd / LND)
```

- SatGate handles everything
- Simpler deployment (one service)
- Supports multiple Lightning backends
- Enables per-request metering

> **Note on OpenNode**: OpenNode is supported as a Lightning backend only if the API exposes a **real Lightning payment hash** (sha256(preimage)). If it does not, SatGate cannot verify LSAT preimages reliably. Prefer **phoenixd** or **LND** for Native Mode.

---

## Migration Steps

### Step 1: Choose Your Lightning Backend

| Backend | Best For | Credentials Needed |
|---------|----------|-------------------|
| **phoenixd** | Self-custodial, easy setup | URL + Password |
| **LND** | Enterprise, existing infrastructure | REST URL + Macaroon |
| **Alby Hub / Start9** | If it provides phoenixd or LND access | phoenixd or LND credentials |
| **mock** | Testing/demos | None |

### Step 2: Get Lightning Credentials

#### Option A: Alby Hub on Start9 (Your Setup)

Alby Hub can connect via its internal phoenixd OR an external LND. Check your Alby Hub settings:

1. **SSH into Start9** or use the web terminal
2. **Check Alby Hub config:**
   ```bash
   # Find Alby Hub data directory
   cat /embassy-data/package-data/volumes/albyhub/data/start9/config.yaml
   ```

3. **If using internal phoenixd:**
   - Look for phoenixd port (usually 9740)
   - Password is in the phoenixd config

4. **If connected to LND:**
   - Get LND REST URL (usually port 8080)
   - Export admin macaroon:
     ```bash
     # On Start9, find LND macaroon
     xxd -p /path/to/admin.macaroon | tr -d '\n'
     ```

#### Option B: Run Standalone phoenixd

```bash
# Download and run phoenixd
curl -L https://github.com/ACINQ/phoenixd/releases/latest/download/phoenixd-linux-x86_64.zip -o phoenixd.zip
unzip phoenixd.zip && cd phoenixd-*

# Run on testnet first
./phoenixd --chain=testnet

# Get credentials
cat ~/.phoenix/phoenix.conf
# Look for: http-password=<your-password>
# API runs at: http://localhost:9740
```

### Step 3: Set Up Network Access

Your Lightning backend needs to be reachable from Railway. Options:

| Method | Setup | Security |
|--------|-------|----------|
| **Tailscale** | Install on Start9 + Railway | ✅ Best (encrypted mesh VPN) |
| **Cloudflare Tunnel** | `cloudflared` on Start9 | ✅ Good (no port forward) |
| **Tor** | Use .onion address | ✅ Good (built into Start9) |
| **Port Forward** | Router config | ⚠️ Risky (exposes node) |

#### Tailscale Setup (Recommended)

1. **On Start9:**
   ```bash
   # Install Tailscale (if not already)
   curl -fsSL https://tailscale.com/install.sh | sh
   tailscale up
   ```

2. **Note your Tailscale IP:**
   ```bash
   tailscale ip -4
   # Example: 100.x.y.z
   ```

3. **On Railway:**
   - The Railway service will also need Tailscale
   - Or use Tailscale Funnel for public access

### Step 4: Create New Railway Service

#### Option A: Modify Existing Deployment

Update `deploy/supervisord.conf` to disable Aperture:

```ini
[supervisord]
nodaemon=true
logfile=/dev/stdout
logfile_maxbytes=0
loglevel=info

[program:backend]
command=node /app/backend/server.js
directory=/app/backend
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
; IMPORTANT: SatGate listens on BACKEND_PORT (default 8083).
; On Railway, set BACKEND_PORT to match Railway's PORT (recommended: BACKEND_PORT="%(ENV_PORT)s").
environment=BACKEND_PORT="%(ENV_PORT)s",NODE_ENV="production",CORS_ORIGINS="%(ENV_CORS_ORIGINS)s"

# DISABLED: Aperture no longer needed in L402 Native Mode
# [program:aperture]
# command=...
```

Note: Railway provides `PORT`. SatGate uses `BACKEND_PORT`, so set `BACKEND_PORT=%(ENV_PORT)s` (or set `BACKEND_PORT=$PORT` in Railway vars).

#### Option B: Create Separate Service (Safer)

1. Create a new Railway project: `satgate-native`
2. Use a simplified Dockerfile:

```dockerfile
FROM node:20-slim
WORKDIR /app
COPY proxy/package*.json ./
RUN npm ci --production
COPY proxy/ ./
EXPOSE 8083
CMD ["node", "server.js"]
```

3. Set environment variables (see Step 5)

### Step 5: Configure Environment Variables

Add these to Railway:

```bash
# Core
MODE=prod
NODE_ENV=production
# Railway: set BACKEND_PORT=$PORT (recommended) or expose 8083
BACKEND_PORT=${PORT}

# L402 Native Mode
L402_MODE=native
LIGHTNING_BACKEND=phoenixd

# Lightning Credentials (phoenixd)
PHOENIXD_URL=http://100.x.y.z:9740  # Your Tailscale IP
PHOENIXD_PASSWORD=your-password-here

# OR Lightning Credentials (LND)
# LIGHTNING_BACKEND=lnd
# LND_REST_URL=https://100.x.y.z:8080
# LND_MACAROON=0201036c6e64...

# Optional generic override (advanced)
# LIGHTNING_URL=http://100.x.y.z:9740   # Overrides PHOENIXD_URL/LND_REST_URL

# L402 Token Settings
L402_DEFAULT_TTL=3600
L402_DEFAULT_MAX_CALLS=100

# Admin & Security
PRICING_ADMIN_TOKEN=your-secure-admin-token
CAPABILITY_ROOT_KEY=your-secure-root-key
L402_ROOT_KEY=your-secure-l402-root-key  # REQUIRED in native mode for production

# Optional: Redis for persistence
# REDIS_URL=redis://...
```

### Step 6: Test the Migration

```bash
# Set your new service URL
export SATGATE_URL=https://your-new-service.up.railway.app
export SATGATE_ADMIN_TOKEN=your-secure-admin-token

# 1. Health check
curl $SATGATE_URL/health
# Expected: {"status":"ok"}

# 2. Check L402 mode
curl -H "X-Admin-Token: $SATGATE_ADMIN_TOKEN" $SATGATE_URL/api/governance/info | jq .features
# Expected: "l402Mode": "native", "l402Native": true

# (Optional) 2b. Check Lightning backend status (admin-only)
curl -H "X-Admin-Token: $SATGATE_ADMIN_TOKEN" $SATGATE_URL/api/governance/lightning | jq .

# 3. Test 402 challenge
curl -i $SATGATE_URL/api/micro/ping
# Expected: HTTP 402 with WWW-Authenticate: L402 macaroon="...", invoice="lnbc..."

# 4. Verify invoice is from your node (not Aperture)
# The invoice should be payable to YOUR phoenixd/LND node
```

### Step 7: Update DNS (When Ready)

Once tested, point your domain to the new service:

```
satgate.io → satgate-native.up.railway.app
```

---

## Rollback Plan

If issues arise, switch back to Aperture mode:

1. Set `L402_MODE=aperture` (or remove the variable)
2. Redeploy with the original `supervisord.conf`
3. Traffic flows through Aperture again

---

## Troubleshooting

### "Lightning backend unreachable"

```bash
# Test connectivity from Railway
curl -v http://YOUR_PHOENIXD_URL:9740/getinfo
```

- Check Tailscale/Cloudflare tunnel is running
- Verify firewall allows port 9740
- Check phoenixd is running: `pgrep phoenixd`

### "Invoice creation failed"

- Verify `PHOENIXD_PASSWORD` is correct
- Check phoenixd has inbound liquidity
- For testnet, ensure you're on the right network

### "LSAT validation failed"

- Ensure `L402_ROOT_KEY` or `CAPABILITY_ROOT_KEY` is set
- Check Redis is connected (for metering)
- Verify preimage format (hex, no 0x prefix)

---

## Feature Comparison

| Feature | Aperture Mode | Native Mode |
|---------|--------------|-------------|
| 402 Challenge | ✅ Aperture | ✅ SatGate |
| LSAT Validation | ✅ Aperture | ✅ SatGate |
| `max_calls` Caveat | ❌ | ✅ Redis counters |
| `budget_sats` Caveat | ❌ | ✅ Redis counters |
| Re-challenge on Exhaust | ❌ | ✅ Automatic |
| Lightning Backends | LND only | phoenixd, LND (OpenNode experimental) |
| Deployment Complexity | Higher (2 services) | Lower (1 service) |

---

## Next Steps After Migration

1. **Enable Redis** for persistent metering across restarts
2. **Set up monitoring** for Lightning backend health
3. **Update SDKs** to handle re-challenges (already supported)
4. **Document new endpoint** for your API consumers

---

*Last Updated: December 2025 | SatGate v1.9.0*


# SatGate Deployment

Deploy Aperture + Demo API to Railway for live demos from anywhere.

## Prerequisites

1. **Voltage Cloud** account with LND node (you already have this)
2. **LNC (Lightning Node Connect) pairing phrase** from Voltage
3. **Railway** account (free tier works)

## Quick Deploy to Railway

### Option 1: One-Click Deploy

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/satgate)

### Option 2: Manual Deploy

1. **Create Railway Project**
   ```bash
   # Install Railway CLI
   npm install -g @railway/cli
   
   # Login
   railway login
   
   # Create new project
   railway init
   ```

2. **Set Environment Variables**
   
   In Railway dashboard, add these variables:
   
   | Variable | Value | Description |
   |----------|-------|-------------|
   | `LNC_PASSPHRASE` | Your Voltage pairing phrase | **Required** - Get from Voltage dashboard |
   | `LNC_MAILBOX` | `mailbox.terminal.lightning.today:443` | Default LNC mailbox |
   | `LNC_NETWORK` | `mainnet` | Or `testnet` for testing |
   | `PORT` | `8081` | Railway will use this |

3. **Deploy**
   ```bash
   railway up
   ```

4. **Get Your URL**
   
   Railway will give you a URL like: `satgate-demo-production.up.railway.app`

## Getting Your LNC Passphrase from Voltage

1. Log into [Voltage Cloud](https://voltage.cloud)
2. Select your node
3. Go to **Connect** → **Lightning Node Connect**
4. Click **Create New Session**
5. Copy the **Pairing Phrase** (keep this secret!)

## Testing the Deployment

```bash
# Check health
curl https://YOUR-RAILWAY-URL.up.railway.app/health

# Test free endpoint
curl https://YOUR-RAILWAY-URL.up.railway.app/api/free/ping

# Test L402 (should return 402 Payment Required)
curl -i https://YOUR-RAILWAY-URL.up.railway.app/api/micro/ping
```

## Connecting the Playground

Update `satgate-landing/app/playground/page.tsx`:

```typescript
const TARGET_URL = 'https://YOUR-RAILWAY-URL.up.railway.app/api/micro/ping';
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Railway Container                                      │
│  ┌─────────────────┐     ┌─────────────────┐           │
│  │  Aperture       │────►│  Backend API    │           │
│  │  (port 8081)    │     │  (port 8083)    │           │
│  └────────┬────────┘     └─────────────────┘           │
│           │ LNC                                         │
└───────────┼─────────────────────────────────────────────┘
            │
            ▼
       ┌──────────┐
       │ Voltage  │  (Your existing node)
       │   LND    │
       └──────────┘
```

## Costs

- **Railway**: ~$5-10/month for basic usage
- **Voltage**: Your existing plan (no change)
- **Lightning fees**: Negligible (1 sat invoices)

## Troubleshooting

### "LNC connection failed"
- Check your pairing phrase is correct
- Ensure your Voltage node is online
- Try creating a new LNC session in Voltage

### "502 Bad Gateway"
- Container may still be starting (wait 30-60 seconds)
- Check Railway logs for errors

### "CORS errors in browser"
- The backend is configured with `CORS_ORIGINS="*"` for demo purposes
- For production, restrict to your domain

## Security Notes

⚠️ This deployment is for **demos only**. For production:
- Use a dedicated Lightning node with limited funds
- Set specific CORS origins
- Add rate limiting at the edge
- Consider adding authentication for admin endpoints


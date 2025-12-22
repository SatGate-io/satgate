# phoenixd on Railway

Deploy ACINQ's phoenixd as a Railway service for automatic Lightning liquidity.

## Why phoenixd?

- **Zero liquidity management** - ACINQ's LSP auto-opens channels when you receive payments
- **Self-custodial** - You control the keys
- **Simple API** - REST interface compatible with SatGate
- **~1% fee** - Small price for zero maintenance

## Quick Deploy

### Step 1: Create New Service in Railway

1. Go to your Railway project dashboard
2. Click **+ New** → **Empty Service**
3. Name it `phoenixd`

### Step 2: Connect GitHub

1. In the new service, click **Settings** → **Source**
2. Connect to your `satgate` repo
3. Set **Root Directory** to `deploy/phoenixd`

### Step 3: Add Volume (CRITICAL)

phoenixd stores your wallet keys in `/data`. Without a volume, you lose funds on restart!

1. Click **+ New** → **Volume**
2. Mount path: `/data`
3. Attach to `phoenixd` service

### Step 4: Set Environment Variables

```
PHOENIX_CHAIN=mainnet
PHOENIXD_PASSWORD=your-secure-password-here
```

Generate a secure password:
```bash
openssl rand -hex 32
```

### Step 5: Deploy

Railway will auto-deploy. First boot takes ~2 minutes.

### Step 6: Get Internal URL

After deploy, go to **Settings** → **Networking** and note the private URL:
```
phoenixd.railway.internal:9740
```

### Step 7: Configure SatGate

Add these env vars to your main SatGate service:

```
LIGHTNING_PROVIDER=phoenixd
PHOENIXD_URL=http://phoenixd.railway.internal:9740
PHOENIXD_PASSWORD=your-secure-password-here
```

## First Payment

When you receive your first Lightning payment to phoenixd:

1. ACINQ LSP opens a channel automatically
2. You pay ~1% of the inbound amount as LSP fee
3. Future payments flow instantly through this channel

**No Amboss, no manual channels, no liquidity stress!**

## Monitoring

### Check Status
```bash
curl -u :$PHOENIXD_PASSWORD http://phoenixd.railway.internal:9740/getinfo
```

### Check Balance
```bash
curl -u :$PHOENIXD_PASSWORD http://phoenixd.railway.internal:9740/getbalance
```

### List Channels
```bash
curl -u :$PHOENIXD_PASSWORD http://phoenixd.railway.internal:9740/listchannels
```

## Withdrawing Funds

Send to another Lightning wallet:
```bash
curl -u :$PHOENIXD_PASSWORD \
  -X POST http://phoenixd.railway.internal:9740/payinvoice \
  -d "invoice=lnbc..."
```

## Security Notes

1. **Backup your seed** - On first run, phoenixd generates a seed. Find it at `/data/.phoenix/seed.dat`
2. **Keep password secret** - Anyone with the password controls your funds
3. **Use Railway's private networking** - Don't expose 9740 to the public internet

## Costs

| Item | Cost |
|------|------|
| Railway Hobby | $5/month |
| phoenixd RAM | ~200MB |
| LSP Fee | ~1% per inbound |

## Troubleshooting

### "Connection refused"
- Wait 2-3 minutes after first deploy
- Check Railway logs for startup errors

### "No route found"  
- This won't happen! LSP always has a route to you

### "Insufficient balance"
- phoenixd is receiving only - send BTC via invoice first

## Architecture

```
[Client] → [SatGate] → [phoenixd] ←→ [ACINQ LSP] ←→ [Lightning Network]
                           ↓
                     [Your Wallet]
                    (self-custodial)
```


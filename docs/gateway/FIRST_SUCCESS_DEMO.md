<p align="center">
  <img src="../assets/brand/logo_blue_transparent.png" alt="SatGate™" width="140" />
</p>

# SatGate™ Gateway Mode — First Success Demo

**Goal:** Verify your gateway is working: 403 fail-closed → 402 challenge → pay → 200 success.

**Time:** 5 minutes

---

## Prerequisites

- Gateway deployed and running (see [DEPLOY_SELF_HOSTED.md](./DEPLOY_SELF_HOSTED.md))
- Gateway URL known (e.g., `https://my-gateway.up.railway.app`)

---

## Step 0: Set Your Gateway URL

```bash
# Replace with your actual gateway URL
export GATEWAY_URL="https://my-gateway.up.railway.app"
```

---

## Step 1: Health Check ✓

```bash
curl -s $GATEWAY_URL/healthz | jq .
```

**Expected output:**
```json
{
  "status": "ok",
  "plane": "data",
  "timestamp": "2025-01-15T10:30:00.000Z"
}
```

**If this fails:** Check that your gateway is running and the URL is correct.

---

## Step 2: Fail-Closed (403) ✓

Test that unknown paths are blocked:

```bash
curl -i $GATEWAY_URL/unknown/path
```

**Expected output:**
```
HTTP/2 403
content-type: application/json

{"error":"Forbidden","message":"Access denied by policy","route":"default-deny"}
```

**What this proves:** Your gateway is fail-closed. Paths not explicitly routed return 403.

---

## Step 3: L402 Challenge (402) ✓

Hit a paid endpoint without payment:

```bash
curl -i $GATEWAY_URL/v1/basic/test
```

**Expected output:**
```
HTTP/2 402
www-authenticate: L402 macaroon="eyJ...", invoice="lnbc..."
x-l402-price: 10
x-l402-tier: basic
content-type: application/json

{
  "error": "Payment Required",
  "code": "PAYMENT_REQUIRED",
  "message": "This endpoint requires 10 sats...",
  "price": 10,
  "tier": "basic",
  "invoice": "lnbc100n1p...",
  "macaroon": "eyJ...",
  ...
}
```

**What this proves:** Your L402 enforcement is working. Lightning invoices are being generated.

---

## Step 4: Extract Invoice and Macaroon

Parse the 402 response to get the invoice and macaroon:

```bash
# Get the 402 response
RESPONSE=$(curl -s $GATEWAY_URL/v1/basic/test)

# Extract invoice and macaroon
INVOICE=$(echo $RESPONSE | jq -r '.invoice')
MACAROON=$(echo $RESPONSE | jq -r '.macaroon')

echo "Invoice: ${INVOICE:0:50}..."
echo "Macaroon: ${MACAROON:0:50}..."
```

---

## Step 5: Pay the Invoice

Pay the Lightning invoice using any wallet:

### Option A: Alby Browser Extension
1. Copy the invoice: `echo $INVOICE`
2. Open Alby → Send → Paste invoice → Pay

### Option B: CLI Wallet (e.g., lncli)
```bash
lncli payinvoice $INVOICE
```

### Option C: Phoenix Mobile
1. Open Phoenix → Send → Scan QR or paste invoice

After payment, you'll receive a **preimage** (64-character hex string).

```bash
# Save the preimage you received
PREIMAGE="<paste-your-preimage-here>"
```

---

## Step 6: Access with LSAT Token (200) ✓

Retry the request with the LSAT token:

```bash
curl -i -H "Authorization: LSAT ${MACAROON}:${PREIMAGE}" \
  $GATEWAY_URL/v1/basic/test
```

**Expected output:**
```
HTTP/2 200
x-satgate-route: basic
x-satgate-tier: basic
content-type: application/json

{
  // Response from your upstream API
}
```

**What this proves:** The full L402 flow works. Paid requests are proxied to your upstream.

---

## Quick Copy-Paste Version

Run this entire flow in one terminal session:

```bash
# 1. Set your gateway URL
GATEWAY_URL="https://my-gateway.up.railway.app"

# 2. Health check
echo "=== Health Check ===" && curl -s $GATEWAY_URL/healthz | jq -r '.status'

# 3. Fail-closed test
echo -e "\n=== Fail-Closed (expect 403) ===" && curl -s -o /dev/null -w "%{http_code}" $GATEWAY_URL/unknown

# 4. L402 challenge
echo -e "\n=== L402 Challenge (expect 402) ===" && curl -s -o /dev/null -w "%{http_code}" $GATEWAY_URL/v1/basic/test

# 5. Get invoice
echo -e "\n=== Invoice ===" 
RESPONSE=$(curl -s $GATEWAY_URL/v1/basic/test)
INVOICE=$(echo $RESPONSE | jq -r '.invoice')
MACAROON=$(echo $RESPONSE | jq -r '.macaroon')
echo "Invoice: ${INVOICE:0:60}..."
echo "Macaroon: ${MACAROON:0:40}..."

# 6. Pay the invoice (manual step)
echo -e "\n=== Pay this invoice, then set PREIMAGE ===" 
echo $INVOICE

# 7. After payment, set preimage and retry:
# PREIMAGE="your-preimage-here"
# curl -H "Authorization: LSAT ${MACAROON}:${PREIMAGE}" $GATEWAY_URL/v1/basic/test
```

---

## Troubleshooting

### Health check returns connection error

- Check gateway URL is correct
- Verify deployment is running (Railway/Fly dashboard)
- Check logs for startup errors

### 402 but no invoice (or invalid invoice)

- Check Lightning backend is configured (`LIGHTNING_BACKEND`, `PHOENIXD_URL`, etc.)
- Verify Lightning backend is reachable from gateway
- Check gateway logs for Lightning errors

### Payment fails with "no route"

- Your Lightning node may not have channels/liquidity
- For small amounts (1-10 sats), try a wallet with better routing (Phoenix, Alby)
- If using LND: ensure channels are active and have inbound liquidity

### 200 returns error from upstream

- Your upstream API returned an error (not a gateway issue)
- Check upstream is configured correctly in `satgate.gateway.yaml`
- Verify upstream URL is reachable

### Token rejected after payment

- Ensure you're using the exact macaroon from the 402 response
- Ensure preimage is correct (64-character hex string)
- Check that the token hasn't expired (default: 1 hour)

---

## Success Checklist

- [ ] `/healthz` returns `{"status":"ok"}`
- [ ] Unknown paths return 403 (fail-closed)
- [ ] Paid endpoints return 402 with invoice
- [ ] After payment, LSAT token grants access (200)
- [ ] Response is proxied from upstream API

**Congratulations!** Your SatGate proxy is fully operational.

---

## Next Steps

1. **Add more routes:** Edit `satgate.gateway.yaml` to protect more endpoints
2. **Adjust pricing:** Change `priceSats` per route
3. **Monitor:** Check logs for request counts and revenue
4. **Go live:** Point your DNS to the gateway


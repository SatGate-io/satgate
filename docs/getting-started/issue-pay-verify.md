# Issue → Pay → Verify Quick Start

Run a public L402 paid-retry path locally: first request gets `402 Payment Required`, the mock Lightning provider settles the invoice, and the retry reaches the upstream with an `Authorization: L402 ...` proof.

> This uses SatGate's `mock` Lightning provider for local demos and CI. Real Lightning backends still require an external wallet/payment flow; `/api/l402/mock-pay` is not available there.

## 1. Configure a paid route

Create `gateway.yaml`:

```yaml
version: 1

server:
  listen: ":8080"

admin:
  token: "my-admin-token"

lightning:
  provider: mock

upstreams:
  httpbin:
    url: "https://httpbin.org"

routes:
  - name: paid-demo
    match:
      pathPrefix: /premium
    upstream: httpbin
    stripPrefix: true
    policy:
      kind: l402
      priceSats: 10
```

Start SatGate:

```bash
export ADMIN_TOKEN=my-admin-token
satgate --config gateway.yaml
```

If you built from source, use `./satgate --config gateway.yaml`.

## 2. Issue: request the paid route

```bash
curl -i http://localhost:8080/premium/anything
```

Expected result:

- HTTP status: `402 Payment Required`
- Header: `WWW-Authenticate: L402 macaroon="...", invoice="..."`
- Body fields: `payment_hash`, `invoice`, `amount_sats`

Capture the challenge:

```bash
CHALLENGE=$(curl -s -D /tmp/satgate-l402.headers \
  http://localhost:8080/premium/anything \
  -o /tmp/satgate-l402.json \
  -w '%{http_code}')

test "$CHALLENGE" = "402"
MACAROON=$(python3 - <<'PY'
import re
headers=open('/tmp/satgate-l402.headers').read()
print(re.search(r'macaroon="([^"]+)"', headers).group(1))
PY
)
PAYMENT_HASH=$(python3 - <<'PY'
import json
print(json.load(open('/tmp/satgate-l402.json'))['payment_hash'])
PY
)
```

## 3. Pay: settle the mock invoice

```bash
AUTHORIZATION=$(curl -s -X POST http://localhost:8080/api/l402/mock-pay \
  -H 'Content-Type: application/json' \
  -H 'X-Admin-Token: my-admin-token' \
  -d "{\"payment_hash\":\"$PAYMENT_HASH\",\"macaroon\":\"$MACAROON\"}" \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["authorization"])')
```

Expected result: `AUTHORIZATION` starts with `L402 ` and contains the paid macaroon plus payment preimage.

## 4. Verify: retry with the L402 proof

```bash
curl -i http://localhost:8080/premium/anything \
  -H "Authorization: $AUTHORIZATION"
```

Expected result:

- HTTP status: `200 OK`
- The upstream response is returned.
- Reusing the same `AUTHORIZATION` fails because SatGate's L402 replay guard permits each payment proof once.

## What this proves

- The route issues a real HTTP `402` L402 challenge.
- The mock Lightning provider can complete a local payment without a wallet.
- The retry is authorized by the L402 macaroon plus preimage.
- SatGate proxies only after payment proof verification.

## What this does not prove

- Real Lightning settlement on Phoenixd, LND, NWC, or Alby.
- SatGate Cloud billing or buyer checkout.
- Native Evidence Pack export from the gateway response.

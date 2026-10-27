# Lightning Provider Configuration

SatGate supports multiple Lightning backends for L402 payments.

## Supported Providers

| Provider | Status | Description |
|----------|--------|-------------|
| `nwc` | ✅ Supported | Nostr Wallet Connect — works with Alby Hub and any NWC-compatible wallet |
| `alby` | ✅ Supported | Alby API (legacy, prefer NWC) |
| `phoenixd` | ✅ Supported | Phoenix daemon — lightweight, auto-managed channels |
| `lnd` | ✅ Supported | LND — full Lightning node |
| `lnbits` | ✅ Supported | LNbits — custodial or self-hosted |
| `mock` | ✅ Supported | Mock provider for development/testing |
| `disabled` | ✅ Supported | Explicitly disable Lightning |

## Configuration

Set the provider via config file or environment variables:

```yaml
lightning:
  provider: nwc
  config:
    connectionString: "${NWC_CONNECTION_STRING}"
```

**Environment variables:** Set `LIGHTNING_BACKEND` or `LIGHTNING_PROVIDER` to choose the provider. Defaults to `mock` if neither is set.

### NWC (Nostr Wallet Connect) — Recommended

Best option for most setups. Works with Alby Hub, Mutiny, and other NWC wallets.

```yaml
lightning:
  provider: nwc
  config:
    connectionString: "nostr+walletconnect://..."
```

Get your connection string from Alby Hub: **Settings → Wallet Connections → Add Connection**.

### Phoenixd

Lightweight Lightning daemon with auto-managed channels.

```yaml
lightning:
  provider: phoenixd
  config:
    apiUrl: "http://localhost:9740"
    apiPassword: "${PHOENIXD_PASSWORD}"
```

### LND

Full Lightning Network Daemon. Works with Start9, Umbrel, or standalone LND.

```yaml
lightning:
  provider: lnd
  config:
    restUrl: "https://your-lnd:8080"
    macaroon: "${LND_MACAROON}"
    tlsCert: "${LND_TLS_CERT}"
    insecureSkipVerify: false
```

### LNbits

```yaml
lightning:
  provider: lnbits
  config:
    url: "https://your-lnbits-instance.com"
    apiKey: "${LNBITS_API_KEY}"
```

### Mock (Development)

Auto-accepts all payments. Use for testing only.

```yaml
lightning:
  provider: mock
```

## L402 Security

```yaml
lightning:
  l402RootKey: "${L402_ROOT_KEY}"          # REQUIRED: 32+ byte random key (base64)
  requireInvoiceRecord: true               # Fail-closed: require invoice in store
  verifyWithNode: false                    # Check node if no local record
```

**Critical:** Set `L402_ROOT_KEY` to a cryptographically random value. Never derive it from provider names or predictable inputs.

## L402 Payment Proof Binding (open-source gateway)

Each `402 Payment Required` challenge carries a macaroon that the gateway
issues for one payment, one route and one price. Besides the expiry and
`scope = l402`, it holds these server-written caveats:

| Caveat | Value |
|--------|-------|
| `payment_hash` | Payment hash of the invoice in the challenge |
| `route` | Name of the route that issued the challenge |
| `price_sats` | The route's `priceSats` when the challenge was issued |
| `invoice_amount_sats` | Invoice amount reported by the provider, when it reports one |

A paid retry (`Authorization: L402 <macaroon>:<preimage>`) is accepted only when:

- the macaroon was issued by this gateway as an L402 challenge (its identifier starts with `satgate-l402-v2:`);
- `route` names the route being served and `price_sats` equals that route's current `priceSats` (and `invoice_amount_sats`, when present, equals it too);
- `SHA256(preimage)` equals `payment_hash`;
- the payment hash has not been used before (see the consumed-proof store below).

So a proof paid on one route does not serve another route, and changing a
route's `priceSats` refuses proofs issued at the old price; the client pays
the new challenge. A refused proof is not consumed.

`payment_hash`, `route`, `price_sats` and `invoice_amount_sats` are
single-valued. A token that carries any of them more than once is refused,
whatever the order. `/api/capability/delegate` refuses to add them. Caveats
that attenuation may repeat, such as `expires` and `scope`, are still checked
on every occurrence, so a holder can narrow a proof but not widen or rebind it.

## L402 Consumed-Proof Store

Each payment hash is accepted once. The gateway records consumed hashes for
24 hours, which is longer than the one-hour lifetime of a challenge macaroon.

- With `redis.enabled: true`, consumed hashes are stored in Redis under
  `satgate:l402:consumed:<payment_hash>` with that TTL, and every gateway
  instance that uses the same Redis sees them. The gateway refuses to start if
  the configured Redis cannot be reached, and a paid retry gets
  `503 Service Unavailable` (the proof is not accepted) while Redis is
  unreachable.
- Without Redis, consumed hashes are kept in process memory. This is correct
  for a **single gateway instance only**: a restart, or a second replica behind
  the same load balancer, does not see hashes the other process consumed. The
  gateway logs a warning at startup when L402 routes are configured without
  Redis.

The admin API rate limiter of the open-source gateway is a separate,
in-process limiter. It does not use Redis; with more than one replica each
replica applies its own limit.

### Upgrading

Challenge macaroons issued before route and price binding carry only
`payment_hash`. After the upgrade they are refused; clients holding one get a
new `402` challenge and pay it. Requests without a token are unaffected.

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

Route names are unique: the gateway refuses to start with two routes of the
same name, and refuses to promote an agent route whose name is in use (see
[Route names](../reference/configuration.md#route-names)). A challenge
therefore names exactly one route. If the live route table ever holds two
routes with the served route's name, the gateway issues no challenge for that
route and accepts no proof on it.

A paid retry (`Authorization: L402 <macaroon>:<preimage>`) is accepted only
when all of these hold; the payment hash is consumed only after every other
check passes:

- the macaroon verifies under this gateway's root key and was issued as an L402 challenge (its identifier starts with `satgate-l402-v2:`);
- every `expires` caveat is in the future;
- `payment_hash`, `route`, `price_sats` and `invoice_amount_sats` each appear at most once, and `payment_hash`, `route` and `price_sats` are present;
- the served route's name is unique, `route` names it, and `price_sats` equals its current `priceSats` (and `invoice_amount_sats`, when present, equals it too);
- every restriction a holder added covers the request (below);
- `SHA256(preimage)` equals `payment_hash`;
- the payment hash has not been used before (see the consumed-proof store below).

So a proof paid on one route does not serve another route, and changing a
route's `priceSats` refuses proofs issued at the old price; the client pays
the new challenge. A refused proof is not consumed, and the client gets a new
`402` challenge.

### Narrowing a proof

A holder can add first-party caveats to a challenge macaroon to narrow it.
On the L402 path the gateway accepts only these keys, and checks every
occurrence of each:

| Caveat | Accepted when |
|--------|---------------|
| `expires = <unix ms or RFC 3339>` | the time is in the future. A relative value such as `5m` refuses the proof. |
| `time < <unix seconds>` | the time is in the future. |
| `scope = <scope>` | the scope grants `l402` (`l402`, `*` or `api:*`). |
| `routes = <pattern>,...` | one pattern covers the request: a pattern starting with `/` is a path, matched exactly or, when it ends in `*`, as a prefix; `*` matches anything; any other pattern is a route name. |
| `methods = <METHOD>,...` | the request method is listed (case-insensitive). |

A proof that carries any other caveat key (other than the server-written
keys above) is refused, because the gateway does not enforce it.
`/api/capability/delegate` refuses to add the server-written keys. A holder
can therefore narrow a proof but cannot widen or rebind it.

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

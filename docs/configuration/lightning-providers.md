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

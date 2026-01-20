# Lightning Provider Configuration

SatGate Gateway supports multiple Lightning Network backends for L402 payments.

## Implemented Providers

| Provider | Type | Status | Recommended For |
|----------|------|--------|-----------------|
| **Phoenixd** | Self-hosted | ✅ Implemented | Development, Small scale |
| **LND** | Self-hosted | ✅ Implemented | Production, High volume |
| **Mock** | Testing | ✅ Implemented | Development only |

## Planned Providers (Not Yet Implemented)

| Provider | Type | Status | Notes |
|----------|------|--------|-------|
| Core Lightning | Self-hosted | 🔜 Planned | PR welcome |
| Alby Hub | Hosted | 🔜 Planned | NWC support |
| LNbits | Hosted/Self | 🔜 Planned | |
| BTCPay Server | Self-hosted | 🔜 Planned | |
| Voltage | Hosted | 🔜 Planned | Uses LND under the hood |
| Strike API | Hosted | 🔜 Planned | Custodial, fiat integration |

---

## Phoenixd

Self-custodial Lightning node with minimal setup. Recommended for development and small-scale deployments.

```yaml
lightning:
  provider: "phoenixd"
  config:
    apiUrl: "http://localhost:9740"
    apiPassword: "${PHOENIXD_PASSWORD}"
```

### Setup

1. Download Phoenixd:
```bash
curl -L https://github.com/acinq/phoenixd/releases/latest/download/phoenixd-linux-amd64.tar.gz | tar xz
```

2. Start Phoenixd:
```bash
./phoenixd --network=mainnet
```

3. Get the API password:
```bash
cat ~/.phoenix/phoenix.conf | grep http-password
```

---

## LND

Production-grade Lightning node. Recommended for high-volume production deployments.

```yaml
lightning:
  provider: "lnd"
  config:
    restUrl: "https://localhost:8080"
    macaroon: "0201036c6e640245..."  # Base64-encoded admin macaroon
    tlsCert: "LS0tLS1CRUdJ..."       # Base64-encoded TLS cert (optional)
    insecureSkipVerify: false        # Set to true only for development
```

### Getting Credentials

```bash
# Get macaroon
base64 ~/.lnd/data/chain/bitcoin/mainnet/admin.macaroon

# Get TLS cert
base64 ~/.lnd/tls.cert
```

### Security Notes

- Use a restricted macaroon with only `invoices:read` and `invoices:write` permissions
- Always use TLS in production
- Set `insecureSkipVerify: false` (the default) in production

---

## Mock Provider

For development and testing without real payments. Invoices are automatically marked as "paid".

```yaml
lightning:
  provider: "mock"
```

⚠️ **Never use mock in production!** All payments will be auto-confirmed.

---

## Testing Your Configuration

### Check Gateway Health

```bash
curl http://localhost:8080/healthz
```

Expected response:
```json
{"status":"ok"}
```

### Test L402 Challenge

```bash
# Without token - should return 402 Payment Required
curl -i http://localhost:8080/v1/premium/
```

Expected: `HTTP/1.1 402 Payment Required` with `WWW-Authenticate: L402 ...` header

---

## Security Best Practices

1. **Use environment variables for secrets**
```yaml
lightning:
  config:
    apiPassword: "${PHOENIXD_PASSWORD}"
```

2. **Limit macaroon permissions** (LND)
   - Only grant `invoices:read` and `invoices:write`
   - Don't use admin macaroons

3. **Use TLS for all connections**

4. **Rotate credentials regularly**

5. **Monitor balance and set alerts**

---

## Contributing New Providers

To add support for a new Lightning provider:

1. Implement the `lightning.Provider` interface in `gateway/internal/lightning/`
2. Add configuration struct in `gateway/internal/config/config.go`
3. Register the provider in `gateway/internal/lightning/interface.go`
4. Add documentation here
5. Submit a PR

See `gateway/internal/lightning/phoenixd.go` as a reference implementation.

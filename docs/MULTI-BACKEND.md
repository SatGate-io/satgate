# SatGate Multi-Backend Lightning Architecture

SatGate supports multiple Lightning backends, giving enterprises flexibility to choose the right infrastructure for their needs.

## Supported Providers

| Provider | Type | Setup | Liquidity | Custody | Best For |
|----------|------|-------|-----------|---------|----------|
| **Aperture/LNC** | Gateway | Medium | Manual | Self | Production, LND users |
| **phoenixd** | Node | Easy | Automatic | Self | Zero-maintenance |
| **LND (REST)** | Node | Hard | Manual | Self | Full control |
| **OpenNode** | Processor | Easy | N/A | Custodial | Quick start, Fiat |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         SatGate Server                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Lightning Provider Interface                 │  │
│  │                                                          │  │
│  │   createInvoice(sats, memo)  →  Invoice                  │  │
│  │   checkPayment(hash)         →  PaymentStatus            │  │
│  │   getStatus()                →  ProviderHealth           │  │
│  └────────────────────┬─────────────────────────────────────┘  │
│                       │                                         │
│         ┌─────────────┼─────────────┬─────────────┐            │
│         ▼             ▼             ▼             ▼            │
│    ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────┐     │
│    │ phoenixd │  │   LND   │  │OpenNode │  │  Aperture   │     │
│    │ Adapter  │  │ Adapter │  │ Adapter │  │   (Stub)    │     │
│    └────┬────┘  └────┬────┘  └────┬────┘  └──────┬──────┘     │
│         │            │            │               │            │
└─────────┼────────────┼────────────┼───────────────┼────────────┘
          │            │            │               │
          ▼            ▼            ▼               ▼
     ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────┐
     │phoenixd │  │   LND   │  │OpenNode │  │  Aperture   │
     │ Server  │  │  Node   │  │   API   │  │   + LNC     │
     └─────────┘  └─────────┘  └─────────┘  └─────────────┘
         │            │            │               │
         ▼            ▼            ▼               ▼
     ACINQ LSP    Lightning    OpenNode       Voltage/
                  Network      Custodial      Your LND
```

## Configuration

Set environment variables to configure your Lightning backend:

### Option 1: Aperture Mode (Current Default)

Aperture handles L402 challenges via LNC. This is the current production setup.

```bash
# These are already set in Railway
LIGHTNING_PROVIDER=aperture  # or omit (default)
LNC_PASSPHRASE=your-10-word-phrase
LNC_MAILBOX=mailbox.terminal.lightning.today:443
LNC_NETWORK=mainnet
```

### Option 2: phoenixd (Recommended for Easy Setup)

Zero liquidity management - ACINQ's LSP handles channel management automatically.

```bash
LIGHTNING_PROVIDER=phoenixd
PHOENIXD_URL=http://localhost:9740
PHOENIXD_PASSWORD=your-phoenixd-password
```

**Setup phoenixd:**
```bash
# Download from https://github.com/ACINQ/phoenixd/releases
./phoenixd

# Password is auto-generated in ~/.phoenix/phoenix.conf
cat ~/.phoenix/phoenix.conf | grep http-password
```

### Option 3: LND Direct (REST API)

Connect directly to your LND node without LNC/Aperture.

```bash
LIGHTNING_PROVIDER=lnd
LND_URL=https://your-lnd-node:8080
LND_MACAROON=hex-encoded-admin-macaroon
LND_CERT=base64-encoded-tls-cert  # Optional
```

### Option 4: OpenNode (Custodial)

Easiest setup - no node required. They handle everything.

```bash
LIGHTNING_PROVIDER=opennode
OPENNODE_API_KEY=your-api-key
OPENNODE_ENV=live  # or 'dev' for testnet
```

## Provider Comparison

### phoenixd (Recommended)

**Pros:**
- ✅ Zero liquidity management (LSP handles channels)
- ✅ Self-custodial (you control keys)
- ✅ Simple REST API
- ✅ Automatic channel creation on first payment
- ✅ Mobile-grade reliability (Phoenix wallet backend)

**Cons:**
- ❌ ~1% fee to ACINQ LSP per inbound payment
- ❌ Requires running a daemon (lightweight)

**Best for:** Production deployments that want simplicity without giving up custody.

### LND Direct

**Pros:**
- ✅ Full control over node
- ✅ No third-party dependencies
- ✅ Access to all LND features

**Cons:**
- ❌ Liquidity management is your problem
- ❌ Requires infrastructure expertise
- ❌ Channel maintenance overhead

**Best for:** Teams with existing LND infrastructure and Lightning expertise.

### OpenNode

**Pros:**
- ✅ Zero infrastructure
- ✅ Fiat settlement option
- ✅ Enterprise support available
- ✅ Works immediately

**Cons:**
- ❌ Custodial (they hold funds)
- ❌ 1% fee per payment
- ❌ KYC required for business accounts

**Best for:** Quick POCs, teams that prefer custodial simplicity.

### Aperture + LNC (Current)

**Pros:**
- ✅ Battle-tested in production
- ✅ Full L402 protocol support
- ✅ Works with remote LND (Voltage)

**Cons:**
- ❌ Liquidity management needed (the current problem)
- ❌ LNC pairing can be finicky
- ❌ Additional process to manage

**Best for:** Existing Voltage/LND users who manage their own liquidity.

## Migration Path

### From Aperture to phoenixd

1. Install phoenixd on your server
2. Update environment variables:
   ```bash
   LIGHTNING_PROVIDER=phoenixd
   PHOENIXD_URL=http://localhost:9740
   PHOENIXD_PASSWORD=your-password
   ```
3. Restart SatGate
4. First inbound payment will auto-create a channel

### Multi-Backend for Redundancy

Future: Run multiple backends with automatic failover.

```bash
LIGHTNING_PROVIDER=multi
LIGHTNING_PRIMARY=phoenixd
LIGHTNING_FALLBACK=opennode
```

## API Endpoints

### Check Provider Status

```bash
curl https://your-api.com/lightning/status
```

Response:
```json
{
  "mode": "native",
  "provider": "phoenixd",
  "ok": true,
  "info": {
    "nodeId": "03abc...",
    "channels": 2,
    "version": "0.1.0"
  }
}
```

## Roadmap

- [x] Provider interface abstraction
- [x] phoenixd adapter
- [x] LND REST adapter  
- [x] OpenNode adapter
- [ ] Breez SDK adapter (mobile-first)
- [ ] Multi-backend failover
- [ ] Automatic provider health monitoring
- [ ] Provider-specific fee optimization

## Enterprise Considerations

### For Compliance

| Requirement | phoenixd | OpenNode | LND |
|-------------|----------|----------|-----|
| Self-custody | ✅ | ❌ | ✅ |
| Audit trail | ✅ | ✅ | ✅ |
| Fiat settlement | ❌ | ✅ | ❌ |
| No KYC | ✅ | ❌ | ✅ |

### For Scale

| Factor | phoenixd | OpenNode | LND |
|--------|----------|----------|-----|
| Max TPS | ~50 | 1000+ | ~100 |
| Liquidity | Auto | Unlimited | Manual |
| Uptime SLA | N/A | 99.9% | Self |

---

*SatGate: Any Lightning backend, one API.*


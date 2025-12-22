# SatGate Python SDK

**Give your AI agents a Lightning wallet in 2 lines of code.**

Automatic L402 payment handling — the "Stripe Moment" for autonomous agents.

## Installation

```bash
pip install satgate
```

## Quick Start

```python
from satgate import SatGateClient, LNBitsWallet

# 1. Connect your wallet
wallet = LNBitsWallet(
    url="https://legend.lnbits.com",
    admin_key="your-admin-key"
)

# 2. Create client
client = SatGateClient(wallet)

# 3. That's it. 402 → Pay → Retry happens automatically.
response = client.get("https://api.example.com/premium/data")
print(response.json())
```

## What Happens Under the Hood

```
1. GET /premium/data
   ↓
2. Server returns 402 + Lightning Invoice
   ↓
3. SDK automatically pays invoice
   ↓
4. SDK retries with L402 token
   ↓
5. You get the response ✓
```

## Wallet Options

### LNBits

```python
from satgate import LNBitsWallet

wallet = LNBitsWallet(
    url="https://legend.lnbits.com",  # or your own instance
    admin_key="your-admin-key"
)
```

### Alby

```python
from satgate import AlbyWallet

wallet = AlbyWallet(access_token="your-alby-token")
```

### Custom Wallet

Implement the `LightningWallet` interface:

```python
from satgate import LightningWallet

class MyWallet(LightningWallet):
    def pay_invoice(self, invoice: str) -> str:
        # Connect to your LND, CLN, etc.
        preimage = my_node.pay(invoice)
        return preimage  # hex string
```

## Features

### Token Caching

Tokens are cached by default to avoid paying twice:

```python
client = SatGateClient(wallet, cache_tokens=True, cache_ttl=3600)

# First call: pays invoice
client.get("/premium")

# Second call: uses cached token (no payment)
client.get("/premium")
```

### Payment Callbacks

Track payments in real-time:

```python
def on_payment(info):
    print(f"Paid {info.amount_sats} sats for {info.endpoint}")
    # Log to your analytics, update UI, etc.

client = SatGateClient(wallet, on_payment=on_payment)
```

### Session Tracking

```python
client = SatGateClient(wallet)

# Make some requests...
client.get("/endpoint1")
client.get("/endpoint2")

print(f"Total spent: {client.total_paid_sats} sats")
```

### Quiet Mode

Disable console output:

```python
client = SatGateClient(wallet, verbose=False)
```

## LangChain Integration

```python
from satgate.langchain_integrations import SatGateTool
from langchain.agents import initialize_agent

# Give your agent a wallet
tools = [SatGateTool(wallet=my_wallet)]
agent = initialize_agent(tools, llm, agent="openai-functions")

# Let it roam the paid API economy
agent.run("Fetch the premium market report from AlphaVantage")
```

## Error Handling

```python
from satgate import PaymentError, L402ParseError

try:
    response = client.get("/premium")
except PaymentError as e:
    print(f"Payment failed: {e}")
except L402ParseError as e:
    print(f"Invalid L402 response: {e}")
```

## API Reference

### SatGateClient / SatGateSession

```python
SatGateClient(
    wallet: LightningWallet,      # Required: wallet for payments
    cache_tokens: bool = True,    # Cache L402 tokens
    cache_ttl: int = 3600,        # Cache TTL in seconds
    on_payment: Callable = None,  # Payment callback
    verbose: bool = True          # Print progress
)
```

### PaymentInfo

```python
@dataclass
class PaymentInfo:
    invoice: str           # BOLT11 invoice
    preimage: str          # Payment proof
    macaroon: str          # L402 macaroon
    amount_sats: int       # Amount paid
    endpoint: str          # URL accessed
    timestamp: float       # Unix timestamp
```

## License

MIT

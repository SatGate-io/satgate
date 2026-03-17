# Python Agent SDK

The `SatGateAgentClient` is designed for AI agents to interact with SatGate-protected APIs. It automatically handles:

- 402 Payment Required challenges (L402 and Fiat402)
- Token minting via Mint (Trust Broker)
- Macaroon management and delegation
- Retry logic for transient failures

## Installation

```bash
pip install satgate
```

## Quick Start

### With Pre-existing Token

```python
from satgate import SatGateAgentClient

client = SatGateAgentClient(
    gateway_url="https://api.example.com",
    token="<your-capability-token>"
)

# Make requests - 402 challenges handled automatically
response = client.get("/api/data")
print(response.json())
```

### With Kubernetes Identity (Badge-in)

```python
from satgate import SatGateAgentClient
from satgate.identity import KubernetesIdentity

# Automatically uses ServiceAccount token mounted at
# /var/run/secrets/kubernetes.io/serviceaccount/token
identity = KubernetesIdentity()

client = SatGateAgentClient(
    gateway_url="https://gateway.internal:8080",
    mint_url="https://gateway.internal:9090",  # Admin listener
    identity=identity
)

# Token obtained automatically from Mint
response = client.get("/api/internal/data")
```

### With AWS Identity

```python
from satgate import SatGateAgentClient
from satgate.identity import AWSIdentity

# Uses presigned GetCallerIdentity URL (recommended)
identity = AWSIdentity()

# For EKS with IRSA
identity = AWSIdentity(
    irsa_role_arn="arn:aws:iam::123456789012:role/MyAgentRole"
)

client = SatGateAgentClient(
    gateway_url="https://gateway.example.com",
    mint_url="https://gateway.example.com:9090",
    identity=identity
)
```

### With Lightning Payments (L402)

```python
from satgate import SatGateAgentClient

# Using Nostr Wallet Connect (NWC)
client = SatGateAgentClient(
    gateway_url="https://api.example.com",
    nwc_connection_string="nostr+walletconnect://..."
)

# Or using Alby
client = SatGateAgentClient(
    gateway_url="https://api.example.com",
    alby_token="<your-alby-access-token>"
)

# Payments happen automatically on 402
response = client.get("/api/premium/data")
```

---

## Configuration

### Environment Variables

```bash
# Gateway URL (required)
SATGATE_GATEWAY_URL=https://api.example.com

# Mint URL (if different from gateway)
SATGATE_MINT_URL=https://gateway.example.com:9090

# Pre-existing token
SATGATE_TOKEN=<macaroon>

# Lightning (for L402 mode)
SATGATE_NWC_CONNECTION_STRING=nostr+walletconnect://...
SATGATE_ALBY_TOKEN=<alby-access-token>

# Budget mode
SATGATE_BUDGET_MODE=fiat402
SATGATE_DEPARTMENT_ID=engineering
```

### Full Configuration

```python
from satgate import SatGateAgentClient

client = SatGateAgentClient(
    gateway_url="https://api.example.com",
    mint_url="https://gateway.example.com:9090",  # Optional separate Mint
    
    # Authentication (pick one)
    token="<capability-token>",           # Pre-existing token
    identity=KubernetesIdentity(),        # Badge-in identity
    
    # Payment handling
    nwc_connection_string="...",          # For L402 mode
    budget_mode="fiat402",                # For Fiat402 mode
    department_id="engineering",          # Cost center for chargebacks
    
    # Behavior
    auto_retry=True,                      # Retry on 402 (default: True)
    max_retries=3,                        # Max retry attempts
    timeout=30,                           # Request timeout (seconds)
    
    # Token management
    auto_refresh=True,                    # Refresh tokens before expiry
    refresh_threshold=300,                # Refresh 5 min before expiry
)
```

---

## Making Requests

### Basic Requests

```python
# GET
response = client.get("/api/users")

# POST with JSON
response = client.post("/api/users", json={"name": "Agent"})

# PUT
response = client.put("/api/users/1", json={"name": "Updated"})

# DELETE
response = client.delete("/api/users/1")

# Generic request
response = client.request("PATCH", "/api/users/1", json={"status": "active"})
```

### With Headers

```python
response = client.get("/api/data", headers={
    "X-Custom-Header": "value",
    "Accept": "application/json"
})
```

### Streaming Responses

```python
# For large responses or real-time data
with client.get("/api/stream", stream=True) as response:
    for chunk in response.iter_content(chunk_size=1024):
        process(chunk)
```

---

## Handling 402 Responses

The client automatically handles 402 Payment Required challenges:

### L402 (Lightning)

```python
client = SatGateAgentClient(
    gateway_url="https://api.example.com",
    nwc_connection_string="nostr+walletconnect://..."
)

# On 402:
# 1. Client parses WWW-Authenticate header
# 2. Pays Lightning invoice via NWC
# 3. Retries request with L402 token
response = client.get("/api/premium")  # Automatic!
```

### Fiat402 (Internal Budget)

```python
client = SatGateAgentClient(
    gateway_url="https://api.example.com",
    budget_mode="fiat402",
    department_id="engineering"
)

# On 402:
# 1. Client parses Fiat402 invoice
# 2. Acknowledges cost (budget deduction)
# 3. Retries request with receipt
response = client.get("/api/expensive")  # Automatic!
```

### Manual 402 Handling

```python
from satgate import SatGateAgentClient
from satgate.exceptions import PaymentRequiredError

client = SatGateAgentClient(
    gateway_url="https://api.example.com",
    auto_retry=False  # Disable auto-retry
)

try:
    response = client.get("/api/premium")
except PaymentRequiredError as e:
    print(f"Payment required: {e.invoice}")
    print(f"Amount: {e.amount} {e.unit}")
    
    # Handle payment manually
    receipt = pay_invoice_somehow(e.invoice)
    
    # Retry with receipt
    response = client.get("/api/premium", headers={
        "Authorization": f"L402 {e.macaroon}:{receipt}"
    })
```

---

## Token Delegation

Create attenuated child tokens for sub-agents:

```python
# Parent agent
parent_client = SatGateAgentClient(
    gateway_url="https://api.example.com",
    token="<parent-token>"
)

# Delegate to sub-agent with restrictions
child_token = parent_client.delegate(
    caveats=[
        "scope = api:read",           # Read-only
        "expires = 3600",              # 1 hour
        "max_requests = 100",          # Rate limit
        "ip = 10.0.0.0/8"             # Network restriction
    ]
)

# Sub-agent uses delegated token
sub_agent = SatGateAgentClient(
    gateway_url="https://api.example.com",
    token=child_token
)
```

---

## LangChain Integration

```python
from langchain.agents import AgentExecutor
from satgate.langchain import SatGateTool

# Create a budget-aware tool
tool = SatGateTool(
    name="api_call",
    description="Call the protected API",
    gateway_url="https://api.example.com",
    nwc_connection_string="nostr+walletconnect://..."
)

# Use in agent
agent = AgentExecutor(
    agent=your_agent,
    tools=[tool, ...],
    verbose=True
)

# Agent automatically handles payments
result = agent.run("Fetch the premium data from the API")
```

---

## Async Support

```python
from satgate import AsyncSatGateAgentClient
import asyncio

async def main():
    client = AsyncSatGateAgentClient(
        gateway_url="https://api.example.com",
        token="<token>"
    )
    
    # Async requests
    response = await client.get("/api/data")
    
    # Parallel requests
    results = await asyncio.gather(
        client.get("/api/users"),
        client.get("/api/orders"),
        client.get("/api/products")
    )
    
    await client.close()

asyncio.run(main())
```

### Async Context Manager

```python
async with AsyncSatGateAgentClient(...) as client:
    response = await client.get("/api/data")
```

---

## Error Handling

```python
from satgate.exceptions import (
    SatGateError,
    AuthenticationError,
    PaymentRequiredError,
    InsufficientBudgetError,
    RateLimitError,
    TokenExpiredError,
    MintError
)

try:
    response = client.get("/api/data")
except TokenExpiredError:
    # Token expired, client will auto-refresh if configured
    pass
except PaymentRequiredError as e:
    print(f"Need to pay {e.amount} {e.unit}")
except InsufficientBudgetError as e:
    print(f"Budget exhausted: {e.remaining} remaining")
except RateLimitError as e:
    print(f"Rate limited, retry after {e.retry_after}s")
except MintError as e:
    print(f"Failed to obtain token from Mint: {e}")
except SatGateError as e:
    print(f"Gateway error: {e}")
```

---

## Best Practices

### 1. Use Identity Providers in Kubernetes/AWS

```python
# Don't hardcode tokens in K8s workloads
# BAD:
client = SatGateAgentClient(token="hardcoded-token")

# GOOD:
client = SatGateAgentClient(identity=KubernetesIdentity())
```

### 2. Set Budget Limits for Sub-Agents

```python
# Always delegate with restrictions
child_token = parent.delegate(caveats=[
    "budget_limit = 10.00 USD",
    "expires = 3600"
])
```

### 3. Handle Payment Failures Gracefully

```python
from satgate.exceptions import PaymentFailedError

try:
    response = client.get("/api/premium")
except PaymentFailedError as e:
    # Fall back to free tier or cached data
    response = client.get("/api/basic")
```

### 4. Use Environment Variables for Configuration

```python
# Let configuration come from environment
client = SatGateAgentClient.from_env()
```

---

## Debugging

```python
import logging

# Enable debug logging
logging.getLogger("satgate").setLevel(logging.DEBUG)

# See all HTTP traffic
logging.getLogger("satgate.http").setLevel(logging.DEBUG)

# See payment handling
logging.getLogger("satgate.payments").setLevel(logging.DEBUG)
```

---

## Complete Example: AI Research Agent

```python
from satgate import SatGateAgentClient
from satgate.identity import KubernetesIdentity
import os

def main():
    # Initialize client with K8s identity
    client = SatGateAgentClient(
        gateway_url=os.environ["SATGATE_GATEWAY_URL"],
        mint_url=os.environ.get("SATGATE_MINT_URL"),
        identity=KubernetesIdentity(),
        auto_retry=True
    )
    
    # Fetch data from protected API
    try:
        response = client.get("/api/research/papers", params={
            "topic": "machine learning",
            "limit": 100
        })
        papers = response.json()
        
        for paper in papers["items"]:
            # Each request may incur cost, but handled automatically
            details = client.get(f"/api/research/papers/{paper['id']}")
            process_paper(details.json())
            
    except Exception as e:
        print(f"Error: {e}")
    finally:
        client.close()

if __name__ == "__main__":
    main()
```

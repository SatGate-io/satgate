# Python SDK

Official Python client for SatGate Gateway.

## Installation

```bash
pip install satgate
```

## Build agents with issue/pay/verify

For application developers, SatGate's Stripe-like primitive is three calls:

1. **Issue** a scoped capability for one task.
2. **Pay** an upstream call with a caller-supplied max budget.
3. **Verify** the returned receipt and Evidence Pack metadata.

```python
import os
from satgate import SatGate

satgate = SatGate(api_key=os.getenv("SATGATE_API_KEY"))

capability = satgate.issue(
    task="summarize vendor invoice",
    agent="invoice-agent",
    allow=["POST /v1/invoices/*"],
    budget_usd=0.25,
    expires_in="10m",
)

receipt = satgate.pay(
    upstream="https://api.vendor.test/v1/invoices/42",
    capability=capability,
    max_usd=0.10,
)

verified = satgate.verify(receipt)
print(verified.decision, getattr(verified, "evidence_pack_id", None))
```

The public package installs today. The `issue/pay/verify` API namespace is in private beta; non-beta credentials raise `SatGateAuthError` with a docs CTA instead of returning mocked success.

---

## Compatibility: OSS Gateway clients

| Client | For | Auth |
|--------|-----|------|
| `SatGateClient` | **Operators** — lower-level capability token and governance endpoints | `X-Admin-Token` header |
| `SatGateAgentClient` | **AI Agents** — compatibility client for existing gateway token and paid-rail flows | Auto-mints tokens, handles L402 |

Use these when targeting the self-hosted OSS gateway's existing token endpoints. New Cloud examples should lead with `issue/pay/verify`.

---

## Admin Client (`SatGateClient`)

For operators managing compatibility token endpoints and governance.

```python
from satgate import SatGateClient

client = SatGateClient(
    base_url="http://localhost:8080",
    admin_token="your-admin-token",
    timeout=30  # optional, default 30s
)
```

### Mint a Token

```python
token = client.tokens.mint(
    scope="api:*",      # scope string (default: "api:*")
    duration="1h"       # Go duration string: "30m", "1h", "24h"
)

print(f"Token: {token.token}")
print(f"Signature: {token.signature}")
print(f"Expires: {token.expires_at}")
```

### Validate a Token

```python
result = client.tokens.validate("token-string-here")
# Returns dict with: valid, identifier, caveats
```

### Delegate a Token

Create a child token with reduced permissions — no server roundtrip needed for verification:

```python
child = client.tokens.delegate(
    parent_token="parent-token-string",
    caveats=["scope = api:read", "expires = 1800"]
)
print(f"Child token: {child.token}")
```

### Governance

```python
# Ban a token
client.governance.ban(
    signature="hex-token-signature",
    reason="Compromised"
)

# Get token lineage graph (nodes, edges, stats)
graph = client.governance.get_graph()

# Reset all governance data
client.governance.reset()
```

### Health Check

```python
is_healthy = client.health()  # Returns bool

# Ping with a specific token
result = client.ping("capability-token")
```

---

## Agent Client (`SatGateAgentClient`)

For AI agents making API calls. Handles token minting, caching, refresh, and L402 payments automatically.

```python
from satgate import SatGateAgentClient

# With admin token (auto-mints capability tokens)
client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    scope="api:*",      # default scope for minted tokens
    duration="1h",      # default token duration
)

# Or with a pre-existing token
client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    token="your-capability-token",
)

# Or via environment variables
# SATGATE_ADMIN_TOKEN or SATGATE_TOKEN
client = SatGateAgentClient(gateway_url="http://localhost:8080")
```

### Making Requests

```python
# Simple HTTP methods
response = client.get("/api/data")
response = client.post("/api/data", json={"key": "value"})
response = client.put("/api/data/1", json={"key": "updated"})
response = client.delete("/api/data/1")

# Response is an AgentResponse
data = response.json()
text = response.text()
print(f"Status: {response.status_code}")
print(f"Cost: {response.cost}")
print(f"Budget remaining: {response.budget_remaining}")
```

### L402 Payments (Lightning)

Add a wallet to automatically pay L402 challenges:

```python
from satgate import SatGateAgentClient, LNDWallet, AlbyWallet

# With LND
client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    wallet=LNDWallet(
        host="localhost:10009",
        macaroon_path="~/.lnd/admin.macaroon",
        cert_path="~/.lnd/tls.cert",
    ),
)

# With Alby
client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    wallet=AlbyWallet(
        access_token="your-alby-token",
        # Or: nwc_url="nostr+walletconnect://..."
    ),
)

# L402 endpoints are handled automatically
response = client.get("/premium/data")  # includes receipt_id / evidence_pack_id when enabled  # Pays Lightning invoice if needed
```

### Token Delegation

Create restricted tokens for worker agents:

```python
# Delegate from the current token
child_token = client.delegate(
    caveats=["scope = api:read", "expires = 3600"]
)

# Give the child token to a worker agent
worker = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    token=child_token,
)
```

### Budget Tracking

```python
client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    admin_token="...",
    budget_limit=100.0,
    on_budget_alert=lambda used, limit: print(f"Budget alert: ${used}/{limit}"),
)

print(f"Total cost: {client.total_cost}")
print(f"Remaining: {client.budget_remaining}")
print(f"Current token: {client.current_token}")
```

---

## Delegation Helpers

Rich helpers for common delegation patterns:

```python
from satgate import delegate, Caveats, DelegationPatterns

# Fluent builder
token = (delegate(parent_token)
    .with_scope("api:read")
    .with_expiry(seconds=3600)
    .for_team("engineering")
    .with_budget(50.0, "USD")
    .with_rate_limit(100, period_seconds=60)
    .delegate(client))

# Common patterns
read_only = DelegationPatterns.read_only(parent_token).delegate(client)
temp_token = DelegationPatterns.temporary(parent_token, hours=24).delegate(client)
team_token = DelegationPatterns.team_budget(parent_token, "eng", 500.0).delegate(client)
api_client = DelegationPatterns.api_client(parent_token, "client-1", 100).delegate(client)
webhook = DelegationPatterns.webhook(parent_token, "/callback", expiry_minutes=30).delegate(client)
agent_swarm = DelegationPatterns.agent_swarm(parent_token, "swarm-1", budget=100.0).delegate(client)

# Individual caveat builders
Caveats.scope("api:read")
Caveats.expires(seconds=3600)
Caveats.routes(["/api/v1/*", "/health"])
Caveats.rate_limit(100, period=60)
Caveats.budget(50.0, "USD")
Caveats.source_ip(["10.0.0.0/8"])
Caveats.methods(["GET", "POST"])
Caveats.team("engineering")
```

---

## LangChain Integration

Use SatGate-protected APIs as LangChain tools:

```python
from satgate.langchain import SatGateTool, SatGateToolkit, create_satgate_tool
from langchain.agents import initialize_agent, AgentType
from langchain.chat_models import ChatOpenAI

# Single tool
tool = SatGateTool(
    name="api_query",
    description="Query the protected API endpoint",
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    endpoint="/api/data/query",
    cost_per_call=0.01,
    budget_limit=10.0,
)

# Or use a toolkit for multiple tools sharing config
toolkit = SatGateToolkit(
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    budget_limit=100.0,
)

data_tool = toolkit.create_tool(
    name="data_query",
    description="Query the data API",
    endpoint="/api/data/query",
)

analytics_tool = toolkit.create_tool(
    name="analytics",
    description="Run analytics queries",
    endpoint="/api/analytics",
)

# Use with a LangChain agent
llm = ChatOpenAI(temperature=0)
agent = initialize_agent(
    tools=toolkit.get_tools(),
    llm=llm,
    agent=AgentType.OPENAI_FUNCTIONS,
    verbose=True,
)

result = agent.run("Get the latest data and run analytics")
```

Tools automatically return structured JSON with `success`, `data`, `cost`, and `budget_remaining` — giving the LLM economic awareness.

---

## Error Handling

```python
from satgate import (
    SatGateError,
    AuthenticationError,
    PaymentRequiredError,
    PaymentFailedError,
    BudgetExceededError,
    TokenExpiredError,
    DelegationError,
    NotFoundError,
)

try:
    response = client.get("/api/data")
except PaymentRequiredError as e:
    print(f"Payment needed: {e.amount} {e.unit}")
    print(f"Invoice: {e.invoice}")
except BudgetExceededError as e:
    print(f"Over budget: used ${e.used}, limit ${e.limit}")
except AuthenticationError:
    print("Invalid credentials")
except SatGateError as e:
    print(f"API error: {e}")
```

## Data Models

```python
from satgate import Token, TokenInfo, BanRecord, Stats, GraphData

# Token — returned from mint/delegate
token.token       # str: the bearer token string
token.signature   # str: hex signature
token.scope       # Optional[str]
token.expires_at  # Optional[datetime]
token.caveats     # Optional[List[str]]

# GraphData — from governance.get_graph()
graph = GraphData.from_dict(data)
graph.nodes  # List[Dict] — token nodes
graph.edges  # List[Dict] — delegation edges
graph.stats  # Stats — active/banned/blocked counts
```

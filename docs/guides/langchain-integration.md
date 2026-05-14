# LangChain Integration

## SatGate voice

LangChain handles chains and tool orchestration. SatGate handles issue/pay/verify at the authority boundary: issue a scoped capability, enforce a max budget before the tool call, then verify the receipt.

Related: [Raw HTTP issue/pay/verify](raw-http.md), [Capability schema](../reference/capability-schema.md), [Receipt schema](../reference/receipt-schema.md).

This guide shows how to use the SatGate Python SDK with LangChain to build AI agents that interact with SatGate-protected APIs.

## Prerequisites

- A running SatGate OSS gateway (see [Quick Start](../../README.md))
- Python 3.9+
- An admin token for your gateway

## Installation

```bash
pip install satgate langchain langchain-openai
```

## 1. Single Tool — SatGateTool

The simplest way to give a LangChain agent access to a SatGate-protected endpoint:

```python
from langchain.agents import initialize_agent, AgentType
from langchain_openai import ChatOpenAI
from satgate.langchain import SatGateTool

# Create a tool that hits a protected API endpoint
data_tool = SatGateTool(
    name="market_data",
    description="Fetch real-time market data from the protected API. Returns JSON with price, volume, and trend data.",
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",  # or set SATGATE_ADMIN_TOKEN env var
    endpoint="/api/data/market",
    method="POST",
)

# Build the agent
llm = ChatOpenAI(model="gpt-4", temperature=0)
agent = initialize_agent(
    tools=[data_tool],
    llm=llm,
    agent=AgentType.OPENAI_FUNCTIONS,
    verbose=True,
)

# Run it
result = agent.run("What's the current Bitcoin price?")
print(result)
```

Under the hood, current OSS `SatGateTool` compatibility mode:
1. Mints a capability token via `POST /api/capability/mint` when an admin token is provided.
2. Caches the token for subsequent requests.
3. Sends the request to the endpoint with a `Bearer` token.
4. Enforces gateway policy before the upstream tool sees the call.
5. Returns the API response or structured denial JSON that the LLM can reason about.

For Cloud/private-beta application code, prefer the higher-level primitive in [Raw HTTP issue/pay/verify](raw-http.md): issue scoped authority, invoke through `pay` with a max budget, then verify the returned receipt and Evidence Pack handle.

### Using a Pre-Existing Token

If you already have a capability token (e.g., from delegation), pass it directly:

```python
tool = SatGateTool(
    name="market_data",
    description="Fetch market data",
    gateway_url="http://localhost:8080",
    token="your-capability-token",  # skip minting, use this token directly
    endpoint="/api/data/market",
)
```

## 2. Multiple Tools — SatGateToolkit

When your agent needs access to multiple endpoints, use `SatGateToolkit` to share a single token and budget across all tools:

```python
from satgate.langchain import SatGateToolkit

# Create a toolkit — all tools share the same gateway connection
toolkit = SatGateToolkit(
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    budget_limit=10.00,  # shared budget across all tools
)

# Register endpoints as tools
market_tool = toolkit.create_tool(
    name="market_data",
    description="Query real-time market prices and volume data",
    endpoint="/api/data/market",
)

analytics_tool = toolkit.create_tool(
    name="analytics",
    description="Run analytics queries on historical data. Accepts SQL-like queries.",
    endpoint="/api/analytics/query",
)

alerts_tool = toolkit.create_tool(
    name="alerts",
    description="Check and manage price alerts",
    endpoint="/api/alerts",
    method="GET",
)

# Get all tools for the agent
tools = toolkit.get_tools()

# Build agent
llm = ChatOpenAI(model="gpt-4", temperature=0)
agent = initialize_agent(
    tools=tools,
    llm=llm,
    agent=AgentType.OPENAI_FUNCTIONS,
    verbose=True,
)

result = agent.run("Show me the top 3 movers today and set an alert if BTC drops below $100k")
print(result)

# Check shared budget usage
print(f"Total cost: ${toolkit.total_cost:.2f}")
print(f"Budget remaining: ${toolkit.budget_remaining:.2f}")
```

## 3. Budget-Aware Agent

The agent can see budget information in tool responses, so it can reason about costs:

```python
from satgate.langchain import SatGateTool

def on_budget_alert(remaining: float, limit: float):
    print(f"⚠️  Budget alert: ${remaining:.2f} remaining of ${limit:.2f}")

expensive_tool = SatGateTool(
    name="premium_analysis",
    description="Run premium AI analysis on market data. Costs ~$0.50 per query. Use sparingly.",
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    endpoint="/api/premium/analyze",
    cost_per_call=0.50,
    budget_limit=5.00,
    on_budget_alert=on_budget_alert,
)

# The tool description automatically includes cost info:
# "Run premium AI analysis on market data. Costs ~$0.50 per query. Use sparingly.
#  (Estimated cost: $0.50 per call) (Budget limit: $5.00)"
#
# When budget is exceeded, the tool returns a structured error:
# {"success": false, "error": "budget_exceeded", "message": "...", "action_required": "..."}
# The LLM sees this and can decide to stop or ask for more budget.
```

## 4. Delegation for Worker Agents

A common pattern is a **supervisor agent** that delegates scoped tokens to **worker agents**. This gives each worker the minimum permissions it needs:

```python
from satgate import SatGateAgentClient, Caveats
from satgate.langchain import SatGateTool

# Supervisor: has broad access
supervisor_client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    scope="api:*",
)

# Create a restricted token for the research worker
research_token = supervisor_client.delegate(
    caveats=[
        Caveats.scope("api:read"),               # read-only
        Caveats.expires(seconds=1800),            # 30 minutes
        Caveats.routes(["/api/data/*"]),           # only data endpoints
    ]
)

# Create a restricted token for the trading worker
trading_token = supervisor_client.delegate(
    caveats=[
        Caveats.scope("api:trade"),               # trading scope
        Caveats.expires(seconds=300),             # 5 minutes
        Caveats.routes(["/api/trade/*"]),          # only trade endpoints
    ]
)

# Worker tools use delegated tokens (no admin token needed)
research_tool = SatGateTool(
    name="research",
    description="Look up market research data (read-only)",
    gateway_url="http://localhost:8080",
    token=research_token,  # delegated, scoped token
    endpoint="/api/data/research",
)

trading_tool = SatGateTool(
    name="execute_trade",
    description="Execute a trade order",
    gateway_url="http://localhost:8080",
    token=trading_token,  # delegated, scoped token
    endpoint="/api/trade/execute",
)
```

### Why Delegation Matters

Delegation uses macaroon attenuation — the child token is cryptographically derived from the parent, so:

- **No extra network calls**: Delegation happens locally (the gateway verifies the chain)
- **Least privilege**: Each worker gets only what it needs
- **Revocation**: Ban the parent to revoke all children instantly
- **Auditability**: The governance graph tracks the full delegation tree

## 5. Optional paid-route settlement context

If your gateway has a paid route, L402/Lightning can be one settlement adapter underneath the same authority boundary. Keep this secondary: SatGate should first decide whether the tool call is authorized, bounded, and receipted.

```python
from satgate.agent_client import LNDWallet
from satgate.langchain import SatGateTool

# Configure a Lightning wallet for optional L402 settlement
wallet = LNDWallet(
    host="localhost:8080",
    macaroon_path="~/.lnd/data/chain/bitcoin/mainnet/admin.macaroon",
    cert_path="~/.lnd/tls.cert",
)

# Tool with optional paid-route settlement
paid_tool = SatGateTool(
    name="premium_api",
    description="Access premium API data through a governed paid route",
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    endpoint="/api/premium/data",
    wallet=wallet,  # optional L402 adapter after policy allows the call
)
```

## 6. REST API Tool

For agents that need flexible, arbitrary API access:

```python
from satgate.langchain import SatGateRESTTool

rest_tool = SatGateRESTTool(
    name="gateway_api",
    description="Make HTTP requests to the SatGate-protected API. "
                "Input should be JSON with 'path', 'method', and optional 'body'.",
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    base_path="/api",
)

# The agent can now make arbitrary requests:
# {"path": "/api/data/users", "method": "GET"}
# {"path": "/api/data/reports", "method": "POST", "body": {"type": "weekly"}}
```

## 7. Environment Variables

The SDK supports these environment variables so you don't need to hardcode secrets:

| Variable | Description |
|----------|-------------|
| `SATGATE_ADMIN_TOKEN` | Admin token for minting capability tokens |
| `SATGATE_TOKEN` | Pre-existing capability token (alternative to admin token) |

```python
import os
os.environ["SATGATE_ADMIN_TOKEN"] = "your-admin-token"

# No need to pass admin_token explicitly
tool = SatGateTool(
    name="data",
    description="Query data",
    gateway_url="http://localhost:8080",
    endpoint="/api/data",
)
```

## Error Handling

Tools return structured JSON that the LLM can understand:

```json
// Success
{
  "success": true,
  "data": { ... },
  "cost": 0.0,
  "budget_remaining": 9.50
}

// Budget exceeded
{
  "success": false,
  "error": "budget_exceeded",
  "message": "I have exceeded my budget limit. Used: $10.00, Limit: $10.00",
  "action_required": "Please approve additional budget or use a different approach."
}

// Denied by policy or budget
{
  "success": false,
  "error": "satgate_denied",
  "decision_reason": "budget_exhausted",
  "receipt_id": "rcpt_...",
  "evidence_pack_id": "evid_...",
  "action_required": "Request a larger budget or choose a cheaper tool."
}

// Optional paid route with no configured settlement adapter
{
  "success": false,
  "error": "settlement_required",
  "message": "This route requires settlement after SatGate policy approval.",
  "action_required": "Configure an approved payment adapter or use a non-paid route."
}
```

The LLM sees structured errors and can reason about them, but the verifier should trust the signed receipt and Evidence Pack handle, not the model's interpretation of the error text.

## Full Example: Research Agent

Putting it all together:

```python
import os
from langchain.agents import initialize_agent, AgentType
from langchain_openai import ChatOpenAI
from satgate.langchain import SatGateToolkit

os.environ["SATGATE_ADMIN_TOKEN"] = "your-admin-token"
os.environ["OPENAI_API_KEY"] = "your-openai-key"

GATEWAY_URL = "http://localhost:8080"

# Create toolkit with shared budget
toolkit = SatGateToolkit(
    gateway_url=GATEWAY_URL,
    budget_limit=25.00,
)

# Register tools
toolkit.create_tool(
    name="search_data",
    description="Search the data catalog. Input: search query string.",
    endpoint="/api/data/search",
)

toolkit.create_tool(
    name="get_details",
    description="Get detailed information about a specific data item. Input: item ID.",
    endpoint="/api/data/details",
    method="GET",
)

toolkit.create_tool(
    name="run_analysis",
    description="Run statistical analysis on a dataset. Input: analysis parameters as JSON.",
    endpoint="/api/analytics/run",
    cost_per_call=1.00,
)

# Build agent
llm = ChatOpenAI(model="gpt-4", temperature=0)
agent = initialize_agent(
    tools=toolkit.get_tools(),
    llm=llm,
    agent=AgentType.OPENAI_FUNCTIONS,
    verbose=True,
)

# Run
result = agent.run(
    "Find datasets about renewable energy, get details on the most recent one, "
    "and run a trend analysis on it."
)
print(result)
print(f"\nTotal cost: ${toolkit.total_cost:.2f}")
```

# Python Agent Client

The `SatGateAgentClient` is included in the main `satgate` Python package. It provides automatic token management, caching, refresh, and L402 payment handling for AI agents.

See the [Python SDK documentation](python.md#agent-client-satgateagentclient) for full usage.

## Quick Start

```python
from satgate import SatGateAgentClient

client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
)

# Compatibility path: OSS gateway capability tokens are managed automatically
response = client.get("/api/data")
print(response.json())
```

## Features

- Auto-mints capability tokens via `POST /api/capability/mint`
- Caches tokens and refreshes on expiry
- Handles 401 → auto-refresh → retry
- Handles 402 → Lightning payment → retry (with wallet)
- Budget tracking and alerts
- Token delegation for worker agents
- LND and Alby wallet support

## LangChain Integration

```python
from satgate.langchain import SatGateTool, SatGateToolkit

toolkit = SatGateToolkit(
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    budget_limit=100.0,
)

tool = toolkit.create_tool(
    name="data_query",
    description="Query the data API",
    endpoint="/api/data/query",
)
```

See the [LangChain Integration Guide](../guides/langchain-integration.md) for full details.

# SatGate Gateway Python SDK

Official Python client for the [SatGate](https://github.com/SatGate-io/satgate) OSS Gateway.

## Installation

```bash
pip install satgate
```

## Quick Start

### Admin Client

The `SatGateClient` provides direct access to gateway admin operations.

```python
from satgate import SatGateClient

client = SatGateClient(
    base_url="http://localhost:8080",
    admin_token="your-admin-token"
)

# Check gateway health
if client.health():
    print("Gateway is healthy!")

# Mint a new capability token
token = client.tokens.mint(scope="api:read", duration="1h")
print(f"Token: {token.token}")
print(f"Signature: {token.signature}")

# Validate a token
result = client.tokens.validate(token.token)
print(f"Valid: {result['valid']}")

# Delegate a token with additional restrictions
child = client.tokens.delegate(
    parent_token=token.token,
    caveats=["scope = api:read"]
)
print(f"Child token: {child.signature}")

# Ban a compromised token
client.governance.ban(token.signature, reason="Compromised")

# Get governance graph (token lineage, stats)
graph = client.governance.get_graph()
print(f"Active tokens: {graph['stats']['active']}")

# Reset governance data
client.governance.reset()
```

### Agent Client

The `SatGateAgentClient` is designed for AI agents — it automatically
mints tokens, caches them, and handles L402 payment challenges.

```python
from satgate import SatGateAgentClient

# With admin token (auto-mints capability tokens)
client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    scope="api:read",
    duration="1h"
)

# Or with a pre-existing capability token
client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    token="your-capability-token"
)

# Or via environment variables
# export SATGATE_ADMIN_TOKEN=your-admin-token
# export SATGATE_TOKEN=your-capability-token (alternative)
client = SatGateAgentClient(gateway_url="http://localhost:8080")

# Make requests — token handling is automatic
response = client.get("/api/data")
print(response.json())

# Ping the gateway to verify your token
result = client.ping()
print(result)  # {"status": "ok", "message": "Token validated successfully", ...}

# Delegate a child token for a worker agent
child_token = client.delegate(caveats=["scope = api:read"])
```

## Token Delegation

### Fluent Delegation Builder

```python
from satgate import SatGateClient, delegate, Caveats, DelegationPatterns

client = SatGateClient(base_url="http://localhost:8080", admin_token="admin-token")
root = client.tokens.mint(scope="api:*", duration="24h")

# Fluent builder pattern
team_token = (delegate(root.token)
    .with_scope('api:read')
    .with_expiry(seconds=24 * 3600)
    .for_team('engineering')
    .delegate(client))

# Pre-built patterns
read_only = DelegationPatterns.read_only(root.token).delegate(client)
temp_token = DelegationPatterns.temporary(root.token, hours=2).delegate(client)

# Agent swarm token
swarm_token = DelegationPatterns.agent_swarm(
    root.token,
    swarm_id='ai-agents',
    budget=500,
    requests_per_minute=5000,
).delegate(client)
```

### Caveat Builders

```python
from satgate import Caveats

caveats = [
    Caveats.scope('api:read'),
    Caveats.expires(seconds=3600),
    Caveats.routes(['/api/*', '/health']),
    Caveats.rate_limit(100, period=60),
    Caveats.methods(['GET', 'POST']),
    Caveats.team('engineering'),
]

child = client.tokens.delegate(parent_token=root.token, caveats=caveats)
```

## OSS Gateway Endpoints

This SDK targets the SatGate OSS gateway endpoints:

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/capability/mint` | X-Admin-Token | Mint a new capability token |
| `POST` | `/api/capability/validate` | — | Validate a token |
| `POST` | `/api/capability/delegate` | — | Delegate a token with caveats |
| `GET` | `/api/capability/ping` | Bearer token | Verify a token is valid |
| `GET` | `/api/capability/admin` | Bearer token | Admin-scope verification |
| `POST` | `/api/governance/ban` | X-Admin-Token | Ban a token |
| `GET` | `/api/governance/graph` | — | Get token lineage graph |
| `POST` | `/api/governance/reset` | X-Admin-Token | Reset governance data |
| `GET` | `/health` | — | Health check |

## Error Handling

```python
from satgate import (
    SatGateClient,
    AuthenticationError,
    NotFoundError,
    SatGateError,
    PaymentRequiredError,
)

try:
    client = SatGateClient(base_url="http://localhost:8080", admin_token="invalid")
    client.tokens.mint()
except AuthenticationError:
    print("Invalid admin token!")
except SatGateError as e:
    print(f"API error: {e}")
```

## LangChain Integration

See the [LangChain Integration Guide](../../docs/guides/langchain-integration.md) for detailed examples.

```python
from satgate.langchain import SatGateTool

tool = SatGateTool(
    name="data_query",
    description="Query the protected data API",
    gateway_url="http://localhost:8080",
    admin_token="your-admin-token",
    endpoint="/api/data/query",
)
```

## License

MIT License

# Python SDK

Official Python client for SatGate Gateway.

## Installation

```bash
pip install satgate
```

## Quick Start

```python
from satgate import SatGateClient

# Create client
client = SatGateClient(
    "https://api.example.com",
    admin_token="your-admin-token"
)

# Mint a token
token = client.tokens.mint(
    scope="api:read",
    expires_in=3600,  # 1 hour
    metadata={"user": "agent-1"}
)

print(f"Token: {token.token}")
print(f"Expires: {token.expires_at}")
```

## Client Configuration

```python
from satgate import SatGateClient

# Basic configuration
client = SatGateClient(
    "https://api.example.com",
    admin_token="your-admin-token"
)

# Full configuration
client = SatGateClient(
    "https://api.example.com",
    admin_token="your-admin-token",
    timeout=30,
    retries=3,
    verify_ssl=True
)

# With JWT authentication
client = SatGateClient(
    "https://api.example.com",
    jwt="your-jwt-token"
)
```

## Async Support

```python
from satgate import AsyncSatGateClient
import asyncio

async def main():
    client = AsyncSatGateClient(
        "https://api.example.com",
        admin_token="your-admin-token"
    )

    token = await client.tokens.mint(
        scope="api:read",
        expires_in=3600
    )
    print(f"Token: {token.token}")

asyncio.run(main())
```

## Token Management

### Mint Token

```python
token = client.tokens.mint(
    scope="api:read,api:write",
    expires_in=86400,  # 24 hours
    metadata={
        "user": "agent-1",
        "purpose": "data-pipeline"
    }
)
```

### List Tokens

```python
tokens = client.tokens.list(limit=100, offset=0)

for t in tokens.items:
    print(f"Token: {t.signature[:8]}, Scope: {t.scope}")
```

### Get Token Details

```python
token = client.tokens.get("token-signature")
```

### Revoke Token

```python
client.tokens.revoke("token-signature")
```

### Delegate Token

```python
delegated = client.tokens.delegate(
    "parent-signature",
    caveats=[
        {"type": "expires", "value": "1h"},
        {"type": "rate_limit", "value": "100/minute"},
        {"type": "scope", "value": "api:read"}
    ]
)
```

## Governance

### Ban Token

```python
client.governance.ban(
    signature="token-to-ban",
    reason="Compromised credentials"
)
```

### Unban Token

```python
client.governance.unban("token-signature")
```

### Get Ban List

```python
banned = client.governance.ban_list()

for sig in banned.signatures:
    print(f"Banned: {sig}")
```

### Get Token Lineage

```python
graph = client.governance.graph()

for node in graph.nodes:
    print(f"Token: {node.signature}, Parent: {node.parent}")
```

## Configuration

### Get Config

```python
config = client.config.get()

print(f"Routes: {len(config.routes)}")
```

### Update Config

```python
client.config.update({
    "routes": [
        {
            "name": "new-api",
            "path": "/v2/*",
            "upstream": "backend",
            "policy": {
                "kind": "observe",
                "scope": "api:v2"
            }
        }
    ]
})
```

### Validate Config

```python
result = client.config.validate(config_yaml)

if not result.valid:
    for error in result.errors:
        print(f"Error at line {error.line}: {error.message}")
```

## Statistics

### Get Gateway Stats

```python
stats = client.stats.get()

print(f"Total Requests: {stats.total_requests}")
print(f"Active Tokens: {stats.active_tokens}")
```

### Get Route Stats

```python
routes = client.stats.routes()

for name, stats in routes.items():
    print(f"{name}: {stats.requests} requests, p99: {stats.latency_p99}ms")
```

## Making Protected Requests

```python
# Using a capability token
response = client.request(
    "GET",
    "/api/users",
    token=token.token
)

users = response.json()
```

## WebSocket Telemetry

```python
from satgate import SatGateClient, EventType

client = SatGateClient("https://api.example.com", admin_token="...")

# Sync version
for event in client.telemetry.subscribe():
    if event.type == EventType.TOKEN_MINT:
        print(f"Token minted: {event.signature}")
    elif event.type == EventType.TOKEN_BAN:
        print(f"Token banned: {event.signature}")
    elif event.type == EventType.REQUEST:
        print(f"Request: {event.method} {event.path} ({event.status})")
```

### Async WebSocket

```python
async def watch_events():
    async for event in client.telemetry.subscribe_async():
        print(f"Event: {event.type}")
```

## Error Handling

```python
from satgate import (
    SatGateError,
    AuthenticationError,
    AuthorizationError,
    RateLimitError,
    NotFoundError,
    ValidationError
)

try:
    token = client.tokens.mint(scope="api:read")
except RateLimitError as e:
    print(f"Rate limited. Retry after {e.retry_after} seconds")
    time.sleep(e.retry_after)
except AuthenticationError:
    print("Invalid credentials")
except ValidationError as e:
    print(f"Validation error: {e.message}")
except SatGateError as e:
    print(f"API error: {e.code} - {e.message}")
```

## Context Manager

```python
from satgate import SatGateClient

with SatGateClient("https://api.example.com", admin_token="...") as client:
    token = client.tokens.mint(scope="api:read")
    # Client is automatically closed when exiting the context
```

## Testing

Use the mock client for testing:

```python
from satgate.testing import MockSatGateClient, MockToken

def test_my_service():
    mock_client = MockSatGateClient()
    mock_client.tokens.mint_returns(MockToken(
        token="mock-token",
        signature="mock-sig"
    ))

    service = MyService(mock_client)
    result = service.do_something()

    assert mock_client.tokens.mint_called
    assert result == expected
```

## Type Hints

The SDK is fully typed for IDE support:

```python
from satgate import SatGateClient, Token, MintRequest

def mint_token(client: SatGateClient) -> Token:
    request: MintRequest = {
        "scope": "api:read",
        "expires_in": 3600
    }
    return client.tokens.mint(**request)
```

## Logging

```python
import logging

# Enable debug logging
logging.getLogger("satgate").setLevel(logging.DEBUG)

# Custom handler
handler = logging.StreamHandler()
handler.setFormatter(logging.Formatter(
    "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
))
logging.getLogger("satgate").addHandler(handler)
```




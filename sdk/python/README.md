# SatGate Gateway Python SDK

Official Python client for the SatGate Enterprise Gateway Admin API.

## Economic Policies

SatGate uses an "Economic Firewall" model with Default Protection (Layer 0) and Economic Policies (Layer 1):

- **Default Protection**: All non-PUBLIC routes require cryptographic credentials (Macaroons)
- **Observe**: Meter and log traffic without blocking (FinOps visibility)
- **Control**: Enforce budgets with Fiat402
- **Charge**: Monetize with L402 (Lightning) or Fiat402 (Stripe)

## Installation

```bash
pip install satgate
```

## Quick Start

```python
from satgate import SatGateClient

# Initialize client
client = SatGateClient(
    base_url="http://localhost:9090",
    admin_token="your-admin-token"
)

# Check gateway health
if client.health():
    print("Gateway is healthy!")

# Mint a new token
token = client.tokens.mint(scope="api:read", expires_in=3600)
print(f"Token: {token.token}")
print(f"Signature: {token.signature}")

# List all tokens
tokens = client.tokens.list()
for t in tokens:
    print(f"{t.signature[:16]}... - {t.status} - {t.total_requests} requests")

# Ban a compromised token
client.governance.ban(token.signature, reason="Compromised credentials")

# Get gateway stats
stats = client.stats.get()
print(f"Total requests: {stats.total_requests}")
print(f"Active tokens: {stats.active_tokens}")
print(f"Banned tokens: {stats.banned_tokens}")
```

## Token Delegation

### Basic Delegation

```python
# Create a child token with reduced scope
child = client.tokens.delegate(
    parent_token=token.token,
    caveats=["scope = api:read:users"]
)
print(f"Child token: {child.signature}")
```

### Fluent Delegation Builder

```python
from satgate import delegate, Caveats, DelegationPatterns

# Fluent builder pattern
team_token = (delegate(root_token)
    .with_scope('api:read')
    .with_expiry(seconds=24 * 3600)  # 24 hours
    .with_budget(100, 'USD')
    .for_team('engineering')
    .delegate(client))

# Pre-built patterns
read_only_token = DelegationPatterns.read_only(root_token).delegate(client)

temp_token = DelegationPatterns.temporary(root_token, hours=2).delegate(client)

api_client_token = DelegationPatterns.api_client(
    root_token,
    client_id='my-app-client',
    requests_per_minute=1000
).delegate(client)

# Webhook callback token (single route, short expiry)
webhook_token = DelegationPatterns.webhook(
    root_token,
    callback_path='/api/webhooks/stripe',
    expiry_minutes=30
).delegate(client)

# CI/CD pipeline token
ci_token = DelegationPatterns.cicd(
    root_token,
    pipeline_id='deploy-prod-123',
    expiry_minutes=60
).delegate(client)

# Agent swarm token with budget
swarm_token = DelegationPatterns.agent_swarm(
    root_token,
    swarm_id='ai-agents-prod',
    budget=500,
    currency='USD',
    requests_per_minute=5000,
    max_agents=100
).delegate(client)
```

### Caveat Builders

```python
from satgate import Caveats

# Build caveats programmatically
caveats = [
    Caveats.scope('api:read:users,api:read:posts'),
    Caveats.expires(seconds=3600),  # 1 hour
    Caveats.routes(['/api/v1/*', '/health']),
    Caveats.rate_limit(100, period=60),  # 100 req/min
    Caveats.budget(50, 'USD'),
    Caveats.source_ip(['10.0.0.0/8']),
    Caveats.methods(['GET', 'POST']),
    Caveats.team('engineering'),
    Caveats.project('api-gateway'),
    Caveats.label('env', 'production'),
]

token = client.tokens.delegate(
    parent_token=root_token,
    caveats=caveats
)
```

## Configuration

```python
# Get current config
config = client.config.get()
print(f"Routes: {len(config['routes'])}")

# Validate a configuration
is_valid = client.config.validate({
    "version": 1,
    "routes": [...]
})
```

## Error Handling

```python
from satgate import SatGateClient, AuthenticationError, NotFoundError, SatGateError

try:
    client = SatGateClient(base_url="http://localhost:9090", admin_token="invalid")
    client.tokens.list()
except AuthenticationError:
    print("Invalid admin token!")
except NotFoundError:
    print("Resource not found!")
except SatGateError as e:
    print(f"API error: {e}")
```

## License

MIT License




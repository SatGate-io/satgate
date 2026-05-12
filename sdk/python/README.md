# satgate

Official Python SDK for SatGate: the Economic Firewall for AI agents.

## Install

```bash
pip install satgate
```

## Build agents with issue/pay/verify

The public package installs today. The `issue/pay/verify` API namespace is in private beta, so calls without beta access raise a structured error instead of returning mocked receipts.

```python
import os
from satgate import SatGate

satgate = SatGate(api_key=os.getenv("SATGATE_API_KEY"))

capability = satgate.issue(
    task="research market prices",
    agent="research-agent",
    allow=["mcp:web.search", "api:prices.read"],
    budget_usd=25,
    expires_in="1h",
)

receipt = satgate.pay(
    upstream="https://api.example.com/search",
    capability=capability,
    max_usd=4.20,
)

verified = satgate.verify(receipt)
print(verified.decision, verified.evidence_pack_id)
```

Without private-beta access:

```text
SatGateAuthError: This API namespace requires private beta access. Visit cloud.satgate.io/docs to request access.
```

Works with: MCP · OpenAI tools · Anthropic tools · LangChain · CrewAI · Raw HTTP

## Compatibility: lower-level OSS Gateway client

The package also includes lower-level OSS gateway clients:

```python
from satgate import SatGateClient, SatGateAgentClient

admin = SatGateClient(
    base_url="http://localhost:8080",
    admin_token="your-admin-token",
)

token = admin.tokens.mint(scope="api:read", duration="1h")
valid = admin.tokens.validate(token.token)

agent = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    token=token.token,
)

response = agent.get("/api/data")
print(valid["valid"], response.status_code)
```

## Docs

- Developer docs: https://cloud.satgate.io/docs
- Build page: https://satgate.io/build
- Repository: https://github.com/SatGate-io/satgate

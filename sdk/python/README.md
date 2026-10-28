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

## Lightning (LND) wallet and TLS

`LNDWallet` talks to LND's REST port over HTTPS and always verifies the
server's TLS certificate:

```python
from satgate import LNDWallet

# LND's own self-signed certificate: verify against that file.
wallet = LNDWallet(
    host="localhost:8080",
    macaroon_hex=open("/path/to/admin.macaroon", "rb").read().hex(),
    tls_cert_path="~/.lnd/tls.cert",  # cert_path= is still accepted
)
```

- With no `tls_cert_path`, the host must present a certificate the system
  trusts. A host whose certificate is not trusted raises `SatGateError`
  naming the host; the macaroon is not sent.
- With `tls_cert_path` (or the older `cert_path`), the host is verified
  against that file. A missing file is an error when the wallet is created.
- `insecure=True` is the only way to turn verification off. It logs a warning
  naming the host and cannot be combined with a certificate path. Use it only
  for local testing.
- Wallet requests never follow redirects, with or without `insecure=True`.
  The macaroon is sent only to the configured host. A 3xx answer raises
  `RedirectRefusedError` (a `SatGateError`) naming the status and the host.

## Redirects

No SDK request that carries a credential follows an HTTP redirect. That
covers every request that sends an `Authorization` header, `X-Admin-Token`,
an LND macaroon or a token in the JSON body: `SatGateClient` (all requests;
its session sends the admin token), `SatGateAgentClient` (minting, every
request with the capability token including the 401 refresh and the L402
retry, `validate_token`, `delegate`, `call` and the admin helpers), `SatGate`
(`issue`, `pay`, `verify`), `LNDWallet` and `AlbyWallet`.

Such a request is sent once, to the configured URL. A 3xx answer raises
`satgate.RedirectRefusedError` (a `SatGateError`) naming only the status and
the host that answered; the credential is not sent again. Point the client at
the final address (for example `https://` rather than an `http://` URL that
redirects to it). `SatGateClient.health()` returns `False` in that case.

## Docs

- Developer docs: https://cloud.satgate.io/docs
- Build page: https://satgate.io/build
- Repository: https://github.com/SatGate-io/satgate

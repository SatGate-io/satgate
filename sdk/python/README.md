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

No HTTP request the SDK sends follows a redirect: `SatGateClient` (all
requests; its session sends the admin token), `SatGateAgentClient` (minting,
every request with the capability token including the 401 refresh and the
L402 retry, `validate_token`, `delegate`, `call`, the admin helpers and
`fetch_jwks`), `SatGate` (`issue`, `pay`, `verify`), `LNDWallet` and
`AlbyWallet`.

Each request is sent once, to the configured URL, with
`allow_redirects=False` whatever the caller passes. A 3xx answer raises
`satgate.RedirectRefusedError` (a `SatGateError`) naming only the status and
the host and port of the request URL (no user information, path or query);
the request is not sent again. Point the client at the final address (for
example `https://` rather than an `http://` URL that redirects to it).
`SatGateClient.health()` returns `False` in that case.

Requests sent through a client's `requests.Session` carry what that session
holds (its headers, its authentication and the cookies the gateway set), to
the configured URL only. `SatGateAgentClient.fetch_jwks()` does not use the
session: it is sent on a new session with no cookies, no session headers, no
authentication (none from `~/.netrc` either) and no client certificate. Only
`agent.session.verify` is applied to it.

## Error messages

Every message the SDK builds or raises about a request is redacted for that
request: an error from `requests` (an invalid URL or header value, a refused
connection, a timeout, a body cut short) or a JSON parse error, raised as
the same exception; a status error quoting the response (`Request failed:
...`, `API error ...`, `Mint failed ...`, `Validation failed: ...`,
`Delegation failed: ...`, `jwks failed: ...`, the wallet errors, the admin
helpers); and an SDK error quoting another (`Request failed: ...`, `Ping
failed: ...`, `Failed to mint token: ...`, `Lightning payment failed: ...`).

In those messages, and in the messages of the exceptions they wrap:

- the request URL (with the query parameters of the call and the session
  merged) is named by its host and port; its path and query read
  `[path and query removed]`, the path alone (or a tail of it from a `/`)
  `[path removed]`, and the query string, each `name=value` pair, each
  query value and each query name `[redacted]`; user information is
  dropped;
- every credential the request carried reads `[redacted]` wherever it
  appears, including text quoted from the response body, also when
  URL-encoded, JSON-escaped or quoted: the values of request headers whose
  names contain `auth`, `token`, `key`, `secret`, `macaroon`, `cookie` or
  `password` (including those the session, its cookies or `~/.netrc` add),
  the token after the scheme, the parts of Basic and L402 credentials, the
  string values of JSON body fields whose names contain `token`,
  `macaroon`, `preimage`, `secret`, `password`, `passwd`, `key`,
  `credential`, `capability`, `auth`, `signature` or `receipt`, the URL's
  user name and password, and the query values.

A credential, query name or query value shorter than 3 characters is not
replaced on its own; it is removed only as part of the URL, the query
string or a `name=value` pair. Other text the server sent is quoted as
before.

Not redacted: exception types, status codes and chained causes are
unchanged, and so are attributes: `exc.request` and `exc.response` of a
`requests` exception (the prepared URL, headers and body), urllib3's
`MaxRetryError.url`, a `JSONDecodeError`'s `doc`,
`PaymentRequiredError.challenge`, and the response data the SDK returns
(`AgentResponse.content`, the `body` from `call()`). Do not log those as if
they were redacted.

## Docs

- Developer docs: https://cloud.satgate.io/docs
- Build page: https://satgate.io/build
- Repository: https://github.com/SatGate-io/satgate

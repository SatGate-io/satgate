# @satgate/sdk

Official Node.js SDK for SatGate: the Economic Firewall for AI agents.

## Install

```bash
npm install @satgate/sdk
```

## Build agents with issue/pay/verify

The public package installs today. The `issue/pay/verify` API namespace is in private beta, so calls without beta access raise a structured error instead of returning mocked receipts.

```ts
import { SatGate } from "@satgate/sdk";

const satgate = new SatGate({ apiKey: process.env.SATGATE_API_KEY });

const capability = await satgate.issue({
  task: "compare supplier prices",
  agent: "procurement-agent",
  allow: ["mcp:browser.search", "api:supplier.quote"],
  budgetUsd: 25,
  expiresIn: "1h",
});

const receipt = await satgate.pay({
  upstream: "https://api.example.com/search",
  capability,
  maxUsd: 4.20,
});

const verified = await satgate.verify(receipt);
console.log(verified.decision, verified.evidencePackId);
```

Without private-beta access:

```text
SatGateAuthError: This API namespace requires private beta access. Visit cloud.satgate.io/docs to request access.
```

Works with: MCP · OpenAI tools · Anthropic tools · LangChain · CrewAI · Raw HTTP

## Compatibility: lower-level OSS Gateway client

The package also includes the lower-level OSS gateway clients:

```ts
import { SatGateClient, SatGateAgentClient } from "@satgate/sdk";

const admin = new SatGateClient({
  url: "http://localhost:8080",
  token: "your-admin-token",
});

const token = await admin.tokens.mint({ scope: "api:read", duration: "1h" });
const valid = await admin.tokens.validate(token.token);

const agent = new SatGateAgentClient({
  gatewayUrl: "http://localhost:8080",
  token: token.token,
});

const response = await agent.get("/api/data");
console.log(valid.valid, response.status);
```

## Redirects

No HTTP request the SDK sends follows a redirect: `SatGateClient` (all
requests, including `health()` and `governance.getGraph()`),
`SatGateAgentClient` (minting, every request with the capability token
including the 401 refresh and the L402 retry, `validateToken` and
`delegate`), `SatGate` (`issue`, `pay`, `verify`), `LNDWallet` and
`AlbyWallet`.

Each request is sent once, with `redirect: 'manual'` (a `redirect` option
you pass is overridden), to the configured URL. A 3xx answer rejects with
`RedirectRefusedError` (a `SatGateError`, with `statusCode` and `host`)
naming only the status and the host and port of the request URL (no user
information, path or query, also when they are part of the `host` given to
`LNDWallet`); the request is not sent again. `SatGateClient.health()`
resolves `false` in that case.

The request URL is checked before anything is sent. A URL with a user name
or password (for example a gateway URL or an LND `host` with `user:pass@`),
or one that cannot be parsed, rejects with `RequestUrlRefusedError` (a
`SatGateError`, with `host`) naming only the host and port, and no request
is sent; put credentials in the SDK options, not in the URL (before, such a
URL failed with a `TypeError` from `fetch`).

The SDK keeps no session or cookie jar: a request carries no cookie or
credential beyond the headers and body the SDK (or your options) set. In a
browser, `fetch` still applies its default cookie rule
(`credentials: 'same-origin'`).

Compatibility: earlier releases followed these redirects. A gateway, LND,
Alby or SatGate API URL that answers with a redirect (for example `http://`
redirecting to `https://`) now fails; configure the final URL.

## Error messages

Every error message the SDK builds or rethrows about a request is redacted
for that request:

- an error from `fetch` (a network error, an invalid header value),
  rethrown as the same error object;
- an error reading or parsing the response body (`SyntaxError`,
  `TypeError`, node-fetch's `FetchError`), rethrown as the same error
  object;
- a status error quoting the response: `Request failed: <status> ...`,
  `Mint failed (...)`, `Delegation failed: ...`, `LND payment failed: ...`,
  `Alby payment failed: ...`, `Payment error: ...`, `Failed to get balance:
  ...`, and the response's `error` or `message` field used as the message of
  `SatGateError` or `ValidationError`;
- an SDK error quoting another: `Request failed: ...`, `Ping failed: ...`,
  `Failed to mint token: ...`, `Delegation failed: ...`, `Lightning payment
  failed: ...`.

In the message and stack of such an error, and of the errors in its `cause`
chain:

- the request URL is named by its host and port. Its path and query read
  `[path and query removed]`; the path alone, or a tail of it from a `/`,
  reads `[path removed]`; the query string, each `name=value` pair, each
  query value and each query name (as a whole word) read `[redacted]`;
- every credential the request carried reads `[redacted]` wherever it
  appears, including text quoted from the response body, also when
  URL-encoded or JSON-escaped: the values of request headers whose names
  contain `auth`, `token`, `key`, `secret`, `macaroon`, `cookie` or
  `password` (so `Authorization`, `X-Admin-Token` and
  `Grpc-Metadata-macaroon`), the token after the scheme, the parts of Basic
  and L402 credentials, the string values of JSON body fields whose names
  contain `token`, `macaroon`, `preimage`, `secret`, `password`, `passwd`,
  `key`, `credential`, `capability`, `auth`, `signature` or `receipt`, and
  the query values;
- the response text a JSON parse error quotes reads `(response text
  removed)`.

A credential, query name or query value shorter than 3 characters is not
replaced on its own (it would match ordinary text such as a status code);
it is removed only as part of the URL, the query string or a `name=value`
pair. Other text the server sent is quoted as before.

Not redacted: error types, `statusCode` and `cause` are unchanged, and only
`message` and `stack` are rewritten, not other properties of an error. The
response data the SDK returns (`AgentResponse.data`, the parsed body of a
successful call) and `PaymentRequiredError.challenge` are the server's
answer as received.

## Docs

- Developer docs: https://cloud.satgate.io/docs
- Build page: https://satgate.io/build
- Repository: https://github.com/SatGate-io/satgate

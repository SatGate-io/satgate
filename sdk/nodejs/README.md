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
URL failed with a `TypeError` from `fetch`). A network or
response-body error names no part of the request URL but its host and port.

The SDK keeps no session or cookie jar: a request carries no cookie or
credential beyond the headers and body the SDK (or your options) set. In a
browser, `fetch` still applies its default cookie rule
(`credentials: 'same-origin'`).

Compatibility: earlier releases followed these redirects. A gateway, LND,
Alby or SatGate API URL that answers with a redirect (for example `http://`
redirecting to `https://`) now fails; configure the final URL.

## Docs

- Developer docs: https://cloud.satgate.io/docs
- Build page: https://satgate.io/build
- Repository: https://github.com/SatGate-io/satgate

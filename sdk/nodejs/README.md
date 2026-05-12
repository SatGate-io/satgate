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

## Docs

- Developer docs: https://cloud.satgate.io/docs
- Build page: https://satgate.io/build
- Repository: https://github.com/SatGate-io/satgate

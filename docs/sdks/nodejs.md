# Node.js SDK

Official Node.js/TypeScript client for SatGate Gateway.

## Installation

```bash
npm install @satgate/sdk
```

## Build agents with issue/pay/verify

For application developers, SatGate's Stripe-like primitive is three calls:

1. **Issue** a scoped capability for one task.
2. **Pay** an upstream call with a caller-supplied max budget.
3. **Verify** the returned receipt and Evidence Pack metadata.

```typescript
import { SatGate } from '@satgate/sdk';

const satgate = new SatGate({ apiKey: process.env.SATGATE_API_KEY });

const capability = await satgate.issue({
  task: 'summarize vendor invoice',
  agent: 'invoice-agent',
  allow: ['POST /v1/invoices/*'],
  budgetUsd: 0.25,
  expiresIn: '10m',
});

const receipt = await satgate.pay({
  upstream: 'https://api.vendor.test/v1/invoices/42',
  capability,
  maxUsd: 0.10,
});

const verified = await satgate.verify(receipt);
console.log(verified.decision, verified.evidencePackId ?? verified.evidence_pack_id);
```

The public package installs today. The `issue/pay/verify` API namespace is in private beta; non-beta credentials raise `SatGateAuthError` with a docs CTA instead of returning mocked success.

---

## Compatibility: OSS Gateway clients

| Client | For | Auth |
|--------|-----|------|
| `SatGateClient` | **Operators** — lower-level capability token and governance endpoints | `X-Admin-Token` header |
| `SatGateAgentClient` | **AI Agents** — compatibility client for existing gateway token and paid-rail flows | Auto-mints tokens, handles L402 |

Use these when targeting the self-hosted OSS gateway's existing token endpoints. New Cloud examples should lead with `issue/pay/verify`.

---

## Admin Client (`SatGateClient`)

```typescript
import { SatGateClient } from '@satgate/sdk';

const client = new SatGateClient({
  url: 'http://localhost:8080',
  token: 'your-admin-token',
  timeout: 30000,  // optional, default 30s
});
```

### Mint a Token

```typescript
const token = await client.tokens.mint({
  scope: 'api:*',   // default: "api:*"
  duration: '1h',   // Go duration string, default: "1h"
});

console.log(`Token: ${token.token}`);
console.log(`Signature: ${token.signature}`);
console.log(`Expires: ${token.expiresAt}`);
```

### Validate a Token

```typescript
const result = await client.tokens.validate('token-string-here');
// { valid: boolean, identifier?: string, caveats?: string[] }
```

### Delegate a Token

```typescript
const child = await client.tokens.delegate({
  parentToken: 'parent-token-string',
  caveats: ['scope = api:read', 'expires = 1800'],
});
```

### Governance

```typescript
// Ban a token
await client.governance.ban('hex-signature', 'Compromised');

// Get token lineage graph
const graph = await client.governance.getGraph();
// { nodes: TokenInfo[], edges: [...], stats: GraphStats }

// Reset governance data
await client.governance.reset();
```

### Health & Ping

```typescript
const healthy = await client.health();  // boolean

const result = await client.ping('capability-token');
```

---

## Agent Client (`SatGateAgentClient`)

For AI agents. Handles token minting, caching, refresh, and L402 payments automatically.

```typescript
import { SatGateAgentClient, LNDWallet, AlbyWallet } from '@satgate/sdk';

// With admin token (auto-mints capability tokens)
const client = new SatGateAgentClient({
  gatewayUrl: 'http://localhost:8080',
  adminToken: 'your-admin-token',
});

// Or with pre-existing token
const client = new SatGateAgentClient({
  gatewayUrl: 'http://localhost:8080',
  token: 'your-capability-token',
});

// With Lightning wallet for L402
const client = new SatGateAgentClient({
  gatewayUrl: 'http://localhost:8080',
  adminToken: 'your-admin-token',
  wallet: new LNDWallet({
    host: 'localhost:10009',
    macaroonPath: '~/.lnd/admin.macaroon',
  }),
});
```

### Making Requests

```typescript
const response = await client.get('/api/data');
const response = await client.post('/api/data', { key: 'value' });

console.log(response.data);
console.log(`Cost: ${response.cost}`);
console.log(`Budget remaining: ${response.budgetRemaining}`);
```

---

## Delegation Helpers

```typescript
import { delegate, Caveats, DelegationPatterns } from '@satgate/sdk';

// Fluent builder
const token = await delegate(parentToken)
  .withScope('api:read')
  .withExpiry(3600)
  .forTeam('engineering')
  .withBudget(50.0, 'USD')
  .delegate(client);

// Common patterns
const readOnly = await DelegationPatterns.readOnly(parentToken).delegate(client);
const temp = await DelegationPatterns.temporary(parentToken, 24).delegate(client);
const team = await DelegationPatterns.teamBudget(parentToken, 'eng', 500).delegate(client);
```

---

## Error Handling

```typescript
import {
  SatGateError,
  AuthenticationError,
  PaymentRequiredError,
  BudgetExceededError,
} from '@satgate/sdk';

try {
  const response = await client.get('/api/data');
} catch (err) {
  if (err instanceof PaymentRequiredError) {
    console.log(`Payment needed: ${err.amount} ${err.unit}`);
  } else if (err instanceof BudgetExceededError) {
    console.log(`Over budget: ${err.used}/${err.limit}`);
  } else if (err instanceof AuthenticationError) {
    console.log('Invalid credentials');
  }
}
```

## Types

All types are fully exported for TypeScript:

```typescript
import type {
  Token,
  TokenInfo,
  GraphData,
  GraphStats,
  DelegateRequest,
  PolicyKind,
  RouteConfig,
} from '@satgate/sdk';
```

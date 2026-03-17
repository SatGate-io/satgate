# Node.js SDK

Official Node.js/TypeScript client for SatGate Gateway.

## Installation

```bash
npm install @satgate/sdk
```

## Two Clients

| Client | For | Auth |
|--------|-----|------|
| `SatGateClient` | **Operators** — mint tokens, ban tokens, manage governance | `X-Admin-Token` header |
| `SatGateAgentClient` | **AI Agents** — make API calls with automatic token & payment handling | Auto-mints tokens, handles L402 |

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

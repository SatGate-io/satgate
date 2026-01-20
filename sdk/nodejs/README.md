# @satgate/client

Official Node.js SDK for the SatGate Enterprise Gateway Admin API.

## Economic Policies

SatGate uses an "Economic Firewall" model with Default Protection (Layer 0) and Economic Policies (Layer 1):

- **Default Protection**: All non-PUBLIC routes require cryptographic credentials (Macaroons)
- **Observe**: Meter and log traffic without blocking (FinOps visibility)
- **Control**: Enforce budgets with Fiat402
- **Charge**: Monetize with L402 (Lightning) or Fiat402 (Stripe)

## Installation

```bash
npm install @satgate/client
```

## Quick Start

```typescript
import { SatGateClient } from '@satgate/client';

// Initialize client
const client = new SatGateClient({
  url: 'http://localhost:9090',
  token: 'your-admin-token',
});

// Check gateway health
const healthy = await client.health();
console.log('Gateway healthy:', healthy);

// Mint a new token
const token = await client.tokens.mint({
  scope: 'api:read',
  expiresIn: 3600,
});
console.log('Token:', token.token);
console.log('Signature:', token.signature);

// List all tokens
const tokens = await client.tokens.list();
for (const t of tokens) {
  console.log(`${t.signature.slice(0, 16)}... - ${t.status} - ${t.totalRequests} requests`);
}

// Ban a compromised token
await client.governance.ban(token.signature, 'Compromised credentials');

// Get gateway stats
const stats = await client.stats.get();
console.log('Total requests:', stats.totalRequests);
console.log('Active tokens:', stats.governance.active);
console.log('Banned tokens:', stats.governance.banned);
```

## Token Delegation

### Basic Delegation

```typescript
// Create a child token with reduced scope
const child = await client.tokens.delegate({
  parentToken: token.token,
  caveats: ['scope = api:read:users'],
});
console.log('Child signature:', child.signature);
```

### Fluent Delegation Builder

```typescript
import { delegate, Caveats, DelegationPatterns } from '@satgate/client';

// Fluent builder pattern
const teamToken = await delegate(rootToken)
  .withScope('api:read')
  .withExpiry(24 * 3600)  // 24 hours
  .withBudget(100, 'USD')
  .forTeam('engineering')
  .delegate(client);

// Pre-built patterns
const readOnlyToken = await DelegationPatterns.readOnly(rootToken).delegate(client);

const tempToken = await DelegationPatterns.temporary(rootToken, 2).delegate(client); // 2 hours

const apiClientToken = await DelegationPatterns.apiClient(
  rootToken,
  'my-app-client',
  1000  // requests per minute
).delegate(client);

// Webhook callback token (single route, short expiry)
const webhookToken = await DelegationPatterns.webhook(
  rootToken,
  '/api/webhooks/stripe',
  30  // 30 minutes
).delegate(client);

// CI/CD pipeline token
const ciToken = await DelegationPatterns.cicd(
  rootToken,
  'deploy-prod-123',
  60  // 60 minutes
).delegate(client);

// Agent swarm token with budget
const swarmToken = await DelegationPatterns.agentSwarm(
  rootToken,
  'ai-agents-prod',
  {
    budget: 500,
    currency: 'USD',
    requestsPerMinute: 5000,
    maxAgents: 100,
  }
).delegate(client);
```

### Caveat Builders

```typescript
import { Caveats } from '@satgate/client';

// Build caveats programmatically
const caveats = [
  Caveats.scope('api:read:users,api:read:posts'),
  Caveats.expires(3600),  // 1 hour
  Caveats.routes(['/api/v1/*', '/health']),
  Caveats.rateLimit(100, 60),  // 100 req/min
  Caveats.budget(50, 'USD'),
  Caveats.sourceIp(['10.0.0.0/8']),
  Caveats.methods(['GET', 'POST']),
  Caveats.team('engineering'),
  Caveats.project('api-gateway'),
  Caveats.label('env', 'production'),
];

const token = await client.tokens.delegate({
  parentToken: rootToken,
  caveats,
});
```

## Configuration

```typescript
// Get current config
const config = await client.config.get();
console.log('Routes:', config.routes?.length);

// Validate a configuration
const isValid = await client.config.validate({
  version: 1,
  routes: [...],
});
```

## Error Handling

```typescript
import {
  SatGateClient,
  AuthenticationError,
  NotFoundError,
  SatGateError,
} from '@satgate/client';

try {
  const client = new SatGateClient({
    url: 'http://localhost:9090',
    token: 'invalid-token',
  });
  await client.tokens.list();
} catch (error) {
  if (error instanceof AuthenticationError) {
    console.error('Invalid admin token!');
  } else if (error instanceof NotFoundError) {
    console.error('Resource not found!');
  } else if (error instanceof SatGateError) {
    console.error('API error:', error.message);
  }
}
```

## TypeScript Support

This package includes TypeScript type definitions. All types are exported:

```typescript
import type { Token, TokenInfo, Stats, MintRequest } from '@satgate/client';
```

## License

MIT License




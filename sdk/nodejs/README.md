# @satgate/client

Official Node.js SDK for the [SatGate](https://github.com/SatGate-io/satgate) OSS Gateway.

## Installation

```bash
npm install @satgate/client
```

## Quick Start

### Admin Client

The `SatGateClient` provides direct access to gateway admin operations.

```typescript
import { SatGateClient } from '@satgate/client';

const client = new SatGateClient({
  url: 'http://localhost:8080',
  token: 'your-admin-token',
});

// Check gateway health
const healthy = await client.health();
console.log('Gateway healthy:', healthy);

// Mint a new capability token
const token = await client.tokens.mint({
  scope: 'api:read',
  duration: '1h',
});
console.log('Token:', token.token);
console.log('Signature:', token.signature);

// Validate a token
const result = await client.tokens.validate(token.token);
console.log('Valid:', result.valid);

// Delegate a token with additional restrictions
const child = await client.tokens.delegate({
  parentToken: token.token,
  caveats: ['scope = api:read'],
});
console.log('Child signature:', child.signature);

// Ping with a capability token
const ping = await client.ping(token.token);
console.log('Ping:', ping);

// Ban a compromised token
await client.governance.ban(token.signature, 'Compromised credentials');

// Get governance graph (token lineage, stats)
const graph = await client.governance.getGraph();
console.log('Active tokens:', graph.stats.active);

// Reset governance data
await client.governance.reset();
```

### Agent Client

The `SatGateAgentClient` is designed for AI agents — it automatically
mints tokens, caches them, and handles L402 payment challenges.

```typescript
import { SatGateAgentClient } from '@satgate/client';

// With admin token (auto-mints capability tokens)
const client = new SatGateAgentClient({
  gatewayUrl: 'http://localhost:8080',
  adminToken: 'your-admin-token',
  scope: 'api:read',
  duration: '1h',
});

// Or with a pre-existing capability token
const client2 = new SatGateAgentClient({
  gatewayUrl: 'http://localhost:8080',
  token: 'your-capability-token',
});

// Or via environment variables
// SATGATE_ADMIN_TOKEN=your-admin-token
// SATGATE_TOKEN=your-capability-token (alternative)
const client3 = new SatGateAgentClient({
  gatewayUrl: 'http://localhost:8080',
});

// Make requests — token handling is automatic
const response = await client.get('/api/data');
console.log(response.data);

// Ping to verify the token
const pingResult = await client.ping();
console.log(pingResult);

// Delegate a child token for a worker
const childToken = await client.delegate({
  caveats: ['scope = api:read'],
});
```

## Token Delegation

### Fluent Delegation Builder

```typescript
import { SatGateClient, delegate, Caveats, DelegationPatterns } from '@satgate/client';

const client = new SatGateClient({
  url: 'http://localhost:8080',
  token: 'admin-token',
});
const root = await client.tokens.mint({ scope: 'api:*', duration: '24h' });

// Fluent builder pattern
const teamToken = await delegate(root.token)
  .withScope('api:read')
  .withExpiry(24 * 3600)
  .forTeam('engineering')
  .delegate(client);

// Pre-built patterns
const readOnlyToken = await DelegationPatterns.readOnly(root.token).delegate(client);
const tempToken = await DelegationPatterns.temporary(root.token, 2).delegate(client);

// Agent swarm token
const swarmToken = await DelegationPatterns.agentSwarm(
  root.token,
  'ai-agents',
  { budget: 500, requestsPerMinute: 5000 }
).delegate(client);
```

### Caveat Builders

```typescript
import { Caveats } from '@satgate/client';

const caveats = [
  Caveats.scope('api:read'),
  Caveats.expires(3600),
  Caveats.routes(['/api/*', '/health']),
  Caveats.rateLimit(100, 60),
  Caveats.methods(['GET', 'POST']),
  Caveats.team('engineering'),
];

const token = await client.tokens.delegate({
  parentToken: root.token,
  caveats,
});
```

## OSS Gateway Endpoints

This SDK targets the SatGate OSS gateway endpoints:

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/capability/mint` | X-Admin-Token | Mint a new capability token |
| `POST` | `/api/capability/validate` | — | Validate a token |
| `POST` | `/api/capability/delegate` | — | Delegate a token with caveats |
| `GET` | `/api/capability/ping` | Bearer token | Verify a token is valid |
| `GET` | `/api/capability/admin` | Bearer token | Admin-scope verification |
| `POST` | `/api/governance/ban` | X-Admin-Token | Ban a token |
| `GET` | `/api/governance/graph` | — | Get token lineage graph |
| `POST` | `/api/governance/reset` | X-Admin-Token | Reset governance data |
| `GET` | `/health` | — | Health check |

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
    url: 'http://localhost:8080',
    token: 'invalid-token',
  });
  await client.tokens.mint();
} catch (error) {
  if (error instanceof AuthenticationError) {
    console.error('Invalid admin token!');
  } else if (error instanceof SatGateError) {
    console.error('API error:', error.message);
  }
}
```

## TypeScript Support

This package includes full TypeScript type definitions:

```typescript
import type { Token, TokenInfo, GraphData, DelegateRequest } from '@satgate/client';
```

## License

MIT License

# Node.js SDK

Official Node.js/TypeScript client for SatGate Gateway.

## Installation

```bash
npm install satgate-sdk
# or
yarn add satgate-sdk
# or
pnpm add satgate-sdk
```

## Quick Start

```typescript
import { SatGateClient } from 'satgate-sdk';

// Create client
const client = new SatGateClient('https://api.example.com', {
  adminToken: 'your-admin-token'
});

// Mint a token
const token = await client.tokens.mint({
  scope: 'api:read',
  expiresIn: 3600, // 1 hour
  metadata: { user: 'agent-1' }
});

console.log(`Token: ${token.token}`);
console.log(`Expires: ${token.expiresAt}`);
```

## Client Configuration

```typescript
import { SatGateClient } from 'satgate-sdk';

// Basic configuration
const client = new SatGateClient('https://api.example.com', {
  adminToken: 'your-admin-token'
});

// Full configuration
const client = new SatGateClient('https://api.example.com', {
  adminToken: 'your-admin-token',
  timeout: 30000,
  retries: 3,
  retryDelay: 1000,
  headers: {
    'X-Custom-Header': 'value'
  }
});

// With JWT authentication
const client = new SatGateClient('https://api.example.com', {
  jwt: 'your-jwt-token'
});
```

## Token Management

### Mint Token

```typescript
const token = await client.tokens.mint({
  scope: 'api:read,api:write',
  expiresIn: 86400, // 24 hours
  metadata: {
    user: 'agent-1',
    purpose: 'data-pipeline'
  }
});
```

### List Tokens

```typescript
const tokens = await client.tokens.list({ limit: 100, offset: 0 });

for (const t of tokens.items) {
  console.log(`Token: ${t.signature.slice(0, 8)}, Scope: ${t.scope}`);
}
```

### Get Token Details

```typescript
const token = await client.tokens.get('token-signature');
```

### Revoke Token

```typescript
await client.tokens.revoke('token-signature');
```

### Delegate Token

```typescript
const delegated = await client.tokens.delegate('parent-signature', {
  caveats: [
    { type: 'expires', value: '1h' },
    { type: 'rate_limit', value: '100/minute' },
    { type: 'scope', value: 'api:read' }
  ]
});
```

## Governance

### Ban Token

```typescript
await client.governance.ban({
  signature: 'token-to-ban',
  reason: 'Compromised credentials'
});
```

### Unban Token

```typescript
await client.governance.unban('token-signature');
```

### Get Ban List

```typescript
const banned = await client.governance.banList();

for (const sig of banned.signatures) {
  console.log(`Banned: ${sig}`);
}
```

### Get Token Lineage

```typescript
const graph = await client.governance.graph();

for (const node of graph.nodes) {
  console.log(`Token: ${node.signature}, Parent: ${node.parent}`);
}
```

## Configuration

### Get Config

```typescript
const config = await client.config.get();

console.log(`Routes: ${config.routes.length}`);
```

### Update Config

```typescript
await client.config.update({
  routes: [
    {
      name: 'new-api',
      path: '/v2/*',
      upstream: 'backend',
      policy: {
        kind: 'observe',
        scope: 'api:v2'
      }
    }
  ]
});
```

### Validate Config

```typescript
const result = await client.config.validate(configYaml);

if (!result.valid) {
  for (const error of result.errors) {
    console.log(`Error at line ${error.line}: ${error.message}`);
  }
}
```

## Statistics

### Get Gateway Stats

```typescript
const stats = await client.stats.get();

console.log(`Total Requests: ${stats.totalRequests}`);
console.log(`Active Tokens: ${stats.activeTokens}`);
```

### Get Route Stats

```typescript
const routes = await client.stats.routes();

for (const [name, stats] of Object.entries(routes)) {
  console.log(`${name}: ${stats.requests} requests, p99: ${stats.latencyP99}ms`);
}
```

## Making Protected Requests

```typescript
// Using a capability token
const response = await client.request('GET', '/api/users', {
  token: token.token
});

const users = await response.json();
```

## WebSocket Telemetry

```typescript
import { SatGateClient, EventType } from 'satgate-sdk';

const client = new SatGateClient('https://api.example.com', {
  adminToken: '...'
});

// Subscribe to events
const unsubscribe = client.telemetry.subscribe((event) => {
  switch (event.type) {
    case EventType.TokenMint:
      console.log(`Token minted: ${event.signature}`);
      break;
    case EventType.TokenBan:
      console.log(`Token banned: ${event.signature}`);
      break;
    case EventType.Request:
      console.log(`Request: ${event.method} ${event.path} (${event.status})`);
      break;
  }
});

// Later: unsubscribe
unsubscribe();
```

### Async Iterator

```typescript
for await (const event of client.telemetry.events()) {
  console.log(`Event: ${event.type}`);
}
```

## Error Handling

```typescript
import {
  SatGateError,
  AuthenticationError,
  AuthorizationError,
  RateLimitError,
  NotFoundError,
  ValidationError
} from 'satgate-sdk';

try {
  const token = await client.tokens.mint({ scope: 'api:read' });
} catch (error) {
  if (error instanceof RateLimitError) {
    console.log(`Rate limited. Retry after ${error.retryAfter} seconds`);
    await sleep(error.retryAfter * 1000);
  } else if (error instanceof AuthenticationError) {
    console.log('Invalid credentials');
  } else if (error instanceof ValidationError) {
    console.log(`Validation error: ${error.message}`);
  } else if (error instanceof SatGateError) {
    console.log(`API error: ${error.code} - ${error.message}`);
  }
}
```

## TypeScript Types

Full TypeScript support with exported types:

```typescript
import type {
  Token,
  MintRequest,
  DelegateRequest,
  Caveat,
  Route,
  Policy,
  GatewayStats,
  TelemetryEvent
} from 'satgate-sdk';

function mintToken(client: SatGateClient, request: MintRequest): Promise<Token> {
  return client.tokens.mint(request);
}
```

## Testing

Use the mock client for testing:

```typescript
import { MockSatGateClient } from 'satgate-sdk/testing';

describe('MyService', () => {
  it('should mint token', async () => {
    const mockClient = new MockSatGateClient();
    mockClient.tokens.mint.mockResolvedValue({
      token: 'mock-token',
      signature: 'mock-sig',
      expiresAt: new Date()
    });

    const service = new MyService(mockClient);
    const result = await service.doSomething();

    expect(mockClient.tokens.mint).toHaveBeenCalled();
    expect(result).toBe(expected);
  });
});
```

## Browser Usage

The SDK works in browsers with bundlers:

```typescript
// Works with Vite, webpack, etc.
import { SatGateClient } from 'satgate-sdk';

const client = new SatGateClient('https://api.example.com', {
  adminToken: 'your-token'
});
```

**Note**: Never expose admin tokens in client-side code. Use delegated tokens with appropriate restrictions.

## CommonJS

```javascript
const { SatGateClient } = require('satgate-sdk');

const client = new SatGateClient('https://api.example.com', {
  adminToken: 'your-admin-token'
});
```

## Logging

```typescript
import { SatGateClient, LogLevel } from 'satgate-sdk';

const client = new SatGateClient('https://api.example.com', {
  adminToken: 'your-token',
  logLevel: LogLevel.Debug,
  logger: (level, message, data) => {
    console.log(`[${level}] ${message}`, data);
  }
});
```




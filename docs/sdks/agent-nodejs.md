# Node.js Agent Client

The `SatGateAgentClient` is included in the `@satgate/sdk` package. It provides automatic token management, caching, refresh, and L402 payment handling for AI agents.

See the [Node.js SDK documentation](nodejs.md#agent-client-satgateagentclient) for full usage.

## Quick Start

```typescript
import { SatGateAgentClient } from '@satgate/sdk';

const client = new SatGateAgentClient({
  gatewayUrl: 'http://localhost:8080',
  adminToken: 'your-admin-token',
});

// Tokens are minted and managed automatically
const response = await client.get('/api/data');
console.log(response.data);
```

## Features

- Auto-mints capability tokens via `POST /api/capability/mint`
- Caches tokens and refreshes on expiry
- Handles 401 → auto-refresh → retry
- Handles 402 → Lightning payment → retry (with wallet)
- Budget tracking
- Token delegation for worker agents
- LND and Alby wallet support

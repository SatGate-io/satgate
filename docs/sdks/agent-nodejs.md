# Node.js Agent SDK

The `SatGateAgentClient` is designed for AI agents to interact with SatGate-protected APIs. It automatically handles:

- 402 Payment Required challenges (L402 and Fiat402)
- Token minting via Mint (Trust Broker)
- Macaroon management and delegation
- Retry logic for transient failures

## Installation

```bash
npm install @satgate/sdk
# or
yarn add @satgate/sdk
# or
pnpm add @satgate/sdk
```

## Quick Start

### With Pre-existing Token

```typescript
import { SatGateAgentClient } from '@satgate/sdk';

const client = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  token: '<your-capability-token>'
});

// Make requests - 402 challenges handled automatically
const response = await client.get('/api/data');
console.log(response.data);
```

### With Kubernetes Identity (Badge-in)

```typescript
import { SatGateAgentClient, KubernetesIdentity } from '@satgate/sdk';

// Automatically uses ServiceAccount token mounted at
// /var/run/secrets/kubernetes.io/serviceaccount/token
const identity = new KubernetesIdentity();

const client = new SatGateAgentClient({
  gatewayUrl: 'https://gateway.internal:8080',
  mintUrl: 'https://gateway.internal:9090',  // Admin listener
  identity
});

// Token obtained automatically from Mint
const response = await client.get('/api/internal/data');
```

### With AWS Identity

```typescript
import { SatGateAgentClient, AWSIdentity } from '@satgate/sdk';

// Uses presigned GetCallerIdentity URL (recommended)
const identity = new AWSIdentity();

// For EKS with IRSA
const identity = new AWSIdentity({
  irsaRoleArn: 'arn:aws:iam::123456789012:role/MyAgentRole'
});

const client = new SatGateAgentClient({
  gatewayUrl: 'https://gateway.example.com',
  mintUrl: 'https://gateway.example.com:9090',
  identity
});
```

### With Lightning Payments (L402)

```typescript
import { SatGateAgentClient } from '@satgate/sdk';

// Using Nostr Wallet Connect (NWC)
const client = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  nwcConnectionString: 'nostr+walletconnect://...'
});

// Or using Alby
const client = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  albyToken: '<your-alby-access-token>'
});

// Payments happen automatically on 402
const response = await client.get('/api/premium/data');
```

---

## Configuration

### Environment Variables

```bash
# Gateway URL (required)
SATGATE_GATEWAY_URL=https://api.example.com

# Mint URL (if different from gateway)
SATGATE_MINT_URL=https://gateway.example.com:9090

# Pre-existing token
SATGATE_TOKEN=<macaroon>

# Lightning (for L402 mode)
SATGATE_NWC_CONNECTION_STRING=nostr+walletconnect://...
SATGATE_ALBY_TOKEN=<alby-access-token>

# Budget mode
SATGATE_BUDGET_MODE=fiat402
SATGATE_DEPARTMENT_ID=engineering
```

### Full Configuration

```typescript
import { SatGateAgentClient, KubernetesIdentity } from '@satgate/sdk';

const client = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  mintUrl: 'https://gateway.example.com:9090',  // Optional separate Mint
  
  // Authentication (pick one)
  token: '<capability-token>',              // Pre-existing token
  identity: new KubernetesIdentity(),       // Badge-in identity
  
  // Payment handling
  nwcConnectionString: '...',               // For L402 mode
  budgetMode: 'fiat402',                    // For Fiat402 mode
  departmentId: 'engineering',              // Cost center for chargebacks
  
  // Behavior
  autoRetry: true,                          // Retry on 402 (default: true)
  maxRetries: 3,                            // Max retry attempts
  timeout: 30000,                           // Request timeout (ms)
  
  // Token management
  autoRefresh: true,                        // Refresh tokens before expiry
  refreshThreshold: 300,                    // Refresh 5 min before expiry
});
```

### From Environment

```typescript
// Automatically reads SATGATE_* environment variables
const client = SatGateAgentClient.fromEnv();
```

---

## Making Requests

### Basic Requests

```typescript
// GET
const response = await client.get('/api/users');

// POST with JSON
const response = await client.post('/api/users', { name: 'Agent' });

// PUT
const response = await client.put('/api/users/1', { name: 'Updated' });

// DELETE
const response = await client.delete('/api/users/1');

// Generic request
const response = await client.request('PATCH', '/api/users/1', {
  data: { status: 'active' }
});
```

### With Headers

```typescript
const response = await client.get('/api/data', {
  headers: {
    'X-Custom-Header': 'value',
    'Accept': 'application/json'
  }
});
```

### With Query Parameters

```typescript
const response = await client.get('/api/search', {
  params: {
    q: 'machine learning',
    limit: 100
  }
});
```

### Streaming Responses

```typescript
// For large responses or real-time data
const response = await client.get('/api/stream', { responseType: 'stream' });

for await (const chunk of response.data) {
  process(chunk);
}
```

---

## Handling 402 Responses

The client automatically handles 402 Payment Required challenges:

### L402 (Lightning)

```typescript
const client = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  nwcConnectionString: 'nostr+walletconnect://...'
});

// On 402:
// 1. Client parses WWW-Authenticate header
// 2. Pays Lightning invoice via NWC
// 3. Retries request with L402 token
const response = await client.get('/api/premium');  // Automatic!
```

### Fiat402 (Internal Budget)

```typescript
const client = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  budgetMode: 'fiat402',
  departmentId: 'engineering'
});

// On 402:
// 1. Client parses Fiat402 invoice
// 2. Acknowledges cost (budget deduction)
// 3. Retries request with receipt
const response = await client.get('/api/expensive');  // Automatic!
```

### Manual 402 Handling

```typescript
import { SatGateAgentClient, PaymentRequiredError } from '@satgate/sdk';

const client = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  autoRetry: false  // Disable auto-retry
});

try {
  const response = await client.get('/api/premium');
} catch (error) {
  if (error instanceof PaymentRequiredError) {
    console.log(`Payment required: ${error.invoice}`);
    console.log(`Amount: ${error.amount} ${error.unit}`);
    
    // Handle payment manually
    const receipt = await payInvoiceSomehow(error.invoice);
    
    // Retry with receipt
    const response = await client.get('/api/premium', {
      headers: {
        'Authorization': `L402 ${error.macaroon}:${receipt}`
      }
    });
  }
}
```

---

## Token Delegation

Create attenuated child tokens for sub-agents:

```typescript
// Parent agent
const parentClient = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  token: '<parent-token>'
});

// Delegate to sub-agent with restrictions
const childToken = await parentClient.delegate({
  caveats: [
    'scope = api:read',           // Read-only
    'expires = 3600',              // 1 hour
    'max_requests = 100',          // Rate limit
    'ip = 10.0.0.0/8'             // Network restriction
  ]
});

// Sub-agent uses delegated token
const subAgent = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  token: childToken
});
```

---

## LangChain.js Integration

```typescript
import { ChatOpenAI } from '@langchain/openai';
import { AgentExecutor } from 'langchain/agents';
import { SatGateTool } from '@satgate/sdk/langchain';

// Create a budget-aware tool
const tool = new SatGateTool({
  name: 'api_call',
  description: 'Call the protected API',
  gatewayUrl: 'https://api.example.com',
  nwcConnectionString: 'nostr+walletconnect://...'
});

// Use in agent
const agent = AgentExecutor.fromAgentAndTools({
  agent: yourAgent,
  tools: [tool],
  verbose: true
});

// Agent automatically handles payments
const result = await agent.call({
  input: 'Fetch the premium data from the API'
});
```

---

## TypeScript Types

```typescript
import type {
  SatGateAgentClientOptions,
  AgentResponse,
  DelegateOptions,
  Identity,
  PaymentInfo
} from '@satgate/sdk';

// Full type safety
const options: SatGateAgentClientOptions = {
  gatewayUrl: 'https://api.example.com',
  token: '<token>'
};

const client = new SatGateAgentClient(options);
const response: AgentResponse = await client.get('/api/data');
```

---

## Error Handling

```typescript
import {
  SatGateError,
  AuthenticationError,
  PaymentRequiredError,
  InsufficientBudgetError,
  RateLimitError,
  TokenExpiredError,
  MintError
} from '@satgate/sdk';

try {
  const response = await client.get('/api/data');
} catch (error) {
  if (error instanceof TokenExpiredError) {
    // Token expired, client will auto-refresh if configured
  } else if (error instanceof PaymentRequiredError) {
    console.log(`Need to pay ${error.amount} ${error.unit}`);
  } else if (error instanceof InsufficientBudgetError) {
    console.log(`Budget exhausted: ${error.remaining} remaining`);
  } else if (error instanceof RateLimitError) {
    console.log(`Rate limited, retry after ${error.retryAfter}s`);
  } else if (error instanceof MintError) {
    console.log(`Failed to obtain token from Mint: ${error.message}`);
  } else if (error instanceof SatGateError) {
    console.log(`Gateway error: ${error.message}`);
  }
}
```

---

## Best Practices

### 1. Use Identity Providers in Kubernetes/AWS

```typescript
// Don't hardcode tokens in K8s workloads
// BAD:
const client = new SatGateAgentClient({ token: 'hardcoded-token' });

// GOOD:
const client = new SatGateAgentClient({ 
  identity: new KubernetesIdentity() 
});
```

### 2. Set Budget Limits for Sub-Agents

```typescript
// Always delegate with restrictions
const childToken = await parent.delegate({
  caveats: [
    'budget_limit = 10.00 USD',
    'expires = 3600'
  ]
});
```

### 3. Handle Payment Failures Gracefully

```typescript
import { PaymentFailedError } from '@satgate/sdk';

try {
  const response = await client.get('/api/premium');
} catch (error) {
  if (error instanceof PaymentFailedError) {
    // Fall back to free tier or cached data
    const response = await client.get('/api/basic');
  }
}
```

### 4. Use Environment Variables for Configuration

```typescript
// Let configuration come from environment
const client = SatGateAgentClient.fromEnv();
```

### 5. Clean Up Resources

```typescript
// Always close the client when done
try {
  const response = await client.get('/api/data');
} finally {
  await client.close();
}

// Or use a wrapper
async function withClient<T>(
  fn: (client: SatGateAgentClient) => Promise<T>
): Promise<T> {
  const client = SatGateAgentClient.fromEnv();
  try {
    return await fn(client);
  } finally {
    await client.close();
  }
}
```

---

## Debugging

```typescript
import { SatGateAgentClient, LogLevel } from '@satgate/sdk';

const client = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  token: '<token>',
  logLevel: LogLevel.Debug,
  logger: (level, message, data) => {
    console.log(`[${level}] ${message}`, data);
  }
});
```

---

## Complete Example: AI Research Agent

```typescript
import { SatGateAgentClient, KubernetesIdentity } from '@satgate/sdk';

async function main() {
  // Initialize client with K8s identity
  const client = new SatGateAgentClient({
    gatewayUrl: process.env.SATGATE_GATEWAY_URL!,
    mintUrl: process.env.SATGATE_MINT_URL,
    identity: new KubernetesIdentity(),
    autoRetry: true
  });
  
  try {
    // Fetch data from protected API
    const response = await client.get('/api/research/papers', {
      params: {
        topic: 'machine learning',
        limit: 100
      }
    });
    
    const papers = response.data;
    
    for (const paper of papers.items) {
      // Each request may incur cost, but handled automatically
      const details = await client.get(`/api/research/papers/${paper.id}`);
      await processPaper(details.data);
    }
  } catch (error) {
    console.error('Error:', error);
  } finally {
    await client.close();
  }
}

main();
```

---

## CommonJS Support

```javascript
const { SatGateAgentClient, KubernetesIdentity } = require('@satgate/sdk');

const client = new SatGateAgentClient({
  gatewayUrl: 'https://api.example.com',
  identity: new KubernetesIdentity()
});
```

---

## Browser Usage

The Agent SDK is primarily designed for server-side Node.js environments. For browser usage, use the admin client SDK instead with pre-delegated tokens:

```typescript
// Server: Delegate a restricted token for the browser
const browserToken = await serverClient.delegate({
  caveats: [
    'scope = api:read',
    'expires = 3600',
    'origin = https://myapp.com'
  ]
});

// Browser: Use the delegated token
import { SatGateClient } from 'satgate-sdk';

const client = new SatGateClient('https://api.example.com', {
  token: browserToken
});
```

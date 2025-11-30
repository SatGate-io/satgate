# SatGate Client SDK

The official JavaScript client for building AI Agents and Apps that pay for API usage via the L402 protocol.

## Installation

```bash
npm install @satgate/client
```

## Usage

### 1. Browser (with Alby/WebLN)

The client automatically detects `window.webln` (Alby, standard WebLN providers).

```javascript
import { SatGateClient } from '@satgate/client';

const client = new SatGateClient();

// This request will automatically:
// 1. Detect a 402 Payment Required response
// 2. Prompt the user/wallet to pay the invoice
// 3. Retry the request with the L402 Authorization token
const response = await client.get('https://api.yourservice.com/premium-data');
const data = await response.json();
```

### 2. Server / AI Agent (Custom Wallet)

For autonomous agents, you can inject a custom wallet provider (e.g., wrapping LND or LNBits).

```javascript
import { SatGateClient } from '@satgate/client';

const myAgentWallet = {
  async payInvoice(invoice) {
    // Logic to pay invoice via LND/LNBits API
    return { preimage: '...' };
  }
};

const client = new SatGateClient({ 
  wallet: myAgentWallet,
  onPayment: (details) => console.log('Paid for resource:', details.url)
});

await client.post('https://api.yourservice.com/agent-action', { task: 'analyze' });
```


# SatGate JavaScript SDK

**Give your AI agents a Lightning wallet in 2 lines of code.**

Automatic L402 payment handling — the "Stripe Moment" for autonomous agents.

Works in both **Browser** (WebLN/Alby) and **Node.js** (LNBits/Alby API).

## Installation

```bash
npm install @satgate/sdk
```

## Quick Start

### Browser (with Alby/WebLN)

```javascript
import { SatGateClient } from '@satgate/sdk';

// 1. Create client (uses WebLN automatically)
const client = new SatGateClient({ wallet: 'webln' });

// 2. Make requests - 402 → Pay → Retry happens automatically
const response = await client.get('https://api.example.com/premium');
const data = await response.json();
```

### Node.js (with LNBits)

```javascript
import { SatGateClient, LNBitsWallet } from '@satgate/sdk';

// 1. Connect your wallet
const wallet = new LNBitsWallet({
  url: 'https://legend.lnbits.com',
  adminKey: 'your-admin-key'
});

// 2. Create client
const client = new SatGateClient({ wallet });

// 3. Make requests - payment is automatic
const response = await client.get('https://api.example.com/premium');
```

### Node.js (with Alby API)

```javascript
import { SatGateClient, AlbyWallet } from '@satgate/sdk';

const wallet = new AlbyWallet({
  accessToken: 'your-alby-access-token'
});

const client = new SatGateClient({ wallet });
const response = await client.get('https://api.example.com/premium');
```

## What Happens Under the Hood

```
1. GET /premium/data
   ↓
2. Server returns 402 + Lightning Invoice
   ↓
3. SDK automatically pays invoice
   ↓
4. SDK retries with L402 token
   ↓
5. You get the response ✓
```

## Features

### Token Caching

Tokens are cached by default to avoid paying twice:

```javascript
const client = new SatGateClient({ 
  wallet,
  cacheTokens: true,  // default
  cacheTtl: 3600      // 1 hour (default)
});

// First call: pays invoice
await client.get('/premium');

// Second call: uses cached token (no payment)
await client.get('/premium');
```

### Payment Callbacks

Track payments in real-time:

```javascript
const client = new SatGateClient({
  wallet,
  onChallenge: ({ invoice, macaroon }) => {
    console.log('Payment required:', invoice);
  },
  onPaymentStart: ({ invoice }) => {
    console.log('Paying...');
  },
  onPayment: (info) => {
    console.log(`Paid ${info.amountSats} sats for ${info.endpoint}`);
  }
});
```

### Session Tracking

```javascript
const client = new SatGateClient({ wallet });

await client.get('/endpoint1');
await client.get('/endpoint2');

console.log(`Total spent: ${client.getTotalPaidSats()} sats`);
```

### Quiet Mode

```javascript
const client = new SatGateClient({ wallet, verbose: false });
```

## Wallet Options

### WebLN (Browser)

```javascript
// Automatically uses window.webln (Alby, etc.)
const client = new SatGateClient({ wallet: 'webln' });
```

### LNBits (Server)

```javascript
import { LNBitsWallet } from '@satgate/sdk';

const wallet = new LNBitsWallet({
  url: 'https://your-lnbits-instance.com',
  adminKey: 'your-admin-key'
});
```

### Alby API (Server)

```javascript
import { AlbyWallet } from '@satgate/sdk';

const wallet = new AlbyWallet({
  accessToken: 'your-access-token'
});
```

### Custom Wallet

Implement the `LightningWallet` interface:

```javascript
class MyWallet {
  async payInvoice(invoice) {
    // Connect to your LND, CLN, etc.
    const preimage = await myNode.pay(invoice);
    return preimage; // hex string
  }
}

const client = new SatGateClient({ wallet: new MyWallet() });
```

## API Reference

### SatGateClient

```javascript
new SatGateClient({
  wallet: LightningWallet | 'webln',  // Required for payments
  baseUrl?: string,                    // Prefix for all URLs
  cacheTokens?: boolean,               // Cache L402 tokens (default: true)
  cacheTtl?: number,                   // Cache TTL in seconds (default: 3600)
  verbose?: boolean,                   // Console logging (default: true)
  onChallenge?: (challenge) => void,   // Called on 402
  onPaymentStart?: (data) => void,     // Called before payment
  onPayment?: (info) => void           // Called after payment
})
```

### Methods

```javascript
// Make any request
await client.fetch(url, init);

// GET request
await client.get(url, init?);

// POST request
await client.post(url, body, init?);

// Get total sats spent
client.getTotalPaidSats();

// Clear token cache
client.clearCache();
```

### PaymentInfo

```typescript
interface PaymentInfo {
  invoice: string;      // BOLT11 invoice
  preimage: string;     // Payment proof (hex)
  macaroon: string;     // L402 macaroon
  amountSats: number;   // Amount paid
  endpoint: string;     // URL accessed
  timestamp: number;    // Unix timestamp
}
```

## TypeScript

Full TypeScript support included:

```typescript
import { SatGateClient, LNBitsWallet, PaymentInfo } from '@satgate/sdk';

const client = new SatGateClient({
  wallet: new LNBitsWallet({ url: '...', adminKey: '...' }),
  onPayment: (info: PaymentInfo) => {
    console.log(info.amountSats);
  }
});
```

## License

MIT

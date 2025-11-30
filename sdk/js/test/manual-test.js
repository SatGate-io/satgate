
import { SatGateClient } from '../src/index.js';

// Mock Fetch - simulates Aperture behavior
let requestCount = 0;
let savedMacaroon = null;
let savedPreimage = null;

global.fetch = async (url, init) => {
  requestCount++;
  console.log(`\n[Network] Request #${requestCount}: ${init?.method || 'GET'} ${url}`);
  
  const authHeader = init?.headers?.Authorization;
  
  if (authHeader) {
    console.log('[Network] Authorization header:', authHeader.substring(0, 60) + '...');
    
    // Parse the auth header
    const match = authHeader.match(/^(LSAT|L402)\s+([^:]+):(.+)$/);
    if (match) {
      const [, scheme, macaroon, preimage] = match;
      
      // Verify the macaroon matches what we issued
      if (macaroon === savedMacaroon && preimage === savedPreimage) {
        console.log('[Network] ✅ Token verified! Granting access.');
        return {
          ok: true,
          status: 200,
          json: async () => ({ success: true, data: "Premium Content", tier: "standard" }),
          headers: { get: () => 'application/json' }
        };
      } else {
        console.log('[Network] ❌ Token mismatch!');
        console.log(`  Expected macaroon: ${savedMacaroon}`);
        console.log(`  Got macaroon: ${macaroon}`);
      }
    }
  }

  // Generate new challenge (like Aperture does for each request)
  savedMacaroon = `macaroon_${Date.now()}`;
  savedPreimage = 'correct_preimage_12345678901234567890123456789012';
  
  console.log('[Network] Returning 402 with new challenge');
  console.log(`  New macaroon: ${savedMacaroon}`);
  
  return {
    ok: false,
    status: 402,
    headers: {
      get: (name) => {
        if (name === 'WWW-Authenticate') {
          return `LSAT macaroon="${savedMacaroon}", invoice="lnbc100n_test_invoice"`;
        }
        return null;
      }
    }
  };
};

// Mock Wallet - returns the "correct" preimage that matches what Aperture expects
const mockWallet = {
  payInvoice: async (invoice) => {
    console.log(`[Wallet] Paying invoice: ${invoice}`);
    // In real life, the Lightning network reveals this preimage upon payment
    // The preimage's SHA256 hash equals the payment hash in the invoice
    return { preimage: savedPreimage };
  }
};

async function run() {
  const client = new SatGateClient({ 
    wallet: mockWallet,
    onPayment: (d) => console.log(`[Client] Payment Event: Paid for ${d.url}`)
  });

  console.log('--- Starting Request ---');
  const res = await client.get('https://api.example.com/resource');
  const data = await res.json();
  console.log('--- Response Data ---');
  console.log(data);
}

run().catch(console.error);


/**
 * SatGate JavaScript/TypeScript SDK
 * 
 * Automatic L402 payment handling - the "Stripe Moment" for AI agents.
 * Works in both Browser (WebLN) and Node.js (LNBits/Alby API).
 * 
 * Browser Example:
 *   const client = new SatGateClient({ wallet: 'webln' });
 *   const response = await client.get('https://api.example.com/premium');
 * 
 * Node.js Example:
 *   const client = new SatGateClient({ 
 *     wallet: new LNBitsWallet({ url: '...', adminKey: '...' })
 *   });
 *   const response = await client.get('https://api.example.com/premium');
 */

// =============================================================================
// WALLET INTERFACES
// =============================================================================

/**
 * Abstract wallet interface - implement payInvoice to use any Lightning wallet
 */
export class LightningWallet {
  async payInvoice(invoice) {
    throw new Error('payInvoice must be implemented');
  }
}

/**
 * WebLN wallet for browser use (Alby extension, etc.)
 */
export class WebLNWallet extends LightningWallet {
  async payInvoice(invoice) {
    if (typeof window === 'undefined' || !window.webln) {
      throw new Error('WebLN not available. Install Alby or use a supported browser extension.');
    }
    await window.webln.enable();
    const result = await window.webln.sendPayment(invoice);
    return result.preimage;
  }
}

/**
 * LNBits wallet for server-side use
 */
export class LNBitsWallet extends LightningWallet {
  constructor({ url, adminKey }) {
    super();
    this.url = url.replace(/\/$/, '');
    this.adminKey = adminKey;
  }

  async payInvoice(invoice) {
    const response = await fetch(`${this.url}/api/v1/payments`, {
      method: 'POST',
      headers: {
        'X-Api-Key': this.adminKey,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ out: true, bolt11: invoice })
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`LNBits payment failed: ${error}`);
    }

    const data = await response.json();
    const paymentHash = data.payment_hash;

    // Get preimage from payment details
    const checkResponse = await fetch(`${this.url}/api/v1/payments/${paymentHash}`, {
      headers: { 'X-Api-Key': this.adminKey }
    });

    if (!checkResponse.ok) {
      throw new Error('Failed to get payment preimage');
    }

    const checkData = await checkResponse.json();
    return checkData.preimage;
  }
}

/**
 * Alby API wallet for server-side use
 */
export class AlbyWallet extends LightningWallet {
  constructor({ accessToken }) {
    super();
    this.accessToken = accessToken;
    this.baseUrl = 'https://api.getalby.com';
  }

  async payInvoice(invoice) {
    const response = await fetch(`${this.baseUrl}/payments/bolt11`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.accessToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ invoice })
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Alby payment failed: ${error}`);
    }

    const data = await response.json();
    return data.payment_preimage;
  }
}

// =============================================================================
// TOKEN CACHE
// =============================================================================

/**
 * Cache L402 tokens to avoid paying twice for the same endpoint
 */
class TokenCache {
  constructor(ttlSeconds = 3600) {
    this.cache = new Map();
    this.ttl = ttlSeconds * 1000;
  }

  _key(url) {
    // Simple hash of URL
    let hash = 0;
    for (let i = 0; i < url.length; i++) {
      const char = url.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return hash.toString(16);
  }

  get(url) {
    const key = this._key(url);
    const entry = this.cache.get(key);
    if (!entry) return null;

    if (Date.now() - entry.timestamp > this.ttl) {
      this.cache.delete(key);
      return null;
    }

    return entry.token;
  }

  set(url, token, info) {
    const key = this._key(url);
    this.cache.set(key, {
      token,
      info,
      timestamp: Date.now()
    });
  }

  clear() {
    this.cache.clear();
  }
}

// =============================================================================
// PAYMENT INFO
// =============================================================================

/**
 * Information about a completed payment
 */
export class PaymentInfo {
  constructor({ invoice, preimage, macaroon, amountSats, endpoint }) {
    this.invoice = invoice;
    this.preimage = preimage;
    this.macaroon = macaroon;
    this.amountSats = amountSats;
    this.endpoint = endpoint;
    this.timestamp = Date.now();
  }
}

// =============================================================================
// MAIN CLIENT
// =============================================================================

/**
 * SatGate Client - automatic L402 payment handling
 * 
 * @example
 * // Browser with WebLN
 * const client = new SatGateClient({ wallet: 'webln' });
 * 
 * // Node.js with LNBits
 * const client = new SatGateClient({
 *   wallet: new LNBitsWallet({ url: '...', adminKey: '...' })
 * });
 * 
 * // Make requests - 402 handling is automatic
 * const response = await client.get('https://api.example.com/premium');
 */
export class SatGateClient {
  constructor(config = {}) {
    this.wallet = this._resolveWallet(config.wallet);
    this.baseUrl = config.baseUrl || '';
    this.cacheTokens = config.cacheTokens !== false;
    this.cacheTtl = config.cacheTtl || 3600;
    this.verbose = config.verbose !== false;
    
    // Callbacks
    this.onChallenge = config.onChallenge || (() => {});
    this.onPaymentStart = config.onPaymentStart || (() => {});
    this.onPayment = config.onPayment || (() => {});
    
    // State
    this.cache = this.cacheTokens ? new TokenCache(this.cacheTtl) : null;
    this.totalPaidSats = 0;
  }

  _resolveWallet(walletConfig) {
    if (!walletConfig || walletConfig === 'webln' || walletConfig === 'alby') {
      return new WebLNWallet();
    }
    if (typeof walletConfig.payInvoice === 'function') {
      return walletConfig;
    }
    throw new Error('Unsupported wallet configuration. Use "webln", LNBitsWallet, AlbyWallet, or a custom wallet.');
  }

  _log(...args) {
    if (this.verbose) {
      console.log('[SatGate]', ...args);
    }
  }

  /**
   * Parse amount from BOLT11 invoice
   */
  _parseInvoiceAmount(invoice) {
    try {
      const match = invoice.toLowerCase().match(/^ln(?:bc|tb)(\d+)([munp])?/);
      if (!match) return null;

      const amount = parseInt(match[1]);
      const multiplier = match[2];

      switch (multiplier) {
        case 'm': return amount * 100000;      // milli-BTC
        case 'u': return amount * 100;          // micro-BTC
        case 'n': return Math.floor(amount / 10);  // nano-BTC
        case 'p': return Math.floor(amount / 10000); // pico-BTC
        default: return amount * 100000000;    // BTC
      }
    } catch {
      return null;
    }
  }

  /**
   * Main fetch method with L402 handling
   */
  async fetch(input, init = {}) {
    const url = this.baseUrl ? new URL(input, this.baseUrl).toString() : input;

    // Check cache first
    if (this.cache) {
      const cachedToken = this.cache.get(url);
      if (cachedToken) {
        this._log('Using cached L402 token');
        init.headers = {
          ...init.headers,
          'Authorization': cachedToken
        };
      }
    }

    const fetchInit = {
      ...init,
      cache: 'no-store',
      mode: 'cors'
    };

    let response = await fetch(url, fetchInit);

    if (response.status === 402) {
      return this._handleL402(response, url, fetchInit);
    }

    return response;
  }

  async get(url, init = {}) {
    return this.fetch(url, { ...init, method: 'GET' });
  }

  async post(url, body, init = {}) {
    return this.fetch(url, {
      ...init,
      method: 'POST',
      body: JSON.stringify(body),
      headers: {
        'Content-Type': 'application/json',
        ...init.headers
      }
    });
  }

  /**
   * Handle L402 payment challenge
   */
  async _handleL402(response, url, init) {
    const authHeader = response.headers.get('WWW-Authenticate');
    if (!authHeader) {
      throw new Error('402 Response missing WWW-Authenticate header');
    }

    const challenge = this._parseL402Header(authHeader);
    if (!challenge) {
      throw new Error('Invalid L402 WWW-Authenticate header format');
    }

    const amountSats = this._parseInvoiceAmount(challenge.invoice);
    
    this._log('L402 Challenge received');
    this._log(`  Invoice: ${challenge.invoice.substring(0, 30)}...`);
    this._log(`  Amount: ${amountSats ? `${amountSats} sats` : 'unknown'}`);

    this.onChallenge({ invoice: challenge.invoice, macaroon: challenge.macaroon });
    this.onPaymentStart({ invoice: challenge.invoice });

    try {
      const preimage = await this.wallet.payInvoice(challenge.invoice);

      this._log('Payment successful!');
      this._log(`  Preimage: ${preimage.substring(0, 16)}...`);

      // Track payment
      if (amountSats) {
        this.totalPaidSats += amountSats;
      }

      const paymentInfo = new PaymentInfo({
        invoice: challenge.invoice,
        preimage,
        macaroon: challenge.macaroon,
        amountSats,
        endpoint: url
      });

      this.onPayment(paymentInfo);

      // Build L402 token
      // Aperture expects: LSAT <macaroon>:<preimage>
      const l402Token = `LSAT ${challenge.macaroon}:${preimage}`;

      // Cache the token
      if (this.cache) {
        this.cache.set(url, l402Token, paymentInfo);
      }

      // Retry with authorization
      this._log('Retrying with L402 token...');

      const retryResponse = await fetch(url, {
        ...init,
        headers: {
          ...init.headers,
          'Authorization': l402Token
        }
      });

      this._log(`Retry response: ${retryResponse.status}`);

      return retryResponse;

    } catch (error) {
      this._log('Payment failed:', error.message);
      throw error;
    }
  }

  _parseL402Header(header) {
    // Aperture returns both LSAT and L402 headers
    // Extract the first valid macaroon/invoice pair
    const macaroonMatch = header.match(/macaroon="([^"]+)"/);
    const invoiceMatch = header.match(/invoice="([^"]+)"/);

    if (macaroonMatch && invoiceMatch) {
      return {
        macaroon: macaroonMatch[1],
        invoice: invoiceMatch[1]
      };
    }
    return null;
  }

  /**
   * Get total satoshis paid in this session
   */
  getTotalPaidSats() {
    return this.totalPaidSats;
  }

  /**
   * Clear the token cache
   */
  clearCache() {
    if (this.cache) {
      this.cache.clear();
    }
  }
}

// Default export
export default SatGateClient;

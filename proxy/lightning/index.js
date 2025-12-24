/**
 * SatGate Lightning Provider Abstraction
 * 
 * Supports multiple Lightning backends for invoice issuance and payment verification.
 * This makes SatGate the L402 authority (not dependent on Aperture for enforcement).
 * 
 * Supported backends:
 *   - phoenixd: Self-custodial, easy setup (recommended for getting started)
 *   - lnd: Enterprise standard (LND REST API)
 *   - opennode: Hosted solution (no node required)
 *   - mock: For testing/demo (no real Lightning)
 */

const crypto = require('crypto');
const https = require('https');

// HTTPS agent that accepts self-signed certificates (required for LND)
const insecureAgent = new https.Agent({ rejectUnauthorized: false });

// =============================================================================
// PROVIDER INTERFACE
// =============================================================================

class LightningProvider {
  /**
   * Create an invoice for the given amount
   * @param {number} amountSats - Amount in satoshis
   * @param {string} memo - Invoice description
   * @param {number} expirySecs - Invoice expiry in seconds
   * @returns {Promise<{paymentHash: string, paymentRequest: string, expiresAt: number}>}
   */
  async createInvoice(amountSats, memo, expirySecs = 3600) {
    throw new Error('createInvoice not implemented');
  }

  /**
   * Check if an invoice has been paid
   * @param {string} paymentHash - The payment hash (hex)
   * @returns {Promise<{paid: boolean, preimage?: string, settledAt?: number}>}
   */
  async checkInvoice(paymentHash) {
    throw new Error('checkInvoice not implemented');
  }

  /**
   * Verify a preimage matches a payment hash
   * @param {string} preimage - The preimage (hex)
   * @param {string} paymentHash - The payment hash (hex)
   * @returns {boolean}
   */
  verifyPreimage(preimage, paymentHash) {
    const computed = crypto.createHash('sha256')
      .update(Buffer.from(preimage, 'hex'))
      .digest('hex');
    return computed === paymentHash;
  }

  /**
   * Get provider status/health
   * @returns {Promise<{ok: boolean, backend: string, error?: string}>}
   */
  async getStatus() {
    throw new Error('getStatus not implemented');
  }
}

// =============================================================================
// PHOENIXD PROVIDER (Self-custodial, recommended)
// =============================================================================

class PhoenixdProvider extends LightningProvider {
  constructor(config) {
    super();
    this.baseUrl = config.url || 'http://localhost:9740';
    this.password = config.password || process.env.PHOENIXD_PASSWORD;
    this.name = 'phoenixd';
  }

  async createInvoice(amountSats, memo, expirySecs = 3600) {
    const response = await fetch(`${this.baseUrl}/createinvoice`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        'Authorization': `Basic ${Buffer.from(`:${this.password}`).toString('base64')}`
      },
      body: new URLSearchParams({
        amountSat: String(amountSats),
        description: memo,
        expirySeconds: String(expirySecs)
      })
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`phoenixd createinvoice failed: ${error}`);
    }

    const data = await response.json();
    return {
      paymentHash: data.paymentHash,
      paymentRequest: data.serialized,
      expiresAt: Date.now() + (expirySecs * 1000)
    };
  }

  async checkInvoice(paymentHash) {
    const response = await fetch(`${this.baseUrl}/payments/incoming/${paymentHash}`, {
      headers: {
        'Authorization': `Basic ${Buffer.from(`:${this.password}`).toString('base64')}`
      }
    });

    if (!response.ok) {
      if (response.status === 404) {
        return { paid: false };
      }
      throw new Error(`phoenixd check failed: ${await response.text()}`);
    }

    const data = await response.json();
    return {
      paid: data.isPaid === true,
      preimage: data.preimage,
      settledAt: data.completedAt
    };
  }

  async getStatus() {
    try {
      const response = await fetch(`${this.baseUrl}/getinfo`, {
        headers: {
          'Authorization': `Basic ${Buffer.from(`:${this.password}`).toString('base64')}`
        }
      });
      if (!response.ok) throw new Error('Not reachable');
      const data = await response.json();
      return { ok: true, backend: 'phoenixd', nodeId: data.nodeId };
    } catch (e) {
      return { ok: false, backend: 'phoenixd', error: e.message };
    }
  }
}

// =============================================================================
// LND PROVIDER (Enterprise standard)
// =============================================================================

class LndProvider extends LightningProvider {
  constructor(config) {
    super();
    this.baseUrl = config.url || process.env.LND_REST_URL || 'https://localhost:8080';
    // LND REST API expects macaroon as HEX, but we store as BASE64 for portability
    const macaroonBase64 = config.macaroon || process.env.LND_MACAROON;
    this.macaroon = macaroonBase64 ? Buffer.from(macaroonBase64, 'base64').toString('hex') : '';
    this.name = 'lnd';
  }

  async createInvoice(amountSats, memo, expirySecs = 3600) {
    // Use https module for self-signed cert support
    const url = new URL(`${this.baseUrl}/v1/invoices`);
    const postData = JSON.stringify({
      value: String(amountSats),
      memo: memo,
      expiry: String(expirySecs)
    });
    
    console.log(`[LND] Creating invoice: ${url.toString()} | host=${url.hostname} port=${url.port || 443} path=${url.pathname}`);
    console.log(`[LND] Macaroon (first 20 chars): ${this.macaroon.substring(0, 20)}...`);
    
    const response = await new Promise((resolve, reject) => {
      const req = https.request({
        hostname: url.hostname,
        port: url.port || 443,
        path: url.pathname,
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(postData),
          'Grpc-Metadata-macaroon': this.macaroon
        },
        rejectUnauthorized: false // Accept self-signed certs
      }, (res) => {
        let data = '';
        res.on('data', chunk => data += chunk);
        res.on('end', () => {
          console.log(`[LND] Response: status=${res.statusCode} body=${data.substring(0, 200)}`);
          resolve({ ok: res.statusCode >= 200 && res.statusCode < 300, status: res.statusCode, text: () => Promise.resolve(data), json: () => Promise.resolve(JSON.parse(data)) });
        });
      });
      req.on('error', (err) => {
        console.error(`[LND] Request error: ${err.message}`);
        reject(err);
      });
      req.write(postData);
      req.end();
    });

    if (!response.ok) {
      throw new Error(`LND createinvoice failed: ${await response.text()}`);
    }

    const data = await response.json();
    return {
      paymentHash: Buffer.from(data.r_hash, 'base64').toString('hex'),
      paymentRequest: data.payment_request,
      expiresAt: Date.now() + (expirySecs * 1000)
    };
  }

  async checkInvoice(paymentHash) {
    const hashBase64 = Buffer.from(paymentHash, 'hex').toString('base64')
      .replace(/\+/g, '-').replace(/\//g, '_');
    
    const url = new URL(`${this.baseUrl}/v1/invoice/${hashBase64}`);
    
    const response = await new Promise((resolve, reject) => {
      const req = https.request({
        hostname: url.hostname,
        port: url.port || 443,
        path: url.pathname,
        method: 'GET',
        headers: { 'Grpc-Metadata-macaroon': this.macaroon },
        rejectUnauthorized: false
      }, (res) => {
        let data = '';
        res.on('data', chunk => data += chunk);
        res.on('end', () => resolve({ ok: res.statusCode >= 200 && res.statusCode < 300, json: () => Promise.resolve(JSON.parse(data)) }));
      });
      req.on('error', reject);
      req.end();
    });

    if (!response.ok) {
      return { paid: false };
    }

    const data = await response.json();
    return {
      paid: data.state === 'SETTLED',
      preimage: data.r_preimage ? Buffer.from(data.r_preimage, 'base64').toString('hex') : undefined,
      settledAt: data.settle_date ? parseInt(data.settle_date) * 1000 : undefined
    };
  }

  async getStatus() {
    try {
      const url = new URL(`${this.baseUrl}/v1/invoices`);
      const postData = JSON.stringify({ value: '0' });
      
      await new Promise((resolve, reject) => {
        const req = https.request({
          hostname: url.hostname,
          port: url.port || 443,
          path: url.pathname,
          method: 'POST',
          headers: { 
            'Content-Type': 'application/json',
            'Content-Length': Buffer.byteLength(postData),
            'Grpc-Metadata-macaroon': this.macaroon 
          },
          rejectUnauthorized: false
        }, (res) => {
          let data = '';
          res.on('data', chunk => data += chunk);
          res.on('end', () => resolve({ ok: true }));
        });
        req.on('error', reject);
        req.write(postData);
        req.end();
      });
      
      return { ok: true, backend: 'lnd', note: 'Invoice macaroon connected' };
    } catch (e) {
      return { ok: false, backend: 'lnd', error: e.message };
    }
  }
}

// =============================================================================
// OPENNODE PROVIDER (Hosted, no node required)
// =============================================================================

class OpenNodeProvider extends LightningProvider {
  constructor(config) {
    super();
    this.apiKey = config.apiKey || process.env.OPENNODE_API_KEY;
    this.baseUrl = 'https://api.opennode.com/v1';
    this.name = 'opennode';
  }

  async createInvoice(amountSats, memo, expirySecs = 3600) {
    const response = await fetch(`${this.baseUrl}/charges`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': this.apiKey
      },
      body: JSON.stringify({
        amount: amountSats,
        description: memo,
        callback_url: null, // We poll instead
        ttl: Math.floor(expirySecs / 60) // OpenNode uses minutes
      })
    });

    if (!response.ok) {
      throw new Error(`OpenNode create failed: ${await response.text()}`);
    }

    const data = await response.json();
    const charge = data?.data || {};
    const ln = charge.lightning_invoice || {};
    const paymentRequest = ln.payreq;

    // IMPORTANT: For LSAT/L402 preimage verification we need the Lightning payment hash
    // (sha256(preimage)). Some hosted providers return a charge id instead; that will
    // break verifyPreimage().
    const paymentHash =
      ln.payment_hash ||
      ln.hash ||
      charge.payment_hash ||
      charge.hash ||
      null;

    if (!paymentRequest) {
      throw new Error('OpenNode create failed: missing lightning_invoice.payreq');
    }
    if (!paymentHash) {
      throw new Error('OpenNode backend does not expose a Lightning payment hash; cannot verify LSAT preimage. Use phoenixd or lnd.');
    }

    return {
      paymentHash: paymentHash,
      paymentRequest: paymentRequest,
      expiresAt: Date.now() + (expirySecs * 1000)
    };
  }

  async checkInvoice(chargeId) {
    const response = await fetch(`${this.baseUrl}/charge/${chargeId}`, {
      headers: { 'Authorization': this.apiKey }
    });

    if (!response.ok) {
      return { paid: false };
    }

    const data = await response.json();
    return {
      paid: data.data.status === 'paid',
      preimage: data.data.lightning_invoice?.settled_at ? chargeId : undefined, // OpenNode doesn't expose preimage
      settledAt: data.data.lightning_invoice?.settled_at
    };
  }

  async getStatus() {
    try {
      const response = await fetch(`${this.baseUrl}/account/balance`, {
        headers: { 'Authorization': this.apiKey }
      });
      if (!response.ok) throw new Error('Not reachable');
      return { ok: true, backend: 'opennode' };
    } catch (e) {
      return { ok: false, backend: 'opennode', error: e.message };
    }
  }
}

// =============================================================================
// MOCK PROVIDER (Testing/Demo)
// =============================================================================

class MockProvider extends LightningProvider {
  constructor() {
    super();
    this.invoices = new Map();
    this.name = 'mock';
  }

  async createInvoice(amountSats, memo, expirySecs = 3600) {
    const paymentHash = crypto.randomBytes(32).toString('hex');
    const preimage = crypto.randomBytes(32).toString('hex');
    
    // Store for later "payment"
    this.invoices.set(paymentHash, {
      preimage,
      amountSats,
      memo,
      paid: false,
      expiresAt: Date.now() + (expirySecs * 1000)
    });

    // Generate a fake bolt11 invoice
    const paymentRequest = `lnbc${amountSats}n1mock${paymentHash.substring(0, 40)}`;

    return {
      paymentHash,
      paymentRequest,
      expiresAt: Date.now() + (expirySecs * 1000),
      _mockPreimage: preimage // For testing: auto-pay by including preimage
    };
  }

  async checkInvoice(paymentHash) {
    const invoice = this.invoices.get(paymentHash);
    if (!invoice) {
      return { paid: false };
    }
    return {
      paid: invoice.paid,
      preimage: invoice.paid ? invoice.preimage : undefined,
      settledAt: invoice.settledAt
    };
  }

  // Mock-only: simulate payment
  simulatePayment(paymentHash) {
    const invoice = this.invoices.get(paymentHash);
    if (invoice) {
      invoice.paid = true;
      invoice.settledAt = Date.now();
      return invoice.preimage;
    }
    return null;
  }

  async getStatus() {
    return { ok: true, backend: 'mock', note: 'Mock provider for testing' };
  }
}

// =============================================================================
// PROVIDER FACTORY
// =============================================================================

function createLightningProvider(config = {}) {
  const backend = config.backend || process.env.LIGHTNING_BACKEND || 'mock';
  
  switch (backend.toLowerCase()) {
    case 'phoenixd':
      return new PhoenixdProvider(config);
    case 'lnd':
      return new LndProvider(config);
    case 'opennode':
      return new OpenNodeProvider(config);
    case 'mock':
    default:
      return new MockProvider();
  }
}

// -----------------------------------------------------------------------------
// Backwards-compatible helpers (older code expected these)
// -----------------------------------------------------------------------------
let _cachedProvider = null;

function getProvider(config = {}) {
  if (_cachedProvider && !config.forceNew) return _cachedProvider;
  _cachedProvider = createLightningProvider({
    backend: config.backend || process.env.LIGHTNING_BACKEND || 'mock',
    url: config.url || process.env.LIGHTNING_URL || process.env.PHOENIXD_URL || process.env.LND_REST_URL,
    password: config.password || process.env.PHOENIXD_PASSWORD,
    macaroon: config.macaroon || process.env.LND_MACAROON,
    apiKey: config.apiKey || process.env.OPENNODE_API_KEY,
  });
  return _cachedProvider;
}

function isApertureMode() {
  return (process.env.L402_MODE || 'aperture') !== 'native';
}

module.exports = {
  LightningProvider,
  PhoenixdProvider,
  LndProvider,
  OpenNodeProvider,
  MockProvider,
  createLightningProvider,
  getProvider,
  isApertureMode,
};

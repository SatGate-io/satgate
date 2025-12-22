/**
 * SatGate Lightning Provider: LND (REST API)
 * 
 * Direct connection to LND via REST API.
 * Use this when you have direct access to LND (not via LNC).
 * 
 * For LNC-based connections, continue using Aperture.
 */

const { LightningProvider } = require('./provider');

class LNDProvider extends LightningProvider {
  /**
   * @param {Object} config
   * @param {string} config.url - LND REST URL (e.g., https://localhost:8080)
   * @param {string} config.macaroon - Admin macaroon (hex)
   * @param {string} [config.cert] - TLS cert (base64, optional for local)
   */
  constructor(config = {}) {
    super(config);
    this.name = 'lnd';
    this.baseUrl = config.url || 'https://localhost:8080';
    this.macaroon = config.macaroon || '';
    this.cert = config.cert || null;
  }

  /**
   * Make authenticated request to LND REST API
   */
  async _request(method, path, body = null) {
    const url = `${this.baseUrl}${path}`;
    const headers = {
      'Grpc-Metadata-macaroon': this.macaroon,
      'Content-Type': 'application/json',
    };

    const options = { 
      method, 
      headers,
      // Skip TLS verification for local dev (configure properly in prod)
      ...(this.cert ? {} : { rejectUnauthorized: false }),
    };
    
    if (body && method !== 'GET') {
      options.body = JSON.stringify(body);
    }

    const response = await fetch(url, options);
    
    if (!response.ok) {
      const error = await response.text();
      throw new Error(`LND error: ${response.status} - ${error}`);
    }

    return response.json();
  }

  /**
   * Create a Lightning invoice via LND
   */
  async createInvoice(amountSats, memo, expiry = 3600) {
    const result = await this._request('POST', '/v1/invoices', {
      value: amountSats.toString(),
      memo: memo,
      expiry: expiry.toString(),
    });

    // LND returns payment_request and r_hash (base64)
    const paymentHash = Buffer.from(result.r_hash, 'base64').toString('hex');

    return {
      paymentRequest: result.payment_request,
      paymentHash: paymentHash,
      amountSats: amountSats,
      expiry: expiry,
      memo: memo,
    };
  }

  /**
   * Check payment status via LND
   */
  async checkPayment(paymentHash) {
    try {
      // Convert hex to base64 URL-safe
      const hashBase64 = Buffer.from(paymentHash, 'hex')
        .toString('base64')
        .replace(/\+/g, '-')
        .replace(/\//g, '_');
      
      const result = await this._request('GET', `/v1/invoice/${hashBase64}`);
      
      const paid = result.state === 'SETTLED' || result.settled === true;
      
      return {
        paid: paid,
        preimage: paid ? Buffer.from(result.r_preimage, 'base64').toString('hex') : null,
        settledAt: result.settle_date ? parseInt(result.settle_date) : null,
      };
    } catch (error) {
      if (error.message.includes('404')) {
        return { paid: false };
      }
      throw error;
    }
  }

  /**
   * Get LND node status
   */
  async getStatus() {
    try {
      const info = await this._request('GET', '/v1/getinfo');
      return {
        ok: true,
        info: {
          nodeId: info.identity_pubkey,
          alias: info.alias,
          channels: info.num_active_channels,
          version: info.version,
          synced: info.synced_to_chain,
        },
      };
    } catch (error) {
      return {
        ok: false,
        info: { error: error.message },
      };
    }
  }

  async getNodePubkey() {
    const info = await this._request('GET', '/v1/getinfo');
    return info.identity_pubkey;
  }
}

module.exports = { LNDProvider };


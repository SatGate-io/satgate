/**
 * SatGate Lightning Provider: phoenixd
 * 
 * phoenixd is ACINQ's headless Lightning daemon with automatic liquidity.
 * https://github.com/ACINQ/phoenixd
 * 
 * Benefits:
 * - Zero liquidity management (LSP model)
 * - Self-custodial
 * - Simple REST API
 * 
 * Trade-off:
 * - ~1% fee to ACINQ LSP per payment
 */

const { LightningProvider } = require('./provider');

class PhoenixdProvider extends LightningProvider {
  /**
   * @param {Object} config
   * @param {string} config.url - phoenixd API URL (default: http://localhost:9740)
   * @param {string} config.password - HTTP Basic auth password
   */
  constructor(config = {}) {
    super(config);
    this.name = 'phoenixd';
    this.baseUrl = config.url || 'http://localhost:9740';
    this.password = config.password || '';
  }

  /**
   * Make authenticated request to phoenixd
   */
  async _request(method, path, body = null) {
    const url = `${this.baseUrl}${path}`;
    const headers = {
      'Authorization': `Basic ${Buffer.from(`:${this.password}`).toString('base64')}`,
      'Content-Type': 'application/x-www-form-urlencoded',
    };

    const options = { method, headers };
    
    if (body && method !== 'GET') {
      options.body = new URLSearchParams(body).toString();
    }

    const response = await fetch(url, options);
    
    if (!response.ok) {
      const error = await response.text();
      throw new Error(`phoenixd error: ${response.status} - ${error}`);
    }

    return response.json();
  }

  /**
   * Create a Lightning invoice via phoenixd
   */
  async createInvoice(amountSats, memo, expiry = 3600) {
    const result = await this._request('POST', '/createinvoice', {
      amountSat: amountSats,
      description: memo,
      expirySeconds: expiry,
    });

    return {
      paymentRequest: result.serialized,
      paymentHash: result.paymentHash,
      amountSats: amountSats,
      expiry: expiry,
      memo: memo,
    };
  }

  /**
   * Check payment status via phoenixd
   */
  async checkPayment(paymentHash) {
    try {
      const result = await this._request('GET', `/payments/incoming/${paymentHash}`);
      
      return {
        paid: result.isPaid === true,
        preimage: result.preimage || null,
        settledAt: result.completedAt ? Math.floor(result.completedAt / 1000) : null,
      };
    } catch (error) {
      // Payment not found = not paid
      if (error.message.includes('404')) {
        return { paid: false };
      }
      throw error;
    }
  }

  /**
   * Get phoenixd node status
   */
  async getStatus() {
    try {
      const info = await this._request('GET', '/getinfo');
      return {
        ok: true,
        info: {
          nodeId: info.nodeId,
          channels: info.channels?.length || 0,
          version: info.version,
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
    const info = await this._request('GET', '/getinfo');
    return info.nodeId;
  }
}

module.exports = { PhoenixdProvider };



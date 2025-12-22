/**
 * SatGate Lightning Provider: OpenNode
 * 
 * OpenNode is a custodial Lightning payment processor.
 * Easiest setup - just need an API key.
 * 
 * Benefits:
 * - Zero setup (no node required)
 * - Instant liquidity
 * - Fiat settlement option
 * 
 * Trade-offs:
 * - Custodial (they hold funds until withdrawal)
 * - 1% fee per payment
 * - Requires KYC for business accounts
 * 
 * https://www.opennode.com/
 */

const { LightningProvider } = require('./provider');

class OpenNodeProvider extends LightningProvider {
  /**
   * @param {Object} config
   * @param {string} config.apiKey - OpenNode API key
   * @param {string} [config.environment] - 'live' or 'dev' (default: live)
   */
  constructor(config = {}) {
    super(config);
    this.name = 'opennode';
    this.apiKey = config.apiKey || '';
    this.environment = config.environment || 'live';
    this.baseUrl = this.environment === 'dev' 
      ? 'https://dev-api.opennode.com'
      : 'https://api.opennode.com';
  }

  /**
   * Make authenticated request to OpenNode API
   */
  async _request(method, path, body = null) {
    const url = `${this.baseUrl}${path}`;
    const headers = {
      'Authorization': this.apiKey,
      'Content-Type': 'application/json',
    };

    const options = { method, headers };
    
    if (body && method !== 'GET') {
      options.body = JSON.stringify(body);
    }

    const response = await fetch(url, options);
    const data = await response.json();
    
    if (!response.ok || data.success === false) {
      throw new Error(`OpenNode error: ${data.message || response.status}`);
    }

    return data.data;
  }

  /**
   * Create a Lightning invoice via OpenNode
   */
  async createInvoice(amountSats, memo, expiry = 3600) {
    const result = await this._request('POST', '/v1/charges', {
      amount: amountSats,
      description: memo,
      currency: 'BTC',
      callback_url: this.config.callbackUrl || null,
      auto_settle: false, // Keep in BTC
    });

    return {
      paymentRequest: result.lightning_invoice.payreq,
      paymentHash: result.id, // OpenNode uses charge ID
      amountSats: amountSats,
      expiry: expiry,
      memo: memo,
      // OpenNode-specific
      chargeId: result.id,
    };
  }

  /**
   * Check payment status via OpenNode
   * Note: OpenNode uses charge IDs, not payment hashes
   */
  async checkPayment(chargeId) {
    try {
      const result = await this._request('GET', `/v1/charge/${chargeId}`);
      
      const paid = result.status === 'paid';
      
      return {
        paid: paid,
        preimage: paid ? result.lightning_invoice?.settled_at : null, // OpenNode doesn't expose preimage
        settledAt: result.lightning_invoice?.settled_at 
          ? Math.floor(new Date(result.lightning_invoice.settled_at).getTime() / 1000)
          : null,
      };
    } catch (error) {
      if (error.message.includes('404')) {
        return { paid: false };
      }
      throw error;
    }
  }

  /**
   * Get OpenNode account status
   */
  async getStatus() {
    try {
      const result = await this._request('GET', '/v1/account/balance');
      return {
        ok: true,
        info: {
          provider: 'OpenNode',
          environment: this.environment,
          balance: result.balance,
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
    // OpenNode is custodial, no node pubkey
    return null;
  }
}

module.exports = { OpenNodeProvider };


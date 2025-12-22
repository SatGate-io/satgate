/**
 * SatGate Lightning Provider Interface
 * 
 * Abstract interface for Lightning backends.
 * Implementations: LND, phoenixd, OpenNode, Alby, etc.
 */

/**
 * @typedef {Object} Invoice
 * @property {string} paymentRequest - BOLT11 invoice string
 * @property {string} paymentHash - Payment hash (hex)
 * @property {number} amountSats - Amount in satoshis
 * @property {number} expiry - Expiry in seconds
 * @property {string} memo - Invoice description
 */

/**
 * @typedef {Object} PaymentStatus
 * @property {boolean} paid - Whether the invoice has been paid
 * @property {string} [preimage] - Payment preimage (hex) if paid
 * @property {number} [settledAt] - Unix timestamp when settled
 */

/**
 * Base class for Lightning providers.
 * Extend this class to add support for new backends.
 */
class LightningProvider {
  constructor(config = {}) {
    this.config = config;
    this.name = 'base';
  }

  /**
   * Create a Lightning invoice
   * @param {number} amountSats - Amount in satoshis
   * @param {string} memo - Invoice description
   * @param {number} [expiry=3600] - Expiry in seconds
   * @returns {Promise<Invoice>}
   */
  async createInvoice(amountSats, memo, expiry = 3600) {
    throw new Error('createInvoice must be implemented by subclass');
  }

  /**
   * Check if an invoice has been paid
   * @param {string} paymentHash - Payment hash (hex)
   * @returns {Promise<PaymentStatus>}
   */
  async checkPayment(paymentHash) {
    throw new Error('checkPayment must be implemented by subclass');
  }

  /**
   * Get provider health/connectivity status
   * @returns {Promise<{ok: boolean, info: Object}>}
   */
  async getStatus() {
    throw new Error('getStatus must be implemented by subclass');
  }

  /**
   * Get node public key (if applicable)
   * @returns {Promise<string|null>}
   */
  async getNodePubkey() {
    return null;
  }
}

module.exports = { LightningProvider };


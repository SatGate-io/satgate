/**
 * SatGate L402 Protocol Handler
 * 
 * Implements the L402 authentication protocol:
 * 1. Client requests paid resource
 * 2. Server returns 402 with WWW-Authenticate header containing:
 *    - Lightning invoice (BOLT11)
 *    - Macaroon (capability token)
 * 3. Client pays invoice, receives preimage
 * 4. Client sends Authorization header with macaroon:preimage
 * 5. Server validates and grants access
 * 
 * This module provides native L402 when not using Aperture.
 */

const crypto = require('crypto');
const { getProvider, isApertureMode } = require('./index');

// In-memory invoice tracking (use Redis in production)
const pendingInvoices = new Map();

/**
 * Generate a random root key for macaroon signing
 * In production, this should be stored persistently
 */
const ROOT_KEY = process.env.L402_ROOT_KEY 
  ? Buffer.from(process.env.L402_ROOT_KEY, 'hex')
  : crypto.randomBytes(32);

/**
 * Create a simple macaroon-like token
 * Note: This is a simplified implementation. Production should use
 * the full macaroon library with proper HMAC chains.
 * 
 * @param {string} paymentHash - Payment hash to bind
 * @param {Object} caveats - Constraints {expires, tier, scope}
 * @returns {string} Base64-encoded token
 */
function createMacaroon(paymentHash, caveats = {}) {
  const payload = {
    version: 1,
    paymentHash,
    caveats: {
      expires: caveats.expires || Date.now() + 86400000, // 24h default
      tier: caveats.tier || 'micro',
      scope: caveats.scope || '*',
    },
    issuedAt: Date.now(),
  };

  // Create signature
  const data = JSON.stringify(payload);
  const signature = crypto
    .createHmac('sha256', ROOT_KEY)
    .update(data)
    .digest('hex');

  const token = {
    ...payload,
    signature,
  };

  return Buffer.from(JSON.stringify(token)).toString('base64');
}

/**
 * Verify a macaroon and preimage
 * @param {string} macaroon - Base64-encoded macaroon
 * @param {string} preimage - Payment preimage (hex)
 * @returns {{valid: boolean, error?: string, payload?: Object}}
 */
function verifyL402(macaroon, preimage) {
  try {
    // Decode macaroon
    const tokenJson = Buffer.from(macaroon, 'base64').toString('utf8');
    const token = JSON.parse(tokenJson);

    // Verify signature
    const { signature, ...payload } = token;
    const expectedSig = crypto
      .createHmac('sha256', ROOT_KEY)
      .update(JSON.stringify(payload))
      .digest('hex');

    if (signature !== expectedSig) {
      return { valid: false, error: 'Invalid macaroon signature' };
    }

    // Verify preimage matches payment hash
    const preimageHash = crypto
      .createHash('sha256')
      .update(Buffer.from(preimage, 'hex'))
      .digest('hex');

    if (preimageHash !== payload.paymentHash) {
      return { valid: false, error: 'Preimage does not match payment hash' };
    }

    // Check expiry
    if (payload.caveats.expires < Date.now()) {
      return { valid: false, error: 'Token expired' };
    }

    return { valid: true, payload };
  } catch (error) {
    return { valid: false, error: `Verification failed: ${error.message}` };
  }
}

/**
 * Parse L402 Authorization header
 * Format: L402 <macaroon>:<preimage>
 * Legacy: LSAT <macaroon>:<preimage>
 * @param {string} authHeader
 * @returns {{macaroon: string, preimage: string}|null}
 */
function parseL402Header(authHeader) {
  if (!authHeader) return null;

  const match = authHeader.match(/^(?:L402|LSAT)\s+([^:]+):([a-fA-F0-9]+)$/i);
  if (!match) return null;

  return {
    macaroon: match[1],
    preimage: match[2],
  };
}

/**
 * Create L402 challenge response (HTTP 402)
 * @param {number} priceSats - Price in satoshis
 * @param {string} tier - Pricing tier name
 * @returns {Promise<{invoice: string, macaroon: string, paymentHash: string}>}
 */
async function createL402Challenge(priceSats, tier = 'micro') {
  const provider = getProvider();
  
  if (isApertureMode()) {
    throw new Error('L402 challenges handled by Aperture in current configuration');
  }

  // Create invoice
  const memo = `SatGate API Access - ${tier} tier`;
  const invoice = await provider.createInvoice(priceSats, memo, 3600);

  // Create macaroon bound to this payment
  const macaroon = createMacaroon(invoice.paymentHash, {
    tier,
    expires: Date.now() + 86400000, // 24 hours
  });

  // Track pending invoice
  pendingInvoices.set(invoice.paymentHash, {
    invoice: invoice.paymentRequest,
    macaroon,
    priceSats,
    tier,
    createdAt: Date.now(),
  });

  // Cleanup old invoices (>2 hours)
  const twoHoursAgo = Date.now() - 7200000;
  for (const [hash, data] of pendingInvoices) {
    if (data.createdAt < twoHoursAgo) {
      pendingInvoices.delete(hash);
    }
  }

  return {
    invoice: invoice.paymentRequest,
    macaroon,
    paymentHash: invoice.paymentHash,
  };
}

/**
 * Express middleware for L402 authentication
 * Use this when running without Aperture
 * 
 * @param {Object} options
 * @param {number} options.price - Price in satoshis
 * @param {string} options.tier - Tier name
 * @returns {Function} Express middleware
 */
function l402Middleware(options = {}) {
  const { price = 1, tier = 'micro' } = options;

  return async (req, res, next) => {
    // Skip if Aperture is handling L402
    if (isApertureMode()) {
      return next();
    }

    // Check for L402 token
    const authHeader = req.headers.authorization;
    const l402 = parseL402Header(authHeader);

    if (l402) {
      // Verify the token
      const result = verifyL402(l402.macaroon, l402.preimage);
      
      if (result.valid) {
        // Attach L402 info to request
        req.l402 = result.payload;
        return next();
      }

      // Invalid token
      return res.status(401).json({
        error: 'Invalid L402 token',
        details: result.error,
      });
    }

    // No token - issue challenge
    try {
      const challenge = await createL402Challenge(price, tier);
      
      res.status(402);
      res.setHeader(
        'WWW-Authenticate',
        `L402 macaroon="${challenge.macaroon}", invoice="${challenge.invoice}"`
      );
      
      return res.json({
        error: 'Payment Required',
        price: price,
        tier: tier,
        invoice: challenge.invoice,
        macaroon: challenge.macaroon,
      });
    } catch (error) {
      console.error('[L402] Challenge creation failed:', error);
      return res.status(500).json({
        error: 'Failed to create payment challenge',
        details: error.message,
      });
    }
  };
}

/**
 * Check if a payment has been received
 * @param {string} paymentHash
 * @returns {Promise<boolean>}
 */
async function checkPaymentReceived(paymentHash) {
  const provider = getProvider();
  if (isApertureMode()) {
    return false; // Aperture handles this
  }
  
  const status = await provider.checkPayment(paymentHash);
  return status.paid;
}

module.exports = {
  createL402Challenge,
  verifyL402,
  parseL402Header,
  l402Middleware,
  checkPaymentReceived,
  createMacaroon,
};


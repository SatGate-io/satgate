/**
 * SatGate L402 Implementation
 * 
 * This module makes SatGate the L402 authority:
 * - Issues 402 challenges with WWW-Authenticate: L402
 * - Validates Authorization: LSAT <macaroon>:<preimage>
 * - Enforces per-request metering via max_calls and budget_sats
 * - Re-challenges when budget/calls exhausted
 * 
 * L402 Protocol:
 *   1. Client requests paid endpoint
 *   2. Server returns 402 with WWW-Authenticate: L402 macaroon="...", invoice="..."
 *   3. Client pays invoice, receives preimage
 *   4. Client retries with Authorization: LSAT <macaroon>:<preimage>
 *   5. Server validates macaroon signature + preimage + caveats
 *   6. If max_calls/budget_sats exhausted, goto step 2 (re-challenge)
 */

const crypto = require('crypto');
const macaroon = require('macaroon');
const { createLightningProvider } = require('../lightning');

// =============================================================================
// L402 CONSTANTS
// =============================================================================

const L402_VERSION = '0';
const MACAROON_LOCATION = 'https://satgate.io';
const MACAROON_IDENTIFIER_PREFIX = 'satgate-l402-v1';

// =============================================================================
// L402 SERVICE
// =============================================================================

class L402Service {
  constructor(config = {}) {
    this.rootKey = config.rootKey || process.env.L402_ROOT_KEY || process.env.CAPABILITY_ROOT_KEY || 'satgate-l402-demo-key-change-in-prod';
    this.lightning = config.lightning || createLightningProvider(config.lightningConfig);
    this.redis = config.redis || null;
    
    // In-memory fallback for metering (single-instance only)
    this.callsMemory = new Map();
    this.budgetMemory = new Map();
    
    // Tier pricing (sats per request)
    this.tierPrices = config.tierPrices || {
      'micro': 1,
      'basic': 10,
      'standard': 100,
      'premium': 1000,
      'default': 10
    };
    
    // Default caveats for new tokens
    this.defaultTTL = config.defaultTTL || 3600; // 1 hour
    this.defaultMaxCalls = config.defaultMaxCalls || 100;
    this.defaultBudgetSats = config.defaultBudgetSats || null; // Optional
  }

  /**
   * Get the price for a tier
   */
  getTierPrice(tier) {
    return this.tierPrices[tier] || this.tierPrices['default'];
  }

  /**
   * Create a new L402 challenge (402 response)
   * @param {string} tier - The pricing tier (micro, basic, standard, premium)
   * @param {object} options - Additional options (scope, ttl, maxCalls, budgetSats)
   * @returns {Promise<{statusCode: number, headers: object, body: object}>}
   */
  async createChallenge(tier = 'basic', options = {}) {
    const price = this.getTierPrice(tier);
    const ttl = options.ttl || this.defaultTTL;
    const maxCalls = options.maxCalls ?? this.defaultMaxCalls;
    const budgetSats = options.budgetSats ?? this.defaultBudgetSats;
    const scope = options.scope || `api:${tier}:*`;

    // Create invoice
    const memo = `SatGate ${tier} access - ${maxCalls ? maxCalls + ' calls' : 'unlimited'} for ${ttl}s`;
    const invoice = await this.lightning.createInvoice(price, memo, Math.min(ttl, 600)); // Invoice expires in max 10 min

    // Create macaroon with payment hash caveat
    // Use shorter identifier to avoid macaroon library overflow issues
    const timestamp = Date.now().toString(36); // Base36 timestamp for brevity
    const hashPrefix = invoice.paymentHash.substring(0, 16); // First 16 chars of hash
    const identifier = `sg:${hashPrefix}:${timestamp}`;
    
    // Use TextEncoder for clean Uint8Array conversion (macaroon lib bug workaround)
    const encoder = new TextEncoder();
    const keyBytes = encoder.encode(this.rootKey.substring(0, 32)); // Use first 32 chars as key
    const idBytes = encoder.encode(identifier);
    
    console.log(`[L402] Creating macaroon: id=${identifier}, keyLen=${keyBytes.length}`);
    
    let m = macaroon.newMacaroon({
      identifier: idBytes,
      location: MACAROON_LOCATION, // location must be string, not Uint8Array
      rootKey: keyBytes
    });

    // Add caveats - keep them short and ASCII-safe
    const expiresAt = Date.now() + (ttl * 1000);
    // Use only first 16 chars of payment hash to keep caveat short
    const shortHash = invoice.paymentHash.substring(0, 16);
    
    const caveats = [
      `ph=${shortHash}`,
      `exp=${expiresAt}`,
      `scope=${scope}`,
      `tier=${tier}`
    ];
    
    if (maxCalls) caveats.push(`mc=${maxCalls}`);
    if (budgetSats) caveats.push(`bs=${budgetSats}`);
    
    for (const caveat of caveats) {
      console.log(`[L402] Adding caveat: ${caveat}`);
      m.addFirstPartyCaveat(encoder.encode(caveat));
    }

    console.log(`[L402] Exporting macaroon...`);
    let macaroonBase64;
    try {
      const binary = m.exportBinary();
      console.log(`[L402] Export success, binary length: ${binary.length}, type: ${binary.constructor.name}`);
      // Convert Uint8Array to base64 (Node.js compatible)
      macaroonBase64 = Buffer.from(binary).toString('base64');
    } catch (exportErr) {
      console.error(`[L402] Export failed: ${exportErr.message}`);
      throw exportErr;
    }

    // Build WWW-Authenticate header (L402 format)
    const wwwAuth = `L402 macaroon="${macaroonBase64}", invoice="${invoice.paymentRequest}"`;

    return {
      statusCode: 402,
      headers: {
        'WWW-Authenticate': wwwAuth,
        'X-L402-Price': String(price),
        'X-L402-Tier': tier,
        'X-L402-TTL': String(ttl),
        'X-L402-Max-Calls': maxCalls ? String(maxCalls) : 'unlimited'
      },
      body: {
        error: 'Payment Required',
        code: 'PAYMENT_REQUIRED',
        message: `This endpoint requires ${price} sats. Pay the invoice and retry with the LSAT token.`,
        price: price,
        tier: tier,
        invoice: invoice.paymentRequest,
        paymentHash: invoice.paymentHash,
        macaroon: macaroonBase64,
        expiresAt: invoice.expiresAt,
        usage: {
          header: `Authorization: LSAT ${macaroonBase64}:<preimage>`,
          note: 'Replace <preimage> with the preimage received after paying the invoice'
        }
      }
    };
  }

  /**
   * Parse an LSAT Authorization header
   * @param {string} authHeader - The Authorization header value
   * @returns {{macaroon: string, preimage: string} | null}
   */
  parseLSATHeader(authHeader) {
    if (!authHeader) return null;
    
    // Support both "LSAT" and "L402" prefixes
    const match = authHeader.match(/^(?:LSAT|L402)\s+([^:]+):([a-fA-F0-9]+)$/i);
    if (!match) return null;
    
    return {
      macaroon: match[1],
      preimage: match[2]
    };
  }

  /**
   * Validate an LSAT token
   * @param {string} macaroonBase64 - The macaroon (base64)
   * @param {string} preimage - The payment preimage (hex)
   * @returns {Promise<{valid: boolean, error?: string, caveats?: object, tokenSignature?: string}>}
   */
  async validateLSAT(macaroonBase64, preimage) {
    try {
      // Decode macaroon
      const tokenBytes = Buffer.from(macaroonBase64, 'base64');
      const m = macaroon.importMacaroon(tokenBytes);
      
      // Extract payment hash from caveats
      let paymentHash = null;
      let expiresAt = null;
      let scope = null;
      let tier = null;
      let maxCalls = null;
      let budgetSats = null;
      
      const caveats = [];
      for (const caveat of m._exportAsJSONObjectV2().c || []) {
        const caveatStr = caveat.i ? Buffer.from(caveat.i, 'base64').toString('utf8') : '';
        caveats.push(caveatStr);
        
        if (caveatStr.startsWith('payment_hash = ')) {
          paymentHash = caveatStr.split(' = ')[1];
        } else if (caveatStr.startsWith('expires = ')) {
          expiresAt = parseInt(caveatStr.split(' = ')[1], 10);
        } else if (caveatStr.startsWith('scope = ')) {
          scope = caveatStr.split(' = ')[1];
        } else if (caveatStr.startsWith('tier = ')) {
          tier = caveatStr.split(' = ')[1];
        } else if (caveatStr.startsWith('max_calls = ')) {
          maxCalls = parseInt(caveatStr.split(' = ')[1], 10);
        } else if (caveatStr.startsWith('budget_sats = ')) {
          budgetSats = parseInt(caveatStr.split(' = ')[1], 10);
        }
      }

      // Verify preimage matches payment hash
      if (!paymentHash) {
        return { valid: false, error: 'Missing payment_hash caveat' };
      }
      
      if (!this.lightning.verifyPreimage(preimage, paymentHash)) {
        return { valid: false, error: 'Invalid preimage' };
      }

      // Check expiry
      if (expiresAt && Date.now() > expiresAt) {
        return { valid: false, error: 'Token expired' };
      }

      // Verify macaroon signature
      const keyBytes = Buffer.from(this.rootKey, 'utf8');
      const checkCaveat = (caveat) => {
        const caveatStr = typeof caveat === 'string' ? caveat : caveat.toString('utf8');
        
        // All our caveats are informational (checked above)
        // Accept known caveats
        if (caveatStr.startsWith('payment_hash = ') ||
            caveatStr.startsWith('expires = ') ||
            caveatStr.startsWith('scope = ') ||
            caveatStr.startsWith('tier = ') ||
            caveatStr.startsWith('max_calls = ') ||
            caveatStr.startsWith('budget_sats = ')) {
          return null; // OK
        }
        
        return 'unknown caveat: ' + caveatStr;
      };
      
      m.verify(keyBytes, checkCaveat);

      // Get token signature for metering key
      const tokenSignature = Buffer.from(m.signature).toString('hex');

      return {
        valid: true,
        caveats: { paymentHash, expiresAt, scope, tier, maxCalls, budgetSats, raw: caveats },
        tokenSignature
      };

    } catch (e) {
      return { valid: false, error: e.message };
    }
  }

  /**
   * Decrement call counter for a token
   * @returns {Promise<{remaining: number, exhausted: boolean, backend: string}>}
   */
  async decrementCalls(tokenSignature, maxCalls, expiresAtMs) {
    const now = Date.now();
    const ttlSeconds = expiresAtMs ? Math.max(1, Math.ceil((expiresAtMs - now) / 1000)) : 3600;

    if (this.redis) {
      const key = `satgate:l402:calls:${tokenSignature}`;
      const script = `
        local key = KEYS[1]
        local init = tonumber(ARGV[1])
        local ttl = tonumber(ARGV[2])
        if redis.call('EXISTS', key) == 0 then
          redis.call('SET', key, init, 'EX', ttl)
        end
        local remaining = redis.call('DECR', key)
        return remaining
      `;
      try {
        const remaining = await this.redis.eval(script, 1, key, String(maxCalls), String(ttlSeconds));
        return { remaining: Number(remaining), exhausted: remaining < 0, backend: 'redis' };
      } catch (e) {
        // Fall through to memory
      }
    }

    // In-memory fallback
    const entry = this.callsMemory.get(tokenSignature);
    if (!entry || (entry.expiresAtMs && entry.expiresAtMs < now)) {
      this.callsMemory.set(tokenSignature, { remaining: maxCalls, expiresAtMs: expiresAtMs || (now + ttlSeconds * 1000) });
    }
    const current = this.callsMemory.get(tokenSignature);
    current.remaining -= 1;
    return { remaining: current.remaining, exhausted: current.remaining < 0, backend: 'memory' };
  }

  /**
   * Decrement budget for a token (conditional - won't go negative)
   * @returns {Promise<{remaining: number, charged: boolean, cost: number, backend: string}>}
   */
  async decrementBudget(tokenSignature, budgetSats, cost, expiresAtMs) {
    const now = Date.now();
    const ttlSeconds = expiresAtMs ? Math.max(1, Math.ceil((expiresAtMs - now) / 1000)) : 3600;

    if (this.redis) {
      const key = `satgate:l402:budget:${tokenSignature}`;
      const script = `
        local key = KEYS[1]
        local init = tonumber(ARGV[1])
        local cost = tonumber(ARGV[2])
        local ttl = tonumber(ARGV[3])
        if redis.call('EXISTS', key) == 0 then
          redis.call('SET', key, init, 'EX', ttl)
        end
        local current = tonumber(redis.call('GET', key))
        if current == nil then return { -999999, 0 } end
        if current < cost then return { current, 0 } end
        local remaining = redis.call('DECRBY', key, cost)
        return { remaining, 1 }
      `;
      try {
        const result = await this.redis.eval(script, 1, key, String(budgetSats), String(cost), String(ttlSeconds));
        const remaining = Array.isArray(result) ? Number(result[0]) : Number(result);
        const charged = Array.isArray(result) ? Number(result[1]) === 1 : true;
        return { remaining, charged, cost, backend: 'redis' };
      } catch (e) {
        // Fall through to memory
      }
    }

    // In-memory fallback
    const entry = this.budgetMemory.get(tokenSignature);
    if (!entry || (entry.expiresAtMs && entry.expiresAtMs < now)) {
      this.budgetMemory.set(tokenSignature, { remaining: budgetSats, expiresAtMs: expiresAtMs || (now + ttlSeconds * 1000) });
    }
    const current = this.budgetMemory.get(tokenSignature);
    if (current.remaining < cost) {
      return { remaining: current.remaining, charged: false, cost, backend: 'memory' };
    }
    current.remaining -= cost;
    return { remaining: current.remaining, charged: true, cost, backend: 'memory' };
  }
}

// =============================================================================
// L402 MIDDLEWARE FACTORY
// =============================================================================

/**
 * Create L402 middleware for Express
 * @param {L402Service} l402Service - The L402 service instance
 * @param {object} options - Middleware options
 * @returns {Function} Express middleware
 */
function createL402Middleware(l402Service, options = {}) {
  const tier = options.tier || 'basic';
  const scope = options.scope || `api:${tier}:*`;
  const tierCost = options.tierCost || l402Service.getTierPrice(tier);
  const challengeOptions = {
    scope,
    ttl: options.ttl,
    maxCalls: options.maxCalls,
    budgetSats: options.budgetSats,
  };

  return async (req, res, next) => {
    const authHeader = req.get('authorization');
    const lsat = l402Service.parseLSATHeader(authHeader);

    // No LSAT token - issue challenge
    if (!lsat) {
      const challenge = await l402Service.createChallenge(tier, challengeOptions);
      
      for (const [key, value] of Object.entries(challenge.headers)) {
        res.setHeader(key, value);
      }
      return res.status(challenge.statusCode).json(challenge.body);
    }

    // Validate LSAT
    const validation = await l402Service.validateLSAT(lsat.macaroon, lsat.preimage);
    
    if (!validation.valid) {
      // Invalid token - issue new challenge
      console.log(`[L402] Invalid token: ${validation.error}`);
      const challenge = await l402Service.createChallenge(tier, challengeOptions);
      for (const [key, value] of Object.entries(challenge.headers)) {
        res.setHeader(key, value);
      }
      return res.status(challenge.statusCode).json({
        ...challenge.body,
        previousError: validation.error
      });
    }

    const { caveats, tokenSignature } = validation;

    // Check scope
    if (caveats.scope && !scopeMatches(caveats.scope, scope)) {
      return res.status(403).json({
        error: 'Scope violation',
        code: 'SCOPE_VIOLATION',
        message: `Token scope '${caveats.scope}' does not cover '${scope}'`
      });
    }

    // Enforce max_calls
    if (caveats.maxCalls) {
      const { remaining, exhausted } = await l402Service.decrementCalls(
        tokenSignature, caveats.maxCalls, caveats.expiresAt
      );
      
      res.setHeader('X-Calls-Limit', String(caveats.maxCalls));
      res.setHeader('X-Calls-Remaining', String(Math.max(0, remaining)));
      
      if (exhausted) {
        // Re-challenge
        console.log(`[L402] Calls exhausted, re-challenging`);
        const challenge = await l402Service.createChallenge(tier, challengeOptions);
        for (const [key, value] of Object.entries(challenge.headers)) {
          res.setHeader(key, value);
        }
        return res.status(challenge.statusCode).json({
          ...challenge.body,
          reason: 'Call limit exhausted'
        });
      }
    }

    // Enforce budget_sats
    if (caveats.budgetSats) {
      const { remaining, charged } = await l402Service.decrementBudget(
        tokenSignature, caveats.budgetSats, tierCost, caveats.expiresAt
      );
      
      res.setHeader('X-Budget-Limit', String(caveats.budgetSats));
      res.setHeader('X-Budget-Remaining', String(Math.max(0, remaining)));
      res.setHeader('X-Budget-Cost', String(tierCost));
      
      if (!charged) {
        // Re-challenge
        console.log(`[L402] Budget exhausted, re-challenging`);
        const challenge = await l402Service.createChallenge(tier, challengeOptions);
        for (const [key, value] of Object.entries(challenge.headers)) {
          res.setHeader(key, value);
        }
        return res.status(challenge.statusCode).json({
          ...challenge.body,
          reason: 'Budget exhausted'
        });
      }
    }

    // Attach L402 info to request
    req.l402 = {
      tier: caveats.tier,
      scope: caveats.scope,
      tokenSignature,
      paymentHash: caveats.paymentHash,
      expiresAt: caveats.expiresAt,
      maxCalls: caveats.maxCalls,
      budgetSats: caveats.budgetSats
    };

    next();
  };
}

/**
 * Check if a token scope covers the required scope
 */
function scopeMatches(tokenScope, requiredScope) {
  if (tokenScope === requiredScope) return true;
  if (tokenScope.endsWith(':*')) {
    const prefix = tokenScope.slice(0, -1); // Remove '*'
    return requiredScope.startsWith(prefix);
  }
  return false;
}

// =============================================================================
// EXPORTS
// =============================================================================

module.exports = {
  L402Service,
  createL402Middleware
};


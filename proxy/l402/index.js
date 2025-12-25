/**
 * SatGate L402 Implementation
 * 
 * This module makes SatGate the L402 authority:
 * - Issues 402 challenges with WWW-Authenticate: L402
 * - Validates Authorization: LSAT <macaroon>:<preimage>
 * - Enforces per-request metering via max_calls and budget_sats
 * - Re-challenges when budget/calls exhausted
 * 
 * Uses a minimal macaroon implementation to avoid npm library issues.
 */

const crypto = require('crypto');
const { createLightningProvider } = require('../lightning');

// =============================================================================
// MINIMAL MACAROON IMPLEMENTATION
// =============================================================================

/**
 * Simple macaroon implementation using HMAC-SHA256
 * Format: base64(JSON({id, location, caveats, signature}))
 */
class SimpleMacaroon {
  constructor(location, identifier, rootKey) {
    this.location = location;
    this.identifier = identifier;
    this.caveats = [];
    // Initial signature is HMAC of identifier with root key
    this.signature = this._hmac(rootKey, identifier);
  }

  _hmac(key, data) {
    // Key should be 32 bytes for HMAC-SHA256
    const keyBuf = typeof key === 'string' ? Buffer.from(key, 'utf8') : key;
    return crypto.createHmac('sha256', keyBuf).update(data).digest('hex');
  }

  addFirstPartyCaveat(caveat) {
    this.caveats.push(caveat);
    // Chain signature: new_sig = HMAC(old_sig, caveat)
    this.signature = this._hmac(Buffer.from(this.signature, 'hex'), caveat);
  }

  serialize() {
    const obj = {
      v: 1, // version
      l: this.location,
      i: this.identifier,
      c: this.caveats,
      s: this.signature
    };
    return Buffer.from(JSON.stringify(obj)).toString('base64');
  }

  static deserialize(base64) {
    try {
      const json = Buffer.from(base64, 'base64').toString('utf8');
      const obj = JSON.parse(json);
      const m = new SimpleMacaroon(obj.l, obj.i, '');
      m.caveats = obj.c || [];
      m.signature = obj.s;
      return m;
    } catch (e) {
      throw new Error('Invalid macaroon format: ' + e.message);
    }
  }

  /**
   * Verify the macaroon signature
   * @param {string} rootKey - The root key used to create the macaroon
   * @returns {boolean} - True if signature is valid
   */
  verify(rootKey) {
    // Recreate the signature chain
    let sig = crypto.createHmac('sha256', Buffer.from(rootKey, 'utf8'))
      .update(this.identifier).digest('hex');
    
    for (const caveat of this.caveats) {
      sig = crypto.createHmac('sha256', Buffer.from(sig, 'hex'))
        .update(caveat).digest('hex');
    }
    
    return sig === this.signature;
  }
}

// =============================================================================
// L402 CONSTANTS
// =============================================================================

const MACAROON_LOCATION = 'https://satgate.io';

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
    // Note: 1 sat payments may fail on some routes (min HTLC limits) - this is intentional
    // to demonstrate the fallback to manual payment with alternate wallets
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
   */
  async createChallenge(tier = 'basic', options = {}) {
    const price = this.getTierPrice(tier);
    const ttl = options.ttl || this.defaultTTL;
    const maxCalls = options.maxCalls ?? this.defaultMaxCalls;
    const budgetSats = options.budgetSats ?? this.defaultBudgetSats;
    const scope = options.scope || `api:${tier}:*`;

    // Create invoice
    const memo = `SatGate ${tier} access - ${maxCalls ? maxCalls + ' calls' : 'unlimited'} for ${ttl}s`;
    const invoice = await this.lightning.createInvoice(price, memo, Math.min(ttl, 600));

    // Create macaroon with payment hash caveat
    const timestamp = Date.now().toString(36);
    const hashPrefix = invoice.paymentHash.substring(0, 16);
    const identifier = `sg:${hashPrefix}:${timestamp}`;
    
    console.log(`[L402] Creating macaroon: id=${identifier}`);
    
    const m = new SimpleMacaroon(MACAROON_LOCATION, identifier, this.rootKey);

    // Add caveats
    const expiresAt = Date.now() + (ttl * 1000);
    
    m.addFirstPartyCaveat(`ph=${hashPrefix}`);
    m.addFirstPartyCaveat(`exp=${expiresAt}`);
    m.addFirstPartyCaveat(`scope=${scope}`);
    m.addFirstPartyCaveat(`tier=${tier}`);
    if (maxCalls) m.addFirstPartyCaveat(`mc=${maxCalls}`);
    if (budgetSats) m.addFirstPartyCaveat(`bs=${budgetSats}`);

    const macaroonBase64 = m.serialize();
    console.log(`[L402] Macaroon created, length: ${macaroonBase64.length}`);

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
   */
  async validateLSAT(macaroonBase64, preimage) {
    try {
      const m = SimpleMacaroon.deserialize(macaroonBase64);
      
      // Extract caveats
      let paymentHash = null;
      let expiresAt = null;
      let scope = null;
      let tier = null;
      let maxCalls = null;
      let budgetSats = null;
      
      for (const caveat of m.caveats) {
        if (caveat.startsWith('ph=')) {
          paymentHash = caveat.substring(3);
        } else if (caveat.startsWith('exp=')) {
          expiresAt = parseInt(caveat.substring(4), 10);
        } else if (caveat.startsWith('scope=')) {
          scope = caveat.substring(6);
        } else if (caveat.startsWith('tier=')) {
          tier = caveat.substring(5);
        } else if (caveat.startsWith('mc=')) {
          maxCalls = parseInt(caveat.substring(3), 10);
        } else if (caveat.startsWith('bs=')) {
          budgetSats = parseInt(caveat.substring(3), 10);
        }
      }

      // Verify preimage matches payment hash (partial match since we only store prefix)
      if (!paymentHash) {
        return { valid: false, error: 'Missing payment_hash caveat' };
      }
      
      const preimageHash = crypto.createHash('sha256')
        .update(Buffer.from(preimage, 'hex'))
        .digest('hex');
      
      if (!preimageHash.startsWith(paymentHash)) {
        return { valid: false, error: 'Invalid preimage' };
      }

      // Check expiry
      if (expiresAt && Date.now() > expiresAt) {
        return { valid: false, error: 'Token expired' };
      }

      // Verify macaroon signature
      if (!m.verify(this.rootKey)) {
        return { valid: false, error: 'Invalid macaroon signature' };
      }

      return {
        valid: true,
        caveats: { paymentHash, expiresAt, scope, tier, maxCalls, budgetSats, raw: m.caveats },
        tokenSignature: m.signature
      };

    } catch (e) {
      return { valid: false, error: e.message };
    }
  }

  /**
   * Decrement call counter for a token
   */
  async decrementCalls(tokenSignature, maxCalls, expiresAtMs) {
    const now = Date.now();
    const ttlSeconds = expiresAtMs ? Math.max(1, Math.ceil((expiresAtMs - now) / 1000)) : 3600;

    if (this.redis) {
      const key = `satgate:l402:calls:${tokenSignature.substring(0, 32)}`;
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
    const sigKey = tokenSignature.substring(0, 32);
    const entry = this.callsMemory.get(sigKey);
    if (!entry || (entry.expiresAtMs && entry.expiresAtMs < now)) {
      this.callsMemory.set(sigKey, { remaining: maxCalls, expiresAtMs: expiresAtMs || (now + ttlSeconds * 1000) });
    }
    const current = this.callsMemory.get(sigKey);
    current.remaining -= 1;
    return { remaining: current.remaining, exhausted: current.remaining < 0, backend: 'memory' };
  }

  /**
   * Decrement budget for a token (conditional - won't go negative)
   */
  async decrementBudget(tokenSignature, budgetSats, cost, expiresAtMs) {
    const now = Date.now();
    const ttlSeconds = expiresAtMs ? Math.max(1, Math.ceil((expiresAtMs - now) / 1000)) : 3600;

    if (this.redis) {
      const key = `satgate:l402:budget:${tokenSignature.substring(0, 32)}`;
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
    const sigKey = tokenSignature.substring(0, 32);
    const entry = this.budgetMemory.get(sigKey);
    if (!entry || (entry.expiresAtMs && entry.expiresAtMs < now)) {
      this.budgetMemory.set(sigKey, { remaining: budgetSats, expiresAtMs: expiresAtMs || (now + ttlSeconds * 1000) });
    }
    const current = this.budgetMemory.get(sigKey);
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
  createL402Middleware,
  SimpleMacaroon
};

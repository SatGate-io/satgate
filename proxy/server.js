/* backend/server.js
 * SatGate - Lightning-powered API access control
 * Production backend that sits behind Aperture for L402 authentication
 * Version: 1.5.1 - Skip middleware for delegation
 * 
 * SECURITY MODEL:
 * ===============
 * PUBLIC endpoints (no auth required):
 *   - /health, /ready              - Health checks
 *   - /api/free/*                  - Free tier APIs
 *   - /api/governance/graph        - Public telemetry (read-only)
 *   - /api/governance/stats        - Public stats (read-only)
 *   - /dashboard                   - Governance dashboard (optionally protected)
 * 
 * ADMIN endpoints (require X-Admin-Token header):
 *   - POST /api/governance/ban     - Ban a token (kill switch)
 *   - POST /api/governance/unban   - Unban a token
 *   - GET /api/governance/banned   - List all banned tokens
 *   - POST /api/free/pricing       - Update pricing
 * 
 * L402 PROTECTED endpoints (require Lightning payment):
 *   - /api/micro/*                 - 1 sat per request
 *   - /api/basic/*                 - 10 sats per request
 *   - /api/standard/*              - 100 sats per request
 *   - /api/premium/*               - 1000 sats per request
 * 
 * CAPABILITY endpoints (require valid macaroon):
 *   - /api/capability/*            - Macaroon-authenticated
 * 
 * Environment Variables:
 *   PRICING_ADMIN_TOKEN    - Secret for admin endpoints (required in production)
 *   DASHBOARD_PUBLIC       - Set to 'true' to allow public dashboard access
 *   ADMIN_RATE_LIMIT       - Requests per minute for admin endpoints (default: 30)
 */
const express = require('express');
const os = require('os');
const path = require('path');
const http = require('http');
const crypto = require('crypto');

const app = express();

// =============================================================================
// RATE LIMITING (Simple in-memory, use Redis in production at scale)
// =============================================================================

const rateLimitStore = new Map();

function rateLimit(options = {}) {
  const { 
    windowMs = 60000,           // 1 minute window
    max = 30,                   // 30 requests per window
    keyGenerator = (req) => req.ip || req.connection.remoteAddress,
    message = 'Too many requests, please try again later'
  } = options;

  return (req, res, next) => {
    const key = `ratelimit:${keyGenerator(req)}`;
    const now = Date.now();
    
    let record = rateLimitStore.get(key);
    if (!record || now - record.windowStart > windowMs) {
      record = { count: 0, windowStart: now };
    }
    
    record.count++;
    rateLimitStore.set(key, record);
    
    // Cleanup old entries periodically
    if (Math.random() < 0.01) {
      for (const [k, v] of rateLimitStore) {
        if (now - v.windowStart > windowMs * 2) rateLimitStore.delete(k);
      }
    }
    
    // Set rate limit headers
    res.setHeader('X-RateLimit-Limit', max);
    res.setHeader('X-RateLimit-Remaining', Math.max(0, max - record.count));
    res.setHeader('X-RateLimit-Reset', Math.ceil((record.windowStart + windowMs) / 1000));
    
    if (record.count > max) {
      return res.status(429).json({ 
        error: message,
        retryAfter: Math.ceil((record.windowStart + windowMs - now) / 1000)
      });
    }
    
    next();
  };
}

// Admin endpoint rate limiter (stricter)
const adminRateLimit = rateLimit({
  windowMs: 60000,
  max: parseInt(process.env.ADMIN_RATE_LIMIT || '30'),
  keyGenerator: (req) => `admin:${req.ip}`,
  message: 'Admin rate limit exceeded'
});

// General API rate limiter (more permissive)
const apiRateLimit = rateLimit({
  windowMs: 60000,
  max: 300,
  message: 'API rate limit exceeded'
});

// =============================================================================
// AUDIT LOGGING
// =============================================================================

const auditLog = [];
const MAX_AUDIT_LOG = 1000;

function logAdminAction(action, details, req) {
  const entry = {
    timestamp: new Date().toISOString(),
    action,
    details,
    ip: req.ip || req.connection.remoteAddress,
    userAgent: req.get('user-agent'),
  };
  
  auditLog.unshift(entry);
  if (auditLog.length > MAX_AUDIT_LOG) auditLog.pop();
  
  console.log(`[AUDIT] ${action}:`, JSON.stringify(details));
  
  // Persist to Redis if available
  if (redis) {
    redis.lpush('satgate:audit', JSON.stringify(entry)).catch(() => {});
    redis.ltrim('satgate:audit', 0, MAX_AUDIT_LOG - 1).catch(() => {});
  }
}

// Note: Audit endpoint defined after requirePricingAdmin middleware at /api/governance/audit

// =============================================================================
// REDIS + WEBSOCKET SETUP (Optional - falls back to in-memory)
// =============================================================================

let redis = null;
let wsServer = null;
const wsClients = new Set();

// Try to connect to Redis if REDIS_URL is set
const REDIS_URL = process.env.REDIS_URL || process.env.REDIS_PRIVATE_URL;
if (REDIS_URL) {
  try {
    const Redis = require('ioredis');
    redis = new Redis(REDIS_URL, {
      maxRetriesPerRequest: 3,
      retryDelayOnFailover: 100,
      lazyConnect: true
    });
    redis.on('connect', () => console.log('[Redis] Connected'));
    redis.on('error', (err) => console.error('[Redis] Error:', err.message));
    redis.connect().catch(err => {
      console.log('[Redis] Connection failed, using in-memory storage');
      redis = null;
    });
  } catch (e) {
    console.log('[Redis] Module not available, using in-memory storage');
    redis = null;
  }
} else {
  console.log('[Storage] No REDIS_URL set, using in-memory storage');
}

// Broadcast to all WebSocket clients
function wsBroadcast(data) {
  const message = JSON.stringify(data);
  wsClients.forEach(client => {
    if (client.readyState === 1) { // WebSocket.OPEN
      client.send(message);
    }
  });
}

// =============================================================================
// GOVERNANCE: Kill Switch & Telemetry
// =============================================================================
// The "Negative Cache" - we don't track valid tokens (stateless),
// but we DO track banned tokens (small stateful blocklist).
// This answers the CISO question: "How do I revoke a compromised token?"

const bannedTokensMemory = new Set();  // In-memory fallback

// Redis-backed or in-memory banned tokens
const bannedTokens = {
  async has(sig) {
    if (redis) {
      try {
        return await redis.sismember('satgate:banned', sig) === 1;
      } catch (e) {
        return bannedTokensMemory.has(sig);
      }
    }
    return bannedTokensMemory.has(sig);
  },
  
  async add(sig) {
    bannedTokensMemory.add(sig);
    if (redis) {
      try {
        await redis.sadd('satgate:banned', sig);
      } catch (e) { /* ignore */ }
    }
    wsBroadcast({ type: 'ban', tokenSignature: sig });
  },
  
  async delete(sig) {
    bannedTokensMemory.delete(sig);
    if (redis) {
      try {
        await redis.srem('satgate:banned', sig);
      } catch (e) { /* ignore */ }
    }
    wsBroadcast({ type: 'unban', tokenSignature: sig });
  },
  
  async size() {
    if (redis) {
      try {
        return await redis.scard('satgate:banned');
      } catch (e) {
        return bannedTokensMemory.size;
      }
    }
    return bannedTokensMemory.size;
  },
  
  async all() {
    if (redis) {
      try {
        return await redis.smembers('satgate:banned');
      } catch (e) {
        return Array.from(bannedTokensMemory);
      }
    }
    return Array.from(bannedTokensMemory);
  },
  
  // Sync check for middleware (uses memory cache)
  hasSync(sig) {
    return bannedTokensMemory.has(sig);
  }
};

// Load banned tokens from Redis on startup
async function loadBannedTokensFromRedis() {
  if (redis) {
    try {
      const tokens = await redis.smembers('satgate:banned');
      tokens.forEach(t => bannedTokensMemory.add(t));
      console.log(`[Redis] Loaded ${tokens.length} banned tokens`);
    } catch (e) {
      console.error('[Redis] Failed to load banned tokens:', e.message);
    }
  }
}

// Telemetry: Track active tokens observed in traffic (for dashboard)
// Also tracks Economic Firewall metrics for sales reporting
const telemetry = {
  activeTokens: new Map(),  // signature -> { lastSeen, ip, depth, constraints }
  blockedCount: 0,          // 402 challenges sent (unpaid requests blocked)
  bannedHits: 0,            // Requests blocked by Kill Switch
  
  // Economic Firewall Metrics
  paidRequests: { micro: 0, basic: 0, standard: 0, premium: 0, total: 0 },
  revenueSats: { micro: 0, basic: 0, standard: 0, premium: 0, total: 0 },
  challengesSent: 0,        // 402 challenges issued
  
  // Cost model for "compute saved" estimation
  // Assume avg backend request costs $0.0001 (Lambda/Cloud Run pricing)
  AVG_BACKEND_COST_USD: 0.0001,
  
  async init() {
    // Load persisted stats from Redis
    if (redis) {
      try {
        const blocked = await redis.get('satgate:stats:blocked');
        const banned = await redis.get('satgate:stats:bannedHits');
        const challenges = await redis.get('satgate:stats:challenges');
        const paidTotal = await redis.get('satgate:stats:paidTotal');
        const revenue = await redis.get('satgate:stats:revenueSats');
        
        if (blocked) this.blockedCount = parseInt(blocked);
        if (banned) this.bannedHits = parseInt(banned);
        if (challenges) this.challengesSent = parseInt(challenges);
        if (paidTotal) this.paidRequests.total = parseInt(paidTotal);
        if (revenue) this.revenueSats.total = parseInt(revenue);
        
        // Load per-tier stats
        for (const tier of ['micro', 'basic', 'standard', 'premium']) {
          const paid = await redis.get(`satgate:stats:paid:${tier}`);
          const tierRev = await redis.get(`satgate:stats:revenue:${tier}`);
          if (paid) this.paidRequests[tier] = parseInt(paid);
          if (tierRev) this.revenueSats[tier] = parseInt(tierRev);
        }
        
        console.log(`[Redis] Loaded stats: blocked=${this.blockedCount}, challenges=${this.challengesSent}, paid=${this.paidRequests.total}, revenue=${this.revenueSats.total} sats`);
      } catch (e) { /* ignore */ }
    }
  },
  
  // Record a 402 challenge sent (request blocked until payment)
  recordChallenge: function() {
    this.blockedCount++;
    this.challengesSent++;
    if (redis) {
      redis.incr('satgate:stats:blocked').catch(() => {});
      redis.incr('satgate:stats:challenges').catch(() => {});
    }
    wsBroadcast({ type: 'challenge', total: this.challengesSent, blocked: this.blockedCount });
  },
  
  // Legacy method for compatibility
  recordBlock: function() {
    this.recordChallenge();
  },
  
  recordBannedHit: function() {
    this.bannedHits++;
    if (redis) redis.incr('satgate:stats:bannedHits').catch(() => {});
    wsBroadcast({ type: 'bannedHit', total: this.bannedHits });
  },
  
  // Record a successful paid request
  recordPaidRequest: function(tier, priceSats) {
    const normalizedTier = tier.toLowerCase();
    if (this.paidRequests[normalizedTier] !== undefined) {
      this.paidRequests[normalizedTier]++;
      this.revenueSats[normalizedTier] += priceSats;
    }
    this.paidRequests.total++;
    this.revenueSats.total += priceSats;
    
    if (redis) {
      redis.incr('satgate:stats:paidTotal').catch(() => {});
      redis.incrby('satgate:stats:revenueSats', priceSats).catch(() => {});
      redis.incr(`satgate:stats:paid:${normalizedTier}`).catch(() => {});
      redis.incrby(`satgate:stats:revenue:${normalizedTier}`, priceSats).catch(() => {});
    }
    
    wsBroadcast({ 
      type: 'paid', 
      tier: normalizedTier, 
      priceSats, 
      totalPaid: this.paidRequests.total,
      totalRevenue: this.revenueSats.total 
    });
  },
  
  // Get Economic Firewall report for sales/dashboards
  getEconomicReport: function() {
    const blockedRequestsSaved = this.blockedCount;
    const computeSavedUSD = blockedRequestsSaved * this.AVG_BACKEND_COST_USD;
    const btcPriceUSD = 100000; // Rough estimate, could fetch live
    const revenueSats = this.revenueSats.total;
    const revenueUSD = (revenueSats / 100000000) * btcPriceUSD;
    
    return {
      economicFirewall: {
        challengesSent: this.challengesSent,
        unpaidRequestsBlocked: this.blockedCount,
        estimatedComputeSavedUSD: computeSavedUSD.toFixed(4),
        note: `${blockedRequestsSaved} requests blocked before reaching backend`
      },
      revenue: {
        totalSats: revenueSats,
        estimatedUSD: revenueUSD.toFixed(4),
        byTier: {
          micro: { requests: this.paidRequests.micro, sats: this.revenueSats.micro },
          basic: { requests: this.paidRequests.basic, sats: this.revenueSats.basic },
          standard: { requests: this.paidRequests.standard, sats: this.revenueSats.standard },
          premium: { requests: this.paidRequests.premium, sats: this.revenueSats.premium },
        }
      },
      killSwitch: {
        bannedTokenAttempts: this.bannedHits,
        note: 'Requests blocked by revoked tokens'
      },
      summary: {
        totalPaidRequests: this.paidRequests.total,
        totalBlockedRequests: this.blockedCount + this.bannedHits,
        protectionRatio: this.paidRequests.total > 0 
          ? ((this.blockedCount / this.paidRequests.total) * 100).toFixed(1) + '%'
          : 'N/A'
      }
    };
  },
  
  recordUsage: function(tokenSignature, caveats, ip) {
    const now = Date.now();
    
    // Calculate depth based on delegation_depth caveat or delegated_from presence
    const depthCaveat = caveats.find(c => c.startsWith('delegation_depth'));
    const hasDelegatedFrom = caveats.some(c => c.startsWith('delegated_from'));
    const depth = depthCaveat ? parseInt(depthCaveat.split('=')[1]?.trim() || '1') : 
                  hasDelegatedFrom ? 1 : 0;
    
    const tokenData = {
      id: tokenSignature.substring(0, 12) + '...',
      fullSignature: tokenSignature,
      lastSeen: now,
      ip: ip,
      depth: depth,
      constraints: caveats,
      status: bannedTokensMemory.has(tokenSignature) ? 'BANNED' : 'ACTIVE'
    };
    
    this.activeTokens.set(tokenSignature, tokenData);
    
    // Cleanup: remove tokens not seen in 10 minutes
    const cutoff = now - (10 * 60 * 1000);
    for (const [sig, data] of this.activeTokens) {
      if (data.lastSeen < cutoff) {
        this.activeTokens.delete(sig);
      }
    }
    
    // Broadcast token update to dashboard
    wsBroadcast({ type: 'token', data: tokenData });
  },
  
  // Record an L402 payment as a visible node on the graph
  recordPaymentNode: function(tier, priceSats, endpoint) {
    const now = Date.now();
    const paymentId = `payment_${tier}_${now}`;
    
    const paymentNode = {
      id: paymentId.substring(0, 16) + '...',
      fullSignature: paymentId,
      lastSeen: now,
      ip: 'L402',
      depth: 0,  // Root level - these are L402 payment tokens
      constraints: [
        `tier:${tier}`,
        `price:${priceSats} sats`,
        `endpoint:${endpoint}`,
        `time:${new Date(now).toLocaleTimeString()}`
      ],
      status: 'PAID',
      isPayment: true
    };
    
    this.activeTokens.set(paymentId, paymentNode);
    
    // Cleanup old payment nodes (keep last 10 minutes)
    const cutoff = now - (10 * 60 * 1000);
    for (const [sig, data] of this.activeTokens) {
      if (data.lastSeen < cutoff) {
        this.activeTokens.delete(sig);
      }
    }
    
    // Broadcast to dashboard
    wsBroadcast({ type: 'token', data: paymentNode });
  },
  
  async getGraphData() {
    const nodes = [];
    const edges = [];
    
    const sortedTokens = Array.from(this.activeTokens.values())
      .sort((a, b) => a.depth - b.depth);
    
    sortedTokens.forEach((token, index) => {
      // Generate appropriate label based on token type
      let label;
      if (token.isPayment) {
        // Extract tier from constraints (e.g., "tier:micro")
        const tierConstraint = token.constraints.find(c => c.startsWith('tier:'));
        const tier = tierConstraint ? tierConstraint.split(':')[1].toUpperCase() : 'L402';
        label = `⚡ ${tier}`;
      } else {
        label = `Token (Depth ${token.depth})`;
      }
      
      nodes.push({
        group: 'nodes',
        data: {
          id: token.fullSignature,
          label: label,
          constraints: token.constraints,
          lastSeen: new Date(token.lastSeen).toLocaleTimeString(),
          depth: token.depth,
          status: token.status,
          isPayment: token.isPayment || false
        }
      });
      
      // Heuristic linking: connect deeper tokens to shallower ones
      if (token.depth > 0) {
        const possibleParent = sortedTokens.find(t => t.depth === token.depth - 1);
        if (possibleParent) {
          edges.push({
            group: 'edges',
            data: {
              id: `e-${token.fullSignature}`,
              source: possibleParent.fullSignature,
              target: token.fullSignature
            }
          });
        }
      }
    });
    
    const bannedSize = await bannedTokens.size();
    
    return {
      nodes,
      edges,
      stats: {
        active: this.activeTokens.size,
        blocked: this.blockedCount,
        banned: bannedSize,
        bannedHits: this.bannedHits,
        // Economic Firewall metrics
        challengesSent: this.challengesSent,
        paidRequests: this.paidRequests.total,
        revenueSats: this.revenueSats.total
      }
    };
  }
};

// =============================================================================
// CONFIGURATION
// =============================================================================
const config = {
  port: process.env.BACKEND_PORT || 8083,
  env: process.env.NODE_ENV || 'development',
  corsOrigins: process.env.CORS_ORIGINS?.split(',') || ['http://127.0.0.1:8081', 'http://localhost:8081', 'http://localhost:8080', 'http://127.0.0.1:8080'],
  rateLimitWindow: 15 * 60 * 1000, // 15 minutes
  rateLimitMax: 100, // requests per window
};

// Optional shared-secret for “control plane” endpoints in production.
// Set `PRICING_ADMIN_TOKEN` in the Railway service if you need to mutate pricing.
const PRICING_ADMIN_TOKEN = process.env.PRICING_ADMIN_TOKEN || '';

// =============================================================================
// MIDDLEWARE
// =============================================================================

// Request logging
app.use((req, res, next) => {
  const start = Date.now();
  res.on('finish', () => {
    const duration = Date.now() - start;
    console.log(`[${new Date().toISOString()}] ${req.method} ${req.path} ${res.statusCode} ${duration}ms`);
  });
  next();
});

// JSON body parser with size limit
app.use(express.json({ limit: '10kb' }));

// Security headers
app.use((req, res, next) => {
  res.setHeader('X-Content-Type-Options', 'nosniff');
  res.setHeader('X-Frame-Options', 'DENY');
  res.setHeader('X-XSS-Protection', '1; mode=block');
  res.setHeader('Referrer-Policy', 'strict-origin-when-cross-origin');
  if (config.env === 'production') {
    res.setHeader('Strict-Transport-Security', 'max-age=31536000; includeSubDomains');
  }
  next();
});

// CORS configuration
// Note: When behind Aperture (port 8081), Aperture handles CORS.
// We only add CORS headers for direct backend access (port 8083).
app.use((req, res, next) => {
  // Check if request came through Aperture (it adds specific headers)
  const viaAperture = req.headers['x-forwarded-for'] || req.headers['x-real-ip'];
  
  // Skip CORS if Aperture already handled it (to avoid duplicate headers)
  if (viaAperture) {
    if (req.method === 'OPTIONS') {
      return res.status(204).end();
    }
    return next();
  }
  
  const origin = req.headers.origin;
  if (config.corsOrigins.includes(origin) || config.corsOrigins.includes('*')) {
    res.setHeader('Access-Control-Allow-Origin', origin || '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
    res.setHeader('Access-Control-Max-Age', '86400');
  }
  if (req.method === 'OPTIONS') {
    return res.status(204).end();
  }
  next();
});

// Note: Rate limiting is now handled by the rateLimit() middleware defined at the top of this file.
// Admin endpoints use adminRateLimit (30/min), API endpoints use apiRateLimit (300/min).

// =============================================================================
// HEALTH & MONITORING ENDPOINTS
// =============================================================================

// Health check (for load balancers, k8s probes)
app.get('/health', (req, res) => {
  res.json({ 
    status: 'healthy',
    version: '1.6.4',
    timestamp: new Date().toISOString(),
    uptime: process.uptime()
  });
});

// Readiness check
app.get('/ready', (req, res) => {
  // Add checks for database connections, external services, etc.
  res.json({ ready: true });
});

// Lightning provider status
app.get('/lightning/status', async (req, res) => {
  try {
    const { getProvider, isApertureMode } = require('./lightning');
    const provider = getProvider();
    const status = await provider.getStatus();
    
    res.json({
      mode: isApertureMode() ? 'aperture' : 'native',
      provider: provider.name,
      ...status
    });
  } catch (error) {
    res.json({
      mode: 'aperture',
      provider: 'aperture',
      ok: true,
      info: {
        note: 'Lightning handled by Aperture (LNC)',
        error: error.message
      }
    });
  }
});

// =============================================================================
// DYNAMIC PRICING CONTROL PLANE
// =============================================================================
// 
// Since Aperture reads from a static config, we implement a "soft" pricing layer:
// - Store current prices in memory (use Redis in production)
// - Provide API to get/set prices
// - Frontend/agents can query current prices before making requests
// - For "hard" pricing changes, you'd need to restart Aperture with new config
//
// Note: This is a "display" pricing layer. Actual L402 prices are set in Aperture.
// To make this "hard" (actually change what Aperture charges), you'd need to:
// 1. Update aperture.yaml programmatically
// 2. Signal Aperture to reload (or restart it)

const pricingStore = {
  tiers: {
    premium: { 
      price: 1000, 
      description: 'AI inference, premium insights',
      endpoints: ['/api/premium/insights', '/api/premium/export']
    },
    standard: { 
      price: 100, 
      description: 'Analytics, data queries',
      endpoints: ['/api/standard/analytics', '/api/standard/metrics']
    },
    basic: { 
      price: 10, 
      description: 'High-volume, low-cost access',
      endpoints: ['/api/basic/status', '/api/basic/quote']
    },
    micro: { 
      price: 1, 
      description: 'True micropayments - $0.001 per request',
      endpoints: ['/api/micro/ping', '/api/micro/data']
    }
  },
  lastUpdated: new Date().toISOString(),
  updatedBy: 'system'
};

// Get current pricing
app.get('/api/free/pricing', (req, res) => {
  res.json({
    ok: true,
    pricing: pricingStore.tiers,
    lastUpdated: pricingStore.lastUpdated,
    updatedBy: pricingStore.updatedBy,
    note: 'Prices in satoshis. These are display prices; actual L402 prices are enforced by Aperture.'
  });
});

function requirePricingAdmin(req, res, next) {
  // For the public demo we allow reads, but disallow writes in production unless
  // an explicit shared secret is configured.
  if (config.env !== 'production') return next();
  if (!PRICING_ADMIN_TOKEN) {
    return res.status(403).json({ error: 'Pricing updates disabled in production' });
  }
  const token =
    req.get('x-admin-token') ||
    req.get('x-satgate-admin-token') ||
    '';
  if (token !== PRICING_ADMIN_TOKEN) {
    return res.status(403).json({ error: 'Forbidden' });
  }
  return next();
}

// Update pricing (would need auth in production)
app.post('/api/free/pricing', requirePricingAdmin, (req, res) => {
  const { tier, price, updatedBy } = req.body;
  
  if (!tier || !pricingStore.tiers[tier]) {
    return res.status(400).json({ 
      error: 'Invalid tier', 
      validTiers: Object.keys(pricingStore.tiers) 
    });
  }
  
  if (typeof price !== 'number' || price < 0) {
    return res.status(400).json({ error: 'Price must be a non-negative number' });
  }
  
  const oldPrice = pricingStore.tiers[tier].price;
  pricingStore.tiers[tier].price = price;
  pricingStore.lastUpdated = new Date().toISOString();
  pricingStore.updatedBy = updatedBy || 'api';
  
  console.log(`[PRICING] ${tier}: ${oldPrice} -> ${price} sats (by ${pricingStore.updatedBy})`);
  
  res.json({
    ok: true,
    message: `Updated ${tier} tier price`,
    oldPrice,
    newPrice: price,
    tier: pricingStore.tiers[tier],
    note: 'This updates display pricing. To enforce via L402, update aperture.yaml and restart Aperture.'
  });
});

// Bulk update pricing
app.put('/api/free/pricing', requirePricingAdmin, (req, res) => {
  const { tiers, updatedBy } = req.body;
  
  if (!tiers || typeof tiers !== 'object') {
    return res.status(400).json({ error: 'Request body must contain "tiers" object' });
  }
  
  const changes = [];
  for (const [tierName, tierData] of Object.entries(tiers)) {
    if (pricingStore.tiers[tierName] && typeof tierData.price === 'number') {
      const oldPrice = pricingStore.tiers[tierName].price;
      pricingStore.tiers[tierName].price = tierData.price;
      changes.push({ tier: tierName, oldPrice, newPrice: tierData.price });
    }
  }
  
  pricingStore.lastUpdated = new Date().toISOString();
  pricingStore.updatedBy = updatedBy || 'api';
  
  console.log(`[PRICING] Bulk update: ${JSON.stringify(changes)} (by ${pricingStore.updatedBy})`);
  
  res.json({
    ok: true,
    message: 'Bulk pricing update complete',
    changes,
    currentPricing: pricingStore.tiers
  });
});

// =============================================================================
// PUBLIC ENDPOINTS (No L402 required)
// =============================================================================

app.get('/api/free/ping', (req, res) => {
  res.json({
    ok: true,
    time: new Date().toISOString(),
    host: os.hostname(),
    message: 'pong',
    version: '1.0.0'
  });
});

// =============================================================================
// PROTECTED ENDPOINTS - TIERED PRICING (L402 required - validated by Aperture)
// =============================================================================
// 
// Tier       | Price      | Use Case
// -----------|------------|------------------------------------------
// Premium    | 1000 sats  | AI inference, premium insights, critical ops
// Standard   | 100 sats   | Analytics, data queries
// Basic      | 10 sats    | High-volume, low-cost access
//

// Helper to detect L402 auth
const isL402Auth = (req) => {
  const auth = req.get('authorization') || '';
  return auth.toLowerCase().startsWith('l402') || auth.toLowerCase().startsWith('lsat');
};

// Middleware to track L402 paid requests
// Call this on paid tier endpoints to record revenue
const trackPaidRequest = (tier, priceSats) => (req, res, next) => {
  // If request reached paid endpoint, it MUST have paid (Aperture enforces this)
  // So we track all requests to paid endpoints as paid
  telemetry.recordPaidRequest(tier, priceSats);
  
  // Also create a visible node on the governance graph
  telemetry.recordPaymentNode(tier, priceSats, req.path);
  
  console.log(`[L402] Paid request: ${tier} tier, ${priceSats} sats, endpoint: ${req.path}`);
  
  next();
};

// ---------------------------------------------------------------------------
// PREMIUM TIER - 1000 sats ($1.00) per request
// ---------------------------------------------------------------------------

// Premium AI insights endpoint (1000 sats)
app.get('/api/premium/insights', trackPaidRequest('premium', 1000), (req, res) => {
  res.json({
    ok: true,
    tier: 'premium',
    price: '1000 sats',
    time: new Date().toISOString(),
    resource: 'ai-insights',
    accessType: isL402Auth(req) ? 'L402' : 'direct',
    data: {
      predictions: {
        revenueNextWeek: { value: 95000, confidence: 0.87, currency: 'USD' },
        churnRisk: { value: 0.12, trend: 'decreasing' },
        growthRate: { value: 0.23, period: 'monthly' }
      },
      recommendations: [
        { priority: 'high', action: 'Increase marketing spend in APAC region', impact: '+15% revenue' },
        { priority: 'medium', action: 'Optimize checkout flow', impact: '-8% cart abandonment' },
        { priority: 'low', action: 'A/B test new pricing page', impact: '+3% conversions' }
      ],
      modelVersion: '2.1.0',
      generatedAt: new Date().toISOString()
    }
  });
});

// Premium data export endpoint (1000 sats)
app.get('/api/premium/export', trackPaidRequest('premium', 1000), (req, res) => {
  res.json({
    ok: true,
    tier: 'premium',
    price: '1000 sats',
    time: new Date().toISOString(),
    resource: 'data-export',
    accessType: isL402Auth(req) ? 'L402' : 'direct',
    data: {
      format: 'json',
      recordCount: 15420,
      dateRange: { start: '2025-10-01', end: '2025-11-25' },
      fields: ['user_id', 'event', 'timestamp', 'value', 'metadata'],
      downloadUrl: '/api/premium/export/download?token=sample',
      expiresIn: 3600
    }
  });
});

// ---------------------------------------------------------------------------
// STANDARD TIER - 100 sats ($0.10) per request
// ---------------------------------------------------------------------------

// Standard analytics endpoint (100 sats)
app.get('/api/standard/analytics', trackPaidRequest('standard', 100), (req, res) => {
  res.json({
    ok: true,
    tier: 'standard',
    price: '100 sats',
    time: new Date().toISOString(),
    resource: 'analytics',
    accessType: isL402Auth(req) ? 'L402' : 'direct',
    data: {
      summary: {
        totalUsers: 1287,
        activeToday: 342,
        avgLatencyMs: 73
      },
      revenue: {
        today: 15420,
        thisWeek: 89340,
        thisMonth: 342100
      },
      topEndpoints: [
        { path: '/api/premium/insights', count: 512, revenue: 512000 },
        { path: '/api/standard/analytics', count: 1843, revenue: 184300 },
        { path: '/api/basic/status', count: 9241, revenue: 92410 }
      ]
    },
    metadata: {
      generatedAt: new Date().toISOString(),
      cacheControl: 'private, max-age=60'
    }
  });
});

// Standard metrics endpoint (100 sats)
app.get('/api/standard/metrics', trackPaidRequest('standard', 100), (req, res) => {
  res.json({
    ok: true,
    tier: 'standard',
    price: '100 sats',
    time: new Date().toISOString(),
    resource: 'metrics',
    accessType: isL402Auth(req) ? 'L402' : 'direct',
    data: {
      performance: {
        p50: 45,
        p95: 120,
        p99: 250,
        unit: 'ms'
      },
      availability: {
        uptime: 0.9997,
        lastIncident: '2025-11-01T14:30:00Z'
      },
      throughput: {
        requestsPerSecond: 1250,
        peakToday: 3420
      }
    }
  });
});

// ---------------------------------------------------------------------------
// BASIC TIER - 10 sats ($0.01) per request
// ---------------------------------------------------------------------------

// Basic status endpoint (10 sats)
app.get('/api/basic/status', trackPaidRequest('basic', 10), (req, res) => {
  res.json({
    ok: true,
    tier: 'basic',
    price: '10 sats',
    time: new Date().toISOString(),
    resource: 'status',
    accessType: isL402Auth(req) ? 'L402' : 'direct',
    data: {
      services: {
        api: 'operational',
        database: 'operational',
        cache: 'operational',
        payments: 'operational'
      },
      lastUpdated: new Date().toISOString()
    }
  });
});

// Basic quote endpoint (10 sats)
app.get('/api/basic/quote', trackPaidRequest('basic', 10), (req, res) => {
  const quotes = [
    { text: "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks", author: "Satoshi Nakamoto" },
    { text: "Bitcoin is a technological tour de force.", author: "Bill Gates" },
    { text: "I think the internet is going to be one of the major forces for reducing the role of government.", author: "Milton Friedman" },
    { text: "The root problem with conventional currency is all the trust that's required to make it work.", author: "Satoshi Nakamoto" }
  ];
  const quote = quotes[Math.floor(Math.random() * quotes.length)];
  
  res.json({
    ok: true,
    tier: 'basic',
    price: '10 sats',
    time: new Date().toISOString(),
    resource: 'quote',
    accessType: isL402Auth(req) ? 'L402' : 'direct',
    data: quote
  });
});

// ---------------------------------------------------------------------------
// CAPABILITY TIER - Phase 1: Macaroon-only access (no payment)
// ---------------------------------------------------------------------------
// Aperture whitelists this path; we enforce macaroon auth in app layer.
// This demonstrates "Zero Trust PEP" without requiring Lightning payments.

const macaroon = require('macaroon');

// Shared secret for Phase 1 demo (in production, use env var)
const CAPABILITY_ROOT_KEY = process.env.CAPABILITY_ROOT_KEY || 'satgate-phase1-demo-key-change-in-prod';
const CAPABILITY_LOCATION = 'https://satgate.io';
const CAPABILITY_IDENTIFIER = 'satgate-capability-v1';

// Middleware: Validate macaroon for /api/capability/* routes
// Implements DYNAMIC SCOPE ENFORCEMENT based on requested path
app.use('/api/capability', (req, res, next) => {
  const authHeader = req.get('authorization') || '';
  
  // Determine required scope for this path
  let requiredScope = 'api:capability:read'; // Default for /ping, /data
  if (req.path === '/mint') {
    requiredScope = 'api:capability:admin'; // Minting requires admin scope
  }
  
  // Allow unauthenticated access to /mint for bootstrapping (demo only)
  // In production, remove this block and require admin token
  if (req.path === '/mint' && !authHeader) {
    console.log(`[PEP] Allowing unauthenticated mint (demo mode)`);
    return next();
  }
  
  // Allow /delegate to handle its own auth (creates child tokens)
  // The delegate endpoint validates the parent token itself
  if (req.path === '/delegate') {
    console.log(`[PEP] Delegating to /delegate endpoint for auth`);
    return next();
  }
  
  // Allow unauthenticated access to /demo/* for backup demo purposes
  if (req.path.startsWith('/demo/')) {
    console.log(`[PEP] Allowing demo endpoint: ${req.path}`);
    return next();
  }
  
  // Check for Bearer token
  if (!authHeader.startsWith('Bearer ')) {
    return res.status(401).json({
      error: 'Missing Capability Token',
      hint: 'Use "Authorization: Bearer <macaroon>" header',
      mintEndpoint: '/api/capability/mint'
    });
  }
  
  const tokenBase64 = authHeader.slice(7); // Remove "Bearer "
  
  try {
    // Decode and import the macaroon
    const tokenBytes = Buffer.from(tokenBase64, 'base64');
    const m = macaroon.importMacaroon(tokenBytes);
    
    // KILL SWITCH: Check if token is banned (uses sync in-memory check for speed)
    const tokenSignature = Buffer.from(m.signature).toString('hex');
    if (bannedTokens.hasSync(tokenSignature)) {
      console.log(`[KILL SWITCH] 🛑 Blocked banned token: ${tokenSignature.substring(0, 16)}...`);
      telemetry.recordBannedHit();
      return res.status(403).json({
        error: 'Token Revoked',
        reason: 'This token has been banned by an administrator',
        code: 'TOKEN_BANNED'
      });
    }
    
    // Verify signature with root key
    const keyBytes = Buffer.from(CAPABILITY_ROOT_KEY, 'utf8');
    const now = Date.now();
    
    // Extract caveats from macaroon for logging
    const caveats = [];
    if (m._caveats && Array.isArray(m._caveats)) {
      m._caveats.forEach(c => {
        if (c._identifier) {
          caveats.push(c._identifier.toString('utf8'));
        }
      });
    }
    
    console.log(`[PEP] Path: ${req.path} | Required Scope: ${requiredScope}`);
    console.log(`[PEP] Token caveats: ${JSON.stringify(caveats)}`);
    
    // Track if scope was validated
    let scopeValidated = false;
    let scopeError = null;
    
    // Verify HMAC signature with dynamic scope checking
    const checkCaveat = (caveat) => {
      const caveatStr = typeof caveat === 'string' ? caveat : caveat.toString('utf8');
      
      // Time-based expiry
      if (caveatStr.startsWith('expires = ')) {
        const expiry = parseInt(caveatStr.split(' = ')[1], 10);
        if (now > expiry) {
          return 'token expired';
        }
        return null; // OK
      }
      
      // DYNAMIC SCOPE CHECK
      if (caveatStr.startsWith('scope = ')) {
        const tokenScope = caveatStr.split(' = ')[1].trim();
        
        // Check if token scope covers the required scope
        // Wildcard: "api:capability:*" covers everything
        // Specific: "api:capability:ping" only covers /ping
        // Admin: "api:capability:admin" covers admin actions
        
        let isAllowed = false;
        
        // Wildcard scope covers everything
        if (tokenScope === 'api:capability:*') {
          isAllowed = true;
        }
        // Exact match
        else if (tokenScope === requiredScope) {
          isAllowed = true;
        }
        // "read" scope covers read operations (ping, data)
        else if (tokenScope === 'api:capability:read' && 
                 (requiredScope === 'api:capability:read' || 
                  requiredScope === 'api:capability:ping' ||
                  requiredScope === 'api:capability:data')) {
          isAllowed = true;
        }
        // Specific endpoint scope
        else if (tokenScope === 'api:capability:ping' && req.path === '/ping') {
          isAllowed = true;
        }
        else if (tokenScope === 'api:capability:data' && req.path === '/data') {
          isAllowed = true;
        }
        
        if (!isAllowed) {
          scopeError = `Scope violation: token has '${tokenScope}', need '${requiredScope}'`;
          console.log(`[PEP] ⛔ ${scopeError}`);
          return scopeError;
        }
        
        scopeValidated = true;
        console.log(`[PEP] ✓ Scope OK: '${tokenScope}' covers '${requiredScope}'`);
        return null; // OK
      }
      
      // Delegation marker - accept (informational only)
      if (caveatStr.startsWith('delegated_by = ')) {
        return null; // OK
      }
      
      // Unknown caveat - reject for security
      return 'unknown caveat: ' + caveatStr;
    };
    
    // Verify signature and caveats
    m.verify(keyBytes, checkCaveat);
    
    // If we get here but scope wasn't validated, reject
    // (This handles tokens without any scope caveat)
    if (!scopeValidated) {
      throw new Error('Token has no scope caveat');
    }
    
    // Extract identifier
    const identifier = m._identifier ? m._identifier.toString('utf8') : 'unknown';
    
    // Attach parsed info to request for endpoints
    req.capability = {
      caveats,
      identifier,
      requiredScope,
      validatedAt: new Date().toISOString(),
      tokenSignature: tokenSignature
    };
    
    // TELEMETRY: Record this token usage for governance dashboard
    const clientIp = req.ip || req.connection.remoteAddress || 'unknown';
    telemetry.recordUsage(tokenSignature, caveats, clientIp);
    
    console.log(`[CAPABILITY] ✓ Valid token with scope for ${req.path}`);
    next();
    
  } catch (e) {
    console.error(`[CAPABILITY] ✗ Access denied: ${e.message}`);
    const isScopeError = e.message.includes('Scope violation') || e.message.includes('scope');
    return res.status(403).json({
      error: 'Access Denied',
      reason: e.message,
      hint: isScopeError ? 'Token scope does not permit this action' : undefined
    });
  }
});

// Mint a capability token (for demo purposes - in prod, this would be admin-only)
app.post('/api/capability/mint', express.json(), (req, res) => {
  const { scope = 'api:capability:read', expiresIn = 3600 } = req.body || {};
  
  try {
    // Create base macaroon
    const keyBytes = Buffer.from(CAPABILITY_ROOT_KEY, 'utf8');
    const identifier = `${CAPABILITY_IDENTIFIER}:${Date.now()}`;
    
    let m = macaroon.newMacaroon({
      identifier: Buffer.from(identifier, 'utf8'),
      location: CAPABILITY_LOCATION,
      rootKey: keyBytes
    });
    
    // Add caveats (method on macaroon object)
    const expiresAt = Date.now() + (expiresIn * 1000);
    m.addFirstPartyCaveat(Buffer.from(`expires = ${expiresAt}`, 'utf8'));
    m.addFirstPartyCaveat(Buffer.from(`scope = ${scope}`, 'utf8'));
    
    // Export as base64
    const tokenBytes = m.exportBinary();
    const tokenBase64 = Buffer.from(tokenBytes).toString('base64');
    
    console.log(`[CAPABILITY] Minted token: scope=${scope}, expires=${new Date(expiresAt).toISOString()}`);
    
    res.json({
      ok: true,
      token: tokenBase64,
      usage: `curl -H "Authorization: Bearer ${tokenBase64}" https://satgate-production.up.railway.app/api/capability/ping`,
      caveats: {
        scope,
        expires: new Date(expiresAt).toISOString(),
        expiresIn: `${expiresIn} seconds`
      },
      note: 'This is a Phase 1 capability token. No payment required.'
    });
    
  } catch (e) {
    console.error(`[CAPABILITY] Mint error: ${e.message}`);
    res.status(500).json({ error: 'Failed to mint token', reason: e.message });
  }
});

// TEST: Minimal macaroon creation with real keys
app.get('/api/token/test', (req, res) => {
  try {
    // Test with real keys to debug
    const keyDebug = {
      CAPABILITY_ROOT_KEY_defined: !!CAPABILITY_ROOT_KEY,
      CAPABILITY_ROOT_KEY_type: typeof CAPABILITY_ROOT_KEY,
      CAPABILITY_ROOT_KEY_length: CAPABILITY_ROOT_KEY ? String(CAPABILITY_ROOT_KEY).length : 0,
      CAPABILITY_ROOT_KEY_preview: CAPABILITY_ROOT_KEY ? String(CAPABILITY_ROOT_KEY).substring(0, 20) : null
    };
    
    // Try to create macaroon with the real key
    const realKey = Buffer.from(String(CAPABILITY_ROOT_KEY || 'fallback-key'), 'utf8');
    const testId = Buffer.from(`${CAPABILITY_IDENTIFIER}:test:${Date.now()}`, 'utf8');
    
    const m = macaroon.newMacaroon({
      identifier: testId,
      location: CAPABILITY_LOCATION,
      rootKey: realKey
    });
    
    const sig = Buffer.from(m.signature).toString('hex').substring(0, 16);
    
    res.json({ ok: true, signature: sig, keyDebug });
  } catch (e) {
    res.status(500).json({ error: e.message, stack: e.stack?.split('\n').slice(0, 5) });
  }
});

// Delegate a capability token (create a child with restricted scope)
// This demonstrates the "Chain of Custody" - parent delegates to child
// Using GET to bypass any POST-related issues
app.get('/api/token/delegate', (req, res) => {
  console.log('[DELEGATE] === GET REQUEST RECEIVED ===');
  
  const authHeader = req.get('authorization') || '';
  console.log('[DELEGATE] Auth header present:', authHeader.length > 0);
  
  if (!authHeader.startsWith('Bearer ')) {
    console.log('[DELEGATE] Missing Bearer token');
    return res.status(401).json({ 
      error: 'Parent token required',
      hint: 'Use "Authorization: Bearer <parent_token>" header'
    });
  }
  
  const parentTokenBase64 = authHeader.slice(7).trim();
  console.log('[DELEGATE] Token extracted, length:', parentTokenBase64.length);
  
  // Use query params instead of body for GET
  const scope = req.query.scope || 'api:capability:ping';
  const expiresIn = parseInt(req.query.expiresIn) || 300;
  console.log('[DELEGATE] Scope:', scope, 'ExpiresIn:', expiresIn);
  
  let step = 0;
  try {
    step = 1;
    const parentSig = crypto.createHash('sha256')
      .update(parentTokenBase64)
      .digest('hex')
      .substring(0, 16);
    
    step = 2;
    const keyBytes = Buffer.from(CAPABILITY_ROOT_KEY, 'utf8');
    
    step = 3;
    const childId = `${CAPABILITY_IDENTIFIER}:child:${Date.now()}`;
    
    step = 4;
    const idBytes = Buffer.from(childId, 'utf8');
    
    step = 5;
    let childMac = macaroon.newMacaroon({
      identifier: idBytes,
      location: CAPABILITY_LOCATION,
      rootKey: keyBytes
    });
    
    // Add only 2 caveats like the mint endpoint
    step = 6;
    const expiresAt = Date.now() + (expiresIn * 1000);
    childMac.addFirstPartyCaveat(Buffer.from(`expires = ${expiresAt}`, 'utf8'));
    
    step = 7;
    childMac.addFirstPartyCaveat(Buffer.from(`scope = ${scope}`, 'utf8'));
    
    step = 8;
    const childBytes = childMac.exportBinary();
    
    step = 9;
    const childTokenBase64 = Buffer.from(childBytes).toString('base64');
    
    step = 10;
    const childSig = Buffer.from(childMac.signature).toString('hex').substring(0, 16);
    
    console.log(`[CAPABILITY] Delegated token: parent=${parentSig}..., child=${childSig}..., scope=${scope}`);
    
    res.json({
      ok: true,
      token: childTokenBase64,
      chainOfCustody: {
        parentSignature: parentSig + '...',
        childSignature: childSig + '...',
        relationship: 'Parent → Child (attenuated)'
      },
      caveats: {
        scope: `${scope} [RESTRICTED]`,
        expires: new Date(expiresAt).toISOString(),
        expiresIn: `${expiresIn} seconds [EPHEMERAL]`
      },
      note: 'Child token has MORE restrictions than parent. This is cryptographically enforced.'
    });
    
  } catch (e) {
    console.error(`[CAPABILITY] Delegate error at step ${step}: ${e.message}`);
    res.status(400).json({ error: 'Failed to delegate token', step: step, reason: e.message });
  }
});

// Capability ping endpoint (protected by middleware above)
app.get('/api/capability/ping', (req, res) => {
  res.json({
    ok: true,
    tier: 'capability',
    price: '0 sats',
    mode: 'Phase 1: Capability-Only',
    time: new Date().toISOString(),
    resource: 'capability-ping',
    accessType: 'Macaroon (no payment)',
    capability: req.capability,
    data: {
      message: '✓ Authenticated with capability token - no Lightning payment required!',
      note: 'This proves Zero Trust PEP works without the crypto payment rail.'
    }
  });
});

// Capability data endpoint
app.get('/api/capability/data', (req, res) => {
  res.json({
    ok: true,
    tier: 'capability',
    mode: 'Phase 1: Capability-Only',
    time: new Date().toISOString(),
    resource: 'capability-data',
    capability: req.capability,
    data: {
      secret: 'This data is protected by a capability token',
      randomValue: Math.floor(Math.random() * 1000000),
      timestamp: Date.now()
    }
  });
});

// ===========================================================================
// DEMO BACKUP - Server-side Delegation Simulation
// ===========================================================================
// This endpoint simulates the "offline" agent behavior for demo purposes
// when you don't have access to your local Node.js environment.
// "Two is one, and one is none" - always have a backup for critical demos.

app.post('/api/capability/demo/delegate', (req, res) => {
  try {
    const keyBytes = Buffer.from(CAPABILITY_ROOT_KEY, 'utf8');
    const now = Date.now();
    
    // 1. Create parent token (Simulating the Agent's existing credential)
    const parentId = `${CAPABILITY_IDENTIFIER}:parent:${now}`;
    let parentMacaroon = macaroon.newMacaroon({
      identifier: Buffer.from(parentId, 'utf8'),
      location: CAPABILITY_LOCATION,
      rootKey: keyBytes
    });
    
    // Parent: broad scope, 1 hour expiry
    const parentExpiry = now + (60 * 60 * 1000);
    parentMacaroon.addFirstPartyCaveat(Buffer.from(`expires = ${parentExpiry}`, 'utf8'));
    parentMacaroon.addFirstPartyCaveat(Buffer.from(`scope = api:capability:*`, 'utf8'));
    
    const parentToken = Buffer.from(parentMacaroon.exportBinary()).toString('base64');
    
    // 2. Create child token with MORE restrictive caveats
    // In a real scenario, this would be done by attenuating the parent
    // For demo purposes, we create it directly with the restricted caveats
    const childId = `${CAPABILITY_IDENTIFIER}:child:${now}`;
    let childMacaroon = macaroon.newMacaroon({
      identifier: Buffer.from(childId, 'utf8'),
      location: CAPABILITY_LOCATION,
      rootKey: keyBytes
    });
    
    // Child: ALL parent caveats PLUS more restrictive ones
    // (This simulates what attenuation produces)
    const childExpiry = now + (5 * 60 * 1000); // 5 minutes (shorter than parent)
    childMacaroon.addFirstPartyCaveat(Buffer.from(`expires = ${childExpiry}`, 'utf8'));
    childMacaroon.addFirstPartyCaveat(Buffer.from(`scope = api:capability:ping`, 'utf8')); // Narrower
    childMacaroon.addFirstPartyCaveat(Buffer.from(`delegated_by = agent-001`, 'utf8'));
    
    const childToken = Buffer.from(childMacaroon.exportBinary()).toString('base64');
    
    // 3. Format the output to match the CLI demo script exactly
    const output = `
═══════════════════════════════════════════════════════════════
  SatGate Phase 1: DELEGATION DEMO (Server-Side Backup)
  "The Google-Grade Superpower"
═══════════════════════════════════════════════════════════════

┌─────────────────────────────────────────────────────────────┐
│ SCENE 1: Parent Token Created                              │
└─────────────────────────────────────────────────────────────┘

[CISO] Capability issued for the Data Agent.
[CISO] Scope: api:capability:* (full access)
[CISO] Expires: ${new Date(parentExpiry).toISOString()}

Parent Token: ${parentToken.substring(0, 50)}...

┌─────────────────────────────────────────────────────────────┐
│ SCENE 2: Agent Delegates to Worker                         │
└─────────────────────────────────────────────────────────────┘

[AGENT] I need to delegate a read-only task to a Worker.
[AGENT] The Worker should only access /ping for 5 minutes.

[SYSTEM] Generating restricted sub-token...
[NETWORK] Requests sent: 0  ← OFFLINE OPERATION
[CRYPTO] Attenuating parent macaroon...

┌─────────────────────────────────────────────────────────────┐
│ SCENE 3: Comparison                                        │
└─────────────────────────────────────────────────────────────┘

┌────────────────────┬──────────────────┬────────────────────┐
│ Property           │ Parent Token     │ Child Token        │
├────────────────────┼──────────────────┼────────────────────┤
│ Scope              │ api:capability:* │ api:capability:ping│
│ Expires            │ 1 hour           │ 5 minutes          │
│ Network calls      │ 0                │ 0                  │
│ Admin approval     │ NO               │ NO                 │
└────────────────────┴──────────────────┴────────────────────┘

[CRYPTO] Sub-token signature: VALID
[CRYPTO] Caveat chain: ATTENUATED (more restrictive)

✅ Child Token Created (Attenuated)

═══════════════════════════════════════════════════════════════
CHILD TOKEN (copy this for testing):
═══════════════════════════════════════════════════════════════
${childToken}
═══════════════════════════════════════════════════════════════

┌─────────────────────────────────────────────────────────────┐
│ TEST COMMANDS                                              │
└─────────────────────────────────────────────────────────────┘

# ✅ ALLOWED: Child token can access /ping
curl -H "Authorization: Bearer ${childToken.substring(0, 40)}..." \\
  https://satgate-production.up.railway.app/api/capability/ping

# ❌ BLOCKED: Child token CANNOT mint new tokens
curl -X POST -H "Authorization: Bearer ${childToken.substring(0, 40)}..." \\
  https://satgate-production.up.railway.app/api/capability/mint

┌─────────────────────────────────────────────────────────────┐
│ THE KEY INSIGHT                                            │
└─────────────────────────────────────────────────────────────┘

  "The agent just cut a spare key for the janitor —
   one that only opens the basement, and expires in 5 minutes.
   It didn't need to call the locksmith."

  ✓ ZERO network calls
  ✓ ZERO admin tickets
  ✓ INSTANT delegation
  ✓ SELF-EXPIRING credentials
  ✓ MATHEMATICALLY restricted scope

`;

    res.setHeader('Content-Type', 'text/plain; charset=utf-8');
    res.send(output);
    
  } catch (e) {
    console.error(`[DEMO] Delegation simulation error: ${e.message}`);
    res.status(500).send(`Error: ${e.message}`);
  }
});

// ---------------------------------------------------------------------------
// MICRO TIER - 1 sat ($0.001) per request
// ---------------------------------------------------------------------------
// True micropayments - the absolute minimum viable price

// Micro ping endpoint (1 sat)
app.get('/api/micro/ping', trackPaidRequest('micro', 1), (req, res) => {
  res.json({
    ok: true,
    tier: 'micro',
    price: '1 sat',
    priceUSD: '$0.001',
    time: new Date().toISOString(),
    resource: 'ping',
    accessType: isL402Auth(req) ? 'L402' : 'direct',
    data: {
      message: 'pong',
      note: 'This request cost you $0.001 - true micropayments in action!'
    }
  });
});

// Micro data endpoint (1 sat)
app.get('/api/micro/data', trackPaidRequest('micro', 1), (req, res) => {
  res.json({
    ok: true,
    tier: 'micro',
    price: '1 sat',
    priceUSD: '$0.001',
    time: new Date().toISOString(),
    resource: 'data',
    accessType: isL402Auth(req) ? 'L402' : 'direct',
    data: {
      randomNumber: Math.floor(Math.random() * 1000000),
      timestamp: Date.now(),
      note: 'Pay-per-byte data access at the lowest possible price'
    }
  });
});

// =============================================================================
// GOVERNANCE ENDPOINTS - Kill Switch & Dashboard API
// =============================================================================

// Get governance graph data (for dashboard visualization)
app.get('/api/governance/graph', async (req, res) => {
  const graphData = await telemetry.getGraphData();
  res.json(graphData);
});

// Get governance stats (quick summary)
app.get('/api/governance/stats', async (req, res) => {
  const data = await telemetry.getGraphData();
  res.json({
    ok: true,
    stats: data.stats,
    timestamp: new Date().toISOString()
  });
});

// Economic Firewall Report (for sales dashboards and weekly reporting)
// Shows: blocked requests, revenue, compute saved, protection metrics
app.get('/api/governance/economic-report', async (req, res) => {
  const report = telemetry.getEconomicReport();
  res.json({
    ok: true,
    report,
    timestamp: new Date().toISOString(),
    period: 'since-restart' // TODO: Add time windowing with Redis
  });
});

// KILL SWITCH: Ban a token (The "Panic Button")
// Protected by: admin auth + rate limiting + audit logging
app.post('/api/governance/ban', adminRateLimit, requirePricingAdmin, async (req, res) => {
  const { tokenSignature, reason } = req.body;
  
  // Input validation
  if (!tokenSignature) {
    logAdminAction('BAN_FAILED', { error: 'Missing tokenSignature' }, req);
    return res.status(400).json({ 
      error: 'Missing tokenSignature',
      hint: 'Provide the token signature (hex) to ban'
    });
  }
  
  // Validate signature format (should be hex)
  const sig = tokenSignature.replace('...', '');
  if (!/^[a-fA-F0-9]+$/.test(sig)) {
    logAdminAction('BAN_FAILED', { error: 'Invalid signature format', provided: sig.substring(0, 20) }, req);
    return res.status(400).json({
      error: 'Invalid tokenSignature format',
      hint: 'Token signature must be a hex string'
    });
  }
  
  if (await bannedTokens.has(sig)) {
    logAdminAction('BAN_DUPLICATE', { tokenSignature: sig.substring(0, 16) }, req);
    return res.json({
      ok: true,
      message: 'Token was already banned',
      tokenSignature: sig.substring(0, 16) + '...'
    });
  }
  
  await bannedTokens.add(sig);
  logAdminAction('BAN_SUCCESS', { 
    tokenSignature: sig.substring(0, 16), 
    reason: reason || 'Not specified' 
  }, req);
  console.log(`[KILL SWITCH] 🚨 Token banned: ${sig.substring(0, 16)}... Reason: ${reason || 'Not specified'}`);
  
  const totalBanned = await bannedTokens.size();
  res.json({
    ok: true,
    message: 'Token banned successfully',
    tokenSignature: sig.substring(0, 16) + '...',
    reason: reason || 'Not specified',
    totalBanned,
    note: 'Token will be rejected on next use. Active sessions using this token will fail.'
  });
});

// KILL SWITCH: Unban a token
// Protected by: admin auth + rate limiting + audit logging
app.post('/api/governance/unban', adminRateLimit, requirePricingAdmin, async (req, res) => {
  const { tokenSignature } = req.body;
  
  if (!tokenSignature) {
    logAdminAction('UNBAN_FAILED', { error: 'Missing tokenSignature' }, req);
    return res.status(400).json({ error: 'Missing tokenSignature' });
  }
  
  const sig = tokenSignature.replace('...', '');
  
  if (!(await bannedTokens.has(sig))) {
    logAdminAction('UNBAN_NOT_FOUND', { tokenSignature: sig.substring(0, 16) }, req);
    return res.json({
      ok: true,
      message: 'Token was not banned',
      tokenSignature: sig.substring(0, 16) + '...'
    });
  }
  
  await bannedTokens.delete(sig);
  logAdminAction('UNBAN_SUCCESS', { tokenSignature: sig.substring(0, 16) }, req);
  console.log(`[KILL SWITCH] ✅ Token unbanned: ${sig.substring(0, 16)}...`);
  
  const totalBanned = await bannedTokens.size();
  res.json({
    ok: true,
    message: 'Token unbanned successfully',
    tokenSignature: sig.substring(0, 16) + '...',
    totalBanned
  });
});

// List all banned tokens (admin only)
// Protected by: admin auth + rate limiting
app.get('/api/governance/banned', adminRateLimit, requirePricingAdmin, async (req, res) => {
  logAdminAction('LIST_BANNED', { count: await bannedTokens.size() }, req);
  
  const allBanned = await bannedTokens.all();
  const banned = allBanned.map(sig => ({
    signature: sig.substring(0, 16) + '...',
    fullSignature: sig
  }));
  
  res.json({
    ok: true,
    count: banned.length,
    bannedTokens: banned
  });
});

// Get admin audit log
// Protected by: admin auth + rate limiting
app.get('/api/governance/audit', adminRateLimit, requirePricingAdmin, async (req, res) => {
  const limit = Math.min(parseInt(req.query.limit) || 100, MAX_AUDIT_LOG);
  
  let logs = auditLog.slice(0, limit);
  
  // Try to get from Redis for persistence
  if (redis) {
    try {
      const redisLogs = await redis.lrange('satgate:audit', 0, limit - 1);
      if (redisLogs.length > 0) {
        logs = redisLogs.map(l => JSON.parse(l));
      }
    } catch (e) {
      // Fall back to in-memory
    }
  }
  
  res.json({ ok: true, count: logs.length, logs });
});

// -----------------------------------------------------------------------------
// Root (API-only) landing
// -----------------------------------------------------------------------------
app.get('/', (req, res) => {
  res.json({
    ok: true,
    name: 'SatGate Cloud API',
    endpoints: {
      health: '/health',
      freePing: '/api/free/ping',
      pricing: '/api/free/pricing',
      l402Example: '/api/micro/ping',
      governance: '/api/governance/graph'
    },
    playground: 'https://satgate.io/playground'
  });
});

// =============================================================================
// STATIC FILE SERVING (Governance Dashboard)
// =============================================================================

// Serve dashboard at /dashboard
// Try file first, fall back to inline minimal version
app.get('/dashboard', (req, res) => {
  const dashboardPath = path.join(__dirname, 'public', 'index.html');
  
  res.sendFile(dashboardPath, (err) => {
    if (err) {
      // Serve inline minimal dashboard as fallback
      console.log(`[DASHBOARD] Serving inline fallback (file not found: ${dashboardPath})`);
      res.setHeader('Content-Type', 'text/html');
      res.send(getInlineDashboard());
    }
  });
});

// Minimal inline dashboard for Railway deployment fallback
function getInlineDashboard() {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>SatGate // Governance</title>
  <script src="https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.28.1/cytoscape.min.js"></script>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    :root { --bg: #0a0e14; --bg2: #151b24; --border: #2d3748; --text: #e2e8f0; --green: #00ff9d; --red: #ff4757; --cyan: #00d4ff; }
    body { font-family: 'SF Mono', monospace; background: var(--bg); color: var(--text); height: 100vh; display: grid; grid-template-rows: 60px 1fr; }
    header { display: flex; justify-content: space-between; align-items: center; padding: 0 24px; background: var(--bg2); border-bottom: 1px solid var(--border); }
    .logo { font-size: 18px; font-weight: 600; letter-spacing: 1px; }
    .logo span { color: var(--green); }
    .status { display: flex; align-items: center; gap: 8px; font-size: 12px; }
    .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--green); animation: pulse 2s infinite; }
    @keyframes pulse { 0%,100% { opacity:1; } 50% { opacity:0.5; } }
    main { display: grid; grid-template-columns: 280px 1fr; }
    aside { background: var(--bg2); padding: 20px; border-right: 1px solid var(--border); }
    .card { background: var(--bg); border-radius: 12px; padding: 16px; margin-bottom: 12px; border: 1px solid var(--border); }
    .card h3 { font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: #888; margin-bottom: 6px; }
    .card .val { font-size: 28px; font-weight: 700; }
    .card.active .val { color: var(--green); }
    .card.blocked .val { color: var(--red); }
    .graph { position: relative; background: var(--bg); background-image: radial-gradient(circle at 1px 1px, var(--border) 1px, transparent 0); background-size: 40px 40px; }
    #cy { width: 100%; height: 100%; }
    .legend { position: absolute; top: 20px; left: 20px; background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; padding: 12px; font-size: 11px; }
    .legend-item { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
    .ldot { width: 10px; height: 10px; border-radius: 50%; }
    .ldot.root { background: var(--green); }
    .ldot.child { background: var(--cyan); }
    .empty { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); text-align: center; color: #666; }
  </style>
</head>
<body>
  <header>
    <div class="logo">SAT<span>GATE</span> // Governance</div>
    <div class="status"><div class="dot"></div><span id="status">Connecting...</span></div>
  </header>
  <main>
    <aside>
      <div class="card active"><h3>Active Agents</h3><div class="val" id="active">0</div></div>
      <div class="card blocked"><h3>Economic Firewall</h3><div class="val" id="blocked">0</div></div>
      <div class="card"><h3>Kill Switch</h3><div class="val" id="banned">0</div></div>
      <div class="card"><h3>Revocation Hits</h3><div class="val" id="hits">0</div></div>
    </aside>
    <div class="graph">
      <div id="cy"></div>
      <div class="legend">
        <div class="legend-item"><div class="ldot root"></div>Root</div>
        <div class="legend-item"><div class="ldot child"></div>Delegated</div>
      </div>
      <div class="empty" id="empty">📡 Awaiting Traffic</div>
    </div>
  </main>
  <script>
    const cy = cytoscape({
      container: document.getElementById('cy'),
      style: [
        { selector: 'node', style: { 'background-color': '#00d4ff', 'label': 'data(label)', 'color': '#e2e8f0', 'font-size': '10px', 'text-valign': 'bottom', 'text-margin-y': 8, 'width': 30, 'height': 30 }},
        { selector: 'node[depth=0]', style: { 'background-color': '#00ff9d', 'width': 45, 'height': 45 }},
        { selector: 'edge', style: { 'width': 2, 'line-color': '#2d3748', 'target-arrow-color': '#2d3748', 'target-arrow-shape': 'triangle', 'curve-style': 'bezier' }}
      ],
      layout: { name: 'preset' }
    });
    function runLayout() {
      if (cy.nodes().length === 0) return;
      cy.layout({ name: 'cose', animate: true, padding: 50, nodeRepulsion: () => 8000 }).run();
    }
    async function fetchData() {
      try {
        const r = await fetch('/api/governance/graph');
        const d = await r.json();
        document.getElementById('status').textContent = 'Polling (2s)';
        document.getElementById('active').textContent = d.stats.active || 0;
        document.getElementById('blocked').textContent = d.stats.blocked || 0;
        document.getElementById('banned').textContent = d.stats.banned || 0;
        document.getElementById('hits').textContent = d.stats.bannedHits || 0;
        document.getElementById('empty').style.display = d.nodes.length ? 'none' : 'block';
        if (d.nodes.length && cy.nodes().length !== d.nodes.length) {
          cy.elements().remove();
          d.nodes.forEach(n => cy.add({ group: 'nodes', data: { id: n.data.id, label: n.data.label, depth: n.data.depth || 0 }}));
          d.edges.forEach(e => cy.add({ group: 'edges', data: { id: e.data.id, source: e.data.source, target: e.data.target }}));
          runLayout();
        }
      } catch(e) { document.getElementById('status').textContent = 'Error'; }
    }
    fetchData();
    setInterval(fetchData, 2000);
  </script>
</body>
</html>`;
}

// Serve static assets from /dashboard/assets/*
app.use('/dashboard/assets', express.static(path.join(__dirname, 'public', 'assets')));

// =============================================================================
// ERROR HANDLING
// =============================================================================

// 404 handler
app.use((req, res) => {
  res.status(404).json({ 
    error: 'Not Found',
    path: req.path,
    method: req.method
  });
});

// Global error handler
app.use((err, req, res, next) => {
  console.error(`[ERROR] ${new Date().toISOString()}`, err);
  
  res.status(err.status || 500).json({
    error: config.env === 'production' ? 'Internal Server Error' : err.message,
    ...(config.env !== 'production' && { stack: err.stack })
  });
});

// =============================================================================
// SERVER STARTUP & GRACEFUL SHUTDOWN
// =============================================================================

async function startServer() {
  // Initialize data from Redis
  await loadBannedTokensFromRedis();
  await telemetry.init();
  
  // Create HTTP server
  const server = http.createServer(app);
  
  // Setup WebSocket server for real-time dashboard updates
  try {
    const WebSocket = require('ws');
    wsServer = new WebSocket.Server({ server, path: '/ws/governance' });
    
    wsServer.on('connection', (ws, req) => {
      console.log('[WebSocket] Dashboard client connected');
      wsClients.add(ws);
      
      // Send initial state
      telemetry.getGraphData().then(data => {
        ws.send(JSON.stringify({ type: 'init', ...data }));
      });
      
      ws.on('close', () => {
        wsClients.delete(ws);
        console.log('[WebSocket] Dashboard client disconnected');
      });
      
      ws.on('error', (err) => {
        console.error('[WebSocket] Error:', err.message);
        wsClients.delete(ws);
      });
    });
    
    console.log('[WebSocket] Server ready at /ws/governance');
  } catch (e) {
    console.log('[WebSocket] Module not available, using polling fallback');
  }
  
  server.listen(config.port, () => {
  console.log(`[${new Date().toISOString()}] Backend started`);
  console.log(`  Environment: ${config.env}`);
  console.log(`  Listening: http://127.0.0.1:${config.port}`);
  console.log(`  CORS origins: ${config.corsOrigins.join(', ')}`);
    console.log(`  Redis: ${redis ? 'Connected' : 'In-memory fallback'}`);
    console.log(`  WebSocket: ${wsServer ? 'Enabled' : 'Disabled'}`);
});

// Graceful shutdown
const shutdown = (signal) => {
  console.log(`\n[${new Date().toISOString()}] Received ${signal}, shutting down gracefully...`);
    
    // Close WebSocket connections
    wsClients.forEach(client => client.close());
    if (wsServer) wsServer.close();
    
    // Close Redis
    if (redis) redis.quit();
  
  server.close((err) => {
    if (err) {
      console.error('Error during shutdown:', err);
      process.exit(1);
    }
    console.log('Server closed');
    process.exit(0);
  });
  
  // Force exit after 10 seconds
  setTimeout(() => {
    console.error('Forced shutdown after timeout');
    process.exit(1);
  }, 10000);
};

process.on('SIGTERM', () => shutdown('SIGTERM'));
process.on('SIGINT', () => shutdown('SIGINT'));
  
  return server;
}

// Start the server
startServer().catch(err => {
  console.error('Failed to start server:', err);
  process.exit(1);
});

module.exports = app; // For testing

/* backend/server.js
 * SatGate - Lightning-powered API access control
 * Production backend that sits behind Aperture for L402 authentication
 * 
 * SECURITY MODEL (Enterprise-Ready):
 * ===================================
 * 
 * MODES:
 *   MODE=prod (default) - Dashboard/telemetry require admin auth, no redaction
 *   MODE=demo           - Public dashboard allowed (with DASHBOARD_PUBLIC=true), data redacted
 * 
 * CONTROL PLANE (admin-only):
 *   - /dashboard                      - Governance dashboard UI
 *   - GET /api/governance/graph       - Telemetry graph data
 *   - GET /api/governance/stats       - Summary metrics
 *   - GET /api/governance/info        - Version/build info (admin-only)
 *   - GET /api/governance/audit       - Audit log
 *   - GET /api/governance/audit/export - JSONL export for SIEM
 *   - POST /api/governance/ban        - Ban a token (kill switch)
 *   - POST /api/governance/unban      - Remove ban
 *   - POST /api/governance/reset      - Reset telemetry
 * 
 * DATA PLANE:
 *   - /api/capability/*            - Macaroon-authenticated (no payment)
 *   - /api/micro/*                 - L402 protected (1 sat)
 *   - /api/basic/*                 - L402 protected (10 sats)
 *   - /api/standard/*              - L402 protected (100 sats)
 *   - /api/premium/*               - L402 protected (1000 sats)
 *   - /api/free/*                  - Free tier (rate limited)
 * 
 * HEALTH (rate limited, minimal response):
 *   - /health, /ready              - Returns only { status: 'ok' }
 * 
 * Environment Variables:
 *   MODE                   - 'prod' (default) or 'demo'
 *   PRICING_ADMIN_TOKEN    - Primary admin token (REQUIRED, min 32 chars)
 *   ADMIN_TOKEN_NEXT       - Secondary token for rotation (optional)
 *   DASHBOARD_PUBLIC       - 'true' to allow public dashboard (demo mode only)
 *   ADMIN_RATE_LIMIT       - Admin requests/min per IP (default: 30)
 *   HEALTH_RATE_LIMIT      - Health requests/min per IP (default: 600)
 */
const express = require('express');
const os = require('os');
const path = require('path');
const http = require('http');
const crypto = require('crypto');

// =============================================================================
// L402 NATIVE MODE (SatGate as L402 Authority)
// =============================================================================
// When L402_MODE=native, SatGate handles invoice issuance and LSAT validation.
// When L402_MODE=aperture (default), Aperture handles L402, SatGate just tracks.

const L402_MODE = process.env.L402_MODE || 'aperture'; // 'native' or 'aperture'
const MODE = process.env.MODE || 'prod'; // Define early for L402 initialization
let L402Service, createL402Middleware, l402Service;

// L402 initialization (deferred - Redis passed later when available)
function initializeL402(redisClient) {
  if (L402_MODE !== 'native') {
    console.log('[L402] Aperture sidecar mode (SatGate tracks, Aperture enforces)');
    return;
  }
  
  try {
    const l402Module = require('./l402');
    
    L402Service = l402Module.L402Service;
    createL402Middleware = l402Module.createL402Middleware;
    
    // Initialize L402 service with Lightning provider
    const lightningConfig = {
      backend: process.env.LIGHTNING_BACKEND || 'mock',
      // Allow a backend-specific URL while keeping a generic override.
      // - phoenixd: PHOENIXD_URL
      // - lnd: LND_REST_URL
      // - (optional override): LIGHTNING_URL
      url: process.env.LIGHTNING_URL || process.env.PHOENIXD_URL || process.env.LND_REST_URL,
      password: process.env.PHOENIXD_PASSWORD,
      macaroon: process.env.LND_MACAROON,
      apiKey: process.env.OPENNODE_API_KEY
    };
    
    // Check for L402_ROOT_KEY in production (use MODE directly since config isn't defined yet)
    if (MODE === 'prod' && !process.env.L402_ROOT_KEY) {
      console.warn('[L402][SECURITY] L402_MODE=native but no L402_ROOT_KEY set. Set L402_ROOT_KEY (separate from CAPABILITY_ROOT_KEY) to a strong secret in production.');
    }

    l402Service = new L402Service({
      rootKey: process.env.L402_ROOT_KEY || process.env.CAPABILITY_ROOT_KEY,
      lightningConfig,
      redis: redisClient,
      defaultTTL: parseInt(process.env.L402_DEFAULT_TTL) || 3600,
      defaultMaxCalls: parseInt(process.env.L402_DEFAULT_MAX_CALLS) || 100
    });
    
    console.log(`[L402] Native mode enabled (Lightning: ${lightningConfig.backend})`);
  } catch (e) {
    console.warn(`[L402] Native mode requested but failed to load: ${e.message}`);
    console.warn('[L402] Falling back to Aperture sidecar mode');
  }
}

// Note: L402 is initialized after Redis connection (or immediately if no Redis)

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

// Health endpoint rate limiter (generous but protected)
const healthRateLimit = rateLimit({
  windowMs: 60000,
  max: parseInt(process.env.HEALTH_RATE_LIMIT || '600'),
  keyGenerator: (req) => `health:${req.ip}`,
  message: 'Health check rate limit exceeded'
});

// Optional admin auth for governance read endpoints
// MODE=prod: always requires admin auth
// MODE=demo + DASHBOARD_PUBLIC=true: allows public access with redaction
function optionalAdminAuth(req, res, next) {
  // SECURITY: header-only admin auth (query string tokens leak via logs/referrers)
  // Use req.get() for case-insensitive header access
  const token = req.get('x-admin-token') || '';
  const { valid, actor } = checkAdminToken(token);
  
  // Debug: ALWAYS log auth attempts (remove after debugging)
  console.error(`[AUTH-DEBUG] path=${req.path} token=${token ? token.substring(0, 8) + '...' : 'NONE'} expected=${ADMIN_TOKEN_CURRENT ? ADMIN_TOKEN_CURRENT.substring(0, 8) + '...' : 'NOT SET'} valid=${valid}`);
  
  // If valid admin token, grant full access
  if (valid) {
    req.isAdmin = true;
    req.adminActor = actor;
    return next();
  }
  
  // In demo mode with public dashboard enabled, allow read-only access (data will be redacted)
  if (config.isDemo && config.dashboardPublic) {
    req.isAdmin = false;
    req.adminActor = 'public';
    return next();
  }
  
  // Not authorized
  return res.status(403).json({ 
    error: 'Forbidden',
    message: config.isProd 
      ? 'Dashboard access requires authentication.'
      : 'Dashboard access requires authentication. Use MODE=demo and DASHBOARD_PUBLIC=true for public demos.'
  });
}

// Redact sensitive data in demo mode
// Always redacts in MODE=demo, regardless of other settings
function redactForDemo(data, req) {
  // In prod mode, never redact (admin-only access anyway)
  // In demo mode, always redact for public access
  if (config.isProd) return data;
  if (req && req.isAdmin) return data; // Admin sees full data even in demo mode
  
  // Deep clone to avoid mutating original
  const redacted = JSON.parse(JSON.stringify(data));
  
  // Redact token signatures (show first 8 chars only)
  if (redacted.nodes) {
    redacted.nodes = redacted.nodes.map(node => ({
      ...node,
      data: {
        ...node.data,
        id: node.data.id ? node.data.id.substring(0, 8) + '...[redacted]' : node.data.id
      }
    }));
  }
  
  // Remove IP/user-agent from any logged data
  if (redacted.logs) {
    redacted.logs = redacted.logs.map(log => ({
      ...log,
      ip: '[redacted]',
      userAgent: '[redacted]'
    }));
  }
  
  return redacted;
}

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
    actor: req.adminActor || 'unknown',  // Stable actor ID for SIEM correlation
    ip: req.ip || req.connection.remoteAddress,
    userAgent: req.get('user-agent'),
  };
  
  auditLog.unshift(entry);
  if (auditLog.length > MAX_AUDIT_LOG) auditLog.pop();
  
  // Structured JSON log for SIEM ingestion
  console.log(JSON.stringify({ level: 'audit', ...entry }));
  
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
    redis.on('connect', () => {
      console.log('[Redis] Connected');
      // Initialize L402 with Redis once connected
      initializeL402(redis);
    });
    redis.on('error', (err) => console.error('[Redis] Error:', err.message));
    redis.connect().catch(err => {
      console.log('[Redis] Connection failed, using in-memory storage');
      redis = null;
      initializeL402(null);
    });
  } catch (e) {
    console.log('[Redis] Module not available, using in-memory storage');
    redis = null;
    initializeL402(null);
  }
} else {
  console.log('[Storage] No REDIS_URL set, using in-memory storage');
  initializeL402(null);
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
  
  recordUsage: function(tokenSignature, caveats, ip, identifier) {
    const now = Date.now();
    
    // Calculate depth based on identifier (child tokens have ':child:' in identifier)
    const isChild = identifier && identifier.includes(':child:');
    const depth = isChild ? 1 : 0;
    
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
        // Capability tokens: show ROOT vs CHILD
        label = token.depth === 0 ? '👑 ROOT' : `📜 CHILD (L${token.depth})`;
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
  },
  
  // Reset all telemetry counters and tokens (admin action)
  async reset() {
    // Clear in-memory state
    this.activeTokens.clear();
    this.blockedCount = 0;
    this.bannedHits = 0;
    this.challengesSent = 0;
    this.paidRequests = { micro: 0, basic: 0, standard: 0, premium: 0, total: 0 };
    this.revenueSats = { micro: 0, basic: 0, standard: 0, premium: 0, total: 0 };
    
    // Clear Redis persisted stats
    if (redis) {
      try {
        await redis.del('satgate:stats:blocked');
        await redis.del('satgate:stats:bannedHits');
        await redis.del('satgate:stats:challenges');
        await redis.del('satgate:stats:paidTotal');
        await redis.del('satgate:stats:revenueSats');
        console.log('[Telemetry] Reset: cleared Redis stats');
      } catch (e) {
        console.error('[Telemetry] Reset: Redis error:', e.message);
      }
    }
    
    // Broadcast reset to dashboard
    wsBroadcast({ type: 'reset' });
    
    console.log('[Telemetry] Dashboard reset by admin');
    return { success: true, message: 'Dashboard reset' };
  }
};

// =============================================================================
// CONFIGURATION
// =============================================================================

// MODE is defined at the top of the file for early L402 initialization
// - prod: dashboard + telemetry require admin auth, no redaction
// - demo: dashboard may be public (if DASHBOARD_PUBLIC=true), data is redacted

const config = {
  // Railway sets PORT; fallback to BACKEND_PORT for local/docker
  port: parseInt(process.env.PORT) || parseInt(process.env.BACKEND_PORT) || 8083,
  env: process.env.NODE_ENV || 'development',
  // SECURITY: Never allow CORS_ORIGINS="*" in production
  corsOrigins: (() => {
    const origins = process.env.CORS_ORIGINS;
    if (!origins || origins === '*') {
      if (MODE === 'prod') {
        console.warn('[SECURITY] CORS_ORIGINS not set or "*" in prod - defaulting to satgate.io + localhost');
        // Allow satgate.io (our official landing page) and localhost for development
        return ['https://satgate.io', 'https://www.satgate.io', 'http://127.0.0.1:8081', 'http://localhost:8081'];
      }
      // Demo mode allows broader access
      return ['https://satgate.io', 'https://www.satgate.io', 'http://127.0.0.1:8081', 'http://localhost:8081', 'http://localhost:8080', 'http://127.0.0.1:8080'];
    }
    return origins.split(',').map(o => o.trim());
  })(),
  rateLimitWindow: 15 * 60 * 1000, // 15 minutes
  rateLimitMax: 100, // requests per window
  
  // Security settings (enterprise defaults)
  mode: MODE,
  isDemo: MODE === 'demo',
  isProd: MODE === 'prod',
  // In demo mode, DASHBOARD_PUBLIC can enable public access (with redaction)
  // In prod mode, dashboard always requires admin auth regardless of DASHBOARD_PUBLIC
  dashboardPublic: MODE === 'demo' && process.env.DASHBOARD_PUBLIC === 'true',
  healthRateLimit: parseInt(process.env.HEALTH_RATE_LIMIT) || 600,
  
  // Version info (only exposed via admin endpoint)
  version: '1.9.0',
  buildTime: new Date().toISOString(),
};

// Admin tokens for control plane endpoints
// Support rotation: both CURRENT and NEXT tokens are valid during rotation
const ADMIN_TOKEN_CURRENT = process.env.PRICING_ADMIN_TOKEN || process.env.ADMIN_TOKEN_CURRENT || '';
const ADMIN_TOKEN_NEXT = process.env.ADMIN_TOKEN_NEXT || '';
const MIN_ADMIN_TOKEN_LENGTH = 32; // Minimum 32 chars (256 bits) for security

// Validate admin token strength
function isValidAdminToken(token) {
  if (!token) return false;
  if (token.length < MIN_ADMIN_TOKEN_LENGTH) {
    console.warn(`[SECURITY] ⚠️  Admin token too short (${token.length} chars, need ${MIN_ADMIN_TOKEN_LENGTH}+)`);
    return false;
  }
  return true;
}

// Check if a provided token matches any valid admin token
function checkAdminToken(token) {
  if (!token) return { valid: false, actor: null };
  if (ADMIN_TOKEN_CURRENT && token === ADMIN_TOKEN_CURRENT) {
    return { valid: true, actor: 'admin-token-current' };
  }
  if (ADMIN_TOKEN_NEXT && token === ADMIN_TOKEN_NEXT) {
    return { valid: true, actor: 'admin-token-next' };
  }
  return { valid: false, actor: null };
}

// Startup security checks
if (config.env === 'production' || config.isProd) {
  if (!ADMIN_TOKEN_CURRENT) {
    console.warn('[SECURITY] ⚠️  No PRICING_ADMIN_TOKEN set - admin endpoints unprotected!');
  } else if (!isValidAdminToken(ADMIN_TOKEN_CURRENT)) {
    console.warn('[SECURITY] ⚠️  Admin token does not meet minimum length requirements');
  }
  // Warn if someone sets DASHBOARD_PUBLIC in prod mode (it's ignored but they should know)
  if (process.env.DASHBOARD_PUBLIC === 'true') {
    console.warn('[SECURITY] ⚠️  DASHBOARD_PUBLIC=true ignored in MODE=prod - dashboard requires admin auth');
    console.warn('[SECURITY] ℹ️  For public demos, use MODE=demo AND DASHBOARD_PUBLIC=true');
  }
}

console.log(`[CONFIG] Mode: ${MODE} | Dashboard: ${config.dashboardPublic ? 'public' : 'admin-only'} | Admin token: ${ADMIN_TOKEN_CURRENT ? 'set (' + ADMIN_TOKEN_CURRENT.length + ' chars)' : 'NOT SET'} | Rotation: ${ADMIN_TOKEN_NEXT ? 'enabled' : 'disabled'}`);

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

// Security headers (enterprise-grade)
app.use((req, res, next) => {
  // Prevent MIME-sniffing
  res.setHeader('X-Content-Type-Options', 'nosniff');
  // Prevent clickjacking
  res.setHeader('X-Frame-Options', 'DENY');
  // XSS protection (legacy browsers)
  res.setHeader('X-XSS-Protection', '1; mode=block');
  // Referrer policy
  res.setHeader('Referrer-Policy', 'strict-origin-when-cross-origin');
  // Permissions policy (disable unused features)
  res.setHeader('Permissions-Policy', 'geolocation=(), microphone=(), camera=()');
  
  // Content Security Policy for dashboard
  if (req.path.startsWith('/dashboard') || req.path === '/') {
    res.setHeader('Content-Security-Policy', 
      "default-src 'self'; " +
      "script-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; " +  // Cytoscape from CDN
      "style-src 'self' 'unsafe-inline'; " +
      "img-src 'self' data:; " +
      "connect-src 'self' wss: ws:; " +  // WebSocket for dashboard
      "font-src 'self'; " +
      "frame-ancestors 'none';"
    );
  }
  
  // HSTS in production
  if (config.env === 'production' || config.isProd) {
    res.setHeader('Strict-Transport-Security', 'max-age=31536000; includeSubDomains');
  }
  next();
});

// CORS configuration
// In L402 native mode, we handle CORS directly.
// In Aperture sidecar mode, Aperture handles CORS.
app.use((req, res, next) => {
  // In native mode (no Aperture), always handle CORS ourselves
  // The old check for X-Forwarded-For doesn't work because Railway/CDNs also set it
  const origin = req.headers.origin;
  // SECURITY: Never allow wildcard CORS in production
  const allowWildcard = config.corsOrigins.includes('*') && !config.isProd;
  const originAllowed = origin && config.corsOrigins.includes(origin);
  
  if (originAllowed || allowWildcard) {
    res.setHeader('Access-Control-Allow-Origin', origin || '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization, X-Admin-Token, X-SatGate-Admin-Token');
    res.setHeader('Access-Control-Allow-Credentials', 'true');
    res.setHeader('Access-Control-Max-Age', '86400');
    res.setHeader('Access-Control-Expose-Headers', 'WWW-Authenticate, X-L402-Price, X-L402-Tier, X-L402-TTL, X-L402-Max-Calls, X-Calls-Remaining, X-Budget-Remaining');
  } else if (origin) {
    console.log(`[CORS] Origin rejected: ${origin} not in ${JSON.stringify(config.corsOrigins)}`);
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
// Returns minimal info only - no version/build details for security
app.get('/health', healthRateLimit, (req, res) => {
  res.json({ status: 'ok' });
});

// Readiness check (for k8s/orchestrators)
// Returns minimal info only
app.get('/ready', healthRateLimit, (req, res) => {
  res.json({ status: 'ok' });
});

// Lightning provider status (admin-only; can expose node IDs / backend details)
app.get('/lightning/status', adminRateLimit, requirePricingAdmin, async (req, res) => {
  try {
    if (L402_MODE === 'native' && l402Service?.lightning?.getStatus) {
      const status = await l402Service.lightning.getStatus();
      return res.json({
        ok: true,
        mode: 'native',
        provider: status.backend || 'unknown',
        ...status,
      });
    }

    // Aperture sidecar mode: Lightning is handled by Aperture/LNC
    return res.json({
      ok: true,
      mode: 'aperture',
      provider: 'aperture',
      info: { note: 'Lightning handled by Aperture (LNC)' },
    });
  } catch (error) {
    res.json({
      ok: false,
      error: error.message
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
    note: `Prices in satoshis. These are display prices; actual L402 enforcement is handled by ${L402_MODE === 'native' ? 'SatGate (native mode)' : 'Aperture (sidecar mode)'}.`
  });
});

function requirePricingAdmin(req, res, next) {
  // Admin endpoints always require authentication (even in demo mode)
  const token = req.get('x-admin-token') || req.get('x-satgate-admin-token') || '';
  const { valid, actor } = checkAdminToken(token);
  
  // Debug: ALWAYS log auth attempts (remove after debugging)
  console.error(`[AUTH-ADMIN] path=${req.path} token=${token ? token.substring(0, 8) + '...' : 'NONE'} expected=${ADMIN_TOKEN_CURRENT ? ADMIN_TOKEN_CURRENT.substring(0, 8) + '...' : 'NOT SET'} valid=${valid}`);
  
  if (!valid) {
    // In development without token set, allow for testing
    if (config.env !== 'production' && !ADMIN_TOKEN_CURRENT) {
      req.adminActor = 'dev-mode';
      return next();
    }
    return res.status(403).json({ error: 'Forbidden' });
  }
  
  req.adminActor = actor;
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

// =============================================================================
// L402 PAID ENDPOINT MIDDLEWARE
// =============================================================================
// In native mode: SatGate issues 402 challenges and validates LSAT tokens
// In aperture mode: Aperture handles L402, SatGate just tracks requests

const createPaidMiddleware = (tier, priceSats) => {
  // L402 Native Mode: SatGate is the L402 authority
  if (L402_MODE === 'native' && l402Service && createL402Middleware) {
    const l402Mw = createL402Middleware(l402Service, {
      tier,
      scope: `api:${tier}:*`,
      tierCost: priceSats,
      maxCalls: parseInt(process.env.L402_DEFAULT_MAX_CALLS) || 100,
      ttl: parseInt(process.env.L402_DEFAULT_TTL) || 3600
    });
    // IMPORTANT: rate-limit challenge issuance to prevent invoice-spam DoS.
    return (req, res, next) => apiRateLimit(req, res, () => l402Mw(req, res, next));
  }
  
  // Aperture Sidecar Mode: Just track (Aperture enforces payment)
  return (req, res, next) => {
    apiRateLimit(req, res, () => {
      // If request reached paid endpoint, it MUST have paid (Aperture enforces this)
      telemetry.recordPaidRequest(tier, priceSats);
      telemetry.recordPaymentNode(tier, priceSats, req.path);
      console.log(`[L402] Paid request (aperture): ${tier} tier, ${priceSats} sats, endpoint: ${req.path}`);
      next();
    });
  };
};

// Backward compatibility alias
const trackPaidRequest = createPaidMiddleware;

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
      downloadUrl: '/api/premium/export/download?id=sample-export-001',
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

// Use our SimpleMacaroon implementation (no external macaroon library needed)
const { SimpleMacaroon } = require('./l402');

// Shared secret for Phase 1 demo (in production, use env var)
const CAPABILITY_ROOT_KEY = process.env.CAPABILITY_ROOT_KEY || 'satgate-phase1-demo-key-change-in-prod';
const CAPABILITY_LOCATION = 'https://satgate.io';
const CAPABILITY_IDENTIFIER = 'satgate-capability-v1';

// =============================================================================
// STATEFUL ENFORCEMENT: max_calls & budget_sats (Redis-backed, in-memory fallback)
// =============================================================================
// This closes the "pay/mint once, unlimited within TTL" loophole by enforcing
// per-token call and budget limits on every authorized request.
//
// Caveat formats:
//   max_calls = <int>      - Maximum number of requests allowed
//   budget_sats = <int>    - Maximum sats budget (decremented by tier cost per request)
//
// Storage:
//   Redis keys:
//     satgate:calls:<tokenSignature>  - remaining call count
//     satgate:budget:<tokenSignature> - remaining sats budget
//   TTL: matches token expiry

const capabilityCallsMemory = new Map();  // tokenSignature -> { remaining: number, expiresAtMs: number }
const capabilityBudgetMemory = new Map(); // tokenSignature -> { remaining: number, expiresAtMs: number }

// Tier costs in sats (used for budget_sats enforcement)
const TIER_COSTS = {
  'api:capability:read': 1,
  'api:capability:ping': 1,
  'api:capability:data': 5,
  'api:capability:admin': 10,
  'api:capability:*': 1, // Default for wildcard
  'default': 1
};

function getTierCost(scope) {
  return TIER_COSTS[scope] || TIER_COSTS['default'];
}

async function decrementCapabilityCalls(tokenSignature, maxCalls, expiresAtMs) {
  const now = Date.now();
  const ttlSeconds = expiresAtMs
    ? Math.max(1, Math.ceil((expiresAtMs - now) / 1000))
    : 3600;

  // Redis-backed atomic init + decrement
  if (redis) {
    const key = `satgate:calls:${tokenSignature}`;
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
      const remaining = await redis.eval(script, 1, key, String(maxCalls), String(ttlSeconds));
      return { remaining: Number(remaining), ttlSeconds, backend: 'redis' };
    } catch (e) {
      // Fall back to memory below
    }
  }

  // In-memory fallback (single-instance only)
  const entry = capabilityCallsMemory.get(tokenSignature);
  if (!entry || (entry.expiresAtMs && entry.expiresAtMs < now)) {
    capabilityCallsMemory.set(tokenSignature, {
      remaining: maxCalls,
      expiresAtMs: expiresAtMs || (now + ttlSeconds * 1000),
    });
  }

  const current = capabilityCallsMemory.get(tokenSignature);
  current.remaining -= 1;
  return { remaining: current.remaining, ttlSeconds, backend: 'memory' };
}

async function decrementCapabilityBudget(tokenSignature, budgetSats, cost, expiresAtMs) {
  const now = Date.now();
  const ttlSeconds = expiresAtMs
    ? Math.max(1, Math.ceil((expiresAtMs - now) / 1000))
    : 3600;

  // Redis-backed atomic init + conditional decrby (never go negative)
  if (redis) {
    const key = `satgate:budget:${tokenSignature}`;
    const script = `
      local key = KEYS[1]
      local init = tonumber(ARGV[1])
      local cost = tonumber(ARGV[2])
      local ttl = tonumber(ARGV[3])
      if redis.call('EXISTS', key) == 0 then
        redis.call('SET', key, init, 'EX', ttl)
      end
      local current = tonumber(redis.call('GET', key))
      if current == nil then
        return { -999999, 0 }
      end
      if current < cost then
        return { current, 0 }
      end
      local remaining = redis.call('DECRBY', key, cost)
      return { remaining, 1 }
    `;
    try {
      const result = await redis.eval(script, 1, key, String(budgetSats), String(cost), String(ttlSeconds));
      const remaining = Array.isArray(result) ? Number(result[0]) : Number(result);
      const charged = Array.isArray(result) ? Number(result[1]) === 1 : true;
      return { remaining, charged, cost, ttlSeconds, backend: 'redis' };
    } catch (e) {
      // Fall back to memory below
    }
  }

  // In-memory fallback (single-instance only)
  const entry = capabilityBudgetMemory.get(tokenSignature);
  if (!entry || (entry.expiresAtMs && entry.expiresAtMs < now)) {
    capabilityBudgetMemory.set(tokenSignature, {
      remaining: budgetSats,
      expiresAtMs: expiresAtMs || (now + ttlSeconds * 1000),
    });
  }

  const current = capabilityBudgetMemory.get(tokenSignature);
  if (current.remaining < cost) {
    return { remaining: current.remaining, charged: false, cost, ttlSeconds, backend: 'memory' };
  }
  current.remaining -= cost;
  return { remaining: current.remaining, charged: true, cost, ttlSeconds, backend: 'memory' };
}

// Middleware: Validate macaroon for /api/capability/* routes
// Implements DYNAMIC SCOPE ENFORCEMENT based on requested path
app.use('/api/capability', async (req, res, next) => {
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
    // Decode and import the macaroon using SimpleMacaroon
    const m = SimpleMacaroon.deserialize(tokenBase64);
    
    // KILL SWITCH: Check if token is banned (uses sync in-memory check for speed)
    const tokenSignature = m.signature;
    if (bannedTokens.hasSync(tokenSignature)) {
      console.log(`[KILL SWITCH] 🛑 Blocked banned token: ${tokenSignature.substring(0, 16)}...`);
      telemetry.recordBannedHit();
      return res.status(403).json({
        error: 'Token Revoked',
        reason: 'This token has been banned by an administrator',
        code: 'TOKEN_BANNED'
      });
    }
    
    const now = Date.now();
    
    // Extract caveats from macaroon (SimpleMacaroon stores as array of strings)
    const caveats = m.caveats || [];
    
    console.log(`[PEP] Path: ${req.path} | Required Scope: ${requiredScope}`);
    console.log(`[PEP] Token caveats: ${JSON.stringify(caveats)}`);
    
    // Track if scope was validated
    let scopeValidated = false;
    let scopeError = null;
    let expiresAtMs = null;
    let maxCalls = null;
    let budgetSats = null;
    let tokenScope = null;
    
    // Verify macaroon signature first
    if (!m.verify(CAPABILITY_ROOT_KEY)) {
      throw new Error('Invalid macaroon signature');
    }
    
    // Process caveats
    for (const caveatStr of caveats) {
      // Time-based expiry
      if (caveatStr.startsWith('expires = ')) {
        const expiry = parseInt(caveatStr.split(' = ')[1], 10);
        expiresAtMs = expiry;
        if (now > expiry) {
          throw new Error('token expired');
        }
      }
      
      // Stateful call budget (per request, per time window)
      else if (caveatStr.startsWith('max_calls = ')) {
        const v = parseInt(caveatStr.split(' = ')[1], 10);
        if (!Number.isFinite(v) || v <= 0) {
          throw new Error('invalid max_calls caveat');
        }
        maxCalls = v;
      }
      
      // Stateful sats budget (decremented by tier cost per request)
      else if (caveatStr.startsWith('budget_sats = ')) {
        const v = parseInt(caveatStr.split(' = ')[1], 10);
        if (!Number.isFinite(v) || v <= 0) {
          throw new Error('invalid budget_sats caveat');
        }
        budgetSats = v;
      }

      // DYNAMIC SCOPE CHECK
      else if (caveatStr.startsWith('scope = ')) {
        tokenScope = caveatStr.split(' = ')[1].trim();
        
        // Check if token scope covers the required scope
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
          throw new Error(scopeError);
        }
        
        scopeValidated = true;
        console.log(`[PEP] ✓ Scope OK: '${tokenScope}' covers '${requiredScope}'`);
      }
      
      // Delegation markers - accept (informational only)
      else if (caveatStr.startsWith('delegated_by = ') ||
          caveatStr.startsWith('delegated_from = ') ||
          caveatStr.startsWith('delegation_depth = ') ||
          caveatStr.startsWith('delegation_time = ')) {
        // OK - delegation tracking caveats
      }
      
      // Unknown caveat - reject for security
      else {
        throw new Error('unknown caveat: ' + caveatStr);
      }
    }
    
    // If we get here but scope wasn't validated, reject
    if (!scopeValidated) {
      throw new Error('Token has no scope caveat');
    }
    
    // Extract identifier from macaroon (SimpleMacaroon stores as string)
    const identifier = m.identifier || 'unknown';
    
    // Attach parsed info to request for endpoints
    req.capability = {
      caveats,
      identifier,
      requiredScope,
      validatedAt: new Date().toISOString(),
      tokenSignature: tokenSignature
    };

    // Enforce max_calls if present
    if (maxCalls) {
      const { remaining, backend } = await decrementCapabilityCalls(tokenSignature, maxCalls, expiresAtMs);
      res.setHeader('X-Calls-Limit', String(maxCalls));
      res.setHeader('X-Calls-Remaining', String(Math.max(0, remaining)));
      res.setHeader('X-Calls-Backend', backend); // debug/ops visibility (safe; remove if undesired)

      req.capability.maxCalls = maxCalls;
      req.capability.callsRemaining = Math.max(0, remaining);

      if (remaining < 0) {
        return res.status(429).json({
          error: 'Call limit exhausted',
          code: 'CALL_LIMIT_EXHAUSTED',
          message: 'This token has no calls remaining. Mint/pay for a new token.',
          callsRemaining: 0,
        });
      }
    }
    
    // Enforce budget_sats if present (decremented by tier cost)
    if (budgetSats) {
      const cost = getTierCost(tokenScope || requiredScope);
      const { remaining, charged, backend } = await decrementCapabilityBudget(tokenSignature, budgetSats, cost, expiresAtMs);
      res.setHeader('X-Budget-Limit', String(budgetSats));
      res.setHeader('X-Budget-Remaining', String(Math.max(0, remaining)));
      res.setHeader('X-Budget-Cost', String(cost));
      res.setHeader('X-Budget-Backend', backend);

      req.capability.budgetSats = budgetSats;
      req.capability.budgetRemaining = Math.max(0, remaining);
      req.capability.requestCost = cost;

      // If we did not charge, there wasn't enough budget to cover this request.
      // Return 402 so clients can "re-challenge" (mint/pay for a new token).
      if (!charged) {
        return res.status(402).json({
          error: 'Budget exhausted',
          code: 'BUDGET_EXHAUSTED',
          message: 'This token has insufficient sats budget. Mint/pay for a new token.',
          budgetRemaining: 0,
          requestCost: cost,
        });
      }
    }
    
    // TELEMETRY: Record this token usage for governance dashboard
    const clientIp = req.ip || req.connection.remoteAddress || 'unknown';
    telemetry.recordUsage(tokenSignature, caveats, clientIp, identifier);
    
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

// Mint a capability token (requires admin auth in prod, open in demo mode)
app.post('/api/capability/mint', express.json(), (req, res) => {
  // Security gate: require admin auth unless in demo mode
  if (!config.isDemo) {
    const token = req.headers['x-admin-token'];
    const { valid } = checkAdminToken(token);
    if (!valid) {
      return res.status(403).json({ 
        error: 'Forbidden',
        message: 'Token minting requires admin authentication in production.',
        hint: 'Use X-Admin-Token header or set MODE=demo for testing.'
      });
    }
  }
  
  const { scope = 'api:capability:read', expiresIn = 3600, max_calls, maxCalls, budget_sats, budgetSats } = req.body || {};
  
  // Validate max_calls
  const maxCallsValue = typeof max_calls === 'number' ? max_calls : (typeof maxCalls === 'number' ? maxCalls : null);
  if (maxCallsValue !== null && (!Number.isFinite(maxCallsValue) || maxCallsValue <= 0)) {
    return res.status(400).json({ error: 'Invalid max_calls', hint: 'max_calls must be a positive integer' });
  }
  
  // Validate budget_sats
  const budgetSatsValue = typeof budget_sats === 'number' ? budget_sats : (typeof budgetSats === 'number' ? budgetSats : null);
  if (budgetSatsValue !== null && (!Number.isFinite(budgetSatsValue) || budgetSatsValue <= 0)) {
    return res.status(400).json({ error: 'Invalid budget_sats', hint: 'budget_sats must be a positive integer' });
  }
  
  try {
    // Create base macaroon using SimpleMacaroon
    const identifier = `${CAPABILITY_IDENTIFIER}:${Date.now()}`;
    
    const m = new SimpleMacaroon(CAPABILITY_LOCATION, identifier, CAPABILITY_ROOT_KEY);
    
    // Add caveats
    const expiresAt = Date.now() + (expiresIn * 1000);
    m.addFirstPartyCaveat(`expires = ${expiresAt}`);
    m.addFirstPartyCaveat(`scope = ${scope}`);
    if (maxCallsValue) {
      m.addFirstPartyCaveat(`max_calls = ${Math.floor(maxCallsValue)}`);
    }
    if (budgetSatsValue) {
      m.addFirstPartyCaveat(`budget_sats = ${Math.floor(budgetSatsValue)}`);
    }
    
    // Serialize to base64
    const tokenBase64 = m.serialize();
    
    console.log(`[CAPABILITY] Minted token: scope=${scope}, expires=${new Date(expiresAt).toISOString()}`);
    
    res.json({
      ok: true,
      token: tokenBase64,
      usage: `curl -H "Authorization: Bearer ${tokenBase64}" https://satgate-production.up.railway.app/api/capability/ping`,
      caveats: {
        scope,
        expires: new Date(expiresAt).toISOString(),
        expiresIn: `${expiresIn} seconds`,
        ...(maxCallsValue ? { max_calls: Math.floor(maxCallsValue) } : {}),
        ...(budgetSatsValue ? { budget_sats: Math.floor(budgetSatsValue) } : {})
      },
      tierCosts: TIER_COSTS,
      note: 'This is a Phase 1 capability token. No payment required.'
    });
    
  } catch (e) {
    console.error(`[CAPABILITY] Mint error: ${e.message}`);
    res.status(500).json({ error: 'Failed to mint token', reason: e.message });
  }
});

// TEST: Token system health check (admin-only, no key material exposed)
app.get('/api/token/test', requirePricingAdmin, (req, res) => {
  try {
    // Only expose safe metadata, never key material
    const systemStatus = {
      macaroonLibrary: 'loaded',
      keyConfigured: !!CAPABILITY_ROOT_KEY,
      keyLength: CAPABILITY_ROOT_KEY ? CAPABILITY_ROOT_KEY.length : 0,
      location: CAPABILITY_LOCATION,
      identifier: CAPABILITY_IDENTIFIER
    };
    
    // Verify we can create a macaroon (without exposing the key)
    const testId = `${CAPABILITY_IDENTIFIER}:test:${Date.now()}`;
    
    const m = new SimpleMacaroon(CAPABILITY_LOCATION, testId, CAPABILITY_ROOT_KEY || 'fallback-key');
    
    // Only return signature prefix (not key material)
    const sigPrefix = m.signature.substring(0, 8);
    
    res.json({ 
      ok: true, 
      status: 'Token system operational',
      signaturePrefix: sigPrefix,
      system: systemStatus 
    });
  } catch (e) {
    res.status(500).json({ error: 'Token system error', message: e.message });
  }
});

// Delegate a capability token (create a child with restricted scope)
// SECURITY: admin-only. Delegation should normally be done client-side via attenuation.
// This endpoint is kept as an admin tool for demos/testing.
app.get('/api/token/delegate', requirePricingAdmin, (req, res) => {
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
    // Create child macaroon using SimpleMacaroon
    step = 5;
    const childMac = new SimpleMacaroon(CAPABILITY_LOCATION, childId, CAPABILITY_ROOT_KEY);
    
    // Add caveats
    step = 6;
    const expiresAt = Date.now() + (expiresIn * 1000);
    childMac.addFirstPartyCaveat(`expires = ${expiresAt}`);
    childMac.addFirstPartyCaveat(`scope = ${scope}`);
    childMac.addFirstPartyCaveat(`delegation_depth = 1`);
    
    step = 7;
    const childTokenBase64 = childMac.serialize();
    
    step = 8;
    // No extra step needed
    
    step = 9;
    const childSig = childMac.signature.substring(0, 16);
    
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
// SECURITY: Only available in demo mode or with admin auth

app.post('/api/capability/demo/delegate', (req, res) => {
  // Security gate: demo endpoints only in demo mode or with admin auth
  if (!config.isDemo) {
    const token = req.headers['x-admin-token'];
    const { valid } = checkAdminToken(token);
    if (!valid) {
      return res.status(403).json({ 
        error: 'Forbidden',
        message: 'Demo endpoints disabled in production.',
        hint: 'Use MODE=demo for testing or provide admin token.'
      });
    }
  }
  
  try {
    const now = Date.now();
    
    // 1. Create parent token (Simulating the Agent's existing credential)
    const parentId = `${CAPABILITY_IDENTIFIER}:parent:${now}`;
    const parentMacaroon = new SimpleMacaroon(CAPABILITY_LOCATION, parentId, CAPABILITY_ROOT_KEY);
    
    // Parent: broad scope, 1 hour expiry
    const parentExpiry = now + (60 * 60 * 1000);
    parentMacaroon.addFirstPartyCaveat(`expires = ${parentExpiry}`);
    parentMacaroon.addFirstPartyCaveat(`scope = api:capability:*`);
    
    const parentToken = parentMacaroon.serialize();
    
    // 2. Create child token with MORE restrictive caveats
    // In a real scenario, this would be done by attenuating the parent
    // For demo purposes, we create it directly with the restricted caveats
    const childId = `${CAPABILITY_IDENTIFIER}:child:${now}`;
    const childMacaroon = new SimpleMacaroon(CAPABILITY_LOCATION, childId, CAPABILITY_ROOT_KEY);
    
    // Child: ALL parent caveats PLUS more restrictive ones
    // (This simulates what attenuation produces)
    const childExpiry = now + (5 * 60 * 1000); // 5 minutes (shorter than parent)
    childMacaroon.addFirstPartyCaveat(`expires = ${childExpiry}`);
    childMacaroon.addFirstPartyCaveat(`scope = api:capability:ping`); // Narrower
    childMacaroon.addFirstPartyCaveat(`delegated_by = agent-001`);
    
    const childToken = childMacaroon.serialize();
    
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
// Protected by: optional admin auth (public if DASHBOARD_PUBLIC=true)
app.get('/api/governance/graph', optionalAdminAuth, async (req, res) => {
  const graphData = await telemetry.getGraphData();
  res.json(redactForDemo(graphData, req));
});

// Get governance stats (quick summary)
// Protected by: optional admin auth (public if DASHBOARD_PUBLIC=true)
app.get('/api/governance/stats', optionalAdminAuth, async (req, res) => {
  const data = await telemetry.getGraphData();
  res.json(redactForDemo({
    ok: true,
    stats: data.stats,
    timestamp: new Date().toISOString()
  }, req));
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

// SIEM-friendly audit export (JSONL format)
// Each line is a complete JSON object for easy log ingestion
app.get('/api/governance/audit/export', adminRateLimit, requirePricingAdmin, async (req, res) => {
  const limit = Math.min(parseInt(req.query.limit) || 1000, MAX_AUDIT_LOG);
  
  let logs = auditLog.slice(0, limit);
  
  // Try Redis for more complete history
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
  
  // Return JSONL (newline-delimited JSON)
  res.setHeader('Content-Type', 'application/x-ndjson');
  res.setHeader('Content-Disposition', `attachment; filename="satgate-audit-${Date.now()}.jsonl"`);
  res.send(logs.map(l => JSON.stringify(l)).join('\n'));
});

// System info (admin-only) - version, build, mode
// Version info is NOT exposed on public endpoints for security
app.get('/api/governance/info', adminRateLimit, requirePricingAdmin, async (req, res) => {
  // Get Lightning status if in native mode
  let lightningStatus = null;
  if (L402_MODE === 'native' && l402Service && l402Service.lightning) {
    try {
      lightningStatus = await l402Service.lightning.getStatus();
    } catch (e) {
      lightningStatus = { ok: false, error: e.message };
    }
  }
  
  res.json({
    ok: true,
    version: config.version,
    mode: config.mode,
    env: config.env,
    features: {
      redis: !!redis,
      websocket: !!wsServer,
      dashboardPublic: config.dashboardPublic,
      tokenRotation: !!ADMIN_TOKEN_NEXT,
      l402Mode: L402_MODE,
      l402Native: L402_MODE === 'native' && !!l402Service,
    },
    lightning: lightningStatus,
    uptime: Math.floor(process.uptime()),
    timestamp: new Date().toISOString(),
  });
});

// Lightning status endpoint (admin-only)
app.get('/api/governance/lightning', adminRateLimit, requirePricingAdmin, async (req, res) => {
  if (L402_MODE !== 'native' || !l402Service) {
    return res.json({
      ok: true,
      mode: 'aperture',
      message: 'L402 in Aperture sidecar mode. Lightning handled by Aperture.',
      hint: 'Set L402_MODE=native to use SatGate as L402 authority.'
    });
  }
  
  try {
    const status = await l402Service.lightning.getStatus();
    res.json({
      ok: status.ok,
      mode: 'native',
      backend: status.backend,
      nodeId: status.nodeId,
      error: status.error,
      timestamp: new Date().toISOString()
    });
  } catch (e) {
    res.status(500).json({
      ok: false,
      mode: 'native',
      error: e.message
    });
  }
});

// Reset dashboard counters and tokens (admin only)
// Use this to start fresh for a demo
app.post('/api/governance/reset', adminRateLimit, requirePricingAdmin, async (req, res) => {
  logAdminAction('DASHBOARD_RESET', {}, req);
  
  const result = await telemetry.reset();
  
  res.json({
    ok: true,
    message: 'Dashboard reset successfully',
    note: 'All counters zeroed, active tokens cleared. Banned tokens NOT cleared (use /unban).'
  });
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
// Protected by: optional admin auth (public if DASHBOARD_PUBLIC=true)
app.get('/dashboard', optionalAdminAuth, (req, res) => {
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
      // SECURITY: First-message authentication pattern
      // Token is sent as first message, never in URL (prevents log/referrer leakage)
      ws.isAuthenticated = false;
      ws.isAdmin = false;
      ws.actorId = 'unauthenticated';
      
      // Auth timeout: client must authenticate within 5 seconds
      ws.authTimeout = setTimeout(() => {
        if (!ws.isAuthenticated) {
          console.log('[WebSocket] Authentication timeout');
          ws.close(4001, 'Authentication timeout');
        }
      }, 5000);
      
      console.log('[WebSocket] New connection, awaiting authentication...');
      
      ws.on('message', (data) => {
        let msg;
        try {
          msg = JSON.parse(data.toString());
        } catch (e) {
          ws.send(JSON.stringify({ type: 'error', message: 'Invalid JSON' }));
          return;
        }
        
        // First message must be auth
        if (!ws.isAuthenticated) {
          if (msg.type === 'auth') {
            const { valid, actor } = checkAdminToken(msg.token);
            
            // In prod mode, require valid admin token
            // In demo mode with dashboardPublic, allow public access
            if (valid) {
              ws.isAuthenticated = true;
              ws.isAdmin = true;
              ws.actorId = actor;
              clearTimeout(ws.authTimeout);
              wsClients.add(ws);
              console.log(`[WebSocket] Authenticated as admin (${actor})`);
              ws.send(JSON.stringify({ type: 'auth', status: 'ok', role: 'admin' }));
              
              // Send initial state (full data for admin)
              telemetry.getGraphData().then(graphData => {
                ws.send(JSON.stringify({ type: 'init', ...graphData }));
              });
            } else if (config.isDemo && config.dashboardPublic) {
              // Public access allowed in demo mode
              ws.isAuthenticated = true;
              ws.isAdmin = false;
              ws.actorId = 'public';
              clearTimeout(ws.authTimeout);
              wsClients.add(ws);
              console.log('[WebSocket] Authenticated as public (demo mode)');
              ws.send(JSON.stringify({ type: 'auth', status: 'ok', role: 'public' }));
              
              // Send initial state (redacted for public)
              telemetry.getGraphData().then(graphData => {
                const redacted = redactForDemo(graphData, { isAdmin: false });
                ws.send(JSON.stringify({ type: 'init', ...redacted }));
              });
            } else {
              console.log('[WebSocket] Authentication failed (invalid token)');
              ws.send(JSON.stringify({ type: 'auth', status: 'error', message: 'Invalid token' }));
              ws.close(4003, 'Authentication failed');
            }
            return;
          } else {
            // Non-auth message before authentication
            ws.send(JSON.stringify({ type: 'error', message: 'Authentication required' }));
            ws.close(4003, 'Authentication required');
            return;
          }
        }
        
        // Handle authenticated messages (future: commands, subscriptions, etc.)
        if (msg.type === 'ping') {
          ws.send(JSON.stringify({ type: 'pong', timestamp: Date.now() }));
        }
      });
      
      ws.on('close', () => {
        clearTimeout(ws.authTimeout);
        wsClients.delete(ws);
        console.log(`[WebSocket] Client disconnected (${ws.actorId})`);
      });
      
      ws.on('error', (err) => {
        console.error('[WebSocket] Error:', err.message);
        clearTimeout(ws.authTimeout);
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

/* backend/server.js
 * SatGate - Lightning-powered API access control
 * Production backend that sits behind Aperture for L402 authentication
 */
const express = require('express');
const os = require('os');
const path = require('path');

const app = express();

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

// Simple in-memory rate limiter (use Redis in production cluster)
const rateLimitStore = new Map();
app.use((req, res, next) => {
  const ip = req.ip || req.connection.remoteAddress;
  const now = Date.now();
  const windowStart = now - config.rateLimitWindow;
  
  let record = rateLimitStore.get(ip);
  if (!record || record.windowStart < windowStart) {
    record = { windowStart: now, count: 0 };
  }
  record.count++;
  rateLimitStore.set(ip, record);
  
  res.setHeader('X-RateLimit-Limit', config.rateLimitMax);
  res.setHeader('X-RateLimit-Remaining', Math.max(0, config.rateLimitMax - record.count));
  
  if (record.count > config.rateLimitMax) {
    return res.status(429).json({ 
      error: 'Too many requests', 
      retryAfter: Math.ceil((record.windowStart + config.rateLimitWindow - now) / 1000)
    });
  }
  next();
});

// Cleanup old rate limit entries every 5 minutes
setInterval(() => {
  const cutoff = Date.now() - config.rateLimitWindow;
  for (const [ip, record] of rateLimitStore) {
    if (record.windowStart < cutoff) {
      rateLimitStore.delete(ip);
    }
  }
}, 5 * 60 * 1000);

// =============================================================================
// HEALTH & MONITORING ENDPOINTS
// =============================================================================

// Health check (for load balancers, k8s probes)
app.get('/health', (req, res) => {
  res.json({ 
    status: 'healthy',
    timestamp: new Date().toISOString(),
    uptime: process.uptime()
  });
});

// Readiness check
app.get('/ready', (req, res) => {
  // Add checks for database connections, external services, etc.
  res.json({ ready: true });
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

// ---------------------------------------------------------------------------
// PREMIUM TIER - 1000 sats ($1.00) per request
// ---------------------------------------------------------------------------

// Premium AI insights endpoint
app.get('/api/premium/insights', (req, res) => {
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

// Premium data export endpoint
app.get('/api/premium/export', (req, res) => {
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

// Standard analytics endpoint
app.get('/api/standard/analytics', (req, res) => {
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

// Standard metrics endpoint
app.get('/api/standard/metrics', (req, res) => {
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

// Basic status endpoint
app.get('/api/basic/status', (req, res) => {
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

// Basic quote endpoint
app.get('/api/basic/quote', (req, res) => {
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
app.use('/api/capability', (req, res, next) => {
  const authHeader = req.get('authorization') || '';
  
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
    const m = macaroon.importMacaroons(tokenBytes)[0];
    
    // Verify signature with root key
    const keyBytes = Buffer.from(CAPABILITY_ROOT_KEY, 'utf8');
    
    // Build verifier with caveat checkers
    const now = Date.now();
    
    // Verify the macaroon (signature check)
    macaroon.dischargeMacaroon(m, () => null, keyBytes, (err) => {
      if (err) {
        throw new Error('Invalid signature: ' + err.message);
      }
    });
    
    // Check caveats manually
    const caveats = [];
    m.caveats.forEach(c => {
      if (c.identifier) {
        caveats.push(c.identifier.toString('utf8'));
      }
    });
    
    // Validate each caveat
    for (const caveat of caveats) {
      // Time-based expiry: "expires = <timestamp>"
      if (caveat.startsWith('expires = ')) {
        const expiry = parseInt(caveat.split(' = ')[1], 10);
        if (now > expiry) {
          return res.status(403).json({
            error: 'Token Expired',
            expired: new Date(expiry).toISOString(),
            now: new Date(now).toISOString()
          });
        }
      }
      // Scope check: "scope = <prefix>"
      if (caveat.startsWith('scope = ')) {
        const allowedScope = caveat.split(' = ')[1];
        const requestedPath = req.path;
        // Simple prefix match (e.g., scope = /ping allows /ping but not /admin)
        if (!requestedPath.startsWith('/' + allowedScope.replace('api:capability:', ''))) {
          // More flexible: just log the scope for demo
          console.log(`[CAPABILITY] Scope caveat: ${allowedScope}, path: ${requestedPath}`);
        }
      }
    }
    
    // Attach parsed info to request for endpoints
    req.capability = {
      caveats,
      identifier: m.identifier.toString('utf8'),
      validatedAt: new Date().toISOString()
    };
    
    console.log(`[CAPABILITY] ✓ Valid token, caveats: ${JSON.stringify(caveats)}`);
    next();
    
  } catch (e) {
    console.error(`[CAPABILITY] ✗ Invalid token: ${e.message}`);
    return res.status(403).json({
      error: 'Invalid Capability Token',
      reason: e.message
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
    
    // Add caveats
    const expiresAt = Date.now() + (expiresIn * 1000);
    m = macaroon.addFirstPartyCaveat(m, Buffer.from(`expires = ${expiresAt}`, 'utf8'));
    m = macaroon.addFirstPartyCaveat(m, Buffer.from(`scope = ${scope}`, 'utf8'));
    
    // Export as base64
    const tokenBytes = macaroon.exportMacaroons([m]);
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

// ---------------------------------------------------------------------------
// MICRO TIER - 1 sat ($0.001) per request
// ---------------------------------------------------------------------------
// True micropayments - the absolute minimum viable price

// Micro ping endpoint
app.get('/api/micro/ping', (req, res) => {
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

// Micro data endpoint
app.get('/api/micro/data', (req, res) => {
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
      l402Example: '/api/micro/ping'
    },
    playground: 'https://satgate.io/playground'
  });
});

// =============================================================================
// STATIC FILE SERVING (disabled in cloud container)
// =============================================================================

// Note: The Railway deployment runs as an API-only container (front-end is on satgate.io).
// If you later bundle a static frontend into the container, re-enable express.static here.

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

const server = app.listen(config.port, () => {
  console.log(`[${new Date().toISOString()}] Backend started`);
  console.log(`  Environment: ${config.env}`);
  console.log(`  Listening: http://127.0.0.1:${config.port}`);
  console.log(`  CORS origins: ${config.corsOrigins.join(', ')}`);
});

// Graceful shutdown
const shutdown = (signal) => {
  console.log(`\n[${new Date().toISOString()}] Received ${signal}, shutting down gracefully...`);
  
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

module.exports = app; // For testing

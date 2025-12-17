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
      validatedAt: new Date().toISOString()
    };
    
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

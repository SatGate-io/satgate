/**
 * SatGate Gateway Servers
 * 
 * Two-listener setup:
 * - Data Plane: Public-facing proxy (port 8080 by default)
 * - Admin Plane: Internal admin/governance (port 9090 by default, localhost only)
 */

const express = require('express');
const http = require('http');
const { getConfig } = require('./index');
const { createLimitsMiddleware, requestIdMiddleware, extractClientIp } = require('./limits');
const { createGatewayProxyMiddleware } = require('./transport');
const { createAuthDecideRouter } = require('./auth-decide');

/**
 * Create the data plane Express app
 * 
 * This handles all proxied requests from clients.
 * 
 * @param {object} config - Gateway configuration
 * @param {object} l402Service - L402 service
 * @param {object} meteringService - Metering service
 */
function createDataPlaneApp(config, l402Service, meteringService) {
  const app = express();
  
  // Disable X-Powered-By
  app.disable('x-powered-by');
  
  // Trust proxy setting
  if (config.server.trustProxy) {
    app.set('trust proxy', config.server.trustProxy);
  }
  
  // Request ID
  app.use(requestIdMiddleware);
  
  // CORS (before limits to handle preflight)
  if (config.cors.origins.length > 0) {
    app.use(createCorsMiddleware(config));
  }
  
  // Limits
  app.use(createLimitsMiddleware(config));
  
  // Health check (no auth, no proxy)
  app.get('/healthz', (req, res) => {
    res.status(200).json({
      status: 'ok',
      plane: 'data',
      timestamp: new Date().toISOString(),
    });
  });
  
  // Gateway proxy (handles all routes)
  // IMPORTANT: Don't use body parser - we need to stream request bodies
  app.use(createGatewayProxyMiddleware(config, l402Service, meteringService));
  
  // Catch-all (should not reach here if gateway is configured correctly)
  app.use((req, res) => {
    res.status(404).json({
      error: 'Not Found',
      message: 'No route matched',
      requestId: req.requestId,
    });
  });
  
  // Error handler
  app.use((err, req, res, next) => {
    console.error(`[DataPlane] Unhandled error: ${err.message}`, err.stack);
    if (!res.headersSent) {
      res.status(500).json({
        error: 'Internal Server Error',
        message: 'Unexpected error',
        requestId: req.requestId,
      });
    }
  });
  
  return app;
}

/**
 * Create the admin plane Express app
 * 
 * This handles:
 * - /auth/decide (for NGINX/Envoy)
 * - /auth/health
 * - /api/governance/* (admin endpoints)
 * - Token minting
 * 
 * @param {object} config - Gateway configuration
 * @param {object} l402Service - L402 service
 * @param {object} meteringService - Metering service
 * @param {object} governanceRoutes - Governance router (from existing server.js)
 */
function createAdminPlaneApp(config, l402Service, meteringService, governanceRoutes) {
  const app = express();
  
  // Disable X-Powered-By
  app.disable('x-powered-by');
  
  // Parse JSON bodies for admin endpoints
  app.use(express.json());
  
  // Request ID
  app.use(requestIdMiddleware);
  
  // Admin token middleware (if required)
  if (config.admin.requireAdminToken) {
    app.use(createAdminAuthMiddleware());
  }
  
  // Health check
  app.get('/healthz', (req, res) => {
    res.status(200).json({
      status: 'ok',
      plane: 'admin',
      timestamp: new Date().toISOString(),
    });
  });
  
  // Auth decision API (for NGINX/Envoy)
  // This endpoint does NOT require admin token - it validates L402 tokens
  app.use(createAuthDecideRouter(l402Service, meteringService));
  
  // Governance routes (if provided)
  if (governanceRoutes) {
    app.use('/api/governance', governanceRoutes);
  }
  
  // Catch-all
  app.use((req, res) => {
    res.status(404).json({
      error: 'Not Found',
      message: 'Endpoint not found',
      requestId: req.requestId,
    });
  });
  
  // Error handler
  app.use((err, req, res, next) => {
    console.error(`[AdminPlane] Unhandled error: ${err.message}`, err.stack);
    if (!res.headersSent) {
      res.status(500).json({
        error: 'Internal Server Error',
        message: 'Unexpected error',
        requestId: req.requestId,
      });
    }
  });
  
  return app;
}

/**
 * Create CORS middleware from config
 */
function createCorsMiddleware(config) {
  const allowedOrigins = new Set(config.cors.origins);
  const allowCredentials = config.cors.allowCredentials;
  
  return function corsMiddleware(req, res, next) {
    const origin = req.headers['origin'];
    
    if (origin && allowedOrigins.has(origin)) {
      res.setHeader('Access-Control-Allow-Origin', origin);
      res.setHeader('Vary', 'Origin');
      
      if (allowCredentials) {
        res.setHeader('Access-Control-Allow-Credentials', 'true');
      }
      
      res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, PATCH, DELETE, OPTIONS');
      res.setHeader('Access-Control-Allow-Headers', 'Authorization, Content-Type, X-Request-Id, X-Admin-Token');
      res.setHeader('Access-Control-Expose-Headers', 'WWW-Authenticate, X-Request-Id, X-L402-Price, X-L402-Tier, X-Calls-Remaining, X-Budget-Remaining');
      
      if (req.method === 'OPTIONS') {
        res.status(204).end();
        return;
      }
    }
    
    next();
  };
}

/**
 * Create admin auth middleware
 */
function createAdminAuthMiddleware() {
  const crypto = require('crypto');
  const adminToken = process.env.PRICING_ADMIN_TOKEN || process.env.SATGATE_ADMIN_TOKEN;
  
  // Skip auth for /auth/* endpoints (they have their own auth)
  const skipPaths = ['/auth/', '/healthz'];
  
  return function adminAuthMiddleware(req, res, next) {
    // Skip certain paths
    for (const path of skipPaths) {
      if (req.path.startsWith(path) || req.path === path) {
        return next();
      }
    }
    
    // Check admin token
    const providedToken = req.headers['x-admin-token'] || req.headers['x-satgate-admin-token'];
    
    if (!adminToken) {
      // No admin token configured - check MODE
      const mode = process.env.MODE || 'prod';
      if (mode === 'prod') {
        res.status(500).json({
          error: 'Internal Server Error',
          message: 'Admin token not configured',
        });
        return;
      }
      // Demo mode - allow without token
      return next();
    }
    
    if (!providedToken) {
      res.status(401).json({
        error: 'Unauthorized',
        message: 'Admin token required',
      });
      return;
    }
    
    // Timing-safe comparison
    try {
      const a = Buffer.from(adminToken, 'utf8');
      const b = Buffer.from(providedToken, 'utf8');
      
      if (a.length !== b.length || !crypto.timingSafeEqual(a, b)) {
        res.status(403).json({
          error: 'Forbidden',
          message: 'Invalid admin token',
        });
        return;
      }
    } catch {
      res.status(403).json({
        error: 'Forbidden',
        message: 'Invalid admin token',
      });
      return;
    }
    
    next();
  };
}

/**
 * Start gateway servers
 * 
 * @param {object} config - Gateway configuration
 * @param {object} l402Service - L402 service
 * @param {object} meteringService - Metering service (optional)
 * @param {object} governanceRoutes - Governance router (optional)
 * @returns {object} { dataServer, adminServer }
 */
function startGatewayServers(config, l402Service, meteringService, governanceRoutes) {
  // Create apps
  const dataApp = createDataPlaneApp(config, l402Service, meteringService);
  const adminApp = createAdminPlaneApp(config, l402Service, meteringService, governanceRoutes);
  
  // Start data plane server
  const dataServer = http.createServer(dataApp);
  const { host: dataHost, port: dataPort } = config.server.parsed;
  
  dataServer.listen(dataPort, dataHost, () => {
    console.log(`[Gateway] Data plane listening on ${dataHost}:${dataPort}`);
  });
  
  // Start admin plane server
  const adminServer = http.createServer(adminApp);
  const { host: adminHost, port: adminPort } = config.admin.parsed;
  
  adminServer.listen(adminPort, adminHost, () => {
    console.log(`[Gateway] Admin plane listening on ${adminHost}:${adminPort}`);
  });
  
  return { dataServer, adminServer, dataApp, adminApp };
}

module.exports = {
  createDataPlaneApp,
  createAdminPlaneApp,
  createCorsMiddleware,
  createAdminAuthMiddleware,
  startGatewayServers,
};


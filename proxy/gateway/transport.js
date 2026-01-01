/**
 * SatGate Proxy Transport
 * 
 * Streaming HTTP/HTTPS proxy using Node.js native modules.
 * 
 * Design principles:
 * - Stream request/response bodies without buffering
 * - Respect upstream timeouts
 * - Handle backpressure correctly
 * - Never allow SSRF (upstream selected only from config)
 */

const http = require('http');
const https = require('https');
const { sanitizeRequestHeaders, sanitizeResponseHeaders, extractClientIp } = require('./limits');

/**
 * Proxy a request to the upstream
 * 
 * @param {object} req - Express request
 * @param {object} res - Express response
 * @param {object} upstreamConfig - Upstream configuration
 * @param {object} satgateContext - SatGate context (route, policy, metering)
 */
async function proxyRequest(req, res, upstreamConfig, satgateContext) {
  const requestId = satgateContext.requestId || req.requestId;
  const DEBUG = process.env.GATEWAY_DEBUG === 'true';
  
  // Build upstream URL
  const upstreamUrl = new URL(upstreamConfig.url);
  upstreamUrl.pathname = req.url.split('?')[0];
  upstreamUrl.search = req.url.includes('?') ? req.url.substring(req.url.indexOf('?')) : '';
  
  // Choose protocol
  const transport = upstreamUrl.protocol === 'https:' ? https : http;
  
  // Sanitize headers
  const upstreamHeaders = sanitizeRequestHeaders(req.headers, upstreamConfig, satgateContext);
  
  // Build request options
  const options = {
    method: req.method,
    hostname: upstreamUrl.hostname,
    port: upstreamUrl.port || (upstreamUrl.protocol === 'https:' ? 443 : 80),
    path: upstreamUrl.pathname + upstreamUrl.search,
    headers: upstreamHeaders,
    timeout: upstreamConfig.timeoutMs,
    // For https, we may need to handle self-signed certs in internal networks
    // This should be configurable per-upstream
    rejectUnauthorized: upstreamConfig.tlsVerify !== false,
  };
  
  if (DEBUG) {
    console.log(`[Gateway] Proxying ${req.method} ${req.url} -> ${upstreamUrl.toString()}`);
  }
  
  return new Promise((resolve, reject) => {
    // Create upstream request
    const proxyReq = transport.request(options, (proxyRes) => {
      // Sanitize response headers
      const responseHeaders = sanitizeResponseHeaders(proxyRes.headers, upstreamConfig);
      
      // Add SatGate response headers
      responseHeaders['x-request-id'] = requestId;
      if (satgateContext.tier) {
        responseHeaders['x-satgate-tier'] = satgateContext.tier;
      }
      if (satgateContext.scope) {
        responseHeaders['x-satgate-scope'] = satgateContext.scope;
      }
      if (satgateContext.callsRemaining !== undefined) {
        responseHeaders['x-calls-remaining'] = String(satgateContext.callsRemaining);
      }
      if (satgateContext.budgetRemaining !== undefined) {
        responseHeaders['x-budget-remaining'] = String(satgateContext.budgetRemaining);
      }
      
      // Set response status and headers
      res.writeHead(proxyRes.statusCode, responseHeaders);
      
      // Stream response body
      proxyRes.pipe(res);
      
      proxyRes.on('end', () => {
        if (DEBUG) {
          console.log(`[Gateway] Completed ${req.method} ${req.url} -> ${proxyRes.statusCode}`);
        }
        resolve();
      });
      
      proxyRes.on('error', (err) => {
        console.error(`[Gateway] Upstream response error: ${err.message}`);
        reject(err);
      });
    });
    
    // Handle upstream connection errors
    proxyReq.on('error', (err) => {
      console.error(`[Gateway] Upstream request error: ${err.message}`);
      
      if (!res.headersSent) {
        res.status(502).json({
          error: 'Bad Gateway',
          message: 'Failed to reach upstream',
          requestId,
        });
      }
      resolve(); // Don't reject, we've handled the error
    });
    
    // Handle timeout
    proxyReq.on('timeout', () => {
      console.error(`[Gateway] Upstream timeout after ${upstreamConfig.timeoutMs}ms`);
      proxyReq.destroy();
      
      if (!res.headersSent) {
        res.status(504).json({
          error: 'Gateway Timeout',
          message: 'Upstream did not respond in time',
          timeout: upstreamConfig.timeoutMs,
          requestId,
        });
      }
      resolve();
    });
    
    // Handle client disconnect
    req.on('close', () => {
      if (!proxyReq.destroyed) {
        proxyReq.destroy();
      }
    });
    
    // Stream request body
    if (req.readable) {
      req.pipe(proxyReq);
    } else {
      // Request body may have been consumed by body parser
      // In gateway mode, we should NOT use body parser
      proxyReq.end();
    }
  });
}

/**
 * Create gateway proxy middleware
 * 
 * This is the main data plane handler for Gateway Mode.
 * It matches routes, enforces policies, and proxies to upstreams.
 * 
 * @param {object} config - Gateway configuration
 * @param {object} l402Service - L402 service
 * @param {object} meteringService - Metering service
 */
function createGatewayProxyMiddleware(config, l402Service, meteringService) {
  const { matchRoute } = require('./index');
  const { parseL402Token, buildWWWAuthenticate } = require('./auth-decide');
  
  return async function gatewayProxyMiddleware(req, res, next) {
    const requestId = req.requestId || req.headers['x-request-id'];
    
    // Match route
    const route = matchRoute(config, req);
    
    if (!route) {
      // No matching route - fail closed
      res.status(403).json({
        error: 'Forbidden',
        message: 'No matching route',
        requestId,
      });
      return;
    }
    
    const policy = route.policy;
    const upstream = route.upstream ? config.upstreams[route.upstream] : null;
    
    // Build context
    const satgateContext = {
      requestId,
      clientIp: extractClientIp(req, config.server.trustProxy),
      host: req.headers['host'],
      proto: req.headers['x-forwarded-proto'] || (req.secure ? 'https' : 'http'),
      route: route.name,
      tier: policy.tier,
      scope: policy.scope,
    };
    
    // Handle policy
    switch (policy.kind) {
      case 'public':
        if (!upstream) {
          res.status(500).json({
            error: 'Internal Server Error',
            message: 'Route has no upstream',
            requestId,
          });
          return;
        }
        return proxyRequest(req, res, upstream, satgateContext);
        
      case 'deny':
        res.status(policy.status || 403).json({
          error: 'Forbidden',
          message: 'Access denied by policy',
          route: route.name,
          requestId,
        });
        return;
        
      case 'l402':
        return handleL402(req, res, config, route, policy, upstream, satgateContext, l402Service, meteringService);
        
      case 'capability':
        return handleCapability(req, res, config, route, policy, upstream, satgateContext, meteringService);
        
      default:
        res.status(500).json({
          error: 'Internal Server Error',
          message: 'Unknown policy kind',
          requestId,
        });
        return;
    }
  };
}

/**
 * Handle L402 policy in gateway mode
 */
async function handleL402(req, res, config, route, policy, upstream, satgateContext, l402Service, meteringService) {
  const { parseL402Token, buildWWWAuthenticate } = require('./auth-decide');
  const requestId = satgateContext.requestId;
  
  // Check for L402 token
  const token = parseL402Token(req.headers['authorization']);
  
  if (!token) {
    return issueChallenge(res, policy, l402Service, requestId);
  }
  
  try {
    const validation = await l402Service.validateLSAT(token.macaroon, token.preimage);
    
    if (!validation.valid) {
      return issueChallenge(res, policy, l402Service, requestId);
    }

    const tokenScope = validation.caveats?.scope || null;
    const tokenSig = validation.tokenSignature;
    
    // Check scope
    if (policy.scope && tokenScope !== policy.scope) {
      return issueChallenge(res, policy, l402Service, requestId);
    }
    
    // Check metering
    if (meteringService && (policy.maxCalls || policy.budgetSats)) {
      const meter = await meteringService.check(tokenSig, {
        maxCalls: policy.maxCalls || config.l402.defaultMaxCalls,
        budgetSats: policy.budgetSats || config.l402.defaultBudgetSats,
        costSats: policy.priceSats,
        expiresAtMs: validation.caveats?.expiresAt || null,
      });
      
      if (meter.exhausted) {
        // Re-challenge on exhaustion so clients can pay for a new token window.
        res.setHeader('X-SatGate-Reason', meter.reason === 'calls' ? 'call_exhausted' : 'budget_exhausted');
        return issueChallenge(res, policy, l402Service, requestId);
      }
      
      satgateContext.callsRemaining = meter.callsRemaining;
      satgateContext.budgetRemaining = meter.budgetRemaining;
    }
    
    // Valid - proxy
    if (!upstream) {
      res.status(500).json({
        error: 'Internal Server Error',
        message: 'Route has no upstream',
        requestId,
      });
      return;
    }
    
    return proxyRequest(req, res, upstream, satgateContext);
    
  } catch (err) {
    console.error(`[Gateway] L402 validation error: ${err.message}`);
    return issueChallenge(res, policy, l402Service, requestId);
  }
}

/**
 * Issue L402 challenge
 */
async function issueChallenge(res, policy, l402Service, requestId) {
  try {
    const tier = policy.tier || 'default';
    const challenge = await l402Service.createChallenge(tier, {
      scope: policy.scope,
      ttl: policy.ttlSeconds,
      maxCalls: policy.maxCalls,
      budgetSats: policy.budgetSats,
      priceSats: policy.priceSats,
    });

    for (const [k, v] of Object.entries(challenge.headers || {})) {
      res.setHeader(k, v);
    }
    res.setHeader('X-Request-Id', requestId);
    res.setHeader('Cache-Control', 'no-store');
    res.setHeader('Pragma', 'no-cache');
    
    res.status(402).json({
      ...challenge.body,
      requestId,
    });
    
  } catch (err) {
    console.error(`[Gateway] Challenge creation error: ${err.message}`);
    res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to create payment challenge',
      requestId,
    });
  }
}

/**
 * Handle capability policy
 */
async function handleCapability(req, res, config, route, policy, upstream, satgateContext, meteringService) {
  const requestId = satgateContext.requestId;
  
  // Extract capability token
  const authHeader = req.headers['authorization'];
  const capMatch = authHeader?.match(/^(?:Capability|Bearer)\s+(.+)$/i);
  if (!capMatch) {
    res.status(401).json({
      error: 'Unauthorized',
      message: 'Capability token required',
      hint: 'Authorization: Capability <token> (or Bearer <token>)',
      requestId,
    });
    return;
  }
  
  try {
    const { SimpleMacaroon } = require('../l402');
    const capabilityRootKey = process.env.CAPABILITY_ROOT_KEY || '';
    if (!capabilityRootKey) {
      return res.status(500).json({
        error: 'Internal Server Error',
        message: 'CAPABILITY_ROOT_KEY not configured',
        requestId,
      });
    }

    const mac = SimpleMacaroon.deserialize(capMatch[1].trim());
    if (!mac.verify(capabilityRootKey)) {
      return res.status(401).json({
        error: 'Unauthorized',
        message: 'Invalid capability token',
        requestId,
      });
    }

    // Extract scope + expiry
    const caveats = mac.caveats || [];
    let tokenScope = null;
    let expiresAtMs = null;
    const now = Date.now();
    for (const c of caveats) {
      if (typeof c !== 'string') throw new Error('invalid caveat');
      if (c.startsWith('expires = ')) {
        const v = parseInt(c.split(' = ')[1], 10);
        expiresAtMs = v;
        if (now > v) {
          return res.status(403).json({
            error: 'Forbidden',
            message: 'Token expired',
            requestId,
          });
        }
      } else if (c.startsWith('scope = ')) {
        tokenScope = c.split(' = ')[1].trim();
      } else if (
        c.startsWith('delegated_by = ') ||
        c.startsWith('delegated_from = ') ||
        c.startsWith('delegation_depth = ') ||
        c.startsWith('delegation_time = ')
      ) {
        // allowed informational caveats
      } else {
        throw new Error('unknown caveat');
      }
    }
    if (!tokenScope) {
      return res.status(403).json({
        error: 'Forbidden',
        message: 'Missing scope caveat',
        requestId,
      });
    }
    
    // Check scope
    if (policy.scope && tokenScope !== policy.scope) {
      res.status(403).json({
        error: 'Forbidden',
        message: 'Scope mismatch',
        requestId,
      });
      return;
    }
    
    // Check metering
    if (meteringService && (policy.maxCalls || policy.budgetSats)) {
      const meter = await meteringService.check(mac.signature, {
        maxCalls: policy.maxCalls,
        budgetSats: policy.budgetSats,
        costSats: 1,
        expiresAtMs,
      });
      
      if (meter.exhausted) {
        res.status(429).json({
          error: 'Too Many Requests',
          message: 'Capability call limit exhausted',
          requestId,
        });
        return;
      }
      
      satgateContext.callsRemaining = meter.callsRemaining;
      if (meter.budgetRemaining !== undefined) {
        satgateContext.budgetRemaining = meter.budgetRemaining;
      }
    }
    
    // Valid - proxy
    if (!upstream) {
      res.status(500).json({
        error: 'Internal Server Error',
        message: 'Route has no upstream',
        requestId,
      });
      return;
    }
    
    return proxyRequest(req, res, upstream, satgateContext);
    
  } catch (err) {
    console.error(`[Gateway] Capability validation error: ${err.message}`);
    res.status(401).json({
      error: 'Unauthorized',
      message: 'Invalid capability token',
      requestId,
    });
  }
}

module.exports = {
  proxyRequest,
  createGatewayProxyMiddleware,
};


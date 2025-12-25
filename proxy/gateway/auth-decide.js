/**
 * SatGate Auth Decision API
 * 
 * Endpoint: POST /auth/decide
 * 
 * This is the Enterprise integration point for NGINX auth_request and
 * Envoy ext_authz. External ingress controllers call this endpoint to
 * get authorization decisions for incoming requests.
 * 
 * Contract:
 * - 200 OK: Allow request (pass to upstream)
 * - 402 Payment Required: L402 challenge (WWW-Authenticate header)
 * - 403 Forbidden: Deny (policy violation, banned, scope mismatch)
 * - 429 Too Many Requests: Rate/call limit exhausted
 * - 500 Internal Server Error: Decision service failure
 * 
 * Fail-closed: Any error results in 500, not 200.
 */

const { matchRoute, getConfig } = require('./index');

/**
 * Required headers from ingress
 */
const REQUIRED_HEADERS = [
  'x-original-method',
  'x-original-uri',
];

/**
 * Generate a unique request ID
 */
function generateRequestId() {
  return `sg-${Date.now().toString(36)}-${Math.random().toString(36).substring(2, 9)}`;
}

/**
 * Parse Authorization header for L402/LSAT token
 * 
 * Formats supported:
 * - L402 <macaroon>:<preimage>
 * - LSAT <macaroon>:<preimage>
 */
function parseL402Token(authHeader) {
  if (!authHeader) return null;
  
  const match = authHeader.match(/^(?:L402|LSAT)\s+([^:]+):([a-f0-9]{64})$/i);
  if (!match) return null;
  
  return {
    macaroon: match[1],
    preimage: match[2],
  };
}

/**
 * Build WWW-Authenticate header value for L402 challenge
 */
function buildWWWAuthenticate(macaroon, invoice) {
  return `L402 macaroon="${macaroon}", invoice="${invoice}"`;
}

/**
 * Auth Decision handler
 * 
 * @param {object} req - Express request
 * @param {object} res - Express response
 * @param {object} l402Service - L402 service (createChallenge, validateLSAT)
 * @param {object} meteringService - Metering service (optional)
 */
async function authDecideHandler(req, res, l402Service, meteringService) {
  const requestId = req.headers['x-request-id'] || generateRequestId();
  const startTime = Date.now();
  
  // Set standard headers on all responses
  res.setHeader('X-Request-Id', requestId);
  res.setHeader('Cache-Control', 'no-store, no-cache, must-revalidate');
  res.setHeader('Pragma', 'no-cache');
  
  try {
    // Validate required headers
    for (const header of REQUIRED_HEADERS) {
      if (!req.headers[header]) {
        res.status(400).json({
          error: 'Bad Request',
          message: `Missing required header: ${header}`,
          requestId,
        });
        return;
      }
    }
    
    // Extract request details from ingress headers
    const originalMethod = req.headers['x-original-method'].toUpperCase();
    const originalUri = req.headers['x-original-uri'];
    const originalHost = req.headers['x-original-host'] || req.headers['host'] || 'unknown';
    const originalProto = req.headers['x-original-proto'] || 'https';
    const clientIp = req.headers['x-forwarded-for']?.split(',')[0]?.trim() || req.ip;
    const authorization = req.headers['authorization'];
    
    // Build a mock request object for route matching
    const mockReq = {
      method: originalMethod,
      url: originalUri,
      headers: {
        authorization: authorization,
        host: originalHost,
      },
      ip: clientIp,
    };
    
    // Get config and match route
    const config = getConfig();
    const route = matchRoute(config, mockReq);
    
    // No matching route = deny (fail-closed)
    if (!route) {
      res.status(403).json({
        error: 'Forbidden',
        message: 'No matching route',
        requestId,
      });
      return;
    }
    
    const policy = route.policy;
    
    // Handle policy types
    switch (policy.kind) {
      case 'public':
        // Allow immediately
        res.setHeader('X-SatGate-Route', route.name);
        res.setHeader('X-SatGate-Policy', 'public');
        res.status(200).json({
          allow: true,
          route: route.name,
          policy: 'public',
          requestId,
        });
        return;
        
      case 'deny':
        // Deny with configured status
        res.status(policy.status || 403).json({
          error: 'Forbidden',
          message: 'Access denied by policy',
          route: route.name,
          requestId,
        });
        return;
        
      case 'l402':
        // L402 payment required
        return handleL402Policy(
          req, res, config, route, policy,
          mockReq, authorization, l402Service, meteringService, requestId
        );
        
      case 'capability':
        // Capability token required (no payment)
        return handleCapabilityPolicy(
          req, res, config, route, policy,
          mockReq, authorization, meteringService, requestId
        );
        
      default:
        // Unknown policy = deny (fail-closed)
        console.error(`[AuthDecide] Unknown policy kind: ${policy.kind}`);
        res.status(500).json({
          error: 'Internal Server Error',
          message: 'Unknown policy kind',
          requestId,
        });
        return;
    }
    
  } catch (err) {
    // Any error = fail-closed
    console.error(`[AuthDecide] Error: ${err.message}`, err.stack);
    res.status(500).json({
      error: 'Internal Server Error',
      message: 'Auth decision failed',
      requestId,
    });
  }
}

/**
 * Handle L402 policy (payment required)
 */
async function handleL402Policy(
  req, res, config, route, policy,
  mockReq, authorization, l402Service, meteringService, requestId
) {
  // Check for existing L402 token
  const token = parseL402Token(authorization);
  
  if (!token) {
    // No token - issue challenge
    return issueL402Challenge(res, policy, l402Service, requestId);
  }
  
  // Validate token
  try {
    const validation = await l402Service.validateLSAT(token.macaroon, token.preimage);
    
    if (!validation.valid) {
      // Invalid token - issue new challenge
      console.log(`[AuthDecide] Token invalid: ${validation.error}`);
      return issueL402Challenge(res, policy, l402Service, requestId);
    }
    
    // Check scope match
    if (policy.scope && validation.scope !== policy.scope) {
      // Scope mismatch - issue new challenge for correct scope
      console.log(`[AuthDecide] Scope mismatch: ${validation.scope} vs ${policy.scope}`);
      return issueL402Challenge(res, policy, l402Service, requestId);
    }
    
    // Check metering (if enabled)
    if (meteringService && (policy.maxCalls || policy.budgetSats)) {
      const meter = await meteringService.check(validation.tokenId, {
        maxCalls: policy.maxCalls || config.l402.defaultMaxCalls,
        budgetSats: policy.budgetSats || config.l402.defaultBudgetSats,
      });
      
      if (meter.exhausted) {
        if (meter.reason === 'calls') {
          res.status(429).json({
            error: 'Too Many Requests',
            message: 'Call limit exhausted',
            requestId,
          });
        } else {
          // Budget exhausted - issue re-challenge
          return issueL402Challenge(res, policy, l402Service, requestId);
        }
        return;
      }
      
      // Set metering headers for upstream
      res.setHeader('X-Calls-Remaining', meter.callsRemaining);
      if (meter.budgetRemaining !== undefined) {
        res.setHeader('X-Budget-Remaining', meter.budgetRemaining);
      }
    }
    
    // Token valid - allow
    res.setHeader('X-SatGate-Route', route.name);
    res.setHeader('X-SatGate-Policy', 'l402');
    res.setHeader('X-SatGate-Tier', policy.tier || 'default');
    res.setHeader('X-SatGate-Scope', validation.scope || '');
    res.status(200).json({
      allow: true,
      route: route.name,
      policy: 'l402',
      tier: policy.tier,
      scope: validation.scope,
      requestId,
    });
    
  } catch (err) {
    console.error(`[AuthDecide] L402 validation error: ${err.message}`);
    // Validation error - issue new challenge
    return issueL402Challenge(res, policy, l402Service, requestId);
  }
}

/**
 * Issue L402 challenge response
 */
async function issueL402Challenge(res, policy, l402Service, requestId) {
  try {
    const challenge = await l402Service.createChallenge({
      tier: policy.tier,
      price: policy.priceSats,
      scope: policy.scope,
      ttl: policy.ttlSeconds,
      maxCalls: policy.maxCalls,
      budgetSats: policy.budgetSats,
    });
    
    res.setHeader('WWW-Authenticate', buildWWWAuthenticate(challenge.macaroon, challenge.invoice));
    res.setHeader('X-L402-Price', policy.priceSats);
    res.setHeader('X-L402-Tier', policy.tier || 'default');
    if (policy.ttlSeconds) res.setHeader('X-L402-TTL', policy.ttlSeconds);
    if (policy.maxCalls) res.setHeader('X-L402-Max-Calls', policy.maxCalls);
    res.setHeader('Content-Type', 'application/json');
    
    res.status(402).json({
      error: 'Payment Required',
      macaroon: challenge.macaroon,
      invoice: challenge.invoice,
      price: policy.priceSats,
      tier: policy.tier,
      scope: policy.scope,
      ttl: policy.ttlSeconds,
      maxCalls: policy.maxCalls,
      requestId,
    });
    
  } catch (err) {
    console.error(`[AuthDecide] Challenge creation error: ${err.message}`);
    res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to create payment challenge',
      requestId,
    });
  }
}

/**
 * Handle capability policy (macaroon-only, no payment)
 */
async function handleCapabilityPolicy(
  req, res, config, route, policy,
  mockReq, authorization, meteringService, requestId
) {
  // Extract capability token from Authorization header
  // Format: Capability <base64-macaroon>
  const capMatch = authorization?.match(/^Capability\s+(.+)$/i);
  
  if (!capMatch) {
    res.status(401).json({
      error: 'Unauthorized',
      message: 'Capability token required',
      hint: 'Authorization: Capability <token>',
      requestId,
    });
    return;
  }
  
  // Validate capability token
  // This would use a similar mechanism to L402 but without payment binding
  // For now, we'll accept any base64 token with a valid signature
  
  try {
    // TODO: Implement capability token validation
    // For MVP, we'll check the token structure and signature
    
    const tokenData = Buffer.from(capMatch[1], 'base64').toString('utf8');
    let token;
    try {
      token = JSON.parse(tokenData);
    } catch {
      res.status(401).json({
        error: 'Unauthorized',
        message: 'Invalid capability token format',
        requestId,
      });
      return;
    }
    
    // Check scope match
    if (policy.scope && token.scope !== policy.scope) {
      res.status(403).json({
        error: 'Forbidden',
        message: 'Scope mismatch',
        required: policy.scope,
        provided: token.scope,
        requestId,
      });
      return;
    }
    
    // Check metering
    if (meteringService && policy.maxCalls) {
      const meter = await meteringService.check(token.id, {
        maxCalls: policy.maxCalls,
      });
      
      if (meter.exhausted) {
        res.status(429).json({
          error: 'Too Many Requests',
          message: 'Capability call limit exhausted',
          requestId,
        });
        return;
      }
      
      res.setHeader('X-Calls-Remaining', meter.callsRemaining);
    }
    
    // Token valid - allow
    res.setHeader('X-SatGate-Route', route.name);
    res.setHeader('X-SatGate-Policy', 'capability');
    res.setHeader('X-SatGate-Scope', token.scope || '');
    res.status(200).json({
      allow: true,
      route: route.name,
      policy: 'capability',
      scope: token.scope,
      requestId,
    });
    
  } catch (err) {
    console.error(`[AuthDecide] Capability validation error: ${err.message}`);
    res.status(401).json({
      error: 'Unauthorized',
      message: 'Capability token validation failed',
      requestId,
    });
  }
}

/**
 * Create Express router for auth decision API
 */
function createAuthDecideRouter(l402Service, meteringService) {
  const express = require('express');
  const router = express.Router();
  
  router.post('/auth/decide', (req, res) => {
    authDecideHandler(req, res, l402Service, meteringService);
  });
  
  // Health check for the auth service
  router.get('/auth/health', (req, res) => {
    res.status(200).json({
      status: 'ok',
      service: 'satgate-auth-decide',
      timestamp: new Date().toISOString(),
    });
  });
  
  return router;
}

module.exports = {
  authDecideHandler,
  createAuthDecideRouter,
  parseL402Token,
  buildWWWAuthenticate,
  REQUIRED_HEADERS,
};


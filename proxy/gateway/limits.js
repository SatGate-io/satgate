/**
 * SatGate Gateway Limits & Header Sanitization
 * 
 * Enforces request/response limits and sanitizes headers
 * to prevent abuse and ensure security.
 */

const { hopByHopHeaders, satgateRequestHeaders } = require('./config/defaults');

/**
 * Create limits middleware
 * 
 * @param {object} config - Gateway configuration
 */
function createLimitsMiddleware(config) {
  const limits = config.limits;
  
  return function limitsMiddleware(req, res, next) {
    // Check Content-Length if present
    const contentLength = req.headers['content-length'];
    if (contentLength) {
      const size = parseInt(contentLength, 10);
      if (!isNaN(size) && size > limits.maxRequestBodyBytes) {
        res.status(413).json({
          error: 'Payload Too Large',
          message: `Request body exceeds ${limits.maxRequestBodyBytes} bytes`,
          limit: limits.maxRequestBodyBytes,
          received: size,
        });
        return;
      }
    }
    
    // Estimate headers size (rough check)
    let headersSize = 0;
    for (const [key, value] of Object.entries(req.headers)) {
      headersSize += key.length + (Array.isArray(value) ? value.join(', ').length : String(value).length);
    }
    
    if (headersSize > limits.maxHeadersBytes) {
      res.status(431).json({
        error: 'Request Header Fields Too Large',
        message: `Headers exceed ${limits.maxHeadersBytes} bytes`,
        limit: limits.maxHeadersBytes,
        estimated: headersSize,
      });
      return;
    }
    
    next();
  };
}

/**
 * Sanitize request headers for upstream
 * 
 * - Removes hop-by-hop headers
 * - Removes headers in deny list
 * - Keeps only headers in allow list (if specified)
 * - Adds SatGate headers
 * 
 * @param {object} incomingHeaders - Original request headers
 * @param {object} upstreamConfig - Upstream configuration
 * @param {object} satgateContext - Context from SatGate (route, policy, etc.)
 * @returns {object} Sanitized headers for upstream
 */
function sanitizeRequestHeaders(incomingHeaders, upstreamConfig, satgateContext) {
  const sanitized = {};
  
  const allowAll = upstreamConfig.allowRequestHeaders.has('*');
  
  for (const [key, value] of Object.entries(incomingHeaders)) {
    const lowerKey = key.toLowerCase();
    
    // Skip hop-by-hop headers
    if (hopByHopHeaders.has(lowerKey)) continue;
    
    // Skip denied headers
    if (upstreamConfig.denyRequestHeaders.has(lowerKey)) continue;
    
    // Skip SatGate-set headers (we'll add our own)
    if (satgateRequestHeaders.includes(lowerKey)) continue;
    
    // Check allow list (unless allow all)
    if (!allowAll && !upstreamConfig.allowRequestHeaders.has(lowerKey)) continue;
    
    sanitized[lowerKey] = value;
  }
  
  // Add upstream-specific headers
  if (upstreamConfig.addHeaders) {
    for (const [key, value] of Object.entries(upstreamConfig.addHeaders)) {
      sanitized[key.toLowerCase()] = value;
    }
  }
  
  // Add SatGate context headers
  if (satgateContext) {
    if (satgateContext.requestId) {
      sanitized['x-request-id'] = satgateContext.requestId;
    }
    if (satgateContext.clientIp) {
      sanitized['x-forwarded-for'] = satgateContext.clientIp;
    }
    if (satgateContext.proto) {
      sanitized['x-forwarded-proto'] = satgateContext.proto;
    }
    if (satgateContext.host && !upstreamConfig.passHostHeader) {
      sanitized['x-forwarded-host'] = satgateContext.host;
    }
    if (satgateContext.tier) {
      sanitized['x-satgate-tier'] = satgateContext.tier;
    }
    if (satgateContext.scope) {
      sanitized['x-satgate-scope'] = satgateContext.scope;
    }
    if (satgateContext.callsRemaining !== undefined) {
      sanitized['x-calls-remaining'] = String(satgateContext.callsRemaining);
    }
    if (satgateContext.budgetRemaining !== undefined) {
      sanitized['x-budget-remaining'] = String(satgateContext.budgetRemaining);
    }
  }
  
  // Set Host header
  if (upstreamConfig.passHostHeader && satgateContext?.host) {
    sanitized['host'] = satgateContext.host;
  } else {
    sanitized['host'] = upstreamConfig.parsedUrl.host;
  }
  
  return sanitized;
}

/**
 * Sanitize response headers from upstream
 * 
 * - Removes hop-by-hop headers
 * - Keeps only headers in allow list
 * - Preserves SatGate headers
 * 
 * @param {object} upstreamHeaders - Headers from upstream response
 * @param {object} upstreamConfig - Upstream configuration
 * @returns {object} Sanitized headers for client
 */
function sanitizeResponseHeaders(upstreamHeaders, upstreamConfig) {
  const sanitized = {};
  
  const allowAll = upstreamConfig.allowResponseHeaders.includes('*');
  const allowSet = new Set(upstreamConfig.allowResponseHeaders.map(h => h.toLowerCase()));
  
  for (const [key, value] of Object.entries(upstreamHeaders)) {
    const lowerKey = key.toLowerCase();
    
    // Skip hop-by-hop headers
    if (hopByHopHeaders.has(lowerKey)) continue;
    
    // Check allow list (unless allow all)
    if (!allowAll && !allowSet.has(lowerKey)) continue;
    
    sanitized[lowerKey] = value;
  }
  
  return sanitized;
}

/**
 * Generate a unique request ID
 */
function generateRequestId() {
  return `sg-${Date.now().toString(36)}-${Math.random().toString(36).substring(2, 9)}`;
}

/**
 * Request ID middleware
 * 
 * Adds X-Request-Id to request if not present
 */
function requestIdMiddleware(req, res, next) {
  if (!req.headers['x-request-id']) {
    req.headers['x-request-id'] = generateRequestId();
  }
  res.setHeader('X-Request-Id', req.headers['x-request-id']);
  req.requestId = req.headers['x-request-id'];
  next();
}

/**
 * Extract client IP from request
 * 
 * @param {object} req - Express request
 * @param {number|boolean} trustProxy - Trust proxy setting
 */
function extractClientIp(req, trustProxy) {
  if (trustProxy) {
    const xff = req.headers['x-forwarded-for'];
    if (xff) {
      const ips = xff.split(',').map(ip => ip.trim());
      // If trustProxy is a number, trust that many hops
      if (typeof trustProxy === 'number' && trustProxy > 0) {
        return ips[Math.max(0, ips.length - trustProxy)] || req.socket?.remoteAddress;
      }
      return ips[0];
    }
  }
  return req.socket?.remoteAddress || req.ip;
}

module.exports = {
  createLimitsMiddleware,
  sanitizeRequestHeaders,
  sanitizeResponseHeaders,
  requestIdMiddleware,
  generateRequestId,
  extractClientIp,
};


/**
 * SatGate Gateway Router
 * 
 * Fast path matching and route resolution for incoming requests.
 * First match wins (order matters).
 */

const { matchRoute, getConfig } = require('./index');

/**
 * Router middleware for Express
 * 
 * Attaches matched route to req.satgateRoute
 */
function routerMiddleware(req, res, next) {
  const config = getConfig();
  const route = matchRoute(config, req);
  
  if (route) {
    req.satgateRoute = route;
    req.satgateConfig = config;
    
    // Attach upstream config if specified
    if (route.upstream) {
      req.satgateUpstream = config.upstreams[route.upstream];
    }
  }
  
  next();
}

/**
 * Create a standalone matcher function for a route definition
 * 
 * @param {object} routeDef - Route definition from config
 * @returns {function} Matcher function
 */
function createMatcher(routeDef) {
  const match = routeDef.match;
  const methods = new Set((match.methods || ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']).map(m => m.toUpperCase()));
  
  let pathMatcher;
  if (match.exactPath) {
    pathMatcher = (path) => path === match.exactPath;
  } else if (match.pathPrefix) {
    const prefix = match.pathPrefix;
    pathMatcher = (path) => path.startsWith(prefix);
  }
  
  const headerMatchers = [];
  if (match.headers) {
    for (const [header, value] of Object.entries(match.headers)) {
      const lowerHeader = header.toLowerCase();
      headerMatchers.push((headers) => headers[lowerHeader] === value);
    }
  }
  
  return function matcher(method, path, headers = {}) {
    if (!methods.has(method.toUpperCase())) return false;
    if (!pathMatcher(path.split('?')[0])) return false;
    for (const hm of headerMatchers) {
      if (!hm(headers)) return false;
    }
    return true;
  };
}

/**
 * Extract path from URL (handles query strings)
 */
function extractPath(url) {
  const idx = url.indexOf('?');
  return idx === -1 ? url : url.substring(0, idx);
}

/**
 * Build route table summary for logging
 */
function buildRouteSummary(config) {
  const lines = ['Route table:'];
  
  for (const route of config.routes) {
    const match = route.match || {};
    const pathSpec = match.exactPath ? `= ${match.exactPath}` : `^ ${match.pathPrefix}`;
    const methods = match.methods ? match.methods.join(',') : '*';
    const upstream = route.upstream || '(none)';
    const policy = route.policy?.kind || 'unknown';
    
    lines.push(`  ${route.name}: [${methods}] ${pathSpec} -> ${upstream} (${policy})`);
  }
  
  return lines.join('\n');
}

module.exports = {
  routerMiddleware,
  createMatcher,
  extractPath,
  buildRouteSummary,
};


/**
 * SatGate Gateway Configuration Loader
 * 
 * Loads and validates gateway configuration from YAML file.
 * Fail-fast: any validation error stops server startup.
 */

const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');
const Ajv = require('ajv');
const addFormats = require('ajv-formats');

const schema = require('./schema.json');
const { defaults, hopByHopHeaders } = require('./defaults');

/**
 * Deep merge two objects (target wins for conflicts)
 */
function deepMerge(base, override) {
  if (!override) return { ...base };
  if (!base) return { ...override };
  
  const result = { ...base };
  
  for (const key of Object.keys(override)) {
    if (override[key] === undefined) continue;
    
    if (
      typeof override[key] === 'object' &&
      !Array.isArray(override[key]) &&
      override[key] !== null &&
      typeof base[key] === 'object' &&
      !Array.isArray(base[key]) &&
      base[key] !== null
    ) {
      result[key] = deepMerge(base[key], override[key]);
    } else {
      result[key] = override[key];
    }
  }
  
  return result;
}

/**
 * Parse listen address string into host and port
 */
function parseListenAddress(listen) {
  const lastColon = listen.lastIndexOf(':');
  if (lastColon === -1) {
    throw new Error(`Invalid listen address: ${listen} (expected host:port)`);
  }
  
  const host = listen.substring(0, lastColon) || '0.0.0.0';
  const port = parseInt(listen.substring(lastColon + 1), 10);
  
  if (isNaN(port) || port < 1 || port > 65535) {
    throw new Error(`Invalid port in listen address: ${listen}`);
  }
  
  return { host, port };
}

/**
 * Validate upstream URLs are safe (no SSRF via config injection)
 */
function validateUpstreamUrl(name, url) {
  let parsed;
  try {
    parsed = new URL(url);
  } catch (e) {
    throw new Error(`Upstream '${name}' has invalid URL: ${url}`);
  }
  
  // Only allow http/https
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error(`Upstream '${name}' must use http or https (got ${parsed.protocol})`);
  }

  // SECURITY: Do not allow credentials in upstream URLs (can leak via logs/errors).
  if (parsed.username || parsed.password) {
    throw new Error(`Upstream '${name}' URL must not include credentials (username/password)`);
  }
  
  // Disallow file://, data://, javascript://, etc.
  // Disallow localhost/127.0.0.1 in production? (configurable)
  
  return parsed;
}

/**
 * Compile route matchers for fast matching
 */
function compileRoutes(routes, upstreams) {
  return routes.map((route, index) => {
    // Validate upstream reference
    if (route.upstream && !upstreams[route.upstream]) {
      throw new Error(
        `Route '${route.name}' references unknown upstream '${route.upstream}'`
      );
    }
    
    // Compile matcher
    const match = route.match;
    const methods = new Set((match.methods || defaults.routeMatchDefaults.methods).map(m => m.toUpperCase()));
    
    let pathMatcher;
    if (match.exactPath) {
      pathMatcher = (path) => path === match.exactPath;
    } else if (match.pathPrefix) {
      const prefix = match.pathPrefix;
      pathMatcher = (path) => path.startsWith(prefix);
    } else {
      throw new Error(`Route '${route.name}' must have exactPath or pathPrefix`);
    }
    
    // Header matchers (if specified)
    const headerMatchers = [];
    if (match.headers) {
      for (const [header, value] of Object.entries(match.headers)) {
        const lowerHeader = header.toLowerCase();
        headerMatchers.push((req) => {
          const reqValue = req.headers[lowerHeader];
          return reqValue === value;
        });
      }
    }
    
    return {
      name: route.name,
      index,
      upstream: route.upstream,
      policy: route.policy,
      match: (req) => {
        // Method check
        if (!methods.has(req.method)) return false;
        
        // Path check
        if (!pathMatcher(req.url.split('?')[0])) return false;
        
        // Header checks
        for (const hm of headerMatchers) {
          if (!hm(req)) return false;
        }
        
        return true;
      },
    };
  });
}

/**
 * Load and validate gateway configuration
 * 
 * @param {string} configPath - Path to satgate.gateway.yaml
 * @returns {object} Validated and normalized configuration
 * @throws {Error} On validation failure (fail-fast)
 */
function loadConfig(configPath) {
  // Check file exists
  if (!fs.existsSync(configPath)) {
    throw new Error(`Gateway config file not found: ${configPath}`);
  }
  
  // Load YAML
  let rawConfig;
  try {
    const content = fs.readFileSync(configPath, 'utf8');
    rawConfig = yaml.load(content);
  } catch (e) {
    throw new Error(`Failed to parse gateway config: ${e.message}`);
  }
  
  // Validate against JSON Schema
  const ajv = new Ajv({ allErrors: true, strict: false });
  addFormats(ajv);
  const validate = ajv.compile(schema);
  
  if (!validate(rawConfig)) {
    const errors = validate.errors.map(e => `  - ${e.instancePath || '/'}: ${e.message}`).join('\n');
    throw new Error(`Gateway config validation failed:\n${errors}`);
  }
  
  // Merge with defaults
  const config = {
    version: rawConfig.version,
    server: deepMerge(defaults.server, rawConfig.server),
    admin: deepMerge(defaults.admin, rawConfig.admin),
    limits: deepMerge(defaults.limits, rawConfig.limits),
    cors: deepMerge(defaults.cors, rawConfig.cors),
    l402: deepMerge(defaults.l402, rawConfig.l402),
    metering: deepMerge(defaults.metering, rawConfig.metering),
    upstreams: {},
    routes: [],
  };
  
  // Parse listen addresses
  config.server.parsed = parseListenAddress(config.server.listen);
  config.admin.parsed = parseListenAddress(config.admin.listen);
  
  // Process upstreams
  for (const [name, upstream] of Object.entries(rawConfig.upstreams)) {
    const parsedUrl = validateUpstreamUrl(name, upstream.url);
    
    // Normalize header lists to lowercase
    const allowRequestHeaders = new Set(
      (upstream.allowRequestHeaders || defaults.upstreamDefaults.allowRequestHeaders)
        .map(h => h.toLowerCase())
    );
    const denyRequestHeaders = new Set(
      (upstream.denyRequestHeaders || defaults.upstreamDefaults.denyRequestHeaders)
        .map(h => h.toLowerCase())
    );
    
    // Add hop-by-hop headers to deny list
    for (const h of hopByHopHeaders) {
      denyRequestHeaders.add(h);
    }
    
    config.upstreams[name] = {
      name,
      url: upstream.url,
      parsedUrl,
      passHostHeader: upstream.passHostHeader ?? defaults.upstreamDefaults.passHostHeader,
      addHeaders: upstream.addHeaders || {},
      allowRequestHeaders,
      denyRequestHeaders,
      allowResponseHeaders: upstream.allowResponseHeaders || defaults.upstreamDefaults.allowResponseHeaders,
      timeoutMs: upstream.timeoutMs || config.limits.upstreamTimeoutMs,
      connectTimeoutMs: config.limits.upstreamConnectTimeoutMs,
    };
  }
  
  // Compile routes
  config.routes = compileRoutes(rawConfig.routes, config.upstreams);

  // SECURITY: If capability routes are configured, require CAPABILITY_ROOT_KEY in production.
  const hasCapabilityRoutes = Array.isArray(rawConfig.routes) && rawConfig.routes.some(r => r?.policy?.kind === 'capability');
  if (hasCapabilityRoutes) {
    const capabilityKey = process.env.CAPABILITY_ROOT_KEY || '';
    const mode = process.env.MODE || 'prod';
    if (mode === 'prod' && !capabilityKey) {
      throw new Error('Capability routes configured but CAPABILITY_ROOT_KEY is not set (required in production)');
    }
  }
  
  // Resolve environment variable references
  config.l402.rootKey = process.env[config.l402.rootKeyEnv];
  if (config.l402.mode === 'native' && !config.l402.rootKey) {
    const mode = process.env.MODE || 'prod';
    if (mode === 'prod') {
      throw new Error(
        `L402 native mode requires ${config.l402.rootKeyEnv} environment variable in production`
      );
    }
    console.warn(`[Gateway] Warning: ${config.l402.rootKeyEnv} not set (demo mode only)`);
  }
  
  if (config.metering.backend === 'redis') {
    config.metering.redisUrl = process.env[config.metering.redisUrlEnv];
    if (!config.metering.redisUrl) {
      console.warn('[Gateway] Redis backend requested but REDIS_URL not set; falling back to memory');
      config.metering.backend = 'memory';
    }
  }
  
  return config;
}

/**
 * Match a request to a route
 * 
 * @param {object} config - Loaded configuration
 * @param {object} req - HTTP request
 * @returns {object|null} Matched route or null
 */
function matchRoute(config, req) {
  for (const route of config.routes) {
    if (route.match(req)) {
      return route;
    }
  }
  return null;
}

module.exports = {
  loadConfig,
  matchRoute,
  parseListenAddress,
  deepMerge,
};


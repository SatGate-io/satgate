/**
 * SatGate Proxy Configuration Defaults
 * 
 * These defaults are applied when values are not specified in the config file.
 * Security-first: defaults are restrictive, not permissive.
 */

const defaults = {
  version: 1,
  
  server: {
    listen: '0.0.0.0:8080',
    trustProxy: 1,
  },
  
  admin: {
    listen: '127.0.0.1:9090',
    requireAdminToken: true,
  },
  
  limits: {
    maxRequestBodyBytes: 10 * 1024 * 1024,  // 10MB
    maxHeadersBytes: 32 * 1024,              // 32KB
    upstreamTimeoutMs: 30000,                // 30s
    upstreamConnectTimeoutMs: 3000,          // 3s
  },
  
  cors: {
    origins: [],
    allowCredentials: false,
  },
  
  l402: {
    mode: 'native',
    rootKeyEnv: 'L402_ROOT_KEY',
    defaultTTLSeconds: 3600,
    defaultMaxCalls: 100,
    // defaultBudgetSats: undefined (optional)
  },
  
  metering: {
    backend: 'memory',
    redisUrlEnv: 'REDIS_URL',
  },
  
  // Default upstream settings (applied per-upstream if not specified)
  upstreamDefaults: {
    passHostHeader: false,
    addHeaders: {},
    allowRequestHeaders: [
      'content-type',
      'accept',
      'accept-language',
      'accept-encoding',
      'user-agent',
      'authorization',
      'x-request-id',
    ],
    denyRequestHeaders: [
      'x-admin-token',
      'x-satgate-admin-token',
      'x-forwarded-host',  // We set this ourselves
    ],
    allowResponseHeaders: ['*'],
  },
  
  // Default route match settings
  routeMatchDefaults: {
    methods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'],
  },
};

/**
 * Hop-by-hop headers that should never be forwarded
 * per HTTP/1.1 spec (RFC 2616) and HTTP/2 considerations
 */
const hopByHopHeaders = new Set([
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
  // Additional headers to strip for security
  'x-powered-by',
]);

/**
 * Headers that SatGate sets on upstream requests
 * (these override any client-provided values)
 */
const satgateRequestHeaders = [
  'x-forwarded-for',
  'x-forwarded-proto',
  'x-forwarded-host',
  'x-request-id',
  'x-satgate-tier',
  'x-satgate-scope',
  'x-calls-remaining',
  'x-budget-remaining',
];

/**
 * Headers that SatGate sets on responses
 */
const satgateResponseHeaders = [
  'x-satgate-tier',
  'x-satgate-scope',
  'x-satgate-reason',
  'x-calls-remaining',
  'x-budget-remaining',
  'x-l402-price',
  'x-l402-tier',
  'x-l402-ttl',
  'x-l402-max-calls',
  'x-request-id',
];

module.exports = {
  defaults,
  hopByHopHeaders,
  satgateRequestHeaders,
  satgateResponseHeaders,
};


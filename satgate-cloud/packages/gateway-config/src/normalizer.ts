/**
 * Config normalizer - applies defaults and transforms
 */

import { GatewayConfig, Route, Upstream } from './types';

const DEFAULT_SERVER = {
  listen: '0.0.0.0:8080',
  trustProxy: 1,
};

const DEFAULT_ADMIN = {
  listen: '127.0.0.1:9090',
  requireAdminToken: true,
};

const DEFAULT_LIMITS = {
  maxRequestBodyBytes: 10 * 1024 * 1024, // 10MB
  maxHeadersBytes: 32 * 1024, // 32KB
  upstreamTimeoutMs: 30000, // 30s
  upstreamConnectTimeoutMs: 3000, // 3s
};

const DEFAULT_L402 = {
  mode: 'native' as const,
  rootKeyEnv: 'L402_ROOT_KEY',
  defaultTTLSeconds: 3600,
  defaultMaxCalls: 100,
};

const DEFAULT_METERING = {
  backend: 'memory' as const,
  redisUrlEnv: 'REDIS_URL',
};

/**
 * Normalize config by applying defaults
 */
export function normalizeConfig(raw: GatewayConfig): GatewayConfig {
  return {
    version: raw.version || 1,
    server: { ...DEFAULT_SERVER, ...raw.server },
    admin: { ...DEFAULT_ADMIN, ...raw.admin },
    limits: { ...DEFAULT_LIMITS, ...raw.limits },
    cors: raw.cors,
    l402: { ...DEFAULT_L402, ...raw.l402 },
    metering: { ...DEFAULT_METERING, ...raw.metering },
    upstreams: normalizeUpstreams(raw.upstreams || {}),
    routes: normalizeRoutes(raw.routes || []),
  };
}

function normalizeUpstreams(upstreams: Record<string, Upstream>): Record<string, Upstream> {
  const result: Record<string, Upstream> = {};
  
  for (const [name, upstream] of Object.entries(upstreams)) {
    result[name] = {
      url: upstream.url,
      passHostHeader: upstream.passHostHeader ?? false,
      addHeaders: upstream.addHeaders ?? {},
      allowRequestHeaders: upstream.allowRequestHeaders,
      denyRequestHeaders: upstream.denyRequestHeaders ?? [
        'x-admin-token',
        'x-satgate-admin-token',
      ],
      timeoutMs: upstream.timeoutMs,
    };
  }
  
  return result;
}

function normalizeRoutes(routes: Route[]): Route[] {
  return routes.map(route => ({
    name: route.name,
    match: {
      pathPrefix: route.match.pathPrefix,
      exactPath: route.match.exactPath,
      methods: route.match.methods ?? ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'],
      headers: route.match.headers,
    },
    upstream: route.upstream,
    policy: normalizePolicy(route.policy),
  }));
}

function normalizePolicy(policy: any): any {
  if (policy.kind === 'deny') {
    return {
      kind: 'deny',
      status: policy.status ?? 403,
    };
  }
  
  if (policy.kind === 'l402') {
    return {
      kind: 'l402',
      tier: policy.tier,
      priceSats: policy.priceSats,
      scope: policy.scope,
      ttlSeconds: policy.ttlSeconds,
      maxCalls: policy.maxCalls,
      budgetSats: policy.budgetSats,
    };
  }
  
  return policy;
}


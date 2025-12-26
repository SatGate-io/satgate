/**
 * Gateway configuration types
 */

export interface GatewayConfig {
  version: number;
  server?: ServerConfig;
  admin?: AdminConfig;
  limits?: LimitsConfig;
  cors?: CorsConfig;
  l402?: L402Config;
  metering?: MeteringConfig;
  upstreams: Record<string, Upstream>;
  routes: Route[];
}

export interface ServerConfig {
  listen?: string;
  trustProxy?: number;
}

export interface AdminConfig {
  listen?: string;
  requireAdminToken?: boolean;
}

export interface LimitsConfig {
  maxRequestBodyBytes?: number;
  maxHeadersBytes?: number;
  upstreamTimeoutMs?: number;
  upstreamConnectTimeoutMs?: number;
}

export interface CorsConfig {
  origins?: string[];
  allowCredentials?: boolean;
}

export interface L402Config {
  mode?: 'native' | 'aperture';
  rootKeyEnv?: string;
  defaultTTLSeconds?: number;
  defaultMaxCalls?: number;
  defaultBudgetSats?: number;
}

export interface MeteringConfig {
  backend?: 'redis' | 'memory';
  redisUrlEnv?: string;
}

export interface Upstream {
  url: string;
  passHostHeader?: boolean;
  addHeaders?: Record<string, string>;
  allowRequestHeaders?: string[];
  denyRequestHeaders?: string[];
  timeoutMs?: number;
}

export interface Route {
  name: string;
  match: RouteMatch;
  upstream?: string;
  policy: Policy;
}

export interface RouteMatch {
  pathPrefix?: string;
  exactPath?: string;
  methods?: string[];
  headers?: Record<string, string>;
}

export type Policy = PublicPolicy | DenyPolicy | L402Policy | CapabilityPolicy;

export interface PublicPolicy {
  kind: 'public';
}

export interface DenyPolicy {
  kind: 'deny';
  status?: number;
}

export interface L402Policy {
  kind: 'l402';
  tier: string;
  priceSats: number;
  scope: string;
  ttlSeconds?: number;
  maxCalls?: number;
  budgetSats?: number;
}

export interface CapabilityPolicy {
  kind: 'capability';
  scope: string;
  maxCalls?: number;
  budgetSats?: number;
}


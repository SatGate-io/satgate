/**
 * SatGate Gateway Type Definitions
 */

export interface Token {
  /** Base64-encoded token */
  token: string;
  /** Hex-encoded token signature */
  signature: string;
  /** Token scope */
  scope?: string;
  /** Expiration timestamp */
  expiresAt?: string;
}

export interface TokenInfo {
  /** Hex-encoded token signature */
  signature: string;
  /** Token status: active, revoked, expired */
  status: 'active' | 'revoked' | 'expired' | 'banned';
  /** Total number of requests */
  totalRequests: number;
  /** Last used timestamp */
  lastUsed?: string;
  /** Usage by route */
  routes?: Record<string, number>;
  /** When token was banned */
  bannedAt?: string;
  /** Reason for ban */
  banReason?: string;
}

export interface BanRecord {
  /** Hex-encoded token signature */
  signature: string;
  /** Reason for ban */
  reason: string;
  /** When token was banned */
  bannedAt: string;
  /** Who banned the token */
  bannedBy: string;
}

export interface Stats {
  /** Total requests processed */
  totalRequests: number;
  /** Total L402 (paid) requests */
  totalL402: number;
  /** Total capability token requests */
  totalCapability: number;
  /** Total denied requests */
  totalDenied: number;
  /** Total error responses */
  totalErrors: number;
  /** Governance statistics */
  governance: {
    active: number;
    banned: number;
    totalRequests: number;
  };
}

export interface MintRequest {
  /** Token scope (e.g., "api:read", "api:*") */
  scope?: string;
  /** Token lifetime in seconds */
  expiresIn?: number;
}

export interface DelegateRequest {
  /** Base64-encoded parent token */
  parentToken: string;
  /** Additional caveats to add */
  caveats?: string[];
}

export interface BanRequest {
  /** Hex-encoded token signature */
  tokenSignature: string;
  /** Reason for ban */
  reason?: string;
}

export interface GatewayConfig {
  version: number;
  server?: object;
  admin?: object;
  lightning?: object;
  upstreams?: Record<string, object>;
  routes?: RouteConfig[];
}

/**
 * Economic Policy types
 * 
 * SatGate uses a Layer 0/Layer 1 model:
 * - Layer 0 (Default Protection): Cryptographic verification on all non-PUBLIC routes
 * - Layer 1 (Economic Policy): Observe, Control, or Charge
 */
export type PolicyKind = 
  | 'public'      // Exception: No protection
  | 'deny'        // Exception: Block all
  | 'protected'   // Layer 0: Verify only (alias: capability, protect)
  | 'observe'     // Layer 1: Verify → meter/log (alias: chargeback, audit)
  | 'control'     // Layer 1: Verify → budget enforcement (alias: fiat402, budget)
  | 'charge';     // Layer 1: Verify → payment required (alias: l402, monetize, pay)

export interface RouteConfig {
  /** Route name */
  name: string;
  /** Path prefix or exact match */
  match: {
    pathPrefix?: string;
    pathExact?: string;
    methods?: string[];
  };
  /** Economic policy */
  policy: {
    kind: PolicyKind;
    /** L402 config for Charge policy */
    l402?: {
      priceSats: number;
      unit?: 'per_request' | 'per_kb';
    };
    /** Budget config for Control policy */
    budget?: {
      amountCents: number;
      period: 'hourly' | 'daily' | 'monthly';
    };
  };
  /** Upstream to proxy to */
  upstream: string;
  /** Rate limiting */
  rateLimit?: {
    requestsPerSecond: number;
    burstSize?: number;
  };
}

/**
 * Usage summary with separate observe/billable counts
 */
export interface UsageSummary {
  tenantId: string;
  periodStart: string;
  periodEnd: string;
  /** Total requests (all policies) */
  totalRequests: number;
  /** Billable requests (Control/Charge only) */
  billableRequests: number;
  /** Free observe requests (unlimited) */
  observeRequests: number;
  /** Total bytes transferred */
  totalBytes: number;
  /** L402 revenue in satoshis */
  l402RevenueSats: number;
  /** Platform fee (2%) in satoshis */
  platformFeeSats: number;
  /** Net revenue after fee */
  netRevenueSats: number;
}




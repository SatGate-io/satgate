/**
 * SatGate Gateway Type Definitions (OSS)
 */

export interface Token {
  /** Base64-encoded token */
  token: string;
  /** Hex-encoded token signature */
  signature: string;
  /** Token scope */
  scope?: string;
  /** Expiration timestamp (ISO 8601) */
  expiresAt?: string;
  /** Token caveats */
  caveats?: string[];
}

export interface TokenInfo {
  /** Hex-encoded token signature (or node id in graph) */
  signature: string;
  /** Token status: active, banned */
  status: 'active' | 'banned';
  /** Token scope */
  scope?: string;
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
  /** Parent token signature (for delegation tree) */
  parentSignature?: string;
  /** Label (e.g., "Agent Token") */
  label?: string;
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

export interface GraphStats {
  /** Active (non-banned) tokens */
  active: number;
  /** Blocked tokens */
  blocked: number;
  /** Banned tokens */
  banned: number;
  /** Number of requests hitting banned tokens */
  bannedHits: number;
}

export interface GraphData {
  /** Token nodes */
  nodes: TokenInfo[];
  /** Delegation edges */
  edges: { source: string; target: string }[];
  /** Summary stats */
  stats: GraphStats;
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

/**
 * Economic Policy types
 * 
 * SatGate uses a Layer 0/Layer 1 model:
 * - Layer 0 (Default Protection): Cryptographic verification on all non-PUBLIC routes
 * - Layer 1 (Economic Policy): Observe, Control, or Charge
 */
export type PolicyKind = 
  | 'public'      // Exception: No protection
  | 'capability'  // Layer 0: Verify only (macaroon-based)
  | 'l402';       // Layer 1: Verify → payment required (Lightning)

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
    /** Required scope for capability policy */
    scope?: string;
    /** Price in sats for L402 policy */
    priceSats?: number;
  };
  /** Upstream to proxy to */
  upstream: string;
}

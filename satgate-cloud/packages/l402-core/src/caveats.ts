/**
 * Caveat parsing and verification
 */

export interface Caveats {
  paymentHash?: string;      // ph=<hash>
  expiresAt?: number;        // exp=<timestamp>
  scope?: string;            // scope=<scope>
  tier?: string;             // tier=<tier>
  maxCalls?: number;         // mc=<count>
  budgetSats?: number;       // bs=<sats>
  audience?: string;         // aud=<audience>
}

/**
 * Parse caveats from string array
 */
export function parseCaveats(caveatStrings: string[]): Caveats {
  const caveats: Caveats = {};
  
  for (const caveat of caveatStrings) {
    const [key, ...valueParts] = caveat.split('=');
    const value = valueParts.join('='); // Handle values with '='
    
    switch (key) {
      case 'ph':
        caveats.paymentHash = value;
        break;
      case 'exp':
        caveats.expiresAt = parseInt(value, 10);
        break;
      case 'scope':
        caveats.scope = value;
        break;
      case 'tier':
        caveats.tier = value;
        break;
      case 'mc':
        caveats.maxCalls = parseInt(value, 10);
        break;
      case 'bs':
        caveats.budgetSats = parseInt(value, 10);
        break;
      case 'aud':
        caveats.audience = value;
        break;
    }
  }
  
  return caveats;
}

export interface VerifyCaveatsOptions {
  /** Required audience (for tenant binding) */
  requiredAudience?: string;
  /** Required scope prefix */
  requiredScopePrefix?: string;
  /** Current time (for testing) */
  now?: number;
}

export interface VerifyCaveatsResult {
  valid: boolean;
  reason?: string;
  caveats: Caveats;
}

/**
 * Verify caveats against requirements
 */
export function verifyCaveats(
  caveatStrings: string[],
  options: VerifyCaveatsOptions = {}
): VerifyCaveatsResult {
  const caveats = parseCaveats(caveatStrings);
  const now = options.now || Date.now();
  
  // Check expiration
  if (caveats.expiresAt && caveats.expiresAt < now) {
    return {
      valid: false,
      reason: 'Token expired',
      caveats,
    };
  }
  
  // Check audience (tenant binding)
  if (options.requiredAudience && caveats.audience !== options.requiredAudience) {
    return {
      valid: false,
      reason: `Audience mismatch: expected ${options.requiredAudience}, got ${caveats.audience}`,
      caveats,
    };
  }
  
  // Check scope prefix
  if (options.requiredScopePrefix && caveats.scope) {
    if (!caveats.scope.startsWith(options.requiredScopePrefix)) {
      return {
        valid: false,
        reason: `Scope mismatch: ${caveats.scope} does not match ${options.requiredScopePrefix}`,
        caveats,
      };
    }
  }
  
  return {
    valid: true,
    caveats,
  };
}


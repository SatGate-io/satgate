/**
 * LSAT validation
 */

import * as crypto from 'crypto';
import { SimpleMacaroon } from './macaroon';
import { parseCaveats, verifyCaveats, Caveats } from './caveats';

export interface ValidationOptions {
  /** Root key for macaroon verification */
  rootKey: string;
  /** Required audience (tenant binding) */
  requiredAudience?: string;
  /** Current time (for testing) */
  now?: number;
}

export interface ValidationResult {
  valid: boolean;
  reason?: string;
  caveats?: Caveats;
  tokenSignature?: string;
}

/**
 * Parse Authorization header
 * Format: "LSAT <macaroon>:<preimage>" or "L402 <macaroon>:<preimage>"
 */
export function parseAuthHeader(authHeader: string): { macaroon: string; preimage: string } | null {
  const match = authHeader.match(/^(?:LSAT|L402)\s+([^:]+):([a-fA-F0-9]+)$/i);
  if (!match) return null;
  
  return {
    macaroon: match[1],
    preimage: match[2],
  };
}

/**
 * Validate an LSAT token
 */
export function validateLSAT(
  authHeader: string,
  options: ValidationOptions
): ValidationResult {
  // Parse header
  const parsed = parseAuthHeader(authHeader);
  if (!parsed) {
    return { valid: false, reason: 'Invalid Authorization header format' };
  }
  
  const { macaroon: macaroonBase64, preimage } = parsed;
  
  // Validate preimage format (64 hex chars)
  if (!/^[a-fA-F0-9]{64}$/.test(preimage)) {
    return { valid: false, reason: 'Invalid preimage format' };
  }
  
  // Import macaroon
  let mac: SimpleMacaroon;
  try {
    mac = SimpleMacaroon.import(macaroonBase64);
  } catch {
    return { valid: false, reason: 'Invalid macaroon format' };
  }
  
  // Verify macaroon signature
  if (!mac.verify(options.rootKey)) {
    return { valid: false, reason: 'Invalid macaroon signature' };
  }
  
  // Parse and verify caveats
  const caveatStrings = mac.getCaveats();
  const caveatResult = verifyCaveats(caveatStrings, {
    requiredAudience: options.requiredAudience,
    now: options.now,
  });
  
  if (!caveatResult.valid) {
    return {
      valid: false,
      reason: caveatResult.reason,
      caveats: caveatResult.caveats,
    };
  }
  
  // Verify preimage matches payment hash
  const paymentHash = caveatResult.caveats.paymentHash;
  if (!paymentHash) {
    return { valid: false, reason: 'Missing payment hash caveat' };
  }
  
  // SHA256(preimage) should equal paymentHash
  const computedHash = crypto.createHash('sha256')
    .update(Buffer.from(preimage, 'hex'))
    .digest('hex');
  
  if (computedHash !== paymentHash) {
    return { valid: false, reason: 'Preimage does not match payment hash' };
  }
  
  return {
    valid: true,
    caveats: caveatResult.caveats,
    tokenSignature: mac.getSignature(),
  };
}


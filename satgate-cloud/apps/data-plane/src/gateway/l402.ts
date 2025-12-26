/**
 * L402 enforcement for Cloud data plane
 */

import * as crypto from 'crypto';
import { Request, Response } from 'express';
import { L402Policy } from '@satgate/gateway-config';
import { SimpleMacaroon, parseCaveats, verifyCaveats, Caveats } from '@satgate/l402-core';
import { logger } from '@satgate/common';

// L402 root key (per-tenant in future, single key for v1)
const L402_ROOT_KEY = process.env.L402_ROOT_KEY || '';

// Lightning service (placeholder - will integrate with managed Lightning)
const LIGHTNING_ENABLED = process.env.LIGHTNING_ENABLED === 'true';

interface L402Result {
  allowed: boolean;
  statusCode?: number;
  headers?: Record<string, string>;
  body?: any;
  tokenId?: string;
  caveats?: Caveats;
}

/**
 * Enforce L402 policy
 */
export async function enforceL402(
  req: Request,
  res: Response,
  policy: L402Policy,
  tenantSlug: string
): Promise<L402Result> {
  // Check for Authorization header
  const authHeader = req.headers.authorization;
  
  if (!authHeader) {
    return createChallenge(policy, tenantSlug);
  }
  
  // Parse L402/LSAT token
  const parsed = parseL402Header(authHeader);
  
  if (!parsed) {
    return createChallenge(policy, tenantSlug);
  }
  
  // Validate macaroon
  const validation = await validateToken(parsed.macaroon, parsed.preimage, policy, tenantSlug);
  
  if (!validation.valid) {
    logger.debug('L402 validation failed', { reason: validation.reason, tenant: tenantSlug });
    return createChallenge(policy, tenantSlug);
  }
  
  // Token is valid - allow request
  return {
    allowed: true,
    tokenId: validation.tokenId,
    caveats: validation.caveats,
  };
}

/**
 * Parse L402/LSAT Authorization header
 */
function parseL402Header(header: string): { macaroon: string; preimage: string } | null {
  // Format: L402 <macaroon>:<preimage> or LSAT <macaroon>:<preimage>
  const match = header.match(/^(L402|LSAT)\s+([^:]+):([a-fA-F0-9]+)$/i);
  
  if (!match) {
    return null;
  }
  
  return {
    macaroon: match[2],
    preimage: match[3],
  };
}

/**
 * Validate L402 token
 */
async function validateToken(
  macaroonStr: string,
  preimage: string,
  policy: L402Policy,
  tenantSlug: string
): Promise<{ valid: boolean; reason?: string; tokenId?: string; caveats?: Caveats }> {
  if (!L402_ROOT_KEY) {
    logger.error('L402_ROOT_KEY not set');
    return { valid: false, reason: 'L402 not configured' };
  }
  
  try {
    // Decode macaroon
    const macaroon = SimpleMacaroon.import(macaroonStr);
    
    // Verify signature
    if (!macaroon.verify(L402_ROOT_KEY)) {
      return { valid: false, reason: 'Invalid signature' };
    }
    
    // Parse and validate caveats
    const verification = verifyCaveats(macaroon.getCaveats(), {
      requiredAudience: `${tenantSlug}.satgate.cloud`,
      requiredScopePrefix: policy.scope,
    });
    
    if (!verification.valid) {
      return { valid: false, reason: verification.reason };
    }
    
    // Validate preimage against payment hash
    const caveats = verification.caveats;
    if (caveats.paymentHash) {
      const preimageHash = crypto.createHash('sha256')
        .update(Buffer.from(preimage, 'hex'))
        .digest('hex');
      
      if (preimageHash !== caveats.paymentHash) {
        return { valid: false, reason: 'Invalid preimage' };
      }
    }
    
    return {
      valid: true,
      tokenId: macaroon.getIdentifier(),
      caveats: caveats,
    };
  } catch (err) {
    logger.error('Token validation error', { error: (err as Error).message });
    return { valid: false, reason: 'Invalid token format' };
  }
}

/**
 * Create L402 challenge response
 */
async function createChallenge(
  policy: L402Policy,
  tenantSlug: string
): Promise<L402Result> {
  if (!L402_ROOT_KEY) {
    return {
      allowed: false,
      statusCode: 500,
      body: { error: 'L402 not configured' },
    };
  }
  
  if (!LIGHTNING_ENABLED) {
    // For v1 without Lightning, return a placeholder challenge
    return {
      allowed: false,
      statusCode: 402,
      headers: {
        'WWW-Authenticate': `L402 macaroon="placeholder", invoice="lnbc1..."`,
        'X-L402-Price': String(policy.priceSats),
        'X-L402-Tier': policy.tier,
      },
      body: {
        error: 'Payment Required',
        price: policy.priceSats,
        tier: policy.tier,
        message: 'Lightning payments coming soon',
      },
    };
  }
  
  try {
    // Generate payment hash
    const paymentPreimage = crypto.randomBytes(32);
    const paymentHash = crypto.createHash('sha256').update(paymentPreimage).digest('hex');
    
    // Create macaroon with caveats
    const tokenId = generateTokenId();
    const expiresAt = Math.floor(Date.now() / 1000) + (policy.ttlSeconds || 3600);
    
    const macaroon = new SimpleMacaroon(
      `${tenantSlug}.satgate.cloud`,
      tokenId,
      L402_ROOT_KEY
    );
    macaroon.addFirstPartyCaveat(`scope=${policy.scope}`);
    macaroon.addFirstPartyCaveat(`aud=${tenantSlug}.satgate.cloud`);
    macaroon.addFirstPartyCaveat(`exp=${expiresAt}`);
    macaroon.addFirstPartyCaveat(`ph=${paymentHash}`);
    
    if (policy.maxCalls) {
      macaroon.addFirstPartyCaveat(`mc=${policy.maxCalls}`);
    }
    if (policy.budgetSats) {
      macaroon.addFirstPartyCaveat(`bs=${policy.budgetSats}`);
    }
    
    // Create invoice (placeholder - will integrate with managed Lightning)
    const invoice = await createInvoice(policy.priceSats, `${tenantSlug}:${policy.tier}`, paymentHash);
    
    return {
      allowed: false,
      statusCode: 402,
      headers: {
        'WWW-Authenticate': `L402 macaroon="${macaroon.serialize()}", invoice="${invoice}"`,
        'X-L402-Price': String(policy.priceSats),
        'X-L402-Tier': policy.tier,
        'Cache-Control': 'no-store',
      },
      body: {
        error: 'Payment Required',
        price: policy.priceSats,
        tier: policy.tier,
        invoice,
      },
    };
  } catch (err) {
    logger.error('Challenge creation error', { error: (err as Error).message });
    return {
      allowed: false,
      statusCode: 500,
      body: { error: 'Failed to create payment challenge' },
    };
  }
}

/**
 * Generate a unique token ID
 */
function generateTokenId(): string {
  return crypto.randomBytes(16).toString('hex');
}

/**
 * Create Lightning invoice (placeholder)
 */
async function createInvoice(amountSats: number, memo: string, paymentHash: string): Promise<string> {
  // TODO: Integrate with managed Lightning (phoenixd)
  return `lnbc${amountSats}n1placeholder`;
}

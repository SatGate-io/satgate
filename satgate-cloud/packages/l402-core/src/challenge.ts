/**
 * L402 Challenge creation
 */

import { SimpleMacaroon } from './macaroon';

export interface ChallengeOptions {
  tier: string;
  priceSats: number;
  scope: string;
  ttlSeconds?: number;
  maxCalls?: number;
  budgetSats?: number;
  /** Audience for tenant binding (e.g., "acme.satgate.cloud") */
  audience?: string;
}

export interface ChallengeResult {
  statusCode: 402;
  headers: {
    'WWW-Authenticate': string;
    'X-L402-Price': string;
    'X-L402-Tier': string;
    'X-L402-TTL': string;
  };
  body: {
    error: 'Payment Required';
    code: 'PAYMENT_REQUIRED';
    message: string;
    price: number;
    tier: string;
    invoice: string;
    paymentHash: string;
    macaroon: string;
    expiresAt: number;
  };
}

export interface LightningBackend {
  createInvoice(amountSats: number, memo: string, expirySecs: number): Promise<{
    paymentRequest: string;
    paymentHash: string;
    expiresAt: number;
  }>;
}

const MACAROON_LOCATION = 'https://satgate.cloud';

/**
 * Create an L402 challenge (402 response with invoice + macaroon)
 */
export async function createChallenge(
  lightning: LightningBackend,
  rootKey: string,
  options: ChallengeOptions
): Promise<ChallengeResult> {
  const ttl = options.ttlSeconds || 3600;
  const expiresAt = Date.now() + (ttl * 1000);
  
  // Create invoice
  const memo = `SatGate ${options.tier} access - ${options.maxCalls || 100} calls for ${ttl}s`;
  const invoice = await lightning.createInvoice(options.priceSats, memo, Math.min(ttl, 600));
  
  // Create macaroon
  const timestamp = Date.now().toString(36);
  const hashPrefix = invoice.paymentHash.substring(0, 16);
  const identifier = `sg:${hashPrefix}:${timestamp}`;
  
  const mac = new SimpleMacaroon(MACAROON_LOCATION, identifier, rootKey);
  
  // Add caveats
  mac.addFirstPartyCaveat(`ph=${invoice.paymentHash}`);
  mac.addFirstPartyCaveat(`exp=${expiresAt}`);
  mac.addFirstPartyCaveat(`scope=${options.scope}`);
  mac.addFirstPartyCaveat(`tier=${options.tier}`);
  
  if (options.maxCalls) {
    mac.addFirstPartyCaveat(`mc=${options.maxCalls}`);
  }
  if (options.budgetSats) {
    mac.addFirstPartyCaveat(`bs=${options.budgetSats}`);
  }
  if (options.audience) {
    mac.addFirstPartyCaveat(`aud=${options.audience}`);
  }
  
  const macaroonBase64 = mac.serialize();
  const wwwAuth = `L402 macaroon="${macaroonBase64}", invoice="${invoice.paymentRequest}"`;
  
  return {
    statusCode: 402,
    headers: {
      'WWW-Authenticate': wwwAuth,
      'X-L402-Price': String(options.priceSats),
      'X-L402-Tier': options.tier,
      'X-L402-TTL': String(ttl),
    },
    body: {
      error: 'Payment Required',
      code: 'PAYMENT_REQUIRED',
      message: `This endpoint requires ${options.priceSats} sats. Pay the invoice and retry with the LSAT token.`,
      price: options.priceSats,
      tier: options.tier,
      invoice: invoice.paymentRequest,
      paymentHash: invoice.paymentHash,
      macaroon: macaroonBase64,
      expiresAt: invoice.expiresAt,
    },
  };
}


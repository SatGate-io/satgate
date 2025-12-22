/**
 * SatGate JavaScript/TypeScript SDK
 * Automatic L402 payment handling for AI agents
 */

// =============================================================================
// WALLET INTERFACES
// =============================================================================

export abstract class LightningWallet {
  abstract payInvoice(invoice: string): Promise<string>;
}

export class WebLNWallet extends LightningWallet {
  payInvoice(invoice: string): Promise<string>;
}

export interface LNBitsWalletConfig {
  url: string;
  adminKey: string;
}

export class LNBitsWallet extends LightningWallet {
  constructor(config: LNBitsWalletConfig);
  payInvoice(invoice: string): Promise<string>;
}

export interface AlbyWalletConfig {
  accessToken: string;
}

export class AlbyWallet extends LightningWallet {
  constructor(config: AlbyWalletConfig);
  payInvoice(invoice: string): Promise<string>;
}

// =============================================================================
// PAYMENT INFO
// =============================================================================

export interface PaymentInfoData {
  invoice: string;
  preimage: string;
  macaroon: string;
  amountSats: number | null;
  endpoint: string;
}

export class PaymentInfo implements PaymentInfoData {
  invoice: string;
  preimage: string;
  macaroon: string;
  amountSats: number | null;
  endpoint: string;
  timestamp: number;
  constructor(data: PaymentInfoData);
}

// =============================================================================
// CLIENT CONFIG
// =============================================================================

export interface L402Challenge {
  invoice: string;
  macaroon: string;
}

export interface SatGateClientConfig {
  /** Lightning wallet instance or 'webln' for browser */
  wallet?: LightningWallet | 'webln' | 'alby';
  /** Base URL for all requests */
  baseUrl?: string;
  /** Cache L402 tokens to avoid re-paying (default: true) */
  cacheTokens?: boolean;
  /** Token cache TTL in seconds (default: 3600) */
  cacheTtl?: number;
  /** Print progress to console (default: true) */
  verbose?: boolean;
  /** Called when 402 challenge received */
  onChallenge?: (challenge: L402Challenge) => void;
  /** Called when payment is about to start */
  onPaymentStart?: (data: { invoice: string }) => void;
  /** Called when payment completes */
  onPayment?: (info: PaymentInfo) => void;
}

// =============================================================================
// MAIN CLIENT
// =============================================================================

export class SatGateClient {
  constructor(config?: SatGateClientConfig);
  
  /**
   * Make a fetch request with automatic L402 handling
   */
  fetch(input: string, init?: RequestInit): Promise<Response>;
  
  /**
   * Make a GET request with automatic L402 handling
   */
  get(url: string, init?: RequestInit): Promise<Response>;
  
  /**
   * Make a POST request with automatic L402 handling
   */
  post(url: string, body: any, init?: RequestInit): Promise<Response>;
  
  /**
   * Get total satoshis paid in this session
   */
  getTotalPaidSats(): number;
  
  /**
   * Clear the token cache
   */
  clearCache(): void;
}

export default SatGateClient;

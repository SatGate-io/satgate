/**
 * SatGate Agent Client - Economic Middleware for AI Agents (OSS)
 * 
 * Automatically handles:
 * - Token minting via POST /api/capability/mint (X-Admin-Token)
 * - Token delegation via POST /api/capability/delegate
 * - L402 Payment Required challenges
 * - Token caching and refresh
 * - Budget tracking and alerts
 * 
 * @example
 * ```typescript
 * import { SatGateAgentClient } from '@satgate/sdk';
 * 
 * // With admin token (auto-mints capability tokens)
 * const client = new SatGateAgentClient({
 *   gatewayUrl: 'http://localhost:8080',
 *   adminToken: 'your-admin-token'
 * });
 * 
 * // Make requests - token management is automatic
 * const response = await client.get('/api/data');
 * console.log(response.data);
 * ```
 */

import * as fs from 'fs';
import {
  SatGateError,
  AuthenticationError,
  PaymentRequiredError,
  PaymentFailedError,
  BudgetExceededError,
} from './errors';

export interface AgentResponse<T = unknown> {
  statusCode: number;
  headers: Record<string, string>;
  data: T;
  cost: number;
  budgetRemaining?: number;
}

export interface CachedToken {
  token: string;
  signature: string;
  expiresAt: number;
  scope: string;
  budgetLimit?: number;
  budgetUsed: number;
  caveats: string[];
}

// Lightning Wallets

export interface LightningWallet {
  payInvoice(invoice: string): Promise<string>;
  getBalance(): Promise<number>;
}

export interface LNDWalletOptions {
  host?: string;
  macaroonPath?: string;
  macaroonHex?: string;
  certPath?: string;
}

export class LNDWallet implements LightningWallet {
  private host: string;
  private macaroon: string;
  private certPath?: string;
  
  constructor(options: LNDWalletOptions) {
    this.host = options.host || 'localhost:8080';
    this.certPath = options.certPath;
    
    if (options.macaroonHex) {
      this.macaroon = options.macaroonHex;
    } else if (options.macaroonPath) {
      const data = fs.readFileSync(options.macaroonPath);
      this.macaroon = data.toString('hex');
    } else {
      throw new SatGateError('LND macaroon required');
    }
  }
  
  async payInvoice(invoice: string): Promise<string> {
    const url = `https://${this.host}/v1/channels/transactions`;
    
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Grpc-Metadata-macaroon': this.macaroon,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ payment_request: invoice }),
    });
    
    if (!response.ok) {
      throw new PaymentFailedError(`LND payment failed: ${await response.text()}`);
    }
    
    const result = await response.json() as { payment_preimage?: string; payment_error?: string };
    
    if (result.payment_error) {
      throw new PaymentFailedError(`Payment error: ${result.payment_error}`);
    }
    
    return result.payment_preimage || '';
  }
  
  async getBalance(): Promise<number> {
    const url = `https://${this.host}/v1/balance/channels`;
    
    const response = await fetch(url, {
      headers: { 'Grpc-Metadata-macaroon': this.macaroon },
    });
    
    if (!response.ok) {
      throw new SatGateError(`Failed to get balance: ${await response.text()}`);
    }
    
    const result = await response.json() as { balance?: string };
    return parseInt(result.balance || '0', 10);
  }
}

export interface AlbyWalletOptions {
  accessToken?: string;
  nwcUrl?: string;
}

export class AlbyWallet implements LightningWallet {
  private accessToken?: string;
  private nwcUrl?: string;
  
  constructor(options: AlbyWalletOptions = {}) {
    this.accessToken = options.accessToken || process.env.ALBY_ACCESS_TOKEN;
    this.nwcUrl = options.nwcUrl || process.env.ALBY_NWC_URL;
    
    if (!this.accessToken && !this.nwcUrl) {
      throw new SatGateError('Alby access token or NWC URL required');
    }
  }
  
  async payInvoice(invoice: string): Promise<string> {
    if (!this.accessToken) {
      throw new SatGateError('Alby access token required for API payments');
    }
    
    const response = await fetch('https://api.getalby.com/payments/bolt11', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.accessToken}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ invoice }),
    });
    
    if (!response.ok) {
      throw new PaymentFailedError(`Alby payment failed: ${await response.text()}`);
    }
    
    const result = await response.json() as { payment_preimage?: string };
    return result.payment_preimage || '';
  }
  
  async getBalance(): Promise<number> {
    if (!this.accessToken) {
      throw new SatGateError('Alby access token required');
    }
    
    const response = await fetch('https://api.getalby.com/balance', {
      headers: { 'Authorization': `Bearer ${this.accessToken}` },
    });
    
    if (!response.ok) {
      throw new SatGateError(`Failed to get balance: ${await response.text()}`);
    }
    
    const result = await response.json() as { balance?: number };
    return result.balance || 0;
  }
}

// Agent Client

export interface SatGateAgentClientOptions {
  /** Gateway URL */
  gatewayUrl: string;
  /** 
   * Admin token for minting capability tokens (X-Admin-Token header).
   * Can also be set via SATGATE_ADMIN_TOKEN environment variable.
   */
  adminToken?: string;
  /**
   * Pre-existing capability token (alternative to adminToken).
   * Can also be set via SATGATE_TOKEN environment variable.
   */
  token?: string;
  /** Lightning wallet for L402 payments */
  wallet?: LightningWallet;
  /** Maximum budget to spend */
  budgetLimit?: number;
  /** Default scope for minted tokens (default: "api:*") */
  scope?: string;
  /** Default duration for minted tokens (default: "1h") */
  duration?: string;
  /** Request timeout in milliseconds */
  timeout?: number;
  /** Maximum retry attempts for 402 challenges */
  maxRetries?: number;
  /** Callback when budget is running low */
  onBudgetAlert?: (remaining: number, limit: number) => void;
}

export class SatGateAgentClient {
  private gatewayUrl: string;
  private adminToken?: string;
  private wallet?: LightningWallet;
  private budgetLimit?: number;
  private defaultScope: string;
  private defaultDuration: string;
  private timeout: number;
  private maxRetries: number;
  private onBudgetAlert?: (remaining: number, limit: number) => void;
  
  private tokenCache?: CachedToken;
  private totalCost: number = 0;
  
  constructor(options: SatGateAgentClientOptions) {
    this.gatewayUrl = options.gatewayUrl.replace(/\/$/, '');
    this.adminToken = options.adminToken || process.env.SATGATE_ADMIN_TOKEN;
    this.wallet = options.wallet;
    this.budgetLimit = options.budgetLimit;
    this.defaultScope = options.scope || 'api:*';
    this.defaultDuration = options.duration || '1h';
    this.timeout = options.timeout || 30000;
    this.maxRetries = options.maxRetries || 3;
    this.onBudgetAlert = options.onBudgetAlert;
    
    // If a pre-existing token was provided, cache it
    const preToken = options.token || process.env.SATGATE_TOKEN;
    if (preToken) {
      this.tokenCache = {
        token: preToken,
        signature: 'pre-provided',
        expiresAt: Date.now() + 3600000,
        scope: this.defaultScope,
        budgetUsed: 0,
        caveats: [],
      };
    }
  }
  
  private async getToken(): Promise<string | undefined> {
    // Check cache
    if (this.tokenCache && Date.now() < this.tokenCache.expiresAt) {
      return this.tokenCache.token;
    }
    
    // No admin token - can't mint
    if (!this.adminToken) {
      return undefined;
    }
    
    // Mint new token via OSS endpoint
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), this.timeout);
      
      const response = await fetch(`${this.gatewayUrl}/api/capability/mint`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Admin-Token': this.adminToken,
        },
        body: JSON.stringify({
          scope: this.defaultScope,
          duration: this.defaultDuration,
        }),
        signal: controller.signal,
      });
      
      clearTimeout(timeoutId);
      
      if (!response.ok) {
        throw new AuthenticationError(`Mint failed (${response.status}): ${await response.text()}`);
      }
      
      const result = await response.json() as {
        token: string;
        signature: string;
        scope?: string;
        expiresAt?: string;
      };
      
      // Parse expiration
      let expiresAt = Date.now() + 3600000; // Default 1 hour
      if (result.expiresAt) {
        const parsed = new Date(result.expiresAt).getTime();
        if (!isNaN(parsed)) {
          expiresAt = parsed;
        }
      }
      
      this.tokenCache = {
        token: result.token,
        signature: result.signature || '',
        expiresAt,
        scope: result.scope || this.defaultScope,
        budgetUsed: 0,
        caveats: [],
      };
      
      return this.tokenCache.token;
    } catch (e) {
      if (e instanceof SatGateError) throw e;
      throw new AuthenticationError(`Failed to mint token: ${e}`);
    }
  }
  
  private async handleL402(
    response: Response,
    method: string,
    url: string,
    options: RequestInit
  ): Promise<AgentResponse> {
    const wwwAuth = response.headers.get('WWW-Authenticate') || '';
    
    let challenge: Record<string, unknown> = {};
    const contentType = response.headers.get('Content-Type') || '';
    if (contentType.includes('application/json')) {
      try {
        challenge = await response.json() as Record<string, unknown>;
      } catch {}
    }
    
    if (!this.wallet) {
      throw new PaymentRequiredError('L402 payment required but no wallet configured', challenge);
    }
    
    // Parse L402 header: L402 macaroon="...", invoice="..."
    let macaroon = '';
    let invoice = '';
    
    for (const part of wwwAuth.split(',')) {
      const trimmed = part.trim();
      if (trimmed.includes('macaroon=')) {
        macaroon = trimmed.split('=')[1]?.replace(/['"]/g, '') || '';
      } else if (trimmed.includes('invoice=')) {
        invoice = trimmed.split('=')[1]?.replace(/['"]/g, '') || '';
      }
    }
    
    // Also check JSON challenge
    if (!invoice) invoice = challenge.invoice as string || '';
    if (!macaroon) macaroon = challenge.macaroon as string || '';
    
    if (!invoice) {
      throw new PaymentRequiredError('No invoice in L402 challenge', challenge);
    }
    
    // Pay the invoice
    let preimage: string;
    try {
      preimage = await this.wallet.payInvoice(invoice);
    } catch (e) {
      throw new PaymentFailedError(`Lightning payment failed: ${e}`);
    }
    
    // Build L402 token and retry
    const l402Token = `L402 ${macaroon}:${preimage}`;
    const headers = new Headers(options.headers);
    headers.set('Authorization', l402Token);
    
    const retryResponse = await fetch(url, { ...options, headers });
    const costSats = (challenge.amount_sats as number) || (challenge.amount as number) || 0;
    
    return {
      statusCode: retryResponse.status,
      headers: Object.fromEntries(retryResponse.headers.entries()),
      data: await retryResponse.json(),
      cost: costSats,
    };
  }
  
  async request<T = unknown>(
    method: string,
    path: string,
    options: RequestInit = {}
  ): Promise<AgentResponse<T>> {
    const url = new URL(path, this.gatewayUrl).toString();
    
    // Add capability token if available
    const token = await this.getToken();
    const headers = new Headers(options.headers);
    if (token && !headers.has('Authorization')) {
      headers.set('Authorization', `Bearer ${token}`);
    }
    
    const requestOptions: RequestInit = {
      ...options,
      method,
      headers,
    };
    
    // Make request with retries
    for (let attempt = 0; attempt < this.maxRetries; attempt++) {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), this.timeout);
      
      try {
        const response = await fetch(url, {
          ...requestOptions,
          signal: controller.signal,
        });
        
        clearTimeout(timeoutId);
        
        // Success
        if (response.ok) {
          return {
            statusCode: response.status,
            headers: Object.fromEntries(response.headers.entries()),
            data: await response.json() as T,
            cost: 0,
            budgetRemaining: this.tokenCache?.budgetLimit
              ? this.tokenCache.budgetLimit - this.tokenCache.budgetUsed
              : undefined,
          };
        }
        
        // Payment required (L402)
        if (response.status === 402) {
          return this.handleL402(response, method, url, requestOptions) as Promise<AgentResponse<T>>;
        }
        
        // Auth error - refresh token
        if (response.status === 401 && attempt < this.maxRetries - 1) {
          this.tokenCache = undefined;
          const newToken = await this.getToken();
          if (newToken) {
            headers.set('Authorization', `Bearer ${newToken}`);
          }
          continue;
        }
        
        // Other error
        throw new SatGateError(`Request failed: ${response.status} ${await response.text()}`);
        
      } catch (e) {
        clearTimeout(timeoutId);
        if (e instanceof SatGateError) throw e;
        throw new SatGateError(`Request failed: ${e}`);
      }
    }
    
    throw new SatGateError('Max retries exceeded');
  }
  
  async get<T = unknown>(path: string, options?: RequestInit): Promise<AgentResponse<T>> {
    return this.request<T>('GET', path, options);
  }
  
  async post<T = unknown>(path: string, body?: unknown, options?: RequestInit): Promise<AgentResponse<T>> {
    return this.request<T>('POST', path, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...Object.fromEntries(new Headers(options?.headers).entries()),
      },
      body: body ? JSON.stringify(body) : undefined,
    });
  }
  
  async put<T = unknown>(path: string, body?: unknown, options?: RequestInit): Promise<AgentResponse<T>> {
    return this.request<T>('PUT', path, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...Object.fromEntries(new Headers(options?.headers).entries()),
      },
      body: body ? JSON.stringify(body) : undefined,
    });
  }
  
  async delete<T = unknown>(path: string, options?: RequestInit): Promise<AgentResponse<T>> {
    return this.request<T>('DELETE', path, options);
  }
  
  /**
   * Ping the gateway to verify the capability token is valid.
   * 
   * GET /api/capability/ping with Bearer token
   */
  async ping(): Promise<Record<string, unknown>> {
    const response = await this.get<Record<string, unknown>>('/api/capability/ping');
    return response.data;
  }
  
  /**
   * Validate the current capability token.
   * 
   * POST /api/capability/validate
   */
  async validateToken(token?: string): Promise<Record<string, unknown>> {
    const t = token || this.tokenCache?.token;
    if (!t) throw new SatGateError('No token available to validate');
    
    const response = await this.post<Record<string, unknown>>(
      '/api/capability/validate',
      { token: t }
    );
    return response.data;
  }
  
  /**
   * Create a delegated (child) token for a worker agent.
   * 
   * POST /api/capability/delegate
   */
  async delegate(options: {
    caveats?: string[];
  } = {}): Promise<string> {
    if (!this.tokenCache) {
      await this.getToken();
    }
    
    if (!this.tokenCache) {
      throw new SatGateError('No token available for delegation');
    }
    
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);
    
    try {
      const response = await fetch(`${this.gatewayUrl}/api/capability/delegate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          parentToken: this.tokenCache.token,
          caveats: options.caveats || [],
        }),
        signal: controller.signal,
      });
      
      clearTimeout(timeoutId);
      
      if (!response.ok) {
        throw new SatGateError(`Delegation failed: ${await response.text()}`);
      }
      
      const result = await response.json() as { token: string };
      return result.token;
    } catch (e) {
      clearTimeout(timeoutId);
      if (e instanceof SatGateError) throw e;
      throw new SatGateError(`Delegation failed: ${e}`);
    }
  }
  
  /** Total cost incurred by this client */
  getTotalCost(): number {
    return this.totalCost;
  }
  
  /** Remaining budget (if budgetLimit was set) */
  getBudgetRemaining(): number | undefined {
    if (this.budgetLimit === undefined) return undefined;
    return Math.max(0, this.budgetLimit - this.totalCost);
  }
  
  /** The current cached token, if any */
  getCurrentToken(): string | undefined {
    if (this.tokenCache && Date.now() < this.tokenCache.expiresAt) {
      return this.tokenCache.token;
    }
    return undefined;
  }
  
  /** The signature of the current cached token, if any */
  getCurrentSignature(): string | undefined {
    return this.tokenCache?.signature;
  }
}

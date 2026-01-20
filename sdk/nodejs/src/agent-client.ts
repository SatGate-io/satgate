/**
 * SatGate Agent Client - Economic Middleware for AI Agents
 * 
 * Automatically handles:
 * - Identity badge-in (K8s, AWS, OIDC)
 * - 402 Payment Required challenges (Fiat402 and L402)
 * - Token caching and refresh
 * - Offline delegation for worker agents
 * - Budget tracking and alerts
 * 
 * @example
 * ```typescript
 * import { SatGateAgentClient } from '@satgate/sdk';
 * 
 * // Auto-detect environment (K8s/AWS)
 * const client = new SatGateAgentClient({
 *   gatewayUrl: 'https://gateway.internal',
 *   identity: 'auto'
 * });
 * 
 * // Make requests - 402 handling is automatic
 * const response = await client.get('/api/v1/data');
 * console.log(`Cost: $${response.cost}`);
 * ```
 */

import * as fs from 'fs';
import * as https from 'https';
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
  budgetLimit?: number;
  budgetUsed: number;
  caveats: string[];
}

// Identity Providers

export interface IdentityProvider {
  getCredentials(): Promise<string>;
  providerType(): string;
}

export class KubernetesIdentity implements IdentityProvider {
  private tokenPath: string;
  
  constructor(tokenPath = '/var/run/secrets/kubernetes.io/serviceaccount/token') {
    this.tokenPath = tokenPath;
  }
  
  async getCredentials(): Promise<string> {
    try {
      return fs.readFileSync(this.tokenPath, 'utf8').trim();
    } catch (e) {
      throw new SatGateError(`K8s token not found at ${this.tokenPath}`);
    }
  }
  
  providerType(): string {
    return 'kubernetes';
  }
  
  static isAvailable(): boolean {
    return fs.existsSync('/var/run/secrets/kubernetes.io/serviceaccount/token');
  }
}

export class AWSIdentity implements IdentityProvider {
  private roleArn?: string;
  
  constructor(roleArn?: string) {
    this.roleArn = roleArn;
  }
  
  async getCredentials(): Promise<string> {
    // Check for EKS IRSA token
    const irsaTokenPath = process.env.AWS_WEB_IDENTITY_TOKEN_FILE;
    if (irsaTokenPath && fs.existsSync(irsaTokenPath)) {
      const token = fs.readFileSync(irsaTokenPath, 'utf8').trim();
      return JSON.stringify({ irsaToken: token });
    }
    
    // Use AWS SDK if available
    try {
      const { STSClient, GetCallerIdentityCommand } = await import('@aws-sdk/client-sts');
      const sts = new STSClient({});
      const identity = await sts.send(new GetCallerIdentityCommand({}));
      return JSON.stringify({
        stsResponse: {
          Account: identity.Account,
          Arn: identity.Arn,
          UserId: identity.UserId,
        }
      });
    } catch (e) {
      throw new SatGateError('AWS SDK required for AWS identity');
    }
  }
  
  providerType(): string {
    return 'aws';
  }
  
  static isAvailable(): boolean {
    return !!(
      process.env.AWS_WEB_IDENTITY_TOKEN_FILE ||
      process.env.AWS_ACCESS_KEY_ID ||
      process.env.AWS_ROLE_ARN
    );
  }
}

export class OIDCIdentity implements IdentityProvider {
  private token?: string;
  private tokenEnv: string;
  
  constructor(token?: string, tokenEnv = 'OIDC_TOKEN') {
    this.token = token;
    this.tokenEnv = tokenEnv;
  }
  
  async getCredentials(): Promise<string> {
    const token = this.token || process.env[this.tokenEnv];
    if (!token) {
      throw new SatGateError(`OIDC token not found in ${this.tokenEnv}`);
    }
    return token;
  }
  
  providerType(): string {
    return 'oidc';
  }
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
  /** Gateway URL (for API requests) */
  gatewayUrl: string;
  /** 
   * Separate URL for Mint service (e.g., "http://gateway-admin:9090")
   * In enterprise deployments, Mint may be on a separate internal listener.
   * If not specified, uses gatewayUrl + "/v1/mint".
   * Can also be set via SATGATE_MINT_URL environment variable.
   */
  mintUrl?: string;
  /** Identity provider: "auto", "k8s", "aws", "oidc", "none", or IdentityProvider instance */
  identity?: string | IdentityProvider;
  /** Lightning wallet for L402 payments */
  wallet?: LightningWallet;
  /** Maximum budget to spend */
  budgetLimit?: number;
  /** Request timeout in milliseconds */
  timeout?: number;
  /** Maximum retry attempts for 402 challenges */
  maxRetries?: number;
  /** Callback when budget is running low */
  onBudgetAlert?: (remaining: number, limit: number) => void;
}

export class SatGateAgentClient {
  private gatewayUrl: string;
  private mintUrl: string;
  private identity?: IdentityProvider;
  private wallet?: LightningWallet;
  private budgetLimit?: number;
  private timeout: number;
  private maxRetries: number;
  private onBudgetAlert?: (remaining: number, limit: number) => void;
  
  private tokenCache?: CachedToken;
  private totalCost: number = 0;
  
  constructor(options: SatGateAgentClientOptions) {
    this.gatewayUrl = options.gatewayUrl.replace(/\/$/, '');
    
    // Mint URL: explicit > env > gateway_url
    this.mintUrl = (options.mintUrl || process.env.SATGATE_MINT_URL || this.gatewayUrl).replace(/\/$/, '');
    
    this.wallet = options.wallet;
    this.budgetLimit = options.budgetLimit;
    this.timeout = options.timeout || 30000;
    this.maxRetries = options.maxRetries || 3;
    this.onBudgetAlert = options.onBudgetAlert;
    
    // Initialize identity provider
    const identity = options.identity || 'auto';
    if (typeof identity === 'object') {
      this.identity = identity;
    } else if (identity === 'auto') {
      this.identity = this.autoDetectIdentity();
    } else if (identity === 'k8s' || identity === 'kubernetes') {
      this.identity = new KubernetesIdentity();
    } else if (identity === 'aws') {
      this.identity = new AWSIdentity();
    } else if (identity === 'oidc') {
      this.identity = new OIDCIdentity();
    } else if (identity !== 'none') {
      throw new SatGateError(`Unknown identity provider: ${identity}`);
    }
  }
  
  private autoDetectIdentity(): IdentityProvider | undefined {
    if (KubernetesIdentity.isAvailable()) {
      return new KubernetesIdentity();
    }
    if (AWSIdentity.isAvailable()) {
      return new AWSIdentity();
    }
    return undefined;
  }
  
  private async getToken(): Promise<string | undefined> {
    // Check cache
    if (this.tokenCache && Date.now() < this.tokenCache.expiresAt) {
      return this.tokenCache.token;
    }
    
    // No identity provider
    if (!this.identity) {
      return undefined;
    }
    
    // Mint new token (use separate mintUrl for enterprise deployments)
    try {
      const credentials = await this.identity.getCredentials();
      
      const response = await fetch(`${this.mintUrl}/v1/mint`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          provider: this.identity.providerType(),
          credentials,
        }),
      });
      
      if (!response.ok) {
        throw new AuthenticationError(`Mint failed: ${await response.text()}`);
      }
      
      const result = await response.json() as {
        macaroon: string;
        signature: string;
        budget?: { limit?: number };
        caveats?: string[];
      };
      
      this.tokenCache = {
        token: result.macaroon,
        signature: result.signature,
        expiresAt: Date.now() + 3600000, // 1 hour
        budgetLimit: result.budget?.limit,
        budgetUsed: 0,
        caveats: result.caveats || [],
      };
      
      return this.tokenCache.token;
    } catch (e) {
      throw new AuthenticationError(`Failed to get token: ${e}`);
    }
  }
  
  private async handle402(
    response: Response,
    method: string,
    url: string,
    options: RequestInit
  ): Promise<AgentResponse> {
    const wwwAuth = response.headers.get('WWW-Authenticate') || '';
    const contentType = response.headers.get('Content-Type') || '';
    
    let challenge: Record<string, unknown> = {};
    if (contentType.includes('application/json')) {
      try {
        challenge = await response.json() as Record<string, unknown>;
      } catch {}
    }
    
    const challengeType = challenge.type as string || '';
    
    // Fiat402 challenge (internal budget)
    if (wwwAuth.includes('SatGate-Billing') || challengeType === 'fiat402') {
      return this.handleFiat402(challenge, method, url, options);
    }
    
    // L402 challenge (Lightning payment)
    if (wwwAuth.includes('L402') || wwwAuth.includes('LSAT') || challengeType === 'l402') {
      return this.handleL402(wwwAuth, challenge, method, url, options);
    }
    
    throw new PaymentRequiredError('Unknown 402 challenge type', challenge);
  }
  
  private async handleFiat402(
    challenge: Record<string, unknown>,
    method: string,
    url: string,
    options: RequestInit
  ): Promise<AgentResponse> {
    const invoiceId = challenge.invoice_id as string;
    const amount = (challenge.amount as number) || 0;
    
    if (!invoiceId) {
      throw new PaymentRequiredError('Missing invoice_id in Fiat402 challenge', challenge);
    }
    
    // Check budget
    if (this.budgetLimit && this.totalCost + amount > this.budgetLimit) {
      throw new BudgetExceededError(
        `Budget exceeded: $${this.totalCost} + $${amount} > $${this.budgetLimit}`,
        this.totalCost,
        this.budgetLimit
      );
    }
    
    // Get receipt
    const receiptUrl = (challenge.receipt_url as string) || `/api/v1/billing/receipts?invoice_id=${invoiceId}`;
    const fullReceiptUrl = new URL(receiptUrl, this.gatewayUrl).toString();
    
    const receiptResponse = await fetch(fullReceiptUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ invoiceId }),
    });
    
    if (!receiptResponse.ok) {
      throw new PaymentRequiredError(`Failed to get receipt: ${await receiptResponse.text()}`, challenge);
    }
    
    const receiptData = await receiptResponse.json() as { receipt?: string; token?: string };
    const receiptToken = receiptData.receipt || receiptData.token;
    
    if (!receiptToken) {
      throw new PaymentRequiredError('No receipt token in response', challenge);
    }
    
    // Retry with receipt
    const headers = new Headers(options.headers);
    headers.set('Authorization', `Receipt ${receiptToken}`);
    
    const retryResponse = await fetch(url, { ...options, headers });
    
    // Track cost
    this.totalCost += amount;
    if (this.tokenCache) {
      this.tokenCache.budgetUsed += amount;
    }
    
    // Check budget alert
    if (this.onBudgetAlert && this.budgetLimit) {
      const remaining = this.budgetLimit - this.totalCost;
      if (remaining < this.budgetLimit * 0.2) {
        this.onBudgetAlert(remaining, this.budgetLimit);
      }
    }
    
    return {
      statusCode: retryResponse.status,
      headers: Object.fromEntries(retryResponse.headers.entries()),
      data: await retryResponse.json(),
      cost: amount,
      budgetRemaining: this.budgetLimit ? this.budgetLimit - this.totalCost : undefined,
    };
  }
  
  private async handleL402(
    wwwAuth: string,
    challenge: Record<string, unknown>,
    method: string,
    url: string,
    options: RequestInit
  ): Promise<AgentResponse> {
    if (!this.wallet) {
      throw new PaymentRequiredError('L402 payment required but no wallet configured', challenge);
    }
    
    // Parse L402 header
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
    
    // Build L402 token
    const l402Token = `L402 ${macaroon}:${preimage}`;
    
    // Retry with L402 token
    const headers = new Headers(options.headers);
    headers.set('Authorization', l402Token);
    
    const retryResponse = await fetch(url, { ...options, headers });
    
    const costSats = (challenge.amount as number) || 0;
    
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
    
    // Add token if available
    const token = await this.getToken();
    const headers = new Headers(options.headers);
    if (token) {
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
        
        // Payment required
        if (response.status === 402) {
          return this.handle402(response, method, url, requestOptions) as Promise<AgentResponse<T>>;
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
   * Create a delegated (child) token for a worker agent.
   */
  async delegate(options: {
    additionalCaveats?: string[];
    ttl?: string;
    budgetLimit?: number;
  } = {}): Promise<string> {
    if (!this.tokenCache) {
      await this.getToken();
    }
    
    if (!this.tokenCache) {
      throw new SatGateError('No token available for delegation');
    }
    
    const caveats = options.additionalCaveats || [];
    if (options.ttl) caveats.push(`ttl = ${options.ttl}`);
    if (options.budgetLimit) caveats.push(`budget_limit = ${options.budgetLimit}`);
    
    // Use mintUrl for delegation (may be on separate listener)
    const response = await fetch(`${this.mintUrl}/v1/mint/delegate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        parentMacaroon: this.tokenCache.token,
        additionalCaveats: caveats,
      }),
    });
    
    if (!response.ok) {
      throw new SatGateError(`Delegation failed: ${await response.text()}`);
    }
    
    const result = await response.json() as { macaroon: string };
    return result.macaroon;
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
}

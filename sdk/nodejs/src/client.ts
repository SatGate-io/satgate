/**
 * SatGate Gateway Client
 */

import {
  Token,
  TokenInfo,
  BanRecord,
  Stats,
  MintRequest,
  DelegateRequest,
  BanRequest,
  GatewayConfig,
} from './types';
import {
  SatGateError,
  AuthenticationError,
  NotFoundError,
  ValidationError,
} from './errors';

export interface SatGateClientOptions {
  /** Gateway admin API URL */
  url: string;
  /** Admin API token */
  token: string;
  /** Request timeout in milliseconds (default: 30000) */
  timeout?: number;
}

export class SatGateClient {
  private readonly baseUrl: string;
  private readonly token: string;
  private readonly timeout: number;

  public readonly tokens: TokensService;
  public readonly governance: GovernanceService;
  public readonly config: ConfigService;
  public readonly stats: StatsService;

  constructor(options: SatGateClientOptions) {
    this.baseUrl = options.url.replace(/\/$/, '');
    this.token = options.token;
    this.timeout = options.timeout ?? 30000;

    this.tokens = new TokensService(this);
    this.governance = new GovernanceService(this);
    this.config = new ConfigService(this);
    this.stats = new StatsService(this);
  }

  async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(`${this.baseUrl}${path}`, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'X-Admin-Token': this.token,
        },
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (response.status === 401) {
        throw new AuthenticationError('Invalid admin token');
      }
      if (response.status === 404) {
        throw new NotFoundError();
      }
      if (response.status === 400) {
        const error = await response.json();
        throw new ValidationError(error.error || 'Validation failed');
      }
      if (!response.ok) {
        const error = await response.json().catch(() => ({}));
        throw new SatGateError(
          error.error || `Request failed with status ${response.status}`,
          response.status
        );
      }

      if (response.status === 204 || response.headers.get('content-length') === '0') {
        return {} as T;
      }

      return response.json();
    } catch (error) {
      clearTimeout(timeoutId);
      if (error instanceof SatGateError) {
        throw error;
      }
      throw new SatGateError(`Request failed: ${error}`);
    }
  }

  /** Check if gateway is healthy */
  async health(): Promise<boolean> {
    try {
      const result = await this.request<{ status: string }>('GET', '/healthz');
      return result.status === 'ok';
    } catch {
      return false;
    }
  }
}

class TokensService {
  constructor(private client: SatGateClient) {}

  /** Mint a new root token */
  async mint(request: MintRequest = {}): Promise<Token> {
    return this.client.request<Token>('POST', '/api/v1/tokens', {
      scope: request.scope ?? 'api:*',
      expiresIn: request.expiresIn ?? 3600,
    });
  }

  /** List all tokens */
  async list(): Promise<TokenInfo[]> {
    return this.client.request<TokenInfo[]>('GET', '/api/v1/tokens');
  }

  /** Get token details */
  async get(signature: string): Promise<TokenInfo> {
    return this.client.request<TokenInfo>('GET', `/api/v1/tokens/${signature}`);
  }

  /** Revoke a token */
  async revoke(signature: string): Promise<void> {
    await this.client.request('DELETE', `/api/v1/tokens/${signature}`);
  }

  /** Delegate (create child) token */
  async delegate(request: DelegateRequest): Promise<Token> {
    return this.client.request<Token>('POST', '/api/v1/tokens/delegate', request);
  }
}

class GovernanceService {
  constructor(private client: SatGateClient) {}

  /** Ban a token */
  async ban(signature: string, reason?: string): Promise<void> {
    await this.client.request('POST', '/api/v1/governance/ban', {
      tokenSignature: signature,
      reason: reason ?? '',
    });
  }

  /** Unban a token */
  async unban(signature: string): Promise<void> {
    await this.client.request('DELETE', `/api/v1/governance/ban/${signature}`);
  }

  /** Get ban list */
  async getBanList(): Promise<BanRecord[]> {
    return this.client.request<BanRecord[]>('GET', '/api/v1/governance/banlist');
  }

  /** Reset all governance data */
  async reset(): Promise<void> {
    await this.client.request('POST', '/api/v1/governance/reset');
  }
}

class ConfigService {
  constructor(private client: SatGateClient) {}

  /** Get current configuration */
  async get(): Promise<GatewayConfig> {
    return this.client.request<GatewayConfig>('GET', '/api/v1/config');
  }

  /** Validate a configuration */
  async validate(config: GatewayConfig): Promise<boolean> {
    try {
      await this.client.request('POST', '/api/v1/config/validate', config);
      return true;
    } catch {
      return false;
    }
  }
}

class StatsService {
  constructor(private client: SatGateClient) {}

  /** Get gateway statistics */
  async get(): Promise<Stats> {
    return this.client.request<Stats>('GET', '/api/v1/stats');
  }

  /** Get per-token statistics */
  async getTokenStats(): Promise<TokenInfo[]> {
    return this.client.request<TokenInfo[]>('GET', '/api/v1/stats/tokens');
  }
}




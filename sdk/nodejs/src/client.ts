/**
 * SatGate Gateway Client (OSS)
 * 
 * Admin client for the SatGate OSS Gateway. Provides access to:
 * - Token minting (POST /api/capability/mint)
 * - Token validation (POST /api/capability/validate)
 * - Token delegation (POST /api/capability/delegate)
 * - Governance: ban, graph, reset
 * - Health checks
 */

import {
  Token,
  DelegateRequest,
  GraphData,
} from './types';
import {
  SatGateError,
  AuthenticationError,
  NotFoundError,
  ValidationError,
} from './errors';

export interface SatGateClientOptions {
  /** Gateway URL (e.g., "http://localhost:8080") */
  url: string;
  /** Admin API token (sent as X-Admin-Token header) */
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

  constructor(options: SatGateClientOptions) {
    this.baseUrl = options.url.replace(/\/$/, '');
    this.token = options.token;
    this.timeout = options.timeout ?? 30000;

    this.tokens = new TokensService(this);
    this.governance = new GovernanceService(this);
  }

  async request<T>(
    method: string,
    path: string,
    body?: unknown,
    options?: { includeAdminToken?: boolean }
  ): Promise<T> {
    const includeAdminToken = options?.includeAdminToken ?? true;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      };
      if (includeAdminToken) {
        headers['X-Admin-Token'] = this.token;
      }

      const response = await fetch(`${this.baseUrl}${path}`, {
        method,
        headers,
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

  /** Check if gateway is healthy (GET /health) */
  async health(): Promise<boolean> {
    try {
      const result = await this.request<{ status: string }>(
        'GET', '/health', undefined, { includeAdminToken: false }
      );
      return result.status === 'healthy';
    } catch {
      return false;
    }
  }

  /** 
   * Ping the gateway with a capability token (GET /api/capability/ping)
   * @param bearerToken The capability token to verify
   */
  async ping(bearerToken: string): Promise<Record<string, unknown>> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);
    
    try {
      const response = await fetch(`${this.baseUrl}/api/capability/ping`, {
        method: 'GET',
        headers: { 'Authorization': `Bearer ${bearerToken}` },
        signal: controller.signal,
      });
      clearTimeout(timeoutId);
      return response.json();
    } catch (error) {
      clearTimeout(timeoutId);
      throw new SatGateError(`Ping failed: ${error}`);
    }
  }
}

class TokensService {
  constructor(private client: SatGateClient) {}

  /**
   * Mint a new capability token.
   * 
   * POST /api/capability/mint (requires X-Admin-Token header)
   * 
   * @param scope Token scope (default: "api:*")
   * @param duration Token lifetime as Go duration string (default: "1h")
   */
  async mint(options: { scope?: string; duration?: string } = {}): Promise<Token> {
    return this.client.request<Token>('POST', '/api/capability/mint', {
      scope: options.scope ?? 'api:*',
      duration: options.duration ?? '1h',
    });
  }

  /**
   * Validate a capability token.
   * 
   * POST /api/capability/validate
   * 
   * @param token The token string to validate
   */
  async validate(token: string): Promise<{ valid: boolean; identifier?: string; caveats?: string[]; error?: string }> {
    return this.client.request('POST', '/api/capability/validate', { token }, { includeAdminToken: false });
  }

  /**
   * Delegate (create child) token.
   * 
   * POST /api/capability/delegate
   */
  async delegate(request: DelegateRequest): Promise<Token> {
    return this.client.request<Token>(
      'POST', '/api/capability/delegate',
      request,
      { includeAdminToken: false }
    );
  }
}

class GovernanceService {
  constructor(private client: SatGateClient) {}

  /**
   * Ban a token by its signature.
   * 
   * POST /api/governance/ban (requires X-Admin-Token header)
   */
  async ban(signature: string, reason?: string): Promise<void> {
    await this.client.request('POST', '/api/governance/ban', {
      tokenSignature: signature,
      reason: reason ?? '',
    });
  }

  /**
   * Get the governance graph (token lineage, stats).
   * 
   * GET /api/governance/graph
   */
  async getGraph(): Promise<GraphData> {
    return this.client.request<GraphData>(
      'GET', '/api/governance/graph',
      undefined,
      { includeAdminToken: false }
    );
  }

  /**
   * Reset all governance data (tokens, bans, usage).
   * 
   * POST /api/governance/reset (requires X-Admin-Token header)
   */
  async reset(): Promise<void> {
    await this.client.request('POST', '/api/governance/reset');
  }
}

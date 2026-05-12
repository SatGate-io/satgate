import fetch, { Response } from 'node-fetch';
import { SatGateAuthError, SatGateError } from './errors';

const BETA_ACCESS_MESSAGE =
  'This API namespace requires private beta access. Visit cloud.satgate.io/docs to request access.';

export interface SatGateOptions {
  apiKey?: string;
  baseUrl?: string;
  tenant?: string;
  timeout?: number;
}

export type JsonValue = string | number | boolean | null | JsonObject | JsonValue[];
export type JsonObject = { [key: string]: JsonValue };

export interface CapabilityRequest {
  [key: string]: unknown;
  task: string;
  agent: string;
  allow: string[];
  budgetUsd?: number;
  budget_usd?: number;
  expiresIn?: string;
  expires_in?: string;
}

export interface PayRequest {
  [key: string]: unknown;
  upstream: string;
  capability: unknown;
  maxUsd?: number;
  max_usd?: number;
}

export type SatGateReceipt = JsonObject & {
  decision?: string;
  evidencePackId?: string;
  evidence_pack_id?: string;
};

export class SatGate {
  private readonly apiKey?: string;
  private readonly baseUrl: string;
  private readonly tenant?: string;
  private readonly timeout: number;

  constructor(options: SatGateOptions = {}) {
    this.apiKey = options.apiKey;
    this.baseUrl = (options.baseUrl ?? 'https://api.satgate.io').replace(/\/$/, '');
    this.tenant = options.tenant;
    this.timeout = options.timeout ?? 30000;
  }

  async issue(request: CapabilityRequest): Promise<SatGateReceipt> {
    try {
      return await this.post<SatGateReceipt>('/v1/issue', this.toApiPayload(request) as JsonObject);
    } catch (err) {
      if (err instanceof SatGateAuthError && err.statusCode === 404) {
        // Compatibility fallback for early private-beta clients and gateways.
        return this.post<SatGateReceipt>('/v1/capabilities', this.toApiPayload(request) as JsonObject);
      }
      throw err;
    }
  }

  async pay(request: PayRequest): Promise<SatGateReceipt> {
    return this.post<SatGateReceipt>('/v1/pay', this.toApiPayload(request) as JsonObject);
  }

  async verify(receipt: unknown): Promise<SatGateReceipt> {
    return this.post<SatGateReceipt>('/v1/verify', { receipt: this.toApiPayload(receipt) });
  }

  private async post<T>(path: string, body: JsonObject): Promise<T> {
    if (!this.apiKey) {
      throw new SatGateAuthError(BETA_ACCESS_MESSAGE, 401);
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);
    const headers: Record<string, string> = {
      'Authorization': `Bearer ${this.apiKey}`,
      'Content-Type': 'application/json',
    };
    if (this.tenant) {
      headers['X-SatGate-Tenant'] = this.tenant;
    }

    let response: Response;
    try {
      response = await fetch(`${this.baseUrl}${path}`, {
        method: 'POST',
        headers,
        body: JSON.stringify(body),
        signal: controller.signal,
      });
    } catch (err: unknown) {
      clearTimeout(timeoutId);
      throw new SatGateError(`Request failed: ${err}`);
    }
    clearTimeout(timeoutId);

    if ([401, 403, 404, 503].includes(response.status)) {
      throw new SatGateAuthError(BETA_ACCESS_MESSAGE, response.status);
    }

    if (!response.ok) {
      const errBody = await response.json().catch(() => ({})) as Record<string, unknown>;
      throw new SatGateError(
        (errBody.message as string) || (errBody.error as string) || `Request failed with status ${response.status}`,
        response.status
      );
    }

    if (response.status === 204) {
      return {} as T;
    }
    return response.json() as Promise<T>;
  }

  private toApiPayload(value: unknown): JsonValue {
    if (Array.isArray(value)) {
      return value.map((item) => this.toApiPayload(item));
    }
    if (value && typeof value === 'object') {
      const output: JsonObject = {};
      for (const [key, val] of Object.entries(value as Record<string, unknown>)) {
        output[this.toSnakeCase(key)] = this.toApiPayload(val);
      }
      return output;
    }
    return value as JsonValue;
  }

  private toSnakeCase(key: string): string {
    return key.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);
  }
}

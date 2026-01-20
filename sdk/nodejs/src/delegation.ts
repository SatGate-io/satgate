/**
 * SatGate Delegation Helpers
 * 
 * Utilities for creating and managing macaroon delegation hierarchies.
 * Supports common patterns like team budgets, time-limited access, and scope restrictions.
 */

import { SatGateClient } from './client';
import { Token, DelegateRequest } from './types';

/**
 * Caveat builders for common delegation patterns
 */
export const Caveats = {
  /**
   * Restrict scope to specific actions
   * @example Caveats.scope('api:read')
   */
  scope: (scope: string): string => `scope = ${scope}`,

  /**
   * Set expiration time
   * @param seconds Seconds from now, or Date object
   */
  expires: (seconds: number | Date): string => {
    const expiry = typeof seconds === 'number'
      ? Math.floor(Date.now() / 1000) + seconds
      : Math.floor(seconds.getTime() / 1000);
    return `expires = ${expiry}`;
  },

  /**
   * Limit to specific routes
   * @example Caveats.routes(['/api/v1/*', '/health'])
   */
  routes: (patterns: string[]): string => `routes = ${patterns.join(',')}`,

  /**
   * Limit requests per period
   * @param count Max requests
   * @param period Period in seconds (default: 60 = per minute)
   */
  rateLimit: (count: number, period = 60): string => 
    `ratelimit = ${count}/${period}`,

  /**
   * Set a budget limit (for billing/metering)
   * @param amount Budget amount
   * @param currency Currency code (default: 'USD')
   */
  budget: (amount: number, currency = 'USD'): string =>
    `budget = ${amount} ${currency}`,

  /**
   * Restrict to specific IP addresses/CIDRs
   * @example Caveats.sourceIp(['10.0.0.0/8', '192.168.1.100'])
   */
  sourceIp: (cidrs: string[]): string => `source_ip = ${cidrs.join(',')}`,

  /**
   * Restrict to specific methods
   * @example Caveats.methods(['GET', 'POST'])
   */
  methods: (methods: string[]): string => `methods = ${methods.join(',')}`,

  /**
   * Add a team/cost-center identifier for attribution
   */
  team: (teamId: string): string => `team = ${teamId}`,

  /**
   * Add a project identifier for attribution
   */
  project: (projectId: string): string => `project = ${projectId}`,

  /**
   * Add a custom label for tracking
   */
  label: (key: string, value: string): string => `label:${key} = ${value}`,

  /**
   * Add an arbitrary caveat
   */
  custom: (caveat: string): string => caveat,
};

/**
 * DelegationBuilder provides a fluent API for creating delegated tokens
 */
export class DelegationBuilder {
  private parentToken: string;
  private caveats: string[] = [];

  constructor(parentToken: string) {
    this.parentToken = parentToken;
  }

  /** Add a scope restriction */
  withScope(scope: string): this {
    this.caveats.push(Caveats.scope(scope));
    return this;
  }

  /** Add an expiration */
  withExpiry(seconds: number | Date): this {
    this.caveats.push(Caveats.expires(seconds));
    return this;
  }

  /** Add route restrictions */
  withRoutes(patterns: string[]): this {
    this.caveats.push(Caveats.routes(patterns));
    return this;
  }

  /** Add rate limiting */
  withRateLimit(count: number, periodSeconds = 60): this {
    this.caveats.push(Caveats.rateLimit(count, periodSeconds));
    return this;
  }

  /** Add a budget limit */
  withBudget(amount: number, currency = 'USD'): this {
    this.caveats.push(Caveats.budget(amount, currency));
    return this;
  }

  /** Restrict to source IPs */
  withSourceIp(cidrs: string[]): this {
    this.caveats.push(Caveats.sourceIp(cidrs));
    return this;
  }

  /** Restrict to HTTP methods */
  withMethods(methods: string[]): this {
    this.caveats.push(Caveats.methods(methods));
    return this;
  }

  /** Add team attribution */
  forTeam(teamId: string): this {
    this.caveats.push(Caveats.team(teamId));
    return this;
  }

  /** Add project attribution */
  forProject(projectId: string): this {
    this.caveats.push(Caveats.project(projectId));
    return this;
  }

  /** Add a custom label */
  withLabel(key: string, value: string): this {
    this.caveats.push(Caveats.label(key, value));
    return this;
  }

  /** Add a raw caveat */
  withCaveat(caveat: string): this {
    this.caveats.push(caveat);
    return this;
  }

  /** Build the delegation request */
  build(): DelegateRequest {
    return {
      parentToken: this.parentToken,
      caveats: this.caveats,
    };
  }

  /** Execute the delegation via client */
  async delegate(client: SatGateClient): Promise<Token> {
    return client.tokens.delegate(this.build());
  }
}

/**
 * Create a new delegation builder
 * @param parentToken The parent token to delegate from
 */
export function delegate(parentToken: string): DelegationBuilder {
  return new DelegationBuilder(parentToken);
}

/**
 * Common delegation patterns
 */
export const DelegationPatterns = {
  /**
   * Create a read-only token
   */
  readOnly: (parentToken: string): DelegationBuilder =>
    delegate(parentToken)
      .withScope('api:read')
      .withMethods(['GET', 'HEAD', 'OPTIONS']),

  /**
   * Create a time-limited token (24 hours by default)
   */
  temporary: (parentToken: string, hours = 24): DelegationBuilder =>
    delegate(parentToken)
      .withExpiry(hours * 3600),

  /**
   * Create a team token with budget
   */
  teamBudget: (
    parentToken: string,
    teamId: string,
    budgetAmount: number,
    currency = 'USD'
  ): DelegationBuilder =>
    delegate(parentToken)
      .forTeam(teamId)
      .withBudget(budgetAmount, currency),

  /**
   * Create a rate-limited API client token
   */
  apiClient: (
    parentToken: string,
    clientId: string,
    requestsPerMinute = 100
  ): DelegationBuilder =>
    delegate(parentToken)
      .withLabel('client_id', clientId)
      .withRateLimit(requestsPerMinute, 60),

  /**
   * Create a webhook callback token (single route, short expiry)
   */
  webhook: (
    parentToken: string,
    callbackPath: string,
    expiryMinutes = 30
  ): DelegationBuilder =>
    delegate(parentToken)
      .withRoutes([callbackPath])
      .withMethods(['POST'])
      .withExpiry(expiryMinutes * 60),

  /**
   * Create a CI/CD pipeline token (specific routes, short-lived)
   */
  cicd: (
    parentToken: string,
    pipelineId: string,
    expiryMinutes = 60
  ): DelegationBuilder =>
    delegate(parentToken)
      .withLabel('pipeline', pipelineId)
      .withExpiry(expiryMinutes * 60),

  /**
   * Create an agent swarm token (budget + rate limit)
   */
  agentSwarm: (
    parentToken: string,
    swarmId: string,
    options: {
      budget?: number;
      currency?: string;
      requestsPerMinute?: number;
      maxAgents?: number;
    } = {}
  ): DelegationBuilder => {
    let builder = delegate(parentToken)
      .withLabel('swarm_id', swarmId);
    
    if (options.budget) {
      builder = builder.withBudget(options.budget, options.currency || 'USD');
    }
    if (options.requestsPerMinute) {
      builder = builder.withRateLimit(options.requestsPerMinute, 60);
    }
    if (options.maxAgents) {
      builder = builder.withLabel('max_agents', String(options.maxAgents));
    }
    
    return builder;
  },
};

/**
 * Delegation tree node for visualizing delegation hierarchies
 */
export interface DelegationNode {
  signature: string;
  scope?: string;
  team?: string;
  project?: string;
  budget?: string;
  expiresAt?: Date;
  children: DelegationNode[];
  metadata: Record<string, string>;
}

/**
 * Parse caveats from a token into structured data
 */
export function parseCaveats(caveats: string[]): Record<string, string> {
  const result: Record<string, string> = {};
  
  for (const caveat of caveats) {
    const match = caveat.match(/^([^=]+)\s*=\s*(.+)$/);
    if (match) {
      result[match[1].trim()] = match[2].trim();
    }
  }
  
  return result;
}

/**
 * Check if a caveat would be more restrictive than the parent
 */
export function isMoreRestrictive(
  parentCaveats: string[],
  childCaveat: string
): boolean {
  const [key] = childCaveat.split('=').map(s => s.trim());
  const parentParsed = parseCaveats(parentCaveats);
  const childParsed = parseCaveats([childCaveat]);
  
  // If parent doesn't have this caveat, child is more restrictive
  if (!(key in parentParsed)) {
    return true;
  }
  
  // For numeric caveats, child must be <= parent
  const numericCaveats = ['budget', 'ratelimit', 'expires'];
  if (numericCaveats.some(nc => key.includes(nc))) {
    const parentVal = parseFloat(parentParsed[key]) || Infinity;
    const childVal = parseFloat(childParsed[key]) || 0;
    return childVal <= parentVal;
  }
  
  // For scope caveats, child must be subset of parent
  if (key === 'scope') {
    const parentScopes = parentParsed[key].split(',');
    const childScopes = childParsed[key].split(',');
    return childScopes.every(cs => 
      parentScopes.some(ps => 
        ps === cs || ps === '*' || (ps.endsWith(':*') && cs.startsWith(ps.slice(0, -1)))
      )
    );
  }
  
  return true; // Default to allowing
}

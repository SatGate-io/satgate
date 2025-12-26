/**
 * Route summary generator - for dashboard display
 */

import { GatewayConfig, Route } from './types';

export interface RouteSummary {
  name: string;
  path: string;
  methods: string[];
  policy: string;
  tier?: string;
  priceSats?: number;
  upstream?: string;
}

/**
 * Generate a summary of routes for dashboard display
 */
export function generateRouteSummary(config: GatewayConfig): RouteSummary[] {
  return config.routes.map(route => {
    const match = route.match || {};
    const policy = route.policy || {};
    
    const summary: RouteSummary = {
      name: route.name,
      path: match.exactPath || match.pathPrefix || '*',
      methods: match.methods || ['ALL'],
      policy: policy.kind || 'unknown',
      upstream: route.upstream,
    };
    
    if (policy.kind === 'l402') {
      summary.tier = (policy as any).tier;
      summary.priceSats = (policy as any).priceSats;
    }
    
    return summary;
  });
}

/**
 * Generate a text summary for CLI output
 */
export function generateTextSummary(config: GatewayConfig): string {
  const lines: string[] = [];
  
  lines.push('Upstreams:');
  for (const [name, upstream] of Object.entries(config.upstreams || {})) {
    lines.push(`  • ${name}: ${upstream.url}`);
  }
  
  lines.push('');
  lines.push('Routes:');
  for (const route of config.routes || []) {
    const match = route.match || {};
    const policy = route.policy || {};
    const path = match.exactPath || match.pathPrefix || '*';
    
    let policyStr: string = policy.kind || 'unknown';
    if (policy.kind === 'l402') {
      const l402 = policy as any;
      policyStr = `l402 (${l402.priceSats} sats, tier: ${l402.tier})`;
    } else if (policy.kind === 'deny') {
      policyStr = `deny (${(policy as any).status || 403})`;
    }
    
    lines.push(`  • ${route.name}: ${path} → ${policyStr}`);
  }
  
  return lines.join('\n');
}


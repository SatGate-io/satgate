/**
 * Two-layer config validation:
 * 1. Schema validation - YAML structure and required fields
 * 2. Cloud policy validation - Cloud-specific rules
 */

import Ajv from 'ajv';
import addFormats from 'ajv-formats';
import { schema } from './schema';
import { GatewayConfig } from './types';

const ajv = new Ajv({ allErrors: true, strict: false });
addFormats(ajv);

const validateSchemaFn = ajv.compile(schema);

export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

/**
 * Layer 1: Schema validation
 * Validates YAML structure and required fields.
 * Does NOT check environment variables.
 */
export function validateSchema(config: unknown): ValidationResult {
  const valid = validateSchemaFn(config);
  
  if (valid) {
    return { valid: true, errors: [] };
  }
  
  const errors = (validateSchemaFn.errors || []).map(err => {
    const path = err.instancePath || '/';
    return `${path}: ${err.message}`;
  });
  
  return { valid: false, errors };
}

/**
 * Layer 2: Cloud policy validation
 * Enforces Cloud-specific rules on top of schema validation.
 */
export function validateCloudPolicy(config: GatewayConfig): ValidationResult {
  const errors: string[] = [];
  
  // Rule 1: Upstreams must be public HTTPS
  for (const [name, upstream] of Object.entries(config.upstreams || {})) {
    if (!upstream.url.startsWith('https://')) {
      errors.push(`Upstream "${name}": must use HTTPS (got: ${upstream.url})`);
    }
    
    if (isPrivateUrl(upstream.url)) {
      errors.push(`Upstream "${name}": cannot be a private/internal address`);
    }
  }
  
  // Rule 2: Must have default-deny catch-all
  const hasDefaultDeny = (config.routes || []).some(route => {
    if (route.policy?.kind !== 'deny') return false;
    const match = route.match || {};
    return match.pathPrefix === '/' || 
           route.name?.toLowerCase().includes('deny') ||
           route.name?.toLowerCase().includes('default');
  });
  
  if (!hasDefaultDeny) {
    errors.push('Config must include a default-deny catch-all route');
  }
  
  // Rule 3: Route limits
  const routeCount = (config.routes || []).length;
  if (routeCount > 50) {
    errors.push(`Too many routes (${routeCount}). Cloud v1 supports max 50 routes`);
  }
  
  // Rule 4: Upstream limits
  const upstreamCount = Object.keys(config.upstreams || {}).length;
  if (upstreamCount > 10) {
    errors.push(`Too many upstreams (${upstreamCount}). Cloud v1 supports max 10 upstreams`);
  }
  
  // Rule 5: L402 routes must reference valid upstreams
  for (const route of config.routes || []) {
    if (route.policy?.kind === 'l402' || route.policy?.kind === 'capability') {
      if (!route.upstream) {
        errors.push(`Route "${route.name}": L402/capability routes must specify an upstream`);
      } else if (!config.upstreams[route.upstream]) {
        errors.push(`Route "${route.name}": references undefined upstream "${route.upstream}"`);
      }
    }
  }
  
  // Rule 6: No admin plane config allowed (Cloud manages this)
  if (config.admin && config.admin.listen !== '127.0.0.1:9090') {
    errors.push('Admin plane configuration is managed by Cloud; remove or use default');
  }
  
  return { valid: errors.length === 0, errors };
}

/**
 * Check if URL points to private/internal address
 */
function isPrivateUrl(url: string): boolean {
  try {
    const parsed = new URL(url);
    const hostname = parsed.hostname.toLowerCase();
    
    // Localhost
    if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1') {
      return true;
    }
    
    // Private IPv4 ranges
    if (/^10\./.test(hostname)) return true;
    if (/^172\.(1[6-9]|2[0-9]|3[0-1])\./.test(hostname)) return true;
    if (/^192\.168\./.test(hostname)) return true;
    
    // Link-local
    if (/^169\.254\./.test(hostname)) return true;
    
    // .local, .internal, .corp domains
    if (/\.(local|internal|corp|lan|home)$/i.test(hostname)) return true;
    
    // Kubernetes internal
    if (/\.svc\.cluster\.local$/i.test(hostname)) return true;
    if (/\.default\.svc$/i.test(hostname)) return true;
    
    return false;
  } catch {
    return true; // Invalid URL = treat as private
  }
}


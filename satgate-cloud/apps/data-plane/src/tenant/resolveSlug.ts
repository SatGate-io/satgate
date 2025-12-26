/**
 * Tenant resolution from Host header
 */

const SLUG_REGEX = /^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$/;
const CLOUD_DOMAIN = '.satgate.cloud';

const RESERVED_SLUGS = new Set([
  'www',
  'api',
  'app',
  'admin',
  'dashboard',
  'cloud',
  'mail',
  'smtp',
  'ftp',
  'ssh',
  'test',
  'staging',
  'prod',
  'production',
  'dev',
  'development',
]);

/**
 * Extract and validate tenant slug from Host header
 * 
 * @param host - Host header value (e.g., "acme.satgate.cloud" or "acme.satgate.cloud:443")
 * @returns Validated slug or null if invalid
 */
export function resolveSlug(host: string): string | null {
  if (!host) return null;
  
  // Normalize: lowercase, strip port
  const normalized = host.toLowerCase().split(':')[0];
  
  // Must end with .satgate.cloud
  if (!normalized.endsWith(CLOUD_DOMAIN)) {
    return null;
  }
  
  // Extract slug (everything before .satgate.cloud)
  const slug = normalized.slice(0, -CLOUD_DOMAIN.length);
  
  // Validate slug format
  if (!SLUG_REGEX.test(slug)) {
    return null;
  }
  
  // Block reserved slugs
  if (RESERVED_SLUGS.has(slug)) {
    return null;
  }
  
  // Block punycode (internationalized domains)
  if (slug.startsWith('xn--')) {
    return null;
  }
  
  return slug;
}

/**
 * Validate a slug for project creation
 */
export function validateSlug(slug: string): { valid: boolean; reason?: string } {
  if (!slug) {
    return { valid: false, reason: 'Slug is required' };
  }
  
  const normalized = slug.toLowerCase();
  
  if (!SLUG_REGEX.test(normalized)) {
    return { 
      valid: false, 
      reason: 'Slug must be 3-64 characters, lowercase alphanumeric with hyphens' 
    };
  }
  
  if (RESERVED_SLUGS.has(normalized)) {
    return { valid: false, reason: 'This slug is reserved' };
  }
  
  if (normalized.startsWith('xn--')) {
    return { valid: false, reason: 'Internationalized slugs are not supported' };
  }
  
  return { valid: true };
}


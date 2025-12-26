/**
 * Config caching for multi-tenant gateway
 */

import { GatewayConfig, loadConfig } from '@satgate/gateway-config';
import { logger } from '@satgate/common';

interface CacheEntry {
  config: GatewayConfig;
  version: string;
  expiresAt: number;
}

// In-memory cache
const cache = new Map<string, CacheEntry>();

// Cache TTL (60 seconds for v1; add pub/sub invalidation later)
const TTL_MS = 60_000;

/**
 * Get config for a tenant, with caching
 */
export async function getConfig(slug: string): Promise<GatewayConfig> {
  const now = Date.now();
  const cached = cache.get(slug);
  
  // Return cached if not expired
  if (cached && cached.expiresAt > now) {
    return cached.config;
  }
  
  // Fetch from database
  const row = await fetchConfigFromDb(slug);
  
  if (!row) {
    throw new Error(`Project not found: ${slug}`);
  }
  
  // Parse and validate config
  const config = loadConfig(row.yaml_content, { skipCloudPolicy: false });
  
  // TODO: Render secrets into config
  // config = await renderSecrets(slug, config);
  
  // Cache it
  cache.set(slug, {
    config,
    version: row.version,
    expiresAt: now + TTL_MS,
  });
  
  logger.debug('Config loaded', { slug, version: row.version });
  
  return config;
}

/**
 * Invalidate cached config for a tenant
 */
export function invalidateConfig(slug: string): void {
  cache.delete(slug);
  logger.debug('Config invalidated', { slug });
}

/**
 * Fetch config from database
 */
async function fetchConfigFromDb(slug: string): Promise<{ yaml_content: string; version: string } | null> {
  // Import dynamically to avoid circular dependencies
  const { query } = await import('../db');
  
  const result = await query<{ yaml_content: string; version: string }>(
    `SELECT cv.yaml_content, cv.id::text as version
     FROM config_versions cv
     JOIN projects p ON cv.project_id = p.id
     WHERE p.slug = $1 AND cv.is_active = true`,
    [slug]
  );
  
  if (result.rowCount === 0) {
    return null;
  }
  
  return result.rows[0];
}


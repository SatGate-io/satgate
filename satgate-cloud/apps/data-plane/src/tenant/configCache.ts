/**
 * Config caching for multi-tenant gateway
 */

import * as crypto from 'crypto';
import { GatewayConfig, loadConfig } from '@satgate/gateway-config';
import { logger } from '@satgate/common';

interface CacheEntry {
  config: GatewayConfig;
  version: string;
  projectId: string;
  expiresAt: number;
}

// In-memory cache (stores rendered configs with secrets)
const cache = new Map<string, CacheEntry>();

// Cache TTL (60 seconds for v1; add pub/sub invalidation later)
const TTL_MS = 60_000;

// Secrets encryption key (must match control plane)
const SECRETS_KEY = process.env.SECRETS_ENCRYPTION_KEY;

/**
 * Get the encryption key as a 32-byte Buffer
 */
function getEncryptionKey(): Buffer {
  if (!SECRETS_KEY || SECRETS_KEY.length < 32) {
    throw new Error('SECRETS_ENCRYPTION_KEY not configured or too short (need 32+ chars)');
  }
  
  // Try base64 decode first (preferred)
  try {
    const decoded = Buffer.from(SECRETS_KEY, 'base64');
    if (decoded.length >= 32) {
      return decoded.slice(0, 32);
    }
  } catch {
    // Not valid base64, fall through
  }
  
  // Fall back to UTF-8 (less secure but backwards compatible)
  return Buffer.from(SECRETS_KEY.slice(0, 32), 'utf8');
}

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
  
  // Parse and validate config (skipCloudPolicy since we already validated on upload)
  let config = loadConfig(row.yaml_content, { skipCloudPolicy: true });
  
  // Render secrets into config
  const secrets = await fetchSecretsFromDb(row.project_id);
  config = renderSecrets(config, secrets);
  
  // Cache the rendered config
  cache.set(slug, {
    config,
    version: row.version,
    projectId: row.project_id,
    expiresAt: now + TTL_MS,
  });
  
  logger.debug('Config loaded', { slug, version: row.version, secretCount: Object.keys(secrets).length });
  
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
 * Get project ID for a tenant (from cache)
 */
export function getProjectId(slug: string): string | undefined {
  return cache.get(slug)?.projectId;
}

/**
 * Fetch config from database
 */
async function fetchConfigFromDb(slug: string): Promise<{ yaml_content: string; version: string; project_id: string } | null> {
  const { query } = await import('../db');
  
  const result = await query<{ yaml_content: string; version: string; project_id: string }>(
    `SELECT cv.yaml_content, cv.id::text as version, p.id::text as project_id
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

/**
 * Fetch and decrypt secrets for a project
 */
async function fetchSecretsFromDb(projectId: string): Promise<Record<string, string>> {
  const { query } = await import('../db');
  
  const result = await query<{ key: string; value_encrypted: Buffer }>(
    `SELECT key, value_encrypted FROM project_secrets WHERE project_id = $1`,
    [projectId]
  );
  
  const secrets: Record<string, string> = {};
  
  for (const row of result.rows) {
    try {
      secrets[row.key] = decrypt(row.value_encrypted);
    } catch (err) {
      logger.error('Failed to decrypt secret', { projectId, key: row.key, error: (err as Error).message });
    }
  }
  
  return secrets;
}

/**
 * Decrypt a secret value using AES-256-GCM
 * Format: Nonce (12 bytes) + Ciphertext + AuthTag (16 bytes)
 */
function decrypt(encrypted: Buffer): string {
  const key = getEncryptionKey();
  
  // Extract components
  const nonce = encrypted.slice(0, 12);
  const authTag = encrypted.slice(-16);
  const ciphertext = encrypted.slice(12, -16);
  
  const decipher = crypto.createDecipheriv('aes-256-gcm', key, nonce);
  decipher.setAuthTag(authTag);
  
  return Buffer.concat([
    decipher.update(ciphertext),
    decipher.final(),
  ]).toString('utf8');
}

/**
 * Render secrets into config
 * Replaces {{SECRET_NAME}} placeholders in addHeaders values
 */
function renderSecrets(config: GatewayConfig, secrets: Record<string, string>): GatewayConfig {
  const rendered = JSON.parse(JSON.stringify(config)) as GatewayConfig;
  
  for (const [name, upstream] of Object.entries(rendered.upstreams || {})) {
    if (upstream.addHeaders) {
      for (const [header, value] of Object.entries(upstream.addHeaders)) {
        // Replace {{SECRET_NAME}} patterns
        upstream.addHeaders[header] = value.replace(
          /\{\{([A-Z][A-Z0-9_]*)\}\}/g,
          (match, secretKey) => {
            if (secrets[secretKey]) {
              return secrets[secretKey];
            }
            logger.warn('Secret not found for placeholder', { upstream: name, header, secretKey });
            return match; // Leave placeholder if secret not found
          }
        );
      }
    }
  }
  
  return rendered;
}


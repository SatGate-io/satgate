/**
 * Project secrets CRUD
 * 
 * Secrets are encrypted at rest and used for:
 * - Upstream API keys (rendered into addHeaders)
 * - Other sensitive configuration
 */

import { Router, Request, Response, IRouter } from 'express';
import * as crypto from 'crypto';
import { query } from '../db';
import { requireAuth, AuthenticatedRequest } from '../auth';
import { logger } from '@satgate/common';

const router: IRouter = Router();

// All secret routes require auth
router.use(requireAuth);

// Encryption key from environment (must be 32 bytes for AES-256)
const SECRETS_KEY = process.env.SECRETS_ENCRYPTION_KEY;

/**
 * Encrypt a secret value
 */
function encrypt(value: string): Buffer {
  if (!SECRETS_KEY || SECRETS_KEY.length < 32) {
    throw new Error('SECRETS_ENCRYPTION_KEY not configured or too short');
  }
  
  const key = Buffer.from(SECRETS_KEY.slice(0, 32), 'utf8');
  const iv = crypto.randomBytes(16);
  const cipher = crypto.createCipheriv('aes-256-cbc', key, iv);
  
  const encrypted = Buffer.concat([
    cipher.update(value, 'utf8'),
    cipher.final(),
  ]);
  
  // Return IV + ciphertext
  return Buffer.concat([iv, encrypted]);
}

/**
 * Decrypt a secret value
 */
function decrypt(encrypted: Buffer): string {
  if (!SECRETS_KEY || SECRETS_KEY.length < 32) {
    throw new Error('SECRETS_ENCRYPTION_KEY not configured or too short');
  }
  
  const key = Buffer.from(SECRETS_KEY.slice(0, 32), 'utf8');
  const iv = encrypted.slice(0, 16);
  const ciphertext = encrypted.slice(16);
  
  const decipher = crypto.createDecipheriv('aes-256-cbc', key, iv);
  
  return Buffer.concat([
    decipher.update(ciphertext),
    decipher.final(),
  ]).toString('utf8');
}

/**
 * Validate secret key name
 */
function isValidSecretKey(key: string): boolean {
  // Alphanumeric + underscore, 1-64 chars, must start with letter
  return /^[A-Z][A-Z0-9_]{0,63}$/i.test(key);
}

/**
 * GET /projects/:slug/secrets
 * List secret keys (not values)
 */
router.get('/:slug/secrets', async (req: Request, res: Response) => {
  const authReq = req as AuthenticatedRequest;
  const { slug } = req.params;
  
  try {
    // Verify project ownership
    const projectResult = await query(
      `SELECT id FROM projects WHERE tenant_id = $1 AND slug = $2`,
      [authReq.tenant.id, slug]
    );
    
    if (projectResult.rowCount === 0) {
      return res.status(404).json({ error: 'Project not found' });
    }
    
    const projectId = projectResult.rows[0].id;
    
    const result = await query(
      `SELECT key, created_at, updated_at FROM project_secrets 
       WHERE project_id = $1 ORDER BY key`,
      [projectId]
    );
    
    return res.json({
      secrets: result.rows.map((row: any) => ({
        key: row.key,
        createdAt: row.created_at,
        updatedAt: row.updated_at,
      })),
    });
  } catch (err) {
    logger.error('List secrets error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * PUT /projects/:slug/secrets/:key
 * Create or update a secret
 */
router.put('/:slug/secrets/:key', async (req: Request, res: Response) => {
  const authReq = req as AuthenticatedRequest;
  const { slug, key } = req.params;
  const { value } = req.body;
  
  try {
    // Validate key
    if (!isValidSecretKey(key)) {
      return res.status(400).json({ 
        error: 'Invalid secret key. Must be alphanumeric with underscores, start with letter, max 64 chars.' 
      });
    }
    
    // Validate value
    if (!value || typeof value !== 'string' || value.length > 4096) {
      return res.status(400).json({ error: 'Secret value required, max 4096 chars' });
    }
    
    // Verify project ownership
    const projectResult = await query(
      `SELECT id FROM projects WHERE tenant_id = $1 AND slug = $2`,
      [authReq.tenant.id, slug]
    );
    
    if (projectResult.rowCount === 0) {
      return res.status(404).json({ error: 'Project not found' });
    }
    
    const projectId = projectResult.rows[0].id;
    
    // Check secret limit (20 per project)
    const countResult = await query(
      `SELECT COUNT(*) as count FROM project_secrets WHERE project_id = $1`,
      [projectId]
    );
    
    const existing = await query(
      `SELECT id FROM project_secrets WHERE project_id = $1 AND key = $2`,
      [projectId, key.toUpperCase()]
    );
    
    if (existing.rowCount === 0 && parseInt(countResult.rows[0].count, 10) >= 20) {
      return res.status(403).json({ error: 'Secret limit reached (max 20 per project)' });
    }
    
    // Encrypt value
    const encryptedValue = encrypt(value);
    
    // Upsert secret
    await query(
      `INSERT INTO project_secrets (project_id, key, value_encrypted)
       VALUES ($1, $2, $3)
       ON CONFLICT (project_id, key) 
       DO UPDATE SET value_encrypted = $3, updated_at = NOW()`,
      [projectId, key.toUpperCase(), encryptedValue]
    );
    
    logger.info('Secret updated', { 
      tenantId: authReq.tenant.id,
      projectSlug: slug,
      key: key.toUpperCase(),
    });
    
    return res.json({ success: true, key: key.toUpperCase() });
  } catch (err) {
    logger.error('Update secret error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * DELETE /projects/:slug/secrets/:key
 * Delete a secret
 */
router.delete('/:slug/secrets/:key', async (req: Request, res: Response) => {
  const authReq = req as AuthenticatedRequest;
  const { slug, key } = req.params;
  
  try {
    // Verify project ownership
    const projectResult = await query(
      `SELECT id FROM projects WHERE tenant_id = $1 AND slug = $2`,
      [authReq.tenant.id, slug]
    );
    
    if (projectResult.rowCount === 0) {
      return res.status(404).json({ error: 'Project not found' });
    }
    
    const projectId = projectResult.rows[0].id;
    
    const result = await query(
      `DELETE FROM project_secrets WHERE project_id = $1 AND key = $2 RETURNING id`,
      [projectId, key.toUpperCase()]
    );
    
    if (result.rowCount === 0) {
      return res.status(404).json({ error: 'Secret not found' });
    }
    
    logger.info('Secret deleted', { 
      tenantId: authReq.tenant.id,
      projectSlug: slug,
      key: key.toUpperCase(),
    });
    
    return res.json({ success: true });
  } catch (err) {
    logger.error('Delete secret error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;

/**
 * Get decrypted secrets for a project (used by data plane)
 * This is an internal function, not exposed via HTTP
 */
export async function getProjectSecrets(projectId: string): Promise<Record<string, string>> {
  const result = await query<{ key: string; value_encrypted: Buffer }>(
    `SELECT key, value_encrypted FROM project_secrets WHERE project_id = $1`,
    [projectId]
  );
  
  const secrets: Record<string, string> = {};
  
  for (const row of result.rows) {
    try {
      secrets[row.key] = decrypt(row.value_encrypted);
    } catch (err) {
      logger.error('Failed to decrypt secret', { projectId, key: row.key });
    }
  }
  
  return secrets;
}


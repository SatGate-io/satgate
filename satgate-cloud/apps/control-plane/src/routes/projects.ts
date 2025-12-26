/**
 * Project CRUD routes
 */

import { Router, Request, Response, IRouter } from 'express';
import * as crypto from 'crypto';
import { query, transaction } from '../db';
import { requireAuth, AuthenticatedRequest } from '../auth';
import { validateConfig, normalizeConfig, printSummary, validateCloudPolicy } from '@satgate/gateway-config';
import { logger, ValidationError } from '@satgate/common';

const router: IRouter = Router();

// All project routes require auth
router.use(requireAuth);

/**
 * Generate a URL-safe project slug
 */
function generateProjectSlug(name: string): string {
  const base = name
    .toLowerCase()
    .replace(/[^a-z0-9]/g, '-')
    .replace(/-+/g, '-')
    .substring(0, 20);
  const suffix = crypto.randomBytes(4).toString('hex');
  return `${base}-${suffix}`;
}

/**
 * GET /projects
 * List all projects for tenant
 */
router.get('/', async (req: Request, res: Response) => {
  const authReq = req as AuthenticatedRequest;
  
  try {
    const result = await query(
      `SELECT id, slug, name, created_at, updated_at,
              (SELECT COUNT(*) FROM config_versions cv WHERE cv.project_id = p.id) as config_count
       FROM projects p
       WHERE tenant_id = $1
       ORDER BY created_at DESC`,
      [authReq.tenant.id]
    );
    
    return res.json({
      projects: result.rows.map((row: any) => ({
        id: row.id,
        slug: row.slug,
        name: row.name,
        configCount: parseInt(row.config_count, 10),
        createdAt: row.created_at,
        updatedAt: row.updated_at,
      })),
    });
  } catch (err) {
    logger.error('List projects error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * POST /projects
 * Create a new project
 */
router.post('/', async (req: Request, res: Response) => {
  const authReq = req as AuthenticatedRequest;
  
  try {
    const { name } = req.body;
    
    if (!name || typeof name !== 'string' || name.length < 2 || name.length > 50) {
      return res.status(400).json({ error: 'Name must be 2-50 characters' });
    }
    
    // Check project limit (10 per tenant for v1)
    const countResult = await query(
      `SELECT COUNT(*) as count FROM projects WHERE tenant_id = $1`,
      [authReq.tenant.id]
    );
    
    if (parseInt(countResult.rows[0].count, 10) >= 10) {
      return res.status(403).json({ error: 'Project limit reached (max 10)' });
    }
    
    const slug = generateProjectSlug(name);
    
    const result = await query(
      `INSERT INTO projects (tenant_id, slug, name) 
       VALUES ($1, $2, $3) 
       RETURNING id, slug, name, created_at`,
      [authReq.tenant.id, slug, name]
    );
    
    const project = result.rows[0];
    
    logger.info('Project created', { 
      tenantId: authReq.tenant.id, 
      projectSlug: slug 
    });
    
    return res.status(201).json({
      project: {
        id: project.id,
        slug: project.slug,
        name: project.name,
        // Data plane resolves host as: <project.slug>.satgate.cloud
        host: `${project.slug}.satgate.cloud`,
        createdAt: project.created_at,
      },
    });
  } catch (err) {
    logger.error('Create project error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * GET /projects/:slug
 * Get project details
 */
router.get('/:slug', async (req: Request, res: Response) => {
  const authReq = req as AuthenticatedRequest;
  const { slug } = req.params;
  
  try {
    const result = await query(
      `SELECT p.id, p.slug, p.name, p.created_at, p.updated_at,
              cv.id as config_id, cv.version, cv.yaml_content, cv.is_active
       FROM projects p
       LEFT JOIN config_versions cv ON cv.project_id = p.id AND cv.is_active = true
       WHERE p.tenant_id = $1 AND p.slug = $2`,
      [authReq.tenant.id, slug]
    );
    
    if (result.rowCount === 0) {
      return res.status(404).json({ error: 'Project not found' });
    }
    
    const row = result.rows[0];
    
    return res.json({
      project: {
        id: row.id,
        slug: row.slug,
        name: row.name,
        host: `${row.slug}.satgate.cloud`,
        activeConfig: row.config_id ? {
          id: row.config_id,
          version: row.version,
        } : null,
        createdAt: row.created_at,
        updatedAt: row.updated_at,
      },
    });
  } catch (err) {
    logger.error('Get project error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * DELETE /projects/:slug
 * Delete a project
 */
router.delete('/:slug', async (req: Request, res: Response) => {
  const authReq = req as AuthenticatedRequest;
  const { slug } = req.params;
  
  try {
    const result = await query(
      `DELETE FROM projects WHERE tenant_id = $1 AND slug = $2 RETURNING id`,
      [authReq.tenant.id, slug]
    );
    
    if (result.rowCount === 0) {
      return res.status(404).json({ error: 'Project not found' });
    }
    
    logger.info('Project deleted', { 
      tenantId: authReq.tenant.id, 
      projectSlug: slug 
    });
    
    return res.json({ success: true });
  } catch (err) {
    logger.error('Delete project error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * POST /projects/:slug/config
 * Upload a new config version
 */
router.post('/:slug/config', async (req: Request, res: Response) => {
  const authReq = req as AuthenticatedRequest;
  const { slug } = req.params;
  
  try {
    const { yaml } = req.body;
    
    if (!yaml || typeof yaml !== 'string') {
      return res.status(400).json({ error: 'YAML config is required' });
    }
    
    // Get project
    const projectResult = await query(
      `SELECT id FROM projects WHERE tenant_id = $1 AND slug = $2`,
      [authReq.tenant.id, slug]
    );
    
    if (projectResult.rowCount === 0) {
      return res.status(404).json({ error: 'Project not found' });
    }
    
    const projectId = projectResult.rows[0].id;
    
    // Validate config
    const validation = validateConfig(yaml);
    
    if (!validation.valid) {
      return res.status(400).json({ 
        error: 'Invalid configuration',
        details: validation.errors,
      });
    }
    
    // Cloud policy validation (schema-valid + Cloud guardrails)
    const normalizedForPolicy = normalizeConfig(validation.config!);
    const cloudValidation = validateCloudPolicy(normalizedForPolicy);
    if (!cloudValidation.valid) {
      return res.status(400).json({
        error: 'Configuration not allowed for Cloud',
        details: cloudValidation.errors,
      });
    }
    
    // Create new version (deactivate old, activate new)
    const result = await transaction(async (client) => {
      // Get next version number
      const versionResult = await client.query(
        `SELECT COALESCE(MAX(version), 0) + 1 as next_version
         FROM config_versions WHERE project_id = $1`,
        [projectId]
      );
      const version = versionResult.rows[0].next_version;
      
      // Deactivate current active
      await client.query(
        `UPDATE config_versions SET is_active = false WHERE project_id = $1`,
        [projectId]
      );
      
      // Insert new version
      const insertResult = await client.query(
        `INSERT INTO config_versions (project_id, version, yaml_content, is_active)
         VALUES ($1, $2, $3, true)
         RETURNING id, version, created_at`,
        [projectId, version, yaml]
      );
      
      return insertResult.rows[0];
    });
    
    // Get summary for response
    const normalized = normalizeConfig(validation.config!);
    const summary = printSummary(normalized);
    
    logger.info('Config uploaded', { 
      tenantId: authReq.tenant.id,
      projectSlug: slug,
      version: result.version,
    });
    
    return res.status(201).json({
      config: {
        id: result.id,
        version: result.version,
        summary,
        createdAt: result.created_at,
      },
    });
  } catch (err) {
    logger.error('Upload config error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * GET /projects/:slug/config
 * Get active config
 */
router.get('/:slug/config', async (req: Request, res: Response) => {
  const authReq = req as AuthenticatedRequest;
  const { slug } = req.params;
  
  try {
    const result = await query(
      `SELECT cv.id, cv.version, cv.yaml_content, cv.created_at
       FROM config_versions cv
       JOIN projects p ON p.id = cv.project_id
       WHERE p.tenant_id = $1 AND p.slug = $2 AND cv.is_active = true`,
      [authReq.tenant.id, slug]
    );
    
    if (result.rowCount === 0) {
      return res.status(404).json({ error: 'No active config found' });
    }
    
    const row = result.rows[0];
    
    return res.json({
      config: {
        id: row.id,
        version: row.version,
        yaml: row.yaml_content,
        createdAt: row.created_at,
      },
    });
  } catch (err) {
    logger.error('Get config error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * POST /projects/:slug/config/validate
 * Validate config without saving
 */
router.post('/:slug/config/validate', async (req: Request, res: Response) => {
  try {
    const { yaml } = req.body;
    
    if (!yaml || typeof yaml !== 'string') {
      return res.status(400).json({ error: 'YAML config is required' });
    }
    
    // Schema validation
    const validation = validateConfig(yaml);
    
    if (!validation.valid) {
      return res.json({ 
        valid: false,
        errors: validation.errors,
      });
    }
    
    // Cloud policy validation
    const normalizedForPolicy = normalizeConfig(validation.config!);
    const cloudValidation = validateCloudPolicy(normalizedForPolicy);
    if (!cloudValidation.valid) {
      return res.json({
        valid: false,
        errors: cloudValidation.errors,
      });
    }
    
    // Get summary
    const normalized = normalizeConfig(validation.config!);
    const summary = printSummary(normalized);
    
    return res.json({
      valid: true,
      summary,
    });
  } catch (err) {
    logger.error('Validate config error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;


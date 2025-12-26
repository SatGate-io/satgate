/**
 * SatGate Cloud Data Plane
 * 
 * Multi-tenant gateway that enforces L402 policies.
 */

import express from 'express';
import { logger } from '@satgate/common';
import { resolveSlug } from './tenant/resolveSlug';
import { getConfig, getProjectId, invalidateConfig } from './tenant/configCache';
import { gatewayMiddleware } from './gateway/middleware';
import { healthCheck as dbHealthCheck } from './db';
import { shutdown as shutdownUsage } from './usage/events';

const app = express();
const PORT = process.env.PORT || 8080;

// Internal auth token for control plane webhooks
const INTERNAL_AUTH_TOKEN = process.env.INTERNAL_AUTH_TOKEN;

// Fail-fast: require INTERNAL_AUTH_TOKEN in production
if (process.env.NODE_ENV === 'production' && !INTERNAL_AUTH_TOKEN) {
  logger.error('INTERNAL_AUTH_TOKEN is required in production');
  process.exit(1);
}

// Health check (not tenant-specific)
app.get('/healthz', async (req, res) => {
  const dbOk = await dbHealthCheck();
  
  if (!dbOk) {
    return res.status(503).json({
      status: 'unhealthy',
      plane: 'data',
      db: false,
    });
  }
  
  res.json({
    status: 'ok',
    plane: 'data',
    db: true,
    timestamp: new Date().toISOString(),
  });
});

// Config invalidation webhook (from control plane)
app.post('/_internal/invalidate/:slug', express.json(), async (req, res) => {
  // Verify internal auth (required if INTERNAL_AUTH_TOKEN is set)
  const authHeader = req.headers.authorization;
  
  if (INTERNAL_AUTH_TOKEN && authHeader !== `Bearer ${INTERNAL_AUTH_TOKEN}`) {
    return res.status(401).json({ error: 'Unauthorized' });
  }
  
  const { slug } = req.params;
  invalidateConfig(slug);
  
  logger.info('Config invalidated via webhook', { slug });
  res.json({ ok: true, slug });
});

// All other requests go through tenant resolution
app.use(async (req, res, next) => {
  const host = req.headers.host || '';
  const slug = resolveSlug(host);
  
  if (!slug) {
    return res.status(403).json({
      error: 'Forbidden',
      message: 'Unknown tenant',
    });
  }
  
  try {
    const config = await getConfig(slug);
    const projectId = getProjectId(slug);
    
    // Attach tenant context to request
    (req as any).tenantSlug = slug;
    (req as any).gatewayConfig = config;
    (req as any).projectId = projectId;
    
    next();
  } catch (err) {
    logger.error('Failed to load tenant config', { slug, error: (err as Error).message });
    return res.status(404).json({
      error: 'Not Found',
      message: 'Project not found or inactive',
    });
  }
});

// Gateway middleware - route matching, L402 enforcement, proxy
app.use(gatewayMiddleware);

// Error handler
app.use((err: Error, req: express.Request, res: express.Response, next: express.NextFunction) => {
  const tenantSlug = (req as any).tenantSlug || 'unknown';
  logger.error('Unhandled error', { tenant: tenantSlug, error: err.message });
  res.status(500).json({ error: 'Internal server error' });
});

// Graceful shutdown
async function shutdown(signal: string) {
  logger.info(`Received ${signal}, shutting down gracefully...`);
  
  // Flush any pending usage events
  await shutdownUsage();
  
  process.exit(0);
}

process.on('SIGTERM', () => shutdown('SIGTERM'));
process.on('SIGINT', () => shutdown('SIGINT'));

// Start server
app.listen(PORT, () => {
  logger.info(`Data plane listening on port ${PORT}`);
});

/**
 * SatGate Cloud Data Plane
 * 
 * Multi-tenant gateway that enforces L402 policies.
 */

import express from 'express';
import { logger } from '@satgate/common';
import { resolveSlug } from './tenant/resolveSlug';
import { getConfig, invalidateConfig } from './tenant/configCache';

const app = express();
const PORT = process.env.PORT || 8080;

// Health check (not tenant-specific)
app.get('/healthz', (req, res) => {
  res.json({ status: 'ok', plane: 'data', timestamp: new Date().toISOString() });
});

// Config invalidation webhook (from control plane)
app.post('/_internal/invalidate/:slug', async (req, res) => {
  const { slug } = req.params;
  
  // TODO: Verify internal auth
  invalidateConfig(slug);
  
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
    
    // Attach tenant context to request
    (req as any).tenantSlug = slug;
    (req as any).gatewayConfig = config;
    
    next();
  } catch (err) {
    logger.error('Failed to load tenant config', { slug, error: (err as Error).message });
    return res.status(404).json({
      error: 'Not Found',
      message: 'Project not found or inactive',
    });
  }
});

// Gateway middleware (TODO: integrate with existing gateway engine)
app.use((req, res) => {
  const slug = (req as any).tenantSlug;
  const config = (req as any).gatewayConfig;
  
  // TODO: Route matching + L402 enforcement + proxy
  res.status(501).json({
    error: 'Not implemented',
    message: 'Gateway middleware not yet connected',
    tenant: slug,
    routes: config?.routes?.length || 0,
  });
});

// Error handler
app.use((err: Error, req: express.Request, res: express.Response, next: express.NextFunction) => {
  logger.error('Unhandled error', { error: err.message });
  res.status(500).json({ error: 'Internal server error' });
});

// Start server
app.listen(PORT, () => {
  logger.info(`Data plane listening on port ${PORT}`);
});


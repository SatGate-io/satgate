/**
 * SatGate Cloud Control Plane
 * 
 * API for managing projects, configs, secrets, and usage.
 */

import express from 'express';
import { logger } from '@satgate/common';

const app = express();
const PORT = process.env.PORT || 3000;

// Middleware
app.use(express.json({ limit: '1mb' }));

// Health check
app.get('/healthz', (req, res) => {
  res.json({ status: 'ok', plane: 'control', timestamp: new Date().toISOString() });
});

// Auth routes
app.post('/auth/magic-link', async (req, res) => {
  // TODO: Implement magic link
  res.status(501).json({ error: 'Not implemented' });
});

app.post('/auth/verify', async (req, res) => {
  // TODO: Implement verify
  res.status(501).json({ error: 'Not implemented' });
});

app.post('/auth/logout', async (req, res) => {
  // TODO: Implement logout
  res.status(501).json({ error: 'Not implemented' });
});

// Project routes
app.get('/api/projects', async (req, res) => {
  // TODO: List projects
  res.status(501).json({ error: 'Not implemented' });
});

app.post('/api/projects', async (req, res) => {
  // TODO: Create project
  res.status(501).json({ error: 'Not implemented' });
});

app.get('/api/projects/:slug', async (req, res) => {
  // TODO: Get project
  res.status(501).json({ error: 'Not implemented' });
});

app.delete('/api/projects/:slug', async (req, res) => {
  // TODO: Delete project
  res.status(501).json({ error: 'Not implemented' });
});

// Config routes
app.post('/api/projects/:slug/config', async (req, res) => {
  // TODO: Upload config
  res.status(501).json({ error: 'Not implemented' });
});

app.get('/api/projects/:slug/config', async (req, res) => {
  // TODO: Get active config
  res.status(501).json({ error: 'Not implemented' });
});

// Usage routes
app.get('/api/projects/:slug/usage', async (req, res) => {
  // TODO: Get usage stats
  res.status(501).json({ error: 'Not implemented' });
});

app.get('/api/projects/:slug/events', async (req, res) => {
  // TODO: Get recent events
  res.status(501).json({ error: 'Not implemented' });
});

// Test route (for onboarding)
app.post('/api/projects/:slug/test', async (req, res) => {
  // TODO: Test a route
  res.status(501).json({ error: 'Not implemented' });
});

// Error handler
app.use((err: Error, req: express.Request, res: express.Response, next: express.NextFunction) => {
  logger.error('Unhandled error', { error: err.message, stack: err.stack });
  res.status(500).json({ error: 'Internal server error' });
});

// Start server
app.listen(PORT, () => {
  logger.info(`Control plane listening on port ${PORT}`);
});


/**
 * SatGate Cloud - Control Plane
 * 
 * Handles:
 * - Authentication (magic link)
 * - Project CRUD
 * - Config management
 * - API key management
 */

import express from 'express';
import cookieParser from 'cookie-parser';
import cors from 'cors';
import routes from './routes';
import { healthCheck as dbHealthCheck } from './db';
import { logger } from '@satgate/common';

const app = express();
const PORT = process.env.PORT || 3000;

// CORS configuration
const corsOrigins = process.env.CORS_ORIGINS?.split(',') || [
  'https://cloud.satgate.io',
  'http://localhost:3001',
];

app.use(cors({
  origin: corsOrigins,
  credentials: true,
}));

app.use(express.json({ limit: '100kb' }));
app.use(cookieParser());

// Health check
app.get('/healthz', async (req, res) => {
  const dbOk = await dbHealthCheck();
  
  if (!dbOk) {
    return res.status(503).json({ 
      status: 'unhealthy', 
      service: 'control-plane',
      db: false,
    });
  }
  
  res.json({ 
    status: 'ok', 
    service: 'control-plane',
    db: true,
  });
});

// API routes
app.use('/api', routes);

// Error handler
app.use((err: any, req: express.Request, res: express.Response, next: express.NextFunction) => {
  logger.error('Unhandled error', { 
    error: err.message, 
    stack: err.stack,
    path: req.path,
  });
  
  res.status(500).json({ error: 'Internal server error' });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({ error: 'Not found' });
});

app.listen(PORT, () => {
  logger.info(`[Control Plane] Listening on port ${PORT}`);
});

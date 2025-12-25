/**
 * SatGate Gateway Mode Entrypoint
 * 
 * This module starts SatGate in Gateway Mode - a full reverse proxy
 * with L402 enforcement and two-listener architecture.
 * 
 * Called from server.js when SATGATE_RUNTIME=gateway or auto-detected.
 */

const path = require('path');

async function startGatewayMode() {
  const {
    initGateway,
    getConfigPath,
    loadConfig,
  } = require('./gateway');
  
  const { L402Service } = require('./l402');
  
  const configPath = getConfigPath();
  console.log(`[Gateway] Loading configuration from: ${configPath}`);
  
  // Pre-validate config before initializing services
  let config;
  try {
    config = loadConfig(path.resolve(configPath));
  } catch (e) {
    console.error(`[Gateway] Configuration error: ${e.message}`);
    process.exit(1);
  }
  
  // Initialize L402 service
  let l402Service = null;
  if (config.l402.mode === 'native') {
    try {
      const lightningConfig = {
        backend: process.env.LIGHTNING_BACKEND || 'mock',
        url: process.env.LIGHTNING_URL || process.env.PHOENIXD_URL || process.env.LND_REST_URL,
        password: process.env.PHOENIXD_PASSWORD,
        macaroon: process.env.LND_MACAROON,
        apiKey: process.env.OPENNODE_API_KEY,
      };
      
      l402Service = new L402Service({
        rootKey: config.l402.rootKey,
        lightningConfig,
        redis: null, // TODO: Pass Redis client if available
        defaultTTL: config.l402.defaultTTLSeconds,
        defaultMaxCalls: config.l402.defaultMaxCalls,
        defaultBudgetSats: config.l402.defaultBudgetSats,
      });
      
      console.log(`[Gateway] L402 service initialized (Lightning: ${lightningConfig.backend})`);
    } catch (e) {
      console.error(`[Gateway] Failed to initialize L402 service: ${e.message}`);
      process.exit(1);
    }
  } else {
    console.log('[Gateway] L402 mode: aperture (external sidecar)');
  }
  
  // Initialize metering service (optional)
  let meteringService = null;
  if (config.metering.backend === 'redis' && config.metering.redisUrl) {
    try {
      const Redis = require('ioredis');
      const redis = new Redis(config.metering.redisUrl, {
        maxRetriesPerRequest: 3,
        lazyConnect: true,
      });
      await redis.connect();
      console.log('[Gateway] Redis metering connected');
      
      // Simple metering service wrapper
      meteringService = {
        async check(tokenId, limits) {
          const callKey = `sg:calls:${tokenId.substring(0, 32)}`;
          const now = Date.now();
          const ttl = 3600; // 1 hour default
          
          // Atomic check-and-decrement for calls
          if (limits.maxCalls) {
            const script = `
              local key = KEYS[1]
              local max = tonumber(ARGV[1])
              local ttl = tonumber(ARGV[2])
              if redis.call('EXISTS', key) == 0 then
                redis.call('SET', key, max, 'EX', ttl)
              end
              local remaining = redis.call('DECR', key)
              return remaining
            `;
            const remaining = await redis.eval(script, 1, callKey, limits.maxCalls, ttl);
            if (remaining < 0) {
              return { exhausted: true, reason: 'calls', callsRemaining: 0 };
            }
            return { exhausted: false, callsRemaining: Math.max(0, remaining) };
          }
          
          return { exhausted: false };
        },
      };
    } catch (e) {
      console.warn(`[Gateway] Redis connection failed, using in-memory metering: ${e.message}`);
    }
  }
  
  // Fallback to in-memory metering
  if (!meteringService) {
    const callCounts = new Map();
    meteringService = {
      async check(tokenId, limits) {
        if (!limits.maxCalls) return { exhausted: false };
        
        const key = tokenId.substring(0, 32);
        let count = callCounts.get(key) || limits.maxCalls;
        count -= 1;
        callCounts.set(key, count);
        
        if (count < 0) {
          return { exhausted: true, reason: 'calls', callsRemaining: 0 };
        }
        return { exhausted: false, callsRemaining: Math.max(0, count) };
      },
    };
    console.log('[Gateway] Using in-memory metering (single-instance only)');
  }
  
  // Create governance routes for admin plane (simplified)
  const express = require('express');
  const governanceRouter = express.Router();
  
  // Minimal governance endpoints for gateway mode
  governanceRouter.get('/stats', (req, res) => {
    res.json({
      ok: true,
      mode: 'gateway',
      upstreams: Object.keys(config.upstreams).length,
      routes: config.routes.length,
      timestamp: new Date().toISOString(),
    });
  });
  
  governanceRouter.get('/info', (req, res) => {
    res.json({
      ok: true,
      version: '2.0.0',
      mode: 'gateway',
      l402Mode: config.l402.mode,
      meteringBackend: config.metering.backend,
      timestamp: new Date().toISOString(),
    });
  });
  
  // Start gateway servers
  try {
    const result = await initGateway(
      path.resolve(configPath),
      l402Service,
      meteringService,
      governanceRouter
    );
    
    console.log('[Gateway] ═══════════════════════════════════════════════');
    console.log('[Gateway] SatGate Gateway Mode v2.0.0 started');
    console.log(`[Gateway]   Data plane:  ${config.server.listen}`);
    console.log(`[Gateway]   Admin plane: ${config.admin.listen}`);
    console.log(`[Gateway]   L402 mode:   ${config.l402.mode}`);
    console.log(`[Gateway]   Metering:    ${config.metering.backend}`);
    console.log(`[Gateway]   Routes:      ${config.routes.length}`);
    console.log('[Gateway] ═══════════════════════════════════════════════');
    
    // Graceful shutdown
    const shutdown = (signal) => {
      console.log(`\n[Gateway] Received ${signal}, shutting down...`);
      if (result.dataServer) result.dataServer.close();
      if (result.adminServer) result.adminServer.close();
      process.exit(0);
    };
    
    process.on('SIGTERM', () => shutdown('SIGTERM'));
    process.on('SIGINT', () => shutdown('SIGINT'));
    
  } catch (e) {
    console.error(`[Gateway] Failed to start: ${e.message}`);
    process.exit(1);
  }
}

module.exports = { startGatewayMode };


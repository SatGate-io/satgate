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
        /**
         * Meter a token in a time window.
         *
         * @param {string} tokenId - stable token id (we use macaroon signature)
         * @param {object} limits
         * @param {number} [limits.maxCalls]
         * @param {number} [limits.budgetSats]
         * @param {number} [limits.costSats] - cost to decrement from budget per request (default 1)
         * @param {number} [limits.expiresAtMs] - token expiry time (ms since epoch) for TTL alignment
         */
        async check(tokenId, limits) {
          const id = String(tokenId || '');
          const keyPrefix = id.substring(0, 32);
          const now = Date.now();
          const expiresAtMs = Number.isFinite(limits?.expiresAtMs) ? limits.expiresAtMs : null;
          const ttlSeconds = expiresAtMs ? Math.max(1, Math.ceil((expiresAtMs - now) / 1000)) : (config.l402.defaultTTLSeconds || 3600);
          const costSats = Number.isFinite(limits?.costSats) && limits.costSats > 0 ? limits.costSats : 1;

          const result = { exhausted: false };

          // Atomic init+decr for calls
          if (Number.isFinite(limits?.maxCalls) && limits.maxCalls > 0) {
            const callKey = `sg:calls:${keyPrefix}`;
            const callScript = `
              local key = KEYS[1]
              local init = tonumber(ARGV[1])
              local ttl = tonumber(ARGV[2])
              if redis.call('EXISTS', key) == 0 then
                redis.call('SET', key, init, 'EX', ttl)
              end
              local remaining = redis.call('DECR', key)
              return remaining
            `;
            const remaining = await redis.eval(callScript, 1, callKey, String(limits.maxCalls), String(ttlSeconds));
            const rem = Number(remaining);
            result.callsRemaining = Math.max(0, rem);
            if (rem < 0) return { exhausted: true, reason: 'calls', callsRemaining: 0 };
          }

          // Atomic init+conditional decr for budget
          if (Number.isFinite(limits?.budgetSats) && limits.budgetSats > 0) {
            const budgetKey = `sg:budget:${keyPrefix}`;
            const budgetScript = `
              local key = KEYS[1]
              local init = tonumber(ARGV[1])
              local cost = tonumber(ARGV[2])
              local ttl = tonumber(ARGV[3])
              if redis.call('EXISTS', key) == 0 then
                redis.call('SET', key, init, 'EX', ttl)
              end
              local current = tonumber(redis.call('GET', key))
              if current == nil then return { -999999, 0 } end
              if current < cost then return { current, 0 } end
              local remaining = redis.call('DECRBY', key, cost)
              return { remaining, 1 }
            `;
            const arr = await redis.eval(budgetScript, 1, budgetKey, String(limits.budgetSats), String(costSats), String(ttlSeconds));
            const remainingBudget = Array.isArray(arr) ? Number(arr[0]) : Number(arr);
            const charged = Array.isArray(arr) ? Number(arr[1]) === 1 : true;
            result.budgetRemaining = Math.max(0, remainingBudget);
            if (!charged) return { exhausted: true, reason: 'budget', budgetRemaining: Math.max(0, remainingBudget) };
          }

          return result;
        },
      };
    } catch (e) {
      console.warn(`[Gateway] Redis connection failed, using in-memory metering: ${e.message}`);
    }
  }
  
  // Fallback to in-memory metering
  if (!meteringService) {
    const calls = new Map();   // keyPrefix -> { remaining, expiresAtMs }
    const budget = new Map();  // keyPrefix -> { remaining, expiresAtMs }
    meteringService = {
      async check(tokenId, limits) {
        const id = String(tokenId || '');
        const key = id.substring(0, 32);
        const now = Date.now();
        const expiresAtMs = Number.isFinite(limits?.expiresAtMs) ? limits.expiresAtMs : null;
        const ttlSeconds = expiresAtMs ? Math.max(1, Math.ceil((expiresAtMs - now) / 1000)) : (config.l402.defaultTTLSeconds || 3600);
        const exp = expiresAtMs || (now + ttlSeconds * 1000);
        const costSats = Number.isFinite(limits?.costSats) && limits.costSats > 0 ? limits.costSats : 1;

        // Calls
        if (Number.isFinite(limits?.maxCalls) && limits.maxCalls > 0) {
          const entry = calls.get(key);
          if (!entry || entry.expiresAtMs < now) calls.set(key, { remaining: limits.maxCalls, expiresAtMs: exp });
          const cur = calls.get(key);
          cur.remaining -= 1;
          if (cur.remaining < 0) return { exhausted: true, reason: 'calls', callsRemaining: 0 };
          // fall through: still may have budget
        }

        // Budget
        if (Number.isFinite(limits?.budgetSats) && limits.budgetSats > 0) {
          const entry = budget.get(key);
          if (!entry || entry.expiresAtMs < now) budget.set(key, { remaining: limits.budgetSats, expiresAtMs: exp });
          const cur = budget.get(key);
          if (cur.remaining < costSats) return { exhausted: true, reason: 'budget', budgetRemaining: Math.max(0, cur.remaining) };
          cur.remaining -= costSats;
        }

        const out = { exhausted: false };
        const c = calls.get(key);
        const b = budget.get(key);
        if (c) out.callsRemaining = Math.max(0, c.remaining);
        if (b) out.budgetRemaining = Math.max(0, b.remaining);
        return out;
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


/**
 * SatGate Proxy Mode
 * 
 * Reverse proxy with L402 enforcement for protecting any upstream API.
 * 
 * Features:
 * - Static upstream selection (SSRF prevention)
 * - L402 payment challenges
 * - Capability token validation
 * - Per-request metering (max_calls, budget_sats)
 * - Streaming request/response
 * - Admin/data plane separation
 * 
 * Runtime Modes (SATGATE_RUNTIME):
 * - "gateway": Full reverse proxy mode with two listeners
 * - "embedded": Original demo mode (single server, no proxy)
 * - undefined/default: Auto-detect based on config file presence
 * 
 * Usage:
 *   SATGATE_RUNTIME=gateway node server.js
 *   SATGATE_GATEWAY_CONFIG=./satgate.gateway.yaml
 */

const fs = require('fs');
const { loadConfig, matchRoute } = require('./config/loader');

/**
 * Gateway runtime state
 */
let config = null;
let servers = null;

/**
 * Initialize Gateway Mode
 * 
 * @param {string} configPath - Path to gateway config file
 * @param {object} l402Service - L402 service instance
 * @param {object} meteringService - Metering service (optional)
 * @param {object} governanceRoutes - Governance router (optional)
 * @returns {object} { config, dataServer, adminServer }
 */
async function initGateway(configPath, l402Service, meteringService, governanceRoutes) {
  console.log('[Gateway] Initializing...');
  console.log(`[Gateway] Loading config from: ${configPath}`);
  
  // Load and validate config (fail-fast)
  config = loadConfig(configPath);
  
  console.log(`[Gateway] Config loaded successfully`);
  console.log(`[Gateway]   Upstreams: ${Object.keys(config.upstreams).join(', ')}`);
  console.log(`[Gateway]   Routes: ${config.routes.map(r => r.name).join(', ')}`);
  console.log(`[Gateway]   L402 mode: ${config.l402.mode}`);
  console.log(`[Gateway]   Metering: ${config.metering.backend}`);
  
  // Start servers
  const { startGatewayServers } = require('./servers');
  servers = startGatewayServers(config, l402Service, meteringService, governanceRoutes);
  
  return { config, ...servers };
}

/**
 * Get the loaded configuration
 */
function getConfig() {
  if (!config) {
    throw new Error('Gateway not initialized. Call initGateway() first.');
  }
  return config;
}

/**
 * Check if gateway mode is enabled
 */
function isGatewayMode() {
  const runtime = process.env.SATGATE_RUNTIME;
  
  if (runtime === 'gateway') return true;
  if (runtime === 'embedded' || runtime === 'demo') return false;
  
  // Auto-detect: check for config file
  const configPath = getConfigPath();
  if (fs.existsSync(configPath)) {
    console.log(`[Gateway] Auto-detected gateway config at ${configPath}`);
    return true;
  }
  
  return false;
}

/**
 * Get config file path from environment
 */
function getConfigPath() {
  return process.env.SATGATE_GATEWAY_CONFIG || './satgate.gateway.yaml';
}

/**
 * Shutdown gateway servers
 */
function shutdownGateway() {
  if (servers) {
    if (servers.dataServer) servers.dataServer.close();
    if (servers.adminServer) servers.adminServer.close();
    servers = null;
  }
  config = null;
}

// Re-export submodules
module.exports = {
  // Core
  initGateway,
  getConfig,
  isGatewayMode,
  getConfigPath,
  shutdownGateway,
  
  // Config
  loadConfig,
  matchRoute,
  
  // Submodules
  get router() { return require('./router'); },
  get limits() { return require('./limits'); },
  get transport() { return require('./transport'); },
  get authDecide() { return require('./auth-decide'); },
  get servers() { return require('./servers'); },
};


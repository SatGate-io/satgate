#!/usr/bin/env node
/**
 * SatGate Config Validator
 * 
 * Validates a gateway configuration file against the schema and prints a summary.
 * 
 * Usage:
 *   node cli/validate-gateway-config.js <config-path>
 *   node cli/validate-gateway-config.js satgate.gateway.yaml
 *   node cli/validate-gateway-config.js --help
 * 
 * Exit codes:
 *   0 - Config valid
 *   1 - Config invalid or error
 */

const fs = require('fs');
const path = require('path');

// Set demo mode for validation (avoids prod-only checks)
process.env.MODE = process.env.MODE || 'demo';

// Help text
function printHelp() {
  console.log(`
SatGate Config Validator

Usage:
  node cli/validate-gateway-config.js <config-path>

Examples:
  node cli/validate-gateway-config.js satgate.gateway.yaml
  node cli/validate-gateway-config.js satgate.gateway.smb.yaml
  node cli/validate-gateway-config.js ./my-config.yaml

Options:
  --help, -h    Show this help message
  --verbose     Show detailed route information

Exit codes:
  0  Config valid
  1  Config invalid or error
`);
}

// Parse arguments
const args = process.argv.slice(2);

if (args.includes('--help') || args.includes('-h') || args.length === 0) {
  printHelp();
  process.exit(args.length === 0 ? 1 : 0);
}

const verbose = args.includes('--verbose');
const configPath = args.find(a => !a.startsWith('--'));

if (!configPath) {
  console.error('Error: No config path provided');
  printHelp();
  process.exit(1);
}

// Resolve path
const resolvedPath = path.resolve(configPath);

if (!fs.existsSync(resolvedPath)) {
  console.error(`✗ Config file not found: ${resolvedPath}`);
  process.exit(1);
}

// Try to load the gateway config loader
let loadConfig;
try {
  // Try relative to cli/ directory
  const loaderPath = path.join(__dirname, '..', 'proxy', 'gateway', 'config', 'loader.js');
  if (fs.existsSync(loaderPath)) {
    loadConfig = require(loaderPath).loadConfig;
  } else {
    // Try from project root
    loadConfig = require('../proxy/gateway/config/loader').loadConfig;
  }
} catch (err) {
  console.error('✗ Could not load gateway config loader');
  console.error('  Make sure you are running from the project root');
  console.error(`  Error: ${err.message}`);
  process.exit(1);
}

// Validate the config
console.log(`\nValidating: ${configPath}\n`);

try {
  const config = loadConfig(resolvedPath);
  
  // Success - print summary
  console.log('✓ Config valid\n');
  
  // Upstreams
  const upstreamNames = Object.keys(config.upstreams || {});
  console.log(`  Upstreams: ${upstreamNames.join(', ') || '(none)'}`);
  
  if (verbose && upstreamNames.length > 0) {
    for (const name of upstreamNames) {
      const upstream = config.upstreams[name];
      console.log(`    • ${name}: ${upstream.url}`);
    }
  }
  
  // Routes
  const routes = config.routes || [];
  const routeNames = routes.map(r => r.name);
  console.log(`  Routes: ${routeNames.join(', ') || '(none)'}`);
  
  if (verbose && routes.length > 0) {
    console.log('\n  Route details:');
    for (const route of routes) {
      const policy = route.policy || {};
      const match = route.match || {};
      const pathMatch = match.exactPath || match.pathPrefix || '*';
      const methods = match.methods ? match.methods.join(',') : 'ALL';
      
      let policyStr = policy.kind || 'unknown';
      if (policy.kind === 'l402') {
        policyStr = `l402 (${policy.priceSats} sats, tier: ${policy.tier})`;
      } else if (policy.kind === 'deny') {
        policyStr = `deny (${policy.status || 403})`;
      } else if (policy.kind === 'capability') {
        policyStr = `capability (scope: ${policy.scope})`;
      }
      
      console.log(`    • ${route.name}`);
      console.log(`      Path: ${pathMatch} [${methods}]`);
      console.log(`      Policy: ${policyStr}`);
      if (route.upstream) {
        console.log(`      Upstream: ${route.upstream}`);
      }
    }
  }
  
  // L402 settings
  const l402 = config.l402 || {};
  console.log(`  L402 mode: ${l402.mode || 'native'}`);
  
  // Metering
  const metering = config.metering || {};
  console.log(`  Metering: ${metering.backend || 'memory'}`);
  
  // Server settings
  const server = config.server || {};
  const admin = config.admin || {};
  console.log(`  Data plane: ${server.listen || '0.0.0.0:8080'}`);
  console.log(`  Admin plane: ${admin.listen || '127.0.0.1:9090'}`);
  
  // Warnings
  console.log('');
  
  // Check for common issues
  const warnings = [];
  
  // No default-deny route (check various ways it could be configured)
  const hasDefaultDeny = routes.some(r => {
    if (r.policy?.kind !== 'deny') return false;
    const match = r.match || {};
    // Check for catch-all patterns
    return match.pathPrefix === '/' || 
           match.exactPath === '/' ||
           r.name?.toLowerCase().includes('deny') ||
           r.name?.toLowerCase().includes('default');
  });
  if (!hasDefaultDeny) {
    warnings.push('No default-deny catch-all route (recommended for fail-closed security)');
  }
  
  // L402 routes without upstream
  const l402RoutesWithoutUpstream = routes.filter(r => 
    r.policy?.kind === 'l402' && !r.upstream
  );
  if (l402RoutesWithoutUpstream.length > 0) {
    warnings.push(`L402 routes without upstream: ${l402RoutesWithoutUpstream.map(r => r.name).join(', ')}`);
  }
  
  // Admin plane exposed publicly
  if (admin.listen && !admin.listen.startsWith('127.0.0.1') && !admin.listen.startsWith('localhost')) {
    warnings.push('Admin plane is not bound to localhost (security risk in production)');
  }
  
  if (warnings.length > 0) {
    console.log('⚠ Warnings:');
    for (const warning of warnings) {
      console.log(`  • ${warning}`);
    }
    console.log('');
  }
  
  console.log('Ready to deploy!\n');
  process.exit(0);
  
} catch (err) {
  console.error(`✗ Config invalid\n`);
  console.error(`  ${err.message}`);
  
  // Try to provide helpful hints
  if (err.message.includes('required')) {
    console.error('\n  Hint: Check that all required fields are present (version, routes, etc.)');
  }
  if (err.message.includes('upstreams')) {
    console.error('\n  Hint: Ensure all routes reference defined upstreams');
  }
  if (err.message.includes('l402') || err.message.includes('tier') || err.message.includes('scope')) {
    console.error('\n  Hint: L402 policies require: tier, priceSats, and scope');
  }
  
  console.error('');
  process.exit(1);
}


#!/usr/bin/env node
/**
 * SatGate Phase 1: Capability Token Minting CLI
 * 
 * Usage:
 *   node cli/mint-token.js [options]
 * 
 * Options:
 *   --scope <scope>       Token scope (default: api:capability:read)
 *   --expires <seconds>   Token TTL in seconds (default: 3600)
 *   --max-calls <n>       Max calls allowed (default: unlimited)
 *   --budget-sats <n>     Max sats budget (default: unlimited)
 *   --key <key>           Root key (default: from CAPABILITY_ROOT_KEY env)
 * 
 * Example:
 *   node cli/mint-token.js --scope api:capability:read --expires 3600
 */

const macaroon = require('macaroon');

// Configuration
const CAPABILITY_ROOT_KEY = process.env.CAPABILITY_ROOT_KEY || 'satgate-phase1-demo-key-change-in-prod';
const CAPABILITY_LOCATION = 'https://satgate.io';
const CAPABILITY_IDENTIFIER = 'satgate-capability-v1';

// Tier costs (must match server)
const TIER_COSTS = {
  'api:capability:read': 1,
  'api:capability:ping': 1,
  'api:capability:data': 5,
  'api:capability:admin': 10,
  'api:capability:*': 1,
  'default': 1
};

// Parse CLI arguments
function parseArgs() {
  const args = process.argv.slice(2);
  const options = {
    scope: 'api:capability:read',
    expires: 3600,
    maxCalls: null,
    budgetSats: null,
    key: CAPABILITY_ROOT_KEY
  };
  
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--scope' && args[i + 1]) {
      options.scope = args[++i];
    } else if (args[i] === '--expires' && args[i + 1]) {
      options.expires = parseInt(args[++i], 10);
    } else if ((args[i] === '--max-calls' || args[i] === '--max_calls') && args[i + 1]) {
      options.maxCalls = parseInt(args[++i], 10);
    } else if ((args[i] === '--budget-sats' || args[i] === '--budget_sats') && args[i + 1]) {
      options.budgetSats = parseInt(args[++i], 10);
    } else if (args[i] === '--key' && args[i + 1]) {
      options.key = args[++i];
    } else if (args[i] === '--help' || args[i] === '-h') {
      console.log(`
SatGate Phase 1: Capability Token Minting CLI

Usage:
  node cli/mint-token.js [options]

Options:
  --scope <scope>       Token scope (default: api:capability:read)
  --expires <seconds>   Token TTL in seconds (default: 3600)
  --max-calls <n>       Max calls allowed (default: unlimited)
  --budget-sats <n>     Max sats budget (default: unlimited)
  --key <key>           Root key (default: from CAPABILITY_ROOT_KEY env)
  --help, -h            Show this help

Tier Costs (sats per request):
  api:capability:read   1 sat
  api:capability:ping   1 sat
  api:capability:data   5 sats
  api:capability:admin  10 sats

Examples:
  # Mint a token valid for 1 hour
  node cli/mint-token.js

  # Mint a token valid for 5 minutes with custom scope
  node cli/mint-token.js --scope api:capability:admin --expires 300

  # Mint a token that can be used 3 times total
  node cli/mint-token.js --max-calls 3

  # Mint a token with 100 sats budget (100 ping requests or 20 data requests)
  node cli/mint-token.js --budget-sats 100

  # Mint using a custom key
  CAPABILITY_ROOT_KEY=my-secret node cli/mint-token.js
      `);
      process.exit(0);
    }
  }
  
  return options;
}

function mintToken(options) {
  const { scope, expires, maxCalls, budgetSats, key } = options;
  
  // Validate max_calls
  if (maxCalls !== null && (!Number.isFinite(maxCalls) || maxCalls <= 0)) {
    console.error('❌ Invalid --max-calls. Must be a positive integer.');
    process.exit(1);
  }
  
  // Validate budget_sats
  if (budgetSats !== null && (!Number.isFinite(budgetSats) || budgetSats <= 0)) {
    console.error('❌ Invalid --budget-sats. Must be a positive integer.');
    process.exit(1);
  }
  
  console.log('\n[SYSTEM] Generating Phase 1 Capability Token...');
  console.log(`[CONFIG] Scope: ${scope}`);
  console.log(`[CONFIG] TTL: ${expires} seconds`);
  if (maxCalls) console.log(`[CONFIG] Max Calls: ${maxCalls}`);
  if (budgetSats) {
    const tierCost = TIER_COSTS[scope] || TIER_COSTS['default'];
    const approxCalls = Math.floor(budgetSats / tierCost);
    console.log(`[CONFIG] Budget: ${budgetSats} sats (~${approxCalls} requests at ${tierCost} sat/req)`);
  }
  console.log(`[NETWORK] Requests sent: 0  ← Offline operation\n`);
  
  try {
    // Create base macaroon
    const keyBytes = Buffer.from(key, 'utf8');
    const identifier = `${CAPABILITY_IDENTIFIER}:${Date.now()}`;
    
    let m = macaroon.newMacaroon({
      identifier: Buffer.from(identifier, 'utf8'),
      location: CAPABILITY_LOCATION,
      rootKey: keyBytes
    });
    
    // Add caveats (method on macaroon object)
    const expiresAt = Date.now() + (expires * 1000);
    m.addFirstPartyCaveat(Buffer.from(`expires = ${expiresAt}`, 'utf8'));
    m.addFirstPartyCaveat(Buffer.from(`scope = ${scope}`, 'utf8'));
    if (maxCalls) {
      m.addFirstPartyCaveat(Buffer.from(`max_calls = ${Math.floor(maxCalls)}`, 'utf8'));
    }
    if (budgetSats) {
      m.addFirstPartyCaveat(Buffer.from(`budget_sats = ${Math.floor(budgetSats)}`, 'utf8'));
    }
    
    // Export as base64
    const tokenBytes = m.exportBinary();
    const tokenBase64 = Buffer.from(tokenBytes).toString('base64');
    
    console.log('✅ Token minted successfully!\n');
    console.log('═══════════════════════════════════════════════════════════════');
    console.log('CAPABILITY TOKEN');
    console.log('═══════════════════════════════════════════════════════════════');
    console.log(tokenBase64);
    console.log('═══════════════════════════════════════════════════════════════\n');
    
    console.log('CAVEATS (Embedded Permissions):');
    console.log(`  • scope = ${scope}`);
    console.log(`  • expires = ${new Date(expiresAt).toISOString()}`);
    if (maxCalls) console.log(`  • max_calls = ${Math.floor(maxCalls)}`);
    if (budgetSats) console.log(`  • budget_sats = ${Math.floor(budgetSats)}`);
    console.log('');
    
    console.log('USAGE:');
    console.log(`  curl -H "Authorization: Bearer ${tokenBase64.substring(0, 40)}..." \\`);
    console.log('    https://satgate-production.up.railway.app/api/capability/ping\n');
    
    // Full curl command for easy copy-paste
    console.log('FULL COMMAND (copy-paste ready):');
    console.log(`curl -H "Authorization: Bearer ${tokenBase64}" https://satgate-production.up.railway.app/api/capability/ping\n`);
    
    return tokenBase64;
    
  } catch (e) {
    console.error('❌ Failed to mint token:', e.message);
    process.exit(1);
  }
}

// Main
const options = parseArgs();
mintToken(options);


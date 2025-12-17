#!/usr/bin/env node
/**
 * SatGate Phase 1: Capability Token Minting CLI
 * 
 * Usage:
 *   node cli/mint-token.js [options]
 * 
 * Options:
 *   --scope <scope>     Token scope (default: api:capability:read)
 *   --expires <seconds> Token TTL in seconds (default: 3600)
 *   --key <key>         Root key (default: from CAPABILITY_ROOT_KEY env)
 * 
 * Example:
 *   node cli/mint-token.js --scope api:capability:read --expires 3600
 */

const macaroon = require('macaroon');

// Configuration
const CAPABILITY_ROOT_KEY = process.env.CAPABILITY_ROOT_KEY || 'satgate-phase1-demo-key-change-in-prod';
const CAPABILITY_LOCATION = 'https://satgate.io';
const CAPABILITY_IDENTIFIER = 'satgate-capability-v1';

// Parse CLI arguments
function parseArgs() {
  const args = process.argv.slice(2);
  const options = {
    scope: 'api:capability:read',
    expires: 3600,
    key: CAPABILITY_ROOT_KEY
  };
  
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--scope' && args[i + 1]) {
      options.scope = args[++i];
    } else if (args[i] === '--expires' && args[i + 1]) {
      options.expires = parseInt(args[++i], 10);
    } else if (args[i] === '--key' && args[i + 1]) {
      options.key = args[++i];
    } else if (args[i] === '--help' || args[i] === '-h') {
      console.log(`
SatGate Phase 1: Capability Token Minting CLI

Usage:
  node cli/mint-token.js [options]

Options:
  --scope <scope>     Token scope (default: api:capability:read)
  --expires <seconds> Token TTL in seconds (default: 3600)
  --key <key>         Root key (default: from CAPABILITY_ROOT_KEY env)
  --help, -h          Show this help

Examples:
  # Mint a token valid for 1 hour
  node cli/mint-token.js

  # Mint a token valid for 5 minutes with custom scope
  node cli/mint-token.js --scope api:capability:admin --expires 300

  # Mint using a custom key
  CAPABILITY_ROOT_KEY=my-secret node cli/mint-token.js
      `);
      process.exit(0);
    }
  }
  
  return options;
}

function mintToken(options) {
  const { scope, expires, key } = options;
  
  console.log('\n[SYSTEM] Generating Phase 1 Capability Token...');
  console.log(`[CONFIG] Scope: ${scope}`);
  console.log(`[CONFIG] TTL: ${expires} seconds`);
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


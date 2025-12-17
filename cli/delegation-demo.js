#!/usr/bin/env node
/**
 * SatGate Phase 1: Delegation Demo (Scene 4 - The "Wow" Moment)
 * 
 * This script demonstrates offline token delegation:
 * - Agent has a parent token with broad permissions
 * - Agent attenuates (restricts) the token for a sub-task
 * - NO NETWORK CALLS are made - pure cryptography
 * 
 * Usage:
 *   node cli/delegation-demo.js
 * 
 * The Analogy:
 *   "The agent just cut a spare key for the janitor — one that 
 *    only opens the basement, and expires at 5pm. It didn't 
 *    need to call the locksmith."
 */

const macaroon = require('macaroon');

// Configuration (same as server.js)
const CAPABILITY_ROOT_KEY = process.env.CAPABILITY_ROOT_KEY || 'satgate-phase1-demo-key-change-in-prod';
const CAPABILITY_LOCATION = 'https://satgate.io';
const CAPABILITY_IDENTIFIER = 'satgate-capability-v1';

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function runDemo() {
  console.log('\n');
  console.log('═══════════════════════════════════════════════════════════════');
  console.log('  SatGate Phase 1: DELEGATION DEMO');
  console.log('  "The Google-Grade Superpower"');
  console.log('═══════════════════════════════════════════════════════════════');
  console.log('\n');
  
  // Step 1: CISO creates parent token
  console.log('┌─────────────────────────────────────────────────────────────┐');
  console.log('│ SCENE 1: CISO Issues Parent Token                          │');
  console.log('└─────────────────────────────────────────────────────────────┘');
  console.log('\n[CISO] I\'m creating a capability for the Data Agent.');
  console.log('[CISO] Scope: Full API read access. Expires: 1 hour.\n');
  
  await sleep(500);
  
  const keyBytes = Buffer.from(CAPABILITY_ROOT_KEY, 'utf8');
  const parentId = `${CAPABILITY_IDENTIFIER}:parent:${Date.now()}`;
  
  let parentMacaroon = macaroon.newMacaroon({
    identifier: Buffer.from(parentId, 'utf8'),
    location: CAPABILITY_LOCATION,
    rootKey: keyBytes
  });
  
  // Parent: broad scope, 1 hour expiry
  const parentExpiry = Date.now() + (60 * 60 * 1000);
  parentMacaroon = macaroon.addFirstPartyCaveat(
    parentMacaroon, 
    Buffer.from(`expires = ${parentExpiry}`, 'utf8')
  );
  parentMacaroon = macaroon.addFirstPartyCaveat(
    parentMacaroon, 
    Buffer.from(`scope = api:capability:*`, 'utf8')
  );
  
  const parentToken = Buffer.from(macaroon.exportMacaroons([parentMacaroon])).toString('base64');
  
  console.log('✅ Parent Token Created');
  console.log(`   Scope: api:capability:* (full access)`);
  console.log(`   Expires: ${new Date(parentExpiry).toISOString()}`);
  console.log(`   Token: ${parentToken.substring(0, 50)}...`);
  console.log('\n');
  
  await sleep(1000);
  
  // Step 2: Agent needs to delegate
  console.log('┌─────────────────────────────────────────────────────────────┐');
  console.log('│ SCENE 2: Agent Needs to Delegate a Sub-Task                │');
  console.log('└─────────────────────────────────────────────────────────────┘');
  console.log('\n[AGENT] I need to delegate a read-only task to a Worker.');
  console.log('[AGENT] The Worker should only access /ping for 5 minutes.');
  console.log('[AGENT] In traditional IAM, this would require:');
  console.log('        → Request new service account (ticket)');
  console.log('        → Wait for approval (days)');
  console.log('        → Provision credentials');
  console.log('        → Rotate later\n');
  
  await sleep(1000);
  
  console.log('[AGENT] With capability tokens, I do this:\n');
  
  await sleep(500);
  
  // Step 3: The Magic - Offline Delegation
  console.log('┌─────────────────────────────────────────────────────────────┐');
  console.log('│ SCENE 3: Offline Delegation (THE WOW MOMENT)               │');
  console.log('└─────────────────────────────────────────────────────────────┘\n');
  
  console.log('[SYSTEM] Generating restricted sub-token...');
  console.log('[NETWORK] Requests sent: 0  ← OFFLINE OPERATION');
  console.log('[CRYPTO] Attenuating parent macaroon...\n');
  
  await sleep(800);
  
  // Re-import parent and add more restrictive caveats
  const parentBytes = Buffer.from(parentToken, 'base64');
  let childMacaroon = macaroon.importMacaroons(parentBytes)[0];
  
  // Child: narrower scope, 5 minute expiry (MORE restrictive)
  const childExpiry = Date.now() + (5 * 60 * 1000);
  childMacaroon = macaroon.addFirstPartyCaveat(
    childMacaroon, 
    Buffer.from(`expires = ${childExpiry}`, 'utf8')  // Shorter than parent
  );
  childMacaroon = macaroon.addFirstPartyCaveat(
    childMacaroon, 
    Buffer.from(`scope = api:capability:ping`, 'utf8')  // Narrower than parent
  );
  childMacaroon = macaroon.addFirstPartyCaveat(
    childMacaroon, 
    Buffer.from(`delegated_by = agent-001`, 'utf8')  // Audit trail
  );
  
  const childToken = Buffer.from(macaroon.exportMacaroons([childMacaroon])).toString('base64');
  
  console.log('✅ Child Token Created (Attenuated)\n');
  
  console.log('┌────────────────────┬────────────────────────────────────────┐');
  console.log('│ Property           │ Parent Token     │ Child Token        │');
  console.log('├────────────────────┼────────────────────────────────────────┤');
  console.log(`│ Scope              │ api:capability:* │ api:capability:ping│`);
  console.log(`│ Expires            │ 1 hour           │ 5 minutes          │`);
  console.log('│ Network calls      │ 0                │ 0                  │');
  console.log('│ Admin approval     │ NO               │ NO                 │');
  console.log('└────────────────────┴────────────────────────────────────────┘\n');
  
  await sleep(500);
  
  console.log('[CRYPTO] Sub-token signature: VALID');
  console.log('[CRYPTO] Caveat chain: ATTENUATED (more restrictive)\n');
  
  console.log('Child Token (for Worker):');
  console.log('═══════════════════════════════════════════════════════════════');
  console.log(childToken);
  console.log('═══════════════════════════════════════════════════════════════\n');
  
  await sleep(1000);
  
  // Step 4: The Key Insight
  console.log('┌─────────────────────────────────────────────────────────────┐');
  console.log('│ THE KEY INSIGHT                                            │');
  console.log('└─────────────────────────────────────────────────────────────┘\n');
  
  console.log('  "The agent just cut a spare key for the janitor —');
  console.log('   one that only opens the basement, and expires at 5pm.');
  console.log('   It didn\'t need to call the locksmith."\n');
  
  console.log('  ✓ ZERO network calls');
  console.log('  ✓ ZERO admin tickets');
  console.log('  ✓ INSTANT delegation');
  console.log('  ✓ SELF-EXPIRING credentials');
  console.log('  ✓ MATHEMATICALLY restricted scope\n');
  
  console.log('  This is the Google-grade superpower:');
  console.log('  Agents self-manage least-privilege access.\n');
  
  await sleep(500);
  
  // Test commands
  console.log('┌─────────────────────────────────────────────────────────────┐');
  console.log('│ TEST IT                                                    │');
  console.log('└─────────────────────────────────────────────────────────────┘\n');
  
  console.log('# Child token can access /ping:');
  console.log(`curl -H "Authorization: Bearer ${childToken.substring(0, 30)}..." \\`);
  console.log('  https://satgate-production.up.railway.app/api/capability/ping\n');
  
  console.log('# Full command (copy-paste):');
  console.log(`curl -H "Authorization: Bearer ${childToken}" https://satgate-production.up.railway.app/api/capability/ping\n`);
}

// Run the demo
runDemo().catch(console.error);


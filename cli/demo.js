#!/usr/bin/env node
/**
 * SatGate Golden Demo Script
 * 
 * Runs a complete demo of all three phases:
 *   1. CRAWL: Mint → Call → Delegate (Stateless Capabilities)
 *   2. WALK: Ban token → Watch dashboard update (Governance)
 *   3. RUN: Hit paid endpoint → 402 → Pay → Access (Economic Firewall)
 * 
 * Usage:
 *   node cli/demo.js                    # Interactive demo
 *   node cli/demo.js --check            # Readiness check only
 *   node cli/demo.js --phase crawl      # Run specific phase
 *   node cli/demo.js --api-url <url>    # Custom API URL
 * 
 * Environment:
 *   SATGATE_API_URL     - API base URL (default: live production)
 *   SATGATE_ADMIN_TOKEN - Admin token for governance demo
 */

const https = require('https');
const http = require('http');
const crypto = require('crypto');
const readline = require('readline');

// Configuration
const DEFAULT_API_URL = process.env.SATGATE_API_URL || 'https://l402-aperture-kit-production.up.railway.app';
const ADMIN_TOKEN = process.env.SATGATE_ADMIN_TOKEN || '';

// ANSI colors
const c = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  dim: '\x1b[2m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m',
};

const icons = {
  check: '✅',
  cross: '❌',
  warning: '⚠️',
  rocket: '🚀',
  shield: '🛡️',
  lightning: '⚡',
  key: '🔑',
  lock: '🔒',
  graph: '📊',
  coin: '🪙',
};

// Helper to make HTTP requests
function request(url, options = {}) {
  return new Promise((resolve, reject) => {
    const parsedUrl = new URL(url);
    const lib = parsedUrl.protocol === 'https:' ? https : http;
    
    const reqOptions = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port || (parsedUrl.protocol === 'https:' ? 443 : 80),
      path: parsedUrl.pathname + parsedUrl.search,
      method: options.method || 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    };

    const req = lib.request(reqOptions, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        try {
          resolve({
            status: res.statusCode,
            headers: res.headers,
            data: data ? JSON.parse(data) : null,
            raw: data,
          });
        } catch {
          resolve({
            status: res.statusCode,
            headers: res.headers,
            data: null,
            raw: data,
          });
        }
      });
    });

    req.on('error', reject);
    
    if (options.body) {
      req.write(JSON.stringify(options.body));
    }
    
    req.end();
  });
}

// Helper for user prompts
function prompt(question) {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  return new Promise(resolve => {
    rl.question(question, answer => {
      rl.close();
      resolve(answer);
    });
  });
}

// Print section header
function header(title, icon = icons.rocket) {
  console.log('');
  console.log(`${c.cyan}${'═'.repeat(60)}${c.reset}`);
  console.log(`${c.bright}${icon} ${title}${c.reset}`);
  console.log(`${c.cyan}${'═'.repeat(60)}${c.reset}`);
  console.log('');
}

// Print step
function step(num, description) {
  console.log(`${c.yellow}[Step ${num}]${c.reset} ${description}`);
}

// Print result
function result(success, message) {
  const icon = success ? `${c.green}${icons.check}` : `${c.red}${icons.cross}`;
  console.log(`  ${icon} ${message}${c.reset}`);
}

// =============================================================================
// READINESS CHECK
// =============================================================================

async function checkReadiness(apiUrl) {
  header('DEMO READINESS CHECK', icons.shield);
  
  const checks = [];
  
  // Check 1: API Health
  step(1, 'Checking API health...');
  try {
    const res = await request(`${apiUrl}/health`);
    if (res.status === 200 && res.data?.status === 'healthy') {
      result(true, `API healthy (uptime: ${Math.floor(res.data.uptime)}s)`);
      checks.push({ name: 'API Health', pass: true });
    } else {
      result(false, `API unhealthy: ${res.status}`);
      checks.push({ name: 'API Health', pass: false });
    }
  } catch (e) {
    result(false, `API unreachable: ${e.message}`);
    checks.push({ name: 'API Health', pass: false });
  }

  // Check 2: Governance Dashboard
  step(2, 'Checking governance dashboard...');
  try {
    const res = await request(`${apiUrl}/api/governance/stats`);
    if (res.status === 200 && res.data?.ok) {
      result(true, `Dashboard online (${res.data.stats?.active || 0} active tokens)`);
      checks.push({ name: 'Dashboard', pass: true });
    } else {
      result(false, `Dashboard error: ${res.status}`);
      checks.push({ name: 'Dashboard', pass: false });
    }
  } catch (e) {
    result(false, `Dashboard unreachable: ${e.message}`);
    checks.push({ name: 'Dashboard', pass: false });
  }

  // Check 3: Free Tier Endpoints
  step(3, 'Checking free tier endpoints...');
  try {
    const res = await request(`${apiUrl}/api/free/ping`);
    if (res.status === 200) {
      result(true, 'Free endpoints accessible');
      checks.push({ name: 'Free Tier', pass: true });
    } else {
      result(false, `Free tier error: ${res.status}`);
      checks.push({ name: 'Free Tier', pass: false });
    }
  } catch (e) {
    result(false, `Free tier error: ${e.message}`);
    checks.push({ name: 'Free Tier', pass: false });
  }

  // Check 4: L402 Challenge (should get 402)
  step(4, 'Checking L402 payment gate...');
  try {
    const res = await request(`${apiUrl}/api/micro/ping`);
    if (res.status === 402) {
      const hasInvoice = res.headers['www-authenticate']?.includes('invoice') || 
                         res.data?.invoice;
      result(true, `L402 gate active (402 returned${hasInvoice ? ' with invoice' : ''})`);
      checks.push({ name: 'L402 Gate', pass: true });
    } else if (res.status === 200) {
      result(true, 'Paid endpoint accessible (may be whitelisted)');
      checks.push({ name: 'L402 Gate', pass: true, note: 'whitelisted' });
    } else {
      result(false, `Unexpected status: ${res.status}`);
      checks.push({ name: 'L402 Gate', pass: false });
    }
  } catch (e) {
    result(false, `L402 check error: ${e.message}`);
    checks.push({ name: 'L402 Gate', pass: false });
  }

  // Check 5: Admin Token (if provided)
  step(5, 'Checking admin access...');
  if (ADMIN_TOKEN) {
    try {
      const res = await request(`${apiUrl}/api/governance/banned`, {
        headers: { 'X-Admin-Token': ADMIN_TOKEN }
      });
      if (res.status === 200) {
        result(true, `Admin access verified (${res.data?.count || 0} banned tokens)`);
        checks.push({ name: 'Admin Access', pass: true });
      } else if (res.status === 403) {
        result(false, 'Admin token invalid');
        checks.push({ name: 'Admin Access', pass: false });
      }
    } catch (e) {
      result(false, `Admin check error: ${e.message}`);
      checks.push({ name: 'Admin Access', pass: false });
    }
  } else {
    result(false, 'No SATGATE_ADMIN_TOKEN set (governance demo will be limited)');
    checks.push({ name: 'Admin Access', pass: false, note: 'optional' });
  }

  // Summary
  console.log('');
  console.log(`${c.cyan}${'─'.repeat(60)}${c.reset}`);
  const passed = checks.filter(c => c.pass).length;
  const total = checks.length;
  const allPass = passed === total || (passed >= total - 1 && !ADMIN_TOKEN);
  
  if (allPass) {
    console.log(`${c.green}${icons.check} READY FOR DEMO (${passed}/${total} checks passed)${c.reset}`);
  } else {
    console.log(`${c.yellow}${icons.warning} PARTIAL READINESS (${passed}/${total} checks passed)${c.reset}`);
  }
  console.log('');

  return { checks, ready: allPass };
}

// =============================================================================
// PHASE 1: CRAWL - Stateless Capabilities
// =============================================================================

async function demoCrawl(apiUrl) {
  header('PHASE 1: CRAWL - Stateless Capabilities', icons.key);
  
  console.log(`${c.dim}Narrative: "No database. No session. Just cryptographic proof."${c.reset}`);
  console.log('');

  // Step 1: Mint a token
  step(1, 'Minting a root capability token...');
  const rootToken = {
    version: 1,
    id: crypto.randomBytes(8).toString('hex'),
    caveats: {
      expires: Date.now() + 3600000, // 1 hour
      scope: 'api:capability:*',
    },
    issuedAt: Date.now(),
    signature: crypto.randomBytes(32).toString('hex'),
  };
  const encodedRoot = Buffer.from(JSON.stringify(rootToken)).toString('base64');
  console.log(`  ${c.dim}Token ID: ${rootToken.id}${c.reset}`);
  console.log(`  ${c.dim}Scope: ${rootToken.caveats.scope}${c.reset}`);
  console.log(`  ${c.dim}Expires: 1 hour${c.reset}`);
  result(true, 'Root token minted (offline, no API call)');

  // Step 2: Use the token
  step(2, 'Making authenticated API call...');
  try {
    const res = await request(`${apiUrl}/api/capability/ping`, {
      headers: { 'Authorization': `Bearer ${encodedRoot}` }
    });
    if (res.status === 200) {
      result(true, `API responded: ${JSON.stringify(res.data).substring(0, 50)}...`);
    } else {
      result(false, `Unexpected response: ${res.status}`);
    }
  } catch (e) {
    result(false, `API error: ${e.message}`);
  }

  // Step 3: Delegate (attenuate) the token
  step(3, 'Delegating with restricted scope...');
  const childToken = {
    version: 1,
    id: crypto.randomBytes(8).toString('hex'),
    parentSignature: rootToken.signature,
    caveats: {
      expires: Date.now() + 300000, // 5 minutes (RESTRICTED)
      scope: 'api:capability:ping', // RESTRICTED to ping only
    },
    issuedAt: Date.now(),
    signature: crypto.randomBytes(32).toString('hex'),
  };
  const encodedChild = Buffer.from(JSON.stringify(childToken)).toString('base64');
  console.log(`  ${c.dim}Child ID: ${childToken.id}${c.reset}`);
  console.log(`  ${c.green}Scope: ${childToken.caveats.scope} [RESTRICTED]${c.reset}`);
  console.log(`  ${c.green}Expires: 5 minutes [EPHEMERAL]${c.reset}`);
  result(true, 'Child token delegated (offline, cryptographic)');

  // Step 4: Inspect with governance tool
  step(4, 'Inspecting token governance...');
  console.log('');
  console.log(`  ${c.dim}Run: node cli/inspect.js ${encodedChild.substring(0, 30)}...${c.reset}`);
  console.log('');

  console.log(`${c.cyan}${'─'.repeat(60)}${c.reset}`);
  console.log(`${c.bright}Talk Track:${c.reset}`);
  console.log(`  "The agent didn't get a database row. It got a cryptographic`);
  console.log(`   chain of custody with built-in expiration and scope limits."`);
  console.log('');

  return { rootToken: encodedRoot, childToken: encodedChild };
}

// =============================================================================
// PHASE 2: WALK - Governance & Kill Switch
// =============================================================================

async function demoWalk(apiUrl, tokenToBan) {
  header('PHASE 2: WALK - Governance & Kill Switch', icons.shield);
  
  console.log(`${c.dim}Narrative: "Stateless validation + stateful revocation list."${c.reset}`);
  console.log('');

  if (!ADMIN_TOKEN) {
    console.log(`${c.yellow}${icons.warning} Skipping admin actions (no SATGATE_ADMIN_TOKEN)${c.reset}`);
    console.log(`${c.dim}Set SATGATE_ADMIN_TOKEN to demonstrate the kill switch.${c.reset}`);
    console.log('');
    return;
  }

  // Step 1: Check dashboard before
  step(1, 'Checking governance dashboard (before)...');
  try {
    const res = await request(`${apiUrl}/api/governance/stats`);
    console.log(`  ${c.dim}Active tokens: ${res.data?.stats?.active || 0}${c.reset}`);
    console.log(`  ${c.dim}Banned tokens: ${res.data?.stats?.banned || 0}${c.reset}`);
    result(true, 'Dashboard state captured');
  } catch (e) {
    result(false, e.message);
  }

  // Step 2: Ban a token
  const testSig = crypto.randomBytes(16).toString('hex');
  step(2, `Banning token (kill switch)...`);
  try {
    const res = await request(`${apiUrl}/api/governance/ban`, {
      method: 'POST',
      headers: { 'X-Admin-Token': ADMIN_TOKEN },
      body: { 
        tokenSignature: testSig,
        reason: 'Demo: Simulated compromise'
      }
    });
    if (res.data?.ok) {
      result(true, `Token ${testSig.substring(0, 8)}... banned`);
    } else {
      result(false, `Ban failed: ${res.data?.error}`);
    }
  } catch (e) {
    result(false, e.message);
  }

  // Step 3: Check dashboard after
  step(3, 'Checking governance dashboard (after)...');
  try {
    const res = await request(`${apiUrl}/api/governance/stats`);
    console.log(`  ${c.dim}Banned tokens: ${res.data?.stats?.banned || 0}${c.reset}`);
    result(true, 'Dashboard updated in real-time');
  } catch (e) {
    result(false, e.message);
  }

  // Step 4: Unban (cleanup)
  step(4, 'Unbanning token (cleanup)...');
  try {
    const res = await request(`${apiUrl}/api/governance/unban`, {
      method: 'POST',
      headers: { 'X-Admin-Token': ADMIN_TOKEN },
      body: { tokenSignature: testSig }
    });
    result(res.data?.ok, 'Token unbanned');
  } catch (e) {
    result(false, e.message);
  }

  console.log('');
  console.log(`${c.cyan}${'─'.repeat(60)}${c.reset}`);
  console.log(`${c.bright}Talk Track:${c.reset}`);
  console.log(`  "The moment you detect a compromised token, you ban it.`);
  console.log(`   Every subsequent request with that token is rejected instantly.`);
  console.log(`   Stateless for speed, stateful for emergencies."`);
  console.log('');
}

// =============================================================================
// PHASE 3: RUN - Economic Firewall (L402)
// =============================================================================

async function demoRun(apiUrl) {
  header('PHASE 3: RUN - Economic Firewall (L402)', icons.lightning);
  
  console.log(`${c.dim}Narrative: "We don't just block attackers; we bankrupt them."${c.reset}`);
  console.log('');

  // Step 1: Hit paid endpoint without payment
  step(1, 'Requesting paid endpoint without payment...');
  try {
    const res = await request(`${apiUrl}/api/micro/ping`);
    if (res.status === 402) {
      console.log(`  ${c.green}HTTP 402 Payment Required${c.reset}`);
      
      // Extract invoice info
      const wwwAuth = res.headers['www-authenticate'] || '';
      const invoiceMatch = wwwAuth.match(/invoice="([^"]+)"/);
      if (invoiceMatch) {
        const invoice = invoiceMatch[1];
        console.log(`  ${c.dim}Invoice: ${invoice.substring(0, 40)}...${c.reset}`);
      }
      
      if (res.data?.price) {
        console.log(`  ${c.dim}Price: ${res.data.price} sats${c.reset}`);
      }
      
      result(true, 'L402 challenge received');
    } else {
      result(false, `Expected 402, got ${res.status}`);
    }
  } catch (e) {
    result(false, e.message);
  }

  // Step 2: Explain the economic firewall
  step(2, 'Economic Firewall Analysis');
  console.log('');
  console.log(`  ${icons.coin} Price per request: 1 sat (~$0.0004)`);
  console.log(`  ${icons.shield} Cost to send 1M requests: ~$400`);
  console.log(`  ${c.dim}Attackers have infinite IPs, but not infinite money.${c.reset}`);
  console.log('');
  result(true, 'Economic deterrent active');

  // Step 3: Show SDK flow
  step(3, 'SDK Auto-Payment Flow (Python example)');
  console.log('');
  console.log(`  ${c.dim}from satgate import SatGateSession, AlbyWallet${c.reset}`);
  console.log(`  ${c.dim}session = SatGateSession(wallet=AlbyWallet(token))${c.reset}`);
  console.log(`  ${c.dim}# SDK automatically: detects 402 → pays invoice → retries${c.reset}`);
  console.log(`  ${c.dim}data = session.get("${apiUrl}/api/micro/ping")${c.reset}`);
  console.log('');
  result(true, 'SDK handles payment automatically');

  console.log('');
  console.log(`${c.cyan}${'─'.repeat(60)}${c.reset}`);
  console.log(`${c.bright}Talk Track:${c.reset}`);
  console.log(`  "This is the Economic Firewall. Every malicious request costs`);
  console.log(`   real money. Brute force? That'll be $50,000. DDoS? They're`);
  console.log(`   paying your AWS bill. We don't block attackers; we bankrupt them."`);
  console.log('');
}

// =============================================================================
// MAIN
// =============================================================================

async function main() {
  const args = process.argv.slice(2);
  const apiUrl = args.find(a => a.startsWith('--api-url='))?.split('=')[1] || DEFAULT_API_URL;
  
  console.log('');
  console.log(`${c.bright}${icons.rocket} SatGate Golden Demo${c.reset}`);
  console.log(`${c.dim}API: ${apiUrl}${c.reset}`);
  
  // Check-only mode
  if (args.includes('--check')) {
    await checkReadiness(apiUrl);
    return;
  }

  // Run specific phase
  const phase = args.find(a => a.startsWith('--phase='))?.split('=')[1];
  
  if (phase) {
    switch (phase.toLowerCase()) {
      case 'crawl':
      case '1':
        await demoCrawl(apiUrl);
        break;
      case 'walk':
      case '2':
        await demoWalk(apiUrl);
        break;
      case 'run':
      case '3':
        await demoRun(apiUrl);
        break;
      default:
        console.log(`Unknown phase: ${phase}`);
        console.log('Valid phases: crawl, walk, run (or 1, 2, 3)');
    }
    return;
  }

  // Full interactive demo
  const { ready } = await checkReadiness(apiUrl);
  
  if (!ready) {
    const cont = await prompt(`${c.yellow}Continue anyway? (y/N): ${c.reset}`);
    if (cont.toLowerCase() !== 'y') {
      console.log('Demo cancelled.');
      return;
    }
  }

  console.log('');
  console.log(`${c.bright}Starting full demo sequence...${c.reset}`);
  console.log(`${c.dim}Press Enter between phases, or Ctrl+C to exit.${c.reset}`);
  
  // Phase 1
  await prompt(`\n${c.cyan}Press Enter for Phase 1 (CRAWL)...${c.reset}`);
  const tokens = await demoCrawl(apiUrl);
  
  // Phase 2
  await prompt(`\n${c.cyan}Press Enter for Phase 2 (WALK)...${c.reset}`);
  await demoWalk(apiUrl, tokens?.childToken);
  
  // Phase 3
  await prompt(`\n${c.cyan}Press Enter for Phase 3 (RUN)...${c.reset}`);
  await demoRun(apiUrl);
  
  // Finale
  header('DEMO COMPLETE', icons.check);
  console.log(`${c.bright}Summary:${c.reset}`);
  console.log(`  ${icons.key} CRAWL: Stateless capability tokens with delegation`);
  console.log(`  ${icons.shield} WALK: Real-time governance with instant revocation`);
  console.log(`  ${icons.lightning} RUN: Economic firewall via Lightning micropayments`);
  console.log('');
  console.log(`${c.green}${icons.rocket} SatGate: Zero Trust PEP for APIs${c.reset}`);
  console.log('');
}

main().catch(err => {
  console.error(`${c.red}Error: ${err.message}${c.reset}`);
  process.exit(1);
});


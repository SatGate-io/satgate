#!/usr/bin/env node
/**
 * SatGate Governance Inspector
 * 
 * Visual proof of Chain of Custody for capability tokens.
 * 
 * Usage:
 *   node cli/inspect.js <macaroon>
 *   node cli/inspect.js <macaroon> --json
 *   echo $TOKEN | node cli/inspect.js -
 * 
 * Demo Talk Track:
 *   "See? I didn't just give the agent a key. I gave it a 
 *    chain of custody that expires in 5 minutes."
 */

const crypto = require('crypto');

// ANSI colors for terminal output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  dim: '\x1b[2m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m',
  white: '\x1b[37m',
  bgRed: '\x1b[41m',
  bgGreen: '\x1b[42m',
};

const icons = {
  crown: '👑',
  key: '🔑',
  lock: '🔒',
  clock: '🕒',
  check: '✅',
  cross: '❌',
  warning: '⚠️',
  chain: '🔗',
  shield: '🛡️',
  tree: '🌳',
  magnify: '🔍',
  delegation: '🔻',
  scope: '📋',
  arrow: '→',
};

/**
 * Decode a base64 macaroon and extract its components
 */
function decodeMacaroon(token) {
  try {
    // Try base64 decode
    let decoded;
    try {
      decoded = Buffer.from(token, 'base64').toString('utf8');
    } catch {
      // Maybe it's already a hex macaroon from aperture
      decoded = token;
    }

    // Try to parse as JSON (our custom format)
    try {
      const parsed = JSON.parse(decoded);
      return {
        type: 'satgate',
        raw: token,
        ...parsed
      };
    } catch {
      // Not JSON - try to parse as binary macaroon
      return parseBinaryMacaroon(token);
    }
  } catch (error) {
    return { error: `Failed to decode: ${error.message}`, raw: token };
  }
}

/**
 * Parse Aperture/standard binary macaroon format
 */
function parseBinaryMacaroon(token) {
  try {
    // Macaroons are typically hex or base64 encoded
    let bytes;
    if (/^[0-9a-fA-F]+$/.test(token)) {
      bytes = Buffer.from(token, 'hex');
    } else {
      bytes = Buffer.from(token, 'base64');
    }

    // Extract what we can from binary format
    // Macaroon v2 format: version(1) | location_len(2) | location | id_len(2) | id | ...
    
    const result = {
      type: 'binary',
      raw: token,
      length: bytes.length,
      signature: bytes.slice(-32).toString('hex'),
      caveats: [],
    };

    // Try to extract readable strings (caveats are often plaintext)
    const str = bytes.toString('utf8', 0, Math.min(bytes.length, 500));
    const caveatMatches = str.match(/[a-z_]+\s*[=<>]\s*[^\x00]+/gi) || [];
    
    for (const match of caveatMatches) {
      if (match.includes('=') || match.includes('<') || match.includes('>')) {
        result.caveats.push(match.trim());
      }
    }

    // Look for common patterns
    if (str.includes('time-before')) {
      const timeMatch = str.match(/time-before\s*<?\s*(\d+)/);
      if (timeMatch) {
        result.expiry = parseInt(timeMatch[1]);
      }
    }

    if (str.includes('capabilities')) {
      const capMatch = str.match(/capabilities\s*=\s*([^\x00]+)/);
      if (capMatch) {
        result.capabilities = capMatch[1].trim();
      }
    }

    return result;
  } catch (error) {
    return { 
      type: 'unknown',
      raw: token,
      error: `Could not parse binary: ${error.message}` 
    };
  }
}

/**
 * Calculate time remaining until expiry
 */
function getTimeRemaining(expiryTimestamp) {
  const now = Date.now();
  const expiry = typeof expiryTimestamp === 'number' 
    ? (expiryTimestamp > 1e12 ? expiryTimestamp : expiryTimestamp * 1000)
    : new Date(expiryTimestamp).getTime();
  
  const diff = expiry - now;
  
  if (diff <= 0) {
    return { expired: true, text: 'EXPIRED', seconds: 0 };
  }

  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  let text;
  if (days > 0) text = `${days}d ${hours % 24}h`;
  else if (hours > 0) text = `${hours}h ${minutes % 60}m`;
  else if (minutes > 0) text = `${minutes}m ${seconds % 60}s`;
  else text = `${seconds}s`;

  return { expired: false, text, seconds };
}

/**
 * Render the governance tree visualization
 */
function renderTree(macaroon, options = {}) {
  const { json } = options;
  
  if (json) {
    console.log(JSON.stringify(macaroon, null, 2));
    return;
  }

  const line = '═'.repeat(50);
  
  console.log('');
  console.log(`${colors.cyan}${icons.magnify} SATGATE GOVERNANCE INSPECTOR${colors.reset}`);
  console.log(`${colors.dim}${line}${colors.reset}`);
  
  // Token ID / Signature
  const sig = macaroon.signature || macaroon.raw?.slice(-16) || 'unknown';
  const shortSig = typeof sig === 'string' ? sig.slice(0, 12) + '...' : sig;
  console.log(`${colors.bright}TOKEN ID:${colors.reset} ${shortSig}`);
  console.log(`${colors.bright}TYPE:${colors.reset} ${macaroon.type || 'unknown'}`);
  
  // Status
  let status = `${colors.green}${icons.check} Valid${colors.reset}`;
  if (macaroon.error) {
    status = `${colors.red}${icons.cross} Error: ${macaroon.error}${colors.reset}`;
  } else if (macaroon.caveats?.expires || macaroon.expiry) {
    const expiry = macaroon.caveats?.expires || macaroon.expiry;
    const remaining = getTimeRemaining(expiry);
    if (remaining.expired) {
      status = `${colors.red}${icons.cross} EXPIRED${colors.reset}`;
    } else if (remaining.seconds < 300) {
      status = `${colors.yellow}${icons.warning} Expiring in ${remaining.text}${colors.reset}`;
    } else {
      status = `${colors.green}${icons.check} Valid (${remaining.text} remaining)${colors.reset}`;
    }
  }
  console.log(`${colors.bright}STATUS:${colors.reset} ${status}`);
  
  console.log(`${colors.dim}${line}${colors.reset}`);
  
  // Lineage Tree
  console.log(`${colors.cyan}${icons.tree} LINEAGE (Chain of Custody)${colors.reset}`);
  console.log('');
  
  // Root
  if (macaroon.type === 'satgate' && macaroon.paymentHash) {
    // L402 token with payment proof
    console.log(`${colors.bright}┌── ${icons.crown} ROOT (L402 Payment)${colors.reset}`);
    console.log(`${colors.dim}│   ${icons.chain} Payment Hash: ${macaroon.paymentHash.slice(0, 16)}...${colors.reset}`);
  } else if (macaroon.parentSignature || macaroon.parent) {
    // Delegated token
    const parentSig = macaroon.parentSignature || macaroon.parent;
    console.log(`${colors.bright}┌── ${icons.crown} ROOT${colors.reset}`);
    console.log(`${colors.dim}│   ${icons.key} Parent: ${parentSig.slice(0, 12)}...${colors.reset}`);
    console.log(`${colors.dim}│${colors.reset}`);
    console.log(`${colors.yellow}└── ${icons.delegation} DELEGATED (This Token)${colors.reset}`);
  } else {
    // Master token
    console.log(`${colors.bright}┌── ${icons.crown} MASTER TOKEN${colors.reset}`);
    console.log(`${colors.dim}│   ${icons.shield} This is a root capability${colors.reset}`);
  }
  
  // Caveats / Constraints
  console.log('');
  console.log(`${colors.dim}${line}${colors.reset}`);
  console.log(`${colors.cyan}${icons.lock} CONSTRAINTS (Caveats)${colors.reset}`);
  console.log('');
  
  // Time constraint
  if (macaroon.caveats?.expires || macaroon.expiry) {
    const expiry = macaroon.caveats?.expires || macaroon.expiry;
    const remaining = getTimeRemaining(expiry);
    const expiryDate = new Date(typeof expiry === 'number' && expiry < 1e12 ? expiry * 1000 : expiry);
    
    if (remaining.expired) {
      console.log(`    ${colors.red}${icons.clock} Time: EXPIRED at ${expiryDate.toISOString()}${colors.reset}`);
    } else if (remaining.seconds < 300) {
      console.log(`    ${colors.yellow}${icons.clock} Time: ${remaining.text} remaining [EPHEMERAL]${colors.reset}`);
    } else {
      console.log(`    ${colors.green}${icons.clock} Time: ${remaining.text} remaining${colors.reset}`);
    }
  } else if (macaroon.caveats && Array.isArray(macaroon.caveats)) {
    // Binary macaroon caveats
    for (const caveat of macaroon.caveats) {
      if (caveat.toLowerCase().includes('time')) {
        console.log(`    ${icons.clock} ${caveat}`);
      }
    }
  } else {
    console.log(`    ${colors.dim}${icons.clock} Time: No expiry (persistent)${colors.reset}`);
  }
  
  // Scope constraint
  if (macaroon.caveats?.scope) {
    const scope = macaroon.caveats.scope;
    const isRestricted = scope !== '*' && scope !== 'api:*';
    console.log(`    ${isRestricted ? colors.green : colors.dim}${icons.scope} Scope: ${scope}${isRestricted ? ' [RESTRICTED]' : ''}${colors.reset}`);
  } else if (macaroon.capabilities) {
    console.log(`    ${icons.scope} Capabilities: ${macaroon.capabilities}`);
  } else if (macaroon.caveats && Array.isArray(macaroon.caveats)) {
    for (const caveat of macaroon.caveats) {
      if (caveat.toLowerCase().includes('cap') || caveat.toLowerCase().includes('scope')) {
        console.log(`    ${icons.scope} ${caveat}`);
      }
    }
  }

  // Max calls (stateful budget enforcement)
  if (macaroon.caveats?.max_calls) {
    console.log(`    ${icons.lock} Max Calls: ${macaroon.caveats.max_calls}`);
  } else if (macaroon.caveats && Array.isArray(macaroon.caveats)) {
    for (const caveat of macaroon.caveats) {
      if (caveat.toLowerCase().includes('max_calls')) {
        console.log(`    ${icons.lock} ${caveat}`);
      }
    }
  }
  
  // Budget sats (stateful sats budget enforcement)
  if (macaroon.caveats?.budget_sats) {
    console.log(`    ${icons.key} Budget: ${macaroon.caveats.budget_sats} sats`);
  } else if (macaroon.caveats && Array.isArray(macaroon.caveats)) {
    for (const caveat of macaroon.caveats) {
      if (caveat.toLowerCase().includes('budget_sats')) {
        console.log(`    ${icons.key} ${caveat}`);
      }
    }
  }
  
  // Tier
  if (macaroon.caveats?.tier) {
    console.log(`    ${icons.key} Tier: ${macaroon.caveats.tier}`);
  }
  
  // Service constraint (for Aperture macaroons)
  if (macaroon.service) {
    console.log(`    ${icons.lock} Service: ${macaroon.service}`);
  }
  
  // Governance Check Summary
  console.log('');
  console.log(`${colors.dim}${line}${colors.reset}`);
  console.log(`${colors.cyan}${icons.shield} GOVERNANCE CHECK${colors.reset}`);
  console.log('');
  
  const checks = [];
  
  // Least Privilege check
  const hasRestrictedScope = macaroon.caveats?.scope && 
    macaroon.caveats.scope !== '*' && 
    macaroon.caveats.scope !== 'api:*';
  checks.push({
    name: 'Least Privilege',
    pass: hasRestrictedScope,
    note: hasRestrictedScope ? 'Scope narrowed' : 'Full access'
  });
  
  // Ephemeral check
  let isEphemeral = false;
  if (macaroon.caveats?.expires || macaroon.expiry) {
    const remaining = getTimeRemaining(macaroon.caveats?.expires || macaroon.expiry);
    isEphemeral = !remaining.expired && remaining.seconds < 600; // < 10 min
  }
  checks.push({
    name: 'Ephemeral',
    pass: isEphemeral,
    note: isEphemeral ? 'Expires < 10 mins' : 'Long-lived'
  });
  
  // Traceable check
  const isTraceable = !!(macaroon.parentSignature || macaroon.paymentHash || macaroon.signature);
  checks.push({
    name: 'Traceable',
    pass: isTraceable,
    note: isTraceable ? 'Linked to root' : 'No lineage'
  });
  
  for (const check of checks) {
    const icon = check.pass ? `${colors.green}${icons.check}` : `${colors.yellow}○`;
    const status = check.pass ? 'YES' : 'NO';
    console.log(`    ${icon} ${check.name}? ${colors.bright}${status}${colors.reset} ${colors.dim}(${check.note})${colors.reset}`);
  }
  
  console.log('');
  console.log(`${colors.dim}${line}${colors.reset}`);
  
  // Footer with talk track
  const allPass = checks.every(c => c.pass);
  if (allPass) {
    console.log(`${colors.green}${icons.shield} AUDIT READY: Full chain of custody with ephemeral, scoped access.${colors.reset}`);
  } else {
    console.log(`${colors.yellow}${icons.warning} REVIEW: Some governance checks did not pass.${colors.reset}`);
  }
  console.log('');
}

/**
 * Main entry point
 */
async function main() {
  const args = process.argv.slice(2);
  
  if (args.length === 0 || args.includes('--help') || args.includes('-h')) {
    console.log(`
${colors.cyan}SatGate Governance Inspector${colors.reset}

${colors.bright}Usage:${colors.reset}
  node cli/inspect.js <macaroon>           Inspect a token
  node cli/inspect.js <macaroon> --json    Output as JSON
  echo $TOKEN | node cli/inspect.js -      Read from stdin

${colors.bright}Examples:${colors.reset}
  # Inspect a base64 macaroon
  node cli/inspect.js AgELc2F0Z2F0ZS5pbw...

  # Inspect and get JSON output
  node cli/inspect.js $CHILD_TOKEN --json

  # Pipe from environment
  echo $SATGATE_TOKEN | node cli/inspect.js -

${colors.bright}What it shows:${colors.reset}
  ${icons.crown} Token lineage (parent chain)
  ${icons.clock} Time constraints (expiry)
  ${icons.scope} Scope restrictions
  ${icons.shield} Governance compliance checks

${colors.bright}Demo Talk Track:${colors.reset}
  "See? I didn't just give the agent a key. I gave it a 
   chain of custody that expires in 5 minutes."
`);
    process.exit(0);
  }

  let token = args[0];
  const jsonOutput = args.includes('--json');

  // Read from stdin if '-'
  if (token === '-') {
    const chunks = [];
    for await (const chunk of process.stdin) {
      chunks.push(chunk);
    }
    token = Buffer.concat(chunks).toString().trim();
  }

  if (!token) {
    console.error(`${colors.red}Error: No token provided${colors.reset}`);
    process.exit(1);
  }

  const macaroon = decodeMacaroon(token);
  renderTree(macaroon, { json: jsonOutput });
}

main().catch(err => {
  console.error(`${colors.red}Error: ${err.message}${colors.reset}`);
  process.exit(1);
});

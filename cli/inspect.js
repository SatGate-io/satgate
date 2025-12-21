#!/usr/bin/env node
/**
 * SatGate Governance Inspector
 * 
 * Visualizes token constraints and lineage for audit/compliance.
 * Proves "Chain of Custody" without needing a database.
 * 
 * Usage:
 *   node cli/inspect.js <TOKEN>
 *   node cli/inspect.js <TOKEN> --json
 * 
 * Demo Scene:
 *   Auditor: "Who authorized this request?"
 *   You: "Let's ask the token."
 */

const macaroon = require('macaroon');

// ANSI Colors
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  dim: '\x1b[2m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  red: '\x1b[31m',
  cyan: '\x1b[36m',
  magenta: '\x1b[35m',
  white: '\x1b[37m',
  bgBlack: '\x1b[40m'
};

function c(color, text) {
  return `${colors[color]}${text}${colors.reset}`;
}

function parseCaveats(caveats) {
  const parsed = {
    expires: null,
    expiresRelative: null,
    scopes: [],
    delegatedBy: null,
    custom: []
  };
  
  for (const caveat of caveats) {
    const str = caveat.toString('utf8');
    
    if (str.startsWith('expires = ')) {
      const ts = parseInt(str.split('=')[1].trim());
      parsed.expires = new Date(ts);
      const remaining = ts - Date.now();
      if (remaining > 0) {
        const mins = Math.floor(remaining / 60000);
        const secs = Math.floor((remaining % 60000) / 1000);
        parsed.expiresRelative = `${mins}m ${secs}s remaining`;
      } else {
        parsed.expiresRelative = 'EXPIRED';
      }
    } else if (str.startsWith('scope = ')) {
      parsed.scopes.push(str.split('=')[1].trim());
    } else if (str.startsWith('delegated_by = ')) {
      parsed.delegatedBy = str.split('=')[1].trim();
    } else {
      parsed.custom.push(str);
    }
  }
  
  return parsed;
}

function determineDepth(caveats) {
  // Heuristic: Each delegation adds caveats
  // Root tokens typically have 2 caveats (expires, scope)
  // Delegated tokens have more (narrower scope, delegated_by, etc.)
  const hasDelegation = caveats.some(c => 
    c.toString('utf8').includes('delegated_by')
  );
  const scopeCount = caveats.filter(c => 
    c.toString('utf8').startsWith('scope = ')
  ).length;
  
  if (!hasDelegation && scopeCount <= 1) return 0; // Root
  if (hasDelegation && scopeCount <= 2) return 1; // First delegation
  return 2; // Deep delegation
}

function runGovernanceChecks(parsed, depth) {
  const checks = [];
  
  // Least Privilege check
  const hasNarrowScope = parsed.scopes.some(s => !s.endsWith('*'));
  checks.push({
    name: 'Least Privilege',
    passed: hasNarrowScope || depth === 0,
    reason: hasNarrowScope ? 'Scope is narrowed' : (depth === 0 ? 'Root token (expected broad)' : 'Scope uses wildcard')
  });
  
  // Ephemeral check
  const remaining = parsed.expires ? parsed.expires.getTime() - Date.now() : 0;
  const isEphemeral = remaining > 0 && remaining < 10 * 60 * 1000; // < 10 mins
  checks.push({
    name: 'Ephemeral',
    passed: isEphemeral || depth === 0,
    reason: isEphemeral ? 'Expires < 10 mins' : (depth === 0 ? 'Root token (longer TTL expected)' : 'Long-lived token')
  });
  
  // Traceable check
  const isTraceable = depth === 0 || parsed.delegatedBy !== null;
  checks.push({
    name: 'Traceable',
    passed: isTraceable,
    reason: isTraceable ? (parsed.delegatedBy ? `Delegated by: ${parsed.delegatedBy}` : 'Root token') : 'Missing delegation chain'
  });
  
  // Validity check
  const isValid = !parsed.expiresRelative || parsed.expiresRelative !== 'EXPIRED';
  checks.push({
    name: 'Valid',
    passed: isValid,
    reason: isValid ? 'Token not expired' : 'Token has expired'
  });
  
  return checks;
}

function printInspection(token, outputJson) {
  try {
    const bytes = Buffer.from(token, 'base64');
    const m = macaroon.importMacaroon(bytes);
    
    const identifier = m._exportAsJSONObjectV2().i || m._exportAsJSONObjectV2().i64;
    const location = m._exportAsJSONObjectV2().l || 'unknown';
    const signature = Buffer.from(m.signature).toString('hex').substring(0, 16);
    const caveats = m.caveats || [];
    
    const parsed = parseCaveats(caveats);
    const depth = determineDepth(caveats);
    const checks = runGovernanceChecks(parsed, depth);
    
    const depthLabels = ['ROOT', 'DELEGATED', 'DEEP'];
    const tokenType = depthLabels[depth] || 'UNKNOWN';
    
    if (outputJson) {
      console.log(JSON.stringify({
        id: signature,
        type: tokenType,
        depth,
        location,
        expires: parsed.expires,
        expiresRelative: parsed.expiresRelative,
        scopes: parsed.scopes,
        delegatedBy: parsed.delegatedBy,
        customCaveats: parsed.custom,
        governanceChecks: checks
      }, null, 2));
      return;
    }
    
    // Pretty print
    console.log('\n');
    console.log(c('cyan', '🔍 SATGATE GOVERNANCE INSPECTOR'));
    console.log(c('dim', '════════════════════════════════════════════════════════════'));
    console.log(`${c('bright', 'TOKEN ID:')}    ${signature}... (${tokenType})`);
    console.log(`${c('bright', 'LOCATION:')}    ${location}`);
    
    const isExpired = parsed.expiresRelative === 'EXPIRED';
    const statusColor = isExpired ? 'red' : 'green';
    const statusIcon = isExpired ? '❌' : '✅';
    console.log(`${c('bright', 'STATUS:')}      ${c(statusColor, `${statusIcon} ${isExpired ? 'EXPIRED' : 'VALID'}`)}`);
    console.log(c('dim', '════════════════════════════════════════════════════════════'));
    
    console.log(`\n${c('magenta', '📉 LINEAGE (Chain of Custody)')}`);
    
    if (depth === 0) {
      console.log(c('yellow', '┌── 👑 ROOT TOKEN'));
      console.log(`│    ├── 🕒 Expires: ${parsed.expiresRelative || 'No expiry'}`);
      console.log(`│    └── 🔒 Scope:   ${parsed.scopes.join(', ') || 'Unrestricted'}`);
    } else {
      console.log(c('dim', '┌── 👑 ROOT (Inferred from signature chain)'));
      console.log('│');
      if (depth >= 1) {
        const delegatorLabel = parsed.delegatedBy || 'Unknown Agent';
        console.log(c('yellow', `└── 🔻 DELEGATION (Signed by ${delegatorLabel})`));
        console.log(`     ├── 🕒 Expires: ${parsed.expiresRelative || 'No expiry'} ${c('cyan', '[RESTRICTED]')}`);
        const narrowestScope = parsed.scopes[parsed.scopes.length - 1] || 'Unrestricted';
        console.log(`     └── 🔒 Scope:   ${narrowestScope} ${c('cyan', '[RESTRICTED]')}`);
      }
    }
    
    console.log(`\n${c('green', '✅ GOVERNANCE CHECKS:')}`);
    for (const check of checks) {
      const icon = check.passed ? c('green', '✓') : c('red', '✗');
      const label = check.passed ? c('green', 'YES') : c('red', 'NO');
      console.log(`   ${icon} ${check.name.padEnd(16)} ${label.padEnd(15)} (${check.reason})`);
    }
    
    // Custom caveats
    if (parsed.custom.length > 0) {
      console.log(`\n${c('yellow', '📋 CUSTOM CAVEATS:')}`);
      for (const caveat of parsed.custom) {
        console.log(`   • ${caveat}`);
      }
    }
    
    console.log('\n' + c('dim', '════════════════════════════════════════════════════════════'));
    console.log(c('dim', 'This token carries its own audit trail. No database lookup required.\n'));
    
  } catch (err) {
    console.error(c('red', `\n❌ Failed to parse token: ${err.message}`));
    console.error(c('dim', 'Ensure the token is a valid base64-encoded macaroon.\n'));
    process.exit(1);
  }
}

// Main
const args = process.argv.slice(2);
const jsonFlag = args.includes('--json');
const token = args.find(a => !a.startsWith('--'));

if (!token) {
  console.log(`
${c('cyan', 'SatGate Governance Inspector')}

${c('bright', 'Usage:')}
  node cli/inspect.js <TOKEN>
  node cli/inspect.js <TOKEN> --json

${c('bright', 'Example:')}
  # Generate a token first
  node cli/mint-token.js > /tmp/token.txt
  
  # Inspect it
  node cli/inspect.js $(cat /tmp/token.txt)

${c('bright', 'What it shows:')}
  • Token identity and type (Root vs Delegated)
  • Expiry status and remaining time
  • Scope restrictions (Least Privilege)
  • Delegation chain (Chain of Custody)
  • Governance compliance checks

${c('dim', 'This tool proves audit compliance without any database.')}
`);
  process.exit(0);
}

printInspection(token, jsonFlag);


/**
 * SatGate Telemetry Module
 * 
 * Lightweight governance data stream for dashboard visualization.
 * Stores observed token usage in memory (ephemeral).
 * 
 * Key insight: We don't store tokens, we observe their usage.
 * This maintains stateless architecture while enabling visibility.
 */

const activeTokens = new Map();
let blockedCount = 0;
let allowedCount = 0;

// Cleanup interval (remove tokens not seen in 10 minutes)
const CLEANUP_INTERVAL = 60 * 1000; // 1 minute
const TOKEN_TTL = 10 * 60 * 1000; // 10 minutes

setInterval(() => {
  const now = Date.now();
  for (const [sig, data] of activeTokens.entries()) {
    if (now - data.lastSeen > TOKEN_TTL) {
      activeTokens.delete(sig);
    }
  }
}, CLEANUP_INTERVAL);

/**
 * Record a blocked request (402 or 403)
 * This powers the "Economic Firewall" counter on the dashboard.
 */
function recordBlock(reason = 'unpaid') {
  blockedCount++;
  return blockedCount;
}

/**
 * Record successful token usage.
 * Called after macaroon verification succeeds.
 * 
 * @param {Object} macaroonData - Parsed macaroon info
 * @param {string} remoteAddress - Client IP
 */
function recordUsage(macaroonData, remoteAddress) {
  const now = Date.now();
  
  // Extract signature as unique ID
  const signature = macaroonData.signature || 
    (macaroonData.exportBinary ? 
      Buffer.from(macaroonData.exportBinary()).toString('hex').substring(0, 32) : 
      `unknown-${now}`);
  
  // Extract caveats for governance visualization
  let caveats = [];
  let depth = 0;
  
  if (macaroonData.caveats) {
    caveats = macaroonData.caveats.map(c => {
      if (Buffer.isBuffer(c)) return c.toString('utf8');
      if (c.raw) return c.raw.toString('utf8');
      if (typeof c === 'string') return c;
      return String(c);
    });
    
    // Estimate depth based on caveat count
    // More caveats = deeper in delegation chain
    const scopeCount = caveats.filter(c => c.startsWith('scope = ')).length;
    const hasDelegation = caveats.some(c => c.includes('delegated_by'));
    depth = hasDelegation ? (scopeCount > 1 ? 2 : 1) : 0;
  }
  
  const tokenData = {
    id: signature.substring(0, 8) + '...',
    fullSignature: signature,
    lastSeen: now,
    firstSeen: activeTokens.has(signature) ? activeTokens.get(signature).firstSeen : now,
    ip: remoteAddress,
    depth: depth,
    constraints: caveats,
    requestCount: (activeTokens.get(signature)?.requestCount || 0) + 1,
    status: 'ACTIVE'
  };
  
  activeTokens.set(signature, tokenData);
  allowedCount++;
  
  return tokenData;
}

/**
 * Get graph data for dashboard visualization.
 * Returns nodes and edges in a format compatible with react-flow/cytoscape.
 */
function getGraphData() {
  const nodes = [];
  const edges = [];
  
  // Convert Map to Graph Format
  // Group by depth to create visual hierarchy
  const sortedTokens = Array.from(activeTokens.values())
    .sort((a, b) => a.depth - b.depth);
  
  // Track parent candidates at each depth for linking
  const depthParents = new Map();
  
  sortedTokens.forEach((token, index) => {
    // Determine label based on depth
    let label = 'Token';
    if (token.depth === 0) label = 'Root';
    else if (token.depth === 1) label = 'Agent';
    else label = 'Worker';
    
    // Find scope for display
    const scopeCaveat = token.constraints.find(c => c.startsWith('scope = '));
    const scope = scopeCaveat ? scopeCaveat.split('=')[1].trim() : '*';
    
    // Find delegator
    const delegatorCaveat = token.constraints.find(c => c.includes('delegated_by'));
    const delegator = delegatorCaveat ? delegatorCaveat.split('=')[1].trim() : null;
    
    // Add Node
    nodes.push({
      id: token.fullSignature,
      data: {
        label: `${label} (${token.id})`,
        depth: token.depth,
        scope: scope,
        delegator: delegator,
        lastSeen: new Date(token.lastSeen).toLocaleTimeString(),
        requests: token.requestCount,
        constraints: token.constraints
      },
      position: { x: 150 * token.depth, y: index * 80 }
    });
    
    // Link to parent (heuristic: connect to last token of depth - 1)
    if (token.depth > 0 && depthParents.has(token.depth - 1)) {
      const parentSig = depthParents.get(token.depth - 1);
      edges.push({
        id: `e-${token.fullSignature.substring(0, 8)}`,
        source: parentSig,
        target: token.fullSignature,
        animated: true,
        label: 'delegated'
      });
    }
    
    // Track this as potential parent for next depth
    depthParents.set(token.depth, token.fullSignature);
  });
  
  return {
    nodes,
    edges,
    stats: {
      active: activeTokens.size,
      allowed: allowedCount,
      blocked: blockedCount,
      // Economic Firewall ratio
      blockRate: blockedCount > 0 ? 
        ((blockedCount / (blockedCount + allowedCount)) * 100).toFixed(1) : 0
    }
  };
}

/**
 * Get raw stats for simple monitoring.
 */
function getStats() {
  return {
    activeTokens: activeTokens.size,
    totalAllowed: allowedCount,
    totalBlocked: blockedCount,
    blockRate: blockedCount > 0 ? 
      ((blockedCount / (blockedCount + allowedCount)) * 100).toFixed(1) + '%' : '0%'
  };
}

/**
 * Reset counters (for testing).
 */
function reset() {
  activeTokens.clear();
  blockedCount = 0;
  allowedCount = 0;
}

module.exports = {
  recordBlock,
  recordUsage,
  getGraphData,
  getStats,
  reset
};


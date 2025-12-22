'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Shield, Activity, Zap, Clock, Lock, RefreshCw, AlertTriangle, CheckCircle, GitBranch, Ban, Wifi, WifiOff } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

// API endpoint - Railway backend
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'https://satgate-production.up.railway.app';

// Token type from API
interface TokenData {
  id: string;
  label: string;
  constraints: string[];
  lastSeen: string;
  depth: number;
  status: string;
}

interface GraphData {
  nodes: Array<{ data: TokenData }>;
  edges: Array<{ data: { source: string; target: string } }>;
  stats: {
    active: number;
    blocked: number;
    banned: number;
    bannedHits: number;
  };
}

// Mock data for demo when API unavailable
const mockGraphData: GraphData = {
  nodes: [
    { data: { id: 'root-001', label: 'Token (Depth 0)', constraints: ['scope = api:capability:*', 'expires = 24h'], lastSeen: '2s ago', depth: 0, status: 'ACTIVE' }},
    { data: { id: 'agent-001', label: 'Token (Depth 1)', constraints: ['scope = api:capability:read', 'expires = 1h'], lastSeen: '5s ago', depth: 1, status: 'ACTIVE' }},
    { data: { id: 'worker-001', label: 'Token (Depth 2)', constraints: ['scope = api:capability:ping', 'expires = 5m'], lastSeen: '1s ago', depth: 2, status: 'ACTIVE' }},
    { data: { id: 'worker-002', label: 'Token (Depth 2)', constraints: ['scope = api:capability:ping', 'expires = 3m'], lastSeen: '12s ago', depth: 2, status: 'ACTIVE' }},
  ],
  edges: [
    { data: { source: 'root-001', target: 'agent-001' }},
    { data: { source: 'agent-001', target: 'worker-001' }},
    { data: { source: 'agent-001', target: 'worker-002' }},
  ],
  stats: { active: 4, blocked: 15847, banned: 0, bannedHits: 0 }
};

export default function DashboardPage() {
  const [graphData, setGraphData] = useState<GraphData>(mockGraphData);
  const [selectedToken, setSelectedToken] = useState<TokenData | null>(null);
  const [isLive, setIsLive] = useState(true);
  const [isConnected, setIsConnected] = useState(false);
  const [lastFetch, setLastFetch] = useState<Date | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Fetch data from API
  const fetchGraphData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/governance/graph`, {
        cache: 'no-store',
        headers: { 'Accept': 'application/json' }
      });
      
      if (!response.ok) {
        throw new Error(`API returned ${response.status}`);
      }
      
      const data = await response.json();
      setGraphData(data);
      setIsConnected(true);
      setLastFetch(new Date());
      setError(null);
    } catch (err) {
      console.error('Failed to fetch governance data:', err);
      setIsConnected(false);
      setError(err instanceof Error ? err.message : 'Connection failed');
      // Keep showing last known data or mock data
    }
  }, []);

  // Initial fetch and polling
  useEffect(() => {
    fetchGraphData(); // Initial fetch
    
    if (!isLive) return;
    
    const interval = setInterval(fetchGraphData, 2000);
    return () => clearInterval(interval);
  }, [isLive, fetchGraphData]);

  // Derive display data from graph
  const tokens = graphData.nodes.map(n => {
    const data = n.data;
    const type = data.depth === 0 ? 'ROOT' : data.depth === 1 ? 'AGENT' : 'WORKER';
    const scope = data.constraints.find(c => c.startsWith('scope ='))?.replace('scope = ', '') || 'unknown';
    const expiresRaw = data.constraints.find(c => c.startsWith('expires ='))?.replace('expires = ', '');
    const expires = expiresRaw ? formatExpiry(expiresRaw) : 'No expiry';
    
    return {
      id: data.id,
      type,
      label: data.label,
      scope,
      expires,
      lastSeen: data.lastSeen,
      depth: data.depth,
      status: data.status,
      constraints: data.constraints,
      parent: graphData.edges.find(e => e.data.target === data.id)?.data.source
    };
  }).sort((a, b) => a.depth - b.depth);

  const stats = {
    activeTokens: graphData.stats.active,
    totalBlocked: graphData.stats.blocked,
    bannedTokens: graphData.stats.banned,
    bannedHits: graphData.stats.bannedHits,
  };

  // Calculate block rate
  const totalRequests = stats.activeTokens + stats.totalBlocked;
  const blockRate = totalRequests > 0 ? ((stats.totalBlocked / totalRequests) * 100).toFixed(1) : '0.0';

  function formatExpiry(expiry: string): string {
    // If it's a timestamp, convert to relative time
    const ts = parseInt(expiry);
    if (!isNaN(ts)) {
      const diff = ts - Date.now();
      if (diff < 0) return 'Expired';
      if (diff < 60000) return `${Math.floor(diff / 1000)}s`;
      if (diff < 3600000) return `${Math.floor(diff / 60000)}m`;
      return `${Math.floor(diff / 3600000)}h`;
    }
    return expiry;
  }

  const getTokenColor = (type: string, status: string) => {
    if (status === 'BANNED') return 'text-red-400 bg-red-900/30 border-red-700/50';
    switch (type) {
      case 'ROOT': return 'text-yellow-400 bg-yellow-900/30 border-yellow-700/50';
      case 'AGENT': return 'text-purple-400 bg-purple-900/30 border-purple-700/50';
      case 'WORKER': return 'text-cyan-400 bg-cyan-900/30 border-cyan-700/50';
      default: return 'text-gray-400 bg-gray-900/30 border-gray-700/50';
    }
  };

  const getTypeIcon = (type: string, status: string) => {
    if (status === 'BANNED') return '🚫';
    switch (type) {
      case 'ROOT': return '👑';
      case 'AGENT': return '🤖';
      case 'WORKER': return '⚙️';
      default: return '🔒';
    }
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      {/* Navigation */}
      <nav className="border-b border-gray-800 backdrop-blur-md fixed w-full z-50 bg-black/80">
        <div className="max-w-7xl mx-auto px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="w-8 h-8" />
            <span className="text-xl font-bold text-white">SatGate<sup className="text-xs font-normal">™</sup></span>
          </Link>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2 text-sm">
              {isConnected ? (
                <Wifi size={14} className="text-green-500" />
              ) : (
                <WifiOff size={14} className="text-red-500" />
              )}
              <div className={`w-2 h-2 rounded-full ${isLive ? 'bg-green-500 animate-pulse' : 'bg-gray-500'}`} />
              <span className="text-gray-400">{isLive ? 'LIVE' : 'PAUSED'}</span>
            </div>
            <button 
              onClick={() => setIsLive(!isLive)}
              className="text-sm text-gray-400 hover:text-white transition"
            >
              {isLive ? 'Pause' : 'Resume'}
            </button>
          </div>
        </div>
      </nav>

      <main className="pt-24 pb-12 px-6">
        <div className="max-w-7xl mx-auto">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-3xl font-bold mb-2">Governance Dashboard</h1>
            <p className="text-gray-500">Real-time visibility into your agent workforce and economic firewall.</p>
            {lastFetch && (
              <p className="text-xs text-gray-600 mt-1">
                Last updated: {lastFetch.toLocaleTimeString()} 
                {!isConnected && error && <span className="text-red-400 ml-2">• {error}</span>}
              </p>
            )}
          </div>

          {/* Stats Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <div className="flex items-center justify-between mb-2">
                <span className="text-gray-500 text-sm">Active Tokens</span>
                <Activity size={16} className="text-green-400" />
              </div>
              <div className="text-2xl font-bold text-white">{stats.activeTokens}</div>
              <div className="text-xs text-gray-600 mt-1">Authorized agents</div>
            </div>
            
            <div className="p-5 rounded-xl bg-gray-900 border border-red-900/30">
              <div className="flex items-center justify-between mb-2">
                <span className="text-gray-500 text-sm">🛡️ Economic Firewall</span>
                <Shield size={16} className="text-red-400" />
              </div>
              <div className="text-2xl font-bold text-red-400">{stats.totalBlocked.toLocaleString()}</div>
              <div className="text-xs text-gray-600 mt-1">Unpaid requests blocked</div>
            </div>
            
            <div className="p-5 rounded-xl bg-gray-900 border border-orange-900/30">
              <div className="flex items-center justify-between mb-2">
                <span className="text-gray-500 text-sm">🚨 Kill Switch Hits</span>
                <Ban size={16} className="text-orange-400" />
              </div>
              <div className="text-2xl font-bold text-orange-400">{stats.bannedHits}</div>
              <div className="text-xs text-gray-600 mt-1">Revoked tokens blocked</div>
            </div>
            
            <div className="p-5 rounded-xl bg-gradient-to-br from-red-950/30 to-orange-950/30 border border-red-800/30">
              <div className="flex items-center justify-between mb-2">
                <span className="text-gray-500 text-sm">Block Rate</span>
                <AlertTriangle size={16} className="text-orange-400" />
              </div>
              <div className="text-2xl font-bold text-orange-400">{blockRate}%</div>
              <div className="text-xs text-gray-600 mt-1">Attacks bankrupted</div>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Token Tree */}
            <div className="lg:col-span-2 p-6 rounded-xl bg-gray-900 border border-gray-800">
              <div className="flex items-center justify-between mb-6">
                <h2 className="font-bold text-lg flex items-center gap-2">
                  <GitBranch size={18} className="text-purple-400" />
                  Token Lineage (Chain of Custody)
                </h2>
                <button 
                  onClick={fetchGraphData}
                  className="text-sm text-gray-500 hover:text-white transition flex items-center gap-1"
                >
                  <RefreshCw size={14} />
                  Refresh
                </button>
              </div>
              
              {/* Visual Tree */}
              {tokens.length === 0 ? (
                <div className="text-center py-12 text-gray-600">
                  <Activity size={32} className="mx-auto mb-3 opacity-50 animate-pulse" />
                  <p className="text-sm">No tokens observed yet</p>
                  <p className="text-xs text-gray-700 mt-1">Make requests to see them appear here</p>
                </div>
              ) : (
                <div className="space-y-2">
                  {tokens.map((token) => (
                    <div 
                      key={token.id}
                      onClick={() => setSelectedToken({
                        id: token.id,
                        label: token.label,
                        constraints: token.constraints,
                        lastSeen: token.lastSeen,
                        depth: token.depth,
                        status: token.status
                      })}
                      className={`
                        flex items-center gap-3 p-4 rounded-lg cursor-pointer transition border
                        ${selectedToken?.id === token.id ? 'ring-2 ring-purple-500' : ''}
                        ${getTokenColor(token.type, token.status)}
                      `}
                      style={{ marginLeft: `${token.depth * 40}px` }}
                    >
                      {token.depth > 0 && (
                        <div className="text-gray-600 text-sm">└──</div>
                      )}
                      <span className="text-lg">{getTypeIcon(token.type, token.status)}</span>
                      <div className="flex-1">
                        <div className="font-medium flex items-center gap-2">
                          {token.type} Token
                          {token.status === 'BANNED' && (
                            <span className="text-xs bg-red-800 text-red-200 px-2 py-0.5 rounded">BANNED</span>
                          )}
                        </div>
                        <div className="text-xs text-gray-500">
                          Scope: {token.scope} • Expires: {token.expires}
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="text-xs text-gray-500">{token.lastSeen}</div>
                        <div className="text-xs font-mono text-gray-600">{token.id.substring(0, 12)}...</div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
              
              <div className="mt-6 pt-4 border-t border-gray-800 text-center">
                <p className="text-xs text-gray-600">
                  This tree builds itself from observed traffic. No central token database—cryptographic provenance.
                </p>
              </div>
            </div>

            {/* Token Inspector */}
            <div className="p-6 rounded-xl bg-gray-900 border border-gray-800">
              <h2 className="font-bold text-lg mb-4 flex items-center gap-2">
                <Lock size={18} className="text-cyan-400" />
                Token Inspector
              </h2>
              
              {selectedToken ? (
                <div className="space-y-4">
                  <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700">
                    <div className="flex items-center gap-2 mb-3">
                      <span className="text-2xl">
                        {getTypeIcon(
                          selectedToken.depth === 0 ? 'ROOT' : selectedToken.depth === 1 ? 'AGENT' : 'WORKER',
                          selectedToken.status
                        )}
                      </span>
                      <div>
                        <div className="font-bold">Depth {selectedToken.depth} Token</div>
                        <div className={`text-xs ${selectedToken.status === 'BANNED' ? 'text-red-400' : 'text-gray-500'}`}>
                          {selectedToken.status}
                        </div>
                      </div>
                    </div>
                    
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-gray-500">ID</span>
                        <span className="font-mono text-gray-300 text-xs">{selectedToken.id.substring(0, 20)}...</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-500">Last Seen</span>
                        <span className="text-gray-300">{selectedToken.lastSeen}</span>
                      </div>
                    </div>
                  </div>
                  
                  {/* Constraints */}
                  <div>
                    <h4 className="text-sm font-semibold text-gray-400 mb-2">CAVEATS</h4>
                    <div className="space-y-1">
                      {selectedToken.constraints.map((c, i) => (
                        <div key={i} className="text-xs font-mono bg-gray-800 p-2 rounded text-cyan-400">
                          {c}
                        </div>
                      ))}
                    </div>
                  </div>
                  
                  {/* Governance Checks */}
                  <div>
                    <h4 className="text-sm font-semibold text-gray-400 mb-2">GOVERNANCE CHECKS</h4>
                    <div className="space-y-2">
                      <div className="flex items-center gap-2 text-sm">
                        <CheckCircle size={14} className="text-green-400" />
                        <span className="text-gray-300">Least Privilege</span>
                        <span className="text-gray-500 text-xs ml-auto">Scope constrained</span>
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                        <CheckCircle size={14} className="text-green-400" />
                        <span className="text-gray-300">Ephemeral</span>
                        <span className="text-gray-500 text-xs ml-auto">Short TTL</span>
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                        <CheckCircle size={14} className="text-green-400" />
                        <span className="text-gray-300">Traceable</span>
                        <span className="text-gray-500 text-xs ml-auto">Crypto signature</span>
                      </div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-center py-12 text-gray-600">
                  <Lock size={32} className="mx-auto mb-3 opacity-50" />
                  <p className="text-sm">Click a token to inspect</p>
                </div>
              )}
            </div>
          </div>

          {/* Connection Status */}
          <div className={`mt-8 p-4 rounded-xl text-center ${isConnected ? 'bg-green-900/20 border border-green-800/30' : 'bg-purple-900/20 border border-purple-800/30'}`}>
            <p className="text-sm text-gray-400">
              {isConnected ? (
                <>
                  <span className="text-green-400 font-medium">🔗 Connected:</span> Live data from <code className="text-green-300">{API_BASE}/api/governance/graph</code>
                </>
              ) : (
                <>
                  <span className="text-purple-400 font-medium">Demo Mode:</span> Showing sample data. 
                  Connect to <code className="text-purple-300">/api/governance/graph</code> for live telemetry.
                </>
              )}
            </p>
          </div>
        </div>
      </main>
    </div>
  );
}

'use client';

import React, { useState, useEffect } from 'react';
import { Shield, Activity, Zap, Clock, Lock, RefreshCw, AlertTriangle, CheckCircle, GitBranch } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

// Mock data for demo - in production this would come from /api/governance/graph
const mockTokens = [
  {
    id: 'root-001',
    type: 'ROOT',
    label: 'CI Pipeline',
    scope: 'api:capability:*',
    expires: '24 hours',
    requests: 1247,
    lastSeen: '2s ago',
    depth: 0,
  },
  {
    id: 'agent-001',
    type: 'AGENT',
    label: 'Data Agent',
    scope: 'api:capability:read',
    expires: '1 hour',
    requests: 892,
    lastSeen: '5s ago',
    depth: 1,
    parent: 'root-001',
  },
  {
    id: 'worker-001',
    type: 'WORKER',
    label: 'Task Worker A',
    scope: 'api:capability:ping',
    expires: '5 min',
    requests: 156,
    lastSeen: '1s ago',
    depth: 2,
    parent: 'agent-001',
  },
  {
    id: 'worker-002',
    type: 'WORKER',
    label: 'Task Worker B',
    scope: 'api:capability:ping',
    expires: '3 min',
    requests: 89,
    lastSeen: '12s ago',
    depth: 2,
    parent: 'agent-001',
  },
];

const mockStats = {
  activeTokens: 4,
  totalAllowed: 2384,
  totalBlocked: 15847,
  blockRate: '86.9%',
};

export default function DashboardPage() {
  const [tokens, setTokens] = useState(mockTokens);
  const [stats, setStats] = useState(mockStats);
  const [selectedToken, setSelectedToken] = useState<typeof mockTokens[0] | null>(null);
  const [isLive, setIsLive] = useState(true);

  // Simulate live updates
  useEffect(() => {
    if (!isLive) return;
    
    const interval = setInterval(() => {
      setStats(prev => ({
        ...prev,
        totalAllowed: prev.totalAllowed + Math.floor(Math.random() * 5),
        totalBlocked: prev.totalBlocked + Math.floor(Math.random() * 20),
      }));
      
      setTokens(prev => prev.map(t => ({
        ...t,
        requests: t.requests + Math.floor(Math.random() * 3),
        lastSeen: Math.random() > 0.5 ? '1s ago' : t.lastSeen,
      })));
    }, 2000);
    
    return () => clearInterval(interval);
  }, [isLive]);

  const getTokenColor = (type: string) => {
    switch (type) {
      case 'ROOT': return 'text-yellow-400 bg-yellow-900/30 border-yellow-700/50';
      case 'AGENT': return 'text-purple-400 bg-purple-900/30 border-purple-700/50';
      case 'WORKER': return 'text-cyan-400 bg-cyan-900/30 border-cyan-700/50';
      default: return 'text-gray-400 bg-gray-900/30 border-gray-700/50';
    }
  };

  const getTypeIcon = (type: string) => {
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
            
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <div className="flex items-center justify-between mb-2">
                <span className="text-gray-500 text-sm">Allowed Requests</span>
                <CheckCircle size={16} className="text-green-400" />
              </div>
              <div className="text-2xl font-bold text-green-400">{stats.totalAllowed.toLocaleString()}</div>
              <div className="text-xs text-gray-600 mt-1">Valid tokens</div>
            </div>
            
            <div className="p-5 rounded-xl bg-gray-900 border border-red-900/30">
              <div className="flex items-center justify-between mb-2">
                <span className="text-gray-500 text-sm">Blocked (Economic Firewall)</span>
                <Shield size={16} className="text-red-400" />
              </div>
              <div className="text-2xl font-bold text-red-400">{stats.totalBlocked.toLocaleString()}</div>
              <div className="text-xs text-gray-600 mt-1">Unpaid / Invalid</div>
            </div>
            
            <div className="p-5 rounded-xl bg-gradient-to-br from-red-950/30 to-orange-950/30 border border-red-800/30">
              <div className="flex items-center justify-between mb-2">
                <span className="text-gray-500 text-sm">Block Rate</span>
                <AlertTriangle size={16} className="text-orange-400" />
              </div>
              <div className="text-2xl font-bold text-orange-400">{stats.blockRate}</div>
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
                <button className="text-sm text-gray-500 hover:text-white transition flex items-center gap-1">
                  <RefreshCw size={14} />
                  Refresh
                </button>
              </div>
              
              {/* Visual Tree */}
              <div className="space-y-2">
                {tokens.map((token, idx) => (
                  <div 
                    key={token.id}
                    onClick={() => setSelectedToken(token)}
                    className={`
                      flex items-center gap-3 p-4 rounded-lg cursor-pointer transition
                      ${selectedToken?.id === token.id ? 'ring-2 ring-purple-500' : ''}
                      ${getTokenColor(token.type)}
                    `}
                    style={{ marginLeft: `${token.depth * 40}px` }}
                  >
                    {token.depth > 0 && (
                      <div className="text-gray-600 text-sm">└──</div>
                    )}
                    <span className="text-lg">{getTypeIcon(token.type)}</span>
                    <div className="flex-1">
                      <div className="font-medium">{token.label}</div>
                      <div className="text-xs text-gray-500">
                        Scope: {token.scope} • Expires: {token.expires}
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="text-sm font-mono">{token.requests} reqs</div>
                      <div className="text-xs text-gray-500">{token.lastSeen}</div>
                    </div>
                  </div>
                ))}
              </div>
              
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
                      <span className="text-2xl">{getTypeIcon(selectedToken.type)}</span>
                      <div>
                        <div className="font-bold">{selectedToken.label}</div>
                        <div className="text-xs text-gray-500">{selectedToken.type} Token</div>
                      </div>
                    </div>
                    
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-gray-500">ID</span>
                        <span className="font-mono text-gray-300">{selectedToken.id}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-500">Scope</span>
                        <span className="font-mono text-cyan-400">{selectedToken.scope}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-500">Expires</span>
                        <span className="text-yellow-400">{selectedToken.expires}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-500">Requests</span>
                        <span className="text-green-400">{selectedToken.requests}</span>
                      </div>
                    </div>
                  </div>
                  
                  {/* Governance Checks */}
                  <div>
                    <h4 className="text-sm font-semibold text-gray-400 mb-2">GOVERNANCE CHECKS</h4>
                    <div className="space-y-2">
                      <div className="flex items-center gap-2 text-sm">
                        <CheckCircle size={14} className="text-green-400" />
                        <span className="text-gray-300">Least Privilege</span>
                        <span className="text-gray-500 text-xs ml-auto">Scope narrowed</span>
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                        <CheckCircle size={14} className="text-green-400" />
                        <span className="text-gray-300">Ephemeral</span>
                        <span className="text-gray-500 text-xs ml-auto">Short TTL</span>
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                        <CheckCircle size={14} className="text-green-400" />
                        <span className="text-gray-300">Traceable</span>
                        <span className="text-gray-500 text-xs ml-auto">Linked to root</span>
                      </div>
                    </div>
                  </div>
                  
                  {selectedToken.parent && (
                    <div className="pt-4 border-t border-gray-800">
                      <h4 className="text-sm font-semibold text-gray-400 mb-2">DELEGATED BY</h4>
                      <div className="text-sm text-purple-400 font-mono">
                        {tokens.find(t => t.id === selectedToken.parent)?.label || selectedToken.parent}
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                <div className="text-center py-12 text-gray-600">
                  <Lock size={32} className="mx-auto mb-3 opacity-50" />
                  <p className="text-sm">Click a token to inspect</p>
                </div>
              )}
            </div>
          </div>

          {/* Demo Notice */}
          <div className="mt-8 p-4 rounded-xl bg-purple-900/20 border border-purple-800/30 text-center">
            <p className="text-sm text-gray-400">
              <span className="text-purple-400 font-medium">Demo Mode:</span> This dashboard shows simulated data. 
              In production, connect to <code className="text-purple-300">/api/governance/graph</code> for live telemetry.
            </p>
          </div>
        </div>
      </main>
    </div>
  );
}


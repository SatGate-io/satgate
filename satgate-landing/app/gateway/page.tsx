'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { 
  Shield, Activity, Zap, Clock, Lock, RefreshCw, AlertTriangle, 
  CheckCircle, Settings, Server, Route, DollarSign, Key, Ban,
  Plus, Trash2, Edit, Eye, EyeOff, Copy, Check, ChevronRight,
  Upload, Download, Play, Pause, BarChart3, TrendingUp, Globe,
  Wifi, WifiOff, FileCode, Layers, Target, ArrowRight
} from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

// Gateway API endpoint
const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'https://satgate-production-4c3c.up.railway.app';

interface GatewayStats {
  status: string;
  uptime: string;
  totalRequests: number;
  totalRevenue: number;
  activeTokens: number;
  bannedTokens: number;
  routes: number;
  upstreams: number;
}

interface RouteConfig {
  name: string;
  pathPrefix: string;
  upstream: string;
  policy: {
    kind: string;
    tier?: string;
    priceSats?: number;
    scope?: string;
  };
}

interface UpstreamConfig {
  name: string;
  url: string;
  status: 'healthy' | 'degraded' | 'down';
}

export default function GatewayDashboard() {
  const [activeTab, setActiveTab] = useState<'overview' | 'routes' | 'config' | 'tokens'>('overview');
  const [isConnected, setIsConnected] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [adminToken, setAdminToken] = useState('');
  const [showAdminInput, setShowAdminInput] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  
  // Gateway data
  const [stats, setStats] = useState<GatewayStats>({
    status: 'unknown',
    uptime: '-',
    totalRequests: 0,
    totalRevenue: 0,
    activeTokens: 0,
    bannedTokens: 0,
    routes: 0,
    upstreams: 0
  });
  
  const [routes, setRoutes] = useState<RouteConfig[]>([]);
  const [upstreams, setUpstreams] = useState<UpstreamConfig[]>([]);
  const [configYaml, setConfigYaml] = useState('');

  // Fetch gateway health
  const fetchHealth = useCallback(async () => {
    try {
      const response = await fetch(`${GATEWAY_URL}/healthz`);
      const data = await response.json();
      setIsConnected(data.status === 'ok');
      setStats(prev => ({ ...prev, status: data.status }));
    } catch (err) {
      setIsConnected(false);
      setStats(prev => ({ ...prev, status: 'unreachable' }));
    }
  }, []);

  // Fetch gateway stats (requires admin)
  const fetchStats = useCallback(async () => {
    if (!adminToken) return;
    
    try {
      const response = await fetch(`${GATEWAY_URL}/api/governance/graph`, {
        headers: { 'X-Admin-Token': adminToken }
      });
      
      if (response.ok) {
        const data = await response.json();
        setStats(prev => ({
          ...prev,
          activeTokens: data.stats?.active || 0,
          bannedTokens: data.stats?.banned || 0,
          totalRequests: data.stats?.blocked || 0,
        }));
      }
    } catch (err) {
      console.error('Failed to fetch stats:', err);
    }
  }, [adminToken]);

  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 10000);
    return () => clearInterval(interval);
  }, [fetchHealth]);

  useEffect(() => {
    if (adminToken) {
      fetchStats();
    }
    setIsLoading(false);
  }, [adminToken, fetchStats]);

  // Load admin token from localStorage
  useEffect(() => {
    const saved = localStorage.getItem('gateway_admin_token');
    if (saved) setAdminToken(saved);
  }, []);

  const saveAdminToken = (token: string) => {
    setAdminToken(token);
    localStorage.setItem('gateway_admin_token', token);
    setShowAdminInput(false);
  };

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 2000);
  };

  // Demo routes for display
  const demoRoutes: RouteConfig[] = [
    { name: 'premium', pathPrefix: '/v1/premium/', upstream: 'test_api', policy: { kind: 'l402', tier: 'premium', priceSats: 1000, scope: 'api:premium:*' }},
    { name: 'standard', pathPrefix: '/v1/standard/', upstream: 'test_api', policy: { kind: 'l402', tier: 'standard', priceSats: 100, scope: 'api:standard:*' }},
    { name: 'basic', pathPrefix: '/v1/basic/', upstream: 'test_api', policy: { kind: 'l402', tier: 'basic', priceSats: 10, scope: 'api:basic:*' }},
    { name: 'micro', pathPrefix: '/v1/micro/', upstream: 'test_api', policy: { kind: 'l402', tier: 'micro', priceSats: 1, scope: 'api:micro:*' }},
    { name: 'capability', pathPrefix: '/v1/capability/', upstream: 'test_api', policy: { kind: 'capability', scope: 'api:capability:*' }},
    { name: 'default-deny', pathPrefix: '/', upstream: '', policy: { kind: 'deny' }},
  ];

  const demoUpstreams: UpstreamConfig[] = [
    { name: 'test_api', url: 'https://httpbin.org', status: 'healthy' },
    { name: 'health_backend', url: 'https://httpbin.org', status: 'healthy' },
  ];

  const getPolicyColor = (kind: string) => {
    switch (kind) {
      case 'l402': return 'text-yellow-400 bg-yellow-900/30 border-yellow-700/50';
      case 'capability': return 'text-cyan-400 bg-cyan-900/30 border-cyan-700/50';
      case 'public': return 'text-green-400 bg-green-900/30 border-green-700/50';
      case 'deny': return 'text-red-400 bg-red-900/30 border-red-700/50';
      default: return 'text-gray-400 bg-gray-900/30 border-gray-700/50';
    }
  };

  const getPolicyIcon = (kind: string) => {
    switch (kind) {
      case 'l402': return <Zap size={14} />;
      case 'capability': return <Key size={14} />;
      case 'public': return <Globe size={14} />;
      case 'deny': return <Ban size={14} />;
      default: return <Lock size={14} />;
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
            <span className="text-gray-500 text-sm ml-2">Gateway</span>
          </Link>
          
          <div className="flex items-center gap-4">
            {/* Connection Status */}
            <div className="flex items-center gap-2 text-sm">
              {isConnected ? (
                <Wifi size={14} className="text-green-500" />
              ) : (
                <WifiOff size={14} className="text-red-500" />
              )}
              <span className={isConnected ? 'text-green-400' : 'text-red-400'}>
                {isConnected ? 'Connected' : 'Disconnected'}
              </span>
            </div>
            
            {/* Admin Token */}
            {!adminToken ? (
              <button
                onClick={() => setShowAdminInput(true)}
                className="text-sm text-gray-400 hover:text-white transition flex items-center gap-1"
              >
                <Key size={14} />
                Set Admin Token
              </button>
            ) : (
              <div className="flex items-center gap-2">
                <span className="text-xs text-green-400">Admin ✓</span>
                <button
                  onClick={() => { setAdminToken(''); localStorage.removeItem('gateway_admin_token'); }}
                  className="text-xs text-gray-500 hover:text-red-400"
                >
                  Logout
                </button>
              </div>
            )}
          </div>
        </div>
      </nav>

      {/* Admin Token Modal */}
      {showAdminInput && (
        <div className="fixed inset-0 bg-black/80 z-50 flex items-center justify-center">
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 w-full max-w-md">
            <h3 className="text-lg font-bold mb-4">Enter Admin Token</h3>
            <input
              type="password"
              placeholder="Your gateway admin token"
              className="w-full px-4 py-3 bg-black border border-gray-700 rounded-lg text-white mb-4"
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  saveAdminToken((e.target as HTMLInputElement).value);
                }
              }}
            />
            <div className="flex gap-2">
              <button
                onClick={() => setShowAdminInput(false)}
                className="flex-1 py-2 bg-gray-800 rounded-lg text-gray-400 hover:text-white transition"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  const input = document.querySelector('input[type="password"]') as HTMLInputElement;
                  if (input) saveAdminToken(input.value);
                }}
                className="flex-1 py-2 bg-purple-600 rounded-lg text-white hover:bg-purple-500 transition"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}

      <main className="pt-24 pb-12 px-6">
        <div className="max-w-7xl mx-auto">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-3xl font-bold mb-2">Gateway Dashboard</h1>
            <p className="text-gray-500">Manage your SatGate Gateway deployment, routes, and tokens.</p>
            <p className="text-xs text-gray-600 mt-1 font-mono">{GATEWAY_URL}</p>
          </div>

          {/* Tabs */}
          <div className="flex gap-2 mb-8 border-b border-gray-800 pb-4">
            {[
              { id: 'overview', label: 'Overview', icon: <BarChart3 size={16} /> },
              { id: 'routes', label: 'Routes', icon: <Route size={16} /> },
              { id: 'config', label: 'Configuration', icon: <FileCode size={16} /> },
              { id: 'tokens', label: 'Tokens', icon: <Key size={16} /> },
            ].map(tab => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as any)}
                className={`px-4 py-2 rounded-lg flex items-center gap-2 transition ${
                  activeTab === tab.id 
                    ? 'bg-purple-600 text-white' 
                    : 'bg-gray-900 text-gray-400 hover:text-white hover:bg-gray-800'
                }`}
              >
                {tab.icon}
                {tab.label}
              </button>
            ))}
          </div>

          {/* Overview Tab */}
          {activeTab === 'overview' && (
            <div className="space-y-6">
              {/* Stats Cards */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-gray-500 text-sm">Status</span>
                    <Server size={16} className={isConnected ? 'text-green-400' : 'text-red-400'} />
                  </div>
                  <div className={`text-2xl font-bold ${isConnected ? 'text-green-400' : 'text-red-400'}`}>
                    {isConnected ? 'Online' : 'Offline'}
                  </div>
                  <div className="text-xs text-gray-600 mt-1">Data Plane</div>
                </div>
                
                <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-gray-500 text-sm">Active Tokens</span>
                    <Key size={16} className="text-cyan-400" />
                  </div>
                  <div className="text-2xl font-bold text-white">{stats.activeTokens}</div>
                  <div className="text-xs text-gray-600 mt-1">Capability tokens</div>
                </div>
                
                <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-gray-500 text-sm">Routes</span>
                    <Route size={16} className="text-purple-400" />
                  </div>
                  <div className="text-2xl font-bold text-white">{demoRoutes.length}</div>
                  <div className="text-xs text-gray-600 mt-1">Configured paths</div>
                </div>
                
                <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-gray-500 text-sm">Upstreams</span>
                    <Globe size={16} className="text-yellow-400" />
                  </div>
                  <div className="text-2xl font-bold text-white">{demoUpstreams.length}</div>
                  <div className="text-xs text-gray-600 mt-1">Backend targets</div>
                </div>
              </div>

              {/* Quick Actions */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <button 
                  onClick={() => setActiveTab('routes')}
                  className="p-6 rounded-xl bg-gradient-to-br from-purple-900/30 to-purple-950/30 border border-purple-800/30 hover:border-purple-600/50 transition group"
                >
                  <Route className="text-purple-400 mb-3" size={24} />
                  <h3 className="font-bold text-white mb-1">Configure Routes</h3>
                  <p className="text-sm text-gray-500">Define pricing tiers and access policies</p>
                  <div className="mt-3 text-purple-400 text-sm flex items-center gap-1 opacity-0 group-hover:opacity-100 transition">
                    Open <ArrowRight size={14} />
                  </div>
                </button>
                
                <button
                  onClick={() => setActiveTab('tokens')}
                  className="p-6 rounded-xl bg-gradient-to-br from-cyan-900/30 to-cyan-950/30 border border-cyan-800/30 hover:border-cyan-600/50 transition group"
                >
                  <Key className="text-cyan-400 mb-3" size={24} />
                  <h3 className="font-bold text-white mb-1">Manage Tokens</h3>
                  <p className="text-sm text-gray-500">Mint, view, and revoke capability tokens</p>
                  <div className="mt-3 text-cyan-400 text-sm flex items-center gap-1 opacity-0 group-hover:opacity-100 transition">
                    Open <ArrowRight size={14} />
                  </div>
                </button>
                
                <Link
                  href="/dashboard"
                  className="p-6 rounded-xl bg-gradient-to-br from-yellow-900/30 to-yellow-950/30 border border-yellow-800/30 hover:border-yellow-600/50 transition group"
                >
                  <BarChart3 className="text-yellow-400 mb-3" size={24} />
                  <h3 className="font-bold text-white mb-1">View Telemetry</h3>
                  <p className="text-sm text-gray-500">Real-time token lineage and stats</p>
                  <div className="mt-3 text-yellow-400 text-sm flex items-center gap-1 opacity-0 group-hover:opacity-100 transition">
                    Open <ArrowRight size={14} />
                  </div>
                </Link>
              </div>

              {/* Test Endpoint */}
              <div className="p-6 rounded-xl bg-gray-900 border border-gray-800">
                <h3 className="font-bold mb-4 flex items-center gap-2">
                  <Play size={18} className="text-green-400" />
                  Test Gateway
                </h3>
                <div className="bg-black rounded-lg p-4 font-mono text-sm">
                  <div className="text-gray-500 mb-2"># Test L402 payment flow</div>
                  <div className="text-cyan-400">
                    curl -i {GATEWAY_URL}/v1/basic/anything
                  </div>
                  <div className="text-gray-500 mt-3 text-xs">
                    Expected: 402 Payment Required with Lightning invoice
                  </div>
                </div>
                <button
                  onClick={() => copyToClipboard(`curl -i ${GATEWAY_URL}/v1/basic/anything`, 'test-cmd')}
                  className="mt-3 text-sm text-gray-400 hover:text-white flex items-center gap-1 transition"
                >
                  {copied === 'test-cmd' ? <Check size={14} className="text-green-400" /> : <Copy size={14} />}
                  Copy command
                </button>
              </div>
            </div>
          )}

          {/* Routes Tab */}
          {activeTab === 'routes' && (
            <div className="space-y-6">
              <div className="flex justify-between items-center">
                <h2 className="text-xl font-bold">Route Configuration</h2>
                <button className="px-4 py-2 bg-purple-600 hover:bg-purple-500 rounded-lg text-sm flex items-center gap-2 transition">
                  <Plus size={16} />
                  Add Route
                </button>
              </div>
              
              <div className="space-y-3">
                {demoRoutes.map((route, i) => (
                  <div 
                    key={i}
                    className="p-4 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-700 transition"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-4">
                        <div className={`px-3 py-1 rounded-full text-xs font-medium border flex items-center gap-1 ${getPolicyColor(route.policy.kind)}`}>
                          {getPolicyIcon(route.policy.kind)}
                          {route.policy.kind.toUpperCase()}
                        </div>
                        <div>
                          <div className="font-semibold text-white">{route.name}</div>
                          <div className="text-sm text-gray-500 font-mono">{route.pathPrefix}</div>
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-6">
                        {route.policy.priceSats && (
                          <div className="text-right">
                            <div className="text-yellow-400 font-bold">{route.policy.priceSats} sats</div>
                            <div className="text-xs text-gray-500">{route.policy.tier}</div>
                          </div>
                        )}
                        <div className="flex gap-2">
                          <button className="p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded transition">
                            <Edit size={16} />
                          </button>
                          <button className="p-2 text-gray-500 hover:text-red-400 hover:bg-gray-800 rounded transition">
                            <Trash2 size={16} />
                          </button>
                        </div>
                      </div>
                    </div>
                    
                    {route.upstream && (
                      <div className="mt-3 pt-3 border-t border-gray-800 flex items-center gap-2 text-sm text-gray-500">
                        <ArrowRight size={14} />
                        <span>Upstream:</span>
                        <span className="text-cyan-400 font-mono">{route.upstream}</span>
                        {route.policy.scope && (
                          <>
                            <span className="mx-2">•</span>
                            <span>Scope:</span>
                            <span className="text-purple-400 font-mono">{route.policy.scope}</span>
                          </>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Config Tab */}
          {activeTab === 'config' && (
            <div className="space-y-6">
              <div className="flex justify-between items-center">
                <h2 className="text-xl font-bold">Gateway Configuration</h2>
                <div className="flex gap-2">
                  <button className="px-4 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-sm flex items-center gap-2 transition">
                    <Upload size={16} />
                    Import YAML
                  </button>
                  <button className="px-4 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-sm flex items-center gap-2 transition">
                    <Download size={16} />
                    Export
                  </button>
                </div>
              </div>
              
              {/* Upstreams */}
              <div className="p-6 rounded-xl bg-gray-900 border border-gray-800">
                <h3 className="font-bold mb-4 flex items-center gap-2">
                  <Globe size={18} className="text-yellow-400" />
                  Upstreams
                </h3>
                <div className="space-y-3">
                  {demoUpstreams.map((upstream, i) => (
                    <div key={i} className="flex items-center justify-between p-3 bg-black rounded-lg">
                      <div className="flex items-center gap-3">
                        <div className={`w-2 h-2 rounded-full ${upstream.status === 'healthy' ? 'bg-green-400' : 'bg-red-400'}`} />
                        <span className="font-medium text-white">{upstream.name}</span>
                        <span className="text-gray-500 font-mono text-sm">{upstream.url}</span>
                      </div>
                      <span className={`text-xs ${upstream.status === 'healthy' ? 'text-green-400' : 'text-red-400'}`}>
                        {upstream.status}
                      </span>
                    </div>
                  ))}
                </div>
                <button className="mt-4 text-sm text-purple-400 hover:text-purple-300 flex items-center gap-1 transition">
                  <Plus size={14} />
                  Add Upstream
                </button>
              </div>

              {/* YAML Preview */}
              <div className="p-6 rounded-xl bg-gray-900 border border-gray-800">
                <h3 className="font-bold mb-4 flex items-center gap-2">
                  <FileCode size={18} className="text-cyan-400" />
                  Configuration Preview
                </h3>
                <pre className="bg-black rounded-lg p-4 text-sm font-mono text-gray-300 overflow-x-auto">
{`version: 1

server:
  listen: "0.0.0.0:8080"

upstreams:
  test_api:
    url: "https://httpbin.org"

routes:
  - name: "premium"
    match:
      pathPrefix: "/v1/premium/"
    upstream: "test_api"
    policy:
      kind: "l402"
      priceSats: 1000
      tier: "premium"

  - name: "basic"
    match:
      pathPrefix: "/v1/basic/"
    upstream: "test_api"
    policy:
      kind: "l402"
      priceSats: 10
      tier: "basic"

  - name: "default-deny"
    match:
      pathPrefix: "/"
    policy:
      kind: "deny"
      status: 403`}
                </pre>
              </div>
            </div>
          )}

          {/* Tokens Tab */}
          {activeTab === 'tokens' && (
            <div className="space-y-6">
              <div className="flex justify-between items-center">
                <h2 className="text-xl font-bold">Token Management</h2>
                <button className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 rounded-lg text-sm flex items-center gap-2 transition">
                  <Key size={16} />
                  Mint Token
                </button>
              </div>
              
              {!adminToken ? (
                <div className="p-12 rounded-xl bg-gray-900 border border-gray-800 text-center">
                  <Key size={48} className="mx-auto text-gray-600 mb-4" />
                  <h3 className="text-lg font-bold text-white mb-2">Admin Access Required</h3>
                  <p className="text-gray-500 mb-4">Enter your admin token to manage gateway tokens.</p>
                  <button
                    onClick={() => setShowAdminInput(true)}
                    className="px-6 py-2 bg-purple-600 hover:bg-purple-500 rounded-lg transition"
                  >
                    Set Admin Token
                  </button>
                </div>
              ) : (
                <div className="p-6 rounded-xl bg-gray-900 border border-gray-800">
                  <p className="text-gray-500 text-center py-8">
                    Token management connected to Gateway API.<br />
                    Use the <Link href="/dashboard" className="text-purple-400 underline">Telemetry Dashboard</Link> for real-time token monitoring.
                  </p>
                </div>
              )}
            </div>
          )}

          {/* Footer */}
          <div className="mt-12 pt-8 border-t border-gray-800">
            <div className="flex justify-between items-center text-sm text-gray-600">
              <span>SatGate Gateway v2.0</span>
              <div className="flex gap-4">
                <Link href="/crawl" className="hover:text-white transition">Protect Demo</Link>
                <Link href="/monetize" className="hover:text-white transition">Monetize Demo</Link>
                <Link href="/dashboard" className="hover:text-white transition">Telemetry</Link>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}


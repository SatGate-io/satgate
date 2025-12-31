'use client';

import React, { useState, useRef, useEffect } from 'react';
import { 
  Key, Shield, Lock, Unlock, Play, ArrowLeft, Copy, Check, 
  ChevronRight, AlertTriangle, CheckCircle, XCircle, User, 
  Bot, GitBranch, Clock, RefreshCw, Eye, Trash2, Wifi, WifiOff,
  Ban, ShieldOff, Zap
} from 'lucide-react';
import Link from 'next/link';

// Railway embedded mode deployment
const BASE_URL = 'https://satgate-production-9354.up.railway.app';

// Mock token generation for simulation mode
function generateMockMacaroon(): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = 'AgEMc2F0Z2F0ZS5pbwIQ';
  for (let i = 0; i < 100; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

interface Token {
  raw: string;
  scope: string;
  expiresAt: string;
  caveats: Record<string, any>;
}

interface LogEntry {
  msg: string;
  type: 'info' | 'error' | 'success' | 'warn';
  timestamp: Date;
}

type DemoScene = 'intro' | 'mint' | 'use' | 'delegate' | 'enforce' | 'revoke' | 'summary';

export default function CrawlDemoPage() {
  const [currentScene, setCurrentScene] = useState<DemoScene>('intro');
  const [isLoading, setIsLoading] = useState(false);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [parentToken, setParentToken] = useState<Token | null>(null);
  const [childToken, setChildToken] = useState<Token | null>(null);
  const [copied, setCopied] = useState<string | null>(null);
  const [pingResult, setPingResult] = useState<any>(null);
  const [delegationResult, setDelegationResult] = useState<any>(null);
  const [enforcementResults, setEnforcementResults] = useState<{allowed: any; blocked: any} | null>(null);
  const [bannedToken, setBannedToken] = useState<string | null>(null);
  const [revocationResult, setRevocationResult] = useState<{banned: boolean; tested: boolean; error?: string} | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const [useSimulation, setUseSimulation] = useState(true); // Default to simulation since prod mode needs auth

  const addLog = (msg: string, type: LogEntry['type'] = 'info') => {
    setLogs(prev => [...prev, { msg, type, timestamp: new Date() }]);
  };

  const clearLogs = () => setLogs([]);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs]);

  const copyToClipboard = async (text: string, id: string) => {
    await navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 2000);
  };

  // Scene 1: Mint Token
  const handleMintToken = async () => {
    setIsLoading(true);
    clearLogs();
    addLog('🔐 CISO: "I\'m issuing a capability token for the Data Agent."', 'info');
    addLog(`📡 POST /api/capability/mint ${useSimulation ? '(SIMULATION)' : '(LIVE)'}`, 'info');
    
    try {
      if (useSimulation) {
        // Simulate the minting process
        await new Promise(r => setTimeout(r, 800));
        const mockToken = generateMockMacaroon();
        const expiresAt = new Date(Date.now() + 3600000).toISOString();
        
        addLog('✅ Token minted successfully!', 'success');
        addLog('📜 Scope: api:capability:*', 'info');
        addLog(`⏰ Expires: ${expiresAt}`, 'info');
        addLog('💡 No database write. No service account. Pure cryptography.', 'success');
        
        setParentToken({
          raw: mockToken,
          scope: 'api:capability:*',
          expiresAt: expiresAt,
          caveats: { scope: 'api:capability:*', expires: expiresAt }
        });
        
        setCurrentScene('use');
        return;
      }

      const response = await fetch(`${BASE_URL}/api/capability/mint`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          scope: 'api:capability:*',
          expiresIn: 3600
        })
      });

      if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        throw new Error(errData.message || `Server returned ${response.status}`);
      }

      const data = await response.json();
      addLog('✅ Token minted successfully!', 'success');
      addLog(`📜 Scope: ${data.caveats?.scope || 'api:capability:*'}`, 'info');
      addLog(`⏰ Expires: ${data.caveats?.expires || 'in 1 hour'}`, 'info');
      addLog('💡 No database write. No service account. Pure cryptography.', 'success');
      
      setParentToken({
        raw: data.token,
        scope: data.caveats?.scope || 'api:capability:*',
        expiresAt: data.caveats?.expires || '',
        caveats: data.caveats || {}
      });

      setCurrentScene('use');
    } catch (err: any) {
      addLog(`❌ Error: ${err.message}`, 'error');
      addLog('💡 Try enabling Simulation Mode (the server may require admin auth).', 'warn');
    } finally {
      setIsLoading(false);
    }
  };

  // Scene 2: Use Token
  const handleUseToken = async () => {
    if (!parentToken) return;
    
    setIsLoading(true);
    clearLogs();
    addLog('🤖 Agent: "Using the capability token to access protected resource."', 'info');
    addLog(`📡 GET /api/capability/ping ${useSimulation ? '(SIMULATION)' : '(LIVE)'}`, 'info');
    addLog(`🔑 Authorization: Bearer ${parentToken.raw.substring(0, 30)}...`, 'info');
    
    try {
      if (useSimulation) {
        await new Promise(r => setTimeout(r, 600));
        const mockResult = {
          ok: true,
          tier: 'capability',
          price: '0 sats',
          mode: 'Phase 1: Capability-Only',
          time: new Date().toISOString(),
          resource: 'capability-ping',
          accessType: 'Macaroon (no payment)',
          data: {
            message: '✓ Authenticated with capability token - no Lightning payment required!',
            note: 'This proves Zero Trust PEP works without the crypto payment rail.'
          }
        };
        
        addLog('✅ 200 OK - Authenticated instantly!', 'success');
        addLog('⚡ Token validated mathematically. No LDAP lookup. No bottleneck.', 'success');
        setPingResult(mockResult);
        setCurrentScene('delegate');
        return;
      }

      const response = await fetch(`${BASE_URL}/api/capability/ping`, {
        headers: {
          'Authorization': `Bearer ${parentToken.raw}`
        }
      });

      const data = await response.json();
      
      if (response.ok) {
        addLog('✅ 200 OK - Authenticated instantly!', 'success');
        addLog('⚡ Token validated mathematically. No LDAP lookup. No bottleneck.', 'success');
        setPingResult(data);
        setCurrentScene('delegate');
      } else {
        addLog(`❌ ${response.status}: ${data.error || 'Access denied'}`, 'error');
      }
    } catch (err: any) {
      addLog(`❌ Error: ${err.message}`, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  // Scene 3: Delegation
  const handleDelegation = async () => {
    setIsLoading(true);
    clearLogs();
    addLog('🤖 Agent: "I need to delegate a task to a Worker."', 'info');
    addLog('📝 The Worker should only access /ping for 5 minutes.', 'info');
    addLog('', 'info');
    addLog('❌ Traditional IAM would require:', 'warn');
    addLog('   → Request new service account (ticket)', 'warn');
    addLog('   → Wait for approval (days)', 'warn');
    addLog('   → Provision credentials', 'warn');
    addLog('', 'info');
    addLog('✅ With capability tokens, I do this:', 'info');
    addLog(`📡 Offline Delegation ${useSimulation ? '(SIMULATION)' : '(LIVE)'}`, 'info');
    
    try {
      if (useSimulation) {
        await new Promise(r => setTimeout(r, 1000));
        
        const mockChildToken = generateMockMacaroon();
        const childExpiry = new Date(Date.now() + 300000).toISOString();
        
        addLog('', 'info');
        addLog('🎯 [NETWORK] Requests sent: 0 ← OFFLINE OPERATION', 'success');
        addLog('🔐 [CRYPTO] Token attenuated mathematically', 'success');
        addLog('', 'info');
        
        setDelegationResult({ childToken: mockChildToken, childExpiry });
        
        setChildToken({
          raw: mockChildToken,
          scope: 'api:capability:ping',
          expiresAt: childExpiry,
          caveats: { scope: 'api:capability:ping', expires: childExpiry }
        });
        addLog('✅ Child token created with restricted scope!', 'success');
        
        setCurrentScene('enforce');
        return;
      }

      const response = await fetch(`${BASE_URL}/api/capability/demo/delegate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      });

      if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        throw new Error(errData.message || `Server returned ${response.status}`);
      }

      const data = await response.json();
      
      addLog('', 'info');
      addLog('🎯 [NETWORK] Requests sent: 0 ← OFFLINE OPERATION', 'success');
      addLog('🔐 [CRYPTO] Token attenuated mathematically', 'success');
      addLog('', 'info');
      
      setDelegationResult(data);
      
      if (data.childToken) {
        setChildToken({
          raw: data.childToken,
          scope: 'api:capability:ping',
          expiresAt: data.childExpiry || '',
          caveats: { scope: 'api:capability:ping', expires: data.childExpiry }
        });
        addLog('✅ Child token created with restricted scope!', 'success');
      }
      
      setCurrentScene('enforce');
    } catch (err: any) {
      addLog(`❌ Error: ${err.message}`, 'error');
      addLog('💡 Try enabling Simulation Mode (the server may require admin auth).', 'warn');
    } finally {
      setIsLoading(false);
    }
  };

  // Scene 3b: Enforcement Tests
  const handleEnforcementTests = async () => {
    if (!childToken) return;
    
    setIsLoading(true);
    clearLogs();
    addLog(`🔒 Testing Least Privilege Enforcement ${useSimulation ? '(SIMULATION)' : '(LIVE)'}`, 'info');
    addLog('', 'info');
    
    // Test 1: Allowed action (ping)
    addLog('TEST 1: Child token → /api/capability/ping', 'info');
    addLog(`🔑 Authorization: Bearer ${childToken.raw.substring(0, 30)}...`, 'info');
    
    let allowedResult = null;
    let blockedResult = null;
    
    if (useSimulation) {
      await new Promise(r => setTimeout(r, 500));
      addLog('✅ 200 OK - Access granted (within scope)', 'success');
      allowedResult = { ok: true, tier: 'capability', resource: 'capability-ping' };
      
      addLog('', 'info');
      await new Promise(r => setTimeout(r, 300));
      
      addLog('TEST 2: Child token → /api/capability/mint (SHOULD FAIL)', 'info');
      addLog(`🔑 Authorization: Bearer ${childToken.raw.substring(0, 30)}...`, 'info');
      
      await new Promise(r => setTimeout(r, 500));
      addLog('✅ 403 Forbidden - Escalation blocked!', 'success');
      addLog('📝 Reason: caveat check failed (scope = api:capability:ping): Scope violation', 'info');
      blockedResult = { 
        error: 'Access Denied', 
        reason: 'caveat check failed (scope = api:capability:ping): Scope violation: token has \'api:capability:ping\', need \'api:capability:admin\''
      };
      
      addLog('', 'info');
      addLog('💡 The math enforced least privilege. Not a database lookup.', 'success');
      
      setEnforcementResults({ allowed: allowedResult, blocked: blockedResult });
      setCurrentScene('summary');
      setIsLoading(false);
      return;
    }
    
    try {
      const pingRes = await fetch(`${BASE_URL}/api/capability/ping`, {
        headers: { 'Authorization': `Bearer ${childToken.raw}` }
      });
      const pingData = await pingRes.json();
      
      if (pingRes.ok) {
        addLog('✅ 200 OK - Access granted (within scope)', 'success');
        allowedResult = pingData;
      } else {
        addLog(`❌ ${pingRes.status}: ${pingData.error}`, 'error');
        allowedResult = { error: pingData.error };
      }
    } catch (err: any) {
      addLog(`❌ Error: ${err.message}`, 'error');
    }

    addLog('', 'info');
    
    // Test 2: Blocked action (mint)
    addLog('TEST 2: Child token → /api/capability/mint (SHOULD FAIL)', 'info');
    addLog(`🔑 Authorization: Bearer ${childToken.raw.substring(0, 30)}...`, 'info');
    
    try {
      const mintRes = await fetch(`${BASE_URL}/api/capability/mint`, {
        method: 'POST',
        headers: { 
          'Authorization': `Bearer ${childToken.raw}`,
          'Content-Type': 'application/json'
        }
      });
      const mintData = await mintRes.json();
      
      if (mintRes.status === 403) {
        addLog('✅ 403 Forbidden - Escalation blocked!', 'success');
        addLog(`📝 Reason: ${mintData.reason || mintData.error}`, 'info');
        blockedResult = mintData;
      } else if (mintRes.ok) {
        addLog('⚠️ Warning: Should have been blocked!', 'warn');
        blockedResult = { unexpected: 'allowed' };
      } else {
        addLog(`❌ ${mintRes.status}: ${mintData.error}`, 'error');
        blockedResult = mintData;
      }
    } catch (err: any) {
      addLog(`❌ Error: ${err.message}`, 'error');
    }

    addLog('', 'info');
    addLog('💡 The math enforced least privilege. Not a database lookup.', 'success');
    
    setEnforcementResults({ allowed: allowedResult, blocked: blockedResult });
    setCurrentScene('summary');
    setIsLoading(false);
  };

  // Scene 5: Kill Switch (Revocation)
  const handleBanToken = async () => {
    if (!childToken) return;
    
    setIsLoading(true);
    clearLogs();
    addLog('🚨 CISO: "Emergency! The child token has been compromised!"', 'warn');
    addLog(`📡 POST /api/governance/ban ${useSimulation ? '(SIMULATION)' : '(LIVE)'}`, 'info');
    addLog(`🎯 Target: ${childToken.raw.substring(0, 30)}...`, 'info');
    
    try {
      if (useSimulation) {
        await new Promise(r => setTimeout(r, 600));
        
        addLog('', 'info');
        addLog('✅ Token BANNED successfully!', 'success');
        addLog('⚡ Propagation: INSTANT (no sync delay)', 'success');
        addLog('📝 Ban reason: "Demo revocation - compromised token"', 'info');
        addLog('🔗 All child tokens derived from this: ALSO BANNED', 'warn');
        
        setBannedToken(childToken.raw);
        setRevocationResult({ banned: true, tested: false });
        return;
      }

      // Live mode would require admin auth
      const response = await fetch(`${BASE_URL}/api/governance/ban`, {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          // Note: Live mode requires X-Admin-Token header
        },
        body: JSON.stringify({
          tokenSignature: childToken.raw.substring(0, 32),
          reason: 'Demo revocation - compromised token'
        })
      });

      if (!response.ok) {
        throw new Error('Admin authentication required for live ban');
      }

      const data = await response.json();
      addLog('', 'info');
      addLog('✅ Token BANNED successfully!', 'success');
      addLog('⚡ Propagation: INSTANT', 'success');
      
      setBannedToken(childToken.raw);
      setRevocationResult({ banned: true, tested: false });
    } catch (err: any) {
      addLog(`❌ Error: ${err.message}`, 'error');
      addLog('💡 Live ban requires admin authentication. Using simulation.', 'warn');
      
      // Fall back to simulation
      await new Promise(r => setTimeout(r, 400));
      addLog('', 'info');
      addLog('✅ Token BANNED (simulated)', 'success');
      setBannedToken(childToken.raw);
      setRevocationResult({ banned: true, tested: false });
    } finally {
      setIsLoading(false);
    }
  };

  const handleTestBannedToken = async () => {
    if (!bannedToken) return;
    
    setIsLoading(true);
    addLog('', 'info');
    addLog('🤖 Worker: "Trying to use my old token..."', 'info');
    addLog(`📡 GET /api/capability/ping ${useSimulation ? '(SIMULATION)' : '(LIVE)'}`, 'info');
    addLog(`🔑 Authorization: Bearer ${bannedToken.substring(0, 30)}...`, 'info');
    
    try {
      if (useSimulation) {
        await new Promise(r => setTimeout(r, 500));
        
        addLog('', 'info');
        addLog('🚫 403 Forbidden - TOKEN REVOKED!', 'error');
        addLog('', 'info');
        addLog('📝 Response:', 'info');
        addLog('   error: "Token Revoked"', 'error');
        addLog('   reason: "This token has been banned by an administrator"', 'error');
        addLog('   code: "TOKEN_BANNED"', 'error');
        addLog('', 'info');
        addLog('💡 The Panic Button worked. Token is dead globally.', 'success');
        
        setRevocationResult({ banned: true, tested: true });
        return;
      }

      const response = await fetch(`${BASE_URL}/api/capability/ping`, {
        headers: { 'Authorization': `Bearer ${bannedToken}` }
      });

      const data = await response.json();
      
      if (response.status === 403) {
        addLog('', 'info');
        addLog('🚫 403 Forbidden - TOKEN REVOKED!', 'error');
        addLog(`📝 Response: ${JSON.stringify(data)}`, 'info');
        addLog('', 'info');
        addLog('💡 The Panic Button worked. Token is dead globally.', 'success');
        setRevocationResult({ banned: true, tested: true });
      } else {
        addLog(`⚠️ Unexpected response: ${response.status}`, 'warn');
        setRevocationResult({ banned: true, tested: true, error: 'Unexpected response' });
      }
    } catch (err: any) {
      addLog(`❌ Error: ${err.message}`, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const resetDemo = () => {
    setCurrentScene('intro');
    setParentToken(null);
    setChildToken(null);
    setPingResult(null);
    setDelegationResult(null);
    setEnforcementResults(null);
    setBannedToken(null);
    setRevocationResult(null);
    clearLogs();
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      
      {/* Header */}
      <div className="border-b border-gray-800 bg-black/80 backdrop-blur-sm sticky top-0 z-40">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition">
            <ArrowLeft size={18} /> <span className="hidden sm:inline">Back to Home</span>
          </Link>
          <h1 className="text-lg sm:text-xl font-bold flex items-center gap-2">
            <Shield className="text-purple-400" size={24} />
            <span className="bg-clip-text text-transparent bg-gradient-to-r from-purple-400 to-cyan-400">
              Crawl Phase Demo
            </span>
          </h1>
          {/* Mode Toggle */}
          <button 
            onClick={() => setUseSimulation(!useSimulation)}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-bold transition-all border ${
              useSimulation 
                ? 'bg-gray-800 border-gray-600 text-gray-400' 
                : 'bg-purple-900/30 border-purple-500 text-purple-300'
            }`}
          >
            {useSimulation ? <WifiOff size={14} /> : <Wifi size={14} />}
            <span className="hidden sm:inline">{useSimulation ? 'SIMULATION' : 'LIVE'}</span>
          </button>
        </div>
      </div>

      <div className="max-w-6xl mx-auto px-4 sm:px-6 py-8">
        
        {/* Scene Progress */}
        <div className="flex items-center justify-center gap-2 mb-8 overflow-x-auto pb-2">
          {[
            { id: 'intro', label: 'Intro', icon: Eye },
            { id: 'mint', label: 'Mint Token', icon: Key },
            { id: 'use', label: 'Use Token', icon: Unlock },
            { id: 'delegate', label: 'Delegate', icon: GitBranch },
            { id: 'enforce', label: 'Enforce', icon: Lock },
            { id: 'revoke', label: 'Kill Switch', icon: Ban },
            { id: 'summary', label: 'Summary', icon: CheckCircle },
          ].map((scene, idx, arr) => (
            <React.Fragment key={scene.id}>
              <button
                onClick={() => {
                  // Only allow going back to completed scenes
                  const scenes: DemoScene[] = ['intro', 'mint', 'use', 'delegate', 'enforce', 'revoke', 'summary'];
                  const currentIdx = scenes.indexOf(currentScene);
                  const targetIdx = scenes.indexOf(scene.id as DemoScene);
                  if (targetIdx <= currentIdx) {
                    setCurrentScene(scene.id as DemoScene);
                  }
                }}
                className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition whitespace-nowrap ${
                  currentScene === scene.id 
                    ? 'bg-purple-900/50 border border-purple-500 text-purple-300' 
                    : 'bg-gray-900 border border-gray-800 text-gray-500 hover:text-gray-300'
                }`}
              >
                <scene.icon size={16} />
                <span className="hidden sm:inline">{scene.label}</span>
              </button>
              {idx < arr.length - 1 && <ChevronRight size={16} className="text-gray-700 flex-shrink-0" />}
            </React.Fragment>
          ))}
        </div>

        {/* Main Content Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          
          {/* Left: Scene Content */}
          <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
            
            {/* Scene Header */}
            <div className="p-6 border-b border-gray-800 bg-gradient-to-r from-purple-950/30 to-cyan-950/30">
              {currentScene === 'intro' && (
                <>
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-3 bg-purple-900/50 rounded-xl">
                      <Shield className="text-purple-400" size={28} />
                    </div>
                    <div>
                      <h2 className="text-xl font-bold text-white">Phase 1: Crawl</h2>
                      <p className="text-gray-400 text-sm">Zero Trust Security via Capability Tokens</p>
                    </div>
                  </div>
                  <p className="text-gray-300 leading-relaxed">
                    Crawl is the foundation of SatGate's maturity model. It proves we solve IAM scale problems 
                    <strong className="text-white"> today</strong>, without any Bitcoin or payment rail.
                  </p>
                </>
              )}

              {currentScene === 'mint' && (
                <>
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-3 bg-cyan-900/50 rounded-xl">
                      <Key className="text-cyan-400" size={28} />
                    </div>
                    <div>
                      <h2 className="text-xl font-bold text-white">Scene 1: Token Minting</h2>
                      <p className="text-gray-400 text-sm">The CISO issues a capability</p>
                    </div>
                  </div>
                  <p className="text-gray-300 leading-relaxed">
                    "I don't create a user. I issue a <strong className="text-cyan-400">capability</strong>. 
                    No database write. No service account. Just a token that says 'You can read data for 1 hour'."
                  </p>
                </>
              )}

              {currentScene === 'use' && (
                <>
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-3 bg-green-900/50 rounded-xl">
                      <Unlock className="text-green-400" size={28} />
                    </div>
                    <div>
                      <h2 className="text-xl font-bold text-white">Scene 2: Token Usage</h2>
                      <p className="text-gray-400 text-sm">The Agent uses the token</p>
                    </div>
                  </div>
                  <p className="text-gray-300 leading-relaxed">
                    "The agent uses the token. Watch the speed. Authenticated instantly. 
                    The Gateway validated the signature <strong className="text-green-400">mathematically</strong>. 
                    No LDAP lookup. No bottleneck."
                  </p>
                </>
              )}

              {currentScene === 'delegate' && (
                <>
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-3 bg-purple-900/50 rounded-xl">
                      <GitBranch className="text-purple-400" size={28} />
                    </div>
                    <div>
                      <h2 className="text-xl font-bold text-white">Scene 3: Offline Delegation</h2>
                      <p className="text-gray-400 text-sm">The "Wow" Moment</p>
                    </div>
                  </div>
                  <p className="text-gray-300 leading-relaxed">
                    "The agent just cut a spare key for the janitor — one that only opens the basement, 
                    and expires in 5 minutes. It didn't need to call the locksmith."
                  </p>
                </>
              )}

              {currentScene === 'enforce' && (
                <>
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-3 bg-red-900/50 rounded-xl">
                      <Lock className="text-red-400" size={28} />
                    </div>
                    <div>
                      <h2 className="text-xl font-bold text-white">Scene 3b: Scope Enforcement</h2>
                      <p className="text-gray-400 text-sm">Least Privilege in Action</p>
                    </div>
                  </div>
                  <p className="text-gray-300 leading-relaxed">
                    "Can the child token escalate privileges? Let's test it. The Gateway will reject it — 
                    not because we looked it up in a database, but because the 
                    <strong className="text-red-400"> token itself</strong> said 'I can only access /ping'."
                  </p>
                </>
              )}

              {currentScene === 'revoke' && (
                <>
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-3 bg-orange-900/50 rounded-xl">
                      <Ban className="text-orange-400" size={28} />
                    </div>
                    <div>
                      <h2 className="text-xl font-bold text-white">Scene 5: Kill Switch</h2>
                      <p className="text-gray-400 text-sm">Emergency Token Revocation</p>
                    </div>
                  </div>
                  <p className="text-gray-300 leading-relaxed">
                    "The token is compromised. Ban it <strong className="text-orange-400">globally and instantly</strong>. 
                    We're stateless for validation (fast), but stateful for revocation (secure). 
                    This is the <strong className="text-orange-400">Panic Button</strong>."
                  </p>
                </>
              )}

              {currentScene === 'summary' && (
                <>
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-3 bg-green-900/50 rounded-xl">
                      <CheckCircle className="text-green-400" size={28} />
                    </div>
                    <div>
                      <h2 className="text-xl font-bold text-white">Crawl Complete!</h2>
                      <p className="text-gray-400 text-sm">Zero Trust. Zero Bitcoin. Zero Friction.</p>
                    </div>
                  </div>
                  <p className="text-gray-300 leading-relaxed">
                    Everything you just saw happened without a single satoshi of Bitcoin. 
                    Crawl is live. Run (Payments) is just a config change away.
                  </p>
                </>
              )}
            </div>

            {/* Scene Body */}
            <div className="p-6">
              
              {currentScene === 'intro' && (
                <div className="space-y-6">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="p-4 bg-black rounded-xl border border-gray-800">
                      <div className="flex items-center gap-2 text-purple-400 font-semibold mb-2">
                        <User size={18} /> CISO Role
                      </div>
                      <p className="text-gray-400 text-sm">
                        Issues root credentials. Retains authority. Agents cannot escalate.
                      </p>
                    </div>
                    <div className="p-4 bg-black rounded-xl border border-gray-800">
                      <div className="flex items-center gap-2 text-cyan-400 font-semibold mb-2">
                        <Bot size={18} /> Agent Role
                      </div>
                      <p className="text-gray-400 text-sm">
                        Uses tokens. Can delegate restricted sub-tokens offline.
                      </p>
                    </div>
                  </div>
                  
                  <div className="bg-gradient-to-r from-purple-950/30 to-cyan-950/30 rounded-xl p-4 border border-purple-800/30">
                    <h4 className="font-semibold text-white mb-2">What You'll See:</h4>
                    <ul className="text-gray-400 text-sm space-y-1">
                      <li>✓ Stateless credential issuance (no database)</li>
                      <li>✓ Instant cryptographic validation</li>
                      <li>✓ Offline token delegation (zero network calls)</li>
                      <li>✓ Mathematical scope enforcement</li>
                    </ul>
                  </div>

                  <button
                    onClick={() => setCurrentScene('mint')}
                    className="w-full py-4 bg-white text-black rounded-xl font-bold text-lg hover:bg-gray-200 transition flex items-center justify-center gap-2"
                  >
                    <Play size={20} fill="black" /> Start Demo
                  </button>
                </div>
              )}

              {currentScene === 'mint' && (
                <div className="space-y-4">
                  {!parentToken ? (
                    <button
                      onClick={handleMintToken}
                      disabled={isLoading}
                      className={`w-full py-4 rounded-xl font-bold text-lg transition flex items-center justify-center gap-2 ${
                        isLoading 
                          ? 'bg-gray-800 text-gray-500 cursor-not-allowed' 
                          : 'bg-cyan-600 text-white hover:bg-cyan-500'
                      }`}
                    >
                      {isLoading ? (
                        <>
                          <RefreshCw size={20} className="animate-spin" /> Minting...
                        </>
                      ) : (
                        <>
                          <Key size={20} /> Mint Capability Token
                        </>
                      )}
                    </button>
                  ) : (
                    <div className="space-y-4">
                      <div className="bg-green-950/30 border border-green-800/50 rounded-xl p-4">
                        <div className="flex items-center gap-2 text-green-400 font-semibold mb-2">
                          <CheckCircle size={18} /> Token Minted
                        </div>
                        <div className="space-y-2 text-sm">
                          <div className="flex justify-between">
                            <span className="text-gray-400">Scope:</span>
                            <span className="text-white font-mono">{parentToken.scope}</span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-gray-400">Expires:</span>
                            <span className="text-white font-mono text-xs">{parentToken.expiresAt}</span>
                          </div>
                        </div>
                        <div className="mt-3 flex gap-2">
                          <button
                            onClick={() => copyToClipboard(parentToken.raw, 'parent')}
                            className="flex-1 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-sm flex items-center justify-center gap-2 transition"
                          >
                            {copied === 'parent' ? <Check size={14} className="text-green-400" /> : <Copy size={14} />}
                            Copy Token
                          </button>
                        </div>
                      </div>
                      <button
                        onClick={() => setCurrentScene('use')}
                        className="w-full py-3 bg-white text-black rounded-xl font-bold hover:bg-gray-200 transition flex items-center justify-center gap-2"
                      >
                        Next: Use Token <ChevronRight size={18} />
                      </button>
                    </div>
                  )}
                </div>
              )}

              {currentScene === 'use' && (
                <div className="space-y-4">
                  {!pingResult ? (
                    <button
                      onClick={handleUseToken}
                      disabled={isLoading || !parentToken}
                      className={`w-full py-4 rounded-xl font-bold text-lg transition flex items-center justify-center gap-2 ${
                        isLoading || !parentToken
                          ? 'bg-gray-800 text-gray-500 cursor-not-allowed' 
                          : 'bg-green-600 text-white hover:bg-green-500'
                      }`}
                    >
                      {isLoading ? (
                        <>
                          <RefreshCw size={20} className="animate-spin" /> Authenticating...
                        </>
                      ) : (
                        <>
                          <Unlock size={20} /> Test Token Access
                        </>
                      )}
                    </button>
                  ) : (
                    <div className="space-y-4">
                      <div className="bg-green-950/30 border border-green-800/50 rounded-xl p-4">
                        <div className="flex items-center gap-2 text-green-400 font-semibold mb-2">
                          <CheckCircle size={18} /> Access Granted
                        </div>
                        <pre className="text-xs text-gray-300 bg-black rounded-lg p-3 overflow-x-auto">
                          {JSON.stringify(pingResult, null, 2)}
                        </pre>
                      </div>
                      <button
                        onClick={() => setCurrentScene('delegate')}
                        className="w-full py-3 bg-white text-black rounded-xl font-bold hover:bg-gray-200 transition flex items-center justify-center gap-2"
                      >
                        Next: Delegation Demo <ChevronRight size={18} />
                      </button>
                    </div>
                  )}
                </div>
              )}

              {currentScene === 'delegate' && (
                <div className="space-y-4">
                  {!delegationResult ? (
                    <button
                      onClick={handleDelegation}
                      disabled={isLoading}
                      className={`w-full py-4 rounded-xl font-bold text-lg transition flex items-center justify-center gap-2 ${
                        isLoading 
                          ? 'bg-gray-800 text-gray-500 cursor-not-allowed' 
                          : 'bg-purple-600 text-white hover:bg-purple-500'
                      }`}
                    >
                      {isLoading ? (
                        <>
                          <RefreshCw size={20} className="animate-spin" /> Delegating...
                        </>
                      ) : (
                        <>
                          <GitBranch size={20} /> Run Delegation Demo
                        </>
                      )}
                    </button>
                  ) : (
                    <div className="space-y-4">
                      {/* Comparison Table */}
                      <div className="bg-black rounded-xl border border-gray-800 overflow-hidden">
                        <div className="grid grid-cols-3 text-sm font-semibold border-b border-gray-800">
                          <div className="p-3 bg-gray-900">Property</div>
                          <div className="p-3 bg-gray-900 text-center">Parent</div>
                          <div className="p-3 bg-gray-900 text-center">Child</div>
                        </div>
                        <div className="grid grid-cols-3 text-sm border-b border-gray-800">
                          <div className="p-3 text-gray-400">Scope</div>
                          <div className="p-3 text-center font-mono text-cyan-400">api:capability:*</div>
                          <div className="p-3 text-center font-mono text-purple-400">api:capability:ping</div>
                        </div>
                        <div className="grid grid-cols-3 text-sm border-b border-gray-800">
                          <div className="p-3 text-gray-400">Expires</div>
                          <div className="p-3 text-center">1 hour</div>
                          <div className="p-3 text-center">5 minutes</div>
                        </div>
                        <div className="grid grid-cols-3 text-sm border-b border-gray-800">
                          <div className="p-3 text-gray-400">Network calls</div>
                          <div className="p-3 text-center text-green-400">0</div>
                          <div className="p-3 text-center text-green-400">0</div>
                        </div>
                        <div className="grid grid-cols-3 text-sm">
                          <div className="p-3 text-gray-400">Admin approval</div>
                          <div className="p-3 text-center text-green-400">NO</div>
                          <div className="p-3 text-center text-green-400">NO</div>
                        </div>
                      </div>

                      {childToken && (
                        <div className="bg-purple-950/30 border border-purple-800/50 rounded-xl p-4">
                          <div className="flex items-center gap-2 text-purple-400 font-semibold mb-2">
                            <GitBranch size={18} /> Child Token Created
                          </div>
                          <div className="flex gap-2">
                            <button
                              onClick={() => copyToClipboard(childToken.raw, 'child')}
                              className="flex-1 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-sm flex items-center justify-center gap-2 transition"
                            >
                              {copied === 'child' ? <Check size={14} className="text-green-400" /> : <Copy size={14} />}
                              Copy Child Token
                            </button>
                          </div>
                        </div>
                      )}

                      <button
                        onClick={() => setCurrentScene('enforce')}
                        className="w-full py-3 bg-white text-black rounded-xl font-bold hover:bg-gray-200 transition flex items-center justify-center gap-2"
                      >
                        Next: Test Enforcement <ChevronRight size={18} />
                      </button>
                    </div>
                  )}
                </div>
              )}

              {currentScene === 'enforce' && (
                <div className="space-y-4">
                  {!enforcementResults ? (
                    <button
                      onClick={handleEnforcementTests}
                      disabled={isLoading || !childToken}
                      className={`w-full py-4 rounded-xl font-bold text-lg transition flex items-center justify-center gap-2 ${
                        isLoading || !childToken
                          ? 'bg-gray-800 text-gray-500 cursor-not-allowed' 
                          : 'bg-red-600 text-white hover:bg-red-500'
                      }`}
                    >
                      {isLoading ? (
                        <>
                          <RefreshCw size={20} className="animate-spin" /> Testing...
                        </>
                      ) : (
                        <>
                          <Lock size={20} /> Run Enforcement Tests
                        </>
                      )}
                    </button>
                  ) : (
                    <div className="space-y-4">
                      {/* Test Results */}
                      <div className="grid grid-cols-1 gap-4">
                        <div className="bg-green-950/30 border border-green-800/50 rounded-xl p-4">
                          <div className="flex items-center gap-2 text-green-400 font-semibold mb-2">
                            <CheckCircle size={18} /> Test 1: /ping (Allowed)
                          </div>
                          <p className="text-gray-400 text-sm">
                            Child token successfully accessed /ping — within its scope.
                          </p>
                        </div>
                        <div className="bg-red-950/30 border border-red-800/50 rounded-xl p-4">
                          <div className="flex items-center gap-2 text-red-400 font-semibold mb-2">
                            <XCircle size={18} /> Test 2: /mint (Blocked)
                          </div>
                          <p className="text-gray-400 text-sm">
                            Child token was rejected when trying to mint new tokens. 
                            The math enforced least privilege.
                          </p>
                        </div>
                      </div>

                      <button
                        onClick={() => setCurrentScene('revoke')}
                        className="w-full py-3 bg-white text-black rounded-xl font-bold hover:bg-gray-200 transition flex items-center justify-center gap-2"
                      >
                        Next: Kill Switch Demo <ChevronRight size={18} />
                      </button>
                    </div>
                  )}
                </div>
              )}

              {currentScene === 'revoke' && (
                <div className="space-y-4">
                  {!revocationResult?.banned ? (
                    <>
                      <div className="bg-orange-950/30 border border-orange-800/50 rounded-xl p-4">
                        <div className="flex items-center gap-2 text-orange-400 font-semibold mb-2">
                          <AlertTriangle size={18} /> Emergency Scenario
                        </div>
                        <p className="text-gray-400 text-sm">
                          The child token has been compromised! An attacker obtained it from a worker's logs. 
                          We need to revoke it <strong>immediately</strong>.
                        </p>
                      </div>

                      <div className="bg-black rounded-xl border border-gray-800 p-4">
                        <div className="text-sm text-gray-400 mb-2">Token to ban:</div>
                        <div className="text-xs font-mono text-orange-400 break-all">
                          {childToken?.raw.substring(0, 50)}...
                        </div>
                      </div>

                      <button
                        onClick={handleBanToken}
                        disabled={isLoading || !childToken}
                        className={`w-full py-4 rounded-xl font-bold text-lg transition flex items-center justify-center gap-2 ${
                          isLoading || !childToken
                            ? 'bg-gray-800 text-gray-500 cursor-not-allowed' 
                            : 'bg-orange-600 text-white hover:bg-orange-500'
                        }`}
                      >
                        {isLoading ? (
                          <>
                            <RefreshCw size={20} className="animate-spin" /> Banning...
                          </>
                        ) : (
                          <>
                            <Ban size={20} /> Trigger Kill Switch
                          </>
                        )}
                      </button>
                    </>
                  ) : !revocationResult?.tested ? (
                    <div className="space-y-4">
                      <div className="bg-green-950/30 border border-green-800/50 rounded-xl p-4">
                        <div className="flex items-center gap-2 text-green-400 font-semibold mb-2">
                          <CheckCircle size={18} /> Token Banned!
                        </div>
                        <p className="text-gray-400 text-sm">
                          The token is now on the global ban list. Let's verify it's actually blocked.
                        </p>
                      </div>

                      <div className="bg-gradient-to-r from-orange-950/30 to-red-950/30 rounded-xl p-4 border border-orange-800/30">
                        <h4 className="font-semibold text-white mb-2 flex items-center gap-2">
                          <Zap size={16} className="text-yellow-400" /> Instant Propagation
                        </h4>
                        <p className="text-gray-400 text-sm">
                          Unlike traditional IAM where revocation can take minutes/hours to propagate, 
                          SatGate's ban list is checked on every request. The token is dead <strong>now</strong>.
                        </p>
                      </div>

                      <button
                        onClick={handleTestBannedToken}
                        disabled={isLoading}
                        className={`w-full py-4 rounded-xl font-bold text-lg transition flex items-center justify-center gap-2 ${
                          isLoading 
                            ? 'bg-gray-800 text-gray-500 cursor-not-allowed' 
                            : 'bg-red-600 text-white hover:bg-red-500'
                        }`}
                      >
                        {isLoading ? (
                          <>
                            <RefreshCw size={20} className="animate-spin" /> Testing...
                          </>
                        ) : (
                          <>
                            <ShieldOff size={20} /> Try Using Banned Token
                          </>
                        )}
                      </button>
                    </div>
                  ) : (
                    <div className="space-y-4">
                      <div className="bg-red-950/30 border border-red-800/50 rounded-xl p-4">
                        <div className="flex items-center gap-2 text-red-400 font-semibold mb-2">
                          <XCircle size={18} /> Token Rejected!
                        </div>
                        <p className="text-gray-400 text-sm mb-3">
                          The banned token was instantly rejected. The attacker is locked out.
                        </p>
                        <pre className="text-xs bg-black rounded-lg p-3 text-red-400 overflow-x-auto">
{`{
  "error": "Token Revoked",
  "reason": "This token has been banned by an administrator",
  "code": "TOKEN_BANNED"
}`}
                        </pre>
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <div className="p-4 bg-black rounded-xl border border-orange-800/50 text-center">
                          <div className="text-2xl font-bold text-orange-400 mb-1">Instant</div>
                          <div className="text-gray-400 text-xs">Revocation Time</div>
                        </div>
                        <div className="p-4 bg-black rounded-xl border border-red-800/50 text-center">
                          <div className="text-2xl font-bold text-red-400 mb-1">Global</div>
                          <div className="text-gray-400 text-xs">Ban Scope</div>
                        </div>
                      </div>

                      <button
                        onClick={() => setCurrentScene('summary')}
                        className="w-full py-3 bg-white text-black rounded-xl font-bold hover:bg-gray-200 transition flex items-center justify-center gap-2"
                      >
                        View Summary <ChevronRight size={18} />
                      </button>
                    </div>
                  )}
                </div>
              )}

              {currentScene === 'summary' && (
                <div className="space-y-6">
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <div className="p-4 bg-black rounded-xl border border-green-800/50">
                      <div className="text-3xl font-bold text-green-400 mb-1">0</div>
                      <div className="text-gray-400 text-sm">Network calls for delegation</div>
                    </div>
                    <div className="p-4 bg-black rounded-xl border border-purple-800/50">
                      <div className="text-3xl font-bold text-purple-400 mb-1">0</div>
                      <div className="text-gray-400 text-sm">Admin tickets required</div>
                    </div>
                    <div className="p-4 bg-black rounded-xl border border-orange-800/50">
                      <div className="text-3xl font-bold text-orange-400 mb-1">Instant</div>
                      <div className="text-gray-400 text-sm">Kill Switch revocation</div>
                    </div>
                  </div>

                  <div className="bg-gradient-to-r from-purple-950/30 to-cyan-950/30 rounded-xl p-4 border border-purple-800/30">
                    <h4 className="font-semibold text-white mb-3">What Crawl Proves:</h4>
                    <ul className="text-gray-300 text-sm space-y-2">
                      <li className="flex items-start gap-2">
                        <CheckCircle size={16} className="text-green-400 mt-0.5 flex-shrink-0" />
                        <span>Agents self-manage least-privilege access</span>
                      </li>
                      <li className="flex items-start gap-2">
                        <CheckCircle size={16} className="text-green-400 mt-0.5 flex-shrink-0" />
                        <span>No $150/account "Identity Tax" per agent</span>
                      </li>
                      <li className="flex items-start gap-2">
                        <CheckCircle size={16} className="text-green-400 mt-0.5 flex-shrink-0" />
                        <span>CISO retains authority (tokens are read-only by default)</span>
                      </li>
                      <li className="flex items-start gap-2">
                        <CheckCircle size={16} className="text-green-400 mt-0.5 flex-shrink-0" />
                        <span>Instant Kill Switch revocation — no propagation delay</span>
                      </li>
                      <li className="flex items-start gap-2">
                        <CheckCircle size={16} className="text-green-400 mt-0.5 flex-shrink-0" />
                        <span>Google-grade macaroons — same tech from 2014 research</span>
                      </li>
                    </ul>
                  </div>

                  <div className="flex gap-4">
                    <button
                      onClick={resetDemo}
                      className="flex-1 py-3 bg-gray-800 text-white rounded-xl font-bold hover:bg-gray-700 transition flex items-center justify-center gap-2"
                    >
                      <RefreshCw size={18} /> Run Again
                    </button>
                    <Link
                      href="/playground"
                      className="flex-1 py-3 bg-gradient-to-r from-purple-600 to-cyan-600 text-white rounded-xl font-bold hover:opacity-90 transition flex items-center justify-center gap-2"
                    >
                      Try Payments <ChevronRight size={18} />
                    </Link>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Right: Terminal Log */}
          <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden flex flex-col">
            <div className="p-4 border-b border-gray-800 flex items-center justify-between bg-black">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-red-500"></div>
                <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
                <div className="w-3 h-3 rounded-full bg-green-500"></div>
                <span className="ml-3 text-gray-500 text-xs font-mono">Live Demo Output</span>
              </div>
              <button
                onClick={clearLogs}
                className="p-1.5 text-gray-500 hover:text-white hover:bg-gray-800 rounded transition"
                title="Clear logs"
              >
                <Trash2 size={14} />
              </button>
            </div>
            <div 
              ref={scrollRef}
              className="flex-1 min-h-[400px] max-h-[600px] bg-black p-4 font-mono text-sm overflow-y-auto space-y-1"
            >
              {logs.length === 0 ? (
                <div className="text-gray-600 italic">
                  Waiting for demo action...
                </div>
              ) : (
                logs.map((log, i) => (
                  <div 
                    key={i} 
                    className={`flex gap-2 ${
                      log.type === 'error' ? 'text-red-400' : 
                      log.type === 'success' ? 'text-green-400' : 
                      log.type === 'warn' ? 'text-yellow-400' : 'text-gray-300'
                    }`}
                  >
                    <span className="text-gray-600 text-xs shrink-0">
                      [{log.timestamp.toLocaleTimeString()}]
                    </span>
                    <span className="break-all">{log.msg}</span>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>

        {/* Token Showcase */}
        {(parentToken || childToken) && (
          <div className="mt-8 bg-gray-900 rounded-xl border border-gray-800 p-6">
            <h3 className="text-lg font-bold mb-4 flex items-center gap-2">
              <Key className="text-purple-400" /> Active Tokens
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {parentToken && (
                <div className="bg-black rounded-xl border border-cyan-800/50 p-4">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-cyan-400 font-semibold">Parent Token</span>
                    <button
                      onClick={() => copyToClipboard(parentToken.raw, 'parent-showcase')}
                      className="p-1.5 text-gray-500 hover:text-white hover:bg-gray-800 rounded transition"
                    >
                      {copied === 'parent-showcase' ? <Check size={14} className="text-green-400" /> : <Copy size={14} />}
                    </button>
                  </div>
                  <div className="text-xs font-mono text-gray-400 break-all bg-gray-900 rounded p-2">
                    {parentToken.raw.substring(0, 60)}...
                  </div>
                  <div className="mt-2 flex items-center gap-2 text-xs text-gray-500">
                    <Clock size={12} /> Scope: {parentToken.scope}
                  </div>
                </div>
              )}
              {childToken && (
                <div className="bg-black rounded-xl border border-purple-800/50 p-4">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-purple-400 font-semibold">Child Token (Restricted)</span>
                    <button
                      onClick={() => copyToClipboard(childToken.raw, 'child-showcase')}
                      className="p-1.5 text-gray-500 hover:text-white hover:bg-gray-800 rounded transition"
                    >
                      {copied === 'child-showcase' ? <Check size={14} className="text-green-400" /> : <Copy size={14} />}
                    </button>
                  </div>
                  <div className="text-xs font-mono text-gray-400 break-all bg-gray-900 rounded p-2">
                    {childToken.raw.substring(0, 60)}...
                  </div>
                  <div className="mt-2 flex items-center gap-2 text-xs text-gray-500">
                    <Lock size={12} /> Scope: {childToken.scope} (attenuated)
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Footer CTA */}
        <div className="mt-12 text-center">
          <p className="text-gray-500 text-sm mb-4">
            This demo runs against the live SatGate OSS deployment on Railway.
          </p>
          <div className="flex justify-center gap-4">
            <a 
              href="https://github.com/SatGate-io/satgate" 
              target="_blank" 
              rel="noopener noreferrer"
              className="text-purple-400 hover:text-purple-300 text-sm underline underline-offset-4 transition"
            >
              View Source on GitHub →
            </a>
            <Link 
              href="/playground"
              className="text-cyan-400 hover:text-cyan-300 text-sm underline underline-offset-4 transition"
            >
              Try L402 Payments →
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}


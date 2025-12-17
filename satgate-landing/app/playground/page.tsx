'use client';

import React, { useState, useRef, useEffect } from 'react';
import { Terminal, Zap, ShieldAlert, CheckCircle, Play, ArrowLeft, Wifi, WifiOff } from 'lucide-react';
import Link from 'next/link';

// --- ENDPOINT OPTIONS ---
const ENDPOINTS = [
  { path: '/api/micro/ping', price: 1, label: '/api/micro/ping (1 sat)' },
  { path: '/api/basic/quote', price: 10, label: '/api/basic/quote (10 sats)' },
  { path: '/api/standard/analytics', price: 100, label: '/api/standard/analytics (100 sats)' },
  { path: '/api/premium/insights', price: 1000, label: '/api/premium/insights (1000 sats)' },
];

// --- 1. MOCK CLIENT (Simulation) ---
class MockSatGateClient {
  async get() {
    await new Promise(r => setTimeout(r, 800));
    const error: any = new Error("Payment Required");
    error.status = 402; 
    error.headers = { get: () => 'L402 macaroon="mock_mac", invoice="lnbc10u1..."' };
    throw error;
  }
}

// --- 2. REAL CLIENT (WebLN / Alby) ---
class RealSatGateClient {
  async get(url: string, token?: string) {
    const headers: Record<string, string> = {};
    if (token) headers['Authorization'] = token;
    
    const res = await fetch(url, { headers, cache: 'no-store' });
    
    if (res.status === 402) {
      const error: any = new Error("Payment Required");
      error.status = 402;
      error.headers = res.headers;
      try { error.body = await res.text(); } catch(e) {} 
      throw error;
    }
    
    if (!res.ok) {
        const txt = await res.text();
        throw new Error(`Server Error ${res.status}: ${txt}`);
    }

    return res.json();
  }
}

export default function PlaygroundPage() {
  const [useRealNetwork, setUseRealNetwork] = useState(false);
  const [selectedEndpoint, setSelectedEndpoint] = useState(ENDPOINTS[0]); // Default to cheapest
  const [logs, setLogs] = useState<Array<{msg: string, type: 'info'|'error'|'success'|'warn'}>>([]);
  const [status, setStatus] = useState<'idle' | 'blocked' | 'paying' | 'success'>('idle');
  const [isLoading, setIsLoading] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  const addLog = (msg: string, type: 'info'|'error'|'success'|'warn' = 'info') => {
    setLogs(prev => [...prev, { msg, type }]);
  };

  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
  }, [logs]);

  const handleFetch = async () => {
    setIsLoading(true);
    setLogs([]);
    setStatus('idle');
    
    const BASE_URL = 'https://satgate-production.up.railway.app';
    const TARGET_URL = `${BASE_URL}${selectedEndpoint.path}`;

    try {
      addLog(`🚀 Initializing Client (${useRealNetwork ? 'REAL NETWORK' : 'SIMULATION'})...`, 'info');
      addLog(`📡 GET ${selectedEndpoint.path}`, 'info');
      addLog(`💵 Price: ${selectedEndpoint.price} sats`, 'info');

      let invoice = "";
      let macaroon = "";

      // --- STEP A: INITIAL REQUEST ---
      try {
        if (useRealNetwork) {
          const realClient = new RealSatGateClient();
          await realClient.get(TARGET_URL); 
        } else {
          const mockClient = new MockSatGateClient();
          await mockClient.get(); 
        }
        addLog("⚠️ Endpoint is already open (No 402 received).", 'warn');
        setStatus('success');
        return;

      } catch (err: any) {
        if (err.status === 402) {
            addLog('🛑 402 Payment Required received', 'warn');
            setStatus('blocked');

            const authHeader = err.headers.get('www-authenticate') || err.headers.get('WWW-Authenticate');
            if (useRealNetwork && !authHeader) {
                 console.error("Full Headers:", err.headers);
                 throw new Error("No L402 header found from backend.");
            }

            addLog('⚡ L402 Header detected. Parsing invoice...', 'info');
            
            if (useRealNetwork) {
               const macMatch = authHeader.match(/macaroon="([^"]+)"/);
               const invMatch = authHeader.match(/invoice="([^"]+)"/);
               if(macMatch && invMatch) {
                   macaroon = macMatch[1];
                   invoice = invMatch[1];
                   // Show truncated invoice
                   addLog(`📜 Invoice: ${invoice.substring(0, 20)}...${invoice.substring(invoice.length - 10)}`, 'info');
               } else {
                   throw new Error(`Invalid header format: ${authHeader}`);
               }
            } else {
               invoice = "lnbc10u1p3qj...xyz"; 
               addLog(`📜 Invoice: ${invoice}`, 'info');
               await new Promise(r => setTimeout(r, 800)); 
            }
        } else {
            throw err;
        }
      }

      // --- STEP B: PAYMENT ---
      setStatus('paying');
      let preimage = "";

      if (useRealNetwork) {
        addLog(`💸 Launching WebLN (Alby) to pay ${selectedEndpoint.price} sats...`, 'info');
        
        // @ts-ignore
        if (typeof window.webln === 'undefined') {
            throw new Error("WebLN not found. Please install Alby extension.");
        }
        // @ts-ignore
        await window.webln.enable();
        // @ts-ignore
        const payment = await window.webln.sendPayment(invoice);
        preimage = payment.preimage;
        
        addLog(`✅ Payment Sent! Preimage: ${preimage.substring(0,10)}...`, 'success');
      } else {
        addLog(`💸 Paying Invoice (${selectedEndpoint.price} sats)...`, 'info');
        await new Promise(r => setTimeout(r, 1500)); 
        preimage = "mock_preimage_123";
        addLog('✅ Payment Confirmed. Preimage secured.', 'success');
      }

      // --- STEP C: RETRY WITH AUTH DISCOVERY ---
      addLog('🔄 Retrying request with L402 Token...', 'info');
      
      if (useRealNetwork) {
          const realClient = new RealSatGateClient();
          
          const candidates = [
              `LSAT ${macaroon}:${preimage}`,
              `L402 ${macaroon}:${preimage}`,
              `LSAT ${btoa(macaroon + ':' + preimage)}`,
              `L402 ${btoa(macaroon + ':' + preimage)}`
          ];

          let success = false;
          let lastError;

          for (const token of candidates) {
              try {
                  const finalRes = await realClient.get(TARGET_URL, token);
                  addLog('✅ 200 OK: Request Authorized.', 'success');
                  addLog(`📦 Payload: ${JSON.stringify(finalRes)}`, 'success');
                  success = true;
                  break;
              } catch (e: any) {
                  if (e.status === 402) {
                      lastError = e;
                      continue;
                  } else {
                      throw e;
                  }
              }
          }

          if (!success) {
              throw new Error(`All Auth formats failed. Last error: ${lastError?.message}`);
          }

      } else {
          await new Promise(r => setTimeout(r, 800));
          addLog('✅ 200 OK: Request Authorized.', 'success');
          addLog('📦 Payload: { "market_sentiment": "bullish", "confidence": 0.98 }', 'success');
      }
      
      setStatus('success');

    } catch (err: any) {
      setStatus('blocked');
      const errMsg = err.message || String(err);
      
      if (errMsg.includes('WebLN not found') || errMsg.includes('webln')) {
        // No wallet extension installed
        addLog(`❌ No Lightning Wallet Detected`, 'error');
        addLog(``, 'info');
        addLog(`📥 To pay real invoices, install the Alby browser extension:`, 'warn');
        addLog(`   👉 https://getalby.com`, 'warn');
        addLog(``, 'info');
        addLog(`💡 After installing, refresh this page and try again.`, 'info');
        addLog(`💡 Or use SIMULATION MODE to test without a wallet.`, 'info');
      } else if (errMsg === 'Load failed' || errMsg.includes('Failed to fetch')) {
        // Network error
        addLog(`❌ Network Error: Could not reach the server`, 'error');
        addLog(``, 'info');
        addLog(`⚠️  The cloud server may be temporarily unavailable.`, 'warn');
        addLog(``, 'info');
        addLog(`Try again in a moment, or use SIMULATION MODE.`, 'info');
      } else if (errMsg.includes('User rejected') || errMsg.includes('cancelled')) {
        addLog('⚠️ Payment Cancelled by User.', 'warn');
        setStatus('idle');
      } else {
        addLog(`❌ Error: ${errMsg}`, 'error');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans flex flex-col items-center py-12 px-4">
      
      {/* Header */}
      <div className="w-full max-w-3xl mb-8 flex items-center justify-between">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition">
          <ArrowLeft size={18} /> Back to Home
        </Link>
        <h1 className="text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-purple-400 to-cyan-400">
          ⚡ SatGate™ Playground
        </h1>
      </div>

      {/* Main Console UI */}
      <div className="w-full max-w-3xl bg-gray-900 rounded-xl border border-gray-800 shadow-2xl overflow-hidden relative">
        
        {/* Glow Effects */}
        <div className={`absolute top-0 left-0 w-full h-1 bg-gradient-to-r transition-all duration-500
          ${status === 'idle' ? 'from-gray-800 to-gray-800' : ''}
          ${status === 'blocked' ? 'from-red-500 via-orange-500 to-red-500 animate-pulse' : ''}
          ${status === 'paying' ? 'from-purple-500 via-cyan-500 to-purple-500 animate-pulse' : ''}
          ${status === 'success' ? 'from-green-500 to-emerald-400' : ''}
        `} />

        {/* Toolbar */}
        <div className="p-4 border-b border-gray-800 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 bg-gray-900/50 backdrop-blur">
          
          {/* Left Side: Mode Toggle + Endpoint Selector */}
          <div className="flex flex-col sm:flex-row items-start sm:items-center gap-3 w-full sm:w-auto">
            {/* Mode Toggle */}
            <div className="flex flex-col gap-1">
              <button 
                  onClick={() => setUseRealNetwork(!useRealNetwork)}
                  className={`flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-bold transition-all border
                  ${useRealNetwork 
                      ? 'bg-purple-900/30 border-purple-500 text-purple-300' 
                      : 'bg-gray-800 border-gray-600 text-gray-400'}`}
              >
                  {useRealNetwork ? <Wifi size={14} /> : <WifiOff size={14} />}
                  {useRealNetwork ? "REAL NETWORK" : "SIMULATION"}
              </button>
              {useRealNetwork && (
                <span className="text-[10px] text-green-500/70">✓ Live Cloud Server</span>
              )}
            </div>

            {/* Endpoint Selector */}
            <div className="flex items-center gap-2">
              <span className="px-2 py-1 bg-emerald-900/50 text-emerald-300 text-xs font-bold rounded border border-emerald-800">GET</span>
              <select 
                value={selectedEndpoint.path}
                onChange={(e) => {
                  const ep = ENDPOINTS.find(ep => ep.path === e.target.value);
                  if (ep) setSelectedEndpoint(ep);
                }}
                className="bg-gray-800 text-gray-300 border border-gray-700 rounded-lg px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-purple-500"
              >
                {ENDPOINTS.map(ep => (
                  <option key={ep.path} value={ep.path}>{ep.label}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Run Button */}
          <button 
            onClick={handleFetch}
            disabled={isLoading}
            className={`
              flex items-center gap-2 px-6 py-2 rounded-lg font-bold text-sm transition-all
              ${isLoading ? 'bg-gray-800 text-gray-500 cursor-not-allowed' : 'bg-white text-black hover:bg-gray-200'}
            `}
          >
            {isLoading ? 'Processing...' : <><Play size={16} fill="black" /> Run Request</>}
          </button>
        </div>

        {/* Terminal Output */}
        <div ref={scrollRef} className="h-96 bg-black p-6 font-mono text-sm overflow-y-auto space-y-2">
            {logs.length === 0 && (
                <div className="text-gray-600 italic">
                    {useRealNetwork 
                        ? "🌐 LIVE MODE: Connected to cloud server. Install Alby to pay real Lightning invoices." 
                        : "Ready to simulate... Click 'Run Request' to start."}
                </div>
            )}
            {logs.map((log, i) => (
                <div key={i} className={`flex gap-3 ${
                log.type === 'error' ? 'text-red-400' : 
                log.type === 'success' ? 'text-green-400' : 
                log.type === 'warn' ? 'text-yellow-400' : 'text-gray-300'
                }`}>
                <span className="opacity-30">[{new Date().toLocaleTimeString()}]</span>
                <span>{log.msg}</span>
                </div>
            ))}
        </div>
      </div>
      
      {/* Visual Status Pipeline */}
      <div className="w-full max-w-3xl mt-8 grid grid-cols-3 gap-4">
        <StatusStep active={status === 'blocked'} completed={status === 'paying' || status === 'success'} icon={<ShieldAlert size={20} />} label="1. 402 Blocked" />
        <StatusStep active={status === 'paying'} completed={status === 'success'} icon={<Zap size={20} />} label="2. Lightning Payment" />
        <StatusStep active={status === 'success'} completed={status === 'success'} icon={<CheckCircle size={20} />} label="3. Data Unlocked" />
      </div>

    </div>
  );
}

const StatusStep = ({ active, completed, icon, label }: any) => (
  <div className={`
    p-4 rounded-xl border flex items-center justify-center gap-3 transition-all duration-500
    ${active ? 'bg-gray-800 border-white text-white scale-105 shadow-lg shadow-purple-500/20' : ''}
    ${completed ? 'bg-gray-900 border-green-900 text-green-500' : ''}
    ${!active && !completed ? 'bg-black border-gray-800 text-gray-600' : ''}
  `}>
    {icon}
    <span className="font-bold">{label}</span>
  </div>
);

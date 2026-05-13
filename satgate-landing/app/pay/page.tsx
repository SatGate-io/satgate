'use client';
// v2.2 - Manual preimage entry with improved UX

import React, { useState, useRef, useEffect } from 'react';
import { Terminal, Zap, ShieldAlert, CheckCircle, Play, ArrowLeft, Wifi, WifiOff, Copy, Check } from 'lucide-react';
import Link from 'next/link';

// --- ENDPOINT OPTIONS ---
// Note: 1 sat may fail on some wallet routes - demonstrates fallback to manual payment
const ENDPOINTS = [
  { path: '/api/micro/ping', price: 1, label: '/api/micro/ping (1 sat)' },
  { path: '/api/basic/quote', price: 10, label: '/api/basic/quote (10 sats)' },
  { path: '/api/standard/analytics', price: 100, label: '/api/standard/analytics (100 sats)' },
  { path: '/api/premium/insights', price: 1000, label: '/api/premium/insights (1000 sats)' },
];

const webPageJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'WebPage',
  name: 'SatGate paid-rail governance Demo',
  url: 'https://satgate.io/pay',
  description: 'Interactive paid-rail demo showing HTTP 402 challenges, Lightning invoices, paid-call receipts, Evidence Pack proof, and request-path API access for paid agents.',
  datePublished: '2026-04-12',
  dateModified: '2026-05-03',
  isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
  about: [
    { '@type': 'Thing', name: 'SatGate paid-rail governance' },
    { '@type': 'Thing', name: 'L402 payment flow' },
    { '@type': 'Thing', name: 'HTTP 402 Payment Required' },
    { '@type': 'Thing', name: 'Lightning invoices for APIs' },
    { '@type': 'Thing', name: 'paid agent API access' },
  ],
};

const softwareJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'SatGate paid-rail governance Demo',
  applicationCategory: 'DeveloperApplication',
  operatingSystem: 'Web',
  url: 'https://satgate.io/pay',
  description: webPageJsonLd.description,
  publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
  dateModified: '2026-05-03',
  featureList: ['HTTP 402 challenge simulation', 'paid-rail context invoice flow', 'Payment proof retry', 'Paid-call receipt creation', 'Evidence Pack proof'],
};

const faqJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'FAQPage',
  mainEntity: [
    {
      '@type': 'Question',
      name: 'What happens during an L402 payment flow?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'An agent requests a protected API, receives HTTP 402 Payment Required with an L402 challenge, pays the Lightning invoice, then retries with proof of payment; SatGate returns a paid-call receipt and records the decision for the Evidence Pack.',
      },
    },
    {
      '@type': 'Question',
      name: 'Why use L402 for agent API access?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'L402 lets autonomous agents pay at request time without subscriptions, credit cards, or long-lived API keys, making API access native to paid agents.',
      },
    },
    {
      '@type': 'Question',
      name: 'Can paid-rail context be combined with access policy?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Yes. SatGate can combine L402 payment with capability tokens, scoped authorization, budget policy, receipt fields, Evidence Pack export, and revocation so payment does not become unrestricted access.',
      },
    },
  ],
};

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
      try { 
        error.body = await res.text();
        console.log('RealSatGateClient: 402 body:', error.body);
      } catch(e) {
        console.error('RealSatGateClient: Failed to read body:', e);
      } 
      throw error;
    }
    
    if (!res.ok) {
        const txt = await res.text();
        throw new Error(`Server Error ${res.status}: ${txt}`);
    }

    return res.json();
  }
}

export default function PayDemoPage() {
  const [useRealNetwork, setUseRealNetwork] = useState(false);
  const [selectedEndpoint, setSelectedEndpoint] = useState(ENDPOINTS[0]); // Default to cheapest
  const [logs, setLogs] = useState<Array<{msg: string, type: 'info'|'error'|'success'|'warn'}>>([]);
  const [status, setStatus] = useState<'idle' | 'blocked' | 'paying' | 'success'>('idle');
  const [isLoading, setIsLoading] = useState(false);
  const [showInvoice, setShowInvoice] = useState<{invoice: string, macaroon: string, price: number, paymentHash: string} | null>(null);
  const [preimageInput, setPreimageInput] = useState('');
  const [copied, setCopied] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const preimageResolverRef = useRef<((value: string) => void) | null>(null);
  
  const copyToClipboard = async (text: string) => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  
  const submitPreimage = () => {
    if (preimageInput.length >= 64 && preimageResolverRef.current) {
      preimageResolverRef.current(preimageInput);
      preimageResolverRef.current = null;
    }
  };

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
    
    const BASE_URL = 'https://satgate-production-9354.up.railway.app';
    const TARGET_URL = `${BASE_URL}${selectedEndpoint.path}`;

    try {
      addLog(`🚀 Initializing Client (${useRealNetwork ? 'REAL NETWORK' : 'SIMULATION'})...`, 'info');
      addLog(`📡 GET ${selectedEndpoint.path}`, 'info');
      addLog(`💵 Price: ${selectedEndpoint.price} sats`, 'info');

      let invoice = "";
      let macaroon = "";
      let paymentHash = "";

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
                   
                   // Try to get payment_hash from response body
                   try {
                     console.log('Response body:', err.body);
                     if (err.body) {
                       const bodyJson = JSON.parse(err.body);
                       paymentHash = bodyJson.payment_hash || '';
                       if (paymentHash) {
                         addLog(`🔑 Payment hash: ${paymentHash.substring(0, 16)}...`, 'info');
                       }
                     } else {
                       console.log('No response body found on error object');
                     }
                   } catch (parseErr) { 
                     console.error('Failed to parse response body:', parseErr);
                   }
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
        // Try WebLN first (Alby), fall back to manual payment
        let webLNAvailable = false;
        try {
          // @ts-ignore
          if (typeof window.webln !== 'undefined') {
            // @ts-ignore
            await window.webln.enable();
            webLNAvailable = true;
          }
        } catch (e) {
          // WebLN enable failed
        }

        if (webLNAvailable) {
          try {
            addLog(`💸 Launching WebLN (Alby) to pay ${selectedEndpoint.price} sats...`, 'info');
            // @ts-ignore
            const payment = await window.webln.sendPayment(invoice);
            preimage = payment.preimage;
            
            // Normalize preimage to hex format (some wallets return base64)
            if (preimage && !/^[a-fA-F0-9]{64}$/.test(preimage)) {
              // Try to decode as base64
              try {
                const decoded = atob(preimage);
                preimage = Array.from(decoded).map(c => c.charCodeAt(0).toString(16).padStart(2, '0')).join('');
                addLog(`🔄 Converted preimage from base64 to hex`, 'info');
              } catch {
                addLog(`⚠️ Preimage format: ${preimage.length} chars, not standard hex`, 'warn');
              }
            }
            
            addLog(`✅ Payment Sent! Preimage: ${preimage?.substring(0, 16)}...`, 'success');
          } catch (e: any) {
            if (e.message?.includes('rejected') || e.message?.includes('cancelled')) {
              throw e; // User cancelled
            }
            // WebLN payment failed - show manual invoice panel
            addLog(`⚠️ WebLN payment failed: ${e.message}`, 'warn');
            addLog(`📱 Showing invoice for manual payment...`, 'info');
            addLog(`💡 Pay the invoice, then paste the preimage below`, 'info');
            
            setShowInvoice({ invoice, macaroon, price: selectedEndpoint.price, paymentHash });
            
            // Wait for manual preimage entry
            preimage = await new Promise<string>((resolve, reject) => {
              preimageResolverRef.current = resolve;
              setTimeout(() => {
                if (preimageResolverRef.current) {
                  preimageResolverRef.current = null;
                  reject(new Error('Payment timeout'));
                }
              }, 600000); // 10 minute timeout
            });
            addLog(`✅ Preimage received!`, 'success');
            setShowInvoice(null);
            setPreimageInput('');
          }
        } else {
          // No WebLN - show invoice panel for manual payment
          addLog(`📱 No WebLN wallet detected. Showing invoice...`, 'info');
          addLog(`💡 Pay the invoice, then paste the preimage below`, 'info');
          
          setShowInvoice({ invoice, macaroon, price: selectedEndpoint.price, paymentHash });
          
          // Wait for manual preimage entry
          preimage = await new Promise<string>((resolve, reject) => {
            preimageResolverRef.current = resolve;
            setTimeout(() => {
              if (preimageResolverRef.current) {
                preimageResolverRef.current = null;
                reject(new Error('Payment timeout'));
              }
            }, 600000); // 10 minute timeout
          });
          addLog(`✅ Preimage received!`, 'success');
          setShowInvoice(null);
          setPreimageInput('');
        }
      } else {
        addLog(`💸 Paying Invoice (${selectedEndpoint.price} sats)...`, 'info');
        await new Promise(r => setTimeout(r, 1500)); 
        preimage = "mock_preimage_123";
        addLog('✅ Payment confirmed. Paid-call receipt created.', 'success');
      }

      // --- STEP C: RETRY WITH AUTH DISCOVERY ---
      addLog('🔄 Retrying request with L402 Token...', 'info');
      
      if (useRealNetwork) {
          const realClient = new RealSatGateClient();
          
          // Debug: show token components
          addLog(`🔑 Macaroon: ${macaroon.substring(0, 30)}...`, 'info');
          addLog(`🔑 Preimage: ${preimage} (${preimage.length} chars)`, 'info');
          
          // Only try the standard formats (base64 encoding is not expected by server)
          const candidates = [
              { format: 'LSAT', token: `LSAT ${macaroon}:${preimage}` },
              { format: 'L402', token: `L402 ${macaroon}:${preimage}` },
          ];

          let success = false;
          let lastError: any;
          let lastErrorBody = '';

          for (const { format, token } of candidates) {
              try {
                  addLog(`🔄 Trying ${format} format...`, 'info');
                  const finalRes = await realClient.get(TARGET_URL, token);
                  addLog('✅ 200 OK: Request Authorized + Receipt Returned.', 'success');
                  addLog(`📦 Payload: ${JSON.stringify(finalRes)}`, 'success');
                  addLog('🧾 Paid-call receipt queued for Evidence Pack.', 'success');
                  success = true;
                  break;
              } catch (e: any) {
                  if (e.status === 402) {
                      lastError = e;
                      lastErrorBody = e.body || '';
                      addLog(`❌ ${format} rejected (402)`, 'warn');
                      continue;
                  } else {
                      throw e;
                  }
              }
          }

          if (!success) {
              // Parse and show the error details
              let errorDetail = 'Unknown';
              try {
                  const parsed = JSON.parse(lastErrorBody);
                  errorDetail = parsed.previousError || parsed.reason || parsed.message || 'See console';
              } catch { 
                  errorDetail = lastErrorBody.substring(0, 100) || lastError?.message;
              }
              throw new Error(`Auth failed: ${errorDetail}`);
          }

      } else {
          await new Promise(r => setTimeout(r, 800));
          addLog('✅ 200 OK: Request Authorized + Receipt Returned.', 'success');
          addLog('📦 Payload: { "market_sentiment": "bullish", "confidence": 0.98, "receipt_id": "rcpt_paid_demo_001" }', 'success');
          addLog('🧾 Paid-call receipt queued for Evidence Pack.', 'success');
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
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      {/* Header */}
      <div className="border-b border-gray-800 bg-black/80 backdrop-blur-sm sticky top-0 z-40">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition">
            <ArrowLeft size={18} /> <span className="hidden sm:inline">Back to Home</span>
          </Link>
          <h1 className="text-lg sm:text-xl font-bold flex items-center gap-2">
            <Zap className="text-yellow-400" size={24} />
            <span className="bg-clip-text text-transparent bg-gradient-to-r from-yellow-400 to-orange-400">
              Charge Demo
            </span>
          </h1>
          <div className="w-[120px]"></div>
        </div>
      </div>

      {/* Hero Description */}
      <div className="bg-gradient-to-b from-yellow-950/20 to-transparent border-b border-gray-800/50">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8 text-center">
          <h2 className="text-2xl sm:text-3xl font-bold mb-4">
            <span className="bg-clip-text text-transparent bg-gradient-to-r from-yellow-400 via-orange-400 to-yellow-400">
              Per-request Receipts. Zero Contracts. Proof in Seconds.
            </span>
          </h2>
          <p className="text-gray-400 text-lg leading-relaxed max-w-2xl mx-auto">
            Watch an AI agent <strong className="text-white">pay for API access in real-time</strong> using 
            Bitcoin Lightning. No credit cards. No monthly bills. Just instant, per-request micropayments 
            that return receipts and feed Evidence Packs.
          </p>
        </div>
      </div>

      <div className="flex flex-col items-center py-8 px-4">

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

      <section className="w-full max-w-3xl mt-12 border-t border-gray-800 pt-10">
        <p className="mb-2 text-center text-xs font-mono uppercase tracking-wide text-yellow-300">FAQ</p>
        <h2 className="mb-8 text-center text-2xl font-bold text-white">L402 payment flow questions</h2>
        <div className="grid gap-4 md:grid-cols-3">
          {[
            ['What happens during an L402 payment flow?', 'An agent requests a protected API, receives HTTP 402 Payment Required with an L402 challenge, pays the Lightning invoice, then retries with proof of payment; SatGate returns a paid-call receipt and records the decision for the Evidence Pack.'],
            ['Why use L402 for agent API access?', 'L402 lets autonomous agents pay at request time without subscriptions, credit cards, or long-lived API keys, making API access native to paid agents.'],
            ['Can paid-rail context be combined with access policy?', 'Yes. SatGate can combine L402 payment with capability tokens, scoped authorization, budget policy, receipt fields, Evidence Pack export, and revocation so payment does not become unrestricted access.'],
          ].map(([question, answer]) => (
            <div key={question} className="rounded-xl border border-gray-800 bg-gray-900 p-5">
              <h3 className="mb-2 font-bold text-white">{question}</h3>
              <p className="text-sm leading-relaxed text-gray-400">{answer}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Invoice Panel Modal */}
      {showInvoice && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4 overflow-y-auto">
          <div className="bg-gray-900 border border-purple-500 rounded-xl max-w-lg w-full p-6 space-y-4 my-4">
            <h3 className="text-xl font-bold text-white flex items-center gap-2">
              <Zap className="text-yellow-400" /> Pay {showInvoice.price} sats
            </h3>
            
            <p className="text-gray-400 text-sm">
              Scan with any Lightning wallet (Phoenix, Wallet of Satoshi, etc.)
            </p>
            
            {/* QR Code */}
            <div className="flex justify-center bg-white rounded-lg p-4">
              <img 
                src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(showInvoice.invoice)}`}
                alt="Lightning Invoice QR Code"
                className="w-48 h-48"
              />
            </div>
            
            {/* Invoice display with copy button */}
            <div className="bg-black rounded-lg p-3 font-mono text-xs break-all text-gray-300 relative">
              <div className="pr-10 max-h-20 overflow-y-auto">{showInvoice.invoice}</div>
              <button 
                onClick={() => copyToClipboard(showInvoice.invoice)}
                className="absolute top-2 right-2 p-2 bg-gray-800 hover:bg-gray-700 rounded transition"
                title="Copy invoice"
              >
                {copied ? <Check size={16} className="text-green-400" /> : <Copy size={16} />}
              </button>
            </div>
            
            {/* Manual Preimage input */}
            <div className="space-y-2">
              <label className="text-sm text-gray-400">
                After paying, paste the <span className="text-purple-400 font-medium">PREIMAGE</span> from your wallet:
              </label>
              <input 
                type="text"
                value={preimageInput}
                onChange={(e) => setPreimageInput(e.target.value)}
                placeholder="64-character hex preimage..."
                className="w-full bg-black border border-gray-700 rounded-lg px-4 py-3 font-mono text-sm text-white focus:border-purple-500 focus:outline-none"
              />
              <p className="text-xs text-gray-500">
                💡 In Phoenix: tap the payment → "Details" → copy "Preimage"
              </p>
            </div>
            
            <div className="flex gap-3">
              <button 
                onClick={() => { 
                  setShowInvoice(null); 
                  setPreimageInput(''); 
                }}
                className="flex-1 px-4 py-2 bg-gray-800 text-gray-300 rounded-lg hover:bg-gray-700 transition"
              >
                Cancel
              </button>
              <button 
                onClick={submitPreimage}
                disabled={preimageInput.length < 64}
                className={`flex-1 px-4 py-2 rounded-lg font-bold transition ${
                  preimageInput.length >= 64 
                    ? 'bg-purple-600 text-white hover:bg-purple-500' 
                    : 'bg-gray-800 text-gray-500 cursor-not-allowed'
                }`}
              >
                Submit Preimage
              </button>
            </div>
          </div>
        </div>
      )}

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

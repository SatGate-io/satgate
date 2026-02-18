'use client';

import React, { useState } from 'react';
import { Terminal, Code, Cpu, Zap, ArrowRight, CheckCircle, Copy, Check, Shield, Key, Lock, Clock, DollarSign, Bot, GitBranch, Activity, RefreshCw, Menu, X, Eye, SlidersHorizontal } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

const LandingPage = () => {
  const [copied, setCopied] = useState(false);
  const [activeTab, setActiveTab] = useState<'python' | 'nodejs' | 'curl'>('python');
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const copyToClipboard = async () => {
    await navigator.clipboard.writeText('pip install satgate');
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans selection:bg-purple-500 selection:text-white">
      
      {/* Navigation */}
      <nav className="border-b border-gray-800 backdrop-blur-md fixed w-full z-50 bg-black/50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2 shrink-0">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="w-7 h-7 sm:w-8 sm:h-8" />
            <span className="text-lg sm:text-xl font-bold text-white whitespace-nowrap">SatGate<sup className="text-xs font-normal">™</sup></span>
          </Link>
          
          {/* Desktop menu */}
          <div className="hidden md:flex gap-6 text-sm font-medium text-gray-400">
            <Link href="/protect" className="hover:text-white transition">Protect Demo</Link>
            <Link href="/pay" className="hover:text-white transition">Pay Demo</Link>
            <Link href="/govern" className="hover:text-white transition">Govern</Link>
            <Link href="/pricing" className="hover:text-white transition">Pricing</Link>
            <Link href="/roi-calculator" className="hover:text-white transition">ROI Calculator</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud</a>
          </div>
          
          {/* Mobile menu button */}
          <button 
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="flex md:hidden items-center justify-center w-10 h-10 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 text-gray-400 hover:text-white transition"
            aria-label="Toggle menu"
            aria-expanded={mobileMenuOpen}
          >
            {mobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
        </div>
        
        {/* Mobile menu dropdown */}
        <div 
          className={`md:hidden overflow-hidden transition-all duration-300 ease-in-out ${
            mobileMenuOpen ? 'max-h-80 opacity-100' : 'max-h-0 opacity-0'
          }`}
        >
          <div className="bg-black/95 backdrop-blur-xl border-t border-gray-800 px-4 py-4 space-y-1">
            <Link 
              href="/protect" 
              onClick={() => setMobileMenuOpen(false)} 
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Protect Demo
            </Link>
            <Link 
              href="/pay" 
              onClick={() => setMobileMenuOpen(false)} 
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Pay Demo
            </Link>
            <Link 
              href="/govern" 
              onClick={() => setMobileMenuOpen(false)} 
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Govern
            </Link>
            <Link 
              href="/pricing" 
              onClick={() => setMobileMenuOpen(false)} 
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Pricing
            </Link>
            <a 
              href="https://cloud.satgate.io/docs" 
              target="_blank" 
              rel="noopener noreferrer"
              onClick={() => setMobileMenuOpen(false)} 
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Docs
            </a>
            <a 
              href="https://github.com/SatGate-io/satgate" 
              target="_blank" 
              rel="noopener noreferrer"
              onClick={() => setMobileMenuOpen(false)} 
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              GitHub
            </a>
            <a 
              href="https://cloud.satgate.io/cloud/login" 
              target="_blank" 
              rel="noopener noreferrer"
              onClick={() => setMobileMenuOpen(false)} 
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Cloud
            </a>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <header className="pt-32 pb-20 px-6">
        <div className="max-w-6xl mx-auto grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
          
          {/* Left: Copy */}
          <div>
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono mb-6">
              <Zap size={12} /> Economic Access Control — the security primitive for the agent economy
            </div>
            <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
              The Economic Firewall<br/>
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
                for AI Agent Requests.
              </span>
            </h1>
            <p className="text-xl text-gray-400 mb-4 max-w-lg leading-relaxed">
              SatGate is an Economic Firewall that enforces <b className="text-white">Economic Access Control</b> for AI agent requests.
            </p>
            <p className="text-lg text-gray-500 mb-8 max-w-lg leading-relaxed">
              Per-agent budgets, per-tool cost attribution, delegation hierarchies. Not &ldquo;who are you?&rdquo; — <b>&ldquo;what can you afford?&rdquo;</b> Math doesn&apos;t care how smart the AI is.
            </p>
            <div className="flex gap-4">
              <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="bg-white text-black px-8 py-3 rounded-lg font-bold hover:bg-gray-200 transition flex items-center gap-2">
                Start Free →
              </a>
              <button 
                onClick={copyToClipboard}
                className="border border-gray-700 px-8 py-3 rounded-lg font-bold hover:border-gray-500 transition font-mono flex items-center gap-2 group"
              >
                pip install satgate
                {copied ? (
                  <Check size={16} className="text-green-400" />
                ) : (
                  <Copy size={16} className="text-gray-500 group-hover:text-gray-300 transition" />
                )}
              </button>
            </div>
          </div>

          {/* Right: Hero Demo Video */}
          <div className="relative group">
            <div className="absolute -inset-1 bg-gradient-to-r from-purple-600 to-cyan-600 rounded-xl blur opacity-25 group-hover:opacity-50 transition duration-1000 pointer-events-none"></div>
            <div className="relative bg-gray-900 rounded-xl border border-gray-800 overflow-hidden shadow-2xl">
              <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-800 bg-gray-900/80">
                <div className="w-3 h-3 rounded-full bg-red-500"></div>
                <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
                <div className="w-3 h-3 rounded-full bg-green-500"></div>
                <div className="text-xs text-gray-500 ml-2 font-mono">hero_demo.py — Live Demo</div>
              </div>
              <video 
                autoPlay 
                loop 
                muted 
                playsInline
                className="w-full"
              >
                <source src="/satgate-hero-demo.mp4" type="video/mp4" />
                {/* Fallback for browsers that don't support video */}
                <img src="/satgate-hero-demo.gif" alt="SatGate Demo" className="w-full" />
              </video>
              <div className="absolute bottom-4 right-4 bg-black/70 backdrop-blur-sm px-3 py-1 rounded-full text-xs text-gray-300 font-mono">
                ⚡ Metered in sats
              </div>
            </div>
            <div className="text-center mt-4">
              <p className="text-sm text-gray-500 mb-3">
                Watch an AI agent autonomously pay for API access in real-time.
              </p>
              <Link 
                href="/playground" 
                className="inline-flex items-center gap-2 text-sm font-medium text-purple-400 hover:text-purple-300 transition"
              >
                Try it live in 15 seconds <ArrowRight size={14} />
              </Link>
            </div>
          </div>
        </div>
      </header>

      {/* RBAC vs EAC Comparison */}
      <section className="py-16 px-6 border-b border-gray-800 bg-gradient-to-b from-purple-950/10 to-black">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-10">
            <h2 className="text-2xl font-bold mb-3">RBAC Was Built for Humans. EAC Is Built for Agents.</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              Stop giving AI agents an all-you-can-eat buffet pass. You can&apos;t out-smart an AI swarm, but you can hard-cap its wallet.
            </p>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-sm border-collapse">
              <thead>
                <tr className="border-b border-gray-800">
                  <th className="text-left py-3 px-4 text-gray-500 font-medium">Feature</th>
                  <th className="text-left py-3 px-4 text-red-400 font-medium">RBAC <span className="text-gray-600 font-normal">(Legacy)</span></th>
                  <th className="text-left py-3 px-4 text-green-400 font-medium">EAC <span className="text-gray-600 font-normal">(SatGate)</span></th>
                </tr>
              </thead>
              <tbody className="text-gray-400">
                <tr className="border-b border-gray-800/50">
                  <td className="py-4 px-4 text-white font-medium">Core Metric</td>
                  <td className="py-4 px-4"><span className="text-red-400/80">Identity &amp; Title</span> — static, context-blind</td>
                  <td className="py-4 px-4"><span className="text-green-400">Budgets &amp; Macaroons</span> — dynamic, self-enforcing</td>
                </tr>
                <tr className="border-b border-gray-800/50">
                  <td className="py-4 px-4 text-white font-medium">Agent Behavior</td>
                  <td className="py-4 px-4">Spams most powerful tool available</td>
                  <td className="py-4 px-4"><span className="text-green-400">Naturally optimizes for efficiency</span></td>
                </tr>
                <tr className="border-b border-gray-800/50">
                  <td className="py-4 px-4 text-white font-medium">Risk Management</td>
                  <td className="py-4 px-4">Context blind — runaway agent can drop DB in seconds</td>
                  <td className="py-4 px-4"><span className="text-green-400">Self-healing</span> — high-risk actions cost more, loops bankrupt themselves</td>
                </tr>
                <tr>
                  <td className="py-4 px-4 text-white font-medium">Financial Impact</td>
                  <td className="py-4 px-4">IT = Cost Center</td>
                  <td className="py-4 px-4"><span className="text-green-400">Internal tools = measurable Profit Centers</span></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Two Products Section */}
      <section className="py-16 px-6 border-b border-gray-800">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-2xl font-bold text-center mb-3">Two Proxies. Total Coverage.</h2>
          <p className="text-gray-500 text-center mb-10">Govern HTTP APIs and MCP tool calls — or both.</p>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="p-6 rounded-xl bg-gradient-to-br from-cyan-950/30 to-cyan-900/10 border border-cyan-800/30 hover:border-cyan-600/50 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-cyan-900/50 rounded-lg">
                  <Shield className="text-cyan-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">HTTP Proxy Gateway</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-3">
                Drop-in reverse proxy for REST APIs, LLM endpoints, and SaaS services. Capability tokens replace API keys. Per-route policies with zero code changes.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ One DNS change deployment</li>
                <li>✓ Cryptographic access control (Macaroons)</li>
                <li>✓ L402 monetization + Fiat402 budgets</li>
              </ul>
            </div>
            <div className="p-6 rounded-xl bg-gradient-to-br from-purple-950/30 to-purple-900/10 border border-purple-800/30 hover:border-purple-600/50 transition relative">
              <div className="absolute top-4 right-4 px-2 py-0.5 rounded-full bg-purple-900/50 border border-purple-700/50 text-purple-300 text-xs font-bold">
                NEW
              </div>
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-purple-900/50 rounded-lg">
                  <DollarSign className="text-purple-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">MCP Proxy</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-3">
                Intercepts every MCP tool call at the JSON-RPC level. Per-tool cost profiles, hard budget enforcement, and delegation hierarchies for sub-agents.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Per-tool cost attribution (web_search: 10cr, code_execute: 50cr)</li>
                <li>✓ Budget delegation — parent agents carve sub-budgets atomically</li>
                <li>✓ Works with Claude Code, Agent Zero, Cursor, any MCP framework</li>
              </ul>
            </div>
          </div>
          
          <p className="text-center text-gray-600 text-xs mt-8">
            Both modes use the same token system (Macaroons), same dashboard, same delegation hierarchy.
          </p>
        </div>
      </section>

      {/* Default Protection + Economic Policies */}
      <section className="py-20 px-6 border-b border-gray-800 bg-gradient-to-b from-gray-900/30 to-black">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-3">Default Protection + Economic Policies</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">Protection is the foundation. Choose your economic policy per route.</p>
          </div>

          {/* Default Protection - Foundation */}
          <div className="p-8 rounded-xl bg-black border-2 border-purple-500/50 mb-6 relative">
            <div className="absolute -top-3 left-6 bg-purple-600 text-xs font-bold px-3 py-1 rounded">
              DEFAULT PROTECTION
            </div>
            <div className="flex items-center gap-3 mb-4 mt-2">
              <div className="p-3 bg-purple-900/30 rounded-lg">
                <Shield className="text-purple-400" size={24} />
              </div>
              <div>
                <h3 className="text-xl font-bold">Cryptographic Capability Verification</h3>
                <p className="text-gray-500 text-sm">Always-on for non-PUBLIC routes</p>
              </div>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              Every protected route requires valid credentials (Macaroons). Capabilities, caveats, 
              delegation, and revocation—built into the protocol, not bolted on.
            </p>
            <div className="flex flex-wrap gap-4 text-sm text-gray-500">
              <span>✓ Capabilities + Caveats</span>
              <span>✓ Delegation chains</span>
              <span>✓ Instant revocation</span>
              <span>✓ Tamper-evident audit</span>
            </div>
          </div>

          {/* Economic Policy Cards with Pricing Tiers */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
            {/* Observe — Free */}
            <div className="p-6 rounded-xl bg-black border border-cyan-800/30 hover:border-cyan-600/50 transition relative">
              <div className="absolute top-4 right-4 px-2 py-0.5 rounded-full bg-cyan-900/50 border border-cyan-700/50 text-cyan-300 text-xs font-bold">
                FREE
              </div>
              <div className="text-xs text-purple-400 mb-2">Protected by default →</div>
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2.5 bg-cyan-900/50 rounded-lg">
                  <Eye className="text-cyan-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Observe</h3>
              </div>
              <p className="text-gray-400 text-sm mb-3">
                verify → allow → meter/log
              </p>
              <p className="text-xs text-cyan-400/80 mb-3 italic">
                Perfect for audit logs and FinOps visibility.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Usage attribution by team</li>
                <li>✓ Cost center tagging</li>
                <li>✓ Zero latency impact</li>
              </ul>
            </div>

            {/* Control — $99/mo */}
            <div className="p-6 rounded-xl bg-black border-2 border-purple-500/50 hover:border-purple-400/70 transition relative">
              <div className="absolute top-4 right-4 px-2 py-0.5 rounded-full bg-purple-900/50 border border-purple-700/50 text-purple-300 text-xs font-bold">
                $99/mo
              </div>
              <div className="text-xs text-purple-400 mb-2">Protected by default →</div>
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2.5 bg-purple-900/50 rounded-lg">
                  <SlidersHorizontal className="text-purple-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Control</h3>
              </div>
              <p className="text-gray-400 text-sm mb-3">
                verify → enforce budget → allow
              </p>
              <p className="text-xs text-purple-400/80 mb-3 italic">
                Enforce strict budgets and spending caps.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Real-time budget enforcement</li>
                <li>✓ Fiat402 challenge/response</li>
                <li>✓ ERP/billing integration</li>
              </ul>
            </div>

            {/* Charge — Custom */}
            <div className="p-6 rounded-xl bg-black border border-green-800/30 hover:border-green-600/50 transition relative">
              <div className="absolute top-4 right-4 px-2 py-0.5 rounded-full bg-green-900/50 border border-green-700/50 text-green-300 text-xs font-bold">
                Custom
              </div>
              <div className="text-xs text-purple-400 mb-2">Protected by default →</div>
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2.5 bg-yellow-900/50 rounded-lg">
                  <DollarSign className="text-yellow-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Charge</h3>
              </div>
              <p className="text-gray-400 text-sm mb-3">
                verify → payment proof → allow
              </p>
              <p className="text-xs text-yellow-400/80 mb-3 italic">
                Monetize via L402 or Fiat billing.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Bitcoin Lightning (instant)</li>
                <li>✓ Per-request pricing</li>
                <li>✓ Agent-native payments</li>
              </ul>
            </div>
          </div>

          {/* PUBLIC callout */}
          <div className="p-4 rounded-lg bg-green-950/20 border border-green-900/30">
            <p className="text-sm text-gray-400">
              <span className="text-green-400 font-medium">PUBLIC</span> is the explicit opt-out for 
              probes (<code className="bg-gray-800 px-1 rounded text-gray-300">/healthz</code>), docs, and webhooks. 
              Everything else is protected by default.
            </p>
          </div>

          <div className="text-center mt-8">
            <Link href="/pricing" className="text-sm text-purple-400 hover:text-purple-300 transition underline underline-offset-4">
              See full pricing details →
            </Link>
          </div>
        </div>
      </section>

      {/* Economic Firewall Section */}
      <section className="py-16 px-6 border-b border-gray-800 bg-gradient-to-b from-red-950/10 to-black">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-10">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-red-900/30 border border-red-500/30 text-red-300 text-xs font-mono mb-6">
              <Shield size={12} /> Economic Friction
            </div>
            <h2 className="text-2xl font-bold mb-3">Economic Access Control in Action</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              Most abuse relies on <span className="text-white font-medium">zero marginal cost</span>. DDoS, credential stuffing, scraping — all free to attempt. 
              EAC changes the cost structure: every request carries a cryptographic cost. Math doesn&apos;t care how smart the AI is.
            </p>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <h4 className="font-semibold text-white mb-2 flex items-center gap-2">
                <span className="text-red-400">⛔</span> DDoS
              </h4>
              <p className="text-gray-500 text-sm mb-3">Without SatGate: <span className="text-red-400">You absorb the cost</span> of rejecting their traffic.</p>
              <p className="text-gray-400 text-sm">With SatGate: <span className="text-green-400">Every request carries a cost</span> — floods become uneconomical.</p>
            </div>
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <h4 className="font-semibold text-white mb-2 flex items-center gap-2">
                <span className="text-red-400">🤖</span> Scraping
              </h4>
              <p className="text-gray-500 text-sm mb-3">Without SatGate: Free data extraction, rotating IPs bypass rate limits.</p>
              <p className="text-gray-400 text-sm">With SatGate: <span className="text-green-400">Economic friction</span> makes bulk scraping unprofitable.</p>
            </div>
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <h4 className="font-semibold text-white mb-2 flex items-center gap-2">
                <span className="text-red-400">🔑</span> Credential Stuffing
              </h4>
              <p className="text-gray-500 text-sm mb-3">Without SatGate: Free brute force — 1M guesses cost nothing.</p>
              <p className="text-gray-400 text-sm">With SatGate: <span className="text-green-400">Each attempt costs real money</span> — the math stops working.</p>
            </div>
          </div>
          
          <div className="bg-gradient-to-r from-red-900/20 to-orange-900/20 border border-red-800/30 rounded-xl p-6 text-center">
            <p className="text-gray-300 text-sm">
              <span className="text-white font-semibold">"We don't fight abuse — we remove its economic incentive."</span>
              <br/>
              <span className="text-gray-500 text-xs mt-2 block">When every request carries a real cost, automated abuse loses its ROI.</span>
            </p>
          </div>
        </div>
      </section>

      {/* Enterprise-Safe Architecture */}
      <section className="py-20 px-6 border-b border-gray-800 bg-gradient-to-b from-purple-950/10 to-black">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono mb-6">
              <Cpu size={12} /> SatGate Architecture
            </div>
            <h2 className="text-3xl font-bold mb-3">Enterprise-Safe by Design</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              Separate trust boundaries, auditable config lifecycle, and per-tenant isolation built in.
            </p>
          </div>

          {/* Architecture Components */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            <div className="p-6 rounded-xl bg-gray-900 border border-purple-800/30">
              <div className="flex items-center gap-3 mb-3">
                <Key className="text-purple-400" size={20} />
                <h4 className="font-semibold text-purple-400">The Mint (Trust Broker)</h4>
              </div>
              <p className="text-sm text-gray-400">
                The &ldquo;Badge Office.&rdquo; Translates platform identities 
                (Kubernetes tokens, AWS roles, OIDC) into SatGate Macaroons.
              </p>
            </div>
            <div className="p-6 rounded-xl bg-gray-900 border border-green-800/30">
              <div className="flex items-center gap-3 mb-3">
                <Shield className="text-green-400" size={20} />
                <h4 className="font-semibold text-green-400">The Gateway (HTTP Proxy)</h4>
              </div>
              <p className="text-sm text-gray-400">
                Reverse proxy for REST APIs, LLM endpoints, SaaS services. Per-route policies, 
                budget enforcement, L402/Fiat402 monetization.
              </p>
            </div>
            <div className="p-6 rounded-xl bg-gray-900 border border-cyan-800/30 relative">
              <div className="absolute top-3 right-3 px-2 py-0.5 rounded-full bg-cyan-900/50 border border-cyan-700/50 text-cyan-300 text-[10px] font-bold">NEW</div>
              <div className="flex items-center gap-3 mb-3">
                <Zap className="text-cyan-400" size={20} />
                <h4 className="font-semibold text-cyan-400">MCP Proxy (Agent Tools)</h4>
              </div>
              <p className="text-sm text-gray-400">
                Intercepts MCP tool calls at the JSON-RPC level. Per-tool cost profiles, 
                budget delegation for sub-agents, real-time MCP dashboard.
              </p>
            </div>
          </div>

          {/* Control Plane vs Data Plane */}
          <div className="p-8 rounded-xl bg-gray-900/50 border border-gray-800 mb-8">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="text-center">
                <div className="w-16 h-16 bg-purple-900/30 rounded-xl flex items-center justify-center mx-auto mb-4">
                  <Lock className="text-purple-400" size={28} />
                </div>
                <h4 className="font-semibold mb-2 text-purple-400">Control Plane</h4>
                <p className="text-sm text-gray-500">
                  Private API for admin + config. Never exposed to public internet.
                </p>
              </div>
              <div className="hidden md:flex items-center justify-center">
                <div className="flex items-center gap-2 text-gray-600">
                  <span>←</span>
                  <span className="text-xs">Isolated</span>
                  <span>→</span>
                </div>
              </div>
              <div className="text-center">
                <div className="w-16 h-16 bg-green-900/30 rounded-xl flex items-center justify-center mx-auto mb-4">
                  <Activity className="text-green-400" size={28} />
                </div>
                <h4 className="font-semibold mb-2 text-green-400">Data Plane</h4>
                <p className="text-sm text-gray-500">
                  Public gateway endpoint. Only proxies tenant traffic. No admin access.
                </p>
              </div>
            </div>

            {/* Request Flow */}
            <div className="mt-8 pt-6 border-t border-gray-800">
              <p className="text-xs text-gray-500 text-center mb-3">Request Flow (Data Plane)</p>
              <div className="flex items-center justify-center gap-2 text-sm flex-wrap">
                <code className="bg-gray-800 px-3 py-1 rounded text-gray-400">Request</code>
                <span className="text-gray-600">→</span>
                <code className="bg-purple-900/30 px-3 py-1 rounded text-purple-400 border border-purple-500/30">Verify capability</code>
                <span className="text-gray-600">→</span>
                <code className="bg-cyan-900/30 px-3 py-1 rounded text-cyan-400 border border-cyan-500/30">Apply policy</code>
                <span className="text-gray-600">→</span>
                <code className="bg-gray-800 px-3 py-1 rounded text-gray-400">Upstream</code>
              </div>
            </div>
          </div>

          {/* Enterprise Checklist */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {[
              { icon: <Shield size={18} className="text-purple-400" />, title: 'SSRF-safe upstream validation', desc: 'Blocks private IPs, metadata endpoints, DNS rebinding' },
              { icon: <GitBranch size={18} className="text-cyan-400" />, title: 'Config versioning + applied gating', desc: 'Draft → Save → Apply workflow with rollback' },
              { icon: <Activity size={18} className="text-green-400" />, title: 'Per-tenant limits enforced', desc: 'Routes, upstreams, requests—all plan-gated' },
              { icon: <RefreshCw size={18} className="text-yellow-400" />, title: 'Audit log for all mutations', desc: 'Actor, timestamp, config hash, diff summary' },
              { icon: <Lock size={18} className="text-red-400" />, title: 'Tenant routing isolation', desc: 'No cross-tenant config bleed, fail-closed on errors' },
              { icon: <Key size={18} className="text-blue-400" />, title: 'HttpOnly session cookies', desc: 'No localStorage tokens, CSRF protected' },
            ].map((item, i) => (
              <div key={i} className="flex items-start gap-3 p-4 rounded-lg bg-gray-900/50 border border-gray-800">
                {item.icon}
                <div>
                  <div className="font-medium text-white">{item.title}</div>
                  <div className="text-sm text-gray-500">{item.desc}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Settlement Rails */}
      <section className="py-20 px-6 border-b border-gray-800">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-3">Charge <span className="text-gray-500 font-normal">(optional)</span>: Settlement Rails</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              When you choose the <span className="text-yellow-400">Charge</span> policy, pick your settlement mechanism. 
              Same gateway, same protection, different payment rails.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="p-6 rounded-xl bg-gray-900 border border-purple-800/30">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-purple-900/30 rounded-lg">
                  <Activity className="text-purple-400" size={20} />
                </div>
                <div>
                  <h3 className="font-bold">Internal</h3>
                  <span className="text-xs text-purple-400">Observe / Control policies</span>
                </div>
              </div>
              <p className="text-gray-400 text-sm mb-4">
                Track usage per tenant/team/project. No end-user payment—meter (Observe) 
                or enforce budgets (Control) internally.
              </p>
              <div className="text-xs text-gray-500">
                <strong className="text-gray-400">Best for:</strong> Internal platforms, FinOps, cost allocation
              </div>
            </div>

            <div className="p-6 rounded-xl bg-gray-900 border border-cyan-800/30">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-cyan-900/30 rounded-lg">
                  <DollarSign className="text-cyan-400" size={20} />
                </div>
                <div>
                  <h3 className="font-bold">Fiat402</h3>
                  <span className="text-xs text-cyan-400">Stripe-backed</span>
                </div>
              </div>
              <p className="text-gray-400 text-sm mb-4">
                Standard billing workflows via card or invoice. Same gateway enforcement, 
                enterprise procurement-friendly.
              </p>
              <div className="text-xs text-gray-500">
                <strong className="text-gray-400">Best for:</strong> Enterprise teams, procurement, SaaS billing
              </div>
            </div>

            <div className="p-6 rounded-xl bg-gray-900 border border-yellow-800/30">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-yellow-900/30 rounded-lg">
                  <Zap className="text-yellow-400" size={20} />
                </div>
                <div>
                  <h3 className="font-bold">L402</h3>
                  <span className="text-xs text-yellow-400">Lightning / instant</span>
                </div>
              </div>
              <p className="text-gray-400 text-sm mb-4">
                Sub-second settlement, no chargebacks, per-request pricing. 
                Perfect for developer APIs, AI agents, and micropayments.
              </p>
              <div className="text-xs text-gray-500">
                <strong className="text-gray-400">Best for:</strong> Developer-first APIs, agents, global users
              </div>
            </div>
          </div>

          <p className="text-center text-gray-500 text-sm mt-8">
            Start with Observe or Control, enable Charge when ready—per route, per tenant.
          </p>
        </div>
      </section>

      {/* Three Deployment Models */}
      <section className="py-20 px-6 border-b border-gray-800 bg-gradient-to-b from-gray-900/30 to-black">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-3">One Platform, Three Deployment Models</h2>
            <p className="text-gray-400">Pick the model that fits your ops, security, and data residency requirements.</p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* SaaS */}
            <div className="p-6 rounded-xl bg-gray-900 border border-green-800/30">
              <div className="flex items-center gap-3 mb-3">
                <Zap className="text-green-400" size={20} />
                <h3 className="text-lg font-bold text-green-400">SaaS</h3>
                <span className="text-xs bg-green-600 px-2 py-0.5 rounded">Fast Start</span>
              </div>
              <p className="text-gray-500 text-xs mb-3 italic">Fully managed gateway + control plane</p>
              <p className="text-gray-400 text-sm mb-3">
                Point DNS to SatGate. We run everything. Zero ops—live in minutes.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Zero infra—we handle everything</li>
                <li>✓ *.satgate.cloud or custom domain</li>
                <li>✓ Dashboard, billing, audit logs</li>
                <li>✓ Best for public APIs &amp; edge endpoints</li>
              </ul>
            </div>

            {/* Hybrid */}
            <div className="p-6 rounded-xl bg-gray-900 border-2 border-purple-500/50 relative">
              <div className="absolute -top-3 left-1/2 -translate-x-1/2 bg-purple-600 text-xs px-3 py-1 rounded-full font-bold">
                Recommended
              </div>
              <div className="flex items-center gap-3 mb-3">
                <Cpu className="text-purple-400" size={20} />
                <h3 className="text-lg font-bold text-purple-400">Hybrid</h3>
              </div>
              <p className="text-gray-500 text-xs mb-3 italic">Managed control plane + gateway in your VPC</p>
              <p className="text-gray-400 text-sm mb-3">
                Policies + dashboard in SatGate Cloud; data plane runs in your network. 
                Payloads never leave your VPC.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Data stays in your network</li>
                <li>✓ Cloud dashboard + config versioning</li>
                <li>✓ Telemetry + audit logs hosted</li>
                <li>✓ Deploy gateway via Helm/Docker</li>
              </ul>
            </div>

            {/* Self-Host */}
            <div className="p-6 rounded-xl bg-gray-900 border border-gray-800">
              <div className="flex items-center gap-3 mb-3">
                <Lock className="text-gray-400" size={20} />
                <h3 className="text-lg font-bold">Self-Host</h3>
              </div>
              <p className="text-gray-500 text-xs mb-3 italic">You run both planes</p>
              <p className="text-gray-400 text-sm mb-3">
                Full control. Deploy control plane + gateway in your own infra. 
                Air-gapped, on-prem, or private cloud.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Complete data sovereignty</li>
                <li>✓ SSO/SCIM, IP allowlists, GitOps</li>
                <li>✓ Air-gapped + compliance exports</li>
                <li>✓ Private networking, no egress</li>
              </ul>
            </div>
          </div>

          <div className="text-center mt-8">
            <a href="mailto:contact@satgate.io" className="text-purple-400 hover:text-purple-300 transition">
              Talk to us about Enterprise or Self-Host deployment →
            </a>
          </div>
        </div>
      </section>

      {/* Where It Fits Section */}
      <section className="py-16 px-6 border-b border-gray-800">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-2xl font-bold text-center mb-3">Where It Fits</h2>
          <p className="text-gray-500 text-center mb-10">Drop-in deployment. Minimal code changes.</p>
          
          {/* Architecture Diagram */}
          <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-8 mb-8">
            <h4 className="text-center text-sm font-semibold text-gray-400 mb-6">STANDARD DEPLOYMENT</h4>
            <div className="flex flex-col md:flex-row items-center justify-center gap-3 md:gap-4">
              <div className="px-5 py-3 rounded-lg bg-gray-800 border border-gray-700 text-center min-w-[140px]">
                <span className="text-gray-300 font-medium">Client / Agent</span>
              </div>
              <span className="text-gray-600 text-xl hidden md:block">→</span>
              <span className="text-gray-600 md:hidden">↓</span>
              <div className="px-5 py-3 rounded-lg bg-cyan-900/30 border border-cyan-700/50 text-center min-w-[140px]">
                <span className="text-cyan-400 font-medium">CDN / WAF</span>
              </div>
              <span className="text-gray-600 text-xl hidden md:block">→</span>
              <span className="text-gray-600 md:hidden">↓</span>
              <div className="px-5 py-3 rounded-lg bg-purple-900/40 border-2 border-purple-500/60 text-center min-w-[140px]">
                <span className="text-purple-300 font-bold">SatGate PEP</span>
              </div>
              <span className="text-gray-600 text-xl hidden md:block">→</span>
              <span className="text-gray-600 md:hidden">↓</span>
              <div className="px-5 py-3 rounded-lg bg-green-900/30 border border-green-700/50 text-center min-w-[140px]">
                <span className="text-green-400 font-medium">Your API</span>
                <span className="text-gray-600 text-xs block">REST / MCP / GraphQL</span>
              </div>
            </div>
            <p className="text-center text-gray-500 text-sm mt-6">
              SatGate sits <span className="text-white">behind</span> your CDN/WAF (volumetric protection) and <span className="text-white">in front of</span> your API origin.
              <br/>Works with <span className="text-white">MCP servers</span>, REST APIs, GraphQL — any HTTP endpoint.
            </p>
          </div>

          {/* Sidecar Deployment */}
          <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-8">
            <h4 className="text-center text-sm font-semibold text-gray-400 mb-6">ENTERPRISE "SIDECAR" MODE</h4>
            <p className="text-center text-gray-500 text-sm mb-6">
              Already have Kong, Apigee, or AWS API Gateway? Deploy SatGate as a <span className="text-white">sidecar</span>—no rip-and-replace.
            </p>
            <div className="flex flex-col md:flex-row items-center justify-center gap-3 md:gap-4">
              <div className="px-5 py-3 rounded-lg bg-gray-800 border border-gray-700 text-center min-w-[140px]">
                <span className="text-gray-300 font-medium">Existing Gateway</span>
                <span className="text-gray-600 text-xs block">Kong / Apigee</span>
              </div>
              <div className="flex flex-col items-center gap-2">
                <div className="flex items-center gap-2">
                  <span className="text-gray-600 text-sm hidden md:block">→</span>
                  <div className="px-4 py-2 rounded-lg bg-gray-800/50 border border-gray-700 text-center text-xs">
                    <span className="text-gray-400">/api/legacy/*</span>
                  </div>
                  <span className="text-gray-600 text-sm hidden md:block">→</span>
                  <span className="text-gray-500 text-xs">Legacy API</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-gray-600 text-sm hidden md:block">→</span>
                  <div className="px-4 py-2 rounded-lg bg-purple-900/40 border border-purple-500/60 text-center text-xs">
                    <span className="text-purple-300">/api/agents/*</span>
                  </div>
                  <span className="text-gray-600 text-sm hidden md:block">→</span>
                  <span className="text-purple-400 text-xs font-medium">SatGate</span>
                  <span className="text-gray-600 text-sm hidden md:block">→</span>
                  <span className="text-green-400 text-xs">Agent API</span>
                </div>
              </div>
            </div>
            <p className="text-center text-gray-600 text-xs mt-6">
              Route only agent traffic through SatGate. Keep existing infrastructure for legacy clients.
            </p>
          </div>

          {/* MCP Proxy Deployment */}
          <div className="bg-gray-900/50 border border-purple-800/30 rounded-xl p-8 mt-8">
            <h4 className="text-center text-sm font-semibold text-purple-400 mb-2">MCP PROXY MODE <span className="ml-2 px-2 py-0.5 rounded-full bg-purple-900/50 border border-purple-700/50 text-purple-300 text-xs font-bold">NEW</span></h4>
            <p className="text-center text-gray-500 text-sm mb-6">
              Govern every MCP tool call. Per-tool budgets, delegation trees, real-time dashboard.
            </p>
            <div className="flex flex-col md:flex-row items-center justify-center gap-3 md:gap-4">
              <div className="px-5 py-3 rounded-lg bg-gray-800 border border-gray-700 text-center min-w-[160px]">
                <span className="text-gray-300 font-medium">AI Agents</span>
                <span className="text-gray-600 text-xs block">Claude Code / Agent Zero / Cursor</span>
              </div>
              <span className="text-gray-600 text-xl hidden md:block">→</span>
              <span className="text-gray-600 md:hidden">↓</span>
              <div className="px-5 py-4 rounded-lg bg-purple-900/40 border-2 border-purple-500/60 text-center min-w-[180px]">
                <span className="text-purple-300 font-bold block">SatGate MCP Proxy</span>
                <span className="text-gray-500 text-xs block mt-1">Auth · Budget · Delegation</span>
                <span className="text-gray-500 text-xs block">Per-tool cost attribution</span>
              </div>
              <span className="text-gray-600 text-xl hidden md:block">→</span>
              <span className="text-gray-600 md:hidden">↓</span>
              <div className="px-5 py-3 rounded-lg bg-green-900/30 border border-green-700/50 text-center min-w-[160px]">
                <span className="text-green-400 font-medium">MCP Servers</span>
                <span className="text-gray-600 text-xs block">Tools / Databases / APIs</span>
              </div>
            </div>
            <p className="text-center text-gray-500 text-sm mt-6">
              Agents get a <span className="text-white">clean MCP error</span> when budgets run out. No agent code changes needed.
              <br/>Supports <span className="text-white">stdio</span> (subprocess) and <span className="text-white">SSE</span> (multi-agent) transports.
            </p>
          </div>
        </div>
      </section>

      {/* Feature Grid */}
      <section id="features" className="py-20 bg-gray-900/50 border-y border-gray-800">
        <div className="max-w-6xl mx-auto px-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
            
            <FeatureCard 
              icon={<Cpu className="text-cyan-400" />}
              title="LangChain Native"
              desc="Plug-and-play 'Tool' for LangChain, AutoGPT, and CrewAI. Give your agent a wallet in 2 lines of code."
            />
            <FeatureCard 
              icon={<Terminal className="text-purple-400" />}
              title="L402 Standard"
              desc="Built on the open L402 protocol. No proprietary lock-in. Compatible with any Lightning Node."
            />
            <FeatureCard 
              icon={<Code className="text-pink-400" />}
              title="Micropayments"
              desc="Meter in sats per call. Sub-cent pricing that's uneconomical on card rails, trivial with SatGate."
            />
            <FeatureCard 
              icon={<Shield className="text-green-400" />}
              title="Economic Friction for L7 Abuse"
              desc="High-volume scraping becomes expensive. Use alongside your WAF/CDN for volumetric protection."
            />
          </div>
        </div>
      </section>

      {/* Capability-Based Security Section */}
      <section className="py-20 px-6 bg-gradient-to-b from-black to-gray-900/50">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-4">Internet-Native Access Control</h2>
            <p className="text-gray-400 max-w-2xl mx-auto mb-4">
              SatGate implements L402 to turn payment into a <span className="text-purple-400 font-semibold">capability token</span> for APIs—ideal for stateless AI agents and zero-PII integrations.
            </p>
            <Link href="/security" className="text-sm text-purple-400 hover:text-purple-300 transition underline underline-offset-4">
              Deep dive: Security architecture →
            </Link>
          </div>
          
          {/* Identity vs Capability comparison */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-12">
            <div className="p-6 rounded-xl bg-red-950/20 border border-red-900/30">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-red-900/30 rounded-lg">
                  <Lock className="text-red-400" size={20} />
                </div>
                <h3 className="font-bold text-red-300">Identity-Based (Traditional)</h3>
              </div>
              <p className="text-gray-400 text-sm mb-4">"Prove <em>who you are</em>, then we decide what you can do."</p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li>• Requires user databases & PII</li>
                <li>• OAuth flows, API keys, sessions</li>
                <li>• Credential stuffing risks</li>
                <li>• Agents can't sign up</li>
              </ul>
            </div>
            <div className="p-6 rounded-xl bg-green-950/20 border border-green-900/30">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-green-900/30 rounded-lg">
                  <Key className="text-green-400" size={20} />
                </div>
                <h3 className="font-bold text-green-300">Capability-Based (L402)</h3>
              </div>
              <p className="text-gray-400 text-sm mb-4">"Present a token that <em>already encodes</em> what you can do."</p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="text-green-400">✓ No accounts required</li>
                <li className="text-green-400">✓ Payment-gated authorization</li>
                <li className="text-green-400">✓ Zero PII, no credentials to steal</li>
                <li className="text-green-400">✓ Perfect for autonomous agents</li>
              </ul>
            </div>
          </div>

          {/* Security Features */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <div className="flex items-center gap-3 mb-3">
                <Shield className="text-cyan-400" size={18} />
                <h4 className="font-semibold">Edge Verification</h4>
              </div>
              <p className="text-gray-500 text-sm">Tokens verified cryptographically at the gateway. No centralized identity store needed. Usage accounting/quotas can be tracked without storing PII.</p>
            </div>
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <div className="flex items-center gap-3 mb-3">
                <Clock className="text-purple-400" size={18} />
                <h4 className="font-semibold">Least Privilege</h4>
              </div>
              <p className="text-gray-500 text-sm">Add caveats to constrain scope, time, and budget (e.g., "valid 5 mins", "max 10 calls").</p>
            </div>
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <div className="flex items-center gap-3 mb-3">
                <Zap className="text-yellow-400" size={18} />
                <h4 className="font-semibold">Economic Friction</h4>
              </div>
              <p className="text-gray-500 text-sm">Spam becomes expensive and self-limiting. High-volume callers must continuously pay.</p>
            </div>
          </div>

          {/* Zero Trust Blurb */}
          <div className="mt-8 p-5 rounded-xl bg-gradient-to-r from-cyan-900/20 to-purple-900/20 border border-cyan-800/30">
            <div className="flex items-start gap-4">
              <Shield className="text-cyan-400 flex-shrink-0 mt-1" size={20} />
              <div>
                <h4 className="font-semibold text-white mb-2">Zero Trust Access Control (PEP)</h4>
                <p className="text-gray-400 text-sm leading-relaxed">
                  Enforces per-request authorization at the edge using cryptographically verifiable, least-privilege capability tokens—no network trust assumptions. 
                  Use alongside your existing WAF/CDN for volumetric protection.
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works - 4-Step Walkthrough */}
      <section className="py-20 px-6 border-b border-gray-800 bg-gradient-to-b from-gray-900/30 to-black">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-3xl font-bold mb-3">How It Works</h2>
            <p className="text-gray-400 max-w-xl mx-auto">
              Four steps to protect your API. No code changes required.
            </p>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            {[
              {
                step: "1",
                title: "Pick Your Policy",
                description: "Define routes with economic policies. PUBLIC for probes/docs, protected for everything else.",
                code: `routes:\n  - path: /healthz\n    policy: public\n  - path: /v1/*\n    policy: observe\n  - path: /premium/*\n    policy: charge`
              },
              {
                step: "2",
                title: "Apply Config",
                description: "Apply when ready. Version history + audit log. Rollback if needed.",
                code: `v3 (applied) ← current\nv2 (available)\nv1 (available)\n\nAudit: who, when, diff`
              },
              {
                step: "3",
                title: "Point Your DNS",
                description: "Use *.satgate.cloud or your custom domain. Traffic flows through SatGate.",
                code: `# Your domain\napi.yoursite.com\n  CNAME → satgate.cloud\n\n# Or use ours\nyourapp.satgate.cloud`
              },
              {
                step: "4",
                title: "See Verified Traffic",
                description: "Real-time: verified vs challenged. Enable Charge policy when ready for revenue.",
                code: `Verified:   1,203 requests\nChallenged: 12,847 (402s)\nMetered:    $847 usage\n\n→ Enable Charge policy?`
              }
            ].map((item, i) => (
              <div key={i} className="relative">
                <div className="absolute -left-3 -top-3 w-10 h-10 bg-purple-600 rounded-full flex items-center justify-center text-lg font-bold z-10">
                  {item.step}
                </div>
                <div className="p-6 pt-10 rounded-xl bg-gray-900 border border-gray-800 h-full">
                  <h3 className="text-lg font-semibold mb-2">{item.title}</h3>
                  <p className="text-gray-400 text-sm mb-4">{item.description}</p>
                  <pre className="bg-black text-xs p-3 rounded-lg overflow-x-auto text-gray-400 font-mono">
                    {item.code}
                  </pre>
                </div>
              </div>
            ))}
          </div>

          {/* Agent Token Flow */}
          <div className="mt-16 p-8 rounded-xl bg-gray-900/50 border border-gray-800">
            <h3 className="text-lg font-semibold text-center mb-2">How do agents get tokens?</h3>
            <p className="text-gray-500 text-sm text-center mb-6">
              SatGate Mint issues tokens at startup. Gateway verifies them on every request—no identity lookups on the hot path.
            </p>
            <div className="flex items-center justify-center gap-2 md:gap-4 flex-wrap text-sm">
              <div className="flex flex-col items-center p-3 bg-gray-800 rounded-lg">
                <Bot className="text-gray-300 mb-1" size={24} />
                <span className="text-gray-300 font-medium">Agent Starts</span>
                <span className="text-[10px] text-gray-500">K8s / AWS / OIDC</span>
              </div>
              <span className="text-gray-600 text-xl">→</span>
              <div className="flex flex-col items-center p-3 bg-purple-900/30 border border-purple-500/30 rounded-lg">
                <Key className="text-purple-400 mb-1" size={24} />
                <span className="text-purple-400 font-medium">Mint</span>
                <span className="text-[10px] text-gray-500">Badge-in (once)</span>
              </div>
              <span className="text-gray-600 text-xl">→</span>
              <div className="flex flex-col items-center p-3 bg-cyan-900/30 border border-cyan-500/30 rounded-lg">
                <Lock className="text-cyan-400 mb-1" size={24} />
                <span className="text-cyan-400 font-medium">Macaroon</span>
                <span className="text-[10px] text-gray-500">With caveats</span>
              </div>
              <span className="text-gray-600 text-xl">→</span>
              <div className="flex flex-col items-center p-3 bg-green-900/30 border border-green-500/30 rounded-lg">
                <Shield className="text-green-400 mb-1" size={24} />
                <span className="text-green-400 font-medium">Gateway</span>
                <span className="text-[10px] text-gray-500">Verify (every req)</span>
              </div>
              <span className="text-gray-600 text-xl">→</span>
              <div className="flex flex-col items-center p-3 bg-gray-800 rounded-lg">
                <Activity className="text-gray-300 mb-1" size={24} />
                <span className="text-gray-300 font-medium">Upstream</span>
                <span className="text-[10px] text-gray-500">Your API</span>
              </div>
            </div>
            <p className="text-xs text-gray-600 text-center mt-4">
              Manual token issuance also available via Dashboard or API. Mint is optional for automated agent provisioning.
            </p>
          </div>
        </div>
      </section>

      {/* Code Integration Section */}
      <section className="py-20 px-6">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">Universal Integration</h2>
          
          <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
            <div className="flex border-b border-gray-800">
              <button 
                onClick={() => setActiveTab('python')}
                className={`px-6 py-3 font-mono text-sm border-r border-gray-700 transition ${activeTab === 'python' ? 'bg-gray-800 text-white' : 'text-gray-500 hover:text-white'}`}
              >
                Python (Agents)
              </button>
              <button 
                onClick={() => setActiveTab('nodejs')}
                className={`px-6 py-3 font-mono text-sm border-r border-gray-700 transition ${activeTab === 'nodejs' ? 'bg-gray-800 text-white' : 'text-gray-500 hover:text-white'}`}
              >
                Node.js
              </button>
              <button 
                onClick={() => setActiveTab('curl')}
                className={`px-6 py-3 font-mono text-sm transition ${activeTab === 'curl' ? 'bg-gray-800 text-white' : 'text-gray-500 hover:text-white'}`}
              >
                cURL
              </button>
            </div>
            <div className="p-8 overflow-x-auto">
              {activeTab === 'python' && (
<pre className="text-sm font-mono text-gray-300 leading-relaxed">
<span className="text-purple-400">from</span> satgate.langchain <span className="text-purple-400">import</span> SatGateTool<br/>
<span className="text-purple-400">from</span> langchain.agents <span className="text-purple-400">import</span> initialize_agent<br/>
<br/>
<span className="text-gray-500"># 1. Give your agent a wallet</span><br/>
tools = [SatGateTool(wallet=my_lnd_node)]<br/>
<br/>
<span className="text-gray-500"># 2. Let it roam the economy</span><br/>
agent = initialize_agent(tools, llm, agent=<span className="text-green-400">"openai-functions"</span>)<br/>
agent.run(<span className="text-green-400">"Buy the latest stock report from AlphaVantage"</span>)
</pre>
              )}
              {activeTab === 'nodejs' && (
<pre className="text-sm font-mono text-gray-300 leading-relaxed">
<span className="text-purple-400">import</span> {'{ SatGateClient }'} <span className="text-purple-400">from</span> <span className="text-green-400">'satgate-sdk'</span>;<br/>
<br/>
<span className="text-gray-500">// Initialize client (uses WebLN in browser)</span><br/>
<span className="text-purple-400">const</span> client = <span className="text-purple-400">new</span> SatGateClient();<br/>
<br/>
<span className="text-gray-500">// Automatic: 402 → Pay → Retry → Response</span><br/>
<span className="text-purple-400">const</span> data = <span className="text-purple-400">await</span> client.get(<span className="text-green-400">'https://api.example.com/premium'</span>);<br/>
console.log(data);
</pre>
              )}
              {activeTab === 'curl' && (
<pre className="text-sm font-mono text-gray-300 leading-relaxed">
<span className="text-gray-500"># 1. Request protected endpoint → get 402 + invoice</span><br/>
curl -i https://api.example.com/api/premium<br/>
<br/>
<span className="text-gray-500"># 2. Pay the Lightning invoice (via your wallet)</span><br/>
<span className="text-gray-500"># Returns preimage as proof of payment</span><br/>
<br/>
<span className="text-gray-500"># 3. Retry with L402 token</span><br/>
curl -H <span className="text-green-400">"Authorization: L402 &lt;macaroon&gt;:&lt;preimage&gt;"</span> \<br/>
&nbsp;&nbsp;https://api.example.com/api/premium
</pre>
              )}
            </div>
          </div>
        </div>
      </section>

      {/* CTA / Footer */}
      <footer className="py-20 border-t border-gray-800">
        <div className="max-w-6xl mx-auto px-6">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-6">Ready to see what your agents are spending?</h2>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-4">
              <a href="mailto:contact@satgate.io" className="inline-block bg-gradient-to-r from-purple-600 to-cyan-600 text-white px-10 py-4 rounded-full font-bold text-lg hover:opacity-90 transition shadow-lg shadow-purple-500/20">
                Get in Touch
              </a>
              <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="inline-block bg-white text-black px-10 py-4 rounded-full font-bold text-lg hover:bg-gray-200 transition">
                Start Free →
              </a>
            </div>
            <p className="text-gray-500 text-sm">
              Or <Link href="/govern" className="text-purple-400 hover:text-purple-300 transition underline underline-offset-4">learn how enterprises are controlling AI agent spend →</Link>
            </p>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8 py-12 border-t border-gray-800">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <Image src="/logo_white_transparent.png" alt="SatGate" width={24} height={24} className="w-6 h-6" />
                <h4 className="font-bold text-white">SatGate</h4>
              </div>
              <p className="text-gray-500 text-sm">The Economic Firewall for AI agent requests.</p>
              <p className="text-gray-600 text-xs mt-3">Non-custodial. We never hold your keys.</p>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Resources</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a></li>
                <li><a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Documentation</a></li>
                <li><Link href="/govern" className="hover:text-white transition">Govern</Link></li>
                <li><Link href="/design-partners" className="hover:text-white transition">Design Partners</Link></li>
                <li><Link href="/pricing" className="hover:text-white transition">Pricing</Link></li>
                <li><Link href="/compare" className="hover:text-white transition">Compare</Link></li>
                <li><Link href="/blog" className="hover:text-white transition">Blog</Link></li>
                <li><a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud Dashboard</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Legal</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/terms" className="hover:text-white transition">Terms of Service</Link></li>
                <li><Link href="/privacy" className="hover:text-white transition">Privacy Policy</Link></li>
                <li><Link href="/security" className="hover:text-white transition">Security</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Contact</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="mailto:contact@satgate.io" className="hover:text-white transition">contact@satgate.io</a></li>
              </ul>
            </div>
          </div>
          
          <div className="pt-8 border-t border-gray-800 text-center text-gray-600 text-sm">
            © 2025 SatGate Inc. All rights reserved. SatGate™ is a trademark of SatGate Inc. Patent Pending.
          </div>
        </div>
      </footer>
    </div>
  );
};

// Simple helper component for features
const FeatureCard = ({ icon, title, desc }: any) => (
  <div className="p-6 rounded-xl bg-black border border-gray-800 hover:border-gray-600 transition group">
    <div className="mb-4 p-3 bg-gray-900 rounded-lg w-fit group-hover:bg-gray-800 transition">{icon}</div>
    <h3 className="text-xl font-bold mb-2">{title}</h3>
    <p className="text-gray-400 leading-relaxed">{desc}</p>
  </div>
);

export default LandingPage;

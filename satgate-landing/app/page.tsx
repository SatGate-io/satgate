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
            <Link href="/agents" className="hover:text-white transition">Agents</Link>
            <Link href="/govern" className="hover:text-white transition">Enterprise</Link>
            <Link href="/pricing" className="hover:text-white transition">Pricing</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="text-purple-400 hover:text-purple-300 transition">Cloud →</a>
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
              href="/agents" 
              onClick={() => setMobileMenuOpen(false)} 
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Agents
            </Link>
            <Link 
              href="/govern" 
              onClick={() => setMobileMenuOpen(false)} 
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Enterprise
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
              Know what your AI agents<br/>
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
                are spending.
              </span>
            </h1>
            <p className="text-xl text-gray-400 mb-4 max-w-lg leading-relaxed">
              Every API request — authenticated, logged, cost-tracked, and budget-enforced. The <b className="text-white">Economic Firewall</b> for AI agent traffic.
            </p>
            <p className="text-lg text-gray-500 mb-8 max-w-lg leading-relaxed">
              Not &ldquo;who are you?&rdquo; — <b>&ldquo;what can you afford?&rdquo;</b> Per-agent budgets. Spending caps. Full delegation control. Connect in 5 minutes.
            </p>
            <div className="flex gap-4">
              <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 text-white px-8 py-3 rounded-lg font-bold transition flex items-center gap-2 shadow-lg shadow-purple-500/20">
                Start Free <ArrowRight size={16} />
              </a>
              <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="border border-gray-700 px-8 py-3 rounded-lg font-bold hover:border-gray-500 transition flex items-center gap-2 text-gray-300">
                View on GitHub
              </a>
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
                🚗💨 EZ Pass — live metering
              </div>
            </div>
            <div className="text-center mt-4">
              <p className="text-sm text-gray-500 mb-3">
                Agents badge in once. Every request — verified, metered, budget-enforced.
              </p>
              <Link 
                href="/protect" 
                className="inline-flex items-center gap-2 text-sm font-medium text-purple-400 hover:text-purple-300 transition"
              >
                See how it works <ArrowRight size={14} />
              </Link>
            </div>
          </div>
        </div>
      </header>

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

            {/* Control — included in Pro */}
            <div className="p-6 rounded-xl bg-black border-2 border-purple-500/50 hover:border-purple-400/70 transition relative">
              <div className="absolute top-4 right-4 px-2 py-0.5 rounded-full bg-purple-900/50 border border-purple-700/50 text-purple-300 text-xs font-bold">
                PRO
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

            {/* Charge — included in Pro */}
            <div className="p-6 rounded-xl bg-black border border-yellow-800/30 hover:border-yellow-600/50 transition relative">
              <div className="absolute top-4 right-4 px-2 py-0.5 rounded-full bg-purple-900/50 border border-purple-700/50 text-purple-300 text-xs font-bold">
                PRO
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
                Monetize via L402 Lightning payments.
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
            <h4 className="text-center text-sm font-semibold text-purple-400 mb-2">MCP PROXY MODE</h4>
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

          {/* EZ Pass — Agent Token Flow */}
          <div className="mt-16 p-8 rounded-xl bg-gray-900/50 border border-purple-800/30">
            <div className="text-center mb-2">
              <span className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono mb-3">
                🚗💨 THE EZ PASS FOR API TRAFFIC
              </span>
              <h3 className="text-2xl font-bold mb-2">Badge in once. Fly through every gate.</h3>
              <p className="text-gray-400 text-sm max-w-xl mx-auto">
                Agents get a credential at startup — like mounting an EZ Pass. Every request after that flows through the gateway: verified, metered, no slowdowns. One token, every toll, full speed.
              </p>
            </div>
            <div className="flex items-center justify-center gap-2 md:gap-4 flex-wrap text-sm mt-8">
              <div className="flex flex-col items-center p-3 bg-gray-800 rounded-lg">
                <Bot className="text-gray-300 mb-1" size={24} />
                <span className="text-gray-300 font-medium">Agent Starts</span>
                <span className="text-[10px] text-gray-500">K8s / AWS / OIDC</span>
              </div>
              <span className="text-gray-600 text-xl">→</span>
              <div className="flex flex-col items-center p-3 bg-purple-900/30 border border-purple-500/30 rounded-lg">
                <Key className="text-purple-400 mb-1" size={24} />
                <span className="text-purple-400 font-medium">Mint</span>
                <span className="text-[10px] text-gray-500">Badge in (once)</span>
              </div>
              <span className="text-gray-600 text-xl">→</span>
              <div className="flex flex-col items-center p-3 bg-cyan-900/30 border border-cyan-500/30 rounded-lg">
                <Lock className="text-cyan-400 mb-1" size={24} />
                <span className="text-cyan-400 font-medium">EZ Pass</span>
                <span className="text-[10px] text-gray-500">Macaroon + caveats</span>
              </div>
              <span className="text-gray-600 text-xl">→</span>
              <div className="flex flex-col items-center p-3 bg-green-900/30 border border-green-500/30 rounded-lg">
                <Shield className="text-green-400 mb-1" size={24} />
                <span className="text-green-400 font-medium">Toll Gate</span>
                <span className="text-[10px] text-gray-500">Verify · Meter · Budget</span>
              </div>
              <span className="text-gray-600 text-xl">→</span>
              <div className="flex flex-col items-center p-3 bg-gray-800 rounded-lg">
                <Activity className="text-gray-300 mb-1" size={24} />
                <span className="text-gray-300 font-medium">Upstream</span>
                <span className="text-[10px] text-gray-500">Your API</span>
              </div>
            </div>
            <p className="text-xs text-gray-600 text-center mt-6">
              No identity lookups on the hot path. No per-request auth round-trips. Just cryptographic verification at wire speed.
            </p>
          </div>
        </div>
      </section>

      {/* Code Integration Section */}
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
            © 2025–2026 SatGate Inc. All rights reserved. SatGate™ is a trademark of SatGate Inc. Patent Pending.
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

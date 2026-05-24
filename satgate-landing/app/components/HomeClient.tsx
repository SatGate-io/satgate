'use client';

import React, { useState } from 'react';
import { Terminal, Code, Cpu, Zap, ArrowRight, CheckCircle, Copy, Check, Shield, Key, Lock, Clock, DollarSign, Bot, GitBranch, Activity, RefreshCw, Menu, X, Eye, SlidersHorizontal, Play, BookOpen, BarChart3 } from 'lucide-react';
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
            <span className="text-lg sm:text-xl font-bold text-white whitespace-nowrap">SatGate<sup className="text-xs font-normal">TM</sup></span>
          </Link>

          {/* Desktop menu */}
          <div className="hidden xl:flex items-center gap-5 text-sm font-medium text-gray-400">
            <Link href="/govern" className="hover:text-white transition">Enterprise</Link>
            <Link href="/mcp-gateway" className="hover:text-white transition">MCP Gateway</Link>
            <Link href="/agent-authority-layer" className="hover:text-white transition">Authority & Accountability</Link>
            <Link href="/build" className="hover:text-white transition">Build</Link>
            <Link href="/sandbox" className="hover:text-white transition">Demo</Link>
            <Link href="/pricing" className="hover:text-white transition">Pricing</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="rounded-full border border-purple-500/40 px-3 py-1.5 text-purple-300 hover:border-purple-400 hover:text-purple-200 transition">Cloud →</a>
          </div>

          {/* Mobile menu button */}
          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="flex xl:hidden items-center justify-center w-10 h-10 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 text-gray-400 hover:text-white transition"
            aria-label="Toggle menu"
            aria-expanded={mobileMenuOpen}
          >
            {mobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
        </div>

        {/* Mobile menu dropdown */}
        <div
          className={`xl:hidden transition-all duration-300 ease-in-out ${
            mobileMenuOpen ? 'max-h-[calc(100vh-4rem)] overflow-y-auto opacity-100' : 'max-h-0 overflow-hidden opacity-0'
          }`}
        >
          <div className="bg-black/95 backdrop-blur-xl border-t border-gray-800 px-4 py-4 space-y-1">
            <Link
              href="/govern"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Enterprise
            </Link>
            <Link
              href="/mcp-gateway"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              MCP Gateway
            </Link>
            <Link
              href="/agent-authority-layer"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Authority & Accountability
            </Link>
            <Link
              href="/build"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Build
            </Link>
            <Link
              href="/sandbox"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Demo
            </Link>
            <Link
              href="/capability-auth"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Capability Auth
            </Link>
            <Link
              href="/agent-control-plane"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Agent Control Plane
            </Link>
            <Link
              href="/pricing"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Pricing
            </Link>
            <Link
              href="/tools"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Tools
            </Link>
            <Link
              href="/integrations"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Integrations
            </Link>
            <Link
              href="/blog"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Blog
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
              <Zap size={12} /> Agent Authority & Accountability Layer
            </div>
            <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
              Govern agent authority<br/>
              <span className="sr-only"> </span><span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
                before execution.
              </span>
            </h1>
            <p className="text-xl text-gray-400 mb-4 max-w-lg leading-relaxed">
              SatGate gives agents bounded economic authority so humans, platforms, and upstream APIs can trust what they consume, spend, and prove.
            </p>
            <p className="text-lg text-gray-500 mb-8 max-w-lg leading-relaxed">
              Humans and platforms set policy. Agents consume approved primitives. Upstreams get receipt-backed proof.
            </p>
            <div className="flex flex-wrap gap-4">
              <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 text-white px-8 py-3 rounded-lg font-bold transition flex items-center gap-2 shadow-lg shadow-purple-500/20">
                Start Free <ArrowRight size={16} />
              </a>
              <Link href="/build" className="border border-cyan-700/50 bg-cyan-900/15 px-8 py-3 rounded-lg font-bold hover:bg-cyan-900/30 transition flex items-center gap-2 text-cyan-300">
                Build with SatGate <ArrowRight size={16} />
              </Link>
              <Link href="/sandbox" className="border border-purple-700/50 bg-purple-900/20 px-8 py-3 rounded-lg font-bold hover:bg-purple-900/40 transition flex items-center gap-2 text-purple-300">
                <Play size={16} /> See Demo
              </Link>
              <Link href="/agent-control-plane" className="border border-cyan-700/50 bg-cyan-900/15 px-8 py-3 rounded-lg font-bold hover:bg-cyan-900/30 transition flex items-center gap-2 text-cyan-300">
                Agent Control Plane <ArrowRight size={16} />
              </Link>
            </div>

            {/* Proof strip */}
            <div className="mt-6 space-y-2 text-xs text-gray-500">
              <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
                <span className="flex items-center gap-1.5"><CheckCircle size={12} className="text-green-500" /> REST · GraphQL · MCP</span>
                <span className="flex items-center gap-1.5"><CheckCircle size={12} className="text-green-500" /> Gateway · Sidecar · MCP Proxy</span>
                <span className="flex items-center gap-1.5"><CheckCircle size={12} className="text-green-500" /> Sub-ms verification</span>
              </div>
              <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
                <span className="flex items-center gap-1.5"><CheckCircle size={12} className="text-green-500" /> MCP · API keys · x402-aware governance</span>
                <span className="flex items-center gap-1.5"><CheckCircle size={12} className="text-green-500" /> <a href="https://github.com/SatGate-io/satgate" className="text-gray-400 hover:text-white transition underline underline-offset-2">Open source</a></span>
              </div>
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
                <div className="text-xs text-gray-500 ml-2 font-mono">hero_demo.py - Demo Preview</div>
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
                🚗💨 EZ Pass - live metering
              </div>
            </div>
            <div className="text-center mt-4">
              <p className="text-sm text-gray-500 mb-3">
                Agents badge in once. Every request - verified, metered, budget-enforced.
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

      {/* Explainer Video Section */}
      <section className="py-16 px-6 border-b border-gray-800">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-3xl font-bold mb-3">See SatGate in Action</h2>
          <p className="text-gray-400 mb-8 max-w-2xl mx-auto">Agents act across tools, APIs, and paid rails. SatGate enforces policy before they act — and leaves evidence after. 30 seconds.</p>
          <div className="relative rounded-xl overflow-hidden border border-gray-700/50 shadow-2xl shadow-purple-500/10">
            <video
              controls
              preload="metadata"
              poster="/satgate-explainer-poster.jpg"
              className="w-full"
              playsInline
            >
              <source src="/satgate-explainer.mp4" type="video/mp4" />
              Your browser does not support the video tag.
            </video>
          </div>
        </div>
      </section>

      {/* Free SEO Tools */}
      <section className="py-20 px-6 border-b border-gray-800 bg-black">
        <div className="max-w-6xl mx-auto">
          <div className="mb-10 max-w-3xl">
            <p className="mb-3 text-sm font-mono uppercase tracking-wide text-cyan-300">Free agent governance tools</p>
            <h2 className="mb-4 text-3xl md:text-4xl font-bold text-white">Measure authority and spend risk before agents run wild</h2>
            <p className="text-gray-400 text-lg leading-relaxed">
              Start with the flagship tools and benchmark. The full tools hub has the calculators, policy generators, and readiness checks for deeper planning.
            </p>
          </div>
          <div className="mb-6">
            <Link href="/tools" className="inline-flex items-center gap-2 text-cyan-300 hover:text-cyan-200 font-semibold transition">
              View all free AI agent governance tools <ArrowRight size={16} />
            </Link>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
            {[
              { href: '/runaway-agent-cost-calculator', title: 'Runaway Agent Cost Calculator', body: 'Model loop, retry, fanout, and paid tool-call exposure before detection.', icon: Activity },
              { href: '/ai-agent-runaway-spend-benchmark', title: 'AI Agent Runaway Spend Benchmark', body: 'Use original JSON/CSV benchmark data to quantify agent loops, retry storms, and avoidable spend.', icon: BarChart3 },
              { href: '/economic-firewall-readiness-grader', title: 'Economic Firewall Readiness Grader', body: 'Score identity, budgets, MCP tools, revocation, Evidence Packs, routing, and paid-rail governance.', icon: Shield },
            ].map(({ href, title, body, icon: Icon }) => (
              <Link key={href} href={href} className="group rounded-xl border border-gray-800 bg-gray-950 p-6 transition hover:border-cyan-500/50 hover:bg-cyan-950/10">
                <Icon className="mb-4 text-cyan-300 transition group-hover:text-cyan-200" size={28} />
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="mb-4 text-sm leading-relaxed text-gray-400">{body}</p>
                <span className="inline-flex items-center gap-2 text-sm font-semibold text-cyan-300">Open tool <ArrowRight size={14} /></span>
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* Default Protection + Economic Policies */}
      <section className="py-20 px-6 border-b border-gray-800 bg-gradient-to-b from-gray-900/30 to-black">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-3">Govern, enforce, prove</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">Start with internal agents: scope authority, enforce policy at runtime, and preserve evidence. Then open external rails - on your terms.</p>
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
              delegation, and revocation-built into the protocol, not bolted on.
            </p>
            <div className="flex flex-wrap gap-4 text-sm text-gray-500">
              <span>✓ Capabilities + Caveats</span>
              <span>✓ Delegation chains</span>
              <span>✓ Next-request revocation</span>
              <span>✓ Tamper-evident Evidence Pack receipts</span>
            </div>
          </div>

          {/* Your Agents */}
          <div className="mb-4">
            <p className="text-sm font-mono text-cyan-400 mb-4 uppercase tracking-wider">Your Agents - Govern Authority and Spend</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
            {/* Observe - Free */}
            <div className="p-6 rounded-xl bg-black border border-cyan-800/30 hover:border-cyan-600/50 transition relative">
              <div className="text-xs text-purple-400 mb-2">Protected by default →</div>
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2.5 bg-cyan-900/50 rounded-lg">
                  <Eye className="text-cyan-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Observe <span className="text-xs font-normal text-gray-500">(Fiat)</span></h3>
              </div>
              <p className="text-gray-400 text-sm mb-3">
                verify → allow → meter/log
              </p>
              <p className="text-xs text-cyan-400/80 mb-3 italic">
                Start here. No workflow changes. Map authority, tools, and spend before enforcing policy.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Observe mode - zero disruption to existing agents</li>
                <li>✓ Usage attribution by team and cost center</li>
                <li>✓ See exactly which agents, tools, and routes create risk before you change anything</li>
                <li>✓ Zero latency impact</li>
              </ul>
            </div>

            {/* Control - included in Pro */}
            <div className="p-6 rounded-xl bg-black border-2 border-purple-500/50 hover:border-purple-400/70 transition relative">
              <div className="text-xs text-purple-400 mb-2">Protected by default →</div>
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2.5 bg-purple-900/50 rounded-lg">
                  <SlidersHorizontal className="text-purple-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Control <span className="text-xs font-normal text-gray-500">(Fiat402)</span></h3>
              </div>
              <p className="text-gray-400 text-sm mb-3">
                verify → enforce budget → allow
              </p>
              <p className="text-xs text-purple-400/80 mb-3 italic">
                Now enforce it. Policy and budget caps stop agents before unauthorized work executes.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Real-time budget enforcement</li>
                <li>✓ Works with Stripe, ERP - no crypto required</li>
                <li>✓ Per-agent spending caps</li>
              </ul>
            </div>

          </div>

          {/* Their Agents */}
          <div className="mb-4">
            <p className="text-sm font-mono text-yellow-400 mb-4 uppercase tracking-wider">Their Agents - Prevent Unauthorized Access</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-1 gap-6 mb-6 max-w-lg">
            {/* Prove - evidence above external rails */}
            <div className="p-6 rounded-xl bg-black border border-yellow-800/30 hover:border-yellow-600/50 transition relative">
              <div className="text-xs text-purple-400 mb-2">Protected by default →</div>
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2.5 bg-yellow-900/50 rounded-lg">
                  <Shield className="text-yellow-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Prove <span className="text-xs font-normal text-gray-500">(evidence above rails)</span></h3>
              </div>
              <p className="text-gray-400 text-sm mb-3">
                verify → authority proof → Evidence Pack
              </p>
              <p className="text-xs text-yellow-400/80 mb-3 italic">
                Govern external agent access without making payment proof equal authorization proof.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Let approved agents consume APIs without long-lived shared secrets</li>
                <li>✓ Preserve authority evidence above x402, L402, API-key, or enterprise billing rails</li>
                <li>✓ Per-request pricing and policy before upstream execution</li>
                <li>✓ Approved agents consume scoped access and leave an Evidence Pack</li>
              </ul>
            </div>
          </div>

          {/* Token Delegation Video */}
          <div className="mt-12 mb-8">
            <div className="text-center mb-6">
              <h3 className="text-2xl font-bold mb-2">Why API Keys Break in Agent Chains</h3>
              <p className="text-gray-400 max-w-xl mx-auto text-sm">API keys are all-or-nothing. Delegated capability tokens let you set any budget, scope, and expiry per agent - and agents can&apos;t escalate beyond what they&apos;re given. Trust flows down, never up.</p>
            </div>
            <div className="max-w-3xl mx-auto relative rounded-xl overflow-hidden border border-gray-700/50 shadow-2xl shadow-purple-500/10">
              <video
                controls
                preload="metadata"
                poster="/satgate-delegation-poster.jpg"
                className="w-full"
                playsInline
              >
                <source src="/satgate-delegation.mp4" type="video/mp4" />
                Your browser does not support the video tag.
              </video>
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

      {/* EZ Pass - Agent Token Flow (pulled up per feedback) */}
      <section className="py-16 px-6 border-b border-gray-800 bg-gradient-to-b from-purple-950/10 to-black">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-2">
            <span className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono mb-3">
              🚗💨 HOW IT WORKS
            </span>
            <h2 className="text-2xl font-bold mb-2">Badge in once. Fly through every gate.</h2>
            <p className="text-gray-400 text-sm max-w-xl mx-auto">
              Agents get a credential at startup - like mounting an EZ Pass. Every request after that flows through the gateway: verified, metered, no slowdowns.
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
              <span className="text-[10px] text-gray-500">Capability token</span>
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
      </section>

      {/* Research Alignment */}
      <section className="py-16 px-6 border-b border-gray-800 bg-gradient-to-b from-gray-900/20 to-black">
        <div className="max-w-3xl mx-auto">
          <div className="text-center mb-8">
            <span className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-900/30 border border-blue-500/30 text-blue-300 text-xs font-mono mb-3">
              <BookOpen size={12} /> RESEARCH ALIGNMENT
            </span>
            <h2 className="text-2xl font-bold mb-3">Built for the agent delegation era</h2>
            <p className="text-gray-400 text-sm max-w-2xl mx-auto">
              Recent research on intelligent AI delegation points to a control problem we see in practice: agents need bounded authority,
              clear caveats, and safe ways to delegate across trust boundaries. One proposed path is attenuated capability tokens,
              including <span className="text-blue-300">macaroons</span>, that restrict what each sub-agent can access.
            </p>
            <p className="text-gray-300 text-sm mt-3 font-medium">
              SatGate implements one version of that control layer.
            </p>
          </div>
          <div className="grid md:grid-cols-3 gap-4 mb-6">
            <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700">
              <Lock className="text-cyan-400 mb-2" size={20} />
              <h3 className="font-bold text-sm mb-1">Scoped Authority</h3>
              <p className="text-gray-400 text-xs">
                Agents only get the permissions they need, attenuated at each delegation layer.
              </p>
            </div>
            <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700">
              <DollarSign className="text-green-400 mb-2" size={20} />
              <h3 className="font-bold text-sm mb-1">Budget Ceilings</h3>
              <p className="text-gray-400 text-xs">
                Per-agent and per-route economic policy, enforced before upstream execution.
              </p>
            </div>
            <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700">
              <Zap className="text-yellow-400 mb-2" size={20} />
              <h3 className="font-bold text-sm mb-1">Immediate Enforcement</h3>
              <p className="text-gray-400 text-xs">
                When limits hit, requests stop. Not after billing. Now.
              </p>
            </div>
          </div>
          <p className="text-gray-500 text-xs text-center">
            We built SatGate because standing API keys and after-the-fact alerts are a bad fit for autonomous systems. The research gives useful language for a problem we were already seeing in deployed agent workflows.{' '}
            <span className="text-gray-600 ml-1">
              - <a href="https://arxiv.org/abs/2602.11865" target="_blank" rel="noopener noreferrer" className="hover:text-gray-400 underline underline-offset-2">Tomasev et al., 2026</a>
            </span>
          </p>
        </div>
      </section>

      {/* Where It Fits Section - Clean diagrams */}
      <section className="py-16 px-6 border-b border-gray-800">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-2xl font-bold text-center mb-3">Where It Fits</h2>
          <p className="text-gray-500 text-center mb-10">Three deployment modes. Drop-in. No rip-and-replace.</p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Standard */}
            <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-6 hover:border-gray-700 transition">
              <h4 className="text-sm font-bold text-gray-400 mb-4 text-center">STANDARD</h4>
              <div className="flex flex-col items-center gap-2 text-sm">
                <div className="w-full px-3 py-2 rounded bg-gray-800 text-center text-gray-400 text-xs">CDN / WAF</div>
                <span className="text-gray-600">↓</span>
                <div className="w-full px-3 py-2.5 rounded bg-purple-900/40 border border-purple-500/50 text-center">
                  <span className="text-purple-300 font-bold text-xs">SatGate</span>
                </div>
                <span className="text-gray-600">↓</span>
                <div className="w-full px-3 py-2 rounded bg-green-900/30 border border-green-800/50 text-center text-green-400 text-xs">Your API</div>
              </div>
              <p className="text-gray-600 text-xs text-center mt-4">REST, GraphQL, any HTTP endpoint</p>
            </div>

            {/* Sidecar */}
            <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-6 hover:border-gray-700 transition">
              <h4 className="text-sm font-bold text-gray-400 mb-4 text-center">SIDECAR</h4>
              <div className="flex flex-col items-center gap-2 text-sm">
                <div className="w-full px-3 py-2 rounded bg-gray-800 text-center text-gray-400 text-xs">Existing Gateway</div>
                <div className="flex items-center gap-2 w-full">
                  <div className="flex-1 flex flex-col items-center gap-1">
                    <span className="text-gray-600 text-xs">↓</span>
                    <div className="w-full px-2 py-1.5 rounded bg-gray-800/50 border border-gray-700 text-center text-gray-500 text-[10px]">Legacy traffic</div>
                  </div>
                  <div className="flex-1 flex flex-col items-center gap-1">
                    <span className="text-purple-400 text-xs">↓</span>
                    <div className="w-full px-2 py-1.5 rounded bg-purple-900/40 border border-purple-500/50 text-center text-purple-300 text-[10px] font-bold">SatGate</div>
                  </div>
                </div>
                <div className="flex items-center gap-2 w-full">
                  <div className="flex-1 text-center"><span className="text-gray-600 text-xs">↓</span></div>
                  <div className="flex-1 text-center"><span className="text-gray-600 text-xs">↓</span></div>
                </div>
                <div className="w-full px-3 py-2 rounded bg-green-900/30 border border-green-800/50 text-center text-green-400 text-xs">Your APIs</div>
              </div>
              <p className="text-gray-600 text-xs text-center mt-4">Route only agent traffic through SatGate</p>
            </div>

            {/* MCP */}
            <div className="bg-gray-900/50 border border-purple-800/30 rounded-xl p-6 hover:border-purple-700/50 transition">
              <h4 className="text-sm font-bold text-purple-400 mb-4 text-center">MCP PROXY</h4>
              <div className="flex flex-col items-center gap-2 text-sm">
                <div className="w-full px-3 py-2 rounded bg-gray-800 text-center text-gray-400 text-xs">AI Agents</div>
                <span className="text-gray-600">↓</span>
                <div className="w-full px-3 py-2.5 rounded bg-purple-900/40 border border-purple-500/50 text-center">
                  <span className="text-purple-300 font-bold text-xs">SatGate MCP Proxy</span>
                </div>
                <span className="text-gray-600">↓</span>
                <div className="w-full px-3 py-2 rounded bg-green-900/30 border border-green-800/50 text-center text-green-400 text-xs">MCP Servers / Tools</div>
              </div>
              <p className="text-gray-600 text-xs text-center mt-4">Per-tool budgets, delegation trees</p>
            </div>
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
              Four steps to govern agent traffic. No code changes required.
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
                description: "Apply when ready. Version history + policy receipt trail. Rollback if needed.",
                code: `v3 (applied) ← current\nv2 (available)\nv1 (available)\n\nReceipt: who, when, diff`
              },
              {
                step: "3",
                title: "Point Your DNS",
                description: "Use *.satgate.cloud or your custom domain. Traffic flows through SatGate.",
                code: `# Your domain\napi.yoursite.com\n  CNAME → satgate.cloud\n\n# Or use ours\nyourapp.satgate.cloud`
              },
              {
                step: "4",
                title: "Prove What Happened",
                description: "Receipts for allowed, denied, paid, delegated, and revoked decisions — ready to export as an Evidence Pack.",
                code: `Allowed receipts: 1,203\nDenied receipts: 12,847\nPaid receipts:   $847 settled\nDelegations:     42\nRevocations:     9\n\n→ Export Evidence Pack`
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


        </div>
      </section>

      {/* Code Integration Section */}
      <section className="py-20 px-6 border-t border-gray-900 bg-gray-950/40">
        <div className="max-w-4xl mx-auto rounded-2xl border border-gray-800 bg-black p-8 md:p-10">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Agent governance questions</h2>
          <div className="space-y-6">
            {[
              ['What is SatGate?', 'SatGate is the Agent Authority & Accountability Layer for governed agent execution. Humans and platforms use it to delegate bounded economic authority to agents, enforce policy and budgets, prove revocation, and preserve evidence across APIs, MCP tools, and paid external calls.'],
              ['How does SatGate govern AI agents?', 'SatGate applies scoped authority, per-agent policy, revocation, and budgets before each request reaches an API or MCP tool, so unauthorized actions and expensive calls can be blocked before they happen.'],
              ['How does SatGate give agents bounded economic authority?', 'Humans and platforms define policy, budgets, scope, and delegation depth. Agents consume approved API and MCP primitives through SatGate, and every approval, denial, spend event, delegation, and revocation leaves receipt-backed proof.'],
            ].map(([question, answer]) => (
              <div key={question} className="border-t border-gray-800 pt-6 first:border-t-0 first:pt-0">
                <h3 className="mb-2 text-xl font-bold text-white">{question}</h3>
                <p className="leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA / Footer */}
      <footer className="py-20 border-t border-gray-800">
        <div className="max-w-6xl mx-auto px-6">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-6">Ready to govern what your agents can do?</h2>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-4">
              <a href="mailto:contact@satgate.io" className="inline-block bg-gradient-to-r from-purple-600 to-cyan-600 text-white px-10 py-4 rounded-full font-bold text-lg hover:opacity-90 transition shadow-lg shadow-purple-500/20">
                Get in Touch
              </a>
              <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="inline-block bg-white text-black px-10 py-4 rounded-full font-bold text-lg hover:bg-gray-200 transition">
                Start Free →
              </a>
            </div>
            <p className="text-gray-500 text-sm">
              Or <Link href="/policy-to-proof" className="text-purple-400 hover:text-purple-300 transition underline underline-offset-4">see the Policy-to-Proof evidence story →</Link>
            </p>
          </div>

          <div className="grid grid-cols-1 gap-8 border-t border-gray-800 py-12 sm:grid-cols-2 lg:grid-cols-6">
            <div className="lg:col-span-2">
              <div className="flex items-center gap-2 mb-4">
                <Image src="/logo_white_transparent.png" alt="SatGate" width={24} height={24} className="w-6 h-6" />
                <h4 className="font-bold text-white">SatGate</h4>
              </div>
              <p className="max-w-xs text-sm text-gray-500">Agent Authority & Accountability Layer: bounded authority, Evidence Pack receipts, and proof.</p>
              <p className="text-gray-600 text-xs mt-3">Humans and platforms buy. Agents consume bounded primitives.</p>
            </div>
            <div>
              <h4 className="mb-4 font-bold text-white">Start here</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/economic-firewall" className="hover:text-white transition">Economic Firewall</Link></li>
                <li><Link href="/govern" className="hover:text-white transition">Enterprise</Link></li>
                <li><Link href="/policy-to-proof" className="hover:text-white transition">Policy-to-Proof</Link></li>
                <li><Link href="/agent-authority-layer" className="hover:text-white transition">Authority & Accountability</Link></li>
                <li><Link href="/partners/rails" className="hover:text-white transition">Rail Partners</Link></li>
                <li><Link href="/pricing" className="hover:text-white transition">Pricing</Link></li>
                <li><a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud Dashboard</a></li>
              </ul>
            </div>
            <div>
              <h4 className="mb-4 font-bold text-white">Developers</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/build" className="hover:text-white transition">Build</Link></li>
                <li><a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Documentation</a></li>
                <li><a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a></li>
                <li><Link href="/mcp" className="hover:text-white transition">MCP Governance Hub</Link></li>
                <li><Link href="/mcp-gateway" className="hover:text-white transition">MCP Gateway</Link></li>
                <li><Link href="/integrations" className="hover:text-white transition">Integrations</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="mb-4 font-bold text-white">Resources</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/tools" className="hover:text-white transition">Tools</Link></li>
                <li><Link href="/compare" className="hover:text-white transition">Compare</Link></li>
                <li><Link href="/blog" className="hover:text-white transition">Blog</Link></li>
                <li><Link href="/ai-agent-cost-control" className="hover:text-white transition">Cost Control</Link></li>
                <li><Link href="/capability-auth" className="hover:text-white transition">Capability Auth</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="mb-4 font-bold text-white">Company</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/design-partners" className="hover:text-white transition">Design Partners</Link></li>
                <li><Link href="/security" className="hover:text-white transition">Security</Link></li>
                <li><Link href="/terms" className="hover:text-white transition">Terms</Link></li>
                <li><Link href="/privacy" className="hover:text-white transition">Privacy</Link></li>
                <li><a href="mailto:contact@satgate.io" className="hover:text-white transition">contact@satgate.io</a></li>
              </ul>
            </div>
          </div>

          <div className="pt-8 border-t border-gray-800 text-center text-gray-600 text-sm">
            © 2025-2026 SatGate Inc. All rights reserved. SatGateTM is a trademark of SatGate Inc. Patent Pending.
          </div>
        </div>
      </footer>
    </div>
  );
};

// Simple helper component for features
type FeatureCardProps = { icon: React.ReactNode; title: string; desc: string };

const FeatureCard = ({ icon, title, desc }: FeatureCardProps) => (
  <div className="p-6 rounded-xl bg-black border border-gray-800 hover:border-gray-600 transition group">
    <div className="mb-4 p-3 bg-gray-900 rounded-lg w-fit group-hover:bg-gray-800 transition">{icon}</div>
    <h3 className="text-xl font-bold mb-2">{title}</h3>
    <p className="text-gray-400 leading-relaxed">{desc}</p>
  </div>
);

export default LandingPage;
export { LandingPage };

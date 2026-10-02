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
            <Link href="/mcp" className="hover:text-white transition">MCP</Link>
            <Link href="/build" className="hover:text-white transition">Build</Link>
            <Link href="/sandbox" className="hover:text-white transition">Demo</Link>
            <Link href="/pricing" className="hover:text-white transition">Pricing</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://cloud.satgate.io/cloud/signup" target="_blank" rel="noopener noreferrer" className="rounded-full bg-white px-3 py-1.5 font-bold text-black hover:bg-gray-200 transition">Start free trial</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="rounded-full border border-purple-500/40 px-3 py-1.5 text-purple-300 hover:border-purple-400 hover:text-purple-200 transition">Cloud login</a>
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
              href="/mcp"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              MCP
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
              href="/security"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Security
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
              href="https://cloud.satgate.io/cloud/signup"
              target="_blank"
              rel="noopener noreferrer"
              onClick={() => setMobileMenuOpen(false)}
              className="block bg-white text-black font-bold transition py-3 px-4 rounded-lg"
            >
              Start free trial
              <span className="block text-xs font-medium text-gray-600">14 days free. No credit card.</span>
            </a>
            <a
              href="https://cloud.satgate.io/cloud/login"
              target="_blank"
              rel="noopener noreferrer"
              onClick={() => setMobileMenuOpen(false)}
              className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
            >
              Cloud login
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
              <Zap size={12} /> Economic firewall for agents
            </div>
            <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
              An economic firewall<br/>
              <span className="sr-only"> </span><span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
                for AI agents.
              </span>
            </h1>
            <p className="text-xl text-gray-400 mb-4 max-w-lg leading-relaxed">
              Keep your agents within budget. Make external agents pay before they use your API. Get a signed receipt for every decision.
            </p>
            <p className="text-lg text-gray-500 mb-8 max-w-lg leading-relaxed">
              SatGate checks each request before it reaches your API or MCP tool. Limit which APIs and tools each agent can use. Your own agents stop when their budget runs out. External agents pay on the routes you choose, so hammering your API gets expensive, and paying never gets them past your access rules.
            </p>
            <div className="flex flex-wrap gap-4">
              <a href="https://cloud.satgate.io/cloud/signup" target="_blank" rel="noopener noreferrer" className="bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 text-white px-8 py-3 rounded-lg font-bold transition flex items-center gap-2 shadow-lg shadow-purple-500/20">
                Start free trial <ArrowRight size={16} />
              </a>
              <Link href="/sandbox#golden-path" className="border border-purple-700/50 bg-purple-900/20 px-8 py-3 rounded-lg font-bold hover:bg-purple-900/40 transition flex items-center gap-2 text-purple-300">
                <Play size={16} /> Try the 90-second demo
              </Link>
            </div>
            <p className="mt-3 text-sm text-gray-500">
              <Link href="/build" className="text-gray-400 hover:text-white underline underline-offset-2">Build with SatGate</Link>
              <span className="mx-2 text-gray-700">·</span>
              <Link href="/evidence-pack-demo" className="text-gray-400 hover:text-white underline underline-offset-2">See a signed receipt (Evidence Pack)</Link>
            </p>
            <p className="mt-2 text-sm text-gray-500">14 days free. No credit card.</p>

            {/* Proof strip */}
            <div className="mt-6 space-y-2 text-xs text-gray-500">
              <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
                <span className="flex items-center gap-1.5"><CheckCircle size={12} className="text-green-500" /> REST · GraphQL · MCP</span>
                <span className="flex items-center gap-1.5"><CheckCircle size={12} className="text-green-500" /> Gateway · Sidecar · MCP Proxy</span>
                <span className="flex items-center gap-1.5"><CheckCircle size={12} className="text-green-500" /> Checks each request first</span>
              </div>
              <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
                <span className="flex items-center gap-1.5"><CheckCircle size={12} className="text-green-500" /> MCP · HTTP APIs · Lightning payments</span>
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
                <div className="text-xs text-gray-500 ml-2 font-mono">live recording · cloud.satgate.io</div>
              </div>
              <video
                autoPlay
                loop
                muted
                playsInline
                poster="/satgate-demo-poster.jpg"
                className="w-full"
                aria-label="An agent makes three paid MCP tool calls, then its fourth call is refused when the budget runs out"
              >
                <source src="/satgate-hero-live.mp4" type="video/mp4" />
              </video>
              <div className="absolute bottom-4 right-4 bg-black/70 backdrop-blur-sm px-3 py-1 rounded-full text-xs text-gray-300 font-mono">
                3¢ budget · call 4 refused
              </div>
            </div>
            <div className="text-center mt-4">
              <p className="text-sm text-gray-500 mb-3">
                Recorded on production: three $0.01 tool calls, then the fourth is refused before the tool runs.
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
      <section id="see-it-live" className="py-16 px-6 border-b border-gray-800 scroll-mt-20">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-3xl font-bold mb-3">See SatGate in Action</h2>
          <p className="text-gray-400 mb-10 max-w-2xl mx-auto">Two short recordings from production, with narration.</p>
          <h3 className="text-xl font-bold mb-2">Your agents: a budget they can&apos;t overspend</h3>
          <p className="text-gray-400 mb-6 max-w-2xl mx-auto">Mint an agent token with a 3¢ budget, watch each MCP tool call get a signed receipt, see call four refused before the tool runs, then check the receipt with the open-source verifier.</p>
          <div className="relative rounded-xl overflow-hidden border border-gray-700/50 shadow-2xl shadow-purple-500/10">
            <video
              controls
              preload="metadata"
              poster="/satgate-demo-voice-poster.jpg"
              className="w-full"
              playsInline
              aria-label="Narrated demo: an agent with a 3 cent budget makes three paid MCP tool calls, its fourth call is refused before the tool runs, and the signed receipt is checked with the open-source verifier"
            >
              <source src="/satgate-demo.mp4" type="video/mp4" />
              Your browser does not support the video tag.
            </video>
          </div>
          <p className="text-xs text-gray-500 mt-3">The agent in the recording is a scripted MCP client using the public <code className="text-gray-400">satgate-mcp-bridge</code> npm package. Tokens are blurred.</p>

          <h3 className="text-xl font-bold mt-16 mb-2">External agents: pay before they get in</h3>
          <p className="text-gray-400 mb-6 max-w-2xl mx-auto">Set a price on a route in the dashboard. An agent with no wallet gets 402 Payment Required and never reaches your API. You choose how external agents pay on each paid route: Lightning, USDC on Base, or both. With USDC, each payment buys one request. On Lightning routes you choose how many requests one payment buys. In the recording, an agent pays a 10-sat invoice and gets through. On sat-priced routes the price can rise under load, up to a ceiling you set, and unpaid 402s do not raise it. Paying never gets an agent past your access rules, and payment decisions, refusals included, get a signed receipt.</p>
          <div className="relative rounded-xl overflow-hidden border border-gray-700/50 shadow-2xl shadow-yellow-500/10">
            <video
              controls
              preload="metadata"
              poster="/satgate-admit-poster.jpg"
              className="w-full"
              playsInline
              aria-label="Narrated demo: an external agent with no wallet is refused with 402 Payment Required, a second external agent settles a 10 sat Lightning invoice and is allowed through, and the signed receipts pass the open-source verifier"
            >
              <source src="/satgate-admit-demo.mp4" type="video/mp4" />
              Your browser does not support the video tag.
            </video>
          </div>
          <p className="text-xs text-gray-500 mt-3">The agents in the recording are scripted HTTP clients. The invoice was paid from a separate Lightning wallet while the recording was paused. Emails are blurred.</p>
          <div className="mt-6 text-left max-w-2xl mx-auto rounded-lg border border-gray-800 bg-black/60 p-4">
            <p className="text-sm text-gray-300 mb-3">
              Check the refusal from the start of the video yourself. Fetch the{' '}
              <a href="https://api.satgate.io/v1/evidence/evid_GrXvKUgtdqNbuQ5lZzqRMpZrOoU2VAnE" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">live receipt</a>{' '}
              or the{' '}
              <a href="/evidence/admit-payment-refusal-20260929.json" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">downloaded copy</a>, then run the{' '}
              <a href="https://github.com/SatGate-io/satgate/tree/main/tools" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">open-source verifier</a>{' '}
              against SatGate&apos;s public key:
            </p>
            <pre className="text-xs text-gray-400 font-mono whitespace-pre-wrap break-all">python3 tools/verify_evidence_pack.py https://api.satgate.io/v1/evidence/evid_GrXvKUgtdqNbuQ5lZzqRMpZrOoU2VAnE --jwks-url https://api.satgate.io/.well-known/jwks.json --require-trusted-issuer</pre>
            <p className="text-xs text-gray-500 mt-3">
              <Link href="/verify-evidence-pack" className="hover:text-gray-300 underline underline-offset-2">What the verifier checks</Link>
            </p>
          </div>
        </div>
      </section>

      {/* Free SEO Tools */}
      <section className="py-20 px-6 border-b border-gray-800 bg-black">
        <div className="max-w-6xl mx-auto">
          <div className="mb-10 max-w-3xl">
            <p className="mb-3 text-sm font-mono uppercase tracking-wide text-cyan-300">Free tools</p>
            <h2 className="mb-4 text-3xl md:text-4xl font-bold text-white">See what a runaway agent could cost you</h2>
            <p className="text-gray-400 text-lg leading-relaxed">
              Start with these three. The tools page has more calculators.
            </p>
          </div>
          <div className="mb-6">
            <Link href="/tools" className="inline-flex items-center gap-2 text-cyan-300 hover:text-cyan-200 font-semibold transition">
              See all free tools <ArrowRight size={16} />
            </Link>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
            {[
              { href: '/runaway-agent-cost-calculator', title: 'Runaway Agent Cost Calculator', body: 'Estimate what loops, retries and fan-out could cost before anyone notices.', icon: Activity },
              { href: '/ai-agent-runaway-spend-benchmark', title: 'AI Agent Runaway Spend Benchmark', body: 'Our benchmark data on agent loops, retry storms and wasted spend, as JSON and CSV.', icon: BarChart3 },
              { href: '/economic-firewall-readiness-grader', title: 'Economic Firewall Readiness Grader', body: 'Grade your setup on budgets, permissions, revocation, MCP tools and audit records.', icon: Shield },
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
            <h2 className="text-3xl font-bold mb-3">Budgets for your agents. Payment from external agents.</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">Control caps what your own agents can spend. Admit makes external agents pay before their request reaches you. Both leave signed receipts.</p>
          </div>

          {/* Default Protection - Foundation */}
          <div className="p-8 rounded-xl bg-black border-2 border-purple-500/50 mb-6 relative">
            <div className="absolute -top-3 left-6 bg-purple-600 text-xs font-bold px-3 py-1 rounded">
              ON BY DEFAULT
            </div>
            <div className="flex items-center gap-3 mb-4 mt-2">
              <div className="p-3 bg-purple-900/30 rounded-lg">
                <Shield className="text-purple-400" size={24} />
              </div>
              <div>
                <h3 className="text-xl font-bold">Agents only get what you allow</h3>
                <p className="text-gray-500 text-sm">Every route except the ones you mark PUBLIC</p>
              </div>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              Before SatGate forwards a request, it checks what that agent is allowed to do.
              Paying doesn&apos;t change that: a paid request still can&apos;t go past its permissions, use an expired token or get around a revoke.
            </p>
            <div className="flex flex-wrap gap-4 text-sm text-gray-500">
              <span>✓ Permissions per agent</span>
              <span>✓ Narrower tokens for sub-agents</span>
              <span>✓ Revoke works on the next request</span>
              <span>✓ Signed receipts</span>
            </div>
          </div>

          {/* Your Agents */}
          <div className="mb-4">
            <p className="text-sm font-mono text-cyan-400 mb-4 uppercase tracking-wider">Your agents: budgets</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
            {/* Observe - Free */}
            <div className="p-6 rounded-xl bg-black border border-cyan-800/30 hover:border-cyan-600/50 transition relative">
              <div className="text-xs text-purple-400 mb-2">Protected by default →</div>
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2.5 bg-cyan-900/50 rounded-lg">
                  <Eye className="text-cyan-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Observe <span className="text-xs font-normal text-gray-500">(see usage)</span></h3>
              </div>
              <p className="text-gray-400 text-sm mb-3">
                See usage and cost without blocking requests.
              </p>
              <p className="text-xs text-cyan-400/80 mb-3 italic">
                Start here. Nothing is blocked while you see which agents call what, and what it costs.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Nothing is blocked; your agents work as before</li>
                <li>✓ Usage broken down by team and cost center</li>
                <li>✓ See which agents, tools and routes cost the most before you change anything</li>
                <li>✓ Built to add little overhead</li>
              </ul>
            </div>

            {/* Control - included in Pro */}
            <div className="p-6 rounded-xl bg-black border-2 border-purple-500/50 hover:border-purple-400/70 transition relative">
              <div className="text-xs text-purple-400 mb-2">Protected by default →</div>
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2.5 bg-purple-900/50 rounded-lg">
                  <SlidersHorizontal className="text-purple-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Control <span className="text-xs font-normal text-gray-500">(budgets)</span></h3>
              </div>
              <p className="text-gray-400 text-sm mb-3">
                Enforce budgets and permissions before a request runs.
              </p>
              <p className="text-xs text-purple-400/80 mb-3 italic">
                Then turn on limits. An agent that is out of budget or asks for something it isn&apos;t allowed is stopped before the request runs.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ Budget checked on every request</li>
                <li>✓ Your agents spend from a budget you set; nobody pays per call</li>
                <li>✓ Per-agent spending caps</li>
              </ul>
            </div>

          </div>

          {/* Their Agents */}
          <div className="mb-4">
            <p className="text-sm font-mono text-yellow-400 mb-4 uppercase tracking-wider">External agents: access and payment</p>
          </div>
          <div className="grid grid-cols-1 gap-6 mb-6 max-w-lg">
            {/* Admit - external agents above payment rails */}
            <div className="p-6 rounded-xl bg-black border border-yellow-800/30 hover:border-yellow-600/50 transition relative">
              <div className="text-xs text-purple-400 mb-2">Protected by default →</div>
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2.5 bg-yellow-900/50 rounded-lg">
                  <Shield className="text-yellow-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Admit <span className="text-xs font-normal text-gray-500">(external access)</span></h3>
              </div>
              <p className="text-gray-400 text-sm mb-3">
                Give external agents limited access, and charge them where you want to.
              </p>
              <p className="text-xs text-yellow-400/80 mb-3 italic">
                On a paid route, payment comes first, before the request reaches your API, so hammering it gets expensive. Paying never gets an agent past your access rules.
              </p>
              <ul className="text-xs text-gray-500 space-y-1">
                <li>✓ No shared API keys to hand out</li>
                <li>✓ Payment only on the routes you choose</li>
                <li>✓ You set the price and how many requests one payment buys</li>
                <li>✓ You decide what each agent can reach</li>
              </ul>
            </div>
          </div>

          {/* Prove - spans both agent lanes */}
          <div className="p-8 rounded-xl bg-black border-2 border-purple-500/50 mb-6 relative">
            <div className="absolute -top-3 left-6 bg-purple-600 text-xs font-bold px-3 py-1 rounded">
              RECEIPTS FOR BOTH
            </div>
            <div className="flex items-center gap-3 mb-4 mt-2">
              <div className="p-3 bg-purple-900/30 rounded-lg">
                <CheckCircle className="text-purple-400" size={24} />
              </div>
              <div>
                <h3 className="text-xl font-bold">Receipts anyone can check</h3>
                <p className="text-gray-500 text-sm">From Observe, Control and Admit</p>
              </div>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              When SatGate allows, refuses, charges or revokes, it signs a receipt. Receipts roll up into an Evidence Pack.
              Your auditor can check the signatures with the open-source verifier, without having to trust us.
              A receipt shows what SatGate decided. It is not a compliance certificate.
            </p>
            <div className="flex flex-wrap gap-4 text-sm text-gray-500">
              <span>✓ Your agents and external agents</span>
              <span>✓ Refusals too</span>
              <span>✓ Any edit breaks the signature</span>
              <span>✓ Open-source verifier</span>
            </div>
          </div>

          <p className="text-sm text-gray-400 mb-8">
            Charging for access makes abuse more expensive. It doesn&apos;t replace rate limits or access controls, and it only covers traffic that goes through SatGate.
          </p>

          {/* Token Delegation Video */}
          <div className="mt-12 mb-8">
            <div className="text-center mb-6">
              <h3 className="text-2xl font-bold mb-2">Why API keys don&apos;t work when agents hand off work</h3>
              <p className="text-gray-400 max-w-xl mx-auto text-sm">An API key gives full access or none. A SatGate token carries its own budget, permissions and expiry. An agent can hand a sub-agent a narrower token, never a broader one.</p>
            </div>
            <div className="max-w-3xl mx-auto relative rounded-xl overflow-hidden border border-gray-700/50 shadow-2xl shadow-purple-500/10">
              <video
                controls
                preload="metadata"
                poster="/satgate-delegation-poster.jpg"
                className="w-full"
                playsInline
              >
                <source src="/satgate-delegation-v2.mp4" type="video/mp4" />
                Your browser does not support the video tag.
              </video>
            </div>
          </div>

          {/* PUBLIC callout */}
          <div className="p-4 rounded-lg bg-green-950/20 border border-green-900/30">
            <p className="text-sm text-gray-400">
              Mark a route <span className="text-green-400 font-medium">PUBLIC</span> to leave it open, for health checks
              (<code className="bg-gray-800 px-1 rounded text-gray-300">/healthz</code>), docs and webhooks.
              Every other route is protected by default.
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
              An agent gets a token when it starts, like putting an E-ZPass on the windshield. After that, SatGate checks and meters each request as it passes.
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
            SatGate checks the token itself, so it doesn&apos;t call your identity provider on every request.
          </p>
        </div>
      </section>

      {/* Research Alignment */}
      <section className="py-16 px-6 border-b border-gray-800 bg-gradient-to-b from-gray-900/20 to-black">
        <div className="max-w-3xl mx-auto">
          <div className="text-center mb-8">
            <span className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-900/30 border border-blue-500/30 text-blue-300 text-xs font-mono mb-3">
              <BookOpen size={12} /> THE RESEARCH
            </span>
            <h2 className="text-2xl font-bold mb-3">Built for agents that hand work to other agents</h2>
            <p className="text-gray-400 text-sm max-w-2xl mx-auto">
              A 2026 paper on AI delegation describes a problem we see in practice: when agents hand work to other agents,
              each one needs clear limits on what it can do and spend. One approach it proposes is tokens that can only be
              narrowed as they are passed down, such as <span className="text-blue-300">macaroons</span>.
            </p>
            <p className="text-gray-300 text-sm mt-3 font-medium">
              SatGate&apos;s tokens are macaroons.
            </p>
          </div>
          <div className="grid md:grid-cols-3 gap-4 mb-6">
            <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700">
              <Lock className="text-cyan-400 mb-2" size={20} />
              <h3 className="font-bold text-sm mb-1">Limited permissions</h3>
              <p className="text-gray-400 text-xs">
                Each agent gets only the permissions it needs, and each handoff can only narrow them.
              </p>
            </div>
            <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700">
              <DollarSign className="text-green-400 mb-2" size={20} />
              <h3 className="font-bold text-sm mb-1">Budget caps</h3>
              <p className="text-gray-400 text-xs">
                Set budgets per agent and per route. SatGate checks them before forwarding.
              </p>
            </div>
            <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700">
              <Zap className="text-yellow-400 mb-2" size={20} />
              <h3 className="font-bold text-sm mb-1">Stops on the next request</h3>
              <p className="text-gray-400 text-xs">
                Once an agent hits a limit, SatGate refuses its next request.
              </p>
            </div>
          </div>
          <p className="text-gray-500 text-xs text-center">
            We built SatGate because standing API keys and after-the-fact alerts are a bad fit for autonomous systems. The paper put words to a problem we were already seeing in agent deployments.{' '}
            <span className="text-gray-600 ml-1">
              - <a href="https://arxiv.org/abs/2602.11865" target="_blank" rel="noopener noreferrer" className="hover:text-gray-400 underline underline-offset-2">Tomasev et al., 2026</a>
            </span>
          </p>
        </div>
      </section>

      {/* Where It Fits Section - Clean diagrams */}
      <section className="py-16 px-6 border-b border-gray-800">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-2xl font-bold text-center mb-3">Where it fits</h2>
          <p className="text-gray-500 text-center mb-10">Three ways to set it up. Start with one route or one tool.</p>

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
              <p className="text-gray-600 text-xs text-center mt-4">Budgets per tool, including for sub-agents</p>
            </div>
          </div>
        </div>
      </section>

      {/* Feature Grid */}
      {/* How It Works - 4-Step Walkthrough */}
      <section className="py-20 px-6 border-b border-gray-800 bg-gradient-to-b from-gray-900/30 to-black">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-3xl font-bold mb-3">How it works</h2>
            <p className="text-gray-400 max-w-xl mx-auto">
              Most teams start by pointing one endpoint or one MCP tool at SatGate.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            {[
              {
                step: "1",
                title: "Pick a policy",
                description: "Set a policy per route: public for health checks and docs, protected for everything else.",
                code: `routes:\n  - path: /healthz\n    policy: public\n  - path: /v1/*\n    policy: observe\n  - path: /premium/*\n    policy: charge`
              },
              {
                step: "2",
                title: "Apply it",
                description: "Apply it when you are ready. Every version is kept, with a receipt for who changed what, so you can roll back.",
                code: `v3 (applied) ← current\nv2 (available)\nv1 (available)\n\nReceipt: who, when, diff`
              },
              {
                step: "3",
                title: "Point your agents",
                description: "Send agent traffic to api.satgate.io with your tenant header, so every request goes through SatGate.",
                code: `# Agent requests\nGET https://api.satgate.io/v1/...\nX-SatGate-Tenant: your-tenant\n\n# MCP clients\nnpx satgate-mcp-bridge`
              },
              {
                step: "4",
                title: "Check what happened",
                description: "Receipts for allowed, denied, paid, delegated, and revoked decisions, ready to export as an Evidence Pack.",
                code: `Illustrative sample, not live customer data\nAllowed receipts: 1,203\nDenied receipts: 12,847\nPaid receipts:   $847 settled\nDelegations:     42\nRevocations:     9\n\n→ Export Evidence Pack`
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
          <h2 className="mb-8 text-3xl font-bold text-white">Questions</h2>
          <div className="space-y-6">
            {[
              ['What is SatGate?', 'SatGate is a gateway that sits in front of your APIs and MCP tools. It keeps your own agents within the budgets and permissions you set, charges external agents on the routes you choose, and signs a receipt for each decision.'],
              ['How does SatGate control what agents do?', 'Before a request reaches your API or MCP tool, SatGate checks the agent’s permissions, its remaining budget and whether its token was revoked. If a check fails, the request stops at SatGate.'],
              ['How do I give an agent a budget?', 'You set the budget, the permissions and how many times it can hand work to a sub-agent. The agent gets a token with those limits built in. Every allow, refusal, payment, handoff and revoke gets a receipt.'],
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
            <h2 className="text-3xl font-bold mb-6">Ready to set limits for your agents?</h2>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-4">
              <a href="https://cloud.satgate.io/cloud/signup" target="_blank" rel="noopener noreferrer" className="inline-block bg-white text-black px-10 py-4 rounded-full font-bold text-lg hover:bg-gray-200 transition">
                Start free trial
              </a>
              <a href="mailto:contact@satgate.io" className="inline-block bg-gradient-to-r from-purple-600 to-cyan-600 text-white px-10 py-4 rounded-full font-bold text-lg hover:opacity-90 transition shadow-lg shadow-purple-500/20">
                Get in Touch
              </a>
            </div>
            <p className="text-gray-500 text-sm mb-4">14 days free. No credit card. Want help rolling out? <Link href="/design-partners" className="text-purple-400 hover:text-purple-300 transition underline underline-offset-4">Ask about a 90-day design-partner pilot</Link></p>
            <p className="text-gray-500 text-sm">
              Or <Link href="/policy-to-proof" className="text-purple-400 hover:text-purple-300 transition underline underline-offset-4">see how the receipts work →</Link>
            </p>
          </div>

          <div className="grid grid-cols-1 gap-8 border-t border-gray-800 py-12 sm:grid-cols-2 lg:grid-cols-6">
            <div className="lg:col-span-2">
              <div className="flex items-center gap-2 mb-4">
                <Image src="/logo_white_transparent.png" alt="SatGate" width={24} height={24} className="w-6 h-6" />
                <h4 className="font-bold text-white">SatGate</h4>
              </div>
              <p className="max-w-xs text-sm text-gray-500">Limits for your agents, payment from external agents, and a receipt for every decision.</p>
              <p className="text-gray-600 text-xs mt-3">You set the rules. SatGate enforces them on every request.</p>
            </div>
            <div>
              <h4 className="mb-4 font-bold text-white">Start here</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/economic-firewall" className="hover:text-white transition">Economic Firewall</Link></li>
                <li><Link href="/govern" className="hover:text-white transition">Enterprise</Link></li>
                <li><Link href="/policy-to-proof" className="hover:text-white transition">Rules and Receipts</Link></li>
                <li><Link href="/pricing" className="hover:text-white transition">Pricing</Link></li>
                <li><a href="https://cloud.satgate.io/cloud/signup" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Start free trial</a></li>
                <li><a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud login</a></li>
              </ul>
            </div>
            <div>
              <h4 className="mb-4 font-bold text-white">Developers</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/build" className="hover:text-white transition">Build</Link></li>
                <li><a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Documentation</a></li>
                <li><a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a></li>
                <li><Link href="/mcp" className="hover:text-white transition">MCP quickstart</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="mb-4 font-bold text-white">Resources</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/tools" className="hover:text-white transition">Tools</Link></li>
                <li><Link href="/compare" className="hover:text-white transition">Compare</Link></li>
                <li><Link href="/blog" className="hover:text-white transition">Blog</Link></li>
                <li><Link href="/ai-agent-cost-control" className="hover:text-white transition">Cost Control</Link></li>
                <li><Link href="/partners/rails" className="hover:text-white transition">Rail Partners</Link></li>
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

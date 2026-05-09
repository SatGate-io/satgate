'use client';

import React, { useState } from 'react';
import {
  Eye, Sliders, Shield, AlertTriangle, Users,
  ArrowRight, CheckCircle, DollarSign, Lock, Key, Bot,
  GitBranch, Activity, Cloud, Container, FileCode, Menu, X,
  ChevronRight, Zap, FileSearch,
  Network, Timer, Coins, Layers
} from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

export default function GovernPage() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans selection:bg-purple-500 selection:text-white">

      {/* Navigation */}
      <nav className="border-b border-gray-800 backdrop-blur-md fixed w-full z-50 bg-black/50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2 shrink-0">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="w-7 h-7 sm:w-8 sm:h-8" />
            <span className="text-lg sm:text-xl font-bold text-white whitespace-nowrap">SatGate<sup className="text-xs font-normal">™</sup></span>
          </Link>

          <div className="hidden md:flex gap-6 text-sm font-medium text-gray-400">
            <Link href="/govern" className="text-white transition">Enterprise</Link>
            <Link href="/pricing" className="hover:text-white transition">Pricing</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud</a>
          </div>

          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="flex md:hidden items-center justify-center w-10 h-10 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 text-gray-400 hover:text-white transition"
            aria-label="Toggle menu"
            aria-expanded={mobileMenuOpen}
          >
            {mobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
        </div>

        <div className={`md:hidden overflow-hidden transition-all duration-300 ease-in-out ${mobileMenuOpen ? 'max-h-96 opacity-100' : 'max-h-0 opacity-0'}`}>
          <div className="bg-black/95 backdrop-blur-xl border-t border-gray-800 px-4 py-4 space-y-1">
            {[
              { href: '/govern', label: 'Enterprise' },
              { href: '/pricing', label: 'Pricing' },
            ].map(item => (
              <Link key={item.href} href={item.href} onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">
                {item.label}
              </Link>
            ))}
            <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">
              Docs
            </a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">
              Cloud
            </a>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <header className="pt-32 pb-20 px-6">
        <div className="max-w-4xl mx-auto text-center">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono mb-6">
            <Shield size={12} /> Enterprise agent governance
          </div>
          <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
            Govern what agents can do.
            <span className="block text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
              Prove every decision.
            </span>
          </h1>
          <p className="text-xl text-gray-400 mb-10 max-w-2xl mx-auto leading-relaxed">
            SatGate is the economic control plane for enterprise agents: scope authority before work starts, enforce policy at runtime, and preserve evidence after every allowed, denied, delegated, or revoked action.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/policy-to-proof" className="bg-white text-black px-8 py-3 rounded-lg font-bold hover:bg-gray-200 transition flex items-center justify-center gap-2">
              See Policy-to-Proof <ArrowRight size={18} />
            </Link>
            <Link href="https://cloud.satgate.io/cloud/login" className="border border-gray-700 px-8 py-3 rounded-lg font-bold hover:border-gray-500 transition flex items-center justify-center gap-2">
              Start Free <Eye size={18} />
            </Link>
          </div>
        </div>
      </header>

      {/* One Token, Two Threats */}
      <section className="py-16 px-6 border-t border-gray-800 bg-gradient-to-b from-gray-900/20 to-black">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-3">Govern internal agents. Preserve proof across external rails.</h2>
          <p className="text-gray-500 text-center mb-10 max-w-2xl mx-auto">
            The same capability chain scopes authority, enforces spend limits, preserves delegation lineage, and proves what happened whether the call stays internal or crosses into paid external access.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-3xl mx-auto">
            {/* Internal */}
            <div className="p-6 rounded-xl bg-black border-2 border-cyan-500/40 hover:border-cyan-400/60 transition">
              <div className="flex items-center gap-2 mb-3">
                <div className="p-2 bg-cyan-900/50 rounded-lg">
                  <DollarSign className="text-cyan-400" size={20} />
                </div>
                <div>
                  <p className="text-xs font-mono text-cyan-400 uppercase tracking-wider">Your Agents (Internal)</p>
                </div>
              </div>
              <h3 className="text-xl font-bold text-white mb-2">Govern <span className="text-cyan-400">Authority</span></h3>
              <p className="text-gray-400 text-sm mb-3">No scoped capability → no action. SatGate decides what each agent can do, spend, delegate, and continue doing before the request reaches your API.</p>
              <div className="flex items-center gap-2 text-xs text-gray-500">
                <Shield size={12} className="text-cyan-400" />
                <span>Govern → enforce → prove policy ratchet</span>
              </div>
            </div>

            {/* External */}
            <div className="p-6 rounded-xl bg-black border-2 border-yellow-500/40 hover:border-yellow-400/60 transition">
              <div className="flex items-center gap-2 mb-3">
                <div className="p-2 bg-yellow-900/50 rounded-lg">
                  <Lock className="text-yellow-400" size={20} />
                </div>
                <div>
                  <p className="text-xs font-mono text-yellow-400 uppercase tracking-wider">Their Agents (External)</p>
                </div>
              </div>
              <h3 className="text-xl font-bold text-white mb-2">Preserve <span className="text-yellow-400">Proof</span></h3>
              <p className="text-gray-400 text-sm mb-3">Payment proves value moved. SatGate proves the agent was allowed to move it across L402, x402, API-key billing, or enterprise ledgers.</p>
              <div className="flex items-center gap-2 text-xs text-gray-500">
                <Zap size={12} className="text-yellow-400" />
                <span>Rail-aware evidence, payment or not</span>
              </div>
            </div>
          </div>

          <div className="text-center mt-8">
            <div className="inline-flex items-center gap-3 px-5 py-2.5 bg-gray-900 border border-gray-800 rounded-full text-sm">
              <Key size={16} className="text-purple-400" />
              <span className="text-gray-400">Same <span className="text-white font-semibold">macaroon capability</span> — same authority chain — one evidence trail</span>
            </div>
          </div>
        </div>
      </section>

      {/* Three Personas */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">Built for enterprise control owners</h2>
          <p className="text-gray-500 text-center mb-4">Security. Finance. Platform.</p>
          <p className="text-xs text-gray-600 text-center mb-12">Scoped authority, enforceable limits, and audit-ready evidence for autonomous agent work.</p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* CFO — Margins (Observe) */}
            <div className="p-6 rounded-xl bg-gradient-to-br from-cyan-950/30 to-cyan-900/10 border-2 border-cyan-500/50 hover:border-cyan-400/70 transition group">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-cyan-900/50 rounded-lg group-hover:bg-cyan-900/70 transition">
                  <DollarSign className="text-cyan-400" size={22} />
                </div>
                <div>
                  <h3 className="font-bold text-lg">FinOps</h3>
                  <span className="text-xs text-cyan-400 font-medium">Attributed agent spend</span>
                </div>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-3">
                <span className="text-cyan-300 font-medium">The problem:</span> Shared API keys and pooled model bills hide which agent, route, token, or workflow created spend.
              </p>
              <p className="text-gray-300 text-sm leading-relaxed mb-4 border-l-2 border-cyan-800 pl-3">
                &ldquo;SatGate turns spend into evidence: who created it, which authority allowed it, which policy governed it, and where the budget stopped.&rdquo;
              </p>
              <div className="pt-3 border-t border-cyan-900/30">
                <p className="text-xs text-cyan-400 font-medium mb-2">Metrics that matter:</p>
                <p className="text-xs text-gray-400">Attribute spend to agent, token, route, tool, and policy before finance has to reconstruct it</p>
              </div>
              <div className="mt-4">
                <Link href="/policy-to-proof" className="text-xs text-cyan-400 hover:text-cyan-300 transition">See the evidence pack →</Link>
              </div>
            </div>

            {/* CISO — Risk (Control) */}
            <div className="p-6 rounded-xl bg-gradient-to-br from-red-950/30 to-red-900/10 border border-red-800/30 hover:border-red-600/50 transition group">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-red-900/50 rounded-lg group-hover:bg-red-900/70 transition">
                  <Shield className="text-red-400" size={22} />
                </div>
                <div>
                  <h3 className="font-bold text-lg">CISO</h3>
                  <span className="text-xs text-red-400 font-medium">Revocation proof</span>
                </div>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-3">
                <span className="text-red-300 font-medium">The problem:</span> Agents now hold authority that used to require human approval. Security needs proof that scope, delegation, denial, and revocation actually worked.
              </p>
              <p className="text-gray-300 text-sm leading-relaxed mb-4 border-l-2 border-red-800 pl-3">
                &ldquo;SatGate gives us a kill switch with evidence: the revocation receipt, the first denied call after revoke, and the authority chain that led there.&rdquo;
              </p>
              <div className="pt-3 border-t border-red-900/30">
                <p className="text-xs text-red-400 font-medium mb-2">Metrics that matter:</p>
                <p className="text-xs text-gray-400">Prove unauthorized actions were blocked before data or spend moved</p>
              </div>
              <div className="mt-4">
                <Link href="/policy-to-proof" className="text-xs text-red-400 hover:text-red-300 transition">Review Policy-to-Proof →</Link>
              </div>
            </div>

            {/* Platform — protocol-independent governance */}
            <div className="p-6 rounded-xl bg-gradient-to-br from-yellow-950/30 to-yellow-900/10 border border-yellow-800/30 hover:border-yellow-600/50 transition group">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-yellow-900/50 rounded-lg group-hover:bg-yellow-900/70 transition">
                  <Zap className="text-yellow-400" size={22} />
                </div>
                <div>
                  <h3 className="font-bold text-lg">Platform</h3>
                  <span className="text-xs text-yellow-400 font-medium">Protocol-independent governance</span>
                </div>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-3">
                <span className="text-yellow-300 font-medium">The problem:</span> Agents are moving across internal APIs, MCP tools, and paid external rails. Governance cannot depend on which protocol the agent used.
              </p>
              <p className="text-gray-300 text-sm leading-relaxed mb-4 border-l-2 border-yellow-800 pl-3">
                &ldquo;SatGate sits above MCP, API keys, x402, L402, and enterprise billing rails so every authority decision is recorded — payment or not.&rdquo;
              </p>
              <div className="pt-3 border-t border-yellow-900/30">
                <p className="text-xs text-yellow-400 font-medium mb-2">Metrics that matter:</p>
                <p className="text-xs text-gray-400">One control path for internal workflows and governed external paid calls</p>
              </div>
              <div className="mt-4">
                <Link href="/agent-control-plane" className="text-xs text-yellow-400 hover:text-yellow-300 transition">See the control plane →</Link>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Govern → Enforce → Prove */}
      <section id="observe" className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-gray-900/30 to-black">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">Govern, enforce, prove.</h2>
          <p className="text-gray-500 text-center mb-12">Start with visibility, move enforcement into the request path, then export the evidence when security, finance, or audit asks what happened.</p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Observe */}
            <div className="relative p-6 rounded-xl bg-gray-900 border-2 border-purple-500/60 hover:border-purple-400/80 transition">
              <div className="flex items-center gap-3 mb-4 mt-2">
                <div className="p-2.5 bg-purple-900/50 rounded-lg">
                  <Eye className="text-purple-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Observe</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-4">
                Map every agent, token, route, MCP tool, model, and workflow before enforcing policy. Observe mode builds the authority and spend picture without disrupting teams.
              </p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="text-purple-400">✓ Agent and token inventory</li>
                <li className="text-purple-400">✓ Route, tool, and tenant attribution</li>
                <li className="text-purple-400">✓ Spend and authority mapping</li>
                <li className="text-purple-400">✓ Evidence-ready event capture</li>
              </ul>
            </div>

            {/* Control */}
            <div className="p-6 rounded-xl bg-gray-900 border border-cyan-800/30 hover:border-cyan-600/50 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-cyan-900/50 rounded-lg">
                  <Sliders className="text-cyan-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Control</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-4">
                Enforce scoped capabilities, budgets, tenant boundaries, delegation depth, expiry, and revocation before the next request reaches an API or MCP tool.
              </p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="text-cyan-400">✓ Per-agent and per-route policy</li>
                <li className="text-cyan-400">✓ Scope and caveat enforcement</li>
                <li className="text-cyan-400">✓ Delegation containment</li>
                <li className="text-cyan-400">✓ Revocation before execution</li>
              </ul>
            </div>

            {/* Prove */}
            <div className="p-6 rounded-xl bg-gray-900 border border-yellow-800/30 hover:border-yellow-600/50 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-yellow-900/50 rounded-lg">
                  <FileSearch className="text-yellow-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Prove</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-4">
                Export mint receipts, delegation chains, spend ledgers, denial reasons, revocation proof, and paid-rail context in one Evidence Pack.
              </p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="text-yellow-400">✓ Evidence Pack exports</li>
                <li className="text-yellow-400">✓ Signed denial and revocation trail</li>
                <li className="text-yellow-400">✓ x402/L402/API-key billing context</li>
                <li className="text-yellow-400">✓ Audit-ready authority timeline</li>
              </ul>
            </div>
          </div>

          <div className="text-center mt-8">
            <div className="inline-flex items-center gap-2 px-4 py-2 bg-gray-900 border border-gray-800 rounded-full text-sm text-gray-400">
              <Eye size={14} className="text-purple-400" />
              <span>Observe</span>
              <ChevronRight size={14} className="text-gray-600" />
              <Sliders size={14} className="text-cyan-400" />
              <span>Control</span>
              <ChevronRight size={14} className="text-gray-600" />
              <FileSearch size={14} className="text-yellow-400" />
              <span>Prove</span>
            </div>
          </div>
        </div>
      </section>

      {/* Runaway Agent Story */}
      <section className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-red-950/10 to-black">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-red-900/30 border border-red-500/30 text-red-300 text-xs font-mono mb-6">
              <AlertTriangle size={12} /> Real-World Scenario
            </div>
            <h2 className="text-3xl font-bold mb-4">The runaway agent story</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              Friday afternoon, an agent workflow gets a broad token “just to test something.” By Monday, it has created $47,000 in API calls. SatGate changes the ending by enforcing policy and preserving proof.
            </p>
          </div>

          <div className="relative">
            {/* Timeline line */}
            <div className="absolute left-6 md:left-1/2 top-0 bottom-0 w-px bg-gradient-to-b from-red-500/50 via-yellow-500/50 via-cyan-500/50 to-green-500/50 hidden sm:block"></div>

            <div className="space-y-8">
              {/* Step 1 */}
              <div className="relative flex flex-col md:flex-row items-start gap-4 md:gap-8">
                <div className="md:w-1/2 md:text-right md:pr-8">
                  <div className="p-5 rounded-xl bg-gray-900 border border-red-800/30">
                    <div className="flex items-center gap-2 mb-2 md:justify-end">
                      <span className="text-red-400 font-mono text-xs">FRIDAY 4:47 PM</span>
                    </div>
                    <h4 className="font-semibold text-white mb-1 flex items-center gap-2 md:justify-end">
                      <Bot size={16} className="text-red-400" /> Workflow Gets Broad Token
                    </h4>
                    <p className="text-gray-500 text-sm">
                      &ldquo;Just a quick test.&rdquo; The workflow gets an API token with no budget limit, no scope restriction, no expiry.
                    </p>
                  </div>
                </div>
                <div className="hidden md:flex absolute left-1/2 -translate-x-1/2 w-10 h-10 bg-red-900/50 border-2 border-red-500 rounded-full items-center justify-center z-10">
                  <span className="text-red-400 font-bold text-sm">1</span>
                </div>
                <div className="md:w-1/2 md:pl-8"></div>
              </div>

              {/* Step 2 */}
              <div className="relative flex flex-col md:flex-row items-start gap-4 md:gap-8">
                <div className="md:w-1/2 md:pr-8"></div>
                <div className="hidden md:flex absolute left-1/2 -translate-x-1/2 w-10 h-10 bg-red-900/50 border-2 border-red-500 rounded-full items-center justify-center z-10">
                  <span className="text-red-400 font-bold text-sm">2</span>
                </div>
                <div className="md:w-1/2 md:pl-8">
                  <div className="p-5 rounded-xl bg-gray-900 border border-red-800/30">
                    <div className="flex items-center gap-2 mb-2">
                      <span className="text-red-400 font-mono text-xs">SATURDAY — SUNDAY</span>
                    </div>
                    <h4 className="font-semibold text-white mb-1 flex items-center gap-2">
                      <Zap size={16} className="text-red-400" /> Agent Burns Budget
                    </h4>
                    <p className="text-gray-500 text-sm">
                      The test script runs in a loop. 2.3 million API calls. $47,000 in compute. Nobody notices.
                    </p>
                  </div>
                </div>
              </div>

              {/* Step 3 — SatGate catches it */}
              <div className="relative flex flex-col md:flex-row items-start gap-4 md:gap-8">
                <div className="md:w-1/2 md:text-right md:pr-8">
                  <div className="p-5 rounded-xl bg-gray-900 border border-yellow-800/30">
                    <div className="flex items-center gap-2 mb-2 md:justify-end">
                      <span className="text-yellow-400 font-mono text-xs">WITH SATGATE: FRIDAY 4:48 PM</span>
                    </div>
                    <h4 className="font-semibold text-white mb-1 flex items-center gap-2 md:justify-end">
                      <AlertTriangle size={16} className="text-yellow-400" /> CISO Gets Alert
                    </h4>
                    <p className="text-gray-500 text-sm">
                      SatGate detects anomalous token creation. Budget threshold hit after $50. Alert fires in 60 seconds.
                    </p>
                  </div>
                </div>
                <div className="hidden md:flex absolute left-1/2 -translate-x-1/2 w-10 h-10 bg-yellow-900/50 border-2 border-yellow-500 rounded-full items-center justify-center z-10">
                  <span className="text-yellow-400 font-bold text-sm">3</span>
                </div>
                <div className="md:w-1/2 md:pl-8"></div>
              </div>

              {/* Step 4 */}
              <div className="relative flex flex-col md:flex-row items-start gap-4 md:gap-8">
                <div className="md:w-1/2 md:pr-8"></div>
                <div className="hidden md:flex absolute left-1/2 -translate-x-1/2 w-10 h-10 bg-cyan-900/50 border-2 border-cyan-500 rounded-full items-center justify-center z-10">
                  <span className="text-cyan-400 font-bold text-sm">4</span>
                </div>
                <div className="md:w-1/2 md:pl-8">
                  <div className="p-5 rounded-xl bg-gray-900 border border-cyan-800/30">
                    <div className="flex items-center gap-2 mb-2">
                      <span className="text-cyan-400 font-mono text-xs">FRIDAY 4:49 PM</span>
                    </div>
                    <h4 className="font-semibold text-white mb-1 flex items-center gap-2">
                      <Lock size={16} className="text-cyan-400" /> CTO Revokes Token
                    </h4>
                    <p className="text-gray-500 text-sm">
                      One click. Token revoked globally. All child tokens die instantly. Total cost: $50 instead of $47,000.
                    </p>
                  </div>
                </div>
              </div>

              {/* Step 5 */}
              <div className="relative flex flex-col md:flex-row items-start gap-4 md:gap-8">
                <div className="md:w-1/2 md:text-right md:pr-8">
                  <div className="p-5 rounded-xl bg-gray-900 border border-green-800/30">
                    <div className="flex items-center gap-2 mb-2 md:justify-end">
                      <span className="text-green-400 font-mono text-xs">AUTOMATIC</span>
                    </div>
                    <h4 className="font-semibold text-white mb-1 flex items-center gap-2 md:justify-end">
                      <CheckCircle size={16} className="text-green-400" /> Audit Trail Complete
                    </h4>
                    <p className="text-gray-500 text-sm">
                      Full Evidence Pack: who authorized it, what it could do, what it spent, what was denied, when it was revoked, and the first denied call after revoke.
                    </p>
                  </div>
                </div>
                <div className="hidden md:flex absolute left-1/2 -translate-x-1/2 w-10 h-10 bg-green-900/50 border-2 border-green-500 rounded-full items-center justify-center z-10">
                  <span className="text-green-400 font-bold text-sm">5</span>
                </div>
                <div className="md:w-1/2 md:pl-8"></div>
              </div>
            </div>
          </div>

          <div className="text-center mt-12">
            <div className="inline-block bg-gradient-to-r from-green-900/20 to-cyan-900/20 border border-green-800/30 rounded-xl px-6 py-4">
              <p className="text-white font-semibold">Turns a $47,000 weekend incident into a bounded policy event with an audit trail.</p>
              <p className="text-gray-500 text-sm mt-1">Govern first. Enforce before execution. Prove every decision afterward.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Dashboard Screenshots */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">Dashboards that end in evidence.</h2>
          <p className="text-gray-500 text-center mb-12">Real-time visibility into every agent, API call, policy decision, revocation, and dollar — with exports when proof matters.</p>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {[
              {
                title: 'Usage Dashboard',
                description: 'Real-time API call volume, latency percentiles, error rates. Broken down by agent, team, and endpoint.',
                gradient: 'from-purple-600 to-cyan-600',
                icon: <Activity size={24} className="text-purple-400" />,
              },
              {
                title: 'CFO Chargeback Report',
                description: 'Automated spend attribution. See which agent, token, route, and policy created each dollar. Export to finance or the Evidence Pack.', 
                gradient: 'from-green-600 to-emerald-600',
                icon: <DollarSign size={24} className="text-green-400" />,
              },
              {
                title: 'Security Dashboard',
                description: 'Active tokens, anomalous behavior, blocked requests, revocation history. CISO-ready at a glance.',
                gradient: 'from-red-600 to-orange-600',
                icon: <Shield size={24} className="text-red-400" />,
              },
              {
                title: 'Compliance Export',
                description: 'SOC2, GDPR, HIPAA audit trails. Every token lifecycle event, every access decision, timestamped and immutable.',
                gradient: 'from-cyan-600 to-blue-600',
                icon: <FileSearch size={24} className="text-cyan-400" />,
              },
            ].map((item) => (
              <div key={item.title} className="relative group">
                <div className={`absolute -inset-0.5 bg-gradient-to-r ${item.gradient} rounded-xl blur opacity-20 group-hover:opacity-40 transition duration-500 pointer-events-none`}></div>
                <div className="relative bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
                  <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-800 bg-gray-900/80">
                    <div className="w-3 h-3 rounded-full bg-red-500"></div>
                    <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
                    <div className="w-3 h-3 rounded-full bg-green-500"></div>
                    <div className="text-xs text-gray-500 ml-2 font-mono">{item.title}</div>
                  </div>
                  <div className="p-8 min-h-[200px] flex flex-col items-center justify-center text-center">
                    <div className="mb-4">{item.icon}</div>
                    <h4 className="font-semibold text-white mb-2">{item.title}</h4>
                    <p className="text-gray-500 text-sm max-w-xs">{item.description}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Token Delegation Hierarchy */}
      <section className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-gray-900/30 to-black">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">Delegation that leaves a trail</h2>
          <p className="text-gray-500 text-center mb-12">
            Tokens flow down. Authority narrows. Every level is scoped, budgeted, time-limited, and preserved for later proof.
          </p>

          {/* Tree Visualization */}
          <div className="max-w-3xl mx-auto">
            {/* Root */}
            <div className="flex justify-center mb-4">
              <div className="p-4 rounded-xl bg-purple-900/30 border-2 border-purple-500/60 text-center min-w-[220px]">
                <div className="flex items-center justify-center gap-2 mb-1">
                  <Key size={18} className="text-purple-400" />
                  <span className="font-bold text-purple-300">Root Token</span>
                </div>
                <p className="text-xs text-gray-400">CTO • All scopes • $∞ budget</p>
              </div>
            </div>

            {/* Connector */}
            <div className="flex justify-center mb-4">
              <div className="w-px h-8 bg-gray-700"></div>
            </div>

            {/* Department Level */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
              {[
                { name: 'Engineering VP', scope: 'api:eng:*', budget: '$10k/mo', color: 'cyan' },
                { name: 'Data Science VP', scope: 'api:data:*', budget: '$25k/mo', color: 'green' },
                { name: 'Marketing VP', scope: 'api:mkt:*', budget: '$5k/mo', color: 'yellow' },
              ].map((dept) => (
                <div key={dept.name} className={`p-4 rounded-xl bg-gray-900 border border-${dept.color}-800/30 text-center`}>
                  <div className="flex items-center justify-center gap-2 mb-1">
                    <Users size={16} className={`text-${dept.color}-400`} />
                    <span className={`font-semibold text-${dept.color}-300 text-sm`}>{dept.name}</span>
                  </div>
                  <p className="text-xs text-gray-500 font-mono">{dept.scope}</p>
                  <p className="text-xs text-gray-500">{dept.budget}</p>
                </div>
              ))}
            </div>

            {/* Connector */}
            <div className="flex justify-center mb-4">
              <div className="w-px h-8 bg-gray-700"></div>
            </div>

            {/* Agent Level */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {[
                { name: 'RAG Agent', scope: 'api:data:read', budget: '$500/day', ttl: '24h' },
                { name: 'Code Agent', scope: 'api:eng:deploy', budget: '$200/day', ttl: '8h' },
                { name: 'Support Bot', scope: 'api:mkt:chat', budget: '$50/day', ttl: '1h' },
                { name: 'Analytics', scope: 'api:data:query', budget: '$1k/day', ttl: '12h' },
              ].map((agent) => (
                <div key={agent.name} className="p-3 rounded-lg bg-gray-900/80 border border-gray-800 text-center">
                  <div className="flex items-center justify-center gap-1.5 mb-1">
                    <Bot size={14} className="text-gray-400" />
                    <span className="font-medium text-gray-300 text-xs">{agent.name}</span>
                  </div>
                  <p className="text-[10px] text-gray-500 font-mono">{agent.scope}</p>
                  <div className="flex items-center justify-center gap-2 mt-1">
                    <span className="text-[10px] text-green-400/70 flex items-center gap-0.5"><Coins size={10} />{agent.budget}</span>
                    <span className="text-[10px] text-purple-400/70 flex items-center gap-0.5"><Timer size={10} />{agent.ttl}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Properties */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-12 max-w-3xl mx-auto">
            {[
              { label: 'Scoped', desc: 'Narrower than parent', icon: <Lock size={16} className="text-cyan-400" /> },
              { label: 'Budgeted', desc: 'Hard spending limits', icon: <Coins size={16} className="text-green-400" /> },
              { label: 'Time-Limited', desc: 'Auto-expire by policy', icon: <Timer size={16} className="text-purple-400" /> },
              { label: 'Delegatable', desc: 'Agents can sub-delegate', icon: <GitBranch size={16} className="text-yellow-400" /> },
            ].map((prop) => (
              <div key={prop.label} className="p-4 rounded-xl bg-gray-900 border border-gray-800 text-center">
                <div className="flex justify-center mb-2">{prop.icon}</div>
                <h4 className="font-semibold text-white text-sm">{prop.label}</h4>
                <p className="text-gray-500 text-xs mt-1">{prop.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Deployment Options */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-4">Single Go Binary. Zero Dependencies.</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              Deploy anywhere in minutes. No JVM, no runtime, no dependency hell. One binary that runs on anything.
            </p>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
            {[
              { icon: <Container size={24} className="text-cyan-400" />, label: 'Docker', desc: 'ghcr.io/satgate-io/gateway' },
              { icon: <Layers size={24} className="text-purple-400" />, label: 'Kubernetes', desc: 'Helm chart included' },
              { icon: <FileCode size={24} className="text-green-400" />, label: 'Terraform', desc: 'IaC modules ready' },
              { icon: <Cloud size={24} className="text-yellow-400" />, label: 'SaaS', desc: 'cloud.satgate.io' },
            ].map((opt) => (
              <div key={opt.label} className="p-5 rounded-xl bg-gray-900 border border-gray-800 text-center hover:border-gray-600 transition">
                <div className="flex justify-center mb-3">{opt.icon}</div>
                <h4 className="font-semibold text-white text-sm mb-1">{opt.label}</h4>
                <p className="text-gray-500 text-xs font-mono">{opt.desc}</p>
              </div>
            ))}
          </div>

          <div className="bg-gradient-to-r from-cyan-900/20 to-purple-900/20 border border-cyan-800/30 rounded-xl p-6 text-center">
            <div className="flex items-center justify-center gap-2 mb-2">
              <Network size={18} className="text-cyan-400" />
              <h4 className="font-semibold text-white">Hybrid Mode</h4>
            </div>
            <p className="text-gray-400 text-sm max-w-lg mx-auto">
              Gateway runs in your VPC. Dashboard in our cloud (or yours). Your data never leaves your infrastructure.
              The best of both worlds: self-hosted security, managed convenience.
            </p>
          </div>
        </div>
      </section>

      {/* Authority Commons */}
      <section className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-orange-950/10 to-black">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-orange-900/30 border border-orange-500/30 text-orange-300 text-xs font-mono mb-6">
              <AlertTriangle size={12} /> The Core Problem
            </div>
            <h2 className="text-3xl font-bold mb-4">The authority commons — inside your company</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              Every shared internal API is now an authority surface. When agents inherit broad credentials, nobody can prove which policy allowed the action, which team owned it, or where the limit should have stopped it. 
              <span className="text-white font-medium"> Governance fails when authority is invisible.</span>
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
            <div className="p-6 rounded-xl bg-red-950/20 border border-red-900/30">
              <h3 className="font-bold text-red-300 mb-3 flex items-center gap-2">
                <AlertTriangle size={18} className="text-red-400" /> Without SatGate
              </h3>
              <ul className="space-y-3 text-sm text-gray-400">
                <li>• Agents inherit broad API keys</li>
                <li>• Delegation happens outside the control plane</li>
                <li>• Spend and access are reconstructed after the fact</li>
                <li>• Revocation requires rotating shared secrets</li>
                <li className="text-red-400 font-medium">• Audit depends on screenshots and log joins</li>
              </ul>
            </div>
            <div className="p-6 rounded-xl bg-green-950/20 border border-green-900/30">
              <h3 className="font-bold text-green-300 mb-3 flex items-center gap-2">
                <CheckCircle size={18} className="text-green-400" /> With SatGate
              </h3>
              <ul className="space-y-3 text-sm text-gray-400">
                <li>• Every agent gets scoped, revocable authority</li>
                <li>• Delegated workers inherit narrower limits</li>
                <li>• Policy, budget, and scope denials stop requests before execution</li>
                <li>• Revocation creates proof plus a post-revoke denial</li>
                <li className="text-green-400 font-medium">• Evidence Pack exports answer who, what, why, spend, denial, and revoke</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* Internal API Market */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono mb-6">
              <Coins size={12} /> The Opportunity
            </div>
            <h2 className="text-3xl font-bold mb-4">Your internal APIs are authority surfaces</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              Every internal service — databases, search indexes, ML models — now receives autonomous agent requests. SatGate makes the authority visible and enforceable before the call runs, then proves what happened afterward. 
              It&apos;s not just usage tracking. It&apos;s <span className="text-white font-medium">policy-to-proof governance</span>.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800 text-center">
              <div className="text-3xl mb-3">📊</div>
              <h4 className="font-semibold text-white mb-2">Authority discovery</h4>
              <p className="text-gray-500 text-sm">See which agents, teams, tools, and workflows are asking for power before you enforce limits.</p>
            </div>
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800 text-center">
              <div className="text-3xl mb-3">⚖️</div>
              <h4 className="font-semibold text-white mb-2">Runtime enforcement</h4>
              <p className="text-gray-500 text-sm">Policy, caveats, budgets, and revocation run in the request path before the agent reaches the upstream service.</p>
            </div>
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800 text-center">
              <div className="text-3xl mb-3">💰</div>
              <h4 className="font-semibold text-white mb-2">Evidence export</h4>
              <p className="text-gray-500 text-sm">Every mint, delegation, spend event, denial, and revocation can land in an Evidence Pack.</p>
            </div>
          </div>

          <div className="bg-gradient-to-r from-cyan-900/20 to-purple-900/20 border border-cyan-800/30 rounded-xl p-6 text-center">
            <p className="text-gray-300 text-sm">
              <span className="text-white font-semibold">&ldquo;Stop giving AI agents standing authority.&rdquo;</span>
              <br/>
              <span className="text-gray-500 text-xs mt-2 block">Give them scoped capabilities. Enforce policy before execution. Export proof after.</span>
            </p>
          </div>
        </div>
      </section>

      {/* Live Mint Demo */}
      <section className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-purple-950/10 to-black">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 bg-purple-500/10 border border-purple-500/20 rounded-full text-purple-300 text-sm mb-6">
              <Key size={14} />
              Live Demo
            </div>
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              See authority minted — then prove the lifecycle
            </h2>
            <p className="text-gray-400 text-lg max-w-2xl mx-auto">
              SatGate Mint exchanges workload identity tokens for capability-bearing macaroons.
              No secrets to manage. No tokens to rotate. Try it live.
            </p>
          </div>
          <div className="text-center">
            <Link
              href="/mint-demo"
              className="inline-flex items-center gap-2 px-8 py-4 bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 rounded-xl text-white font-semibold text-lg transition shadow-lg shadow-purple-500/20"
            >
              Launch Interactive Demo <ArrowRight size={20} />
            </Link>
            <p className="text-gray-600 text-sm mt-4">
              3-step flow: Mock IdP → Mint Exchange → Verified Macaroon
            </p>
          </div>
        </div>
      </section>

      {/* At Scale */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              From Demo to Thousands of Agents
            </h2>
            <p className="text-gray-400 text-lg max-w-2xl mx-auto">
              The same exchange you just saw works identically at scale. No secrets management. No credential rotation. Identity is the credential.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-12">
            <div className="bg-[#12121a] border border-gray-800 rounded-xl p-6">
              <div className="text-2xl font-bold text-purple-400 mb-2">1</div>
              <h3 className="text-white font-semibold mb-2">Define trust policies</h3>
              <p className="text-gray-400 text-sm">Map identity claims to budgets, scopes, and TTLs. One policy per agent class — not per agent instance.</p>
            </div>
            <div className="bg-[#12121a] border border-gray-800 rounded-xl p-6">
              <div className="text-2xl font-bold text-cyan-400 mb-2">2</div>
              <h3 className="text-white font-semibold mb-2">Agents self-provision</h3>
              <p className="text-gray-400 text-sm">Each agent reads its environment identity (K8s SA, IAM role, OIDC token) and exchanges it for a macaroon. One API call. No human in the loop.</p>
            </div>
            <div className="bg-[#12121a] border border-gray-800 rounded-xl p-6">
              <div className="text-2xl font-bold text-green-400 mb-2">3</div>
              <h3 className="text-white font-semibold mb-2">Export the evidence</h3>
              <p className="text-gray-400 text-sm">Every grant, spend event, denial, delegation, and revocation becomes part of the evidence trail when you need to prove what happened.</p>
            </div>
          </div>

          <div className="bg-[#0a0a0a] border border-gray-800 rounded-xl p-6 font-mono text-sm text-gray-400 overflow-x-auto">
            <pre>{`# Agent startup (3 lines — works in any runtime)
IDENTITY=$(cat /var/run/secrets/tokens/satgate-token)
TOKEN=$(curl -s -X POST $SATGATE_MINT_URL \\
  -d "{\\"credentials\\": \\"$IDENTITY\\"}" | jq -r '.token')
export SATGATE_TOKEN=$TOKEN
# That's it. Agent connects through SatGate with budget-scoped access.`}</pre>
          </div>

          <div className="text-center mt-8">
            <Link
              href="https://cloud.satgate.io/docs/guides/mint-identity"
              className="text-purple-400 hover:text-purple-300 transition text-sm underline underline-offset-2"
            >
              Read the full integration guide →
            </Link>
          </div>
        </div>
      </section>

      <section className="py-20 px-6 border-t border-gray-800 bg-black">
        <div className="max-w-5xl mx-auto rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-8 md:p-10">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Governance rollout kit</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Turn enterprise governance into request-path controls</h2>
          <p className="mb-8 max-w-3xl text-gray-400 leading-relaxed">
            High-level AI governance only matters when it becomes enforceable policy: identity, budgets, scoped credentials, MCP tool limits, audit, revocation, and evidence before agents act.
          </p>
          <div className="grid gap-4 md:grid-cols-2">
            {[
              ['/economic-firewall-readiness-grader', 'Economic firewall readiness grader', 'Score identity, budgets, MCP governance, revocation, delegation, audit, routing, and rail-aware readiness.'],
              ['/agent-api-key-risk-assessment', 'Agent API key risk assessment', 'Find static-key blast radius before autonomous agents inherit unlimited API access.'],
              ['/agent-spend-policy-template', 'Agent spend policy template', 'Generate YAML/JSON policy for per-agent budgets, MCP caps, delegation, revocation, and audit.'],
              ['/mcp-cost-control', 'MCP cost control', 'Treat MCP tool calls as billable economic events with per-tool prices, caps, and deny decisions.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-black/50 p-5 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="py-20 px-6 border-t border-gray-800 bg-black">
        <div className="max-w-4xl mx-auto rounded-2xl border border-gray-800 bg-gray-950 p-8 md:p-10">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">AI agent governance questions</h2>
          <div className="space-y-6">
            {[
              ['What is AI agent governance?', 'AI agent governance is the set of controls that determines which agents can call which APIs, tools, and models; how much they can spend; what authority they can delegate; and when access must be revoked. For autonomous agents, governance needs request-path enforcement, not just logs and dashboards.'],
              ['What is an economic control plane for AI agents?', 'An economic control plane for AI agents sits in the request path and applies scopes, budgets, delegation rules, revocation, and audit before an agent reaches an upstream API, model, or MCP tool. It turns agent activity into governed decisions with evidence.'],
              ['How should enterprises govern MCP tool usage?', 'Enterprises should govern MCP tools with per-tool budgets, scoped capability tokens, task and tenant attribution, audit trails, revocation, and hard request-path policy decisions. Rate limits and dashboards are useful, but they do not replace enforcement before tool calls execute.'],
              ['What is the difference between AI governance and AI agent governance?', 'AI governance usually covers model risk, data policy, compliance, and human review. AI agent governance adds request-path controls for autonomous actions: scopes, budgets, delegated authority, revocation, denial reasons, spend attribution, and proof before APIs or MCP tools execute.'],
            ].map(([question, answer]) => (
              <div key={question} className="border-t border-gray-800 pt-6 first:border-t-0 first:pt-0">
                <h3 className="mb-2 text-xl font-bold text-white">{question}</h3>
                <p className="leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Final CTA */}
      <section className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-purple-950/20 to-black">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="text-3xl md:text-4xl font-bold mb-6">
            Ready to govern your AI agents?
          </h2>
          <p className="text-gray-400 text-lg mb-8">
            We&apos;re working with enterprise teams that need scoped authority, runtime enforcement, and evidence before agents scale.
            Start with Observe, then move to Control and Policy-to-Proof when you&apos;re ready.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/design-partners" className="bg-gradient-to-r from-purple-600 to-cyan-600 text-white px-10 py-4 rounded-full font-bold text-lg hover:opacity-90 transition shadow-lg shadow-purple-500/20 flex items-center justify-center gap-2">
              Book a governance walkthrough <ArrowRight size={20} />
            </Link>
          </div>
          <p className="text-gray-600 text-sm mt-6">
            Observe first • Enforce next • Export proof when it matters
          </p>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 border-t border-gray-800">
        <div className="max-w-6xl mx-auto px-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8 py-8">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <Image src="/logo_white_transparent.png" alt="SatGate" width={24} height={24} className="w-6 h-6" />
                <h4 className="font-bold text-white">SatGate</h4>
              </div>
              <p className="text-gray-500 text-sm">Govern what agents can do — and prove it.</p>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Product</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/protect" className="hover:text-white transition">Protect</Link></li>
                <li><Link href="/pay" className="hover:text-white transition">Pay</Link></li>
                <li><Link href="/govern" className="hover:text-white transition">Govern</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Resources</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/economic-firewall" className="hover:text-white transition">Economic Firewall</Link></li>
                <li><Link href="/llm-cost-dashboard" className="hover:text-white transition">LLM Cost Dashboard</Link></li>
                <li><Link href="/llm-cost-monitoring" className="hover:text-white transition">LLM Cost Monitoring</Link></li>
                <li><Link href="/ai-agent-cost-control" className="hover:text-white transition">AI Agent Cost Control</Link></li>
                <li><Link href="/mcp-governance" className="hover:text-white transition">MCP Governance</Link></li>
                <li><Link href="/agent-api-governance" className="hover:text-white transition">Agent API Governance</Link></li>
                <li><Link href="/l402-agent-payments" className="hover:text-white transition">L402 Agent Payments</Link></li>
                <li><a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a></li>
                <li><a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud Dashboard</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Company</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="mailto:contact@satgate.io" className="hover:text-white transition">contact@satgate.io</a></li>
                <li><Link href="/terms" className="hover:text-white transition">Terms</Link></li>
                <li><Link href="/privacy" className="hover:text-white transition">Privacy</Link></li>
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
}

'use client';

import React, { useState } from 'react';
import {
  Eye, Sliders, CreditCard, Shield, AlertTriangle, Users, Building2,
  ArrowRight, CheckCircle, Clock, DollarSign, Lock, Key, Bot,
  GitBranch, Activity, Server, Cloud, Container, FileCode, Menu, X,
  ChevronRight, Zap, UserCheck, FileSearch, BarChart3, ShieldCheck,
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
            <Link href="/protect" className="hover:text-white transition">Protect Demo</Link>
            <Link href="/pay" className="hover:text-white transition">Pay Demo</Link>
            <Link href="/govern" className="text-white transition">Govern</Link>
            <Link href="#" className="hover:text-white transition">Pricing</Link>
            <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://cloud.satgate.io" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud</a>
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
              { href: '/protect', label: 'Protect Demo' },
              { href: '/pay', label: 'Pay Demo' },
              { href: '/govern', label: 'Govern' },
              { href: '#', label: 'Pricing' },
            ].map(item => (
              <Link key={item.href} href={item.href} onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">
                {item.label}
              </Link>
            ))}
            <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">
              Docs
            </a>
            <a href="https://cloud.satgate.io" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">
              Cloud
            </a>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <header className="pt-32 pb-20 px-6">
        <div className="max-w-4xl mx-auto text-center">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono mb-6">
            <Shield size={12} /> Enterprise AI Governance
          </div>
          <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
            See Everything.{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
              Control What Matters.
            </span>
          </h1>
          <p className="text-xl text-gray-400 mb-10 max-w-2xl mx-auto leading-relaxed">
            Your AI agents are spending money, calling APIs, and making decisions—with zero visibility.
            SatGate gives you the governance layer: <strong className="text-white">observe</strong> every call,{' '}
            <strong className="text-white">control</strong> every budget, and{' '}
            <strong className="text-white">charge back</strong> every dollar to the right team.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/design-partners" className="bg-white text-black px-8 py-3 rounded-lg font-bold hover:bg-gray-200 transition flex items-center justify-center gap-2">
              Become a Design Partner <ArrowRight size={18} />
            </Link>
            <Link href="#observe" className="border border-gray-700 px-8 py-3 rounded-lg font-bold hover:border-gray-500 transition flex items-center justify-center gap-2">
              Start Free <Eye size={18} />
            </Link>
          </div>
        </div>
      </header>

      {/* Three Personas */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">Built for Every Stakeholder</h2>
          <p className="text-gray-500 text-center mb-12">One platform, three perspectives. Everyone gets what they need.</p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* CFO */}
            <div className="p-6 rounded-xl bg-gradient-to-br from-green-950/30 to-green-900/10 border border-green-800/30 hover:border-green-600/50 transition group">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-green-900/50 rounded-lg group-hover:bg-green-900/70 transition">
                  <DollarSign className="text-green-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">CFO</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-4">
                "Where is the money going?"
              </p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><BarChart3 size={14} className="text-green-400 flex-shrink-0" /> Cost attribution by team & agent</li>
                <li className="flex items-center gap-2"><CreditCard size={14} className="text-green-400 flex-shrink-0" /> Automated chargeback reports</li>
                <li className="flex items-center gap-2"><Activity size={14} className="text-green-400 flex-shrink-0" /> Budget alerts before overruns</li>
              </ul>
            </div>

            {/* CISO */}
            <div className="p-6 rounded-xl bg-gradient-to-br from-red-950/30 to-red-900/10 border border-red-800/30 hover:border-red-600/50 transition group">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-red-900/50 rounded-lg group-hover:bg-red-900/70 transition">
                  <Shield className="text-red-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">CISO</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-4">
                "Are we secure and compliant?"
              </p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><ShieldCheck size={14} className="text-red-400 flex-shrink-0" /> Security dashboard & threat detection</li>
                <li className="flex items-center gap-2"><AlertTriangle size={14} className="text-red-400 flex-shrink-0" /> Anomalous usage alerts</li>
                <li className="flex items-center gap-2"><FileSearch size={14} className="text-red-400 flex-shrink-0" /> SOC2/GDPR compliance exports</li>
              </ul>
            </div>

            {/* CTO */}
            <div className="p-6 rounded-xl bg-gradient-to-br from-purple-950/30 to-purple-900/10 border border-purple-800/30 hover:border-purple-600/50 transition group">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-purple-900/50 rounded-lg group-hover:bg-purple-900/70 transition">
                  <Key className="text-purple-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">CTO</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-4">
                "Who has access, and how do I control it?"
              </p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><GitBranch size={14} className="text-purple-400 flex-shrink-0" /> Full audit trail of every token</li>
                <li className="flex items-center gap-2"><Lock size={14} className="text-purple-400 flex-shrink-0" /> Token delegation & revocation</li>
                <li className="flex items-center gap-2"><Sliders size={14} className="text-purple-400 flex-shrink-0" /> Governance policies as code</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* Observe → Control → Charge */}
      <section id="observe" className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-gray-900/30 to-black">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">Three Modes. One Gateway.</h2>
          <p className="text-gray-500 text-center mb-12">Start with visibility. Add control when you're ready. Monetize when it makes sense.</p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Observe */}
            <div className="relative p-6 rounded-xl bg-gray-900 border-2 border-purple-500/60 hover:border-purple-400/80 transition">
              <div className="absolute -top-3 left-4 px-3 py-0.5 bg-purple-600 text-white text-xs font-bold rounded-full">FREE</div>
              <div className="flex items-center gap-3 mb-4 mt-2">
                <div className="p-2.5 bg-purple-900/50 rounded-lg">
                  <Eye className="text-purple-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Observe</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-4">
                See every API call, every token, every agent — across MCP servers, REST APIs, and LLM endpoints. Full visibility with zero enforcement.
              </p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="text-purple-400">✓ Real-time request logging</li>
                <li className="text-purple-400">✓ Cost attribution dashboards</li>
                <li className="text-purple-400">✓ Agent inventory</li>
                <li className="text-purple-400">✓ Anomaly detection</li>
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
                Set budgets, enforce policies, gate access. Observe + enforcement.
              </p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="text-cyan-400">✓ Per-agent budget limits</li>
                <li className="text-cyan-400">✓ Token scope enforcement</li>
                <li className="text-cyan-400">✓ Rate limiting by team</li>
                <li className="text-cyan-400">✓ Automatic revocation</li>
              </ul>
            </div>

            {/* Charge */}
            <div className="p-6 rounded-xl bg-gray-900 border border-yellow-800/30 hover:border-yellow-600/50 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-yellow-900/50 rounded-lg">
                  <CreditCard className="text-yellow-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Charge</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed mb-4">
                Meter usage, generate invoices, bill internal teams or external consumers.
              </p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="text-yellow-400">✓ Per-request metering</li>
                <li className="text-yellow-400">✓ Internal chargeback automation</li>
                <li className="text-yellow-400">✓ External billing integration</li>
                <li className="text-yellow-400">✓ Lightning micropayments</li>
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
              <CreditCard size={14} className="text-yellow-400" />
              <span>Charge</span>
            </div>
          </div>
        </div>
      </section>

      {/* The Rogue Intern Story */}
      <section className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-red-950/10 to-black">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-red-900/30 border border-red-500/30 text-red-300 text-xs font-mono mb-6">
              <AlertTriangle size={12} /> Real-World Scenario
            </div>
            <h2 className="text-3xl font-bold mb-4">The Rogue Intern Story</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              Friday afternoon. An intern creates an API token "to test something." By Monday, $47,000 in OpenAI charges. Here's how SatGate changes the ending.
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
                      <Bot size={16} className="text-red-400" /> Intern Creates Token
                    </h4>
                    <p className="text-gray-500 text-sm">
                      "Just a quick test." Generates an API token with no budget limit, no scope restriction, no expiry.
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
                      Full timeline: who created the token, what it accessed, when it was revoked. Compliance-ready export.
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
              <p className="text-white font-semibold">$47,000 saved. 2 minutes to resolution. Zero code changes.</p>
              <p className="text-gray-500 text-sm mt-1">SatGate Observe mode would have caught this for free.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Dashboard Screenshots */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">60+ Dashboard Pages. Day One.</h2>
          <p className="text-gray-500 text-center mb-12">Real-time visibility into every agent, every API call, every dollar.</p>

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
                description: 'Automated cost attribution. See exactly which team spent what on which API. Export to CSV or push to your billing system.',
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
          <h2 className="text-3xl font-bold text-center mb-4">Hierarchical Token Delegation</h2>
          <p className="text-gray-500 text-center mb-12">
            Tokens flow down. Authority narrows. Every level is scoped, budgeted, and time-limited.
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
              { icon: <Container size={24} className="text-cyan-400" />, label: 'Docker', desc: 'docker pull satgate/satgate' },
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

      {/* Final CTA */}
      <section className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-purple-950/20 to-black">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="text-3xl md:text-4xl font-bold mb-6">
            Ready to govern your AI agents?
          </h2>
          <p className="text-gray-400 text-lg mb-8">
            We're working with 10 enterprises to build the governance layer for the agent economy.
            Start with free Observe mode—no risk, full visibility.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/design-partners" className="bg-gradient-to-r from-purple-600 to-cyan-600 text-white px-10 py-4 rounded-full font-bold text-lg hover:opacity-90 transition shadow-lg shadow-purple-500/20 flex items-center justify-center gap-2">
              Become a Design Partner <ArrowRight size={20} />
            </Link>
          </div>
          <p className="text-gray-600 text-sm mt-6">
            Free Observe mode • 5-minute setup • No credit card required
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
              <p className="text-gray-500 text-sm">The governance layer for AI agents.</p>
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
                <li><a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a></li>
                <li><a href="https://cloud.satgate.io" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud Dashboard</a></li>
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
            © 2025 SatGate Inc. All rights reserved. SatGate™ is a trademark of SatGate Inc. Patent Pending.
          </div>
        </div>
      </footer>
    </div>
  );
}

'use client';

import React from 'react';
import Link from 'next/link';
import Image from 'next/image';
import {
  ArrowRight, Shield, DollarSign, Activity, Eye,
  SlidersHorizontal, Zap, GitBranch, Bot, Code,
  Lock, CheckCircle, MonitorDot, Terminal, Plug,
  Fingerprint, Key, Ban
} from 'lucide-react';

export default function MCPLandingPage() {
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
            <Link href="/" className="hover:text-white transition">Home</Link>
            <Link href="/mcp" className="text-white">MCP</Link>
            <Link href="/govern" className="hover:text-white transition">Govern</Link>
            <Link href="/pricing" className="hover:text-white transition">Pricing</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud</a>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <section className="pt-32 pb-20 px-4">
        <div className="max-w-4xl mx-auto text-center">
          <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-purple-500/10 border border-purple-500/20 text-purple-300 text-sm mb-8">
            <Bot size={14} />
            MCP Agent Governance
          </div>

          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold leading-tight mb-6">
            Know what your AI agents <br className="hidden sm:block" />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 to-cyan-400">
              are spending
            </span>
          </h1>

          <p className="text-lg sm:text-xl text-gray-400 max-w-2xl mx-auto mb-10">
            Every tool call through Cursor, Claude Code, or any MCP client — authenticated, 
            logged, cost-tracked, and budget-enforced. Connect in 5 minutes.
          </p>

          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <a
              href="https://cloud.satgate.io/cloud/mcp/connect"
              className="inline-flex items-center justify-center gap-2 px-8 py-4 bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 rounded-xl text-white font-semibold text-lg transition shadow-lg shadow-purple-500/20"
            >
              Connect Your Agent <ArrowRight size={20} />
            </a>
            <a
              href="https://cloud.satgate.io/cloud/mcp"
              className="inline-flex items-center justify-center gap-2 px-8 py-4 bg-gray-800 hover:bg-gray-700 border border-gray-700 rounded-xl text-gray-200 font-semibold text-lg transition"
            >
              <MonitorDot size={20} />
              See Live Demo
            </a>
          </div>
        </div>
      </section>

      {/* Three Modes */}
      <section className="py-20 px-4 border-t border-gray-800/50">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center text-white mb-4">
            Start observing. Enforce when ready.
          </h2>
          <p className="text-gray-400 text-center max-w-2xl mx-auto mb-12">
            Every SatGate deployment starts in Observe mode. Upgrade to Control when you see the data.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Observe */}
            <div className="bg-gray-900/50 border border-gray-800 rounded-2xl p-8 relative">
              <div className="absolute top-4 right-4">
                <span className="px-2 py-1 text-xs bg-green-500/20 text-green-400 rounded-full">Free</span>
              </div>
              <Eye className="text-purple-400 mb-4" size={32} />
              <h3 className="text-xl font-semibold text-white mb-2">Observe</h3>
              <p className="text-gray-400 text-sm leading-relaxed">
                See every tool call, every agent session, every credit spent — without blocking anything. 
                Shadow reporting shows what enforcement <em>would</em> have caught.
              </p>
              <ul className="mt-4 space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-green-400 shrink-0" /> Real-time MCP Monitor</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-green-400 shrink-0" /> Per-tool cost tracking</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-green-400 shrink-0" /> Shadow Report analytics</li>
              </ul>
            </div>

            {/* Control */}
            <div className="bg-gray-900/50 border border-purple-500/30 rounded-2xl p-8 ring-1 ring-purple-500/10">
              <SlidersHorizontal className="text-cyan-400 mb-4" size={32} />
              <h3 className="text-xl font-semibold text-white mb-2">Control</h3>
              <p className="text-gray-400 text-sm leading-relaxed">
                Set per-agent budgets and per-tool cost limits. Get alerts when agents approach limits. 
                Block tool calls when budgets are exhausted — agents get a 402.
              </p>
              <ul className="mt-4 space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-cyan-400 shrink-0" /> Per-agent credit budgets</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-cyan-400 shrink-0" /> Budget exhaustion alerts</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-cyan-400 shrink-0" /> 402 enforcement on overspend</li>
              </ul>
            </div>

            {/* Charge */}
            <div className="bg-gray-900/50 border border-gray-800 rounded-2xl p-8">
              <Zap className="text-yellow-400 mb-4" size={32} />
              <h3 className="text-xl font-semibold text-white mb-2">Charge</h3>
              <p className="text-gray-400 text-sm leading-relaxed">
                Monetize your APIs for external consumers. Agents pay per request 
                via Lightning micropayments (L402). No API keys, no subscriptions — just pay and go.
              </p>
              <ul className="mt-4 space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-yellow-400 shrink-0" /> L402 Lightning payments</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-yellow-400 shrink-0" /> Per-request pricing</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-yellow-400 shrink-0" /> Agent-native monetization</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section className="py-20 px-4 border-t border-gray-800/50 bg-gray-900/20">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center text-white mb-12">
            Five minutes to full visibility
          </h2>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
            {[
              {
                step: '1',
                icon: <Plug className="text-purple-400" size={24} />,
                title: 'Sign up & get a token',
                desc: 'One-click signup. SatGate mints a macaroon token scoped to your tenant.',
              },
              {
                step: '2',
                icon: <Terminal className="text-cyan-400" size={24} />,
                title: 'Add to Cursor',
                desc: 'Paste the config into ~/.cursor/mcp.json. Restart. Tools appear automatically.',
              },
              {
                step: '3',
                icon: <Activity className="text-green-400" size={24} />,
                title: 'Use tools normally',
                desc: 'Every tool call routes through SatGate — zero latency impact, full visibility.',
              },
              {
                step: '4',
                icon: <MonitorDot className="text-yellow-400" size={24} />,
                title: 'Watch the dashboard',
                desc: 'Real-time Monitor shows sessions, tool calls, costs. Set budgets when ready.',
              },
            ].map((item) => (
              <div key={item.step} className="text-center">
                <div className="w-12 h-12 rounded-xl bg-gray-800 border border-gray-700 flex items-center justify-center mx-auto mb-4">
                  {item.icon}
                </div>
                <div className="text-xs text-gray-500 uppercase tracking-wider mb-2">Step {item.step}</div>
                <h3 className="text-white font-semibold mb-2">{item.title}</h3>
                <p className="text-gray-500 text-sm">{item.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Why SatGate for MCP */}
      <section className="py-20 px-4 border-t border-gray-800/50">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center text-white mb-4">
            Why agents need an Economic Firewall
          </h2>
          <p className="text-gray-400 text-center max-w-2xl mx-auto mb-12">
            AI agents don&apos;t have credit cards. They have tool access. Without governance, 
            every tool call is an unaudited transaction on your cloud bill.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {[
              {
                icon: <Shield className="text-purple-400" size={24} />,
                title: 'Macaroon Authentication',
                desc: 'Every agent gets a cryptographic bearer token with built-in scope, expiry, and budget caveats. Not an API key — a capability.',
              },
              {
                icon: <DollarSign className="text-yellow-400" size={24} />,
                title: 'Per-Tool Cost Attribution',
                desc: 'Know exactly which agent called which tool and what it cost. Shadow reports show where the money goes before you enforce.',
              },
              {
                icon: <GitBranch className="text-cyan-400" size={24} />,
                title: 'Delegation Hierarchies',
                desc: 'Department → team → agent. Each level gets a budget. Parent tokens delegate to children. Spending rolls up automatically.',
              },
              {
                icon: <Lock className="text-red-400" size={24} />,
                title: 'Zero Trust by Default',
                desc: 'Agents start with zero access. Every tool call is authenticated and authorized. No ambient authority, no overprivileged keys.',
              },
            ].map((item) => (
              <div key={item.title} className="bg-gray-900/50 border border-gray-800 rounded-xl p-6 flex gap-4">
                <div className="shrink-0 mt-1">{item.icon}</div>
                <div>
                  <h3 className="text-white font-semibold mb-1">{item.title}</h3>
                  <p className="text-gray-400 text-sm">{item.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* EZ Pass — Autonomous Agent Onboarding */}
      <section className="py-20 px-4 border-t border-gray-800/50 bg-gradient-to-b from-black to-gray-900/20">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono mb-4">
              <Zap size={12} /> EZ Pass for AI Agents
            </div>
            <h2 className="text-3xl font-bold text-white mb-4">
              Agents authenticate themselves. No humans required.
            </h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              Your agents present an identity credential, SatGate mints a budgeted token, 
              and they&apos;re through the gate. When the budget runs out or you revoke access — they stop. Instantly.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-5 gap-4 items-center">
            {[
              { icon: <Fingerprint className="text-cyan-400" size={20} />, label: 'Agent authenticates', desc: 'JWT from any OIDC provider' },
              { icon: <ArrowRight className="text-gray-600" size={16} />, label: '', desc: '' },
              { icon: <Key className="text-yellow-400" size={20} />, label: 'SatGate Mint', desc: 'Issues budgeted macaroon' },
              { icon: <ArrowRight className="text-gray-600" size={16} />, label: '', desc: '' },
              { icon: <Shield className="text-green-400" size={20} />, label: 'Agent calls APIs', desc: 'Budget-enforced, revocable' },
            ].map((step, i) => (
              step.label ? (
                <div key={i} className="bg-gray-900/50 border border-gray-800 rounded-xl p-4 text-center">
                  <div className="w-10 h-10 rounded-lg bg-gray-800 flex items-center justify-center mx-auto mb-2">
                    {step.icon}
                  </div>
                  <h4 className="text-white text-sm font-semibold mb-1">{step.label}</h4>
                  <p className="text-gray-500 text-xs">{step.desc}</p>
                </div>
              ) : (
                <div key={i} className="hidden md:flex justify-center">{step.icon}</div>
              )
            ))}
          </div>

          <div className="mt-10 grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="bg-red-950/20 border border-red-900/30 rounded-xl p-5">
              <div className="flex items-center gap-2 mb-2">
                <Ban className="text-red-400" size={16} />
                <h4 className="text-red-300 font-semibold text-sm">Instant Revocation</h4>
              </div>
              <p className="text-gray-500 text-xs">Kill any agent&apos;s access with one click. Token is rejected on the next request. No propagation delay.</p>
            </div>
            <div className="bg-amber-950/20 border border-amber-900/30 rounded-xl p-5">
              <div className="flex items-center gap-2 mb-2">
                <DollarSign className="text-amber-400" size={16} />
                <h4 className="text-amber-300 font-semibold text-sm">Budget Enforcement</h4>
              </div>
              <p className="text-gray-500 text-xs">Every token carries a spending cap. When credits run out, SatGate returns HTTP 402. No surprise bills.</p>
            </div>
            <div className="bg-cyan-950/20 border border-cyan-900/30 rounded-xl p-5">
              <div className="flex items-center gap-2 mb-2">
                <GitBranch className="text-cyan-400" size={16} />
                <h4 className="text-cyan-300 font-semibold text-sm">Full Visibility</h4>
              </div>
              <p className="text-gray-500 text-xs">Every agent token appears in the Delegation Tree. See who&apos;s spending what, in real-time. Audit-ready from day one.</p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-20 px-4 border-t border-gray-800/50 bg-gradient-to-b from-gray-900/20 to-black">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="text-3xl font-bold text-white mb-4">
            See what your agents are really doing
          </h2>
          <p className="text-gray-400 mb-8">
            Free Observe mode. No credit card. Connect your first agent in 5 minutes.
          </p>
          <a
            href="https://cloud.satgate.io/cloud/mcp/connect"
            className="inline-flex items-center gap-2 px-8 py-4 bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 rounded-xl text-white font-semibold text-lg transition shadow-lg shadow-purple-500/20"
          >
            Connect Your Agent — Free <ArrowRight size={20} />
          </a>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-8 px-4">
        <div className="max-w-5xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={20} height={20} />
            <span className="text-sm text-gray-500">SatGate™ — The Economic Firewall</span>
          </div>
          <div className="flex gap-6 text-sm text-gray-500">
            <Link href="/" className="hover:text-gray-300 transition">Home</Link>
            <Link href="/pricing" className="hover:text-gray-300 transition">Pricing</Link>
            <a href="https://github.com/SatGate-io/satgate" className="hover:text-gray-300 transition">GitHub</a>
            <Link href="/privacy" className="hover:text-gray-300 transition">Privacy</Link>
            <Link href="/terms" className="hover:text-gray-300 transition">Terms</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}

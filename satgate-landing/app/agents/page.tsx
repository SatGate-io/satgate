'use client';

import React from 'react';
import Link from 'next/link';
import Image from 'next/image';
import {
  ArrowRight, Shield, DollarSign, Activity, Eye,
  SlidersHorizontal, Zap, GitBranch, Bot, Code,
  Lock, CheckCircle, MonitorDot, Terminal, Plug,
  Fingerprint, Key, Ban, Globe, Server
} from 'lucide-react';

const webPageJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'WebPage',
  name: 'Economic Firewall for AI Agents',
  description: 'Protect HTTP APIs and MCP tools that AI agents call with request-path budget enforcement, revocation, audit, and delegated capability controls.',
  url: 'https://satgate.io/agents',
  dateModified: '2026-05-04',
  isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
  about: [
    { '@type': 'Thing', name: 'economic firewall for AI agents' },
    { '@type': 'Thing', name: 'HTTP API protection for agents' },
    { '@type': 'Thing', name: 'MCP tool governance' },
    { '@type': 'Thing', name: 'per-agent budget enforcement' },
    { '@type': 'Thing', name: 'delegation hierarchy controls' },
  ],
};

const faqJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'FAQPage',
  mainEntity: [
    {
      '@type': 'Question',
      name: 'What is an economic firewall for AI agents?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'An economic firewall for AI agents sits in the request path to observe, control, and audit every API or MCP tool call before autonomous agents create cost or access risk.',
      },
    },
    {
      '@type': 'Question',
      name: 'Can SatGate protect both HTTP APIs and MCP tools?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Yes. SatGate protects HTTP APIs and MCP tool servers with the same request-path policies for identity, budgets, revocation, audit, and tool-cost enforcement.',
      },
    },
    {
      '@type': 'Question',
      name: 'How does SatGate stop runaway agent spend?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'SatGate stops runaway agent spend by enforcing per-agent budgets, per-tool caps, request attribution, delegation limits, and revocable capabilities before the next upstream call executes.',
      },
    },
  ],
};

export default function AgentsLandingPage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans selection:bg-purple-500 selection:text-white">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      {/* Navigation */}
      <nav className="border-b border-gray-800 backdrop-blur-md fixed w-full z-50 bg-black/50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2 shrink-0">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="w-7 h-7 sm:w-8 sm:h-8" />
            <span className="text-lg sm:text-xl font-bold text-white whitespace-nowrap">SatGate<sup className="text-xs font-normal">™</sup></span>
          </Link>
          <div className="hidden md:flex gap-6 text-sm font-medium text-gray-400">
            <Link href="/mint-demo" className="hover:text-white transition">Mint Demo</Link>
            <Link href="/protect" className="hover:text-white transition">Control Demo</Link>
            <Link href="/pay" className="hover:text-white transition">Charge Demo</Link>
            <Link href="/govern" className="hover:text-white transition">Enterprise</Link>
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
            The Economic Firewall for AI Agents
          </div>

          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold leading-tight mb-6">
            Protect any API <br className="hidden sm:block" />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 to-cyan-400">
              your agents call
            </span>
          </h1>

          <p className="text-lg sm:text-xl text-gray-400 max-w-2xl mx-auto mb-10">
            HTTP APIs and MCP tool calls — authenticated, logged, cost-tracked, and budget-enforced. 
            Per-agent budgets. Delegation hierarchies. Next-request revocation. Connect in 5 minutes.
          </p>

          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <a
              href="https://cloud.satgate.io/cloud/login"
              className="inline-flex items-center justify-center gap-2 px-8 py-4 bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 rounded-xl text-white font-semibold text-lg transition shadow-lg shadow-purple-500/20"
            >
              Start Free <ArrowRight size={20} />
            </a>
            <Link
              href="/protect"
              className="inline-flex items-center justify-center gap-2 px-8 py-4 bg-gray-800 hover:bg-gray-700 border border-gray-700 rounded-xl text-gray-200 font-semibold text-lg transition"
            >
              <MonitorDot size={20} />
              See Live Demo
            </Link>
          </div>
        </div>
      </section>

      {/* Two Paths: HTTP + MCP */}
      <section className="py-20 px-4 border-t border-gray-800/50">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center text-white mb-4">
            One gateway. Every protocol.
          </h2>
          <p className="text-gray-400 text-center max-w-2xl mx-auto mb-12">
            SatGate protects HTTP REST APIs and MCP tool servers equally. Same policies, same dashboard, same enforcement.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
            {/* HTTP APIs */}
            <div className="bg-gray-900/50 border border-gray-800 rounded-2xl p-8">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-green-900/50 rounded-lg">
                  <Globe className="text-green-400" size={24} />
                </div>
                <div>
                  <h3 className="text-xl font-semibold text-white">HTTP APIs</h3>
                  <span className="text-xs text-gray-500">REST · GraphQL · Any HTTP endpoint</span>
                </div>
              </div>
              <p className="text-gray-400 text-sm mb-4 leading-relaxed">
                Your agents call OpenAI, Stripe, internal services — SatGate sits in front as a reverse proxy. 
                Every request is authenticated, metered, and budget-checked before reaching the upstream.
              </p>
              <div className="bg-black rounded-lg p-4 font-mono text-xs text-gray-400 mb-4">
                <div className="text-gray-600"># Route config</div>
                <div><span className="text-purple-400">path:</span> /openai/*</div>
                <div><span className="text-purple-400">upstream:</span> https://api.openai.com</div>
                <div><span className="text-purple-400">policy:</span> control</div>
                <div><span className="text-purple-400">cost_credits:</span> 10</div>
              </div>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-green-400 shrink-0" /> Reverse proxy for any HTTP API</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-green-400 shrink-0" /> Per-route cost attribution</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-green-400 shrink-0" /> API key injection (agent never sees upstream keys)</li>
              </ul>
            </div>

            {/* MCP Tools */}
            <div className="bg-gray-900/50 border border-gray-800 rounded-2xl p-8">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-purple-900/50 rounded-lg">
                  <Terminal className="text-purple-400" size={24} />
                </div>
                <div>
                  <h3 className="text-xl font-semibold text-white">MCP Tool Servers</h3>
                  <span className="text-xs text-gray-500">Cursor · Claude Code · Any MCP client</span>
                </div>
              </div>
              <p className="text-gray-400 text-sm mb-4 leading-relaxed">
                SatGate proxies MCP tool calls from AI agents to your tool servers. Every tool/call is intercepted, 
                cost-attributed, and governed — the agent sees standard MCP, your tools see standard MCP.
              </p>
              <div className="bg-black rounded-lg p-4 font-mono text-xs text-gray-400 mb-4">
                <div className="text-gray-600"># Cursor MCP config</div>
                <div>{`{`}</div>
                <div>&nbsp;&nbsp;<span className="text-purple-400">&quot;url&quot;</span>: <span className="text-green-400">&quot;https://satgate.cloud/sse&quot;</span>,</div>
                <div>&nbsp;&nbsp;<span className="text-purple-400">&quot;headers&quot;</span>: {`{ "Authorization": "Bearer <token>" }`}</div>
                <div>{`}`}</div>
              </div>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-purple-400 shrink-0" /> Per-tool cost profiles</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-purple-400 shrink-0" /> SSE + Streamable HTTP transports</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-purple-400 shrink-0" /> Real-time MCP Monitor dashboard</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* Three Modes */}
      <section className="py-20 px-4 border-t border-gray-800/50 bg-gray-900/20">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center text-white mb-4">
            Start observing. Enforce when ready.
          </h2>
          <p className="text-gray-400 text-center max-w-2xl mx-auto mb-12">
            Every SatGate deployment starts in Observe mode. See the data. Then enforce.
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
                See every API call, every tool invocation, every credit spent — without blocking anything.
                Shadow reporting shows what enforcement <em>would</em> have caught.
              </p>
              <ul className="mt-4 space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-green-400 shrink-0" /> Real-time monitoring</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-green-400 shrink-0" /> Cost attribution by agent</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-green-400 shrink-0" /> Shadow Report analytics</li>
              </ul>
            </div>

            {/* Control */}
            <div className="bg-gray-900/50 border border-purple-500/30 rounded-2xl p-8 ring-1 ring-purple-500/10">
              <SlidersHorizontal className="text-cyan-400 mb-4" size={32} />
              <h3 className="text-xl font-semibold text-white mb-2">Control</h3>
              <p className="text-gray-400 text-sm leading-relaxed">
                Set per-agent budgets. Get alerts at thresholds.
                Block requests when budgets are exhausted — agents get HTTP 402. No surprise bills.
              </p>
              <ul className="mt-4 space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-cyan-400 shrink-0" /> Per-agent credit budgets</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-cyan-400 shrink-0" /> Budget exhaustion alerts</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-cyan-400 shrink-0" /> Request-path budget enforcement</li>
              </ul>
            </div>

            {/* Charge */}
            <div className="bg-gray-900/50 border border-gray-800 rounded-2xl p-8">
              <Zap className="text-yellow-400 mb-4" size={32} />
              <h3 className="text-xl font-semibold text-white mb-2">Charge</h3>
              <p className="text-gray-400 text-sm leading-relaxed">
                Monetize your APIs. Agents consume approved access per request
                via paid-rail context (L402). No API keys, no subscriptions — just pay and go.
              </p>
              <ul className="mt-4 space-y-2 text-sm text-gray-500">
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-yellow-400 shrink-0" /> paid-rail context</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-yellow-400 shrink-0" /> Per-request pricing</li>
                <li className="flex items-center gap-2"><CheckCircle size={14} className="text-yellow-400 shrink-0" /> Agent-native monetization</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* EZ Pass — Autonomous Agent Onboarding */}
      <section className="py-20 px-4 border-t border-gray-800/50">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono mb-4">
              <Zap size={12} /> EZ Pass — Autonomous Agent Authentication
            </div>
            <h2 className="text-3xl font-bold text-white mb-4">
              Agents authenticate themselves. No humans required.
            </h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              Your agents present an identity credential, SatGate mints a budgeted macaroon, 
              and they&apos;re through the gate. When the budget runs out or you revoke access, the next governed request stops before upstream.
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
                <h4 className="text-red-300 font-semibold text-sm">Admin Kill Switch</h4>
              </div>
              <p className="text-gray-500 text-xs">Revoke any agent&apos;s token. The next governed request gets 401, including cascade to delegated child tokens.</p>
            </div>
            <div className="bg-amber-950/20 border border-amber-900/30 rounded-xl p-5">
              <div className="flex items-center gap-2 mb-2">
                <DollarSign className="text-amber-400" size={16} />
                <h4 className="text-amber-300 font-semibold text-sm">Economic Firewall</h4>
              </div>
              <p className="text-gray-500 text-xs">Every token carries a spending cap. When credits hit zero, SatGate returns HTTP 402. No surprise bills. No runaway agents.</p>
            </div>
            <div className="bg-cyan-950/20 border border-cyan-900/30 rounded-xl p-5">
              <div className="flex items-center gap-2 mb-2">
                <GitBranch className="text-cyan-400" size={16} />
                <h4 className="text-cyan-300 font-semibold text-sm">Delegation Hierarchies</h4>
              </div>
              <p className="text-gray-500 text-xs">Agents can delegate sub-tokens to other agents. Revoke the parent — the entire swarm stops. Full tree visibility in the dashboard.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Why SatGate */}
      <section className="py-20 px-4 border-t border-gray-800/50 bg-gray-900/20">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center text-white mb-4">
            Why agents need an Economic Firewall
          </h2>
          <p className="text-gray-400 text-center max-w-2xl mx-auto mb-12">
            AI agents don&apos;t have credit cards. They have API access. Without governance,
            every request is an unaudited transaction on your cloud bill.
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
                title: 'Per-Request Cost Attribution',
                desc: 'Know exactly which agent called which API or tool and what it cost. Shadow reports show where the money goes before you enforce.',
              },
              {
                icon: <GitBranch className="text-cyan-400" size={24} />,
                title: 'Delegation Hierarchies',
                desc: 'Department → team → agent. Each level gets a budget. Parent tokens delegate to children. Spending rolls up automatically.',
              },
              {
                icon: <Lock className="text-red-400" size={24} />,
                title: 'Zero Trust by Default',
                desc: 'Agents start with zero access. Every request is authenticated and authorized. No ambient authority, no overprivileged keys.',
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

      {/* FAQ */}
      <section className="py-20 px-4 border-t border-gray-800/50 bg-black">
        <div className="max-w-4xl mx-auto">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300 text-center">FAQ</p>
          <h2 className="text-3xl font-bold text-center text-white mb-10">AI agent economic firewall questions</h2>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              ['What is an economic firewall for AI agents?', 'An economic firewall for AI agents sits in the request path to observe, control, and audit every API or MCP tool call before autonomous agents create cost or access risk.'],
              ['Can SatGate protect both HTTP APIs and MCP tools?', 'Yes. SatGate protects HTTP APIs and MCP tool servers with the same request-path policies for identity, budgets, revocation, audit, and tool-cost enforcement.'],
              ['How does SatGate stop runaway agent spend?', 'SatGate enforces per-agent budgets, per-tool caps, request attribution, delegation limits, and revocable capabilities before the next upstream call executes.'],
            ].map(([question, answer]) => (
              <div key={question} className="rounded-2xl border border-gray-800 bg-gray-900/50 p-6">
                <h3 className="mb-3 text-lg font-bold text-white">{question}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-20 px-4 border-t border-gray-800/50 bg-gradient-to-b from-gray-900/20 to-black">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="text-3xl font-bold text-white mb-4">
            See what your agents are really spending
          </h2>
          <p className="text-gray-400 mb-8">
            Free Observe mode. No credit card. Works with any HTTP API or MCP server.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <a
              href="https://cloud.satgate.io/cloud/login"
              className="inline-flex items-center gap-2 px-8 py-4 bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 rounded-xl text-white font-semibold text-lg transition shadow-lg shadow-purple-500/20"
            >
              Start Free <ArrowRight size={20} />
            </a>
            <Link
              href="/protect"
              className="inline-flex items-center gap-2 px-8 py-4 bg-gray-800 hover:bg-gray-700 border border-gray-700 rounded-xl text-gray-200 font-semibold text-lg transition"
            >
              <Bot size={20} />
              Try Live Demo
            </Link>
          </div>
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

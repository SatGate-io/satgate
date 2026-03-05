import Link from 'next/link';
import { ArrowLeft, Calendar, Clock, Shield, DollarSign, ArrowRight, CheckCircle, Zap } from 'lucide-react';

export const metadata = {
  title: 'What Is an Economic Firewall? The Security Primitive for the Agent Economy',
  description: 'An economic firewall enforces budget limits on AI agent API calls at the gateway layer. Learn how it differs from traditional API security and why agents need it.',
  alternates: { canonical: 'https://satgate.io/blog/what-is-an-economic-firewall' },
  keywords: ['economic firewall', 'AI agent security', 'API budget enforcement', 'economic access control', 'agent economy', 'API gateway AI'],
};

export default function WhatIsEconomicFirewallPage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>

        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">Concepts</span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">Economic Firewall</span>
            <span className="px-2 py-1 rounded-full bg-green-900/30 border border-green-500/30 text-green-300 text-xs font-mono">Agent Economy</span>
          </div>
          <h1 className="text-4xl md:text-5xl font-extrabold tracking-tight mb-4">
            What Is an Economic Firewall?
          </h1>
          <p className="text-xl text-gray-400 mb-4">The security primitive for the agent economy</p>
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> March 5, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 6 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          <p className="text-xl text-gray-300 leading-relaxed">
            An <strong>economic firewall</strong> is an API gateway component that enforces financial constraints 
            on AI agent traffic. Where a traditional firewall asks &ldquo;is this request allowed?&rdquo; and a 
            WAF asks &ldquo;is this request safe?&rdquo;, an economic firewall asks: 
            <strong> &ldquo;should this agent spend this?&rdquo;</strong>
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Why We Need a New Primitive</h2>
          <p className="text-gray-300 leading-relaxed">
            The API security stack has evolved in layers:
          </p>
          
          <div className="space-y-3 my-8">
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex items-center gap-4">
              <div className="bg-gray-800 rounded-lg p-2.5 shrink-0">
                <Shield className="text-gray-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold">Network Firewall</p>
                <p className="text-gray-500 text-sm">&ldquo;Can this IP reach this port?&rdquo; — Layer 3/4 filtering</p>
              </div>
            </div>
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex items-center gap-4">
              <div className="bg-gray-800 rounded-lg p-2.5 shrink-0">
                <Shield className="text-blue-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold">Web Application Firewall (WAF)</p>
                <p className="text-gray-500 text-sm">&ldquo;Is this request malicious?&rdquo; — SQL injection, XSS, OWASP Top 10</p>
              </div>
            </div>
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex items-center gap-4">
              <div className="bg-gray-800 rounded-lg p-2.5 shrink-0">
                <Shield className="text-green-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold">API Gateway</p>
                <p className="text-gray-500 text-sm">&ldquo;Is this request authenticated and authorized?&rdquo; — Identity, RBAC, rate limiting</p>
              </div>
            </div>
            <div className="bg-gray-900 border border-purple-500/50 rounded-lg p-4 flex items-center gap-4">
              <div className="bg-purple-900/50 rounded-lg p-2.5 shrink-0">
                <DollarSign className="text-purple-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold">Economic Firewall</p>
                <p className="text-purple-300 text-sm font-medium">&ldquo;Should this agent spend this?&rdquo; — Budgets, attribution, economic governance</p>
              </div>
            </div>
          </div>

          <p className="text-gray-300 leading-relaxed">
            None of the existing layers answer the economic question. An authenticated agent with valid permissions 
            can still burn through $10,000 in API calls. Identity says who. Authorization says what. Economics says how much.
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">How It Works</h2>
          <p className="text-gray-300 leading-relaxed">
            An economic firewall intercepts API traffic at the gateway layer and enforces three types of constraints:
          </p>

          <div className="space-y-6 my-8">
            <div>
              <h3 className="text-lg font-bold text-cyan-400 mb-2">1. Budget Enforcement</h3>
              <p className="text-gray-300 leading-relaxed">
                Every agent has a spending cap, encoded in its credential. The gateway checks remaining budget 
                before forwarding the request. When the budget is exhausted, the agent gets an HTTP 402 — 
                Payment Required. Not a vague error. A specific, actionable signal: &ldquo;you&apos;re out of money.&rdquo;
              </p>
            </div>
            <div>
              <h3 className="text-lg font-bold text-purple-400 mb-2">2. Spend Attribution</h3>
              <p className="text-gray-300 leading-relaxed">
                Every request is tagged with the agent&apos;s identity, cost center, department, and delegation chain. 
                Finance teams get a clear view: which team&apos;s agents are spending what, on which APIs, at what cost.
              </p>
            </div>
            <div>
              <h3 className="text-lg font-bold text-yellow-400 mb-2">3. Delegation Hierarchies</h3>
              <p className="text-gray-300 leading-relaxed">
                A manager agent can delegate a subset of its budget to sub-agents using cryptographic capability 
                tokens (macaroons). Each delegation attenuates the original capability — you can only give away 
                less than you have, never more.
              </p>
            </div>
          </div>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Economic Access Control vs Identity-Based Security</h2>
          
          <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden my-8">
            <div className="grid grid-cols-3 text-sm">
              <div className="p-4 border-b border-r border-gray-800 font-semibold text-gray-400"></div>
              <div className="p-4 border-b border-r border-gray-800 font-semibold text-green-400">Identity-Based</div>
              <div className="p-4 border-b border-gray-800 font-semibold text-purple-400">Economic</div>
              
              <div className="p-4 border-b border-r border-gray-800 text-gray-400">Core question</div>
              <div className="p-4 border-b border-r border-gray-800 text-gray-300">&ldquo;Who are you?&rdquo;</div>
              <div className="p-4 border-b border-gray-800 text-gray-300">&ldquo;What can you afford?&rdquo;</div>
              
              <div className="p-4 border-b border-r border-gray-800 text-gray-400">Credential</div>
              <div className="p-4 border-b border-r border-gray-800 text-gray-300">OAuth token, API key</div>
              <div className="p-4 border-b border-gray-800 text-gray-300">Capability token (macaroon)</div>
              
              <div className="p-4 border-b border-r border-gray-800 text-gray-400">Enforcement</div>
              <div className="p-4 border-b border-r border-gray-800 text-gray-300">Allow/deny</div>
              <div className="p-4 border-b border-gray-800 text-gray-300">Allow/deny + budget check</div>
              
              <div className="p-4 border-b border-r border-gray-800 text-gray-400">Delegation</div>
              <div className="p-4 border-b border-r border-gray-800 text-gray-300">New credential per agent</div>
              <div className="p-4 border-b border-gray-800 text-gray-300">Attenuated from parent</div>
              
              <div className="p-4 border-r border-gray-800 text-gray-400">Attribution</div>
              <div className="p-4 border-r border-gray-800 text-gray-300">Per-user</div>
              <div className="p-4 border-gray-800 text-gray-300">Per-agent, per-tool, per-cost-center</div>
            </div>
          </div>

          <p className="text-gray-300 leading-relaxed">
            These aren&apos;t competing approaches — they&apos;re complementary layers. You still need identity. 
            But when agents autonomously make expensive API calls, identity alone isn&apos;t enough.
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Where It Fits in the Stack</h2>
          <p className="text-gray-300 leading-relaxed">
            An economic firewall deploys as a reverse proxy — identical to where you&apos;d put Kong, Envoy, 
            or any API gateway. It can run standalone, as a sidecar to your existing gateway (routing only 
            agent traffic through it), or as an MCP proxy for AI tool calls.
          </p>

          <pre className="bg-gray-900 border border-gray-800 rounded-xl p-6 text-sm font-mono text-gray-300 overflow-x-auto my-8">
{`CDN / WAF
  ↓
Existing API Gateway (optional)
  ↓
Economic Firewall  ← budget check here
  ↓
Your API / MCP Servers`}
          </pre>

          <p className="text-gray-300 leading-relaxed">
            Sub-millisecond overhead. No code changes to your API. Drop-in deployment.
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">The Agent Economy Needs This</h2>
          <p className="text-gray-300 leading-relaxed">
            As AI agents proliferate — autonomous coding agents, customer support bots, data pipelines, 
            multi-agent orchestrations — the economic governance gap will only widen. Every agent that makes 
            API calls is spending someone&apos;s money. The question is whether you&apos;re tracking it, 
            controlling it, or finding out about it on the monthly invoice.
          </p>
          <p className="text-gray-300 leading-relaxed mt-4">
            An economic firewall closes that gap at the infrastructure layer, the same way network firewalls 
            closed the connectivity gap and WAFs closed the application security gap.
          </p>

          <div className="bg-gradient-to-r from-purple-900/30 to-cyan-900/30 border border-purple-500/20 rounded-xl p-8 mt-12 text-center">
            <h3 className="text-xl font-bold text-white mb-3">SatGate is an open-source economic firewall</h3>
            <p className="text-gray-400 mb-6">Per-agent budgets, per-tool cost attribution, delegation hierarchies. Connect in ~5 minutes.</p>
            <div className="flex flex-wrap gap-4 justify-center">
              <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="bg-gradient-to-r from-purple-600 to-cyan-600 text-white px-6 py-3 rounded-lg font-bold hover:opacity-90 transition flex items-center gap-2">
                GitHub <ArrowRight size={16} />
              </a>
              <Link href="/sandbox" className="border border-purple-700/50 bg-purple-900/20 px-6 py-3 rounded-lg font-bold hover:bg-purple-900/40 transition text-purple-300">
                Try the Sandbox
              </Link>
            </div>
          </div>
        </article>

        <footer className="mt-16 pt-8 border-t border-gray-800">
          <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition">
            <ArrowLeft size={18} /> More articles
          </Link>
        </footer>
      </div>
    </div>
  );
}

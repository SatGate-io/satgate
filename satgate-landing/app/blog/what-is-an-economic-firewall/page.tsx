import Link from 'next/link';
import { ArrowLeft, Calendar, Clock, Shield, DollarSign, ArrowRight, AlertTriangle, Eye, Lock, Zap } from 'lucide-react';

export const metadata = {
  title: 'What Is an Economic Firewall? | SatGate',
  description: 'An economic firewall enforces real-time budget limits on AI agent API calls. Learn why rate limiting fails for autonomous agents, how macaroon delegation works, and why economic governance is the missing security primitive.',
  alternates: { canonical: 'https://satgate.io/blog/what-is-an-economic-firewall' },
  keywords: ['economic firewall', 'AI agent security', 'API budget enforcement', 'macaroon delegation', 'agent economy', 'MCP cost control', 'L402', 'API gateway AI'],
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
          <p className="text-xl text-gray-400 mb-4">The missing security primitive for autonomous AI agents</p>
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> March 18, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 8 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">

          {/* --- The $480 Story --- */}
          <p className="text-xl text-gray-300 leading-relaxed">
            A developer we work with left a Claude Code agent running overnight on a research task. Six hours later, it had made 3,200 web search calls. The invoice: <strong>$480</strong>.
          </p>
          <p className="text-gray-300 leading-relaxed">
            The agent wasn&rsquo;t malicious. It wasn&rsquo;t buggy. It was doing exactly what it was told &mdash; recursively searching, cross-referencing, and expanding its context. It just never had a reason to stop. Nothing in the stack told it that &ldquo;enough&rdquo; existed.
          </p>
          <p className="text-gray-300 leading-relaxed">
            With SatGate in front of those same APIs, the agent would have hit its budget cap and received an HTTP 402 &mdash; Payment Required. Hard stop. No soft alert buried in a dashboard. No email that arrives three hours after the damage is done. The request is blocked, and the agent gets a clear, actionable signal: <em>you&rsquo;re out of budget</em>.
          </p>
          <p className="text-gray-300 leading-relaxed">
            That&rsquo;s an economic firewall. And it&rsquo;s the security primitive that the entire API stack is missing.
          </p>

          {/* --- The Problem --- */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">The Problem: Authentication Without Economics</h2>
          <p className="text-gray-300 leading-relaxed">
            The current API security model was built for humans clicking buttons, not agents making thousands of autonomous decisions per hour. It answers one question well &mdash; <em>&ldquo;who are you?&rdquo;</em> &mdash; and ignores a second one entirely: <em>&ldquo;what can you afford?&rdquo;</em>
          </p>
          <p className="text-gray-300 leading-relaxed">
            Consider what you have today:
          </p>
          <ul className="text-gray-300 space-y-2">
            <li><strong>API keys</strong> are all-or-nothing. A valid key grants full access to every endpoint it&rsquo;s scoped to. There&rsquo;s no concept of &ldquo;this key has $50 left.&rdquo;</li>
            <li><strong>Rate limiting</strong> controls frequency, not cost. 1,000 requests per minute tells you nothing about spend. One request might cost $0.001 (a cache hit); another might cost $2.50 (a GPT-4 completion with a 32k context window). Rate limits treat them identically.</li>
            <li><strong>Usage dashboards</strong> are retrospective. They show you what already happened. By the time you see the spike, you&rsquo;ve already paid for it.</li>
          </ul>
          <p className="text-gray-300 leading-relaxed">
            This worked when the caller was a human developer running tests or a web app with predictable traffic patterns. It breaks catastrophically when the caller is an autonomous agent that can generate 10,000 API calls before anyone checks a dashboard.
          </p>

          {/* --- The Concept --- */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">From &ldquo;Who Are You?&rdquo; to &ldquo;What Can You Afford?&rdquo;</h2>
          <p className="text-gray-300 leading-relaxed">
            An economic firewall adds a financial dimension to API access control. It sits at the gateway layer &mdash; the same place you&rsquo;d deploy Kong, Envoy, or any reverse proxy &mdash; and enforces budget constraints on every request that passes through it.
          </p>

          <div className="space-y-3 my-8">
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex items-center gap-4">
              <div className="bg-gray-800 rounded-lg p-2.5 shrink-0">
                <Shield className="text-gray-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold">Network Firewall</p>
                <p className="text-gray-500 text-sm">&ldquo;Can this IP reach this port?&rdquo;</p>
              </div>
            </div>
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex items-center gap-4">
              <div className="bg-gray-800 rounded-lg p-2.5 shrink-0">
                <Shield className="text-blue-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold">WAF</p>
                <p className="text-gray-500 text-sm">&ldquo;Is this request malicious?&rdquo;</p>
              </div>
            </div>
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex items-center gap-4">
              <div className="bg-gray-800 rounded-lg p-2.5 shrink-0">
                <Shield className="text-green-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold">API Gateway</p>
                <p className="text-gray-500 text-sm">&ldquo;Is this caller authenticated and authorized?&rdquo;</p>
              </div>
            </div>
            <div className="bg-gray-900 border border-purple-500/50 rounded-lg p-4 flex items-center gap-4">
              <div className="bg-purple-900/50 rounded-lg p-2.5 shrink-0">
                <DollarSign className="text-purple-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold">Economic Firewall</p>
                <p className="text-purple-300 text-sm font-medium">&ldquo;Can this agent afford this call?&rdquo;</p>
              </div>
            </div>
          </div>

          <p className="text-gray-300 leading-relaxed">
            The key shift: the credential itself carries economic constraints. Not just &ldquo;you have access&rdquo; but &ldquo;you have $200 of access, and you&rsquo;ve used $147.30 so far.&rdquo; The enforcement happens at the gateway, in real time, before the request ever reaches your backend.
          </p>

          {/* --- Hard Caps --- */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Hard Caps, Not Soft Alerts</h2>
          <p className="text-gray-300 leading-relaxed">
            Most cost management tools send you an email when spending exceeds a threshold. That&rsquo;s a soft alert. Useful for humans who check email regularly. Useless for an agent loop burning $80/hour at 3am.
          </p>
          <p className="text-gray-300 leading-relaxed">
            An economic firewall enforces <strong>hard caps</strong>. When budget reaches zero, the next request gets a 402 response. The agent can handle that gracefully &mdash; finish its current task, report partial results, request a budget increase. But it cannot continue spending.
          </p>
          <p className="text-gray-300 leading-relaxed">
            This is the difference between a smoke alarm and a firewall. A smoke alarm tells you there&rsquo;s a problem. A firewall prevents the damage from spreading.
          </p>

          {/* --- Tool-Level Attribution --- */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Cost Attribution at the Tool Level</h2>
          <p className="text-gray-300 leading-relaxed">
            Endpoint-level cost tracking isn&rsquo;t granular enough for AI agents. When an agent calls an MCP server, a single &ldquo;endpoint&rdquo; might expose dozens of tools with wildly different costs: <code>web_search</code> at $0.015 per call, <code>code_execution</code> at $0.002, <code>dalle_generate</code> at $0.08.
          </p>
          <p className="text-gray-300 leading-relaxed">
            An economic firewall tracks costs at the <strong>tool level</strong>, not just the endpoint level. You can see that Agent-47 spent $12.30 on web searches, $0.80 on code execution, and $34.00 on image generation &mdash; all through the same MCP server. You can set per-tool budgets. You can block expensive tools while allowing cheap ones.
          </p>
          <p className="text-gray-300 leading-relaxed">
            This matters because cost attribution is the foundation of cost control. If you can&rsquo;t see where money is going at the tool level, you can&rsquo;t make informed decisions about where to set limits.
          </p>

          {/* --- Macaroon Delegation --- */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Cryptographic Delegation with Macaroons</h2>
          <p className="text-gray-300 leading-relaxed">
            Here&rsquo;s a scenario that breaks traditional access control: Agent A needs to delegate a subtask to Agent B. Agent A has $500 of budget and full read-write access. It wants to give Agent B $50 and read-only access.
          </p>
          <p className="text-gray-300 leading-relaxed">
            With API keys, you&rsquo;d need to provision a new key with the right scopes, register it in your identity provider, and manage its lifecycle. With an economic firewall using <strong>macaroon-based tokens</strong>, Agent A simply attenuates its own credential:
          </p>

          <pre className="bg-gray-900 border border-gray-800 rounded-xl p-6 text-sm font-mono text-gray-300 overflow-x-auto my-8">
{`// Agent A's token: $500 budget, read-write scope
const agentAToken = "macaroon:v1:abc...";

// Agent A attenuates for Agent B
const agentBToken = attenuate(agentAToken, {
  budget: 50,          // $50 cap (deducted from A's budget)
  scope: "read-only",  // Can't write
  tools: ["web_search", "summarize"],  // Only these tools
  expires: "2h"        // Auto-expires
});

// Agent B CANNOT:
// - Spend more than $50
// - Write anything
// - Call dalle_generate or code_execution
// - Escalate its own permissions`}
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The constraints are cryptographic, not policy-based. Agent B can&rsquo;t modify or forge the token to escalate its privileges. It can only further attenuate &mdash; passing an even more restricted token to Agent C. This creates a natural delegation hierarchy where capabilities only flow downward.
          </p>
          <p className="text-gray-300 leading-relaxed">
            No central policy server. No admin portal. No RBAC matrix to maintain. The token <em>is</em> the policy.
          </p>

          {/* --- Three-Layer Model --- */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">The Three-Layer Model: Observe, Control, Charge</h2>
          <p className="text-gray-300 leading-relaxed">
            An economic firewall operates across three functional layers:
          </p>

          <div className="space-y-4 my-8">
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5 flex items-start gap-4">
              <div className="bg-blue-900/40 rounded-lg p-2.5 shrink-0 mt-0.5">
                <Eye className="text-blue-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold text-lg">Observe</p>
                <p className="text-gray-400">Audit and log every agent API call with full cost attribution. Which agent, which tool, which cost center, how much. This is the foundation &mdash; you can&rsquo;t control what you can&rsquo;t see.</p>
              </div>
            </div>
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5 flex items-start gap-4">
              <div className="bg-purple-900/40 rounded-lg p-2.5 shrink-0 mt-0.5">
                <Lock className="text-purple-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold text-lg">Control</p>
                <p className="text-gray-400">Enforce budgets in real time. Hard caps, per-tool limits, delegation constraints. When the budget is spent, the request is blocked.</p>
              </div>
            </div>
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5 flex items-start gap-4">
              <div className="bg-yellow-900/40 rounded-lg p-2.5 shrink-0 mt-0.5">
                <Zap className="text-yellow-400" size={20} />
              </div>
              <div>
                <p className="text-white font-semibold text-lg">Charge</p>
                <p className="text-gray-400">Monetize API access via L402 payments. Agents pay per call using cryptographic payment proofs &mdash; no accounts, no invoices, no billing portals.</p>
              </div>
            </div>
          </div>

          <p className="text-gray-300 leading-relaxed">
            A critical distinction: <strong>Control and Charge are parallel use cases, not sequential stages.</strong> They both build on the Observe layer, but they serve different audiences:
          </p>
          <ul className="text-gray-300 space-y-2">
            <li><strong>Observe → Control</strong> is the enterprise path. You&rsquo;re running agents internally and need to govern their spending. Budget enforcement, cost attribution, delegation hierarchies.</li>
            <li><strong>Observe → Charge</strong> is the API monetization path. You&rsquo;re exposing APIs or MCP tools to external agents and want to get paid per call. L402 micropayments, usage-based billing, no signup required.</li>
          </ul>
          <p className="text-gray-300 leading-relaxed">
            Most organizations will start with Observe (because you need visibility before you can set sensible limits), then branch into Control, Charge, or both depending on whether they&rsquo;re consuming or selling API access.
          </p>

          {/* --- Why Now --- */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Why This Matters Now</h2>
          <p className="text-gray-300 leading-relaxed">
            The agent economy is scaling fast. Coding agents run overnight. Customer support agents handle thousands of conversations. Research agents crawl the web autonomously. Multi-agent orchestrations delegate tasks across dozens of sub-agents.
          </p>
          <p className="text-gray-300 leading-relaxed">
            Every one of these agents is spending someone&rsquo;s money on API calls. And the question isn&rsquo;t whether runaway spend will happen &mdash; it&rsquo;s whether you&rsquo;ll catch it in real time or on the monthly invoice.
          </p>
          <p className="text-gray-300 leading-relaxed">
            Economic governance is becoming a prerequisite for safe agentic AI deployment. Not a nice-to-have. Not a future concern. A prerequisite &mdash; the same way you wouldn&rsquo;t deploy a web application without authentication, or expose an API without rate limiting. The cost dimension is now a first-class security concern.
          </p>

          <div className="bg-gradient-to-r from-purple-900/30 to-cyan-900/30 border border-purple-500/20 rounded-xl p-8 mt-12 text-center">
            <h3 className="text-xl font-bold text-white mb-3">SatGate is an open-source economic firewall</h3>
            <p className="text-gray-400 mb-6">Per-agent budgets, per-tool cost attribution, macaroon delegation. Deploy in minutes.</p>
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

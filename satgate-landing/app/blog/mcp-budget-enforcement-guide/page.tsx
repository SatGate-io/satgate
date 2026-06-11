import Link from 'next/link';
import RoiCta from '../../components/RoiCta';
import { ArrowLeft, Calendar, Clock, ArrowRight, CheckCircle, Shield, DollarSign, Zap } from 'lucide-react';

export const metadata = {
  title: 'MCP Budget Enforcement Guide: Per-Tool Costs and Hard Agent Spend Caps',
  description: 'Practical MCP budget enforcement guide: set per-tool costs, cap agent spend, delegate budgets, and block runaway MCP tool calls before execution.',
  alternates: { canonical: 'https://satgate.io/blog/mcp-budget-enforcement-guide' },
  keywords: ['MCP budget enforcement', 'MCP proxy', 'Model Context Protocol budget', 'AI tool cost control', 'MCP gateway', 'per-tool cost attribution'],
  openGraph: {
    title: 'MCP Budget Enforcement: Per-Tool Costs and Hard Spend Caps',
    description: 'Set MCP per-tool costs, cap agent spend, delegate budgets, and block runaway tool calls before execution.',
    url: 'https://satgate.io/blog/mcp-budget-enforcement-guide',
    type: 'article',
    publishedTime: '2026-03-05T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Budget Enforcement: Per-Tool Costs and Hard Spend Caps',
    description: 'Practical MCP budget enforcement for per-tool pricing, delegated spend caps, and request-path blocks.',
  },
};

export default function McpBudgetEnforcementGuidePage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'MCP Budget Enforcement: Set Per-Tool Costs and Stop Runaway Agent Spend',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-03-05',
    dateModified: '2026-06-11',
    mainEntityOfPage: 'https://satgate.io/blog/mcp-budget-enforcement-guide',
    about: [
      { '@type': 'Thing', name: 'MCP budget enforcement' },
      { '@type': 'Thing', name: 'per-tool MCP costs' },
      { '@type': 'Thing', name: 'request-path budget checks' },
      { '@type': 'Thing', name: 'runaway agent spend control' },
      { '@type': 'Thing', name: 'MCP gateway economic governance' },
    ],
  };

  const faqSchema = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is MCP budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'MCP budget enforcement prices each MCP tool call, assigns an agent or workflow budget, checks remaining budget before the tool executes, and blocks or downgrades calls that would exceed the limit.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do you set per-tool MCP costs?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Set a default cost for ordinary MCP tools, then override expensive tools with explicit prices or wildcard rules. The gateway resolves the tool cost before each tools/call request and deducts it from the agent budget atomically.',
        },
      },
      {
        '@type': 'Question',
        name: 'Where should MCP budget checks happen?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Budget checks should happen in the request path, before the MCP server executes the tool. Post-hoc dashboards and alerts are useful for reporting, but they cannot stop runaway tool calls once an autonomous agent has already spent the money.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can MCP budget enforcement hard-cap specific expensive tools?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. MCP budget enforcement can assign higher prices, stricter daily caps, approval requirements, or deny rules to expensive tools such as code execution, web search, database queries, or deployment actions.',
        },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(faqSchema) }}
      />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>

        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">MCP</span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">Budget Enforcement</span>
            <span className="px-2 py-1 rounded-full bg-green-900/30 border border-green-500/30 text-green-300 text-xs font-mono">Tutorial</span>
          </div>
          <h1 className="text-4xl md:text-5xl font-extrabold tracking-tight mb-4">
            MCP Budget Enforcement: Set Per-Tool Costs and Stop Runaway Agent Spend
          </h1>
          <div className="mb-6 rounded-2xl border border-purple-900/60 bg-purple-950/20 p-5">
            <p className="mb-2 text-sm font-semibold uppercase tracking-[0.2em] text-purple-300">Practical answer</p>
            <p className="text-gray-300">MCP budget enforcement means pricing every tool call, giving each agent a spend allowance, checking budget before the MCP server executes, and returning a structured error when a tool would exceed the limit.</p>
          </div>
          <p className="text-xl text-gray-400 mb-4">Per-tool costs, spending caps, and real-time enforcement for MCP servers</p>
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> March 5, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 10 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          <p className="text-xl text-gray-300 leading-relaxed">
            MCP (Model Context Protocol) is becoming the standard way AI agents interact with tools. But MCP 
            has no built-in concept of cost. A <code className="bg-gray-800 px-1.5 rounded text-purple-300">tools/call</code> request 
            to a cheap lookup function and a $2 code execution tool look identical at the protocol level. 
            This guide shows how to add economic governance to any MCP server.
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">The Problem with Unmetered MCP</h2>
          <p className="text-gray-300 leading-relaxed">
            When you connect an AI agent to MCP servers, the agent can call any tool it has access to, as 
            many times as it wants. There&apos;s no protocol-level mechanism for:
          </p>
          <ul className="text-gray-300 space-y-2">
            <li>Assigning costs to different tool calls</li>
            <li>Tracking cumulative spend per agent</li>
            <li>Enforcing a budget ceiling</li>
            <li>Attributing costs to teams or departments</li>
            <li>Delegating budget from a parent agent to sub-agents</li>
          </ul>
          <p className="text-gray-300 leading-relaxed mt-4">
            Most MCP gateways solve the connectivity problem — routing, auth, discovery. But connectivity 
            without cost control is like an open highway with no toll booths. Traffic flows, but nobody&apos;s 
            tracking who owes what.
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Step 1: Define Per-Tool Costs</h2>
          <p className="text-gray-300 leading-relaxed">
            The first step is mapping tool calls to costs. SatGate&apos;s MCP proxy lets you define a cost 
            profile with per-tool pricing:
          </p>

          <pre className="bg-gray-900 border border-gray-800 rounded-xl p-6 text-sm font-mono text-gray-300 overflow-x-auto my-6">
{`# MCP Cost Profile
mcp:
  costProfile:
    defaultCostCredits: 1
    tools:
      - name: "web_search"
        costCredits: 2
      - name: "code_execute"
        costCredits: 10
      - name: "file_read"
        costCredits: 1
      - name: "database_query"
        costCredits: 5
      - name: "send_email"
        costCredits: 3
      - name: "*_premium"
        costCredits: 20  # wildcard matching`}
          </pre>

          <p className="text-gray-300 leading-relaxed">
            Wildcard matching (<code className="bg-gray-800 px-1.5 rounded text-purple-300">*_premium</code>) 
            lets you price tool families without listing every variant. Any tool not matched gets the 
            <code className="bg-gray-800 px-1.5 rounded text-purple-300">defaultCostCredits</code>.
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Step 2: Mint Tokens with Budgets</h2>
          <p className="text-gray-300 leading-relaxed">
            Agents authenticate with capability tokens (macaroons) that carry their budget as a cryptographic caveat:
          </p>

          <pre className="bg-gray-900 border border-gray-800 rounded-xl p-6 text-sm font-mono text-gray-300 overflow-x-auto my-6">
{`# Mint a token with 500-credit budget
satgate mint \\
  --agent "research-bot" \\
  --budget 500 \\
  --scope "/mcp/*" \\
  --expiry 24h \\
  --cost-center "engineering"

# Token carries:
#   budget_total_credits: 500
#   scope: /mcp/*
#   cost_center: engineering
#   expires: 2026-03-06T07:00:00Z`}
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The budget is embedded in the token itself. The gateway doesn&apos;t need a database lookup to 
            know the ceiling — it reads the caveat. Spend tracking uses a lightweight ledger (Redis-backed 
            by default).
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Step 3: Agent Sees Its Own Budget</h2>
          <p className="text-gray-300 leading-relaxed">
            SatGate injects economic metadata into the MCP <code className="bg-gray-800 px-1.5 rounded text-purple-300">tools/list</code> response, 
            so agents can make cost-aware decisions:
          </p>

          <pre className="bg-gray-900 border border-gray-800 rounded-xl p-6 text-sm font-mono text-gray-300 overflow-x-auto my-6">
{`// tools/list response (with SatGate metadata)
{
  "tools": [
    {
      "name": "web_search",
      "description": "Search the web [SatGate: 2 credits per call]",
      "x-satgate": {
        "cost_credits": 2,
        "budget_remaining": 487,
        "calls_affordable": 243
      }
    }
  ]
}`}
          </pre>

          <p className="text-gray-300 leading-relaxed">
            Smart agents can use this metadata to choose cheaper alternatives when budget is low, 
            or batch operations to minimize calls.
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Step 4: Enforce at the Gateway</h2>
          <p className="text-gray-300 leading-relaxed">
            When an agent makes a <code className="bg-gray-800 px-1.5 rounded text-purple-300">tools/call</code>, 
            the MCP proxy intercepts the JSON-RPC message, looks up the tool cost, checks the agent&apos;s 
            remaining budget, and either:
          </p>

          <div className="space-y-3 my-6">
            <div className="flex items-start gap-3">
              <CheckCircle className="text-green-400 mt-1 shrink-0" size={18} />
              <p className="text-gray-300"><strong className="text-white">Forwards the request</strong> if budget is sufficient (deducts the cost from the ledger)</p>
            </div>
            <div className="flex items-start gap-3">
              <Shield className="text-red-400 mt-1 shrink-0" size={18} />
              <p className="text-gray-300"><strong className="text-white">Returns a budget error</strong> if the agent has exceeded its cap — with remaining balance and cost in the error payload</p>
            </div>
          </div>

          <p className="text-gray-300 leading-relaxed">
            This happens at the proxy layer, before the request reaches your MCP server. Your tools 
            don&apos;t need any changes.
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Step 5: Delegation Trees</h2>
          <p className="text-gray-300 leading-relaxed">
            In multi-agent systems, a coordinator agent often delegates tasks to specialist agents. 
            With macaroon-based delegation, the coordinator can mint sub-tokens from its own token:
          </p>

          <pre className="bg-gray-900 border border-gray-800 rounded-xl p-6 text-sm font-mono text-gray-300 overflow-x-auto my-6">
{`# Coordinator has 500 credits
# Delegate 100 to research-agent
satgate delegate \\
  --from <coordinator-token> \\
  --to "research-agent" \\
  --budget 100 \\
  --scope "/mcp/search*"

# Delegate 200 to code-agent
satgate delegate \\
  --from <coordinator-token> \\
  --to "code-agent" \\
  --budget 200 \\
  --scope "/mcp/code*"

# Coordinator keeps 200 for itself
# Total across tree: still ≤ 500`}
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The parent&apos;s budget is the ceiling. Sub-agents can never collectively exceed what 
            was delegated. If you revoke the parent token, the entire tree is instantly invalidated.
          </p>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Step 6: Monitor and Attribute</h2>
          <p className="text-gray-300 leading-relaxed">
            Every tool call generates an event with full attribution:
          </p>
          <ul className="text-gray-300 space-y-2">
            <li><strong className="text-white">Agent identity:</strong> which agent made the call</li>
            <li><strong className="text-white">Tool name:</strong> which tool was invoked</li>
            <li><strong className="text-white">Cost:</strong> credits charged</li>
            <li><strong className="text-white">Budget remaining:</strong> after this call</li>
            <li><strong className="text-white">Cost center:</strong> department/team tag</li>
            <li><strong className="text-white">Delegation chain:</strong> full path from root to leaf agent</li>
          </ul>
          <p className="text-gray-300 leading-relaxed mt-4">
            The SatGate dashboard aggregates this into a Shadow Report — showing spend by agent, by tool, 
            by team, with trends over time. Start in Observe mode to see the data, then switch to Control 
            when you&apos;re ready to enforce.
          </p>

          <div className="my-8 rounded-2xl border border-purple-900/60 bg-purple-950/20 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">MCP Budget Enforcement FAQ</h2>
            <div className="space-y-5">
              <div>
                <h3 className="text-xl font-bold text-white mb-2">What is MCP budget enforcement?</h3>
                <p className="text-gray-300 leading-relaxed mb-0">
                  MCP budget enforcement prices each MCP tool call, assigns an agent or workflow budget, checks remaining budget before the tool executes, and blocks or downgrades calls that would exceed the limit.
                </p>
              </div>
              <div>
                <h3 className="text-xl font-bold text-white mb-2">How do you set per-tool MCP costs?</h3>
                <p className="text-gray-300 leading-relaxed mb-0">
                  Set a default cost for ordinary MCP tools, then override expensive tools with explicit prices or wildcard rules. The gateway resolves the tool cost before each tools/call request and deducts it from the agent budget atomically.
                </p>
              </div>
              <div>
                <h3 className="text-xl font-bold text-white mb-2">Where should MCP budget checks happen?</h3>
                <p className="text-gray-300 leading-relaxed mb-0">
                  Budget checks should happen in the request path, before the MCP server executes the tool. Dashboards and alerts are useful for reporting, but they cannot stop runaway tool calls after an autonomous agent has already spent the money.
                </p>
              </div>
              <div>
                <h3 className="text-xl font-bold text-white mb-2">Can MCP budget enforcement hard-cap specific expensive tools?</h3>
                <p className="text-gray-300 leading-relaxed mb-0">
                  Yes. MCP budget enforcement can assign higher prices, stricter daily caps, approval requirements, or deny rules to expensive tools such as code execution, web search, database queries, or deployment actions.
                </p>
              </div>
            </div>
          </div>

          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Quick Start</h2>
          <p className="text-gray-300 leading-relaxed">
            Get MCP budget enforcement running in 5 minutes:
          </p>

          <pre className="bg-gray-900 border border-gray-800 rounded-xl p-6 text-sm font-mono text-gray-300 overflow-x-auto my-6">
{`# 1. Install
go install github.com/SatGate-io/satgate@latest

# 2. Configure (mcp-proxy.yaml)
listen: :8080
admin:
  separateListener: :9090
routes:
  - path: /mcp/*
    upstream: http://your-mcp-server:3000
    policy:
      kind: control
      control:
        enforceBudget: true
        budgetMode: request_path
mcp:
  costProfile:
    defaultCostCredits: 1
    tools:
      - name: "expensive_tool"
        costCredits: 10

# 3. Run
satgate -config mcp-proxy.yaml

# 4. Mint a token
curl -X POST http://localhost:9090/admin/mint \\
  -d '{"agent":"my-bot","budget_total_credits":100}'

# 5. Connect your agent to localhost:8080/mcp`}
          </pre>

          <div className="bg-gradient-to-r from-purple-900/30 to-cyan-900/30 border border-purple-500/20 rounded-xl p-8 mt-12 text-center">
            <h3 className="text-xl font-bold text-white mb-3">Try it live</h3>
            <p className="text-gray-400 mb-6">See MCP budget enforcement in the interactive sandbox — no setup required.</p>
            <div className="flex flex-wrap gap-4 justify-center">
              <Link href="/sandbox" className="bg-gradient-to-r from-purple-600 to-cyan-600 text-white px-6 py-3 rounded-lg font-bold hover:opacity-90 transition flex items-center gap-2">
                Open Sandbox <ArrowRight size={16} />
              </Link>
              <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="border border-purple-700/50 bg-purple-900/20 px-6 py-3 rounded-lg font-bold hover:bg-purple-900/40 transition text-purple-300">
                View on GitHub
              </a>
            </div>
          </div>
          <div className="my-10 rounded-2xl border border-cyan-900/60 bg-cyan-950/20 p-6">
            <h3 className="mb-3 text-xl font-bold text-white">Generate an MCP budget policy</h3>
            <p className="mb-4 text-gray-300">Turn this guide into copyable policy: price MCP tools, set per-agent budgets, and generate proxy config for Cursor, Claude, OpenClaw, or custom clients.</p>
            <div className="flex flex-wrap gap-3 text-sm font-semibold">
              <Link href="/mcp-tool-cost-policy-generator" className="text-cyan-300 hover:text-cyan-200">MCP tool cost policy generator →</Link>
              <Link href="/mcp-proxy-config-generator" className="text-cyan-300 hover:text-cyan-200">MCP proxy config generator →</Link>
              <Link href="/mcp-cost-control" className="text-cyan-300 hover:text-cyan-200">MCP cost control →</Link>
              <Link href="/mcp-budget-enforcement" className="text-cyan-300 hover:text-cyan-200">MCP budget enforcement →</Link>
            </div>
          </div>
          <RoiCta
            title="Put MCP tool budgets into dollars"
            body="Use the calculator to translate per-tool calls, loop frequency, and agent volume into monthly savings."
          />

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

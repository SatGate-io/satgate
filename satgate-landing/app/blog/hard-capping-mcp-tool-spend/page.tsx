import Link from 'next/link';
import { ArrowLeft, Calendar, Clock, ExternalLink } from 'lucide-react';

export const metadata = {
  title: 'Hard-Cap MCP Tool Spend: Stop Runaway Claude Code and Cursor Agents',
  description: 'Hard-cap MCP tool spend for Claude Code, Cursor, and agent loops with request-path budget enforcement, per-tool costs, L402, and macaroons.',
  openGraph: {
    title: 'Hard-Cap MCP Tool Spend: Stop Runaway Claude Code and Cursor Agents',
    description: 'Hard-cap MCP tool spend for Claude Code, Cursor, and agent loops with request-path budget enforcement.',
    url: 'https://satgate.io/blog/hard-capping-mcp-tool-spend',
    type: 'article',
    authors: ['Matt Dean'],
    publishedTime: '2026-02-14T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Hard-Cap MCP Tool Spend: Stop Runaway Claude Code and Cursor Agents',
    description: 'Hard-cap MCP tool spend for Claude Code, Cursor, and agent loops with request-path budget enforcement.',
  },
  keywords: ['MCP tool cost', 'Claude Code spending limit', 'MCP budget control', 'AI agent cost management', 'MCP proxy', 'L402', 'macaroons'],
  alternates: { canonical: 'https://satgate.io/blog/hard-capping-mcp-tool-spend' },
};

const mcpHardCapFlow = [
  ['Price every MCP tool', 'Assign a cost to search, database, browser, code execution, and paid API tools before agents can call them.'],
  ['Mint a scoped budget', 'Give Claude Code, Cursor, or Claude Desktop a token with tool, route, spend, expiry, and delegation caveats.'],
  ['Check before forwarding', 'SatGate verifies the remaining budget before each tools/call request reaches the upstream MCP server.'],
  ['Return 402 on exhaustion', 'When the cap is exhausted, the proxy returns a clean budget-exceeded error instead of letting the loop keep spending.'],
];

export default function HardCappingMcpToolSpendPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Hard-Cap MCP Tool Spend: Stop Runaway Claude Code and Cursor Agents',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-02-14',
    dateModified: '2026-05-06',
    mainEntityOfPage: 'https://satgate.io/blog/hard-capping-mcp-tool-spend',
    about: [
      { '@type': 'Thing', name: 'hard-capping MCP tool spend' },
      { '@type': 'Thing', name: 'Claude Code spending limits' },
      { '@type': 'Thing', name: 'Cursor MCP budget control' },
      { '@type': 'Thing', name: 'request-path MCP proxy enforcement' },
      { '@type': 'Thing', name: 'L402 and macaroons for MCP tools' },
    ],
  };

  const mcpHardCapFlowJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'MCP tool spend hard-cap flow',
    description: 'How SatGate hard-caps MCP tool spend for Claude Code, Cursor, Claude Desktop, and other agent clients.',
    itemListElement: mcpHardCapFlow.map(([name, description], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name,
      description,
    })),
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'How do you hard-cap MCP tool spend?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Hard-cap MCP tool spend by putting an MCP proxy or economic firewall in the request path, assigning costs to tools, and blocking tools/call requests when the agent, workflow, or token budget is exhausted.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why are alerts not enough for MCP cost control?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Alerts fire after spend has already happened. Hard caps prevent the expensive MCP tool call from reaching the upstream server once the budget is exhausted.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can MCP budgets be scoped per tool or agent?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. MCP budgets can be scoped by agent, delegated sub-agent, tool, route, workflow, time window, and token caveat so each agent receives only the spend authority it needs.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do you stop Claude Code or Cursor from running up MCP tool bills?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Put a budget-aware MCP proxy between Claude Code, Cursor, or another MCP client and the upstream tool servers. The proxy prices each tool call, deducts from the agent budget, and blocks requests once the cap is exhausted.',
        },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(mcpHardCapFlowJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>
        
        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">MCP</span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">Budget Control</span>
            <span className="px-2 py-1 rounded-full bg-yellow-900/30 border border-yellow-500/30 text-yellow-300 text-xs font-mono">L402</span>
            <span className="px-2 py-1 rounded-full bg-green-900/30 border border-green-500/30 text-green-300 text-xs font-mono">Cost Management</span>
          </div>
          
          <h1 className="text-4xl font-bold mb-4">Hard-Capping MCP Tool Spend with SatGate Proxy</h1>
          
          <p className="text-xl text-gray-400 mb-6 italic">
            Your AI agent burned $500 overnight. Here&apos;s how to make sure it never happens again.
          </p>
          
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> February 14, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 10 min read</span>
            <span>Matt Dean</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          
          {/* Hook */}
          <p className="text-gray-300 text-lg leading-relaxed">
            Your AI agent just burned $500 overnight calling Google Search in a loop. You found out when the bill arrived. Sound familiar?
          </p>
          <p className="text-gray-300 leading-relaxed">
            If you&apos;re running Claude Code, Cursor, or Claude Desktop with MCP tools, you&apos;ve probably had a version of this moment. Maybe it was $50, maybe $5,000. The pattern is always the same: an agent gets stuck, loops on a tool call, and your API bill explodes while you sleep.
          </p>
          <p className="text-gray-300 leading-relaxed">
            There&apos;s a fix. And it doesn&apos;t involve monitoring dashboards, Slack alerts, or hoping you catch it in time.
          </p>

          {/* The Problem */}
          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Problem: MCP Has No Credit Card Limit</h2>
          
          <p className="text-gray-300 leading-relaxed">
            The <a href="https://spec.modelcontextprotocol.io" className="text-purple-400 hover:text-purple-300 underline" target="_blank" rel="noopener noreferrer">Model Context Protocol</a> is brilliant at what it does: giving AI agents structured access to tools. Search engines, databases, code execution, image generation — MCP makes it all available through a clean JSON-RPC interface.
          </p>
          <p className="text-gray-300 leading-relaxed">
            What MCP doesn&apos;t do is care about cost. Every <code className="text-purple-300">tools/call</code> request flows through to the upstream server with no budget awareness whatsoever. The spec has no concept of &quot;you&apos;ve spent too much&quot; or &quot;stop here.&quot;
          </p>
          
          <p className="text-gray-300 leading-relaxed font-semibold">This creates a specific, expensive failure mode:</p>
          
          <ul className="text-gray-300 space-y-2">
            <li><strong className="text-white">Agent loops</strong> — A stuck agent can make thousands of tool calls before anyone notices. Each call costs real money: API credits, compute, tokens.</li>
            <li><strong className="text-white">No built-in limits</strong> — The MCP spec includes no budget, quota, or cost mechanism. It&apos;s a capability protocol, not an economic one.</li>
            <li><strong className="text-white">Rate limits don&apos;t help</strong> — Rate limits protect <em>servers</em> from overload. They don&apos;t protect <em>your wallet</em> from a runaway agent staying within rate limits but burning money for hours.</li>
            <li><strong className="text-white">Manual monitoring is reactive</strong> — By the time you check a dashboard or get a Slack alert, the damage is done.</li>
          </ul>

          <div className="bg-red-900/20 border border-red-800/40 rounded-lg p-6 my-8">
            <p className="text-red-300 font-mono text-sm mb-0">
              💸 Real scenario: A developer left Claude Code running overnight with a web search MCP server. The agent hit a reasoning loop, called <code>brave_search</code> 3,200 times in 6 hours. Cost: $480 in API credits. The agent was technically working — just not productively.
            </p>
          </div>

          {/* The Solution */}
          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Solution: Economic Governance at the Protocol Level</h2>
          
          <p className="text-gray-300 leading-relaxed">
            SatGate MCP Proxy sits between your MCP client and your MCP servers. It intercepts every <code className="text-purple-300">tools/call</code>, tracks cost in real-time, and enforces hard budget caps — not soft alerts, not warnings, actual enforcement.
          </p>
          <p className="text-gray-300 leading-relaxed">
            When the budget is exhausted, the proxy returns <code className="text-purple-300">HTTP 402 Payment Required</code>. The agent receives a clean &quot;Budget exceeded&quot; message and stops. No infinite loops. No surprise bills.
          </p>
          <p className="text-gray-300 leading-relaxed">
            The enforcement mechanism uses <strong className="text-white">L402 macaroons</strong> — cryptographic tokens with embedded budget constraints. Unlike API keys (which grant unlimited access until revoked), a macaroon can encode: &quot;This agent can spend up to $5 on search tools, expiring in 1 hour.&quot; The constraints are baked into the token itself and verified on every call.
          </p>

          {/* Comparison Table */}
          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Governance Gap</h2>
          
          <p className="text-gray-300 leading-relaxed mb-6">
            Here&apos;s what changes when you add economic governance to MCP:
          </p>
          
          <div className="overflow-x-auto my-8">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-700">
                  <th className="text-left py-3 px-4 text-gray-400 font-mono">Feature</th>
                  <th className="text-left py-3 px-4 text-gray-400 font-mono">Standard MCP</th>
                  <th className="text-left py-3 px-4 text-purple-400 font-mono">SatGate-Enabled MCP</th>
                </tr>
              </thead>
              <tbody className="text-gray-300">
                <tr className="border-b border-gray-800">
                  <td className="py-3 px-4 font-medium text-white">Budget Enforcement</td>
                  <td className="py-3 px-4 text-red-400">Faith-based (wait for bill)</td>
                  <td className="py-3 px-4 text-green-400">Real-time hard caps</td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-3 px-4 font-medium text-white">Cost Attribution</td>
                  <td className="py-3 px-4 text-red-400">Aggregate (one big bill)</td>
                  <td className="py-3 px-4 text-green-400">Per-tool, per-agent</td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-3 px-4 font-medium text-white">Access Control</td>
                  <td className="py-3 px-4 text-red-400">Static API keys</td>
                  <td className="py-3 px-4 text-green-400">Attenuated macaroons</td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-3 px-4 font-medium text-white">Visibility</td>
                  <td className="py-3 px-4 text-red-400">Post-mortem logs</td>
                  <td className="py-3 px-4 text-green-400">Live economic telemetry</td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-3 px-4 font-medium text-white">Agent Loops</td>
                  <td className="py-3 px-4 text-red-400">Infinite spend potential</td>
                  <td className="py-3 px-4 text-green-400">Automated kill-switch</td>
                </tr>
              </tbody>
            </table>
          </div>

          {/* How It Works */}
          <h2 className="text-2xl font-bold text-white mt-12 mb-4">How It Works</h2>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Architecture</h3>
          <p className="text-gray-300 leading-relaxed">
            The proxy is transparent to both sides. Your MCP client thinks it&apos;s talking to the server. The server thinks it&apos;s talking to the client. SatGate sits in the middle, enforcing policy.
          </p>
          
          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-gray-300">{`┌─────────────┐     ┌──────────────────┐     ┌────────────────┐
│  Claude Code │────▶│  SatGate Proxy   │────▶│  MCP Server    │
│  Cursor      │◀────│                  │◀────│  (search, db…) │
│  Claude      │     │  ✓ Budget check  │     │                │
│  Desktop     │     │  ✓ Cost tracking │     │                │
│              │     │  ✓ 402 on limit  │     │                │
└─────────────┘     └──────────────────┘     └────────────────┘`}</code>
          </pre>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Configuration</h3>
          <p className="text-gray-300 leading-relaxed">
            Point your MCP client to SatGate instead of directly to the upstream server. Here&apos;s an example for Claude Desktop:
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`{
  "mcpServers": {
    "search": {
      "command": "satgate-proxy",
      "args": [
        "--upstream", "npx @anthropic/mcp-server-brave-search",
        "--budget", "500",
        "--budget-window", "1h",
        "--cost-per-call", "5"
      ]
    }
  }
}`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            That&apos;s it. Your agent now has a hard cap of 500 sats per hour on search calls. No code changes. No agent modifications.
          </p>

          <section className="not-prose my-10 rounded-2xl border border-purple-900/40 bg-purple-950/10 p-6">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">Hard-cap flow</p>
            <h3 className="mb-6 text-2xl font-bold text-white">From MCP access to enforceable spend control</h3>
            <div className="grid gap-4 md:grid-cols-2">
              {mcpHardCapFlow.map(([title, body], index) => (
                <div key={title} className="rounded-xl border border-gray-800 bg-black/60 p-4">
                  <p className="mb-2 text-xs font-mono text-purple-300">0{index + 1}</p>
                  <h4 className="mb-2 font-bold text-white">{title}</h4>
                  <p className="text-sm leading-relaxed text-gray-400">{body}</p>
                </div>
              ))}
            </div>
            <div className="mt-6 flex flex-col gap-3 sm:flex-row">
              <Link href="/mcp-budget-enforcement" className="inline-flex items-center justify-center rounded-lg border border-gray-700 px-4 py-2 text-sm font-semibold text-gray-300 transition hover:border-purple-500 hover:text-white">
                MCP budget enforcement
              </Link>
              <Link href="/mcp-tool-cost-policy-generator" className="inline-flex items-center justify-center rounded-lg border border-gray-700 px-4 py-2 text-sm font-semibold text-gray-300 transition hover:border-cyan-500 hover:text-white">
                Generate tool cost policy
              </Link>
            </div>
          </section>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">What Happens on Each Tool Call</h3>
          <ol className="text-gray-300 space-y-2">
            <li><strong className="text-white">Intercept</strong> — Proxy receives the <code className="text-purple-300">tools/call</code> JSON-RPC request</li>
            <li><strong className="text-white">Resolve cost</strong> — Looks up the tool name in the cost table (exact match or wildcard)</li>
            <li><strong className="text-white">Check budget</strong> — Compares accumulated spend against the macaroon&apos;s budget caveat</li>
            <li><strong className="text-white">Forward or reject</strong> — If within budget: forward to upstream, debit cost. If over budget: return 402 with a clear error message</li>
          </ol>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-red-300">{`// What the agent sees when budget is exhausted:
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "Budget exceeded: 500/500 sats used. Reset in 23m."
  }
}`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The agent receives a clean error, stops calling the tool, and continues with other work. No crash, no hang — just a boundary.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Macaroon Delegation</h3>
          <p className="text-gray-300 leading-relaxed">
            This is where it gets powerful. L402 macaroons support <em>attenuation</em> — you can take a token and add restrictions to it, but never remove them. This lets you create delegation chains:
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-cyan-300">{`# Create a root macaroon with $10 budget
satgate token create --budget 1000 --tools "web_search,database_query"

# Delegate to an agent: $5 budget, expires in 1 hour
satgate token attenuate <root-token> \\
  --max-budget 500 \\
  --expires 1h \\
  --tools "web_search"

# Result: agent can spend up to $5 on web_search only, for 1 hour
# No way to escalate beyond these constraints`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            Give your coding agent a tight budget for search. Give your research agent more for database access. Each gets exactly the permissions and budget they need — cryptographically enforced.
          </p>

          {/* Getting Started */}
          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Getting Started</h2>
          
          <div className="grid gap-4 my-8">
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="block bg-gray-900 border border-gray-800 rounded-lg p-6 hover:border-purple-600/50 transition no-underline">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-bold text-white mb-1">☁️ SatGate Cloud</h3>
                  <p className="text-gray-400 text-sm mb-0">Managed proxy — deploy in minutes, no infrastructure to manage.</p>
                </div>
                <ExternalLink size={18} className="text-gray-500" />
              </div>
            </a>
            
            <a href="https://github.com/nicethings/satgate" target="_blank" rel="noopener noreferrer" className="block bg-gray-900 border border-gray-800 rounded-lg p-6 hover:border-purple-600/50 transition no-underline">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-bold text-white mb-1">🔧 Self-Hosted</h3>
                  <p className="text-gray-400 text-sm mb-0">Open-source proxy — run it on your own infrastructure.</p>
                </div>
                <ExternalLink size={18} className="text-gray-500" />
              </div>
            </a>
            
            <Link href="/roi-calculator" className="block bg-gradient-to-r from-purple-900/30 to-cyan-900/30 border border-purple-800/40 rounded-lg p-6 hover:border-purple-600/50 transition no-underline">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-bold text-white mb-1">📊 ROI Calculator</h3>
                  <p className="text-gray-400 text-sm mb-0">Calculate how much you&apos;re losing to uncontrolled agent spend.</p>
                </div>
                <ExternalLink size={18} className="text-gray-500" />
              </div>
            </Link>
          </div>

          <section className="not-prose mt-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">FAQ</p>
            <h2 className="mb-6 text-2xl font-bold text-white">MCP hard-cap questions</h2>
            <div className="space-y-5">
              {[
                ['How do you hard-cap MCP tool spend?', 'Put an MCP proxy or economic firewall in the request path, assign costs to tools, and block tools/call requests when the agent, workflow, or token budget is exhausted.'],
                ['Why are alerts not enough for MCP cost control?', 'Alerts fire after spend has already happened. Hard caps prevent the expensive MCP tool call from reaching the upstream server once the budget is exhausted.'],
                ['Can MCP budgets be scoped per tool or agent?', 'Yes. MCP budgets can be scoped by agent, delegated sub-agent, tool, route, workflow, time window, and token caveat so each agent receives only the spend authority it needs.'],
                ['How do you stop Claude Code or Cursor from running up MCP tool bills?', 'Put a budget-aware MCP proxy between Claude Code, Cursor, or another MCP client and the upstream tool servers. The proxy prices each tool call, deducts from the agent budget, and blocks requests once the cap is exhausted.'],
              ].map(([question, answer]) => (
                <div key={question} className="border-t border-gray-800 pt-5 first:border-t-0 first:pt-0">
                  <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                  <p className="leading-relaxed text-gray-400">{answer}</p>
                </div>
              ))}
            </div>
          </section>

          {/* Close */}
          <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-8 my-12">
            <p className="text-gray-300 text-lg leading-relaxed mb-0">
              MCP is powerful. But power without governance is just risk. SatGate adds the economic guardrails that let you deploy agents with confidence — and a budget.
            </p>
          </div>

        </article>

        {/* Share */}
        <div className="border-t border-gray-800 pt-8 mt-12">
          <p className="text-gray-400 text-sm mb-4">Share this post:</p>
          <div className="flex gap-4">
            <a 
              href="https://twitter.com/intent/tweet?text=Hard-Capping%20MCP%20Tool%20Spend%20with%20SatGate%20Proxy&url=https://satgate.io/blog/hard-capping-mcp-tool-spend"
              target="_blank"
              rel="noopener noreferrer"
              className="px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-300 hover:border-purple-600/50 hover:text-white transition"
            >
              Share on Twitter / X
            </a>
            <a 
              href="https://www.linkedin.com/sharing/share-offsite/?url=https://satgate.io/blog/hard-capping-mcp-tool-spend"
              target="_blank"
              rel="noopener noreferrer"
              className="px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-300 hover:border-purple-600/50 hover:text-white transition"
            >
              Share on LinkedIn
            </a>
          </div>
        </div>

        {/* CTA */}
        <div className="mt-12 flex flex-wrap gap-4">
          <Link href="/llm-cost-dashboard" className="inline-flex items-center gap-2 bg-white text-black px-6 py-3 rounded-lg font-bold hover:bg-gray-200 transition">
            See Cost Telemetry
          </Link>
          <Link href="/roi-calculator" className="inline-flex items-center gap-2 bg-gray-900 border border-gray-700 text-white px-6 py-3 rounded-lg font-bold hover:border-purple-600/50 transition">
            Calculate Your ROI
          </Link>
          <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center gap-2 bg-gray-900 border border-gray-700 text-white px-6 py-3 rounded-lg font-bold hover:border-cyan-600/50 transition">
            Grade Readiness
          </Link>
        </div>
      </div>
    </div>
  );
}

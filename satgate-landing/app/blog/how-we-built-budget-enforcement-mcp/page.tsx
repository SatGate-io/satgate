import Link from 'next/link';
import { ArrowLeft, Calendar, Clock } from 'lucide-react';

export const metadata = {
  title: 'How We Built Budget Enforcement for MCP Tool Calls - SatGate Blog',
  description: 'We shipped an open-source MCP proxy that enforces per-tool budgets with cryptographic delegation. Here\'s how we built it and what we learned.',
  alternates: { canonical: 'https://satgate.io/blog/how-we-built-budget-enforcement-mcp' },
  openGraph: {
    title: 'How We Built Budget Enforcement for MCP Tool Calls',
    description: 'Inside SatGate’s MCP proxy architecture for per-tool budgets, cryptographic delegation, and request-path enforcement.',
    url: 'https://satgate.io/blog/how-we-built-budget-enforcement-mcp',
    type: 'article',
    publishedTime: '2026-02-13T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'How We Built Budget Enforcement for MCP Tool Calls',
    description: 'A technical look at MCP budget checks, macaroon delegation, structured errors, and per-tool cost attribution.',
  },
};

export default function McpProxyBlogPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'How We Built Budget Enforcement for MCP Tool Calls',
    description: 'We shipped an open-source MCP proxy that enforces per-tool budgets with cryptographic delegation for AI agent tool calls.',
    url: 'https://satgate.io/blog/how-we-built-budget-enforcement-mcp',
    datePublished: '2026-02-13',
    dateModified: '2026-05-04',
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'MCP budget enforcement' },
      { '@type': 'Thing', name: 'MCP proxy architecture' },
      { '@type': 'Thing', name: 'per-tool cost attribution' },
      { '@type': 'Thing', name: 'macaroon delegation for AI agents' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'How does MCP budget enforcement work?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'MCP budget enforcement works by intercepting tools/call messages, resolving the tool cost, checking the agent or token budget, and only forwarding the request to the upstream MCP server when budget remains.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why use macaroons for MCP delegation?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Macaroons support attenuated delegation: each child token can add stricter caveats for budget, scope, expiry, and tools, but cannot remove parent constraints or expand authority.',
        },
      },
      {
        '@type': 'Question',
        name: 'What happens when an MCP agent exhausts its budget?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'The proxy returns a structured budget_exhausted error instead of forwarding the tool call, giving the agent a clear failure it can handle without creating an infinite retry or surprise bill.',
        },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>
        
        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">MCP</span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">Go</span>
            <span className="px-2 py-1 rounded-full bg-yellow-900/30 border border-yellow-500/30 text-yellow-300 text-xs font-mono">Macaroons</span>
            <span className="px-2 py-1 rounded-full bg-green-900/30 border border-green-500/30 text-green-300 text-xs font-mono">Engineering</span>
          </div>
          
          <h1 className="text-4xl font-bold mb-4">How We Built Budget Enforcement for MCP Tool Calls</h1>
          
          <p className="text-xl text-gray-400 mb-6 italic">
            2,164 lines of Go, 28 tests, and one evening. Here&apos;s the architecture.
          </p>
          
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> February 13, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 8 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          
          <p className="text-gray-300 text-lg leading-relaxed">
            MCP gives AI agents structured access to tools. What it doesn&apos;t give them is a credit card limit.
          </p>
          <p className="text-gray-300 leading-relaxed">
            We shipped an open-source MCP proxy that intercepts <code>tools/call</code> JSON-RPC messages and enforces per-tool budgets with cryptographic delegation. Here&apos;s how we built it.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Architecture</h2>
          <p className="text-gray-300 leading-relaxed">
            The proxy sits between any MCP client and any MCP server. It speaks standard MCP protocol — agents don&apos;t know it&apos;s there.
          </p>
          
          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-gray-300">{`Agent → [stdio/SSE] → SatGate MCP Proxy → [stdio] → Upstream MCP Server
                            │
                     Budget Enforcement
                     Per-tool Cost Attribution
                     Token Delegation`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">Two transport modes:</p>
          <ul className="text-gray-300 space-y-2">
            <li><strong className="text-white">stdio</strong> — Local sidecar. One agent, one process. Zero network overhead.</li>
            <li><strong className="text-white">SSE/HTTP</strong> — Remote server. Multiple agents connect over HTTP, each with an independent SSE event stream.</li>
          </ul>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Per-Tool Cost Resolution</h2>
          <p className="text-gray-300 leading-relaxed">
            Not all tool calls cost the same. A <code>web_search</code> is cheap. A <code>dalle_generate</code> is expensive. Our cost resolver supports exact match and wildcard prefixes:
          </p>
          
          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`tools:
  defaultCost: 5
  costs:
    web_search: 5
    database_query: 5
    gpt4_summarize: 25
    gpt4_*: 25        # wildcard: gpt4_analyze, gpt4_translate...
    dalle_generate: 50
    code_execute: 15`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            Resolution order: exact match → longest wildcard prefix → catch-all <code>*</code> → default. Same pattern as the enterprise cost attribution engine, but running locally.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The BudgetEnforcer Interface</h2>
          <p className="text-gray-300 leading-relaxed">
            This is the split point between OSS and Enterprise:
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-blue-300">{`type BudgetEnforcer interface {
    Check(ctx, tokenID string, cost int64) (*BudgetResult, error)
    Spend(ctx, tokenID, toolName string, cost int64, requestID string) (*BudgetResult, error)
    Remaining(ctx, tokenID string) (int64, error)
    Initialize(ctx, tokenID string, credits int64) error
}`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            OSS provides <code>InMemoryBudgetEnforcer</code> — a mutex-protected map. Simple, fast, not durable across restarts. Enterprise provides <code>RedisBudgetEnforcer</code> — atomic Lua scripts, idempotent spend tracking, Postgres audit trail.
          </p>

          <p className="text-gray-300 leading-relaxed">When budget hits zero:</p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-red-300">{`{"jsonrpc":"2.0","id":42,"error":{
  "code":-32000,
  "message":"Budget exhausted",
  "data":{
    "error":"budget_exhausted",
    "tool":"dalle_generate",
    "cost_credits":50,
    "remaining_credits":0
  }
}}`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The agent gets a structured error it can handle gracefully — not a crashed process or an infinite retry.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Delegation: The Hard Part</h2>
          <p className="text-gray-300 leading-relaxed">
            When an orchestrator agent spawns sub-agents, each needs its own budget. The parent carves credits from its own allocation:
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-cyan-300">{`Orchestrator (1000 credits)
├── satgate/delegate(300, "research-agent")  → child token
├── satgate/delegate(200, "content-agent")   → child token
└── 500 credits remaining

Result:
  research-agent: 60 calls → 402 EXHAUSTED
  content-agent:  still operational ✓
  orchestrator:   500 credits remaining ✓`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The key design decision: <strong className="text-white">token identity = hash(identifier + signature)</strong>. Macaroon delegation produces child tokens with the same identifier as the parent (that&apos;s how HMAC chaining works). But each delegation adds caveats that change the signature, making the hash unique. This gives us a stable budget key per token without requiring a separate identity system.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">What We Learned</h2>
          
          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Macaroons are underrated for agent auth.</strong> HMAC chain means verification is constant-time with no database lookup. Delegation is just appending caveats. Permissions can only narrow, never widen. Perfect for agent-to-agent delegation.
          </p>
          
          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">stdio transport is simpler than you&apos;d think.</strong> Newline-delimited JSON over stdin/stdout. No HTTP overhead. The upstream manager spawns the MCP server as a subprocess and correlates request/response IDs via a <code>sync.Map</code> of pending channels.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">SSE needs keepalive.</strong> Connections through load balancers drop after 30-60s of silence. A periodic SSE comment line prevents this. Also: make your message handler async — if tool calls block the HTTP response goroutine, you get head-of-line blocking across sessions.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Fail-mode matters.</strong> When the budget backend is unreachable, deny all (closed) or allow and log (open)? We default to closed — secure first. But making it configurable was worth it.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">By the Numbers</h2>
          <ul className="text-gray-300 space-y-2">
            <li><strong className="text-white">18</strong> source files, <strong className="text-white">2,164</strong> lines of Go</li>
            <li><strong className="text-white">10</strong> test files, <strong className="text-white">1,365</strong> lines of tests</li>
            <li><strong className="text-white">28</strong> tests: budget, auth, delegation, config, JSON-RPC, SSE, integration</li>
            <li><strong className="text-white">63%</strong> test-to-source ratio</li>
          </ul>

          <section className="not-prose mt-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
            <h2 className="mb-6 text-2xl font-bold text-white">MCP budget enforcement engineering questions</h2>
            <div className="space-y-5">
              {[
                ['How does MCP budget enforcement work?', 'It intercepts tools/call messages, resolves the tool cost, checks the agent or token budget, and only forwards the request to the upstream MCP server when budget remains.'],
                ['Why use macaroons for MCP delegation?', 'Macaroons support attenuated delegation: each child token can add stricter caveats for budget, scope, expiry, and tools, but cannot remove parent constraints or expand authority.'],
                ['What happens when an MCP agent exhausts its budget?', 'The proxy returns a structured budget_exhausted error instead of forwarding the tool call, giving the agent a clear failure it can handle without creating an infinite retry or surprise bill.'],
              ].map(([question, answer]) => (
                <div key={question} className="border-t border-gray-800 pt-5 first:border-t-0 first:pt-0">
                  <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                  <p className="leading-relaxed text-gray-400">{answer}</p>
                </div>
              ))}
            </div>
          </section>

          <div className="mt-12 p-6 bg-gray-900/50 border border-gray-800 rounded-lg">
            <p className="text-gray-300 mb-4">
              The code is open source. Try it:
            </p>
            <pre className="bg-gray-900/70 rounded p-3 text-sm overflow-x-auto">
              <code className="text-green-300">{`go install github.com/satgate-io/satgate/cmd/satgate-mcp@latest`}</code>
            </pre>
            <p className="text-gray-400 text-sm mt-3">
              <a href="https://github.com/SatGate-io/satgate/tree/main/pkg/mcpserver" className="text-cyan-400 hover:text-cyan-300">GitHub →</a>
              {' · '}
              <a href="https://satgate.io/pricing" className="text-cyan-400 hover:text-cyan-300">Enterprise →</a>
            </p>
          </div>

        </article>

        <footer className="mt-16 pt-8 border-t border-gray-800 text-center">
          <Link href="/blog" className="text-gray-500 hover:text-white transition">
            ← Back to all posts
          </Link>
        </footer>
      </div>
    </div>
  );
}

import Link from 'next/link';
import { ArrowLeft, Calendar, Clock } from 'lucide-react';

export const metadata = {
  title: 'AI Agent Spending Limits: Hard Budgets by Agent, Tool, and Workflow',
  description: 'Set AI agent spending limits with hard budgets by agent, tool, model, workflow, and time window before API or MCP calls execute.',
  alternates: { canonical: 'https://satgate.io/blog/ai-agent-spending-limits' },
  keywords: ['AI agent spending limits', 'API budget control', 'agent cost management', 'economic firewall', 'API key vs budget', 'autonomous agent costs'],
  openGraph: {
    title: 'AI Agent Spending Limits: Hard Budgets by Agent and Tool',
    description: 'Set AI agent spending limits with hard budgets by agent, tool, model, workflow, and time window before calls execute.',
    url: 'https://satgate.io/blog/ai-agent-spending-limits',
    type: 'article',
    publishedTime: '2026-03-10T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI Agent Spending Limits: Hard Budgets by Agent and Tool',
    description: 'Enforce AI agent spend caps by agent, MCP tool, model, workflow, and time window before API calls execute.',
  },
};

export default function AiAgentSpendingLimitsBlogPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'AI Agent Spending Limits: Hard Budgets by Agent, Tool, and Workflow',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-03-10',
    dateModified: '2026-08-04',
    mainEntityOfPage: 'https://satgate.io/blog/ai-agent-spending-limits',
    about: [
      { '@type': 'Thing', name: 'AI agent spending limits' },
      { '@type': 'Thing', name: 'hard budgets by agent and workflow' },
      { '@type': 'Thing', name: 'API budget control' },
      { '@type': 'Thing', name: 'MCP tool spending caps' },
      { '@type': 'Thing', name: 'economic firewall budget enforcement' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What are AI agent spending limits?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'AI agent spending limits are request-path policies that cap how much an autonomous agent can spend by agent, tool, model, route, workflow, or time window at the gateway before forwarding to upstream APIs or MCP tools.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why are rate limits not enough for AI agent cost control?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Rate limits control request volume, not money. AI agents can still choose expensive tools, retry costly calls, or fan out across subtasks while staying under a request-per-minute limit.',
        },
      },
      {
        '@type': 'Question',
        name: 'Where should teams enforce AI agent spend limits?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Teams should enforce spending limits in the request path at an economic firewall or MCP proxy so budget, revocation, routing, and audit policy are checked before a costly call executes.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can AI agent spending limits be set per workflow or time window?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. AI agent spending limits can be scoped by workflow, task, agent, sub-agent, model, MCP tool, route, customer, environment, day, week, or token expiry window so each workload receives a precise hard budget.',
        },
      },
    ],
  };

  const policyChecklistJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent spending limits policy checklist',
    description: 'The minimum fields teams should define before giving autonomous agents access to paid APIs, models, or MCP tools.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Identity and scope',
        description: 'Tie each budget to tenant, agent, workflow, route, model, tool, customer, and delegated sub-agent identity.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Budget and time window',
        description: 'Set per-request, per-session, daily, weekly, workflow, and delegated-child budgets before execution.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Enforcement outcome',
        description: 'Define whether over-budget requests are denied, downgraded, queued, routed, revoked, or escalated.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Evidence Pack receipt fields',
        description: 'Record policy version, remaining budget, estimated cost, decision reason, agent identity, route, and upstream outcome.',
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(policyChecklistJsonLd) }} />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>
        
        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">Cost Control</span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">AI Agents</span>
            <span className="px-2 py-1 rounded-full bg-yellow-900/30 border border-yellow-500/30 text-yellow-300 text-xs font-mono">Budget Enforcement</span>
          </div>
          
          <h1 className="text-4xl font-bold mb-4">AI Agent Spending Limits: Why API Keys Aren't Enough</h1>
          
          <p className="text-xl text-gray-400 mb-6 italic">
            API rate limits don't control agent costs. Here's how economic firewalls enforce real-time budgets.
          </p>
          
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> March 10, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 9 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          
          <p className="text-gray-300 text-lg leading-relaxed">
            In a traditional setup, you guard your API with rate limits: <code>1000 RPM</code>. Any client exceeding that gets HTTP 429 "Too Many Requests."
          </p>
          <p className="text-gray-300 leading-relaxed">
            In contrast, AI agents auto-retry failed calls. Against a rate limit, many agents will simply retry blocked calls until they get through. In slowdown mode, they wait. In budget exhaustion mode, they fail gracefully.
          </p>

          <p className="text-gray-300 leading-relaxed">
            The problem isn't volume — it's unpredictability.
          </p>

          <p className="text-gray-300 leading-relaxed">
            For agents, you need <strong>budget limits</strong> — not rate limits. Predictable <em>spending</em>, not just predictable <em>requests</em>.
          </p>

          <div className="not-prose my-10 rounded-2xl border border-cyan-900/60 bg-cyan-950/20 p-6">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Implementation path</p>
            <h2 className="mb-3 text-2xl font-bold text-white">Turn this article into request-path policy</h2>
            <p className="mb-5 text-gray-300">
              If you are evaluating agent spend caps, start with the commercial overview, generate a policy, then model the downside from runaway loops.
            </p>
            <div className="grid gap-3 text-sm font-semibold sm:grid-cols-2">
              <Link href="/agent-spending-limits" className="rounded-lg border border-gray-800 bg-black/40 p-4 text-cyan-300 hover:border-cyan-500/50">Agent spending limits pillar →</Link>
              <Link href="/ai-agent-cost-control" className="rounded-lg border border-gray-800 bg-black/40 p-4 text-cyan-300 hover:border-cyan-500/50">AI agent cost-control page →</Link>
              <Link href="/agent-spend-policy-template" className="rounded-lg border border-gray-800 bg-black/40 p-4 text-cyan-300 hover:border-cyan-500/50">Generate spend policy →</Link>
              <Link href="/runaway-agent-cost-calculator" className="rounded-lg border border-gray-800 bg-black/40 p-4 text-cyan-300 hover:border-cyan-500/50">Model runaway spend →</Link>
            </div>
          </div>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Runaway Agent Horror Story</h2>
          <p className="text-gray-300 leading-relaxed">
            Imagine a user asks a research agent: <em>"Find me all AI startups in California."</em>
          </p>
          
          <p className="text-gray-300 leading-relaxed">
            The agent is designed to: 
          </p>

          <ul className="text-gray-300 space-y-2">
            <li>Search Google.</li>
            <li>For every result, visit the website.</li>
            <li>If the website mentions "AI," save it.</li>
          </ul>

          <p className="text-gray-300 leading-relaxed">
            What happens when it finds a "List of 1,000 Startups" directory?
          </p>

          <p className="text-gray-300 leading-relaxed">
            The agent dutifully visits all 1,000 links. Each visit requires a browser tool call and a summarization call (GPT-4).
          </p>

          <p className="text-gray-300 leading-relaxed">
            Cost per link: $0.10. Total Links: 1,000. Total Cost: $100.00 for a single query.
          </p>

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

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The agent spending limit policy checklist</h2>
          <p className="text-gray-300 leading-relaxed">
            A real spending limit needs more than one dollar value. It needs enough context to decide whether a request should run before the upstream API or MCP tool creates cost.
          </p>
          <ul className="text-gray-300 space-y-2">
            <li><strong className="text-white">Identity and scope:</strong> tenant, agent, workflow, model, route, tool, customer, and delegated sub-agent.</li>
            <li><strong className="text-white">Budget and time window:</strong> per-request, per-session, per-day, per-workflow, and delegated-child budgets.</li>
            <li><strong className="text-white">Enforcement outcome:</strong> deny, downgrade, queue, route, revoke, or escalate when budget is exhausted.</li>
            <li><strong className="text-white">Evidence fields:</strong> policy version, remaining budget, estimated cost, decision reason, receipt ID, and upstream outcome.</li>
          </ul>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Cost Granularity Matters</h2>
          <p className="text-gray-300 leading-relaxed">
            Not all tool calls cost the same. Our resolver supports exact match and wildcard prefixes:
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
            Resolution order: exact match → longest wildcard prefix → catch-all <code>*</code> → default.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">For Production Teams</h2>
          <p className="text-gray-300 leading-relaxed">
            Enterprise features like <code>RedisBudgetEnforcer</code> unlock:
          </p>

          <ul className="text-gray-300 space-y-2">
            <li><strong className="text-white">_RedisBudgetEnforcer_</strong>: Atomic spend tracking across replicas</li>
            <li><strong className="text-white">_Postgres Evidence Pack_</strong>: Spend attribution for chargebacks</li>
            <li><strong className="text-white">_paid-rail governance_</strong>: paid-rail context for external agent/API monetization</li>
          </ul>

          <section className="not-prose mt-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">FAQ</p>
            <h2 className="mb-6 text-2xl font-bold text-white">AI agent spending limit questions</h2>
            <div className="space-y-5">
              {[
                ['What are AI agent spending limits?', 'AI agent spending limits are request-path policies that cap how much an autonomous agent can spend by agent, tool, model, route, workflow, or time window at the gateway before forwarding to upstream APIs or MCP tools.'],
                ['Why are rate limits not enough for AI agent cost control?', 'Rate limits control request volume, not money. AI agents can still choose expensive tools, retry costly calls, or fan out across subtasks while staying under a request-per-minute limit.'],
                ['Where should teams enforce AI agent spend limits?', 'Enforce spending limits in the request path at an economic firewall or MCP proxy so budget, revocation, routing, and audit policy are checked before a costly call executes.'],
                ['Can AI agent spending limits be set per workflow or time window?', 'Yes. AI agent spending limits can be scoped by workflow, task, agent, sub-agent, model, MCP tool, route, customer, environment, day, week, or token expiry window so each workload receives a precise hard budget.'],
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
              The code is open source. Try it, or start with a generated policy before routing production traffic:
            </p>
            <pre className="bg-gray-900/70 rounded p-3 text-sm overflow-x-auto">
              <code className="text-green-300">{`go install github.com/satgate-io/satgate/cmd/satgate-mcp@latest`}</code>
            </pre>
            <p className="text-gray-400 text-sm mt-3">
              <a href="https://github.com/SatGate-io/satgate" className="text-cyan-400 hover:text-cyan-300">GitHub →</a>
              {' · '}
              <a href="https://satgate.io/pricing" className="text-cyan-400 hover:text-cyan-300">Enterprise →</a>
              {' · '}
              <Link href="/agent-spend-policy-template" className="text-cyan-400 hover:text-cyan-300">Spend policy template →</Link>
              {' · '}
              <Link href="/economic-firewall-readiness-grader" className="text-cyan-400 hover:text-cyan-300">Readiness grader →</Link>
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

import Link from 'next/link';
import { ArrowRight, BadgeDollarSign, Bot, ClipboardList, Gauge, KeyRound, ShieldCheck, Wrench, Zap } from 'lucide-react';

export const metadata = {
  title: 'MCP Budget Enforcement for AI Agents',
  description: 'Enforce budgets, prices, risk tiers, revocation, and Evidence Pack receipts before AI agents execute MCP tools. SatGate puts authority before execution in the MCP request path.',
  alternates: { canonical: 'https://satgate.io/mcp-budget-enforcement' },
  keywords: [
    'MCP budget enforcement',
    'MCP spend limits',
    'MCP tool cost control',
    'MCP governance',
    'MCP proxy budget limits',
    'Claude MCP budget enforcement',
    'Cursor MCP budget limits',
    'AI agent tool budgets',
    'Model Context Protocol governance',
    'MCP Evidence Pack receipts',
  ],
  openGraph: {
    title: 'MCP Budget Enforcement for AI Agents',
    description: 'Set enforceable budgets, prices, risk tiers, revocation, and Evidence Pack receipts before autonomous agents execute MCP tools.',
    url: 'https://satgate.io/mcp-budget-enforcement',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Budget Enforcement for AI Agents',
    description: 'Stop MCP tools from becoming an unbounded spend surface for autonomous agents.',
  },
};

const controls = [
  {
    icon: BadgeDollarSign,
    title: 'Per-tool pricing',
    body: 'Assign cost to search, browser, code, data, cloud, enrichment, and premium API tools before execution.',
  },
  {
    icon: Gauge,
    title: 'Budget ceilings',
    body: 'Enforce per-agent, per-session, per-tool, per-day, per-tenant, and per-request MCP spend limits.',
  },
  {
    icon: ShieldCheck,
    title: 'Risk tiers',
    body: 'Treat harmless local tools differently from expensive external APIs, write actions, or privileged cloud tools.',
  },
  {
    icon: KeyRound,
    title: 'Scoped capabilities',
    body: 'Replace broad static access with expiring, revocable capabilities constrained by tool, route, budget, and calls.',
  },
  {
    icon: ClipboardList,
    title: 'Policy evidence',
    body: 'Record who called which tool, estimated cost, remaining budget, policy decision, and upstream result.',
  },
  {
    icon: Zap,
    title: 'Paid-rail context',
    body: 'Preserve L402, x402, AgentCore Payments, Pay.sh, or enterprise billing context while SatGate decides authority before tool access.',
  },
];

const examples = [
  ['web_search', '$0.01/call', '$2/session', 'Allow until session budget is exhausted'],
  ['browser_automation', '$0.05/min', '$10/day', 'Require justification after 10 minutes'],
  ['code_execution', '$0.03/run', '$5/workflow', 'Block unsafe commands and log artifacts'],
  ['premium_data_api', '$0.25/call', '$25/tenant/day', 'Require paid-rail context and Evidence Pack receipt before access'],
  ['cloud_write_action', '$0.00 + risk', 'approval required', 'Deny unless capability includes write scope'],
];

export default function McpBudgetEnforcementPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'MCP Budget Enforcement for AI Agents',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-05-03',
    mainEntityOfPage: 'https://satgate.io/mcp-budget-enforcement',
    about: [
      { '@type': 'Thing', name: 'MCP budget enforcement' },
      { '@type': 'Thing', name: 'MCP spend limits' },
      { '@type': 'Thing', name: 'AI agent tool budgets' },
      { '@type': 'Thing', name: 'per-tool pricing policy' },
      { '@type': 'Thing', name: 'L402 paid MCP tool access' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is MCP budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'MCP budget enforcement means assigning prices, limits, policy, and Evidence Pack receipts to Model Context Protocol tool calls before an AI agent executes the tool.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do MCP tools need budget limits?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Autonomous agents can call tools repeatedly, delegate to sub-agents, or trigger expensive external APIs. Budget limits prevent MCP tools from becoming an unbounded spend surface.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can dashboards enforce MCP spend?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Dashboards can report spend after the fact. MCP budget enforcement needs to sit in the request path so policy can allow, deny, route, approve, require paid-rail context, or revoke before the tool executes.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate enforce MCP budgets?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate sits around MCP tool calls to observe activity, enforce authority and budget policy, preserve paid-rail context, and record each decision in an Evidence Pack.',
        },
      },
      {
        '@type': 'Question',
        name: 'What should an MCP tool budget include?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'An MCP tool budget should include tool identity, price per call or minute, agent and tenant scope, session and daily caps, risk tier, enforcement action, revocation behavior, and audit fields.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can MCP budget enforcement stop runaway tool loops?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. Request-path MCP budget enforcement can block or downgrade tool calls once a session, agent, tool, or tenant budget is exhausted, before additional paid work executes.',
        },
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'MCP Budget Enforcement', item: 'https://satgate.io/mcp-budget-enforcement' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_0%,rgba(34,211,238,0.18),transparent_30%),radial-gradient(circle_at_82%_12%,rgba(168,85,247,0.16),transparent_34%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/25 px-4 py-2 text-sm text-cyan-200 mb-8">
            <Wrench size={16} /> Budget enforcement for MCP tools
          </div>
          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            MCP Budget Enforcement for AI Agents
          </h1>
          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-10">
            MCP gives agents tools. SatGate gives those tools prices, budgets, risk tiers, revocation, and Evidence Pack receipts before autonomous agents can spend, loop, delegate, or call paid APIs.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Govern MCP tool budgets <ArrowRight size={18} />
            </Link>
            <Link href="/mcp-governance" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              See MCP governance
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12 items-start">
        <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
          <h2 className="text-3xl font-bold text-white mb-6">MCP made tool use easy. It did not make tool spend safe.</h2>
          <p>
            Model Context Protocol lets AI agents call search, browser, database, code, cloud, SaaS, and internal tools through a common interface. That is powerful — and economically dangerous when every call can trigger paid APIs, infrastructure, human review, or irreversible actions.
          </p>
          <p>
            Rate limits are too crude. Dashboards are too late. Approval queues do not scale when agents make hundreds of small decisions. MCP budget enforcement belongs in the request path, where each tool call can be priced, evaluated, allowed, denied, routed, approved, or bound to paid-rail context before execution.
          </p>
          <p>
            SatGate is the authority layer for that path: observe MCP activity, control risky spend before execution, and prove each budget or paid-rail decision with an Evidence Pack receipt.
          </p>
        </div>
        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">MCP budget policy answers</h3>
          <ul className="space-y-3 text-gray-300">
            {['What does this tool call cost?', 'Which agent, tenant, workflow, and delegated sub-agent made it?', 'Is the call inside budget right now?', 'Should this route allow, deny, downgrade, ask approval, or require paid-rail context?', 'Can finance and security explain the decision later?'].map((item) => (
              <li key={item} className="rounded-lg border border-gray-800 bg-black/50 p-3">{item}</li>
            ))}
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">Controls every MCP proxy needs</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            The goal is not to stop agents from using tools. The goal is to let them use tools with bounded economics, scoped authority, and proof for every decision.
          </p>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {controls.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 hover:border-cyan-500/40 transition">
                <Icon className="text-cyan-300 mb-4" size={28} />
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-4">Example MCP cost policy</h2>
        <p className="text-gray-400 max-w-3xl mb-8 text-lg">
          A useful policy starts by pricing tool classes, then setting hard ceilings, enforcement behavior, revocation rules, and Evidence Pack fields by risk.
        </p>
        <div className="overflow-hidden rounded-2xl border border-gray-800">
          <div className="grid grid-cols-4 bg-gray-950 px-5 py-3 text-sm font-bold uppercase tracking-wide text-gray-400">
            <div>Tool</div>
            <div>Price</div>
            <div>Budget</div>
            <div>Enforcement</div>
          </div>
          {examples.map(([tool, price, budget, enforcement]) => (
            <div key={tool} className="grid grid-cols-4 gap-4 border-t border-gray-800 px-5 py-4 text-sm text-gray-300">
              <div className="font-mono text-cyan-200">{tool}</div>
              <div>{price}</div>
              <div>{budget}</div>
              <div>{enforcement}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">MCP budget enforcement questions</h2>
          <div className="grid gap-5 md:grid-cols-2 mb-16">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is MCP budget enforcement?</h3>
              <p className="text-gray-400 leading-relaxed">
                MCP budget enforcement means assigning prices, limits, policy, and Evidence Pack receipts to Model Context Protocol tool calls before an AI agent executes the tool.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why do MCP tools need budget limits?</h3>
              <p className="text-gray-400 leading-relaxed">
                Autonomous agents can call tools repeatedly, delegate to sub-agents, or trigger expensive external APIs. Budget limits prevent MCP tools from becoming an unbounded spend surface.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Can dashboards enforce MCP spend?</h3>
              <p className="text-gray-400 leading-relaxed">
                Dashboards can report spend after the fact. MCP budget enforcement needs to sit in the request path so policy can allow, deny, route, approve, require paid-rail context, or revoke before the tool executes.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does SatGate enforce MCP budgets?</h3>
              <p className="text-gray-400 leading-relaxed">
                SatGate sits around MCP tool calls to observe activity, enforce authority and budget policy, preserve paid-rail context, and record each decision in an Evidence Pack.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What should an MCP tool budget include?</h3>
              <p className="text-gray-400 leading-relaxed">
                An MCP tool budget should include tool identity, price per call or minute, agent and tenant scope, session and daily caps, risk tier, enforcement action, revocation behavior, and audit fields.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Can MCP budget enforcement stop runaway tool loops?</h3>
              <p className="text-gray-400 leading-relaxed">
                Yes. Request-path MCP budget enforcement can block or downgrade tool calls once a session, agent, tool, or tenant budget is exhausted, before additional paid work executes.
              </p>
            </div>
          </div>

          <h2 className="text-3xl font-bold text-white mb-8">Related MCP governance resources</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
            {[
              ['/mcp-governance', 'MCP governance', 'Control, audit, and revoke MCP tool activity.'],
              ['/mcp-tool-cost-policy-generator', 'MCP policy generator', 'Generate a concrete tool-cost policy.'],
              ['/blog/mcp-budget-enforcement-guide', 'Budget enforcement guide', 'Deep dive on MCP budgets and spend controls.'],
              ['/blog/hard-capping-mcp-tool-spend', 'Hard-cap MCP spend', 'How to stop runaway tool loops.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 transition hover:border-cyan-500/50 hover:bg-cyan-950/10">
                <h3 className="font-bold text-white mb-2">{title}</h3>
                <p className="text-sm text-gray-400 leading-relaxed">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-5xl mx-auto px-6 py-20 text-center">
        <Bot className="mx-auto mb-6 text-cyan-300" size={42} />
        <h2 className="text-4xl font-bold text-white mb-5">Put budgets before MCP execution</h2>
        <p className="text-xl text-gray-300 leading-relaxed mb-8">
          If agents can call tools, they can spend. SatGate makes tool spend visible, enforceable, revocable, and provable in the request path.
        </p>
        <div className="flex flex-col sm:flex-row gap-4 justify-center">
          <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-cyan-300 text-black px-6 py-3 font-bold hover:bg-cyan-200 transition">
            Govern MCP budgets <ArrowRight size={18} />
          </Link>
          <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
            See Policy-to-Proof
          </Link>
        </div>
      </section>
    </main>
  );
}

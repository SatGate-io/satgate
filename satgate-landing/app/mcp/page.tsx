import Link from 'next/link';
import { ArrowRight, Bot, Gauge, ShieldCheck, Terminal, Wrench } from 'lucide-react';

export const metadata = {
  title: 'MCP for Budgeting: Tool Spend Governance for AI Agents',
  description: 'Use MCP for budgeting AI agent tools with per-tool costs, spend caps, scoped capabilities, revocation, expense tracking, and Evidence Packs.',
  alternates: { canonical: 'https://satgate.io/mcp' },
  keywords: [
    'MCP governance',
    'MCP for budgeting',
    'Model Context Protocol expense tracker',
    'MCP tool budget governance',
    'MCP expense tracking',
    'MCP budget enforcement',
    'MCP cost control',
    'MCP tool spend control',
    'economic firewall for MCP',
    'AI agent MCP security',
    'Cursor MCP budget control',
    'Claude Desktop MCP governance',
  ],
  openGraph: {
    title: 'MCP for Budgeting: Tool Spend Governance for AI Agents',
    description: 'Use MCP for budgeting with per-tool costs, spend caps, expense tracking, scoped capabilities, revocation, and Evidence Packs.',
    url: 'https://satgate.io/mcp',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP for Budgeting',
    description: 'Control MCP tool spend, authority, revocation, expense tracking, and audit evidence before tool calls execute.',
  },
};

const cards = [
  {
    href: '/mcp-governance',
    title: 'MCP Governance',
    description: 'The pillar guide for budget, identity, revocation, audit, and risk controls around MCP tool use.',
    icon: ShieldCheck,
  },
  {
    href: '/mcp-budget-enforcement',
    title: 'MCP Budget Enforcement',
    description: 'How to enforce session, workflow, tool, and per-call spend limits before expensive tools execute.',
    icon: Gauge,
  },
  {
    href: '/mcp-cost-control',
    title: 'MCP Cost Control',
    description: 'Turn MCP tool calls into priced, attributed, and governed economic events.',
    icon: Wrench,
  },
  {
    href: '/mcp-tool-cost-policy-generator',
    title: 'MCP Tool Cost Policy Generator',
    description: 'Generate YAML and JSON policy for MCP tool budgets, risk actions, audit fields, and revocation.',
    icon: Terminal,
  },
];

export default function MCPPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'MCP for Budgeting: Tool Spend Governance for AI Agents',
    url: 'https://satgate.io/mcp',
    description: metadata.description,
    datePublished: '2026-05-01',
    dateModified: '2026-08-06',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'MCP governance' },
      { '@type': 'Thing', name: 'MCP for budgeting' },
      { '@type': 'Thing', name: 'Model Context Protocol expense tracker' },
      { '@type': 'Thing', name: 'MCP tool budget governance' },
      { '@type': 'Thing', name: 'MCP expense tracking' },
      { '@type': 'Thing', name: 'MCP budget enforcement' },
      { '@type': 'Thing', name: 'MCP tool spend control' },
      { '@type': 'Thing', name: 'economic firewall for MCP' },
      { '@type': 'Thing', name: 'scoped capabilities for AI agents' },
    ],
  };

  const itemListJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'MCP governance resources',
    description: metadata.description,
    itemListElement: cards.map((card, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: card.title,
      url: `https://satgate.io${card.href}`,
      description: card.description,
    })),
  };

  const budgetingJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'MCP for budgeting controls',
    description: 'Budgeting and expense-tracking controls for Model Context Protocol tools used by AI agents.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Per-tool price table',
        description: 'Assign unit costs to MCP tools so search, browser, data export, code execution, and paid APIs can be budgeted separately.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Agent and workflow budgets',
        description: 'Track remaining budget by tenant, user, parent agent, child agent, workflow, session, and tool.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Pre-tool-call enforcement',
        description: 'Allow, deny, downgrade, route, or escalate before the MCP server executes an expensive tool call.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Expense tracker receipts',
        description: 'Emit Evidence Pack receipts showing agent, tool, unit price, spend, remaining budget, policy version, and denial reason.',
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'MCP Governance', item: 'https://satgate.io/mcp' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is MCP governance?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'MCP governance is the control layer around Model Context Protocol tool calls: budgets, scoped authority, revocation, Evidence Packs, and risk actions before agents execute tools.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do MCP tools need budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Autonomous agents can call paid or risky tools repeatedly, delegate work, or loop. MCP budget enforcement stops over-budget tool calls in the request path instead of discovering spend after the fact.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can MCP be used for budgeting AI agent tools?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. MCP can be used for budgeting when a gateway or proxy prices each tool call, attributes spend to the agent and workflow, checks remaining budget before execution, and emits expense-tracking receipts afterward.',
        },
      },
      {
        '@type': 'Question',
        name: 'What should a Model Context Protocol expense tracker record?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'A Model Context Protocol expense tracker should record agent identity, tenant, workflow, MCP tool name, unit price, call count, remaining budget, policy version, allow or deny decision, and Evidence Pack receipt id.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate control MCP spend?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate can proxy MCP traffic and enforce per-tool prices, session caps, workflow budgets, capability caveats, revocation, and audit requirements before tool calls reach the upstream MCP server.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(itemListJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(budgetingJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(168,85,247,0.2),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.16),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-purple-500/30 bg-purple-950/30 px-4 py-2 text-sm text-purple-200">
            <Bot size={16} /> Economic firewall for MCP tools
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            MCP for Budgeting AI Agent Tool Spend
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            MCP gives agents tools. SatGate turns those tools into budgeted, priced, and auditable economic events: per-tool budgets, expense tracking, scoped capabilities, revocation, Evidence Packs, and policy before execution.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/mcp-tool-cost-policy-generator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Generate MCP policy <ArrowRight size={18} />
            </Link>
            <Link href="/mcp-governance" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Read the governance guide
            </Link>
          </div>
        </div>
      </section>

      <section className="border-b border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <div className="mb-10 max-w-4xl">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">MCP for budgeting</p>
            <h2 className="mb-4 text-3xl font-bold text-white">Use MCP as the expense boundary for autonomous agent tools</h2>
            <p className="text-lg leading-relaxed text-gray-300">
              A Model Context Protocol expense tracker is only useful if it can stop the next tool call. SatGate prices MCP tools, checks the remaining budget before execution, and records the decision in an Evidence Pack.
            </p>
          </div>
          <div className="grid gap-5 md:grid-cols-4">
            {[
              ['Price table', 'Assign per-call costs to each MCP tool, route, model, data export, or paid API action.'],
              ['Budget scope', 'Limit spend by tenant, user, parent agent, child agent, workflow, session, and tool.'],
              ['Pre-call decision', 'Allow, deny, downgrade, route, or escalate before the MCP server executes.'],
              ['Expense proof', 'Record unit price, spend, remaining budget, policy version, and Evidence Pack receipt id.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-2xl border border-gray-800 bg-black p-5">
                <h3 className="mb-2 text-xl font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="grid gap-5 md:grid-cols-2">
          {cards.map(({ href, title, description, icon: Icon }) => (
            <Link key={href} href={href} className="group rounded-2xl border border-gray-800 bg-gray-950 p-6 transition hover:border-purple-500/50 hover:bg-purple-950/10">
              <Icon className="mb-5 text-purple-300 transition group-hover:text-purple-200" size={32} />
              <h2 className="mb-3 text-2xl font-bold text-white">{title}</h2>
              <p className="mb-5 leading-relaxed text-gray-400">{description}</p>
              <span className="inline-flex items-center gap-2 font-semibold text-purple-300">Open resource <ArrowRight size={16} /></span>
            </Link>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto grid max-w-6xl gap-8 px-6 py-20 lg:grid-cols-3">
          {[
            ['Price every tool', 'Assign cost profiles to MCP tools so agents cannot treat expensive operations like free function calls.'],
            ['Enforce before execution', 'Block over-budget or out-of-scope tool calls before they reach the upstream MCP server.'],
            ['Audit every decision', 'Emit receipt-backed proof for agent, workflow, tool, policy, budget, paid-call, denial, delegation, and revocation decisions.'],
          ].map(([title, body]) => (
            <div key={title} className="rounded-2xl border border-gray-800 bg-black p-6">
              <h2 className="mb-3 text-2xl font-bold text-white">{title}</h2>
              <p className="leading-relaxed text-gray-400">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto max-w-4xl px-6 py-20">
        <h2 className="mb-8 text-3xl font-bold text-white">MCP governance FAQ</h2>
        <div className="space-y-5">
          {[
            ['What is MCP governance?', 'MCP governance is the control layer around Model Context Protocol tool calls: budgets, scoped authority, revocation, Evidence Packs, and risk actions before agents execute tools.'],
            ['Can MCP be used for budgeting AI agent tools?', 'Yes. MCP can be used for budgeting when a gateway or proxy prices each tool call, attributes spend to the agent and workflow, checks remaining budget before execution, and emits expense-tracking receipts afterward.'],
            ['What should a Model Context Protocol expense tracker record?', 'It should record agent identity, tenant, workflow, MCP tool name, unit price, call count, remaining budget, policy version, allow or deny decision, and Evidence Pack receipt id.'],
            ['Why do MCP tools need budget enforcement?', 'Autonomous agents can call paid or risky tools repeatedly, delegate work, or loop. MCP budget enforcement stops over-budget tool calls in the request path instead of discovering spend after the fact.'],
            ['How does SatGate control MCP spend?', 'SatGate can proxy MCP traffic and enforce per-tool prices, session caps, workflow budgets, capability caveats, revocation, and audit requirements before tool calls reach the upstream MCP server.'],
          ].map(([question, answer]) => (
            <div key={question} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-3 text-xl font-bold text-white">{question}</h3>
              <p className="leading-relaxed text-gray-400">{answer}</p>
            </div>
          ))}
        </div>
      </section>
    </main>
  );
}

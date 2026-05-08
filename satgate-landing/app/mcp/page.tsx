import Link from 'next/link';
import { ArrowRight, Bot, Gauge, KeyRound, ShieldCheck, Terminal, Wrench } from 'lucide-react';

export const metadata = {
  title: 'MCP Governance for AI Agents',
  description: 'Govern MCP tool calls with per-tool budgets, scoped capabilities, revocation, audit trails, and economic firewall controls for Cursor, Claude, and OpenClaw.',
  alternates: { canonical: 'https://satgate.io/mcp' },
  keywords: [
    'MCP governance',
    'MCP budget enforcement',
    'MCP cost control',
    'MCP tool spend control',
    'economic firewall for MCP',
    'AI agent MCP security',
    'Cursor MCP budget control',
    'Claude Desktop MCP governance',
  ],
  openGraph: {
    title: 'MCP Governance for AI Agents',
    description: 'Request-path budgets, scoped capabilities, revocation, and audit trails for MCP tool calls.',
    url: 'https://satgate.io/mcp',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Governance for AI Agents',
    description: 'Control MCP tool spend, authority, revocation, and audit evidence before agent tool calls execute.',
  },
};

const controlPath = [
  ['Identify the caller', 'Bind each MCP tool call to an agent, user, workflow, tenant, and MCP server before policy is evaluated.'],
  ['Price the tool', 'Attach per-call, per-token, per-job, or per-export cost profiles so MCP usage becomes measurable economic activity.'],
  ['Enforce budget', 'Apply session, workflow, tenant, and per-tool caps before the upstream MCP server receives the request.'],
  ['Scope authority', 'Use revocable capabilities and caveats so tools cannot exceed route, time, data, or delegation limits.'],
  ['Audit the decision', 'Record the policy, budget, tool, cost estimate, allow/deny action, and revocation evidence for every call.'],
];

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
    name: 'MCP Governance for AI Agents',
    url: 'https://satgate.io/mcp',
    description: metadata.description,
    datePublished: '2026-05-01',
    dateModified: '2026-05-05',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'MCP governance' },
      { '@type': 'Thing', name: 'MCP budget enforcement' },
      { '@type': 'Thing', name: 'MCP tool spend control' },
      { '@type': 'Thing', name: 'economic firewall for MCP' },
      { '@type': 'Thing', name: 'scoped capabilities for AI agents' },
    ],
  };

  const controlPathJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'MCP request-path governance controls',
    description: 'The control sequence SatGate applies to MCP tool calls before upstream execution.',
    itemListElement: controlPath.map(([name, description], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name,
      description,
    })),
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
          text: 'MCP governance is the control layer around Model Context Protocol tool calls: budgets, scoped authority, revocation, audit trails, and risk actions before agents execute tools.',
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
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(controlPathJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(itemListJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(168,85,247,0.2),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.16),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-purple-500/30 bg-purple-950/30 px-4 py-2 text-sm text-purple-200">
            <Bot size={16} /> Economic firewall for MCP tools
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            MCP Governance for AI Agents
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            MCP gives agents tools. SatGate gives teams the economic firewall around those tools: per-tool budgets, scoped capabilities, revocation, audit trails, and policy before execution.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/mcp-tool-cost-policy-generator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Generate MCP policy <ArrowRight size={18} />
            </Link>
            <Link href="/mcp-governance" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Read the governance guide
            </Link>
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-purple-500">
              Grade MCP readiness
            </Link>
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
        <div className="mx-auto max-w-6xl px-6 py-20">
          <div className="mb-8 max-w-3xl">
            <h2 className="mb-3 text-3xl font-bold text-white">MCP request-path control sequence</h2>
            <p className="text-gray-400">A governed MCP call is not just routed. It is identified, priced, budgeted, scoped, and audited before a tool can create cost or risk.</p>
          </div>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-5">
            {controlPath.map(([title, body]) => (
              <div key={title} className="rounded-2xl border border-purple-900/50 bg-purple-950/10 p-5">
                <h3 className="mb-3 text-lg font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-black">
        <div className="mx-auto grid max-w-6xl gap-8 px-6 py-20 lg:grid-cols-3">
          {[
            ['Price every tool', 'Assign cost profiles to MCP tools so agents cannot treat expensive operations like free function calls.'],
            ['Enforce before execution', 'Block over-budget or out-of-scope tool calls before they reach the upstream MCP server.'],
            ['Audit every decision', 'Record agent, workflow, tool, policy, budget, and revocation evidence for every governed call.'],
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
            ['What is MCP governance?', 'MCP governance is the control layer around Model Context Protocol tool calls: budgets, scoped authority, revocation, audit trails, and risk actions before agents execute tools.'],
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

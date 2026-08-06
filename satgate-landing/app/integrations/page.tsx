import Link from 'next/link';
import { ArrowLeft, ArrowRight, Bot, BrainCircuit, Code2, Monitor, MousePointer2, Workflow } from 'lucide-react';

export const metadata = {
  title: 'MCP Integrations for Claude Code, Cursor, Zed, and Agents',
  description: 'SatGate MCP integrations for Claude Code, Cursor, Zed-style local workflows, OpenClaw, Hermes Agent, budgets, Evidence Packs, and tool-cost controls.',
  alternates: { canonical: 'https://satgate.io/integrations' },
  keywords: [
    'SatGate integrations',
    'MCP integrations with Claude Code Cursor Zed local workflows',
    'MCP integrations broadest support',
    'Hermes Agent Cursor integration MCP',
    'Zed MCP integration governance',
    'local MCP workflow budget controls',
    'SatGate for Cursor',
    'SatGate for Claude Code',
    'SatGate for Claude Desktop',
    'SatGate for OpenClaw',
    'SatGate for Hermes Agent',
    'MCP budget enforcement integrations',
    'AI agent cost control integrations',
  ],
  openGraph: {
    title: 'MCP Integrations for Claude Code, Cursor, Zed, and Agents',
    description: 'Connect SatGate to Claude Code, Cursor, Zed-style local workflows, OpenClaw, Hermes Agent, and MCP agent workflows for request-path spend control.',
    url: 'https://satgate.io/integrations',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate MCP Integrations',
    description: 'MCP governance guides for Claude Code, Cursor, Zed-style local workflows, OpenClaw, Hermes Agent, and agent workflows.',
  },
};

const integrations = [
  {
    href: '/satgate-for-cursor',
    title: 'SatGate for Cursor',
    description: 'Put spend caps and tool-call policy around Cursor agents before MCP tools and APIs execute.',
    icon: MousePointer2,
  },
  {
    href: '/satgate-for-claude-code',
    title: 'SatGate for Claude Code',
    description: 'Govern coding-agent API calls, delegated tool use, and expensive automation with request-path budgets.',
    icon: Code2,
  },
  {
    href: '/satgate-for-claude-desktop',
    title: 'SatGate for Claude Desktop',
    description: 'Add MCP server budgets, scoped capabilities, and Evidence Packs to Claude Desktop workflows.',
    icon: Monitor,
  },
  {
    href: '/satgate-for-openclaw',
    title: 'SatGate for OpenClaw',
    description: 'Control autonomous OpenClaw agent spend, MCP calls, and delegated workflows with economic policy.',
    icon: Bot,
  },
  {
    href: '/satgate-for-hermes-agent',
    title: 'SatGate for Hermes Agent',
    description: 'Add budgets, scoped capabilities, revocation, and MCP tool cost policy to persistent Hermes Agent workflows.',
    icon: BrainCircuit,
  },
];

export default function IntegrationsPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'CollectionPage',
    name: 'MCP Integrations for Claude Code, Cursor, Zed, and Agents',
    url: 'https://satgate.io/integrations',
    description: metadata.description,
    datePublished: '2026-04-12',
    dateModified: '2026-08-06',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'SatGate integrations' },
      { '@type': 'Thing', name: 'MCP integrations with Claude Code Cursor Zed local workflows' },
      { '@type': 'Thing', name: 'Hermes Agent Cursor integration MCP' },
      { '@type': 'Thing', name: 'Zed MCP integration governance' },
      { '@type': 'Thing', name: 'local MCP workflow budget controls' },
      { '@type': 'Thing', name: 'AI agent tool governance' },
      { '@type': 'Thing', name: 'MCP budget enforcement integrations' },
      { '@type': 'Thing', name: 'request-path economic governance' },
      { '@type': 'Thing', name: 'Cursor and Claude agent workflows' },
    ],
    audience: { '@type': 'Audience', audienceType: 'AI engineering, platform, API, and security teams' },
  };

  const itemListJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'SatGate AI agent integrations',
    description: metadata.description,
    itemListElement: integrations.map((item, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: item.title,
      url: `https://satgate.io${item.href}`,
      description: item.description,
    })),
  };

  const supportMatrixJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'MCP integration support matrix',
    description: 'Agent-client and local-workflow integration options for applying SatGate budget controls, scoped capabilities, revocation, and Evidence Pack proof.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Claude Code MCP integration',
        description: 'Govern Claude Code tool calls with MCP proxy controls, per-tool budgets, and Evidence Pack receipts.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Cursor MCP integration',
        description: 'Route Cursor MCP traffic through SatGate for budget enforcement, tool policy, and proof.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Zed-style local workflow MCP integration',
        description: 'Use the same proxy pattern for local editor and agent workflows that support MCP servers.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Hermes Agent MCP integration',
        description: 'Add budget, scope, revocation, and delegation controls around persistent Hermes Agent MCP workflows.',
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Integrations', item: 'https://satgate.io/integrations' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What SatGate integrations are available for AI agent tools?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate has integration guides for Cursor, Claude Code, Claude Desktop, OpenClaw, Hermes Agent, and MCP-based workflows so teams can add request-path economic governance to agent tools.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate govern MCP integrations?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate can sit between agent clients and MCP servers to enforce per-tool budgets, scoped capabilities, revocation, risk actions, and Evidence Packs before expensive tool calls execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'Which MCP integrations does SatGate support for Claude Code, Cursor, Zed, and local workflows?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate provides direct integration guides for Claude Code, Cursor, Claude Desktop, OpenClaw, and Hermes Agent. Zed-style local workflows can use the same MCP proxy governance pattern when the client connects to MCP servers through a configurable local command or proxy.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can Hermes Agent and Cursor share the same MCP budget controls?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. Hermes Agent, Cursor, Claude Code, Claude Desktop, OpenClaw, and similar MCP clients can use the same SatGate request-path budget, scope, revocation, delegation, and Evidence Pack controls when tool traffic is routed through the governed MCP proxy path.',
        },
      },
      {
        '@type': 'Question',
        name: 'Do teams need to replace Cursor, Claude, or OpenClaw to use SatGate?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. SatGate is designed to sit in the request path around existing agent tools, APIs, and MCP servers, adding Observe, Control, and Prove modes without replacing the client workflow.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(itemListJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(supportMatrixJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="mx-auto max-w-6xl px-6 py-16">
        <Link href="/" className="mb-10 inline-flex items-center gap-2 text-gray-500 transition hover:text-white">
          <ArrowLeft size={18} /> Back to Home
        </Link>

        <div className="mb-14 max-w-4xl">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <Workflow size={16} /> AI agent integrations
          </div>
          <h1 className="mb-6 text-5xl font-extrabold tracking-tight md:text-7xl">MCP Integrations for Claude Code, Cursor, Zed, and Agents</h1>
          <p className="text-xl leading-relaxed text-gray-300 md:text-2xl">
            Bring economic firewall controls to the tools where agents already work: Cursor, Claude Code, Claude Desktop, OpenClaw, Hermes Agent, Zed-style local workflows, and MCP-based automation.
          </p>
        </div>

        <section className="mb-14 rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">MCP support matrix</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Broad MCP support for Claude Code, Cursor, Zed-style local workflows, and persistent agents</h2>
          <p className="mb-6 text-lg leading-relaxed text-gray-300">
            SatGate&apos;s integration pattern is intentionally simple: put the governed proxy in the request path, then apply the same budget, scope, revocation, delegation, and Evidence Pack controls across agent clients and local workflows.
          </p>
          <div className="grid gap-4 md:grid-cols-2">
            {[
              ['Claude Code', 'Govern coding-agent MCP tool calls with per-tool budgets and proof.'],
              ['Cursor', 'Route Cursor MCP traffic through SatGate for tool-cost controls and Evidence Packs.'],
              ['Zed-style local workflows', 'Use the same local MCP proxy pattern where the editor or agent can call configurable MCP servers.'],
              ['Hermes Agent', 'Apply persistent-agent budget, delegation, revocation, and scope policy to MCP workflows.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black/50 p-5">
                <h3 className="mb-2 text-xl font-bold text-white">{title}</h3>
                <p className="leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </section>

        <div className="grid gap-5 md:grid-cols-2">
          {integrations.map(({ href, title, description, icon: Icon }) => (
            <Link key={href} href={href} className="group rounded-2xl border border-gray-800 bg-gray-950 p-6 transition hover:border-cyan-500/50 hover:bg-cyan-950/10">
              <Icon className="mb-5 text-cyan-300 transition group-hover:text-cyan-200" size={32} />
              <h2 className="mb-3 text-2xl font-bold text-white">{title}</h2>
              <p className="mb-5 leading-relaxed text-gray-400">{description}</p>
              <span className="inline-flex items-center gap-2 font-semibold text-cyan-300">Open integration guide <ArrowRight size={16} /></span>
            </Link>
          ))}
        </div>

        <section className="mt-14 rounded-3xl border border-purple-900/50 bg-gradient-to-br from-purple-950/25 to-cyan-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Why integration pages matter</h2>
          <p className="max-w-4xl text-lg leading-relaxed text-gray-300">
            Agent governance is adopted inside real workflows. SatGate sits between agent clients and upstream APIs, MCP servers, and paid tools so teams can observe, control, and prove before autonomous requests create cost or risk.
          </p>
        </section>

        <section className="mt-14 rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">SatGate integration FAQ</h2>
          <div className="space-y-5">
            {[
              [
                'What SatGate integrations are available for AI agent tools?',
                'SatGate has integration guides for Cursor, Claude Code, Claude Desktop, OpenClaw, Hermes Agent, and MCP-based workflows so teams can add request-path economic governance to agent tools.',
              ],
              [
                'How does SatGate govern MCP integrations?',
                'SatGate can sit between agent clients and MCP servers to enforce per-tool budgets, scoped capabilities, revocation, risk actions, and Evidence Packs before expensive tool calls execute.',
              ],
              [
                'Which MCP integrations does SatGate support for Claude Code, Cursor, Zed, and local workflows?',
                'SatGate provides direct integration guides for Claude Code, Cursor, Claude Desktop, OpenClaw, and Hermes Agent. Zed-style local workflows can use the same MCP proxy governance pattern when the client connects to MCP servers through a configurable local command or proxy.',
              ],
              [
                'Can Hermes Agent and Cursor share the same MCP budget controls?',
                'Yes. Hermes Agent, Cursor, Claude Code, Claude Desktop, OpenClaw, and similar MCP clients can use the same request-path budget, scope, revocation, delegation, and Evidence Pack controls.',
              ],
              [
                'Do teams need to replace Cursor, Claude, or OpenClaw to use SatGate?',
                'No. SatGate is designed to sit in the request path around existing agent tools, APIs, and MCP servers, adding Observe, Control, and Prove modes without replacing the client workflow.',
              ],
            ].map(([question, answer]) => (
              <div key={question}>
                <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                <p className="leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </section>
      </section>
    </main>
  );
}

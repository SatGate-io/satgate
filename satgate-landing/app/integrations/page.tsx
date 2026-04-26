import Link from 'next/link';
import { ArrowLeft, ArrowRight, Bot, Code2, Monitor, MousePointer2, Workflow } from 'lucide-react';

export const metadata = {
  title: 'SatGate Integrations for AI Agent Tools',
  description: 'Deploy SatGate economic governance with Cursor, Claude Code, Claude Desktop, OpenClaw, and MCP-based AI agent workflows.',
  alternates: { canonical: 'https://satgate.io/integrations' },
  keywords: [
    'SatGate integrations',
    'SatGate for Cursor',
    'SatGate for Claude Code',
    'SatGate for Claude Desktop',
    'SatGate for OpenClaw',
    'MCP budget enforcement integrations',
    'AI agent cost control integrations',
  ],
  openGraph: {
    title: 'SatGate Integrations for AI Agent Tools',
    description: 'Connect SatGate to Cursor, Claude Code, Claude Desktop, OpenClaw, and MCP agent workflows for request-path spend control.',
    url: 'https://satgate.io/integrations',
    type: 'website',
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
    description: 'Add MCP server budgets, scoped capabilities, and audit trails to Claude Desktop workflows.',
    icon: Monitor,
  },
  {
    href: '/satgate-for-openclaw',
    title: 'SatGate for OpenClaw',
    description: 'Control autonomous OpenClaw agent spend, MCP calls, and delegated workflows with economic policy.',
    icon: Bot,
  },
];

export default function IntegrationsPage() {
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
          text: 'SatGate has integration guides for Cursor, Claude Code, Claude Desktop, OpenClaw, and MCP-based workflows so teams can add request-path economic governance to agent tools.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate govern MCP integrations?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate can sit between agent clients and MCP servers to enforce per-tool budgets, scoped capabilities, revocation, risk actions, and audit trails before expensive tool calls execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'Do teams need to replace Cursor, Claude, or OpenClaw to use SatGate?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. SatGate is designed to sit in the request path around existing agent tools, APIs, and MCP servers, adding Observe, Control, and Charge modes without replacing the client workflow.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(itemListJsonLd) }} />
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
          <h1 className="mb-6 text-5xl font-extrabold tracking-tight md:text-7xl">SatGate Integrations</h1>
          <p className="text-xl leading-relaxed text-gray-300 md:text-2xl">
            Bring economic firewall controls to the tools where agents already work: Cursor, Claude Code, Claude Desktop, OpenClaw, and MCP-based automation.
          </p>
        </div>

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
            Agent governance is adopted inside real workflows. SatGate sits between agent clients and upstream APIs, MCP servers, and paid tools so teams can observe, control, and charge before autonomous requests create cost or risk.
          </p>
        </section>

        <section className="mt-14 rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">SatGate integration FAQ</h2>
          <div className="space-y-5">
            {[
              [
                'What SatGate integrations are available for AI agent tools?',
                'SatGate has integration guides for Cursor, Claude Code, Claude Desktop, OpenClaw, and MCP-based workflows so teams can add economic governance where agents already operate.',
              ],
              [
                'How does SatGate govern MCP integrations?',
                'SatGate can sit between agent clients and MCP servers to enforce per-tool budgets, scoped capabilities, revocation, risk actions, and audit trails before expensive tool calls execute.',
              ],
              [
                'Do teams need to replace their agent tools?',
                'No. SatGate is designed to wrap existing tools, APIs, and MCP servers with Observe, Control, and Charge modes instead of replacing the client workflow.',
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

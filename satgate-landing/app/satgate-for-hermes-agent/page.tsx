import Link from 'next/link';
import { ArrowRight, Bot, BrainCircuit, DollarSign, KeyRound, ShieldCheck, Terminal, Workflow } from 'lucide-react';

export const metadata = {
  title: 'SatGate for Hermes Agent MCP Workflows',
  description: 'Add request-path budgets, MCP tool cost policy, scoped capabilities, revocation, and Evidence Packs to Hermes Agent workflows with SatGate.',
  alternates: { canonical: 'https://satgate.io/satgate-for-hermes-agent' },
  keywords: [
    'SatGate for Hermes Agent',
    'Hermes Agent MCP budget enforcement',
    'Hermes Agent spend control',
    'Nous Hermes Agent MCP',
    'AI agent cost control',
    'economic firewall for AI agents',
    'revocable agent credentials',
  ],
  openGraph: {
    title: 'SatGate for Hermes Agent MCP Workflows',
    description: 'Give Hermes Agent workflows request-path budgets, MCP tool cost policy, scoped credentials, revocation, and Evidence Packs with SatGate.',
    url: 'https://satgate.io/satgate-for-hermes-agent',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate for Hermes Agent MCP Workflows',
    description: 'Economic governance for Hermes Agent MCP tools, API calls, credentials, and autonomous workflows.',
  },
};

const controls = [
  { icon: DollarSign, title: 'Budget before tool calls', body: 'Check session budget, per-tool price, request ceiling, and route policy before Hermes Agent forwards a paid call.' },
  { icon: KeyRound, title: 'Scoped capabilities', body: 'Replace broad static keys with expiring, revocable credentials constrained by route, MCP tool, calls, delegation, and spend.' },
  { icon: ShieldCheck, title: 'Revocation and kill switches', body: 'Stop a risky skill, workflow, tool, or session immediately without rotating every shared API secret.' },
  { icon: Workflow, title: 'MCP and API governance', body: 'Apply one economic policy across MCP servers, internal APIs, model routes, data tools, and paid resources.' },
];

export default function SatGateForHermesAgentPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'SatGate for Hermes Agent MCP Workflows',
    description: metadata.description,
    url: 'https://satgate.io/satgate-for-hermes-agent',
    dateModified: '2026-05-04',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'Hermes Agent MCP budget enforcement' },
      { '@type': 'Thing', name: 'Hermes Agent spend control' },
      { '@type': 'Thing', name: 'persistent agent economic governance' },
      { '@type': 'Thing', name: 'MCP tool cost policy' },
      { '@type': 'Thing', name: 'revocable agent credentials' },
    ],
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate for Hermes Agent',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Cloud, API Gateway, MCP Proxy',
    description: metadata.description,
    url: 'https://satgate.io/satgate-for-hermes-agent',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-04',
    about: webPageJsonLd.about,
    featureList: ['Hermes Agent MCP budget enforcement', 'AI agent spend control', 'MCP tool cost policy', 'Revocable capability tokens', 'Request-path Evidence Packs'],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Can SatGate govern Hermes Agent MCP tools?',
        acceptedAnswer: { '@type': 'Answer', text: 'Yes. SatGate can sit between Hermes Agent workflows and MCP servers or upstream APIs to enforce budgets, allowed tools, scoped credentials, revocation, and Evidence Packs before tool calls execute.' },
      },
      {
        '@type': 'Question',
        name: 'Why does a self-improving agent need economic governance?',
        acceptedAnswer: { '@type': 'Answer', text: 'Persistent or learning agents can reuse skills, retry workflows, call tools, and delegate work over time. SatGate adds request-path economic policy so those actions have budgets, scopes, expiry, and kill switches.' },
      },
      {
        '@type': 'Question',
        name: 'Does SatGate replace Hermes Agent?',
        acceptedAnswer: { '@type': 'Answer', text: 'No. Hermes Agent remains the agent workflow. SatGate adds the economic firewall around MCP tools, APIs, model routes, paid data sources, and credentials.' },
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Integrations', item: 'https://satgate.io/integrations' },
      { '@type': 'ListItem', position: 3, name: 'SatGate for Hermes Agent', item: 'https://satgate.io/satgate-for-hermes-agent' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_34%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.16),transparent_30%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <BrainCircuit size={16} /> Hermes Agent MCP spend control
          </div>

          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">Give Hermes Agent workflows an economic firewall</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Hermes Agent is built for persistent, skillful agent workflows. SatGate adds the missing economic firewall: per-agent budgets, MCP tool cost policy, scoped credentials, revocation, and audit before upstream APIs or tools execute.
          </p>

          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/mcp-proxy-config-generator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Generate MCP proxy config <ArrowRight size={18} />
            </Link>
            <Link href="/mcp-governance" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              MCP governance
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1fr_0.9fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">Why Hermes Agent workflows need request-path economics</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>Persistent agents can learn patterns, reuse skills, chain tools, and act across sessions. That is powerful — and it makes static API keys and delayed spend dashboards a bad control model.</p>
            <p>SatGate sits between Hermes Agent and upstream APIs, MCP servers, model providers, or paid resources. Every request gets an economic decision before access is granted.</p>
            <p>Teams can start in Observe mode to map real agent/tool spend, then move to Control policies for budgets, scopes, expiry, revocation, and per-tool limits.</p>
          </div>
        </div>
        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">What this prevents</h3>
          <ul className="space-y-3 text-gray-300">
            <li className="rounded-lg border border-gray-800 bg-black/60 p-4">Skill or workflow loops that silently burn paid model/API budget.</li>
            <li className="rounded-lg border border-gray-800 bg-black/60 p-4">Broad API keys inherited by autonomous tools without expiry or spend caps.</li>
            <li className="rounded-lg border border-gray-800 bg-black/60 p-4">MCP tool calls with no per-tool pricing, risk tier, audit field, or kill switch.</li>
            <li className="rounded-lg border border-gray-800 bg-black/60 p-4">Persistent sessions accumulating spend faster than finance or engineering can react.</li>
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-8 text-3xl font-bold text-white">Controls SatGate adds around Hermes Agent</h2>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
            {controls.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6">
                <Icon className="mb-4 text-cyan-300" size={28} />
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">Hermes Agent governance FAQ</h2>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              ['Can SatGate govern Hermes Agent MCP tools?', 'Yes. SatGate can sit between Hermes Agent workflows and MCP servers or upstream APIs to enforce budgets, allowed tools, scoped credentials, revocation, and Evidence Packs before tool calls execute.'],
              ['Why does a self-improving agent need economic governance?', 'Persistent or learning agents can reuse skills, retry workflows, call tools, and delegate work over time. SatGate adds request-path economic policy so those actions have budgets, scopes, expiry, and kill switches.'],
              ['Does SatGate replace Hermes Agent?', 'No. Hermes Agent remains the agent workflow. SatGate adds the economic firewall around MCP tools, APIs, model routes, paid data sources, and credentials.'],
            ].map(([question, answer]) => (
              <div key={question}>
                <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                <p className="leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 pb-20">
        <div className="rounded-3xl border border-purple-900/60 bg-gradient-to-br from-purple-950/40 to-cyan-950/20 p-8 md:p-12">
          <Bot className="mb-5 text-purple-300" size={32} />
          <h2 className="mb-4 text-3xl font-bold text-white">Hermes Agent can use MCP. SatGate makes MCP economically safe.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            Put SatGate in front of paid MCP servers, model routes, data APIs, and internal tools so Hermes Agent can act autonomously without receiving a blank check.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/agent-spend-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Generate spend policy <ArrowRight size={18} /></Link>
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">Grade readiness</Link>
          </div>
        </div>
      </section>
    </main>
  );
}

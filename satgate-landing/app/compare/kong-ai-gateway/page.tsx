import Link from 'next/link';
import { ArrowLeft, ArrowRight, Check, Gauge, KeyRound, ShieldCheck, Zap } from 'lucide-react';

export const metadata = {
  title: 'SatGate vs Kong AI Gateway: Agent Spend Controls',
  description: 'Compare SatGate and Kong AI Gateway for agent spend, MCP tools, scoped credentials, paid-rail context, and DeepInspect-vs-Kong AI gateway evaluation criteria.',
  alternates: { canonical: 'https://satgate.io/compare/kong-ai-gateway' },
  keywords: [
    'SatGate vs Kong AI Gateway',
    'DeepInspect vs Kong AI Gateway',
    'DeepInspect Kong AI Gateway comparison',
    'Kong AI Gateway agent spend controls',
    'Kong AI Gateway alternative',
    'AI gateway vs Policy-to-Proof governance',
    'Kong AI Gateway comparison',
    'agent API spend control',
    'SatGate comparison',
    'Policy-to-Proof governance',
    'AI agent cost control',
    'MCP budget enforcement',
  ],
  openGraph: {
    title: 'SatGate vs Kong AI Gateway: Agent Spend Controls',
    description: 'Compare SatGate and Kong AI Gateway for agent economics, MCP tool controls, scoped credentials, paid-rail context, and DeepInspect criteria.',
    url: 'https://satgate.io/compare/kong-ai-gateway',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate vs Kong AI Gateway',
    description: 'Agent spend controls, MCP tool budgets, scoped credentials, paid rails, and DeepInspect-vs-Kong evaluation criteria.',
  },
};

const rows: Array<[string, string, string]> = [
  ['Primary job', 'Policy-to-Proof governance for enterprise agents', 'API gateway, AI gateway, service connectivity, plugins, traffic policy, API platform operations'],
  ['Best fit', 'Agent/API spend governance, MCP tool budgets, scoped credentials, revocation, Evidence Packs, and paid-rail context', 'API gateway, AI gateway, service connectivity, plugins, traffic policy, API platform operations'],
  ['Request-path hard budget enforcement', 'Yes: at the gateway before forwarding to an upstream API, model, or MCP tool', 'Partial / depends on gateway policy and traffic type'],
  ['MCP tool budget enforcement', 'Yes: per-tool budgets, cost attribution, and deny decisions', 'Not the primary category focus'],
  ['Scoped revocable agent capabilities', 'Yes: route, tool, call, budget, expiry, delegation, and revocation caveats', 'Typically API keys, policies, tokens, or platform auth primitives'],
  ['Runaway agent spend benchmark/data', 'Yes: benchmark page plus JSON/CSV dataset', 'No direct equivalent'],
  ['L402 paid-agent API payments', 'Yes: governs paid-rail context before access and preserves Evidence Pack proof', 'No native SatGate-style paid-rail governance focus'],
  ['Broad API/AI platform management', 'Focused on economic governance layer', 'Yes / stronger fit'],
];

const evaluationCriteria = [
  ['Gateway policy', 'How API traffic, auth, routing, plugins, and AI provider connectivity are configured and operated.'],
  ['Inspection depth', 'Whether prompt, response, trace, tool, and policy signals can be inspected deeply enough for security review.'],
  ['Economic enforcement', 'Whether budgets, paid-tool costs, MCP tool limits, and model-routing decisions happen before spend is created.'],
  ['Proof', 'Whether the platform leaves Evidence Pack receipts for allow, deny, downgrade, revocation, delegation, and paid-rail context.'],
];

const satgateWins = [
  { icon: ShieldCheck, title: 'Policy-to-Proof for agents', body: 'SatGate decides whether an autonomous agent can spend, access, delegate, route, revoke, or pay before the next request executes.' },
  { icon: Gauge, title: 'Budgets beyond LLM tokens', body: 'Enforce cost controls across APIs, MCP tools, models, routes, workflows, tenants, agents, and delegated sub-agents.' },
  { icon: KeyRound, title: 'Scoped, revocable authority', body: 'Replace broad static keys with expiring capabilities constrained by route, tool, budget, calls, expiry, and delegation.' },
  { icon: Zap, title: 'Govern paid-rail access', body: 'Govern paid-rail context before external agents access APIs, tools, datasets, or premium capabilities at request time.' },
];

const competitorWins = [
  { icon: Check, title: 'Mature API gateway platform', body: 'Kong has deep API gateway, ingress, service connectivity, plugin, runtime, and platform-management capabilities.' },
  { icon: Check, title: 'Enterprise API operations', body: 'Kong fits teams standardizing broad API traffic, service connectivity, developer portals, API observability, and gateway operations.' },
];

export default function ComparePage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'SatGate vs Kong AI Gateway: Agent Spend Controls and DeepInspect Criteria',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-08-06',
    mainEntityOfPage: 'https://satgate.io/compare/kong-ai-gateway',
    about: [
      { '@type': 'Thing', name: 'SatGate vs Kong AI Gateway' },
      { '@type': 'Thing', name: 'DeepInspect vs Kong AI Gateway' },
      { '@type': 'Thing', name: 'Kong AI Gateway agent spend controls' },
      { '@type': 'Thing', name: 'AI gateway vs Policy-to-Proof governance' },
      { '@type': 'Thing', name: 'MCP budget enforcement' },
    ],
  };

  const criteriaJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'DeepInspect vs Kong AI Gateway evaluation criteria',
    description: 'Criteria teams can use when comparing DeepInspect, Kong AI Gateway, SatGate, and other AI gateway or inspection tools for agent governance.',
    itemListElement: evaluationCriteria.map(([name, description], index) => ({
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
      { '@type': 'Question', name: 'Is SatGate a Kong AI Gateway replacement?', acceptedAnswer: { '@type': 'Answer', text: 'Not directly. Kong AI Gateway is the AI-facing extension of a mature API gateway platform. SatGate is a Policy-to-Proof governance layer for AI agents, API spend, MCP tools, scoped capabilities, revocation, Evidence Packs, and paid-rail context.' } },
      { '@type': 'Question', name: 'Can SatGate and Kong AI Gateway work together?', acceptedAnswer: { '@type': 'Answer', text: 'Yes. SatGate can sit in front of or alongside gateway, API management, or observability infrastructure to enforce agent economics at the gateway before forwarding.' } },
      { '@type': 'Question', name: 'How should I evaluate DeepInspect vs Kong AI Gateway?', acceptedAnswer: { '@type': 'Answer', text: 'When evaluating DeepInspect vs Kong AI Gateway, separate inspection depth, gateway operations, and economic enforcement. Inspection tools help analyze prompts, traces, responses, or risks; API gateways manage traffic and connectivity; SatGate adds request-path agent spend controls, MCP tool budgets, scoped authority, and Evidence Pack proof.' } },
      { '@type': 'Question', name: 'When should I choose SatGate?', acceptedAnswer: { '@type': 'Answer', text: 'Choose SatGate when the core problem is autonomous agent economic governance: hard budgets, MCP tool spend, revocable credentials, delegated authority, Evidence Packs, and paid-agent payment.' } },
      { '@type': 'Question', name: 'When should I choose Kong AI Gateway?', acceptedAnswer: { '@type': 'Answer', text: 'Choose Kong when the primary need is a broad API gateway/API management platform with AI traffic support.' } },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(criteriaJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="mx-auto max-w-6xl px-6 py-16">
        <Link href="/compare" className="mb-8 flex items-center gap-2 text-gray-500 transition hover:text-white"><ArrowLeft size={18} /> Back to comparisons</Link>
        <div className="mb-12 max-w-4xl">
          <div className="mb-6 inline-flex rounded-full border border-cyan-500/30 bg-cyan-950/25 px-4 py-2 text-sm text-cyan-200">Comparison</div>
          <h1 className="mb-5 text-5xl font-extrabold tracking-tight md:text-7xl">SatGate vs Kong AI Gateway</h1>
          <p className="text-xl leading-relaxed text-gray-300 md:text-2xl">Kong AI Gateway is the AI-facing extension of a mature API gateway platform. SatGate is different: it is the request-path Policy-to-Proof governance layer for autonomous agents, API spend, MCP tools, scoped credentials, Evidence Packs, and paid-rail context.</p>
        </div>

        <section className="mb-14 rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">DeepInspect vs Kong AI Gateway</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Compare inspection, gateway operations, and economic enforcement separately</h2>
          <p className="mb-6 text-lg leading-relaxed text-gray-300">
            Teams comparing DeepInspect vs Kong AI Gateway are usually deciding where AI traffic inspection ends and runtime control begins. SatGate belongs in that evaluation as the agent economic control layer: budgets, MCP tool costs, scoped credentials, revocation, paid-rail context, and proof before requests execute.
          </p>
          <div className="grid gap-4 md:grid-cols-2">
            {evaluationCriteria.map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black/50 p-5">
                <h3 className="mb-2 text-xl font-bold text-white">{title}</h3>
                <p className="leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="mb-14 overflow-hidden rounded-2xl border border-gray-800">
          <div className="grid md:grid-cols-3 bg-gray-900/70 text-sm font-bold text-white"><div className="p-4">Capability</div><div className="p-4">SatGate</div><div className="p-4">Kong AI Gateway</div></div>
          {rows.map(([capability, satgate, competitor]) => (
            <div key={capability} className="grid md:grid-cols-3 border-t border-gray-800 text-gray-300"><div className="p-4 font-semibold text-white">{capability}</div><div className="p-4">{satgate}</div><div className="p-4">{competitor}</div></div>
          ))}
        </section>

        <section className="grid gap-8 lg:grid-cols-2">
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6"><h2 className="mb-5 text-2xl font-bold text-white">Where SatGate wins</h2><div className="space-y-4">{satgateWins.map(({ icon: Icon, title, body }) => (<div key={title} className="rounded-xl border border-gray-800 bg-black p-5"><Icon className="mb-3 text-cyan-300" size={24} /><h3 className="mb-2 font-bold text-white">{title}</h3><p className="text-sm leading-relaxed text-gray-400">{body}</p></div>))}</div></div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6"><h2 className="mb-5 text-2xl font-bold text-white">Where Kong AI Gateway wins</h2><div className="space-y-4">{competitorWins.map(({ icon: Icon, title, body }) => (<div key={title} className="rounded-xl border border-gray-800 bg-black p-5"><Icon className="mb-3 text-green-300" size={24} /><h3 className="mb-2 font-bold text-white">{title}</h3><p className="text-sm leading-relaxed text-gray-400">{body}</p></div>))}</div></div>
        </section>

        <section className="mt-14 rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">SatGate vs Kong AI Gateway FAQ</h2>
          <div className="grid gap-5 md:grid-cols-2">
            {[
              ['Is SatGate a Kong AI Gateway replacement?', 'Not directly. Kong AI Gateway is the AI-facing extension of a mature API gateway platform. SatGate is a Policy-to-Proof governance layer for AI agents, API spend, MCP tools, scoped capabilities, revocation, Evidence Packs, and paid-rail context.'],
              ['Can SatGate and Kong AI Gateway work together?', 'Yes. SatGate can sit in front of or alongside gateway, API management, or observability infrastructure to enforce agent economics at the gateway before forwarding.'],
              ['How should I evaluate DeepInspect vs Kong AI Gateway?', 'Separate inspection depth, gateway operations, and economic enforcement. Inspection tools analyze prompts, traces, responses, or risks; API gateways manage traffic and connectivity; SatGate adds request-path agent spend controls, MCP tool budgets, scoped authority, and Evidence Pack proof.'],
              ['When should I choose SatGate?', 'Choose SatGate when the core problem is autonomous agent economic governance: hard budgets, MCP tool spend, revocable credentials, delegated authority, Evidence Packs, and paid-agent payment.'],
              ['When should I choose Kong AI Gateway?', 'Choose Kong when the primary need is a broad API gateway/API management platform with AI traffic support.'],
            ].map(([question, answer]) => (
              <div key={question} className="rounded-xl border border-gray-800 bg-black p-5">
                <h3 className="mb-2 font-bold text-white">{question}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="mt-14 rounded-3xl border border-purple-900/60 bg-gradient-to-br from-purple-950/35 to-cyan-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Use the right layer.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">Gateways, API management, and observability tools are useful. They do not automatically solve agent economics. SatGate adds the pre-request decision layer: should this agent spend, access, delegate, revoke, route, or pay right now?</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Policy-to-Proof <ArrowRight size={18} /></Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">AI agent cost control</Link>
          </div>
        </section>
      </section>
    </main>
  );
}

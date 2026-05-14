import Link from 'next/link';
import { ArrowLeft, ArrowRight, Check, DollarSign, Gauge, KeyRound, Minus, ShieldCheck, Zap } from 'lucide-react';

export const metadata = {
  title: 'SatGate vs Tyk - API Management vs Agent Economic Governance',
  description: 'Compare SatGate and Tyk. Tyk is API management infrastructure; SatGate governs AI agent spend, MCP tools, scoped authority, revocation, and payments.',
  alternates: { canonical: 'https://satgate.io/compare/tyk' },
  keywords: [
    'SatGate vs Tyk',
    'Tyk alternative',
    'Tyk AI governance',
    'API gateway vs economic firewall',
    'AI native APIM comparison',
    'SatGate comparison',
    'economic firewall',
    'AI agent cost control',
    'MCP budget enforcement',
  ],
  openGraph: {
    title: 'SatGate vs Tyk - API Management vs Agent Economic Governance',
    description: 'Compare SatGate and Tyk for API management, AI agent spend governance, MCP tools, scoped authority, revocation, and paid-rail governance.',
    url: 'https://satgate.io/compare/tyk',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate vs Tyk - API Management vs Economic Firewall',
    description: 'Tyk manages API programs. SatGate enforces AI agent budgets, MCP tool costs, scoped authority, revocation, and paid-rail context.',
  },
};

const rows: Array<[string, string, string]> = [
  ['Primary job', 'Economic Firewall for AI agents', 'API gateway, API management, policies, developer portal, analytics, self-managed/hybrid/cloud deployment, AI-native APIM'],
  ['Best fit', 'Agent/API spend governance, MCP tool budgets, scoped credentials, revocation, audit, and paid-rail context', 'API gateway, API management, policies, developer portal, analytics, self-managed/hybrid/cloud deployment, AI-native APIM'],
  ['Request-path hard budget enforcement', 'Yes: before upstream API, model, or MCP tool access', 'Partial / depends on gateway policy and traffic type'],
  ['MCP tool budget enforcement', 'Yes: per-tool budgets, cost attribution, and deny decisions', 'Not the primary category focus'],
  ['Scoped revocable agent capabilities', 'Yes: route, tool, call, budget, expiry, delegation, and revocation caveats', 'Typically API keys, policies, tokens, or platform auth primitives'],
  ['Runaway agent spend benchmark/data', 'Yes: benchmark page plus JSON/CSV dataset', 'No direct equivalent'],
  ['L402 paid-agent API payments', 'Yes: Charge uses paid-rail context payment before access', 'No native SatGate-style paid-rail governance focus'],
  ['Broad API/AI platform management', 'Focused on economic governance layer', 'Yes / stronger fit'],
];

const satgateWins = [
  { icon: ShieldCheck, title: 'Economic firewall for agents', body: 'SatGate decides whether an autonomous agent can spend, access, delegate, route, revoke, or pay before the next request executes.' },
  { icon: Gauge, title: 'Budgets beyond LLM tokens', body: 'Enforce cost controls across APIs, MCP tools, models, routes, workflows, tenants, agents, and delegated sub-agents.' },
  { icon: KeyRound, title: 'Scoped, revocable authority', body: 'Replace broad static keys with expiring capabilities constrained by route, tool, budget, calls, expiry, and delegation.' },
  { icon: Zap, title: 'Charge paid agents', body: 'Use paid-rail context when external agents should pay for APIs, tools, datasets, or premium capabilities at request time.' },
];

const competitorWins = [
  { icon: Check, title: 'Flexible API management', body: 'Tyk offers API gateway and management features across self-managed, hybrid, and cloud deployment models.' },
  { icon: Check, title: 'API policy and portal workflows', body: 'Tyk fits teams managing API products, access policy, analytics, developer portals, and multi-protocol API operations.' },
];

export default function ComparePage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'SatGate vs Tyk - API Management vs Agent Economic Governance',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-05-04',
    mainEntityOfPage: 'https://satgate.io/compare/tyk',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'Is SatGate a Tyk replacement?', acceptedAnswer: { '@type': 'Answer', text: 'Not directly. Tyk is an API management platform with gateway, governance, analytics, and portal capabilities. SatGate is an economic firewall for AI agents, API spend, MCP tools, scoped capabilities, revocation, audit, and paid-rail context.' } },
      { '@type': 'Question', name: 'Can SatGate and Tyk work together?', acceptedAnswer: { '@type': 'Answer', text: 'Yes. SatGate can sit in front of or alongside gateway, API management, or observability infrastructure to enforce agent economics before upstream access.' } },
      { '@type': 'Question', name: 'When should I choose SatGate?', acceptedAnswer: { '@type': 'Answer', text: 'Choose SatGate when the core problem is autonomous agent economic governance: hard budgets, MCP tool spend, revocable credentials, delegated authority, Evidence Packs, and paid-agent payment.' } },
      { '@type': 'Question', name: 'When should I choose Tyk?', acceptedAnswer: { '@type': 'Answer', text: 'Choose Tyk when the primary need is a flexible API management platform for publishing and operating APIs.' } },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="mx-auto max-w-6xl px-6 py-16">
        <Link href="/compare" className="mb-8 flex items-center gap-2 text-gray-500 transition hover:text-white"><ArrowLeft size={18} /> Back to comparisons</Link>
        <div className="mb-12 max-w-4xl">
          <div className="mb-6 inline-flex rounded-full border border-cyan-500/30 bg-cyan-950/25 px-4 py-2 text-sm text-cyan-200">Comparison</div>
          <h1 className="mb-5 text-5xl font-extrabold tracking-tight md:text-7xl">SatGate vs Tyk</h1>
          <p className="text-xl leading-relaxed text-gray-300 md:text-2xl">Tyk is an API management platform with gateway, governance, analytics, and portal capabilities. SatGate is different: it is the request-path economic firewall for autonomous agents, API spend, MCP tools, scoped credentials, audit, and L402 paid-agent payments.</p>
        </div>

        <section className="mb-14 overflow-hidden rounded-2xl border border-gray-800">
          <div className="grid md:grid-cols-3 bg-gray-900/70 text-sm font-bold text-white"><div className="p-4">Capability</div><div className="p-4">SatGate</div><div className="p-4">Tyk</div></div>
          {rows.map(([capability, satgate, competitor]) => (
            <div key={capability} className="grid md:grid-cols-3 border-t border-gray-800 text-gray-300"><div className="p-4 font-semibold text-white">{capability}</div><div className="p-4">{satgate}</div><div className="p-4">{competitor}</div></div>
          ))}
        </section>

        <section className="grid gap-8 lg:grid-cols-2">
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6"><h2 className="mb-5 text-2xl font-bold text-white">Where SatGate wins</h2><div className="space-y-4">{satgateWins.map(({ icon: Icon, title, body }) => (<div key={title} className="rounded-xl border border-gray-800 bg-black p-5"><Icon className="mb-3 text-cyan-300" size={24} /><h3 className="mb-2 font-bold text-white">{title}</h3><p className="text-sm leading-relaxed text-gray-400">{body}</p></div>))}</div></div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6"><h2 className="mb-5 text-2xl font-bold text-white">Where Tyk wins</h2><div className="space-y-4">{competitorWins.map(({ icon: Icon, title, body }) => (<div key={title} className="rounded-xl border border-gray-800 bg-black p-5"><Icon className="mb-3 text-green-300" size={24} /><h3 className="mb-2 font-bold text-white">{title}</h3><p className="text-sm leading-relaxed text-gray-400">{body}</p></div>))}</div></div>
        </section>

        <section className="mt-14 rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">SatGate vs Tyk FAQ</h2>
          <div className="grid gap-5 md:grid-cols-2">
            {[
              ['Is SatGate a Tyk replacement?', 'Not directly. Tyk is an API management platform with gateway, governance, analytics, and portal capabilities. SatGate is an economic firewall for AI agents, API spend, MCP tools, scoped capabilities, revocation, audit, and paid-rail context.'],
              ['Can SatGate and Tyk work together?', 'Yes. SatGate can sit in front of or alongside gateway, API management, or observability infrastructure to enforce agent economics before upstream access.'],
              ['When should I choose SatGate?', 'Choose SatGate when the core problem is autonomous agent economic governance: hard budgets, MCP tool spend, revocable credentials, delegated authority, Evidence Packs, and paid-agent payment.'],
              ['When should I choose Tyk?', 'Choose Tyk when the primary need is a flexible API management platform for publishing and operating APIs.'],
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
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Economic firewall <ArrowRight size={18} /></Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">AI agent cost control</Link>
          </div>
        </section>
      </section>
    </main>
  );
}

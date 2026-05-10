import Link from 'next/link';
import { ArrowLeft, ArrowRight, Check, Gauge, KeyRound, Minus, ShieldCheck, Zap } from 'lucide-react';

export const metadata = {
  title: 'SatGate vs Helicone - LLM Observability vs Economic Firewall',
  description: 'Compare SatGate and Helicone. Helicone helps teams debug and analyze AI apps; SatGate adds economic governance for agents, MCP budgets, and paid-rail context.',
  alternates: { canonical: 'https://satgate.io/compare/helicone' },
  keywords: [
    'SatGate vs Helicone',
    'Helicone alternative',
    'LLM observability vs economic firewall',
    'AI agent cost control',
    'LLM cost governance',
  ],
  openGraph: {
    title: 'SatGate vs Helicone - LLM Observability vs Economic Firewall',
    description: 'Compare SatGate and Helicone for LLM observability, AI agent economic governance, MCP budgets, capabilities, and paid-rail context.',
    url: 'https://satgate.io/compare/helicone',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate vs Helicone - Observability vs Economic Firewall',
    description: 'Helicone observes LLM apps. SatGate enforces AI agent budgets, MCP tool costs, scoped authority, audit, and paid-rail context.',
  },
};

const rows: Array<[string, string, string]> = [
  ['Primary job', 'Economic control plane for AI agents', 'LLM observability / AI gateway / debugging and analytics'],
  ['Best fit', 'Agent/API spend governance, MCP tool budgets, scoped access, audit, robot payments', 'Routing, debugging, request analytics, provider integrations, LLM app monitoring'],
  ['Request-path hard budget enforcement', 'Yes', 'Partial or adjacent, depending on gateway limits and usage policy'],
  ['MCP tool budget enforcement', 'Yes', 'Not the primary economic-control focus'],
  ['Per-tool cost attribution beyond LLM calls', 'Yes', 'Primarily LLM/app traffic or gateway telemetry'],
  ['Scoped revocable agent capabilities', 'Yes', 'Adjacent access controls, not SatGate-style attenuated capabilities'],
  ['L402 paid-agent API payments', 'Yes', 'No native paid-rail governance equivalent'],
  ['AI gateway / model traffic management', 'Partial', 'Yes'],
  ['Observability and analytics', 'Yes, economic audit oriented', 'Yes'],
  ['Open-source/self-hostable posture', 'Yes', 'Varies by product and deployment tier'],
];

const satgateWins = [
  { icon: ShieldCheck, title: 'Economic firewall in the request path', body: 'SatGate decides whether an agent should access, spend, route, delegate, or pay before upstream APIs, MCP tools, and model calls execute.' },
  { icon: Gauge, title: 'Hard budgets for autonomous workflows', body: 'Control spend by tenant, agent, workflow, delegated sub-agent, route, model, tool, session, day, and request.' },
  { icon: KeyRound, title: 'Scoped, revocable agent authority', body: 'Issue expiring capabilities constrained by route, tool, budget, call count, expiry, and delegation rules instead of broad static keys.' },
  { icon: Zap, title: 'Charge paid agents', body: 'Use paid-rail context when external agents should pay for APIs, datasets, tools, or premium capabilities at request time.' },
];

const competitorWins: Array<{ title: string; body: string }> = [
  { title: 'LLM observability', body: 'Helicone is a natural fit when the team needs deep visibility into LLM requests, debugging, and analytics.' },
  { title: 'Developer debugging workflow', body: 'Helicone is strong when developers want to inspect LLM application behavior across common providers and frameworks.' },
];

export default function CompareHeliconePage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'SatGate vs Helicone - LLM Observability vs Economic Firewall',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-05-04',
    mainEntityOfPage: 'https://satgate.io/compare/helicone',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Is SatGate a Helicone replacement?',
        acceptedAnswer: { '@type': 'Answer', text: 'Not directly. Helicone and SatGate solve overlapping but different problems. SatGate is focused on economic governance for AI agents, APIs, MCP tools, scoped authority, and request-time payments.' },
      },
      {
        '@type': 'Question',
        name: 'Can SatGate and Helicone work together?',
        acceptedAnswer: { '@type': 'Answer', text: 'Yes. SatGate can enforce agent, workflow, budget, route, MCP tool, and capability policy before requests reach an upstream AI gateway or observability layer.' },
      },
      {
        '@type': 'Question',
        name: 'When should I choose SatGate?',
        acceptedAnswer: { '@type': 'Answer', text: 'Choose SatGate when autonomous agents need hard budget enforcement, MCP tool spend controls, scoped revocation, delegation policy, economic Evidence Packs, or paid-rail context for paid agents.' },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="mx-auto max-w-6xl px-6 py-16">
        <Link href="/compare" className="mb-10 inline-flex items-center gap-2 text-gray-500 transition hover:text-white">
          <ArrowLeft size={18} /> Back to comparisons
        </Link>

        <div className="mb-12">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-3 py-1 text-xs font-mono text-cyan-300">Comparison</div>
          <h1 className="mb-6 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">SatGate vs Helicone</h1>
          <p className="max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">Helicone is strong for LLM observability, debugging, routing, and analytics. SatGate is built for the moment visibility is not enough: preventing autonomous agents from overspending, overdelegating, or accessing paid tools without policy.</p>
        </div>

        <section className="mb-12 rounded-2xl border border-cyan-800/30 bg-gradient-to-r from-cyan-950/20 to-purple-950/20 p-6">
          <h2 className="mb-3 text-xl font-bold text-white">TL;DR</h2>
          <p className="leading-relaxed text-gray-300">Use Helicone when the main problem is understanding, debugging, and analyzing LLM application behavior. Use SatGate when the main problem is enforcing economic policy before agents spend money, call MCP tools, or access paid APIs.</p>
        </section>

        <section className="mb-16 overflow-hidden rounded-2xl border border-gray-800">
          <div className="grid grid-cols-3 bg-gray-950 px-5 py-4 text-sm font-bold uppercase tracking-wide text-gray-400">
            <div>Question</div><div className="text-cyan-300">SatGate</div><div>Helicone</div>
          </div>
          {rows.map(([feature, satgate, competitor]) => (
            <div key={feature} className="grid grid-cols-3 gap-4 border-t border-gray-800 px-5 py-4 text-sm text-gray-300">
              <div className="font-medium text-white">{feature}</div>
              <div>{satgate}</div>
              <div className="text-gray-400">{competitor}</div>
            </div>
          ))}
        </section>

        <section className="mb-16 grid gap-8 lg:grid-cols-2">
          <div>
            <h2 className="mb-6 text-3xl font-bold text-white">Where SatGate wins</h2>
            <div className="space-y-4">
              {satgateWins.map(({ icon: Icon, title, body }) => (
                <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-5">
                  <Icon className="mb-3 text-cyan-300" size={26} />
                  <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                  <p className="leading-relaxed text-gray-400">{body}</p>
                </div>
              ))}
            </div>
          </div>
          <div>
            <h2 className="mb-6 text-3xl font-bold text-white">Where Helicone wins</h2>
            <div className="space-y-4">
              {competitorWins.map(({ title, body }) => (
                <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-5">
                  <Minus className="mb-3 text-purple-300" size={26} />
                  <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                  <p className="leading-relaxed text-gray-400">{body}</p>
                </div>
              ))}
            </div>
            <div className="mt-6 rounded-xl border border-yellow-900/50 bg-yellow-950/10 p-5">
              <h3 className="mb-2 text-lg font-bold text-white">The clean architecture</h3>
              <p className="leading-relaxed text-gray-300">SatGate can sit in front of or beside a gateway/observability layer. SatGate enforces economic policy before the request reaches tools, models, APIs, or payment-gated resources.</p>
            </div>
          </div>
        </section>

        <section className="mb-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">Feature signal</h2>
          <div className="grid gap-4 md:grid-cols-2">
            {([
              ['Hard agent budget enforcement', true, 'Policy decisions before expensive agent/API activity executes.'],
              ['MCP tool cost policy', true, 'Prices, limits, risk tiers, and audit evidence per tool call.'],
              ['L402 robot payments', true, 'Request-time paid-rail context for external agent/API access.'],
              ['Broad AI gateway/observability suite', false, 'This is where Helicone may be the better fit; SatGate governs the economics around that layer.'],
            ] as Array<[string, boolean, string]>).map(([label, yes, body]) => (
              <div key={label} className="rounded-xl border border-gray-800 bg-black p-5">
                <div className="mb-3 flex items-center gap-3">
                  {yes ? <Check className="text-green-400" size={18} /> : <Minus className="text-gray-500" size={18} />}
                  <h3 className="font-bold text-white">{label}</h3>
                </div>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="mb-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">SatGate vs Helicone FAQ</h2>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              ['Is SatGate a Helicone replacement?', 'Not directly. Helicone and SatGate solve overlapping but different problems. SatGate is focused on economic governance for AI agents, APIs, MCP tools, scoped authority, and request-time payments.'],
              ['Can SatGate and Helicone work together?', 'Yes. SatGate can enforce agent, workflow, budget, route, MCP tool, and capability policy before requests reach an upstream AI gateway or observability layer.'],
              ['When should I choose SatGate?', 'Choose SatGate when autonomous agents need hard budget enforcement, MCP tool spend controls, scoped revocation, delegation policy, economic Evidence Packs, or paid-rail context for paid agents.'],
            ].map(([question, answer]) => (
              <div key={question} className="rounded-xl border border-gray-800 bg-black p-5">
                <h3 className="mb-2 font-bold text-white">{question}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Gateway features are useful. Economic governance is different.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">SatGate helps platform, finance, and security teams control what autonomous agents can spend, access, delegate, and monetize before the next request leaves the building.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Learn economic firewalls <ArrowRight size={18} /></Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">See agent cost control</Link>
          </div>
        </section>
      </section>
    </main>
  );
}

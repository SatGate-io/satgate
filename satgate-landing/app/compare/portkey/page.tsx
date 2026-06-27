import Link from 'next/link';
import { ArrowLeft, ArrowRight, Check, Gauge, KeyRound, Minus, ShieldCheck, Zap } from 'lucide-react';

export const metadata = {
  title: 'SatGate vs Portkey - AI Gateway vs Economic Firewall',
  description: 'Compare SatGate and Portkey. Portkey is a GenAI gateway and observability stack; SatGate governs agent spend, MCP tools, and paid-rail context.',
  alternates: { canonical: 'https://satgate.io/compare/portkey' },
  keywords: [
    'SatGate vs Portkey',
    'Portkey alternative',
    'AI gateway economic governance',
    'Portkey cost control',
    'MCP gateway budget enforcement',
  ],
  openGraph: {
    title: 'SatGate vs Portkey - AI Gateway vs Economic Firewall',
    description: 'Compare SatGate and Portkey for AI gateway, observability, guardrails, MCP access, agent economics, and paid-rail context.',
    url: 'https://satgate.io/compare/portkey',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate vs Portkey - GenAI Gateway vs Economic Firewall',
    description: 'Portkey runs GenAI gateway ops. SatGate enforces agent budgets, MCP tool costs, scoped authority, audit, and paid-rail context.',
  },
};

const rows: Array<[string, string, string]> = [
  ['Primary job', 'Policy-to-Proof governance for enterprise agents', 'Production GenAI stack / AI gateway / observability / guardrails'],
  ['Best fit', 'Agent/API spend governance, MCP tool budgets, scoped access, audit, paid-rail governance', 'AI gateway, observability, guardrails, prompt management, governance, MCP access centralization'],
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
  { icon: ShieldCheck, title: 'Economic firewall in the request path', body: 'SatGate decides whether an agent should access, spend, route, delegate, or pay at the gateway before forwarding to upstream APIs, MCP tools, and model calls.' },
  { icon: Gauge, title: 'Hard budgets for autonomous workflows', body: 'Control spend by tenant, agent, workflow, delegated sub-agent, route, model, tool, session, day, and request.' },
  { icon: KeyRound, title: 'Scoped, revocable agent authority', body: 'Issue expiring capabilities constrained by route, tool, budget, call count, expiry, and delegation rules instead of broad static keys.' },
  { icon: Zap, title: 'Govern paid-rail access', body: 'Govern paid-rail context before external agents access APIs, datasets, tools, or premium capabilities at request time.' },
];

const competitorWins: Array<{ title: string; body: string }> = [
  { title: 'Broad GenAI production stack', body: 'Portkey covers a wider GenAI operations surface: gateway, observability, guardrails, prompt management, and model operations.' },
  { title: 'LLM app observability and guardrails', body: 'Portkey is a better fit when the main pain is monitoring LLM app behavior and enforcing output guardrails.' },
];

export default function ComparePortkeyPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'SatGate vs Portkey - AI Gateway vs Economic Firewall',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-05-04',
    mainEntityOfPage: 'https://satgate.io/compare/portkey',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Is SatGate a Portkey replacement?',
        acceptedAnswer: { '@type': 'Answer', text: 'Not directly. Portkey and SatGate solve overlapping but different problems. SatGate is focused on economic governance for AI agents, APIs, MCP tools, scoped authority, and request-time payments.' },
      },
      {
        '@type': 'Question',
        name: 'Can SatGate and Portkey work together?',
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
          <h1 className="mb-6 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">SatGate vs Portkey</h1>
          <p className="max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">Portkey is a serious production GenAI platform: AI Gateway, observability, guardrails, governance, prompt management, and MCP access. SatGate is narrower and sharper: economic governance for autonomous agent/API activity in the request path.</p>
        </div>

        <section className="mb-12 rounded-2xl border border-cyan-800/30 bg-gradient-to-r from-cyan-950/20 to-purple-950/20 p-6">
          <h2 className="mb-3 text-xl font-bold text-white">TL;DR</h2>
          <p className="leading-relaxed text-gray-300">Use Portkey when you want a broad GenAI production stack around gateway, observability, guardrails, prompts, and model operations. Use SatGate when the core problem is economic control: what agents can spend, which MCP tools they can call, what authority they can delegate, and when external agents should pay.</p>
        </section>

        <section className="mb-16 overflow-hidden rounded-2xl border border-gray-800">
          <div className="grid grid-cols-3 bg-gray-950 px-5 py-4 text-sm font-bold uppercase tracking-wide text-gray-400">
            <div>Question</div><div className="text-cyan-300">SatGate</div><div>Portkey</div>
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
            <h2 className="mb-6 text-3xl font-bold text-white">Where Portkey wins</h2>
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
              ['L402 paid-rail governance', true, 'Request-time paid-rail context for external agent/API access.'],
              ['Broad AI gateway/observability suite', false, 'This is where Portkey may be the better fit; SatGate governs the economics around that layer.'],
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
          <h2 className="mb-6 text-3xl font-bold text-white">SatGate vs Portkey FAQ</h2>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              ['Is SatGate a Portkey replacement?', 'Not directly. Portkey and SatGate solve overlapping but different problems. SatGate is focused on economic governance for AI agents, APIs, MCP tools, scoped authority, and request-time payments.'],
              ['Can SatGate and Portkey work together?', 'Yes. SatGate can enforce agent, workflow, budget, route, MCP tool, and capability policy before requests reach an upstream AI gateway or observability layer.'],
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
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">See Policy-to-Proof <ArrowRight size={18} /></Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">See agent cost control</Link>
          </div>
        </section>
      </section>
    </main>
  );
}

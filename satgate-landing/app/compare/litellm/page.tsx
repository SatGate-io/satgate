import Link from 'next/link';
import { ArrowLeft, ArrowRight, Check, DollarSign, Gauge, KeyRound, Minus, Route, ShieldCheck, Zap } from 'lucide-react';

export const metadata = {
  title: 'SatGate vs LiteLLM - AI Gateway vs Economic Firewall',
  description: 'Compare SatGate and LiteLLM. LiteLLM handles model routing, budgets, and spend tracking; SatGate governs agent spend, MCP tools, and paid-rail context.',
  alternates: { canonical: 'https://satgate.io/compare/litellm' },
  keywords: [
    'SatGate vs LiteLLM',
    'LiteLLM alternative',
    'AI gateway budget enforcement',
    'LLM gateway vs economic firewall',
    'LiteLLM spend tracking',
    'AI agent cost control',
    'MCP budget enforcement',
    'economic firewall for AI agents',
  ],
  openGraph: {
    title: 'SatGate vs LiteLLM - AI Gateway vs Economic Firewall',
    description: 'LiteLLM gives teams model access, routing, fallbacks, budgets, and spend tracking. SatGate governs agent/API economics in the request path.',
    url: 'https://satgate.io/compare/litellm',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate vs LiteLLM - LLM Gateway vs Economic Firewall',
    description: 'LiteLLM routes model calls. SatGate enforces agent budgets, MCP tool costs, scoped authority, audit, and paid-rail context.',
  },
};

const rows: Array<[string, string, string]> = [
  ['Primary job', 'Policy-to-Proof governance for enterprise agents', 'LLM gateway / OpenAI-compatible proxy'],
  ['Best fit', 'Agent/API spend governance, MCP tool budgets, scoped access, paid-rail governance', 'Model access, provider abstraction, fallbacks, routing, developer LLM access'],
  ['Request-path hard budget enforcement', 'Yes', 'Partial: budgets and rate limits for LLM gateway usage'],
  ['MCP tool budget enforcement', 'Yes', 'No native MCP economic firewall focus'],
  ['Per-tool cost attribution beyond LLM calls', 'Yes', 'Primarily LLM/model spend tracking'],
  ['Scoped revocable agent capabilities', 'Yes', 'Virtual keys for LLM gateway access'],
  ['L402 paid-agent API payments', 'Yes', 'No'],
  ['100+ LLM provider abstraction', 'No', 'Yes'],
  ['LLM fallbacks/load balancing', 'Partial', 'Yes'],
  ['Open-source/self-hostable', 'Yes', 'Yes'],
];

const satgateWins: Array<{ icon: typeof ShieldCheck; title: string; body: string }> = [
  {
    icon: ShieldCheck,
    title: 'Economic firewall, not just LLM gateway',
    body: 'SatGate governs whether an agent should access, spend, route, delegate, or pay before upstream APIs and tools execute.',
  },
  {
    icon: Gauge,
    title: 'Budgets for agent workflows',
    body: 'Control spend by tenant, agent, workflow, delegated sub-agent, route, model, MCP tool, session, day, and request.',
  },
  {
    icon: KeyRound,
    title: 'Scoped agent authority',
    body: 'Issue expiring, revocable capabilities constrained by route, tool, budget, calls, expiry, and delegation rules.',
  },
  {
    icon: Zap,
    title: 'Govern paid-rail access',
    body: 'Govern paid-rail context before external agents access APIs, tools, datasets, or premium capabilities at request time.',
  },
];

const litellmWins: Array<{ icon: typeof Route; title: string; body: string }> = [
  {
    icon: Route,
    title: 'Model access and routing',
    body: 'LiteLLM is strong when teams need one OpenAI-compatible interface across many LLM providers, deployments, and models.',
  },
  {
    icon: DollarSign,
    title: 'LLM spend tracking',
    body: 'LiteLLM tracks model usage and spend across keys, users, teams, organizations, and providers inside the LLM gateway layer.',
  },
];

export default function CompareLiteLLMPage() {
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'SatGate vs LiteLLM: AI Gateway vs Economic Firewall',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-05-04',
    mainEntityOfPage: 'https://satgate.io/compare/litellm',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Is SatGate a LiteLLM replacement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Not directly. LiteLLM is primarily an LLM gateway and OpenAI-compatible proxy for model access, routing, fallbacks, budgets, and spend tracking. SatGate is an economic firewall for AI agents, APIs, MCP tools, scoped capabilities, and paid-rail context.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can SatGate and LiteLLM work together?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. LiteLLM can sit behind SatGate as an upstream LLM gateway. SatGate can enforce agent, workflow, route, MCP tool, and budget policy before requests reach LiteLLM or other upstream services.',
        },
      },
      {
        '@type': 'Question',
        name: 'When should I choose SatGate over LiteLLM?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Choose SatGate when the problem is economic governance across autonomous agent/API activity: hard budget enforcement, MCP tool spend, scoped revocation, delegation, Evidence Packs, and request-time monetization.',
        },
      },
      {
        '@type': 'Question',
        name: 'When should I choose LiteLLM?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Choose LiteLLM when the main problem is developer model access: one interface across many LLM providers, model routing, load balancing, fallbacks, virtual keys, and LLM spend tracking.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="max-w-6xl mx-auto px-6 py-16">
        <Link href="/compare" className="mb-10 inline-flex items-center gap-2 text-gray-500 transition hover:text-white">
          <ArrowLeft size={18} /> Back to comparisons
        </Link>

        <div className="mb-12">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-3 py-1 text-xs font-mono text-cyan-300">
            Comparison
          </div>
          <h1 className="mb-6 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            SatGate vs LiteLLM
          </h1>
          <p className="max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            LiteLLM is a strong LLM gateway for model access, routing, fallbacks, budgets, and spend tracking. SatGate is the economic firewall for agent/API activity: budgets, MCP tools, revocation, Evidence Packs, and paid-rail context in the request path.
          </p>
        </div>

        <section className="mb-12 rounded-2xl border border-cyan-800/30 bg-gradient-to-r from-cyan-950/20 to-purple-950/20 p-6">
          <h2 className="mb-3 text-xl font-bold text-white">TL;DR</h2>
          <p className="leading-relaxed text-gray-300">
            Use <strong className="text-white">LiteLLM</strong> when you need one gateway for many LLM providers. Use <strong className="text-cyan-300">SatGate</strong> when you need to govern autonomous agents before they spend money, call MCP tools, delegate authority, access protected APIs, or pay for resources.
          </p>
        </section>

        <section className="mb-16 overflow-hidden rounded-2xl border border-gray-800">
          <div className="grid grid-cols-3 bg-gray-950 px-5 py-4 text-sm font-bold uppercase tracking-wide text-gray-400">
            <div>Question</div>
            <div className="text-cyan-300">SatGate</div>
            <div>LiteLLM</div>
          </div>
          {rows.map(([feature, satgate, litellm]) => (
            <div key={feature} className="grid grid-cols-3 gap-4 border-t border-gray-800 px-5 py-4 text-sm text-gray-300">
              <div className="font-medium text-white">{feature}</div>
              <div>{satgate}</div>
              <div className="text-gray-400">{litellm}</div>
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
            <h2 className="mb-6 text-3xl font-bold text-white">Where LiteLLM wins</h2>
            <div className="space-y-4">
              {litellmWins.map(({ icon: Icon, title, body }) => (
                <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-5">
                  <Icon className="mb-3 text-purple-300" size={26} />
                  <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                  <p className="leading-relaxed text-gray-400">{body}</p>
                </div>
              ))}
            </div>

            <div className="mt-6 rounded-xl border border-yellow-900/50 bg-yellow-950/10 p-5">
              <h3 className="mb-2 text-lg font-bold text-white">The clean architecture</h3>
              <p className="leading-relaxed text-gray-300">
                SatGate can sit in front of LiteLLM. SatGate decides whether the agent, workflow, budget, route, and capability are allowed; LiteLLM handles provider routing and model access behind that policy boundary.
              </p>
            </div>
          </div>
        </section>

        <section className="mb-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">Feature signal</h2>
          <div className="grid gap-4 md:grid-cols-2">
            {([
              ['Hard agent budget enforcement', true, 'Budget decisions before expensive agent/API activity executes.'],
              ['MCP tool cost policy', true, 'Prices, limits, risk tiers, and audit evidence per tool call.'],
              ['L402 paid-rail governance', true, 'Request-time paid-rail context for external agent/API access.'],
              ['100+ model provider gateway', false, 'This is LiteLLM territory; SatGate can govern traffic before it reaches that layer.'],
            ] as Array<[string, boolean, string]>).map(([label, yes, body]) => (
              <div key={String(label)} className="rounded-xl border border-gray-800 bg-black p-5">
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
          <h2 className="mb-6 text-3xl font-bold text-white">SatGate vs LiteLLM FAQ</h2>
          <div className="grid gap-5 md:grid-cols-2">
            {[
              ['Is SatGate a LiteLLM replacement?', 'Not directly. LiteLLM is primarily an LLM gateway and OpenAI-compatible proxy for model access, routing, fallbacks, budgets, and spend tracking. SatGate is an economic firewall for AI agents, APIs, MCP tools, scoped capabilities, and paid-rail context.'],
              ['Can SatGate and LiteLLM work together?', 'Yes. LiteLLM can sit behind SatGate as an upstream LLM gateway. SatGate can enforce agent, workflow, route, MCP tool, and budget policy before requests reach LiteLLM or other upstream services.'],
              ['When should I choose SatGate over LiteLLM?', 'Choose SatGate when the problem is economic governance across autonomous agent/API activity: hard budget enforcement, MCP tool spend, scoped revocation, delegation, Evidence Packs, and request-time monetization.'],
              ['When should I choose LiteLLM?', 'Choose LiteLLM when the main problem is developer model access: one interface across many LLM providers, model routing, load balancing, fallbacks, virtual keys, and LLM spend tracking.'],
            ].map(([question, answer]) => (
              <div key={question} className="rounded-xl border border-gray-800 bg-black p-5">
                <h3 className="mb-2 font-bold text-white">{question}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Routing is useful. Governance is different.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            LiteLLM helps developers reach models. SatGate helps platform, finance, and security teams control what autonomous agents can spend, access, delegate, and monetize before the next request leaves the building.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              See Policy-to-Proof <ArrowRight size={18} />
            </Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              See agent cost control
            </Link>
          </div>
        </section>
      </section>
    </main>
  );
}

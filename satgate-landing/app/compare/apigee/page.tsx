import Link from 'next/link';
import { ArrowLeft, ArrowRight, Check, DollarSign, Gauge, KeyRound, Minus, ShieldCheck, Zap } from 'lucide-react';

export const metadata = {
  title: 'SatGate vs Apigee: API Management vs Agent Spending Controls',
  description: 'Compare SatGate and Google Apigee. Apigee is enterprise API management; SatGate adds spending controls for AI agents, MCP tool budgets, and charging external agents.',
  alternates: { canonical: 'https://satgate.io/compare/apigee' },
  keywords: [
    'SatGate vs Apigee',
    'Apigee alternative',
    'Apigee AI agent governance',
    'API management vs rules and receipts for AI agents',
    'AI agent API governance',
    'SatGate comparison',
    'rules and receipts for AI agents',
    'AI agent cost control',
    'MCP budget enforcement',
  ],
  openGraph: {
    title: 'SatGate vs Apigee: API Management vs Agent Spending Controls',
    description: 'Compare SatGate and Google Apigee for API management, AI agent spending controls, MCP tool budgets, revoking access, and charging external agents.',
    url: 'https://satgate.io/compare/apigee',
    type: 'article',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate vs Apigee: API Management vs Agent Spending Controls',
    description: 'Apigee manages APIs. SatGate enforces AI agent budgets, MCP tool costs, scoped credentials, revocation, and charging external agents.',
  },
};

const rows: Array<[string, string, string]> = [
  ['Primary job', 'Rules and receipts for AI agents', 'Enterprise API management, API products, developer portals, analytics, policy, monetization, and Google Cloud integration'],
  ['Best fit', 'Agent/API spend governance, MCP tool budgets, scoped credentials, revocation, signed receipts, and charging external agents', 'Enterprise API management, API products, developer portals, analytics, policy, monetization, and Google Cloud integration'],
  ['Hard budgets before the request goes through', 'Yes: at the gateway before forwarding to an upstream API, model, or MCP tool', 'Partial / depends on gateway policy and traffic type'],
  ['MCP tool budget enforcement', 'Yes: per-tool budgets, cost attribution, and deny decisions', 'Not the primary category focus'],
  ['Scoped revocable agent capabilities', 'Yes: route, tool, call, budget, expiry, delegation, and revocation caveats', 'Typically API keys, policies, tokens, or platform auth building blocks'],
  ['Runaway agent spend benchmark/data', 'Yes: benchmark page plus JSON/CSV dataset', 'No direct equivalent'],
  ['L402 paid-agent API payments', 'Yes: checks payment details before access and preserves a signed receipt (Evidence Pack)', 'No native SatGate-style payment rules focus'],
  ['Broad API/AI platform management', 'Focused on spending controls', 'Yes / stronger fit'],
];

const satgateWins = [
  { icon: ShieldCheck, title: 'Rules and Receipts', body: 'SatGate decides whether an autonomous agent can spend, access, delegate, route, revoke, or pay before the next request executes.' },
  { icon: Gauge, title: 'Budgets beyond LLM tokens', body: 'Enforce cost controls across APIs, MCP tools, models, routes, workflows, tenants, agents, and delegated sub-agents.' },
  { icon: KeyRound, title: 'Scoped, revocable authority', body: 'Replace broad static keys with expiring capabilities constrained by route, tool, budget, calls, expiry, and delegation.' },
  { icon: Zap, title: 'Govern payment access', body: 'Check payment details before external agents access APIs, tools, datasets, or premium capabilities at request time.' },
];

const competitorWins = [
  { icon: Check, title: 'Enterprise API management', body: 'Apigee is built for API products, lifecycle management, developer programs, analytics, security policy, and large enterprise API operations.' },
  { icon: Check, title: 'Google Cloud ecosystem', body: 'Apigee is a natural fit for organizations standardized on Google Cloud API management and enterprise governance workflows.' },
];

export default function ComparePage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'SatGate vs Apigee: API Management vs Agent Spending Controls',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-05-04',
    mainEntityOfPage: 'https://satgate.io/compare/apigee',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'Is SatGate a Google Apigee replacement?', acceptedAnswer: { '@type': 'Answer', text: 'Not directly. Apigee is a full enterprise API management platform. SatGate is rules and receipts for AI agents: API spend, MCP tools, scoped capabilities, revocation, signed receipts, and payment details.' } },
      { '@type': 'Question', name: 'Can SatGate and Google Apigee work together?', acceptedAnswer: { '@type': 'Answer', text: 'Yes. SatGate can sit in front of or alongside gateway, API management, or observability infrastructure to enforce agent economics at the gateway before forwarding.' } },
      { '@type': 'Question', name: 'When should I choose SatGate?', acceptedAnswer: { '@type': 'Answer', text: 'Choose SatGate when the core problem is spending controls for AI agents: hard budgets, MCP tool spend, revocable credentials, permissions passed down to a sub-agent, signed receipts, and paid-agent payment.' } },
      { '@type': 'Question', name: 'When should I choose Google Apigee?', acceptedAnswer: { '@type': 'Answer', text: 'Choose Apigee when the primary need is broad enterprise API management across human and application consumers.' } },
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
          <h1 className="mb-5 text-5xl font-extrabold tracking-tight md:text-7xl">SatGate vs Google Apigee</h1>
          <p className="text-xl leading-relaxed text-gray-300 md:text-2xl">Apigee is a full enterprise API management platform. SatGate is different: it is the rules-and-receipts layer for AI agents, checked before the request goes through, covering API spend, MCP tools, scoped credentials, signed receipts, and payment details.</p>
        </div>

        <section className="mb-14 overflow-hidden rounded-2xl border border-gray-800">
          <div className="grid md:grid-cols-3 bg-gray-900/70 text-sm font-bold text-white"><div className="p-4">Capability</div><div className="p-4">SatGate</div><div className="p-4">Google Apigee</div></div>
          {rows.map(([capability, satgate, competitor]) => (
            <div key={capability} className="grid md:grid-cols-3 border-t border-gray-800 text-gray-300"><div className="p-4 font-semibold text-white">{capability}</div><div className="p-4">{satgate}</div><div className="p-4">{competitor}</div></div>
          ))}
        </section>

        <section className="grid gap-8 lg:grid-cols-2">
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6"><h2 className="mb-5 text-2xl font-bold text-white">Where SatGate wins</h2><div className="space-y-4">{satgateWins.map(({ icon: Icon, title, body }) => (<div key={title} className="rounded-xl border border-gray-800 bg-black p-5"><Icon className="mb-3 text-cyan-300" size={24} /><h3 className="mb-2 font-bold text-white">{title}</h3><p className="text-sm leading-relaxed text-gray-400">{body}</p></div>))}</div></div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6"><h2 className="mb-5 text-2xl font-bold text-white">Where Google Apigee wins</h2><div className="space-y-4">{competitorWins.map(({ icon: Icon, title, body }) => (<div key={title} className="rounded-xl border border-gray-800 bg-black p-5"><Icon className="mb-3 text-green-300" size={24} /><h3 className="mb-2 font-bold text-white">{title}</h3><p className="text-sm leading-relaxed text-gray-400">{body}</p></div>))}</div></div>
        </section>

        <section className="mt-14 rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">SatGate vs Apigee FAQ</h2>
          <div className="grid gap-5 md:grid-cols-2">
            {[
              ['Is SatGate a Google Apigee replacement?', 'Not directly. Apigee is a full enterprise API management platform. SatGate is rules and receipts for AI agents: API spend, MCP tools, scoped capabilities, revocation, signed receipts, and payment details.'],
              ['Can SatGate and Google Apigee work together?', 'Yes. SatGate can sit in front of or alongside gateway, API management, or observability infrastructure to enforce agent economics at the gateway before forwarding.'],
              ['When should I choose SatGate?', 'Choose SatGate when the core problem is spending controls for AI agents: hard budgets, MCP tool spend, revocable credentials, permissions passed down to a sub-agent, signed receipts, and paid-agent payment.'],
              ['When should I choose Google Apigee?', 'Choose Apigee when the primary need is broad enterprise API management across human and application consumers.'],
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
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Rules and Receipts <ArrowRight size={18} /></Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">AI agent cost control</Link>
          </div>
        </section>
      </section>
    </main>
  );
}

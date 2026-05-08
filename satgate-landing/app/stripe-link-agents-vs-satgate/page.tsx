import Link from 'next/link';
import { ArrowRight, CheckCircle2, CreditCard, Gauge, KeyRound, ShieldCheck, WalletCards, XCircle } from 'lucide-react';

export const metadata = {
  title: 'Stripe Link for Agents vs SatGate',
  description: 'Compare Stripe Link for Agents and SatGate: agent wallets and payment credentials vs request-path economic control, budgets, audit, and L402.',
  alternates: { canonical: 'https://satgate.io/stripe-link-agents-vs-satgate' },
  keywords: [
    'Stripe Link for Agents vs SatGate',
    'Link for Agents',
    'agent payment controls',
    'AI agent wallet governance',
    'economic control plane for AI agents',
    'economic firewall for AI agents',
    'HTTP 402 agents',
    'L402 agent payments',
  ],
  openGraph: {
    title: 'Stripe Link for Agents vs SatGate',
    description: 'Stripe Link helps agents pay. SatGate controls what agents may access, spend, meter, delegate, and monetize before upstream API calls execute.',
    url: 'https://satgate.io/stripe-link-agents-vs-satgate',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Stripe Link for Agents vs SatGate',
    description: 'Wallets authorize payment. Economic firewalls authorize behavior.',
  },
};

const comparison = [
  ['Primary job', 'Give agents payment credentials and approval flows', 'Govern agent/API economic activity in the request path'],
  ['Best fit', 'Purchases on merchant sites and payment-token flows', 'APIs, models, MCP tools, delegated agents, budgets, and monetization'],
  ['Control point', 'Wallet / credential issuance', 'Before upstream API, model, or tool access'],
  ['Budget enforcement', 'User approval and future granular controls', 'Per-agent, route, tool, tenant, workflow, and time-window budgets'],
  ['API metering', 'Not the core product', 'Core Observe capability'],
  ['Payment rail', 'Cards and shared payment tokens', 'SatGate Charge uses L402 Lightning for API monetization'],
  ['Governance question', 'Can this agent pay?', 'Should this agent access, spend, delegate, route, or pay now?'],
];

export default function StripeLinkAgentsVsSatGatePage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Stripe Link for Agents vs SatGate',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-05-01',
    dateModified: '2026-05-05',
    mainEntityOfPage: 'https://satgate.io/stripe-link-agents-vs-satgate',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Is Stripe Link for Agents a competitor to SatGate?',
        acceptedAnswer: { '@type': 'Answer', text: 'Stripe Link for Agents and SatGate operate at different layers. Link gives agents payment credentials and approval flows. SatGate governs request-path access, budgets, metering, revocation, audit, and API monetization.' },
      },
      {
        '@type': 'Question',
        name: 'What is the difference between an agent wallet and an economic firewall?',
        acceptedAnswer: { '@type': 'Answer', text: 'An agent wallet authorizes payment. An economic firewall authorizes behavior: whether an agent may access an API, spend budget, call an MCP tool, delegate authority, or unlock paid access.' },
      },
      {
        '@type': 'Question',
        name: 'Does SatGate Charge use Stripe shared payment tokens?',
        acceptedAnswer: { '@type': 'Answer', text: 'No. SatGate Charge is L402 Lightning-native API monetization. Stripe shared payment tokens are a separate payment-credential flow.' },
      },
      {
        '@type': 'Question',
        name: 'Can companies need both Link and SatGate?',
        acceptedAnswer: { '@type': 'Answer', text: 'Yes. A wallet can help an agent pay at checkout. SatGate helps API providers and enterprises control what agents can access, meter usage, enforce budgets, and charge for API or MCP activity.' },
      },
    ],
  };

  const comparisonJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'Stripe Link for Agents vs SatGate comparison',
    description: 'Layer-by-layer comparison of agent wallet/payment credentials and SatGate request-path economic governance for AI agents.',
    itemListElement: comparison.map(([layer, link, satgate], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: layer,
      description: `Stripe Link for Agents: ${link}. SatGate: ${satgate}.`,
    })),
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Stripe Link for Agents vs SatGate', item: 'https://satgate.io/stripe-link-agents-vs-satgate' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(comparisonJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(34,211,238,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(250,204,21,0.14),transparent_30%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200 mb-8">
            <WalletCards size={16} /> Agent payments are now mainstream
          </div>
          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            Stripe Link for Agents vs SatGate
          </h1>
          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-8">
            Stripe Link helps agents pay. SatGate controls whether agents are allowed to access, spend, meter, delegate, and monetize before upstream API calls execute.
          </p>
          <p className="text-2xl md:text-3xl font-bold text-white max-w-4xl mb-10">
            Wallets authorize payment. Economic firewalls authorize behavior.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/agent-payment-controls" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Build agent payment controls <ArrowRight size={18} />
            </Link>
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Learn the economic firewall
            </Link>
            <Link href="/l402-api-pricing-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-yellow-500 transition">
              Price L402 access
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-2 gap-8">
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <CreditCard className="text-yellow-300 mb-5" size={34} />
          <h2 className="text-2xl font-bold text-white mb-4">What Link for Agents is good at</h2>
          <p className="text-gray-300 leading-relaxed mb-5">
            Link for Agents is a wallet and payment-credential layer. It gives an agent a way to request approval, use one-time cards or shared payment tokens, and complete purchases without exposing the user's underlying payment credentials.
          </p>
          <ul className="space-y-3 text-gray-300">
            {['Agent purchase approval', 'One-time-use payment credentials', 'Shared payment-token flows', 'Purchase history and notifications'].map((item) => (
              <li key={item} className="flex gap-3"><CheckCircle2 className="text-yellow-300 shrink-0 mt-1" size={18} />{item}</li>
            ))}
          </ul>
        </div>
        <div className="rounded-2xl border border-cyan-900/60 bg-cyan-950/10 p-8">
          <ShieldCheck className="text-cyan-300 mb-5" size={34} />
          <h2 className="text-2xl font-bold text-white mb-4">What SatGate is built for</h2>
          <p className="text-gray-300 leading-relaxed mb-5">
            SatGate is the economic control plane in front of APIs, models, MCP tools, and delegated agent workflows. It observes traffic, enforces policy, meters usage, revokes access, and charges for API access when needed.
          </p>
          <ul className="space-y-3 text-gray-300">
            {['Request-path budget enforcement', 'Per-agent and per-tool metering', 'Revocable capability and API access', 'L402 Lightning-native API monetization'].map((item) => (
              <li key={item} className="flex gap-3"><CheckCircle2 className="text-cyan-300 shrink-0 mt-1" size={18} />{item}</li>
            ))}
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">Layer-by-layer comparison</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            Stripe validates the agent payment market. SatGate owns the control plane that decides whether agent economic activity should be allowed before it reaches an upstream API, model, or tool.
          </p>
          <div className="overflow-hidden rounded-2xl border border-gray-800">
            <div className="grid md:grid-cols-3 bg-gray-900/70 text-sm font-bold text-white">
              <div className="p-4">Layer</div><div className="p-4">Stripe Link for Agents</div><div className="p-4">SatGate</div>
            </div>
            {comparison.map(([layer, link, satgate]) => (
              <div key={layer} className="grid md:grid-cols-3 border-t border-gray-800 text-gray-300">
                <div className="p-4 font-semibold text-white">{layer}</div><div className="p-4">{link}</div><div className="p-4">{satgate}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-8">Why payment approval is not enough</h2>
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-5">
          {[
            ['Budgets', 'A paid credential does not prove the agent is within route, tenant, model, or workflow budget.'],
            ['Scope', 'Agents need scoped authority, not broad long-lived access just because a payment method exists.'],
            ['Audit', 'Finance and security need to know which agent, tool, route, policy, and proof were involved.'],
            ['Monetization', 'API providers need request-native pricing and L402 Charge when agents become customers.'],
          ].map(([title, body]) => (
            <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="text-xl font-bold text-white mb-2">{title}</h3>
              <p className="text-gray-400 leading-relaxed">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-2 gap-8">
          <div className="rounded-2xl border border-gray-800 bg-black p-8">
            <Gauge className="text-purple-300 mb-5" size={34} />
            <h2 className="text-2xl font-bold text-white mb-4">Use a wallet when the question is payment</h2>
            <p className="text-gray-300 leading-relaxed">
              If an agent needs to buy from a merchant checkout, a wallet and approval flow is the right layer. It can provide temporary payment credentials without exposing the underlying card.
            </p>
          </div>
          <div className="rounded-2xl border border-cyan-900/60 bg-cyan-950/10 p-8">
            <KeyRound className="text-cyan-300 mb-5" size={34} />
            <h2 className="text-2xl font-bold text-white mb-4">Use SatGate when the question is permission</h2>
            <p className="text-gray-300 leading-relaxed">
              If agents are calling your APIs, models, MCP tools, or delegated workflows, you need policy before access: identity, budget, route, revocation, metering, audit, and optional L402 monetization.
            </p>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
        <h2 className="mb-8 text-3xl font-bold text-white">Stripe Link for Agents and SatGate questions</h2>
        <div className="grid md:grid-cols-2 gap-5">
          {[
            ['Is Stripe Link for Agents a competitor to SatGate?', 'Stripe Link for Agents and SatGate operate at different layers. Link gives agents payment credentials and approval flows. SatGate governs request-path access, budgets, metering, revocation, audit, and API monetization.'],
            ['What is the difference between an agent wallet and an economic firewall?', 'An agent wallet authorizes payment. An economic firewall authorizes behavior: whether an agent may access an API, spend budget, call an MCP tool, delegate authority, or unlock paid access.'],
            ['Does SatGate Charge use Stripe shared payment tokens?', 'No. SatGate Charge is L402 Lightning-native API monetization. Stripe shared payment tokens are a separate payment-credential flow.'],
            ['Can companies need both Link and SatGate?', 'Yes. A wallet can help an agent pay at checkout. SatGate helps API providers and enterprises control what agents can access, meter usage, enforce budgets, and charge for API or MCP activity.'],
          ].map(([question, answer]) => (
            <div key={question} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="text-xl font-bold text-white mb-2">{question}</h3>
              <p className="text-gray-400 leading-relaxed">{answer}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-yellow-950/20 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">Control agent economics before payment becomes risk</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            SatGate gives API teams the request-path layer for Observe, Control, and Charge: meter every agent call, enforce budgets, revoke authority, and monetize with L402 when APIs become products for robot customers.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Grade your controls <ArrowRight size={18} />
            </Link>
            <Link href="/http-402-for-ai-agents" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Understand HTTP 402 for agents
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

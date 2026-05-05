import Link from 'next/link';
import { ArrowRight, BadgeDollarSign, Bot, Braces, CheckCircle2, Network, ShieldCheck, Zap } from 'lucide-react';

export const metadata = {
  title: 'HTTP 402 for AI Agents',
  description: 'A practical guide to HTTP 402 for AI agents: payment challenges, Stripe-style shared payment tokens, L402 Lightning, and SatGate economic firewall policy.',
  alternates: { canonical: 'https://satgate.io/http-402-for-ai-agents' },
  keywords: [
    'HTTP 402 for AI agents',
    '402 payment required AI agents',
    'AI agent payments',
    'shared payment token agents',
    'L402 agent payments',
    'HTTP 402 API monetization',
    'robot customer payments',
    'agent payment controls',
  ],
  openGraph: {
    title: 'HTTP 402 for AI Agents',
    description: 'Understand how HTTP 402 lets APIs quote payment to agents — and why payment challenges still need economic firewall policy.',
    url: 'https://satgate.io/http-402-for-ai-agents',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'HTTP 402 for AI Agents',
    description: 'HTTP 402 is becoming the payment handshake for robot customers. SatGate adds policy, budgets, audit, and L402 Charge.',
  },
};

const flows = [
  ['Card checkout', 'Agent receives a temporary card credential and fills a merchant checkout.', 'Good for web purchases, but not request-native API monetization.'],
  ['Shared payment token', 'Agent receives a payment token for a supported 402 machine-payment flow.', 'Rail-specific; separate from SatGate Charge/L402.'],
  ['L402 Lightning', 'API returns a Lightning-backed 402 challenge and verifies proof before access.', 'SatGate Charge uses this for robot-customer API monetization.'],
  ['Policy-only 402 observation', 'SatGate records and evaluates payment challenges even when another rail completes payment.', 'Useful for audit, deny/allow rules, and spend governance.'],
];

export default function Http402ForAiAgentsPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'HTTP 402 for AI Agents',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-05-01',
    dateModified: '2026-05-02',
    mainEntityOfPage: 'https://satgate.io/http-402-for-ai-agents',
    about: [
      { '@type': 'Thing', name: 'HTTP 402 for AI agents' },
      { '@type': 'Thing', name: 'robot customer payments' },
      { '@type': 'Thing', name: 'L402 Lightning API monetization' },
      { '@type': 'Thing', name: 'AI agent payment policy' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'What is HTTP 402 for AI agents?', acceptedAnswer: { '@type': 'Answer', text: 'HTTP 402 lets an API tell an AI agent that payment is required before access. The response can include a machine-readable challenge describing how to pay.' } },
      { '@type': 'Question', name: 'Is HTTP 402 the same as L402?', acceptedAnswer: { '@type': 'Answer', text: 'No. HTTP 402 is the status code. L402 is a Lightning-based payment and access pattern that uses HTTP 402. SatGate Charge uses L402 Lightning.' } },
      { '@type': 'Question', name: 'How are Stripe shared payment tokens different from L402?', acceptedAnswer: { '@type': 'Answer', text: 'Stripe-style shared payment tokens are a payment-credential method for supported 402 flows. L402 uses Lightning payment proof to unlock scoped API access. They are separate rails.' } },
      { '@type': 'Question', name: 'Why do 402 payment challenges need policy?', acceptedAnswer: { '@type': 'Answer', text: 'A payment challenge tells the agent how to pay, but it does not decide whether the agent should be allowed to spend, which budget applies, whether the route is in scope, or how the event should be audited.' } },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'HTTP 402 for AI Agents', item: 'https://satgate.io/http-402-for-ai-agents' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(250,204,21,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.15),transparent_32%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-yellow-500/30 bg-yellow-950/20 px-4 py-2 text-sm text-yellow-200 mb-8">
            <BadgeDollarSign size={16} /> Payment Required for robot customers
          </div>
          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            HTTP 402 for AI Agents
          </h1>
          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-10">
            HTTP 402 is becoming the handshake between autonomous agents and paid APIs. The protocol can request payment; SatGate decides whether the agent should be allowed to pay, access, and continue.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/l402-agent-payments" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              See L402 agent payments <ArrowRight size={18} />
            </Link>
            <Link href="/agent-payment-controls" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-yellow-500 transition">
              Agent payment controls
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12 items-start">
        <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
          <h2 className="text-3xl font-bold text-white mb-6">What HTTP 402 means in an agent world</h2>
          <p>
            For years, HTTP 402 Payment Required was mostly dormant. AI agents make it useful: a paid API can respond with a machine-readable challenge instead of forcing a human through checkout.
          </p>
          <p>
            That challenge may point to different rails: card-based credentials, shared payment tokens, L402 Lightning invoices, or future protocols. But the payment challenge is not the governance layer.
          </p>
          <p>
            SatGate sits before upstream access and applies policy: identify the agent, estimate cost, enforce budgets, decide whether the rail is allowed, record the challenge, and unlock only scoped access after proof.
          </p>
        </div>
        <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">A 402-aware control plane asks</h3>
          <div className="space-y-3 text-sm">
            {['Which payment method is being requested?', 'Is this route approved for agent payments?', 'Does the agent have budget?', 'Should policy require human approval?', 'Is this L402 Charge, shared payment token, or another rail?', 'What should be audited before forwarding?'].map((item) => (
              <div key={item} className="flex items-start gap-3 rounded-lg border border-gray-800 bg-black/50 p-3">
                <CheckCircle2 className="text-yellow-300 mt-0.5" size={18} />
                <span className="text-gray-300">{item}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">HTTP 402 flow types</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            Treat 402 as a protocol surface, not a single payment system. SatGate Charge is L402 Lightning; other payment-token flows are separate rails that still need governance.
          </p>
          <div className="overflow-hidden rounded-2xl border border-gray-800">
            <div className="grid md:grid-cols-3 bg-gray-900/70 text-sm font-bold text-white">
              <div className="p-4">Flow</div><div className="p-4">How it works</div><div className="p-4">SatGate implication</div>
            </div>
            {flows.map(([flow, how, implication]) => (
              <div key={flow} className="grid md:grid-cols-3 border-t border-gray-800 text-gray-300">
                <div className="p-4 font-semibold text-white">{flow}</div><div className="p-4">{how}</div><div className="p-4">{implication}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-8">Where SatGate fits in a 402 exchange</h2>
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-5">
          {[
            { icon: Bot, title: 'Agent requests', body: 'An agent calls a paid API, model endpoint, MCP tool, or dataset.' },
            { icon: Network, title: 'SatGate evaluates', body: 'Policy checks identity, scope, budget, route, tenant, and allowed payment rail.' },
            { icon: Braces, title: '402 is handled', body: 'SatGate can issue, observe, or audit a payment challenge depending on the route.' },
            { icon: ShieldCheck, title: 'Access is governed', body: 'Only approved, scoped, metered, auditable access proceeds upstream.' },
          ].map(({ icon: Icon, title, body }) => (
            <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <Icon className="text-cyan-300 mb-4" size={28} />
              <h3 className="text-xl font-bold text-white mb-2">{title}</h3>
              <p className="text-gray-400 leading-relaxed">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-2 gap-8">
          <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-8">
            <Zap className="text-yellow-300 mb-5" size={34} />
            <h2 className="text-2xl font-bold text-white mb-4">L402 is SatGate Charge</h2>
            <p className="text-gray-300 leading-relaxed">
              SatGate Charge uses L402 Lightning: an API returns a 402 challenge, the agent pays a Lightning invoice, and payment proof unlocks scoped access. This is SatGate's robot-customer monetization rail.
            </p>
          </div>
          <div className="rounded-2xl border border-cyan-900/60 bg-cyan-950/10 p-8">
            <ShieldCheck className="text-cyan-300 mb-5" size={34} />
            <h2 className="text-2xl font-bold text-white mb-4">Other 402 rails still need policy</h2>
            <p className="text-gray-300 leading-relaxed">
              Stripe-style shared payment tokens, card credentials, and future payment protocols can help agents pay. They do not replace request-path controls for budget, scope, revocation, metering, or audit.
            </p>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-yellow-300">FAQ</p>
          <h2 className="text-3xl font-bold text-white mb-8">HTTP 402 for AI agents questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="text-xl font-bold text-white mb-2">What is HTTP 402 for AI agents?</h3>
              <p className="text-gray-400 leading-relaxed">HTTP 402 lets an API tell an AI agent that payment is required before access. The response can include a machine-readable challenge describing how to pay.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="text-xl font-bold text-white mb-2">Is HTTP 402 the same as L402?</h3>
              <p className="text-gray-400 leading-relaxed">No. HTTP 402 is the status code. L402 is a Lightning-based payment and access pattern that uses HTTP 402. SatGate Charge uses L402 Lightning.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="text-xl font-bold text-white mb-2">How are Stripe shared payment tokens different from L402?</h3>
              <p className="text-gray-400 leading-relaxed">Stripe-style shared payment tokens are a payment-credential method for supported 402 flows. L402 uses Lightning payment proof to unlock scoped API access. They are separate rails.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="text-xl font-bold text-white mb-2">Why do 402 payment challenges need policy?</h3>
              <p className="text-gray-400 leading-relaxed">A payment challenge tells the agent how to pay, but it does not decide whether the agent should be allowed to spend, which budget applies, whether the route is in scope, or how the event should be audited.</p>
            </div>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-8">Related guides</h2>
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[
            ['/stripe-link-agents-vs-satgate', 'Stripe Link for Agents vs SatGate', 'Compare wallet credentials with request-path economic governance.'],
            ['/agent-payment-controls', 'Agent payment controls', 'Policy, budgets, approval, audit, and payment rails for AI agents.'],
            ['/l402-agent-payments', 'L402 agent payments', 'Use SatGate Charge to monetize APIs with Lightning-native robot-customer payments.'],
            ['/robot-customer-payments', 'Robot customer payments', 'Turn autonomous agents into paying API customers.'],
            ['/economic-firewall', 'Economic firewall', 'Control agent access and spend before upstream API calls execute.'],
            ['/l402-api-pricing-calculator', 'L402 API pricing calculator', 'Estimate request-native pricing for agent/API monetization.'],
          ].map(([href, title, body]) => (
            <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 transition hover:border-yellow-500/50 hover:bg-yellow-950/10">
              <h3 className="font-bold text-white mb-2">{title}</h3>
              <p className="text-sm text-gray-400 leading-relaxed">{body}</p>
            </Link>
          ))}
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-yellow-900/60 bg-gradient-to-br from-yellow-950/20 to-cyan-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">Make 402 programmable, governed, and auditable</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            SatGate gives API teams the economic firewall for agent-native payments: Observe every 402 and paid request, Control who may spend, and Charge with L402 Lightning when APIs become robot-customer products.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/l402-api-pricing-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Price L402 API access <ArrowRight size={18} />
            </Link>
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-yellow-500 transition">
              Grade your readiness
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

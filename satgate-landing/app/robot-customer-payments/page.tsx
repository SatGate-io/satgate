import Link from 'next/link';
import { ArrowRight, Bot, CircleDollarSign, KeyRound, Network, ReceiptText, ShieldCheck, Zap } from 'lucide-react';

export const metadata = {
  title: 'Paid Agent Payments for APIs',
  description: 'A practical guide to letting AI agents and paid agents pay for APIs per request with L402, Lightning, scoped access, and request-path Evidence Packs.',
  alternates: { canonical: 'https://satgate.io/paid-agent-payments' },
  keywords: [
    'paid agent payments',
    'paid agents',
    'AI agent payments',
    'machine customer payments',
    'autonomous agent API payments',
    'paid-rail context',
    'HTTP 402 API payments',
    'Lightning API monetization',
    'agent API monetization',
  ],
  openGraph: {
    title: 'Paid Agent Payments for APIs',
    description: 'How API companies can charge autonomous agents at request time with paid-rail context, scoped access, and economic Evidence Packs.',
    url: 'https://satgate.io/paid-agent-payments',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Paid Agent Payments for APIs',
    description: 'Turn autonomous agents into paying API customers with request-path paid-rail context.',
  },
};

const paymentFlow = [
  {
    icon: Bot,
    title: 'Agent discovers value',
    body: 'A paid agent finds a protected API, dataset, model endpoint, or MCP tool while completing a task.',
  },
  {
    icon: CircleDollarSign,
    title: 'API quotes a price',
    body: 'SatGate returns an HTTP 402 challenge with a machine-readable price for the requested capability.',
  },
  {
    icon: Zap,
    title: 'Agent pays with Lightning',
    body: 'The agent or its wallet pays the invoice using L402 instead of waiting for a human checkout flow.',
  },
  {
    icon: KeyRound,
    title: 'Access is scoped',
    body: 'Payment unlocks only the route, capability, quota, or expiry the agent bought — not a permanent broad key.',
  },
  {
    icon: ShieldCheck,
    title: 'Policy still applies',
    body: 'Observe and Control policies can still enforce budget, identity, route, risk tier, and revocation rules.',
  },
  {
    icon: ReceiptText,
    title: 'Revenue is auditable',
    body: 'Each paid request carries agent identity, price, route, policy decision, payment proof, and upstream outcome.',
  },
];

const useCases = [
  'Premium search, crawl, and research APIs agents call only when the answer is worth paying for.',
  'Data enrichment APIs where per-request value is clearer than a monthly seat subscription.',
  'MCP tools exposed to external agents that need pricing before tool execution.',
  'Model endpoints, inference tools, or specialized agents sold per job rather than per user.',
  'Enterprise APIs that want agent access without issuing long-lived static API keys.',
];

export default function RobotCustomerPaymentsPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Paid Agent Payments for APIs',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-05-03',
    mainEntityOfPage: 'https://satgate.io/paid-agent-payments',
    about: [
      { '@type': 'Thing', name: 'paid agent payments' },
      { '@type': 'Thing', name: 'machine customer payments' },
      { '@type': 'Thing', name: 'autonomous agent API payments' },
      { '@type': 'Thing', name: 'HTTP 402 API payments' },
      { '@type': 'Thing', name: 'Lightning API monetization' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is a paid agent?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'A paid agent is autonomous software, usually an AI agent, that can discover an API, evaluate its usefulness, pay for access, and call it without a human checkout step.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do paid agents pay for APIs?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate paid-rail governance uses paid-rail context: an API returns an HTTP 402 payment challenge, the agent pays the invoice, and SatGate verifies proof before forwarding or unlocking access.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why are subscriptions a bad fit for paid agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Subscriptions assume humans pick plans, enter cards, and manage keys. Paid agents need request-native pricing, payment, scope, revocation, and audit evidence at machine speed.',
        },
      },
      {
        '@type': 'Question',
        name: 'Is paid agent monetization only about payments?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. Payments need governance around identity, budgets, scoped access, revocation, routing, and audit. SatGate combines Observe, Control, and Charge in the request path.',
        },
      },
      {
        '@type': 'Question',
        name: 'What is the difference between paid agents and human API customers?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Human API customers can choose plans, enter cards, and manage accounts. Paid agents need machine-readable pricing, L402 payment challenges, scoped access, budget limits, and audit evidence inside the request path.',
        },
      },
      {
        '@type': 'Question',
        name: 'Does SatGate paid-rail governance use L402?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. SatGate paid-rail governance is based on paid-rail context for external agent and API monetization. Payment proof unlocks only the scoped capability, route, quota, or expiry allowed by policy.',
        },
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Paid Agent Payments', item: 'https://satgate.io/paid-agent-payments' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(250,204,21,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.14),transparent_32%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-yellow-500/30 bg-yellow-950/20 px-4 py-2 text-sm text-yellow-200 mb-8">
            <Bot size={16} /> Paid agents are the next API buyers
          </div>
          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            Paid Agent Payments for APIs
          </h1>
          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-10">
            AI agents will not fill out lead forms, wait for sales, or keep a corporate card on file. They need APIs that can quote a price, accept payment, scope access, and produce an Evidence Pack in the request path.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/l402-agent-payments" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              See L402 agent payments <ArrowRight size={18} />
            </Link>
            <Link href="/blog/api-monetization-ai" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-yellow-500 transition">
              API monetization guide
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12 items-start">
        <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
          <h2 className="text-3xl font-bold text-white mb-6">Human billing does not survive agent-native usage</h2>
          <p>
            Most API monetization was designed around humans: signup pages, cards, monthly plans, account dashboards, and API keys copied into code. That works when a developer is the buyer and integration is deliberate.
          </p>
          <p>
            Paid agents behave differently. An agent may need one premium search, one enrichment call, one specialized tool execution, or one dataset lookup while completing a workflow. The value is immediate, contextual, and often small enough that a subscription is absurd.
          </p>
          <p>
            SatGate paid-rail governance makes payment part of the same control plane that observes and governs agent/API activity. The agent pays for a capability, SatGate verifies the proof, and the API gets paid without surrendering control.
          </p>
        </div>
        <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">Robot-customer monetization requires</h3>
          <ul className="space-y-3 text-gray-300">
            {['Machine-readable pricing before access', 'Payment verification before forwarding', 'Scoped, revocable credentials after payment', 'Budget controls so agents cannot overspend', 'Audit trails finance and security can explain'].map((item) => (
              <li key={item} className="rounded-lg border border-gray-800 bg-black/50 p-3">{item}</li>
            ))}
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">The paid agent payment flow</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            The key shift is simple: price, payment, policy, and access happen at request time, not in a separate human billing workflow.
          </p>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {paymentFlow.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 hover:border-yellow-900/70 transition">
                <Icon className="text-yellow-300 mb-4" size={28} />
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-8 items-start">
          <div>
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-yellow-300">Agent wallets are one layer</p>
            <h2 className="text-3xl font-bold text-white mb-5">Wallets help agents pay. SatGate helps APIs govern paid agents.</h2>
            <div className="space-y-4 text-gray-300 text-lg leading-relaxed">
              <p>
                Agent wallets and payment credentials make it easier for autonomous software to complete purchases. API companies still need the control plane around those payments: identity, budgets, scoped access, revocation, metering, and audit.
              </p>
              <p>
                SatGate paid-rail governance is paid-rail context-native API monetization. Stripe-style shared payment tokens and card credentials are separate payment rails; SatGate's durable role is governing agent economic behavior before access.
              </p>
            </div>
          </div>
          <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6">
            <h3 className="text-xl font-bold text-white mb-4">Payment landscape</h3>
            <div className="space-y-3">
              {[
                ['/stripe-link-agents-vs-satgate', 'Stripe Link for Agents vs SatGate'],
                ['/agent-payment-controls', 'Agent payment controls'],
                ['/http-402-for-ai-agents', 'HTTP 402 for AI agents'],
              ].map(([href, title]) => (
                <Link key={href} href={href} className="flex items-center justify-between rounded-lg border border-gray-800 bg-black/50 p-4 text-white transition hover:border-yellow-500/50">
                  <span>{title}</span><ArrowRight size={16} />
                </Link>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-2 gap-8">
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <Network className="text-cyan-300 mb-5" size={34} />
          <h2 className="text-2xl font-bold text-white mb-4">Where robot payments fit</h2>
          <ul className="space-y-4 text-gray-300 leading-relaxed">
            {useCases.map((item) => (
              <li key={item} className="flex gap-3">
                <span className="mt-2 h-2 w-2 rounded-full bg-cyan-300 shrink-0" />
                <span>{item}</span>
              </li>
            ))}
          </ul>
        </div>
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <ShieldCheck className="text-green-300 mb-5" size={34} />
          <h2 className="text-2xl font-bold text-white mb-4">Where governance still matters</h2>
          <p className="text-gray-300 leading-relaxed mb-5">
            Payment alone is not governance. A paid agent can still be the wrong agent, on the wrong route, exceeding its budget, using stale authority, or chaining calls in ways finance cannot explain.
          </p>
          <p className="text-gray-300 leading-relaxed">
            That is why SatGate treats paid agent payments as Charge inside a broader economic control plane: Observe every request, Control risky activity, and Charge only when payment should unlock access.
          </p>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-yellow-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Paid agent payment questions</h2>
          <div className="grid gap-5 md:grid-cols-2 mb-16">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is a paid agent?</h3>
              <p className="text-gray-400 leading-relaxed">
                A paid agent is autonomous software, usually an AI agent, that can discover an API, evaluate its usefulness, pay for access, and call it without a human checkout step.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How do paid agents pay for APIs?</h3>
              <p className="text-gray-400 leading-relaxed">
                SatGate paid-rail governance uses paid-rail context: an API returns an HTTP 402 payment challenge, the agent pays the invoice, and SatGate verifies proof before forwarding or unlocking access.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why are subscriptions a bad fit for paid agents?</h3>
              <p className="text-gray-400 leading-relaxed">
                Subscriptions assume humans pick plans, enter cards, and manage keys. Paid agents need request-native pricing, payment, scope, revocation, and audit evidence at machine speed.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Is paid agent monetization only about payments?</h3>
              <p className="text-gray-400 leading-relaxed">
                No. Payments need governance around identity, budgets, scoped access, revocation, routing, and audit. SatGate combines Observe, Control, and Charge in the request path.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is the difference between paid agents and human API customers?</h3>
              <p className="text-gray-400 leading-relaxed">
                Human API customers can choose plans, enter cards, and manage accounts. Paid agents need machine-readable pricing, L402 payment challenges, scoped access, budget limits, and audit evidence inside the request path.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Does SatGate paid-rail governance use L402?</h3>
              <p className="text-gray-400 leading-relaxed">
                Yes. SatGate paid-rail governance is based on paid-rail context for external agent and API monetization. Payment proof unlocks only the scoped capability, route, quota, or expiry allowed by policy.
              </p>
            </div>
          </div>

          <h2 className="text-3xl font-bold text-white mb-8">Related paid-agent controls</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[
              ['/stripe-link-agents-vs-satgate', 'Stripe Link for Agents vs SatGate', 'Compare agent wallets with request-path economic governance.'],
              ['/agent-payment-controls', 'Agent payment controls', 'Govern approval, budget, audit, payment rails, and access policy.'],
              ['/http-402-for-ai-agents', 'HTTP 402 for AI agents', 'Understand payment challenges for paid agents.'],
              ['/l402-agent-payments', 'L402 agent payments', 'Collect Lightning payment before protected API access.'],
              ['/agent-capability-tokens', 'Agent capability tokens', 'Give paid agents scoped, budgeted, expiring access after proof.'],
              ['/revocable-agent-credentials', 'Revocable agent credentials', 'Revoke paid-agent access when policy, budget, or risk changes.'],
              ['/economic-firewall', 'Economic firewall', 'Observe, Control, and Charge paid-agent requests in one request path.'],
              ['/mcp-budget-enforcement', 'MCP budget enforcement', 'Apply the same budget logic to paid tools and MCP servers.'],
              ['/ai-agent-cost-control', 'AI agent cost control', 'Stop agent overspend before upstream requests execute.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 transition hover:border-yellow-500/50 hover:bg-yellow-950/10">
                <h3 className="font-bold text-white mb-2">{title}</h3>
                <p className="text-sm text-gray-400 leading-relaxed">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900">
        <div className="max-w-5xl mx-auto px-6 py-20 text-center">
          <h2 className="text-4xl font-bold text-white mb-5">Turn APIs into agent-native businesses</h2>
          <p className="text-xl text-gray-300 leading-relaxed mb-8">
            If autonomous agents are going to become API customers, monetization has to move into the same request path as identity, budget, access, and audit.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/monetize" className="inline-flex items-center justify-center gap-2 rounded-lg bg-yellow-300 text-black px-6 py-3 font-bold hover:bg-yellow-200 transition">
              Monetize APIs with SatGate <ArrowRight size={18} />
            </Link>
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Learn the economic firewall
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

import Link from 'next/link';
import { ArrowRight, Bot, Coins, KeyRound, LockKeyhole, ReceiptText, Zap } from 'lucide-react';

export const metadata = {
  title: 'L402 Agent Payments for APIs',
  description: 'Let autonomous agents pay for APIs with L402 Lightning payments. SatGate verifies payment before access and keeps API monetization in the request path.',
  alternates: { canonical: 'https://satgate.io/l402-agent-payments' },
  keywords: [
    'L402 agent payments',
    'robot customer payments',
    'AI agent payments',
    'Lightning API payments',
    'HTTP 402 agent payments',
    'API monetization for AI agents',
    'autonomous agent payments',
    'AI agents paying APIs',
    'machine customer payments',
    'L402 API monetization',
  ],
  openGraph: {
    title: 'L402 Agent Payments for APIs',
    description: 'Let robot customers pay for protected APIs with L402 Lightning challenges, proof verification, and request-path access control.',
    url: 'https://satgate.io/l402-agent-payments',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'L402 Agent Payments for APIs',
    description: 'Use SatGate Charge to let autonomous agents pay for APIs with L402 Lightning before access is unlocked.',
  },
};

const steps = [
  {
    icon: Bot,
    title: 'Agent requests a resource',
    body: 'An autonomous agent calls a protected API, model, dataset, tool, or premium endpoint.',
  },
  {
    icon: LockKeyhole,
    title: 'SatGate returns a 402 challenge',
    body: 'Instead of a subscription flow or sales gate, SatGate issues an L402 payment challenge tied to the request.',
  },
  {
    icon: Zap,
    title: 'Agent pays with Lightning',
    body: 'The robot customer pays the invoice through a wallet such as SaturnZap or another L402-capable client.',
  },
  {
    icon: KeyRound,
    title: 'Proof unlocks access',
    body: 'SatGate verifies payment and releases a scoped credential or forwards the approved request.',
  },
  {
    icon: ReceiptText,
    title: 'Usage is attributed',
    body: 'Every paid request is tied to agent identity, route, price, policy, and settlement evidence.',
  },
  {
    icon: Coins,
    title: 'API revenue becomes native',
    body: 'APIs can charge autonomous agents per request, per tool, per dataset, or per premium capability.',
  },
];

export default function L402AgentPaymentsPage() {
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'L402 Agent Payments for APIs',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-25',
    dateModified: '2026-04-25',
    mainEntityOfPage: 'https://satgate.io/l402-agent-payments',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What are L402 agent payments?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'L402 agent payments let an API return an HTTP 402 payment challenge that an autonomous agent can satisfy with a Lightning payment before receiving access.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why use L402 for robot customers?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Robot customers need programmatic, low-friction payment. L402 combines HTTP 402, Lightning invoices, and access credentials so agents can pay for APIs without manual checkout.',
        },
      },
      {
        '@type': 'Question',
        name: 'Is SatGate Charge the same as Fiat402?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. SatGate Charge is L402 Lightning for request-path agent/API payments. Fiat402 is a separate concept and should not be conflated with Charge.',
        },
      },
      {
        '@type': 'Question',
        name: 'What APIs are good candidates for robot-customer payments?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Premium search, data enrichment, research APIs, MCP tools, datasets, model endpoints, and agent-native services are good L402 candidates when value is tied to each request.',
        },
      },
    ],
  };

  const howToJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'HowTo',
    name: 'How to monetize APIs with L402 agent payments',
    description: 'Use SatGate Charge to issue an HTTP 402 payment challenge, verify Lightning payment proof, and unlock API access for autonomous agents.',
    totalTime: 'PT20M',
    step: [
      { '@type': 'HowToStep', name: 'Protect the API route', text: 'Put SatGate in front of the API, tool, dataset, or premium endpoint that autonomous agents should pay to access.' },
      { '@type': 'HowToStep', name: 'Set a request price', text: 'Define per-request, per-tool, dataset, or premium capability pricing for robot customers.' },
      { '@type': 'HowToStep', name: 'Return an L402 challenge', text: 'When an unpaid agent requests the resource, return an HTTP 402 challenge with a Lightning invoice and access credential.' },
      { '@type': 'HowToStep', name: 'Verify payment proof', text: 'Accept the agent response only after SatGate validates the payment proof and request policy.' },
      { '@type': 'HowToStep', name: 'Forward and audit access', text: 'Unlock the approved request and record agent identity, route, price, proof, and policy outcome.' },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'L402 Agent Payments', item: 'https://satgate.io/l402-agent-payments' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(howToJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(250,204,21,0.17),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.15),transparent_32%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-yellow-500/30 bg-yellow-950/20 px-4 py-2 text-sm text-yellow-200 mb-8">
            <Zap size={16} /> L402 Lightning payments for autonomous agents
          </div>

          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            Let AI Agents Pay for APIs Like Customers
          </h1>

          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-10">
            SatGate Charge uses L402 Lightning payments to let robot customers unlock protected APIs, tools, datasets, and premium capabilities at request time.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/blog/l402-protocol-explained" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Read the L402 guide <ArrowRight size={18} />
            </Link>
            <Link href="/monetize" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-yellow-500 transition">
              Monetize APIs
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12 items-start">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">The next API customer is not always a human</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              AI agents increasingly discover, evaluate, call, and combine APIs on behalf of people and companies. That creates a new customer type: software that can decide an API is worth paying for right now.
            </p>
            <p>
              Traditional API monetization assumes a human signs up, enters a card, picks a plan, stores a key, and then integrates. Robot customers need a request-native flow: ask for resource, receive price, pay, unlock access, continue task.
            </p>
            <p>
              L402 gives that flow an internet-native shape. SatGate puts it in front of APIs as the Charge mode of the economic control plane.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">Good fits for L402 agent payments</h3>
          <ul className="space-y-3 text-gray-300">
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Premium search, research, or enrichment endpoints.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Datasets agents query occasionally but value highly.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">AI tools sold per use instead of by seat.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">MCP tools exposed to external agent ecosystems.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">APIs where authorization and payment should happen together.</li>
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">How L402 agent payments work with SatGate</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            SatGate sits in the request path. The API stays protected while the agent receives a machine-readable price and payment challenge.
          </p>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {steps.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 hover:border-yellow-900/70 transition">
                <Icon className="text-yellow-300 mb-4" size={28} />
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-8">Robot-customer payment requirements</h2>
        <div className="grid md:grid-cols-2 gap-5 mb-16">
          {[
            ['Machine-readable price', 'Agents need a price and payment challenge in the protocol flow, not a human checkout page or sales form.'],
            ['Payment before access', 'SatGate verifies L402 Lightning payment proof before forwarding the protected API request upstream.'],
            ['Scoped unlocks', 'Payment should unlock the requested route, tool, dataset, or capability — not a broad reusable API key.'],
            ['Revenue audit trail', 'Every paid request should record agent identity, route, price, proof, settlement evidence, and access decision.'],
          ].map(([title, body]) => (
            <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="text-xl font-bold text-white mb-2">{title}</h3>
              <p className="text-gray-400 leading-relaxed">{body}</p>
            </div>
          ))}
        </div>

        <h2 className="text-3xl font-bold text-white mb-8">A robot-customer request flow</h2>
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6 overflow-x-auto">
          <pre className="text-sm text-gray-300"><code>{`# Agent attempts a protected API call
sz fetch https://api.example.com/premium-research --max-sats 25

# SatGate replies with an L402 challenge
HTTP/1.1 402 Payment Required
WWW-Authenticate: L402 invoice=lnbc..., macaroon=...

# Agent pays, presents proof, and receives the resource
HTTP/1.1 200 OK
{ "answer": "paid premium result" }`}</code></pre>
        </div>
        <p className="text-gray-400 mt-5 leading-relaxed">
          In this pattern, the merchant runs SatGate. The robot customer runs a wallet/client. Payment, authorization, and audit happen before upstream access.
        </p>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-3 gap-6">
          <div className="rounded-2xl border border-gray-800 bg-black p-6">
            <div className="text-cyan-300 font-mono text-sm mb-3">OBSERVE</div>
            <h3 className="text-xl font-bold text-white mb-3">See agent demand</h3>
            <p className="text-gray-400 leading-relaxed">Understand which agents and routes create monetizable demand before forcing a pricing model.</p>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-black p-6">
            <div className="text-purple-300 font-mono text-sm mb-3">CONTROL</div>
            <h3 className="text-xl font-bold text-white mb-3">Protect access</h3>
            <p className="text-gray-400 leading-relaxed">Limit which agents, routes, prices, budgets, and proofs are accepted before unlocking protected resources.</p>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-black p-6">
            <div className="text-yellow-300 font-mono text-sm mb-3">CHARGE</div>
            <h3 className="text-xl font-bold text-white mb-3">Collect per request</h3>
            <p className="text-gray-400 leading-relaxed">Use L402 Lightning payments to charge autonomous agents for API access at the moment of demand.</p>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">Related robot-payment guides</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[
              ['/robot-customer-payments', 'Robot customer payments', 'How autonomous agents become paying API customers in the request path.'],
              ['/agent-capability-tokens', 'Agent capability tokens', 'Scope paid access with route, budget, expiry, delegation, and revocation caveats.'],
              ['/revocable-agent-credentials', 'Revocable agent credentials', 'Revoke or narrow paid agent access before the next request.'],
              ['/blog/l402-protocol-explained', 'L402 protocol explained', 'How HTTP 402, Lightning, and macaroons enable API payments.'],
              ['/blog/api-monetization-ai', 'API monetization for AI', 'How to price, meter, and collect from machine consumers.'],
              ['/economic-firewall', 'Economic firewall', 'Observe, Control, and Charge in one request-path control plane.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 transition hover:border-yellow-500/50 hover:bg-yellow-950/10">
                <h3 className="font-bold text-white mb-2">{title}</h3>
                <p className="text-sm text-gray-400 leading-relaxed">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-yellow-900/60 bg-gradient-to-br from-yellow-950/20 to-cyan-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">Turn protected APIs into robot-customer products</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            SatGate gives API teams a request-path control plane for observing, controlling, and charging AI agents. Charge is L402 Lightning: simple, programmatic, and built for autonomous access.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/pay" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Try the payment demo <ArrowRight size={18} />
            </Link>
            <Link href="/blog/http-402-payment-required-use-cases" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-yellow-500 transition">
              HTTP 402 use cases
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

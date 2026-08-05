import Link from 'next/link';
import { ArrowRight, Bot, Coins, KeyRound, LockKeyhole, ReceiptText, Zap } from 'lucide-react';

export const metadata = {
  title: 'L402 Paid-Rail Governance for APIs',
  description: 'Understand L402 as one paid rail for agent/API access, governed by bounded economic authority, Policy-to-Proof, and Evidence Pack receipts.',
  alternates: { canonical: 'https://satgate.io/l402-agent-payments' },
  keywords: [
    'L402 paid-rail governance',
    'paid-rail agent governance',
    'AI agent payment governance',
    'Lightning API payments',
    'HTTP 402 payment governance',
    'API monetization for AI agents',
    'delegated agent payment proof',
    'agents consuming paid APIs under delegated authority',
    'machine payment rail governance',
    'L402 paid rail governance',
  ],
  openGraph: {
    title: 'L402 Paid-Rail Governance for APIs',
    description: 'Understand L402 as one paid rail for protected API access, governed by SatGate policy and Evidence Packs.',
    url: 'https://satgate.io/l402-agent-payments',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'L402 Paid-Rail Governance for APIs',
    description: 'L402 can carry Lightning payment proof for agent/API access. SatGate checks the action, budget, rail, and receipt at the gateway before forwarding.',
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
    title: 'Payment proof is presented',
    body: 'A wallet, platform, or delegated payment primitive satisfies the invoice through SaturnZap or another L402-capable client.',
  },
  {
    icon: KeyRound,
    title: 'Proof unlocks access',
    body: 'SatGate verifies payment and releases a scoped credential or forwards the approved request.',
  },
  {
    icon: ReceiptText,
    title: 'Usage is attributed',
    body: 'Every paid request is tied to delegated authority, agent identity, route, price, policy, and settlement evidence.',
  },
  {
    icon: Coins,
    title: 'Proof becomes request-native',
    body: 'Each paid request can carry identity, policy, price, payment proof, and access decision.',
  },
];

export default function L402AgentPaymentsPage() {
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'L402 Paid-Rail Governance for APIs',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-25',
    dateModified: '2026-05-03',
    mainEntityOfPage: 'https://satgate.io/l402-agent-payments',
    about: [
      { '@type': 'Thing', name: 'L402 paid-rail governance' },
      { '@type': 'Thing', name: 'paid-rail agent governance' },
      { '@type': 'Thing', name: 'Lightning API payments' },
      { '@type': 'Thing', name: 'HTTP 402 payment governance' },
      { '@type': 'Thing', name: 'Policy-to-Proof for paid agent access' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is L402 paid-rail governance?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'L402 paid-rail governance lets an API return an HTTP 402 challenge while SatGate verifies delegated authority, budget, scope, and payment proof before access.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why use L402 for agent/API paid access?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Delegated agent/API access needs programmatic, low-friction payment proof. L402 combines HTTP 402, Lightning invoices, and access credentials while SatGate keeps authority, budget, and scope explicit.',
        },
      },
      {
        '@type': 'Question',
        name: 'Is L402 enough to govern paid agent access?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. L402 is a payment/access rail. SatGate Policy-to-Proof governs authority, scope, budget, revocation, and Evidence Pack evidence.',
        },
      },
      {
        '@type': 'Question',
        name: 'What APIs are good candidates for agent/API paid access?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Premium search, data enrichment, research APIs, MCP tools, datasets, model endpoints, and delegated agent services are good L402 candidates when value is tied to each request.',
        },
      },
      {
        '@type': 'Question',
        name: 'How is L402 different from a normal API subscription?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'A normal API subscription requires a human signup, plan, billing account, and long-lived key. L402 lets a delegated client receive a machine-readable HTTP 402 challenge, present payment proof, and unlock scoped access for the specific request.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can paid-rail context include budget and access policy?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. SatGate can combine L402 payment proof with request-path policy for identity, route, tool, quota, expiry, revocation, and audit so paid access is still governed.',
        },
      },
    ],
  };

  const howToJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'HowTo',
    name: 'How to monetize APIs with L402 paid-rail governance',
    description: 'Use SatGate to govern an HTTP 402/L402 paid-access flow, verify Lightning payment proof, preserve receipts, and unlock scoped API access for delegated agents.',
    totalTime: 'PT20M',
    step: [
      { '@type': 'HowToStep', name: 'Protect the API route', text: 'Put SatGate in front of the API, tool, dataset, or premium endpoint that delegated agents may consume under paid-access policy.' },
      { '@type': 'HowToStep', name: 'Set a request price', text: 'Define per-request, per-tool, dataset, or premium capability pricing for agent/API paid access.' },
      { '@type': 'HowToStep', name: 'Return an L402 challenge', text: 'When a delegated agent requests the resource, return an HTTP 402 challenge with a Lightning invoice and access credential.' },
      { '@type': 'HowToStep', name: 'Verify payment proof', text: 'Accept the response only after SatGate validates payment proof and request policy.' },
      { '@type': 'HowToStep', name: 'Forward and audit access', text: 'Unlock the approved request and record agent identity, route, price, proof, and policy outcome.' },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'L402 Paid-Rail Governance', item: 'https://satgate.io/l402-agent-payments' },
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
            <Zap size={16} /> paid-rail context for delegated agents
          </div>

          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            L402 Agent Payments, Governed Before Access
          </h1>

          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-10">
            paid-rail context can carry HTTP 402 payment proof. SatGate gives delegated clients bounded economic authority: it decides whether a human or platform delegated enough authority, unlocks only scoped access, and preserves proof for every paid action.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Govern L402 access <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-yellow-500 transition">
              See Policy-to-Proof
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12 items-start">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">The next API call may require delegated paid access</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              AI agents increasingly call and combine APIs on behalf of people and companies. That creates a delegation problem: software may consume valuable APIs, but humans and platforms still need to define what is allowed and prove why.
            </p>
            <p>
              Traditional API monetization assumes a human signs up, enters a card, picks a plan, stores a key, and then integrates. Delegated agent/API access needs a request-native flow: request resource, receive price, present proof, unlock scoped access, continue task.
            </p>
            <p>
              L402 gives that flow an internet-native shape. SatGate treats it as paid-rail context governed by request-path policy and proof.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">Good fits for L402 paid-rail governance</h3>
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
          <h2 className="text-3xl font-bold text-white mb-4">How L402 paid-rail governance works with SatGate</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            SatGate sits in the request path. The API stays protected while the delegated client receives a machine-readable price and payment challenge.
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

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-8 items-start">
          <div>
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-yellow-300">402 is a surface, not one rail</p>
            <h2 className="text-3xl font-bold text-white mb-5">L402, shared payment tokens, and payment credentials are different layers</h2>
            <div className="space-y-4 text-gray-300 text-lg leading-relaxed">
              <p>
                HTTP 402 can carry different payment challenges. Some flows use card credentials or shared payment tokens. paid-rail context is one paid rail for request-native API access. Other paid rails — x402, AgentCore Payments, and Pay.sh — also use HTTP 402 as their surface but settle differently.
              </p>
              <p>
                The important control-plane question is broader than payment: whether the request has delegated authority, budget, scope, and policy approval before paid access is unlocked.
              </p>
            </div>
          </div>
          <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6">
            <h3 className="text-xl font-bold text-white mb-4">Explore the payment-control stack</h3>
            <div className="space-y-3">
              {[
                ['/http-402-for-ai-agents', 'HTTP 402 for AI agents'],
                ['/agent-payment-controls', 'Agent payment controls'],
                ['/stripe-link-agents-vs-satgate', 'Stripe Link for Agents vs SatGate'],
              ].map(([href, title]) => (
                <Link key={href} href={href} className="flex items-center justify-between rounded-lg border border-gray-800 bg-black/50 p-4 text-white transition hover:border-yellow-500/50">
                  <span>{title}</span><ArrowRight size={16} />
                </Link>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-8">Governed L402 payment requirements</h2>
        <div className="grid md:grid-cols-2 gap-5 mb-16">
          {[
            ['Machine-readable price', 'Delegated clients need a price and payment challenge in the protocol flow, not a human checkout page or sales form.'],
            ['Payment before access', 'SatGate verifies paid-rail context payment proof before forwarding the protected API request upstream.'],
            ['Scoped unlocks', 'Payment should unlock the requested route, tool, dataset, or capability — not a broad reusable API key.'],
            ['Evidence Pack receipts', 'Every paid request should record delegated authority, agent identity, route, price, payment proof, policy decision, and Evidence Pack receipt.'],
          ].map(([title, body]) => (
            <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="text-xl font-bold text-white mb-2">{title}</h3>
              <p className="text-gray-400 leading-relaxed">{body}</p>
            </div>
          ))}
        </div>

        <h2 className="text-3xl font-bold text-white mb-8">A governed L402 request flow</h2>
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6 overflow-x-auto">
          <pre className="text-sm text-gray-300"><code>{`# Saturnzap CLI; any L402-capable client can present proof
sz fetch https://api.example.com/premium-research --max-sats 25

# SatGate replies with an L402 challenge
HTTP/1.1 402 Payment Required
WWW-Authenticate: L402 invoice=lnbc..., macaroon=...

# Delegated client presents proof and receives the resource
HTTP/1.1 200 OK
{ "answer": "paid premium result" }`}</code></pre>
        </div>
        <p className="text-gray-400 mt-5 leading-relaxed">
          In this pattern, the merchant runs SatGate. The delegated client or wallet presents payment proof. Payment, authorization, receipt capture, and audit happen at the gateway before forwarding.
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
            <div className="text-yellow-300 font-mono text-sm mb-3">PROVE</div>
            <h3 className="text-xl font-bold text-white mb-3">Preserve the receipt</h3>
            <p className="text-gray-400 leading-relaxed">Use L402 payment proof as one input to a Policy-to-Proof receipt that records authority, rail, price, and decision.</p>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-yellow-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">L402 paid-rail governance questions</h2>
          <div className="grid gap-5 md:grid-cols-2 mb-16">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is L402 paid-rail governance?</h3>
              <p className="text-gray-400 leading-relaxed">
                L402 paid-rail governance lets an API return an HTTP 402 challenge while SatGate verifies delegated authority, budget, scope, and payment proof before access.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why use L402 for agent/API paid access?</h3>
              <p className="text-gray-400 leading-relaxed">
                Delegated agent/API access needs programmatic, low-friction payment proof. L402 combines HTTP 402, Lightning invoices, and access credentials while SatGate keeps authority, budget, and scope explicit.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Is L402 enough to govern paid agent access?</h3>
              <p className="text-gray-400 leading-relaxed">
                No. L402 is a payment/access rail. SatGate Policy-to-Proof governs authority, scope, budget, revocation, and Evidence Pack evidence.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What APIs are good candidates for agent/API paid access?</h3>
              <p className="text-gray-400 leading-relaxed">
                Premium search, data enrichment, research APIs, MCP tools, datasets, model endpoints, and delegated agent services are good L402 candidates when value is tied to each request.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How is L402 different from a normal API subscription?</h3>
              <p className="text-gray-400 leading-relaxed">
                A normal API subscription requires a human signup, plan, billing account, and long-lived key. L402 lets a delegated client receive a machine-readable HTTP 402 challenge, present payment proof, and unlock scoped access for the specific request.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Can paid-rail context include budget and access policy?</h3>
              <p className="text-gray-400 leading-relaxed">
                Yes. SatGate can combine L402 payment proof with request-path policy for identity, route, tool, quota, expiry, revocation, and audit so paid access is still governed.
              </p>
            </div>
          </div>

          <h2 className="text-3xl font-bold text-white mb-8">Related delegated paid-access guides</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[
              ['/http-402-for-ai-agents', 'HTTP 402 for AI agents', 'Compare payment challenges, shared payment tokens, and L402 for autonomous agents.'],
              ['/agent-payment-controls', 'Agent payment governance', 'Govern budgets, approval, payment rails, audit, and access policy.'],
              ['/stripe-link-agents-vs-satgate', 'Stripe Link for Agents vs SatGate', 'See how payment credentials differ from SatGate economic governance.'],
              ['/policy-to-proof', 'Policy-to-Proof', 'See how paid access becomes Evidence Pack proof.'],
              ['/govern', 'Govern AI agents', 'Govern paid agent actions before execution.'],
              ['/agent-capability-tokens', 'Agent capability tokens', 'Scope paid access with route, budget, expiry, delegation, and revocation caveats.'],
              ['/blog/l402-protocol-explained', 'L402 protocol explained', 'How HTTP 402, Lightning, and macaroons enable API payments.'],
              ['/l402-api-pricing-calculator', 'L402 API pricing calculator', 'Estimate per-request agent/API paid-access pricing.'],
              ['/govern', 'AI agent governance', 'Bound delegated agent authority before paid-rail execution.'],
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
          <h2 className="text-3xl font-bold text-white mb-4">Govern L402-paid access with Policy-to-Proof</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            L402 can prove payment. SatGate applies Policy-to-Proof before execution and proves the action was authorized: who acted, what policy applied, which rail was used, what was paid, and why access was allowed or denied.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Govern paid agent actions <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-yellow-500 transition">
              View Policy-to-Proof
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

import Link from 'next/link';
import { ArrowRight, BellRing, CheckCircle2, CreditCard, FileSearch, Gauge, KeyRound, ShieldCheck } from 'lucide-react';

export const metadata = {
  title: 'Agent Payment Governance | Policy Before Paid Access',
  description: 'Agent payment controls combine budgets, policy, scoped authority, revocation, metering, paid-rail context, and Evidence Pack receipts before requests execute.',
  alternates: { canonical: 'https://satgate.io/agent-payment-controls' },
  keywords: [
    'agent payment controls',
    'AI agent payment controls',
    'agent spend approval',
    'AI agent wallet governance',
    'agent payment policy',
    'agent spend control',
    'HTTP 402 agents',
    'paid-rail agent governance',
  ],
  openGraph: {
    title: 'Agent Payment Governance | Policy Before Paid Access',
    description: 'Wallet approval is necessary but not sufficient. SatGate adds request-path budgets, scoped authority, metering, revocation, and Evidence Pack receipts.',
    url: 'https://satgate.io/agent-payment-controls',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Agent Payment Governance | Policy Before Paid Access',
    description: 'Control delegated paid access before it becomes runaway API, MCP, and model spend.',
  },
};

const controls = [
  { icon: KeyRound, title: 'Agent identity', body: 'Know which tenant, agent, workflow, delegated sub-agent, route, and token caused the economic action.' },
  { icon: Gauge, title: 'Budgets', body: 'Enforce hard limits by agent, route, model, MCP tool, workflow, tenant, and time window before requests execute.' },
  { icon: ShieldCheck, title: 'Policy', body: 'Allow, deny, meter, require approval, preserve paid context, or revoke based on risk, scope, price, and authority.' },
  { icon: FileSearch, title: 'Evidence Pack receipts', body: 'Record request, cost, payment challenge, policy decision, credential, proof, and upstream outcome.' },
  { icon: CreditCard, title: 'Payment rail awareness', body: 'Understand whether a flow uses card credentials, shared payment tokens, paid-rail context, or another 402 challenge.' },
  { icon: BellRing, title: 'Human approval', body: 'Escalate only the decisions humans should make, instead of turning every agent request into a manual checkpoint.' },
];

export default function AgentPaymentControlsPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Agent Payment Governance | Policy Before Paid Access',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-05-01',
    dateModified: '2026-05-02',
    mainEntityOfPage: 'https://satgate.io/agent-payment-controls',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'What are agent payment controls?', acceptedAnswer: { '@type': 'Answer', text: 'Agent payment controls are the policies, budgets, approvals, Evidence Packs, and request-path enforcement that govern how AI agents spend money or unlock paid API access.' } },
      { '@type': 'Question', name: 'Is payment approval enough for delegated agent access?', acceptedAnswer: { '@type': 'Answer', text: 'No. Payment approval can authorize value movement, but teams also need identity, budgets, scoped access, revocation, API metering, and audit before agent requests execute.' } },
      { '@type': 'Question', name: 'How does SatGate help with agent payment controls?', acceptedAnswer: { '@type': 'Answer', text: 'SatGate sits in the request path to observe agent activity, enforce budgets and policy, preserve paid-rail context, and record receipts before requests execute.' } },
      { '@type': 'Question', name: 'How are HTTP 402 and L402 related to agent payment controls?', acceptedAnswer: { '@type': 'Answer', text: 'HTTP 402 gives APIs a protocol-level way to request payment. L402, x402, shared payment tokens, cards, and enterprise billing are payment rails; agent payment controls decide whether the agent has authority before access is granted.' } },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Agent Payment Controls', item: 'https://satgate.io/agent-payment-controls' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(250,204,21,0.17),transparent_30%),radial-gradient(circle_at_80%_20%,rgba(34,211,238,0.14),transparent_32%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-yellow-500/30 bg-yellow-950/20 px-4 py-2 text-sm text-yellow-200 mb-8">
            <CreditCard size={16} /> Agent payment governance
          </div>
          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            Agent Payment Controls Start With Policy Before Payment
          </h1>
          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-10">
            AI agents can use wallets, cards, shared payment tokens, HTTP 402 challenges, L402, x402, or enterprise billing rails. The control layer is policy: who may spend, how much, on what authority, and with what receipt.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Govern paid access <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-yellow-500 transition">
              See Policy-to-Proof
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12 items-start">
        <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
          <h2 className="text-3xl font-bold text-white mb-6">The payment credential is only one decision</h2>
          <p>
            Agent wallets are useful. They can issue temporary credentials, request approval, and keep the user&apos;s raw payment method away from the agent. But payment approval does not answer whether the request should happen.
          </p>
          <p>
            A company still needs to know which agent is acting, which route or MCP tool it is touching, what the action will cost, whether budget remains, whether scope is valid, and whether the outcome should be allowed, denied, paid, or recorded in the Evidence Pack.
          </p>
          <p>
            SatGate adds that missing request-path layer: observe economic activity, enforce policy and budgets, preserve paid-rail context, and record a receipt before access is granted.
          </p>
        </div>
        <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">Before an agent spends, ask</h3>
          <div className="space-y-3 text-sm">
            {['Who is the agent?', 'What authority does it have?', 'What scoped authority is it using?', 'Does budget remain?', 'Is the payment rail allowed?', 'Should the request be allowed, denied, paid, or recorded in the Evidence Pack?'].map((item) => (
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
          <h2 className="text-3xl font-bold text-white mb-4">The agent payment control stack</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            The right stack separates payment credentials from economic governance. Wallets can authorize payment; SatGate enforces behavior before API, model, and MCP access.
          </p>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {controls.map(({ icon: Icon, title, body }) => (
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
        <h2 className="text-3xl font-bold text-white mb-8">Payment rails SatGate can govern around</h2>
        <div className="grid md:grid-cols-2 gap-5">
          {[
            ['Cards and one-time credentials', 'Useful for merchant checkout. SatGate still governs API and tool access before downstream spend patterns become uncontrolled.'],
            ['Shared payment tokens', 'Useful for some machine-payment 402 flows. Treat them as one rail that still needs request-path policy, scope, and audit.'],
            ['L402 and x402 payment rails', 'Useful payment contexts for agent-access flows. SatGate should preserve the rail, proof, policy decision, and receipt without making the rail the control layer.'],
            ['MCP priced tool calls', 'Agents need budget and policy on tool execution whether the tool charges directly or triggers paid upstream work.'],
          ].map(([title, body]) => (
            <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="text-xl font-bold text-white mb-2">{title}</h3>
              <p className="text-gray-400 leading-relaxed">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">From approval to enforcement</h2>
          <div className="overflow-hidden rounded-2xl border border-gray-800">
            <div className="grid md:grid-cols-3 bg-gray-900/70 text-sm font-bold text-white">
              <div className="p-4">Control</div><div className="p-4">Payment credential layer</div><div className="p-4">Economic Firewall layer</div>
            </div>
            {[
              ['Purchase approval', 'Ask the user to approve a purchase', 'Decide whether policy allows the agent to attempt the spend'],
              ['Credential safety', 'Issue temporary credentials or tokens', 'Scope, revoke, and audit capability after authorization'],
              ['Budgeting', 'May cap specific approved transactions', 'Enforce budgets across routes, tools, models, tenants, and workflows'],
              ['Paid API access', 'Pay a merchant or endpoint', 'Preserve payment context, verify policy, and unlock access only when authority and budget allow it'],
            ].map(([control, wallet, firewall]) => (
              <div key={control} className="grid md:grid-cols-3 border-t border-gray-800 text-gray-300">
                <div className="p-4 font-semibold text-white">{control}</div><div className="p-4">{wallet}</div><div className="p-4">{firewall}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="mb-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">Agent payment controls FAQ</h2>
          <div className="grid gap-5 md:grid-cols-2">
            {[
              ['What are agent payment controls?', 'Agent payment controls are the policies, budgets, approvals, Evidence Packs, and request-path enforcement that govern how AI agents spend money or unlock paid API access.'],
              ['Is payment approval enough for delegated agent access?', 'No. Payment approval can authorize value movement, but teams also need identity, budgets, scoped access, revocation, API metering, and audit before agent requests execute.'],
              ['How does SatGate help with agent payment controls?', 'SatGate sits in the request path to observe agent activity, enforce budgets and policy, preserve paid-rail context, and record receipts before requests execute.'],
              ['How are HTTP 402 and L402 related to agent payment controls?', 'HTTP 402 gives APIs a protocol-level way to request payment. L402, x402, shared payment tokens, cards, and enterprise billing are payment rails; agent payment controls decide whether the agent has authority before access is granted.'],
            ].map(([question, answer]) => (
              <div key={question} className="rounded-xl border border-gray-800 bg-black p-5">
                <h3 className="mb-2 font-bold text-white">{question}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-3xl border border-yellow-900/60 bg-gradient-to-br from-yellow-950/20 to-cyan-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">Put policy before payment</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            SatGate gives teams the Economic Firewall for delegated paid access: request-path metering, spend limits, revocation, paid-rail context, Evidence Pack receipts, and Policy-to-Proof evidence when access is granted.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Govern paid access <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-yellow-500 transition">
              Review Policy-to-Proof
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

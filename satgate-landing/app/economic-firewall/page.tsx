import Link from 'next/link';
import { ArrowRight, CheckCircle2, Shield, Gauge, WalletCards, Activity, KeyRound } from 'lucide-react';

export const metadata = {
  title: 'Economic Firewall for AI Agents',
  description: 'Learn what an economic firewall is and how SatGate controls AI agent access, spend, budgets, and payments before each API request.',
  alternates: { canonical: 'https://satgate.io/economic-firewall' },
  keywords: [
    'economic firewall',
    'economic firewall for AI agents',
    'AI agent spend control',
    'AI agent budget enforcement',
    'economic control plane for AI agents',
    'request-layer cost control',
    'API budget enforcement',
    'agent API governance',
  ],
  openGraph: {
    title: 'Economic Firewall for AI Agents',
    description: 'The request-path control layer for AI agent identity, access, budgets, routing, audit, and L402 payments.',
    url: 'https://satgate.io/economic-firewall',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Economic Firewall for AI Agents',
    description: 'Define the economic firewall category: request-path control for AI agent access, spend, budgets, routing, and payments.',
  },
};

const capabilities = [
  {
    icon: KeyRound,
    title: 'Agent identity',
    body: 'Attribute every call to the tenant, agent, workflow, delegated sub-agent, token, route, and tool behind it.',
  },
  {
    icon: Shield,
    title: 'Access control',
    body: 'Enforce allow, deny, expiry, scope, and revocation before a request reaches the upstream API.',
  },
  {
    icon: Gauge,
    title: 'Spend control',
    body: 'Apply per-agent, per-tool, per-model, per-session, and per-day budgets in the request path.',
  },
  {
    icon: Activity,
    title: 'Observe + audit',
    body: 'Record cost attribution, policy decisions, route choices, revocation events, and request outcomes.',
  },
  {
    icon: WalletCards,
    title: 'Optional payment',
    body: 'Turn protected APIs into agent-native products with Charge/L402 or other payment modes when needed.',
  },
];

export default function EconomicFirewallPage() {
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Economic Firewall for AI Agents',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-25',
    dateModified: '2026-04-25',
    mainEntityOfPage: 'https://satgate.io/economic-firewall',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is an economic firewall?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'An economic firewall is an inline control layer that governs what AI agents can access and spend before each API request reaches the upstream provider.',
        },
      },
      {
        '@type': 'Question',
        name: 'How is an economic firewall different from rate limiting?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Rate limiting counts requests. An economic firewall enforces budgets, costs, revocation, agent identity, tool policy, and payment decisions in the request path.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do AI agents need economic firewalls?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Autonomous agents can loop, delegate, retry, and call paid tools without a human approving each request. Economic firewalls prevent runaway spend and create auditable governance.',
        },
      },
      {
        '@type': 'Question',
        name: 'Is an economic firewall the same as an API gateway?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. An API gateway can route and secure traffic, but an economic firewall adds per-agent cost attribution, budget enforcement, delegated credentials, policy decisions, and optional payment before requests execute.',
        },
      },
    ],
  };

  const definedTermJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'DefinedTerm',
    name: 'Economic firewall',
    description: 'A request-path control layer that governs AI agent access, spend, budgets, routing, audit, and payment before upstream API calls execute.',
    inDefinedTermSet: 'https://satgate.io/economic-firewall',
    url: 'https://satgate.io/economic-firewall',
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Economic Firewall', item: 'https://satgate.io/economic-firewall' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(definedTermJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(34,211,238,0.18),transparent_32%),radial-gradient(circle_at_80%_20%,rgba(168,85,247,0.16),transparent_28%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200 mb-8">
            <Shield size={16} /> Category definition
          </div>

          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-4xl mb-8">
            Economic Firewall for AI Agents
          </h1>

          <p className="text-xl md:text-2xl text-gray-300 max-w-3xl leading-relaxed mb-10">
            An economic firewall controls what autonomous agents can access and spend before each API request reaches the upstream provider.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Grade your readiness <ArrowRight size={18} />
            </Link>
            <Link href="/blog/what-is-an-economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Read the deep-dive
            </Link>
          </div>
        </div>
      </section>

      <section className="border-b border-gray-900 bg-gray-950/40">
        <div className="max-w-6xl mx-auto px-6 py-12">
          <p className="text-sm font-mono uppercase tracking-wide text-cyan-300 mb-3">Definition</p>
          <div className="rounded-2xl border border-cyan-900/50 bg-black/60 p-6 md:p-8">
            <p className="text-2xl md:text-3xl font-bold leading-snug text-white">
              An economic firewall is the request-path control layer that decides whether an AI agent may access, spend, route, or pay before an upstream API call executes.
            </p>
            <p className="mt-5 text-gray-400 text-lg leading-relaxed">
              It extends the API gateway pattern with agent identity, cost attribution, budget enforcement, revocation, audit trails, and payment policy — the pieces autonomous agent traffic needs and traditional routing does not provide.
            </p>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1.1fr_0.9fr] gap-12 items-start">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">The problem: agents spend money at machine speed</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              Traditional API security assumes humans or predictable applications are behind requests. AI agents change the shape of the problem. They plan, retry, delegate, call tools, summarize results, and loop. Every step can create cost.
            </p>
            <p>
              Rate limits can slow traffic. Dashboards can explain yesterday&apos;s bill. Neither can answer the question that matters before a request happens: <strong className="text-white">is this agent allowed to spend this money right now?</strong>
            </p>
            <p>
              Economic firewalls are the missing control plane between autonomous agents and billable APIs. They combine identity, access policy, budget enforcement, observability, audit, provider routing, and optional payment into one request-path decision.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">Economic firewall decision</h3>
          <div className="space-y-3 text-sm">
            {[
              'Who is the agent?',
              'What capability/token is it using?',
              'Is this route allowed?',
              'What will this request cost?',
              'Does budget remain?',
              'Should the request be observed, controlled, charged, routed, or blocked?',
            ].map((item) => (
              <div key={item} className="flex items-start gap-3 rounded-lg border border-gray-800 bg-black/50 p-3">
                <CheckCircle2 className="text-cyan-300 mt-0.5" size={18} />
                <span className="text-gray-300">{item}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">What an economic firewall controls</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            The core is not one feature. It is a request-path governance loop: identify the agent, evaluate policy, enforce spend, record the decision, and optionally collect payment.
          </p>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {capabilities.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 hover:border-cyan-900/70 transition">
                <Icon className="text-cyan-300 mb-4" size={28} />
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="grid lg:grid-cols-3 gap-6">
          <div className="rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Observe</h2>
            <p className="text-gray-300 leading-relaxed">
              Start by measuring agent/API activity without blocking it. Attribute spend by agent, model, route, tool, team, and workflow so finance and engineering can see what is actually happening.
            </p>
          </div>
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Control</h2>
            <p className="text-gray-300 leading-relaxed">
              Move risky paths into hard enforcement. Apply budgets, spend caps, route policy, revocation, expiry, and kill switches before the upstream provider is called.
            </p>
          </div>
          <div className="rounded-2xl border border-yellow-800/50 bg-yellow-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Charge</h2>
            <p className="text-gray-300 leading-relaxed">
              When an API becomes a product for external agents, issue a payment challenge, collect proof, and unlock access. SatGate Charge uses L402 Lightning; Fiat402 is a separate path.
            </p>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">Economic firewall vs traditional controls</h2>
          <div className="overflow-hidden rounded-2xl border border-gray-800">
            <div className="grid md:grid-cols-3 bg-gray-900/70 text-sm font-bold text-white">
              <div className="p-4">Control</div>
              <div className="p-4">What it answers</div>
              <div className="p-4">Where it fails for agents</div>
            </div>
            {[
              ['Rate limiting', 'How many requests?', 'Does not understand money, model cost, tool price, or delegated budgets.'],
              ['Provider billing dashboard', 'What did we spend?', 'Reports after the fact and usually lacks per-agent attribution.'],
              ['Static API keys', 'Who has access?', 'Cannot express scoped budgets, expiry, revocation, delegation, or per-request economics.'],
              ['Economic firewall', 'Should this agent spend/access/route/pay now?', 'Designed for autonomous agent economics in the request path.'],
            ].map(([a, b, c]) => (
              <div key={a} className="grid md:grid-cols-3 border-t border-gray-800 text-gray-300">
                <div className="p-4 font-semibold text-white">{a}</div>
                <div className="p-4">{b}</div>
                <div className="p-4">{c}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">Related economic control-plane topics</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[
              ['/ai-agent-cost-control', 'AI agent cost control', 'Commercial controls for runaway agent spend and budget enforcement.'],
              ['/ai-api-budget-enforcement', 'AI API budget enforcement', 'Hard budget checks before model, tool, or API calls leave the request path.'],
              ['/agent-spending-limits', 'Agent spending limits', 'Spend caps by task, workflow, delegated sub-agent, route, model, and tool.'],
              ['/mcp-cost-control', 'MCP cost control', 'Control paid tool calls, retries, SaaS actions, cloud tasks, and data lookups.'],
              ['/agent-api-governance', 'Agent API governance', 'Identity, delegation, revocation, and audit for autonomous API calls.'],
              ['/l402-agent-payments', 'L402 agent payments', 'Charge robot customers before unlocking protected API access.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
                <h3 className="font-bold text-white mb-2">{title}</h3>
                <p className="text-sm text-gray-400 leading-relaxed">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">SatGate is the economic control plane for AI agents</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            Put SatGate in the request path to observe every agent call, control what agents can spend or access, and charge when APIs become products for robot customers.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              See SatGate governance <ArrowRight size={18} />
            </Link>
            <Link href="/blog/ai-agent-api-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Learn agent cost control
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

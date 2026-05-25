import Link from 'next/link';
import { ArrowRight, CheckCircle2, Shield, Gauge, WalletCards, Activity, KeyRound } from 'lucide-react';

export const metadata = {
  title: 'Economic Firewall for AI Agents',
  description: 'Learn what an economic firewall is and how SatGate governs AI agent authority, spend, paid rails, and revocation before execution, then exports Evidence Pack proof.',
  alternates: { canonical: 'https://satgate.io/economic-firewall' },
  keywords: [
    'economic firewall',
    'economic firewall for AI agents',
    'AI agent spend control',
    'AI agent budget enforcement',
    'economic firewall for AI agents',
    'request-layer cost control',
    'API budget enforcement',
    'agent API governance',
  ],
  openGraph: {
    title: 'Economic Firewall for AI Agents',
    description: 'The request-path control layer for AI agent authority, spend, revocation, audit evidence, and paid-rail context.',
    url: 'https://satgate.io/economic-firewall',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Economic Firewall for AI Agents',
    description: 'Define the economic firewall category: request-path control for AI agent authority, spend, revocation, evidence, and paid rails.',
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
    title: 'Budget and authority limits',
    body: 'Apply per-agent, per-tool, per-model, per-session, and per-day budgets as caveats on scoped authority.',
  },
  {
    icon: Activity,
    title: 'Evidence capture',
    body: 'Record authority chains, policy decisions, denial reasons, revocation events, spend context, and request outcomes for Evidence Pack export.',
  },
  {
    icon: WalletCards,
    title: 'Paid-rail context',
    body: 'Govern paid calls and agent payments across x402, L402, AgentCore Payments, Pay.sh, API-key billing, or enterprise ledgers when value moves.',
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
    dateModified: '2026-05-09',
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
          text: 'An economic firewall is an inline control layer that governs what AI agents can access, how much they can spend, what they can delegate, and which Evidence Pack artifacts are captured before each API request reaches the upstream provider.',
        },
      },
      {
        '@type': 'Question',
        name: 'How is an economic firewall different from rate limiting?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Rate limiting counts requests. An economic firewall enforces scoped authority, budgets, revocation, agent identity, tool policy, denial reasons, and payment context in the request path.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do AI agents need economic firewalls?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Autonomous agents can loop, delegate, retry, and call paid tools without a human approving each request. SatGate denies unauthorized actions before execution and preserves auditable proof afterward.',
        },
      },
      {
        '@type': 'Question',
        name: 'Is an economic firewall the same as an API gateway?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. An API gateway can route and secure traffic, but an economic firewall adds per-agent authority, budget caveats, delegated credentials, denial reasons, revocation proof, and rail-aware payment context before requests execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do I know whether I need an economic firewall?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'You need an economic firewall when agents can call paid models, APIs, MCP tools, or delegated workflows faster than humans can review authority and spend. Start by mapping agent authority, grading readiness, and generating request-path policy for budgets, credentials, denial, revocation, and Evidence Pack proof.',
        },
      },
      {
        '@type': 'Question',
        name: 'What is the first economic firewall control to implement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Start with Observe mode: attribute every request to an agent, workflow, route, tool, and tenant. Then move high-risk routes into Control mode with scoped credentials, hard budgets, denial reasons, revocation, and Evidence Pack capture before governing external paid rails.',
        },
      },
    ],
  };

  const definedTermJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'DefinedTerm',
    name: 'Economic firewall',
    description: 'A request-path control layer that governs AI agent authority, spend, budgets, revocation, audit evidence, and payment context before upstream API calls execute.',
    inDefinedTermSet: 'https://satgate.io/economic-firewall',
    url: 'https://satgate.io/economic-firewall',
  };

  const implementationPathJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'Economic firewall implementation path',
    description: 'A progressive rollout path for moving AI agent traffic from visibility to request-path authority enforcement and proof preservation across paid rails.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Map agent authority',
        description: 'Identify agents, tenants, workflows, routes, models, MCP tools, budgets, caveats, and delegated sub-agents before changing behavior.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Enforce scoped authority',
        description: 'Move risky routes into request-path Control mode with spend caps, scoped credentials, expiry, revocation, and deny decisions.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Preserve proof across paid rails',
        description: 'Record payment context across x402, L402, AgentCore Payments, Pay.sh, API-key billing, or enterprise ledgers when external agents unlock protected resources.',
      },
    ],
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
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(implementationPathJsonLd) }} />
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

          <p className="text-xl md:text-2xl text-gray-300 max-w-3xl leading-relaxed mb-6">
            An economic firewall controls what autonomous agents can access, how much they can spend, what they can delegate, and which Evidence Pack artifacts are captured before each API request reaches the upstream provider.
          </p>
          <p className="max-w-3xl rounded-2xl border border-purple-900/50 bg-purple-950/20 p-5 text-lg leading-relaxed text-purple-100 mb-10">
            Think of this as the legacy SEO/category term. The current SatGate product narrative is Policy-to-Proof governance: authority before execution, Observe/Control/Prove, and Evidence Pack receipts after every agent action.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              See Policy-to-Proof <ArrowRight size={18} />
            </Link>
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Govern AI agents
            </Link>
            <Link href="/mcp-governance" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Govern MCP tools
            </Link>
          </div>
        </div>
      </section>

      <section className="border-b border-gray-900 bg-gray-950/40">
        <div className="max-w-6xl mx-auto px-6 py-12">
          <p className="text-sm font-mono uppercase tracking-wide text-cyan-300 mb-3">Definition</p>
          <div className="rounded-2xl border border-cyan-900/50 bg-black/60 p-6 md:p-8">
            <p className="text-2xl md:text-3xl font-bold leading-snug text-white">
              An economic firewall is the request-path control layer that decides whether an AI agent may access, spend, delegate, route, or pay before an upstream API call executes.
            </p>
            <p className="mt-5 text-gray-400 text-lg leading-relaxed">
              It extends the API gateway pattern with agent identity, scoped authority, cost attribution, budget enforcement, revocation, denial reasons, Evidence Pack capture, and payment context — the pieces autonomous agent traffic needs and traditional routing does not provide.
            </p>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1.1fr_0.9fr] gap-12 items-start">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">The problem: agents exercise authority at machine speed</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              Traditional API security assumes humans or predictable applications are behind requests. AI agents change the shape of the problem. They plan, retry, delegate, call tools, summarize results, and loop. Every step can create cost, move data, or expand authority.
            </p>
            <p>
              Rate limits can slow traffic. Dashboards can explain yesterday&apos;s bill. Neither can answer the question that matters before a request happens: <strong className="text-white">is this agent allowed to take this action right now?</strong>
            </p>
            <p>
              Economic firewalls are the missing control plane between autonomous agents and governed APIs. They combine identity, authority policy, budget enforcement, observability, revocation, provider routing, Evidence Pack capture, and optional payment context into one request-path decision.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">Economic firewall decision</h3>
          <div className="space-y-3 text-sm">
            {[
              'Who is the agent?',
              'What scoped capability is it using?',
              'Is the requested action allowed under policy?',
              'Does authority remain — scope, budget, expiry, revocation?',
              'Should the request be allowed, denied, delegated, paid, or recorded in the Evidence Pack?',
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
            The core is not one feature. It is a request-path governance loop: identify the agent, evaluate policy, enforce scoped authority, record the decision, and preserve proof across paid rails when needed.
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

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-8 items-start">
          <div>
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Paid agent rails validate the category</p>
            <h2 className="text-3xl font-bold text-white mb-5">Why paid agent rails need economic firewalls</h2>
            <div className="space-y-4 text-gray-300 text-lg leading-relaxed">
              <p>
                Paid rails such as x402, L402, AgentCore Payments, Pay.sh, API-key billing, and wallet flows can help value move between agents and services. That is useful, but payment approval is not the same as governing agent behavior.
              </p>
              <p>
                An economic firewall sits earlier in the path. It decides whether an agent may access an API, consume budget, call an MCP tool, delegate authority, or unlock a paid resource before upstream work happens.
              </p>
              <p className="font-semibold text-white">
                Payment rails authorize value movement. Economic firewalls authorize behavior — and preserve the proof.
              </p>
            </div>
          </div>
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
            <h3 className="text-xl font-bold text-white mb-4">Related paid-rail guides</h3>
            <div className="space-y-3">
              {[
                ['/stripe-link-agents-vs-satgate', 'Stripe Link for Agents vs SatGate'],
                ['/agent-payment-controls', 'Agent payment controls'],
                ['/http-402-for-ai-agents', 'HTTP 402 for AI agents'],
              ].map(([href, title]) => (
                <Link key={href} href={href} className="flex items-center justify-between rounded-lg border border-gray-800 bg-black/50 p-4 text-white transition hover:border-cyan-500/50">
                  <span>{title}</span><ArrowRight size={16} />
                </Link>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="grid lg:grid-cols-3 gap-6">
          <div className="rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Observe</h2>
            <p className="text-gray-300 leading-relaxed">
              Start by measuring agent/API activity without blocking it. Attribute authority and spend by agent, model, route, tool, team, and workflow so security, finance, and platform teams can see what is actually happening.
            </p>
          </div>
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Control</h2>
            <p className="text-gray-300 leading-relaxed">
              Move risky paths into hard enforcement. Apply scoped authority, budgets, route policy, revocation, expiry, and kill switches before the upstream provider is called — and record denial reasons when policy blocks a request.
            </p>
          </div>
          <div className="rounded-2xl border border-yellow-800/50 bg-yellow-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Prove</h2>
            <p className="text-gray-300 leading-relaxed">
              Every authority decision — allowed, denied, delegated, revoked, or paid — feeds the Evidence Pack. Payment proves value moved; SatGate proves the agent was allowed to move it.
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
              ['Economic firewall', 'Should this agent access, spend, delegate, or pay right now?', 'Designed for autonomous agent authority and economics in the request path.'],
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

      <section className="max-w-6xl mx-auto px-6 py-20">
        <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Implementation path</p>
        <h2 className="mb-4 text-3xl font-bold text-white">How to roll out an economic firewall</h2>
        <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
          The safe path is progressive: observe real traffic first, enforce scoped authority on risky routes next, then govern external paid access only after identity, audit, revocation, and Evidence Pack capture are working.
        </p>
        <div className="grid gap-5 md:grid-cols-3">
          {[
            ['1', 'Map agent authority', 'Identify agents, tenants, workflows, routes, models, MCP tools, budgets, caveats, and delegated sub-agents before changing behavior.', '/agent-control-plane'],
            ['2', 'Enforce scoped authority', 'Move risky routes into request-path Control mode with spend caps, scoped credentials, expiry, revocation, and deny decisions.', '/agent-spend-policy-template'],
            ['3', 'Preserve proof across paid rails', 'For external agents, capture payment context across x402, L402, AgentCore Payments, Pay.sh, API-key billing, or enterprise ledgers.', '/policy-to-proof'],
          ].map(([step, title, body, href]) => (
            <Link key={step} href={href} className="rounded-2xl border border-gray-800 bg-gray-950 p-6 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
              <div className="mb-5 flex h-10 w-10 items-center justify-center rounded-full bg-cyan-500/15 font-mono text-cyan-200">{step}</div>
              <h3 className="mb-3 text-xl font-bold text-white">{title}</h3>
              <p className="mb-4 leading-relaxed text-gray-400">{body}</p>
              <span className="inline-flex items-center gap-2 text-sm font-semibold text-cyan-300">Open implementation step <ArrowRight size={16} /></span>
            </Link>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Free tools</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Test your economic firewall posture</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            Category definitions are useful, but teams need numbers and enforceable policy. Use these tools to move from risk awareness to request-path controls.
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {[
              ['/economic-firewall-readiness-grader', 'Readiness grader', 'Score identity, budgets, revocation, audit, routing, MCP, and paid-rail governance controls.'],
              ['/roi-calculator', 'ROI calculator', 'Estimate runaway agent spend, ghost cost, payback period, and annual ROI.'],
              ['/agent-spend-policy-template', 'Spend policy template', 'Generate budget, MCP tool, delegation, denial, revocation, and audit policy.'],
              ['/revocable-capability-token-policy-template', 'Capability-token policy', 'Generate scoped, expiring, revocable agent authority with budget caveats.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-black p-5 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">Related economic control-plane topics</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[
              ['/policy-to-proof', 'Policy-to-Proof', 'Turn every mint, delegation, paid call, denial, and revocation into receipts and exportable Evidence Pack proof.'],
              ['/agent-authority-layer', 'Agent Authority & Accountability Layer', 'The rail-neutral authority and signed Evidence Pack proof layer above every payment rail.'],
              ['/govern', 'Govern AI agents', 'Govern internal agents, preserve proof across external rails, and export Evidence Packs.'],
              ['/agent-control-plane', 'Agent control plane', 'Govern enterprise agent authority, delegation lineage, spend, audit, and revocation.'],
              ['/mcp-governance', 'MCP governance', 'Apply budget, revocation, and audit controls to MCP tool calls.'],
              ['/agent-api-governance', 'Agent API governance', 'Identity, delegation, revocation, and audit for autonomous API calls.'],
              ['/ai-agent-cost-control', 'AI agent cost control', 'Commercial controls for runaway agent spend and budget enforcement.'],
              ['/ai-api-budget-enforcement', 'AI API budget enforcement', 'Hard budget checks before model, tool, or API calls leave the request path.'],
              ['/agent-spending-limits', 'Agent spending limits', 'Spend caps by task, workflow, delegated sub-agent, route, model, and tool.'],
              ['/mcp-cost-control', 'MCP cost control', 'Control paid tool calls, retries, SaaS actions, cloud tasks, and data lookups.'],
              ['/agent-payment-controls', 'Agent payment controls', 'Govern wallet approval, budgets, 402 challenges, and paid-rail context.'],
              ['/http-402-for-ai-agents', 'HTTP 402 for AI agents', 'Understand payment challenges, shared payment tokens, and L402.'],
              ['/l402-agent-payments', 'L402 agent payments', 'Preserve payment context before unlocking protected API access.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
                <h3 className="font-bold text-white mb-2">{title}</h3>
                <p className="text-sm text-gray-400 leading-relaxed">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Economic firewall questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            {[
              ['What is an economic firewall?', 'An economic firewall is an inline control layer that governs what AI agents can access, how much they can spend, what they can delegate, and which Evidence Pack artifacts are captured before each API request reaches the upstream provider.'],
              ['How is an economic firewall different from rate limiting?', 'Rate limiting counts requests. An economic firewall enforces scoped authority, budgets, revocation, agent identity, tool policy, denial reasons, and payment context in the request path.'],
              ['Why do AI agents need economic firewalls?', 'Autonomous agents can loop, delegate, retry, and call paid tools without a human approving each request. SatGate denies unauthorized actions before execution and preserves auditable proof afterward.'],
              ['Is an economic firewall the same as an API gateway?', 'No. An API gateway can route and secure traffic, but an economic firewall adds per-agent authority, budget caveats, delegated credentials, denial reasons, revocation proof, and rail-aware payment context before requests execute.'],
              ['How do I know whether I need an economic firewall?', 'You need an economic firewall when agents can call paid models, APIs, MCP tools, or delegated workflows faster than humans can review authority and spend. Start by mapping agent authority, grading readiness, and generating request-path policy for budgets, credentials, denial, revocation, and Evidence Pack proof.'],
              ['What is the first economic firewall control to implement?', 'Start with Observe mode: attribute every request to an agent, workflow, route, tool, and tenant. Then move high-risk routes into Control mode with scoped credentials, hard budgets, denial reasons, revocation, and Evidence Pack capture before governing external paid rails.'],
            ].map(([question, answer]) => (
              <div key={question} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
                <h3 className="mb-2 text-xl font-bold text-white">{question}</h3>
                <p className="leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">SatGate governs agent authority before value moves</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            Put SatGate in the request path to observe every agent call, control what agents can access or spend, preserve Evidence Pack proof across mint, delegation, spend, denial, and revocation, and govern paid rails when value moves.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              See SatGate governance <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Review Policy-to-Proof
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

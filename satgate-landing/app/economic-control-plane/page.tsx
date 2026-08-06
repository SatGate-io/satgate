import Link from 'next/link';
import { ArrowRight, Gauge, KeyRound, Route, ShieldCheck, WalletCards } from 'lucide-react';

export const metadata = {
  title: 'Economic Control Plane for AI Agents',
  description: 'SatGate is the economic control plane for AI agents: request-path authority, budgets, revocation, MCP governance, paid-rail context, and Evidence Pack proof.',
  alternates: { canonical: 'https://satgate.io/economic-control-plane' },
  keywords: [
    'economic control plane for AI agents',
    'AI agent economic control plane',
    'economic firewall',
    'AI agent budget enforcement',
    'AI agent spend control',
    'MCP governance',
    'agent API governance',
    'request-path budget enforcement',
    'AI agent economic governance',
  ],
  openGraph: {
    title: 'Economic Control Plane for AI Agents',
    description: 'The request-path layer that observes, controls, and proves agent/API authority and spend before execution.',
    url: 'https://satgate.io/economic-control-plane',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Economic Control Plane for AI Agents',
    description: 'Define the economic control plane category for AI agents: authority, budgets, revocation, MCP governance, and proof.',
  },
};

const controls = [
  {
    icon: KeyRound,
    title: 'Authority',
    body: 'Bind each request to tenant, customer, workflow, agent, delegated sub-agent, token, route, model, MCP tool, scope, expiry, and revocation state.',
  },
  {
    icon: Gauge,
    title: 'Budgets',
    body: 'Enforce per-request, session, daily, route, model, MCP tool, customer, tenant, and delegated-agent budgets before upstream cost is created.',
  },
  {
    icon: Route,
    title: 'Routing',
    body: 'Allow, deny, downgrade, route, meter, revoke, require payment, or record-only based on policy and remaining budget.',
  },
  {
    icon: ShieldCheck,
    title: 'Proof',
    body: 'Preserve policy inputs, spend context, denial reasons, revocation events, and receipt fields for Evidence Pack review.',
  },
];

export default function EconomicControlPlanePage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Economic Control Plane for AI Agents',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-08-06',
    dateModified: '2026-08-06',
    mainEntityOfPage: 'https://satgate.io/economic-control-plane',
    about: [
      { '@type': 'Thing', name: 'economic control plane for AI agents' },
      { '@type': 'Thing', name: 'AI agent economic control plane' },
      { '@type': 'Thing', name: 'economic firewall' },
      { '@type': 'Thing', name: 'request-path budget enforcement' },
      { '@type': 'Thing', name: 'MCP governance' },
      { '@type': 'Thing', name: 'agent API governance' },
    ],
  };

  const definedTermJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'DefinedTerm',
    name: 'Economic control plane for AI agents',
    description: 'A request-path system that governs AI agent authority, spend, budget, MCP tool access, revocation, routing, paid-rail context, and Evidence Pack proof before execution.',
    inDefinedTermSet: 'https://satgate.io/economic-control-plane',
    url: 'https://satgate.io/economic-control-plane',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is an economic control plane for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'An economic control plane for AI agents is a request-path system that governs agent authority, spend, budgets, tool access, revocation, routing, paid-rail context, and Evidence Pack proof before an autonomous request executes.',
        },
      },
      {
        '@type': 'Question',
        name: 'How is an economic control plane different from an economic firewall?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'The economic control plane is the broader operating layer for agent authority, budgets, routing, revocation, proof, and paid-rail context. The economic firewall is the enforcement boundary inside that layer where allow, deny, route, revoke, meter, or require-payment decisions happen.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do autonomous agents need an economic control plane?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Autonomous agents can retry, loop, delegate, call paid tools, and consume budgets faster than humans can review requests. The economic control plane makes authority and spend decisions before upstream APIs, models, MCP tools, or paid services execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'What should an AI agent economic control plane include?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'It should include agent and tenant identity, scoped authority, per-request and period budgets, MCP tool policy, route policy, revocation, structured denials, payment context when value moves, and receipts that feed Evidence Packs.',
        },
      },
    ],
  };

  const controlPlaneJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent economic control plane functions',
    description: 'The core functions required to observe, control, and prove AI agent/API activity before execution.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Observe',
        description: 'Attribute every request by tenant, customer, workflow, agent, route, model, MCP tool, capability, and spend context.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Control',
        description: 'Enforce scoped authority, budgets, route policy, revocation, denial, routing, metering, and payment requirements before execution.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Prove',
        description: 'Preserve policy inputs, decisions, denial reasons, payment context, and receipt fields for Evidence Pack review.',
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Economic Control Plane', item: 'https://satgate.io/economic-control-plane' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(definedTermJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(controlPlaneJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_0%,rgba(34,211,238,0.16),transparent_32%),radial-gradient(circle_at_85%_12%,rgba(168,85,247,0.14),transparent_28%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <WalletCards size={16} /> Category definition
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            Economic Control Plane for AI Agents
          </h1>
          <p className="mb-6 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            An economic control plane for AI agents is the request-path system that governs authority, spend, budgets, MCP tool access, revocation, routing, paid-rail context, and Evidence Pack proof before autonomous requests execute.
          </p>
          <p className="mb-10 max-w-4xl text-lg leading-relaxed text-gray-400">
            SatGate turns agent/API activity into Observe, Control, and Prove decisions: attribute every request, enforce economic policy before execution, and preserve receipts finance, security, and platform teams can inspect.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Govern AI agents <ArrowRight size={18} />
            </Link>
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Economic firewall definition
            </Link>
            <Link href="/roi-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Estimate avoided spend
            </Link>
          </div>
        </div>
      </section>

      <section className="border-b border-gray-900 bg-gray-950/50">
        <div className="mx-auto grid max-w-6xl gap-6 px-6 py-16 lg:grid-cols-[0.9fr_1.1fr]">
          <div>
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Direct answer</p>
            <h2 className="mb-4 text-3xl font-bold text-white">Economic governance has to happen before execution</h2>
            <p className="text-lg leading-relaxed text-gray-400">
              Agent traffic is not just authenticated traffic. It is delegated, budget-consuming, tool-calling, retrying, sometimes paid traffic. The control plane has to decide whether the next action is allowed before it creates cost or expands authority.
            </p>
          </div>
          <div className="rounded-2xl border border-cyan-900/50 bg-black p-6">
            <div className="grid gap-4 sm:grid-cols-3">
              {[
                ['Observe', 'Attribute agent, tenant, route, model, MCP tool, budget, and paid-rail context.'],
                ['Control', 'Allow, deny, route, meter, revoke, downgrade, or require payment before upstream execution.'],
                ['Prove', 'Emit receipts with policy inputs, decisions, denial reasons, spend context, and Evidence Pack fields.'],
              ].map(([title, body]) => (
                <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-4">
                  <h3 className="mb-2 font-bold text-white">{title}</h3>
                  <p className="text-sm leading-relaxed text-gray-400">{body}</p>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <h2 className="mb-4 text-3xl font-bold text-white">What the control plane controls</h2>
        <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
          The economic control plane is broader than a dashboard and sharper than a traditional gateway. It turns agent requests into pre-execution economic decisions.
        </p>
        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
          {controls.map(({ icon: Icon, title, body }) => (
            <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-6 transition hover:border-cyan-700/70">
              <Icon className="mb-4 text-cyan-300" size={28} />
              <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
              <p className="text-sm leading-relaxed text-gray-400">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">Control plane vs firewall</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Economic control plane vs economic firewall</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            SatGate uses both terms deliberately. The economic control plane is the operating layer. The economic firewall is the enforcement boundary inside it.
          </p>
          <div className="overflow-hidden rounded-2xl border border-gray-800">
            <div className="grid bg-gray-900/80 text-sm font-bold text-white md:grid-cols-3">
              <div className="p-4">Layer</div>
              <div className="p-4">Question it answers</div>
              <div className="p-4">SatGate behavior</div>
            </div>
            {[
              ['Economic control plane', 'How should agent authority, budgets, routing, revocation, paid context, and proof work across the system?', 'Observe traffic, define policy, enforce decisions, preserve receipts, and improve controls over time.'],
              ['Economic firewall', 'Should this specific agent request access, spend, delegate, route, or pay right now?', 'Make the request-path allow, deny, route, meter, revoke, or require-payment decision.'],
              ['Dashboard', 'What happened after usage occurred?', 'Useful context, but too late to stop runaway loops or over-budget requests by itself.'],
            ].map(([layer, question, behavior]) => (
              <div key={layer} className="grid border-t border-gray-800 text-gray-300 md:grid-cols-3">
                <div className="p-4 font-semibold text-white">{layer}</div>
                <div className="p-4">{question}</div>
                <div className="p-4">{behavior}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Implementation path</p>
        <h2 className="mb-4 text-3xl font-bold text-white">Build the economic control plane in layers</h2>
        <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
          Start with attribution, then enforce budgets and authority, then preserve proof across delegated and paid agent workflows.
        </p>
        <div className="grid gap-5 md:grid-cols-3">
          {[
            ['1', 'Map authority and spend', 'Inventory agents, tenants, customers, API keys, MCP tools, routes, budgets, and delegated workflows.', '/agent-api-key-risk-assessment'],
            ['2', 'Generate enforceable policy', 'Create budgets, route policy, tool caps, revocation triggers, denials, and audit fields.', '/agent-spend-policy-template'],
            ['3', 'Preserve Policy-to-Proof evidence', 'Record allow, deny, revoke, delegate, route, meter, and paid decisions as receipt-backed evidence.', '/policy-to-proof'],
          ].map(([step, title, body, href]) => (
            <Link key={step} href={href} className="rounded-2xl border border-gray-800 bg-gray-950 p-6 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
              <div className="mb-5 flex h-10 w-10 items-center justify-center rounded-full bg-cyan-500/15 font-mono text-cyan-200">{step}</div>
              <h3 className="mb-3 text-xl font-bold text-white">{title}</h3>
              <p className="mb-4 leading-relaxed text-gray-400">{body}</p>
              <span className="inline-flex items-center gap-2 text-sm font-semibold text-cyan-300">Open step <ArrowRight size={16} /></span>
            </Link>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Related assets</p>
          <h2 className="mb-4 text-3xl font-bold text-white">The category graph behind SatGate</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            Category ownership needs definitions, tools, templates, implementation guides, and buyer pages. These are the core pages that make the economic control plane concrete.
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {[
              ['/economic-firewall', 'Economic firewall definition', 'The enforcement boundary inside the economic control plane.'],
              ['/ai-agent-cost-control', 'AI agent cost control', 'Commercial buyer page for budgets, spend controls, and request-path enforcement.'],
              ['/mcp-governance', 'MCP governance', 'Govern Model Context Protocol tools with budget, revocation, and proof controls.'],
              ['/agent-api-governance', 'Agent API governance', 'Scoped authority, delegated API access, revocation, and audit.'],
              ['/roi-calculator', 'AI agent ROI calculator', 'Estimate avoided ghost spend and enforcement payback.'],
              ['/economic-firewall-readiness-grader', 'Readiness grader', 'Score whether identity, budgets, revocation, MCP, and proof are ready.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-black p-5 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="mb-4 text-sm leading-relaxed text-gray-400">{body}</p>
                <span className="inline-flex items-center gap-2 text-sm font-semibold text-cyan-300">Open page <ArrowRight size={16} /></span>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
        <h2 className="mb-8 text-3xl font-bold text-white">Economic control plane questions</h2>
        <div className="grid gap-5 md:grid-cols-2">
          {[
            ['What is an economic control plane for AI agents?', 'An economic control plane for AI agents is a request-path system that governs agent authority, spend, budgets, tool access, revocation, routing, paid-rail context, and Evidence Pack proof before an autonomous request executes.'],
            ['How is an economic control plane different from an economic firewall?', 'The economic control plane is the broader operating layer for agent authority, budgets, routing, revocation, proof, and paid-rail context. The economic firewall is the enforcement boundary inside that layer where allow, deny, route, revoke, meter, or require-payment decisions happen.'],
            ['Why do autonomous agents need an economic control plane?', 'Autonomous agents can retry, loop, delegate, call paid tools, and consume budgets faster than humans can review requests. The economic control plane makes authority and spend decisions before upstream APIs, models, MCP tools, or paid services execute.'],
            ['What should an AI agent economic control plane include?', 'It should include agent and tenant identity, scoped authority, per-request and period budgets, MCP tool policy, route policy, revocation, structured denials, payment context when value moves, and receipts that feed Evidence Packs.'],
          ].map(([question, answer]) => (
            <div key={question} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">{question}</h3>
              <p className="leading-relaxed text-gray-400">{answer}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Put economic policy in the request path</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            SatGate observes every agent call, controls what agents can access or spend before execution, and proves decisions afterward with receipt-backed Evidence Packs.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              See SatGate governance <ArrowRight size={18} />
            </Link>
            <Link href="/tools" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Open the tool suite
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

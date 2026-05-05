import Link from 'next/link';
import { ArrowRight, Bot, DollarSign, Gauge, KeyRound, ShieldCheck, Terminal, Workflow } from 'lucide-react';

export const metadata = {
  title: 'SatGate for Cursor AI Agents',
  description: 'Control Cursor agent spend, MCP tool access, and API credentials with SatGate request-path budgets, revocation, and audit trails.',
  alternates: { canonical: 'https://satgate.io/satgate-for-cursor' },
  keywords: [
    'SatGate for Cursor',
    'Cursor agent spend control',
    'Cursor MCP budget enforcement',
    'AI agent cost control',
    'economic firewall for AI agents',
    'revocable agent credentials',
  ],
  openGraph: {
    title: 'SatGate for Cursor AI Agents',
    description: 'Control Cursor agent spend, MCP tool access, and API credentials with SatGate request-path budgets, revocation, and audit trails.',
    url: 'https://satgate.io/satgate-for-cursor',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate for Cursor AI Agents',
    description: 'Control Cursor agent spend, MCP tool access, and API credentials with SatGate request-path budgets, revocation, and audit trails.',
  },
};

const controls = [
  { icon: DollarSign, title: 'Budget before execution', body: 'Check remaining spend, request ceilings, route cost, and policy before forwarding the call.' },
  { icon: KeyRound, title: 'Scoped capabilities', body: 'Replace broad static keys with expiring, revocable credentials constrained by route, tool, calls, and spend.' },
  { icon: ShieldCheck, title: 'Revocation and kill switches', body: 'Stop a risky task, tool, or agent session immediately without rotating every shared secret.' },
  { icon: Workflow, title: 'MCP and API governance', body: 'Apply the same economic policy across MCP servers, internal APIs, model providers, and paid tools.' },
];

export default function SatGateIntegrationPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'SatGate for Cursor AI Agents',
    description: metadata.description,
    url: 'https://satgate.io/satgate-for-cursor',
    dateModified: '2026-05-04',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'Cursor agent spend control' },
      { '@type': 'Thing', name: 'Cursor MCP budget enforcement' },
      { '@type': 'Thing', name: 'economic firewall for coding agents' },
      { '@type': 'Thing', name: 'revocable agent credentials' },
      { '@type': 'Thing', name: 'request-path audit trails' },
    ],
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate for Cursor AI Agents',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Cloud, API Gateway, MCP Proxy',
    description: metadata.description,
    url: 'https://satgate.io/satgate-for-cursor',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-04',
    about: webPageJsonLd.about,
    featureList: ['AI agent spend control', 'MCP budget enforcement', 'Revocable capability tokens', 'Request-path audit trails', 'L402 API monetization'],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Can SatGate control Cursor MCP tools?',
        acceptedAnswer: { '@type': 'Answer', text: 'Yes. SatGate can sit in front of MCP servers or paid API tools used by Cursor and enforce budgets, allowed routes, tool scopes, expiry, and revocation before the tool call executes.' },
      },
      {
        '@type': 'Question',
        name: 'Is SatGate just another observability dashboard?',
        acceptedAnswer: { '@type': 'Answer', text: 'No. SatGate can observe traffic, but its core role is request-path enforcement: budgets, revocation, route policy, capabilities, audit, and L402 payment before upstream access.' },
      },
      {
        '@type': 'Question',
        name: 'Can SatGate start in observe-only mode?',
        acceptedAnswer: { '@type': 'Answer', text: 'Yes. Teams can start with Observe to map agent and tool spend, then graduate to Control policies once safe limits are clear.' },
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'SatGate for Cursor AI Agents', item: 'https://satgate.io/satgate-for-cursor' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_34%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.16),transparent_30%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <Terminal size={16} /> Cursor MCP and agent spend control
          </div>

          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">Give Cursor agents budgets, not blank checks</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">Cursor makes software agents fast enough to call tools, APIs, and model routes at machine speed. SatGate adds the missing economic firewall: per-agent budgets, scoped credentials, MCP tool limits, revocation, and audit before requests execute.</p>

          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Learn the economic firewall <ArrowRight size={18} />
            </Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              AI agent cost control
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1fr_0.9fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">Why Cursor workflows need request-path economics</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>Autonomous agents are not normal SaaS users. They can retry, loop, delegate, and call tools faster than a human operator can review a bill.</p>
            <p>SatGate sits between those agents and the upstream API, MCP server, model provider, or protected resource. Every request gets an economic decision before access is granted.</p>
            <p>That makes agent workflows safer to deploy: start by observing real traffic, then enforce the limits that match each task, team, tenant, or customer.</p>
          </div>
        </div>
        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">What this prevents</h3>
          <ul className="space-y-3 text-gray-300">
              <li className="rounded-lg border border-gray-800 bg-black/60 p-4">Prevent coding loops from silently burning premium model or API budget.</li>
              <li className="rounded-lg border border-gray-800 bg-black/60 p-4">Scope MCP tools and repo-adjacent API calls to the exact task.</li>
              <li className="rounded-lg border border-gray-800 bg-black/60 p-4">Give each agent run its own expiring, revocable capability instead of sharing one static key.</li>
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">SatGate controls for Cursor</h2>
          <p className="mb-10 max-w-3xl text-lg text-gray-400">Use SatGate as the economic control plane around agentic tool use: Observe first, Control when limits are known, Charge when external agents should pay for access.</p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
            {controls.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 transition hover:border-cyan-900/70">
                <Icon className="mb-4 text-cyan-300" size={28} />
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[0.9fr_1.1fr]">
        <div>
          <h2 className="mb-5 text-3xl font-bold text-white">Implementation pattern</h2>
          <p className="text-lg leading-relaxed text-gray-300">You do not need to rewrite every tool. Put SatGate at the gateway, proxy, sidecar, or MCP boundary where economic decisions matter.</p>
        </div>
        <ol className="space-y-5 text-gray-300">
              <li className="flex gap-4"><span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-cyan-400 text-sm font-black text-black">1</span><span>Route Cursor MCP/tool traffic through SatGate or a SatGate-protected proxy.</span></li>
              <li className="flex gap-4"><span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-cyan-400 text-sm font-black text-black">2</span><span>Mint a scoped capability for the workspace, task, model route, or developer.</span></li>
              <li className="flex gap-4"><span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-cyan-400 text-sm font-black text-black">3</span><span>Set per-run budgets, max calls, allowed tools, and expiry.</span></li>
              <li className="flex gap-4"><span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-cyan-400 text-sm font-black text-black">4</span><span>Observe denied/allowed requests with agent, route, tool, and spend attribution.</span></li>
        </ol>
      </section>

      <section className="mx-auto max-w-6xl px-6 pb-20">
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">Cursor governance FAQ</h2>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              ['Can SatGate control Cursor MCP tools?', 'Yes. SatGate can sit in front of MCP servers or paid API tools used by Cursor and enforce budgets, allowed routes, tool scopes, expiry, and revocation before the tool call executes.'],
              ['Is SatGate just another observability dashboard?', 'No. SatGate can observe traffic, but its core role is request-path enforcement: budgets, revocation, route policy, capabilities, audit, and L402 payment before upstream access.'],
              ['Can SatGate start in observe-only mode?', 'Yes. Teams can start with Observe to map agent and tool spend, then graduate to Control policies once safe limits are clear.'],
            ].map(([question, answer]) => (
              <div key={question}>
                <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                <p className="leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 pb-24">
        <div className="rounded-3xl border border-purple-900/40 bg-gradient-to-br from-purple-950/40 to-cyan-950/20 p-8 md:p-10">
          <div className="mb-4 flex items-center gap-3 text-purple-200"><Bot size={24} /><span className="font-semibold">Observe → Control → Charge</span></div>
          <h2 className="mb-4 text-3xl font-bold text-white">Make Cursor agent activity governable.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">SatGate gives agent teams the missing economic layer: budgets, scoped authority, revocation, audit, and L402 payments where machine customers need to pay for APIs.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/mcp-budget-enforcement" className="inline-flex items-center justify-center gap-2 rounded-lg bg-cyan-300 px-6 py-3 font-bold text-black transition hover:bg-cyan-200">
              MCP budget enforcement <Gauge size={18} />
            </Link>
            <Link href="/robot-customer-payments" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-purple-500">
              Robot customer payments
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

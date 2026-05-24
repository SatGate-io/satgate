import Link from 'next/link';
import { ArrowRight, Bot, DollarSign, Gauge, KeyRound, ShieldCheck, Terminal, Workflow } from 'lucide-react';

export const metadata = {
  title: 'SatGate for Claude Desktop MCP',
  description: 'Govern Claude Desktop MCP tool calls with SatGate budgets, scoped capabilities, revocation, and request-path Evidence Packs.',
  alternates: { canonical: 'https://satgate.io/satgate-for-claude-desktop' },
  keywords: [
    'SatGate for Claude Desktop',
    'Claude Desktop agent spend control',
    'Claude Desktop MCP budget enforcement',
    'AI agent cost control',
    'economic firewall for AI agents',
    'revocable agent credentials',
  ],
  openGraph: {
    title: 'SatGate for Claude Desktop MCP',
    description: 'Govern Claude Desktop MCP tool calls with SatGate budgets, scoped capabilities, revocation, and request-path Evidence Packs.',
    url: 'https://satgate.io/satgate-for-claude-desktop',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate for Claude Desktop MCP',
    description: 'Govern Claude Desktop MCP tool calls with SatGate budgets, scoped capabilities, revocation, and request-path Evidence Packs.',
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
    name: 'SatGate for Claude Desktop MCP',
    url: 'https://satgate.io/satgate-for-claude-desktop',
    description: metadata.description,
    datePublished: '2026-04-12',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'Claude Desktop MCP budget enforcement' },
      { '@type': 'Thing', name: 'SatGate for Claude Desktop' },
      { '@type': 'Thing', name: 'MCP tool spend governance' },
      { '@type': 'Thing', name: 'request-path economic firewall' },
      { '@type': 'Thing', name: 'revocable agent credentials for MCP' },
    ],
    audience: { '@type': 'Audience', audienceType: 'AI engineering, platform, API, and security teams using Claude Desktop' },
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate for Claude Desktop MCP',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Cloud, API Gateway, MCP Proxy',
    description: metadata.description,
    url: 'https://satgate.io/satgate-for-claude-desktop',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-03',
    audience: webPageJsonLd.audience,
    featureList: ['AI agent spend control', 'MCP budget enforcement', 'Revocable capability tokens', 'Request-path Evidence Packs', 'rail-neutral paid-rail governance'],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Does SatGate replace MCP?',
        acceptedAnswer: { '@type': 'Answer', text: 'No. MCP connects assistants to tools. SatGate governs the economic and access policy around those tool calls so MCP usage can be budgeted, revoked, and audited.' },
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
      { '@type': 'ListItem', position: 2, name: 'SatGate for Claude Desktop MCP', item: 'https://satgate.io/satgate-for-claude-desktop' },
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
            <Terminal size={16} /> Claude Desktop MCP budget enforcement
          </div>

          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">Put authority controls around Claude Desktop MCP tools</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">Claude Desktop plus MCP gives assistants access to real tools. That is exactly where static keys and best-effort prompts break down. SatGate enforces budget, scope, expiry, revocation, and audit at the request layer around MCP servers and paid APIs.</p>

          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              See Policy-to-Proof <ArrowRight size={18} />
            </Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              AI agent cost control
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1fr_0.9fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">Why Claude Desktop workflows need request-path economics</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>Autonomous agents are not normal SaaS users. They can retry, loop, delegate, and call tools faster than a human operator can review a bill.</p>
            <p>SatGate sits between those agents and the upstream API, MCP server, model provider, or protected resource. Every request gets an economic decision before access is granted.</p>
            <p>That makes agent workflows safer to deploy: start by observing real traffic, then enforce the limits that match each task, team, tenant, or customer.</p>
          </div>
        </div>
        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">What this prevents</h3>
          <ul className="space-y-3 text-gray-300">
              <li className="rounded-lg border border-gray-800 bg-black/60 p-4">Keep local assistant workflows from reaching paid APIs without a budget decision.</li>
              <li className="rounded-lg border border-gray-800 bg-black/60 p-4">Limit MCP tools by task, route, user, tenant, and time window.</li>
              <li className="rounded-lg border border-gray-800 bg-black/60 p-4">Turn every tool call into auditable economic activity instead of invisible automation.</li>
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">SatGate controls for Claude Desktop</h2>
          <p className="mb-10 max-w-3xl text-lg text-gray-400">Use SatGate as the governance layer around agentic tool use: Observe first, Control when limits are known, and Prove every approval, denial, and paid-access decision with Evidence Pack receipts.</p>
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
              <li className="flex gap-4"><span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-cyan-400 text-sm font-black text-black">1</span><span>Proxy sensitive MCP servers or downstream APIs through SatGate.</span></li>
              <li className="flex gap-4"><span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-cyan-400 text-sm font-black text-black">2</span><span>Mint a capability for the Claude Desktop workflow with explicit caveats.</span></li>
              <li className="flex gap-4"><span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-cyan-400 text-sm font-black text-black">3</span><span>Set max calls, max spend, route allowlists, and expiration.</span></li>
              <li className="flex gap-4"><span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-cyan-400 text-sm font-black text-black">4</span><span>Review Observe data before promoting workflows into stricter Control mode.</span></li>
        </ol>
      </section>

      <section className="border-y border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Claude Desktop MCP governance questions</h2>
          <div className="grid gap-5 md:grid-cols-3">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Does SatGate replace MCP?</h3>
              <p className="leading-relaxed text-gray-400">No. MCP connects assistants to tools. SatGate governs the economic and access policy around those tool calls so MCP usage can be budgeted, revoked, and audited.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Is SatGate just another observability dashboard?</h3>
              <p className="leading-relaxed text-gray-400">No. SatGate can observe traffic, but its core role is request-path enforcement: budgets, revocation, route policy, capabilities, audit, and L402 payment before upstream access.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Can SatGate start in observe-only mode?</h3>
              <p className="leading-relaxed text-gray-400">Yes. Teams can start with Observe to map agent and tool spend, then graduate to Control policies once safe limits are clear.</p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-purple-900/40 bg-gradient-to-br from-purple-950/40 to-cyan-950/20 p-8 md:p-10">
          <div className="mb-4 flex items-center gap-3 text-purple-200"><Bot size={24} /><span className="font-semibold">Observe → Control → Prove</span></div>
          <h2 className="mb-4 text-3xl font-bold text-white">Make Claude Desktop agent activity governable.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">SatGate gives agent teams the missing economic layer: budgets, scoped authority, revocation, audit, and paid-rail context where machine customers need to pay for APIs.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/mcp-budget-enforcement" className="inline-flex items-center justify-center gap-2 rounded-lg bg-cyan-300 px-6 py-3 font-bold text-black transition hover:bg-cyan-200">
              MCP budget enforcement <Gauge size={18} />
            </Link>
            <Link href="/paid-agent-payments" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-purple-500">
              Paid agent payments
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

import Link from 'next/link';
import { ArrowRight, Ban, BarChart3, Bot, DollarSign, Gauge, KeyRound, ReceiptText } from 'lucide-react';

export const metadata = {
  title: 'AI API Budget Enforcement | Hard Caps for Agent API Spend',
  description: 'Enforce AI API budgets before agents call OpenAI, Claude, MCP tools, paid APIs, or internal services, with Evidence Pack receipts.',
  alternates: { canonical: 'https://satgate.io/ai-api-budget-enforcement' },
  keywords: [
    'AI API budget enforcement',
    'AI API spend control',
    'agent API budget caps',
    'OpenAI API budget enforcement',
    'Claude API budget enforcement',
    'AI API cost control',
    'economic firewall',
    'SatGate',
    'AI agent cost control',
  ],
  openGraph: {
    title: 'AI API Budget Enforcement | Hard Caps for Agent API Spend',
    description: 'Enforce AI API budgets before agents call OpenAI, Claude, MCP tools, paid APIs, or internal services with Evidence Pack receipts.',
    url: 'https://satgate.io/ai-api-budget-enforcement',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI API Budget Enforcement | Hard Caps for Agent API Spend',
    description: 'Enforce AI API budgets before agents call OpenAI, Claude, MCP tools, paid APIs, or internal services with Evidence Pack receipts.',
  },
};

const controls = [
  { icon: Bot, title: 'Agent identity', body: 'Attribute requests to the tenant, agent, task, workflow, route, model, MCP tool, and delegated sub-agent.' },
  { icon: DollarSign, title: 'Budget checks', body: 'Evaluate remaining spend, per-request ceilings, daily caps, tool caps, and route budgets before forwarding.' },
  { icon: KeyRound, title: 'Scoped credentials', body: 'Use expiring capabilities instead of broad static keys so authority matches the job.' },
  { icon: Ban, title: 'Revocation', body: 'Block the next request when a credential, workflow, route, budget, or agent should stop.' },
  { icon: ReceiptText, title: 'Evidence receipts', body: 'Record allow/deny decisions with policy, budget remaining, route, tool, estimated cost, and outcome.' },
  { icon: BarChart3, title: 'Benchmark risk', body: 'Model loops, retry storms, fanout, detection delay, and avoided spend with benchmark-backed scenarios.' },
];

export default function Page() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'AI API Budget Enforcement | Hard Caps for Agent API Spend',
    url: 'https://satgate.io/ai-api-budget-enforcement',
    description: metadata.description,
    datePublished: '2026-05-01',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI API budget enforcement' },
      { '@type': 'Thing', name: 'agent API spend caps' },
      { '@type': 'Thing', name: 'OpenAI API budget enforcement' },
      { '@type': 'Thing', name: 'MCP tool cost control' },
      { '@type': 'Thing', name: 'request-path economic policy' },
    ],
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'AI API Budget Enforcement',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Cloud, API Gateway, MCP Proxy',
    description: metadata.description,
    url: 'https://satgate.io/ai-api-budget-enforcement',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-03',
    featureList: ['Request-path budget enforcement', 'AI agent spend caps', 'MCP tool cost control', 'Revocable credentials', 'Audit receipts', 'Policy-to-Proof evidence'],
    audience: { '@type': 'Audience', audienceType: 'AI platform, API, finance, and security teams' },
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'What is AI API budget enforcement?', acceptedAnswer: { '@type': 'Answer', text: 'AI API budget enforcement is the request-path control that checks budgets, per-request cost, route policy, tool scope, expiry, and revocation before an autonomous agent can spend against an API or model provider.' } },
      { '@type': 'Question', name: 'Why are dashboards not enough?', acceptedAnswer: { '@type': 'Answer', text: 'Dashboards and billing alerts report spend after requests complete. Autonomous agents can loop, retry, and delegate fast enough that budget policy must be enforced before upstream access.' } },
      { '@type': 'Question', name: 'How does SatGate help?', acceptedAnswer: { '@type': 'Answer', text: 'SatGate sits in the request path and checks identity, budget, route, tool scope, credential caveats, expiry, revocation, and audit policy before forwarding the request.' } },
      { '@type': 'Question', name: 'How is AI API budget enforcement different from provider spend alerts?', acceptedAnswer: { '@type': 'Answer', text: 'Provider spend alerts notify teams after usage crosses a threshold. AI API budget enforcement checks request cost, remaining budget, identity, route, and policy before the API call executes.' } },
      { '@type': 'Question', name: 'What should happen when an AI agent exceeds its API budget?', acceptedAnswer: { '@type': 'Answer', text: 'The request should be blocked, downgraded, routed to a cheaper provider, sent for approval, or challenged for payment depending on policy, with an audit record explaining the decision.' } },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'AI Agent Cost Control', item: 'https://satgate.io/ai-agent-cost-control' },
      { '@type': 'ListItem', position: 3, name: 'AI API Budget Enforcement', item: 'https://satgate.io/ai-api-budget-enforcement' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_24%_0%,rgba(34,211,238,0.18),transparent_34%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200"><Gauge size={16} /> Hard caps before API spend happens</div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">AI API budget enforcement belongs in the request path</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">AI agents can call APIs faster than finance, dashboards, or alerts can react. SatGate puts authority before execution by enforcing budget, route, scope, revocation, and audit policy before upstream API, model, or MCP tool calls execute.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Govern AI API budgets <ArrowRight size={18} /></Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">See Policy-to-Proof</Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1fr_0.9fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">The control point is before the call</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>Autonomous agents can generate real costs through model calls, API requests, MCP tools, delegated sub-agents, retries, and background workflows. If the policy check happens after the request, the money is already spent.</p>
            <p>SatGate enforces economic policy at the gateway boundary. Every important request can be evaluated against budget, scope, identity, revocation, route, tool, receipt, and audit rules before upstream access.</p>
            <p>That is the difference between cost reporting and economic control.</p>
          </div>
        </div>
        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">What good policy includes</h3>
          <div className="space-y-4">
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Budget before forwarding</h3><p className="leading-relaxed text-gray-400">Check remaining daily, session, per-agent, per-tool, and per-request budget before traffic reaches the upstream provider.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Policy by agent and workflow</h3><p className="leading-relaxed text-gray-400">Separate budgets by tenant, workflow, route, model, MCP server, delegated sub-agent, and task instead of one account-level cap.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Stop loops automatically</h3><p className="leading-relaxed text-gray-400">Deny the next request when a loop, retry storm, fanout chain, or high-cost tool call crosses policy.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Audit every decision</h3><p className="leading-relaxed text-gray-400">Record allow/deny decisions with identity, route, estimated cost, budget remaining, policy, and revocation state.</p></div>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">SatGate controls</h2>
          <p className="mb-10 max-w-3xl text-lg text-gray-400">Use SatGate to move from Observe to Control: measure real agent economics first, then enforce the limits that matter.</p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
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

      <section className="border-t border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">AI API budget enforcement questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is AI API budget enforcement?</h3>
              <p className="text-gray-400 leading-relaxed">
                AI API budget enforcement is the request-path control that checks budgets, per-request cost, route policy, tool scope, expiry, and revocation before an autonomous agent can spend against an API or model provider.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why are dashboards not enough?</h3>
              <p className="text-gray-400 leading-relaxed">
                Dashboards and billing alerts report spend after requests complete. Autonomous agents can loop, retry, and delegate fast enough that budget policy must be enforced before upstream access.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does SatGate help?</h3>
              <p className="text-gray-400 leading-relaxed">
                SatGate sits in the request path and checks identity, budget, route, tool scope, credential caveats, expiry, revocation, and audit policy before forwarding the request.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How is AI API budget enforcement different from provider spend alerts?</h3>
              <p className="text-gray-400 leading-relaxed">
                Provider spend alerts notify teams after usage crosses a threshold. AI API budget enforcement checks request cost, remaining budget, identity, route, and policy before the API call executes.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What should happen when an AI agent exceeds its API budget?</h3>
              <p className="text-gray-400 leading-relaxed">
                The request should be blocked, downgraded, routed to a cheaper provider, sent for approval, or challenged for payment depending on policy, with an audit record explaining the decision.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-purple-900/60 bg-gradient-to-br from-purple-950/35 to-cyan-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Make agent economics enforceable.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">SatGate is the economic firewall for AI agents: observe every request, enforce spend before execution, and preserve Policy-to-Proof receipts when paid access or budget decisions occur.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Govern AI API spend <ArrowRight size={18} /></Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">Review Policy-to-Proof</Link>
          </div>
        </div>
      </section>
    </main>
  );
}

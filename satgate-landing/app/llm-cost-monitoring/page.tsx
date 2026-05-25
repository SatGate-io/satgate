import Link from 'next/link';
import { ArrowRight, Bell, Bot, Eye, Gauge, LineChart, LockKeyhole, Route, ShieldAlert } from 'lucide-react';

export const metadata = {
  title: 'LLM Cost Monitoring: Dashboards, Alerts, and Real-Time Enforcement',
  description: 'LLM cost monitoring should track tokens, latency, model spend, agents, tools, and alerts — then enforce budgets before runaway requests execute.',
  alternates: { canonical: 'https://satgate.io/llm-cost-monitoring' },
  keywords: [
    'LLM cost monitoring',
    'LLM cost management',
    'LLM monitoring dashboard',
    'LLM cost control',
    'token cost monitoring',
    'AI cost monitoring',
    'LLM budget management',
    'agent cost monitoring',
  ],
  openGraph: {
    title: 'LLM Cost Monitoring: Dashboards, Alerts, and Real-Time Enforcement',
    description: 'A practical guide to monitoring LLM cost and converting observability into request-path budget enforcement for AI agents.',
    url: 'https://satgate.io/llm-cost-monitoring',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'LLM Cost Monitoring: Dashboards, Alerts, and Real-Time Enforcement',
    description: 'Track tokens, latency, model spend, agents, tools, and alerts — then enforce budgets before runaway requests execute.',
  },
};

const stages = [
  { icon: Eye, title: 'Observe', body: 'Capture every model, API, and MCP tool request with agent, user, tenant, route, model, latency, and cost context.' },
  { icon: LineChart, title: 'Monitor', body: 'Trend spend velocity, token growth, retry storms, model drift, error spikes, and unusual agent behavior.' },
  { icon: Bell, title: 'Alert', body: 'Notify teams when spend crosses thresholds, but treat alerting as a signal — not the control itself.' },
  { icon: Gauge, title: 'Budget', body: 'Assign per-agent, per-session, per-route, per-tool, and per-tenant ceilings in dollars or credits.' },
  { icon: Route, title: 'Route', body: 'Move low-value calls to cheaper models or tools while reserving premium routes for justified work.' },
  { icon: LockKeyhole, title: 'Enforce', body: 'Block, downgrade, revoke, or require payment before the upstream call executes.' },
];

export default function LlmCostMonitoringPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'LLM Cost Monitoring: Dashboards, Alerts, and Real-Time Enforcement',
    url: 'https://satgate.io/llm-cost-monitoring',
    description: metadata.description,
    datePublished: '2026-05-01',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'LLM cost monitoring' },
      { '@type': 'Thing', name: 'AI agent cost control' },
      { '@type': 'Thing', name: 'spend velocity alerts' },
      { '@type': 'Thing', name: 'request-path enforcement' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is LLM cost monitoring?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'LLM cost monitoring tracks token usage, model spend, latency, errors, retries, users, teams, agents, workflows, MCP tools, and API routes so teams can understand where AI spend is created.',
        },
      },
      {
        '@type': 'Question',
        name: 'What is the difference between LLM cost monitoring and LLM cost control?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Monitoring observes and alerts on spend. Cost control enforces budget policy before requests execute by blocking, routing, revoking, downgrading, or requiring payment in the request path.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do AI agents need more than cost monitoring?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'AI agents can retry, loop, call tools, and delegate faster than humans can react to alerts. They need budget enforcement in the request path, not only dashboards after spend is created.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do you turn LLM cost monitoring signals into controls?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Convert monitoring signals into policy objects: per-agent and per-route budgets, MCP tool caps, model-routing rules, scoped capability tokens, revocation triggers, and Evidence Pack requirements enforced before upstream calls execute.',
        },
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'LLM Cost Monitoring', item: 'https://satgate.io/llm-cost-monitoring' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_16%_0%,rgba(168,85,247,0.18),transparent_32%),radial-gradient(circle_at_84%_14%,rgba(34,211,238,0.16),transparent_30%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-purple-500/30 bg-purple-950/30 px-4 py-2 text-sm text-purple-200">
            <ShieldAlert size={16} /> LLM cost monitoring
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            LLM Cost Monitoring Is the Warning Light. Enforcement Is the Brake.
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Monitoring tells you which models, tools, users, and agents create spend. SatGate turns that visibility into request-path budgets, routing, revocation, and structured denials before runaway cost becomes a bill.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/llm-cost-dashboard" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              See dashboard checklist <ArrowRight size={18} />
            </Link>
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-purple-500">
              Learn economic firewalls
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1fr_0.9fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">Monitoring is necessary. It is not sufficient.</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>
              LLM cost monitoring gives engineering, finance, and security teams the visibility they need: token usage, model costs, latency, traces, error rates, user attribution, and spend by route.
            </p>
            <p>
              But autonomous agents change the failure mode. They can call tools while you sleep, retry through failures, fan out to sub-agents, and move spend from model providers into MCP servers and paid APIs.
            </p>
            <p>
              That means monitoring must feed enforcement. When the system detects cost risk, it should not only alert a human — it should enforce budgets, route cheaper, revoke stale authority, or deny the request before spend is created.
            </p>
          </div>
        </div>
        <div className="rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">Monitoring signals worth enforcing</h3>
          <ul className="space-y-3 text-gray-300">
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Spend velocity is above normal for this agent or team.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">A workflow switched from cheap to premium models unexpectedly.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">MCP tool calls are repeating after upstream errors.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">A delegated sub-agent is spending outside its task budget.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">A token is active after the task or session should have ended.</li>
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">From LLM cost monitoring to economic control</h2>
          <p className="mb-10 max-w-3xl text-lg text-gray-400">A mature stack does not stop at graphs. It turns signals into policy.</p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {stages.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 transition hover:border-purple-900/70">
                <Icon className="mb-4 text-purple-300" size={28} />
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <h2 className="mb-8 text-3xl font-bold text-white">Monitoring vs enforcement</h2>
        <div className="overflow-hidden rounded-2xl border border-gray-800">
          <div className="grid grid-cols-3 bg-gray-950 text-sm font-semibold uppercase tracking-[0.18em] text-gray-400">
            <div className="p-4">Need</div>
            <div className="border-l border-gray-800 p-4">Monitoring</div>
            <div className="border-l border-gray-800 p-4">SatGate enforcement</div>
          </div>
          {[
            ['Token spend visibility', 'Shows spend after requests execute', 'Shows spend and attaches it to enforceable policy'],
            ['Runaway agent loops', 'Alerts when spend spikes', 'Blocks or downgrades requests before budget is exceeded'],
            ['MCP tool cost', 'May miss non-model tool spend', 'Prices and caps each tool call in the request path'],
            ['Shared API keys', 'Shows account-level cost', 'Uses scoped, revocable agent authority and attribution'],
            ['Finance controls', 'Exports reports', 'Enforces team budgets and chargeback boundaries inline'],
          ].map(([need, monitoring, enforcement]) => (
            <div key={need} className="grid grid-cols-3 border-t border-gray-800 text-gray-300">
              <div className="p-4 font-semibold text-white">{need}</div>
              <div className="border-l border-gray-800 p-4">{monitoring}</div>
              <div className="border-l border-gray-800 p-4">{enforcement}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="border-t border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">Policy generators</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Make monitoring actionable</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            When monitoring exposes a risky agent, model route, or MCP tool, the next step is not another chart. Generate the request-path policy that can stop the next bad call.
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {[
              ['/agent-spend-policy-template', 'Agent spend policy', 'Budgets, delegation, revocation, MCP tool caps, and Evidence Pack fields.'],
              ['/mcp-tool-cost-policy-generator', 'MCP tool cost policy', 'Per-tool prices, risk tiers, limits, and deny behavior.'],
              ['/revocable-capability-token-policy-template', 'Capability-token policy', 'Scoped, expiring, revocable agent authority with budget caveats.'],
              ['/economic-firewall-readiness-grader', 'Readiness grader', 'Find gaps across identity, budgets, routing, revocation, Evidence Pack proof, and paid-rail context.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-black p-5 transition hover:border-purple-500/50 hover:bg-purple-950/20">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">LLM cost monitoring questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is LLM cost monitoring?</h3>
              <p className="leading-relaxed text-gray-400">LLM cost monitoring tracks token usage, model spend, latency, errors, retries, users, teams, agents, workflows, MCP tools, and API routes so teams can understand where AI spend is created.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is the difference between LLM cost monitoring and LLM cost control?</h3>
              <p className="leading-relaxed text-gray-400">Monitoring observes and alerts on spend. Cost control enforces budget policy before requests execute by blocking, routing, revoking, downgrading, or requiring payment in the request path.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why do AI agents need more than cost monitoring?</h3>
              <p className="leading-relaxed text-gray-400">AI agents can retry, loop, call tools, and delegate faster than humans can react to alerts. They need budget enforcement in the request path, not only dashboards after spend is created.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How do you turn LLM cost monitoring signals into controls?</h3>
              <p className="leading-relaxed text-gray-400">Convert monitoring signals into policy objects: per-agent and per-route budgets, MCP tool caps, model-routing rules, scoped capability tokens, revocation triggers, and Evidence Pack requirements enforced before upstream calls execute.</p>
            </div>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-gradient-to-r from-purple-950/30 to-cyan-950/20">
        <div className="mx-auto max-w-6xl px-6 py-16">
          <div className="grid gap-6 md:grid-cols-3">
            <Link href="/llm-cost-dashboard" className="rounded-2xl border border-gray-800 bg-black/70 p-6 transition hover:border-purple-600">
              <h3 className="mb-2 text-lg font-bold text-white">LLM cost dashboard →</h3>
              <p className="text-gray-400">The dashboard checklist for model, agent, and MCP spend.</p>
            </Link>
            <Link href="/roi-calculator" className="rounded-2xl border border-gray-800 bg-black/70 p-6 transition hover:border-purple-600">
              <h3 className="mb-2 text-lg font-bold text-white">ROI calculator →</h3>
              <p className="text-gray-400">Estimate loop waste, ghost spend, payback, and avoided cost.</p>
            </Link>
            <Link href="/agent-spend-policy-template" className="rounded-2xl border border-gray-800 bg-black/70 p-6 transition hover:border-purple-600">
              <h3 className="mb-2 text-lg font-bold text-white">Agent spend policy →</h3>
              <p className="text-gray-400">Turn monitoring signals into budget, revocation, and Evidence Pack policy.</p>
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

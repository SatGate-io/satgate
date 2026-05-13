import Link from 'next/link';
import { ArrowRight, BarChart3, Bell, Bot, DollarSign, Gauge, ShieldCheck, Workflow } from 'lucide-react';

export const metadata = {
  title: 'LLM Cost Dashboard: Spend, Latency, Tokens, and Budget Risk',
  description: 'What an LLM cost dashboard should track: token cost, latency, model spend, user attribution, agent loops, MCP tools, and budget enforcement gaps.',
  alternates: { canonical: 'https://satgate.io/llm-cost-dashboard' },
  keywords: [
    'LLM cost dashboard',
    'LLM cost monitoring dashboard',
    'token cost dashboard',
    'AI cost dashboard',
    'LLM spend dashboard',
    'agent cost dashboard',
    'LLM cost attribution',
    'AI agent budget dashboard',
  ],
  openGraph: {
    title: 'LLM Cost Dashboard: Spend, Latency, Tokens, and Budget Risk',
    description: 'A practical checklist for LLM cost dashboards — and why dashboards still need request-path budget enforcement for AI agents.',
    url: 'https://satgate.io/llm-cost-dashboard',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'LLM Cost Dashboard: Spend, Latency, Tokens, and Budget Risk',
    description: 'Track model spend, token usage, latency, user attribution, MCP tool cost, and enforcement gaps before agents run away.',
  },
};

const dashboardMetrics = [
  { icon: DollarSign, title: 'Cost by model and route', body: 'Show spend by OpenAI, Anthropic, local model, API route, endpoint, and fallback path — not just aggregate token totals.' },
  { icon: Bot, title: 'Cost by agent and workflow', body: 'Attribute every request to the agent, user, tenant, workflow, delegated sub-agent, and task that caused the spend.' },
  { icon: BarChart3, title: 'Tokens, latency, and errors', body: 'Correlate cost with prompt tokens, completion tokens, tool latency, retry rates, and upstream failure patterns.' },
  { icon: Workflow, title: 'MCP and tool spend', body: 'Track paid tool calls, MCP server usage, per-tool prices, search calls, code execution, enrichment APIs, and premium actions.' },
  { icon: Bell, title: 'Alert thresholds', body: 'Warn on abnormal spend velocity, daily budget burn, retry storms, and expensive model drift — but do not confuse alerts with control.' },
  { icon: ShieldCheck, title: 'Enforcement gaps', body: 'Highlight where a dashboard can see spend but cannot block it: shared API keys, missing budgets, stale tokens, and no kill switch.' },
];

export default function LlmCostDashboardPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'LLM Cost Dashboard: Spend, Latency, Tokens, and Budget Risk',
    url: 'https://satgate.io/llm-cost-dashboard',
    description: metadata.description,
    datePublished: '2026-05-01',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'LLM cost dashboard' },
      { '@type': 'Thing', name: 'AI agent cost attribution' },
      { '@type': 'Thing', name: 'MCP tool spend' },
      { '@type': 'Thing', name: 'request-path budget enforcement' },
    ],
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate LLM Cost Dashboard',
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Cloud, API Gateway',
    url: 'https://satgate.io/llm-cost-dashboard',
    description: metadata.description,
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-03',
    featureList: dashboardMetrics.map((m) => m.title),
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What should an LLM cost dashboard track?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'An LLM cost dashboard should track spend by model, route, user, team, agent, workflow, tenant, token, MCP tool, latency, error rate, retry behavior, and remaining budget.',
        },
      },
      {
        '@type': 'Question',
        name: 'Is an LLM cost dashboard enough to stop runaway spend?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. Dashboards and alerts show spend after or during usage. Autonomous agents need request-path budget enforcement that can block, downgrade, route, or revoke requests before expensive calls execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate turn LLM cost dashboards into enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate observes agent/API spend, attributes it by agent and route, then enforces budgets, revocation, routing, and MCP tool policy in the request path before upstream calls execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'What should teams do after finding LLM spend risk?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Turn the dashboard finding into enforceable policy: set per-agent budgets, MCP tool caps, model-routing rules, scoped token authority, revocation triggers, and receipt and Evidence Pack fields in the request path.',
        },
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'LLM Cost Dashboard', item: 'https://satgate.io/llm-cost-dashboard' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(34,211,238,0.18),transparent_32%),radial-gradient(circle_at_82%_12%,rgba(168,85,247,0.16),transparent_30%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <BarChart3 size={16} /> LLM cost dashboard
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            LLM Cost Dashboard: Track Spend Before Agents Run Away
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            A useful LLM cost dashboard shows token cost, latency, model spend, user attribution, MCP tool calls, and agent budget risk. A great one also tells you where dashboards stop and request-path enforcement must begin.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/roi-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Estimate runaway spend <ArrowRight size={18} />
            </Link>
            <Link href="/llm-cost-monitoring" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Compare monitoring vs enforcement
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1fr_0.9fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">The dashboard checklist</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>
              Most LLM dashboards start with tokens and cost per model. That is necessary, but not enough for agents. Autonomous systems create spend through workflows, retries, MCP tools, delegation, and paid APIs outside the provider dashboard.
            </p>
            <p>
              The useful question is not just “what did GPT-4o cost yesterday?” It is “which agent, acting for which user, on which task, called which tool, through which route, and should that request have been allowed?”
            </p>
            <p>
              SatGate answers that question by combining dashboard visibility with the economic firewall controls needed to block over-budget requests before they become spend.
            </p>
          </div>
        </div>
        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">Dashboard-only blind spots</h3>
          <ul className="space-y-3 text-gray-300">
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Alerts arrive after costly requests already executed.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Shared API keys hide the agent or workflow responsible.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">MCP tool costs often live outside model-provider billing.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Dashboards cannot revoke a runaway sub-agent by themselves.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Account caps can break every workload when one agent misbehaves.</li>
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">Metrics that matter in an LLM cost dashboard</h2>
          <p className="mb-10 max-w-3xl text-lg text-gray-400">Track cost like an economic system, not a static billing report.</p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {dashboardMetrics.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 transition hover:border-cyan-900/70">
                <Icon className="mb-4 text-cyan-300" size={28} />
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <h2 className="mb-8 text-3xl font-bold text-white">Dashboard → enforcement workflow</h2>
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <div className="mb-3 font-mono text-sm text-cyan-300">01 / OBSERVE</div>
            <h3 className="mb-3 text-xl font-bold text-white">See every cost center</h3>
            <p className="leading-relaxed text-gray-400">Capture model, API, MCP, and tool spend by agent, team, tenant, route, and workflow.</p>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <div className="mb-3 font-mono text-sm text-purple-300">02 / ANALYZE</div>
            <h3 className="mb-3 text-xl font-bold text-white">Find budget risk</h3>
            <p className="leading-relaxed text-gray-400">Identify retry storms, model drift, prompt bloat, expensive tool paths, missing attribution, and high-risk agents.</p>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <div className="mb-3 font-mono text-sm text-green-300">03 / CONTROL</div>
            <h3 className="mb-3 text-xl font-bold text-white">Block overspend inline</h3>
            <p className="leading-relaxed text-gray-400">Turn dashboard findings into budgets, route policy, revocation, model ceilings, and structured denial responses.</p>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">From dashboard to control</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Convert cost visibility into policy</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            A dashboard should not be a dead end. Once it exposes spend risk, generate the policy objects that let SatGate block, route, revoke, or issue receipts for Evidence Packs the next request.
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {[
              ['/agent-spend-policy-template', 'Agent spend policy', 'Budgets, MCP caps, delegation, revocation, and receipt and Evidence Pack fields.'],
              ['/mcp-tool-cost-policy-generator', 'MCP tool cost policy', 'Per-tool prices, risk tiers, limits, and deny behavior.'],
              ['/revocable-capability-token-policy-template', 'Capability-token policy', 'Scoped, expiring, revocable authority for agents and sub-agents.'],
              ['/openai-budget-policy-generator', 'OpenAI budget policy', 'Model, route, session, daily, and per-request budget limits.'],
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
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">LLM cost dashboard questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What should an LLM cost dashboard track?</h3>
              <p className="leading-relaxed text-gray-400">An LLM cost dashboard should track spend by model, route, user, team, agent, workflow, tenant, token, MCP tool, latency, error rate, retry behavior, and remaining budget.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Is an LLM cost dashboard enough to stop runaway spend?</h3>
              <p className="leading-relaxed text-gray-400">No. Dashboards and alerts show spend after or during usage. Autonomous agents need request-path budget enforcement that can block, downgrade, route, or revoke requests before expensive calls execute.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does SatGate turn LLM cost dashboards into enforcement?</h3>
              <p className="leading-relaxed text-gray-400">SatGate observes agent/API spend, attributes it by agent and route, then enforces budgets, revocation, routing, and MCP tool policy in the request path before upstream calls execute.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What should teams do after finding LLM spend risk?</h3>
              <p className="leading-relaxed text-gray-400">Turn the dashboard finding into enforceable policy: set per-agent budgets, MCP tool caps, model-routing rules, scoped token authority, revocation triggers, and receipt and Evidence Pack fields in the request path.</p>
            </div>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-gradient-to-r from-cyan-950/30 to-purple-950/20">
        <div className="mx-auto max-w-6xl px-6 py-16">
          <div className="grid gap-6 md:grid-cols-3">
            <Link href="/llm-cost-monitoring" className="rounded-2xl border border-gray-800 bg-black/70 p-6 transition hover:border-cyan-600">
              <h3 className="mb-2 text-lg font-bold text-white">LLM cost monitoring →</h3>
              <p className="text-gray-400">Compare dashboards, alerts, and request-path enforcement.</p>
            </Link>
            <Link href="/ai-agent-cost-control" className="rounded-2xl border border-gray-800 bg-black/70 p-6 transition hover:border-cyan-600">
              <h3 className="mb-2 text-lg font-bold text-white">AI agent cost control →</h3>
              <p className="text-gray-400">Control model, API, and MCP spend before it happens.</p>
            </Link>
            <Link href="/economic-firewall" className="rounded-2xl border border-gray-800 bg-black/70 p-6 transition hover:border-cyan-600">
              <h3 className="mb-2 text-lg font-bold text-white">Economic firewall →</h3>
              <p className="text-gray-400">Move from observability to request-path economic governance.</p>
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

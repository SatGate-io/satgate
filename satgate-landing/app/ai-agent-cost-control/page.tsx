import Link from 'next/link';
import { ArrowRight, Bot, DollarSign, Gauge, ShieldCheck, Workflow, BarChart3 } from 'lucide-react';

export const metadata = {
  title: 'AI Agent Cost Control Software',
  description: 'Control AI agent API spend before it happens. SatGate enforces budgets, revocation, routing, and audit trails in the request path.',
  alternates: { canonical: 'https://satgate.io/ai-agent-cost-control' },
  keywords: [
    'AI agent cost control',
    'AI agent spend control',
    'AI agent budget enforcement',
    'LLM cost control',
    'OpenAI API budget limits',
    'MCP budget enforcement',
    'runaway agent spend',
    'AI cost governance',
  ],
  openGraph: {
    title: 'AI Agent Cost Control Software',
    description: 'Enforce per-agent budgets, spend caps, revocation, routing, and audit trails before autonomous API calls execute.',
    url: 'https://satgate.io/ai-agent-cost-control',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI Agent Cost Control Software',
    description: 'Stop runaway AI agent spend with request-path budget enforcement, revocation, routing, and audit trails.',
  },
};

const controls = [
  {
    icon: Bot,
    title: 'Per-agent identity',
    body: 'Separate spend by agent, workflow, tenant, team, token, model, and route instead of sharing one blind API key.',
  },
  {
    icon: DollarSign,
    title: 'Real-time budgets',
    body: 'Stop runaway spend before the upstream API call, not after a dashboard or billing alert catches up.',
  },
  {
    icon: Gauge,
    title: 'Per-tool cost caps',
    body: 'Set limits for MCP tools, paid APIs, premium models, search calls, code agents, and delegated sub-agents.',
  },
  {
    icon: ShieldCheck,
    title: 'Revocation and kill switches',
    body: 'Expire, revoke, or narrow capabilities immediately when an agent misbehaves or a task is complete.',
  },
  {
    icon: Workflow,
    title: 'Provider routing',
    body: 'Route routine work to lower-cost providers and reserve premium models for tasks that justify the spend.',
  },
  {
    icon: BarChart3,
    title: 'Audit and attribution',
    body: 'Record who spent what, on which tool, through which route, and why the policy allowed or denied it.',
  },
];

export default function AiAgentCostControlPage() {
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate AI Agent Cost Control',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Cloud, API Gateway',
    description: metadata.description,
    url: 'https://satgate.io/ai-agent-cost-control',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    offers: { '@type': 'Offer', url: 'https://satgate.io/pricing' },
    featureList: [
      'Per-agent budget enforcement',
      'Per-tool cost attribution',
      'MCP budget enforcement',
      'Revocable agent credentials',
      'Request-path audit trails',
      'L402 API monetization',
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is AI agent cost control?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'AI agent cost control is the practice of attributing, budgeting, limiting, and auditing autonomous agent API and tool spend before requests execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why are provider dashboards not enough for AI agent spend control?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Dashboards report spend after the fact. Autonomous agents can retry, loop, and delegate fast enough that budget enforcement must happen inline before upstream API calls complete.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate enforce AI agent budgets?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate sits in the request path and checks agent identity, route, tool cost, remaining budget, revocation status, and policy before forwarding each request.',
        },
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'AI Agent Cost Control', item: 'https://satgate.io/ai-agent-cost-control' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(168,85,247,0.18),transparent_32%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.16),transparent_30%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-purple-500/30 bg-purple-950/30 px-4 py-2 text-sm text-purple-200 mb-8">
            <DollarSign size={16} /> Real-time agent spend control
          </div>

          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            Control AI Agent API Spend Before It Happens
          </h1>

          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-10">
            SatGate puts budget enforcement, revocation, routing, and audit trails in the request path so autonomous agents cannot silently burn through OpenAI, Claude, MCP, or paid API budgets.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/roi-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Estimate avoided spend <ArrowRight size={18} />
            </Link>
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Learn economic firewalls
            </Link>
            <Link href="/llm-cost-dashboard" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              LLM cost dashboard checklist
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12 items-start">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">Dashboards do not stop agent loops</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              LLM dashboards, provider billing pages, and token reports are useful after the fact. They tell you what happened. They do not stop an autonomous agent from calling a premium model, retrying a failed tool, or delegating a task into a thousand-dollar loop.
            </p>
            <p>
              AI agent cost control has to be inline. Every request needs an economic decision before it reaches the upstream provider: who is calling, what route is allowed, what the request costs, whether budget remains, and what should be recorded.
            </p>
            <p>
              SatGate enforces those decisions at the gateway layer across internal agents, MCP tools, hosted APIs, model providers, and external robot customers.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">Common failure modes</h3>
          <ul className="space-y-3 text-gray-300">
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">One shared API key hides which agent created the spend.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Account-level provider caps take down every workload at once.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Alerts fire after the expensive request already completed.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">MCP tools create cost outside the LLM provider dashboard.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Agents retry, loop, and delegate faster than humans can intervene.</li>
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">What SatGate controls</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            SatGate is not just an observability dashboard. It is the request-path economic control plane for agent/API activity.
          </p>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {controls.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 hover:border-purple-900/70 transition">
                <Icon className="text-purple-300 mb-4" size={28} />
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-8">Start with Observe. Graduate to Control. Add Charge when agents become customers.</h2>
        <div className="grid lg:grid-cols-3 gap-6">
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <div className="text-purple-300 font-mono text-sm mb-3">01 / OBSERVE</div>
            <h3 className="text-xl font-bold text-white mb-3">Attribute every call</h3>
            <p className="text-gray-400 leading-relaxed">Route agent traffic through SatGate to see spend by agent, model, route, tool, tenant, and workflow without blocking production workloads.</p>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <div className="text-cyan-300 font-mono text-sm mb-3">02 / CONTROL</div>
            <h3 className="text-xl font-bold text-white mb-3">Enforce budgets</h3>
            <p className="text-gray-400 leading-relaxed">Apply hard caps, per-request ceilings, route policy, revocation, expiry, and kill switches before the expensive call happens.</p>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <div className="text-yellow-300 font-mono text-sm mb-3">03 / CHARGE</div>
            <h3 className="text-xl font-bold text-white mb-3">Monetize agent access</h3>
            <p className="text-gray-400 leading-relaxed">When APIs become products for external autonomous agents, collect payment with Charge/L402 before unlocking the protected resource.</p>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">High-intent use cases</h2>
          <div className="grid md:grid-cols-2 gap-5">
            {[
              ['AI API budget enforcement', 'Enforce per-agent and per-workflow spend caps before OpenAI, Claude, MCP, or paid API requests leave your environment.', '/ai-api-budget-enforcement'],
              ['Agent spending limits', 'Set hard caps by task, route, model, tool, tenant, session, and delegated sub-agent.', '/agent-spending-limits'],
              ['Agent spend policy template', 'Generate copyable YAML/JSON policy for budgets, tools, delegation, revocation, and audit fields.', '/agent-spend-policy-template'],
              ['MCP tool spend control', 'Attach cost to tool calls and stop runaway Cursor, Claude Desktop, Claude Code, or OpenClaw workflows.', '/mcp-cost-control'],
              ['Revocable agent credentials', 'Replace broad static keys with scoped, expiring credentials and kill switches for autonomous workers.', '/revocable-agent-credentials'],
              ['Robot customer payments', 'Let external agents pay for protected APIs through a governed request path.', '/robot-customer-payments'],
            ].map(([title, body, href]) => (
              <Link key={title} href={href} className="rounded-xl border border-gray-800 bg-black p-6 hover:border-cyan-800/70 transition block">
                <h3 className="text-xl font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed mb-4">{body}</p>
                <span className="text-cyan-300 font-semibold inline-flex items-center gap-2">Read guide <ArrowRight size={16} /></span>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">AI agent cost-control requirements</h2>
          <div className="grid md:grid-cols-2 gap-5">
            {[
              ['Attribute spend before optimizing it', 'Every request should carry tenant, agent, workflow, token, route, model, and tool context so finance and platform teams can see who created the cost.'],
              ['Enforce budgets before API calls execute', 'Budget policy belongs in the request path. Alerts, dashboards, and billing exports are useful, but they are too late to stop runaway loops.'],
              ['Use scoped, revocable credentials', 'Autonomous agents should not hold unlimited API keys. Capabilities need expiry, caveats, spend ceilings, route limits, and emergency revocation.'],
              ['Treat MCP tools as economic resources', 'MCP tool calls can trigger paid APIs, searches, code agents, or data lookups. Cost policy has to follow the tool call, not just the LLM token bill.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
                <h3 className="text-xl font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-purple-900/60 bg-gradient-to-br from-purple-950/30 to-cyan-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">Find your avoidable agent spend</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            Use the SatGate ROI calculator to model ghost spend, runaway loops, wasted tool calls, and the payback period for request-path budget enforcement.
          </p>
          <Link href="/roi-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
            Open the ROI calculator <ArrowRight size={18} />
          </Link>
        </div>
      </section>
    </main>
  );
}

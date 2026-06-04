import Link from 'next/link';
import { ArrowRight, Bot, DollarSign, Gauge, ShieldCheck, Workflow, BarChart3 } from 'lucide-react';

export const metadata = {
  title: 'AI Agent Cost Control | Request-Path Budget Enforcement',
  description: 'Control AI agent API spend before it happens. SatGate enforces budgets, revocation, routing, and Evidence Pack receipts in the request path.',
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
    title: 'AI Agent Cost Control | Request-Path Budget Enforcement',
    description: 'Enforce per-agent budgets, spend caps, revocation, routing, and Evidence Pack receipts before autonomous API calls execute.',
    url: 'https://satgate.io/ai-agent-cost-control',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI Agent Cost Control | Request-Path Budget Enforcement',
    description: 'Stop runaway AI agent spend with request-path budget enforcement, revocation, routing, and Evidence Pack receipts.',
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
    body: 'Emit receipts showing who spent what, on which tool, through which route, why policy allowed or denied it, and which Evidence Pack preserves the proof.',
  },
];

export default function AiAgentCostControlPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'AI Agent Cost Control | Request-Path Budget Enforcement',
    description: metadata.description,
    url: 'https://satgate.io/ai-agent-cost-control',
    dateModified: '2026-06-04',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI agent cost control' },
      { '@type': 'Thing', name: 'AI agent spend control' },
      { '@type': 'Thing', name: 'request-path budget enforcement' },
      { '@type': 'Thing', name: 'MCP budget enforcement' },
      { '@type': 'Thing', name: 'runaway AI agent spend prevention' },
    ],
  };

  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate AI Agent Cost Control',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Cloud, API Gateway',
    description: metadata.description,
    url: 'https://satgate.io/ai-agent-cost-control',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-06-04',
    about: webPageJsonLd.about,
    offers: { '@type': 'Offer', url: 'https://satgate.io/pricing' },
    featureList: [
      'Per-agent budget enforcement',
      'Per-tool cost attribution',
      'MCP budget enforcement',
      'Revocable agent credentials',
      'Request-path Evidence Pack receipts',
      'Policy-to-Proof Evidence Packs',
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
          text: 'AI agent cost control is the practice of attributing, budgeting, limiting, and preserving receipts for autonomous agent API and tool spend before requests execute.',
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
      {
        '@type': 'Question',
        name: 'What is the difference between AI agent cost control and LLM cost management?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'LLM cost management usually tracks model and token spend after usage occurs. AI agent cost control adds request-path enforcement across agents, MCP tools, paid APIs, delegated sub-agents, budgets, revocation, and audit before cost is created.',
        },
      },
      {
        '@type': 'Question',
        name: 'Which policies should AI agent cost control include?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'A practical policy should include tenant and agent identity, route and tool scope, per-request and session budgets, MCP tool caps, delegated sub-agent limits, expiry, revocation triggers, kill switches, and audit fields.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can rate limits control AI agent costs?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Rate limits control request frequency, not economic exposure. AI agent cost control needs per-request pricing, remaining-budget checks, tool-level caps, and request-path decisions that account for expensive model or MCP tool calls.',
        },
      },
      {
        '@type': 'Question',
        name: 'When should a team add AI agent budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Add budget enforcement before agents receive access to paid APIs, premium models, MCP tools, data providers, or external services where retries, loops, or delegation can create real cost.',
        },
      },
    ],
  };

  const buyingChecklistJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent cost-control software buying checklist',
    description: 'Required capabilities for AI agent cost-control software that enforces spend before autonomous agents call paid APIs, models, or MCP tools.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Inline enforcement',
        description: 'Budget policy runs before model, API, or MCP tool execution, not after a billing export.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Agent-level attribution',
        description: 'Every request maps to tenant, workflow, agent, delegated sub-agent, token, route, and tool.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Revocable authority',
        description: 'Credentials can expire, narrow, delegate safely, or be killed without rotating shared API keys.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Evidence Pack capture',
        description: 'Allowed, denied, delegated, routed, paid, and revoked requests leave receipts finance and security can review.',
      },
    ],
  };

  const rolloutJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: '90-day AI agent cost-control rollout plan',
    description: 'A practical rollout path for moving AI agent spend from visibility to request-path budget enforcement.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Inventory agent spend exposure',
        description: 'Map agents, shared API keys, MCP tools, paid APIs, premium models, and workflows that can create cost.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Start in Observe mode',
        description: 'Route traffic through SatGate to attribute spend by tenant, agent, workflow, route, model, and tool before blocking anything.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Enforce scoped budgets',
        description: 'Apply per-agent budgets, MCP tool caps, route ceilings, expiry, delegation limits, and revocation policies in the request path.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Preserve proof for paid access',
        description: 'When paid access is allowed, preserve the policy decision, payment context, and receipt before granting access.',
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
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(buyingChecklistJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(rolloutJsonLd) }} />
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
            SatGate puts authority before execution: budget enforcement, revocation, routing, and Evidence Pack receipts run in the request path before autonomous agents can spend against OpenAI, Claude, MCP, or paid API budgets.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Govern agent spend <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              See Policy-to-Proof
            </Link>
            <Link href="/roi-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Estimate avoided spend
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
              AI agent cost control has to be inline. Every request needs an economic decision before it reaches the upstream provider: who is calling, what authority applies, what route is allowed, what the request costs, whether budget remains, and which receipt should be recorded.
            </p>
            <p>
              SatGate enforces those decisions at the gateway layer across internal agents, MCP tools, hosted APIs, model providers, and paid external-access flows.
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
            SatGate is not just an observability dashboard. It is the request-path authority and proof layer for agent/API activity.
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
        <h2 className="text-3xl font-bold text-white mb-8">Start with Observe. Graduate to Control. Preserve proof for every decision.</h2>
        <div className="grid lg:grid-cols-3 gap-6">
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <div className="text-purple-300 font-mono text-sm mb-3">01 / OBSERVE</div>
            <h3 className="text-xl font-bold text-white mb-3">Attribute every request</h3>
            <p className="text-gray-400 leading-relaxed">See tenant, workflow, agent, delegated sub-agent, model, route, MCP tool, and spend before turning on enforcement.</p>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <div className="text-cyan-300 font-mono text-sm mb-3">02 / CONTROL</div>
            <h3 className="text-xl font-bold text-white mb-3">Enforce before execution</h3>
            <p className="text-gray-400 leading-relaxed">Apply hard caps, per-request ceilings, route policy, revocation, expiry, and kill switches before the expensive call happens.</p>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <div className="text-yellow-300 font-mono text-sm mb-3">03 / PROVE</div>
            <h3 className="text-xl font-bold text-white mb-3">Preserve decision evidence</h3>
            <p className="text-gray-400 leading-relaxed">Record every authority decision — allowed, denied, delegated, revoked, or paid — in the Evidence Pack. Payment proves value moved; SatGate proves the agent was allowed to move it.</p>
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
              ['Capability-token policy template', 'Generate scoped, expiring, revocable capability-token policy with budget, delegation, and audit caveats.', '/revocable-capability-token-policy-template'],
              ['Agent payment controls', 'Govern wallet approval, payment context, budgets, and audit before protected API access.', '/agent-payment-controls'],
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
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Buying checklist</p>
          <h2 className="mb-4 text-3xl font-bold text-white">What to demand from AI agent cost-control software</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            If the product only reports spend after the fact, it is observability — not cost control. AI agent cost-control software should make a deny/allow/reroute decision before every expensive call.
          </p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
            {[
              ['Inline enforcement', 'Budget policy runs before model, API, or MCP tool execution — not after a billing export.'],
              ['Agent-level attribution', 'Every request maps to tenant, workflow, agent, delegated sub-agent, token, route, and tool.'],
              ['Revocable authority', 'Credentials can expire, narrow, delegate safely, or be killed without rotating shared API keys.'],
              ['Evidence Pack capture', 'Allowed, denied, delegated, routed, paid, and revoked requests leave receipts finance and security can review.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-5">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
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

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">90-day rollout</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Move from visibility to hard budget enforcement</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            The safest path is not a big-bang control rollout. Start by attributing spend, then tighten policies until every agent call has a budget, scope, expiry, and revocation path.
          </p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
            {[
              ['Inventory exposure', 'Map agents, shared API keys, MCP tools, paid APIs, premium models, and workflows that can create cost.', '/agent-api-key-risk-assessment'],
              ['Observe first', 'Route traffic through SatGate to attribute spend by tenant, agent, workflow, route, model, and tool before blocking.', '/llm-cost-monitoring'],
              ['Enforce budgets', 'Apply per-agent budgets, MCP caps, route ceilings, expiry, delegation limits, and revocation policy in the request path.', '/agent-spend-policy-template'],
              ['Preserve paid-access proof', 'Record policy decisions, payment context, and receipts before granting paid external access.', '/policy-to-proof'],
            ].map(([title, body, href]) => (
              <Link key={title} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="mb-4 text-sm leading-relaxed text-gray-400">{body}</p>
                <span className="text-sm font-semibold text-cyan-300">Open step →</span>
              </Link>
            ))}
          </div>
          <div className="mt-8 rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6">
            <h3 className="mb-2 text-xl font-bold text-white">Need a readiness score first?</h3>
            <p className="mb-4 text-gray-400">Use the grader to see whether identity, budget policy, MCP governance, revocation, audit, routing, and paid-rail evidence are ready for autonomous agents.</p>
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center gap-2 font-semibold text-purple-300 hover:text-purple-200">Run the economic firewall readiness grader <ArrowRight size={16} /></Link>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">Cost-control toolkit</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Turn spend exposure into enforceable controls</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            The commercial page should lead buyers from awareness to action: estimate the risk, generate the policy, then enforce it in the request path.
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {[
              ['/roi-calculator', 'ROI calculator', 'Estimate ghost spend, loop waste, payback period, and annual ROI.'],
              ['/runaway-agent-cost-calculator', 'Runaway cost calculator', 'Model retry storms, fanout, MCP tool calls, and detection delay.'],
              ['/agent-spend-policy-template', 'Spend policy template', 'Generate YAML/JSON budgets, MCP caps, revocation, and audit policy.'],
              ['/openai-budget-policy-generator', 'OpenAI budget policy', 'Create per-model, per-route, per-agent, and per-session OpenAI limits.'],
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
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">AI agent cost control questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is AI agent cost control?</h3>
              <p className="text-gray-400 leading-relaxed">
                AI agent cost control is the practice of attributing, budgeting, limiting, and preserving receipts for autonomous agent API and tool spend before requests execute.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why are provider dashboards not enough for AI agent spend control?</h3>
              <p className="text-gray-400 leading-relaxed">
                Dashboards report spend after the fact. Autonomous agents can retry, loop, and delegate fast enough that budget enforcement must happen inline before upstream API calls complete.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does SatGate enforce AI agent budgets?</h3>
              <p className="text-gray-400 leading-relaxed">
                SatGate checks agent identity, route, tool cost, remaining budget, revocation status, and policy in the request path before forwarding each request.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is the difference between AI agent cost control and LLM cost management?</h3>
              <p className="text-gray-400 leading-relaxed">
                LLM cost management usually tracks model and token spend after usage occurs. AI agent cost control adds request-path enforcement across agents, MCP tools, paid APIs, delegated sub-agents, budgets, revocation, and audit before cost is created.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Can rate limits control AI agent costs?</h3>
              <p className="text-gray-400 leading-relaxed">
                Rate limits control request frequency, not economic exposure. AI agent cost control needs per-request pricing, remaining-budget checks, tool-level caps, and request-path decisions that account for expensive model or MCP tool calls.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Which policies should AI agent cost control include?</h3>
              <p className="text-gray-400 leading-relaxed">
                A practical policy should include tenant and agent identity, route and tool scope, per-request and session budgets, MCP tool caps, delegated sub-agent limits, expiry, revocation triggers, kill switches, and audit fields.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">When should a team add AI agent budget enforcement?</h3>
              <p className="text-gray-400 leading-relaxed">
                Add budget enforcement before agents receive access to paid APIs, premium models, MCP tools, data providers, or external services where retries, loops, or delegation can create real cost.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-purple-900/60 bg-gradient-to-br from-purple-950/30 to-cyan-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">Find your avoidable agent spend</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            Use the SatGate ROI calculator to model ghost spend, runaway loops, wasted tool calls, and the payback period for request-path budget enforcement with Policy-to-Proof receipt coverage.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/roi-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Open the ROI calculator <ArrowRight size={18} />
            </Link>
            <Link href="/ai-agent-runaway-spend-index" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-purple-500 transition">
              Read runaway spend index <ArrowRight size={18} />
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

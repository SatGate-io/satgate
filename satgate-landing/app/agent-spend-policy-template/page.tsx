'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { ArrowRight, ClipboardList, Copy, Gauge, KeyRound, ReceiptText, ShieldCheck, Wrench } from 'lucide-react';

const tiers = {
  prototype: {
    daily: 50,
    session: 5,
    request: 0.25,
    tool: 1,
    delegate: false,
    mode: 'observe_then_control',
  },
  production: {
    daily: 500,
    session: 25,
    request: 1,
    tool: 5,
    delegate: true,
    mode: 'control',
  },
  enterprise: {
    daily: 5000,
    session: 100,
    request: 5,
    tool: 25,
    delegate: true,
    mode: 'control_and_charge',
  },
};

const workloads = {
  coding: { route: '/v1/responses', model: 'premium-code-model', tool: 'repo_search', risk: 'medium' },
  support: { route: '/v1/chat/completions', model: 'fast-support-model', tool: 'crm_lookup', risk: 'low' },
  research: { route: '/v1/responses', model: 'premium-research-model', tool: 'web_search', risk: 'high' },
  mcp: { route: '/mcp/tools/*', model: 'agent-runtime', tool: 'expensive_mcp_tool', risk: 'high' },
};

type TierKey = keyof typeof tiers;
type WorkloadKey = keyof typeof workloads;

export default function AgentSpendPolicyTemplatePage() {
  const [tier, setTier] = useState<TierKey>('production');
  const [workload, setWorkload] = useState<WorkloadKey>('coding');
  const [agentName, setAgentName] = useState('coding-agent');

  const policy = useMemo(() => {
    const t = tiers[tier];
    const w = workloads[workload];
    const slug = agentName.trim().toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-|-$/g, '') || 'agent';
    const yaml = `policy: ${slug}-spend-policy\nmode: ${t.mode}\nagent:\n  id: ${slug}\n  workload: ${workload}\n  risk_tier: ${w.risk}\nidentity:\n  require_agent_id: true\n  require_tenant_id: true\n  require_task_id: true\nbudgets:\n  daily_usd: ${t.daily}\n  session_usd: ${t.session}\n  per_request_usd: ${t.request}\n  on_exhausted: block\nroutes:\n  - path: ${w.route}\n    model: ${w.model}\n    max_request_usd: ${t.request}\nmcp_tools:\n  - name: ${w.tool}\n    max_call_usd: ${t.tool}\n    risk_tier: ${w.risk}\n    on_budget_exhausted: block\ndelegation:\n  allowed: ${t.delegate}\n  child_budget_pct: 20\n  child_expiry_minutes: 60\ncredentials:\n  type: revocable_capability\n  expiry_minutes: 240\n  revoke_on_loop: true\n  revoke_on_policy_violation: true\naudit:\n  include:\n    - tenant_id\n    - agent_id\n    - task_id\n    - route\n    - model\n    - mcp_tool\n    - estimated_cost_usd\n    - remaining_budget_usd\n    - policy_decision\n    - credential_id\n    - revocation_state`;
    const json = JSON.stringify({
      policy: `${slug}-spend-policy`,
      mode: t.mode,
      agent: { id: slug, workload, risk_tier: w.risk },
      identity: { require_agent_id: true, require_tenant_id: true, require_task_id: true },
      budgets: { daily_usd: t.daily, session_usd: t.session, per_request_usd: t.request, on_exhausted: 'block' },
      routes: [{ path: w.route, model: w.model, max_request_usd: t.request }],
      mcp_tools: [{ name: w.tool, max_call_usd: t.tool, risk_tier: w.risk, on_budget_exhausted: 'block' }],
      delegation: { allowed: t.delegate, child_budget_pct: 20, child_expiry_minutes: 60 },
      credentials: { type: 'revocable_capability', expiry_minutes: 240, revoke_on_loop: true, revoke_on_policy_violation: true },
      audit: { include: ['tenant_id', 'agent_id', 'task_id', 'route', 'model', 'mcp_tool', 'estimated_cost_usd', 'remaining_budget_usd', 'policy_decision', 'credential_id', 'revocation_state'] },
    }, null, 2);
    return { yaml, json };
  }, [agentName, tier, workload]);

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'Agent Spend Policy Template',
    url: 'https://satgate.io/agent-spend-policy-template',
    description: 'Generate copyable YAML and JSON policy templates for AI agent budgets, MCP tool costs, delegation, revocation, and audit fields.',
    datePublished: '2026-04-12',
    dateModified: '2026-05-05',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI agent spend policy template' },
      { '@type': 'Thing', name: 'request-path budget enforcement' },
      { '@type': 'Thing', name: 'MCP tool cost policy' },
      { '@type': 'Thing', name: 'agent delegation limits' },
      { '@type': 'Thing', name: 'revocation and audit policy' },
    ],
    audience: { '@type': 'Audience', audienceType: 'AI engineering, platform, API, security, and FinOps teams' },
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'Agent Spend Policy Template',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/agent-spend-policy-template',
    description: 'Generate copyable YAML and JSON policy templates for AI agent budgets, MCP tool costs, delegation, revocation, and audit fields.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-05',
    audience: webPageJsonLd.audience,
    featureList: ['YAML spend policy generation', 'JSON spend policy generation', 'MCP tool cost caps', 'Delegation limit templates', 'Revocation and audit field templates'],
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const checklistJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'Agent spend policy checklist',
    description: 'Required controls for an enforceable AI agent spend policy across budgets, MCP tools, credentials, delegation, audit, and operating mode.',
    itemListElement: [
      ['Budgets', 'Daily, session, per-request, route, model, and MCP tool caps.'],
      ['Tools', 'Per-tool price, risk tier, deny behavior, and audit fields for MCP servers.'],
      ['Credentials', 'Scoped capabilities with expiry, revocation, and loop-kill behavior.'],
      ['Delegation', 'Sub-agent budget percentages, shorter expiry, and attenuated authority.'],
      ['Audit', 'Tenant, agent, task, route, model, tool, cost, budget, and decision fields.'],
      ['Mode', 'Observe first, Control when trusted thresholds are clear, Charge when robot customers pay.'],
    ].map(([name, description], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name,
      description,
    })),
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'AI Agent Cost Control', item: 'https://satgate.io/ai-agent-cost-control' },
      { '@type': 'ListItem', position: 3, name: 'Agent Spend Policy Template', item: 'https://satgate.io/agent-spend-policy-template' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is an agent spend policy?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'An agent spend policy defines the budgets, per-request limits, route rules, MCP tool caps, delegation limits, revocation behavior, and audit fields that should be checked before autonomous agent requests execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why should spend policy be enforced in the request path?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Autonomous agents can loop, retry, delegate, and call expensive tools faster than dashboards or alerts can react. Request-path enforcement blocks over-budget activity before cost is created.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does this template relate to SatGate?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate is the economic firewall that can enforce spend policy in the request path across model calls, APIs, MCP tools, revocable credentials, delegation, audit, and L402 payments.',
        },
      },
      {
        '@type': 'Question',
        name: 'What fields should every AI agent spend policy include?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Every AI agent spend policy should include tenant, agent, task, route, model, tool, per-request cap, session budget, daily budget, delegation limits, credential expiry, revocation triggers, and audit fields.',
        },
      },
      {
        '@type': 'Question',
        name: 'Should agent spend policy start in Observe or Control mode?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Most teams should start in Observe to learn normal spend patterns, then move high-risk agents, expensive tools, and external-facing workflows into Control mode with hard caps and revocation.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(checklistJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(34,211,238,0.18),transparent_32%),radial-gradient(circle_at_82%_10%,rgba(168,85,247,0.16),transparent_34%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <ClipboardList size={16} /> Free agent spend policy template
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            Agent Spend Policy Template
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Generate a practical YAML or JSON policy for AI agent budgets, per-request caps, MCP tool costs, delegation limits, revocation, and audit fields.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <a href="#template" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Generate template <ArrowRight size={18} />
            </a>
            <Link href="/ai-api-budget-enforcement" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              AI API budget enforcement
            </Link>
          </div>
        </div>
      </section>

      <section id="template" className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[0.75fr_1fr]">
        <div className="space-y-5">
          <h2 className="text-3xl font-bold text-white">Configure the policy</h2>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">Agent name</span>
            <input value={agentName} onChange={(event) => setAgentName(event.target.value)} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500" />
          </label>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">Environment</span>
            <select value={tier} onChange={(event) => setTier(event.target.value as TierKey)} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500">
              <option value="prototype">Prototype</option>
              <option value="production">Production</option>
              <option value="enterprise">Enterprise</option>
            </select>
          </label>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">Workload</span>
            <select value={workload} onChange={(event) => setWorkload(event.target.value as WorkloadKey)} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500">
              <option value="coding">Coding agent</option>
              <option value="support">Support agent</option>
              <option value="research">Research agent</option>
              <option value="mcp">MCP tool workflow</option>
            </select>
          </label>
          <div className="rounded-xl border border-cyan-900/50 bg-cyan-950/10 p-5 text-sm leading-relaxed text-cyan-100">
            This template is deliberately conservative. Start in Observe, then enforce Control once you know normal request and tool economics.
          </div>
        </div>

        <div className="grid gap-5">
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-5">
            <div className="mb-3 flex items-center justify-between gap-4">
              <h2 className="text-xl font-bold text-white">YAML policy</h2>
              <button onClick={() => navigator.clipboard?.writeText(policy.yaml)} className="inline-flex items-center gap-2 rounded-lg border border-gray-700 px-3 py-2 text-sm font-semibold text-white transition hover:border-cyan-500"><Copy size={15} /> Copy</button>
            </div>
            <pre className="max-h-[520px] overflow-auto rounded-xl bg-black p-5 text-sm leading-relaxed text-cyan-100"><code>{policy.yaml}</code></pre>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-5">
            <div className="mb-3 flex items-center justify-between gap-4">
              <h2 className="text-xl font-bold text-white">JSON policy</h2>
              <button onClick={() => navigator.clipboard?.writeText(policy.json)} className="inline-flex items-center gap-2 rounded-lg border border-gray-700 px-3 py-2 text-sm font-semibold text-white transition hover:border-cyan-500"><Copy size={15} /> Copy</button>
            </div>
            <pre className="max-h-[520px] overflow-auto rounded-xl bg-black p-5 text-sm leading-relaxed text-purple-100"><code>{policy.json}</code></pre>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">What a useful agent spend policy covers</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            A policy is useful only if it can be enforced before the next model, API, or MCP tool call. These are the fields that make spend governable.
          </p>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              [Gauge, 'Budgets', 'Daily, session, per-request, route, model, and MCP tool caps.'],
              [Wrench, 'Tools', 'Per-tool price, risk tier, deny behavior, and audit fields for MCP servers.'],
              [KeyRound, 'Credentials', 'Scoped capabilities with expiry, revocation, and loop-kill behavior.'],
              [ShieldCheck, 'Delegation', 'Sub-agent budget percentages, shorter expiry, and attenuated authority.'],
              [ReceiptText, 'Audit', 'Tenant, agent, task, route, model, tool, cost, budget, and decision fields.'],
              [ClipboardList, 'Mode', 'Observe first, Control when trusted thresholds are clear, Charge when robot customers pay.'],
            ].map(([Icon, title, body]) => {
              const CardIcon = Icon as typeof Gauge;
              return (
                <div key={String(title)} className="rounded-2xl border border-gray-800 bg-black p-6">
                  <CardIcon className="mb-4 text-cyan-300" size={30} />
                  <h3 className="mb-2 text-xl font-bold text-white">{String(title)}</h3>
                  <p className="leading-relaxed text-gray-400">{String(body)}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Agent spend policy questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is an agent spend policy?</h3>
              <p className="text-gray-400 leading-relaxed">
                An agent spend policy defines the budgets, per-request limits, route rules, MCP tool caps, delegation limits, revocation behavior, and audit fields that should be checked before autonomous agent requests execute.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why should spend policy be enforced in the request path?</h3>
              <p className="text-gray-400 leading-relaxed">
                Autonomous agents can loop, retry, delegate, and call expensive tools faster than dashboards or alerts can react. Request-path enforcement blocks over-budget activity before cost is created.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does this template relate to SatGate?</h3>
              <p className="text-gray-400 leading-relaxed">
                SatGate is the economic firewall that can enforce spend policy in the request path across model calls, APIs, MCP tools, revocable credentials, delegation, audit, and L402 payments.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What fields should every AI agent spend policy include?</h3>
              <p className="text-gray-400 leading-relaxed">
                Every AI agent spend policy should include tenant, agent, task, route, model, tool, per-request cap, session budget, daily budget, delegation limits, credential expiry, revocation triggers, and audit fields.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Should agent spend policy start in Observe or Control mode?</h3>
              <p className="text-gray-400 leading-relaxed">
                Most teams should start in Observe to learn normal spend patterns, then move high-risk agents, expensive tools, and external-facing workflows into Control mode with hard caps and revocation.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-purple-900/60 bg-gradient-to-br from-purple-950/30 to-cyan-950/25 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Turn templates into enforcement.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            SatGate sits in the request path and applies these policies before agents spend money, call MCP tools, delegate work, or unlock paid API access.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/agent-spending-limits" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Agent spending limits <ArrowRight size={18} />
            </Link>
            <Link href="/mcp-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              MCP cost control
            </Link>
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Grade readiness
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

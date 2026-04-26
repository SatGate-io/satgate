'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { ArrowRight, ClipboardList, Gauge, KeyRound, Route, ShieldCheck } from 'lucide-react';

type Mode = 'observe' | 'control';
type Risk = 'low' | 'medium' | 'high';

type SelectOption<T extends string> = { label: string; value: T };

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block rounded-xl border border-gray-800 bg-black/50 p-5">
      <span className="mb-3 block font-medium text-gray-300">{label}</span>
      {children}
    </label>
  );
}

function Select<T extends string>({ value, options, onChange }: { value: T; options: SelectOption<T>[]; onChange: (value: T) => void }) {
  return (
    <select
      value={value}
      onChange={(event) => onChange(event.target.value as T)}
      className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400"
    >
      {options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
    </select>
  );
}

export default function OpenAiBudgetPolicyGeneratorPage() {
  const [workflow, setWorkflow] = useState('research-agent');
  const [mode, setMode] = useState<Mode>('control');
  const [risk, setRisk] = useState<Risk>('medium');
  const [dailyBudget, setDailyBudget] = useState(50);
  const [perRequest, setPerRequest] = useState(0.75);
  const [sessionBudget, setSessionBudget] = useState(10);
  const [premiumModelPct, setPremiumModelPct] = useState(15);

  const policy = useMemo(() => {
    const escalation = risk === 'high' ? 'block_and_revoke' : risk === 'medium' ? 'route_to_economy_model' : 'log_and_allow';
    const auditLevel = mode === 'control' ? 'decision_and_cost' : 'cost_attribution_only';
    return `agent_policy: ${workflow || 'research-agent'}\nprovider: openai\nmode: ${mode}\nrisk_profile: ${risk}\nbudgets:\n  daily: ${dailyBudget.toFixed(2)} USD\n  per_session: ${sessionBudget.toFixed(2)} USD\n  per_request: ${perRequest.toFixed(2)} USD\nmodel_policy:\n  default_route: economy\n  premium_model_budget_share: ${premiumModelPct}%\n  require_justification_for:\n    - gpt-5.5\n    - gpt-5.5-pro\ncontrols:\n  on_daily_budget_exhausted: block\n  on_per_request_exceeded: ${escalation}\n  on_loop_detected: revoke_session_capability\n  on_unknown_agent: deny\ncapability:\n  expiry: task_or_24h\n  delegation: attenuated_only\n  child_budget_max: ${(sessionBudget * 0.25).toFixed(2)} USD\naudit:\n  level: ${auditLevel}\n  include:\n    - tenant\n    - agent\n    - workflow\n    - model\n    - estimated_cost\n    - remaining_budget\n    - policy_decision\n    - upstream_status`;
  }, [dailyBudget, mode, perRequest, premiumModelPct, risk, sessionBudget, workflow]);

  const jsonPolicy = useMemo(() => ({
    agent_policy: workflow || 'research-agent',
    provider: 'openai',
    mode,
    risk_profile: risk,
    budgets: {
      daily_usd: dailyBudget,
      per_session_usd: sessionBudget,
      per_request_usd: perRequest,
    },
    model_policy: {
      default_route: 'economy',
      premium_model_budget_share_percent: premiumModelPct,
      require_justification_for: ['gpt-5.5', 'gpt-5.5-pro'],
    },
    controls: {
      on_daily_budget_exhausted: 'block',
      on_per_request_exceeded: risk === 'high' ? 'block_and_revoke' : risk === 'medium' ? 'route_to_economy_model' : 'log_and_allow',
      on_loop_detected: 'revoke_session_capability',
      on_unknown_agent: 'deny',
    },
    audit: ['tenant', 'agent', 'workflow', 'model', 'estimated_cost', 'remaining_budget', 'policy_decision', 'upstream_status'],
  }), [dailyBudget, mode, perRequest, premiumModelPct, risk, sessionBudget, workflow]);

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'OpenAI API Budget Limit Generator',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/openai-budget-policy-generator',
    description: 'Generate OpenAI API budget policy for autonomous agents, workflows, model routes, per-request caps, daily budgets, and audit controls.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const howToJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'HowTo',
    name: 'How to create OpenAI API budget limits for AI agents',
    description: 'Use the generator to produce request-path budget policy for OpenAI API calls made by autonomous agents.',
    totalTime: 'PT5M',
    step: [
      { '@type': 'HowToStep', name: 'Name the workflow', text: 'Identify the agent, task, tenant, or workflow that should receive its own OpenAI budget policy.' },
      { '@type': 'HowToStep', name: 'Choose Observe or Control', text: 'Use Observe to measure spend first, or Control to block and route calls when budget policy is exceeded.' },
      { '@type': 'HowToStep', name: 'Set budget limits', text: 'Define daily, session, per-request, and premium-model budget ceilings for the workflow.' },
      { '@type': 'HowToStep', name: 'Copy policy into the control plane', text: 'Use the generated YAML or JSON as a starting policy for SatGate request-path enforcement.' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(howToJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <ClipboardList size={16} /> OpenAI spend policy generator
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            OpenAI API Budget Limit Generator
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Generate a request-path budget policy for AI agents calling OpenAI: per-request caps, daily spend limits, session budgets, model routing, revocation, and audit rules.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/blog/how-to-add-budget-limits-to-openai-api-calls" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Read the OpenAI budget guide <ArrowRight size={18} />
            </Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              See agent cost control
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[0.9fr_1.1fr]">
        <div className="space-y-4">
          <Field label="Agent or workflow name">
            <input value={workflow} onChange={(event) => setWorkflow(event.target.value)} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400" />
          </Field>
          <Field label="Policy mode">
            <Select value={mode} onChange={setMode} options={[{ label: 'Observe — measure first', value: 'observe' }, { label: 'Control — enforce limits', value: 'control' }]} />
          </Field>
          <Field label="Workflow risk">
            <Select value={risk} onChange={setRisk} options={[{ label: 'Low — internal helper', value: 'low' }, { label: 'Medium — production workflow', value: 'medium' }, { label: 'High — autonomous / external-facing', value: 'high' }]} />
          </Field>
          <Field label="Daily budget (USD)">
            <input type="number" min="1" value={dailyBudget} onChange={(event) => setDailyBudget(Number(event.target.value))} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400" />
          </Field>
          <Field label="Session budget (USD)">
            <input type="number" min="1" value={sessionBudget} onChange={(event) => setSessionBudget(Number(event.target.value))} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400" />
          </Field>
          <Field label="Per-request cap (USD)">
            <input type="number" min="0.01" step="0.01" value={perRequest} onChange={(event) => setPerRequest(Number(event.target.value))} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400" />
          </Field>
          <Field label="Premium model budget share (%)">
            <input type="number" min="0" max="100" value={premiumModelPct} onChange={(event) => setPremiumModelPct(Number(event.target.value))} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400" />
          </Field>
        </div>

        <div className="space-y-6">
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6 md:p-8">
            <h2 className="mb-4 text-2xl font-bold text-white">Generated YAML policy</h2>
            <pre className="max-h-[620px] overflow-auto rounded-xl border border-gray-800 bg-black p-5 text-sm leading-relaxed text-gray-300"><code>{policy}</code></pre>
          </div>
          <details className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <summary className="cursor-pointer font-bold text-white">Show JSON version</summary>
            <pre className="mt-4 overflow-auto rounded-xl border border-gray-800 bg-black p-5 text-sm leading-relaxed text-gray-300"><code>{JSON.stringify(jsonPolicy, null, 2)}</code></pre>
          </details>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-8 text-3xl font-bold text-white">What good OpenAI budget policy controls</h2>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
            {[
              [Gauge, 'Spend ceilings', 'Daily, session, per-request, and premium-model limits by agent or workflow.'],
              [Route, 'Model routing', 'Route routine calls to economy models and require justification for premium models.'],
              [KeyRound, 'Scoped capability', 'Expire and revoke agent credentials without rotating broad provider keys.'],
              [ShieldCheck, 'Inline enforcement', 'Block, route, revoke, or audit before OpenAI API calls execute.'],
            ].map(([Icon, title, body]) => {
              const TypedIcon = Icon as typeof Gauge;
              return (
                <div key={title as string} className="rounded-xl border border-gray-800 bg-black p-6">
                  <TypedIcon className="mb-4 text-cyan-300" size={28} />
                  <h3 className="mb-2 text-lg font-bold text-white">{title as string}</h3>
                  <p className="leading-relaxed text-gray-400">{body as string}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Budget policy belongs in the request path</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            Provider dashboards explain the bill after the fact. SatGate checks agent identity, model route, estimated cost, remaining budget, and revocation status before forwarding OpenAI requests.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Learn economic firewalls <ArrowRight size={18} />
            </Link>
            <Link href="/runaway-agent-cost-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Calculate runaway cost
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

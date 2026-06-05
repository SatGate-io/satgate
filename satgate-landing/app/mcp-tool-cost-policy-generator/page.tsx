'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { ArrowRight, Cable, ClipboardList, DollarSign, Eye, ShieldAlert, Wrench } from 'lucide-react';

type Client = 'cursor' | 'claude_desktop' | 'claude_code' | 'openclaw' | 'custom';
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
    <select value={value} onChange={(event) => onChange(event.target.value as T)} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400">
      {options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
    </select>
  );
}

export default function McpToolCostPolicyGeneratorPage() {
  const [client, setClient] = useState<Client>('cursor');
  const [server, setServer] = useState('github-tools');
  const [mode, setMode] = useState<Mode>('control');
  const [risk, setRisk] = useState<Risk>('medium');
  const [sessionBudget, setSessionBudget] = useState(8);
  const [perToolCall, setPerToolCall] = useState(0.25);
  const [expensiveTool, setExpensiveTool] = useState('browser_search');
  const [expensiveToolCap, setExpensiveToolCap] = useState(2);

  const policy = useMemo(() => {
    const riskAction = risk === 'high' ? 'block_and_revoke_capability' : risk === 'medium' ? 'require_budget_and_audit' : 'log_cost_only';
    return `mcp_policy: ${client}_${server || 'tools'}\nclient: ${client}\nmcp_server: ${server || 'github-tools'}\nmode: ${mode}\nrisk_profile: ${risk}\nbudget:\n  per_session: ${sessionBudget.toFixed(2)} USD\n  per_tool_call_default: ${perToolCall.toFixed(2)} USD\n  expensive_tool_cap: ${expensiveToolCap.toFixed(2)} USD\ntools:\n  repo_search:\n    decision: allow\n    estimated_cost: ${(perToolCall * 0.5).toFixed(2)} USD\n  issue_create:\n    decision: allow\n    estimated_cost: ${perToolCall.toFixed(2)} USD\n  ${expensiveTool || 'browser_search'}:\n    decision: ${mode === 'control' ? 'budget_required' : 'observe'}\n    max_cost: ${expensiveToolCap.toFixed(2)} USD\n  deploy_prod:\n    decision: deny\ncontrols:\n  on_session_budget_exhausted: block\n  on_tool_cost_unknown: ${riskAction}\n  on_loop_detected: block_and_revoke_session\n  on_sensitive_tool: require_explicit_policy\naudit:\n  include:\n    - tenant\n    - agent\n    - client\n    - mcp_server\n    - tool\n    - estimated_cost\n    - remaining_budget\n    - policy_decision\n    - outcome\nevidence_pack:\n  required: true\n  receipt_id: generated_per_tool_call\n  evidence_pack_id: generated_per_workflow_or_export\n  decision_reason: required\n  policy_version: required\n  include_paid_rail_context: true`;
  }, [client, expensiveTool, expensiveToolCap, mode, perToolCall, risk, server, sessionBudget]);

  const jsonPolicy = useMemo(() => ({
    mcp_policy: `${client}_${server || 'tools'}`,
    client,
    mcp_server: server || 'github-tools',
    mode,
    risk_profile: risk,
    budget: {
      per_session_usd: sessionBudget,
      per_tool_call_default_usd: perToolCall,
      expensive_tool_cap_usd: expensiveToolCap,
    },
    tools: {
      repo_search: { decision: 'allow', estimated_cost_usd: Number((perToolCall * 0.5).toFixed(2)) },
      issue_create: { decision: 'allow', estimated_cost_usd: perToolCall },
      [expensiveTool || 'browser_search']: { decision: mode === 'control' ? 'budget_required' : 'observe', max_cost_usd: expensiveToolCap },
      deploy_prod: { decision: 'deny' },
    },
    evidence_pack: { required: true, receipt_id: 'generated_per_tool_call', evidence_pack_id: 'generated_per_workflow_or_export', decision_reason: 'required', policy_version: 'required', include_paid_rail_context: true },
    controls: {
      on_session_budget_exhausted: 'block',
      on_tool_cost_unknown: risk === 'high' ? 'block_and_revoke_capability' : risk === 'medium' ? 'require_budget_and_audit' : 'log_cost_only',
      on_loop_detected: 'block_and_revoke_session',
      on_sensitive_tool: 'require_explicit_policy',
    },
  }), [client, expensiveTool, expensiveToolCap, mode, perToolCall, risk, server, sessionBudget]);

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'MCP Tool Cost Policy Generator',
    url: 'https://satgate.io/mcp-tool-cost-policy-generator',
    description: 'Generate MCP tool cost policy for per-tool budgets, session caps, risk actions, revocation, and Evidence Pack receipts.',
    datePublished: '2026-04-12',
    dateModified: '2026-06-04',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'MCP tool cost policy generator' },
      { '@type': 'Thing', name: 'per-tool MCP budgets' },
      { '@type': 'Thing', name: 'MCP session budget enforcement' },
      { '@type': 'Thing', name: 'unknown tool cost risk actions' },
      { '@type': 'Thing', name: 'request-path MCP Evidence Pack receipts' },
    ],
    audience: { '@type': 'Audience', audienceType: 'AI engineering, platform, API, security, and FinOps teams using MCP' },
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'MCP Tool Cost Policy Generator',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/mcp-tool-cost-policy-generator',
    description: 'Generate MCP tool cost policy for per-tool budgets, session caps, risk actions, revocation, and Evidence Pack receipts.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-06-04',
    audience: webPageJsonLd.audience,
    featureList: ['MCP policy YAML generation', 'MCP policy JSON generation', 'Per-tool budget controls', 'Unknown cost risk actions', 'Revocation and Evidence Pack policy templates'],
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'What is an MCP tool cost policy?', acceptedAnswer: { '@type': 'Answer', text: 'An MCP tool cost policy assigns spend limits, allowed actions, risk rules, revocation behavior, and Evidence Pack receipt fields to tool calls made through Model Context Protocol.' } },
      { '@type': 'Question', name: 'Why do MCP tools need per-tool prices?', acceptedAnswer: { '@type': 'Answer', text: 'MCP tools can hide paid APIs, searches, browser sessions, compute jobs, or data calls. Pricing each tool lets budget enforcement happen before expensive work executes.' } },
      { '@type': 'Question', name: 'Can SatGate govern Cursor or Claude MCP tool use?', acceptedAnswer: { '@type': 'Answer', text: 'Yes. SatGate can sit around MCP-capable clients such as Cursor, Claude Desktop, Claude Code, OpenClaw, and custom agents to enforce budgets and audit tool calls.' } },
      { '@type': 'Question', name: 'What should happen when an MCP tool cost is unknown?', acceptedAnswer: { '@type': 'Answer', text: 'Unknown MCP tool costs should trigger a conservative policy action such as observe-only logging, explicit budget review, blocking, or revoking the session capability depending on risk tier.' } },
      { '@type': 'Question', name: 'Which MCP tools should be marked high risk?', acceptedAnswer: { '@type': 'Answer', text: 'High-risk MCP tools include browser automation, paid search, code execution, cloud write actions, data export, production deploys, premium APIs, and any tool that can spend money or change state.' } },
    ],
  };


  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'MCP Governance', item: 'https://satgate.io/mcp' },
      { '@type': 'ListItem', position: 3, name: 'MCP Tool Cost Policy Generator', item: 'https://satgate.io/mcp-tool-cost-policy-generator' },
    ],
  };
  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <Cable size={16} /> MCP budget enforcement tool
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">MCP Tool Cost Policy Generator</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Generate request-path policy for MCP tools: per-tool prices, session budgets, expensive-tool caps, denial rules, revocation behavior, and Evidence Pack receipts before agents execute paid work.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/mcp-governance" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              See MCP governance <ArrowRight size={18} />
            </Link>
            <Link href="/blog/mcp-budget-enforcement-guide" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Read budget guide
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[0.9fr_1.1fr]">
        <div className="space-y-4">
          <Field label="MCP client"><Select value={client} onChange={setClient} options={[{ label: 'Cursor', value: 'cursor' }, { label: 'Claude Desktop', value: 'claude_desktop' }, { label: 'Claude Code', value: 'claude_code' }, { label: 'OpenClaw', value: 'openclaw' }, { label: 'Custom agent runtime', value: 'custom' }]} /></Field>
          <Field label="MCP server name"><input value={server} onChange={(event) => setServer(event.target.value)} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400" /></Field>
          <Field label="Policy mode"><Select value={mode} onChange={setMode} options={[{ label: 'Observe — attribute costs', value: 'observe' }, { label: 'Control — enforce budgets', value: 'control' }]} /></Field>
          <Field label="Risk tier"><Select value={risk} onChange={setRisk} options={[{ label: 'Low', value: 'low' }, { label: 'Medium', value: 'medium' }, { label: 'High', value: 'high' }]} /></Field>
          <Field label="Session budget (USD)"><input type="number" min="1" value={sessionBudget} onChange={(event) => setSessionBudget(Number(event.target.value))} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400" /></Field>
          <Field label="Default cost per tool call (USD)"><input type="number" min="0.01" step="0.01" value={perToolCall} onChange={(event) => setPerToolCall(Number(event.target.value))} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400" /></Field>
          <Field label="Expensive tool name"><input value={expensiveTool} onChange={(event) => setExpensiveTool(event.target.value)} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400" /></Field>
          <Field label="Expensive tool cap (USD)"><input type="number" min="0.01" step="0.01" value={expensiveToolCap} onChange={(event) => setExpensiveToolCap(Number(event.target.value))} className="w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-3 text-white outline-none focus:border-cyan-400" /></Field>
        </div>

        <div className="space-y-6">
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6 md:p-8">
            <h2 className="mb-4 text-2xl font-bold text-white">Generated MCP policy</h2>
            <pre className="max-h-[680px] overflow-auto rounded-xl border border-gray-800 bg-black p-5 text-sm leading-relaxed text-gray-300"><code>{policy}</code></pre>
          </div>
          <details className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <summary className="cursor-pointer font-bold text-white">Show JSON version</summary>
            <pre className="mt-4 overflow-auto rounded-xl border border-gray-800 bg-black p-5 text-sm leading-relaxed text-gray-300"><code>{JSON.stringify(jsonPolicy, null, 2)}</code></pre>
          </details>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-8 text-3xl font-bold text-white">What the policy should control</h2>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
            {[
              [DollarSign, 'Per-tool economics', 'Attach cost to searches, browser sessions, cloud tasks, code agents, and paid APIs.'],
              [ShieldAlert, 'Risk actions', 'Block, route, revoke, or require explicit policy when unknown or sensitive tools appear.'],
              [Eye, 'Evidence Pack receipts', 'Record agent, MCP server, tool, cost, remaining budget, policy decision, outcome, and paid-rail context.'],
              [Wrench, 'Server unchanged', 'Wrap governance around existing MCP servers without rewriting every tool implementation.'],
            ].map(([Icon, title, body]) => {
              const TypedIcon = Icon as typeof DollarSign;
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

      <section className="border-t border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">MCP tool policy questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is an MCP tool cost policy?</h3>
              <p className="text-gray-400 leading-relaxed">
                An MCP tool cost policy assigns spend limits, allowed actions, risk rules, revocation behavior, and Evidence Pack receipt fields to tool calls made through Model Context Protocol.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why do MCP tools need per-tool prices?</h3>
              <p className="text-gray-400 leading-relaxed">
                MCP tools can hide paid APIs, searches, browser sessions, compute jobs, or data calls. Pricing each tool lets budget enforcement happen before expensive work executes.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Can SatGate govern Cursor or Claude MCP tool use?</h3>
              <p className="text-gray-400 leading-relaxed">
                Yes. SatGate can sit around MCP-capable clients such as Cursor, Claude Desktop, Claude Code, OpenClaw, and custom agents to enforce budgets and audit tool calls.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What should happen when an MCP tool cost is unknown?</h3>
              <p className="text-gray-400 leading-relaxed">
                Unknown MCP tool costs should trigger a conservative policy action such as observe-only logging, explicit budget review, blocking, or revoking the session capability depending on risk tier.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Which MCP tools should be marked high risk?</h3>
              <p className="text-gray-400 leading-relaxed">
                High-risk MCP tools include browser automation, paid search, code execution, cloud write actions, data export, production deploys, premium APIs, and any tool that can spend money or change state.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">MCP makes tools easy. SatGate makes them governable.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            Route MCP traffic through SatGate to observe, control, and preserve Evidence Pack receipts before autonomous agents trigger paid or risky work.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/mcp-governance" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Govern MCP tools <ArrowRight size={18} />
            </Link>
            <Link href="/openai-budget-policy-generator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Generate OpenAI budget policy
            </Link>
            <Link href="/mcp-budget-enforcement" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              MCP budget enforcement
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

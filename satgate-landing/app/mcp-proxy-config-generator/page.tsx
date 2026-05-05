'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { ArrowRight, Cable, ClipboardList, Copy, Gauge, KeyRound, ReceiptText, ShieldCheck, Wrench } from 'lucide-react';

const clients = {
  cursor: { label: 'Cursor', command: 'satgate-mcp-proxy', configPath: '.cursor/mcp.json' },
  claudeDesktop: { label: 'Claude Desktop', command: 'satgate-mcp-proxy', configPath: 'claude_desktop_config.json' },
  claudeCode: { label: 'Claude Code', command: 'satgate-mcp-proxy', configPath: '.mcp.json' },
  openclaw: { label: 'OpenClaw', command: 'satgate mcp proxy', configPath: 'openclaw.yaml' },
  custom: { label: 'Custom MCP client', command: 'satgate-mcp-proxy', configPath: 'mcp-client.json' },
};

const modes = {
  observe: { label: 'Observe', block: false, charge: false, audit: 'full' },
  control: { label: 'Control', block: true, charge: false, audit: 'full' },
  charge: { label: 'Control + Charge', block: true, charge: true, audit: 'full' },
};

type ClientKey = keyof typeof clients;
type ModeKey = keyof typeof modes;

export default function McpProxyConfigGeneratorPage() {
  const [client, setClient] = useState<ClientKey>('cursor');
  const [mode, setMode] = useState<ModeKey>('control');
  const [serverName, setServerName] = useState('internal-tools');
  const [budgetUsd, setBudgetUsd] = useState(50);
  const [expensiveToolUsd, setExpensiveToolUsd] = useState(5);

  const generated = useMemo(() => {
    const c = clients[client];
    const m = modes[mode];
    const slug = serverName.trim().toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-|-$/g, '') || 'mcp-server';
    const json = JSON.stringify({
      mcpServers: {
        [`satgate-${slug}`]: {
          command: c.command,
          args: [
            '--upstream', `stdio://${slug}`,
            '--mode', mode,
            '--budget-usd', String(budgetUsd),
            '--max-tool-call-usd', String(expensiveToolUsd),
            '--audit', m.audit,
            ...(m.block ? ['--on-budget-exhausted', 'block'] : ['--on-budget-exhausted', 'observe']),
            ...(m.charge ? ['--charge', 'l402'] : []),
          ],
          env: {
            SATGATE_POLICY: `${slug}-mcp-policy`,
            SATGATE_REQUIRE_AGENT_ID: 'true',
            SATGATE_REQUIRE_TASK_ID: 'true',
          },
        },
      },
    }, null, 2);
    const yaml = `mcp_proxy:\n  name: satgate-${slug}\n  client: ${c.label}\n  config_path: ${c.configPath}\n  upstream: stdio://${slug}\n  mode: ${mode}\npolicy:\n  require_agent_id: true\n  require_task_id: true\n  session_budget_usd: ${budgetUsd}\n  max_tool_call_usd: ${expensiveToolUsd}\n  on_budget_exhausted: ${m.block ? 'block' : 'observe'}\n  charge: ${m.charge ? 'l402' : 'disabled'}\ncredentials:\n  type: revocable_capability\n  expiry_minutes: 240\n  allow_delegation: false\naudit:\n  level: ${m.audit}\n  include:\n    - tenant_id\n    - agent_id\n    - task_id\n    - mcp_server\n    - mcp_tool\n    - estimated_cost_usd\n    - remaining_budget_usd\n    - policy_decision\n    - revocation_state`;
    return { json, yaml, c };
  }, [budgetUsd, client, expensiveToolUsd, mode, serverName]);

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'MCP Proxy Config Generator',
    url: 'https://satgate.io/mcp-proxy-config-generator',
    description: 'Generate MCP proxy configs for Cursor, Claude, OpenClaw, and custom MCP clients with budgets, audit, revocation, and L402 Charge options.',
    datePublished: '2026-04-12',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'MCP proxy config generator' },
      { '@type': 'Thing', name: 'SatGate MCP proxy' },
      { '@type': 'Thing', name: 'Cursor and Claude MCP governance' },
      { '@type': 'Thing', name: 'MCP request-path budget enforcement' },
      { '@type': 'Thing', name: 'L402 Charge for MCP tools' },
    ],
    audience: { '@type': 'Audience', audienceType: 'AI engineering, platform, API, and security teams using MCP clients' },
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'MCP Proxy Config Generator',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/mcp-proxy-config-generator',
    description: 'Generate MCP proxy configs for Cursor, Claude, OpenClaw, and custom MCP clients with budgets, audit, revocation, and L402 Charge options.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-03',
    audience: webPageJsonLd.audience,
    featureList: ['Cursor MCP config generation', 'Claude MCP config generation', 'OpenClaw MCP config generation', 'Budget and audit policy generation', 'Optional L402 Charge config'],
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'MCP Governance', item: 'https://satgate.io/mcp' },
      { '@type': 'ListItem', position: 3, name: 'MCP Proxy Config Generator', item: 'https://satgate.io/mcp-proxy-config-generator' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is an MCP proxy?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'An MCP proxy sits between an AI agent client and MCP servers so tool calls can be observed, audited, budgeted, denied, revoked, or charged before expensive actions execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why put SatGate in front of MCP servers?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'MCP tools can trigger paid APIs, cloud work, searches, code agents, or sensitive data access. SatGate adds request-path economic governance around those tool calls.',
        },
      },
      {
        '@type': 'Question',
        name: 'Which MCP clients can use this pattern?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'The proxy pattern applies to Cursor, Claude Desktop, Claude Code, OpenClaw, and custom MCP-capable clients that can route tool calls through an MCP server command or proxy.',
        },
      },
      {
        '@type': 'Question',
        name: 'Should an MCP proxy start in Observe or Control mode?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Start an MCP proxy in Observe mode when mapping normal tool use, then move expensive, sensitive, external, or high-volume tools into Control mode with hard budgets and revocation.',
        },
      },
      {
        '@type': 'Question',
        name: 'What policy should an MCP proxy enforce?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'An MCP proxy should enforce agent and task identity, session budgets, per-tool cost caps, unknown-tool behavior, revocation triggers, audit fields, and optional L402 payment before tool execution.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(34,211,238,0.18),transparent_32%),radial-gradient(circle_at_82%_10%,rgba(168,85,247,0.16),transparent_34%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <Cable size={16} /> Free MCP proxy config generator
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            MCP Proxy Config Generator
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Generate a starter SatGate MCP proxy config for Cursor, Claude Desktop, Claude Code, OpenClaw, or custom MCP clients with budgets, audit, revocation, and optional L402 Charge.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <a href="#generator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Generate config <ArrowRight size={18} />
            </a>
            <Link href="/mcp-governance" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              MCP governance
            </Link>
          </div>
        </div>
      </section>

      <section id="generator" className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[0.72fr_1fr]">
        <div className="space-y-5">
          <h2 className="text-3xl font-bold text-white">Configure the proxy</h2>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">MCP client</span>
            <select value={client} onChange={(event) => setClient(event.target.value as ClientKey)} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500">
              {Object.entries(clients).map(([key, value]) => <option key={key} value={key}>{value.label}</option>)}
            </select>
          </label>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">SatGate mode</span>
            <select value={mode} onChange={(event) => setMode(event.target.value as ModeKey)} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500">
              {Object.entries(modes).map(([key, value]) => <option key={key} value={key}>{value.label}</option>)}
            </select>
          </label>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">Upstream MCP server name</span>
            <input value={serverName} onChange={(event) => setServerName(event.target.value)} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500" />
          </label>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">Session budget USD</span>
            <input type="number" min="1" value={budgetUsd} onChange={(event) => setBudgetUsd(Number(event.target.value))} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500" />
          </label>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">Max expensive tool call USD</span>
            <input type="number" min="0.01" step="0.01" value={expensiveToolUsd} onChange={(event) => setExpensiveToolUsd(Number(event.target.value))} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500" />
          </label>
        </div>

        <div className="grid gap-5">
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-5">
            <div className="mb-3 flex items-center justify-between gap-4">
              <h2 className="text-xl font-bold text-white">{generated.c.configPath}</h2>
              <button onClick={() => navigator.clipboard?.writeText(generated.json)} className="inline-flex items-center gap-2 rounded-lg border border-gray-700 px-3 py-2 text-sm font-semibold text-white transition hover:border-cyan-500"><Copy size={15} /> Copy</button>
            </div>
            <pre className="max-h-[520px] overflow-auto rounded-xl bg-black p-5 text-sm leading-relaxed text-cyan-100"><code>{generated.json}</code></pre>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-5">
            <div className="mb-3 flex items-center justify-between gap-4">
              <h2 className="text-xl font-bold text-white">Policy YAML</h2>
              <button onClick={() => navigator.clipboard?.writeText(generated.yaml)} className="inline-flex items-center gap-2 rounded-lg border border-gray-700 px-3 py-2 text-sm font-semibold text-white transition hover:border-cyan-500"><Copy size={15} /> Copy</button>
            </div>
            <pre className="max-h-[520px] overflow-auto rounded-xl bg-black p-5 text-sm leading-relaxed text-purple-100"><code>{generated.yaml}</code></pre>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">What the proxy makes governable</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            MCP turns tools into runtime authority. The proxy turns that authority into policy decisions before the tool runs.
          </p>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              [Wrench, 'Tool cost', 'Attach prices, budgets, and risk tiers to individual MCP tools.'],
              [Gauge, 'Spend caps', 'Block or observe calls when session, tool, or per-request budgets are exhausted.'],
              [KeyRound, 'Capabilities', 'Require scoped, revocable, expiring authority instead of broad ambient access.'],
              [ReceiptText, 'Audit', 'Record agent, task, server, tool, cost, budget, policy, and decision.'],
              [ShieldCheck, 'Revocation', 'Stop future tool calls when a loop, violation, or risky delegation appears.'],
              [ClipboardList, 'Modes', 'Start in Observe, enforce in Control, and use Charge/L402 when tools become paid products.'],
            ].map(([Icon, title, body]) => {
              const CardIcon = Icon as typeof Wrench;
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
          <h2 className="mb-8 text-3xl font-bold text-white">MCP proxy config questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is an MCP proxy?</h3>
              <p className="text-gray-400 leading-relaxed">
                An MCP proxy sits between an AI agent client and MCP servers so tool calls can be observed, audited, budgeted, denied, revoked, or charged before expensive actions execute.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why put SatGate in front of MCP servers?</h3>
              <p className="text-gray-400 leading-relaxed">
                MCP tools can trigger paid APIs, cloud work, searches, code agents, or sensitive data access. SatGate adds request-path economic governance around those tool calls.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Which MCP clients can use this pattern?</h3>
              <p className="text-gray-400 leading-relaxed">
                The proxy pattern applies to Cursor, Claude Desktop, Claude Code, OpenClaw, and custom MCP-capable clients that can route tool calls through an MCP server command or proxy.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Should an MCP proxy start in Observe or Control mode?</h3>
              <p className="text-gray-400 leading-relaxed">
                Start an MCP proxy in Observe mode when mapping normal tool use, then move expensive, sensitive, external, or high-volume tools into Control mode with hard budgets and revocation.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What policy should an MCP proxy enforce?</h3>
              <p className="text-gray-400 leading-relaxed">
                An MCP proxy should enforce agent and task identity, session budgets, per-tool cost caps, unknown-tool behavior, revocation triggers, audit fields, and optional L402 payment before tool execution.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/25 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Put MCP tools behind an economic firewall.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            SatGate lets agent teams observe, control, revoke, audit, and charge MCP tool calls in the request path — before expensive or sensitive work executes.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/mcp-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              MCP cost control <ArrowRight size={18} />
            </Link>
            <Link href="/mcp-tool-cost-policy-generator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              MCP tool policy generator
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

import Link from 'next/link';
import { ArrowRight, Cable, Eye, Gauge, KeyRound, ShieldAlert, Wrench } from 'lucide-react';

export const metadata = {
  title: 'MCP Governance: Authority, Policy, and Audit Receipts',
  description: 'Govern Model Context Protocol tools with SatGate. Check agent authority, enforce policy, revoke access, and produce Evidence Pack receipts.',
  alternates: { canonical: 'https://satgate.io/mcp-governance' },
  keywords: [
    'MCP governance',
    'MCP budget enforcement',
    'MCP security',
    'MCP Evidence Pack',
    'Cursor MCP budget control',
    'Claude Desktop MCP governance',
    'Model Context Protocol governance',
    'MCP proxy',
    'MCP tool spend limits',
  ],
  openGraph: {
    title: 'MCP Governance: Authority, Policy, and Audit Receipts',
    description: 'Control Model Context Protocol tool calls with authority policy, scoped budgets, revocation, Evidence Pack receipts, and Evidence Pack proof.',
    url: 'https://satgate.io/mcp-governance',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Governance for AI Agents',
    description: 'Check authority, policy, revocation, and Evidence Pack receipts around MCP tool calls before agents execute work.',
  },
};

const controls = [
  {
    icon: Cable,
    title: 'Proxy MCP tool calls',
    body: 'Put SatGate between agents and MCP servers so tool traffic becomes visible, attributable, and enforceable.',
  },
  {
    icon: Gauge,
    title: 'Enforce authority and budgets',
    body: 'Check scope, allowed tools, delegation depth, and budget before forwarding each MCP call.',
  },
  {
    icon: KeyRound,
    title: 'Scope agent capabilities',
    body: 'Issue constrained credentials for Cursor, Claude Desktop, Claude Code, OpenClaw, and other MCP-capable agents.',
  },
  {
    icon: ShieldAlert,
    title: 'Revoke risky access',
    body: 'Kill or expire access immediately when an agent loops, delegates too broadly, or touches a sensitive tool.',
  },
  {
    icon: Eye,
    title: 'Create Evidence Pack receipts',
    body: 'Record who called which tool, why it was allowed or denied, what policy applied, and how the decision feeds the Evidence Pack.',
  },
  {
    icon: Wrench,
    title: 'Keep servers unchanged',
    body: 'Add governance around existing MCP servers without rewriting every tool implementation.',
  },
];

export default function McpGovernancePage() {
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'MCP Governance: Authority, Policy, and Audit Receipts',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-25',
    dateModified: '2026-06-04',
    mainEntityOfPage: 'https://satgate.io/mcp-governance',
    about: [
      { '@type': 'Thing', name: 'MCP governance' },
      { '@type': 'Thing', name: 'MCP budget enforcement' },
      { '@type': 'Thing', name: 'Model Context Protocol Evidence Pack receipts' },
      { '@type': 'Thing', name: 'MCP proxy policy' },
      { '@type': 'Thing', name: 'agent tool spend limits' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is MCP governance?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'MCP governance is the authority, policy, budget, access-control, revocation, and audit receipt layer around Model Context Protocol tool calls made by AI agents.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do MCP tools need budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Agents can call MCP tools repeatedly, delegate work, and trigger paid APIs or compute-heavy operations. Authority and budget enforcement stop unauthorized tool calls before they execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can MCP governance work without changing existing MCP servers?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. A proxy/control-plane approach can wrap existing MCP tool traffic so governance is enforced before tool calls reach the server.',
        },
      },
      {
        '@type': 'Question',
        name: 'Which MCP clients can use budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Any MCP-capable client can be routed through a governance proxy, including Cursor, Claude Desktop, Claude Code, OpenClaw, and custom agent runtimes.',
        },
      },
      {
        '@type': 'Question',
        name: 'How is MCP governance different from MCP security?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'MCP security focuses on safe tool access, secrets, permissions, and malicious behavior. MCP governance adds authority proof: who can call which tool, what budget applies, what policy allows or denies the request, and which receipt proves the decision.',
        },
      },
      {
        '@type': 'Question',
        name: 'Do I need an MCP proxy for budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'A proxy is the cleanest way to enforce MCP budgets because it places policy in the request path between agents and tools. That lets teams identify, price, allow, block, revoke, and audit tool calls without rewriting every MCP server.',
        },
      },
    ],
  };

  const howToJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'HowTo',
    name: 'How to add MCP budget enforcement',
    description: 'Route MCP tool calls through SatGate so each call can be identified, checked against authority policy, allowed or denied before execution, and recorded as an audit receipt.',
    totalTime: 'PT15M',
    step: [
      { '@type': 'HowToStep', name: 'Proxy MCP traffic', text: 'Place SatGate between the agent runtime and MCP servers so tool calls pass through the request-path governance layer.' },
      { '@type': 'HowToStep', name: 'Identify agents and tools', text: 'Attach tenant, agent, session, token, server, and tool metadata to each MCP call.' },
      { '@type': 'HowToStep', name: 'Map authority and risk', text: 'Set scope, budget, risk, and approval rules for MCP tools, paid APIs, searches, browser sessions, code agents, or cloud tasks.' },
      { '@type': 'HowToStep', name: 'Enforce authority policy', text: 'Block, route, allow, or revoke MCP calls based on scope, budget, per-tool limits, delegation depth, and capability policy.' },
      { '@type': 'HowToStep', name: 'Create Evidence Pack receipts', text: 'Record the decision, estimated cost, tool, route, policy, outcome, and receipt ID for finance, security, and platform teams.' },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'MCP Governance', item: 'https://satgate.io/mcp-governance' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(howToJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_35%_0%,rgba(34,211,238,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.16),transparent_30%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200 mb-8">
            <Cable size={16} /> Model Context Protocol governance
          </div>

          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            MCP Governance for Agents That Need Authority Before Execution
          </h1>

          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-10">
            MCP makes tools easy for agents to use. SatGate makes tool use governable: every MCP call is identified, checked against policy, allowed or denied before execution, and recorded as proof.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/mcp-proxy-config-generator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Generate MCP proxy config <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              See Policy-to-Proof
            </Link>
            <Link href="/mcp-budget-enforcement" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Enforce MCP budgets
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12 items-start">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">MCP connects agents to tools. It does not govern authority.</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              Model Context Protocol is becoming the common tool interface for agents. Cursor, Claude Desktop, Claude Code, OpenClaw, and other runtimes can connect to tools without every integration being custom-built.
            </p>
            <p>
              That is powerful, but it creates a new control problem. An agent with tool access can trigger searches, database calls, code execution, paid APIs, browser sessions, cloud tasks, and expensive workflows. A tool connection is not the same thing as a policy boundary.
            </p>
            <p>
              MCP governance means every tool call is identified, checked against authority and budget policy, routed or denied, and recorded as an audit receipt before it runs.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">Without MCP authority governance</h3>
          <ul className="space-y-3 text-gray-300">
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Agents call tools without scoped authority.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Expensive tools look identical to cheap tools.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Revocation requires changing configs or killing whole agents.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Finance sees a bill, not the tool or workflow that caused it.</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Security gets logs after the risky call already executed.</li>
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">What SatGate adds around MCP</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            SatGate acts as the request-path governance layer around MCP tool traffic, so agent authority becomes measurable, enforceable, and provable without rewriting every tool. When MCP calls cross paid rails like x402, AgentCore Payments, or Pay.sh, SatGate keeps the same policy decision and Evidence Pack receipt above the rail.
          </p>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {controls.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 hover:border-cyan-900/70 transition">
                <Icon className="text-cyan-300 mb-4" size={28} />
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-8">A practical MCP governance loop</h2>
        <div className="grid lg:grid-cols-5 gap-4">
          {[
            ['1', 'Identify', 'Which agent, tenant, token, session, and tool?'],
            ['2', 'Authorize', 'What scope, budget, and risk policy applies?'],
            ['3', 'Check', 'Does policy allow this action before execution?'],
            ['4', 'Execute', 'Forward only approved tool calls.'],
            ['5', 'Prove', 'Record the receipt, decision, spend, and outcome.'],
          ].map(([n, title, body]) => (
            <div key={n} className="rounded-xl border border-gray-800 bg-gray-950 p-5">
              <div className="text-cyan-300 font-mono text-sm mb-3">{n}</div>
              <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
              <p className="text-gray-400 text-sm leading-relaxed">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">MCP governance by runtime</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-5">
            {[
              ['Cursor', 'Budget code-search, issue, repo, browser, and shell-adjacent tools so coding agents cannot quietly burn spend.'],
              ['Claude Desktop', 'Give local desktop agents governed tool access with revocable capabilities and auditable decisions.'],
              ['Claude Code', 'Cap expensive build, test, search, and deployment-adjacent tool calls during delegated coding sessions.'],
              ['OpenClaw', 'Apply policy to proactive workflows, cron-like tasks, delegated sub-agents, and autonomous MCP tool use.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-5">
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 text-sm leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-4xl mx-auto px-6 py-20">
          <div className="rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">MCP authority policy example</h2>
            <pre className="bg-black border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm text-gray-300"><code>{`parent_agent: finance-automation
worker_agent: invoice-reconciler
mcp_server: accounts-payable-tools
authority:
  tools:
    invoice_lookup: allow
    vendor_match: allow
    payment_schedule: require_approval
    erp_write: deny
budget:
  workflow: 25.00 USD
  per_tool_call: 0.50 USD
delegation:
  max_depth: 1
  child_budget_max: 5.00 USD
decision:
  enforce_before_execution: true
evidence:
  include: [parent_agent, worker_agent, tool, policy, decision, outcome, receipt_id]`}</code></pre>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">MCP governance questions</h2>
          <div className="grid gap-5 md:grid-cols-2 mb-16">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is MCP governance?</h3>
              <p className="text-gray-400 leading-relaxed">
                MCP governance is the authority, policy, budget, access-control, revocation, and audit receipt layer around Model Context Protocol tool calls made by AI agents.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why do MCP tools need budget enforcement?</h3>
              <p className="text-gray-400 leading-relaxed">
                Agents can call MCP tools repeatedly, delegate work, and trigger paid APIs or compute-heavy operations. Authority and budget enforcement stop unauthorized tool calls before they execute.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Can MCP governance work without changing existing MCP servers?</h3>
              <p className="text-gray-400 leading-relaxed">
                Yes. A proxy or control-plane approach can wrap existing MCP tool traffic so governance is enforced before tool calls reach the server.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Which MCP clients can use budget enforcement?</h3>
              <p className="text-gray-400 leading-relaxed">
                Any MCP-capable client can be routed through a governance proxy, including Cursor, Claude Desktop, Claude Code, OpenClaw, and custom agent runtimes.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How is MCP governance different from MCP security?</h3>
              <p className="text-gray-400 leading-relaxed">
                MCP security focuses on safe tool access, secrets, permissions, and malicious behavior. MCP governance adds authority proof: who can call which tool, what budget applies, what policy allows or denies the request, and which receipt proves the decision.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Do I need an MCP proxy for budget enforcement?</h3>
              <p className="text-gray-400 leading-relaxed">
                A proxy is the cleanest way to enforce MCP budgets because it places policy in the request path between agents and tools. That lets teams identify, price, allow, block, revoke, and audit tool calls without rewriting every MCP server.
              </p>
            </div>
          </div>

          <h2 className="text-3xl font-bold text-white mb-8">Related MCP governance topics</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[
              ['/policy-to-proof', 'Policy-to-Proof', 'See how MCP allow, deny, delegate, and revoke decisions become Evidence Pack proof.'],
              ['/govern', 'Govern AI agents', 'Govern MCP authority before tool execution.'],
              ['/mcp-budget-enforcement', 'MCP budget enforcement', 'Hard caps, per-tool prices, and request-path budget decisions for MCP servers.'],
              ['/mcp-cost-control', 'MCP cost control', 'Control paid tool calls, retries, SaaS actions, cloud tasks, and data lookups before MCP tools execute.'],
              ['/mcp-proxy-config-generator', 'MCP proxy config generator', 'Generate governed MCP proxy JSON and policy YAML for Cursor, Claude, OpenClaw, and custom clients.'],
              ['/mcp-tool-cost-policy-generator', 'MCP tool cost policy generator', 'Generate practical policy for Cursor, Claude Desktop, Claude Code, OpenClaw, and custom clients.'],
              ['/satgate-for-cursor', 'SatGate for Cursor', 'Govern Cursor MCP/tool workflows with budgets and audit.'],
              ['/satgate-for-openclaw', 'SatGate for OpenClaw', 'Apply economic policy to proactive agents, sub-agents, and tools.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 transition hover:border-purple-500/50 hover:bg-purple-950/10">
                <h3 className="font-bold text-white mb-2">{title}</h3>
                <p className="text-sm text-gray-400 leading-relaxed">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">Make MCP tools safe enough for autonomous agents</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            Connect tools quickly with MCP. Govern them with SatGate. Check authority before execution, revoke risky capabilities, and turn every allow/deny decision into audit evidence for the Evidence Pack.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/mcp-proxy-config-generator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Generate MCP proxy config <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              See Policy-to-Proof
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

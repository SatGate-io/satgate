import Link from 'next/link';
import { ArrowRight, Cable, Eye, Gauge, KeyRound, ShieldAlert, Wrench } from 'lucide-react';

export const metadata = {
  title: 'MCP Governance: Budget Enforcement, Audit, and Policy',
  description: 'Govern Model Context Protocol tools with SatGate. Enforce MCP budgets, audit tool calls, revoke agent access, and control spend before tools execute.',
  alternates: { canonical: 'https://satgate.io/mcp-governance' },
  keywords: [
    'MCP governance',
    'MCP budget enforcement',
    'MCP security',
    'MCP audit trail',
    'Cursor MCP budget control',
    'Claude Desktop MCP governance',
    'Model Context Protocol governance',
  ],
};

const controls = [
  {
    icon: Cable,
    title: 'Proxy MCP tool calls',
    body: 'Put SatGate between agents and MCP servers so tool traffic becomes visible, attributable, and enforceable.',
  },
  {
    icon: Gauge,
    title: 'Enforce per-tool budgets',
    body: 'Assign cost to expensive tools, cap spend per session or agent, and block calls before they run.',
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
    title: 'Audit every decision',
    body: 'Record who called which tool, why it was allowed or denied, what it cost, and which policy applied.',
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
    headline: 'MCP Governance: Budget Enforcement, Audit, and Policy',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-25',
    dateModified: '2026-04-25',
    mainEntityOfPage: 'https://satgate.io/mcp-governance',
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
          text: 'MCP governance is the policy, budget, access-control, revocation, and audit layer around Model Context Protocol tool calls made by AI agents.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do MCP tools need budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Agents can call MCP tools repeatedly, delegate work, and trigger paid APIs or compute-heavy operations. Budget enforcement stops expensive tool calls before they execute.',
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
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_35%_0%,rgba(34,211,238,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.16),transparent_30%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200 mb-8">
            <Cable size={16} /> Model Context Protocol governance
          </div>

          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            MCP Governance for Agents That Can Actually Spend Money
          </h1>

          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-10">
            MCP makes tools easy for agents to use. SatGate makes them governable: budgets, access policy, revocation, audit trails, and spend attribution before tools execute.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/blog/mcp-budget-enforcement-guide" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Read the MCP budget guide <ArrowRight size={18} />
            </Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              See agent cost control
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12 items-start">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">MCP connects agents to tools. It does not govern the economics.</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              Model Context Protocol is becoming the common tool interface for agents. Cursor, Claude Desktop, Claude Code, OpenClaw, and other runtimes can connect to tools without every integration being custom-built.
            </p>
            <p>
              That is powerful, but it creates a new control problem. An agent with tool access can trigger searches, database calls, code execution, paid APIs, browser sessions, cloud tasks, and expensive workflows. A tool connection is not the same thing as a policy boundary.
            </p>
            <p>
              MCP governance means every tool call is identified, priced, checked, routed, and audited before it runs.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">Without MCP governance</h3>
          <ul className="space-y-3 text-gray-300">
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Agents call tools without per-tool spend caps.</li>
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
            SatGate acts as the economic control plane around MCP tool traffic, so agent access becomes measurable and enforceable without rewriting every tool.
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
            ['2', 'Price', 'What does this tool call cost or risk?'],
            ['3', 'Check', 'Does policy and budget allow it?'],
            ['4', 'Execute', 'Forward only approved tool calls.'],
            ['5', 'Audit', 'Record decision, spend, and outcome.'],
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
        <div className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-2 gap-8">
          <div>
            <h2 className="text-3xl font-bold text-white mb-6">Use cases</h2>
            <div className="space-y-4">
              {[
                ['Cursor MCP proxy', 'Give coding agents budgeted tool access without handing them unlimited API authority.'],
                ['Claude Desktop governance', 'Let desktop agents call tools while preserving per-agent policy and auditability.'],
                ['Claude Code tool spend', 'Cap expensive build, search, browser, or cloud actions during coding sessions.'],
                ['OpenClaw autonomous workflows', 'Govern proactive agent tasks, recurring jobs, and delegated sub-agents.'],
              ].map(([title, body]) => (
                <div key={title} className="rounded-xl border border-gray-800 bg-black p-5">
                  <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                  <p className="text-gray-400 leading-relaxed">{body}</p>
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">MCP budget policy example</h2>
            <pre className="bg-black border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm text-gray-300"><code>{`agent: cursor-coder
mcp_server: github-tools
mode: control
budget:
  session: 5.00 USD
  per_tool_call: 0.25 USD
tools:
  repo_search: allow
  issue_create: allow
  deploy_prod: deny
on_budget_exhausted: block
audit:
  include: [agent, tool, route, estimated_cost, decision]`}</code></pre>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">Make MCP tools safe enough for autonomous agents</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            Connect tools quickly with MCP. Govern them with SatGate. Observe every call, control spend and access, revoke risky capabilities, and charge when tools become products for external agents.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/blog/cursor-mcp-proxy-setup-guide" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Cursor MCP setup guide <ArrowRight size={18} />
            </Link>
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Economic firewall category
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

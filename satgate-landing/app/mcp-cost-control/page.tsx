import Link from 'next/link';
import { ArrowRight, Ban, BarChart3, Bot, DollarSign, Gauge, KeyRound, ReceiptText } from 'lucide-react';

export const metadata = {
  title: 'MCP Cost Control | Budget Enforcement for Tool-Calling Agents',
  description: 'Control MCP tool costs with per-tool budgets, scoped credentials, revocation, Evidence Pack receipts, and request-path enforcement for Cursor, Claude, and OpenClaw.',
  alternates: { canonical: 'https://satgate.io/mcp-cost-control' },
  keywords: [
    'MCP cost control',
    'MCP budget enforcement',
    'MCP tool cost control',
    'MCP spend control',
    'tool calling budget enforcement',
    'AI agent tool cost control',
    'economic firewall',
    'SatGate',
    'AI agent cost control',
  ],
  openGraph: {
    title: 'MCP Cost Control | Budget Enforcement for Tool-Calling Agents',
    description: 'Control MCP tool costs with per-tool budgets, risk tiers, scoped credentials, revocation, Evidence Pack receipts, and request-path enforcement.',
    url: 'https://satgate.io/mcp-cost-control',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Cost Control | Budget Enforcement for Tool-Calling Agents',
    description: 'Control MCP tool costs with per-tool budgets, risk tiers, scoped credentials, revocation, Evidence Pack receipts, and request-path enforcement.',
  },
};

const controls = [
  { icon: Bot, title: 'Agent identity', body: 'Attribute requests to the tenant, agent, task, workflow, route, model, MCP tool, and delegated sub-agent.' },
  { icon: DollarSign, title: 'Budget checks', body: 'Evaluate remaining spend, per-request ceilings, daily caps, tool caps, and route budgets before forwarding.' },
  { icon: KeyRound, title: 'Scoped credentials', body: 'Use expiring capabilities instead of broad static keys so authority matches the job.' },
  { icon: Ban, title: 'Revocation', body: 'Block the next request when a credential, workflow, route, budget, or agent should stop.' },
  { icon: ReceiptText, title: 'Audit trails', body: 'Record allow/deny decisions with policy, budget remaining, route, tool, estimated cost, and outcome.' },
  { icon: BarChart3, title: 'Benchmark risk', body: 'Model loops, retry storms, fanout, detection delay, and avoided spend with benchmark-backed scenarios.' },
];

export default function Page() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'MCP Cost Control | Budget Enforcement for Tool-Calling Agents',
    url: 'https://satgate.io/mcp-cost-control',
    description: metadata.description,
    datePublished: '2026-05-01',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'MCP cost control' },
      { '@type': 'Thing', name: 'MCP tool cost control' },
      { '@type': 'Thing', name: 'tool-calling budget enforcement' },
      { '@type': 'Thing', name: 'AI agent tool cost control' },
      { '@type': 'Thing', name: 'request-path MCP policy' },
    ],
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'MCP Cost Control',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Cloud, API Gateway, MCP Proxy',
    description: metadata.description,
    url: 'https://satgate.io/mcp-cost-control',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-03',
    featureList: ['Request-path budget enforcement', 'AI agent spend caps', 'MCP tool cost control', 'Revocable credentials', 'Audit trails'],
    audience: { '@type': 'Audience', audienceType: 'AI platform, MCP, finance, and security teams' },
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'What is MCP cost control?', acceptedAnswer: { '@type': 'Answer', text: 'MCP cost control is the practice of pricing, limiting, attributing, and auditing Model Context Protocol tool calls before they execute so agents cannot create hidden API, SaaS, cloud, search, or data spend.' } },
      { '@type': 'Question', name: 'Why are dashboards not enough?', acceptedAnswer: { '@type': 'Answer', text: 'Dashboards and billing alerts report spend after requests complete. Autonomous agents can loop, retry, and delegate fast enough that budget policy must be enforced before upstream access.' } },
      { '@type': 'Question', name: 'How does SatGate help?', acceptedAnswer: { '@type': 'Answer', text: 'SatGate sits in the request path and checks identity, budget, route, tool scope, credential caveats, expiry, revocation, and audit policy before forwarding the request.' } },
      { '@type': 'Question', name: 'How is MCP cost control different from LLM cost control?', acceptedAnswer: { '@type': 'Answer', text: 'LLM cost control focuses on model and token usage. MCP cost control covers tool calls that can trigger paid search, browser automation, cloud actions, SaaS APIs, data lookups, code execution, or delegated workflows outside the LLM bill.' } },
      { '@type': 'Question', name: 'Where should MCP tool cost policy be enforced?', acceptedAnswer: { '@type': 'Answer', text: 'MCP tool cost policy should be enforced in the request path before the MCP server executes the tool, so expensive calls can be blocked, downgraded, routed, approved, revoked, paid, or recorded before cost is created.' } },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'MCP Governance', item: 'https://satgate.io/mcp' },
      { '@type': 'ListItem', position: 3, name: 'MCP Cost Control', item: 'https://satgate.io/mcp-cost-control' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_24%_0%,rgba(34,211,238,0.18),transparent_34%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200"><Gauge size={16} /> Cost controls for MCP tool calls</div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">MCP cost control belongs before tool execution</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">MCP moves agent cost beyond LLM tokens. Tool calls can trigger search, data, cloud, code, SaaS, or premium API spend. SatGate attaches authority, budget, revocation, and receipt policy to the tool call before it executes.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Govern MCP tool spend <ArrowRight size={18} /></Link>
            <Link href="/mcp-tool-cost-policy-generator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">Generate MCP policy</Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">See Policy-to-Proof</Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1fr_0.9fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">The control point is before the call</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>Autonomous agents can generate real costs through model calls, API requests, MCP tools, delegated sub-agents, retries, and background workflows. If the policy check happens after the request, the money is already spent.</p>
            <p>SatGate enforces economic policy at the gateway boundary. Every important request can be evaluated against budget, scope, identity, revocation, route, tool, receipt, and audit rules before upstream access.</p>
            <p>That is the difference between cost reporting and economic control.</p>
          </div>
        </div>
        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">What good policy includes</h3>
          <div className="space-y-4">
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Price tools explicitly</h3><p className="leading-relaxed text-gray-400">Assign cost and risk to each MCP tool instead of treating tool calls as invisible agent behavior.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Enforce per-tool caps</h3><p className="leading-relaxed text-gray-400">Set call limits, spend caps, expensive-tool approval rules, and deny behavior before the MCP server runs.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Scope agent authority</h3><p className="leading-relaxed text-gray-400">Use capability tokens and revocable credentials so each agent can access only the MCP tools its task requires.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Audit tool economics</h3><p className="leading-relaxed text-gray-400">Attribute MCP spend by client, agent, tenant, server, tool, route, workflow, and policy decision.</p></div>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">SatGate controls</h2>
          <p className="mb-10 max-w-3xl text-lg text-gray-400">Use SatGate to move from Observe to Control: measure real agent economics first, then enforce the limits that matter.</p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {controls.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 transition hover:border-cyan-900/70">
                <Icon className="mb-4 text-cyan-300" size={28} />
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">MCP cost control questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is MCP cost control?</h3>
              <p className="text-gray-400 leading-relaxed">
                MCP cost control is the practice of pricing, limiting, attributing, and auditing Model Context Protocol tool calls before they execute so agents cannot create hidden API, SaaS, cloud, search, or data spend.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why are dashboards not enough?</h3>
              <p className="text-gray-400 leading-relaxed">
                Dashboards and billing alerts report spend after requests complete. Autonomous agents can loop, retry, and delegate fast enough that budget policy must be enforced before upstream access.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does SatGate help?</h3>
              <p className="text-gray-400 leading-relaxed">
                SatGate sits in the request path and checks identity, budget, route, tool scope, credential caveats, expiry, revocation, and audit policy before forwarding the request.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How is MCP cost control different from LLM cost control?</h3>
              <p className="text-gray-400 leading-relaxed">
                LLM cost control focuses on model and token usage. MCP cost control covers tool calls that can trigger paid search, browser automation, cloud actions, SaaS APIs, data lookups, code execution, or delegated workflows outside the LLM bill.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Where should MCP tool cost policy be enforced?</h3>
              <p className="text-gray-400 leading-relaxed">
                MCP tool cost policy should be enforced in the request path before the MCP server executes the tool, so expensive calls can be blocked, downgraded, routed, approved, revoked, paid, or recorded before cost is created.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-purple-900/60 bg-gradient-to-br from-purple-950/35 to-cyan-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Make agent economics enforceable.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">SatGate is the economic firewall for AI agents: observe every MCP tool call, enforce spend before execution, and preserve receipts for policy, revocation, and access decisions.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Govern MCP tool spend <ArrowRight size={18} /></Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">Review Policy-to-Proof</Link>
          </div>
        </div>
      </section>
    </main>
  );
}

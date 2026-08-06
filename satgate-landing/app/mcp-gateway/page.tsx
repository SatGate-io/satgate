import Link from 'next/link';
import { ArrowLeft, ArrowRight, Bot, Cable, CheckCircle2, Download, Eye, FileJson, Gauge, GitBranch, LockKeyhole, ReceiptText, ServerCog, ShieldCheck } from 'lucide-react';

const policyTemplates = [
  {
    name: 'Evidence Pack sample',
    href: '/policy-templates/mcp-governance/mcp-evidence-pack.sample.v1.json',
    body: 'Sample MCP Evidence Pack showing the receipt fields buyers expect after allow, deny, budget, tenant, and delegation decisions.',
    format: 'JSON',
  },
  {
    name: 'Spend caps',
    href: '/policy-templates/mcp-governance/spend-caps.v1.yaml',
    body: 'Per-session, per-agent, per-tool, and per-tenant MCP budget enforcement with fail-closed spend controls.',
    format: 'YAML',
  },
  {
    name: 'Tool allowlists',
    href: '/policy-templates/mcp-governance/tool-allowlist.v1.yaml',
    body: 'Default-deny MCP tool access by tenant, principal, agent, server, risk tier, and explicit tool scope.',
    format: 'YAML',
  },
  {
    name: 'Tenant isolation',
    href: '/policy-templates/mcp-governance/tenant-isolation.v1.yaml',
    body: 'Tenant-bound credentials, budgets, ledgers, MCP servers, and Evidence Packs without trusting client-supplied tenant headers.',
    format: 'YAML',
  },
  {
    name: 'Delegation depth',
    href: '/policy-templates/mcp-governance/delegation-depth.v1.yaml',
    body: 'Macaroon-style delegation ceilings, child-budget attenuation, parent revocation, and receipt-ready chain hashes.',
    format: 'YAML',
  },
];

const controls = [
  { icon: Eye, title: 'Capture MCP tool receipts', body: 'Preserve a receipt for each MCP call with tenant, principal, agent, token, client, server, tool, budget, workflow, and Evidence Pack linkage.' },
  { icon: LockKeyhole, title: 'Control access and budgets', body: 'Enforce scoped capabilities, tool allowlists, MCP budget enforcement, spend caps, tenant isolation, delegation depth, expiry, and next-request revocation before execution.' },
  { icon: ReceiptText, title: 'Produce MCP Evidence Packs', body: 'Record the agent, tool, policy version, decision, budget state, delegation chain hash, receipt ID, and outcome so MCP activity can be reviewed as proof.' },
];

const proofRows = [
  ['Claude Desktop / Claude Code', 'MCP stdio config points the client at satgate-mcp before the upstream server.', 'Verified MCP-compatible path: allowed web_search call, budget-exhausted code_execute denial, Evidence Pack-style decision transcript.'],
  ['Ollama agent wrapper', 'Local Ollama agents use an MCP-capable wrapper that speaks stdio or SSE to SatGate.', 'Same protocol path: list tools, call allowed tool, burn budget, receive denial at the gateway policy check.'],
  ['Hermes agent runtime', 'Hermes agents running through an MCP client receive no standing tool authority; SatGate grants scoped calls per policy.', 'The sample Evidence Pack shows the receipt fields that bind tenant, agent, tool, policy digest, budget ID, decision reason, and remaining credits.'],
];

const faqs = [
  ['What is an MCP gateway?', 'An MCP gateway sits between AI agents and Model Context Protocol servers. SatGate observes tool calls, applies access policy and MCP budget enforcement, records MCP Evidence Pack receipts, and proves decisions before tools execute.'],
  ['What is an MCP gate?', 'MCP gate is shorthand for the policy gate between an AI agent and MCP servers. SatGate acts as that gate by checking identity, capability scope, budget, tenant boundary, and Evidence Pack receipt requirements before a tool call executes.'],
  ['What is MCP budget enforcement?', 'MCP budget enforcement checks the cost of a requested tool call against a tenant, agent, session, delegation, or tool budget before the call reaches the upstream MCP server. If the budget is missing or exhausted, SatGate denies the call in the request path.'],
  ['What is an MCP Evidence Pack?', 'An MCP Evidence Pack is the proof artifact for governed tool activity: who called which MCP tool, through which client and server, under which policy and budget, with which allow or deny decision, and what receipt proves it.'],
  ['What does Evidence MCP mean?', 'Evidence MCP usually means proof for Model Context Protocol tool activity. In SatGate, that proof is an MCP Evidence Pack: a receipt-backed record of the agent, tenant, client, server, tool, budget, policy version, and decision outcome.'],
  ['Can SatGate govern Claude, Hermes, or Ollama MCP agents?', 'Yes. Claude Desktop, Claude Code, Hermes, Ollama wrappers, Cursor, OpenClaw, and custom MCP-capable clients can route tool calls through SatGate. The agent gets no standing authority; SatGate grants or denies each tool call.'],
  ['How is an MCP gateway different from an API gateway?', 'A traditional API gateway mostly routes HTTP traffic and checks identity. An MCP gateway also understands agent tool calls, capability scope, per-tool cost, budget policy, tenant isolation, delegation lineage, and Evidence Pack outcomes.'],
  ['Can SatGate host MCP servers?', 'Yes. SatGate supports SaaS MCP for managed hosted deployment. Enterprise buyers needing isolated runtime boundaries can contract for a Dedicated deployment with custody and operations defined during onboarding.'],
];

const evidenceFields = [
  ['Caller identity', 'Tenant, principal, agent, workflow, and MCP client that requested the tool.'],
  ['Tool target', 'MCP server, tool name, route, action, risk tier, and requested arguments summary.'],
  ['Authority state', 'Capability scope, expiry, delegation lineage, revocation status, and policy version.'],
  ['Budget state', 'Price, remaining budget, budget scope, enforcement action, and paid-rail context when present.'],
  ['Decision proof', 'Allow or deny result, reason code, receipt ID, Evidence Pack link, and upstream outcome.'],
];

export const metadata = {
  title: 'MCP Gateway: MCP Gate for Evidence Packs and Budgets',
  description: 'Use SatGate as an MCP gateway and policy gate to check tool authority, enforce budgets, and export MCP Evidence Pack receipts.',
  alternates: { canonical: 'https://satgate.io/mcp-gateway' },
  keywords: [
    'MCP gateway',
    'MCP gate',
    'evidence MCP',
    'Model Context Protocol gateway',
    'MCP budget enforcement',
    'MCP Evidence Pack',
    'MCP evidence receipts',
    'MCP gateway evidence',
    'MCP policy templates',
    'MCP tool allowlist',
    'MCP tenant isolation',
    'MCP delegation depth',
    'Claude MCP budget enforcement',
    'Ollama MCP budget enforcement',
    'Hermes MCP governance',
  ],
  openGraph: {
    title: 'MCP Gateway: MCP Gate for Evidence Packs and Budgets',
    description: 'Put SatGate in the MCP request path: check authority before tool execution, enforce policy, and export MCP Evidence Pack receipts.',
    url: 'https://satgate.io/mcp-gateway',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Gateway: MCP Gate for Evidence Packs and Budgets',
    description: 'Govern Claude, Hermes, Ollama, Cursor, and custom MCP agents with authority checks, budgets, and Evidence Pack proof.',
  },
};

export default function McpGatewayPage() {
  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate MCP Gateway',
    applicationCategory: 'SecurityApplication',
    operatingSystem: 'Web',
    description: metadata.description,
    url: 'https://satgate.io/mcp-gateway',
    dateModified: '2026-08-06',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    featureList: [
      'MCP budget enforcement',
      'MCP Evidence Pack receipts',
      'MCP gate policy enforcement',
      'Evidence MCP receipt fields',
      'MCP tool allowlist policy',
      'Tenant isolation for MCP servers',
      'Delegation-depth enforcement for agent capabilities',
    ],
  };
  const mcpGatewayControlsJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'MCP gateway governance controls',
    description: 'Commercial MCP gateway controls for AI agent tool access, MCP gate enforcement, budget enforcement, capability authorization, audit proof, and Evidence Pack export.',
    dateModified: '2026-08-06',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Tool authority before execution',
        description: 'Identify the tenant, agent, capability, MCP client, server, and tool before an upstream MCP tool call runs.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'MCP tool budget enforcement',
        description: 'Price each governed tool call and enforce per-agent, per-session, per-tenant, and per-tool budgets before spend occurs.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Capability-based authorization',
        description: 'Use scoped, revocable, budget-aware capabilities instead of standing MCP tool authority or shared API keys.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Evidence Pack proof',
        description: 'Export signed receipts for allowed calls, denied calls, budget state, delegation lineage, revocation, and paid-rail context.',
      },
    ],
  };
  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: faqs.map(([name, text]) => ({ '@type': 'Question', name, acceptedAnswer: { '@type': 'Answer', text } })),
  };
  const datasetJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'Dataset',
    name: 'SatGate MCP governance policy templates',
    description: 'Downloadable MCP governance templates for spend caps, tool allowlists, tenant isolation, delegation depth, and Evidence Pack proof obligations.',
    url: 'https://satgate.io/mcp-gateway',
    distribution: policyTemplates.map((template) => ({
      '@type': 'DataDownload',
      name: template.name,
      contentUrl: `https://satgate.io${template.href}`,
      encodingFormat: template.format === 'JSON' ? 'application/json' : 'application/x-yaml',
    })),
  };
  const evidenceFieldsJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'MCP Evidence Pack receipt fields',
    description:
      'Evidence MCP receipt fields that prove governed Model Context Protocol tool calls: caller identity, tool target, authority state, budget state, and decision proof.',
    dateModified: '2026-08-06',
    itemListElement: evidenceFields.map(([title, body], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: title,
      description: body,
    })),
  };
  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'MCP Gateway', item: 'https://satgate.io/mcp-gateway' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(mcpGatewayControlsJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(datasetJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(evidenceFieldsJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <div className="mx-auto max-w-6xl px-6 pt-8">
        <Link href="/" className="inline-flex items-center gap-2 text-sm font-medium text-gray-500 transition hover:text-white">
          <ArrowLeft size={16} /> Back to Home
        </Link>
      </div>

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_30%_0%,rgba(34,211,238,0.18),transparent_32%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.15),transparent_34%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200 mb-8">
            <Cable size={16} /> Model Context Protocol governance gateway
          </div>
          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            MCP Gateway for Agent Governance, Budgets, and Evidence Packs
          </h1>
          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-8">
            SatGate sits between AI agents — Claude, Hermes, Ollama, Cursor, OpenClaw, or custom MCP clients — and the tools they want to call. Every MCP request is checked for authority, budget, tenant, tool scope, and delegation before execution — then preserved as Evidence Pack proof.
          </p>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-cyan-100">
            MCP Gateway for Budget Enforcement and Evidence Packs is the commercial control path: connect agents to tools, enforce authority before tool execution, and export proof after each governed decision.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Govern MCP tool access <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              See Policy-to-Proof
            </Link>
            <Link href="/mcp-budget-enforcement" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-800 px-6 py-3 font-bold text-gray-200 hover:border-cyan-500 transition">
              MCP budget enforcement
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-16">
        <div className="rounded-2xl border border-cyan-900/50 bg-gray-950 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Direct answer</p>
          <h2 className="mb-4 text-3xl font-bold text-white">An MCP gateway governs tool calls before agents reach the server</h2>
          <p className="mb-6 text-lg leading-relaxed text-gray-300">
            An MCP gateway sits between AI agents and Model Context Protocol servers. SatGate uses that position to identify the agent, check capability scope, price the requested tool call, enforce MCP budget policy, deny unauthorized execution, and export Evidence Pack receipts for the decision.
          </p>
          <div className="grid gap-4 md:grid-cols-4">
            {[
              ['Identify', 'Tenant, agent, workflow, MCP client, server, tool, token, and delegation chain.'],
              ['Authorize', 'Capability scope, tool allowlist, tenant boundary, expiry, and revocation state.'],
              ['Budget', 'Per-tool cost, remaining budget, rate context, and fail-closed spend controls.'],
              ['Prove', 'Allowed and denied MCP calls preserved as Evidence Pack receipts.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black/50 p-4">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
          <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
            <Link href="/capability-auth" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-500">
              Capability authorization <ArrowRight size={16} />
            </Link>
            <Link href="/blog/api-gateway-for-ai-agents" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-500">
              API gateway for AI agents <ArrowRight size={16} />
            </Link>
            <Link href="/blog/mcp-budget-enforcement-guide" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-500">
              MCP budget guide <ArrowRight size={16} />
            </Link>
            <Link href="/mcp-tool-cost-policy-generator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-500">
              Tool cost policy generator <ArrowRight size={16} />
            </Link>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-16">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Evidence MCP</p>
          <h2 className="mb-5 text-3xl font-bold text-white">An MCP gate should leave proof, not just logs</h2>
          <p className="max-w-4xl text-lg leading-relaxed text-gray-300">
            Teams searching for evidence MCP usually need proof that agent tool calls were governed. SatGate turns the MCP gate into a receipt layer: every supported allow, deny, budget, tenant, delegation, and revocation decision can be exported as an MCP Evidence Pack instead of disappearing into raw logs.
          </p>
          <div className="mt-8 grid gap-4 md:grid-cols-2 lg:grid-cols-5">
            {evidenceFields.map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-5">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">MCP connection is not MCP governance</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              MCP makes tools reachable. It does not answer whether this agent, tenant, budget, delegation chain, or tool call should be trusted right now. That missing decision point is where runaway spend, cross-tenant mistakes, and unauditable agent actions sneak in.
            </p>
            <p>
              SatGate turns the MCP gateway into a Zero Trust policy enforcement point for agents: authority before execution, signed receipts on supported governed paths, and Evidence Pack proof when a security, platform, finance, or buyer team asks what happened. That is Policy-to-Proof applied to MCP.
            </p>
          </div>
        </div>
        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">The governed MCP request path</h3>
          {['Agent asks for an MCP tool', 'SatGate identifies tenant, agent, token, MCP client, and server', 'Policy checks tool allowlist, budget, tenant boundary, and delegation depth', 'Allowed calls execute; denied calls stop before the upstream tool', 'MCP Evidence Pack receipt records policy, decision, budget, and proof'].map((step, index) => (
            <div key={step} className="flex gap-3 border-b border-gray-800 py-3 last:border-b-0">
              <span className="text-cyan-300 font-bold">{index + 1}</span>
              <span className="text-gray-300">{step}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">Observe, Control, Charge MCP tool use</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            The point is not just to connect agents to tools. The point is to prove what happened, stop what should not happen, and preserve receipts for what was allowed or denied.
          </p>
          <div className="grid md:grid-cols-3 gap-5">
            {controls.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6">
                <Icon className="text-cyan-300 mb-4" size={28} />
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="flex items-end justify-between gap-6 mb-8">
          <div>
            <p className="text-sm font-mono uppercase tracking-wide text-cyan-300 mb-2">Downloadable templates</p>
            <h2 className="text-3xl font-bold text-white">MCP policy templates teams can start from</h2>
          </div>
          <Link href="/policy-templates/mcp-governance/mcp-governance-policy-bundle.v1.json" className="hidden md:inline-flex items-center gap-2 rounded-lg border border-cyan-800 px-4 py-3 text-sm font-bold text-cyan-100 hover:border-cyan-400 transition">
            <FileJson size={18} /> Policy bundle JSON
          </Link>
        </div>
        <div className="grid md:grid-cols-2 lg:grid-cols-5 gap-5">
          {policyTemplates.map((template) => (
            <Link key={template.name} href={template.href} className="rounded-xl border border-gray-800 bg-gray-950 p-6 hover:border-cyan-700 transition">
              <Download className="text-cyan-300 mb-4" size={26} />
              <h3 className="text-xl font-bold text-white mb-2">{template.name}</h3>
              <p className="text-gray-400 leading-relaxed mb-4">{template.body}</p>
              <span className="text-sm font-bold text-cyan-200">Download {template.format} →</span>
            </Link>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="text-sm font-mono uppercase tracking-wide text-cyan-300 mb-2">Verified MCP-compatible path</p>
          <h2 className="text-3xl font-bold text-white mb-4">Claude, Hermes, and Ollama get scoped MCP authority — not standing authority</h2>
          <p className="text-gray-400 max-w-4xl mb-10 text-lg">
            SatGate’s verification uses an MCP-compatible stdio client path: initialize, list tools, call an allowed tool, attempt expensive work until the budget blocks, and preserve the decision transcript as proof. Claude, Hermes, Ollama, and other agents route through MCP clients or wrappers; the governance boundary is the MCP call path, not the model vendor.
          </p>
          <div className="grid lg:grid-cols-3 gap-5">
            {proofRows.map(([agent, path, proof]) => (
              <div key={agent} className="rounded-xl border border-gray-800 bg-black p-6">
                <Bot className="text-cyan-300 mb-4" size={28} />
                <h3 className="text-xl font-bold text-white mb-3">{agent}</h3>
                <p className="text-gray-400 leading-relaxed mb-4">{path}</p>
                <p className="text-gray-300 leading-relaxed"><CheckCircle2 className="inline mr-2 text-green-300" size={18} />{proof}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-2 gap-8">
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-7">
          <ServerCog className="text-cyan-300 mb-4" size={30} />
          <h2 className="text-2xl font-bold text-white mb-3">SaaS MCP is Fly-hosted</h2>
          <p className="text-gray-300 leading-relaxed">Use SaaS MCP when the buyer wants fast onboarding, managed runtime, and immediate MCP call receipts without operating infrastructure.</p>
        </div>
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-7">
          <ShieldCheck className="text-cyan-300 mb-4" size={30} />
          <h2 className="text-2xl font-bold text-white mb-3">Dedicated MCP is contract-triggered</h2>
          <p className="text-gray-300 leading-relaxed">Use Dedicated MCP when enterprise requirements call for isolated runtime boundaries and agreed operational controls. Deployment and custody boundaries are defined by contract; it is not a shared hybrid tier.</p>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">MCP gateway questions</h2>
          <div className="grid md:grid-cols-2 gap-5">
            {faqs.map(([q, a]) => (
              <div key={q} className="rounded-xl border border-gray-800 bg-black p-6">
                <h3 className="text-xl font-bold text-white mb-3">{q}</h3>
                <p className="text-gray-400 leading-relaxed">{a}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-6">Related MCP governance resources</h2>
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            ['/govern', 'AI agent governance', 'Govern agent authority, budgets, revocation, and Evidence Pack proof before execution.'],
            ['/capability-auth', 'Capability authorization', 'Replace standing MCP tool authority with scoped, revocable, budget-aware capabilities.'],
            ['/blog/api-gateway-for-ai-agents', 'API gateway for AI agents', 'See why agent gateways need MCP tool control, budgets, and proof beyond routing.'],
            ['/blog/mcp-budget-enforcement-guide', 'MCP budget enforcement guide', 'Walk through hard caps, tool prices, and request-path denial for MCP calls.'],
            ['/mcp-governance', 'MCP governance', 'Govern MCP tool calls with authority, policy, revocation, and Evidence Pack receipts.'],
            ['/mcp-budget-enforcement', 'MCP budget enforcement', 'Hard-cap per-tool spend before MCP tools execute.'],
            ['/mcp-tool-cost-policy-generator', 'MCP tool policy generator', 'Generate MCP tool cost and Evidence Pack policy.'],
            ['/evidence-pack-demo', 'Evidence Pack demo', 'See the machine-readable proof artifact.'],
          ].map(([href, title, body]) => (
            <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 hover:border-cyan-700 transition">
              <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
              <p className="text-sm text-gray-400">{body}</p>
            </Link>
          ))}
        </div>
      </section>

      <section className="border-t border-gray-900 bg-cyan-950/10">
        <div className="max-w-4xl mx-auto px-6 py-20 text-center">
          <Gauge className="mx-auto mb-6 text-cyan-300" size={36} />
          <h2 className="text-3xl font-bold text-white mb-4">Launch an MCP gateway agents can safely use.</h2>
          <p className="text-gray-300 mb-8">Start with policy templates, enforce authority before execution, and produce MCP Evidence Pack receipts that prove what each agent was allowed or denied to do.</p>
          <div className="flex flex-col sm:flex-row justify-center gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black hover:bg-gray-200 transition">
              Govern MCP tool access <ArrowRight size={18} />
            </Link>
            <Link href="/policy-templates/mcp-governance/mcp-governance-policy-bundle.v1.json" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Download policy bundle <GitBranch size={18} />
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

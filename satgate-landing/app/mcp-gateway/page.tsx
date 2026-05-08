import Link from 'next/link';
import { ArrowRight, Cable, DollarSign, Eye, Gauge, LockKeyhole, ServerCog } from 'lucide-react';

export const metadata = {
  title: 'MCP Gateway for Governed Agent Tool Access',
  description: 'Use SatGate as an MCP gateway to observe, control, and charge for agent tool access across SaaS and hybrid deployments.',
  alternates: { canonical: 'https://satgate.io/mcp-gateway' },
  keywords: ['MCP gateway', 'Model Context Protocol gateway', 'MCP access control', 'MCP budget enforcement', 'MCP tool metering', 'hosted MCP gateway', 'hybrid MCP gateway'],
  openGraph: {
    title: 'MCP Gateway for Governed Agent Tool Access',
    description: 'Observe, control, and charge for agent tool access across SaaS MCP and Hybrid MCP deployments with SatGate.',
    url: 'https://satgate.io/mcp-gateway',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Gateway for Governed Agent Tool Access',
    description: 'Put SatGate in the MCP request path so every agent tool call can be metered, governed, and audited.',
  },
};

const controls = [
  { icon: Eye, title: 'Observe MCP tool usage', body: 'Attribute each MCP call to tenant, agent, token, server, tool, customer, and workflow before finance or security asks for proof.' },
  { icon: LockKeyhole, title: 'Control access and budgets', body: 'Enforce scoped capabilities, tool allowlists, per-tool prices, spend caps, delegation depth, expiry, and instant revocation.' },
  { icon: DollarSign, title: 'Charge for tool access', body: 'Turn governed MCP usage into meterable events for chargeback, enterprise billing, or robot-customer monetization.' },
  { icon: ServerCog, title: 'Run SaaS or Hybrid MCP', body: 'Use Fly-hosted SaaS MCP for fast onboarding or Hetzner-hosted Hybrid MCP when buyers need dedicated runtime control.' },
];

const faqs = [
  ['What is an MCP gateway?', 'An MCP gateway sits between AI agents and Model Context Protocol servers. It observes tool calls, applies access policy and budgets, records audit trails, and can turn usage into chargeable events before tools execute.'],
  ['Why do MCP tools need access control?', 'MCP makes tools easy for agents to reach, which also makes expensive or sensitive tools easy to overuse. Access control limits which agents can call which tools, for how long, under which budget, and with what delegation.'],
  ['Can SatGate host MCP servers?', 'Yes. SatGate supports SaaS MCP for fast hosted deployment and Hybrid MCP for dedicated enterprise runtime control. The critical split is simple: SaaS MCP is Fly-hosted; Hybrid MCP is Hetzner-hosted.'],
  ['How is an MCP gateway different from an API gateway?', 'A traditional API gateway mostly routes HTTP traffic and checks identity. An MCP gateway also understands agent tool calls, capability scope, per-tool cost, budget policy, delegation lineage, and audit outcomes.'],
  ['Can MCP usage be monetized?', 'Yes. Once tool usage is identified, priced, and metered, SatGate can support chargeback, invoiceable usage, or Charge-mode payment flows where appropriate.'],
];

export default function McpGatewayPage() {
  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate MCP Gateway',
    applicationCategory: 'SecurityApplication',
    operatingSystem: 'Cloud, Hybrid',
    description: metadata.description,
    url: 'https://satgate.io/mcp-gateway',
    dateModified: '2026-05-08',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    featureList: controls.map((item) => item.title),
  };
  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: faqs.map(([name, text]) => ({ '@type': 'Question', name, acceptedAnswer: { '@type': 'Answer', text } })),
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
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_30%_0%,rgba(34,211,238,0.18),transparent_32%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.15),transparent_34%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200 mb-8">
            <Cable size={16} /> Model Context Protocol control plane
          </div>
          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            MCP Gateway for Governed Agent Tool Access
          </h1>
          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-8">
            SatGate is an MCP gateway for teams that need to observe, control, and charge for how agents use tools. Put policy in the request path before MCP calls execute.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              See SatGate governance <ArrowRight size={18} />
            </Link>
            <Link href="/mcp-tool-cost-policy-generator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Generate MCP tool policy
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">What is an MCP gateway?</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              An MCP gateway is the control point between agent runtimes and Model Context Protocol servers. Instead of letting agents call tools directly, traffic flows through a policy layer that can identify the agent, price the tool, check budget, enforce scope, and record the decision.
            </p>
            <p>
              That matters because MCP connections are not governance. Agents can trigger searches, code execution, paid APIs, database calls, browser sessions, and cloud tasks. SatGate makes those calls observable, controllable, and chargeable.
            </p>
          </div>
        </div>
        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">The request path</h3>
          {['Agent asks for a tool', 'SatGate checks capability, policy, and budget', 'Allowed calls execute; denied calls stop cleanly', 'Usage is attributed to agent, customer, tool, and tenant', 'Billing, chargeback, or audit events are recorded'].map((step, index) => (
            <div key={step} className="flex gap-3 border-b border-gray-800 py-3 last:border-b-0">
              <span className="text-cyan-300 font-bold">{index + 1}</span>
              <span className="text-gray-300">{step}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">Observe, Control, Charge for MCP tools</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            SatGate turns MCP tool calls into governed economic events. The point is not just to connect agents to tools. The point is to prove what happened, stop what should not happen, and meter what should be paid for.
          </p>
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-5">
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
        <h2 className="text-3xl font-bold text-white mb-8">SaaS MCP vs Hybrid MCP</h2>
        <div className="grid md:grid-cols-2 gap-6">
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <h3 className="text-2xl font-bold text-white mb-3">SaaS MCP is Fly-hosted</h3>
            <p className="text-gray-300 leading-relaxed">Use SaaS MCP when the buyer wants fast onboarding, managed runtime, and immediate visibility into MCP tool usage without operating infrastructure.</p>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <h3 className="text-2xl font-bold text-white mb-3">Hybrid MCP is Hetzner-hosted</h3>
            <p className="text-gray-300 leading-relaxed">Use Hybrid MCP when enterprise buyers need dedicated runtime boundaries, stronger operational control, and deployment evidence separate from the shared SaaS plane.</p>
          </div>
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
        <h2 className="text-3xl font-bold text-white mb-6">Related MCP gateway resources</h2>
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            ['/capability-auth', 'Capability auth', 'Scope agent authority before tools run.'],
            ['/blog/api-gateway-for-ai-agents', 'API gateway for agents', 'Compare routing with governance.'],
            ['/blog/mcp-budget-enforcement-guide', 'MCP budget enforcement', 'Hard-cap per-tool spend.'],
            ['/blog/http-402-payment-required-use-cases', 'HTTP 402 use cases', 'Turn access into chargeable events.'],
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
          <p className="text-gray-300 mb-8">Start with visibility, add budget and capability controls, then charge or bill for usage when tool access becomes economic activity.</p>
          <Link href="/design-partners" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black hover:bg-gray-200 transition">
            Work with SatGate <ArrowRight size={18} />
          </Link>
        </div>
      </section>
    </main>
  );
}

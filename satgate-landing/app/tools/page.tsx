import Link from 'next/link';
import { ArrowRight, BarChart3, Calculator, ClipboardList, Gauge, ShieldCheck, Wrench } from 'lucide-react';

export const metadata = {
  title: 'AI Agent Cost Control Tools',
  description: 'Free calculators and policy generators for AI agent spend control, MCP governance, OpenAI budget limits, and economic firewall readiness.',
  alternates: { canonical: 'https://satgate.io/tools' },
  keywords: [
    'AI agent cost control tools',
    'AI agent spend calculator',
    'OpenAI budget policy generator',
    'MCP tool cost policy generator',
    'economic firewall readiness grader',
    'agent budget enforcement tools',
  ],
  openGraph: {
    title: 'AI Agent Cost Control Tools',
    description: 'Calculators and generators for agent spend, runaway loops, OpenAI budgets, MCP tool costs, and economic firewall readiness.',
    url: 'https://satgate.io/tools',
    type: 'website',
  },
};

const tools = [
  {
    href: '/roi-calculator',
    title: 'AI Agent ROI Calculator',
    description: 'Estimate ghost spend, loop waste, payback period, and ROI from request-path budget enforcement.',
    icon: Calculator,
  },
  {
    href: '/runaway-agent-cost-calculator',
    title: 'Runaway Agent Cost Calculator',
    description: 'Model loop duration, paid call velocity, sub-agent fanout, and monthly exposure before detection.',
    icon: Gauge,
  },
  {
    href: '/ai-agent-runaway-spend-benchmark',
    title: 'AI Agent Runaway Spend Benchmark',
    description: 'Original benchmark scenarios for agent loops, MCP retry storms, fanout, detection delay, and avoided spend.',
    icon: BarChart3,
  },
  {
    href: '/openai-budget-policy-generator',
    title: 'OpenAI API Budget Limit Generator',
    description: 'Generate daily, session, per-request, model-routing, revocation, and audit policy for OpenAI API calls.',
    icon: ClipboardList,
  },
  {
    href: '/mcp-tool-cost-policy-generator',
    title: 'MCP Tool Cost Policy Generator',
    description: 'Create per-tool budgets, prices, risk tiers, revocation behavior, and audit fields for MCP agents.',
    icon: Wrench,
  },
  {
    href: '/economic-firewall-readiness-grader',
    title: 'Economic Firewall Readiness Grader',
    description: 'Score readiness across identity, budgets, MCP tools, revocation, delegation, audit, routing, and L402 payments.',
    icon: ShieldCheck,
  },
];

export default function ToolsPage() {
  const itemListJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI Agent Cost Control Tools',
    description: metadata.description,
    itemListElement: tools.map((tool, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: tool.title,
      url: `https://satgate.io${tool.href}`,
      description: tool.description,
    })),
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(itemListJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <Calculator size={16} /> Free calculators and policy generators
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            AI Agent Cost Control Tools
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Quantify runaway agent spend, generate enforceable budget policy, govern MCP tools, and grade your economic firewall readiness before autonomous agents hit production scale.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Learn economic firewalls <ArrowRight size={18} />
            </Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              See agent cost control
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
          {tools.map(({ href, title, description, icon: Icon }) => (
            <Link key={href} href={href} className="group rounded-2xl border border-gray-800 bg-gray-950 p-6 transition hover:border-cyan-500/50 hover:bg-cyan-950/10">
              <Icon className="mb-5 text-cyan-300 transition group-hover:text-cyan-200" size={32} />
              <h2 className="mb-3 text-xl font-bold text-white">{title}</h2>
              <p className="mb-5 leading-relaxed text-gray-400">{description}</p>
              <span className="inline-flex items-center gap-2 font-semibold text-cyan-300">Open tool <ArrowRight size={16} /></span>
            </Link>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto grid max-w-6xl gap-8 px-6 py-20 lg:grid-cols-3">
          {[
            ['Measure', 'Start with calculators to estimate ghost spend, runaway loop exposure, and payback period.'],
            ['Generate', 'Turn risk models into concrete OpenAI and MCP budget policies your control plane can enforce.'],
            ['Govern', 'Use readiness scoring to prioritize identity, revocation, audit, routing, and Charge/L402 gaps.'],
          ].map(([title, body]) => (
            <div key={title} className="rounded-2xl border border-gray-800 bg-black p-6">
              <h2 className="mb-3 text-2xl font-bold text-white">{title}</h2>
              <p className="leading-relaxed text-gray-400">{body}</p>
            </div>
          ))}
        </div>
      </section>
    </main>
  );
}

import Link from 'next/link';
import { ArrowLeft, ArrowRight, BarChart3, Calculator, Gauge, KeyRound, ShieldCheck, Wrench, Zap } from 'lucide-react';
import ToolLeadCaptureCta from '../components/ToolLeadCaptureCta';

export const metadata = {
  title: 'AI Agent Cost Control Tools',
  description: 'Free calculators for AI agent spend, runaway loops, L402 pricing and economic firewall readiness, plus the MCP connect snippet.',
  alternates: { canonical: 'https://satgate.io/tools' },
  keywords: [
    'AI agent cost control tools',
    'AI agent spend calculator',
    'AI agent runaway spend index',
    'LLM cost dashboard',
    'LLM cost monitoring',
    'MCP proxy config generator',
    'agent API key risk assessment',
    'L402 API pricing calculator',
    'economic firewall readiness grader',
    'agent budget enforcement tools',
  ],
  openGraph: {
    title: 'AI Agent Cost Control Tools',
    description: 'Calculators and benchmarks for agent spend, runaway loops, L402 pricing and economic firewall readiness, plus the MCP connect snippet.',
    url: 'https://satgate.io/tools',
    type: 'website',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI Agent Cost Control Tools',
    description: 'Free calculators and benchmarks for AI agent spend, runaway loops and economic firewall readiness.',
  },
};

const tools = [
  {
    href: '/llm-cost-dashboard',
    title: 'LLM Cost Dashboard Checklist',
    description: 'Track token cost, latency, model spend, agent attribution, MCP tools, and budget enforcement gaps.',
    icon: BarChart3,
  },
  {
    href: '/llm-cost-monitoring',
    title: 'LLM Cost Monitoring Guide',
    description: 'Compare dashboards, alerts, budget policy, routing, revocation, and request-path enforcement.',
    icon: BarChart3,
  },
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
    href: '/ai-agent-runaway-spend-index',
    title: 'AI Agent Runaway Spend Index',
    description: 'Track monthly modeled runaway spend exposure, MCP tool cost failures, fanout risk, and avoided cost from request-path controls.',
    icon: BarChart3,
  },
  {
    href: '/mcp-proxy-config-generator',
    title: 'MCP connect snippet',
    description: 'The real Cloud MCP snippet for Cursor and Claude Code. Credits, not a dollar budget.',
    icon: Wrench,
  },
  {
    href: '/economic-firewall-readiness-grader',
    title: 'Economic Firewall Readiness Grader',
    description: 'Score readiness across identity, budgets, MCP tools, revocation, delegation, audit, routing, and paid-rail context.',
    icon: ShieldCheck,
  },
  {
    href: '/agent-api-key-risk-assessment',
    title: 'Agent API Key Risk Assessment',
    description: 'Score static API key risk across scope, budget, expiry, revocation, delegation, and audit gaps for autonomous agents.',
    icon: KeyRound,
  },
  {
    href: '/l402-api-pricing-calculator',
    title: 'L402 API Pricing Calculator',
    description: 'Estimate per-request paid-agent pricing, gross margin, paid demand, and Lightning sats per API request.',
    icon: Zap,
  },
];

export default function ToolsPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'CollectionPage',
    name: 'AI Agent Cost Control Tools',
    url: 'https://satgate.io/tools',
    description: metadata.description,
    datePublished: '2026-04-12',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI agent cost control tools' },
      { '@type': 'Thing', name: 'economic firewall readiness' },
      { '@type': 'Thing', name: 'MCP connect snippet' },
      { '@type': 'Thing', name: 'OpenAI budget limits' },
      { '@type': 'Thing', name: 'runaway agent spend calculators' },
    ],
    audience: { '@type': 'Audience', audienceType: 'API, platform, security, and AI engineering teams' },
  };

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

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'AI Agent Cost Control Tools', item: 'https://satgate.io/tools' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What are AI agent cost control tools?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'AI agent cost control tools help teams estimate autonomous agent spend risk, model runaway loops, generate enforceable budget policy, and evaluate whether an economic firewall can stop expensive requests before they execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'Which SatGate tool should I start with?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Start with the AI Agent ROI Calculator if you need a business case, the Runaway Agent Cost Calculator if you need incident exposure, the MCP connect snippet if you want to route Cursor or Claude Code through SatGate, and the Economic Firewall Readiness Grader if you need a gap assessment.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do these tools relate to an economic firewall?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'The calculators estimate the risk. SatGate enforces budgets, permissions and revocation in the request path, and signs a receipt for each decision.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(itemListJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <div className="mx-auto max-w-6xl px-6 pt-8">
        <Link href="/" className="inline-flex items-center gap-2 text-sm font-medium text-gray-500 transition hover:text-white">
          <ArrowLeft size={16} /> Back to Home
        </Link>
      </div>

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <Calculator size={16} /> Free calculators
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            AI Agent Cost Control Tools
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Quantify runaway agent spend, generate enforceable budget policy, govern MCP tools, and grade your economic firewall readiness before autonomous agents hit production scale.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              See AI agent governance <ArrowRight size={18} />
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
            ['Govern', 'Use readiness scoring to prioritize identity, revocation, audit, routing, and paid-rail governance gaps.'],
          ].map(([title, body]) => (
            <div key={title} className="rounded-2xl border border-gray-800 bg-black p-6">
              <h2 className="mb-3 text-2xl font-bold text-white">{title}</h2>
              <p className="leading-relaxed text-gray-400">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 pb-4">
        <ToolLeadCaptureCta />
      </section>

      <section className="mx-auto max-w-4xl px-6 py-20">
        <h2 className="mb-8 text-3xl font-bold text-white">AI agent cost control tools FAQ</h2>
        <div className="space-y-5">
          {[
            [
              'What are AI agent cost control tools?',
              'They quantify autonomous agent spend risk, model runaway loops, and assess whether economic controls can stop expensive requests before execution.',
            ],
            [
              'Which SatGate tool should I start with?',
              'Start with the AI Agent ROI Calculator if you need a business case, the Runaway Agent Cost Calculator if you need incident exposure, the MCP connect snippet if you want to route Cursor or Claude Code through SatGate, and the Economic Firewall Readiness Grader if you need a gap assessment.',
            ],
            [
              'How do these tools relate to an economic firewall?',
              'The calculators estimate the risk. SatGate enforces budgets, permissions and revocation in the request path, and signs a receipt for each decision.',
            ],
          ].map(([question, answer]) => (
            <div key={question} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-3 text-xl font-bold text-white">{question}</h3>
              <p className="leading-relaxed text-gray-400">{answer}</p>
            </div>
          ))}
        </div>
      </section>
    </main>
  );
}

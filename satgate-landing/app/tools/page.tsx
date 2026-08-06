import Link from 'next/link';
import { ArrowLeft, ArrowRight, BarChart3, Calculator, ClipboardList, Gauge, KeyRound, Megaphone, ShieldCheck, Wrench, Zap } from 'lucide-react';
import ToolLeadCaptureCta from '../components/ToolLeadCaptureCta';

export const metadata = {
  title: 'AI Agent Orchestration Security Tools and Pricing',
  description: 'Free calculators and policy generators for AI agent orchestration security, cost control, MCP governance, pricing, budgets, and economic firewall readiness.',
  alternates: { canonical: 'https://satgate.io/tools' },
  keywords: [
    'AI agent cost control tools',
    'AI agent orchestration security tools and pricing',
    'AI agent orchestration security tools',
    'AI agent security tools pricing',
    'FactSet MCP pricing governance',
    'MCP pricing calculator',
    'AI agent spend calculator',
    'AI agent runaway spend index',
    'LLM cost dashboard',
    'LLM cost monitoring',
    'OpenAI budget policy generator',
    'MCP tool cost policy generator',
    'MCP proxy config generator',
    'agent spend policy template',
    'agent API key risk assessment',
    'revocable capability token policy template',
    'L402 API pricing calculator',
    'economic firewall readiness grader',
    'agent budget enforcement tools',
  ],
  openGraph: {
    title: 'AI Agent Orchestration Security Tools and Pricing',
    description: 'Calculators, benchmarks, policy templates, and generators for agent spend, orchestration security, MCP pricing, budgets, and economic firewall readiness.',
    url: 'https://satgate.io/tools',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI Agent Security Tools and Pricing',
    description: 'Free calculators, benchmarks, and policy generators for agent orchestration security, MCP pricing, budgets, and governance.',
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
  {
    href: '/seo-distribution-kit',
    title: 'SEO Distribution Kit',
    description: 'Launch copy, social snippets, and backlink targets for distributing SatGate cost-control tools and data assets.',
    icon: Megaphone,
  },
  {
    href: '/revocable-capability-token-policy-template',
    title: 'Revocable Capability Token Policy Template',
    description: 'Generate scoped, expiring, revocable capability-token policy for agents, sub-agents, MCP tools, budgets, and audit.',
    icon: KeyRound,
  },
  {
    href: '/agent-spend-policy-template',
    title: 'Agent Spend Policy Template',
    description: 'Generate copyable YAML and JSON policy for budgets, MCP tool costs, delegation, revocation, and audit fields.',
    icon: ClipboardList,
  },
  {
    href: '/mcp-proxy-config-generator',
    title: 'MCP Proxy Config Generator',
    description: 'Generate MCP proxy config for Cursor, Claude Desktop, Claude Code, OpenClaw, and custom clients.',
    icon: Wrench,
  },
];

export default function ToolsPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'CollectionPage',
    name: 'AI Agent Orchestration Security Tools and Pricing',
    url: 'https://satgate.io/tools',
    description: metadata.description,
    datePublished: '2026-04-12',
    dateModified: '2026-08-06',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI agent cost control tools' },
      { '@type': 'Thing', name: 'AI agent orchestration security tools and pricing' },
      { '@type': 'Thing', name: 'AI agent security tools pricing' },
      { '@type': 'Thing', name: 'FactSet MCP pricing governance' },
      { '@type': 'Thing', name: 'MCP pricing calculator' },
      { '@type': 'Thing', name: 'economic firewall readiness' },
      { '@type': 'Thing', name: 'MCP governance policy generators' },
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

  const pricingCriteriaJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent orchestration security tool pricing criteria',
    description: 'Criteria for evaluating AI agent orchestration security tools and pricing before selecting controls for MCP, API, model, and paid-tool workflows.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Runtime control surface',
        description: 'Check whether the tool controls agent access to APIs, MCP tools, models, data exports, delegated workers, and paid services before execution.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Pricing unit',
        description: 'Compare pricing by request, token, tool call, agent, workflow, seat, tenant, deployment, or governed spend volume.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'MCP and data-tool pricing',
        description: 'Model third-party tool costs such as FactSet MCP pricing or paid data-enrichment calls as per-tool budget events.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Proof and audit',
        description: 'Require receipts for allow, deny, downgrade, revocation, delegation, spend, and Evidence Pack export.',
      },
    ],
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
          text: 'Start with the AI Agent ROI Calculator if you need a business case, the Runaway Agent Cost Calculator if you need incident exposure, the OpenAI or MCP policy generators if you need enforceable policy, and the Economic Firewall Readiness Grader if you need a gap assessment.',
        },
      },
      {
        '@type': 'Question',
        name: 'How should I compare AI agent orchestration security tools and pricing?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Compare AI agent orchestration security tools by runtime control surface, pricing unit, MCP and paid-tool budget support, revocation, delegation controls, and proof. A useful evaluation separates dashboards from request-path controls that can block, downgrade, route, or charge before execution.',
        },
      },
      {
        '@type': 'Question',
        name: 'How should teams model FactSet MCP pricing or other paid data-tool costs?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Teams should model FactSet MCP pricing and similar paid data-tool costs as per-tool budget events: assign unit prices, cap calls by agent and workflow, check remaining budget before execution, and record Evidence Pack receipts for each allow or deny decision.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do these tools relate to an economic firewall?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'The calculators quantify economic risk. The policy generators turn that risk into budget, routing, revocation, and audit controls. SatGate enforces those controls in the request path as an economic firewall for AI agents.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(itemListJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(pricingCriteriaJsonLd) }} />
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
            <Calculator size={16} /> Free calculators and policy generators
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            AI Agent Orchestration Security Tools and Pricing
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Quantify runaway agent spend, compare orchestration security controls, model MCP and paid-tool pricing, generate enforceable budget policy, and grade economic firewall readiness before autonomous agents hit production scale.
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

      <section className="border-b border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <div className="mb-10 max-w-4xl">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Pricing criteria</p>
            <h2 className="mb-4 text-3xl font-bold text-white">List AI agent orchestration security tools by what they can price, control, and prove</h2>
            <p className="text-lg leading-relaxed text-gray-300">
              Tool lists are useless if they only compare features. For autonomous agents, pricing has to include the governed unit: request, token, MCP tool call, paid data lookup, workflow, agent, tenant, or deployment.
            </p>
          </div>
          <div className="grid gap-5 md:grid-cols-4">
            {[
              ['Runtime control', 'Can it block or downgrade API, MCP, model, data export, and paid-service calls before execution?'],
              ['Pricing unit', 'Does pricing map to seats, requests, tool calls, agents, workflows, tenants, or governed spend?'],
              ['Paid data tools', 'Can FactSet MCP pricing or other paid data-tool costs become per-tool budget policy?'],
              ['Proof', 'Does it export receipts for spend, denial, downgrade, delegation, revocation, and Evidence Packs?'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-2xl border border-gray-800 bg-black p-5">
                <h3 className="mb-2 text-xl font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
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
              'They quantify autonomous agent spend risk, model runaway loops, generate enforceable budget policy, and assess whether economic controls can stop expensive requests before execution.',
            ],
            [
              'Which SatGate tool should I start with?',
              'Start with the AI Agent ROI Calculator if you need a business case, the Runaway Agent Cost Calculator if you need incident exposure, the OpenAI or MCP policy generators if you need enforceable policy, and the Economic Firewall Readiness Grader if you need a gap assessment.',
            ],
            [
              'How should I compare AI agent orchestration security tools and pricing?',
              'Compare runtime control surface, pricing unit, MCP and paid-tool budget support, revocation, delegation controls, and proof. Separate dashboards from request-path controls that can block, downgrade, route, or charge before execution.',
            ],
            [
              'How should teams model FactSet MCP pricing or other paid data-tool costs?',
              'Model FactSet MCP pricing and similar paid data-tool costs as per-tool budget events: assign unit prices, cap calls by agent and workflow, check remaining budget before execution, and record Evidence Pack receipts.',
            ],
            [
              'How do these tools relate to an economic firewall?',
              'The calculators quantify economic risk. The policy generators turn that risk into budget, routing, revocation, and audit controls. SatGate enforces those controls in the request path as an economic firewall for AI agents.',
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

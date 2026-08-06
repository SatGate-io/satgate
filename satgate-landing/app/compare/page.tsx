import Link from 'next/link';
import { ArrowLeft, ArrowRight, BarChart3, Cloud, Eye, KeyRound, Route, Shield, Workflow, Zap } from 'lucide-react';

export const metadata = {
  title: 'AI Gateway Comparison: Cloudflare, Portkey, Helicone, Kong',
  description: 'Compare Cloudflare AI Gateway, Portkey, Helicone, Kong, LiteLLM, LangSmith, and SatGate for routing, observability, spend control, MCP policy, and payments.',
  alternates: { canonical: 'https://satgate.io/compare' },
  keywords: [
    'SatGate comparisons',
    'SatGate vs LiteLLM',
    'SatGate vs Portkey',
    'SatGate vs Helicone',
    'SatGate vs Cloudflare AI Gateway',
    'SatGate vs Kong AI Gateway',
    'SatGate vs Apigee',
    'SatGate vs Tyk',
    'SatGate vs Langfuse',
    'AI gateway comparison',
    'Cloudflare AI Gateway vs LangSmith vs Helicone',
    'Kong AI Gateway vs Portkey',
    'Helicone vs Portkey',
    'LiteLLM vs Portkey',
    'Portkey alternative',
    'economic firewall comparison',
  ],
  openGraph: {
    title: 'AI Gateway Comparison: Cloudflare, Portkey, Helicone, Kong',
    description: 'Compare AI gateways and observability tools against SatGate\'s request-path economic governance for autonomous agents.',
    url: 'https://satgate.io/compare',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI Gateway Comparison: Cloudflare, Portkey, Helicone, Kong',
    description: 'Compare AI gateways, observability tools, and API management platforms against SatGate\'s economic firewall category.',
  },
};

const comparisons = [
  {
    href: '/compare/litellm',
    title: 'SatGate vs LiteLLM',
    description: 'LLM gateway vs economic firewall. LiteLLM routes model access; SatGate governs agent/API spend, MCP tools, and payments.',
    icon: Route,
    color: 'green',
  },
  {
    href: '/compare/portkey',
    title: 'SatGate vs Portkey',
    description: 'GenAI production stack vs economic firewall. Portkey covers gateway, observability, guardrails, and prompts; SatGate governs agent economics.',
    icon: BarChart3,
    color: 'orange',
  },
  {
    href: '/compare/helicone',
    title: 'SatGate vs Helicone',
    description: 'LLM observability vs economic firewall. Helicone helps debug and analyze AI apps; SatGate enforces budgets before agents spend.',
    icon: Eye,
    color: 'blue',
  },
  {
    href: '/compare/aws-agentcore-payments',
    title: 'SatGate vs AWS AgentCore Payments',
    description: 'Managed AWS agent payments vs cross-provider economic firewall. SatGate governs authority, spend, MCP tools, paid rails, and Evidence Packs before execution.',
    icon: Cloud,
    color: 'yellow',
  },
  {
    href: '/compare/cloudflare-ai-gateway',
    title: 'SatGate vs Cloudflare AI Gateway',
    description: 'AI traffic gateway vs pre-execution economic governance. Cloudflare routes and observes AI traffic; SatGate controls delegated spend and authority.',
    icon: Cloud,
    color: 'yellow',
  },
  {
    href: '/compare/langsmith-helicone-datadog',
    title: 'SatGate vs LangSmith, Helicone, Datadog',
    description: 'Observability explains what agents did. SatGate controls what they are allowed to do before they spend, call MCP tools, or cross paid rails.',
    icon: Eye,
    color: 'blue',
  },
  {
    href: '/compare/api-gateway-rate-limits',
    title: 'SatGate vs API Gateway Rate Limits',
    description: 'Rate limits answer how many requests. SatGate answers whether this delegated agent can spend this budget on this resource right now.',
    icon: Shield,
    color: 'green',
  },
  {
    href: '/compare/openai-anthropic-budget-controls',
    title: 'SatGate vs OpenAI / Anthropic Budgets',
    description: 'Native provider budgets are useful guardrails. SatGate adds one cross-provider control layer for agents, MCP tools, APIs, and paid rails.',
    icon: KeyRound,
    color: 'purple',
  },

  {
    href: '/compare/kong-ai-gateway',
    title: 'SatGate vs Kong AI Gateway',
    description: 'API/AI gateway platform vs economic firewall. Kong is strong gateway infrastructure; SatGate governs autonomous agent economics.',
    icon: Workflow,
    color: 'cyan',
  },
  {
    href: '/compare/apigee',
    title: 'SatGate vs Apigee',
    description: 'Enterprise API management vs agent economic governance. Apigee manages APIs; SatGate enforces spend, authority, and payments.',
    icon: Shield,
    color: 'blue',
  },
  {
    href: '/compare/tyk',
    title: 'SatGate vs Tyk',
    description: 'API management vs request-path agent economics. Tyk operates APIs; SatGate controls what agents can spend or access.',
    icon: KeyRound,
    color: 'green',
  },
  {
    href: '/compare/langfuse',
    title: 'SatGate vs Langfuse',
    description: 'LLM observability vs economic firewall. Langfuse traces and evaluates; SatGate blocks over-budget agent requests before execution.',
    icon: Eye,
    color: 'orange',
  },
  {
    href: '/compare/bifrost',
    title: 'SatGate vs Bifrost',
    description: 'LLM routing vs economic governance. Both have MCP — only one enforces per-tool budgets on agent tool calls.',
    icon: Zap,
    color: 'purple',
  },
  {
    href: '/compare/zuplo',
    title: 'SatGate vs Zuplo',
    description: 'API gateway vs economic firewall. Zuplo exposes APIs — SatGate governs spend.',
    icon: Shield,
    color: 'cyan',
  },
  {
    href: '/compare/cloud-native',
    title: 'SatGate vs Cloud-Native AI Governance',
    description: 'Cloud IAM and billing tools vs provider-neutral economic control. SatGate governs agent spend across clouds, MCP tools, and APIs.',
    icon: Cloud,
    color: 'purple',
  },
];

const colorClasses: Record<string, { border: string; bg: string; text: string }> = {
  green: { border: 'hover:border-green-600/50', bg: 'bg-green-900/30', text: 'text-green-400' },
  orange: { border: 'hover:border-orange-600/50', bg: 'bg-orange-900/30', text: 'text-orange-400' },
  blue: { border: 'hover:border-blue-600/50', bg: 'bg-blue-900/30', text: 'text-blue-400' },
  yellow: { border: 'hover:border-yellow-600/50', bg: 'bg-yellow-900/30', text: 'text-yellow-400' },
  purple: { border: 'hover:border-purple-600/50', bg: 'bg-purple-900/30', text: 'text-purple-400' },
  cyan: { border: 'hover:border-cyan-600/50', bg: 'bg-cyan-900/30', text: 'text-cyan-400' },
};

const popularComparisonQuestions = [
  {
    question: 'How does Cloudflare AI Gateway compare to LangSmith or Helicone?',
    answer:
      'Cloudflare AI Gateway sits in the AI traffic path for routing, caching, rate limits, analytics, and provider access. LangSmith and Helicone focus more on tracing, debugging, observability, and evaluation after requests run. SatGate is different again: it is the economic firewall that decides whether an agent has the budget, authority, MCP tool policy, and paid-rail context to make the next call before the spend happens.',
    links: [
      { href: '/compare/cloudflare-ai-gateway', label: 'Cloudflare comparison' },
      { href: '/compare/langsmith-helicone-datadog', label: 'LangSmith and Helicone comparison' },
    ],
  },
  {
    question: 'How does Kong AI Gateway compare to Portkey?',
    answer:
      'Kong AI Gateway is strongest when the buyer wants enterprise gateway infrastructure, traffic control, and platform operations around APIs and AI traffic. Portkey is a GenAI gateway and operations layer for model routing, observability, prompts, and guardrails. SatGate complements or fronts both when the missing requirement is enforceable agent spend, revocable authority, MCP tool budgets, audit evidence, and request-path payment control.',
    links: [
      { href: '/compare/kong-ai-gateway', label: 'Kong comparison' },
      { href: '/compare/portkey', label: 'Portkey comparison' },
    ],
  },
  {
    question: 'How do Helicone and Portkey compare?',
    answer:
      'Helicone is usually evaluated as LLM observability: logs, traces, sessions, cost visibility, and debugging. Portkey spans more of the GenAI gateway stack: routing, retries, observability, prompts, and guardrails. SatGate is not another dashboard in that lane; it enforces whether autonomous agents are allowed to spend, use MCP tools, delegate authority, or cross paid rails before upstream systems execute.',
    links: [
      { href: '/compare/helicone', label: 'Helicone comparison' },
      { href: '/compare/portkey', label: 'Portkey comparison' },
    ],
  },
  {
    question: 'How do LiteLLM and Portkey compare?',
    answer:
      'LiteLLM is typically the provider-abstraction and model-routing layer: one interface across models, keys, and deployments. Portkey adds a broader GenAI gateway and ops surface around routing, observability, prompts, and governance workflows. SatGate owns a different control point: economic policy, budget enforcement, scoped authority, Evidence Packs, revocation, and L402 payment context in the request path.',
    links: [
      { href: '/compare/litellm', label: 'LiteLLM comparison' },
      { href: '/compare/portkey', label: 'Portkey comparison' },
    ],
  },
];

const faqItems = [
  {
    question: 'How is SatGate different from AI gateways?',
    answer:
      'Most AI gateways focus on routing, provider abstraction, caching, rate limits, observability, or prompt operations. SatGate focuses on request-path economic governance: hard budgets, scoped agent authority, MCP tool cost policy, audit evidence, revocation, and paid-rail context at the gateway before forwarding.',
  },
  {
    question: 'Does SatGate replace LiteLLM, Portkey, Helicone, or Cloudflare AI Gateway?',
    answer:
      'Not always. SatGate can sit in front of those systems as the economic firewall. Existing gateways can still handle routing, observability, traces, or provider access while SatGate decides whether an autonomous agent is allowed to spend, access, delegate, or pay.',
  },
  {
    question: 'When should teams use SatGate?',
    answer:
      'Use SatGate when the core problem is autonomous agent risk: runaway spend, MCP tool costs, delegated authority, static API keys, missing revocation, audit gaps, or machine customers that need to pay for API access at request time.',
  },
  {
    question: 'What should teams look for in an AI gateway comparison?',
    answer:
      'Teams should compare routing, provider coverage, observability, caching, rate limits, policy enforcement, budget controls, MCP tool governance, revocation, audit evidence, and whether decisions happen before or after an agent spends money.',
  },
  ...popularComparisonQuestions.map(({ question, answer }) => ({ question, answer })),
];

export default function ComparePage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'CollectionPage',
    name: 'Compare SatGate',
    description: metadata.description,
    url: 'https://satgate.io/compare',
    dateModified: '2026-08-05',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI gateway comparison' },
      { '@type': 'Thing', name: 'Cloudflare AI Gateway vs LangSmith vs Helicone' },
      { '@type': 'Thing', name: 'Kong AI Gateway vs Portkey' },
      { '@type': 'Thing', name: 'Helicone vs Portkey' },
      { '@type': 'Thing', name: 'LiteLLM vs Portkey' },
      { '@type': 'Thing', name: 'Portkey alternative' },
      { '@type': 'Thing', name: 'economic firewall comparison' },
      { '@type': 'Thing', name: 'request-path economic governance' },
      { '@type': 'Thing', name: 'MCP tool cost policy' },
      { '@type': 'Thing', name: 'L402 API payments' },
    ],
  };

  const itemListJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'SatGate comparison pages',
    description: metadata.description,
    dateModified: '2026-08-05',
    about: webPageJsonLd.about,
    itemListElement: comparisons.map((item, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: item.title,
      url: `https://satgate.io${item.href}`,
      description: item.description,
    })),
  };

  const comparisonQuestionJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'Popular AI gateway comparison questions',
    description:
      'Direct answers for high-intent AI gateway comparison searches involving Cloudflare AI Gateway, LangSmith, Helicone, Kong AI Gateway, Portkey, LiteLLM, and SatGate.',
    dateModified: '2026-08-05',
    about: webPageJsonLd.about,
    itemListElement: popularComparisonQuestions.map((item, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: item.question,
      description: item.answer,
      url: 'https://satgate.io/compare',
    })),
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Compare SatGate', item: 'https://satgate.io/compare' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: faqItems.map((item) => ({
      '@type': 'Question',
      name: item.question,
      acceptedAnswer: {
        '@type': 'Answer',
        text: item.answer,
      },
    })),
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(itemListJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(comparisonQuestionJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="mx-auto max-w-6xl px-6 py-16">
        <Link href="/" className="mb-8 flex items-center gap-2 text-gray-500 transition hover:text-white">
          <ArrowLeft size={18} /> Back to Home
        </Link>

        <div className="mb-12 max-w-4xl">
          <div className="mb-6 inline-flex rounded-full border border-cyan-500/30 bg-cyan-950/25 px-4 py-2 text-sm text-cyan-200">
            AI gateway comparisons
          </div>
          <h1 className="mb-5 text-5xl font-extrabold tracking-tight md:text-7xl">Compare SatGate</h1>
          <p className="text-xl leading-relaxed text-gray-300 md:text-2xl">
            Most AI gateways help route, observe, or expose model/API traffic. SatGate focuses on economic governance: hard budgets, MCP tool cost attribution, scoped agent authority, Evidence Packs, and paid-rail context at the gateway before forwarding.
          </p>
        </div>

        <section className="mb-12 border-y border-gray-800 py-8">
          <h2 className="mb-6 text-2xl font-bold text-white">Popular AI gateway comparison questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            {popularComparisonQuestions.map((item) => (
              <article key={item.question} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
                <h3 className="mb-3 text-lg font-bold text-white">{item.question}</h3>
                <p className="mb-4 leading-relaxed text-gray-400">{item.answer}</p>
                <div className="flex flex-wrap gap-3">
                  {item.links.map((link) => (
                    <Link key={link.href} href={link.href} className="inline-flex items-center gap-2 text-sm font-semibold text-cyan-300 transition hover:text-cyan-100">
                      {link.label} <ArrowRight size={14} />
                    </Link>
                  ))}
                </div>
              </article>
            ))}
          </div>
        </section>

        <div className="grid gap-5 md:grid-cols-2">
          {comparisons.map(({ href, title, description, icon: Icon, color }) => {
            const classes = colorClasses[color];
            return (
              <Link key={href} href={href} className={`group block rounded-xl border border-gray-800 bg-gray-900 p-6 transition ${classes.border}`}>
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <div className="mb-3 flex items-center gap-3">
                      <div className={`rounded-lg p-2 ${classes.bg}`}>
                        <Icon className={classes.text} size={20} />
                      </div>
                      <h2 className="text-xl font-bold text-white">{title}</h2>
                    </div>
                    <p className="leading-relaxed text-gray-400">{description}</p>
                  </div>
                  <ArrowRight className={`mt-2 shrink-0 text-gray-600 transition group-hover:${classes.text}`} size={24} />
                </div>
              </Link>
            );
          })}
        </div>

        <section className="mt-12 rounded-2xl border border-cyan-800/30 bg-gradient-to-r from-cyan-950/20 to-purple-950/20 p-8">
          <h2 className="mb-4 text-2xl font-bold text-white">The short version</h2>
          <p className="max-w-4xl leading-relaxed text-gray-300">
            LiteLLM, Portkey, Helicone, Cloudflare AI Gateway, Kong AI Gateway, Apigee, Tyk, Langfuse, Bifrost, and Zuplo are useful infrastructure. The difference is category: SatGate is the economic firewall — the request-path layer that decides what autonomous agents can spend, access, delegate, revoke, audit, or pay for before the next call executes.
          </p>
        </section>

        <section className="mt-12 rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-2xl font-bold text-white">Comparison FAQ</h2>
          <div className="space-y-5">
            {faqItems.map(({ question, answer }) => (
              <div key={question}>
                <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                <p className="leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </section>
      </section>
    </main>
  );
}

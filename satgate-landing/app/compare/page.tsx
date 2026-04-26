import Link from 'next/link';
import { ArrowLeft, ArrowRight, BarChart3, Cloud, Eye, KeyRound, Route, Shield, Workflow, Zap } from 'lucide-react';

export const metadata = {
  title: 'Compare SatGate - AI Gateway and Economic Firewall Comparisons',
  description: 'Compare SatGate with LiteLLM, Portkey, Helicone, Cloudflare AI Gateway, Kong AI Gateway, Apigee, Tyk, Langfuse, Bifrost, Zuplo, and other AI infrastructure. Routing is not economic governance.',
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
    'economic firewall comparison',
  ],
  openGraph: {
    title: 'Compare SatGate - AI Gateway and Economic Firewall Comparisons',
    description: 'Routing, observability, and API gateways are useful. SatGate focuses on request-path economic governance for autonomous agents.',
    url: 'https://satgate.io/compare',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Compare SatGate - AI Gateway and Economic Firewall Comparisons',
    description: 'Compare AI gateways, observability tools, and API management platforms against SatGate’s economic firewall category.',
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
    description: 'GenAI production stack vs economic control plane. Portkey covers gateway, observability, guardrails, and prompts; SatGate governs agent economics.',
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
    href: '/compare/cloudflare-ai-gateway',
    title: 'SatGate vs Cloudflare AI Gateway',
    description: 'AI traffic gateway vs agent economic governance. Cloudflare brings analytics, caching, rate limits, and fallback; SatGate controls spend and authority.',
    icon: Cloud,
    color: 'yellow',
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
];

const colorClasses: Record<string, { border: string; bg: string; text: string }> = {
  green: { border: 'hover:border-green-600/50', bg: 'bg-green-900/30', text: 'text-green-400' },
  orange: { border: 'hover:border-orange-600/50', bg: 'bg-orange-900/30', text: 'text-orange-400' },
  blue: { border: 'hover:border-blue-600/50', bg: 'bg-blue-900/30', text: 'text-blue-400' },
  yellow: { border: 'hover:border-yellow-600/50', bg: 'bg-yellow-900/30', text: 'text-yellow-400' },
  purple: { border: 'hover:border-purple-600/50', bg: 'bg-purple-900/30', text: 'text-purple-400' },
  cyan: { border: 'hover:border-cyan-600/50', bg: 'bg-cyan-900/30', text: 'text-cyan-400' },
};

export default function ComparePage() {
  const itemListJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'SatGate comparison pages',
    description: metadata.description,
    itemListElement: comparisons.map((item, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: item.title,
      url: `https://satgate.io${item.href}`,
      description: item.description,
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
    mainEntity: [
      {
        '@type': 'Question',
        name: 'How is SatGate different from AI gateways?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Most AI gateways focus on routing, provider abstraction, caching, rate limits, observability, or prompt operations. SatGate focuses on request-path economic governance: hard budgets, scoped agent authority, MCP tool cost policy, audit evidence, revocation, and L402 payments before upstream access.',
        },
      },
      {
        '@type': 'Question',
        name: 'Does SatGate replace LiteLLM, Portkey, Helicone, or Cloudflare AI Gateway?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Not always. SatGate can sit in front of those systems as the economic firewall. Existing gateways can still handle routing, observability, traces, or provider access while SatGate decides whether an autonomous agent is allowed to spend, access, delegate, or pay.',
        },
      },
      {
        '@type': 'Question',
        name: 'When should teams use SatGate?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Use SatGate when the core problem is autonomous agent risk: runaway spend, MCP tool costs, delegated authority, static API keys, missing revocation, audit gaps, or machine customers that need to pay for API access at request time.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(itemListJsonLd) }} />
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
            Most AI gateways help route, observe, or expose model/API traffic. SatGate focuses on economic governance: hard budgets, MCP tool cost attribution, scoped agent authority, audit trails, and L402 payments before upstream access.
          </p>
        </div>

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
            {[
              [
                'How is SatGate different from AI gateways?',
                'AI gateways usually handle routing, provider access, caching, rate limits, observability, or prompt operations. SatGate handles economic governance: budgets, scoped authority, MCP tool policy, audit evidence, revocation, and payments before upstream access.',
              ],
              [
                'Does SatGate replace existing gateways?',
                'Not necessarily. SatGate can sit in front of LiteLLM, Portkey, Helicone, Cloudflare AI Gateway, Kong, Apigee, Tyk, Bifrost, Zuplo, or custom APIs as the economic firewall layer.',
              ],
              [
                'When is SatGate the better fit?',
                'SatGate is the better fit when autonomous agents create spend, MCP tool-call, delegation, revocation, audit, or machine-payment risk that must be controlled in the request path.',
              ],
            ].map(([question, answer]) => (
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

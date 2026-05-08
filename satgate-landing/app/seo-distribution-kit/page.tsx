import Link from 'next/link';
import { ArrowRight, Megaphone, Radio, Send, Sparkles } from 'lucide-react';

export const metadata = {
  title: 'SatGate SEO Distribution Kit: AI Agent Cost Control Launch Copy',
  description: 'Distribution copy, social snippets, backlink targets, and launch positioning for SatGate AI agent cost control tools and runaway spend data assets.',
  alternates: { canonical: 'https://satgate.io/seo-distribution-kit' },
  keywords: ['AI agent cost control distribution', 'SatGate launch kit', 'AI agent spend control social copy', 'economic firewall distribution'],
  openGraph: {
    title: 'SatGate SEO Distribution Kit',
    description: 'Launch copy, promotion angles, and backlink targets for AI agent cost control tools and runaway spend data assets.',
    url: 'https://satgate.io/seo-distribution-kit',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate SEO Distribution Kit',
    description: 'Distribution copy and backlink targets for AI agent cost control and economic firewall assets.',
  },
};

const assets = [
  ['Tools hub', '/tools'],
  ['AI agent cost control', '/ai-agent-cost-control'],
  ['Economic firewall', '/economic-firewall'],
  ['LLM cost dashboard', '/llm-cost-dashboard'],
  ['LLM cost monitoring', '/llm-cost-monitoring'],
  ['Revocable capability-token policy template', '/revocable-capability-token-policy-template'],
  ['Agent spend policy template', '/agent-spend-policy-template'],
  ['Runaway spend index', '/ai-agent-runaway-spend-index'],
  ['MCP proxy config generator', '/mcp-proxy-config-generator'],
  ['Runaway spend benchmark', '/ai-agent-runaway-spend-benchmark'],
  ['Governance dashboard demo', '/dashboard'],
  ['Protect / scoped credential demo', '/protect'],
  ['L402 monetization demo', '/monetize'],
  ['Economic firewall readiness grader', '/economic-firewall-readiness-grader'],
  ['L402 API pricing calculator', '/l402-api-pricing-calculator'],
];

const distributionMotions = [
  ['Category definition', 'Lead with “economic control plane for AI agents” and explain why routing, logs, and IAM are not enough for autonomous spend.'],
  ['Problem proof', 'Point cost-control traffic to runaway spend benchmarks, dashboard pages, calculators, and policy generators that show the risk in operational terms.'],
  ['Security bridge', 'Route security audiences toward revocable credentials, capability-token caveats, API-key risk, and request-path authorization controls.'],
  ['Monetization bridge', 'Route API and data-product audiences toward L402, HTTP 402, robot-customer payments, and pricing calculator assets.'],
];

export default function SeoDistributionKitPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'CollectionPage',
    name: 'SatGate SEO Distribution Kit',
    url: 'https://satgate.io/seo-distribution-kit',
    description: metadata.description,
    datePublished: '2026-04-26',
    dateModified: '2026-05-06',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI agent cost control distribution' },
      { '@type': 'Thing', name: 'economic firewall launch positioning' },
      { '@type': 'Thing', name: 'MCP governance backlink strategy' },
      { '@type': 'Thing', name: 'runaway agent spend data assets' },
      { '@type': 'Thing', name: 'L402 API monetization promotion' },
    ],
    audience: { '@type': 'Audience', audienceType: 'Founders, developer advocates, content teams, sales engineers, and partners' },
  };

  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'SatGate SEO Distribution Kit: AI Agent Cost Control Launch Copy',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-05-06',
    mainEntityOfPage: 'https://satgate.io/seo-distribution-kit',
    about: assets.map(([name, url]) => ({ '@type': 'WebPage', name, url: `https://satgate.io${url}` })),
    mentions: webPageJsonLd.about,
    audience: webPageJsonLd.audience,
  };

  const distributionMotionsJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'SatGate SEO distribution motions',
    description: 'Reusable promotion motions for category definition, problem proof, security routing, and L402 monetization routing.',
    itemListElement: distributionMotions.map(([name, description], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name,
      description,
    })),
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'SEO Distribution Kit', item: 'https://satgate.io/seo-distribution-kit' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is the SatGate SEO Distribution Kit?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'The SatGate SEO Distribution Kit packages launch copy, backlink angles, promotion targets, and reusable positioning for AI agent cost control, economic firewall, MCP governance, and L402 payment assets.',
        },
      },
      {
        '@type': 'Question',
        name: 'Which SatGate pages should be promoted first?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Promote the tools hub, AI agent cost control, economic firewall, runaway spend index, ROI calculator, MCP policy generators, revocable capability-token template, L402 pricing calculator, governance dashboard, Protect demo, and monetization demo first.',
        },
      },
      {
        '@type': 'Question',
        name: 'Who should use the distribution kit?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'The kit is for founders, developer advocates, content teams, sales engineers, and partners who need consistent language for SatGate as the economic control plane for AI agents.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(distributionMotionsJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(168,85,247,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-purple-500/30 bg-purple-950/30 px-4 py-2 text-sm text-purple-200">
            <Megaphone size={16} /> Distribution package
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">SatGate SEO Distribution Kit</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Copy, launch angles, and promotion targets for SatGate&apos;s AI agent cost control tools, LLM cost dashboard guides, MCP policy generators, revocable capability-token templates, L402 pricing calculator, and recurring runaway spend data assets.
          </p>
          <a href="/distribution/satgate-seo-machine-launch-kit.md" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
            Download Markdown kit <ArrowRight size={18} />
          </a>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-6 px-6 py-20 lg:grid-cols-3">
        {[
          ['LinkedIn angle', 'AI agents create economic risk, not just security risk. SatGate controls spend before the request executes.', Radio],
          ['Short social angle', 'Dashboards report agent spend. Economic firewalls decide whether the next request is allowed to spend.', Send],
          ['Backlink angle', 'Original benchmark data, dashboard checklists, and policy templates for FinOps, MCP, API monetization, and agent security communities.', Sparkles],
        ].map(([title, body, Icon]) => (
          <div key={title as string} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <Icon className="mb-4 text-purple-300" size={28} />
            <h2 className="mb-3 text-2xl font-bold text-white">{title as string}</h2>
            <p className="leading-relaxed text-gray-400">{body as string}</p>
          </div>
        ))}
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-8 text-3xl font-bold text-white">Distribution motions</h2>
          <div className="mb-12 grid gap-4 md:grid-cols-4">
            {distributionMotions.map(([title, body], index) => (
              <div key={title} className="rounded-2xl border border-purple-900/40 bg-purple-950/10 p-5">
                <p className="mb-3 text-xs font-mono text-purple-300">0{index + 1}</p>
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
          <h2 className="mb-8 text-3xl font-bold text-white">Promote these pages first</h2>
          <div className="grid gap-4 md:grid-cols-2">
            {assets.map(([label, href]) => (
              <Link key={href} href={href} className="rounded-2xl border border-gray-800 bg-black p-6 transition hover:border-purple-500/50">
                <h3 className="mb-2 text-xl font-bold text-white">{label}</h3>
                <p className="text-cyan-300">satgate.io{href}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-4xl px-6 py-20">
        <p className="mb-2 text-center text-sm font-mono uppercase tracking-wide text-purple-300">FAQ</p>
        <h2 className="mb-8 text-center text-3xl font-bold text-white">Distribution kit questions</h2>
        <div className="space-y-5">
          {[
            ['What is the SatGate SEO Distribution Kit?', 'The SatGate SEO Distribution Kit packages launch copy, backlink angles, promotion targets, and reusable positioning for AI agent cost control, economic firewall, MCP governance, and L402 payment assets.'],
            ['Which SatGate pages should be promoted first?', 'Promote the tools hub, AI agent cost control, economic firewall, runaway spend index, ROI calculator, MCP policy generators, revocable capability-token template, L402 pricing calculator, governance dashboard, Protect demo, and monetization demo first.'],
            ['Who should use the distribution kit?', 'The kit is for founders, developer advocates, content teams, sales engineers, and partners who need consistent language for SatGate as the economic control plane for AI agents.'],
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

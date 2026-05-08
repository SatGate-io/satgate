import Link from 'next/link';
import { ArrowRight, BarChart3, Download, Gauge, ShieldCheck, Zap } from 'lucide-react';

export const metadata = {
  title: 'AI Agent Runaway Spend Index: Monthly Cost Risk Benchmark',
  description: 'Monthly SatGate index tracking modeled runaway AI agent spend, MCP tool cost failures, retry loops, fanout risk, and avoided cost from request-path controls.',
  alternates: { canonical: 'https://satgate.io/ai-agent-runaway-spend-index' },
  keywords: [
    'AI agent runaway spend index',
    'AI agent cost benchmark',
    'MCP tool cost benchmark',
    'runaway AI spend',
    'AI agent spend control data',
  ],
  openGraph: {
    title: 'AI Agent Runaway Spend Index',
    description: 'Monthly benchmark for runaway AI agent spend, MCP tool cost failures, and avoided cost from request-path controls.',
    url: 'https://satgate.io/ai-agent-runaway-spend-index',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI Agent Runaway Spend Index',
    description: 'Monthly benchmark for runaway AI agent spend and MCP tool cost failures.',
  },
};

const scenarios = [
  ['OpenAI retry loop', '$7,200', '$250', '96.5%'],
  ['MCP browser automation loop', '$1,840', '$120', '93.5%'],
  ['Sub-agent research fanout', '$18,480', '$900', '95.1%'],
  ['Paid data API polling loop', '$9,600', '$600', '93.8%'],
  ['Multi-tenant agent swarm', '$134,400', '$6,000', '95.5%'],
];

const controlChecklist = [
  ['Price expensive tools', 'Assign explicit costs to model calls, MCP tools, browser automation, paid data APIs, and fanout workflows.'],
  ['Cap each session', 'Set per-agent and per-session budgets so retry loops and browser storms hit a hard stop.'],
  ['Constrain delegation', 'Carve sub-agent budgets from parent budgets so fanout cannot multiply spend authority.'],
  ['Require revocation', 'Keep kill switches and expiring capability tokens in place for compromised or inefficient agents.'],
  ['Monitor avoided spend', 'Track controlled-vs-uncontrolled deltas so finance sees dollars avoided, not just alerts fired.'],
];

const webPageJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'WebPage',
  name: 'AI Agent Runaway Spend Index',
  description: metadata.description,
  url: 'https://satgate.io/ai-agent-runaway-spend-index',
  dateModified: '2026-05-06',
  isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
  about: [
    { '@type': 'Thing', name: 'AI agent runaway spend' },
    { '@type': 'Thing', name: 'AI agent cost benchmark' },
    { '@type': 'Thing', name: 'MCP tool cost failures' },
    { '@type': 'Thing', name: 'request-path cost controls' },
    { '@type': 'Thing', name: 'delegated sub-agent fanout risk' },
  ],
};

const jsonLd = {
  '@context': 'https://schema.org',
  '@type': 'Dataset',
  name: 'SatGate AI Agent Runaway Spend Index — April 2026',
  description: 'Monthly benchmark of modeled runaway AI agent spend, MCP tool cost failure modes, retry loops, fanout risk, and avoided cost from request-path controls.',
  url: 'https://satgate.io/ai-agent-runaway-spend-index',
  creator: { '@type': 'Organization', name: 'SatGate' },
  datePublished: '2026-04-26',
  dateModified: '2026-05-06',
  about: webPageJsonLd.about,
  keywords: ['AI agent spend control', 'MCP cost control', 'runaway AI spend', 'economic firewall'],
  distribution: [
    { '@type': 'DataDownload', encodingFormat: 'application/json', contentUrl: 'https://satgate.io/data/ai-agent-runaway-spend-index-2026-04.json' },
    { '@type': 'DataDownload', encodingFormat: 'text/csv', contentUrl: 'https://satgate.io/data/ai-agent-runaway-spend-index-2026-04.csv' },
  ],
};

const controlChecklistJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'ItemList',
  name: 'AI agent runaway spend control checklist',
  description: 'Controls teams should apply after reviewing runaway AI agent spend benchmark scenarios.',
  itemListElement: controlChecklist.map(([name, description], index) => ({
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
    { '@type': 'ListItem', position: 2, name: 'AI Agent Runaway Spend Index', item: 'https://satgate.io/ai-agent-runaway-spend-index' },
  ],
};

const faqJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'FAQPage',
  mainEntity: [
    {
      '@type': 'Question',
      name: 'What is the AI Agent Runaway Spend Index?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'The AI Agent Runaway Spend Index is a recurring benchmark of modeled autonomous agent cost failures, including retry loops, MCP tool storms, delegated fanout, paid API polling, and avoided spend from request-path controls.',
      },
    },
    {
      '@type': 'Question',
      name: 'Why do runaway AI agents create cost risk?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Agents can loop, retry, delegate, and call paid tools or APIs much faster than humans. Without request-path budgets and kill switches, small mistakes can become expensive incidents before dashboards report the damage.',
      },
    },
    {
      '@type': 'Question',
      name: 'How does SatGate reduce runaway agent spend?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'SatGate reduces runaway agent spend by enforcing per-request budgets, MCP tool cost policy, revocable capabilities, delegation caps, audit requirements, and kill switches before upstream calls execute.',
      },
    },
    {
      '@type': 'Question',
      name: 'Which failure modes does the index track?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'The index tracks OpenAI retry loops, MCP browser automation loops, sub-agent research fanout, paid data API polling loops, and multi-tenant agent swarms.',
      },
    },
    {
      '@type': 'Question',
      name: 'How should teams use the index?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Teams should use the index as a control-plan checklist: price expensive tools, set per-request and session budgets, cap delegation, require revocable capabilities, and block loops in the request path.',
      },
    },
  ],
};

export default function AiAgentRunawaySpendIndexPage() {
  return (
    <main className="min-h-screen bg-black text-gray-100">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(controlChecklistJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_10%,rgba(249,115,22,0.18),transparent_30%),radial-gradient(circle_at_80%_0%,rgba(34,211,238,0.16),transparent_28%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-orange-500/30 bg-orange-950/30 px-4 py-2 text-sm text-orange-200">
            <BarChart3 size={16} /> April 2026 index
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            AI Agent Runaway Spend Index
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            A recurring benchmark for autonomous agent cost failures: retry loops, MCP tool storms, delegated sub-agent fanout, paid data API polling, and the spend avoided by request-path controls.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <a href="/data/ai-agent-runaway-spend-index-2026-04.json" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              <Download size={18} /> Download JSON
            </a>
            <a href="/data/ai-agent-runaway-spend-index-2026-04.csv" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-orange-500">
              <Download size={18} /> Download CSV
            </a>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="grid gap-5 md:grid-cols-4">
          {[
            ['Median incident', '$1,840', Gauge],
            ['P90 incident', '$18,480', Zap],
            ['Largest modeled', '$134,400', BarChart3],
            ['Median avoided', '94.2%', ShieldCheck],
          ].map(([label, value, Icon]) => (
            <div key={label as string} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
              <Icon className="mb-4 text-orange-300" size={28} />
              <div className="text-sm text-gray-500">{label as string}</div>
              <div className="mt-2 text-3xl font-extrabold text-white">{value as string}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-6 text-3xl font-bold text-white">April 2026 modeled incidents</h2>
          <div className="overflow-hidden rounded-2xl border border-gray-800">
            <table className="w-full min-w-[760px] border-collapse text-left text-sm">
              <thead className="bg-black text-gray-300">
                <tr>{['Failure mode', 'Uncontrolled', 'Controlled', 'Avoided'].map((h) => <th key={h} className="border-b border-gray-800 px-4 py-4 font-semibold">{h}</th>)}</tr>
              </thead>
              <tbody>
                {scenarios.map(([mode, uncontrolled, controlled, avoided]) => (
                  <tr key={mode} className="border-b border-gray-900 bg-gray-950/60 last:border-0">
                    <td className="px-4 py-4 font-semibold text-white">{mode}</td>
                    <td className="px-4 py-4 font-bold text-orange-300">{uncontrolled}</td>
                    <td className="px-4 py-4 font-bold text-cyan-300">{controlled}</td>
                    <td className="px-4 py-4 font-bold text-green-400">{avoided}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <p className="mb-2 text-sm font-mono uppercase tracking-wide text-orange-300">Control checklist</p>
        <h2 className="mb-8 text-3xl font-bold text-white">Turn the index into request-path controls</h2>
        <div className="grid gap-4 md:grid-cols-5">
          {controlChecklist.map(([title, body], index) => (
            <div key={title} className="rounded-2xl border border-orange-900/40 bg-orange-950/10 p-5">
              <p className="mb-3 text-xs font-mono text-orange-300">0{index + 1}</p>
              <h3 className="mb-2 font-bold text-white">{title}</h3>
              <p className="text-sm leading-relaxed text-gray-400">{body}</p>
            </div>
          ))}
        </div>
        <div className="mt-8 flex flex-col gap-3 sm:flex-row">
          <Link href="/llm-cost-dashboard" className="inline-flex items-center justify-center rounded-lg border border-gray-700 px-5 py-3 text-sm font-semibold text-gray-300 transition hover:border-cyan-500 hover:text-white">See cost telemetry</Link>
          <Link href="/mcp-tool-cost-policy-generator" className="inline-flex items-center justify-center rounded-lg border border-gray-700 px-5 py-3 text-sm font-semibold text-gray-300 transition hover:border-orange-500 hover:text-white">Generate MCP tool policy</Link>
          <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center rounded-lg border border-gray-700 px-5 py-3 text-sm font-semibold text-gray-300 transition hover:border-green-500 hover:text-white">Grade readiness</Link>
        </div>
      </section>

      <section className="mx-auto max-w-4xl px-6 py-20">
        <h2 className="mb-8 text-3xl font-bold text-white">Runaway spend index FAQ</h2>
        <div className="space-y-5">
          {[
            ['What is the AI Agent Runaway Spend Index?', 'The AI Agent Runaway Spend Index is a recurring benchmark of modeled autonomous agent cost failures, including retry loops, MCP tool storms, delegated fanout, paid API polling, and avoided spend from request-path controls.'],
            ['Why do runaway AI agents create cost risk?', 'Agents can loop, retry, delegate, and call paid tools or APIs much faster than humans. Without request-path budgets and kill switches, small mistakes can become expensive incidents before dashboards report the damage.'],
            ['How does SatGate reduce runaway agent spend?', 'SatGate reduces runaway agent spend by enforcing per-request budgets, MCP tool cost policy, revocable capabilities, delegation caps, audit requirements, and kill switches before upstream calls execute.'],
            ['Which failure modes does the index track?', 'The index tracks OpenAI retry loops, MCP browser automation loops, sub-agent research fanout, paid data API polling loops, and multi-tenant agent swarms.'],
            ['How should teams use the index?', 'Teams should use the index as a control-plan checklist: price expensive tools, set per-request and session budgets, cap delegation, require revocable capabilities, and block loops in the request path.'],
          ].map(([question, answer]) => (
            <div key={question} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-3 text-xl font-bold text-white">{question}</h3>
              <p className="leading-relaxed text-gray-400">{answer}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-orange-900/60 bg-gradient-to-br from-orange-950/40 to-cyan-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Use the index as a control-plan checklist</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            The pattern is consistent: agent spend incidents are not solved by better dashboards. They are solved by request-path budget enforcement, MCP tool cost policy, revocable capabilities, delegation caps, and kill switches before upstream calls execute.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/agent-spend-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Generate spend policy <ArrowRight size={18} /></Link>
            <Link href="/llm-cost-dashboard" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">See cost dashboard</Link>
            <Link href="/ai-agent-runaway-spend-benchmark" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-orange-500">Read benchmark methodology</Link>
          </div>
        </div>
      </section>
    </main>
  );
}

import Link from 'next/link';
import { ArrowRight, BarChart3, Bot, Clock, DollarSign, Gauge, ShieldCheck, Workflow, Zap } from 'lucide-react';

export const metadata = {
  title: 'AI Agent Runaway Spend Benchmark',
  description: 'Original benchmark modeling how fast autonomous AI agents can create runaway API, model, and MCP tool spend without request-path budget enforcement.',
  alternates: { canonical: 'https://satgate.io/ai-agent-runaway-spend-benchmark' },
  keywords: [
    'AI agent runaway spend benchmark',
    'AI agent cost benchmark',
    'runaway AI agent spend',
    'agent loop cost benchmark',
    'MCP tool spend benchmark',
    'AI agent budget enforcement benchmark',
    'economic firewall benchmark',
  ],
  openGraph: {
    title: 'AI Agent Runaway Spend Benchmark',
    description: 'A practical benchmark for agent loops, retries, MCP tool fanout, detection delay, and request-path budget enforcement.',
    url: 'https://satgate.io/ai-agent-runaway-spend-benchmark',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI Agent Runaway Spend Benchmark',
    description: 'How fast autonomous agents can burn API and MCP tool budget without an economic firewall.',
  },
};

const scenarios = [
  { name: 'Single coding agent loop', agents: 1, callsPerMinute: 18, fanout: 1, costPerCall: '$0.06', detectionDelay: '45 min', uncontrolledCost: '$49', controlledCost: '$5', avoided: '90%' },
  { name: 'MCP tool retry storm', agents: 12, callsPerMinute: 8, fanout: 3, costPerCall: '$0.12', detectionDelay: '60 min', uncontrolledCost: '$2,074', controlledCost: '$173', avoided: '92%' },
  { name: 'Support-agent swarm', agents: 50, callsPerMinute: 6, fanout: 4, costPerCall: '$0.04', detectionDelay: '90 min', uncontrolledCost: '$4,320', controlledCost: '$240', avoided: '94%' },
  { name: 'Premium research workflow', agents: 20, callsPerMinute: 10, fanout: 5, costPerCall: '$0.25', detectionDelay: '30 min', uncontrolledCost: '$7,500', controlledCost: '$1,250', avoided: '83%' },
  { name: 'Enterprise background agents', agents: 200, callsPerMinute: 4, fanout: 2, costPerCall: '$0.03', detectionDelay: '120 min', uncontrolledCost: '$5,760', controlledCost: '$240', avoided: '96%' },
];

const findings = [
  { icon: Clock, title: 'Detection delay dominates cost', body: 'A dashboard that notices spend after 30-120 minutes is too late. The expensive decision has already happened thousands of times.' },
  { icon: Workflow, title: 'Fanout multiplies every mistake', body: 'Sub-agents, MCP tools, retries, and background workers turn one bad loop into a parallel spend event.' },
  { icon: DollarSign, title: 'Small unit costs still become material', body: 'A few cents per call looks harmless until agents generate thousands of paid requests before anyone sees the bill.' },
  { icon: ShieldCheck, title: 'Inline enforcement changes the curve', body: 'Budget checks, per-tool caps, route policy, expiry, and revocation stop the next request instead of explaining the last one.' },
];

export default function AiAgentRunawaySpendBenchmarkPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'AI Agent Runaway Spend Benchmark',
    description: metadata.description,
    url: 'https://satgate.io/ai-agent-runaway-spend-benchmark',
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-04-26',
    about: ['AI agent cost control', 'MCP budget enforcement', 'Economic firewall', 'Runaway agent spend'],
  };

  const datasetJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'Dataset',
    name: 'AI Agent Runaway Spend Benchmark Scenarios',
    description: 'Modeled benchmark scenarios estimating uncontrolled and request-path controlled spend for autonomous AI agent loops, MCP retry storms, and agent swarms.',
    creator: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    license: 'https://satgate.io/terms',
    variableMeasured: ['agents', 'calls per minute', 'fanout', 'cost per call', 'detection delay', 'uncontrolled cost', 'controlled cost'],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is AI agent runaway spend?',
        acceptedAnswer: { '@type': 'Answer', text: 'AI agent runaway spend is cost created when autonomous agents loop, retry, delegate, or continue calling paid APIs and MCP tools after the work is no longer economically justified.' },
      },
      {
        '@type': 'Question',
        name: 'Why do dashboards fail to control runaway agent cost?',
        acceptedAnswer: { '@type': 'Answer', text: 'Dashboards report spend after requests complete. Autonomous agents can generate hundreds or thousands of paid calls before a human sees an alert, so enforcement has to happen before forwarding each request.' },
      },
      {
        '@type': 'Question',
        name: 'How does an economic firewall reduce runaway spend?',
        acceptedAnswer: { '@type': 'Answer', text: 'An economic firewall checks identity, budget, route, tool scope, request cost, expiry, and revocation before upstream access, blocking the next expensive request when policy says stop.' },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(datasetJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_24%_0%,rgba(249,115,22,0.18),transparent_34%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.16),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-orange-500/30 bg-orange-950/30 px-4 py-2 text-sm text-orange-200">
            <BarChart3 size={16} /> Original agent spend benchmark
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">AI Agent Runaway Spend Benchmark</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Autonomous agents do not need malicious intent to create expensive incidents. Loops, retries, delegated sub-agents, and MCP tool fanout can turn small unit costs into thousands of dollars before a dashboard catches up.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/runaway-agent-cost-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Model your exposure <ArrowRight size={18} /></Link>
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-orange-500">Learn economic firewalls</Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[0.9fr_1.1fr]">
        <div>
          <h2 className="mb-5 text-3xl font-bold text-white">Benchmark method</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>This benchmark models common autonomous-agent failure modes using five variables: active agents, paid calls per minute, delegation fanout, cost per call, and detection delay.</p>
            <p>Uncontrolled cost assumes the loop continues until a human, dashboard alert, or provider billing alarm catches it. Controlled cost assumes a request-path economic firewall stops new paid calls after five minutes through budget, per-tool cap, route policy, expiry, or revocation.</p>
            <p>The point is not that every workload has these exact numbers. The point is the curve: once agents can act in parallel, cost grows with time and fanout faster than humans can approve individual requests.</p>
          </div>
        </div>
        <div className="rounded-2xl border border-orange-900/50 bg-orange-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">Formula</h3>
          <div className="rounded-xl border border-gray-800 bg-black p-5 font-mono text-sm leading-relaxed text-orange-100">cost = agents × calls/min × fanout × minutes × cost/call</div>
          <div className="mt-5 space-y-3 text-gray-300">
            <p><strong className="text-white">Uncontrolled:</strong> minutes = detection delay</p>
            <p><strong className="text-white">Controlled:</strong> minutes = five-minute enforcement window</p>
            <p><strong className="text-white">Avoided:</strong> cost blocked before the next upstream API or MCP tool call</p>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-7xl px-6 py-20">
          <div className="mb-8 flex flex-col justify-between gap-4 md:flex-row md:items-end">
            <div>
              <h2 className="mb-3 text-3xl font-bold text-white">Benchmark scenarios</h2>
              <p className="max-w-3xl text-gray-400">Representative agent failure modes, modeled with and without request-path budget enforcement.</p>
            </div>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center gap-2 font-semibold text-cyan-300 hover:text-cyan-200">See AI agent cost control <ArrowRight size={16} /></Link>
          </div>
          <div className="overflow-hidden rounded-2xl border border-gray-800">
            <div className="overflow-x-auto">
              <table className="w-full min-w-[980px] border-collapse text-left text-sm">
                <thead className="bg-black text-gray-300">
                  <tr>{['Scenario', 'Agents', 'Calls/min', 'Fanout', 'Cost/call', 'Detection', 'Uncontrolled', 'Controlled', 'Avoided'].map((head) => <th key={head} className="border-b border-gray-800 px-4 py-4 font-semibold">{head}</th>)}</tr>
                </thead>
                <tbody>
                  {scenarios.map((scenario) => (
                    <tr key={scenario.name} className="border-b border-gray-900 bg-gray-950/60 last:border-0">
                      <td className="px-4 py-4 font-semibold text-white">{scenario.name}</td>
                      <td className="px-4 py-4 text-gray-300">{scenario.agents}</td>
                      <td className="px-4 py-4 text-gray-300">{scenario.callsPerMinute}</td>
                      <td className="px-4 py-4 text-gray-300">{scenario.fanout}×</td>
                      <td className="px-4 py-4 text-gray-300">{scenario.costPerCall}</td>
                      <td className="px-4 py-4 text-gray-300">{scenario.detectionDelay}</td>
                      <td className="px-4 py-4 font-bold text-orange-300">{scenario.uncontrolledCost}</td>
                      <td className="px-4 py-4 font-bold text-cyan-300">{scenario.controlledCost}</td>
                      <td className="px-4 py-4 font-bold text-green-400">{scenario.avoided}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <h2 className="mb-8 text-3xl font-bold text-white">Findings</h2>
        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
          {findings.map(({ icon: Icon, title, body }) => (
            <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-6 transition hover:border-orange-900/70">
              <Icon className="mb-4 text-orange-300" size={28} />
              <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
              <p className="leading-relaxed text-gray-400">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-black">
        <div className="mx-auto grid max-w-6xl gap-8 px-6 py-20 lg:grid-cols-3">
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6"><Zap className="mb-4 text-yellow-300" size={28} /><h2 className="mb-3 text-2xl font-bold text-white">Observe</h2><p className="leading-relaxed text-gray-400">Route agent traffic through SatGate to attribute cost by agent, workflow, route, tool, tenant, and MCP server before enforcing hard limits.</p></div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6"><Gauge className="mb-4 text-cyan-300" size={28} /><h2 className="mb-3 text-2xl font-bold text-white">Control</h2><p className="leading-relaxed text-gray-400">Enforce per-agent budgets, per-tool caps, route policy, revocation, expiry, and kill switches before upstream API calls execute.</p></div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6"><Bot className="mb-4 text-purple-300" size={28} /><h2 className="mb-3 text-2xl font-bold text-white">Charge</h2><p className="leading-relaxed text-gray-400">When external agents become API customers, use SatGate Charge with L402 Lightning payments to collect before access is granted.</p></div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-orange-900/60 bg-gradient-to-br from-orange-950/40 to-cyan-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">The fix is not a better bill. It is a pre-request decision.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">SatGate is the economic control plane for AI agents: observe cost, control spend before execution, and charge robot customers when autonomous systems need paid API access.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/mcp-budget-enforcement" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">MCP budget enforcement <ArrowRight size={18} /></Link>
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-orange-500">Grade your readiness</Link>
          </div>
        </div>
      </section>
    </main>
  );
}

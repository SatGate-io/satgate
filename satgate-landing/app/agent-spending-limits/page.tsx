import Link from 'next/link';
import { ArrowRight, Ban, BarChart3, Bot, DollarSign, Gauge, KeyRound, ReceiptText, ShieldCheck } from 'lucide-react';

export const metadata = {
  title: 'Agent Spending Limits | Spend Caps for Autonomous AI Agents',
  description: 'Set AI agent spending limits by task, route, tool, model, tenant, workflow, session, and day. Block over-budget requests before spend occurs.',
  alternates: { canonical: 'https://satgate.io/agent-spending-limits' },
  keywords: [
    'agent spending limits',
    'AI agent spending limits',
    'AI agent spend caps',
    'autonomous agent budget limits',
    'AI agent hard caps',
    'agent cost governance',
    'economic firewall',
    'SatGate',
    'AI agent cost control',
  ],
  openGraph: {
    title: 'Agent Spending Limits | Spend Caps for Autonomous AI Agents',
    description: 'Set AI agent spending limits by task, route, tool, model, tenant, workflow, session, and day before spend occurs.',
    url: 'https://satgate.io/agent-spending-limits',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Agent Spending Limits | Spend Caps for Autonomous AI Agents',
    description: 'Set AI agent spending limits by task, route, tool, model, tenant, workflow, session, and day before spend occurs.',
  },
};

const controls = [
  { icon: Bot, title: 'Agent identity', body: 'Attribute requests to the tenant, agent, task, workflow, route, model, MCP tool, and delegated sub-agent.' },
  { icon: DollarSign, title: 'Budget checks', body: 'Evaluate remaining spend, per-request ceilings, daily caps, tool caps, and route budgets before forwarding.' },
  { icon: KeyRound, title: 'Scoped credentials', body: 'Use expiring capabilities instead of broad static keys so authority matches the job.' },
  { icon: Ban, title: 'Revocation', body: 'Block the next request when a credential, workflow, route, budget, or agent should stop.' },
  { icon: ReceiptText, title: 'Audit trails', body: 'Record allow/deny decisions with policy, budget remaining, route, tool, estimated cost, and outcome.' },
  { icon: BarChart3, title: 'Benchmark risk', body: 'Model loops, retry storms, fanout, detection delay, and avoided spend with benchmark-backed scenarios.' },
];

export default function Page() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'Agent Spending Limits | Spend Caps for Autonomous AI Agents',
    url: 'https://satgate.io/agent-spending-limits',
    description: metadata.description,
    datePublished: '2026-05-01',
    dateModified: '2026-05-05',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI agent spending limits' },
      { '@type': 'Thing', name: 'autonomous agent budget caps' },
      { '@type': 'Thing', name: 'delegated sub-agent limits' },
      { '@type': 'Thing', name: 'economic firewall controls' },
      { '@type': 'Thing', name: 'request-path spend enforcement' },
    ],
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'Agent Spending Limits',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Cloud, API Gateway, MCP Proxy',
    description: metadata.description,
    url: 'https://satgate.io/agent-spending-limits',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-05',
    featureList: ['Request-path budget enforcement', 'AI agent spend caps', 'MCP tool cost control', 'Revocable credentials', 'Audit trails'],
    audience: { '@type': 'Audience', audienceType: 'AI platform, API, finance, and security teams' },
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'What are agent spending limits?', acceptedAnswer: { '@type': 'Answer', text: 'Agent spending limits are request-path budgets and caps that constrain what autonomous AI agents can spend by task, route, tool, model, workflow, tenant, session, or day before requests execute.' } },
      { '@type': 'Question', name: 'Why are dashboards not enough?', acceptedAnswer: { '@type': 'Answer', text: 'Dashboards and billing alerts report spend after requests complete. Autonomous agents can loop, retry, and delegate fast enough that budget policy must be enforced before upstream access.' } },
      { '@type': 'Question', name: 'How does SatGate help?', acceptedAnswer: { '@type': 'Answer', text: 'SatGate sits in the request path and checks identity, budget, route, tool scope, credential caveats, expiry, revocation, and audit policy before forwarding the request.' } },
      { '@type': 'Question', name: 'What spending limits should AI agents have?', acceptedAnswer: { '@type': 'Answer', text: 'AI agents should have spending limits by tenant, agent, task, workflow, session, model, tool, route, delegated sub-agent, and time window, with per-request ceilings and emergency revocation.' } },
      { '@type': 'Question', name: 'Are spending limits better than rate limits for AI agents?', acceptedAnswer: { '@type': 'Answer', text: 'They solve different problems. Rate limits control frequency, while spending limits control economic exposure by checking request price, remaining budget, scope, and policy before cost is created.' } },
    ],
  };

  const controlsJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'Agent spending limit controls',
    description: 'Request-path controls for setting and enforcing autonomous AI agent spend caps before model, API, and MCP tool requests execute.',
    itemListElement: controls.map(({ title, body }, index) => ({
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
      { '@type': 'ListItem', position: 2, name: 'AI Agent Cost Control', item: 'https://satgate.io/ai-agent-cost-control' },
      { '@type': 'ListItem', position: 3, name: 'Agent Spending Limits', item: 'https://satgate.io/agent-spending-limits' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(controlsJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_24%_0%,rgba(34,211,238,0.18),transparent_34%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200"><Gauge size={16} /> Spend caps for autonomous workers</div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">Agent spending limits should stop the next request, not explain the last bill</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">Autonomous agents need hard spending limits that apply per task, workflow, delegated sub-agent, model, tool, API route, and time window. SatGate enforces those limits before requests execute.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/runaway-agent-cost-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Model runaway spend <ArrowRight size={18} /></Link>
            <Link href="/ai-agent-runaway-spend-benchmark" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">See benchmark data</Link>
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">Grade readiness</Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1fr_0.9fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">The control point is before the call</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>Autonomous agents can generate real costs through model calls, API requests, MCP tools, delegated sub-agents, retries, and background workflows. If the policy check happens after the request, the money is already spent.</p>
            <p>SatGate enforces economic policy at the gateway boundary. Every important request can be evaluated against budget, scope, identity, revocation, route, tool, and audit rules before upstream access.</p>
            <p>That is the difference between cost reporting and economic control.</p>
          </div>
        </div>
        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">What good policy includes</h3>
          <div className="space-y-4">
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Per-agent budgets</h3><p className="leading-relaxed text-gray-400">Track and limit spend by agent identity, tenant, task, workflow, route, model, and MCP tool.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Delegated sub-agent limits</h3><p className="leading-relaxed text-gray-400">Give sub-agents smaller budgets, narrower tools, shorter expiry, and separate audit trails than their parent.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Kill switches and revocation</h3><p className="leading-relaxed text-gray-400">Stop future spend by revoking or narrowing credentials before the next API or MCP request.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">From Observe to Control</h3><p className="leading-relaxed text-gray-400">Start by measuring real agent spend, then enforce hard caps where risk, cost, or autonomy justifies it.</p></div>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">SatGate controls</h2>
          <p className="mb-10 max-w-3xl text-lg text-gray-400">Use SatGate to move from Observe to Control: measure real agent economics first, then enforce the limits that matter.</p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {controls.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 transition hover:border-cyan-900/70">
                <Icon className="mb-4 text-cyan-300" size={28} />
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Agent spending limit questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What are agent spending limits?</h3>
              <p className="text-gray-400 leading-relaxed">
                Agent spending limits are request-path budgets and caps that constrain what autonomous AI agents can spend by task, route, tool, model, workflow, tenant, session, or day before requests execute.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why are dashboards not enough?</h3>
              <p className="text-gray-400 leading-relaxed">
                Dashboards and billing alerts report spend after requests complete. Autonomous agents can loop, retry, and delegate fast enough that budget policy must be enforced before upstream access.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does SatGate help?</h3>
              <p className="text-gray-400 leading-relaxed">
                SatGate sits in the request path and checks identity, budget, route, tool scope, credential caveats, expiry, revocation, and audit policy before forwarding the request.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What spending limits should AI agents have?</h3>
              <p className="text-gray-400 leading-relaxed">
                AI agents should have spending limits by tenant, agent, task, workflow, session, model, tool, route, delegated sub-agent, and time window, with per-request ceilings and emergency revocation.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Are spending limits better than rate limits for AI agents?</h3>
              <p className="text-gray-400 leading-relaxed">
                They solve different problems. Rate limits control frequency, while spending limits control economic exposure by checking request price, remaining budget, scope, and policy before cost is created.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-purple-900/60 bg-gradient-to-br from-purple-950/35 to-cyan-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Make agent economics enforceable.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">SatGate is the economic firewall for AI agents: observe every request, control spend before execution, and charge robot customers when paid API access should unlock.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/tools" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Open free tools <ArrowRight size={18} /></Link>
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">Economic firewall</Link>
            <Link href="/agent-spend-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">Policy template</Link>
          </div>
        </div>
      </section>
    </main>
  );
}

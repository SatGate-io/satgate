import Link from 'next/link';
import { ArrowRight, CheckCircle2, Shield, Gauge, WalletCards, Activity, KeyRound } from 'lucide-react';

export const metadata = {
  title: 'Economic Firewall for AI Agents',
  description: 'An economic firewall checks every AI agent request before it reaches your API. Your own agents stay within budget, and external agents pay on the routes you choose.',
  alternates: { canonical: 'https://satgate.io/economic-firewall' },
  keywords: [
    'economic firewall',
    'economic firewall for AI agents',
    'AI agent spend control',
    'AI agent budget enforcement',
    'economic firewall for AI agents',
    'request-layer cost control',
    'API budget enforcement',
    'agent API governance',
  ],
  openGraph: {
    title: 'Economic Firewall for AI Agents',
    description: 'A gateway that checks what each AI agent may do and spend before its request reaches your API, and signs a receipt for every decision.',
    url: 'https://satgate.io/economic-firewall',
    type: 'article',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Economic Firewall for AI Agents',
    description: 'A gateway that checks what each AI agent may do and spend before its request reaches your API, and signs a receipt for every decision.',
  },
};

const capabilities = [
  {
    icon: KeyRound,
    title: 'Agent identity',
    body: 'Know which agent, sub-agent, token, route and tool is behind every call.',
  },
  {
    icon: Shield,
    title: 'Access control',
    body: 'Allow or block each request, and honor expiry and revokes, before it reaches your API.',
  },
  {
    icon: Gauge,
    title: 'Budgets and limits',
    body: 'Set budgets per agent, tool, model, session or day. The limits travel inside the agent’s token.',
  },
  {
    icon: Activity,
    title: 'Receipts',
    body: 'Sign a receipt for each decision: who allowed it, why a request was refused, what it cost. Export them as an Evidence Pack.',
  },
  {
    icon: WalletCards,
    title: 'Paid access for external agents',
    body: 'On the routes you choose, external agents pay before their request goes through. Paying never gets them past your access rules.',
  },
];

export default function EconomicFirewallPage() {
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Economic Firewall for AI Agents',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-25',
    dateModified: '2026-09-26',
    mainEntityOfPage: 'https://satgate.io/economic-firewall',
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is an economic firewall?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'A gateway that checks each AI agent request before it reaches your API. Your own agents spend from budgets you set. External agents must follow your access rules and, on paid routes, pay first.',
        },
      },
      {
        '@type': 'Question',
        name: 'How is an economic firewall different from rate limiting?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Rate limiting counts requests. An economic firewall also knows which agent is calling, what it may do, how much budget it has left and whether it has paid.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do AI agents need economic firewalls?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Agents loop, retry, hand off work and call paid tools with no person approving each request. SatGate blocks what isn’t allowed before it runs and signs a receipt for each decision.',
        },
      },
      {
        '@type': 'Question',
        name: 'Is an economic firewall the same as an API gateway?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. An API gateway routes and secures traffic. An economic firewall adds per-agent permissions and budgets, tokens that can be narrowed for sub-agents, revokes, payment, and a receipt for each decision.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do I know whether I need an economic firewall?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'When your agents can call paid models, APIs or MCP tools faster than a person can review what they do and spend. Start by listing which agents call what, then grade your setup with the readiness grader.',
        },
      },
      {
        '@type': 'Question',
        name: 'What is the first economic firewall control to implement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Start with Observe: see which agent, route and tool is behind each request, without blocking anything. Then turn on Control for the risky routes, with budgets and revokes. Add Admit when you want external agents to pay.',
        },
      },
    ],
  };

  const definedTermJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'DefinedTerm',
    name: 'Economic firewall',
    description: 'A gateway that checks what each AI agent may do and spend, and whether it has paid, before forwarding its request to your API.',
    inDefinedTermSet: 'https://satgate.io/economic-firewall',
    url: 'https://satgate.io/economic-firewall',
  };

  const implementationPathJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'Economic firewall implementation path',
    description: 'Roll out in three steps: see agent traffic, enforce limits, then charge external agents.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'See what your agents do',
        description: 'List which agents, sub-agents, routes, models and MCP tools are in use, and what they cost, before you change anything.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Set limits',
        description: 'Turn on Control for the risky routes: budget caps, narrower tokens, expiry and revokes.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Charge external agents',
        description: 'Decide what external agents can reach and which routes they pay for. Keep the receipts for allowed and refused requests.',
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Economic Firewall', item: 'https://satgate.io/economic-firewall' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(definedTermJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(implementationPathJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(34,211,238,0.18),transparent_32%),radial-gradient(circle_at_80%_20%,rgba(168,85,247,0.16),transparent_28%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200 mb-8">
            <Shield size={16} /> What it is
          </div>

          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-4xl mb-8">
            Economic Firewall for AI Agents
          </h1>

          <p className="text-xl md:text-2xl text-gray-300 max-w-3xl leading-relaxed mb-6">
            Keep your agents within budget. Make external agents pay before they use your API. Get a signed receipt for every decision.
          </p>
          <p className="max-w-3xl rounded-2xl border border-purple-900/50 bg-purple-950/20 p-5 text-lg leading-relaxed text-purple-100 mb-10">
            SatGate has three controls. Observe shows what your agents do and spend. Control enforces their budgets. Admit decides what external agents can reach and charges them where you choose. All three sign receipts.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              See how receipts work <ArrowRight size={18} />
            </Link>
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Control your agents
            </Link>
            <Link href="/mcp" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Control MCP tools
            </Link>
          </div>
        </div>
      </section>

      <section className="border-b border-gray-900 bg-gray-950/40">
        <div className="max-w-6xl mx-auto px-6 py-12">
          <p className="text-sm font-mono uppercase tracking-wide text-cyan-300 mb-3">Definition</p>
          <div className="rounded-2xl border border-cyan-900/50 bg-black/60 p-6 md:p-8">
            <p className="text-2xl md:text-3xl font-bold leading-snug text-white">
              An economic firewall sits in front of your API and decides, for each AI agent request, whether it can go through: is the agent allowed, does it have budget left, and has it paid if the route charges?
            </p>
            <p className="mt-5 text-gray-400 text-lg leading-relaxed">
              It works like an API gateway, plus the parts agent traffic needs and a normal gateway lacks: knowing which agent is calling, per-agent permissions and budgets, revokes, payment, and a signed receipt for each decision.
            </p>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-16">
        <h2 className="text-3xl font-bold text-white mb-6">Where the idea came from</h2>
        <div className="space-y-5 max-w-4xl text-lg leading-relaxed text-gray-300">
          <p>It started with one rule: an external agent pays before it uses your API. That makes hammering your API expensive. It raises the cost of an attack; it does not make one impossible.</p>
          <p>The same idea works for your own agents. Each one gets a budget and a list of what it may do, and SatGate checks both before a request runs. Observe shows usage without blocking; Control enforces the limits. Your own agents don&apos;t pay anyone. They spend from the budget you set.</p>
          <p>Make SatGate the only way to reach your API, and keep your API keys behind it, so agents can&apos;t go around it.</p>
          <p><strong className="text-white">Receipts from all three.</strong> SatGate signs a receipt for each decision, on HTTP APIs and MCP tools, and anyone can check it. A receipt shows what SatGate decided. It is not a compliance certificate.</p>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1.1fr_0.9fr] gap-12 items-start">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">The problem: agents act faster than anyone can review</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              Most API security assumes a person or a predictable app is behind each request. Agents plan, retry, hand work to other agents, call tools and loop. Every step can cost money, move data or widen what the agent can reach.
            </p>
            <p>
              Rate limits can slow traffic. Dashboards can explain yesterday&apos;s bill. Neither can answer the question that matters before a request happens: <strong className="text-white">is this agent allowed to take this action right now?</strong>
            </p>
            <p>
              An external agent can run up your costs on purpose. Your own agent can burn through a budget by accident, through retries or handoffs. Both need limits that are checked before each request, and the limits have to hold whether or not the agent wants them.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">What it checks on every request</h3>
          <div className="space-y-3 text-sm">
            {[
              'Who is the agent?',
              'Which token is it using, and what does that token allow?',
              'Is this action allowed?',
              'Is there budget left, and is the token still valid and not revoked?',
              'Does this route charge, and has the agent paid?',
              'Allow or refuse, and sign a receipt either way.',
            ].map((item) => (
              <div key={item} className="flex items-start gap-3 rounded-lg border border-gray-800 bg-black/50 p-3">
                <CheckCircle2 className="text-cyan-300 mt-0.5" size={18} />
                <span className="text-gray-300">{item}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">What it controls</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            On every request, SatGate identifies the agent, checks your rules, allows or refuses, and signs a receipt.
          </p>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {capabilities.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 hover:border-cyan-900/70 transition">
                <Icon className="text-cyan-300 mb-4" size={28} />
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-8 items-start">
          <div>
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Payment and permission</p>
            <h2 className="text-3xl font-bold text-white mb-5">Paying never gets an agent past your rules</h2>
            <div className="space-y-4 text-gray-300 text-lg leading-relaxed">
              <p>
                An external agent might pay and still ask for something you have ruled out. SatGate checks both: the agent has to pay where you charge, and it has to stay within what you allow.
              </p>
              <p>
                SatGate makes that call before it forwards the request, so a refused request never reaches your API.
              </p>
              <p className="font-semibold text-white">
                Payment never widens what an agent can do. A receipt records what SatGate decided, not whether your API&apos;s answer was any good.
              </p>
            </div>
          </div>
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
            <h3 className="text-xl font-bold text-white mb-4">Guides on agent payments</h3>
            <div className="space-y-3">
              {[
                ['/stripe-link-agents-vs-satgate', 'Stripe Link for Agents vs SatGate'],
                ['/agent-payment-controls', 'Agent payment controls'],
                ['/http-402-for-ai-agents', 'HTTP 402 for AI agents'],
              ].map(([href, title]) => (
                <Link key={href} href={href} className="flex items-center justify-between rounded-lg border border-gray-800 bg-black/50 p-4 text-white transition hover:border-cyan-500/50">
                  <span>{title}</span><ArrowRight size={16} />
                </Link>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="grid lg:grid-cols-3 gap-6">
          <div className="rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Observe</h2>
            <p className="text-gray-300 leading-relaxed">
              Start by watching agent traffic without blocking anything. See usage and cost by agent, model, route, tool, team and workflow, so security, finance and platform teams all see the same numbers.
            </p>
          </div>
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Control</h2>
            <p className="text-gray-300 leading-relaxed">
              Then enforce limits on the risky routes: permissions, budgets, expiry and revokes, all checked before your API is called. When SatGate blocks a request, the receipt says why.
            </p>
          </div>
          <div className="rounded-2xl border border-yellow-800/50 bg-yellow-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Admit</h2>
            <p className="text-gray-300 leading-relaxed">
              Admit lets external agents in on your terms. On paid routes they pay before the request goes through, and your access rules and revokes still apply to agents that paid.
            </p>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-12">
        <details className="rounded-xl border border-gray-800 p-6">
          <summary className="cursor-pointer text-xl font-bold">Developer details: payments and internal budgets</summary>
          <p className="mt-4 text-gray-400">Paid routes use L402, which pairs a Lightning invoice with an access token. Fiat402 is internal budget control, not a payment rail: when your own agent spends, SatGate subtracts from its budget and no money moves.</p>
        </details>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">Economic firewall vs. the usual controls</h2>
          <div className="overflow-hidden rounded-2xl border border-gray-800">
            <div className="grid md:grid-cols-3 bg-gray-900/70 text-sm font-bold text-white">
              <div className="p-4">Control</div>
              <div className="p-4">What it answers</div>
              <div className="p-4">Where it fails for agents</div>
            </div>
            {[
              ['Rate limiting', 'How many requests?', 'Knows nothing about money, model cost, tool prices or budgets.'],
              ['Provider billing dashboard', 'What did we spend?', 'Tells you after the fact, and usually not per agent.'],
              ['Static API keys', 'Who has access?', 'No budgets, no expiry, no narrower keys for sub-agents, no per-request pricing.'],
              ['Economic firewall', 'Should this agent be allowed to do this, right now, at this price?', 'Checks permissions, budget and payment on every request.'],
            ].map(([a, b, c]) => (
              <div key={a} className="grid md:grid-cols-3 border-t border-gray-800 text-gray-300">
                <div className="p-4 font-semibold text-white">{a}</div>
                <div className="p-4">{b}</div>
                <div className="p-4">{c}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Implementation path</p>
        <h2 className="mb-4 text-3xl font-bold text-white">How to roll out an economic firewall</h2>
        <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
          Pick one workload and put SatGate in front of it. Check that agents still get their work done, that refusals happen when they should, and that nothing reaches your API without going through SatGate. Observe alone doesn&apos;t block anything.
        </p>
        <div className="grid gap-5 md:grid-cols-3">
          {[
            ['1', 'See what your agents do', 'List which agents, sub-agents, routes, models and MCP tools are in use, and what they cost, before you change anything.', '/agent-control-plane'],
            ['2', 'Set limits', 'Turn on Control for the risky routes: budget caps, narrower tokens, expiry and revokes.', '/build'],
            ['3', 'Charge external agents', 'Decide what external agents can reach and which routes they pay for. Keep the receipts for allowed and refused requests.', '/policy-to-proof'],
          ].map(([step, title, body, href]) => (
            <Link key={step} href={href} className="rounded-2xl border border-gray-800 bg-gray-950 p-6 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
              <div className="mb-5 flex h-10 w-10 items-center justify-center rounded-full bg-cyan-500/15 font-mono text-cyan-200">{step}</div>
              <h3 className="mb-3 text-xl font-bold text-white">{title}</h3>
              <p className="mb-4 leading-relaxed text-gray-400">{body}</p>
              <span className="inline-flex items-center gap-2 text-sm font-semibold text-cyan-300">Read this step <ArrowRight size={16} /></span>
            </Link>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Free tools</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Check your setup</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            Free tools to put numbers on the risk and write your first policies.
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {[
              ['/economic-firewall-readiness-grader', 'Readiness grader', 'Grade your setup on identity, budgets, revokes, audit records, MCP tools and payments.'],
              ['/roi-calculator', 'ROI calculator', 'Estimate runaway agent spend, wasted cost and payback.'],
              ['/build', 'Build with SatGate', 'Mint an agent token with a budget, expiry and revoke.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-black p-5 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">Related topics</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[
              ['/policy-to-proof', 'Policy-to-Proof', 'How each token, handoff, payment, refusal and revoke becomes a signed receipt.'],
              ['/mcp', 'MCP', 'Budgets and permissions for MCP tools, and a signed receipt when a call is allowed or refused.'],
              ['/govern', 'Control AI agents', 'Budgets and permissions for your own agents, with receipts you can export.'],
              ['/agent-control-plane', 'Agent control plane', 'Permissions, handoffs, spend, audit and revokes for large fleets of agents.'],
              ['/mcp', 'MCP governance', 'Budgets, revokes and receipts for MCP tool calls.'],
              ['/agent-api-governance', 'Agent API governance', 'Identity, handoffs, revokes and audit for agent API calls.'],
              ['/ai-agent-cost-control', 'AI agent cost control', 'Stop runaway agent spend with hard budgets.'],
              ['/ai-api-budget-enforcement', 'AI API budget enforcement', 'Check the budget before a model, tool or API call goes out.'],
              ['/agent-spending-limits', 'Agent spending limits', 'Spending caps by task, workflow, sub-agent, route, model and tool.'],
              ['/mcp', 'MCP cost control', 'Control paid tool calls, retries, SaaS actions, cloud tasks, and data lookups.'],
              ['/agent-payment-controls', 'Agent payment controls', 'Rules for agent wallets, budgets and 402 payment requests.'],
              ['/http-402-for-ai-agents', 'HTTP 402 for AI agents', 'Understand payment challenges, shared payment tokens, and L402.'],
              ['/l402-agent-payments', 'L402 agent payments', 'How L402 ties a Lightning payment to API access.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 transition hover:border-cyan-500/50 hover:bg-cyan-950/20">
                <h3 className="font-bold text-white mb-2">{title}</h3>
                <p className="text-sm text-gray-400 leading-relaxed">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Economic firewall questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            {[
              ['What is an economic firewall?', 'A gateway that checks each AI agent request before it reaches your API. Your own agents spend from budgets you set. External agents must follow your access rules and, on paid routes, pay first.'],
              ['How is an economic firewall different from rate limiting?', 'Rate limiting counts requests. An economic firewall also knows which agent is calling, what it may do, how much budget it has left and whether it has paid.'],
              ['Why do AI agents need economic firewalls?', 'Agents loop, retry, hand off work and call paid tools with no person approving each request. SatGate blocks what isn’t allowed before it runs and signs a receipt for each decision.'],
              ['Is an economic firewall the same as an API gateway?', 'No. An API gateway routes and secures traffic. An economic firewall adds per-agent permissions and budgets, tokens that can be narrowed for sub-agents, revokes, payment, and a receipt for each decision.'],
              ['How do I know whether I need an economic firewall?', 'When your agents can call paid models, APIs or MCP tools faster than a person can review what they do and spend. Start by listing which agents call what, then grade your setup with the readiness grader.'],
              ['What is the first economic firewall control to implement?', 'Start with Observe: see which agent, route and tool is behind each request, without blocking anything. Then turn on Control for the risky routes, with budgets and revokes. Add Admit when you want external agents to pay.'],
            ].map(([question, answer]) => (
              <div key={question} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
                <h3 className="mb-2 text-xl font-bold text-white">{question}</h3>
                <p className="leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">SatGate checks every agent request before it runs</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            Put SatGate in front of your API or MCP tools. See every agent call, limit what agents can reach and spend, charge external agents where you choose, and keep a signed receipt of each decision.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              See how it works <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              How receipts work
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

import Link from 'next/link';
import { ArrowLeft, ArrowRight, Bot, Gauge, ShieldCheck, Terminal } from 'lucide-react';

export const metadata = {
  title: 'MCP Gateway for AI Agents: Budgets, Permissions, Receipts',
  description: 'Put SatGate in front of MCP tools: per-tool budgets, scoped permissions, revocation, and a signed receipt for every allowed or refused call.',
  alternates: { canonical: 'https://satgate.io/mcp' },
  keywords: [
    'MCP governance',
    'MCP budget enforcement',
    'MCP cost control',
    'MCP tool spend control',
    'economic firewall for MCP',
    'AI agent MCP security',
    'Cursor MCP budget control',
    'Claude Desktop MCP governance',
  ],
  openGraph: {
    title: 'MCP Gateway for AI Agents: Budgets, Permissions, Receipts',
    description: 'Budgets and permissions for MCP tools, and a signed receipt when a call is allowed or refused.',
    url: 'https://satgate.io/mcp',
    type: 'website',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Gateway for AI Agents: Budgets, Permissions, Receipts',
    description: 'Budgets and permissions for MCP tools, and a signed receipt when a call is allowed or refused.',
  },
};

const cards = [
  {
    href: '/mcp-proxy-config-generator',
    title: 'MCP connect snippet',
    description: 'The npx satgate-mcp-bridge snippet for Cursor and Claude Code. Credits, not a dollar budget.',
    icon: Terminal,
  },
  {
    href: '/blog/mcp-budget-enforcement-guide',
    title: 'MCP budget enforcement guide',
    description: 'How per-tool costs, session caps and delegated budgets stop a tool call before it runs.',
    icon: Gauge,
  },
  {
    href: '/verify-evidence-pack',
    title: 'Verify an Evidence Pack',
    description: 'Check signed MCP receipts against the published keys with the open-source verifier.',
    icon: ShieldCheck,
  },
];

export default function MCPPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'MCP Gateway for AI Agents: Budgets, Permissions, Receipts',
    url: 'https://satgate.io/mcp',
    description: metadata.description,
    datePublished: '2026-05-01',
    dateModified: '2026-09-30',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'MCP governance' },
      { '@type': 'Thing', name: 'MCP budget enforcement' },
      { '@type': 'Thing', name: 'MCP tool spend control' },
      { '@type': 'Thing', name: 'economic firewall for MCP' },
      { '@type': 'Thing', name: 'scoped capabilities for AI agents' },
    ],
  };

  const itemListJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'MCP governance resources',
    description: metadata.description,
    itemListElement: cards.map((card, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: card.title,
      url: `https://satgate.io${card.href}`,
      description: card.description,
    })),
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'MCP', item: 'https://satgate.io/mcp' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is MCP governance?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'MCP governance is the control layer around Model Context Protocol tool calls: budgets, scoped authority, revocation, Evidence Packs, and risk actions before agents execute tools.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do MCP tools need budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Autonomous agents can call paid or risky tools repeatedly, delegate work, or loop. MCP budget enforcement stops over-budget tool calls in the request path instead of discovering spend after the fact.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate control MCP spend?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate can proxy MCP traffic and enforce per-tool prices, session caps, workflow budgets, capability caveats, revocation, and audit requirements before tool calls reach the upstream MCP server.',
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
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(168,85,247,0.2),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.16),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-purple-500/30 bg-purple-950/30 px-4 py-2 text-sm text-purple-200">
            <Bot size={16} /> Economic firewall for MCP tools
          </div>
          <h1 className="mb-6 max-w-5xl text-5xl font-extrabold tracking-tight md:text-6xl">
            Connect MCP in four steps
          </h1>
          <ol className="mb-8 max-w-3xl list-decimal space-y-3 pl-5 text-lg leading-relaxed text-gray-300">
            <li>Start a 14-day trial. No card.</li>
            <li>Open MCP Setup in the dashboard and copy the connect snippet. Cursor uses this:</li>
          </ol>
          <pre className="mb-6 max-w-3xl overflow-x-auto rounded-xl border border-gray-800 bg-black p-4 text-sm leading-6 text-gray-300">{`{
  "mcpServers": {
    "satgate": {
      "command": "npx",
      "args": ["-y", "satgate-mcp-bridge"],
      "env": {
        "SATGATE_URL": "https://satgate-mcp-saas.fly.dev",
        "SATGATE_TOKEN": "paste-the-token-from-cloud.satgate.io"
      }
    }
  }
}`}</pre>
          <ol className="mb-8 max-w-3xl list-decimal space-y-3 pl-5 text-lg leading-relaxed text-gray-300" start={3}>
            <li>A new token starts at 1,000 credits. A tool call costs 1 credit unless you set per-tool costs.</li>
            <li>Watch calls in the MCP Monitor.</li>
          </ol>
          <div className="flex flex-col gap-4 sm:flex-row">
            <a href="https://cloud.satgate.io/cloud/signup" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Start free trial <ArrowRight size={18} />
            </a>
            <a href="https://cloud.satgate.io/cloud/mcp/onboarding" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Open MCP Setup
            </a>
          </div>
          <p className="mt-4 text-sm text-gray-500">Claude Code uses the same npx snippet. Claude Desktop and OpenClaw use the tenant /sse URL with an Authorization Bearer header, copied from MCP Setup.</p>
        </div>
      </section>

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(168,85,247,0.2),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.16),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-purple-500/30 bg-purple-950/30 px-4 py-2 text-sm text-purple-200">
            <Bot size={16} /> Economic firewall for MCP tools
          </div>
          <h2 className="mb-8 max-w-5xl text-4xl font-extrabold tracking-tight md:text-5xl">
            MCP for your agents, with a budget
          </h2>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300">
            SatGate sits in front of MCP tools. You see what each agent calls, you cap it, and you get a signed receipt when a call is allowed or refused.
          </p>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="grid gap-5 md:grid-cols-2">
          {cards.map(({ href, title, description, icon: Icon }) => (
            <Link key={href} href={href} className="group rounded-2xl border border-gray-800 bg-gray-950 p-6 transition hover:border-purple-500/50 hover:bg-purple-950/10">
              <Icon className="mb-5 text-purple-300 transition group-hover:text-purple-200" size={32} />
              <h2 className="mb-3 text-2xl font-bold text-white">{title}</h2>
              <p className="mb-5 leading-relaxed text-gray-400">{description}</p>
              <span className="inline-flex items-center gap-2 font-semibold text-purple-300">Open resource <ArrowRight size={16} /></span>
            </Link>
          ))}
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 pb-4">
        <h2 className="mb-6 text-3xl font-bold text-white">Three controls, one gateway</h2>
        <ul className="grid gap-5 md:grid-cols-3">
          <li className="rounded-2xl border border-gray-800 bg-black p-6 text-gray-400"><span className="font-bold text-white">Observe.</span> See which agent called which tool, and what each call would cost, without blocking anything.</li>
          <li className="rounded-2xl border border-gray-800 bg-black p-6 text-gray-400"><span className="font-bold text-white">Control.</span> Give each agent token a credit budget and a tool allowlist. A call past either limit is refused before it reaches the MCP server.</li>
          <li className="rounded-2xl border border-gray-800 bg-black p-6 text-gray-400"><span className="font-bold text-white">Admit.</span> Charge external agents on the HTTP routes you choose. You set how many requests a Lightning payment buys; a USDC payment buys one. See <Link href="/pricing" className="text-purple-300 hover:text-purple-200">pricing</Link>.</li>
        </ul>
        <p className="mt-6 max-w-4xl text-gray-400">
          Each allowed or refused call gets a signed receipt, so you can prove what an agent was allowed to do. Export receipts as an Evidence Pack and <Link href="/verify-evidence-pack" className="text-purple-300 hover:text-purple-200">verify them yourself</Link>. For the budget model in detail, read the <Link href="/blog/mcp-budget-enforcement-guide" className="text-purple-300 hover:text-purple-200">MCP budget enforcement guide</Link>, or start from the <Link href="/build" className="text-purple-300 hover:text-purple-200">developer page</Link>.
        </p>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto grid max-w-6xl gap-8 px-6 py-20 lg:grid-cols-3">
          {[
            ['Price every tool', 'Assign cost profiles to MCP tools so agents cannot treat expensive operations like free function calls.'],
            ['Enforce before execution', 'Block over-budget or out-of-scope tool calls before they reach the upstream MCP server.'],
            ['Audit every decision', 'Emit receipt-backed proof for agent, workflow, tool, policy, budget, paid-call, denial, delegation, and revocation decisions.'],
          ].map(([title, body]) => (
            <div key={title} className="rounded-2xl border border-gray-800 bg-black p-6">
              <h2 className="mb-3 text-2xl font-bold text-white">{title}</h2>
              <p className="leading-relaxed text-gray-400">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto max-w-4xl px-6 py-20">
        <h2 className="mb-8 text-3xl font-bold text-white">MCP governance FAQ</h2>
        <div className="space-y-5">
          {[
            ['What is MCP governance?', 'MCP governance is the control layer around Model Context Protocol tool calls: budgets, scoped authority, revocation, Evidence Packs, and risk actions before agents execute tools.'],
            ['Why do MCP tools need budget enforcement?', 'Autonomous agents can call paid or risky tools repeatedly, delegate work, or loop. MCP budget enforcement stops over-budget tool calls in the request path instead of discovering spend after the fact.'],
            ['How does SatGate control MCP spend?', 'SatGate can proxy MCP traffic and enforce per-tool prices, session caps, workflow budgets, capability caveats, revocation, and audit requirements before tool calls reach the upstream MCP server.'],
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

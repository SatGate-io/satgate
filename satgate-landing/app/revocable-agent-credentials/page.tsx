import Link from 'next/link';
import { ArrowRight, Ban, Clock, Fingerprint, GitBranch, KeyRound, ReceiptText, ShieldCheck } from 'lucide-react';

export const metadata = {
  title: 'Revocable Agent Credentials | Kill Switches for AI Agents',
  description: 'Replace static API keys with revocable, scoped, expiring credentials for AI agents. Stop runaway access before the next API or MCP tool call.',
  alternates: { canonical: 'https://satgate.io/revocable-agent-credentials' },
  keywords: [
    'revocable agent credentials',
    'AI agent credentials',
    'agent API key rotation',
    'AI agent kill switch',
    'revocable API keys',
    'scoped agent credentials',
    'MCP revocation',
    'economic control plane for AI agents',
  ],
  openGraph: {
    title: 'Revocable Agent Credentials | Kill Switches for AI Agents',
    description: 'Replace static API keys with revocable, scoped, expiring credentials for AI agents. Stop runaway access before the next API or MCP tool call.',
    url: 'https://satgate.io/revocable-agent-credentials',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Revocable Agent Credentials | Kill Switches for AI Agents',
    description: 'Replace static API keys with revocable, scoped, expiring credentials for AI agents. Stop runaway access before the next API or MCP tool call.',
  },
};

const controls = [
  { icon: Fingerprint, title: 'Agent identity', body: 'Bind each credential to an agent, task, tenant, workflow, route, model, or MCP server.' },
  { icon: KeyRound, title: 'Scoped authority', body: 'Limit routes, tools, methods, customers, delegation, and request types instead of issuing broad API keys.' },
  { icon: Clock, title: 'Expiry by default', body: 'Make credentials expire with the work: minutes, sessions, jobs, customers, or delegated sub-tasks.' },
  { icon: Ban, title: 'Revocation checks', body: 'Block the next request when a token, task, agent, route, or budget is no longer allowed.' },
  { icon: ReceiptText, title: 'Budget caveats', body: 'Attach spend caps, call ceilings, per-tool limits, and remaining-budget checks to the request path.' },
  { icon: GitBranch, title: 'Attenuated delegation', body: 'Let agents delegate narrower credentials to sub-agents without expanding parent authority.' },
];

export default function Page() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Revocable Agent Credentials | Kill Switches for AI Agents',
    description: metadata.description,
    url: 'https://satgate.io/revocable-agent-credentials',
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-08-07',
    about: ['AI agent security', 'Agent API governance', 'Capability-based security', 'Economic firewall', 'economic control plane for AI agents'],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'What is a revocable agent credential?', acceptedAnswer: { '@type': 'Answer', text: 'A revocable agent credential is a scoped, expiring capability issued to an autonomous agent for a specific task, workflow, route, tool, budget, or time window. It can be invalidated before the next request without rotating global API keys.' } },
      { '@type': 'Question', name: 'Why are static API keys risky for AI agents?', acceptedAnswer: { '@type': 'Answer', text: 'Static API keys are broad, long-lived, and hard to delegate safely. Autonomous agents need credentials with scoped authority, budget limits, expiry, revocation, and audit context.' } },
      { '@type': 'Question', name: 'How does SatGate enforce agent credentials?', acceptedAnswer: { '@type': 'Answer', text: 'SatGate sits in the request path and checks identity, token scope, route, tool, budget, expiry, delegation rules, and revocation state before forwarding upstream.' } },
      { '@type': 'Question', name: 'Where do revocable credentials fit in an economic control plane?', acceptedAnswer: { '@type': 'Answer', text: 'Revocable credentials are the kill-switch layer inside an economic control plane for AI agents. They let the gateway stop the next API, model, or MCP request when authority, budget, scope, or risk changes.' } },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Revocable Agent Credentials', item: 'https://satgate.io/revocable-agent-credentials' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_34%),radial-gradient(circle_at_78%_12%,rgba(245,158,11,0.16),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200"><ShieldCheck size={16} /> Kill switches for autonomous access</div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">Revocable agent credentials are the missing kill switch for AI autonomy</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">AI agents should not hold broad, long-lived API keys. They need task-scoped credentials that can be narrowed, expired, or revoked before the next API, model, or MCP tool call executes.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/revocable-capability-token-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Generate token policy <ArrowRight size={18} /></Link>
            <Link href="/economic-control-plane" className="inline-flex items-center justify-center gap-2 rounded-lg border border-cyan-500 px-6 py-3 font-bold text-cyan-200 transition hover:bg-cyan-950/30">Economic control plane</Link>
            <Link href="/agent-api-governance" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">Agent API governance</Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">AI agent cost control</Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1fr_0.9fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">Credentials have to carry economic policy</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>Human access systems assume stable users, managed devices, predictable sessions, and human-scale request rates. Agent systems are different: credentials can be copied into tools, delegated to sub-agents, retried in loops, and used faster than a billing alert can fire.</p>
            <p>The safe model is not a single permanent secret. It is a request-path capability inside an <Link href="/economic-control-plane" className="text-cyan-300 hover:text-cyan-200">economic control plane for AI agents</Link> that answers: what can this agent do, on which route, for how long, with what budget, and can it still be revoked right now?</p>
            <p>SatGate turns those answers into enforceable policy at the gateway before forwarding to an upstream API, model, or MCP tool happens.</p>
          </div>
        </div>
        <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">Why static keys fail</h3>
          <div className="space-y-4">
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Static keys are too broad</h3><p className="leading-relaxed text-gray-400">A leaked or copied key usually carries account-level authority until someone rotates it.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Agent tasks are short lived</h3><p className="leading-relaxed text-gray-400">A credential should die with the session, task, workflow, or customer it was minted for.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Revocation must be request-path</h3><p className="leading-relaxed text-gray-400">If a loop is already spending money, revocation has to block the next request, not a future deploy.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Delegation needs shrinking authority</h3><p className="leading-relaxed text-gray-400">Sub-agents should inherit less scope, less budget, and shorter expiry than their parent.</p></div>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">What SatGate checks before forwarding</h2>
          <p className="mb-10 max-w-3xl text-lg text-gray-400">The credential is only useful if policy is enforced inline, before the expensive or sensitive request reaches the upstream service.</p>
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

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-8 flex flex-col justify-between gap-4 md:flex-row md:items-end">
          <h2 className="text-3xl font-bold text-white">Credential policy example</h2>
          <Link href="/revocable-capability-token-policy-template" className="inline-flex items-center gap-2 font-semibold text-cyan-300 transition hover:text-cyan-200">
            Generate a revocable token policy <ArrowRight size={16} />
          </Link>
        </div>
        <pre className="overflow-x-auto rounded-2xl border border-gray-800 bg-gray-950 p-6 text-sm text-green-300"><code>{`agent: research-agent-17
scope:
  routes: [/api/research/*]
  tools: [web_search, document_fetch]
budget:
  max_usd: 25
  max_calls: 300
delegation:
  allowed: true
  child_budget_max_usd: 5
  child_ttl_minutes: 15
expiry: 2026-04-26T14:00:00Z
revocation:
  check: before_each_request
audit:
  fields: [agent, route, tool, budget_remaining, decision]`}</code></pre>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-6 text-3xl font-bold text-white">Revocable agent credentials FAQ</h2>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              ['What is a revocable agent credential?', 'A revocable agent credential is a scoped, expiring capability issued to an autonomous agent for a specific task, workflow, route, tool, budget, or time window. It can be invalidated before the next request without rotating global API keys.'],
              ['Why are static API keys risky for AI agents?', 'Static API keys are broad, long-lived, and hard to delegate safely. Autonomous agents need credentials with scoped authority, budget limits, expiry, revocation, and audit context.'],
              ['How does SatGate enforce agent credentials?', 'SatGate sits in the request path and checks identity, token scope, route, tool, budget, expiry, delegation rules, and revocation state before forwarding upstream.'],
              ['Where do revocable credentials fit in an economic control plane?', 'Revocable credentials are the kill-switch layer inside an economic control plane for AI agents. They let the gateway stop the next API, model, or MCP request when authority, budget, scope, or risk changes.'],
            ].map(([question, answer]) => (
              <div key={question} className="rounded-xl border border-gray-800 bg-black p-5">
                <h3 className="mb-2 font-bold text-white">{question}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 pb-24">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/35 to-yellow-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Agent autonomy needs scoped authority, not bigger secrets.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">SatGate provides the <Link href="/economic-control-plane" className="text-cyan-300 hover:text-cyan-200">economic control plane</Link> underneath agent credentials: observe who is calling, control what they can spend and access, and revoke authority before the next request.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Grade readiness <ArrowRight size={18} /></Link>
            <Link href="/economic-control-plane" className="inline-flex items-center justify-center gap-2 rounded-lg border border-cyan-500 px-6 py-3 font-bold text-cyan-200 transition hover:bg-cyan-950/30">Read the category page</Link>
            <Link href="/mcp-budget-enforcement" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">MCP budget enforcement</Link>
          </div>
        </div>
      </section>
    </main>
  );
}

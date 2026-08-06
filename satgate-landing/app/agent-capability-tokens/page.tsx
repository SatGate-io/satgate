import Link from 'next/link';
import { ArrowRight, Ban, Clock, Fingerprint, GitBranch, KeyRound, ReceiptText, ShieldCheck } from 'lucide-react';

export const metadata = {
  title: 'Agent Capability Tokens: Scoped, Time-Limited AI Agent Authority',
  description: 'Design capability tokens for AI agents with scoped actions, time limits, budgets, delegation attenuation, revocation, and request-path proof.',
  alternates: { canonical: 'https://satgate.io/agent-capability-tokens' },
  keywords: [
    'agent capability tokens',
    'capability tokens for AI agents',
    'AI agent capability token design',
    'scoped AI agent authority',
    'time-limited agent credentials',
    'capability token caveats',
    'macaroon tokens AI agents',
    'scoped agent authority',
    'delegated API access',
    'agent API governance',
    'budget-aware credentials',
  ],
  openGraph: {
    title: 'Agent Capability Tokens: Scoped Authority for AI Agents',
    description: 'Use capability tokens for AI agents to encode scoped actions, budgets, expiry, delegation attenuation, revocation, and request-path proof.',
    url: 'https://satgate.io/agent-capability-tokens',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Agent Capability Tokens for AI Agents',
    description: 'Design scoped, time-limited, revocable capability tokens for autonomous agents.',
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

const designPrinciples = [
  ['Specific actions', 'Grant only the API routes, MCP tools, model endpoints, and methods required for the current task.'],
  ['Time limits', 'Expire credentials after minutes, sessions, jobs, customers, or delegated sub-tasks instead of creating permanent access.'],
  ['Budget caveats', 'Attach spend caps, call ceilings, per-tool limits, and remaining-budget checks to every governed request.'],
  ['Delegation attenuation', 'Let child agents receive only narrower authority: smaller budget, shorter TTL, fewer scopes, and visible parent lineage.'],
  ['Revocation before reuse', 'Check revocation state before every request so a policy failure stops the next call, not the next billing review.'],
  ['Proof receipts', 'Record identity, capability caveats, route, tool, remaining budget, decision reason, policy version, and Evidence Pack id.'],
];

const designSteps = [
  ['Start with task scope', 'Define the exact action, route, tenant, tool, model, or data boundary the agent needs for one workflow.'],
  ['Add economic limits', 'Attach per-request, per-tool, session, lifetime, and child-delegation budgets before issuing authority.'],
  ['Constrain delegation', 'Require child capability tokens to be strict subsets of the parent authority with shorter TTLs and lower budgets.'],
  ['Enforce every use', 'Validate identity, scope, expiry, budget, delegation depth, and revocation state in the request path before forwarding.'],
  ['Emit proof', 'Write a signed receipt and Evidence Pack link for allow, deny, delegation, revocation, and budget decisions.'],
];

export default function Page() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Agent Capability Tokens: Scoped, Time-Limited AI Agent Authority',
    description: metadata.description,
    url: 'https://satgate.io/agent-capability-tokens',
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-26',
    dateModified: '2026-08-06',
    about: [
      { '@type': 'Thing', name: 'agent capability tokens' },
      { '@type': 'Thing', name: 'capability tokens for AI agents' },
      { '@type': 'Thing', name: 'AI agent capability token design' },
      { '@type': 'Thing', name: 'scoped AI agent authority' },
      { '@type': 'Thing', name: 'time-limited agent credentials' },
      { '@type': 'Thing', name: 'capability token caveats' },
      { '@type': 'Thing', name: 'budget-aware credentials' },
      { '@type': 'Thing', name: 'attenuated delegation' },
      { '@type': 'Thing', name: 'revocable request-path authority' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'What is an agent capability token?', acceptedAnswer: { '@type': 'Answer', text: 'An agent capability token is a credential that carries constrained authority for an autonomous agent, such as allowed routes, tools, budget, expiry, delegation, and revocation behavior. Macaroons are one implementation pattern for capability-based tokens.' } },
      { '@type': 'Question', name: 'How should agent capability tokens be designed?', acceptedAnswer: { '@type': 'Answer', text: 'Agent capability tokens should be scoped to specific actions, time-limited, budget-aware, revocable before every request, and safe for delegation only when child tokens are narrower than parent authority.' } },
      { '@type': 'Question', name: 'Why are static API keys risky for AI agents?', acceptedAnswer: { '@type': 'Answer', text: 'Static API keys are broad, long-lived, and hard to delegate safely. Autonomous agents need credentials with scoped authority, budget limits, expiry, revocation, and audit context.' } },
      { '@type': 'Question', name: 'How does SatGate enforce agent credentials?', acceptedAnswer: { '@type': 'Answer', text: 'SatGate sits in the request path and checks identity, token scope, route, tool, budget, expiry, delegation rules, and revocation state before forwarding upstream.' } },
    ],
  };

  const designPrinciplesJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent capability token design principles',
    itemListElement: designPrinciples.map(([name, description], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name,
      description,
    })),
  };

  const howToJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'HowTo',
    name: 'How to design capability tokens for AI agents',
    description: 'Define task scope, add economic limits, constrain delegation, enforce every use, and emit proof for each governed request.',
    step: designSteps.map(([name, text], index) => ({
      '@type': 'HowToStep',
      position: index + 1,
      name,
      text,
    })),
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Agent Capability Tokens', item: 'https://satgate.io/agent-capability-tokens' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(designPrinciplesJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(howToJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_34%),radial-gradient(circle_at_78%_12%,rgba(245,158,11,0.16),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200"><ShieldCheck size={16} /> Capability-based security for AI agents</div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">Agent capability tokens say what an agent can do — not just who it is</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">Identity proves the caller. Capability proves authority. For autonomous agents, the token should be scoped to specific actions, time-limited, budget-aware, revocable before every request, and safe for delegation only when child authority is narrower.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/capability-lifecycle-demo" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Walk lifecycle demo <ArrowRight size={18} /></Link>
            <Link href="/revocable-capability-token-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">Generate token policy</Link>
            <Link href="/blog/macaroon-tokens-vs-api-keys" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">Macaroons vs API keys</Link>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <div className="mb-10 max-w-4xl">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Direct answer</p>
            <h2 className="mb-4 text-3xl font-bold text-white">Agent capability tokens should be scoped, time-limited, budget-aware, revocable, and attenuated.</h2>
            <p className="text-lg leading-relaxed text-gray-300">
              The wrong design is permanent, broad, freely shareable authority that only gets validated once. The right design is a narrow capability checked before every API, model, MCP tool, or paid-rail request, with proof left behind for security and finance.
            </p>
          </div>
          <div className="grid gap-5 md:grid-cols-3">
            {designPrinciples.map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6">
                <h3 className="mb-2 text-xl font-bold text-white">{title}</h3>
                <p className="leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1fr_0.9fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">Credentials have to carry economic policy</h2>
          <div className="space-y-5 text-lg leading-relaxed text-gray-300">
            <p>Human access systems assume stable users, managed devices, predictable sessions, and human-scale request rates. Agent systems are different: credentials can be copied into tools, delegated to sub-agents, retried in loops, and used faster than a billing alert can fire.</p>
            <p>The safe model is not a single permanent secret. It is a request-path capability that answers: what can this agent do, on which route, for how long, with what budget, and can it still be revoked right now?</p>
            <p>SatGate turns those answers into enforceable policy at the gateway before forwarding to an upstream API, model, or MCP tool happens.</p>
          </div>
        </div>
        <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6">
          <h3 className="mb-4 text-xl font-bold text-white">Why static keys fail</h3>
          <div className="space-y-4">
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Identity is not authority</h3><p className="leading-relaxed text-gray-400">A verified agent can still be overpowered if the token does not constrain what it can do.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Budgets belong in the credential path</h3><p className="leading-relaxed text-gray-400">Spend limits, call ceilings, and route policy need to be evaluated before forwarding traffic.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Delegation should attenuate</h3><p className="leading-relaxed text-gray-400">A parent agent should be able to create a narrower child token, never a broader one.</p></div>
              <div className="rounded-xl border border-gray-800 bg-black p-6"><h3 className="mb-2 text-xl font-bold text-white">Audit needs token context</h3><p className="leading-relaxed text-gray-400">Every decision should emit a receipt with identity, capability, caveats, remaining budget, route, outcome, decision_reason, policy_version, and Evidence Pack id.</p></div>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <div className="mb-10 max-w-4xl">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Design path</p>
            <h2 className="mb-4 text-3xl font-bold text-white">How to design capability tokens for autonomous agents</h2>
            <p className="text-lg leading-relaxed text-gray-300">
              Capability-token design starts from the task, not the account. Every caveat should make the next request easier to allow, deny, revoke, or explain.
            </p>
          </div>
          <div className="grid gap-5 md:grid-cols-5">
            {designSteps.map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-5">
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-8 flex flex-col justify-between gap-4 md:flex-row md:items-end">
          <div>
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Lifecycle demo</p>
            <h2 className="text-3xl font-bold text-white">Issue → delegate → attenuate → revoke → prove</h2>
          </div>
          <Link href="/capability-lifecycle-demo" className="inline-flex items-center gap-2 font-semibold text-cyan-300 transition hover:text-cyan-200">
            Walk the visible lifecycle <ArrowRight size={16} />
          </Link>
        </div>
        <div className="grid gap-4 md:grid-cols-5">
          {[
            ['Issue', 'Parent gets scoped authority for one tenant, task, budget, and tool set.'],
            ['Delegate', 'A child worker receives linked authority from the parent.'],
            ['Attenuate', 'Child authority shrinks: lower spend cap, fewer tools, shorter TTL, visible depth.'],
            ['Revoke', 'SatGate rejects the next governed request at the gateway policy check.'],
            ['Prove', 'Evidence Pack shows lineage, caveats, decision, spend, and receipt IDs.'],
          ].map(([title, body]) => (
            <Link key={title} href="/capability-lifecycle-demo" className="rounded-2xl border border-gray-800 bg-gray-950 p-5 transition hover:border-cyan-500/60 hover:bg-cyan-950/10">
              <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
              <p className="text-sm leading-relaxed text-gray-400">{body}</p>
            </Link>
          ))}
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
            Generate a scoped token policy <ArrowRight size={16} />
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

      <section className="border-t border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Agent capability token questions</h2>
          <div className="grid gap-5 md:grid-cols-3">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is an agent capability token?</h3>
              <p className="leading-relaxed text-gray-400">
                An agent capability token is a credential that carries constrained authority for an autonomous agent, such as allowed routes, tools, budget, expiry, delegation, and revocation behavior. Macaroons are one implementation pattern.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How should agent capability tokens be designed?</h3>
              <p className="leading-relaxed text-gray-400">
                Agent capability tokens should be scoped to specific actions, time-limited, budget-aware, revocable before every request, and safe for delegation only when child tokens are narrower than parent authority.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why are static API keys risky for AI agents?</h3>
              <p className="leading-relaxed text-gray-400">
                Static API keys are broad, long-lived, and hard to delegate safely. Autonomous agents need credentials with scoped authority, budget limits, expiry, revocation, and audit context.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does SatGate enforce agent credentials?</h3>
              <p className="leading-relaxed text-gray-400">
                SatGate sits in the request path and checks identity, token scope, route, tool, budget, expiry, delegation rules, and revocation state before forwarding upstream.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 pb-24 pt-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/35 to-yellow-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Agent autonomy needs scoped authority, not bigger secrets.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">SatGate provides the economic firewall underneath agent credentials: observe who is calling, control what they can spend and access, and revoke authority before the next request.</p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">Grade readiness <ArrowRight size={18} /></Link>
            <Link href="/mcp-budget-enforcement" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">MCP budget enforcement</Link>
          </div>
        </div>
      </section>
    </main>
  );
}

import Link from 'next/link';
import { ArrowRight, BadgeCheck, Ban, Clock, Fingerprint, GitBranch, KeyRound, ShieldCheck } from 'lucide-react';

export const metadata = {
  title: 'Agent API Governance | Identity, Revocation, Budgets, Audit',
  description: 'Govern AI agent API access with scoped capabilities, delegation limits, revocation, policy checks, and Evidence Pack receipts before execution.',
  alternates: { canonical: 'https://satgate.io/agent-api-governance' },
  keywords: [
    'agent API governance',
    'AI agent API governance',
    'agent-scoped API keys',
    'revocable agent credentials',
    'capability tokens for AI agents',
    'delegated API access',
    'AI agent access control',
    'macaroons vs API keys',
    'zero trust for AI agents',
  ],
  openGraph: {
    title: 'Agent API Governance',
    description: 'Replace unlimited API keys with scoped, revocable, budget-aware agent capabilities enforced in the request path.',
    url: 'https://satgate.io/agent-api-governance',
    type: 'article',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Agent API Governance',
    description: 'Govern AI agent identity, delegated access, revocation, policy checks, and Evidence Pack receipts before API calls execute.',
  },
};

const principles = [
  {
    icon: Fingerprint,
    title: 'Identity is not enough',
    body: 'Knowing which agent called is useful. Governing what that agent can do, spend, delegate, and access is the actual control point.',
  },
  {
    icon: KeyRound,
    title: 'Capabilities beat static keys',
    body: 'Replace broad API keys with scoped, expiring, attenuated capabilities that carry policy with the request.',
  },
  {
    icon: Ban,
    title: 'Revocation must be immediate',
    body: 'A misbehaving agent needs a kill switch that works on the next request, not after a deploy or manual key rotation.',
  },
  {
    icon: GitBranch,
    title: 'Delegation must shrink authority',
    body: 'Sub-agents should inherit less power than their parent: smaller budgets, narrower routes, shorter expiry, tighter tools.',
  },
  {
    icon: Clock,
    title: 'Expiry is a safety primitive',
    body: 'Agent credentials should expire with the task, session, customer, or workflow they were created for.',
  },
  {
    icon: BadgeCheck,
    title: 'Audit should explain decisions',
    body: 'A governance trail must show identity, capability, policy, budget, route, decision, outcome, and Evidence Pack receipt for every important call.',
  },
];

export default function AgentApiGovernancePage() {
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Agent API Governance',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-25',
    dateModified: '2026-06-03',
    mainEntityOfPage: 'https://satgate.io/agent-api-governance',
    about: [
      { '@type': 'Thing', name: 'agent API governance' },
      { '@type': 'Thing', name: 'revocable agent credentials' },
      { '@type': 'Thing', name: 'capability tokens for AI agents' },
      { '@type': 'Thing', name: 'delegated API access control' },
      { '@type': 'Thing', name: 'request-path API policy' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is agent API governance?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Agent API governance is the request-path policy layer for AI agent identity, delegated authority, budgets, revocation, routing, and Evidence Pack receipts.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why are static API keys risky for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Static API keys are often broad, long-lived, and easy to copy. Autonomous agents need scoped, expiring, revocable capabilities that limit access and spend per task or workflow.',
        },
      },
      {
        '@type': 'Question',
        name: 'How should sub-agent delegation be controlled?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Delegated sub-agents should receive less authority than their parent: smaller budgets, narrower tools, shorter expiry, route limits, and separate audit records.',
        },
      },
      {
        '@type': 'Question',
        name: 'What should replace static API keys for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'AI agents should use scoped, expiring, revocable capability tokens or credentials that bind authority to an agent, tenant, task, route, tool, budget, expiry, and audit policy.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does agent API governance reduce blast radius?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Agent API governance reduces blast radius by narrowing each credential to the minimum routes, tools, budget, delegation depth, and lifetime needed for the task, with revocation before the next request.',
        },
      },
    ],
  };

  const requirementsJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'Agent API governance requirements',
    description: 'Core request-path controls required to govern AI agent API access without broad static keys, then preserve Evidence Pack receipts for Evidence Pack proof. See the visible capability lifecycle demo for issue, delegate, attenuate, revoke, and prove.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Agent-scoped identity',
        description: 'Every request identifies the tenant, agent, task, workflow, parent agent, delegated sub-agent, token, route, and tool behind the call.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Budget-aware authority',
        description: 'Access policy includes spend limits, per-request ceilings, daily caps, tool limits, and remaining budget checks before forwarding traffic.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Revocation before the next request',
        description: 'When an agent loops, leaks a token, or finishes a task, access can be narrowed, expired, or revoked immediately without rotating global keys.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Delegation with attenuation',
        description: 'Sub-agents never inherit full parent authority; each delegation shrinks scope, budget, lifetime, and allowed tools.',
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Agent API Governance', item: 'https://satgate.io/agent-api-governance' },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(requirementsJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.17),transparent_32%),radial-gradient(circle_at_80%_20%,rgba(245,158,11,0.14),transparent_30%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200 mb-8">
            <ShieldCheck size={16} /> Access governance for autonomous agents
          </div>

          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            Agent API Governance Starts Where API Keys Fail
          </h1>

          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-10">
            AI agents need more than static API keys. They need scoped capabilities, delegation limits, expiry, revocation, and policy checks before every API request — with a receipt proving each decision.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/capability-lifecycle-demo" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Walk capability lifecycle <ArrowRight size={18} />
            </Link>
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              See SatGate governance
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              See Policy-to-Proof
            </Link>
            <Link href="/revocable-agent-credentials" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              Revocable agent credentials
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12 items-start">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">The API key model was built for apps, not autonomous workers</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              Traditional API keys assume the caller is an application, service, or human-operated integration. Once issued, the key often carries broad authority until someone rotates it, revokes it, or discovers it leaked.
            </p>
            <p>
              AI agents change the risk model. They can receive goals, create sub-tasks, delegate work, retry operations, and use tools without a human approving each request. A single broad key becomes too much authority in too little context.
            </p>
            <p>
              Agent API governance means every call is constrained by identity, capability, policy, budget, route, and time. The request itself carries or references the rules that make it safe.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">A governed agent credential should answer</h3>
          <ul className="space-y-3 text-gray-300">
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Which agent, task, tenant, or workflow is using it?</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Which routes and tools are allowed?</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">What spend budget remains?</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">Can authority be delegated, and how far?</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">When does it expire or become invalid?</li>
            <li className="rounded-lg border border-gray-800 bg-black/50 p-3">What audit receipt proves the decision and feeds the Evidence Pack?</li>
          </ul>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">Agent API governance principles</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            Governance is not a login screen. It is a request-path policy system that checks authority before execution and turns every decision into evidence.
          </p>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {principles.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6 hover:border-yellow-900/70 transition">
                <Icon className="text-yellow-300 mb-4" size={28} />
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">Agent API governance requirements</h2>
          <div className="grid md:grid-cols-2 gap-5">
            {[
              ['Agent-scoped identity', 'Every request should identify the tenant, agent, task, workflow, parent agent, delegated sub-agent, token, route, and tool behind the call.'],
              ['Budget-aware authority', 'Access policy should include spend limits, per-request ceilings, daily caps, tool limits, and remaining budget checks before forwarding traffic.'],
              ['Revocation before the next request', 'When an agent loops, leaks a token, or finishes a task, access should be narrowed, expired, or revoked immediately without rotating global keys.'],
              ['Delegation with attenuation', 'Sub-agents should never inherit full parent authority. Each delegation should shrink scope, budget, lifetime, and allowed tools.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
                <h3 className="text-xl font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-8">Static API key vs agent capability</h2>
        <div className="overflow-hidden rounded-2xl border border-gray-800">
          <div className="grid md:grid-cols-3 bg-gray-900/70 text-sm font-bold text-white">
            <div className="p-4">Dimension</div>
            <div className="p-4">Static API key</div>
            <div className="p-4">Agent capability</div>
          </div>
          {[
            ['Authority', 'Usually broad and long-lived', 'Scoped to route, task, tenant, or tool'],
            ['Budget', 'External or manual', 'Embedded or enforced inline'],
            ['Delegation', 'Copied or shared', 'Attenuated: sub-agents get less authority'],
            ['Revocation', 'Rotate key or change config', 'Revoke/expire capability before next request'],
            ['Audit', 'Often aggregate usage only', 'Evidence Pack receipt per agent/tool/request'],
          ].map(([a, b, c]) => (
            <div key={a} className="grid md:grid-cols-3 border-t border-gray-800 text-gray-300">
              <div className="p-4 font-semibold text-white">{a}</div>
              <div className="p-4">{b}</div>
              <div className="p-4">{c}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-2 gap-8">
          <div>
            <h2 className="text-3xl font-bold text-white mb-6">Observe, Control, Prove agent API use</h2>
            <div className="space-y-4">
              {[
                ['Observe', 'Attribute calls by agent, worker, route, tenant, and workflow so real access and spend patterns are visible before policy tightens.'],
                ['Control', 'Issue scoped capabilities, enforce budgets and route policy, attenuate delegation, and revoke authority before the next request.'],
                ['Prove', 'Issue receipts for policy decisions, paid actions, denials, delegations, revocations, and Evidence Pack exports for each governed API action.'],
              ].map(([title, body]) => (
                <div key={title} className="rounded-xl border border-gray-800 bg-black p-5">
                  <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                  <p className="text-gray-400 leading-relaxed">{body}</p>
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Example capability policy</h2>
            <pre className="bg-black border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm text-gray-300"><code>{`parent_agent: finance-automation
worker_agent: invoice-reconciler
scope:
  routes: [/invoices/read, /vendors/match, /payments/schedule]
  denied_routes: [/payments/release]
authority:
  tenant: acme-finance
  workflow: monthly-close
budget:
  workflow: 25.00 USD
  per_request: 0.50 USD
delegation:
  allowed: true
  max_depth: 1
  child_budget_max: 5.00 USD
expiry: 2026-12-31T00:00:00Z
revocation: before_next_request
evidence:
  receipt: required
  include: [parent_agent, worker_agent, route, policy, decision, outcome]`}</code></pre>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-yellow-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Agent API governance questions</h2>
          <div className="grid gap-5 md:grid-cols-2 mb-16">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is agent API governance?</h3>
              <p className="text-gray-400 leading-relaxed">
                Agent API governance is the request-path policy layer for AI agent identity, delegated authority, budgets, revocation, routing, and Evidence Pack receipts.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why are static API keys risky for AI agents?</h3>
              <p className="text-gray-400 leading-relaxed">
                Static API keys are often broad, long-lived, and easy to copy. Autonomous agents need scoped, expiring, revocable capabilities that limit access and spend per task or workflow.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How should sub-agent delegation be controlled?</h3>
              <p className="text-gray-400 leading-relaxed">
                Delegated sub-agents should receive less authority than their parent: smaller budgets, narrower tools, shorter expiry, route limits, and separate audit records.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What should replace static API keys for AI agents?</h3>
              <p className="text-gray-400 leading-relaxed">
                AI agents should use scoped, expiring, revocable capability tokens or credentials that bind authority to an agent, tenant, task, route, tool, budget, expiry, and audit policy.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does agent API governance reduce blast radius?</h3>
              <p className="text-gray-400 leading-relaxed">
                Agent API governance reduces blast radius by narrowing each credential to the minimum routes, tools, budget, delegation depth, and lifetime needed for the task, with revocation before the next request.
              </p>
            </div>
          </div>

          <h2 className="text-3xl font-bold text-white mb-8">Related governance guides</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[
              ['/agent-api-key-risk-assessment', 'Agent API key risk assessment', 'Score static key risk across scope, budget, expiry, revocation, delegation, and audit gaps.'],
              ['/revocable-agent-credentials', 'Revocable agent credentials', 'Scoped credentials, expiry, and kill switches for autonomous access.'],
              ['/agent-capability-tokens', 'Agent capability tokens', 'Encode scope, budget, route, delegation, and revocation into agent authority.'],
              ['/policy-to-proof', 'Policy-to-Proof', 'See how agent API decisions become Evidence Pack proof.'],
              ['/blog/macaroon-tokens-vs-api-keys', 'Macaroons vs API keys', 'Why attenuated capabilities beat static API keys for agents.'],
              ['/agent-control-plane', 'Agent control plane', 'Govern enterprise agent authority, delegation lineage, spend, audit, and revocation.'],
              ['/evidence-pack-demo', 'Evidence Pack demo', 'Show how allow, deny, budget, delegation, and revocation decisions become receipts.'],
              ['/mcp-governance', 'MCP governance', 'Apply authority, budgets, revocation, and Evidence Pack receipts to agent tool calls.'],
            ].map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 transition hover:border-yellow-500/50 hover:bg-yellow-950/10">
                <h3 className="font-bold text-white mb-2">{title}</h3>
                <p className="text-sm text-gray-400 leading-relaxed">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <div className="rounded-3xl border border-yellow-900/60 bg-gradient-to-br from-yellow-950/20 to-cyan-950/30 p-8 md:p-12">
          <h2 className="text-3xl font-bold text-white mb-4">SatGate makes agent API access governable</h2>
          <p className="text-gray-300 text-lg leading-relaxed max-w-3xl mb-8">
            Put SatGate in the request path to move from unlimited API keys to scoped, revocable agent capabilities. Check authority before execution and produce Evidence Pack receipts for the Evidence Pack.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              See SatGate governance <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-cyan-500 transition">
              See Policy-to-Proof
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

import Link from 'next/link';
import { ArrowLeft, ArrowRight, BadgeCheck, KeyRound, Layers3, ShieldCheck, TimerReset, WalletCards } from 'lucide-react';

export const metadata = {
  title: 'Capability-Based Authorization for AI Agents',
  description: 'Replace broad API keys with scoped, revocable, budget-aware capabilities for AI agents using SatGate.',
  alternates: { canonical: 'https://satgate.io/capability-auth' },
  keywords: ['capability based authorization', 'capability auth', 'agent authorization', 'capability tokens', 'macaroon tokens', 'delegated authorization', 'AI agent permissions'],
  openGraph: {
    title: 'Capability-Based Authorization for AI Agents',
    description: 'Replace broad API keys with scoped, revocable, budget-aware capabilities and Evidence Pack proof for AI agents.',
    url: 'https://satgate.io/capability-auth',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Capability-Based Authorization for AI Agents',
    description: 'Use SatGate to give AI agents scoped, revocable, budget-aware authority before execution.',
  },
};

const capabilities = [
  { icon: KeyRound, title: 'Scope', body: 'Limit which APIs, MCP tools, routes, methods, customers, and environments an agent can access.' },
  { icon: WalletCards, title: 'Budget', body: 'Attach spend caps and per-tool prices so authorization includes economic policy, not just identity.' },
  { icon: Layers3, title: 'Delegate', body: 'Allow agents to pass down narrower authority to sub-agents without handing them a master key.' },
  { icon: TimerReset, title: 'Expire and revoke', body: 'Keep authority short-lived and kill risky access before the next request reaches a model, API, or tool.' },
];

const faqs = [
  ['What is capability-based authorization?', 'Capability-based authorization gives a caller a specific, constrained capability: what it can do, where it can do it, for how long, with what budget, and whether it can delegate narrower authority.'],
  ['Why are capabilities useful for AI agents?', 'Agents act autonomously, call tools repeatedly, and delegate work. Capabilities limit blast radius by encoding scope, budget, expiry, revocation, and delegation into the authority the agent actually uses.'],
  ['How are capabilities different from API keys?', 'API keys usually prove broad account ownership. Capabilities are narrower: they describe what this agent can access and spend right now, and they can be attenuated or revoked without rotating a shared secret.'],
  ['Can capabilities include budget limits?', 'Yes. SatGate treats economic policy as part of authorization. A capability can carry or reference budget, per-tool pricing, route scope, tenant context, and delegation depth.'],
  ['Are macaroons capability tokens?', 'Macaroons are a practical way to implement attenuated capability-style authority because caveats can constrain scope, time, budget, route, and delegation.'],
  ['How does UCAN relate to capability-based authorization?', 'UCAN is a capability-based authorization model built around user-controlled authorization networks. SatGate uses the same core idea of delegated, scoped, attenuable authority and adds request-path budget enforcement, revocation, and Evidence Pack proof for AI agents.'],
];

export default function CapabilityAuthPage() {
  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate Capability Auth',
    applicationCategory: 'SecurityApplication',
    description: metadata.description,
    url: 'https://satgate.io/capability-auth',
    dateModified: '2026-08-05',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    featureList: capabilities.map((item) => item.title),
  };
  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: faqs.map(([name, text]) => ({ '@type': 'Question', name, acceptedAnswer: { '@type': 'Answer', text } })),
  };
  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Capability Auth', item: 'https://satgate.io/capability-auth' },
    ],
  };
  const ucanJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'UCAN and capability-based authorization for AI agents',
    description: 'How UCAN-style capability delegation maps to AI agent authority, scoped access, attenuation, budget limits, revocation, and proof.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Delegated authority',
        description: 'A principal grants a specific capability to an agent instead of sharing broad account credentials.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Attenuation',
        description: 'A child capability can be narrower than the parent by route, tool, tenant, task, budget, expiry, and delegation depth.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Request-path enforcement',
        description: 'SatGate checks the capability before forwarding each API, model, MCP tool, or paid-access request.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Evidence Pack proof',
        description: 'Each allow, deny, delegation, revocation, and paid decision can leave a receipt for finance, security, and audit review.',
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(ucanJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <div className="mx-auto max-w-6xl px-6 pt-8">
        <Link href="/" className="inline-flex items-center gap-2 text-sm font-medium text-gray-500 transition hover:text-white">
          <ArrowLeft size={16} /> Back to Home
        </Link>
      </div>

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_32%),radial-gradient(circle_at_80%_15%,rgba(16,185,129,0.14),transparent_34%)]" />
        <div className="relative max-w-6xl mx-auto px-6 py-24">
          <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-950/30 px-4 py-2 text-sm text-emerald-200 mb-8">
            <ShieldCheck size={16} /> Scoped authority for autonomous agents
          </div>
          <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight max-w-5xl mb-8">
            Capability-Based Authorization for AI Agents
          </h1>
          <p className="text-xl md:text-2xl text-gray-300 max-w-4xl leading-relaxed mb-8">
            Identity tells you who is calling. Capability auth tells you what this agent can access, spend, delegate, and revoke before every API or MCP request.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/revocable-capability-token-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white text-black px-6 py-3 font-bold hover:bg-gray-200 transition">
              Generate capability policy <ArrowRight size={18} />
            </Link>
            <Link href="/blog/macaroon-tokens-vs-api-keys" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-emerald-500 transition">
              Compare macaroons and API keys
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-16">
        <div className="rounded-2xl border border-emerald-900/50 bg-gray-950 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-emerald-300">Direct answer</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Capability-based authorization replaces identity-only access with bounded authority</h2>
          <p className="mb-6 text-lg leading-relaxed text-gray-300">
            Capability-based authorization gives an AI agent a specific, bounded grant: which API, model, route, MCP tool, tenant, budget, expiry window, and delegation depth it can use. SatGate checks that capability in the request path before execution, then preserves Evidence Pack proof for allowed calls, denied calls, revocation, and delegated authority.
          </p>
          <div className="grid gap-4 md:grid-cols-4">
            {[
              ['Identity', 'Who is this agent, tenant, workflow, or delegated worker?'],
              ['Authority', 'Which route, tool, action, scope, and capability caveat is allowed?'],
              ['Economics', 'How much can it spend per request, session, tool, workflow, or tenant?'],
              ['Proof', 'Which receipt proves the grant, denial, revocation, or delegation decision?'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black/50 p-4">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
          <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
            <Link href="/blog/macaroon-tokens-vs-api-keys" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-emerald-500">
              Macaroons vs API keys <ArrowRight size={16} />
            </Link>
            <Link href="/agent-capability-tokens" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-emerald-500">
              Agent capability tokens <ArrowRight size={16} />
            </Link>
            <Link href="/mcp-gateway" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-emerald-500">
              MCP gateway <ArrowRight size={16} />
            </Link>
            <Link href="/revocable-capability-token-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-emerald-500">
              Capability policy template <ArrowRight size={16} />
            </Link>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20 grid lg:grid-cols-[1fr_0.9fr] gap-12">
        <div>
          <h2 className="text-3xl font-bold text-white mb-6">Why identity auth is too blunt for agents</h2>
          <div className="space-y-5 text-gray-300 text-lg leading-relaxed">
            <p>
              Traditional identity-based auth asks, “who is this?” That works for humans and stable apps, but autonomous agents need narrower authority. They need credentials that say which tool is allowed, what task it is for, how much it can spend, how long it lasts, and whether it can delegate.
            </p>
            <p>
              SatGate puts that capability check in the request path. The result is not just authentication. It is Policy-to-Proof governance: Observe the call, Control authority and budget before execution, and Prove each decision with an Evidence Pack receipt.
            </p>
          </div>
        </div>
        <div className="rounded-2xl border border-emerald-900/50 bg-emerald-950/10 p-6">
          <h3 className="text-xl font-bold text-white mb-4">Bad default vs better default</h3>
          <div className="space-y-4">
            <div className="rounded-xl border border-red-900/40 bg-red-950/10 p-4">
              <h4 className="font-bold text-red-200 mb-2">Broad API key</h4>
              <p className="text-gray-400">One shared secret, broad account access, weak delegation, painful rotation, and no native budget boundary.</p>
            </div>
            <div className="rounded-xl border border-emerald-900/40 bg-emerald-950/20 p-4">
              <h4 className="font-bold text-emerald-200 mb-2">Scoped capability</h4>
              <p className="text-gray-400">A constrained token for one agent, task, route, tool set, budget, expiry window, and delegation depth.</p>
            </div>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-4">What a capability should carry</h2>
          <p className="text-gray-400 max-w-3xl mb-10 text-lg">
            For agent systems, authorization has to include economic policy. A safe capability answers more than who. It answers what, where, how much, how long, and who can inherit less authority.
          </p>
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-5">
            {capabilities.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6">
                <Icon className="text-emerald-300 mb-4" size={28} />
                <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-8">Capabilities vs API keys, OAuth, and macaroons</h2>
        <div className="grid md:grid-cols-3 gap-6">
          {[
            ['API keys', 'Useful for simple server-to-server access, but usually too broad for autonomous agents and weak at scoped delegation.'],
            ['OAuth', 'Strong for user consent and identity delegation, but often too heavyweight and human-centered for short-lived agent tool authority.'],
            ['Macaroons', 'A practical token format for attenuated capabilities because caveats can narrow scope, time, budget, route, and delegation.'],
          ].map(([title, body]) => (
            <div key={title} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="text-2xl font-bold text-white mb-3">{title}</h3>
              <p className="text-gray-300 leading-relaxed">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-[0.2em] text-emerald-300">UCAN-style authority</p>
          <h2 className="mb-4 text-3xl font-bold text-white">UCAN capability authorization is the right mental model for agents</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            UCAN-style authorization starts from a better premise than static keys: give software a specific delegated capability, let it pass down less authority, and make every later use prove it stayed inside bounds.
          </p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
            {[
              ['Delegated authority', 'A principal grants a specific capability to an agent instead of sharing broad account credentials.'],
              ['Attenuation', 'A child capability can be narrower than the parent by route, tool, tenant, task, budget, expiry, and delegation depth.'],
              ['Request-path enforcement', 'SatGate checks the capability before forwarding each API, model, MCP tool, or paid-access request.'],
              ['Evidence Pack proof', 'Each allow, deny, delegation, revocation, and paid decision leaves a receipt for finance, security, and audit review.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6">
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="max-w-6xl mx-auto px-6 py-20">
          <h2 className="text-3xl font-bold text-white mb-8">Capability auth questions</h2>
          <div className="grid md:grid-cols-2 gap-5">
            {faqs.map(([q, a]) => (
              <div key={q} className="rounded-xl border border-gray-800 bg-black p-6">
                <h3 className="text-xl font-bold text-white mb-3">{q}</h3>
                <p className="text-gray-400 leading-relaxed">{a}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-white mb-6">Related authorization resources</h2>
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            ['/blog/macaroon-tokens-vs-api-keys', 'Macaroon tokens vs API keys', 'Compare broad account keys with attenuated capability-style credentials.'],
            ['/mcp-gateway', 'MCP gateway', 'Apply capabilities to tool calls.'],
            ['/agent-capability-tokens', 'Agent capability tokens', 'See how scoped authority is encoded.'],
            ['/govern', 'AI agent governance', 'Observe, Control, and Prove agent actions.'],
            ['/revocable-capability-token-policy-template', 'Revocable capability template', 'Generate scoped, expiring, budget-aware policy in YAML or JSON.'],
            ['/blog/how-to-add-budget-limits-to-openai-api-calls', 'Budget limits', 'Attach spend policy to access.'],
          ].map(([href, title, body]) => (
            <Link key={href} href={href} className="rounded-xl border border-gray-800 bg-gray-950 p-5 hover:border-emerald-700 transition">
              <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
              <p className="text-sm text-gray-400">{body}</p>
            </Link>
          ))}
        </div>
      </section>

      <section className="border-t border-gray-900 bg-emerald-950/10">
        <div className="max-w-4xl mx-auto px-6 py-20 text-center">
          <BadgeCheck className="mx-auto mb-6 text-emerald-300" size={36} />
          <h2 className="text-3xl font-bold text-white mb-4">Stop handing agents broad API keys.</h2>
          <p className="text-gray-300 mb-8">Use capabilities that expire, attenuate, meter, delegate safely, stop spend before the next request, and leave Evidence Pack proof.</p>
          <Link href="/design-partners" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black hover:bg-gray-200 transition">
            Work with SatGate <ArrowRight size={18} />
          </Link>
        </div>
      </section>
    </main>
  );
}

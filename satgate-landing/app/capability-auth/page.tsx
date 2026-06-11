import Link from 'next/link';
import { ArrowLeft, ArrowRight, BadgeCheck, KeyRound, Layers3, ShieldCheck, TimerReset, WalletCards } from 'lucide-react';

export const metadata = {
  title: 'Capability-Based Authorization for AI Agents',
  description: 'Replace broad API keys with scoped, revocable capabilities, authority before execution, and Evidence Pack proof.',
  alternates: { canonical: 'https://satgate.io/capability-auth' },
  keywords: ['capability based authorization', 'capability auth', 'agent authorization', 'capability tokens', 'macaroon tokens', 'delegated authorization', 'AI agent permissions'],
  openGraph: {
    title: 'Capability-Based Authorization for AI Agents',
    description: 'Give agents scoped, revocable capabilities and Evidence Pack proof instead of broad static API keys.',
    url: 'https://satgate.io/capability-auth',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Capability-Based Authorization for AI Agents',
    description: 'Use SatGate to scope agent authority before execution and prove decisions with Evidence Packs.',
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
];

export default function CapabilityAuthPage() {
  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate Capability Auth',
    applicationCategory: 'SecurityApplication',
    description: metadata.description,
    url: 'https://satgate.io/capability-auth',
    dateModified: '2026-06-11',
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

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
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
            <Link href="/agent-capability-tokens" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white hover:border-emerald-500 transition">
              Agent capability tokens
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
            ['/mcp-gateway', 'MCP gateway', 'Apply capabilities to tool calls.'],
            ['/agent-capability-tokens', 'Agent capability tokens', 'See how scoped authority is encoded.'],
            ['/revocable-capability-token-policy-template', 'Capability policy template', 'Generate a scoped, revocable policy.'],
            ['/revocable-agent-credentials', 'Revocable credentials', 'Kill risky agent access before the next request.'],
            ['/govern', 'AI agent governance', 'Observe, Control, and Prove agent actions.'],
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

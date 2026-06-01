import Link from 'next/link';
import { ArrowLeft, ArrowRight, CheckCircle2, CircleAlert, GitBranch, KeyRound, Network, PackageCheck, Route, Server, ShieldCheck, WalletCards, XCircle } from 'lucide-react';

export type BrutalComparison = {
  slug: string;
  competitor: string;
  eyebrow: string;
  title: string;
  description: string;
  verdict: string;
  competitorGoodAt: string[];
  satgateGoodAt: string[];
  rows: Array<{
    axis: string;
    satgate: string;
    competitor: string;
    winner: 'SatGate' | 'Competitor' | 'Tie';
  }>;
  bullets: Array<{ title: string; body: string }>;
  faqs: Array<{ question: string; answer: string }>;
  ctaPrimary: { href: string; label: string };
  ctaSecondary: { href: string; label: string };
};

const iconMap = [ShieldCheck, Network, WalletCards, GitBranch, PackageCheck, Server, Route, KeyRound];

export function BrutalComparisonPage({ config }: { config: BrutalComparison }) {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: config.title,
    description: config.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-05-10',
    dateModified: '2026-05-10',
    mainEntityOfPage: `https://satgate.io/compare/${config.slug}`,
    about: [
      { '@type': 'Thing', name: 'agent authority governance' },
      { '@type': 'Thing', name: 'pre-execution policy enforcement' },
      { '@type': 'Thing', name: 'MCP-native proxying' },
      { '@type': 'Thing', name: 'Evidence Packs' },
      { '@type': 'Thing', name: config.competitor },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: config.faqs.map((faq) => ({
      '@type': 'Question',
      name: faq.question,
      acceptedAnswer: { '@type': 'Answer', text: faq.answer },
    })),
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Compare SatGate', item: 'https://satgate.io/compare' },
      { '@type': 'ListItem', position: 3, name: config.title, item: `https://satgate.io/compare/${config.slug}` },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_0%,rgba(34,211,238,0.18),transparent_32%),radial-gradient(circle_at_80%_8%,rgba(168,85,247,0.16),transparent_30%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-16 md:py-24">
          <Link href="/compare" className="mb-10 inline-flex items-center gap-2 text-gray-500 transition hover:text-white">
            <ArrowLeft size={18} /> Back to comparisons
          </Link>
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-red-500/30 bg-red-950/20 px-4 py-2 text-sm font-mono text-red-200">
            <CircleAlert size={16} /> Direct comparison
          </div>
          <h1 className="mb-7 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">{config.title}</h1>
          <p className="mb-8 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">{config.description}</p>
          <div className="max-w-4xl rounded-2xl border border-cyan-800/40 bg-cyan-950/15 p-6">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Verdict</p>
            <p className="text-2xl font-bold leading-snug text-white">{config.verdict}</p>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-8 px-6 py-16 lg:grid-cols-2">
        <div className="rounded-2xl border border-gray-800 bg-gray-950 p-8">
          <h2 className="mb-5 text-2xl font-bold text-white">Where {config.competitor} is genuinely useful</h2>
          <ul className="space-y-3 text-gray-300">
            {config.competitorGoodAt.map((item) => (
              <li key={item} className="flex gap-3 leading-relaxed">
                <CheckCircle2 className="mt-1 shrink-0 text-purple-300" size={18} /> {item}
              </li>
            ))}
          </ul>
        </div>
        <div className="rounded-2xl border border-cyan-900/60 bg-cyan-950/10 p-8">
          <h2 className="mb-5 text-2xl font-bold text-white">Where SatGate evaluates agent authority</h2>
          <ul className="space-y-3 text-gray-300">
            {config.satgateGoodAt.map((item) => (
              <li key={item} className="flex gap-3 leading-relaxed">
                <ShieldCheck className="mt-1 shrink-0 text-cyan-300" size={18} /> {item}
              </li>
            ))}
          </ul>
          <div className="mt-6 flex flex-col gap-4 sm:flex-row">
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-800 px-6 py-3 font-bold text-gray-300 transition hover:border-cyan-500 hover:text-white">
              Policy-to-Proof
            </Link>
            <Link href="/mcp-governance" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-800 px-6 py-3 font-bold text-gray-300 transition hover:border-cyan-500 hover:text-white">
              MCP governance
            </Link>
            <Link href="/evidence-pack-demo" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-800 px-6 py-3 font-bold text-gray-300 transition hover:border-cyan-500 hover:text-white">
              Evidence Pack demo
            </Link>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-16">
          <h2 className="mb-4 text-3xl font-bold text-white">What to compare for agent governance</h2>
          <p className="mb-10 max-w-4xl text-lg leading-relaxed text-gray-400">
            Routing, dashboards, billing caps, and rate limits are useful. They are not the same as cross-provider, cross-rail, pre-execution authority for autonomous agents. SatGate makes the operational loop explicit: Observe the request, Control the delegated budget before execution, and Prove the outcome with an Evidence Pack receipt.
          </p>
          <div className="overflow-hidden rounded-2xl border border-gray-800">
            <div className="grid gap-4 bg-gray-900/80 px-5 py-4 text-sm font-bold uppercase tracking-wide text-gray-400 md:grid-cols-[1.1fr_1.4fr_1.4fr_.7fr]">
              <div>Axis</div>
              <div className="text-cyan-300">SatGate</div>
              <div>{config.competitor}</div>
              <div>Edge</div>
            </div>
            {config.rows.map((row) => (
              <div key={row.axis} className="grid gap-4 border-t border-gray-800 px-5 py-5 text-sm leading-relaxed text-gray-300 md:grid-cols-[1.1fr_1.4fr_1.4fr_.7fr]">
                <div className="font-semibold text-white">{row.axis}</div>
                <div>{row.satgate}</div>
                <div className="text-gray-400">{row.competitor}</div>
                <div className={row.winner === 'SatGate' ? 'font-bold text-cyan-300' : row.winner === 'Competitor' ? 'font-bold text-purple-300' : 'font-bold text-gray-300'}>{row.winner}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-16">
        <h2 className="mb-8 text-3xl font-bold text-white">Why this matters in production</h2>
        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
          {config.bullets.map((item, index) => {
            const Icon = iconMap[index % iconMap.length];
            return (
              <div key={item.title} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
                <Icon className="mb-4 text-cyan-300" size={28} />
                <h3 className="mb-2 text-xl font-bold text-white">{item.title}</h3>
                <p className="leading-relaxed text-gray-400">{item.body}</p>
              </div>
            );
          })}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-black">
        <div className="mx-auto grid max-w-6xl gap-8 px-6 py-16 lg:grid-cols-[1.1fr_.9fr]">
          <div>
            <p className="mb-3 text-sm font-mono uppercase tracking-wide text-cyan-300">Policy-to-Proof layer</p>
            <h2 className="mb-5 text-3xl font-bold text-white">The hard question is not routing. It is who had authority before execution.</h2>
            <div className="space-y-4 text-lg leading-relaxed text-gray-300">
              <p>
                Most gateways, observability tools, and payment rails explain a narrow part of the transaction: where a request went, how much it cost, or whether a token was valid. Enterprise agent governance needs a pre-execution decision that binds identity, tenant, delegated scope, budget, tool, payment context, and revocation state before the upstream system sees the call.
              </p>
              <p>
                That is the SatGate distinction in these comparisons. SatGate is not trying to replace every model router, tracing stack, API gateway, or paid rail. It sits above them as an Agent Authority &amp; Accountability Layer: Observe the agent request, Control what it is allowed to do, and Prove the decision with an Evidence Pack that security, finance, and compliance can inspect later.
              </p>
            </div>
          </div>
          <div className="rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
            <h3 className="mb-4 text-xl font-bold text-white">What an Evidence Pack should preserve</h3>
            <ul className="space-y-3 text-gray-300">
              <li><strong className="text-white">Authority:</strong> the agent, user, tenant, token caveats, and delegated depth behind the request.</li>
              <li><strong className="text-white">Policy:</strong> the budget, tool, paid-rail, allowlist, and revocation checks evaluated before execution.</li>
              <li><strong className="text-white">Decision:</strong> whether SatGate allowed, denied, downgraded, routed, or required additional approval.</li>
              <li><strong className="text-white">Proof:</strong> signed receipt metadata that can survive dashboards, vendor logs, and postmortem guesswork.</li>
            </ul>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-16">
          <h2 className="mb-8 text-3xl font-bold text-white">FAQ</h2>
          <div className="grid gap-5 md:grid-cols-3">
            {config.faqs.map((faq) => (
              <div key={faq.question} className="rounded-xl border border-gray-800 bg-black p-6">
                <h3 className="mb-3 text-lg font-bold text-white">{faq.question}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{faq.answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-16">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <div className="mb-5 flex items-center gap-3 text-cyan-300"><XCircle size={22} /> <span className="font-mono text-sm uppercase tracking-wide">The governance gap</span></div>
          <h2 className="mb-4 max-w-4xl text-3xl font-bold text-white">Dashboards explain what happened. SatGate controls what agents are allowed to do.</h2>
          <p className="mb-8 max-w-4xl text-lg leading-relaxed text-gray-300">
            Put SatGate before the paid API call, MCP tool invocation, delegated sub-agent, or model spend. Give agents bounded authority, enforce it before execution, and leave an Evidence Pack when finance, security, or compliance asks why it happened.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href={config.ctaPrimary.href} className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              {config.ctaPrimary.label} <ArrowRight size={18} />
            </Link>
            <Link href={config.ctaSecondary.href} className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              {config.ctaSecondary.label}
            </Link>
            <Link href="/compare" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-800 px-6 py-3 font-bold text-gray-300 transition hover:border-cyan-500 hover:text-white">
              See SatGate comparisons
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

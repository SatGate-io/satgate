'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { ArrowRight, BadgeCheck, CircleAlert, Gauge, KeyRound, ReceiptText, ShieldCheck } from 'lucide-react';

type Answer = 0 | 1 | 2;

type Question = {
  id: string;
  category: string;
  prompt: string;
  fix: string;
};

const questions: Question[] = [
  {
    id: 'identity',
    category: 'Agent identity',
    prompt: 'Can you attribute every API/tool call to tenant, agent, workflow, delegated sub-agent, token, route, and model?',
    fix: 'Start in Observe mode and require agent/workflow metadata on every request before enforcing authority or optimizing spend.',
  },
  {
    id: 'budgets',
    category: 'Budget enforcement',
    prompt: 'Can you block or route requests before they exceed per-agent, per-session, per-tool, daily, or per-request budgets?',
    fix: 'Move high-risk routes from dashboard-only monitoring to request-path Control policy.',
  },
  {
    id: 'mcp',
    category: 'MCP governance',
    prompt: 'Do MCP tool calls have prices, authority scope, risk tiers, denial reasons, and Evidence Pack trails before tools execute?',
    fix: 'Proxy MCP traffic and assign explicit policy, price, risk, and Evidence Pack fields to search, browser, cloud, code, data, and premium API tools.',
  },
  {
    id: 'revocation',
    category: 'Revocation',
    prompt: 'Can you revoke or narrow an agent capability before the next request without rotating global API keys?',
    fix: 'Replace broad keys with scoped, expiring capabilities and immediate kill switches.',
  },
  {
    id: 'delegation',
    category: 'Delegation',
    prompt: 'When agents spawn sub-agents, does delegated authority shrink by budget, scope, tool list, route, and expiry?',
    fix: 'Require attenuation: child agents should inherit less power, not a copy of parent credentials.',
  },
  {
    id: 'audit',
    category: 'Audit evidence',
    prompt: 'Can finance/security/platform teams reconstruct who had authority, what happened, why it was allowed or denied, and which policy decided?',
    fix: 'Record the authority chain, policy decision, denial reason, estimated cost, remaining budget, route, tool, credential, and upstream outcome for Evidence Pack export.',
  },
  {
    id: 'routing',
    category: 'Economic routing',
    prompt: 'Can routine agent work route to cheaper providers while premium models require budget or justification?',
    fix: 'Add model/route policy that defaults to economical paths and reserves premium calls for justified work.',
  },
  {
    id: 'paidRails',
    category: 'Paid-rail context',
    prompt: 'When value moves across paid rails, can you preserve payment context alongside authority, denial, delegation, and revocation evidence?',
    fix: 'Govern paid calls across x402, L402, AgentCore Payments, Pay.sh, API-key billing, or enterprise ledgers without making payment the center of the control model.',
  },
];

const labels: Record<Answer, string> = {
  0: 'No / unknown',
  1: 'Partial',
  2: 'Yes',
};

function grade(score: number) {
  if (score >= 85) return { label: 'A', title: 'Economically governable', color: 'text-green-400', summary: 'Your stack has the core controls for autonomous agent authority, spend, and Evidence Pack proof.' };
  if (score >= 70) return { label: 'B', title: 'Close, with gaps', color: 'text-cyan-300', summary: 'You have real governance primitives, but a few request-path controls need tightening.' };
  if (score >= 50) return { label: 'C', title: 'Observable but fragile', color: 'text-yellow-300', summary: 'You can probably explain activity after the fact, but agents can still outrun authority policy.' };
  if (score >= 30) return { label: 'D', title: 'High authority risk', color: 'text-orange-400', summary: 'Agents can likely access, spend, delegate, or call tools faster than you can stop them.' };
  return { label: 'F', title: 'Not ready for autonomous authority', color: 'text-red-400', summary: 'This is static-key/dashboard territory. Put governance in the request path before scaling agents.' };
}

export default function EconomicFirewallReadinessGraderPage() {
  const [answers, setAnswers] = useState<Record<string, Answer>>(() => Object.fromEntries(questions.map((q) => [q.id, 0])) as Record<string, Answer>);

  const result = useMemo(() => {
    const earned = questions.reduce((sum, q) => sum + answers[q.id], 0);
    const max = questions.length * 2;
    const score = Math.round((earned / max) * 100);
    const missing = questions.filter((q) => answers[q.id] < 2);
    return { score, grade: grade(score), missing };
  }, [answers]);

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'Economic Firewall Readiness Grader',
    url: 'https://satgate.io/economic-firewall-readiness-grader',
    description: 'Grade AI agent economic governance readiness across identity, request-path budgets, MCP tools, revocation, delegation, audit, routing, and paid-rail context.',
    dateModified: '2026-05-05',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'economic firewall readiness' },
      { '@type': 'Thing', name: 'AI agent economic governance' },
      { '@type': 'Thing', name: 'request-path budget controls' },
      { '@type': 'Thing', name: 'MCP governance assessment' },
      { '@type': 'Thing', name: 'paid-rail governance readiness' },
    ],
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'Economic Firewall Readiness Grader',
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/economic-firewall-readiness-grader',
    description: 'Grade AI agent economic governance readiness across identity, budgets, MCP tools, revocation, delegation, audit, routing, and payments.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-05',
    about: webPageJsonLd.about,
    featureList: ['Agent identity attribution scoring', 'Request-path budget readiness scoring', 'MCP tool cost governance checks', 'Revocation and delegation readiness checks', 'Audit, routing, and paid-rail context readiness checks'],
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'What is economic firewall readiness?', acceptedAnswer: { '@type': 'Answer', text: 'Economic firewall readiness measures whether an organization can observe, control, audit, revoke, route, budget, and preserve paid-rail context for AI agent/API activity before requests execute.' } },
      { '@type': 'Question', name: 'What score means we are ready for autonomous agents?', acceptedAnswer: { '@type': 'Answer', text: 'A score above 85 means most core request-path controls are in place. Lower scores indicate gaps in identity, budget enforcement, MCP governance, revocation, audit, routing, or payment.' } },
      { '@type': 'Question', name: 'How does SatGate improve readiness?', acceptedAnswer: { '@type': 'Answer', text: 'SatGate sits in the request path to observe agent/API activity, enforce budget and access policy, revoke scoped capabilities, preserve Evidence Pack proof, route economically, and govern paid rails when value moves.' } },
      { '@type': 'Question', name: 'Which gaps should teams fix first?', acceptedAnswer: { '@type': 'Answer', text: 'Teams should fix request attribution, hard budget enforcement, MCP tool policy, scoped revocable credentials, and Evidence Pack capture first because those controls stop unauthorized actions before execution.' } },
      { '@type': 'Question', name: 'Is a dashboard enough for economic firewall readiness?', acceptedAnswer: { '@type': 'Answer', text: 'No. Dashboards help explain activity after it happens, but economic firewall readiness requires request-path controls that can allow, deny, route, revoke, delegate, or preserve paid-rail context before agents execute work.' } },
    ],
  };

  const readinessCriteriaJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'Economic firewall readiness criteria',
    description: 'Eight readiness checks for governing AI agent authority, spend, MCP tools, scoped authority, Evidence Pack capture, routing, and paid-rail context before requests execute.',
    itemListElement: questions.map((question, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: question.category,
      description: `${question.prompt} Recommended fix: ${question.fix}`,
    })),
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Economic Firewall', item: 'https://satgate.io/economic-firewall' },
      { '@type': 'ListItem', position: 3, name: 'Economic Firewall Readiness Grader', item: 'https://satgate.io/economic-firewall-readiness-grader' },
    ],
  };
  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(readinessCriteriaJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <ShieldCheck size={16} /> AI agent governance assessment
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">Economic Firewall Readiness Grader</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Grade whether your agent/API stack can handle autonomous authority: identity, request-path budgets, MCP tool policy, revocation, delegation, Evidence Pack capture, routing, and paid-rail context.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Learn economic firewalls <ArrowRight size={18} />
            </Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              See agent authority control
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[1.05fr_0.95fr]">
        <div className="space-y-4">
          {questions.map((q) => (
            <div key={q.id} className="rounded-2xl border border-gray-800 bg-gray-950 p-5">
              <div className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">{q.category}</div>
              <h2 className="mb-4 text-lg font-bold text-white">{q.prompt}</h2>
              <div className="grid gap-2 sm:grid-cols-3">
                {([0, 1, 2] as Answer[]).map((value) => (
                  <button
                    key={value}
                    type="button"
                    onClick={() => setAnswers((current) => ({ ...current, [q.id]: value }))}
                    className={`rounded-lg border px-3 py-2 text-sm font-semibold transition ${answers[q.id] === value ? 'border-cyan-400 bg-cyan-950/50 text-white' : 'border-gray-800 bg-black text-gray-400 hover:border-gray-600 hover:text-white'}`}
                  >
                    {labels[value]}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div className="space-y-6 lg:sticky lg:top-6 lg:self-start">
          <div className="rounded-3xl border border-cyan-900/60 bg-cyan-950/10 p-8">
            <div className="mb-4 flex items-center justify-between gap-6">
              <div>
                <div className="text-sm text-gray-400">Readiness score</div>
                <div className="text-5xl font-extrabold text-white">{result.score}</div>
              </div>
              <div className={`text-7xl font-black ${result.grade.color}`}>{result.grade.label}</div>
            </div>
            <h2 className="mb-3 text-2xl font-bold text-white">{result.grade.title}</h2>
            <p className="leading-relaxed text-gray-300">{result.grade.summary}</p>
          </div>

          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <h3 className="mb-4 flex items-center gap-2 text-xl font-bold text-white"><CircleAlert className="text-yellow-300" size={22} /> Priority fixes</h3>
            <div className="space-y-3">
              {result.missing.slice(0, 5).map((q) => (
                <div key={q.id} className="rounded-xl border border-gray-800 bg-black p-4">
                  <div className="mb-1 font-bold text-white">{q.category}</div>
                  <p className="text-sm leading-relaxed text-gray-400">{q.fix}</p>
                </div>
              ))}
              {result.missing.length === 0 && (
                <div className="rounded-xl border border-green-900/60 bg-green-950/20 p-4 text-green-300">
                  Strong posture. Next step: validate controls with live traffic and conversion paths.
                </div>
              )}
            </div>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-8 text-3xl font-bold text-white">What the grader measures</h2>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
            {[
              [KeyRound, 'Authority', 'Agent identity, scoped credentials, expiry, attenuation, and revocation.'],
              [Gauge, 'Budget and authority limits', 'Request-path budgets, scoped authority, per-tool caps, model routing, and loop prevention.'],
              [ReceiptText, 'Evidence', 'Evidence Pack trails that explain authority, cost, policy decision, denial reason, route, and outcome.'],
              [BadgeCheck, 'Paid-rail context', 'Observe, Control, and Prove paths for internal agents and rail-aware paid calls.'],
            ].map(([Icon, title, body]) => {
              const TypedIcon = Icon as typeof KeyRound;
              return (
                <div key={title as string} className="rounded-xl border border-gray-800 bg-black p-6">
                  <TypedIcon className="mb-4 text-cyan-300" size={28} />
                  <h3 className="mb-2 text-lg font-bold text-white">{title as string}</h3>
                  <p className="leading-relaxed text-gray-400">{body as string}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Economic firewall readiness questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            {[
              ['What is economic firewall readiness?', 'Economic firewall readiness measures whether an organization can observe, control, audit, revoke, route, budget, and preserve paid-rail context for AI agent/API activity before requests execute.'],
              ['What score means we are ready for autonomous agents?', 'A score above 85 means most core request-path controls are in place. Lower scores indicate gaps in identity, budget enforcement, MCP governance, revocation, audit, routing, or payment.'],
              ['How does SatGate improve readiness?', 'SatGate sits in the request path to observe agent/API activity, enforce budget and access policy, revoke scoped capabilities, preserve Evidence Pack proof, route economically, and govern paid rails when value moves.'],
              ['Which gaps should teams fix first?', 'Teams should fix request attribution, hard budget enforcement, MCP tool policy, scoped revocable credentials, and Evidence Pack capture first because those controls stop unauthorized actions before execution.'],
              ['Is a dashboard enough for economic firewall readiness?', 'No. Dashboards help explain activity after it happens, but economic firewall readiness requires request-path controls that can allow, deny, route, revoke, delegate, or preserve paid-rail context before agents execute work.'],
            ].map(([question, answer]) => (
              <div key={question} className="rounded-xl border border-gray-800 bg-gray-950 p-6">
                <h3 className="mb-2 text-xl font-bold text-white">{question}</h3>
                <p className="text-gray-400 leading-relaxed">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Move weak areas into request-path control</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            SatGate governs agent authority before execution: Observe every agent call, Control risky requests before they execute, and Prove allowed, denied, delegated, revoked, or paid decisions with an Evidence Pack.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/mcp-tool-cost-policy-generator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Generate MCP tool policy <ArrowRight size={18} />
            </Link>
            <Link href="/openai-budget-policy-generator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Generate OpenAI budget policy
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

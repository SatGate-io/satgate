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
    fix: 'Start in Observe mode and require agent/workflow metadata on every request before optimizing spend.',
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
    prompt: 'Do MCP tool calls have prices, spend caps, risk tiers, and audit trails before tools execute?',
    fix: 'Proxy MCP traffic and assign explicit costs to search, browser, cloud, code, data, and premium API tools.',
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
    prompt: 'Can finance/security/platform teams reconstruct who spent what, why it was allowed, and which policy decided?',
    fix: 'Record policy decision, estimated cost, remaining budget, route, tool, credential, and upstream outcome.',
  },
  {
    id: 'routing',
    category: 'Economic routing',
    prompt: 'Can routine agent work route to cheaper providers while premium models require budget or justification?',
    fix: 'Add model/route policy that defaults to economical paths and reserves premium calls for justified work.',
  },
  {
    id: 'charge',
    category: 'Robot payments',
    prompt: 'If external agents call your APIs, can you charge them per request with L402 before access is unlocked?',
    fix: 'Use Charge/L402 for robot-customer APIs where payment and authorization should happen in the request path.',
  },
];

const labels: Record<Answer, string> = {
  0: 'No / unknown',
  1: 'Partial',
  2: 'Yes',
};

function grade(score: number) {
  if (score >= 85) return { label: 'A', title: 'Economically governable', color: 'text-green-400', summary: 'Your stack has the core controls for autonomous agent/API spend.' };
  if (score >= 70) return { label: 'B', title: 'Close, with gaps', color: 'text-cyan-300', summary: 'You have real governance primitives, but a few request-path controls need tightening.' };
  if (score >= 50) return { label: 'C', title: 'Observable but fragile', color: 'text-yellow-300', summary: 'You can probably explain spend after the fact, but agents can still outrun policy.' };
  if (score >= 30) return { label: 'D', title: 'High runaway-spend risk', color: 'text-orange-400', summary: 'Agents can likely spend, delegate, or call tools faster than you can stop them.' };
  return { label: 'F', title: 'Not ready for autonomous spend', color: 'text-red-400', summary: 'This is static-key/dashboard territory. Put governance in the request path before scaling agents.' };
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

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'Economic Firewall Readiness Grader',
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/economic-firewall-readiness-grader',
    description: 'Grade AI agent economic governance readiness across identity, budgets, MCP tools, revocation, delegation, audit, routing, and payments.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      { '@type': 'Question', name: 'What is economic firewall readiness?', acceptedAnswer: { '@type': 'Answer', text: 'Economic firewall readiness measures whether an organization can observe, control, audit, revoke, route, budget, and optionally charge AI agent/API activity before requests execute.' } },
      { '@type': 'Question', name: 'What score means we are ready for autonomous agents?', acceptedAnswer: { '@type': 'Answer', text: 'A score above 85 means most core request-path controls are in place. Lower scores indicate gaps in identity, budget enforcement, MCP governance, revocation, audit, routing, or payment.' } },
      { '@type': 'Question', name: 'How does SatGate improve readiness?', acceptedAnswer: { '@type': 'Answer', text: 'SatGate sits in the request path to observe agent/API spend, enforce budget and access policy, revoke scoped capabilities, audit decisions, route economically, and use L402 Charge when agents become customers.' } },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(34,211,238,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <ShieldCheck size={16} /> AI agent governance assessment
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">Economic Firewall Readiness Grader</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Grade whether your agent/API stack can handle autonomous spend: identity, request-path budgets, MCP tool costs, revocation, delegation, audit, routing, and L402 robot payments.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Learn economic firewalls <ArrowRight size={18} />
            </Link>
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              See agent cost control
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
              [Gauge, 'Spend control', 'Request-path budgets, per-tool caps, model routing, and loop prevention.'],
              [ReceiptText, 'Evidence', 'Audit trails that explain every cost, policy decision, route, and outcome.'],
              [BadgeCheck, 'Commercial readiness', 'Observe, Control, and Charge paths for internal agents and robot customers.'],
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

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-purple-950/30 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Move weak areas into request-path control</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            SatGate is the economic control plane for AI agents: Observe what agents spend, Control risky requests before they execute, and Charge robot customers with L402 when APIs become products.
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

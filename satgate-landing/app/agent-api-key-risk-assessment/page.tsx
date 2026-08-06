'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { AlertTriangle, ArrowRight, KeyRound, LockKeyhole, ReceiptText, ShieldCheck } from 'lucide-react';

function Toggle({ label, checked, onChange, weight }: { label: string; checked: boolean; onChange: (checked: boolean) => void; weight: number }) {
  return (
    <label className="flex cursor-pointer items-start gap-4 rounded-xl border border-gray-800 bg-black/50 p-5 transition hover:border-orange-500/50">
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        className="mt-1 h-5 w-5 accent-orange-400"
      />
      <span>
        <span className="block font-semibold text-white">{label}</span>
        <span className="mt-1 block text-sm text-gray-500">+{weight} risk points</span>
      </span>
    </label>
  );
}

const risks = [
  { key: 'shared', label: 'One shared API key is used by multiple agents, tools, or workflows', weight: 18 },
  { key: 'longLived', label: 'The key is long-lived and does not expire automatically', weight: 14 },
  { key: 'broadScope', label: 'The key can access more routes, tools, or data than the agent needs', weight: 16 },
  { key: 'noBudget', label: 'There is no per-agent, per-task, or per-workflow spend budget attached to the key', weight: 18 },
  { key: 'noRevocation', label: 'Revocation requires rotating secrets or redeploying services', weight: 12 },
  { key: 'delegation', label: 'Agents can pass the key or equivalent authority to sub-agents or tools', weight: 12 },
  { key: 'noAudit', label: 'Audit logs cannot prove which agent, route, tool, budget, and policy decision created spend', weight: 10 },
];

const replacementControls = [
  ['Shared production key', 'Agent-scoped API keys', 'Issue authority per agent, tenant, task, workflow, route, and tool instead of sharing one credential across automations.'],
  ['Manual API key rotation', 'Automatic expiry and revocation', 'Shorten lifetime, revoke by capability, and stop a single agent without rotating every integration.'],
  ['Broad route access', 'Attenuated scope', 'Constrain allowed methods, routes, MCP tools, data classes, and downstream providers before forwarding.'],
  ['No spend limit', 'Budget-aware capabilities', 'Attach per-request, per-tool, per-session, and delegated child budgets directly to authority.'],
  ['Generic request logs', 'Policy-to-proof receipts', 'Record identity, scope, budget, route, tool, policy decision, revocation state, and outcome for audit.'],
];

const assessmentSteps = [
  ['Inventory keys', 'Map every static API key, shared secret, MCP server token, service account, and production credential an agent can reach.'],
  ['Score blast radius', 'Check whether each key is shared, long-lived, broad, unbudgeted, delegated, hard to revoke, or impossible to attribute.'],
  ['Replace authority', 'Move high-risk workflows to scoped, revocable, budget-aware capabilities enforced in the request path.'],
  ['Prove decisions', 'Record policy decisions, remaining budget, route, tool, receipt ID, and outcome so security and finance can audit usage.'],
];

export default function AgentApiKeyRiskAssessmentPage() {
  const [answers, setAnswers] = useState<Record<string, boolean>>({
    shared: true,
    longLived: true,
    broadScope: true,
    noBudget: true,
    noRevocation: false,
    delegation: true,
    noAudit: true,
  });

  const result = useMemo(() => {
    const score = risks.reduce((sum, risk) => sum + (answers[risk.key] ? risk.weight : 0), 0);
    const grade = score >= 75 ? 'Critical' : score >= 50 ? 'High' : score >= 25 ? 'Medium' : 'Low';
    const summary = score >= 75
      ? 'Your agents are holding broad economic authority. Move this workflow behind request-path budget enforcement before expanding autonomy.'
      : score >= 50
        ? 'This key model will create attribution, revocation, and spend-control gaps as agents scale.'
        : score >= 25
          ? 'Some controls exist, but missing scope, budget, expiry, or audit can still turn one agent mistake into real spend.'
          : 'This key posture is relatively constrained. Keep tightening with scoped, revocable, budget-aware capabilities.';
    return { score, grade, summary };
  }, [answers]);

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'Agent API Key Risk Assessment',
    url: 'https://satgate.io/agent-api-key-risk-assessment',
    description: 'Free assessment for AI agent API key security risk in autonomous workflows, including shared keys, scope, budget, API key rotation, revocation, delegation, and audit gaps.',
    datePublished: '2026-04-12',
    dateModified: '2026-08-06',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI agent API key risk assessment' },
      { '@type': 'Thing', name: 'AI agent API key security' },
      { '@type': 'Thing', name: 'agent API key management' },
      { '@type': 'Thing', name: 'API key rotation for AI agents' },
      { '@type': 'Thing', name: 'agent-scoped API keys' },
      { '@type': 'Thing', name: 'static API key blast radius' },
      { '@type': 'Thing', name: 'revocable agent credentials' },
      { '@type': 'Thing', name: 'budget-aware capability tokens' },
      { '@type': 'Thing', name: 'request-path agent API governance' },
    ],
    audience: { '@type': 'Audience', audienceType: 'Security, API, platform, and AI engineering teams' },
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'Agent API Key Risk Assessment',
    applicationCategory: 'SecurityApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/agent-api-key-risk-assessment',
    description: 'Free assessment for AI agent API key security risk in autonomous workflows, including shared keys, scope, budget, API key rotation, revocation, delegation, and audit gaps.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-08-06',
    audience: webPageJsonLd.audience,
    featureList: ['Static API key risk scoring', 'AI agent API key security scoring', 'Scope gap assessment', 'Budget control checklist', 'API key rotation gap assessment', 'Revocation gap assessment', 'Delegation and audit risk scoring'],
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const riskFactorsJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent API key risk factors',
    description: 'Risk factors that make static API keys dangerous when autonomous AI agents, MCP tools, and delegated sub-agents can access paid APIs.',
    itemListElement: risks.map((risk, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: risk.label,
      description: `${risk.weight} risk points in the SatGate Agent API Key Risk Assessment.`,
    })),
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Agent API Governance', item: 'https://satgate.io/agent-api-governance' },
      { '@type': 'ListItem', position: 3, name: 'Agent API Key Risk Assessment', item: 'https://satgate.io/agent-api-key-risk-assessment' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Why are static API keys risky for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Static API keys are usually broad, long-lived, copyable, and disconnected from task-level budgets. Autonomous agents can loop, retry, delegate, or call paid tools quickly, so key authority needs scope, expiry, revocation, budget, and audit controls.',
        },
      },
      {
        '@type': 'Question',
        name: 'How should teams manage API keys for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'AI agent API key management should avoid shared long-lived secrets. Use scoped, expiring, revocable, budget-aware credentials, bind them to tenant and task identity, and enforce policy in the request path before each API or MCP tool call.',
        },
      },
      {
        '@type': 'Question',
        name: 'What should replace broad API keys for agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Use scoped, revocable, budget-aware agent capabilities enforced in the request path. Each capability should limit route, tool, spend, delegation, expiry, and audit requirements for one task or workflow.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate reduce API key risk?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate sits in the request path and checks identity, budget, route, scope, expiry, revocation, and policy at the gateway before forwarding to an upstream API or MCP tool.',
        },
      },
    ],
  };

  const replacementControlsJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent API key replacement controls',
    itemListElement: replacementControls.map(([risk, replacement, description], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: `${risk} -> ${replacement}`,
      description,
    })),
  };

  const assessmentHowToJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'HowTo',
    name: 'How to assess AI agent API key risk',
    description: 'Inventory static API keys, score blast radius, replace broad authority with scoped capabilities, and prove request-path decisions.',
    step: assessmentSteps.map(([name, text], index) => ({
      '@type': 'HowToStep',
      position: index + 1,
      name,
      text,
    })),
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(riskFactorsJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(replacementControlsJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(assessmentHowToJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_22%_0%,rgba(249,115,22,0.18),transparent_34%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-orange-500/30 bg-orange-950/30 px-4 py-2 text-sm text-orange-200">
            <KeyRound size={16} /> Free agent security assessment
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            Agent API Key Risk Assessment
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Score your AI agent API key security posture: shared keys, broad scopes, weak API key rotation, missing budgets, delegated sub-agents, revocation gaps, and audit blind spots before agents can spend or access resources without a human in the loop.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <a href="#assessment" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Run assessment <ArrowRight size={18} />
            </a>
            <Link href="/revocable-capability-token-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-orange-500">
              Generate token policy
            </Link>
            <Link href="/agent-api-governance" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-orange-500">
              Agent API governance
            </Link>
          </div>
        </div>
      </section>

      <section id="assessment" className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[1fr_0.8fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">Check every risk that applies</h2>
          <div className="grid gap-4">
            {risks.map((risk) => (
              <Toggle
                key={risk.key}
                label={risk.label}
                weight={risk.weight}
                checked={Boolean(answers[risk.key])}
                onChange={(checked) => setAnswers((current) => ({ ...current, [risk.key]: checked }))}
              />
            ))}
          </div>
        </div>

        <aside className="sticky top-6 h-fit rounded-3xl border border-orange-900/50 bg-orange-950/10 p-8">
          <div className="mb-3 flex items-center gap-2 text-orange-300"><AlertTriangle size={22} /> Risk score</div>
          <div className="mb-3 text-7xl font-extrabold text-white">{result.score}</div>
          <div className="mb-6 inline-flex rounded-full border border-orange-500/30 bg-black px-4 py-2 font-bold text-orange-200">{result.grade} risk</div>
          <p className="mb-8 text-lg leading-relaxed text-gray-300">{result.summary}</p>
          <div className="space-y-3 text-sm text-gray-400">
            <p><strong className="text-white">0-24:</strong> Low</p>
            <p><strong className="text-white">25-49:</strong> Medium</p>
            <p><strong className="text-white">50-74:</strong> High</p>
            <p><strong className="text-white">75-100:</strong> Critical</p>
          </div>
        </aside>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">What safer agent authority looks like</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            API keys were designed for applications. Autonomous agents need attenuated, revocable, budget-aware capabilities enforced before the request reaches the upstream API or MCP server.
          </p>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              [LockKeyhole, 'Scope', 'Limit routes, tools, tenants, actions, data, and delegation for one task or workflow.'],
              [ShieldCheck, 'Control', 'Attach budgets, expiry, per-request ceilings, kill switches, and revocation checks.'],
              [ReceiptText, 'Audit', 'Record identity, capability, budget, route, tool, policy, decision, and outcome.'],
            ].map(([Icon, title, body]) => {
              const CardIcon = Icon as typeof LockKeyhole;
              return (
                <div key={String(title)} className="rounded-2xl border border-gray-800 bg-black p-6">
                  <CardIcon className="mb-4 text-cyan-300" size={30} />
                  <h3 className="mb-2 text-xl font-bold text-white">{String(title)}</h3>
                  <p className="leading-relaxed text-gray-400">{String(body)}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-10 max-w-4xl">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-orange-300">Replacement model</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Agent API key management should move from static secrets to scoped capabilities</h2>
          <p className="text-lg leading-relaxed text-gray-300">
            API key rotation is not enough when agents can loop, delegate, and spend. The safer pattern is to issue narrow authority for one task, enforce cost and scope before every request, and revoke that authority without breaking unrelated services.
          </p>
        </div>
        <div className="overflow-x-auto rounded-2xl border border-gray-800">
          <table className="min-w-[760px] w-full border-collapse text-left text-sm">
            <thead className="bg-gray-950 text-gray-300">
              <tr>
                <th className="p-4 font-semibold">Risky key pattern</th>
                <th className="p-4 font-semibold">Replacement control</th>
                <th className="p-4 font-semibold">Why it matters for agents</th>
              </tr>
            </thead>
            <tbody>
              {replacementControls.map(([risk, replacement, description]) => (
                <tr key={risk} className="border-t border-gray-800 bg-black/60 align-top">
                  <td className="p-4 font-bold text-white">{risk}</td>
                  <td className="p-4 text-cyan-200">{replacement}</td>
                  <td className="p-4 leading-relaxed text-gray-400">{description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <div className="mb-10 max-w-4xl">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-orange-300">Assessment path</p>
            <h2 className="mb-4 text-3xl font-bold text-white">How to assess API key blast radius before agents inherit it</h2>
            <p className="text-lg leading-relaxed text-gray-300">
              The assessment starts with inventory, but the real question is whether one autonomous workflow can turn a leaked, copied, or overbroad credential into spend, data exposure, or irreversible API actions.
            </p>
          </div>
          <div className="grid gap-5 md:grid-cols-4">
            {assessmentSteps.map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-6">
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-b border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-orange-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Agent API key risk questions</h2>
          <div className="grid gap-5 md:grid-cols-3">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why are static API keys risky for AI agents?</h3>
              <p className="leading-relaxed text-gray-400">Static API keys are usually broad, long-lived, copyable, and disconnected from task-level budgets. Autonomous agents can loop, retry, delegate, or call paid tools quickly, so key authority needs scope, expiry, revocation, budget, and audit controls.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How should teams manage API keys for AI agents?</h3>
              <p className="leading-relaxed text-gray-400">AI agent API key management should avoid shared long-lived secrets. Use scoped, expiring, revocable, budget-aware credentials, bind them to tenant and task identity, and enforce policy in the request path before each API or MCP tool call.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What should replace broad API keys for agents?</h3>
              <p className="leading-relaxed text-gray-400">Use scoped, revocable, budget-aware agent capabilities enforced in the request path. Each capability should limit route, tool, spend, delegation, expiry, and audit requirements for one task or workflow.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does SatGate reduce API key risk?</h3>
              <p className="leading-relaxed text-gray-400">SatGate sits in the request path and checks identity, budget, route, scope, expiry, revocation, and policy at the gateway before forwarding to an upstream API or MCP tool.</p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/60 bg-gradient-to-br from-cyan-950/30 to-orange-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Move from API keys to economic capabilities.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            SatGate turns agent access into request-path policy: scoped authority, spend limits, revocation, audit, and payment controls at the gateway before forwarding.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/revocable-capability-token-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Generate token policy <ArrowRight size={18} />
            </Link>
            <Link href="/agent-capability-tokens" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Agent capability tokens
            </Link>
            <Link href="/revocable-agent-credentials" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Revocable credentials
            </Link>
            <Link href="/economic-firewall-readiness-grader" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Economic firewall readiness
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

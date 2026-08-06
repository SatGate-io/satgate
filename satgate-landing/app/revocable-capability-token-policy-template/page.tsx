'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { ArrowRight, ClipboardList, Copy, KeyRound, LockKeyhole, ReceiptText, ShieldCheck, TimerReset } from 'lucide-react';

const profiles = {
  coding: {
    subject: 'agent:coding-assistant',
    audience: 'repo-and-model-tools',
    scopes: ['model:responses:write', 'mcp:repo_search:read', 'mcp:pull_request:write'],
    maxBudgetUsd: 25,
    ttlMinutes: 240,
    delegation: true,
    childBudgetPct: 20,
    risk: 'medium',
  },
  support: {
    subject: 'agent:support-copilot',
    audience: 'support-and-crm-tools',
    scopes: ['model:chat:write', 'api:crm:read', 'api:tickets:write'],
    maxBudgetUsd: 10,
    ttlMinutes: 120,
    delegation: false,
    childBudgetPct: 0,
    risk: 'low',
  },
  research: {
    subject: 'agent:research-analyst',
    audience: 'web-and-data-tools',
    scopes: ['model:responses:write', 'mcp:web_search:read', 'api:data_vendor:read'],
    maxBudgetUsd: 50,
    ttlMinutes: 90,
    delegation: true,
    childBudgetPct: 15,
    risk: 'high',
  },
  externalAccess: {
    subject: 'agent:external-client',
    audience: 'governed-api-access',
    scopes: ['api:metered:read', 'policy:decision:read', 'receipt:audit:write'],
    maxBudgetUsd: 100,
    ttlMinutes: 60,
    delegation: false,
    childBudgetPct: 0,
    risk: 'high',
  },
};

type ProfileKey = keyof typeof profiles;

const caveatRows = [
  ['Subject caveat', 'subject, tenant_id, task_id', 'Prevents one agent credential from becoming ambient authority across tenants or workflows.'],
  ['Scope caveat', 'scopes, audience, allowed routes/tools', 'Limits the token to the model, API, MCP tool, data class, or route the task actually needs.'],
  ['Economic caveat', 'token_lifetime_usd, per_request_usd', 'Turns authorization into budget-aware authority instead of a blank check.'],
  ['Delegation caveat', 'child_budget_pct, child_ttl_minutes, subset scopes', 'Allows sub-agents only with attenuated scope, shorter lifetime, and smaller budget.'],
  ['Revocation caveat', 'kill_switch, parent revocation, loop detection', 'Lets SatGate stop future requests without rotating unrelated service credentials.'],
];

const revocationRows = [
  ['Budget exhausted', 'Revoke and block when the token, child token, task, or session budget reaches zero.'],
  ['Loop detected', 'Revoke the session when repeated retries or tool calls show runaway behavior.'],
  ['Policy violation', 'Revoke when requested route, tool, scope, tenant, or audience does not match the caveats.'],
  ['Parent revoked', 'Cascade revocation through delegated child tokens when parent authority is removed.'],
  ['Manual kill switch', 'Let security or platform teams revoke all matching agent authority for a tenant or incident.'],
];

const generationSteps = [
  ['Choose profile', 'Select the agent workflow type so default scopes, budgets, risk tier, expiry, and delegation behavior match expected risk.'],
  ['Bind context', 'Set tenant and task identifiers so the capability token is tied to a specific authority boundary.'],
  ['Add caveats', 'Encode scope, budget, expiry, delegation, revocation, and audit requirements as policy fields.'],
  ['Enforce in path', 'Check caveats, revocation state, remaining budget, and receipt requirements at the gateway before forwarding.'],
];

export default function RevocableCapabilityTokenPolicyTemplatePage() {
  const [profile, setProfile] = useState<ProfileKey>('coding');
  const [tenant, setTenant] = useState('acme-prod');
  const [task, setTask] = useState('ship-agent-workflow');

  const policy = useMemo(() => {
    const p = profiles[profile];
    const tenantSlug = tenant.trim().toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-|-$/g, '') || 'tenant';
    const taskSlug = task.trim().toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-|-$/g, '') || 'task';
    const yaml = `capability_token_policy: ${tenantSlug}-${taskSlug}\nmode: enforce\nissuer: satgate-economic-firewall\nsubject: ${p.subject}\naudience: ${p.audience}\ntenant_id: ${tenantSlug}\ntask_id: ${taskSlug}\nrisk_tier: ${p.risk}\nvalidity:\n  ttl_minutes: ${p.ttlMinutes}\n  not_before: now\n  expires_at: now + ${p.ttlMinutes}m\nscopes:\n${p.scopes.map((scope) => `  - ${scope}`).join('\n')}\nbudgets:\n  token_lifetime_usd: ${p.maxBudgetUsd}\n  per_request_usd: ${Math.max(0.25, p.maxBudgetUsd / 50)}\n  on_exhausted: revoke_and_block\ndelegation:\n  allowed: ${p.delegation}\n  child_budget_pct: ${p.childBudgetPct}\n  child_ttl_minutes: ${Math.min(60, p.ttlMinutes)}\n  child_scopes_must_be_subset: true\nrevocation:\n  revoke_on_budget_exhausted: true\n  revoke_on_loop_detected: true\n  revoke_on_policy_violation: true\n  revoke_on_parent_revoked: true\n  kill_switch: tenant:${tenantSlug}:agents\naudit:\n  required_fields:\n    - token_id\n    - parent_token_id\n    - tenant_id\n    - agent_id\n    - task_id\n    - scope\n    - estimated_cost_usd\n    - remaining_budget_usd\n    - revocation_state\n    - policy_decision\n    - decision_reason\n    - policy_version\n    - receipt_id\n    - evidence_pack_id`;
    const json = JSON.stringify({
      capability_token_policy: `${tenantSlug}-${taskSlug}`,
      mode: 'enforce',
      issuer: 'satgate-economic-firewall',
      subject: p.subject,
      audience: p.audience,
      tenant_id: tenantSlug,
      task_id: taskSlug,
      risk_tier: p.risk,
      validity: { ttl_minutes: p.ttlMinutes, not_before: 'now', expires_at: `now + ${p.ttlMinutes}m` },
      scopes: p.scopes,
      budgets: { token_lifetime_usd: p.maxBudgetUsd, per_request_usd: Math.max(0.25, p.maxBudgetUsd / 50), on_exhausted: 'revoke_and_block' },
      delegation: { allowed: p.delegation, child_budget_pct: p.childBudgetPct, child_ttl_minutes: Math.min(60, p.ttlMinutes), child_scopes_must_be_subset: true },
      revocation: { revoke_on_budget_exhausted: true, revoke_on_loop_detected: true, revoke_on_policy_violation: true, revoke_on_parent_revoked: true, kill_switch: `tenant:${tenantSlug}:agents` },
      audit: { required_fields: ['token_id', 'parent_token_id', 'tenant_id', 'agent_id', 'task_id', 'scope', 'estimated_cost_usd', 'remaining_budget_usd', 'revocation_state', 'policy_decision', 'decision_reason', 'policy_version', 'receipt_id', 'evidence_pack_id'] },
    }, null, 2);
    return { yaml, json };
  }, [profile, task, tenant]);

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'Revocable Capability Token Policy Template',
    url: 'https://satgate.io/revocable-capability-token-policy-template',
    description: 'Generate scoped, expiring, revocable capability-token policy for AI agents, sub-agents, MCP tools, caveats, delegation attenuation, budgets, receipts, and Evidence Pack evidence.',
    datePublished: '2026-04-12',
    dateModified: '2026-08-06',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'revocable capability token policy template' },
      { '@type': 'Thing', name: 'AI agent capability token' },
      { '@type': 'Thing', name: 'capability token caveats' },
      { '@type': 'Thing', name: 'delegation attenuation for agents' },
      { '@type': 'Thing', name: 'revocable agent credentials' },
      { '@type': 'Thing', name: 'scoped AI agent credentials' },
      { '@type': 'Thing', name: 'agent token attenuation' },
      { '@type': 'Thing', name: 'macaroon-style caveats for agents' },
      { '@type': 'Thing', name: 'budget-aware credential revocation' },
    ],
    audience: { '@type': 'Audience', audienceType: 'Security, platform, API, and AI engineering teams' },
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'Revocable Capability Token Policy Template',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/revocable-capability-token-policy-template',
    description: 'Generate scoped, expiring, revocable capability-token policy for AI agents, sub-agents, MCP tools, caveats, delegation attenuation, budgets, receipts, and Evidence Pack evidence.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-08-06',
    audience: webPageJsonLd.audience,
    featureList: ['YAML capability-token policy generation', 'JSON capability-token policy generation', 'Capability token caveats', 'Delegation attenuation controls', 'Budget exhaustion revocation rules', 'Receipt field templates', 'Evidence Pack export fields'],
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const checklistJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'Capability-token policy checklist for AI agents',
    description: 'Required fields for scoped, expiring, revocable, budget-aware AI agent capability tokens that produce receipts and Evidence Pack evidence.',
    itemListElement: [
      ['Scope', 'Bind authority to tenant, agent, task, audience, route, and MCP tool permissions.'],
      ['Expiry', 'Use short token lifetimes and shorter child-token TTLs for delegated sub-agents.'],
      ['Revocation', 'Revoke on budget exhaustion, loops, parent revocation, policy violation, or kill switch.'],
      ['Delegation', 'Require child capabilities to be strict subsets with attenuated budgets and scopes.'],
      ['Audit', 'Log token id, parent id, spend context, remaining budget, scope, revocation state, decision, receipt id, and Evidence Pack id.'],
      ['Economic control', 'Pair identity with budgets so authentication, spend context, and proof capture happen together.'],
    ].map(([name, description], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name,
      description,
    })),
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'AI Agent Cost Control Tools', item: 'https://satgate.io/tools' },
      { '@type': 'ListItem', position: 3, name: 'Revocable Capability Token Policy Template', item: 'https://satgate.io/revocable-capability-token-policy-template' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is a revocable capability token for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'A revocable capability token gives an agent narrowly scoped authority for a tenant, task, tool, budget, and time window. Unlike a static API key, it can expire, be attenuated for sub-agents, and be revoked when policy fails.',
        },
      },
      {
        '@type': 'Question',
        name: 'What caveats should AI agent capability tokens include?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'AI agent capability-token caveats should bind subject, tenant, task, audience, scope, route, tool, budget, expiry, delegation limits, revocation triggers, and audit fields so every request can be checked before access.',
        },
      },
      {
        '@type': 'Question',
        name: 'What is delegation attenuation for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Delegation attenuation means a child or sub-agent receives less authority than the parent: smaller budget, shorter expiry, subset scopes, and stricter revocation rules.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why are capability tokens better than shared API keys for agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Shared API keys are broad, long-lived, and hard to revoke safely. Capability tokens bind authority to a specific agent task with budget limits, expiry, delegation rules, receipts, and Evidence Pack fields.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate enforce these token policies?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate sits in the request path as an economic firewall, checking token scope, budget, delegation, revocation state, and receipt policy at the gateway before forwarding to model, API, MCP, or externally exposed agent access.',
        },
      },
    ],
  };

  const caveatsJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent capability token caveats',
    itemListElement: caveatRows.map(([name, fields, description], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name,
      description: `${fields}: ${description}`,
    })),
  };

  const revocationJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'Capability token revocation triggers',
    itemListElement: revocationRows.map(([name, description], index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name,
      description,
    })),
  };

  const howToJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'HowTo',
    name: 'How to generate revocable AI agent capability-token policy',
    description: 'Choose an agent profile, bind tenant and task context, add caveats, then enforce the policy in the request path.',
    step: generationSteps.map(([name, text], index) => ({
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
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(checklistJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(caveatsJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(revocationJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(howToJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(34,211,238,0.18),transparent_32%),radial-gradient(circle_at_82%_10%,rgba(168,85,247,0.16),transparent_34%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <KeyRound size={16} /> Free capability-token policy template
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            Revocable Capability Token Policy Template
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Generate scoped, expiring, revocable AI agent capability-token policy with caveats for sub-agents, MCP tools, request budgets, delegation attenuation, kill switches, receipts, and Evidence Pack evidence.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <a href="#template" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Generate policy <ArrowRight size={18} />
            </a>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              See Policy-to-Proof
            </Link>
          </div>
        </div>
      </section>

      <section id="template" className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[0.72fr_1fr]">
        <div className="space-y-5">
          <h2 className="text-3xl font-bold text-white">Configure the token policy</h2>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">Tenant</span>
            <input value={tenant} onChange={(event) => setTenant(event.target.value)} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500" />
          </label>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">Task</span>
            <input value={task} onChange={(event) => setTask(event.target.value)} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500" />
          </label>
          <label className="block rounded-xl border border-gray-800 bg-gray-950 p-5">
            <span className="mb-2 block font-semibold text-white">Agent profile</span>
            <select value={profile} onChange={(event) => setProfile(event.target.value as ProfileKey)} className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500">
              <option value="coding">Coding agent</option>
              <option value="support">Support copilot</option>
              <option value="research">Research analyst</option>
              <option value="externalAccess">External agent access</option>
            </select>
          </label>
          <div className="rounded-xl border border-cyan-900/50 bg-cyan-950/10 p-5 text-sm leading-relaxed text-cyan-100">
            Static API keys are wrong for autonomous agents. Capabilities should be narrow, priced, time-boxed, receipt-producing, and revocable.
          </div>
        </div>

        <div className="grid gap-5">
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-5">
            <div className="mb-3 flex items-center justify-between gap-4">
              <h2 className="text-xl font-bold text-white">YAML token policy</h2>
              <button onClick={() => navigator.clipboard?.writeText(policy.yaml)} className="inline-flex items-center gap-2 rounded-lg border border-gray-700 px-3 py-2 text-sm font-semibold text-white transition hover:border-cyan-500"><Copy size={15} /> Copy</button>
            </div>
            <pre className="max-h-[560px] overflow-auto rounded-xl bg-black p-5 text-sm leading-relaxed text-cyan-100"><code>{policy.yaml}</code></pre>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-950 p-5">
            <div className="mb-3 flex items-center justify-between gap-4">
              <h2 className="text-xl font-bold text-white">JSON token policy</h2>
              <button onClick={() => navigator.clipboard?.writeText(policy.json)} className="inline-flex items-center gap-2 rounded-lg border border-gray-700 px-3 py-2 text-sm font-semibold text-white transition hover:border-cyan-500"><Copy size={15} /> Copy</button>
            </div>
            <pre className="max-h-[560px] overflow-auto rounded-xl bg-black p-5 text-sm leading-relaxed text-purple-100"><code>{policy.json}</code></pre>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">Capability-token policy checklist</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            Agent tokens need economic constraints, not just authentication. These fields make authority governable before execution and proof exportable after the decision.
          </p>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              [LockKeyhole, 'Scope', 'Bind authority to tenant, agent, task, audience, route, and MCP tool permissions.'],
              [TimerReset, 'Expiry', 'Use short token lifetimes and shorter child-token TTLs for delegated sub-agents.'],
              [ShieldCheck, 'Revocation', 'Revoke on budget exhaustion, loops, parent revocation, policy violation, or kill switch.'],
              [ClipboardList, 'Delegation', 'Require child capabilities to be strict subsets with attenuated budgets and scopes.'],
              [ReceiptText, 'Audit', 'Log token id, parent id, spend context, remaining budget, scope, revocation state, decision, receipt id, and Evidence Pack id.'],
              [KeyRound, 'Economic control', 'Pair identity with budgets so authentication, spend context, and proof capture happen together.'],
            ].map(([Icon, title, body]) => {
              const CardIcon = Icon as typeof KeyRound;
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
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Caveats</p>
          <h2 className="mb-4 text-3xl font-bold text-white">Capability token caveats make agent authority enforceable</h2>
          <p className="text-lg leading-relaxed text-gray-300">
            A capability token is only useful for autonomous agents if its caveats are specific enough to enforce before a request. SatGate treats identity, scope, budget, expiry, delegation, and revocation as one policy object instead of separate logs.
          </p>
        </div>
        <div className="overflow-x-auto rounded-2xl border border-gray-800">
          <table className="min-w-[760px] w-full border-collapse text-left text-sm">
            <thead className="bg-gray-950 text-gray-300">
              <tr>
                <th className="p-4 font-semibold">Caveat</th>
                <th className="p-4 font-semibold">Policy fields</th>
                <th className="p-4 font-semibold">Why it matters</th>
              </tr>
            </thead>
            <tbody>
              {caveatRows.map(([caveat, fields, description]) => (
                <tr key={caveat} className="border-t border-gray-800 bg-black/60 align-top">
                  <td className="p-4 font-bold text-white">{caveat}</td>
                  <td className="p-4 font-mono text-cyan-200">{fields}</td>
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
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Revocation</p>
            <h2 className="mb-4 text-3xl font-bold text-white">Revocation policy should be explicit before the token is minted</h2>
            <p className="text-lg leading-relaxed text-gray-300">
              API key rotation reacts after authority spreads. Revocable capability-token policy lets SatGate stop the next request when budget, loop, delegation, or scope rules fail.
            </p>
          </div>
          <div className="grid gap-5 md:grid-cols-5">
            {revocationRows.map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black p-5">
                <h3 className="mb-2 text-lg font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-b border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">Revocable capability token questions</h2>
          <div className="grid gap-5 md:grid-cols-3">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is a revocable capability token for AI agents?</h3>
              <p className="leading-relaxed text-gray-400">A revocable capability token gives an agent narrowly scoped authority for a tenant, task, tool, budget, and time window. Unlike a static API key, it can expire, be attenuated for sub-agents, and be revoked when policy fails.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What caveats should AI agent capability tokens include?</h3>
              <p className="leading-relaxed text-gray-400">AI agent capability-token caveats should bind subject, tenant, task, audience, scope, route, tool, budget, expiry, delegation limits, revocation triggers, and audit fields so every request can be checked before access.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is delegation attenuation for AI agents?</h3>
              <p className="leading-relaxed text-gray-400">Delegation attenuation means a child or sub-agent receives less authority than the parent: smaller budget, shorter expiry, subset scopes, and stricter revocation rules.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Why are capability tokens better than shared API keys for agents?</h3>
              <p className="leading-relaxed text-gray-400">Shared API keys are broad, long-lived, and hard to revoke safely. Capability tokens bind authority to a specific agent task with budget limits, expiry, delegation rules, receipts, and Evidence Pack fields.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How does SatGate enforce these token policies?</h3>
              <p className="leading-relaxed text-gray-400">SatGate sits in the request path as an economic firewall, checking token scope, budget, delegation, revocation state, and receipt policy at the gateway before forwarding to model, API, MCP, or externally exposed agent access.</p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-purple-900/60 bg-gradient-to-br from-purple-950/30 to-cyan-950/25 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Turn agent authority into Policy-to-Proof evidence.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            Every scoped token decision should produce a receipt that can be exported into an Evidence Pack.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Govern agent authority <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Create Evidence Pack trail
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

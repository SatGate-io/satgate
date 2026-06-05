'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { ArrowRight, Ban, CheckCircle2, GitBranch, KeyRound, Lock, ReceiptText, ShieldCheck, Sparkles } from 'lucide-react';

const lifecycle = [
  {
    id: 'issue',
    label: '1. Issue capability',
    title: 'Issue scoped authority to the parent agent',
    buyer: 'Grant authority for one defined job — not open-ended access.',
    technical: 'Issue a macaroon-style capability with caveats for tenant, task, allowed tools, budget, expiry, and delegation depth.',
    status: 'Allowed',
    evidence: 'parent capability created',
  },
  {
    id: 'delegate',
    label: '2. Delegate',
    title: 'Delegate a child capability for a sub-agent',
    buyer: 'Delegate a narrower capability while keeping the work inside the parent’s approved scope.',
    technical: 'Create a child capability linked to the parent, with depth=1, a child spend cap, subset tool scope, and shorter TTL.',
    status: 'Allowed',
    evidence: 'child capability linked to parent',
  },
  {
    id: 'attenuate',
    label: '3. Attenuate',
    title: 'Shrink what the child can do',
    buyer: 'The child receives narrower authority than the parent: fewer tools, less budget, and a shorter time window.',
    technical: 'Enforce caveats: max_depth=1, child_budget_usd=5, tools=[repo_search], denied_tools=[payment_release], ttl=15m.',
    status: 'Constrained',
    evidence: 'caveats evaluated before the illustrated tool call',
  },
  {
    id: 'revoke',
    label: '4. Revoke',
    title: 'Revoke authority before the next request',
    buyer: 'When the job is complete, behavior falls outside policy, or the parent is revoked, the child is denied on its next gateway-enforced request.',
    technical: 'Set revocation_state=revoked for the child capability; SatGate rejects the next request at the gateway before upstream execution.',
    status: 'Denied',
    evidence: 'revocation decision recorded',
  },
  {
    id: 'prove',
    label: '5. Prove',
    title: 'Export the proof trail',
    buyer: 'Security, finance, and compliance receive the receipt: who authorized what, which limits applied, why the request was blocked, and what spend remained.',
    technical: 'The Evidence Pack includes parent/child lineage, caveats, policy version, decision, remaining budget, revocation state, and receipt IDs.',
    status: 'Proved',
    evidence: 'Evidence Pack ready',
  },
] as const;

type StepId = typeof lifecycle[number]['id'];

const caveats = [
  { label: 'Tenant', value: 'acme-prod', buyer: 'only this customer environment' },
  { label: 'Task', value: 'invoice-reconciliation', buyer: 'only this job' },
  { label: 'Route scope', value: '/invoices/read-only', buyer: 'only this API path and action set' },
  { label: 'Parent budget', value: '$25.00', buyer: 'total spend ceiling' },
  { label: 'Child spend cap', value: '$5.00', buyer: 'child capability cannot exhaust the parent budget' },
  { label: 'Child spend used', value: '$0.28 / $5.00', buyer: 'remaining spend stays visible' },
  { label: 'Delegation depth', value: '1 child level', buyer: 'limits downstream delegation chains' },
  { label: 'Allowed child tools', value: 'repo_search, invoice_read', buyer: 'approved read-only tools' },
  { label: 'Denied tools', value: 'payment_release, vendor_update', buyer: 'high-risk actions are blocked' },
  { label: 'TTL', value: '15 minutes', buyer: 'expires with the task window' },
];

const proofRows = [
  ['receipt_id', 'rcpt_cap_8f41', 'The receipt for this decision.'],
  ['parent_token_id', 'cap_parent_1042', 'Shows who delegated the authority.'],
  ['child_token_id', 'cap_child_77ac', 'Shows which worker tried to act.'],
  ['delegation_depth', '1 / 1', 'Shows this governed child is at the configured delegation limit.'],
  ['decision', 'deny_after_revoke', 'Shows the gateway blocked before execution.'],
  ['remaining_budget_usd', '4.72', 'Shows spend left when revoked.'],
  ['policy_version', 'capability-lifecycle-v3', 'Shows the exact rule set used.'],
  ['evidence_pack_id', 'evp_agent_authority_20260510', 'Bundles the proof for audit.'],
];

const controlTranslations = [
  ['Macaroon', 'A portable credential that carries bounded-authority caveats.'],
  ['Caveat', 'An enforceable limit on budget, tool, route, tenant, time, delegation depth, or revocation state.'],
  ['Attenuation', 'Making a child capability narrower than the parent. Delegation adds caveats without removing or widening parent limits; gateway policy enforces the result.'],
  ['Delegation depth', 'How many handoffs are allowed before the chain must stop.'],
  ['Revocation check', 'A revocation control evaluated before the next model, API, or MCP tool call.'],
  ['Evidence Pack', 'An audit bundle showing who authorized what, which limits were evaluated, and why SatGate allowed or denied the request.'],
];

export default function CapabilityLifecycleDemo() {
  const [step, setStep] = useState<StepId>('issue');
  const active = useMemo(() => lifecycle.find((item) => item.id === step) ?? lifecycle[0], [step]);
  const activeIndex = lifecycle.findIndex((item) => item.id === active.id);

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_0%,rgba(34,211,238,0.20),transparent_32%),radial-gradient(circle_at_80%_10%,rgba(168,85,247,0.18),transparent_34%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-20 md:py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <ShieldCheck size={16} /> Capability lifecycle control
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            Control the full authority lifecycle: issue, delegate, attenuate, revoke, prove
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            SatGate turns macaroons and caveats into auditable controls: scoped authority, child spend caps, delegation depth, next-request revocation, and Evidence Pack audit records before governed access to paid APIs, models, or MCP tools. Observe who is acting, control what can happen, and prove the decision trail.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <a href="#demo" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Walk the lifecycle <ArrowRight size={18} />
            </a>
            <Link href="/govern" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              See SatGate governance
            </Link>
            <Link href="/agent-capability-tokens" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Read capability-token guide
            </Link>
          </div>
        </div>
      </section>

      <section id="demo" className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-8 grid gap-3 md:grid-cols-5">
          {lifecycle.map((item, index) => {
            const complete = index < activeIndex;
            const current = item.id === active.id;
            return (
              <button
                key={item.id}
                onClick={() => setStep(item.id)}
                className={`rounded-2xl border p-4 text-left transition ${current ? 'border-cyan-400 bg-cyan-950/30' : complete ? 'border-green-900/60 bg-green-950/10' : 'border-gray-800 bg-gray-950 hover:border-gray-600'}`}
              >
                <div className="mb-2 flex items-center gap-2 text-sm font-mono text-gray-400">
                  {complete ? <CheckCircle2 size={16} className="text-green-300" /> : current ? <Sparkles size={16} className="text-cyan-300" /> : <Lock size={16} />}
                  Step {index + 1}
                </div>
                <div className="font-bold text-white">{item.label.replace(/^\d\. /, '')}</div>
              </button>
            );
          })}
        </div>

        <div className="grid gap-8 lg:grid-cols-[0.9fr_1.1fr]">
          <div className="rounded-3xl border border-cyan-900/60 bg-cyan-950/10 p-8">
            <div className="mb-5 flex items-center justify-between gap-4">
              <p className="font-mono text-sm uppercase tracking-wide text-cyan-300">{active.label}</p>
              <span className={`rounded-full px-3 py-1 text-sm font-bold ${active.status === 'Denied' ? 'bg-red-950 text-red-200' : active.status === 'Constrained' ? 'bg-yellow-950 text-yellow-200' : 'bg-green-950 text-green-200'}`}>{active.status}</span>
            </div>
            <h2 className="mb-4 text-3xl font-bold text-white">{active.title}</h2>
            <p className="mb-6 text-xl leading-relaxed text-gray-200">{active.buyer}</p>
            <p className="mb-6 leading-relaxed text-gray-400">{active.technical}</p>
            <div className="rounded-2xl border border-gray-800 bg-black p-5">
              <p className="mb-2 text-sm font-mono uppercase tracking-wide text-gray-500">Proof event</p>
              <p className="text-lg font-bold text-white">{active.evidence}</p>
            </div>
          </div>

          <div className="rounded-3xl border border-gray-800 bg-gray-950 p-8">
            <h2 className="mb-5 text-2xl font-bold text-white">Visible delegation state</h2>
            <div className="mb-6 rounded-2xl border border-gray-800 bg-black p-5">
              <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                <div>
                  <div className="mb-1 text-sm text-gray-500">Parent capability</div>
                  <div className="font-mono text-cyan-200">cap_parent_1042</div>
                </div>
                <ArrowRight className="hidden text-gray-600 md:block" />
                <div>
                  <div className="mb-1 text-sm text-gray-500">Child capability</div>
                  <div className="font-mono text-purple-200">cap_child_77ac</div>
                </div>
                <div className="rounded-xl border border-yellow-900/60 bg-yellow-950/20 px-4 py-3 text-sm text-yellow-100">
                  depth: <strong>1 / 1</strong>
                </div>
              </div>
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              {caveats.map((item) => (
                <div key={item.label} className="rounded-xl border border-gray-800 bg-black p-4">
                  <div className="mb-1 text-sm text-gray-500">{item.label}</div>
                  <div className="mb-2 font-bold text-white">{item.value}</div>
                  <div className="text-sm leading-relaxed text-gray-400">{item.buyer}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">Macaroons and caveats, translated into enterprise controls</h2>
          <p className="mb-10 max-w-4xl text-lg leading-relaxed text-gray-400">
            Attenuation becomes bounded authority: what the agent may do, how far it may delegate, how much the child may spend, when authority expires or is revoked, and what proof remains.
          </p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {controlTranslations.map(([term, translation]) => (
              <div key={term} className="rounded-2xl border border-gray-800 bg-black p-6">
                <h3 className="mb-2 text-xl font-bold text-white">{term}</h3>
                <p className="leading-relaxed text-gray-400">{translation}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="grid gap-8 lg:grid-cols-[0.85fr_1.15fr]">
          <div>
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Evidence Pack preview</p>
            <h2 className="mb-4 text-3xl font-bold text-white">Proof after revocation</h2>
            <p className="mb-6 text-lg leading-relaxed text-gray-400">
              A complete lifecycle record does not stop at “token issued.” It captures the child capability’s narrower caveats, visible depth limit, next-request revocation result, and receipt context.
            </p>
            <div className="grid gap-4 sm:grid-cols-2">
              {[
                [KeyRound, 'Scoped authority', 'route, tool, budget, tenant, task'],
                [GitBranch, 'Delegation lineage', 'parent → child with depth visible'],
                [Ban, 'Next-request revocation', 'blocked before the next request'],
                [ReceiptText, 'Audit proof', 'receipt and Evidence Pack IDs'],
              ].map(([Icon, title, body]) => {
                const CardIcon = Icon as typeof KeyRound;
                return (
                  <div key={String(title)} className="rounded-xl border border-gray-800 bg-gray-950 p-5">
                    <CardIcon className="mb-3 text-cyan-300" size={24} />
                    <h3 className="mb-1 font-bold text-white">{String(title)}</h3>
                    <p className="text-sm text-gray-400">{String(body)}</p>
                  </div>
                );
              })}
            </div>
          </div>
          <div className="overflow-hidden rounded-2xl border border-gray-800">
            <div className="grid grid-cols-[0.85fr_1.05fr_1.45fr] gap-3 bg-gray-900/80 px-4 py-3 text-sm font-bold uppercase tracking-wide text-gray-400">
              <div>Field</div><div>Value</div><div>Decision proof</div>
            </div>
            {proofRows.map(([field, value, proof]) => (
              <div key={field} className="grid grid-cols-[0.85fr_1.05fr_1.45fr] gap-3 border-t border-gray-800 px-4 py-4 text-sm">
                <div className="break-words font-mono text-cyan-200">{field}</div>
                <div className="break-words font-mono text-gray-200">{value}</div>
                <div className="break-words text-gray-400">{proof}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 pb-24 pt-4">
        <div className="rounded-3xl border border-purple-900/60 bg-gradient-to-br from-purple-950/30 to-cyan-950/25 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">This is the capability lifecycle enterprises can govern.</h2>
          <p className="mb-8 max-w-4xl text-lg leading-relaxed text-gray-300">
            Issue the capability, delegate a narrower child, express the caveats in plain language, revoke before the next governed request, and export the proof. That is how macaroon-style caveats become enterprise-ready agent authority controls.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/revocable-capability-token-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Generate capability policy <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              See Policy-to-Proof
            </Link>
            <Link href="/accept-satgate-capabilities" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              Accept SatGate capabilities
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

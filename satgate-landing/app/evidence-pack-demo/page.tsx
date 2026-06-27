import type { Metadata } from 'next';
import Link from 'next/link';
import { ArrowRight, BadgeCheck, ClipboardCheck, Download, FileJson, KeyRound, ReceiptText, ShieldCheck, WalletCards } from 'lucide-react';
import evidencePack from '../../public/evidence-packs/sample-evidence-pack.v1.json';

export const metadata: Metadata = {
  title: 'Evidence Pack Demo: Policy-to-Proof Artifact',
  description:
    'Open a sample SatGate Evidence Pack and see who authorized what, under which policy, budget, delegation chain, and paid-rail context.',
  alternates: { canonical: 'https://satgate.io/evidence-pack-demo' },
  keywords: [
    'Evidence Pack demo',
    'Policy-to-Proof artifact',
    'AI agent authorization evidence',
    'agent delegation proof',
    'agent paid rail audit evidence',
  ],
  openGraph: {
    title: 'Evidence Pack Demo: Who Authorized What?',
    description:
      'A visible Policy-to-Proof artifact showing agent identity, policy, budget, delegation, paid-rail context, receipt chain, and export proof.',
    url: 'https://satgate.io/evidence-pack-demo',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Evidence Pack Demo: Who Authorized What?',
    description:
      'Open the SatGate sample Evidence Pack: authority before execution, receipts after every decision.',
  },
};

const pack = evidencePack;
const policy = pack.policy_snapshot;
const budget = pack.budget_snapshot;
const subject = pack.subject;
const receipts = pack.receipts;
const authorityChain = pack.authority_chain;
const rootGrant = authorityChain[0];
const delegatedGrant = authorityChain[1];
const paidEvents = pack.payment_context.events;

if (!rootGrant || !delegatedGrant) {
  throw new Error('Evidence Pack demo fixture must include root and delegated authority grants.');
}

const rootIssuerName = rootGrant.issuer?.display_name ?? rootGrant.issuer?.id ?? 'authorized issuer';
const rootSubjectName = rootGrant.subject.display_name ?? rootGrant.subject.id;
const delegatedParentName = delegatedGrant.parent_subject?.display_name ?? delegatedGrant.parent_subject?.id ?? 'parent agent';
const delegatedSubjectName = delegatedGrant.subject.display_name ?? delegatedGrant.subject.id;

const answerCards = [
  {
    icon: KeyRound,
    question: 'Who authorized it?',
    answer: `${rootIssuerName} minted authority for ${rootSubjectName}.`,
    detail: `Issuer key: ${rootGrant.issuer_kid}`,
  },
  {
    icon: BadgeCheck,
    question: 'Which agent acted?',
    answer: `${subject.display_name} acted as ${subject.id}.`,
    detail: `Tenant: ${pack.tenant.name}`,
  },
  {
    icon: ShieldCheck,
    question: 'Under which policy?',
    answer: `${policy.policy_name} ${policy.policy_version} enforced authority before execution.`,
    detail: `Policy ID: ${policy.policy_id}; mode: ${policy.mode}; digest: ${policy.policy_digest}`,
  },
  {
    icon: WalletCards,
    question: 'Under which budget?',
    answer: `${budget.budget_id} delegated ${budget.delegated_limit} ${budget.currency}; ${budget.spent} spent before denials.`,
    detail: `Cost center: ${budget.cost_center}; exhausted: ${budget.exhausted ? 'yes' : 'no'}`,
  },
  {
    icon: ReceiptText,
    question: 'Was it delegated?',
    answer: `${delegatedParentName} delegated a narrower capability to ${delegatedSubjectName}.`,
    detail: 'Scope, budget, depth, and expiry are all attenuated in the child grant.',
  },
  {
    icon: ClipboardCheck,
    question: 'Was a paid rail involved?',
    answer: `${paidEvents.length} payment-context events are preserved across ${pack.payment_context.internal_rail} and optional paid rails.`,
    detail: `External rails: ${pack.payment_context.external_rails.join(', ')}`,
  },
];

const resultStyles: Record<string, string> = {
  allowed: 'border-emerald-400/30 bg-emerald-400/10 text-emerald-100',
  blocked: 'border-red-400/30 bg-red-400/10 text-red-100',
  issued: 'border-cyan-400/30 bg-cyan-400/10 text-cyan-100',
  attenuated: 'border-purple-400/30 bg-purple-400/10 text-purple-100',
  revoked: 'border-amber-400/30 bg-amber-400/10 text-amber-100',
  evidence_pack_issued: 'border-white/20 bg-white/10 text-white',
};

const jsonLd = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'SatGate Evidence Pack Demo',
  applicationCategory: 'SecurityApplication',
  operatingSystem: 'Web',
  url: 'https://satgate.io/evidence-pack-demo',
  description: metadata.description,
  featureList: [
    'Canonical Evidence Pack v1 schema',
    'Authority chain viewer',
    'Policy and budget snapshot',
    'Delegation proof',
    'Paid-rail context',
    'Receipt-chain export',
  ],
};

export default function EvidencePackDemoPage() {
  return (
    <main className="min-h-screen bg-black text-white">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />

      <section className="relative overflow-hidden border-b border-white/10 px-6 py-24">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(34,211,238,0.2),transparent_30%),radial-gradient(circle_at_85%_10%,rgba(168,85,247,0.18),transparent_35%)]" />
        <div className="relative mx-auto max-w-6xl">
          <p className="mb-4 inline-flex items-center gap-2 rounded-full border border-cyan-300/30 bg-cyan-300/10 px-4 py-2 text-sm font-bold text-cyan-100">
            <FileJson size={16} /> Evidence Pack Demo Artifact
          </p>
          <div className="grid gap-10 lg:grid-cols-[1.05fr_0.95fr] lg:items-center">
            <div>
              <h1 className="text-5xl font-black tracking-tight sm:text-6xl lg:text-7xl">
                Who authorized what?
              </h1>
              <p className="mt-6 max-w-3xl text-xl leading-8 text-gray-300">
                This is the visible SatGate Evidence Pack: a canonical JSON artifact and buyer-readable viewer proving the agent, policy, budget, delegation, paid-rail context, receipts, and export integrity behind one workflow.
              </p>
              <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
                <a href="/evidence-packs/sample-evidence-pack.v1.json" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-3 font-bold text-black transition hover:bg-gray-200">
                  Download canonical JSON <Download size={18} />
                </a>
                <a href="/evidence-packs/evidence-pack.schema.v1.json" className="inline-flex items-center justify-center gap-2 rounded-lg border border-cyan-300/40 bg-cyan-300/10 px-5 py-3 font-bold text-cyan-100 transition hover:border-cyan-200">
                  View schema v1 <ArrowRight size={18} />
                </a>
                <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/20 px-5 py-3 font-bold text-white transition hover:border-white/50">
                  Policy-to-Proof narrative <ArrowRight size={18} />
                </Link>
              </div>
            </div>
            <div className="rounded-3xl border border-cyan-300/20 bg-white/[0.04] p-6 shadow-2xl shadow-cyan-950/30">
              <p className="text-sm font-bold uppercase tracking-[0.2em] text-cyan-200">Executive summary</p>
              <dl className="mt-6 grid gap-4 text-sm">
                <div className="rounded-2xl border border-white/10 bg-black/40 p-4">
                  <dt className="text-gray-500">Evidence Pack</dt>
                  <dd className="mt-1 font-mono text-cyan-100">{pack.evidence_pack_id}</dd>
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/40 p-4">
                  <dt className="text-gray-500">Subject</dt>
                  <dd className="mt-1 font-semibold text-white">{subject.display_name}</dd>
                  <dd className="font-mono text-xs text-gray-400">{subject.id}</dd>
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/40 p-4">
                  <dt className="text-gray-500">Budget</dt>
                  <dd className="mt-1 font-semibold text-white">{budget.spent} / {budget.delegated_limit} {budget.currency} spent</dd>
                  <dd className="text-xs text-gray-400">Remaining shown as {budget.remaining} after the budget denial event.</dd>
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/40 p-4">
                  <dt className="text-gray-500">Verification</dt>
                  <dd className="mt-1 font-semibold text-white">{pack.receipt_chain.receipt_count} linked receipts</dd>
                  <dd className="break-all font-mono text-xs text-gray-400">{pack.chain_root}</dd>
                </div>
              </dl>
            </div>
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-3xl">
            <p className="mb-3 text-sm font-bold uppercase tracking-[0.24em] text-cyan-300">Prospect comprehension check</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">The artifact answers the buyer's core questions.</h2>
          </div>
          <div className="mt-10 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {answerCards.map(({ icon: Icon, question, answer, detail }) => (
              <article key={question} className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
                <Icon className="mb-5 text-cyan-300" size={28} />
                <h3 className="text-xl font-black text-white">{question}</h3>
                <p className="mt-3 text-sm leading-6 text-gray-300">{answer}</p>
                <p className="mt-3 text-xs leading-5 text-gray-500">{detail}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 bg-white/[0.02] px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="grid gap-8 lg:grid-cols-[0.85fr_1.15fr]">
            <div>
              <p className="mb-3 text-sm font-bold uppercase tracking-[0.24em] text-purple-300">Authority chain</p>
              <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Root grant → attenuated worker capability.</h2>
              <p className="mt-5 text-lg leading-8 text-gray-400">
                The child capability is narrower than the parent: less scope, a smaller budget, no customer-data export, and no additional delegation depth.
              </p>
            </div>
            <div className="grid gap-4">
              {authorityChain.map((grant, index) => (
                <article key={grant.grant_id} className="rounded-3xl border border-white/10 bg-black p-6">
                  <div className="flex items-center justify-between gap-4">
                    <div>
                      <p className="text-sm font-bold text-cyan-300">Step {index + 1}: {grant.kind}</p>
                      <h3 className="mt-2 text-2xl font-black text-white">{grant.subject.display_name}</h3>
                    </div>
                    <span className="rounded-full border border-white/10 px-3 py-1 text-xs text-gray-400">depth {grant.delegation_depth_current}/{grant.delegation_depth_max}</span>
                  </div>
                  <div className="mt-5 grid gap-3 md:grid-cols-2">
                    <div className="rounded-2xl bg-white/[0.04] p-4">
                      <p className="text-xs text-gray-500">Effective scope</p>
                      <p className="mt-2 text-sm text-gray-200">{grant.effective_scope.join(', ')}</p>
                    </div>
                    <div className="rounded-2xl bg-white/[0.04] p-4">
                      <p className="text-xs text-gray-500">Budget limit</p>
                      <p className="mt-2 text-sm text-gray-200">{grant.budget_limit} USD</p>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="mb-10 max-w-3xl">
            <p className="mb-3 text-sm font-bold uppercase tracking-[0.24em] text-cyan-300">Receipt timeline</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Every allow, deny, revoke, paid call, and export leaves a receipt.</h2>
          </div>
          <div className="space-y-3">
            {receipts.map((receipt) => (
              <article key={receipt.receipt_id} className="grid gap-4 rounded-2xl border border-white/10 bg-white/[0.03] p-5 md:grid-cols-[0.25fr_0.9fr_1fr_0.8fr] md:items-center">
                <div className="font-mono text-xs text-gray-500">
                  <div>#{receipt.seq} · {receipt.ts.slice(11, 19)}</div>
                  <div className="mt-1 break-all text-cyan-300">{receipt.receipt_id}</div>
                </div>
                <div>
                  <p className="font-bold text-white">{receipt.type}</p>
                  <p className="text-xs text-gray-500">{receipt.action}</p>
                </div>
                <div>
                  <p className="font-mono text-sm text-gray-200">{receipt.resource}</p>
                  <p className="text-xs text-gray-500">{receipt.reason_code ?? receipt.policy_decision_ref}</p>
                </div>
                <div className={`rounded-full border px-3 py-2 text-center text-xs font-bold ${resultStyles[receipt.result] ?? 'border-white/10 bg-white/5 text-gray-200'}`}>
                  {receipt.result}
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 bg-white/[0.02] px-6 py-20">
        <div className="mx-auto grid max-w-6xl gap-8 lg:grid-cols-2">
          <div className="rounded-3xl border border-cyan-300/20 bg-cyan-300/5 p-7">
            <p className="text-sm font-bold uppercase tracking-[0.2em] text-cyan-200">Paid-rail context</p>
            <h2 className="mt-4 text-3xl font-black text-white">Rail-neutral by design.</h2>
            <p className="mt-4 text-sm leading-6 text-gray-300">
              The pack records internal enterprise ledger spend and the x402 paid document-AI call without making the payment rail the product. Payment proves value moved. SatGate proves the worker had authority to move it.
            </p>
            <div className="mt-6 flex flex-wrap gap-2">
              {[pack.payment_context.internal_rail, ...pack.payment_context.external_rails].map((rail) => (
                <span key={rail} className="rounded-full border border-cyan-300/30 bg-black/30 px-3 py-1 text-xs font-bold text-cyan-100">{rail}</span>
              ))}
            </div>
          </div>
          <div className="rounded-3xl border border-purple-300/20 bg-purple-300/5 p-7">
            <p className="text-sm font-bold uppercase tracking-[0.2em] text-purple-200">Verification block</p>
            <h2 className="mt-4 text-3xl font-black text-white">Machine-readable export.</h2>
            <dl className="mt-6 space-y-4 text-sm">
              <div><dt className="text-gray-500">Canonicalization</dt><dd className="font-mono text-gray-200">{pack.receipt_chain.canonicalization}</dd></div>
              <div><dt className="text-gray-500">Hash algorithm</dt><dd className="font-mono text-gray-200">{pack.receipt_chain.hash_algorithm}</dd></div>
              <div><dt className="text-gray-500">Signature</dt><dd className="break-all font-mono text-gray-200">{pack.signature}</dd></div>
              <div><dt className="text-gray-500">Demo caveat</dt><dd className="text-gray-300">{pack.verification.reason}</dd></div>
            </dl>
          </div>
        </div>
      </section>

      <section className="px-6 py-20">
        <div className="mx-auto max-w-6xl rounded-3xl border border-emerald-300/20 bg-emerald-300/5 p-8">
          <div className="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <p className="text-sm font-bold uppercase tracking-[0.2em] text-emerald-200">Buyer takeaway</p>
              <h2 className="mt-3 max-w-3xl text-3xl font-black text-white">A prospect can see who authorized what, what failed, what spent, what paid rail was involved, and what proof was exported.</h2>
            </div>
            <div className="flex shrink-0 flex-col gap-3">
              <a href="/evidence-packs/sample-evidence-pack.v1.json" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-3 font-bold text-black transition hover:bg-gray-200">Download JSON <Download size={18} /></a>
              <a href="/evidence-packs/evidence-pack.schema.v1.json" className="inline-flex items-center justify-center gap-2 rounded-lg border border-emerald-300/40 px-5 py-3 font-bold text-emerald-100 transition hover:border-emerald-200">Schema v1 <ArrowRight size={18} /></a>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}

import type { Metadata } from 'next';
import Link from 'next/link';
import { ArrowRight, BadgeCheck, ClipboardCheck, Download, FileJson, KeyRound, ReceiptText, ShieldCheck, WalletCards } from 'lucide-react';
import evidencePack from '../../public/evidence-packs/sample-evidence-pack.v1.json';

export const metadata: Metadata = {
  title: 'Sample signed receipt: Who Allowed What?',
  description:
    'Open a sample SatGate signed receipt (Evidence Pack) and see who allowed what, under which rules and budget, who handed work to whom, and what was paid.',
  alternates: { canonical: 'https://satgate.io/evidence-pack-demo' },
  keywords: [
    'signed receipt demo',
    'signed receipt',
    'AI agent authorization evidence',
    'agent delegation proof',
    'agent payment method audit evidence',
  ],
  openGraph: {
    title: 'Sample signed receipt: Who Allowed What?',
    description:
      'One agent workflow, laid out: which agent acted, the rules and budget it ran under, the handoff to a sub-agent, payments, and every receipt.',
    url: 'https://satgate.io/evidence-pack-demo',
    type: 'website',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Sample signed receipt: Who Allowed What?',
    description:
      'Open a sample of SatGate signed receipts for budget spends and for refusals over a budget, a missing payment or a tool permission.',
  },
};

const pack = evidencePack;
const budget = pack.budget_snapshot;
const receipts = pack.receipts;
const authorityChain = pack.authority_chain;
const rootGrant = authorityChain[0];
const delegatedGrant = authorityChain[1];

if (!rootGrant || !delegatedGrant) {
  throw new Error('Receipt demo fixture must include a root grant and a delegated grant.');
}

// What a real signed receipt carries today (public/evidence/mcp-budget-refusal-20261005.json).
const answerCards = [
  {
    icon: ReceiptText,
    question: 'Which tool or route?',
    answer: 'The tool or route the agent called.',
    detail: 'For example, a tool name on an MCP server or a path on your API.',
  },
  {
    icon: ShieldCheck,
    question: 'What did SatGate decide?',
    answer: 'Whether SatGate allowed the call or refused it.',
    detail: 'With the time SatGate decided.',
  },
  {
    icon: ClipboardCheck,
    question: 'Why was it refused?',
    answer: 'A refusal code, such as budget_exhausted.',
    detail: 'Other codes cover a missing payment or a tool the agent may not use.',
  },
  {
    icon: WalletCards,
    question: 'What did it cost?',
    answer: 'The amount checked and the budget left.',
    detail: 'On calls that spend from a budget.',
  },
  {
    icon: KeyRound,
    question: 'Which token?',
    answer: 'A hash of the token used, on calls that spend from a budget.',
    detail: 'The raw token is blanked out.',
  },
  {
    icon: BadgeCheck,
    question: 'Who signed it?',
    answer: 'SatGate, with a key anyone can check against SatGate’s published keys.',
    detail: 'The agent never writes or edits its own record.',
  },
];

const resultStyles: Record<string, string> = {
  allowed: 'border-emerald-400/30 bg-emerald-400/10 text-emerald-100',
  blocked: 'border-red-400/30 bg-red-400/10 text-red-100',
  denied: 'border-red-400/30 bg-red-400/10 text-red-100',
  issued: 'border-cyan-400/30 bg-cyan-400/10 text-cyan-100',
  attenuated: 'border-purple-400/30 bg-purple-400/10 text-purple-100',
  revoked: 'border-amber-400/30 bg-amber-400/10 text-amber-100',
  evidence_pack_issued: 'border-white/20 bg-white/10 text-white',
};

const jsonLd = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'SatGate signed receipt demo',
  applicationCategory: 'SecurityApplication',
  operatingSystem: 'Web',
  url: 'https://satgate.io/evidence-pack-demo',
  description: metadata.description,
  featureList: [
    'Canonical signed receipt (Evidence Pack) v1 schema',
    'Token chain viewer',
    'Policy and budget snapshot',
    'Sub-agent handoffs',
    'Payment records',
    'Receipt export',
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
            <FileJson size={16} /> Sample signed receipt
          </p>
          <div className="grid gap-10 lg:grid-cols-[1.05fr_0.95fr] lg:items-center">
            <div>
              <h1 className="text-5xl font-black tracking-tight sm:text-6xl lg:text-7xl">
                Who allowed what?
              </h1>
              <p className="mt-6 max-w-3xl text-xl leading-8 text-gray-300">
                This sample shows a planned audit view of one workflow. Today a real Evidence Pack holds one signed receipt, which shows the tool or route, the decision and a refusal code.
              </p>
              <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
                <a href="/evidence-packs/sample-evidence-pack.v1.json" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-3 font-bold text-black transition hover:bg-gray-200">
                  Download the JSON <Download size={18} />
                </a>
                <a href="/evidence-packs/evidence-pack.schema.v1.json" className="inline-flex items-center justify-center gap-2 rounded-lg border border-cyan-300/40 bg-cyan-300/10 px-5 py-3 font-bold text-cyan-100 transition hover:border-cyan-200">
                  See the schema <ArrowRight size={18} />
                </a>
                <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/20 px-5 py-3 font-bold text-white transition hover:border-white/50">
                  How receipts work <ArrowRight size={18} />
                </Link>
              </div>
            </div>
            <div className="rounded-3xl border border-cyan-300/20 bg-white/[0.04] p-6 shadow-2xl shadow-cyan-950/30">
              <p className="text-sm font-bold uppercase tracking-[0.2em] text-cyan-200">Summary</p>
              <dl className="mt-6 grid gap-4 text-sm">
                <div className="rounded-2xl border border-white/10 bg-black/40 p-4">
                  <dt className="text-gray-500">Receipt</dt>
                  <dd className="mt-1 font-mono text-cyan-100">{pack.evidence_pack_id}</dd>
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/40 p-4">
                  <dt className="text-gray-500">Budget</dt>
                  <dd className="mt-1 font-semibold text-white">{budget.spent} / {budget.delegated_limit} {budget.currency} spent</dd>
                  <dd className="text-xs text-gray-400">{budget.remaining} left after the budget ran out and requests were refused.</dd>
                </div>
              </dl>
            </div>
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-3xl">
            <p className="mb-3 text-sm font-bold uppercase tracking-[0.24em] text-cyan-300">What a receipt tells you</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">What a real receipt shows.</h2>
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
              <p className="mb-3 text-sm font-bold uppercase tracking-[0.24em] text-purple-300">Token chain</p>
              <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Main agent → sub-agent with a narrower token.</h2>
              <p className="mt-5 text-lg leading-8 text-gray-400">
                The sub-agent&apos;s token allows less than the main agent&apos;s: fewer permissions, a smaller budget, no customer-data export and no further handoffs.
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
                    <span className="rounded-full border border-white/10 px-3 py-1 text-xs text-gray-400">handoff {grant.delegation_depth_current} of {grant.delegation_depth_max}</span>
                  </div>
                  <div className="mt-5 grid gap-3 md:grid-cols-2">
                    <div className="rounded-2xl bg-white/[0.04] p-4">
                      <p className="text-xs text-gray-500">Allowed</p>
                      <p className="mt-2 text-sm text-gray-200">{grant.effective_scope.join(', ')}</p>
                    </div>
                    <div className="rounded-2xl bg-white/[0.04] p-4">
                      <p className="text-xs text-gray-500">Budget limit</p>
                      <p className="mt-2 text-sm text-gray-200">{grant.budget_limit} credits</p>
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
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Signed receipts for budget spends and for refusals over a budget, a missing payment or a tool permission.</h2>
          </div>
          <div className="space-y-3">
            {receipts.filter((receipt) => receipt.type === 'spend' || receipt.type === 'denial').map((receipt) => (
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
            <p className="text-sm font-bold uppercase tracking-[0.2em] text-cyan-200">Payments</p>
            <h2 className="mt-4 text-3xl font-black text-white">Not tied to one payment method.</h2>
            <p className="mt-4 text-sm leading-6 text-gray-300">
              This sample records spending from an internal budget and one paid call to a document-AI service over x402. The payment shows money moved. The receipt shows the sub-agent was allowed to spend it.
            </p>
          </div>
          <div className="rounded-3xl border border-purple-300/20 bg-purple-300/5 p-7">
            <p className="text-sm font-bold uppercase tracking-[0.2em] text-purple-200">Signature</p>
            <h2 className="mt-4 text-3xl font-black text-white">How the export is signed.</h2>
            <dl className="mt-6 space-y-4 text-sm">
              <div><dt className="text-gray-500">JSON format signed</dt><dd className="font-mono text-gray-200">{pack.receipt_chain.canonicalization}</dd></div>
              <div><dt className="text-gray-500">Hash algorithm</dt><dd className="font-mono text-gray-200">{pack.receipt_chain.hash_algorithm}</dd></div>
              <div><dt className="text-gray-500">Signature</dt><dd className="break-all font-mono text-gray-200">{pack.signature}</dd></div>
              <div><dt className="text-gray-500">Note</dt><dd className="text-gray-300">This sample uses placeholder hashes and a placeholder signature, so it won&apos;t pass the verifier. Real exports are signed. <Link href="/verify-evidence-pack" className="text-purple-200 underline underline-offset-4">Check a real one</Link>.</dd></div>
            </dl>
          </div>
        </div>
      </section>

      <section className="px-6 py-20">
        <div className="mx-auto max-w-6xl rounded-3xl border border-emerald-300/20 bg-emerald-300/5 p-8">
          <div className="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <p className="text-sm font-bold uppercase tracking-[0.2em] text-emerald-200">In short</p>
              <h2 className="mt-3 max-w-3xl text-3xl font-black text-white">A real Evidence Pack holds one signed receipt for one decision. Check a real one on the verify page.</h2>
            </div>
            <div className="flex shrink-0 flex-col gap-3">
              <Link href="/verify-evidence-pack" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-3 font-bold text-black transition hover:bg-gray-200">Check a real receipt <ArrowRight size={18} /></Link>
              <a href="/evidence-packs/sample-evidence-pack.v1.json" className="inline-flex items-center justify-center gap-2 rounded-lg border border-emerald-300/40 px-5 py-3 font-bold text-emerald-100 transition hover:border-emerald-200">Download JSON <Download size={18} /></a>
              <a href="/evidence-packs/evidence-pack.schema.v1.json" className="inline-flex items-center justify-center gap-2 rounded-lg border border-emerald-300/40 px-5 py-3 font-bold text-emerald-100 transition hover:border-emerald-200">Schema <ArrowRight size={18} /></a>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}

import type { Metadata } from "next";
import Link from "next/link";
import {
  ArrowRight,
  BadgeCheck,
  Ban,
  CheckCircle2,
  CircleDollarSign,
  FileText,
  KeyRound,
  ShieldCheck,
  TimerReset,
} from "lucide-react";

export const metadata: Metadata = {
  title: "SatGate Policy-to-Proof | Evidence for Enterprise Agent Authority",
  description:
    "SatGate turns AI agent policy decisions into evidence: mint receipts, delegation chains, spend ledgers, denial reasons, revocation proof, and exportable Evidence Packs.",
  keywords: [
    "policy-to-proof",
    "AI agent audit trail",
    "agent authority evidence",
    "agent control plane evidence",
    "tamper-evident audit trail",
    "AI agent revocation proof",
    "AI agent spend ledger",
    "macaroon delegation audit",
  ],
  alternates: {
    canonical: "https://satgate.io/policy-to-proof",
  },
  openGraph: {
    title: "SatGate Policy-to-Proof",
    description:
      "Every grant, spend event, denial, and revocation leaves evidence your CISO, finance team, and auditor can trust.",
    url: "https://satgate.io/policy-to-proof",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate Policy-to-Proof",
    description:
      "Run agents without permanent credentials, unlimited spend, or unobservable authority — and export the proof.",
  },
};

const evidenceQuestions = [
  {
    question: "Who authorized this?",
    artifact: "Mint receipt",
    body: "Identity claim, policy match, issuer, timestamp, and the capability minted for the agent.",
  },
  {
    question: "Which agent got access?",
    artifact: "Capability token + delegation chain",
    body: "The root capability, parent agent, worker agent, and every attenuation in the handoff path.",
  },
  {
    question: "What exactly could it do?",
    artifact: "Scope + caveat record",
    body: "Routes, tools, tenant boundary, expiry, budget, delegation depth, and revocation status at issue time.",
  },
  {
    question: "What did it spend?",
    artifact: "Per-token spend ledger",
    body: "Request-path and MCP-tool attribution by agent, token, tenant, route, amount, and policy mode.",
  },
  {
    question: "What was denied?",
    artifact: "Denial receipt",
    body: "A policy, budget, scope, tenant, or revocation reason code attached to the blocked call.",
  },
  {
    question: "Can we prove revocation worked?",
    artifact: "Revocation receipt + post-revoke denial",
    body: "The revocation event and the first failed call after access ended, tied to the same authority chain.",
  },
];

const demoSteps = [
  ["Mint", "Agent presents identity and receives a scoped macaroon capability with caveats."],
  ["Delegate", "The parent agent hands narrower authority to a worker; the chain is preserved."],
  ["Spend", "The worker calls an API or MCP tool under budget; the ledger updates by token and path."],
  ["Deny", "The next call hits a cap, scope, or tenant boundary and returns a reason-coded receipt."],
  ["Revoke", "Authority is killed without rotating every upstream provider key."],
  ["Export", "The lifecycle becomes an Evidence Pack a CISO, auditor, or incident reviewer can read."],
];

const evidencePack = {
  evidence_pack_id: "ep_demo_2026_05_09_001",
  subject: "worker-agent:invoice-reconciler",
  authority_chain: ["root:finance-automation", "delegate:invoice-worker"],
  receipts: [
    { type: "mint", result: "issued", caveats: ["tenant=acme", "budget_usd=25", "delegation_depth<=1"] },
    { type: "spend", result: "allowed", route: "/v1/invoices/search", amount_usd: "0.18" },
    { type: "denial", result: "blocked", reason: "budget_exhausted", route: "/v1/invoices/export" },
    { type: "revocation", result: "revoked", revoked_by: "security-admin" },
    { type: "post_revoke_denial", result: "blocked", reason: "capability_revoked" },
  ],
};

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebPage",
      name: "SatGate Policy-to-Proof",
      url: "https://satgate.io/policy-to-proof",
      description: metadata.description,
      datePublished: "2026-05-09",
      dateModified: "2026-05-09",
      isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
      about: [
        { "@type": "Thing", name: "AI agent authority evidence" },
        { "@type": "Thing", name: "tamper-evident audit trail" },
        { "@type": "Thing", name: "agent control plane" },
        { "@type": "Thing", name: "macaroon delegation chain" },
      ],
    },
    {
      "@type": "ItemList",
      name: "Policy-to-Proof evidence questions",
      itemListElement: evidenceQuestions.map((item, index) => ({
        "@type": "ListItem",
        position: index + 1,
        name: item.question,
        description: `${item.artifact}: ${item.body}`,
      })),
    },
  ],
};

export default function PolicyToProofPage() {
  return (
    <main className="min-h-screen bg-black text-white">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />

      <section className="relative overflow-hidden border-b border-white/10 px-6 py-24 sm:py-32">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(56,189,248,0.20),transparent_35%),radial-gradient(circle_at_bottom_right,rgba(168,85,247,0.18),transparent_40%)]" />
        <div className="relative mx-auto max-w-6xl">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-cyan-400/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-cyan-200">
            <FileText size={14} /> Policy-to-Proof
          </div>
          <div className="grid gap-10 lg:grid-cols-[1.05fr_0.95fr] lg:items-center">
            <div>
              <h1 className="max-w-4xl text-5xl font-black tracking-tight sm:text-6xl lg:text-7xl">
                Every agent action leaves a receipt.
              </h1>
              <p className="mt-6 max-w-3xl text-xl leading-8 text-gray-300">
                SatGate turns policy decisions into evidence. Mint, delegate, spend, deny, revoke — every step produces a tamper-evident artifact your security, finance, and audit teams can trust.
              </p>
              <p className="mt-5 max-w-3xl text-lg leading-8 text-gray-400">
                Run agents without permanent credentials, unlimited spend, or unobservable authority. Then export the proof when your CISO, auditor, board, or incident reviewer asks what happened.
              </p>
              <div className="mt-8 flex flex-col gap-3 sm:flex-row">
                <Link
                  href="/agent-control-plane"
                  className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-3 font-bold text-black transition hover:bg-gray-200"
                >
                  See the agent control plane <ArrowRight size={18} />
                </Link>
                <Link
                  href="https://cloud.satgate.io/cloud/login"
                  className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/20 px-5 py-3 font-bold text-white transition hover:border-cyan-300/60"
                >
                  Open SatGate Cloud
                </Link>
              </div>
            </div>

            <div className="rounded-3xl border border-cyan-300/20 bg-white/[0.03] p-5 shadow-2xl shadow-cyan-950/30">
              <div className="mb-4 flex items-center justify-between border-b border-white/10 pb-3">
                <div>
                  <p className="text-sm font-semibold text-cyan-200">Evidence Pack</p>
                  <p className="text-xs text-gray-500">Lifecycle export preview</p>
                </div>
                <BadgeCheck className="text-emerald-300" />
              </div>
              <pre className="overflow-x-auto rounded-2xl bg-black/70 p-4 text-xs leading-6 text-cyan-50">
                {JSON.stringify(evidencePack, null, 2)}
              </pre>
            </div>
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-3xl">
            <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-purple-300">The narrative shift</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Enforcement is table stakes. Proof is the product.</h2>
            <p className="mt-5 text-lg leading-8 text-gray-400">
              Gateways eventually claim they can block calls. SatGate goes further: every grant, spend event, denial, and revocation becomes a structured receipt. The audit trail is not a logging afterthought; it is generated by the same authority path that enforces the decision.
            </p>
          </div>
          <div className="mt-10 grid gap-4 md:grid-cols-4">
            {[
              [KeyRound, "Grant evidence", "Who minted authority and under which caveats."],
              [CircleDollarSign, "Spend ledger", "Which agent, route, tool, and token created cost."],
              [Ban, "Denial reason", "Why the call stopped before risk or spend escaped."],
              [TimerReset, "Revocation proof", "When authority ended and the next call failed."],
            ].map(([Icon, title, body]) => (
              <div key={String(title)} className="rounded-2xl border border-white/10 bg-white/[0.03] p-5">
                <Icon className="mb-4 text-cyan-300" size={24} />
                <h3 className="font-bold text-white">{String(title)}</h3>
                <p className="mt-2 text-sm leading-6 text-gray-400">{String(body)}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 bg-white/[0.02] px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="mb-10 flex flex-col justify-between gap-4 lg:flex-row lg:items-end">
            <div className="max-w-3xl">
              <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-cyan-300">Six-question evidence framework</p>
              <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Answer the questions buyers ask after an agent acts.</h2>
            </div>
            <p className="max-w-md text-sm leading-6 text-gray-500">
              The Evidence Pack bundles these artifacts into one export instead of sending teams on a forensics project across logs, invoices, and gateway dashboards.
            </p>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {evidenceQuestions.map((item) => (
              <article key={item.question} className="rounded-2xl border border-white/10 bg-black p-6">
                <p className="text-sm font-bold text-cyan-300">{item.question}</p>
                <h3 className="mt-3 text-xl font-black text-white">{item.artifact}</h3>
                <p className="mt-3 text-sm leading-6 text-gray-400">{item.body}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-3xl">
            <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-purple-300">5-minute demo path</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Mint → Delegate → Spend → Deny → Revoke → Export.</h2>
            <p className="mt-5 text-lg leading-8 text-gray-400">
              The demo should end on the exported Evidence Pack. That is the buyer moment: one artifact proving authority, spend, denial, and revocation across the agent lifecycle.
            </p>
          </div>

          <div className="mt-12 grid gap-4 lg:grid-cols-6">
            {demoSteps.map(([title, body], index) => (
              <div key={title} className="relative rounded-2xl border border-white/10 bg-white/[0.03] p-5">
                <div className="mb-5 flex h-9 w-9 items-center justify-center rounded-full bg-cyan-400 text-sm font-black text-black">
                  {index + 1}
                </div>
                <h3 className="text-lg font-black text-white">{title}</h3>
                <p className="mt-3 text-sm leading-6 text-gray-400">{body}</p>
              </div>
            ))}
          </div>

          <div className="mt-12 rounded-3xl border border-emerald-400/20 bg-emerald-400/5 p-8">
            <div className="flex flex-col gap-5 md:flex-row md:items-start md:justify-between">
              <div className="max-w-3xl">
                <div className="mb-3 flex items-center gap-2 text-emerald-300">
                  <CheckCircle2 size={20} />
                  <span className="text-sm font-bold uppercase tracking-[0.2em]">Buyer promise</span>
                </div>
                <p className="text-2xl font-black leading-snug text-white">
                  SatGate lets enterprises delegate authority to AI agents without permanent credentials, unlimited spend, or unobservable authority — and proves every step.
                </p>
              </div>
              <ShieldCheck className="hidden shrink-0 text-emerald-300 md:block" size={56} />
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}

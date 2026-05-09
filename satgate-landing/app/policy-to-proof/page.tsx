import type { Metadata } from "next";
import Link from "next/link";
import {
  ArrowRight,
  BadgeCheck,
  CheckCircle2,
  ClipboardCheck,
  Download,
  FileText,
  PlayCircle,
  ShieldAlert,
  ShieldCheck,
  WalletCards,
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
    body: "Identity claim, policy match, issuer, timestamp, and the capability minted for the invoice-reconciler worker.",
  },
  {
    question: "Which agent got access?",
    artifact: "Capability token + delegation chain",
    body: "The root capability, parent agent, invoice-reconciler worker, and every attenuation in the handoff path.",
  },
  {
    question: "What exactly could it do?",
    artifact: "Scope + caveat record",
    body: "Routes, tools, tenant boundary, expiry, budget, delegation depth, and revocation status at issue time.",
  },
  {
    question: "What did it spend?",
    artifact: "Per-token spend ledger",
    body: "Request-path and MCP-tool attribution by worker, token, tenant, route, amount, and policy mode.",
  },
  {
    question: "What was denied?",
    artifact: "Denial receipt",
    body: "A policy, budget, scope, tenant, or revocation reason code attached to the blocked invoice call.",
  },
  {
    question: "Can we prove revocation worked?",
    artifact: "Revocation receipt + post-revoke denial",
    body: "The revocation event and the first failed invoice-reconciler call after access ended, tied to the same authority chain.",
  },
];

const demoSteps = [
  ["Mint", "invoice-reconciler gets a scoped macaroon capability with tenant, budget, route, and expiry caveats."],
  ["Delegate", "The parent finance agent hands narrower authority to the worker; the chain is preserved."],
  ["Spend", "The worker calls invoice APIs or MCP tools under budget; the ledger updates by token and path."],
  ["Deny", "Export and over-budget calls return reason-coded receipts before data or spend escapes."],
  ["Revoke", "Security kills the worker capability without rotating every upstream provider key."],
  ["Export", "The lifecycle becomes one Evidence Pack a CISO, auditor, or incident reviewer can read."],
];

const standardsMappings = [
  ["Mint receipt", "SOC 2 CC6.1", "Logical access provisioning tied to identity, policy, issuer, and timestamp."],
  ["Mint receipt", "ISO 27001 A.9.2.1", "User registration and de-registration evidence for agent authority issuance."],
  ["Delegation chain", "SOC 2 CC6.3 / NIST AC-3", "Least-privilege attenuation across parent and worker authority."],
  ["Revocation receipt", "SOC 2 CC6.2/CC6.3 / NIST AC-2(3)", "Deprovisioning event plus first post-revoke denial trail."],
  ["Spend ledger", "SOC 2 CC1.4 / FinOps attribution", "Governance evidence for who created spend, on which route/tool, under which token."],
];

const personas = [
  {
    title: "Auditor",
    icon: ClipboardCheck,
    quote: "I get the authority trail in one export instead of opening a forensics ticket.",
  },
  {
    title: "CISO",
    icon: ShieldAlert,
    quote: "Revocation isn’t a promise — it’s a signed receipt followed by a denied call.",
  },
  {
    title: "FinOps lead",
    icon: WalletCards,
    quote: "Spend is attributed to the agent, the token, and the route — not a shared API key.",
  },
];

const evidencePack = {
  evidence_pack_id: "ep_demo_2026_05_09_001",
  issued_at: "2026-05-09T14:22:31Z",
  expires_at: "2026-05-16T14:22:31Z",
  tenant: "acme-finance",
  subject: "worker-agent:invoice-reconciler",
  authority_chain: [
    {
      kind: "root_grant",
      subject: "dean-agent:finance-automation",
      issuer_kid: "satgate-mint-2026-05",
      caveats: ["tenant=acme-finance", "budget_usd<=25", "delegation_depth<=1"],
      receipt_hash: "sha256:7a2ca1b8d5e0a0d7c9545c7f8e6d03d12761b687a7a1f27c0dd7ed2e643a01b5",
    },
    {
      kind: "delegation",
      subject: "worker-agent:invoice-reconciler",
      parent: "dean-agent:finance-automation",
      caveats: ["budget_usd<=3", "no_customer_data_export", "route_prefix=/v1/invoices"],
      receipt_hash: "sha256:95f1c3df9d1f8d7a0e5b60b850f4b6013dd2f0e18496f9e5a8c0319fd51382bf",
    },
  ],
  receipts: [
    { type: "mint", ts: "2026-05-09T14:22:31Z", issuer_kid: "satgate-mint-2026-05", result: "issued", caveats: ["tenant=acme-finance", "budget_usd<=25", "delegation_depth<=1"], receipt_hash: "sha256:7a2c..." },
    { type: "delegation", ts: "2026-05-09T14:23:04Z", result: "attenuated", receipt_hash: "sha256:95f1..." },
    { type: "spend", ts: "2026-05-09T14:23:18Z", route: "/v1/invoices/search", amount_usd: "0.18", result: "allowed", receipt_hash: "sha256:01d8..." },
    { type: "spend", ts: "2026-05-09T14:24:02Z", route: "/v1/invoices/compare", amount_usd: "0.42", result: "allowed", receipt_hash: "sha256:a923..." },
    { type: "spend", ts: "2026-05-09T14:24:44Z", route: "/v1/invoices/reconcile", amount_usd: "1.12", result: "allowed", receipt_hash: "sha256:deb5..." },
    { type: "denial", ts: "2026-05-09T14:25:08Z", reason_code: "scope_violation:no_customer_data_export", result: "blocked", receipt_hash: "sha256:9b0f..." },
    { type: "denial", ts: "2026-05-09T14:25:33Z", reason_code: "budget_exhausted", result: "blocked", receipt_hash: "sha256:c3f6..." },
    { type: "revocation", ts: "2026-05-09T14:26:11Z", revoked_by: "security-admin", result: "revoked", receipt_hash: "sha256:37d1..." },
    { type: "post_revoke_denial", ts: "2026-05-09T14:26:16Z", reason_code: "capability_revoked", result: "blocked", receipt_hash: "sha256:8b95..." },
    { type: "export", ts: "2026-05-09T14:26:31Z", result: "evidence_pack_issued", receipt_hash: "sha256:e1b3..." },
  ],
  chain_root: "sha256:f04ed8430b11c8975cc5ef35919ee078fc4cb166cd8d611ed0d94b7da69df09d",
  signature: "ed25519:REDACTED_DEMO_SAMPLE_DO_NOT_VERIFY",
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
              <p className="mb-5 max-w-2xl rounded-2xl border border-white/10 bg-white/[0.04] p-4 text-sm leading-6 text-gray-300">
                Monday morning, your auditor asks who authorized the agent that tried to export customer data on Friday night. This is what they get.
              </p>
              <h1 className="max-w-4xl text-5xl font-black tracking-tight sm:text-6xl lg:text-7xl">
                Every agent action leaves a receipt.
              </h1>
              <p className="mt-6 max-w-3xl text-xl leading-8 text-gray-300">
                SatGate turns policy decisions into evidence. Mint, delegate, spend, deny, revoke — every step produces a tamper-evident artifact your security, finance, and audit teams can trust.
              </p>
              <p className="mt-5 max-w-3xl text-lg leading-8 text-gray-400">
                Run agents without permanent credentials, unlimited spend, or unobservable authority. Then export the proof when your CISO, auditor, board, or incident reviewer asks what happened.
              </p>
              <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
                <a
                  href="/evidence-packs/sample-evidence-pack.pdf"
                  className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-3 font-bold text-black transition hover:bg-gray-200"
                >
                  Download sample Evidence Pack <Download size={18} />
                </a>
                <a
                  href="/evidence-packs/sample-evidence-pack.json"
                  className="inline-flex items-center justify-center gap-2 rounded-lg border border-cyan-300/40 bg-cyan-300/10 px-5 py-3 font-bold text-cyan-100 transition hover:border-cyan-200"
                >
                  View JSON export <ArrowRight size={18} />
                </a>
              </div>
            </div>

            <div className="rounded-3xl border border-cyan-300/20 bg-white/[0.03] p-5 shadow-2xl shadow-cyan-950/30">
              <div className="mb-4 flex items-center justify-between border-b border-white/10 pb-3">
                <div>
                  <p className="text-sm font-semibold text-cyan-200">Evidence Pack</p>
                  <p className="text-xs text-gray-500">Signed lifecycle export preview</p>
                </div>
                <BadgeCheck className="text-emerald-300" />
              </div>
              <pre className="max-h-[560px] overflow-x-auto rounded-2xl bg-black/70 p-4 text-xs leading-6 text-cyan-50">
                {JSON.stringify(evidencePack, null, 2)}
              </pre>
              <p className="mt-3 text-xs leading-5 text-gray-500">
                Inline hashes are shortened for readability. Full hashes, ed25519 signature, and verification block are in the downloadable JSON.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-3xl">
            <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-purple-300">The contrast</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Logs tell you something happened. Evidence proves who had authority.</h2>
          </div>
          <div className="mt-10 grid gap-5 lg:grid-cols-2">
            <div className="rounded-3xl border border-red-400/20 bg-red-400/5 p-7">
              <p className="text-sm font-bold uppercase tracking-[0.22em] text-red-200">Other gateways</p>
              <h3 className="mt-4 text-2xl font-black text-white">200/402 status codes in a log.</h3>
              <p className="mt-4 text-base leading-7 text-gray-400">
                You reconstruct authority from six systems, three hours of joins, shared API keys, cloud invoices, and dashboard screenshots — with no chain-of-custody.
              </p>
            </div>
            <div className="rounded-3xl border border-emerald-400/20 bg-emerald-400/5 p-7">
              <p className="text-sm font-bold uppercase tracking-[0.22em] text-emerald-200">SatGate</p>
              <h3 className="mt-4 text-2xl font-black text-white">Signed mint receipt, attenuation chain, spend ledger, denial reason, revocation proof — one export.</h3>
              <p className="mt-4 text-base leading-7 text-gray-400">
                The audit trail is not a logging afterthought. It is generated by the same authority path that enforces the decision.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 bg-white/[0.02] px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="mb-10 flex flex-col justify-between gap-4 lg:flex-row lg:items-end">
            <div className="max-w-3xl">
              <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-cyan-300">Six-question evidence framework</p>
              <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Answer the questions buyers ask after invoice-reconciler acts.</h2>
            </div>
            <p className="max-w-md text-sm leading-6 text-gray-500">
              The Evidence Pack bundles these artifacts into one export instead of sending teams on a forensics project across logs, invoices, and gateway dashboards. Authority-chain entries preserve lineage; matching receipts preserve the event log, so auditors can verify both structure and sequence.
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

      <section className="border-b border-white/10 px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-3xl">
            <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-purple-300">Maps to audit controls</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Audit-fluent, not just audit-flavored.</h2>
            <p className="mt-5 text-lg leading-8 text-gray-400">
              The Evidence Pack gives security and audit teams a starting control map instead of making them translate raw gateway logs themselves.
            </p>
          </div>
          <div className="mt-10 grid gap-4 md:grid-cols-2">
            {standardsMappings.map(([artifact, control, body]) => (
              <div key={artifact} className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
                <p className="text-sm font-bold text-cyan-300">{artifact}</p>
                <h3 className="mt-2 text-xl font-black text-white">{control}</h3>
                <p className="mt-3 text-sm leading-6 text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 bg-white/[0.02] px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-3xl">
            <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-cyan-300">Who receives the pack</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">One export, three enterprise conversations.</h2>
          </div>
          <div className="mt-10 grid gap-4 md:grid-cols-3">
            {personas.map(({ title, quote, icon: Icon }) => (
              <figure key={title} className="rounded-2xl border border-white/10 bg-black p-6">
                <div className="mb-5 flex h-11 w-11 items-center justify-center rounded-2xl bg-purple-400/10 text-purple-200">
                  <Icon size={22} />
                </div>
                <figcaption className="mb-3 text-sm font-bold uppercase tracking-[0.2em] text-purple-300">
                  What the {title} gets
                </figcaption>
                <blockquote className="text-lg font-semibold leading-8 text-white">“{quote}”</blockquote>
              </figure>
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
              The demo ends on the exported Evidence Pack. That is the buyer moment: one artifact proving authority, spend, denial, and revocation across the invoice-reconciler lifecycle. Even producing the Evidence Pack is itself an auditable event.
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

          <div className="mt-12 grid gap-6 lg:grid-cols-[1.15fr_0.85fr] lg:items-center">
            <div className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-3 shadow-2xl shadow-black/60">
              <video
                className="aspect-video w-full rounded-[1.35rem] bg-black object-cover"
                controls
                preload="metadata"
                poster="/evidence-packs/evidence-pack-export-poster.svg"
              >
                <source src="/evidence-packs/satgate-evidence-pack-walkthrough.mp4" type="video/mp4" />
              </video>
            </div>
            <div className="rounded-3xl border border-cyan-300/20 bg-cyan-300/5 p-7">
              <div className="mb-4 flex items-center gap-2 text-cyan-200">
                <PlayCircle size={22} />
                <span className="text-sm font-bold uppercase tracking-[0.2em]">90-second Evidence Pack cut</span>
              </div>
              <h3 className="text-2xl font-black text-white">Show the authority lifecycle, then end on the exported proof.</h3>
              <p className="mt-4 text-sm leading-6 text-gray-400">
                A focused walkthrough of the buyer moment: signed lifecycle export, receipt chain, audit mappings, and the downloadable Evidence Pack.
              </p>
              <div className="mt-6 flex flex-col gap-3 text-sm font-bold sm:flex-row sm:flex-wrap">
                <a href="/evidence-packs/satgate-evidence-pack-walkthrough.mp4" className="inline-flex items-center gap-2 text-cyan-200 hover:text-cyan-100">
                  Watch the 90-second cut <ArrowRight size={16} />
                </a>
                <Link href="/agent-control-plane" className="inline-flex items-center gap-2 text-gray-300 hover:text-white">
                  See the agent control plane <ArrowRight size={16} />
                </Link>
              </div>
            </div>
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
                <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
                  <a
                    href="/evidence-packs/sample-evidence-pack.pdf"
                    className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-3 font-bold text-black transition hover:bg-gray-200"
                  >
                    Download sample Evidence Pack <Download size={18} />
                  </a>
                  <a
                    href="/evidence-packs/sample-evidence-pack.json"
                    className="inline-flex items-center justify-center gap-2 rounded-lg border border-emerald-300/40 bg-emerald-300/10 px-5 py-3 font-bold text-emerald-100 transition hover:border-emerald-200"
                  >
                    View JSON export <ArrowRight size={18} />
                  </a>
                  <a
                    href="mailto:contact@satgate.io?subject=SatGate%20Policy-to-Proof%20walkthrough"
                    className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/20 px-5 py-3 font-bold text-white transition hover:border-emerald-300/60"
                  >
                    Book a 15-minute walkthrough <ArrowRight size={18} />
                  </a>
                </div>
              </div>
              <ShieldCheck className="hidden shrink-0 text-emerald-300 md:block" size={56} />
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}

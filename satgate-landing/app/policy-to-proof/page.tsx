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
  title: { absolute: "SatGate Rules and Receipts | Evidence for Enterprise Agent Permissions" },
  description:
    "Define what an agent is allowed to do, enforce it at the gateway, and produce evidence humans and upstreams can trust.",
  keywords: [
    "rules and receipts",
    "AI agent signed receipt (Evidence Pack)",
    "agent permissions evidence",
    "agent control plane evidence",
    "tamper-evident signed receipt",
    "AI agent revocation",
    "AI agent spend ledger",
    "macaroon delegation audit",
  ],
  alternates: {
    canonical: "https://satgate.io/policy-to-proof",
  },
  openGraph: {
    title: "SatGate Rules and Receipts | Evidence for Enterprise Agent Permissions",
    description:
      "Signed receipts for budget spends and for refusals over a budget, a missing payment or a tool permission, which your CISO, finance team and auditor can check.",
    url: "https://satgate.io/policy-to-proof",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate Rules and Receipts | Evidence for Enterprise Agent Permissions",
    description:
      "Run agents without permanent credentials, unlimited spend, or unobservable authority — and export the proof.",
  },
};

const evidenceQuestions = [
  {
    question: "Who authorized this?",
    artifact: "Mint record",
    body: "The audit log entry for the token issued to the invoice-reconciler worker.",
  },
  {
    question: "Which agent got access?",
    artifact: "Capability token + delegation chain",
    body: "The root capability, parent agent, invoice-reconciler worker, and every narrowing in the handoff path.",
  },
  {
    question: "What exactly could it do?",
    artifact: "Scope + caveat record",
    body: "Routes, tools, tenant boundary, expiry, budget, delegation depth, and revocation status at issue time.",
  },
  {
    question: "What did it spend?",
    artifact: "Per-token spend ledger",
    body: "Attribution of each request that goes through SatGate, and of MCP tools, by worker, token, tenant, route, amount, and policy mode.",
  },
  {
    question: "What was denied?",
    artifact: "Denial receipt",
    body: "A budget, missing-payment or tool-permission reason code attached to the blocked invoice call.",
  },
  {
    question: "Can we prove revocation worked?",
    artifact: "Post-revoke denial",
    body: "The next request after a revoke is refused.",
  },
];

const demoSteps = [
  ["Mint", "invoice-reconciler gets a scoped macaroon capability with tenant, budget, route, and expiry caveats."],
  ["Delegate", "The parent finance agent hands narrower authority to the worker; the chain is preserved."],
  ["Spend", "The worker calls invoice APIs or MCP tools under budget; the ledger updates by token and path."],
  ["Deny", "Export and over-budget calls return reason-coded receipts before data or spend escapes."],
  ["Revoke", "Security kills the worker capability without rotating every upstream provider key."],
  ["Export", "The workflow's signed receipts are exported as a list a CISO, auditor or incident reviewer can read."],
];

const standardsMappings = [
  ["Delegation chain", "SOC 2 CC6.3 / NIST AC-3", "Least-privilege narrowing across parent and worker permissions."],
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
    quote: "Revocation isn’t a promise. The next request after a revoke is refused.",
  },
  {
    title: "FinOps lead",
    icon: WalletCards,
    quote: "Spend is attributed to the agent, the token, and the route — not a shared API key.",
  },
];

const evidencePack = {
  evidence_pack_id: "ep_demo_2026_05_09_001",
  policy_version: "pol_invoice_reconciliation@2026-05-09",
  issued_at: "2026-05-09T14:22:31Z",
  expires_at: "2026-05-16T14:22:31Z",
  tenant: "acme-finance",
  subject: "worker-agent:invoice-reconciler",
  payment_context: {
    default_market: "internal_enterprise_agents",
    internal_rail: "existing_iam_service_account_or_api_key_billing",
    optional_external_rails: ["x402", "l402"],
    note: "The same authority chain and receipts can cover an internal agent call, or a governed bridge call to an external paid API.",
  },
  authority_chain: [
    {
      kind: "root_grant",
      subject: "dean-agent:finance-automation",
      issuer_kid: "satgate-mint-2026-05",
      caveats: ["tenant_id = acme-finance", "delegation_depth = 1"],
    },
    {
      kind: "delegation",
      subject: "worker-agent:invoice-reconciler",
      parent: "dean-agent:finance-automation",
    },
  ],
  receipts: [
    { receipt_id: "rcpt_spend_search_003", evidence_pack_id: "ep_demo_2026_05_09_001", policy_version: "pol_invoice_reconciliation@2026-05-09", type: "spend", ts: "2026-05-09T14:23:18Z", route: "/v1/invoices/search", amount_usd: "0.18", payment_protocol: "internal_api", settlement: { rail: "internal_ledger", cost_center: "FIN-AP-042" }, decision_reason: "allowed_under_policy", result: "allowed", receipt_hash: "sha256:01d8..." },
    { receipt_id: "rcpt_spend_compare_004", evidence_pack_id: "ep_demo_2026_05_09_001", policy_version: "pol_invoice_reconciliation@2026-05-09", type: "spend", ts: "2026-05-09T14:24:02Z", route: "/v1/invoices/compare", amount_usd: "0.42", payment_protocol: "internal_api", settlement: { rail: "internal_ledger", cost_center: "FIN-AP-042" }, decision_reason: "allowed_under_policy", result: "allowed", receipt_hash: "sha256:a923..." },
    { receipt_id: "rcpt_paid_ocr_005", evidence_pack_id: "ep_demo_2026_05_09_001", policy_version: "pol_invoice_reconciliation@2026-05-09", type: "spend", ts: "2026-05-09T14:24:44Z", route: "/v1/invoices/ocr", mcp_tool: "document_ai.ocr", amount_usd: "0.18", payment_protocol: "x402", settlement: { chain: "base", tx: "REDACTED_DEMO_SAMPLE", ms: 187 }, decision_reason: "allowed_under_policy", result: "allowed", receipt_hash: "sha256:deb5..." },
    { receipt_id: "rcpt_denial_scope_006", evidence_pack_id: "ep_demo_2026_05_09_001", policy_version: "pol_invoice_reconciliation@2026-05-09", type: "denial", ts: "2026-05-09T14:25:08Z", decision_reason: "scope_violation:no_customer_data_export", reason_code: "scope_violation:no_customer_data_export", result: "blocked", receipt_hash: "sha256:9b0f..." },
    { receipt_id: "rcpt_denial_budget_007", evidence_pack_id: "ep_demo_2026_05_09_001", policy_version: "pol_invoice_reconciliation@2026-05-09", type: "denial", ts: "2026-05-09T14:25:33Z", decision_reason: "budget_exhausted", reason_code: "budget_exhausted", result: "blocked", receipt_hash: "sha256:c3f6..." },
  ],
  chain_root: "sha256:f04ed8430b11c8975cc5ef35919ee078fc4cb166cd8d611ed0d94b7da69df09d",
  signature: "ed25519:REDACTED_DEMO_SAMPLE_DO_NOT_VERIFY",
};

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebPage",
      name: "SatGate Rules and Receipts | Evidence for Enterprise Agent Permissions",
      url: "https://satgate.io/policy-to-proof",
      description: metadata.description,
      datePublished: "2026-05-09",
      dateModified: "2026-05-09",
      isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
      about: [
        { "@type": "Thing", name: "agent permissions evidence" },
        { "@type": "Thing", name: "tamper-evident signed receipt" },
        { "@type": "Thing", name: "agent control plane" },
        { "@type": "Thing", name: "macaroon delegation chain" },
      ],
    },
    {
      "@type": "ItemList",
      name: "Rules and receipts questions",
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
            <FileText size={14} /> Rules and Receipts
          </div>
          <div className="grid gap-10 lg:grid-cols-[1.05fr_0.95fr] lg:items-center">
            <div>
              <p className="mb-5 max-w-2xl rounded-2xl border border-white/10 bg-white/[0.04] p-4 text-sm leading-6 text-gray-300">
                Monday morning, your auditor asks who authorized the agent that tried to export customer data on Friday night. This is what they get.
              </p>
              <h1 className="max-w-4xl text-5xl font-black tracking-tight sm:text-6xl lg:text-7xl">
                From policy to proof for agentic API access.
              </h1>
              <p className="mt-6 max-w-3xl text-xl leading-8 text-gray-300">
                Define what an agent is allowed to do, enforce it at the gateway, and produce evidence humans and upstreams can trust.
              </p>
              <p className="mt-5 max-w-3xl text-lg leading-8 text-gray-400">
                Bind agent actions to human or platform authority; apply spend, scope, rate, and escalation limits before access; export the proof when your CISO, auditor, board, or incident reviewer asks what happened.
              </p>
              <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
                <Link
                  href="/evidence-pack-demo"
                  className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-3 font-bold text-black transition hover:bg-gray-200"
                >
                  Open signed receipt (Evidence Pack) viewer <ArrowRight size={18} />
                </Link>
                <a
                  href="/evidence-packs/sample-evidence-pack.v1.json"
                  className="inline-flex items-center justify-center gap-2 rounded-lg border border-cyan-300/40 bg-cyan-300/10 px-5 py-3 font-bold text-cyan-100 transition hover:border-cyan-200"
                >
                  Download JSON export <Download size={18} />
                </a>
                <a
                  href="/evidence-packs/evidence-pack.schema.v1.json"
                  className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/20 px-5 py-3 font-bold text-white transition hover:border-white/50"
                >
                  Schema v1 <ArrowRight size={18} />
                </a>
              </div>
            </div>

            <div className="rounded-3xl border border-cyan-300/20 bg-white/[0.03] p-5 shadow-2xl shadow-cyan-950/30">
              <div className="mb-4 flex items-center justify-between border-b border-white/10 pb-3">
                <div>
                  <p className="text-sm font-semibold text-cyan-200">Sample, not a live receipt</p>
                  <p className="text-xs text-gray-500">x402 in this sample is USDC on Base</p>
                </div>
                <BadgeCheck className="text-emerald-300" />
              </div>
              <pre className="max-h-[560px] overflow-x-auto rounded-2xl bg-black/70 p-4 text-xs leading-6 text-cyan-50">
                {JSON.stringify(evidencePack, null, 2)}
              </pre>
              <p className="mt-3 text-xs leading-5 text-gray-500">
                Inline hashes are shortened for readability. Full hashes, payment rail context, ed25519 signature, and verification block are in the downloadable JSON.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 bg-white/[0.02] px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-3xl">
            <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-cyan-300">Internal first, rail-aware when needed</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Built for internal enterprise agents. Extends across paid external calls.</h2>
            <p className="mt-5 text-lg leading-8 text-gray-400">
              Most enterprise agents do not need a wallet to call internal APIs. They need bounded delegated permissions, budget controls, revocation, and audit evidence around the credentials they already have.
            </p>
            <p className="mt-4 text-base leading-7 text-gray-500">
              When that same internal workflow crosses into an external paid API, SatGate keeps the proof intact: internal scope and delegation, plus spend attribution for x402 and L402 payments. Payment proves value moved. SatGate proves the agent was allowed to move it.
            </p>
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-4xl">
            <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-purple-300">Why now</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Agents are moving from demos to workflows. Spend rails are arriving with them.</h2>
            <p className="mt-5 text-xl leading-8 text-gray-300">
              Enterprises need proof of authority before those rails scale. Payment rails authorize value movement. SatGate authorizes behavior and preserves proof. Payment proves value moved. SatGate proves the agent was allowed to move it.
            </p>
          </div>

          <div className="mt-10 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <article className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
              <p className="text-sm font-bold uppercase tracking-[0.2em] text-cyan-300">Internal agents</p>
              <h3 className="mt-4 text-xl font-black text-white">Leaving the lab.</h3>
              <p className="mt-3 text-sm leading-6 text-gray-400">
                Agents now hold authority that used to require human approval. The control question shifts from whether the API call worked to who authorized what, under which policy, and whether you can prove it.
              </p>
            </article>
            <article className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
              <p className="text-sm font-bold uppercase tracking-[0.2em] text-cyan-300">Paid calls</p>
              <h3 className="mt-4 text-xl font-black text-white">Becoming part of the path.</h3>
              <p className="mt-3 text-sm leading-6 text-gray-400">
                x402, L402, AgentCore Payments, Pay.sh and similar payment methods make it easier for agents to call paid services. SatGate adds policy, limits and receipts around x402 and L402 today. AgentCore Payments and Pay.sh are planned.
              </p>
            </article>
            <article className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
              <p className="text-sm font-bold uppercase tracking-[0.2em] text-cyan-300">Enterprise scale</p>
              <h3 className="mt-4 text-xl font-black text-white">Authority before scale.</h3>
              <p className="mt-3 text-sm leading-6 text-gray-400">
                CISOs, platform teams, and FinOps leaders need more than logs after the fact. They need evidence that each action matched policy before agents trigger budget, data access, or external execution.
              </p>
            </article>
            <article className="rounded-2xl border border-emerald-300/20 bg-emerald-300/5 p-6">
              <p className="text-sm font-bold uppercase tracking-[0.2em] text-emerald-200">SatGate</p>
              <h3 className="mt-4 text-xl font-black text-white">Sits in front of your APIs.</h3>
              <p className="mt-3 text-sm leading-6 text-gray-400">
                SatGate signs a receipt when it refuses an agent over a budget, a missing payment or a tool permission, and when an agent spends from a budget in Control. This covers HTTP APIs and MCP tools.
              </p>
            </article>
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
              <h3 className="mt-4 text-2xl font-black text-white">Signed receipts for budget spends and for refusals over a budget, a missing payment or a tool permission, exportable as a list.</h3>
              <p className="mt-4 text-base leading-7 text-gray-400">
                The signed receipt (Evidence Pack) is not a logging afterthought. It is generated by the same authority path that enforces the decision.
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
              <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Answer the audit questions after the invoice-reconciler acts.</h2>
            </div>
            <p className="max-w-md text-sm leading-6 text-gray-500">
              Receipts and the audit log answer these questions without a forensics project across logs, invoices and gateway dashboards. Token lineage shows who handed work to whom; matching receipts preserve the allow and deny sequence, including paid requests.
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
              The receipt gives security and audit teams a starting control map instead of making them translate raw gateway logs without receipt hashes, policy versions, and decision reasons.
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

          <div className="mt-10 rounded-3xl border border-cyan-300/20 bg-cyan-300/5 p-7">
            <div className="flex flex-col gap-5 md:flex-row md:items-center md:justify-between">
              <div>
                <p className="text-sm font-bold uppercase tracking-[0.2em] text-cyan-200">Want this for your stack?</p>
                <p className="mt-3 max-w-2xl text-xl font-black leading-8 text-white">
                  Bring an API, an agent workflow, and the evidence your auditor already asks for.
                </p>
              </div>
              <a
                href="mailto:contact@satgate.io?subject=SatGate%20Policy-to-Proof%20walkthrough"
                className="inline-flex shrink-0 items-center justify-center gap-2 rounded-lg bg-white px-5 py-3 font-bold text-black transition hover:bg-gray-200"
              >
                Book a 15-minute walkthrough <ArrowRight size={18} />
              </a>
            </div>
          </div>
        </div>
      </section>

      <section className="px-6 py-20">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-3xl">
            <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-purple-300">Demo path</p>
            <h2 className="text-3xl font-black tracking-tight sm:text-5xl">Mint → Delegate → Spend → Deny → Revoke → Export.</h2>
            <p className="mt-5 text-lg leading-8 text-gray-400">
              The demo ends on the exported receipts across the invoice-reconciler lifecycle: signed receipts for budget spends and for refusals over a budget, a missing payment or a tool permission.
            </p>
            <p className="mt-4 text-sm leading-6 text-gray-500">
              Read the six-step lifecycle below, or watch the 90-second cut.
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
                <source src="/evidence-packs/satgate-evidence-pack-walkthrough-audio.mp4" type="video/mp4" />
              </video>
            </div>
            <div className="rounded-3xl border border-cyan-300/20 bg-cyan-300/5 p-7">
              <div className="mb-4 flex items-center gap-2 text-cyan-200">
                <PlayCircle size={22} />
                <span className="text-sm font-bold uppercase tracking-[0.2em]">Receipt walkthrough</span>
              </div>
              <h3 className="text-2xl font-black text-white">See how agent permissions become audit-ready proof.</h3>
              <p className="mt-4 text-sm leading-6 text-gray-400">
                Watch a governed agent receive scoped authority, delegate work, hit policy decisions, and export a signed receipt your security and audit teams can review.
              </p>
              <div className="mt-6 flex flex-col gap-3 text-sm font-bold sm:flex-row sm:flex-wrap">
                <a href="/evidence-packs/satgate-evidence-pack-walkthrough-audio.mp4" className="inline-flex items-center gap-2 text-cyan-200 hover:text-cyan-100">
                  Watch the walkthrough <ArrowRight size={16} />
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
                  SatGate lets enterprises delegate limited authority to AI agents. SatGate signs a receipt when it refuses an agent over a budget, a missing payment or a tool permission, and when an agent spends from a budget in Control.
                </p>
                <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
                  <a
                    href="/evidence-pack-demo"
                    className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-3 font-bold text-black transition hover:bg-gray-200"
                  >
                    Open signed receipt (Evidence Pack) viewer <ArrowRight size={18} />
                  </a>
                  <a
                    href="/evidence-packs/sample-evidence-pack.v1.json"
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

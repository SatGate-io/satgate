import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight, BadgeCheck, FileText, GitBranch, Handshake, Layers3, ReceiptText, ShieldCheck } from "lucide-react";

export const metadata: Metadata = {
  title: "SatGate for Payment Methods | Agent Permissions and Receipts",
  description:
    "SatGate gives payment methods, API platforms, and agent ecosystems a permissions-and-receipts layer that works with any payment method: policy before action, receipts after action, and signed receipts (Evidence Packs) for dispute-ready proof.",
  keywords: [
    "agent payment method governance",
    "AI agent permissions and receipts",
    "x402 governance",
    "L402 governance",
    "agent payment receipts",
    "signed receipt (Evidence Pack) verifier",
    "agent accountability layer",
  ],
  alternates: { canonical: "https://satgate.io/partners/rails" },
  openGraph: {
    title: "SatGate for Payment Methods | Agent Permissions and Receipts",
    description:
      "An Economic Firewall that works with any payment method, for partners that need authorization, scope, budget, and signed receipt proof around agent-initiated transactions.",
    url: "https://satgate.io/partners/rails",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate for Payment Methods",
    description:
      "Policy before agent action. Receipts after action. Signed receipts for partner review.",
  },
};

const partnerFit = [
  ["Payment methods", "x402, L402, Stripe, AgentCore Payments, Pay.sh, card, wallet, and ledger teams that need proof around autonomous-agent payment attempts."],
  ["API platforms", "Marketplaces and API providers that want scoped agent acceptance without inheriting every identity, delegation, or dispute question."],
  ["Agent platforms", "Runtimes, MCP gateways, and orchestration layers that need one permissions contract across payment methods and unpaid APIs."],
];

const flow = [
  ["1", "Permission", "The principal or platform passes down a bounded capability: agent, tenant, route/tool, budget, expiry, policy version, and delegation depth."],
  ["2", "Decision", "SatGate enforces policy before the request goes through to the upstream API, MCP tool, or payment method."],
  ["3", "Receipt", "Every allowed, denied, delegated, revoked, or paid decision emits a signed receipt with the policy basis and payment details."],
  ["4", "Signed receipt", "Receipts roll into a verifiable bundle (Evidence Pack), anchored by issuer JWKS and independently checked by the open verifier."],
];

const railQuestions = [
  "Were permissions passed down to this agent before the transaction?",
  "Which policy, tenant, budget, route, tool, and delegation chain applied?",
  "Did the decision happen before value moved or an upstream action executed?",
  "Can a partner, customer, auditor, or fraud team verify the artifact without trusting a dashboard screenshot?",
];

const publicArtifacts = [
  ["Public verifier", "https://github.com/SatGate-io/evidence-pack-verifier"],
  ["Live signed receipt (Evidence Pack)", "https://api.satgate.io/v1/evidence/evid_GrXvKUgtdqNbuQ5lZzqRMpZrOoU2VAnE"],
  ["Receipt schema", "https://satgate.io/.well-known/satgate-receipt.schema.json"],
  ["Partner brief PDF", "/briefs/satgate-agent-authority-rails-brief.pdf"],
];

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebPage",
      name: "SatGate for Payment Methods",
      url: "https://satgate.io/partners/rails",
      description: metadata.description,
      isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
      about: [
        { "@type": "Thing", name: "agent payment method governance" },
        { "@type": "Thing", name: "Agent Permissions and Receipts" },
        { "@type": "Thing", name: "signed receipt (Evidence Pack) verification" },
        { "@type": "Thing", name: "Economic Firewall that works with any payment method" },
      ],
    },
    {
      "@type": "BreadcrumbList",
      itemListElement: [
        { "@type": "ListItem", position: 1, name: "Home", item: "https://satgate.io" },
        { "@type": "ListItem", position: 2, name: "Partners", item: "https://satgate.io/partners/rails" },
        { "@type": "ListItem", position: 3, name: "Payment methods", item: "https://satgate.io/partners/rails" },
      ],
    },
  ],
};

export default function RailPartnersPage() {
  return (
    <main className="min-h-screen bg-black text-white">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />

      <section className="relative overflow-hidden border-b border-white/10 px-6 py-24 sm:py-32">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.2),transparent_35%),radial-gradient(circle_at_bottom_right,rgba(168,85,247,0.18),transparent_38%)]" />
        <div className="relative mx-auto grid max-w-6xl gap-12 lg:grid-cols-[1.05fr_0.95fr] lg:items-center">
          <div>
            <Link href="/" className="mb-8 inline-flex text-sm font-semibold text-gray-400 transition hover:text-white">← Back to Home</Link>
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-cyan-400/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-cyan-200">
              <Handshake size={14} /> Partner brief
            </div>
            <h1 className="max-w-4xl text-5xl font-black tracking-tight sm:text-6xl lg:text-7xl">
              Agent permissions for every payment method.
            </h1>
            <p className="mt-6 max-w-3xl text-xl leading-8 text-gray-300">
              Payment methods can authorize value movement. SatGate proves the agent was allowed to attempt it: who passed permissions down, which policy applied, what budget was left, and what evidence exists after the decision.
            </p>
            <p className="mt-5 max-w-3xl text-lg leading-8 text-gray-400">
              Payment methods keep changing. SatGate’s Agent Permissions and Receipts layer sits above x402, L402, Stripe, AgentCore Payments, Pay.sh, API-key billing, and enterprise ledgers.
            </p>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
              <a href="/briefs/satgate-agent-authority-rails-brief.pdf" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
                Download partner brief <FileText size={18} />
              </a>
              <Link href="/mcp" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
                See agent permissions and receipts <ArrowRight size={18} />
              </Link>
              <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-purple-500">
                See how rules and receipts work <ArrowRight size={18} />
              </Link>
            </div>
          </div>

          <div className="rounded-3xl border border-cyan-300/20 bg-white/[0.03] p-6 shadow-2xl shadow-cyan-950/30">
            <div className="mb-5 flex items-center gap-2 text-emerald-200">
              <BadgeCheck /> Partner questions SatGate answers
            </div>
            <div className="space-y-4">
              {railQuestions.map((item) => (
                <div key={item} className="rounded-2xl border border-white/10 bg-black/50 p-4 text-gray-300">
                  {item}
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-10 max-w-3xl">
          <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-purple-300">Partner fit</p>
          <h2 className="text-3xl font-bold text-white sm:text-4xl">SatGate is useful when a payment method does not want to become the whole governance stack.</h2>
          <p className="mt-5 text-lg leading-8 text-gray-400">
            A single payment method sees the transaction. Enterprises need proof across tools, APIs, delegations, budgets, denials, and retries. SatGate preserves that chain of permissions and gives partners a verifiable artifact to evaluate.
          </p>
        </div>
        <div className="grid gap-6 md:grid-cols-3">
          {partnerFit.map(([title, body]) => (
            <div key={title} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
              <Layers3 className="mb-4 text-cyan-300" />
              <h3 className="text-xl font-bold text-white">{title}</h3>
              <p className="mt-3 leading-7 text-gray-400">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <div className="mb-10 max-w-3xl">
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-emerald-300">Rules and Receipts</p>
            <h2 className="text-3xl font-bold text-white sm:text-4xl">The handoff is simple: policy in, receipt out.</h2>
          </div>
          <div className="grid gap-4 md:grid-cols-4">
            {flow.map(([step, title, body]) => (
              <div key={title} className="rounded-2xl border border-gray-800 bg-black p-5">
                <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-full bg-cyan-400/10 font-black text-cyan-200">{step}</div>
                <h3 className="font-bold text-white">{title}</h3>
                <p className="mt-2 text-sm leading-6 text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[0.95fr_1.05fr] lg:items-start">
        <div>
          <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-cyan-300">Public proof</p>
          <h2 className="text-3xl font-bold text-white sm:text-4xl">Partners should not have to trust a screenshot.</h2>
          <p className="mt-5 text-lg leading-8 text-gray-400">
            A signed receipt (Evidence Pack) can be checked outside SatGate. A partner, customer, or reviewer can fetch it, discover issuer keys, recompute receipt hashes, and verify the signature with the open-source verifier.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
            <a href="https://github.com/SatGate-io/evidence-pack-verifier" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Open verifier <ArrowRight size={18} />
            </a>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-purple-500">
              See how rules and receipts work
            </Link>
          </div>
        </div>
        <div className="rounded-2xl border border-gray-800 bg-black p-6">
          <div className="mb-4 flex items-center gap-2 text-cyan-200">
            <ReceiptText size={20} /> Partner artifacts
          </div>
          <div className="space-y-3">
            {publicArtifacts.map(([label, href]) => (
              <a key={label} href={href} className="block rounded-xl border border-gray-800 bg-gray-950 p-4 text-sm text-gray-300 transition hover:border-cyan-500 hover:text-white">
                <span className="font-semibold text-white">{label}</span>
                <span className="mt-1 block break-all text-gray-500">{href}</span>
              </a>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 px-6 py-20 text-center">
        <ShieldCheck className="mx-auto mb-6 text-cyan-300" size={36} />
        <h2 className="mx-auto max-w-3xl text-3xl font-bold text-white sm:text-4xl">Build the payment method. Let SatGate carry agent permissions and evidence across it.</h2>
        <p className="mx-auto mt-5 max-w-2xl text-lg leading-8 text-gray-400">
          The partner conversation starts with receipts: what the payment method sees, what SatGate proves, and what the ecosystem can verify later.
        </p>
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <a href="mailto:contact@satgate.io?subject=SatGate%20rail%20partner%20brief" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
            Talk to SatGate <ArrowRight size={18} />
          </a>
          <Link href="/build" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-400">
            Developer building blocks <GitBranch size={18} />
          </Link>
        </div>
      </section>
    </main>
  );
}

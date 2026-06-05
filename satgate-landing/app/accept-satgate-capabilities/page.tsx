import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft, BadgeCheck, CheckCircle2, FileCheck2, ShieldCheck, XCircle } from "lucide-react";

export const metadata: Metadata = {
  title: "Accept SatGate Capabilities",
  description:
    "Define what it means for an upstream API to accept SatGate-scoped capabilities: verify bounded authority, execute the allowed action, and return receipts without implying marketplace reputation.",
  keywords: [
    "SatGate capabilities",
    "capability accepting API",
    "agent capability verification",
    "SatGate receipts",
    "upstream API acceptance",
    "AI agent authorization",
  ],
  alternates: { canonical: "https://satgate.io/accept-satgate-capabilities" },
  openGraph: {
    title: "Accept SatGate capabilities",
    description:
      "A precise upstream acceptance story: scoped capability verification in, receipt emission out. No marketplace, no reputation claim.",
    url: "https://satgate.io/accept-satgate-capabilities",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Accept SatGate capabilities",
    description: "Capability verification in. Receipt emission out. No network-wide reputation claim.",
  },
};

const badgeCopy = [
  {
    label: "Badge text",
    value: "Accepts SatGate capabilities",
    body: "Use when an upstream verifies SatGate-scoped capabilities and returns SatGate-compatible receipts for accepted calls.",
  },
  {
    label: "Short disclaimer",
    value: "Capability verification + receipts only",
    body: "The badge does not mean SatGate endorses the upstream, ranks it, audits its business, or provides network-wide reputation.",
  },
  {
    label: "Machine signal",
    value: "roles: [\"acceptor\"]",
    body: "Acceptor metadata advertises accepted capability formats, trust anchors, recognized prior receipt decisions, emitted receipt decisions, and the claim boundary.",
  },
];

const checklist = [
  "Publish or document the upstream acceptor identity and contact path.",
  "Accept `Authorization: Bearer <capability>` using `satgate.capability.v1` or `macaroon-bearer`.",
  "Verify issuer trust anchor before fetching issuer JWKS.",
  "Verify signature, expiry, audience, route/tool scope, caveats, budget, and delegation depth before execution.",
  "Execute only the action authorized by the capability.",
  "Return a SatGate-compatible receipt that validates against `satgate.receipt.v1` for the acceptor v0 decision subset: allowed, denied, or paid.",
  "Expose a test vector or mock endpoint before claiming real production acceptance.",
];

const criteria = [
  { title: "Capability accepted", body: "A valid scoped capability for the upstream audience and route is accepted before execution." },
  { title: "Capability denied", body: "An expired, wrong-audience, wrong-route, over-budget, or untrusted-issuer capability is rejected with a receipt-grade denial." },
  { title: "Receipt returned", body: "Every allowed or denied call returns a receipt with `schema_version`, `schema_url`, `receipt_id`, `evidence_pack_id`, `issuer`, `issuer_kid`, `decision`, `decision_reason`, `policy_version`, `timestamp`, `canonicalization`, `hash_algorithm`, `signature_algorithm`, `receipt_hash`, and `signature`." },
  { title: "Trust remains bounded", body: "The upstream makes no marketplace, ranking, reputation, or endorsement claim. Acceptance proves verification behavior, not global trust." },
];

const mockReceipt = {
  schema_version: "satgate.receipt.v1",
  schema_url: "https://satgate.io/.well-known/satgate-receipt.schema.json",
  receipt_id: "rcpt_mock_accept_001",
  evidence_pack_id: "ep_mock_accept_001",
  issuer: "https://satgate.io",
  issuer_kid: "satgate-mock-2026-05",
  acceptor_id: "https://api.internal.example/satgate-mock",
  decision: "allowed",
  decision_reason: "capability_scope_audience_and_budget_ok",
  policy_version: "policy_mock_acceptance_v0",
  timestamp: "2026-05-13T00:00:00Z",
  canonicalization: "jcs-rfc8785",
  hash_algorithm: "sha256",
  signature_algorithm: "ed25519",
  capability_hash: "sha256:mock_capability_hash",
  receipt_hash: "sha256:mock_receipt_hash",
  signature: "ed25519:mock_signature_not_for_production",
  mock_only: true,
};

const jsonLd = {
  "@context": "https://schema.org",
  "@type": "TechArticle",
  headline: "Accept SatGate capabilities",
  description: metadata.description,
  url: "https://satgate.io/accept-satgate-capabilities",
  datePublished: "2026-05-13",
  dateModified: "2026-06-04",
  about: ["SatGate capabilities", "upstream API acceptance", "verifiable receipts"],
};

export default function AcceptSatGateCapabilitiesPage() {
  return (
    <main className="min-h-screen overflow-x-hidden bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />

      <div className="mx-auto max-w-6xl px-6 pt-8">
        <Link href="/" className="inline-flex items-center gap-2 text-sm font-medium text-gray-500 transition hover:text-white">
          <ArrowLeft size={16} /> Back to Home
        </Link>
      </div>

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_15%_10%,rgba(56,189,248,0.18),transparent_30%),radial-gradient(circle_at_82%_15%,rgba(16,185,129,0.16),transparent_32%)]" />
        <div className="relative mx-auto grid max-w-6xl gap-12 px-6 py-24 lg:grid-cols-[0.95fr_1.05fr] lg:items-center">
          <div>
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-emerald-400/30 bg-emerald-400/10 px-4 py-2 text-sm font-semibold uppercase tracking-[0.22em] text-emerald-200">
              <BadgeCheck size={16} /> Upstream acceptance
            </div>
            <h1 className="max-w-4xl text-5xl font-black tracking-tight text-white sm:text-6xl">
              Accept SatGate capabilities without pretending there is a marketplace.
            </h1>
            <p className="mt-6 max-w-2xl text-xl leading-9 text-gray-300">
              Acceptance means an upstream verifies SatGate-scoped authority before action and returns receipt-grade evidence after the decision. It does not mean SatGate ranks, endorses, or reputationally scores that upstream.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Link href="/build" className="rounded-full bg-white px-5 py-3 text-sm font-bold text-black transition hover:bg-cyan-100">Build with capabilities</Link>
              <Link href="/capability-auth" className="rounded-full border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-400">Review capability auth</Link>
              <Link href="https://github.com/SatGate-io/satgate/blob/main/docs/reference/accept-satgate-capabilities.md" className="rounded-full border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-400">Read the acceptance docs</Link>
            </div>
          </div>

          <div className="rounded-3xl border border-gray-800 bg-gray-950/85 p-6 shadow-2xl shadow-cyan-500/10">
            <div className="mb-4 flex items-center gap-2 text-emerald-200"><ShieldCheck size={20} /> Badge definition</div>
            <div className="rounded-2xl border border-emerald-400/30 bg-emerald-400/10 p-5">
              <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-emerald-300 px-3 py-1 text-xs font-black uppercase tracking-[0.18em] text-black">
                <BadgeCheck size={14} /> Accepts SatGate capabilities
              </div>
              <p className="text-sm leading-6 text-emerald-50">Verifies scoped SatGate capabilities and returns SatGate-compatible receipts. Acceptance means capability verification and receipt emission. No marketplace, ranking, reputation, or endorsement claim.</p>
            </div>
            <pre className="mt-5 max-w-full overflow-x-auto rounded-2xl bg-black p-5 text-xs leading-6 text-gray-300"><code>{JSON.stringify(mockReceipt, null, 2)}</code></pre>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-10 max-w-3xl">
          <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-cyan-300">Badge and copy</p>
          <h2 className="text-3xl font-bold text-white sm:text-4xl">The badge says exactly one thing.</h2>
        </div>
        <div className="grid gap-6 md:grid-cols-3">
          {badgeCopy.map((item) => (
            <div key={item.label} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
              <p className="mb-3 text-xs font-mono uppercase tracking-[0.18em] text-gray-500">{item.label}</p>
              <h3 className="mb-3 text-xl font-bold text-white">{item.value}</h3>
              <p className="leading-7 text-gray-400">{item.body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-2">
          <div>
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-emerald-300">Minimal integration checklist</p>
            <h2 className="mb-6 text-3xl font-bold text-white">Enough to prove acceptance. Not enough to claim reputation.</h2>
            <div className="space-y-4">
              {checklist.map((item) => (
                <div key={item} className="flex gap-3 rounded-2xl border border-gray-800 bg-black p-4">
                  <CheckCircle2 className="mt-1 shrink-0 text-emerald-300" size={18} />
                  <p className="leading-7 text-gray-300">{item}</p>
                </div>
              ))}
            </div>
          </div>
          <div>
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-purple-300">Verification criteria</p>
            <h2 className="mb-6 text-3xl font-bold text-white">A verifier should be able to reproduce these outcomes.</h2>
            <div className="space-y-4">
              {criteria.map((item) => (
                <div key={item.title} className="rounded-2xl border border-gray-800 bg-black p-5">
                  <div className="mb-2 flex items-center gap-2 text-white"><FileCheck2 className="text-purple-300" size={18} /><h3 className="font-bold">{item.title}</h3></div>
                  <p className="leading-7 text-gray-400">{item.body}</p>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="grid gap-8 lg:grid-cols-[0.9fr_1.1fr] lg:items-start">
          <div>
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-red-300">No overclaiming</p>
            <h2 className="text-3xl font-bold text-white sm:text-4xl">What the badge does not mean.</h2>
            <div className="mt-6 space-y-4 text-lg leading-8 text-gray-400">
              <p>It does not mean SatGate operates a directory of trusted APIs.</p>
              <p>It does not mean the upstream is endorsed, ranked, audited, or safer than another upstream.</p>
              <p>It means one integration path has evidence: scoped capability verification before execution and receipt emission after the decision.</p>
            </div>
          </div>
          <div className="rounded-2xl border border-red-400/20 bg-red-400/5 p-6">
            <div className="mb-4 flex items-center gap-2 text-red-200"><XCircle size={20} /> Forbidden badge copy</div>
            <ul className="space-y-3 text-gray-300">
              <li>“SatGate trusted marketplace member”</li>
              <li>“SatGate certified reputation score”</li>
              <li>“SatGate endorsed API”</li>
              <li>“Network-approved upstream”</li>
            </ul>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-gray-950/70">
        <div className="mx-auto max-w-6xl px-6 py-16">
          <div className="rounded-3xl border border-gray-800 bg-black p-8 md:flex md:items-center md:justify-between md:gap-8">
            <div>
              <p className="mb-2 text-sm font-mono uppercase tracking-[0.22em] text-cyan-300">Internal mock first</p>
              <h2 className="text-3xl font-bold text-white">Use the mock acceptor until a real upstream is ready.</h2>
              <p className="mt-3 max-w-2xl text-gray-400">The mock example is deliberately labeled internal: it proves the acceptance shape without claiming public network adoption.</p>
            </div>
            <div className="mt-6 flex flex-wrap gap-3 md:mt-0">
              <Link href="/examples/mock-acceptor-metadata.v0.json" className="rounded-full border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-400">Mock metadata</Link>
              <a href="https://satgate.io/.well-known/satgate-acceptor.schema.json" className="rounded-full border border-gray-700 px-5 py-3 text-sm font-bold text-white transition hover:border-cyan-400">v0 schema</a>
              <Link href="/examples/mock-accepted-satgate-receipt.v1.json" className="rounded-full bg-cyan-300 px-5 py-3 text-sm font-bold text-black transition hover:bg-cyan-200">Mock receipt</Link>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}

import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight, BadgeCheck, FileCheck2, KeyRound, Layers3, ReceiptText, ShieldCheck } from "lucide-react";

export const metadata: Metadata = {
  title: "Agent Authority & Accountability Layer | SatGate",
  description:
    "SatGate is the rail-neutral authority and accountability layer for AI agents. Signed, verifiable receipts before any rail moves value.",
  keywords: [
    "agent authority and accountability layer",
    "AI agent accountability",
    "AI agent receipts",
    "Evidence Pack verifier",
    "rail-neutral agent governance",
    "x402 governance",
    "L402 governance",
  ],
  alternates: { canonical: "https://satgate.io/agent-authority-layer" },
  openGraph: {
    title: "Agent Authority & Accountability Layer | SatGate",
    description:
      "Rail-neutral authority, pre-flight policy enforcement, and signed Evidence Pack proof for autonomous agents.",
    url: "https://satgate.io/agent-authority-layer",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Agent Authority & Accountability Layer",
    description:
      "SatGate proves which agent was authorized, under which policy, before any rail or upstream service is touched.",
  },
};

const layers = [
  {
    icon: KeyRound,
    title: "Authority — before the call",
    body: "Delegate scoped capability to an agent: routes, tools, budget, expiry, tenant, policy version, and delegation depth.",
  },
  {
    icon: ShieldCheck,
    title: "Decision — at the call",
    body: "SatGate sits in the request path and enforces policy before APIs, MCP tools, or paid rails execute.",
  },
  {
    icon: ReceiptText,
    title: "Evidence — after the call",
    body: "Every allow, deny, delegation, revocation, and paid event leaves a signed receipt rolled into an Evidence Pack.",
  },
];

const audiences = [
  ["Enterprises", "Govern agent scope, budgets, revocation, and tenant boundaries before spend or data leaves."],
  ["Payment rails", "Get a recourse trail for autonomous-agent activity without owning every governance question."],
  ["Upstream APIs", "Decide which agents deserve preferential trust, rate limits, and lower fraud friction."],
  ["Agent platforms", "Plug in one authority model across MCP, AgentCore, LangGraph, Vercel AI SDK, and custom runtimes."],
  ["Insurers and fraud teams", "Underwrite or score agent behavior from verifiable Evidence Pack artifacts."],
];

const publicSpecs = [
  ["Evidence Pack schema", "https://github.com/SatGate-io/satgate/blob/main/docs/reference/receipt-schema.md"],
  ["Live Evidence Pack example", "https://api.satgate.io/v1/evidence/evid_LrlgUSR1R3SEYtxy0npX7mgneWZFa5ek"],
  ["Receipt JSON schema", "https://satgate.io/.well-known/satgate-receipt.schema.json"],
  ["Issuer JWKS", "https://api.satgate.io/.well-known/jwks.json"],
  ["Open verifier", "https://github.com/SatGate-io/evidence-pack-verifier"],
  ["Rail partner brief", "/partners/rails"],
];

const faqs = [
  {
    question: "Is SatGate a payment processor?",
    answer:
      "No. SatGate governs authority before any rail moves value. It is rail-neutral and works across L402, x402, Stripe, AgentCore, internal ledgers, and whatever comes next.",
  },
  {
    question: "How is this different from OAuth or API keys?",
    answer:
      "OAuth proves identity. API keys prove possession. SatGate proves this agent was authorized under a specific policy, with a specific scope and budget, and produces a signed artifact of that decision.",
  },
  {
    question: "Why not just trust the rail authorization?",
    answer:
      "Rails authorize payment. They do not enforce delegated scope, policy versions, revocation, budget ceilings, or proof across multiple rails at once.",
  },
  {
    question: "How is a SatGate receipt verified?",
    answer:
      "Fetch the Evidence Pack, discover the issuer JWKS, canonicalize the receipt with RFC8785 JCS, recompute the SHA-256 receipt hash, and verify the Ed25519 signature.",
  },
];

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebPage",
      name: "Agent Authority & Accountability Layer",
      url: "https://satgate.io/agent-authority-layer",
      description: metadata.description,
      dateModified: '2026-06-10',
      isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
      about: [
        { "@type": "Thing", name: "AI agent authority" },
        { "@type": "Thing", name: "Evidence Pack" },
        { "@type": "Thing", name: "rail-neutral agent governance" },
        { "@type": "Thing", name: "Ed25519 receipt verification" },
      ],
    },
    {
      "@type": "SoftwareApplication",
      name: "SatGate",
      applicationCategory: "BusinessApplication",
      featureList: [
        "Delegated agent authority",
        "Pre-flight policy enforcement",
        "Signed Evidence Packs",
        "Rail-neutral receipt verification",
      ],
    },
    {
      "@type": "FAQPage",
      mainEntity: faqs.map((faq) => ({
        "@type": "Question",
        name: faq.question,
        acceptedAnswer: { "@type": "Answer", text: faq.answer },
      })),
    },
    {
      "@type": "BreadcrumbList",
      itemListElement: [
        { "@type": "ListItem", position: 1, name: "Home", item: "https://satgate.io" },
        { "@type": "ListItem", position: 2, name: "Platform", item: "https://satgate.io/govern" },
        {
          "@type": "ListItem",
          position: 3,
          name: "Agent Authority & Accountability Layer",
          item: "https://satgate.io/agent-authority-layer",
        },
      ],
    },
  ],
};

export default function AgentAuthorityLayerPage() {
  return (
    <main className="min-h-screen bg-black text-white">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />

      <section className="relative overflow-hidden border-b border-white/10 px-6 py-24 sm:py-32">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(56,189,248,0.22),transparent_34%),radial-gradient(circle_at_bottom_right,rgba(168,85,247,0.18),transparent_38%)]" />
        <div className="relative mx-auto grid max-w-6xl gap-12 lg:grid-cols-[1.05fr_0.95fr] lg:items-center">
          <div>
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-cyan-400/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-cyan-200">
              <Layers3 size={14} /> Rail-neutral governance
            </div>
            <h1 className="max-w-4xl text-5xl font-black tracking-tight sm:text-6xl lg:text-7xl">
              Agent Authority & Accountability Layer
            </h1>
            <p className="mt-6 max-w-3xl text-xl leading-8 text-gray-300">
              Payment rails answer one question: <span className="text-white">can this agent pay?</span> SatGate answers the question that comes first: was this agent authorized, under whose policy, within what budget and scope, and can we prove it later?
            </p>
            <p className="mt-5 max-w-3xl text-lg leading-8 text-gray-400">
              Every rail — x402, L402, Stripe Agent Toolkit, AgentCore, Coinbase, and internal ledgers — has the same governance gap. SatGate fills it once, above the rail.
            </p>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
              <Link href="/build" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
                Read the spec <ArrowRight size={18} />
              </Link>
              <Link href="/design-partners" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
                Talk to us
              </Link>
            </div>
          </div>

          <div className="rounded-3xl border border-cyan-300/20 bg-white/[0.03] p-6 shadow-2xl shadow-cyan-950/30">
            <div className="mb-5 flex items-center gap-2 text-emerald-200">
              <BadgeCheck /> What SatGate proves
            </div>
            <div className="space-y-4">
              {[
                "This agent had delegated authority before execution.",
                "The requested action matched policy, budget, tenant, and scope.",
                "The decision was signed, archived, and externally verifiable.",
                "The proof travels across rails instead of being trapped inside one provider.",
              ].map((item) => (
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
          <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-purple-300">The authority gap</p>
          <h2 className="text-3xl font-bold text-white sm:text-4xl">Identity proves who. Rails prove value moved. SatGate proves authority.</h2>
          <p className="mt-5 text-lg leading-8 text-gray-400">
            Agents now spend money, call APIs, delegate to other agents, and act on behalf of humans and platforms. Infrastructure built for humans clicking buttons does not prove the agent had bounded authority before the action happened.
          </p>
        </div>
        <div className="grid gap-6 md:grid-cols-3">
          {layers.map((layer) => {
            const Icon = layer.icon;
            return (
              <div key={layer.title} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
                <Icon className="mb-4 text-cyan-300" />
                <h3 className="text-xl font-bold text-white">{layer.title}</h3>
                <p className="mt-3 leading-7 text-gray-400">{layer.body}</p>
              </div>
            );
          })}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <div className="mb-10 max-w-3xl">
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-emerald-300">Rail-neutral is the moat</p>
            <h2 className="text-3xl font-bold text-white sm:text-4xl">The governance contract has to travel with the agent.</h2>
            <p className="mt-5 text-lg leading-8 text-gray-400">
              Stripe cannot govern an L402 payment. Coinbase cannot govern a Stripe Connect transfer. A single-provider rail cannot be neutral across an enterprise agent stack. SatGate sits above the rails and turns authority into portable proof.
            </p>
          </div>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {audiences.map(([title, body]) => (
              <div key={title} className="rounded-2xl border border-gray-800 bg-black p-5">
                <h3 className="font-bold text-white">{title}</h3>
                <p className="mt-2 text-sm leading-6 text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="grid gap-10 lg:grid-cols-[0.9fr_1.1fr] lg:items-start">
          <div>
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-cyan-300">Public proof surface</p>
            <h2 className="text-3xl font-bold text-white sm:text-4xl">Anyone can verify a SatGate receipt. That is the point.</h2>
            <p className="mt-5 text-lg leading-8 text-gray-400">
              Evidence Packs are signed, hash-linked, and anchored by public JWKS. The open verifier reproduces the receipt hash and validates the Ed25519 signature without trusting a SatGate dashboard.
            </p>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
              <a href="https://github.com/SatGate-io/evidence-pack-verifier" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
                Open verifier <ArrowRight size={18} />
              </a>
              <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-purple-500">
                See Policy-to-Proof
              </Link>
            </div>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-black p-6">
            <div className="mb-4 flex items-center gap-2 text-cyan-200">
              <FileCheck2 size={20} /> Public specs
            </div>
            <div className="space-y-3">
              {publicSpecs.map(([label, href]) => (
                <a key={label} href={href} className="block rounded-xl border border-gray-800 bg-gray-950 p-4 text-sm text-gray-300 transition hover:border-cyan-500 hover:text-white">
                  <span className="font-semibold text-white">{label}</span>
                  <span className="mt-1 block break-all text-gray-500">{href}</span>
                </a>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <div className="mb-10 max-w-3xl">
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-purple-300">FAQ</p>
            <h2 className="text-3xl font-bold text-white sm:text-4xl">The category, without the fog.</h2>
          </div>
          <div className="grid gap-5 md:grid-cols-2">
            {faqs.map((faq) => (
              <div key={faq.question} className="rounded-2xl border border-gray-800 bg-black p-6">
                <h3 className="text-lg font-bold text-white">{faq.question}</h3>
                <p className="mt-3 leading-7 text-gray-400">{faq.answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-cyan-400/20 bg-cyan-400/10 p-8 text-center">
          <h2 className="text-3xl font-bold text-white">Agents should not get standing authority.</h2>
          <p className="mx-auto mt-4 max-w-2xl text-lg leading-8 text-gray-300">
            Give them scoped authority, enforce it before execution, and leave signed Evidence Pack proof for every decision.
          </p>
          <div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
            <Link href="/build" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Build with SatGate <ArrowRight size={18} />
            </Link>
            <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
              See Policy-to-Proof
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

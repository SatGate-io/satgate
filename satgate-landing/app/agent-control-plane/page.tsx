import type { Metadata } from "next";
import Link from "next/link";
import Image from "next/image";
import {
  ArrowRight,
  Bot,
  CheckCircle2,
  CircleDollarSign,
  FileText,
  GitBranch,
  KeyRound,
  LockKeyhole,
  ShieldCheck,
  Siren,
  TimerReset,
} from "lucide-react";

export const metadata: Metadata = {
  title: "SatGate Agent Control Plane | Govern Local AI Agents",
  description:
    "SatGate is an agent control plane for local AI agents: scoped capabilities, delegation lineage, request-path budget enforcement, audit trails, and instant revocation without standing API keys.",
  keywords: [
    "agent control plane",
    "AI agent control plane",
    "local AI agent governance",
    "AI agent delegation control",
    "agent capability management",
    "revocable agent credentials",
    "agent kill switch",
    "agent audit trail",
    "AI agent spend governance",
    "economic control plane for AI agents",
  ],
  alternates: {
    canonical: "https://satgate.io/agent-control-plane",
  },
  openGraph: {
    title: "SatGate Agent Control Plane | Govern Local AI Agents",
    description:
      "Govern local AI agents with scoped capabilities, delegation lineage, request-path budget enforcement, audit trails, and instant revocation — no standing API keys.",
    url: "https://satgate.io/agent-control-plane",
    type: "website",
    images: [
      {
        url: "/acp-demo/satgate-acp-thumbnail.jpg",
        width: 1920,
        height: 1080,
        alt: "SatGate Agent Control Plane gateway enforcement timeline",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate Agent Control Plane",
    description:
      "Control the agent. Control delegation. Prove the lineage. Meter spend. Shut it down instantly.",
    images: ["/acp-demo/satgate-acp-thumbnail.jpg"],
  },
};

const proofPoints = [
  {
    icon: KeyRound,
    title: "No standing authority",
    body: "The local agent starts with no reusable keys and no pre-granted provider access.",
  },
  {
    icon: ShieldCheck,
    title: "Scoped capability",
    body: "SatGate issues short-lived authority with route scope, spend budget, expiry, and delegation depth.",
  },
  {
    icon: GitBranch,
    title: "Delegation controlled",
    body: "The parent can delegate only narrower authority. The scoped worker cannot pass broader power onward.",
  },
  {
    icon: CircleDollarSign,
    title: "Metered and revocable",
    body: "Every action goes through the gateway, appears in usage/audit, hits spend limits, and can be killed immediately.",
  },
];

const statusCodes = [
  ["200", "Gateway-approved action", "Allowed only through SatGate"],
  ["402", "Budget cap enforced", "Spend limit stops the agent"],
  ["401", "Kill switch enforced", "Revocation cuts access immediately"],
  ["403", "Customer isolation denied", "Tenant boundary holds"],
  ["400", "Further delegation denied", "Delegation depth is exhausted"],
];

const controlPlaneCapabilities = [
  [
    "Identity and check-in",
    "The runtime starts with no reusable provider key. It checks in to receive a bounded capability for a specific route, tenant, workflow, and time window.",
  ],
  [
    "Capability-scoped authority",
    "Authority is encoded as policy: allowed routes, spend budget, expiry, delegation depth, tenant boundary, and revocation state.",
  ],
  [
    "Delegation lineage",
    "Parent agents can hand narrower authority to workers, but cannot create broader power or unlimited redelegation chains.",
  ],
  [
    "Request-path enforcement",
    "Every model, API, MCP tool, or paid action crosses SatGate before upstream access, so policy is enforced before money or data leaves.",
  ],
  [
    "Metering and audit",
    "The control plane records who acted, which capability authorized it, what it cost, and why a request was allowed, charged, or denied.",
  ],
  [
    "Kill switch and revocation",
    "Operators can revoke an agent, parent capability, delegated worker, route, tenant, or budget immediately without rotating every upstream secret.",
  ],
];

const comparisons = [
  ["API gateway", "Routes and protects API traffic", "Usually controls services, not autonomous agent authority, delegated capability, lineage, and per-agent economic limits."],
  ["MCP gateway", "Connects agents to tools", "Connection is not governance unless tool calls are metered, scoped, budgeted, revocable, and audited in the request path."],
  ["Agent observability", "Shows traces after execution", "Post-hoc visibility does not stop runaway spend, overbroad delegation, or revoked authority before the next call."],
  ["Agent control plane", "Controls runtime authority", "Defines what each agent can access, spend, delegate, prove, and lose before each upstream action is allowed."],
];

const faqs = [
  [
    "What is an agent control plane?",
    "An agent control plane is the governance layer that decides what an AI agent is allowed to access, spend, delegate, and continue doing at runtime. For SatGate, that control happens in the request path using scoped capabilities, budget policy, audit, metering, and revocation.",
  ],
  [
    "How is an agent control plane different from an API gateway?",
    "An API gateway usually manages routes, authentication, rate limits, and service protection. An agent control plane governs autonomous authority: no standing keys, delegated capability, budget enforcement, lineage proof, tenant isolation, and kill-switch revocation for agents and sub-agents.",
  ],
  [
    "Why do local AI agents need no standing authority?",
    "Standing API keys give an agent reusable power even after the task, budget, tenant, or policy context changes. SatGate keeps local agents untrusted by default; they check in, receive short-lived scoped authority, and lose that authority when policy, budget, expiry, or revocation says stop.",
  ],
  [
    "How does SatGate control agent delegation?",
    "SatGate attaches delegation depth, route scope, spend budget, expiry, and tenant caveats to agent capabilities. A parent can delegate narrower authority to a worker, but the worker cannot exceed the parent, inflate budget, cross tenants, or keep delegating after depth is exhausted.",
  ],
];

const relatedTopics = [
  ["/economic-firewall", "Economic firewall", "The request-path enforcement layer for agent access, spend, and Charge."],
  ["/agent-api-governance", "Agent API governance", "Replace broad API keys with policy-bound, auditable agent authority."],
  ["/agent-capability-tokens", "Agent capability tokens", "Encode route, budget, expiry, delegation, and revocation into agent access."],
  ["/revocable-agent-credentials", "Revocable agent credentials", "Kill agent access without rotating every upstream provider secret."],
  ["/mcp-governance", "MCP governance", "Apply budget, revocation, and audit controls to agent tool calls."],
  ["/satgate-for-hermes-agent", "SatGate for Hermes Agent", "Govern local Hermes/Open WebUI agent workflows with SatGate in the request path."],
];

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebPage",
      name: "SatGate Agent Control Plane",
      url: "https://satgate.io/agent-control-plane",
      description:
        "An agent control plane for local AI agents: scoped capabilities, delegation lineage, request-path budget enforcement, audit trails, and revocation without standing API keys.",
      isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
      about: [
        { "@type": "Thing", name: "AI agent control plane" },
        { "@type": "Thing", name: "AI agent delegation control" },
        { "@type": "Thing", name: "agent capability management" },
        { "@type": "Thing", name: "request-path budget enforcement" },
      ],
    },
    {
      "@type": "DefinedTerm",
      name: "Agent control plane",
      termCode: "agent-control-plane",
      url: "https://satgate.io/agent-control-plane",
      description:
        "A governance layer that controls what AI agents can access, spend, delegate, prove, and continue doing at runtime.",
      inDefinedTermSet: "https://satgate.io/agent-control-plane",
    },
    {
      "@type": "SoftwareApplication",
      name: "SatGate Agent Control Plane",
      applicationCategory: "SecurityApplication",
      operatingSystem: "Any",
      url: "https://satgate.io/agent-control-plane",
      offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
      featureList: [
        "No standing API keys for local AI agents",
        "Scoped agent capabilities",
        "Delegation lineage proof",
        "Request-path budget enforcement",
        "MCP tool governance",
        "Agent audit trails",
        "Instant revocation and kill switch",
      ],
    },
    {
      "@type": "FAQPage",
      mainEntity: faqs.map(([question, answer]) => ({
        "@type": "Question",
        name: question,
        acceptedAnswer: { "@type": "Answer", text: answer },
      })),
    },
    {
      "@type": "BreadcrumbList",
      itemListElement: [
        { "@type": "ListItem", position: 1, name: "Home", item: "https://satgate.io" },
        { "@type": "ListItem", position: 2, name: "Agent Control Plane", item: "https://satgate.io/agent-control-plane" },
      ],
    },
  ],
};

export default function AgentControlPlanePage() {
  return (
    <main className="min-h-screen bg-[#030711] text-white">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <nav className="border-b border-white/10 bg-black/35 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-5 sm:px-8">
          <Link href="/" className="flex items-center gap-3">
            <Image
              src="/logo_white_transparent.png"
              alt="SatGate"
              width={36}
              height={36}
              className="h-8 w-8"
            />
            <span className="text-lg font-bold tracking-tight">SatGate</span>
          </Link>
          <div className="hidden items-center gap-6 text-sm text-slate-300 md:flex">
            <Link href="/govern" className="hover:text-white">Enterprise</Link>
            <Link href="/agents" className="hover:text-white">Agents</Link>
            <a href="https://cloud.satgate.io/docs" className="hover:text-white">Docs</a>
            <a
              href="https://cloud.satgate.io/cloud/login"
              className="rounded-full border border-cyan-400/40 px-4 py-2 text-cyan-200 hover:bg-cyan-400/10"
            >
              Open Cloud
            </a>
          </div>
        </div>
      </nav>

      <section className="relative overflow-hidden border-b border-white/10 px-5 py-20 sm:px-8 lg:py-28">
        <div className="absolute left-1/2 top-0 h-[520px] w-[900px] -translate-x-1/2 rounded-full bg-purple-600/20 blur-3xl" />
        <div className="absolute right-0 top-24 h-96 w-96 rounded-full bg-cyan-500/15 blur-3xl" />
        <div className="relative mx-auto grid max-w-7xl items-center gap-12 lg:grid-cols-[0.95fr_1.05fr]">
          <div>
            <div className="mb-7 inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-200">
              <Bot size={16} />
              Agent Control Plane demo
            </div>
            <h1 className="max-w-4xl text-5xl font-black leading-[0.95] tracking-[-0.055em] text-white sm:text-6xl lg:text-7xl">
              SatGate Agent Control Plane
            </h1>
            <p className="mt-7 max-w-2xl text-xl leading-8 text-slate-300">
              A local AI agent has no standing authority. It checks in, receives scoped capability, delegates only within policy, gets metered, appears in audit, and can be shut down instantly.
            </p>
            <p className="mt-5 max-w-2xl text-lg font-semibold text-cyan-200">
              Control the agent. Control delegation. Prove the lineage. Meter spend. Shut it down instantly.
            </p>
            <div className="mt-9 flex flex-col gap-4 sm:flex-row">
              <a
                href="https://cloud.satgate.io/cloud/login"
                className="inline-flex items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-purple-600 to-cyan-500 px-6 py-4 text-base font-bold text-white shadow-2xl shadow-cyan-500/20 transition hover:scale-[1.01]"
              >
                Book a demo <ArrowRight size={18} />
              </a>
              <a
                href="/acp-demo/satgate-acp-ciso-proof-card.pdf"
                className="inline-flex items-center justify-center gap-2 rounded-2xl border border-white/15 bg-white/5 px-6 py-4 text-base font-bold text-slate-100 transition hover:bg-white/10"
              >
                Download proof card <FileText size={18} />
              </a>
            </div>
          </div>

          <div className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-3 shadow-2xl shadow-black/60">
            <video
              className="aspect-video w-full rounded-[1.35rem] bg-black object-cover"
              controls
              preload="metadata"
              poster="/acp-demo/satgate-acp-thumbnail.jpg"
            >
              <source src="/acp-demo/satgate-acp-first-touch.mp4" type="video/mp4" />
            </video>
            <div className="flex items-center justify-between gap-4 px-3 py-4 text-sm text-slate-300">
              <span>40s first-touch teaser</span>
              <a href="/acp-demo/satgate-acp-walkthrough.mp4" className="text-cyan-200 hover:text-cyan-100">
                Watch 73s walkthrough →
              </a>
            </div>
          </div>
        </div>
      </section>

      <section className="px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto max-w-7xl">
          <div className="grid gap-5 md:grid-cols-4">
            {proofPoints.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-3xl border border-white/10 bg-white/[0.035] p-6">
                <div className="mb-5 flex h-12 w-12 items-center justify-center rounded-2xl bg-cyan-400/10 text-cyan-200">
                  <Icon size={24} />
                </div>
                <h2 className="text-lg font-bold text-white">{title}</h2>
                <p className="mt-3 text-sm leading-6 text-slate-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-white/10 bg-white/[0.025] px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto grid max-w-7xl gap-10 lg:grid-cols-[0.85fr_1.15fr]">
          <div>
            <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-sm font-semibold text-cyan-200">
              <ShieldCheck size={15} />
              AI agent control plane
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
              What is an agent control plane?
            </h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              An agent control plane is the runtime governance layer for autonomous AI agents. It decides what each agent can access, spend, delegate, prove, and continue doing before the next model, API, SaaS, or MCP tool call leaves the request path.
            </p>
            <p className="mt-4 text-lg leading-8 text-slate-300">
              SatGate makes that control economic: authority carries scope, budget, expiry, delegation depth, tenant boundary, audit context, and revocation state. That is why the agent starts with no standing API key.
            </p>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            {controlPlaneCapabilities.map(([title, body]) => (
              <div key={title} className="rounded-3xl border border-white/10 bg-[#07111f] p-5">
                <h3 className="font-bold text-white">{title}</h3>
                <p className="mt-3 text-sm leading-6 text-slate-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto max-w-7xl">
          <div className="mb-9 max-w-3xl">
            <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-purple-400/30 bg-purple-400/10 px-3 py-1 text-sm font-semibold text-purple-200">
              <Bot size={15} />
              Not another gateway page
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
              Agent control plane vs API gateway vs MCP gateway
            </h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              Gateways connect and route. Observability records what happened. A control plane governs live authority, spend, delegation, and revocation while the agent is acting.
            </p>
          </div>
          <div className="grid gap-4 lg:grid-cols-4">
            {comparisons.map(([title, role, gap]) => (
              <div key={title} className="rounded-3xl border border-white/10 bg-white/[0.035] p-5">
                <h3 className="text-lg font-bold text-white">{title}</h3>
                <p className="mt-3 text-sm font-semibold text-cyan-200">{role}</p>
                <p className="mt-3 text-sm leading-6 text-slate-400">{gap}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-y border-white/10 bg-white/[0.025] px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto grid max-w-7xl gap-10 lg:grid-cols-[0.92fr_1.08fr]">
          <div>
            <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-purple-400/30 bg-purple-400/10 px-3 py-1 text-sm font-semibold text-purple-200">
              <LockKeyhole size={15} />
              Govern what power can be handed onward
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
              The proof is delegation lineage, not another chat demo.
            </h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              The governed parent agent receives a bounded capability. The scoped worker gets less scope, less budget, and zero remaining delegation depth. When it tries to delegate again, policy stops it and the audit trail records the denial.
            </p>
          </div>
          <div className="rounded-3xl border border-white/10 bg-[#061120] p-6">
            <div className="mb-5 flex items-center gap-3 text-cyan-200">
              <GitBranch size={20} />
              <span className="text-sm font-bold uppercase tracking-[0.22em]">Lineage shape</span>
            </div>
            <div className="space-y-4 font-mono text-sm">
              <div className="rounded-2xl border border-cyan-400/35 bg-cyan-400/10 p-4">
                local-agent-parent <span className="text-slate-400">scope + budget + depth=1</span>
              </div>
              <div className="ml-8 rounded-2xl border border-purple-400/40 bg-purple-400/10 p-4">
                ↳ scoped-worker <span className="text-slate-400">narrower scope + depth=0</span>
              </div>
              <div className="ml-16 rounded-2xl border border-red-400/40 bg-red-400/10 p-4 text-red-200">
                ↳ further delegation denied
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto max-w-7xl">
          <div className="mb-9 flex flex-col justify-between gap-4 md:flex-row md:items-end">
            <div>
              <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-sm font-semibold text-cyan-200">
                <Siren size={15} />
                Runtime enforcement
              </div>
              <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
                200 / 402 / 401 / 403 / 400
              </h2>
            </div>
            <p className="max-w-2xl text-slate-300">
              Buyers see policy outcomes, not raw verifier logs: allowed calls, hard budget stops, kill switch enforcement, customer isolation, and denied delegation.
            </p>
          </div>
          <div className="grid gap-4 md:grid-cols-5">
            {statusCodes.map(([code, title, body]) => (
              <div key={code} className="rounded-3xl border border-white/10 bg-white/[0.035] p-5">
                <div className="text-4xl font-black text-cyan-200">{code}</div>
                <h3 className="mt-4 text-base font-bold text-white">{title}</h3>
                <p className="mt-2 text-sm leading-6 text-slate-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-white/10 px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto grid max-w-7xl gap-10 lg:grid-cols-[1fr_0.85fr]">
          <div className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-3">
            <video className="aspect-video w-full rounded-[1.35rem] bg-black object-cover" controls preload="metadata" poster="/acp-demo/satgate-acp-thumbnail.jpg">
              <source src="/acp-demo/satgate-acp-walkthrough.mp4" type="video/mp4" />
            </video>
          </div>
          <div className="flex flex-col justify-center">
            <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-sm font-semibold text-cyan-200 w-fit">
              <TimerReset size={15} />
              Meeting walkthrough
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white">Use the 73-second version live.</h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              The short video opens the door. The walkthrough shows the full operator view: lineage proof, local runtime check-in, enforcement timeline, audit, spend, and revocation.
            </p>
            {/* Use a plain img here; Next's dev image optimizer rejects this local PNG in some environments. */}
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="/acp-demo/satgate-acp-ciso-proof-card.png"
              alt="SatGate Agent Control Plane CISO proof card"
              className="mt-8 w-full rounded-2xl border border-white/10"
            />
          </div>
        </div>
      </section>

      <section className="border-t border-white/10 px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto grid max-w-7xl gap-10 lg:grid-cols-[0.9fr_1.1fr]">
          <div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white">Agent control-plane FAQ</h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              Short answers for buyers comparing SatGate against API gateways, MCP gateways, agent frameworks, and observability tools.
            </p>
          </div>
          <div className="space-y-4">
            {faqs.map(([question, answer]) => (
              <div key={question} className="rounded-3xl border border-white/10 bg-white/[0.035] p-6">
                <h3 className="text-lg font-bold text-white">{question}</h3>
                <p className="mt-3 leading-7 text-slate-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-white/10 bg-white/[0.025] px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto max-w-7xl">
          <h2 className="text-3xl font-black tracking-[-0.04em] text-white">Related economic control-plane topics</h2>
          <div className="mt-8 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {relatedTopics.map(([href, title, body]) => (
              <Link key={href} href={href} className="rounded-3xl border border-white/10 bg-[#07111f] p-5 transition hover:border-cyan-400/50 hover:bg-cyan-400/10">
                <h3 className="font-bold text-white">{title}</h3>
                <p className="mt-3 text-sm leading-6 text-slate-400">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      <section className="px-5 pb-20 sm:px-8 lg:pb-28">
        <div className="mx-auto max-w-5xl rounded-[2rem] border border-cyan-400/25 bg-gradient-to-br from-cyan-400/10 to-purple-500/10 p-8 text-center sm:p-12">
          <CheckCircle2 className="mx-auto mb-5 text-cyan-200" size={40} />
          <h2 className="text-3xl font-black tracking-[-0.04em] text-white sm:text-4xl">
            Turn local agents into governed enterprise actors.
          </h2>
          <p className="mx-auto mt-4 max-w-2xl text-lg leading-8 text-slate-300">
            No standing API keys. No unmanaged delegation. No invisible spend.
          </p>
          <a
            href="https://cloud.satgate.io/cloud/login"
            className="mt-8 inline-flex items-center justify-center gap-2 rounded-2xl bg-white px-6 py-4 text-base font-black text-slate-950 transition hover:bg-cyan-100"
          >
            Book a demo <ArrowRight size={18} />
          </a>
        </div>
      </section>
    </main>
  );
}

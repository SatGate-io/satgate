import type { Metadata } from "next";
import Link from "next/link";
import Image from "next/image";
import {
  ArrowRight,
  Bot,
  CheckCircle2,
  CircleDollarSign,
  FileSearch,
  FileText,
  GitBranch,
  KeyRound,
  LockKeyhole,
  ShieldCheck,
  Siren,
  TimerReset,
} from "lucide-react";

export const metadata: Metadata = {
  title: "SatGate Agent Control Plane | Govern Enterprise AI Agents",
  description:
    "SatGate is an agent control plane for enterprise AI agents: scoped capabilities, delegation lineage, request-path budget enforcement, Evidence Pack exports, and next-request revocation.",
  keywords: [
    "agent control plane",
    "AI agent control plane",
    "enterprise AI agent governance",
    "AI agent delegation control",
    "agent capability management",
    "revocable agent credentials",
    "agent kill switch",
    "agent Evidence Pack",
    "AI agent spend governance",
    "economic firewall for AI agents",
  ],
  alternates: {
    canonical: "https://satgate.io/agent-control-plane",
  },
  openGraph: {
    title: "SatGate Agent Control Plane | Govern Enterprise AI Agents",
    description:
      "Govern enterprise AI agents with scoped capabilities, delegation lineage, request-path budgets, Evidence Pack exports, and next-request revocation.",
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
    title: "No permanent keys",
    body: "Agents get temporary authority for the job, not reusable secrets that linger after the work is done.",
  },
  {
    icon: ShieldCheck,
    title: "Business limits built in",
    body: "Each agent works inside the scope, budget, customer boundary, and expiration you set.",
  },
  {
    icon: GitBranch,
    title: "Delegation you can prove",
    body: "When an agent hands work to another agent, SatGate shows who delegated what — and where the limits held.",
  },
  {
    icon: CircleDollarSign,
    title: "Spend visible and stoppable",
    body: "Every action is measured before cost gets out of hand, with revocation ready when risk changes.",
  },
];

const enforcementOutcomes = [
  ["Approved work", "The agent completes authorized tasks through SatGate, with policy and tenant context attached."],
  ["Budget protected", "Spend caps stop runaway loops before the next expensive call leaves the request path."],
  ["Access revoked", "A kill switch cuts off a risky agent or delegated worker immediately."],
  ["Tenant isolated", "Customer boundaries stay intact even when agents call shared tools or APIs."],
  ["Delegation contained", "A worker can only receive the narrower authority the parent was allowed to hand off."],
];

const controlPlaneCapabilities = [
  [
    "Start with zero standing access",
    "Agents do not wake up holding broad, reusable keys. They earn temporary authority for the job in front of them.",
  ],
  [
    "Give every agent a bounded mandate",
    "Each capability carries the business limits that matter: scope, budget, tenant, expiry, delegation, and revocation.",
  ],
  [
    "Control delegated work",
    "When an agent spins up a worker, SatGate keeps the worker narrower than the parent and proves the handoff later.",
  ],
  [
    "Stop spend before it happens",
    "Budget checks run before model, API, SaaS, or MCP tool calls turn into real cost.",
  ],
  [
    "Show proof teams can trust",
    "Security and finance get a clear record of who acted, what was allowed, what was denied, and what it cost.",
  ],
  [
    "Revoke without drama",
    "Shut down an agent, worker, route, tenant, or budget without rotating every upstream provider secret.",
  ],
];

const comparisons = [
  ["Connection", "Agents can reach models, APIs, and MCP tools."],
  ["Control", "SatGate decides what authority, budget, and delegation each agent receives."],
  ["Proof", "Every allowed, denied, charged, or revoked action leaves a receipt that feeds the Evidence Pack."],
  ["Revenue", "The same control path governs paid calls across x402, L402, AgentCore Payments, and Pay.sh while preserving proof of agent authority."],
];

const personaCards = [
  ["Platform team", "Gets a control plane for enterprise agents across APIs, MCP tools, models, and paid rails."],
  ["Security team", "Gets revocation with evidence: the authority chain, denial reason, and first blocked call after revoke."],
  ["FinOps team", "Gets spend attributed to the agent, token, route, tool, and policy before finance has to reconstruct it."],
];

const faqs = [
  [
    "What is an agent control plane?",
    "An agent control plane gives teams a live way to govern AI agents: what they can access, how much they can spend, what they can delegate, and when their authority should end. SatGate puts that control in the path of each agent action.",
  ],
  [
    "How is an agent control plane different from an API gateway?",
    "An API gateway protects services. An agent control plane governs the agents using those services: their budget, customer boundary, delegation rights, Evidence Pack, and ability to have governed requests denied after revocation.",
  ],
  [
    "Why do enterprise AI agents need no standing authority?",
    "Standing API keys give an agent reusable power even after the task, budget, tenant, or policy context changes. SatGate keeps enterprise agents untrusted by default; they check in, receive short-lived scoped authority, and lose that authority when policy, budget, expiry, or revocation says stop.",
  ],
  [
    "How does SatGate control agent delegation?",
    "SatGate lets a parent agent hand off only the authority it is allowed to share. The worker gets narrower limits, the handoff is visible, and attempts to exceed policy are blocked before they become spend or risk.",
  ],
];

const relatedTopics = [
  ["/policy-to-proof", "Policy-to-Proof", "Turn every mint, delegation, paid call, denial, and revocation into receipts and exportable Evidence Pack proof."],
  ["/mcp-governance", "MCP governance", "Apply budget, revocation, and audit controls to agent tool calls."],
  ["/agent-api-governance", "Agent API governance", "Replace broad API keys with policy-bound, auditable agent authority."],
  ["/agent-capability-tokens", "Agent capability tokens", "Encode route, budget, expiry, delegation, and revocation into agent access."],
  ["/revocable-agent-credentials", "Revocable agent credentials", "Kill agent access without rotating every upstream provider secret."],
  ["/economic-firewall", "Economic firewall", "The request-path enforcement layer for agent access, spend, and governed paid calls."],
  ["/ai-agent-cost-control", "AI agent cost control", "Control model, API, MCP, and delegated sub-agent spend before cost is created."],
  ["/economic-firewall-readiness-grader", "Economic firewall readiness grader", "Score identity, budgets, revocation, audit, routing, MCP, and paid-call governance readiness."],
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
        "An agent control plane for enterprise AI agents: scoped capabilities, delegation lineage, request-path budget enforcement, Evidence Pack exports, and revocation without standing API keys.",
      datePublished: "2026-04-27",
      dateModified: "2026-05-05",
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
        "No standing API keys for enterprise AI agents",
        "Scoped agent capabilities",
        "Delegation lineage proof",
        "Request-path budget enforcement",
        "MCP tool governance",
        "Agent Evidence Packs",
        "Next-request revocation and kill switch",
      ],
    },
    {
      "@type": "ItemList",
      name: "Agent control-plane capabilities",
      description:
        "Core controls required to govern enterprise AI agents without standing keys, unmanaged delegation, or invisible spend.",
      itemListElement: controlPlaneCapabilities.map(([name, description], index) => ({
        "@type": "ListItem",
        position: index + 1,
        name,
        description,
      })),
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
              Governed AI agents
            </div>
            <h1 className="max-w-4xl text-5xl font-black leading-[0.95] tracking-[-0.055em] text-white sm:text-6xl lg:text-7xl">
              SatGate Agent Control Plane
            </h1>
            <p className="mt-7 max-w-2xl text-xl leading-8 text-slate-300">
              Govern enterprise AI agents before they touch expensive models, sensitive APIs, paid tools, or customer data. SatGate gives every agent bounded authority, visible spend, controlled delegation, and next-request revocation.
            </p>
            <p className="mt-5 max-w-2xl text-lg font-semibold text-cyan-200">
              Let agents work — without giving them blank checks, permanent keys, or invisible authority.
            </p>
            <div className="mt-9 flex flex-col gap-4 sm:flex-row">
              <a
                href="mailto:contact@satgate.io?subject=SatGate%20Agent%20Control%20Plane%20demo"
                className="inline-flex items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-purple-600 to-cyan-500 px-6 py-4 text-base font-bold text-white shadow-2xl shadow-cyan-500/20 transition hover:scale-[1.01]"
              >
                Book a demo <ArrowRight size={18} />
              </a>
              <a
                href="/acp-demo/satgate-acp-security-proof-card.pdf"
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
              <span>Agent Control Plane overview</span>
              <a href="/acp-demo/satgate-acp-walkthrough.mp4" className="text-cyan-200 hover:text-cyan-100">
                Watch the full walkthrough →
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
        <div className="mx-auto max-w-7xl">
          <div className="mb-10 max-w-3xl">
            <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-yellow-400/30 bg-yellow-400/10 px-3 py-1 text-sm font-semibold text-yellow-200">
              <FileSearch size={15} />
              Internal first, rail-aware when needed
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
              One control plane for internal agents and governed paid calls.
            </h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              Internal agent calls and governed external paid calls move through the same authority model — same scoped capability, same delegation chain, one Evidence Pack.
            </p>
          </div>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-3xl border border-cyan-400/30 bg-cyan-400/10 p-6">
              <div className="mb-3 text-xs font-bold uppercase tracking-[0.22em] text-cyan-200">Govern Authority</div>
              <h3 className="text-2xl font-black text-white">Policy enforced before execution</h3>
              <p className="mt-4 leading-7 text-slate-300">
                No scoped capability means no action. SatGate decides what each enterprise agent can access, how much it can spend, what it can delegate, and whether it can continue before the request reaches an API, model, or MCP tool.
              </p>
            </div>
            <div className="rounded-3xl border border-yellow-400/30 bg-yellow-400/10 p-6">
              <div className="mb-3 text-xs font-bold uppercase tracking-[0.22em] text-yellow-200">Preserve Proof</div>
              <h3 className="text-2xl font-black text-white">Evidence preserved across rails</h3>
              <p className="mt-4 leading-7 text-slate-300">
                Payment proves value moved. SatGate proves the agent was allowed to move it across internal APIs, MCP tools, x402, L402, AgentCore Payments, Pay.sh, API-key billing, or enterprise ledgers.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="border-y border-white/10 bg-white/[0.025] px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto grid max-w-7xl gap-10 lg:grid-cols-[0.85fr_1.15fr]">
          <div>
            <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-sm font-semibold text-cyan-200">
              <ShieldCheck size={15} />
              Built for enterprise autonomy
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
              Run autonomous agents with authority you can explain.
            </h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              AI agents are starting to call APIs, use paid tools, delegate work, and act across customer environments. A production team needs more than connection and logs — it needs a way to decide what each agent is allowed to do before the action happens.
            </p>
            <p className="mt-4 text-lg leading-8 text-slate-300">
              SatGate turns agent authority into a governed business object: scoped, budgeted, delegated, metered, audited, and revocable. The agent can move fast, but it never gets unlimited power.
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
              Why this is different
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
              From connected agents to governed agents.
            </h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              Most agent stacks focus on getting tools connected. SatGate focuses on whether the agent should be allowed to act, spend, delegate, or continue — and proving that decision afterward.
            </p>
          </div>
          <div className="grid gap-4 lg:grid-cols-4">
            {comparisons.map(([title, body]) => (
              <div key={title} className="rounded-3xl border border-white/10 bg-white/[0.035] p-5">
                <h3 className="text-lg font-bold text-white">{title}</h3>
                <p className="mt-3 text-sm leading-6 text-slate-400">{body}</p>
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
              Delegation without runaway authority
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
              See exactly how agent authority moves.
            </h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              When one agent delegates work to another, SatGate keeps the child narrower than the parent. Security teams can see the chain of authority, the budget attached to it, and where policy stopped the handoff.
            </p>
          </div>
          <div className="rounded-3xl border border-white/10 bg-[#061120] p-6">
            <div className="mb-5 flex items-center gap-3 text-cyan-200">
              <GitBranch size={20} />
              <span className="text-sm font-bold uppercase tracking-[0.22em]">Authority chain</span>
            </div>
            <div className="space-y-4 font-mono text-sm">
              <div className="rounded-2xl border border-cyan-400/35 bg-cyan-400/10 p-4">
                Finance parent agent <span className="text-slate-400">approved scope and budget</span>
              </div>
              <div className="ml-8 rounded-2xl border border-purple-400/40 bg-purple-400/10 p-4">
                ↳ Invoice-reconciler worker <span className="text-slate-400">narrower authority</span>
              </div>
              <div className="ml-16 rounded-2xl border border-red-400/40 bg-red-400/10 p-4 text-red-200">
                ↳ Customer-data export blocked by policy
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto max-w-7xl">
          <div className="mb-9 max-w-3xl">
            <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-sm font-semibold text-cyan-200">
              <ShieldCheck size={15} />
              Built for enterprise control owners
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
              The control plane speaks to platform, security, and finance.
            </h2>
          </div>
          <div className="grid gap-5 md:grid-cols-3">
            {personaCards.map(([title, body]) => (
              <div key={title} className="rounded-3xl border border-white/10 bg-[#07111f] p-6">
                <h3 className="text-xl font-bold text-white">{title}</h3>
                <p className="mt-4 leading-7 text-slate-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto max-w-7xl">
          <div className="mb-9 flex flex-col justify-between gap-4 md:flex-row md:items-end">
            <div>
              <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-sm font-semibold text-cyan-200">
                <Siren size={15} />
                Clear policy outcomes
              </div>
              <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
                Clear outcomes, not raw logs.
              </h2>
            </div>
            <p className="max-w-2xl text-slate-300">
              SatGate shows what happened in plain English: approved work, protected budget, revoked access, tenant isolation, and contained delegation.
            </p>
          </div>
          <div className="grid gap-4 md:grid-cols-5">
            {enforcementOutcomes.map(([title, body]) => (
              <div key={title} className="rounded-3xl border border-white/10 bg-white/[0.035] p-5">
                <div className="text-sm font-bold uppercase tracking-[0.18em] text-cyan-200">Outcome</div>
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
              Product walkthrough
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white">See the control loop end to end.</h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              The walkthrough shows the control loop end to end: an agent receives bounded authority, delegates safely, hits real policy checks, leaves an Evidence Pack, and can be shut down immediately.
            </p>
          </div>
        </div>
      </section>

      <section className="border-t border-white/10 bg-white/[0.025] px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto grid max-w-7xl items-center gap-10 lg:grid-cols-[0.9fr_1.1fr]">
          <div>
            <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-purple-400/30 bg-purple-400/10 px-3 py-1 text-sm font-semibold text-purple-200 w-fit">
              <FileText size={15} />
              Security proof card
            </div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white">A one-page proof card. The full lifecycle exports as an Evidence Pack.</h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              Use the proof card to explain the Agent Control Plane in a security review: no permanent keys, bounded authority, controlled delegation, visible spend, next-request revocation, and an exportable Evidence Pack for the full Policy-to-Proof lifecycle.
            </p>
            <Link href="/policy-to-proof" className="mt-5 inline-flex text-sm font-bold text-cyan-200 hover:text-cyan-100">
              See the full Evidence Pack lifecycle →
            </Link>
            <a
              href="/acp-demo/satgate-acp-security-proof-card.pdf"
              className="mt-8 inline-flex items-center justify-center gap-2 rounded-2xl bg-white px-6 py-4 text-base font-black text-slate-950 transition hover:bg-cyan-100"
            >
              Download proof card <ArrowRight size={18} />
            </a>
          </div>
          <a
            href="/acp-demo/satgate-acp-security-proof-card.pdf"
            className="block rounded-[2rem] border border-white/10 bg-white p-2 shadow-2xl shadow-black/40 transition hover:border-cyan-300/50"
            aria-label="Download SatGate Agent Control Plane proof card"
          >
            {/* Use a plain img here; Next's dev image optimizer rejects this local PNG in some environments. */}
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="/acp-demo/satgate-acp-security-proof-card.png"
              alt="SatGate Agent Control Plane proof card preview"
              className="aspect-video w-full rounded-[1.4rem] object-contain"
            />
          </a>
        </div>
      </section>

      <section className="border-t border-white/10 px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto grid max-w-7xl gap-10 lg:grid-cols-[0.9fr_1.1fr]">
          <div>
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white">Agent control-plane FAQ</h2>
            <p className="mt-5 text-lg leading-8 text-slate-300">
              SatGate is for teams that want autonomous agents in production without handing them permanent keys, unmanaged delegation, or uncontrolled spend.
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
          <h2 className="text-3xl font-black tracking-[-0.04em] text-white">Related agent-governance topics</h2>
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
            Govern enterprise agents end to end.
          </h2>
          <p className="mx-auto mt-4 max-w-2xl text-lg leading-8 text-slate-300">
            No standing API keys. No unmanaged delegation. No invisible spend.
          </p>
          <a
            href="mailto:contact@satgate.io?subject=SatGate%20Agent%20Control%20Plane%20demo"
            className="mt-8 inline-flex items-center justify-center gap-2 rounded-2xl bg-white px-6 py-4 text-base font-black text-slate-950 transition hover:bg-cyan-100"
          >
            Book a demo <ArrowRight size={18} />
          </a>
        </div>
      </section>
    </main>
  );
}

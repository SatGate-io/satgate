import type { Metadata } from "next";
import Link from "next/link";
import Image from "next/image";
import {
  ArrowRight,
  Bot,
  CheckCircle2,
  GitBranch,
  LockKeyhole,
  ShieldCheck,
  TimerReset,
} from "lucide-react";

const pageTitle = "SatGate Agent Control Plane | Limits and an off switch for AI agents";
const pageDescription =
  "Give each AI agent its own limits, let helpers get only part of them, and stop an agent and every helper it started from one place.";

export const metadata: Metadata = {
  title: pageTitle,
  description: pageDescription,
  keywords: [
    "agent control plane",
    "AI agent control plane",
    "enterprise AI agent governance",
    "AI agent delegation control",
    "agent capability management",
    "revocable agent credentials",
    "agent kill switch",
    "agent signed receipt",
    "AI agent spend governance",
    "economic firewall for AI agents",
  ],
  alternates: {
    canonical: "https://satgate.io/agent-control-plane",
  },
  openGraph: {
    title: pageTitle,
    description: pageDescription,
    url: "https://satgate.io/agent-control-plane",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: pageTitle,
    description: pageDescription,
    images: ["/og.png"],
  },
};

const demoMailto = "mailto:contact@satgate.io?subject=SatGate%20Agent%20Control%20Plane%20demo";

const heroLead =
  "Treat every model like a capable insider that can make mistakes or be tricked. SatGate gives each agent its own limits, checks each request against them before it reaches your services, and lets you stop an agent and every helper it started from one place.";

const heroSecondLine = "Your agents do the work. You keep the authority.";

const exampleTitle = "Example: one agent and one helper";

const exampleLines = [
  "Research agent: $50 budget, 3 tools, ends Friday",
  "Helper: $10 carved from that $50, 1 of those tools, can't hand off again",
  "The helper calls a tool it wasn't given: refused, with a signed receipt",
  "You revoke the research agent: the helper's next request is refused too",
];

const exampleCaption = "An example of limits you set in SatGate. Your numbers and tools will differ.";

const proofPoints = [
  {
    icon: LockKeyhole,
    title: "Limits the model can't change.",
    body: "Budgets, allowed tools and end dates live in SatGate, not in a prompt or a provider's settings. SatGate checks each request against them before it reaches the service. In Control, a request outside those limits is refused. Observe records it and does not block on budget.",
  },
  {
    icon: GitBranch,
    title: "Handoffs that only narrow.",
    body: "An agent can give a helper part of what it has: the same tools or fewer, and a budget carved out of what it has left. It can never pass on more. You can also limit how many levels of handoff a token allows, down to none.",
  },
  {
    icon: TimerReset,
    title: "One off switch for the whole chain.",
    body: "Revoke an agent and its next request is refused, along with every helper it handed work to. If a revoke can't be confirmed, the dashboard says so instead of showing it as done.",
  },
  {
    icon: ShieldCheck,
    title: "Evidence the model didn't write.",
    body: "SatGate signs a receipt when it refuses an agent over a budget, a missing payment or a permission, and when an agent spends from a budget in Control. The agent never writes or edits its own record. A SatGate Cloud receipt can be checked with the open-source verifier against our published keys.",
  },
];

const apiKeyHeading = "Why not just give the agent an API key?";
const apiKeyBody =
  "A key in an agent's settings works for anyone who copies it, has no budget, and keeps working until someone rotates it. With SatGate, the key to your service stays in SatGate. The agent gets its own token, with its own limits and an end date, and stopping it doesn't mean changing the key your other agents and apps use.";

const audiences = [
  ["Platform teams", "One place to connect agents to your APIs and MCP tools, with limits per agent instead of shared keys."],
  ["Security teams", "Signed receipts for refusals, the chain of handoffs in the dashboard, and an off switch that reaches every helper."],
  ["Finance teams", "Budgets checked before the spend in Control, and spending per agent on the Usage page."],
];

const faqs = [
  [
    "What is an agent control plane?",
    "It's the one place where you decide what each AI agent can reach, how much it can spend, who it can hand work to and when its access ends, and where you stop it. SatGate checks those limits on each request, before the request reaches your services.",
  ],
  [
    "How is an agent control plane different from an API gateway?",
    "An API gateway protects one service. SatGate follows the agent: one budget across the services it calls, the helpers it hands work to, and an off switch that reaches all of them.",
  ],
  [
    "Why shouldn't AI agents hold API keys?",
    "A key works for anyone who holds it, has no budget and lasts until someone rotates it. With SatGate, each agent gets its own token with its own limits and end date, and you can stop one agent without changing the keys anything else uses.",
  ],
  [
    "How does SatGate control handoffs between agents?",
    "An agent can give a helper only part of what it has: the same tools or fewer, and a budget carved out of what it has left. You can limit how many levels of handoff are allowed. Revoking an agent refuses its helpers' next requests too.",
  ],
];

const relatedTopics = [
  ["/policy-to-proof", "Rules and Receipts", "Turn every mint, delegation, paid call, denial, and revocation into receipts and an exportable signed receipt (Evidence Pack)."],
  ["/mcp", "MCP governance", "Apply budget, revocation, and audit controls to agent tool calls."],
  ["/agent-api-governance", "Agent API governance", "Replace broad API keys with policy-bound, auditable agent permissions."],
  ["/agent-capability-tokens", "Agent capability tokens", "Encode route, budget, expiry, delegation, and revocation into agent access."],
  ["/revocable-agent-credentials", "Revocable agent credentials", "Kill agent access without rotating every upstream provider secret."],
  ["/economic-firewall", "Economic firewall", "Enforcement before the request goes through, for agent access, spend, and governed paid calls."],
  ["/ai-agent-cost-control", "AI agent cost control", "Control model, API, MCP, and delegated sub-agent spend before cost is created."],
  ["/economic-firewall-readiness-grader", "Economic firewall readiness grader", "Score identity, budgets, revocation, audit, routing, MCP, and paid-call governance readiness."],
  ["/satgate-for-hermes-agent", "SatGate for Hermes Agent", "Govern local Hermes/Open WebUI agent workflows with SatGate checking each request before it goes through."],
];

const closingHeading = "Give every agent its own limits, and keep the off switch.";
const closingLine = "Start in Observe to see what your agents do, then turn on Control when you're ready.";

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebPage",
      name: "SatGate Agent Control Plane",
      url: "https://satgate.io/agent-control-plane",
      description: pageDescription,
      datePublished: "2026-04-27",
      dateModified: "2026-10-16",
      isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
      about: [
        { "@type": "Thing", name: "AI agent control plane" },
        { "@type": "Thing", name: "per-agent limits" },
        { "@type": "Thing", name: "agent handoffs" },
        { "@type": "Thing", name: "signed receipts" },
      ],
    },
    {
      "@type": "DefinedTerm",
      name: "Agent control plane",
      termCode: "agent-control-plane",
      url: "https://satgate.io/agent-control-plane",
      description: faqs[0][1],
      inDefinedTermSet: "https://satgate.io/agent-control-plane",
    },
    {
      "@type": "SoftwareApplication",
      name: "SatGate Agent Control Plane",
      applicationCategory: "SecurityApplication",
      operatingSystem: "Any",
      url: "https://satgate.io/agent-control-plane",
      offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
      featureList: proofPoints.map(({ title }) => title),
    },
    {
      "@type": "ItemList",
      name: "Agent control plane limits",
      description: pageDescription,
      itemListElement: proofPoints.map(({ title, body }, index) => ({
        "@type": "ListItem",
        position: index + 1,
        name: title,
        description: body,
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
          <div className="min-w-0">
            <div className="mb-7 inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-200">
              <Bot size={16} />
              Agent control plane
            </div>
            <h1 className="max-w-4xl text-5xl font-black leading-[0.95] tracking-[-0.055em] text-white sm:text-6xl lg:text-7xl">
              SatGate Agent Control Plane
            </h1>
            <p className="mt-7 max-w-2xl text-xl leading-8 text-slate-300">
              {heroLead}
            </p>
            <p className="mt-5 max-w-2xl text-lg font-semibold text-cyan-200">
              {heroSecondLine}
            </p>
            <div className="mt-9 flex flex-col gap-4 sm:flex-row">
              <a
                href={demoMailto}
                className="inline-flex items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-purple-600 to-cyan-500 px-6 py-4 text-base font-bold text-white shadow-2xl shadow-cyan-500/20 transition hover:scale-[1.01]"
              >
                Book a demo <ArrowRight size={18} />
              </a>
              <Link
                href="/blog/zero-trust-in-the-model"
                className="inline-flex items-center justify-center gap-2 rounded-2xl border border-white/15 bg-white/5 px-6 py-4 text-base font-bold text-slate-100 transition hover:bg-white/10"
              >
                Why zero trust in the model <ArrowRight size={18} />
              </Link>
            </div>
          </div>

          <div className="min-w-0 rounded-[2rem] border border-white/10 bg-white/[0.04] p-6 shadow-2xl shadow-black/60 sm:p-8">
            <h2 className="text-xl font-bold text-white">{exampleTitle}</h2>
            <ol className="mt-6 list-decimal space-y-4 pl-5 text-base leading-7 text-slate-200 marker:font-semibold marker:text-cyan-200">
              {exampleLines.map((line) => (
                <li key={line}>{line}</li>
              ))}
            </ol>
            <p className="mt-6 text-sm leading-6 text-slate-400">{exampleCaption}</p>
          </div>
        </div>
      </section>

      <section className="px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto max-w-7xl">
          <div className="grid gap-5 md:grid-cols-4">
            {proofPoints.map(({ icon: Icon, title, body }) => (
              <div key={title} className="min-w-0 rounded-3xl border border-white/10 bg-white/[0.035] p-6">
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
        <div className="mx-auto max-w-3xl">
          <h2 className="text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
            {apiKeyHeading}
          </h2>
          <p className="mt-5 text-lg leading-8 text-slate-300">
            {apiKeyBody}
          </p>
        </div>
      </section>

      <section className="px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto max-w-7xl">
          <h2 className="mb-9 text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl">
            Who it&apos;s for
          </h2>
          <div className="grid gap-5 md:grid-cols-3">
            {audiences.map(([title, body]) => (
              <div key={title} className="min-w-0 rounded-3xl border border-white/10 bg-[#07111f] p-6">
                <h3 className="text-lg font-semibold leading-8 text-slate-200">{`${title}: ${body}`}</h3>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-white/10 px-5 py-16 sm:px-8 lg:py-24">
        <div className="mx-auto grid max-w-7xl gap-10 lg:grid-cols-[0.9fr_1.1fr]">
          <div className="min-w-0">
            <h2 className="text-4xl font-black tracking-[-0.04em] text-white">Questions</h2>
          </div>
          <div className="min-w-0 space-y-4">
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
              <Link key={href} href={href} className="min-w-0 rounded-3xl border border-white/10 bg-[#07111f] p-5 transition hover:border-cyan-400/50 hover:bg-cyan-400/10">
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
            {closingHeading}
          </h2>
          <p className="mx-auto mt-4 max-w-2xl text-lg leading-8 text-slate-300">
            {closingLine}
          </p>
          <div className="mt-8 flex flex-col items-center justify-center gap-4 sm:flex-row">
            <a
              href={demoMailto}
              className="inline-flex items-center justify-center gap-2 rounded-2xl bg-white px-6 py-4 text-base font-black text-slate-950 transition hover:bg-cyan-100"
            >
              Book a demo <ArrowRight size={18} />
            </a>
            <a
              href="https://cloud.satgate.io/cloud/login"
              className="inline-flex items-center justify-center gap-2 rounded-2xl border border-white/15 bg-white/5 px-6 py-4 text-base font-bold text-slate-100 transition hover:bg-white/10"
            >
              Open SatGate Cloud <ArrowRight size={18} />
            </a>
          </div>
        </div>
      </section>
    </main>
  );
}

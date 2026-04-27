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
    "See how SatGate gives local AI agents scoped capability, delegation controls, metered spend, audit proof, and instant revocation without standing API keys.",
  alternates: {
    canonical: "https://satgate.io/agent-control-plane",
  },
  openGraph: {
    title: "SatGate Agent Control Plane | Govern Local AI Agents",
    description:
      "A local AI agent has no standing authority. It checks in, receives scoped capability, delegates only within policy, gets metered, appears in audit, and can be shut down instantly.",
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

export default function AgentControlPlanePage() {
  return (
    <main className="min-h-screen bg-[#030711] text-white">
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

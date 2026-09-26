import type { Metadata } from "next";
import Link from "next/link";
import {
  ArrowLeft,
  ArrowRight,
  BadgeCheck,
  Braces,
  CheckCircle2,
  Code2,
  KeyRound,
  Layers3,
  ReceiptText,
  Route,
  ShieldCheck,
  TerminalSquare,
} from "lucide-react";

export const metadata: Metadata = {
  title: "Build Agents With Bounded Economic Authority",
  description:
    "Complete useful agent work within an explicit budget. Discover public docs and demos, request a bounded evaluation, and inspect delegated authority and recorded outcomes.",
  keywords: [
    "SatGate build",
    "AI agent capabilities",
    "agent receipts",
    "agent governance receipts",
    "MCP capability tokens",
    "AI agent SDK",
    "Agent Authority & Accountability Layer",
  ],
  alternates: {
    canonical: "https://satgate.io/build",
  },
  openGraph: {
    title: "Build agents with bounded economic authority",
    description:
      "Start with a useful task and a finite budget. Check access, delegation, refusal and recovery on the path to your tools.",
    url: "https://satgate.io/build",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Build agents with bounded economic authority",
    description:
      "Build useful agent workflows with scoped authority and finite budgets. Explore public documentation and distinguish demonstrations from private-beta APIs.",
  },
};

const discovery = `# Read public discovery metadata; no account or installation required
curl -fsS https://satgate.io/.well-known/satgate
curl -fsS https://satgate.io/llms.txt`;

const quickstart = `import os
from satgate import SatGate

satgate = SatGate(api_key=os.getenv("SATGATE_API_KEY"))

capability = satgate.issue(
    task="research market prices",
    agent="research-agent",
    allow=["mcp:web.search", "api:prices.read"],
    budget_usd=25,
    expires_in="1h",
)

receipt = satgate.pay(
    upstream="https://api.example.com/search",
    capability=capability,
    max_usd=4.20,
)

verified = satgate.verify(receipt)
print(verified.decision, verified.evidence_pack_id)`;

const nodeExample = `import { SatGate } from "@satgate/sdk";

const satgate = new SatGate({ apiKey: process.env.SATGATE_API_KEY });

const capability = await satgate.issue({
  task: "compare supplier prices",
  agent: "procurement-agent",
  allow: ["mcp:browser.search", "api:supplier.quote"],
  budgetUsd: 25,
  expiresIn: "1h",
});

const receipt = await satgate.pay({
  upstream: "https://api.example.com/search",
  capability,
  maxUsd: 4.20,
});

const verified = await satgate.verify(receipt);
console.log(verified.decision, verified.evidencePackId);`;

const installCommands = String.raw`# Install today (public packages):
pip install satgate
npm install @satgate/sdk`;

const curlExample = String.raw`curl https://api.satgate.io/v1/issue \
  -H "Authorization: Bearer $SATGATE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "research-agent",
    "task": "research market prices",
    "allow": ["mcp:web.search", "api:prices.read"],
    "budget_usd": 25,
    "expires_in": "1h"
  }'

curl https://api.satgate.io/v1/pay \
  -H "Authorization: Bearer $SATGATE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"capability":"cap_...","upstream":"https://api.example.com/search","max_usd":4.20}'

curl https://api.satgate.io/v1/verify \
  -H "Authorization: Bearer $SATGATE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"receipt":"rcpt_..."}'`;

const primitives = [
  {
    icon: KeyRound,
    title: "Issue scoped capabilities",
    label: "satgate.issue",
    body: "Give an agent bounded authority for one task, budget, route set, expiry window, and delegation depth.",
  },
  {
    icon: Route,
    title: "Consume upstream with max budget",
    label: "satgate.pay",
    body: "Let the agent reach MCP tools, APIs, or paid routes through SatGate while a caller-supplied max budget and policy are enforced before execution or settlement.",
  },
  {
    icon: ReceiptText,
    title: "Verify receipts",
    label: "satgate.verify",
    body: "Verify the receipt returned by pay, then attach or fetch Evidence Pack proof for audits, incidents, billing review, or revocation proof.",
  },
];

const docsBase = "https://github.com/SatGate-io/satgate/blob/main/docs";

const voiceCards = [
  { title: "Agent task", label: "useful work", body: "Choose a bounded comparison or research task. Assess the result separately from its authorization evidence." },
  { title: "Owner boundary", label: "budget and scope", body: "Define a finite allowance and permitted tools. Prevent direct upstream access from bypassing the gateway." },
  { title: "Independent evidence", label: "verify the outcome", body: "Verify the recorded authority and accounting against independently admitted trust. A receipt does not establish answer quality." },
];

const trustMetadataNote = 'Trust metadata documents capability acceptance, receipt verification fields, and is canonical for rail adapter status.';

const buildDocLinks = [
  { title: "Gateway quickstart", href: `${docsBase}/getting-started/quickstart.md`, body: "Local OSS gateway instructions. Read prerequisites and separate the optional paid-route examples from internal budget control." },
  { title: "Capability schema", href: `${docsBase}/reference/capability-schema.md`, body: "The bounded authority contract: issuer, subject, allowlist, budget, expiry, caveats, and delegation depth." },
  { title: "Receipt schema", href: `${docsBase}/reference/receipt-schema.md`, body: "The signed decision artifact for allowed, denied, delegated, revoked, and paid outcomes." },
  { title: "Trust metadata", href: `${docsBase}/reference/satgate-trust-metadata.md`, body: trustMetadataNote },
  { title: "Open verifier", href: "https://github.com/SatGate-io/evidence-pack-verifier", body: "Verify a live Evidence Pack from the issuer JWKS with RFC8785 canonicalization and Ed25519 signatures." },
  { title: "MCP integration", href: `${docsBase}/guides/mcp-gateway.md`, body: "Put SatGate in front of MCP tools and preserve a receipt per tool invocation." },
  { title: "Raw HTTP", href: `${docsBase}/guides/raw-http.md`, body: "Copy-paste curl for issue, pay, and verify without an SDK." },
  { title: "OpenAI tools", href: `${docsBase}/guides/openai-tools.md`, body: "Wrap OpenAI tool execution with SatGate authority and receipt verification." },
  { title: "Anthropic tools", href: `${docsBase}/guides/anthropic-tools.md`, body: "Govern Anthropic tool use outside the provider boundary." },
  { title: "LangChain", href: `${docsBase}/guides/langchain-integration.md`, body: "Keep LangChain orchestration, add SatGate at the tool authority boundary." },
  { title: "CrewAI", href: `${docsBase}/guides/crewai.md`, body: "Give each CrewAI tool wrapper scoped authority and Evidence Pack proof." },

  { title: "API overview", href: `${docsBase}/api/overview.md`, body: "Low-level gateway compatibility APIs and how they relate to issue/pay/verify." },
  { title: "Python SDK", href: `${docsBase}/sdks/python.md`, body: "Python SDK setup and compatibility paths." },
  { title: "Node.js SDK", href: `${docsBase}/sdks/nodejs.md`, body: "Node.js SDK setup and compatibility paths." },
];

const runtimeChips = [
  { label: "MCP", href: `${docsBase}/guides/mcp-gateway.md` },
  { label: "OpenAI tools", href: `${docsBase}/guides/openai-tools.md` },
  { label: "Anthropic tools", href: `${docsBase}/guides/anthropic-tools.md` },
  { label: "LangChain", href: `${docsBase}/guides/langchain-integration.md` },
  { label: "CrewAI", href: `${docsBase}/guides/crewai.md` },
  { label: "Raw HTTP", href: `${docsBase}/guides/raw-http.md` },
];

const allowedReceipt = {
  receipt_id: "rcpt_7J4xQf9",
  decision: "allowed",
  decision_reason: "capability_scope_and_budget_ok",
  agent_id: "research-agent",
  capability_id: "cap_2Xn83k",
  policy_version: "policy_2026_05_build_v1",
  route_or_tool: "api.example.com/search",
  amount_usd: "0.42",
  rail: "enterprise_ledger",
  evidence_pack_id: "ep_2026_05_12_001",
  signature: "ed25519:demo_redacted",
};

const deniedReceipt = {
  receipt_id: "rcpt_9Kp1vM2",
  decision: "denied",
  decision_reason: "budget_exhausted",
  agent_id: "research-agent",
  capability_id: "cap_2Xn83k",
  policy_version: "policy_2026_05_build_v1",
  route_or_tool: "api.example.com/search",
  attempted_amount_usd: "4.20",
  remaining_budget_usd: "0.00",
  evidence_pack_id: "ep_2026_05_12_001",
  signature: "ed25519:demo_redacted",
};

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebPage",
      name: "Build agents with bounded economic authority",
      url: "https://satgate.io/build",
      description: metadata.description,
      datePublished: "2026-05-12",
      dateModified: "2026-09-26",
      isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
      about: [
        { "@type": "Thing", name: "Agent Authority & Accountability Layer" },
        { "@type": "Thing", name: "agent capabilities" },
        { "@type": "Thing", name: "verifiable receipts" },
        { "@type": "Thing", name: "rail-neutral payment governance" },
      ],
    },
    {
      "@type": "SoftwareSourceCode",
      name: "SatGate issue/pay/verify private-beta illustration",
      codeSampleType: "code snippet",
      programmingLanguage: "Python",
      text: quickstart,
    },
  ],
};

export default function BuildPage() {
  return (
    <main className="min-h-screen overflow-x-hidden bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />

      <div className="mx-auto max-w-6xl px-6 pt-8">
        <Link href="/" className="inline-flex items-center gap-2 text-sm font-medium text-gray-500 transition hover:text-white">
          <ArrowLeft size={16} /> Back to Home
        </Link>
      </div>

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(56,189,248,0.18),transparent_34%),radial-gradient(circle_at_85%_10%,rgba(168,85,247,0.18),transparent_32%),radial-gradient(circle_at_55%_80%,rgba(16,185,129,0.10),transparent_34%)]" />
        <div className="relative mx-auto grid max-w-6xl min-w-0 gap-12 px-6 py-24 lg:grid-cols-[0.95fr_1.05fr] lg:items-center">
          <div>
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-2 text-sm font-semibold uppercase tracking-[0.22em] text-cyan-200">
              <Code2 size={16} /> Agent starting point
            </div>
            <h1 className="max-w-4xl text-5xl font-black tracking-tight text-white sm:text-6xl lg:text-7xl">
              Build agents with bounded economic authority
            </h1>
            <p className="mt-6 max-w-3xl text-xl leading-8 text-gray-300">
              Complete a useful task within an explicit budget. Discover the available tools, delegate narrower allowances, and handle refusal without bypassing your owner’s policy.
            </p>
            <p className="mt-5 max-w-3xl text-lg leading-8 text-gray-400">
              Agents can choose SatGate for a useful workflow. Owners still need an enforced boundary on the protected-call path, independent of that choice.
            </p>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
              <a
                href="https://github.com/SatGate-io/satgate/blob/main/docs/index.md"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200"
              >
                Explore public docs <ArrowRight size={18} />
              </a>
              <a
                href="https://github.com/SatGate-io/satgate"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-400"
              >
                View public source <TerminalSquare size={18} />
              </a>
            </div>
          </div>

          <div className="min-w-0 rounded-2xl border border-cyan-900/50 bg-gray-950/90 shadow-2xl shadow-cyan-950/30">
            <div className="flex items-center gap-2 border-b border-gray-800 px-4 py-3">
              <div className="h-3 w-3 rounded-full bg-red-500" />
              <div className="h-3 w-3 rounded-full bg-yellow-500" />
              <div className="h-3 w-3 rounded-full bg-green-500" />
              <span className="ml-2 text-xs font-mono text-gray-500">Discover before installing</span>
            </div>
            <pre className="max-w-full overflow-x-auto p-5 text-sm leading-6 text-gray-300"><code>{discovery}</code></pre>
            <div className="border-t border-gray-800 p-5">
              <p className="mb-2 text-xs font-mono uppercase tracking-[0.18em] text-cyan-300">SDK access</p>
              <pre className="max-w-full overflow-x-auto rounded-xl bg-black p-4 text-sm leading-6 text-gray-300"><code>{installCommands}</code></pre>
              <p className="mt-3 text-sm leading-6 text-gray-500">
                The issue/pay/verify API namespace is in private beta. <a href="https://github.com/SatGate-io/satgate/blob/main/docs/index.md" target="_blank" rel="noopener noreferrer" className="text-cyan-300 hover:text-cyan-200">Request access →</a>
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 pt-16">
        <h2 className="text-3xl font-bold text-white">Public paths you can inspect today</h2>
        <p className="mt-4 max-w-3xl text-lg text-gray-400">The public kit is a Synthetic HTTP demo with simulated traffic and local-only credentials. It demonstrates price discovery, allowed calls and budget exhaustion. It does not establish trusted issuer identity, delegated-worker accounting, MCP behavior or live settlement.</p>
        <div className="mt-6 flex flex-wrap gap-4">
          <a href="https://github.com/SatGate-io/satgate/tree/main/demo" className="rounded-lg bg-white px-6 py-3 font-bold text-black">Read the public demo instructions</a>
          <Link href="/sandbox#golden-path" className="rounded-lg border border-cyan-600 px-6 py-3 font-bold text-cyan-200">Try the browser simulation</Link>
          <Link href="/design-partners" className="rounded-lg border border-gray-700 px-6 py-3 font-bold text-white">Request a bounded evaluation</Link>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-10 max-w-3xl">
          <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-cyan-300">Start with the task</p>
          <h2 className="text-3xl font-bold text-white sm:text-4xl">Useful work inside an enforced boundary.</h2>
          <p className="mt-4 text-lg leading-8 text-gray-400">
            Economic defense protects the resources behind the task. Observe measures activity; Control applies internal budgets and scope; Admit authorizes external access. Proof records decisions across all three.
          </p>
        </div>
        <div className="grid gap-5 md:grid-cols-3">
          {voiceCards.map((item) => (
            <div key={item.title} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
              <div className="mb-3 inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/10 px-3 py-1 font-mono text-xs text-cyan-200">{item.label}</div>
              <h3 className="mb-3 text-2xl font-bold text-white">{item.title}</h3>
              <p className="leading-relaxed text-gray-400">{item.body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-10 max-w-3xl">
          <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-emerald-300">Private-beta API</p>
          <h2 className="text-3xl font-bold text-white sm:text-4xl">Issue. Pay. Verify.</h2>
          <p className="mt-4 text-lg leading-8 text-gray-400">
            The issue/pay/verify API namespace is in private beta. The examples below illustrate that API, not an available end-to-end fresh-agent installation. Public SDK packages can be installed separately.
          </p>
        </div>
        <div className="grid gap-5 md:grid-cols-3">
          {primitives.map(({ icon: Icon, title, label, body }) => (
            <div key={title} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
              <Icon className="mb-5 text-cyan-300" size={30} />
              <div className="mb-3 inline-flex rounded-full border border-gray-800 bg-black px-3 py-1 font-mono text-xs text-gray-400">{label}</div>
              <h3 className="mb-3 text-2xl font-bold text-white">{title}</h3>
              <p className="leading-relaxed text-gray-400">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto grid max-w-6xl min-w-0 gap-10 px-6 py-20 lg:grid-cols-[0.9fr_1.1fr] lg:items-start">
          <div>
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-purple-300">Rail-neutral by design</p>
            <h2 className="text-3xl font-bold text-white sm:text-4xl">Authority and evidence sit above the rail.</h2>
            <div className="mt-5 space-y-5 text-lg leading-8 text-gray-400">
              <p>
                Control/Fiat402 enforces internal allowances without requiring payment settlement. Admit checks scoped external access and any payment required by the configured route. Authorization remains separate from settlement.
              </p>
              <p>
                Consult <a href="https://satgate.io/.well-known/satgate" className="text-cyan-300 hover:text-cyan-200">/.well-known/satgate</a> for published adapter status. AgentCore Payments and Pay.sh remain planned. Confirm the chosen adapter in your evaluation rather than assuming that a guide establishes runtime support.
              </p>
              <p>
                Humans and platforms deploy the policies. Agents consume capabilities. Upstreams receive verifiable proof that the action was authorized, bounded, and recorded.
              </p>
            </div>
          </div>
          <div className="min-w-0 rounded-2xl border border-gray-800 bg-black p-6">
            <div className="mb-4 flex items-center gap-2 text-emerald-200">
              <BadgeCheck size={20} /> Illustrative receipt previews
            </div>
            <div className="grid gap-4">
              <div>
                <div className="mb-2 text-xs font-mono uppercase tracking-[0.18em] text-emerald-300">Allowed</div>
                <pre className="max-w-full overflow-x-auto rounded-xl bg-gray-950 p-5 text-sm leading-6 text-gray-300"><code>{JSON.stringify(allowedReceipt, null, 2)}</code></pre>
              </div>
              <div>
                <div className="mb-2 text-xs font-mono uppercase tracking-[0.18em] text-red-300">Denied</div>
                <pre className="max-w-full overflow-x-auto rounded-xl bg-gray-950 p-5 text-sm leading-6 text-gray-300"><code>{JSON.stringify(deniedReceipt, null, 2)}</code></pre>
              </div>
            </div>
            <div className="mt-4 flex flex-wrap gap-3 text-sm">
              <Link href="/policy-to-proof" className="text-cyan-300 hover:text-cyan-200">What is evidence_pack_id? →</Link>
              <Link href="/security" className="text-cyan-300 hover:text-cyan-200">How are signatures verified? →</Link>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-10 flex flex-col justify-between gap-6 lg:flex-row lg:items-end">
          <div>
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-cyan-300">Private-beta examples</p>
            <h2 className="text-3xl font-bold text-white sm:text-4xl">Illustrative calls for accounts with beta access.</h2>
          </div>
        </div>

        <div className="mb-6 min-w-0 rounded-2xl border border-gray-800 bg-gray-950">
          <div className="border-b border-gray-800 px-5 py-3 font-mono text-xs text-gray-500">Python example (private beta)</div>
          <pre className="max-w-full overflow-x-auto p-5 text-sm leading-6 text-gray-300"><code>{quickstart}</code></pre>
        </div>
        <div className="grid min-w-0 gap-6 lg:grid-cols-2">
          <div className="min-w-0 rounded-2xl border border-gray-800 bg-gray-950">
            <div className="border-b border-gray-800 px-5 py-3 font-mono text-xs uppercase tracking-[0.18em] text-gray-500">Node example</div>
            <pre className="max-w-full overflow-x-auto p-5 text-sm leading-6 text-gray-300"><code>{nodeExample}</code></pre>
          </div>
          <div className="min-w-0 rounded-2xl border border-gray-800 bg-gray-950">
            <div className="border-b border-gray-800 px-5 py-3 font-mono text-xs uppercase tracking-[0.18em] text-gray-500">HTTP example</div>
            <pre className="max-w-full overflow-x-auto p-5 text-sm leading-6 text-gray-300"><code>{curlExample}</code></pre>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <div className="mb-10 max-w-3xl">
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-emerald-300">Agent integrations</p>
            <h2 className="text-3xl font-bold text-white sm:text-4xl">Give every runtime bounded authority.</h2>
            <p className="mt-4 text-lg leading-8 text-gray-400">
              The runtime changes. The contract stays the same: capability before action, receipt after decision. The machine-readable trust metadata lives at <a href="https://satgate.io/.well-known/satgate" className="text-cyan-300 hover:text-cyan-200">/.well-known/satgate</a>.
            </p>
            <div className="mt-6 flex flex-wrap gap-2 text-sm text-gray-300">
              <span className="mr-1 py-1 text-gray-500">Integration guides:</span>
              {runtimeChips.map((chip) => (
                <a key={chip.label} href={chip.href} target="_blank" rel="noopener noreferrer" className="rounded-full border border-gray-800 bg-black px-3 py-1 transition hover:border-cyan-400 hover:text-white">
                  {chip.label}
                </a>
              ))}
            </div>
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            {buildDocLinks.map((item) => {
              const card = (
                <>
                  <div className="mb-3 flex items-center gap-2 text-white">
                    <CheckCircle2 className="text-emerald-300" size={18} />
                    <h3 className="font-bold">{item.title}</h3>
                  </div>
                  <p className="text-sm leading-6 text-gray-400">{item.body}</p>
                  <div className="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-cyan-300">Open doc <ArrowRight size={14} /></div>
                </>
              );
              return (
                <a key={item.title} href={item.href} target="_blank" rel="noopener noreferrer" className="rounded-xl border border-gray-800 bg-black p-5 transition hover:border-cyan-500">
                  {card}
                </a>
              );
            })}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-cyan-900/50 bg-gradient-to-br from-cyan-950/30 via-gray-950 to-purple-950/30 p-8 md:p-10">
          <div className="grid gap-8 lg:grid-cols-[0.9fr_1.1fr] lg:items-center">
            <div>
              <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm text-gray-300">
                <ShieldCheck size={16} /> Not a new marketplace. Not a separate brand.
              </div>
              <h2 className="text-3xl font-bold text-white sm:text-4xl">Delegation, refusal and recovery</h2>
              <p className="mt-5 text-lg leading-8 text-gray-300">
                Give each worker narrower authority and enough reservation capacity for its task. Exercise an out-of-scope or exhausted-budget refusal. After an ambiguous result, recover the same operation before starting another; do not treat an unknown outcome as a failed call.
              </p>
            </div>
            <div className="grid gap-3 text-sm text-gray-300">
              {[
                "Finite task and worker allowances",
                "Receipts for allowed, denied, delegated, revoked, and paid decisions",
                "Useful results graded separately from proof",
                "Same-operation recovery without double spending",
                "Owner-enforced policy on the protected-call path",
              ].map((item) => (
                <div key={item} className="flex items-start gap-3 rounded-xl border border-gray-800 bg-black/60 p-4">
                  <Layers3 className="mt-0.5 text-cyan-300" size={18} />
                  <span>{item}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black px-6 py-20 text-center">
        <Braces className="mx-auto mb-6 text-cyan-300" size={36} />
        <h2 className="mx-auto max-w-3xl text-3xl font-bold text-white sm:text-4xl">Evaluate one useful workflow.</h2>
        <p className="mx-auto mt-5 max-w-2xl text-lg leading-8 text-gray-400">
          For a fresh-agent evaluation with delegated workers, request a bounded evaluation. Confirm setup, available integrations and authority before starting. Public demo results and private-beta examples are not a complete evaluation package.
        </p>
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <a href="https://github.com/SatGate-io/satgate/blob/main/docs/index.md" target="_blank" rel="noopener noreferrer" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
            Open docs <ArrowRight size={18} />
          </a>
          <Link href="/capability-auth" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-400">
            Capability auth model <ArrowRight size={18} />
          </Link>
        </div>
      </section>
    </main>
  );
}

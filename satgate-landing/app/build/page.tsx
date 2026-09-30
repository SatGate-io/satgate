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
  title: "Build Agents That Work Within a Budget",
  description:
    "Give each agent a task, a budget and a list of tools it may use. Read the public docs and demo, start a 14-day free trial, and check every decision with a signed receipt.",
  keywords: [
    "SatGate build",
    "AI agent capabilities",
    "agent receipts",
    "agent governance receipts",
    "MCP capability tokens",
    "AI agent SDK",
    "Agent Permissions and Receipts",
  ],
  alternates: {
    canonical: "https://satgate.io/build",
  },
  openGraph: {
    title: "Build agents that work within a budget",
    description:
      "Give an agent a task and a budget. SatGate checks every tool call, lets you hand smaller budgets to sub-agents, and signs a receipt when it says no.",
    url: "https://satgate.io/build",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "Build agents that work within a budget",
    description:
      "Give agents a task, a budget and a list of allowed tools. Read the public docs, and see which APIs are public and which are in private beta.",
  },
};

const discovery = `# Public metadata. No account or install needed
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
    title: "Issue a token",
    label: "satgate.issue",
    body: "Give an agent a token for one task: its budget, the routes it may call, when it expires and how many times it can hand work off.",
  },
  {
    icon: Route,
    title: "Make the call",
    label: "satgate.pay",
    body: "The agent calls MCP tools, APIs or paid routes through SatGate. SatGate checks your rules and the agent’s spending cap before the call runs or any money moves.",
  },
  {
    icon: ReceiptText,
    title: "Check the receipt",
    label: "satgate.verify",
    body: "Verify the receipt the call returned. Add it to a signed receipt (Evidence Pack) for audits, incident reviews, billing questions or proof of a revoke.",
  },
];

const docsBase = "https://github.com/SatGate-io/satgate/blob/main/docs";

const voiceCards = [
  { title: "The task", label: "the task", body: "Pick a task with a clear finish, like comparing prices or researching a topic. Judge the result on its own. The receipt only shows what was allowed." },
  { title: "Your limits", label: "budget and tools", body: "Set a budget and the tools the agent may use. Make sure the agent can’t reach the API directly and skip SatGate." },
  { title: "Receipts anyone can check", label: "check the result", body: "Check the receipts against SatGate’s published keys. A receipt shows what was allowed and what it cost, not whether the answer was right." },
];

const trustMetadataNote = 'What SatGate publishes for machines: which tokens it accepts, how to verify receipts, and which payment methods are live.';

const buildDocLinks = [
  { title: "Gateway quickstart", href: `${docsBase}/getting-started/quickstart.md`, body: "Run the open-source gateway on your own machine. Paid routes are optional; you can start with budgets only." },
  { title: "Capability schema", href: `${docsBase}/reference/capability-schema.md`, body: "What a token holds: who issued it, who it’s for, allowed tools, budget, expiry, extra limits and how far it can be handed off." },
  { title: "Receipt schema", href: `${docsBase}/reference/receipt-schema.md`, body: "The signed record SatGate writes when it allows, refuses, hands off, revokes or charges." },
  { title: "Trust metadata", href: `${docsBase}/reference/satgate-trust-metadata.md`, body: trustMetadataNote },
  { title: "Open verifier", href: "https://github.com/SatGate-io/evidence-pack-verifier", body: "Check a live receipt against the issuer’s published keys (JWKS). Uses RFC 8785 canonical JSON and Ed25519 signatures." },
  { title: "MCP integration", href: `${docsBase}/guides/mcp-gateway.md`, body: "Put SatGate in front of MCP tools and get a receipt for each tool call." },
  { title: "Raw HTTP", href: `${docsBase}/guides/raw-http.md`, body: "Copy-paste curl commands for issue, pay and verify. No SDK needed." },
  { title: "OpenAI tools", href: `${docsBase}/guides/openai-tools.md`, body: "Run OpenAI tool calls through SatGate and verify the receipts." },
  { title: "Anthropic tools", href: `${docsBase}/guides/anthropic-tools.md`, body: "Enforce limits on Anthropic tool use in your own gateway, not inside the model provider." },
  { title: "LangChain", href: `${docsBase}/guides/langchain-integration.md`, body: "Keep LangChain for orchestration and add SatGate where the tools are called." },
  { title: "CrewAI", href: `${docsBase}/guides/crewai.md`, body: "Give each CrewAI tool its own token and a receipt for every call." },

  { title: "API overview", href: `${docsBase}/api/overview.md`, body: "The lower-level gateway APIs and how they map to issue, pay and verify." },
  { title: "Python SDK", href: `${docsBase}/sdks/python.md`, body: "Install and set up the Python SDK." },
  { title: "Node.js SDK", href: `${docsBase}/sdks/nodejs.md`, body: "Install and set up the Node.js SDK." },
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
      name: "Build agents that work within a budget",
      url: "https://satgate.io/build",
      description: metadata.description,
      datePublished: "2026-05-12",
      dateModified: "2026-09-26",
      isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
      about: [
        { "@type": "Thing", name: "Agent Permissions and Receipts" },
        { "@type": "Thing", name: "agent capabilities" },
        { "@type": "Thing", name: "verifiable receipts" },
        { "@type": "Thing", name: "agent payment controls" },
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
              <Code2 size={16} /> For developers
            </div>
            <h1 className="max-w-4xl text-5xl font-black tracking-tight text-white sm:text-6xl lg:text-7xl">
              Build agents that work within a budget
            </h1>
            <p className="mt-6 max-w-3xl text-xl leading-8 text-gray-300">
              Give your agent a task and a budget. It can see which tools it may use, hand smaller budgets to sub-agents, and handle a refusal without working around your rules.
            </p>
            <p className="mt-5 max-w-3xl text-lg leading-8 text-gray-400">
              Agents may choose SatGate because it helps them get work done. Your limits have to hold either way, so SatGate sits between the agent and the tools it calls.
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
              <span className="ml-2 text-xs font-mono text-gray-500">Look before you install</span>
            </div>
            <pre className="max-w-full overflow-x-auto p-5 text-sm leading-6 text-gray-300"><code>{discovery}</code></pre>
            <div className="border-t border-gray-800 p-5">
              <p className="mb-2 text-xs font-mono uppercase tracking-[0.18em] text-cyan-300">SDK access</p>
              <pre className="max-w-full overflow-x-auto rounded-xl bg-black p-4 text-sm leading-6 text-gray-300"><code>{installCommands}</code></pre>
              <p className="mt-3 text-sm leading-6 text-gray-500">
                The issue, pay and verify API is in private beta. <a href="https://github.com/SatGate-io/satgate/blob/main/docs/index.md" target="_blank" rel="noopener noreferrer" className="text-cyan-300 hover:text-cyan-200">Request access →</a>
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 pt-16">
        <h2 className="text-3xl font-bold text-white">What you can try today</h2>
        <p className="mt-4 max-w-3xl text-lg text-gray-400">The public demo is a simulated HTTP demo that runs on your machine with local test keys. It shows an agent finding the price, making allowed calls and running out of budget. It doesn’t cover trusted issuers, sub-agent budgets, MCP or real payments.</p>
        <div className="mt-6 flex flex-wrap gap-4">
          <a href="https://github.com/SatGate-io/satgate/tree/main/demo" className="rounded-lg bg-white px-6 py-3 font-bold text-black">Read the demo instructions</a>
          <a href="https://cloud.satgate.io/cloud/signup" target="_blank" rel="noopener noreferrer" className="rounded-lg border border-gray-700 px-6 py-3 font-bold text-white">Start free trial</a>
          <Link href="/sandbox#golden-path" className="rounded-lg border border-cyan-600 px-6 py-3 font-bold text-cyan-200">Try the demo in your browser</Link>
          <Link href="/design-partners" className="rounded-lg border border-gray-700 px-6 py-3 font-bold text-white">Talk to us about a pilot</Link>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-10 max-w-3xl">
          <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-cyan-300">Start with the task</p>
          <h2 className="text-3xl font-bold text-white sm:text-4xl">Real work, within limits you set.</h2>
          <p className="mt-4 text-lg leading-8 text-gray-400">
            SatGate protects the APIs and tools the task depends on. Observe shows usage without blocking. Control enforces your agents’ budgets and permissions. Admit charges external agents on paid routes, after checking they’re allowed in. All three sign receipts anyone can check. A receipt is a record, not a compliance guarantee.
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
            The issue, pay and verify API is in private beta. The examples below show how it works; you can’t run them end to end without beta access. The public SDK packages install on their own.
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
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-purple-300">Developer details</p>
            <h2 className="text-3xl font-bold text-white sm:text-4xl">Payments and internal budgets.</h2>
            <div className="mt-5 space-y-5 text-lg leading-8 text-gray-400">
              <p>
                Fiat402 handles internal budgets: your agent spends from its budget and no money moves. L402 handles paid access for external agents: a Lightning payment unlocks the route. Either way, SatGate still checks permissions. Paying never replaces them.
              </p>
              <p>
                See <a href="https://satgate.io/.well-known/satgate" className="text-cyan-300 hover:text-cyan-200">/.well-known/satgate</a> for published adapter status, meaning which payment methods are live. AgentCore Payments and Pay.sh remain planned. A guide for a payment method doesn’t mean it’s live, so check before you build on it.
              </p>
              <p>
                You set the rules. Agents use the tokens you give them. The APIs they call get proof that each request was allowed, within budget and recorded.
              </p>
            </div>
          </div>
          <div className="min-w-0 rounded-2xl border border-gray-800 bg-black p-6">
            <div className="mb-4 flex items-center gap-2 text-emerald-200">
              <BadgeCheck size={20} /> Example receipts
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
            <h2 className="text-3xl font-bold text-white sm:text-4xl">Example calls for accounts with beta access.</h2>
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
            <h2 className="text-3xl font-bold text-white sm:text-4xl">Works with the framework you already use.</h2>
            <p className="mt-4 text-lg leading-8 text-gray-400">
              Whatever framework you use, the pattern is the same: the agent gets a token before it acts and a receipt after SatGate decides. The machine-readable details are at <a href="https://satgate.io/.well-known/satgate" className="text-cyan-300 hover:text-cyan-200">/.well-known/satgate</a>.
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
                <ShieldCheck size={16} /> What to test
              </div>
              <h2 className="text-3xl font-bold text-white sm:text-4xl">Handoffs, refusals and retries</h2>
              <p className="mt-5 text-lg leading-8 text-gray-300">
                Give each sub-agent a smaller budget and fewer permissions, with enough set aside to finish its part. Try a request it isn’t allowed to make and one that runs past its budget, and check that both are refused. If a call times out and you don’t know whether it went through, check whether your setup can look up that same call before you retry. Treat an unknown result as unknown, not as a failure.
              </p>
            </div>
            <div className="grid gap-3 text-sm text-gray-300">
              {[
                "A set budget for the task and for each sub-agent",
                "Receipts for every allow, refusal, handoff, revoke and payment",
                "The result judged on its own, apart from the receipts",
                "Check that a retried call isn’t charged twice",
                "Your rules enforced between the agent and your tools",
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
        <h2 className="mx-auto max-w-3xl text-3xl font-bold text-white sm:text-4xl">Try it on one real workflow.</h2>
        <p className="mx-auto mt-5 max-w-2xl text-lg leading-8 text-gray-400">
          To test a new agent with sub-agents, talk to us about a pilot. We’ll agree on the setup, the integrations and the limits before you start. The public demo and the beta examples alone won’t give you the full picture.
        </p>
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <a href="https://github.com/SatGate-io/satgate/blob/main/docs/index.md" target="_blank" rel="noopener noreferrer" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
            Open docs <ArrowRight size={18} />
          </a>
          <Link href="/capability-auth" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-400">
            How tokens work <ArrowRight size={18} />
          </Link>
        </div>
      </section>
    </main>
  );
}

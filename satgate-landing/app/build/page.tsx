import type { Metadata } from "next";
import Link from "next/link";
import {
  ArrowRight,
  BadgeCheck,
  Braces,
  CheckCircle2,
  Code2,
  FileCheck2,
  KeyRound,
  Layers3,
  ReceiptText,
  Route,
  ShieldCheck,
  TerminalSquare,
} from "lucide-react";

export const metadata: Metadata = {
  title: "Build Agents That Can Spend Safely",
  description:
    "Issue scoped capabilities, route paid calls, and verify receipts with SatGate's developer surface for agent authority, payment context, and Evidence Pack proof.",
  keywords: [
    "SatGate build",
    "AI agent capabilities",
    "agent receipts",
    "agent payment rails",
    "MCP capability tokens",
    "AI agent SDK",
    "Economic Firewall for AI agents",
  ],
  alternates: {
    canonical: "https://satgate.io/build",
  },
  openGraph: {
    title: "Build agents that can spend safely",
    description:
      "Capabilities in. Receipts out. Rails abstracted. Build agents with scoped authority, paid-call routing, and verifiable receipts.",
    url: "https://satgate.io/build",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Build agents that can spend safely",
    description:
      "Issue capabilities, route paid calls, and verify receipts with SatGate's rail-neutral developer primitive.",
  },
};

const quickstart = `from satgate import SatGate

satgate = SatGate(api_key=os.environ["SATGATE_API_KEY"])

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

const installCommands = String.raw`# Public packages are available today.
# The issue/pay/verify namespace is private beta API access.
pip install satgate
npm install @satgate/sdk`;

const curlExample = String.raw`curl https://api.satgate.io/v1/capabilities \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "research-agent",
    "task": "research market prices",
    "allow": ["mcp:web.search", "api:prices.read"],
    "budget_usd": 25,
    "expires_in": "1h"
  }'

curl https://api.satgate.io/v1/pay \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"capability":"cap_...","upstream":"https://api.example.com/search","max_usd":4.20}'`;

const primitives = [
  {
    icon: KeyRound,
    title: "Issue scoped capabilities",
    label: "satgate.issue",
    body: "Give an agent bounded authority for one task, budget, route set, expiry window, and delegation depth.",
  },
  {
    icon: Route,
    title: "Route paid calls",
    label: "satgate.pay",
    body: "Let the agent reach MCP tools, APIs, or paid rails through SatGate while policy is enforced before value moves.",
  },
  {
    icon: ReceiptText,
    title: "Verify receipts",
    label: "satgate.verify",
    body: "Return a signed decision receipt that can feed an Evidence Pack for audits, incidents, billing review, or revocation proof.",
  },
];

const integrationLinks = [
  { title: "MCP Gateway", href: "/mcp-gateway", body: "Put SatGate in front of MCP tools and preserve a receipt per tool invocation." },
  { title: "Runtime integrations", href: "/integrations", body: "See current agent-client surfaces and where OpenAI, Anthropic, LangChain, and CrewAI adapters fit." },
  { title: "Developer docs", href: "https://cloud.satgate.io/docs", body: "Use the docs for setup details while issue/pay/verify API access is in private beta." },
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
      name: "Build agents that can spend safely",
      url: "https://satgate.io/build",
      description: metadata.description,
      datePublished: "2026-05-12",
      dateModified: "2026-05-12",
      isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
      about: [
        { "@type": "Thing", name: "Economic Firewall for AI agents" },
        { "@type": "Thing", name: "agent capabilities" },
        { "@type": "Thing", name: "verifiable receipts" },
        { "@type": "Thing", name: "rail-neutral payment governance" },
      ],
    },
    {
      "@type": "SoftwareSourceCode",
      name: "SatGate issue/pay/verify quickstart",
      codeSampleType: "full",
      programmingLanguage: "Python",
      text: quickstart,
    },
  ],
};

export default function BuildPage() {
  return (
    <main className="min-h-screen overflow-x-hidden bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(56,189,248,0.18),transparent_34%),radial-gradient(circle_at_85%_10%,rgba(168,85,247,0.18),transparent_32%),radial-gradient(circle_at_55%_80%,rgba(16,185,129,0.10),transparent_34%)]" />
        <div className="relative mx-auto grid max-w-6xl min-w-0 gap-12 px-6 py-24 lg:grid-cols-[0.95fr_1.05fr] lg:items-center">
          <div>
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-2 text-sm font-semibold uppercase tracking-[0.22em] text-cyan-200">
              <Code2 size={16} /> Developer primitive
            </div>
            <h1 className="max-w-4xl text-5xl font-black tracking-tight text-white sm:text-6xl lg:text-7xl">
              Build agents that can spend safely
            </h1>
            <p className="mt-6 max-w-3xl text-xl leading-8 text-gray-300">
              Issue scoped capabilities, route paid calls, and return verifiable receipts your principal can trust.
            </p>
            <p className="mt-5 max-w-3xl text-lg leading-8 text-gray-400">
              SatGate is the <strong className="font-semibold text-white">Economic Firewall for AI agents</strong>. This is the developer surface: <strong className="font-semibold text-white">Capabilities in. Receipts out. Rails abstracted.</strong>
            </p>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
              <a
                href="https://cloud.satgate.io/docs"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200"
              >
                Read the docs <ArrowRight size={18} />
              </a>
              <a
                href="https://github.com/SatGate-io/satgate"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-400"
              >
                View SDK examples <TerminalSquare size={18} />
              </a>
              <Link href="/policy-to-proof" className="inline-flex items-center justify-center gap-2 rounded-lg border border-purple-700/60 bg-purple-950/20 px-6 py-3 font-bold text-purple-200 transition hover:border-purple-400">
                See proof model <FileCheck2 size={18} />
              </Link>
            </div>
          </div>

          <div className="min-w-0 rounded-2xl border border-cyan-900/50 bg-gray-950/90 shadow-2xl shadow-cyan-950/30">
            <div className="flex items-center gap-2 border-b border-gray-800 px-4 py-3">
              <div className="h-3 w-3 rounded-full bg-red-500" />
              <div className="h-3 w-3 rounded-full bg-yellow-500" />
              <div className="h-3 w-3 rounded-full bg-green-500" />
              <span className="ml-2 text-xs font-mono text-gray-500">issue_pay_verify.py</span>
            </div>
            <pre className="max-w-full overflow-x-auto p-5 text-sm leading-6 text-gray-300"><code>{quickstart}</code></pre>
            <div className="border-t border-gray-800 p-5">
              <p className="mb-2 text-xs font-mono uppercase tracking-[0.18em] text-cyan-300">SDK access</p>
              <pre className="max-w-full overflow-x-auto rounded-xl bg-black p-4 text-sm leading-6 text-gray-300"><code>{installCommands}</code></pre>
              <p className="mt-3 text-sm leading-6 text-gray-500">
                Public packages install today; the issue/pay/verify API namespace is in private beta. <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="text-cyan-300 hover:text-cyan-200">Request API access in the docs</a>.
              </p>
              <p className="mt-2 text-xs leading-5 text-gray-600">
                Duration fields accept human-readable values like <code className="rounded bg-gray-900 px-1 text-gray-400">1h</code>; ISO 8601 duration support belongs in the integration docs for clients that require structured durations.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mb-10 max-w-3xl">
          <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-emerald-300">Three calls</p>
          <h2 className="text-3xl font-bold text-white sm:text-4xl">Issue. Pay. Verify.</h2>
          <p className="mt-4 text-lg leading-8 text-gray-400">
            Developers should not wire payment rails, revocation logic, audit trails, and tool policy by hand. SatGate makes agent authority feel like a simple primitive while preserving enterprise proof.
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
                Payment rails change. The authority and evidence layer is the durable abstraction. SatGate can govern MCP tools, REST APIs, API-key billing, x402, L402, AgentCore Payments, Pay.sh, and enterprise ledgers without forcing your agent code to care which rail settled underneath.
              </p>
              <p>
                Humans and platforms deploy the policies. Agents consume capabilities. Upstreams receive verifiable proof that the action was authorized, bounded, and recorded.
              </p>
            </div>
          </div>
          <div className="min-w-0 rounded-2xl border border-gray-800 bg-black p-6">
            <div className="mb-4 flex items-center gap-2 text-emerald-200">
              <BadgeCheck size={20} /> Receipt previews
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
            <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-cyan-300">Copy-paste paths</p>
            <h2 className="text-3xl font-bold text-white sm:text-4xl">Use the same primitive from SDKs, MCP, or raw HTTP.</h2>
          </div>
          <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-2 text-cyan-300 transition hover:text-cyan-200">
            Open developer docs <ArrowRight size={16} />
          </a>
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
              The runtime changes. The contract stays the same: capability before action, receipt after decision.
            </p>
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            {integrationLinks.map((item) => {
              const card = (
                <>
                  <div className="mb-3 flex items-center gap-2 text-white">
                    <CheckCircle2 className="text-emerald-300" size={18} />
                    <h3 className="font-bold">{item.title}</h3>
                  </div>
                  <p className="text-sm leading-6 text-gray-400">{item.body}</p>
                  <div className="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-cyan-300">Open <ArrowRight size={14} /></div>
                </>
              );
              return item.href.startsWith("http") ? (
                <a key={item.title} href={item.href} target="_blank" rel="noopener noreferrer" className="rounded-xl border border-gray-800 bg-black p-5 transition hover:border-cyan-500">
                  {card}
                </a>
              ) : (
                <Link key={item.title} href={item.href} className="rounded-xl border border-gray-800 bg-black p-5 transition hover:border-cyan-500">
                  {card}
                </Link>
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
              <h2 className="text-3xl font-bold text-white sm:text-4xl">Start with the primitive. Let network effects come from receipts.</h2>
              <p className="mt-5 text-lg leading-8 text-gray-300">
                A trusted agent is one that can prove what it was allowed to do, what it actually did, and which policy governed the outcome. Build that path first; reputation and upstream acceptance can grow from the receipt trail later.
              </p>
            </div>
            <div className="grid gap-3 text-sm text-gray-300">
              {[
                "Scoped authority before every action",
                "Receipts for allowed, denied, delegated, revoked, and paid decisions",
                "Evidence Pack IDs your principal can audit",
                "Rail adapters below the authority layer",
                "Developer docs instead of payment-company ceremony",
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
        <h2 className="mx-auto max-w-3xl text-3xl font-bold text-white sm:text-4xl">Build the agent path, then prove every decision.</h2>
        <p className="mx-auto mt-5 max-w-2xl text-lg leading-8 text-gray-400">
          Start with issue/pay/verify. Keep the buyer story on Economic Firewall and Policy-to-Proof. Let agents consume the primitives.
        </p>
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
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

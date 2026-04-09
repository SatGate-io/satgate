import Link from 'next/link';
import { ArrowLeft, Calendar, Clock } from 'lucide-react';

export const metadata = {
  title: 'Cursor MCP Proxy Setup Guide: Add Budget Controls and Audit Trails to Your Tools - SatGate Blog',
  description: 'Learn how to set up a Cursor MCP proxy with SatGate to enforce budgets, meter tool usage, and add audit trails without changing your MCP servers.',
  alternates: { canonical: 'https://satgate.io/blog/cursor-mcp-proxy-setup-guide' },
  keywords: ['Cursor MCP proxy setup guide', 'Cursor MCP proxy', 'MCP budget control', 'Cursor tool audit trail', 'SatGate MCP proxy', 'Cursor MCP security']
};

export default function CursorMCPProxySetupGuidePage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>

        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">Cursor</span>
            <span className="px-2 py-1 rounded-full bg-blue-900/30 border border-blue-500/30 text-blue-300 text-xs font-mono">MCP</span>
            <span className="px-2 py-1 rounded-full bg-green-900/30 border border-green-500/30 text-green-300 text-xs font-mono">Tutorial</span>
            <span className="px-2 py-1 rounded-full bg-yellow-900/30 border border-yellow-500/30 text-yellow-300 text-xs font-mono">Governance</span>
          </div>

          <h1 className="text-4xl font-bold mb-4">Cursor MCP Proxy Setup Guide: Add Budget Controls and Audit Trails to Your Tools</h1>

          <p className="text-xl text-gray-400 mb-6">
            Cursor makes MCP tools easy to connect. It does not give you budget enforcement, spend attribution, or strong policy control out of the box. Here&apos;s how to add a proxy layer that does.
          </p>

          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> April 9, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 10 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Why proxy Cursor MCP traffic at all?</h2>

          <p className="text-gray-300 leading-relaxed">
            Cursor&apos;s MCP support is great for one thing: getting tools into the editor fast. You point Cursor at a server, the model sees new capabilities, and suddenly it can search codebases, call internal APIs, or trigger automations.
          </p>

          <p className="text-gray-300 leading-relaxed">
            That convenience becomes a governance problem the minute those tools have real cost or real blast radius.
          </p>

          <p className="text-gray-300 leading-relaxed">
            A plain MCP connection usually tells you who connected. It does not reliably enforce how much the agent can spend, which tools it can use under what limits, or how to attribute usage back to a team, environment, or workflow. If Cursor gets stuck in a loop, retries aggressively, or delegates work across multiple tools, you find out after the damage is done.
          </p>

          <p className="text-gray-300 leading-relaxed">
            That&apos;s why a Cursor MCP proxy matters. It gives you a policy point between the client and the tools. Instead of trusting every connected tool equally, you can insert an economic and security control layer that decides what gets through.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">What a good Cursor MCP proxy should do</h2>

          <p className="text-gray-300 leading-relaxed">
            If all you want is transport, you do not need a proxy. If you want governance, you do. A useful MCP proxy for Cursor should add at least four things:
          </p>

          <ul className="list-disc list-inside space-y-2 text-gray-300 my-4">
            <li><strong>Budget enforcement:</strong> block or cap tool usage before spend runs away</li>
            <li><strong>Per-tool policy:</strong> different limits for code search, web access, CI actions, or paid APIs</li>
            <li><strong>Audit trails:</strong> who used what tool, when, and with what result</li>
            <li><strong>Attribution:</strong> map usage to a developer, team, project, or environment</li>
          </ul>

          <p className="text-gray-300 leading-relaxed">
            SatGate sits in that layer. It does not require you to rewrite your MCP servers. It wraps access to them with policy, metering, and enforcement.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">The target architecture</h2>

          <p className="text-gray-300 leading-relaxed">
            The setup is simple:
          </p>

          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`Cursor -> SatGate MCP proxy -> Your MCP servers -> Internal APIs / SaaS / infra`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            Cursor talks to SatGate, not directly to each downstream service. SatGate checks the token, applies caveats and budgets, records the call, and then forwards the allowed request to the relevant MCP server.
          </p>

          <p className="text-gray-300 leading-relaxed">
            This matters because policy is now centralized. You stop baking spending logic into every single tool implementation. That is the sane way to scale.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 1: Start SatGate as your MCP control plane</h2>

          <p className="text-gray-300 leading-relaxed">
            First, run SatGate where it can reach your MCP servers. That can be local for development or centralized for a team deployment.
          </p>

          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`# example startup
satgate gateway start

# or, depending on your deployment style
satgate-gateway --config ./satgate.yaml`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The exact startup command depends on how you deploy SatGate, but the idea stays the same: you want one reachable gateway that owns metering and policy decisions.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 2: Register the MCP tools behind the proxy</h2>

          <p className="text-gray-300 leading-relaxed">
            Next, define the MCP servers or tools SatGate can expose. Think in categories, not just endpoints. Group expensive tools separately from harmless ones. Your code search server should not share the same policy as production deployment actions.
          </p>

          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`mcpServers:
  github-read:
    url: https://mcp.internal/github-read
    policy:
      price: 1
      dailyLimit: 100

  web-fetch:
    url: https://mcp.internal/web-fetch
    policy:
      price: 2
      dailyLimit: 50

  ci-actions:
    url: https://mcp.internal/ci
    policy:
      price: 10
      requireApproval: true`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The values above are illustrative, but the pattern is the point. Price the tools. Cap them. Decide which ones need extra friction. If everything is free and unrestricted, you are not doing governance, you are doing vibes.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 3: Mint a token for Cursor instead of exposing raw access</h2>

          <p className="text-gray-300 leading-relaxed">
            Do not point Cursor at your backend with unlimited credentials. Mint a constrained token that says exactly what Cursor is allowed to do.
          </p>

          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`satgate token create \
  --name "cursor-dev" \
  --audience "cursor" \
  --daily-limit 25 \
  --allow-tool "github-read" \
  --allow-tool "web-fetch" \
  --deny-tool "ci-actions"`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            This is where SatGate&apos;s capability model earns its keep. Instead of a single secret that unlocks everything, you issue a token with scoped permissions and economic boundaries. If it leaks, the damage is bounded. If Cursor misbehaves, the budget ends the party.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 4: Point Cursor at the proxy</h2>

          <p className="text-gray-300 leading-relaxed">
            In Cursor, configure the MCP connection to use the SatGate endpoint and the constrained token you just created. Depending on your local or team setup, that may look like a local URL during development or a hosted gateway URL for shared use.
          </p>

          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`{
  "mcpServers": {
    "satgate": {
      "url": "https://gateway.satgate.internal/mcp",
      "headers": {
        "Authorization": "Bearer sg_cursor_dev_token"
      }
    }
  }
}`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            Once Cursor connects, it still sees tools. The difference is that every tool call now passes through a layer that can meter, allow, deny, or log it.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 5: Add pricing and per-tool limits</h2>

          <p className="text-gray-300 leading-relaxed">
            This is the part most teams skip, and it&apos;s the whole reason to do the setup. You need a cost model, even a rough one.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Start simple. Assign each tool a credit cost based on external spend, operational risk, or scarcity. A cheap internal read tool might cost 1 credit. A web search tool that hits paid APIs might cost 5. A production action might require explicit approval plus a steep price.
          </p>

          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`policies:
  cursor-dev:
    totalDailyCredits: 25
    tools:
      github-read:
        cost: 1
        maxCallsPerHour: 100
      web-fetch:
        cost: 2
        maxCallsPerHour: 20
      ci-actions:
        enabled: false`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            That gets you two wins immediately. First, runaway loops get cut off. Second, developers learn which actions are cheap, expensive, or prohibited. Economic signals shape behavior better than angry Slack messages after the invoice lands.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Step 6: Turn on auditability</h2>

          <p className="text-gray-300 leading-relaxed">
            If Wayne asks, “what exactly did Cursor do with that token yesterday,” you should be able to answer without archaeology.
          </p>

          <p className="text-gray-300 leading-relaxed">
            A proper Cursor MCP proxy should log at least:
          </p>

          <ul className="list-disc list-inside space-y-2 text-gray-300 my-4">
            <li>timestamp</li>
            <li>token or delegated identity</li>
            <li>tool invoked</li>
            <li>estimated or assigned cost</li>
            <li>allow or deny decision</li>
            <li>project, team, or environment labels</li>
          </ul>

          <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm text-gray-300 my-4">
            <code>{`{
  "time": "2026-04-09T18:00:00Z",
  "subject": "cursor-dev",
  "project": "satgate-landing",
  "tool": "web-fetch",
  "cost": 2,
  "decision": "allow"
}`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            That is enough to support audit trails, chargebacks, incident review, and policy tuning later.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">How this prevents the common failure modes</h2>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">Runaway tool loops</h3>
          <p className="text-gray-300 leading-relaxed">
            Cursor keeps retrying a flaky tool. Without a proxy, it burns time and money until someone notices. With SatGate, the hourly or daily budget stops the loop automatically.
          </p>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">Overpowered editor access</h3>
          <p className="text-gray-300 leading-relaxed">
            A developer wants code search, but the same credentials also allow deploy actions. That is sloppy. A capability token fixes it by limiting what the editor can call in the first place.
          </p>

          <h3 className="text-xl font-semibold mt-6 mb-3 text-white">No team attribution</h3>
          <p className="text-gray-300 leading-relaxed">
            Finance sees a bill. Nobody knows whether it came from engineering experiments, support workflows, or one intern having a spicy afternoon with automation. Put project and team labels into the proxy layer and that ambiguity disappears.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Best practices for a sane rollout</h2>

          <ol className="list-decimal list-inside space-y-2 text-gray-300 my-4">
            <li><strong>Start in observe mode for a week.</strong> Measure usage first if you do not know the right prices yet.</li>
            <li><strong>Separate read tools from write tools.</strong> They deserve different policies.</li>
            <li><strong>Use small budgets at first.</strong> It is easier to loosen a cap than explain a surprise bill.</li>
            <li><strong>Mint tokens per environment.</strong> Local development and production-adjacent access should never share the same limits.</li>
            <li><strong>Keep policy centralized.</strong> Do not duplicate budget logic across every MCP server.</li>
          </ol>

          <p className="text-gray-300 leading-relaxed">
            My strong opinion: if you are connecting Cursor to tools with real spend or real operational impact and you are not proxying that traffic, you are being careless. Fast demos are fine. Team workflows need controls.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">What success looks like</h2>

          <p className="text-gray-300 leading-relaxed">
            After setup, your developers still use Cursor the same way. The UX barely changes. But under the hood, you gain a bunch of things you did not have before:
          </p>

          <ul className="list-disc list-inside space-y-2 text-gray-300 my-4">
            <li>hard limits instead of polite warnings</li>
            <li>tool-level policy instead of blanket trust</li>
            <li>auditable logs instead of guesswork</li>
            <li>chargeback-ready attribution instead of shared mystery spend</li>
          </ul>

          <p className="text-gray-300 leading-relaxed">
            That is the difference between an MCP demo and production-grade MCP governance.
          </p>

          <h2 className="text-2xl font-bold mt-8 mb-4 text-white">Final takeaway</h2>

          <p className="text-gray-300 leading-relaxed">
            Cursor MCP is not the problem. Unbounded access is. A proxy layer gives you the missing economic control plane, so the editor can stay fast without turning your tools into an unmetered free-for-all.
          </p>

          <p className="text-gray-300 leading-relaxed">
            If you want Cursor to use MCP tools safely at team scale, put SatGate in the middle and make policy explicit.
          </p>

          <div className="bg-gray-900 border border-gray-800 p-6 rounded-lg mt-8">
            <h3 className="text-xl font-semibold mb-4">Want to govern Cursor MCP usage instead of just trusting it?</h3>
            <p className="mb-4 text-gray-300">
              SatGate adds budget enforcement, per-tool pricing, and audit trails to MCP traffic, without requiring you to rebuild your tools.
            </p>
            <div className="flex gap-4">
              <a href="https://github.com/satgatelabs/satgate" className="inline-flex items-center px-4 py-2 bg-white text-black font-bold rounded-lg hover:bg-gray-200 transition">
                Explore SatGate →
              </a>
              <a href="/docs" className="inline-flex items-center px-4 py-2 border border-gray-600 text-gray-300 rounded-lg hover:bg-gray-900 transition">
                Read the Docs
              </a>
            </div>
          </div>
        </article>
      </div>
    </div>
  );
}

import Link from 'next/link';
import { ArrowLeft, Calendar, Clock } from 'lucide-react';

export const metadata = {
  title: 'Economic Governance in MCP: Beyond Tool Connection',
  description: 'The MCP ecosystem talks about capability. Nobody talks about cost. Here\'s why economic policy is the missing layer — and how to enforce it at the protocol level.',
  alternates: { canonical: 'https://satgate.io/blog/beyond-connection-economic-governance-mcp' },
  openGraph: {
    title: 'Economic Governance in MCP: Beyond Tool Connection',
    description: 'MCP agents need more than connection: enforce cost, budget, revocation, delegation, and audit policy before tool calls.',
    url: 'https://satgate.io/blog/beyond-connection-economic-governance-mcp',
    type: 'article',
    publishedTime: '2026-02-12T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Economic Governance in MCP: Beyond Tool Connection',
    description: 'Move MCP from tool connection to economic governance with per-tool costs, scoped budgets, revocation, and audit.',
  },
};

export default function BeyondConnectionPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Beyond Connection: The Case for Economic Governance in MCP',
    description: 'The MCP ecosystem talks about capability. Economic governance adds cost, budget, revocation, delegation, and audit policy to MCP tool calls.',
    url: 'https://satgate.io/blog/beyond-connection-economic-governance-mcp',
    datePublished: '2026-02-12',
    dateModified: '2026-05-04',
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'MCP economic governance' },
      { '@type': 'Thing', name: 'MCP tool cost policy' },
      { '@type': 'Thing', name: 'economic firewall for MCP' },
      { '@type': 'Thing', name: 'AI agent tool spend control' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is economic governance in MCP?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Economic governance in MCP means assigning cost, budget, revocation, delegation, and audit policy to MCP tool calls so agents can use tools safely without unbounded spend.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why is MCP connection not enough for production agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Connection only tells an agent which tools exist and how to call them. Production teams also need to know what each call costs, which agent is responsible, and when to block or revoke expensive behavior.',
        },
      },
      {
        '@type': 'Question',
        name: 'Where should MCP economic policy be enforced?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'MCP economic policy should be enforced in the request path at an MCP proxy or economic firewall before the tool call reaches the upstream server or paid API.',
        },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>
        
        {/* Header */}
        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">
              MCP
            </span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">
              L402
            </span>
            <span className="px-2 py-1 rounded-full bg-yellow-900/30 border border-yellow-500/30 text-yellow-300 text-xs font-mono">
              Macaroons
            </span>
            <span className="px-2 py-1 rounded-full bg-green-900/30 border border-green-500/30 text-green-300 text-xs font-mono">
              Agent Economy
            </span>
          </div>
          
          <h1 className="text-4xl font-bold mb-4">Beyond Connection: The Case for Economic Governance in MCP</h1>
          
          <p className="text-xl text-gray-400 mb-6 italic">
            Your AI agent just connected to 12 tools. Who&apos;s watching the bill?
          </p>
          
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1">
              <Calendar size={14} />
              February 12, 2026
            </span>
            <span className="flex items-center gap-1">
              <Clock size={14} />
              12 min read
            </span>
          </div>
        </header>

        {/* Article Content */}
        <article className="prose prose-invert prose-gray max-w-none">
          <div className="space-y-6 text-gray-300 leading-relaxed">
            
            <p className="text-lg">
              The Model Context Protocol (MCP) is the most important thing happening in the agent ecosystem 
              right now. Anthropic&apos;s open standard gives AI agents the ability to move beyond chat — connecting 
              them to GitHub, Slack, databases, cloud APIs, and anything else with a JSON-RPC interface.
            </p>

            <p>MCP turns chatbots into operators. That&apos;s the promise, and it&apos;s real.</p>

            <p className="text-xl font-medium text-white">But there&apos;s a problem nobody&apos;s talking about.</p>

            {/* --- Every Tool Call Has a Price Tag --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">Every Tool Call Has a Price Tag</h2>

            <p>
              When an agent calls a tool, something happens on the other end. A database query runs. An API 
              request fires. Tokens get consumed. Compute spins up. Every one of those actions costs money.
            </p>

            <p>The current MCP specification is excellent at answering two questions:</p>

            <ol className="list-decimal list-inside space-y-2 text-gray-400">
              <li><strong className="text-white">Transport</strong>: How does the agent talk to the tool?</li>
              <li><strong className="text-white">Schema</strong>: What does the tool do and what parameters does it accept?</li>
            </ol>

            <p>But it&apos;s silent on a third question that matters more than either of those in production:</p>

            <blockquote className="border-l-4 border-purple-500 pl-4 py-2 text-white italic text-lg">
              How much can this agent spend, and what happens when it hits the limit?
            </blockquote>

            <p>
              There is no native mechanism in MCP to say: <em>&quot;This agent can use the Google Search tool, 
              but stop it if it spends more than $2.00 this hour.&quot;</em>
            </p>

            {/* --- The $500 Hallucination --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">The $500 Hallucination</h2>

            <p>We all know agents hallucinate text. That&apos;s annoying but survivable.</p>

            <p>
              In an MCP-enabled world, agents can hallucinate <em>actions</em>. A coding agent stuck in a loop 
              doesn&apos;t just waste tokens — it fires real API calls, creates real cloud resources, runs real 
              database queries. Each one costs real money.
            </p>

            <p>Here&apos;s a scenario that&apos;s already happening in the wild:</p>

            <ol className="list-decimal list-inside space-y-2 text-gray-400">
              <li>You give an agent access to a Cloud Resource Manager MCP tool</li>
              <li>You ask it to &quot;optimize our staging environment&quot;</li>
              <li>The agent tries an approach, it doesn&apos;t work, so it tries another</li>
              <li>And another. And another.</li>
              <li>200 API calls later, you get a bill for $200</li>
            </ol>

            <p>
              The agent wasn&apos;t malicious. It was doing exactly what you asked — trying to solve the problem. 
              It just didn&apos;t have a budget.
            </p>

            <p>
              This isn&apos;t a hypothetical. Teams running autonomous agents against cloud APIs are discovering this 
              the hard way. The gap isn&apos;t in the agent&apos;s intelligence. It&apos;s in the infrastructure&apos;s ability to say &quot;stop.&quot;
            </p>

            {/* --- The Missing Layer --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Missing Layer: Economic Policy</h2>

            <p>Consider what we already enforce at the infrastructure level for human users:</p>

            <ul className="list-disc list-inside space-y-2 text-gray-400">
              <li><strong className="text-white">Authentication</strong>: Who are you?</li>
              <li><strong className="text-white">Authorization</strong>: What can you access?</li>
              <li><strong className="text-white">Rate limiting</strong>: How fast can you go?</li>
            </ul>

            <p>For AI agents, we need a fourth layer:</p>

            <ul className="list-disc list-inside space-y-2 text-gray-400">
              <li><strong className="text-purple-400">Economic policy</strong>: How much can you spend?</li>
            </ul>

            <p>
              This isn&apos;t rate limiting. Rate limits are blunt instruments — they cap requests per second 
              regardless of cost. A single request to GPT-4 costs 100x more than a request to an embeddings 
              API. You need cost-aware enforcement, not just velocity caps.
            </p>

            {/* Architecture Diagram */}
            <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 my-8 font-mono text-sm">
              <pre className="text-gray-300 overflow-x-auto">{`┌─────────┐         ┌──────────────────┐         ┌────────────┐
│  Agent   │────────▶│  Economic Proxy  │────────▶│ MCP Server │
│          │◀────────│  (SatGate)       │◀────────│  (Tools)   │
└─────────┘         └──────────────────┘         └────────────┘
                           │
                    ┌──────┴──────┐
                    │  Per-Tool   │
                    │  Cost       │
                    │  Attribution│
                    │  + Budget   │
                    │  Enforcement│
                    └─────────────┘`}</pre>
            </div>

            <p>
              SatGate is an <strong className="text-white">economic proxy</strong> — it sits between the agent 
              and the MCP server, intercepts every tool call, attributes its cost, and enforces budget policy 
              in real time.
            </p>

            {/* --- Intercepting MCP --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">How It Works: Intercepting MCP at the Wire Level</h2>

            <p>
              MCP uses JSON-RPC 2.0 over stdio or HTTP. Every tool invocation is a <code className="bg-gray-800 px-1.5 py-0.5 rounded text-purple-300">tools/call</code> method 
              with the tool name in the params:
            </p>

            <div className="bg-gray-900 border border-gray-700 rounded-xl p-4 my-6 font-mono text-sm overflow-x-auto">
              <pre className="text-gray-300">{`{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "search_database",
    "arguments": {
      "query": "SELECT * FROM orders WHERE status = 'pending'"
    }
  },
  "id": 42
}`}</pre>
            </div>

            <p>
              SatGate&apos;s MCP middleware parses this at the proxy layer — no agent modification required. It 
              extracts the method and tool name, matches them against cost profiles, and makes an enforce/allow 
              decision before the request ever reaches the upstream:
            </p>

            <div className="bg-gray-900 border border-gray-700 rounded-xl p-4 my-6 font-mono text-sm overflow-x-auto">
              <pre className="text-gray-300">{`# SatGate route config — per-tool cost profiles
routes:
  - name: mcp-tools
    match:
      pathPrefix: /v1/mcp
    policy:
      kind: fiat402
    mcp:
      costs:
        "search_database": 50      # 50 credits per call
        "create_resource": 500     # 500 credits — expensive!
        "read_file": 5             # cheap
        "execute_code": 200        # moderate
        "cloud_*": 300             # wildcard: any cloud tool`}</pre>
            </div>

            <p>
              The parser handles JSON-RPC batches, nested tool calls, and streaming responses. It runs at the 
              proxy layer with{' '}
              <a href="https://github.com/SatGate-io/satgate" className="text-purple-400 hover:text-purple-300 underline">
                96% test coverage
              </a>{' '}
              — this isn&apos;t a proof of concept.
            </p>

            {/* --- Hope-Based Governance --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">Why API Keys Fail: &quot;Hope-Based Governance&quot;</h2>

            <p>
              Most teams try to control agent spend with traditional API keys. This is like giving a teenager 
              your credit card and asking them to &quot;be careful.&quot; Once the key is out, you have no way to limit 
              what they buy or how fast they spend — until you manually revoke the key, usually after the damage is done.
            </p>

            <p>
              API keys are static. They can&apos;t express &quot;spend up to $5 on search tools this hour.&quot; JWTs are 
              better but still server-dependent — every authorization check hits a database. At agent scale, 
              with thousands of tool calls per minute, that latency adds up.
            </p>

            <p>
              SatGate replaces this hope-based governance with <strong className="text-purple-400">L402</strong> and{' '}
              <strong className="text-purple-400">macaroons</strong>.
            </p>

            {/* --- L402 --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">L402: The Economic Handshake</h2>

            <p>
              L402 (formerly LSAT) combines Lightning Network payments with macaroons. When an agent makes a 
              request through SatGate, we don&apos;t just check if they have a &quot;key.&quot; We initiate an economic handshake:
            </p>

            <ol className="list-decimal list-inside space-y-3 text-gray-400">
              <li>
                <strong className="text-white">Challenge</strong>: SatGate intercepts the tool call and returns{' '}
                <code className="bg-gray-800 px-1.5 py-0.5 rounded text-yellow-300">402 Payment Required</code>{' '}
                with a Lightning invoice and a macaroon
              </li>
              <li>
                <strong className="text-white">Proof</strong>: The agent&apos;s wallet pays the invoice (microsatoshis — 
                fractions of a cent) and receives a <em>preimage</em> — a cryptographic receipt
              </li>
              <li>
                <strong className="text-white">Access</strong>: The agent retries with the preimage. SatGate 
                verifies it cryptographically and unlocks the tool
              </li>
            </ol>

            <p>
              No accounts. No signup. No OAuth dance. Just cryptographic proof that you paid for what 
              you&apos;re about to use. This is how machine-to-machine commerce should work.
            </p>

            <p>
              For enterprises not ready for Lightning, <strong className="text-white">Fiat402</strong> provides 
              the same enforcement model using credit-based budgets — same hard caps, same delegation, 
              familiar accounting.
            </p>

            {/* --- Macaroons --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">Macaroons: The Smart Contract in Your Header</h2>

            <p>
              If L402 is the handshake, macaroons are the rules of the conversation. Originally designed by 
              Google, adopted by the Lightning Network, and purpose-built for delegated authorization.
            </p>

            <p>
              Unlike static API keys, macaroons support <strong className="text-white">caveats</strong> — 
              logic-based restrictions baked directly into the token. SatGate can issue a token to an agent that says:
            </p>

            <ul className="list-disc list-inside space-y-2 text-gray-400 italic">
              <li>&quot;This token is only valid for the next 60 minutes&quot;</li>
              <li>&quot;This token can only call the GitHub_PR_Review tool&quot;</li>
              <li>&quot;This token is capped at a total spend of $5.00&quot;</li>
            </ul>

            <p>
              And here&apos;s the critical property: macaroons are <strong className="text-white">self-verifying</strong>. 
              Using a root key and HMAC chain, SatGate validates a token in microseconds without hitting a database. 
              At agent scale — thousands of tool calls per minute — this is the difference between a governance 
              layer you can deploy and one that becomes a bottleneck.
            </p>

            <p>A macaroon is a permission slip that can be narrowed but never widened:</p>

            <div className="bg-gray-900 border border-gray-700 rounded-xl p-4 my-6 font-mono text-sm overflow-x-auto">
              <pre className="text-gray-300">{`Root Token (Acme Corp)
  └─ budget = 30,000 credits
  └─ expires = 2027-02-11
  │
  ├─ Division Token (Acme Digital)
  │   └─ budget = 8,000 credits  (narrowed from 30,000)
  │   └─ routes = /v1/chat/*
  │   │
  │   └─ Agent Token (CS Bot)
  │       └─ budget = 7,000 credits  (narrowed from 8,000)
  │       └─ routes = /v1/chat/completions
  │
  └─ Division Token (Acme Creative)
      └─ budget = 3,500 credits
      │
      └─ Agent Token (Design Generator)
          └─ budget = 2,500 credits
          └─ routes = /v1/images/*`}</pre>
            </div>

            <p>
              Each level can only attenuate — add restrictions, reduce budgets, narrow scope. A division token 
              can&apos;t grant more than its parent allocated. An agent token can&apos;t exceed its division&apos;s budget. 
              This is enforced cryptographically, not by policy checks in a database.
            </p>

            <p>When an agent hits its budget ceiling, SatGate returns a <code className="bg-gray-800 px-1.5 py-0.5 rounded text-yellow-300">402 Payment Required</code>:</p>

            <div className="bg-gray-900 border border-gray-700 rounded-xl p-4 my-6 font-mono text-sm overflow-x-auto">
              <pre className="text-gray-300">{`HTTP/1.1 402 Payment Required
Content-Type: application/json
X-SatGate-Budget-Remaining: 0
X-SatGate-Budget-Limit: 2500

{
  "error": "budget_exceeded",
  "message": "Agent 'acme-design-gen' has exhausted its budget",
  "spent": 2500,
  "limit": 2500
}`}</pre>
            </div>

            <p className="text-xl font-medium text-white">
              The agent stops. Gracefully. No runaway spend. No surprise bill.
            </p>

            {/* --- In Practice --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">What This Looks Like in Practice</h2>

            <p>
              We run a demo environment modeled on a real enterprise structure — a technology company with 
              four divisions, each running multiple AI agents:
            </p>

            <div className="overflow-x-auto my-6">
              <table className="w-full text-sm text-gray-300 border border-gray-700">
                <thead className="bg-gray-900 text-gray-400">
                  <tr>
                    <th className="text-left p-3 border-b border-gray-700">Agent</th>
                    <th className="text-left p-3 border-b border-gray-700">Division</th>
                    <th className="text-right p-3 border-b border-gray-700">Monthly Spend</th>
                    <th className="text-right p-3 border-b border-gray-700">Budget</th>
                    <th className="text-left p-3 border-b border-gray-700">Role</th>
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-b border-gray-800">
                    <td className="p-3 font-mono text-purple-300">CS Bot</td>
                    <td className="p-3">Acme Digital</td>
                    <td className="p-3 text-right text-yellow-300">$3,712</td>
                    <td className="p-3 text-right">$7,000</td>
                    <td className="p-3 text-gray-400">Customer service (GPT-4o)</td>
                  </tr>
                  <tr className="border-b border-gray-800">
                    <td className="p-3 font-mono text-purple-300">Design Generator</td>
                    <td className="p-3">Acme Creative</td>
                    <td className="p-3 text-right text-yellow-300">$1,064</td>
                    <td className="p-3 text-right">$2,500</td>
                    <td className="p-3 text-gray-400">Image generation (DALL-E 3)</td>
                  </tr>
                  <tr className="border-b border-gray-800">
                    <td className="p-3 font-mono text-purple-300">Script Assistant</td>
                    <td className="p-3">Acme Media</td>
                    <td className="p-3 text-right text-yellow-300">$571</td>
                    <td className="p-3 text-right">$1,800</td>
                    <td className="p-3 text-gray-400">Script writing (Claude)</td>
                  </tr>
                  <tr className="border-b border-gray-800">
                    <td className="p-3 font-mono text-purple-300">Safety Review</td>
                    <td className="p-3">Acme Creative</td>
                    <td className="p-3 text-right text-yellow-300">$138</td>
                    <td className="p-3 text-right">$800</td>
                    <td className="p-3 text-gray-400">Compliance checks (Claude)</td>
                  </tr>
                  <tr className="border-b border-gray-800">
                    <td className="p-3 font-mono text-purple-300">Subtitle Gen</td>
                    <td className="p-3">Acme Media</td>
                    <td className="p-3 text-right text-yellow-300">$83</td>
                    <td className="p-3 text-right">$300</td>
                    <td className="p-3 text-gray-400">Transcription (Whisper)</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-mono text-purple-300">Embeddings</td>
                    <td className="p-3">Corporate</td>
                    <td className="p-3 text-right text-yellow-300">$83</td>
                    <td className="p-3 text-right">$400</td>
                    <td className="p-3 text-gray-400">Shared search (OpenAI)</td>
                  </tr>
                </tbody>
              </table>
            </div>

            <p>
              The Design Generator spikes every Tuesday — batch jobs for weekly design reviews. Without 
              per-tool attribution, that spike is invisible in aggregate API spend. With SatGate, the CFO sees 
              exactly which agent, which division, which tool, and which day.
            </p>

            <p>
              The revoked intern bot? Someone&apos;s experimental agent tried to escalate its own delegation 
              permissions. SatGate caught the attempt, blocked it, and revoked the token. That&apos;s the security 
              story: not just cost control, but <strong className="text-white">agent governance</strong>.
            </p>

            {/* --- Three Modes --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Three Modes: Observe → Control → Charge</h2>

            <p>Not every organization is ready to hard-block their agents on day one. SatGate supports a progressive rollout:</p>

            <div className="space-y-4 my-6">
              <div className="bg-gray-900 border border-purple-500/30 rounded-xl p-4">
                <p className="font-bold text-purple-300 mb-1">👁 Observe (Free)</p>
                <p className="text-gray-400">
                  Deploy as a transparent proxy. See what your agents are doing and what they&apos;re spending. 
                  No blocking, no enforcement. This is your &quot;holy shit&quot; moment when you see the actual numbers.
                </p>
              </div>
              <div className="bg-gray-900 border border-cyan-500/30 rounded-xl p-4">
                <p className="font-bold text-cyan-300 mb-1">🎛 Control ($99/mo)</p>
                <p className="text-gray-400">
                  Turn on budget enforcement. Hard caps, per-tool cost attribution, cascade delegation. Agents 
                  that exceed their budget get a 402, not a surprise invoice.
                </p>
              </div>
              <div className="bg-gray-900 border border-yellow-500/30 rounded-xl p-4">
                <p className="font-bold text-yellow-300 mb-1">💲 Charge (Custom)</p>
                <p className="text-gray-400">
                  Monetize your APIs with L402 micropayments over Lightning. Agents pay per-call with 
                  cryptographic proof of payment. Your MCP tools become revenue generators, not cost centers.
                </p>
              </div>
            </div>

            <p>Most teams start with Observe. They come for visibility. They stay for control.</p>

            {/* --- Uncomfortable Truth --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Uncomfortable Truth</h2>

            <p>
              The MCP ecosystem is growing fast. Anthropic, OpenAI, and every agent framework is building 
              tool-use capabilities. The number of agent-to-API interactions is about to explode.
            </p>

            <p>
              Right now, the economic layer is an afterthought. Teams are discovering cost problems <em>after</em> deployment, <em>after</em> the 
              bill arrives, <em>after</em> an agent loops 200 times against a paid API.
            </p>

            <p>
              That&apos;s backwards. Economic policy should be as fundamental as authentication. You wouldn&apos;t deploy 
              an API without auth. Why would you deploy agents without budgets?
            </p>

            {/* --- Get Started --- */}
            <h2 className="text-2xl font-bold text-white mt-12 mb-4">Get Started</h2>

            <p>
              <strong className="text-white">Open Source</strong>: The{' '}
              <a href="https://github.com/SatGate-io/satgate" className="text-purple-400 hover:text-purple-300 underline">
                MCP parser and proxy middleware
              </a>{' '}
              are open source. Deploy it, see what your agents are actually spending, and decide if you need enforcement.
            </p>

            <div className="bg-gray-900 border border-gray-700 rounded-xl p-4 my-6 font-mono text-sm overflow-x-auto">
              <pre className="text-gray-300">{`# Install the CLI
clawhub install satgate

# Or via Claude Code
claude plugin marketplace add SatGate-io/satgate-cli

# Check your agent spend
satgate tokens
satgate spend`}</pre>
            </div>

            <p>
              <strong className="text-white">SatGate Cloud</strong>: Full per-tool cost attribution, delegation 
              hierarchies, and budget enforcement.{' '}
              <a href="https://cloud.satgate.io/cloud/login" className="text-purple-400 hover:text-purple-300 underline">
                Sign up for the beta
              </a>{' '}
              or{' '}
              <a href="https://satgate.io/design-partners" className="text-purple-400 hover:text-purple-300 underline">
                talk to us about a design partnership
              </a>.
            </p>

            <p className="text-xl font-medium text-white mt-8">
              Don&apos;t wait for the bill to learn how your agents behave.
            </p>

            <section className="not-prose mt-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
              <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">FAQ</p>
              <h2 className="mb-6 text-2xl font-bold text-white">MCP economic governance questions</h2>
              <div className="space-y-5">
                {[
                  ['What is economic governance in MCP?', 'Economic governance in MCP means assigning cost, budget, revocation, delegation, and audit policy to MCP tool calls so agents can use tools safely without unbounded spend.'],
                  ['Why is MCP connection not enough for production agents?', 'Connection only tells an agent which tools exist and how to call them. Production teams also need to know what each call costs, which agent is responsible, and when to block or revoke expensive behavior.'],
                  ['Where should MCP economic policy be enforced?', 'MCP economic policy should be enforced in the request path at an MCP proxy or economic firewall before the tool call reaches the upstream server or paid API.'],
                ].map(([question, answer]) => (
                  <div key={question} className="border-t border-gray-800 pt-5 first:border-t-0 first:pt-0">
                    <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                    <p className="leading-relaxed text-gray-400">{answer}</p>
                  </div>
                ))}
              </div>
            </section>

          </div>
        </article>

        {/* Footer */}
        <footer className="mt-16 pt-8 border-t border-gray-800">
          <p className="text-gray-500 text-sm italic">
            SatGate is an economic firewall for AI agent traffic. We help enterprises see, control, and 
            monetize what their agents do.{' '}
            <a href="https://satgate.io" className="text-purple-400 hover:text-purple-300 underline">
              Learn more at satgate.io
            </a>.
          </p>
          <div className="mt-6">
            <Link href="/blog" className="text-purple-400 hover:text-purple-300 flex items-center gap-2 transition">
              <ArrowLeft size={18} /> More from the blog
            </Link>
          </div>
        </footer>
      </div>
    </div>
  );
}

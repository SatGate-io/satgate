import Link from 'next/link';
import { ArrowLeft, Calendar, Clock } from 'lucide-react';

export const metadata = {
  title: "Zero Trust for AI Agents: Capability Tokens, Revocation, and Budgets",
  description: "Zero Trust for AI agents needs more than identity: scoped capability tokens, revocation, delegation limits, and request-path budget enforcement.",
  alternates: { canonical: 'https://satgate.io/blog/zero-trust-for-ai-agents' },
  keywords: ['Zero Trust AI agents', 'Zero Trust for AI', 'AI agent security', 'capability-based security', 'API security AI agents', 'macaroon tokens', 'agent delegation security'],
  openGraph: {
    title: 'Zero Trust for AI Agents: Tokens, Revocation, and Budgets',
    description: 'Zero Trust for AI agents needs scoped capability tokens, revocation, delegation limits, and request-path budget enforcement.',
    url: 'https://satgate.io/blog/zero-trust-for-ai-agents',
    type: 'article',
    publishedTime: '2026-04-03T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Zero Trust for AI Agents: Tokens, Revocation, and Budgets',
    description: 'Extend Zero Trust for AI agents with capability tokens, delegated budgets, revocation, and economic firewalls.',
  },
};

export default function ZeroTrustForAIAgentsBlogPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Zero Trust for AI Agents: Capability Tokens, Revocation, and Budgets',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-04-03',
    dateModified: '2026-06-04',
    mainEntityOfPage: 'https://satgate.io/blog/zero-trust-for-ai-agents',
    about: [
      { '@type': 'Thing', name: 'Zero Trust for AI agents' },
      { '@type': 'Thing', name: 'capability-based agent security' },
      { '@type': 'Thing', name: 'scoped and revocable agent tokens' },
      { '@type': 'Thing', name: 'delegation limits for autonomous agents' },
      { '@type': 'Thing', name: 'budget-aware authorization' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Why does traditional Zero Trust break down for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Traditional Zero Trust assumes stable human identities, managed devices, and predictable access patterns. AI agents are ephemeral, delegate to sub-agents, and can generate thousands of API calls from one task.',
        },
      },
      {
        '@type': 'Question',
        name: 'What replaces identity-based security for autonomous agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Autonomous agents need capability-based security: scoped, revocable tokens that encode what the agent can do, how much it can spend, where it can call, and when authority expires.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does an economic firewall extend Zero Trust for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'An economic firewall enforces cost, scope, and delegation in the request path before upstream APIs execute, giving teams budget-aware authorization that identity systems alone cannot provide.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can Zero Trust policies express agent budgets and delegated authority?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Most Zero Trust policies can express identity, device posture, location, and application access, but they usually cannot express per-agent spend limits, delegated budget attenuation, MCP tool costs, or proof-of-payment requirements before each request.',
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
        
        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-red-900/30 border border-red-500/30 text-red-300 text-xs font-mono">Zero Trust</span>
            <span className="px-2 py-1 rounded-full bg-blue-900/30 border border-blue-500/30 text-blue-300 text-xs font-mono">Security</span>
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">AI Agents</span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">Architecture</span>
          </div>
          
          <h1 className="text-4xl font-bold mb-4">Zero Trust for AI Agents: Why Identity-Based Security Collapses When Machines Call the Shots</h1>
          
          <p className="text-xl text-gray-400 mb-6 italic">
            Zero Trust says &ldquo;never trust, always verify.&rdquo; But verify <em>what</em>, exactly, when the requester is an autonomous agent that spawns sub-agents, delegates credentials, and makes 1,500 API calls per prompt? Identity-based security was designed for humans. The agent economy needs something fundamentally different.
          </p>
          
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> April 3, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 12 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          
          <p className="text-gray-300 text-lg leading-relaxed">
            If you&apos;ve spent any time in enterprise security, you know Zero Trust. Verify every request. Authenticate every user. Trust no network segment implicitly. It&apos;s the dominant security paradigm for good reason — it replaced the broken &ldquo;castle and moat&rdquo; model that assumed everything inside the perimeter was safe.
          </p>

          <p className="text-gray-300 leading-relaxed">
            But Zero Trust was built for a world where humans sit at keyboards, devices have certificates, and access patterns are predictable. AI agents break every one of those assumptions. And the security industry hasn&apos;t caught up yet.
          </p>

          <p className="text-gray-300 leading-relaxed">
            This post lays out exactly where Zero Trust fails for AI agents, why patching the existing model won&apos;t work, and what a purpose-built security architecture looks like.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">What Zero Trust Actually Assumes</h2>

          <p className="text-gray-300 leading-relaxed">
            Before we can talk about where Zero Trust breaks, we need to be precise about what it assumes. The NIST SP 800-207 Zero Trust Architecture standard defines five core tenets. Every one of them has an implicit dependency on human-scale behavior:
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">1. All resources are accessed in a secure manner regardless of network location.</strong> This works. Agents use APIs over HTTPS. No issue here.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">2. Access is granted on a per-session basis.</strong> Implies sessions have bounded duration and scope. A human session lasts minutes to hours. An agent session might spawn 50 sub-agents in seconds, each needing different access levels. What&apos;s a &ldquo;session&rdquo; when the requester can clone itself?
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">3. Access is determined by dynamic policy.</strong> Policy evaluates identity, device posture, behavioral patterns, and risk signals. But agents don&apos;t have &ldquo;device posture.&rdquo; Their behavioral patterns are non-deterministic — the same agent prompt can produce wildly different API call sequences. And identity is the weakest signal of all because agents delegate constantly.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">4. The enterprise ensures all resources are in their most secure state.</strong> Assumes the enterprise controls the endpoints. In the agent economy, your API serves agents you&apos;ve never seen before, running on infrastructure you don&apos;t manage, with capabilities you didn&apos;t grant.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">5. Authentication and authorization are strictly enforced before access.</strong> This is where Zero Trust&apos;s assumptions shatter completely for agents. Let&apos;s dig into why.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Identity Problem: Who Is an Agent, Really?</h2>

          <p className="text-gray-300 leading-relaxed">
            Zero Trust&apos;s entire enforcement model revolves around identity. Verify the user. Check their role. Evaluate their device. Make an access decision. The assumption is that identity is stable, verifiable, and meaningful.
          </p>

          <p className="text-gray-300 leading-relaxed">
            AI agents demolish this assumption in three ways:
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Agents Delegate — Identities Don&apos;t</h3>

          <p className="text-gray-300 leading-relaxed">
            When a human uses an application, they authenticate once and the application acts on their behalf within defined OAuth scopes. The delegation chain is short: human → application → API. Zero Trust can verify each link.
          </p>

          <p className="text-gray-300 leading-relaxed">
            An AI agent orchestrating a complex task might delegate to five sub-agents, each of which delegates to three more. That&apos;s a delegation chain five levels deep with 15+ entities making API calls. Each sub-agent needs different permissions. The parent agent needs to constrain what its children can do. And the API receiving the request needs to verify the entire chain.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Traditional identity systems don&apos;t model this. RBAC gives you roles. ABAC gives you attributes. Neither gives you <em>delegated authority that attenuates at each level</em>. You can&apos;t express &ldquo;this agent has a $100 budget, and it can give sub-agents portions of that budget, but the total can never exceed $100&rdquo; in an IAM policy.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Agent Identity Is Ephemeral</h3>

          <p className="text-gray-300 leading-relaxed">
            A human employee has an identity that persists for years. Their access patterns develop over time, allowing behavioral analysis and anomaly detection. An AI agent might exist for 30 seconds — spun up to handle a single task, then terminated. There&apos;s no behavioral baseline to compare against. There&apos;s no device posture to evaluate. There&apos;s barely an identity at all.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Zero Trust&apos;s continuous verification model assumes it can <em>learn</em> what normal looks like for each identity. For ephemeral agents, every access is the first access. Every request is an anomaly by default.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Identity ≠ Authority</h3>

          <p className="text-gray-300 leading-relaxed">
            This is the deepest problem. Zero Trust answers &ldquo;who are you?&rdquo; and then maps that identity to permissions. But for agents, the right question isn&apos;t &ldquo;who are you?&rdquo; — it&apos;s &ldquo;what are you allowed to do, right now, with what budget, for what purpose?&rdquo;
          </p>

          <p className="text-gray-300 leading-relaxed">
            An agent&apos;s authority should be defined by its <em>token</em>, not its <em>identity</em>. The token says: you can call these endpoints, spend up to this amount, until this time. It doesn&apos;t matter <em>who</em> you are. It matters <em>what you hold</em>.
          </p>

          <p className="text-gray-300 leading-relaxed">
            This is the fundamental shift from identity-based to capability-based security. And it&apos;s not a minor tweak to Zero Trust — it&apos;s a different paradigm entirely.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Scale Problem: Zero Trust Can&apos;t Keep Up</h2>

          <p className="text-gray-300 leading-relaxed">
            Even if you could solve the identity problem, Zero Trust has a scale problem with agents that&apos;s fundamentally architectural.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Zero Trust evaluates every request against a centralized policy engine. Okta, Azure AD, Google BeyondCorp — they all work this way. Request comes in, policy engine evaluates identity + context + risk, returns allow/deny.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Now consider an agent swarm. A research agent spawns 20 sub-agents to gather data from different sources. Each sub-agent makes 50 API calls. That&apos;s 1,000 policy evaluations in seconds. Each evaluation requires identity lookup, role resolution, contextual risk assessment, and policy computation.
          </p>

          <p className="text-gray-300 leading-relaxed">
            This isn&apos;t a throughput problem you solve with bigger servers. It&apos;s a latency problem. Every API call waits for the policy engine to respond. At human scale — a user making 10 requests per minute — the latency is invisible. At agent scale — 1,000 requests in 5 seconds — it&apos;s a bottleneck that degrades the entire system.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Capability-based tokens eliminate this bottleneck entirely. The token <em>is</em> the policy decision, pre-computed and cryptographically sealed. Validating a macaroon token is a local operation — check the HMAC chain, verify the caveats haven&apos;t been violated, done. No round-trip to a policy engine. No identity lookup. No contextual risk assessment. The authorization decision was made when the token was minted.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Budget Problem: Zero Trust Has No Concept of Cost</h2>

          <p className="text-gray-300 leading-relaxed">
            Here&apos;s where the gap becomes a chasm. Zero Trust is a security framework. It answers: &ldquo;Is this request authorized?&rdquo; It does <em>not</em> answer: &ldquo;Can this requester afford this request?&rdquo;
          </p>

          <p className="text-gray-300 leading-relaxed">
            For humans, this distinction didn&apos;t matter. A human making API calls generates costs that correlate with their work patterns — predictable, bounded, reviewable. An AI agent with valid credentials can burn through $50,000 in API costs in an afternoon. It&apos;s fully authorized by Zero Trust standards. It also just bankrupted your department&apos;s quarterly budget.
          </p>

          <p className="text-gray-300 leading-relaxed">
            This isn&apos;t a theoretical risk. It&apos;s happening right now. Companies deploying AI agents are discovering that traditional security gives them a binary answer — access or no access — when what they need is a continuous answer: access <em>within these economic constraints</em>.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Zero Trust practitioners might argue that rate limiting or quota policies can address this. But rate limits are crude instruments. They cap <em>volume</em>, not <em>cost</em>. An agent making 100 calls to a $0.01 endpoint is very different from 100 calls to a $5.00 endpoint — same rate, 500x cost difference. You need budget enforcement that understands economics, not just traffic patterns.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Capability-Based Security: The Agent-Native Alternative</h2>

          <p className="text-gray-300 leading-relaxed">
            If identity-based security fails for agents, what replaces it? The answer comes from a concept that&apos;s older than Zero Trust: capability-based security.
          </p>

          <p className="text-gray-300 leading-relaxed">
            In capability-based systems, access is controlled by <em>tokens that carry permissions</em>, not by <em>identities that map to permissions</em>. The distinction is subtle but transformative:
          </p>

          <div className="bg-red-900/20 border border-red-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">❌ Identity-Based (Zero Trust)</h4>
            <p className="text-gray-300 text-sm">
              <br />• Who are you? → Look up your permissions → Allow/deny
              <br />• Permissions live in a centralized directory (IAM, RBAC)
              <br />• Delegation requires creating new identities with mapped roles
              <br />• Budget enforcement is bolted on as a separate system
              <br />• Every request requires a round-trip to the policy engine
            </p>
          </div>

          <div className="bg-green-900/20 border border-green-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">✅ Capability-Based (Agent-Native)</h4>
            <p className="text-gray-300 text-sm">
              <br />• What token do you hold? → Verify the token&apos;s constraints → Allow/deny
              <br />• Permissions travel <em>with</em> the token (embedded as caveats)
              <br />• Delegation = create an attenuated copy of the token (weaker, never stronger)
              <br />• Budget is a first-class constraint inside the token
              <br />• Validation is local — no external lookup required
            </p>
          </div>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Macaroons: Capabilities Made Practical</h3>

          <p className="text-gray-300 leading-relaxed">
            Macaroon tokens are the practical implementation of capability-based security for APIs. Developed by Google Research in 2014, macaroons are bearer tokens with a unique property: anyone holding a macaroon can create a more restricted version of it, but nobody can create a less restricted version.
          </p>

          <p className="text-gray-300 leading-relaxed">
            This property — called <strong className="text-white">attenuation</strong> — solves the agent delegation problem elegantly:
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`# Enterprise admin mints a root token
Token: team-research-q2
  budget: $10,000
  scope: /api/v1/*
  expires: 2026-06-30

# Orchestrator agent attenuates for sub-agent
Token: research-subtask-47
  budget: $500          ← can only reduce, never increase
  scope: /api/v1/search ← can only narrow, never widen
  expires: 2026-04-04   ← can only shorten, never extend

# Sub-agent attenuates further for its own child
Token: search-worker-12
  budget: $50
  scope: /api/v1/search?source=arxiv
  expires: 2026-04-03T18:00:00Z

# At every level: the child can NEVER exceed the parent's authority
# No IAM policy updates. No admin approval. No identity provisioning.`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            This is what safe delegation looks like. The parent agent gives its child exactly the authority needed — no more — and it&apos;s mathematically impossible for the child to escalate. The budget constraint is cryptographically enforced, not policy-enforced. No one can edit a database to give themselves more budget. The math doesn&apos;t allow it.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">What Agent-Native Zero Trust Actually Looks Like</h2>

          <p className="text-gray-300 leading-relaxed">
            This isn&apos;t about throwing away Zero Trust. The &ldquo;never trust, always verify&rdquo; philosophy is sound. What changes is <em>how</em> you verify and <em>what</em> you verify:
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Verify the token, not the identity.</strong> When an agent presents a macaroon, verify its HMAC chain back to the root. Check that every caveat is satisfied. This tells you exactly what the bearer is allowed to do — regardless of who they are.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Enforce budgets at the gateway.</strong> Every API call has a cost. The gateway tracks cumulative spend against the token&apos;s budget caveat. When the budget is exhausted, access stops — instantly, automatically, with no human intervention.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Audit the chain, not the session.</strong> Traditional Evidence Packs track user sessions. Agent Evidence Packs need to track delegation chains — who minted the token, who attenuated it, what was spent at each level, and which specific API calls were made.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Make policy decisions at mint time.</strong> Instead of evaluating policy on every request, encode the policy decision into the token when it&apos;s minted. The runtime check becomes: &ldquo;is this token valid and within its constraints?&rdquo; — a local, fast, scalable operation.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">SatGate&apos;s Approach: Economic Governance as the Security Layer</h2>

          <p className="text-gray-300 leading-relaxed">
            SatGate implements this capability-based model as an HTTP gateway that sits in front of your APIs. Instead of integrating with your identity provider to evaluate who&apos;s calling, it evaluates <em>what token they hold</em> and <em>what that token permits</em>.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Three policy modes handle different security postures:
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Observe mode</strong> — lets everything through but logs every request with full cost attribution. Deploy this first to understand your agent traffic patterns before enforcing anything. Zero Trust equivalent: monitoring phase.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Control mode</strong> — enforces budget limits via macaroon tokens. Every agent must present a valid token with sufficient budget remaining. Over-budget requests get rejected immediately with a clear error. Zero Trust equivalent: enforcement phase, but with economic constraints that Zero Trust can&apos;t express.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Charge mode (L402)</strong> — requires payment before access. No budget tracking needed because each request is prepaid via Lightning micropayment. This mode enables permissionless API access for agents you&apos;ve never seen before — they pay, they get access, no signup required.
          </p>

          <p className="text-gray-300 leading-relaxed">
            The key insight: security and economics are the same enforcement layer. When an agent exceeds its budget, it&apos;s both a security event (unauthorized access attempt) and an economic event (cost overrun). SatGate treats them identically — the gateway rejects the request and logs the violation.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Practical Migration: Zero Trust → Agent-Native Security</h2>

          <p className="text-gray-300 leading-relaxed">
            If you&apos;re running Zero Trust today, you don&apos;t rip it out. You layer agent-native security on top for the workloads that need it:
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Phase 1: Observe.</strong> Deploy SatGate in observe mode alongside your existing Zero Trust stack. Let human traffic continue through your identity provider. Route agent traffic through SatGate. You now have visibility into what agents are doing and what they&apos;re costing you — data your Zero Trust tools can&apos;t provide.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Phase 2: Token-gate agent traffic.</strong> Start minting macaroon tokens for your agents with budget constraints. Agents that exceed their budgets get cut off automatically. You still verify identity at the human level (who minted the token), but runtime enforcement is capability-based.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Phase 3: Enable delegation.</strong> Allow orchestrator agents to attenuate tokens for sub-agents. This is where the model pays off — multi-agent workflows operate with proper economic boundaries at every level, without your security team manually provisioning identities for ephemeral sub-agents.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Phase 4: Open external access.</strong> If you want external agents to consume your APIs, add L402 payment support. Now any agent on the internet can pay for access without your sales team being involved. Zero Trust stays in place for your internal users. Capability-based security handles the agent economy.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Bottom Line</h2>

          <p className="text-gray-300 leading-relaxed">
            Zero Trust was a generational improvement over perimeter security. It correctly identified that network location is a terrible proxy for trust. But it replaced &ldquo;trust the network&rdquo; with &ldquo;trust the identity&rdquo; — and identity is just as unreliable when your requesters are ephemeral, autonomous, and multiplying.
          </p>

          <p className="text-gray-300 leading-relaxed">
            The next evolution isn&apos;t &ldquo;better Zero Trust.&rdquo; It&apos;s recognizing that for machine-to-machine interactions, <strong className="text-white">what a requester holds</strong> matters more than <strong className="text-white">who a requester is</strong>. Capability tokens that carry permissions, budgets, and expiration — verifiable locally, delegatable safely, attenuatable mathematically — are how you secure a world where agents outnumber humans 1,000 to 1.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Zero Trust got us here. Capability-based security takes us where we&apos;re going.
          </p>

          <section className="not-prose mt-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-red-300">FAQ</p>
            <h2 className="mb-6 text-2xl font-bold text-white">Zero Trust for AI agents questions</h2>
            <div className="space-y-5">
              {[
                ['Why does traditional Zero Trust break down for AI agents?', 'Traditional Zero Trust assumes stable human identities, managed devices, and predictable access patterns. AI agents are ephemeral, delegate to sub-agents, and can generate thousands of API calls from one task.'],
                ['What replaces identity-based security for autonomous agents?', 'Autonomous agents need capability-based security: scoped, revocable tokens that encode what the agent can do, how much it can spend, where it can call, and when authority expires.'],
                ['How does an economic firewall extend Zero Trust for AI agents?', 'An economic firewall enforces cost, scope, and delegation in the request path before upstream APIs execute, giving teams budget-aware authorization that identity systems alone cannot provide.'],
                ['Can Zero Trust policies express agent budgets and delegated authority?', 'Most Zero Trust policies can express identity, device posture, location, and application access, but they usually cannot express per-agent spend limits, delegated budget attenuation, MCP tool costs, or proof-of-payment requirements before each request.'],
              ].map(([question, answer]) => (
                <div key={question} className="border-t border-gray-800 pt-5 first:border-t-0 first:pt-0">
                  <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                  <p className="leading-relaxed text-gray-400">{answer}</p>
                </div>
              ))}
            </div>
          </section>

          {/* CTA Section */}
          <div className="mt-16 bg-gradient-to-r from-red-900/20 to-blue-900/20 border border-red-800/30 rounded-xl p-8">
            <h3 className="text-xl font-bold text-white mb-3">Ready to Secure Your AI Agent Traffic?</h3>
            <p className="text-gray-300 mb-4">
              SatGate adds capability-based security and budget enforcement to any API — without replacing your existing identity stack. Deploy in observe mode today, enforce tomorrow.
            </p>
            <div className="flex flex-wrap gap-3">
              <a href="https://github.com/SatGate-io/satgate" className="inline-flex items-center gap-2 bg-white text-black px-6 py-3 rounded-lg font-bold hover:bg-gray-200 transition text-sm">
                View on GitHub
              </a>
              <Link href="/design-partners" className="inline-flex items-center gap-2 border border-red-500 text-red-300 px-6 py-3 rounded-lg font-bold hover:bg-red-900/30 transition text-sm">
                Become a Design Partner
              </Link>
              <Link href="/capability-auth" className="inline-flex items-center gap-2 border border-red-500 text-red-300 px-6 py-3 rounded-lg font-bold hover:bg-red-900/30 transition text-sm">
                Capability auth
              </Link>
            </div>
          </div>

        </article>
      </div>
    </div>
  );
}

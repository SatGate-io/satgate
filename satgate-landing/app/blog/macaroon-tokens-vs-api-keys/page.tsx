import Link from 'next/link';
import { ArrowLeft, Calendar, Clock } from 'lucide-react';

export const metadata = {
  title: "Macaroon Tokens vs API Keys for Agent Access",
  description: "Compare macaroon tokens and API keys for scoped authorization, delegated access, and safer AI agent permissions.",
  alternates: { canonical: 'https://satgate.io/blog/macaroon-tokens-vs-api-keys' },
  keywords: ['macaroon tokens vs API keys', 'capability-based authentication', 'API authentication AI agents', 'delegated authority tokens', 'macaroon authentication', 'AI agent security', 'capability tokens'],
  openGraph: {
    title: 'Macaroon Tokens vs API Keys for AI Agents',
    description: 'Compare macaroon authentication and API keys for scoped AI agent credentials, revocation, delegation, and budget limits.',
    url: 'https://satgate.io/blog/macaroon-tokens-vs-api-keys',
    type: 'article',
    publishedTime: '2026-03-31T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Macaroon Tokens vs API Keys for AI Agents',
    description: 'Macaroon authentication gives AI agents scoped credentials with expiry, delegation, revocation, and budget caveats.',
  },
};

export default function MacaroonTokensVsApiKeysBlogPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Macaroon Tokens vs API Keys for Agent Access',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-03-31',
    dateModified: '2026-05-04',
    mainEntityOfPage: 'https://satgate.io/blog/macaroon-tokens-vs-api-keys',
    about: [
      { '@type': 'Thing', name: 'macaroon tokens versus API keys' },
      { '@type': 'Thing', name: 'capability-based authentication for AI agents' },
      { '@type': 'Thing', name: 'scoped agent credentials' },
      { '@type': 'Thing', name: 'budget caveats for agent tokens' },
      { '@type': 'Thing', name: 'revocable delegated authority' },
    ],
  };

  const faqSchema = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is the difference between macaroon tokens and API keys?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'API keys are static bearer secrets that usually identify an account and grant broad access. Macaroon tokens are capability tokens that can carry caveats for scope, expiry, budget, delegation, revocation, and audit, making them safer for autonomous AI agents.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why are macaroons better for AI agents than API keys?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'AI agents delegate work, call tools autonomously, and can spend money quickly. Macaroons let teams issue scoped and attenuated authority so each agent or sub-agent can only use specific APIs, within explicit budgets and time windows.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can macaroon tokens enforce AI agent spending limits?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. Budget caveats can bind a macaroon to a spending limit, task, tool, route, or customer account. A gateway can verify those caveats before each request and block or downgrade calls that exceed policy.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does macaroon authentication support revocation?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Macaroon authentication supports revocation by combining short expirations, caveats, revocation identifiers, and gateway-side deny lists. A compromised agent token can be invalidated without rotating every account-level API key.',
        },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(faqSchema) }}
      />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/blog" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Blog
        </Link>
        
        <header className="mb-12">
          <div className="flex flex-wrap gap-2 mb-4">
            <span className="px-2 py-1 rounded-full bg-blue-900/30 border border-blue-500/30 text-blue-300 text-xs font-mono">Authentication</span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">AI Agents</span>
            <span className="px-2 py-1 rounded-full bg-green-900/30 border border-green-500/30 text-green-300 text-xs font-mono">Security</span>
          </div>
          
          <h1 className="text-4xl font-bold mb-4">Macaroon Tokens vs API Keys for AI Agents</h1>
          <div className="mb-6 rounded-2xl border border-purple-900/60 bg-purple-950/20 p-5">
            <p className="mb-2 text-sm font-semibold uppercase tracking-[0.2em] text-purple-300">Quick comparison</p>
            <p className="text-gray-300">API keys identify an account and usually grant broad access until manually rotated. Macaroons act like attenuable capabilities: each delegated token can carry caveats for scope, expiry, tool access, spend budget, revocation, and audit.</p>
          </div>
          
          <p className="text-xl text-gray-400 mb-6 italic">
            API keys answer "who are you?" Macaroon tokens answer "what can you do?" When autonomous agents need to delegate tasks and respect budget limits, the difference between identity and capability determines success or failure.
          </p>
          
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> March 31, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 12 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          
          <p className="text-gray-300 text-lg leading-relaxed">
            Every API authentication system makes a fundamental choice: identify <em>who</em> the caller is, or specify <em>what</em> the caller can do. For twenty years, web APIs have chosen identity. Get an API key, prove you're legitimate, access everything your account allows. This worked beautifully when humans called APIs through carefully written code.
          </p>

          <p className="text-gray-300 leading-relaxed">
            AI agents break that model. An agent doesn't just call your API — it delegates to sub-agents, spawns parallel tasks, and operates under budgets set by entities three delegation layers up the chain. Your API key knows the agent's identity, but it has no idea what that agent is authorized to spend, which sub-tasks it can delegate, or when its authority expires.
          </p>

          <p className="text-gray-300 leading-relaxed">
            This is why the agent economy needs <strong className="text-white">capability-based authentication</strong> instead of identity-based authentication. And the best implementation of capability-based auth for APIs is macaroon tokens.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Let's break down why API keys fail for AI workloads, how macaroons solve the problem, and what this means for your API's security model.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The API Key Model: Identity Without Constraints</h2>

          <p className="text-gray-300 leading-relaxed">
            API keys are fundamentally about <em>who</em>, not <em>what</em>. When you authenticate with an API key, you're proving your identity to the system. The system then checks its access control list: does this identity have permission to access this resource? If yes, full access. If no, complete denial.
          </p>

          <p className="text-gray-300 leading-relaxed">
            This binary model works for human developers because humans make deliberate, accountable decisions. A developer with access to the `/admin` endpoints uses that access responsibly because their career depends on it. A developer with a production API key doesn't accidentally burn through a $10,000 budget because they're watching the dashboard.
          </p>

          <p className="text-gray-300 leading-relaxed">
            But AI agents aren't humans. They don't have careers to protect or dashboards to watch. They have objectives to optimize and constraints to respect — but those constraints have to be encoded into the authentication system, not left to agent discretion.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Problem 1: All-or-Nothing Access</h3>

          <p className="text-gray-300 leading-relaxed">
            API keys grant binary access: you either have permission to access an endpoint, or you don't. There's no concept of "partial permission" or "permission with constraints." If an agent has access to the `/translate` endpoint, it can make unlimited translation requests until the monthly quota is exhausted.
          </p>

          <p className="text-gray-300 leading-relaxed">
            In a multi-agent system, this creates impossible tradeoffs. Either you give every agent full account-level access (unsafe), or you create separate API keys for every agent (unmanageable), or you implement a proxy layer that enforces constraints before forwarding to your API (complex).
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Problem 2: No Delegation Support</h3>

          <p className="text-gray-300 leading-relaxed">
            API keys can't be safely delegated. If Agent A needs to give Agent B limited access to an API, Agent A has two bad options: share its full API key (no constraints), or ask the API owner to issue a separate key for Agent B (manual process that doesn't scale).
          </p>

          <p className="text-gray-300 leading-relaxed">
            There's no cryptographic way for Agent A to create a "sub-key" that has fewer permissions than the original. The API key model has no concept of attenuated delegation — giving someone less access than you have.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Problem 3: No Context About Authority</h3>

          <p className="text-gray-300 leading-relaxed">
            When an API receives a request with an API key, it knows <em>who</em> is calling, but nothing about the <em>context</em> of that call. Is this agent operating under a $10 budget or a $10,000 budget? Is this call part of a low-priority background task or a critical production workflow? Should this agent be able to delegate authority to other agents?
          </p>

          <p className="text-gray-300 leading-relaxed">
            The API key carries identity, but not authority. The API has no way to make different decisions based on the caller's economic constraints, delegation status, or operational context.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Macaroon Model: Capabilities With Built-In Constraints</h2>

          <p className="text-gray-300 leading-relaxed">
            Macaroons flip the authentication model. Instead of asking "who are you?" they embed the answer to "what can you do?" directly into the token. A macaroon is a <strong className="text-white">capability token</strong> — it carries specific permissions, constraints, and delegation rules as part of its cryptographic structure.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Think of it this way: an API key is like a driver's license (proves identity, grants general permission to drive). A macaroon is like a rental car agreement (specifies exactly which car, for how long, with what mileage limit, and whether you can add additional drivers).
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">How Macaroons Work: Capabilities + Caveats</h3>

          <p className="text-gray-300 leading-relaxed">
            A macaroon has two parts: a root capability (what you can access) and a set of caveats (constraints on that access). Both are cryptographically bound together, so you can't remove caveats without invalidating the token.
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`# Root macaroon: access to translation API
root_secret = "ultra-secure-shared-key"
location = "https://api.example.com/translate"
identifier = "user-12345-translate-access"

# Create basic macaroon
macaroon = new_macaroon(root_secret, identifier, location)

# Add constraining caveats
macaroon.add_first_party_caveat("budget_max = 50.00")
macaroon.add_first_party_caveat("endpoints = /translate/*")
macaroon.add_first_party_caveat("expires = 2026-04-01T00:00:00Z")
macaroon.add_first_party_caveat("rate_limit = 100/hour")

# Result: a token that grants translate access with:
# - $50 maximum spend
# - Only translate endpoints
# - Expires April 1st
# - Max 100 calls per hour`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The critical insight: <strong className="text-white">these constraints are enforced by the authentication system, not by the application code.</strong> The API gateway validates the macaroon, checks each caveat, and either allows or denies the request based on the embedded constraints. The API itself never has to think about budgets, rate limits, or expiration — it's all handled at the auth layer.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Attenuation: The Secret Sauce of Delegation</h3>

          <p className="text-gray-300 leading-relaxed">
            Here's where macaroons get powerful: <strong className="text-white">anyone holding a macaroon can add more caveats to create a more restricted token.</strong> This is called attenuation, and it's the foundation of safe delegation.
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`# Agent A has a macaroon with $50 budget
agent_a_macaroon = <root macaroon with budget_max = 50.00>

# Agent A delegates to Agent B, but with stricter limits
agent_b_macaroon = attenuate(agent_a_macaroon, [
  "budget_max = 10.00",      # Stricter than parent ($10 vs $50)
  "endpoints = /translate/en-es",  # More specific endpoints
  "expires = 2026-03-31T12:00:00Z"  # Shorter duration
])

# Agent B delegates to Agent C, even stricter
agent_c_macaroon = attenuate(agent_b_macaroon, [
  "budget_max = 1.00",       # Even stricter ($1 vs $10)
  "rate_limit = 5/hour"      # Additional constraint
])

# Critical: Agent C can NEVER spend more than $1, 
# even if Agent A's original macaroon had $50.
# Attenuation only restricts, never expands.`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            This solves the delegation problem elegantly. Any agent can safely create more restrictive tokens for sub-agents without involving the original API provider. The math is guaranteed by cryptography: attenuated tokens can only have fewer permissions, never more.
          </p>

          <div className="my-8 rounded-2xl border border-purple-900/60 bg-purple-950/20 p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Macaroon Tokens vs API Keys FAQ</h2>
            <div className="space-y-5">
              <div>
                <h3 className="text-xl font-bold text-white mb-2">What is the difference between macaroon tokens and API keys?</h3>
                <p className="text-gray-300 leading-relaxed mb-0">
                  API keys are static bearer secrets that usually identify an account and grant broad access. Macaroon tokens are capability tokens that carry caveats for scope, expiry, budget, delegation, revocation, and audit.
                </p>
              </div>
              <div>
                <h3 className="text-xl font-bold text-white mb-2">Why are macaroons better for AI agents than API keys?</h3>
                <p className="text-gray-300 leading-relaxed mb-0">
                  AI agents delegate work, call tools autonomously, and can spend money quickly. Macaroons let teams issue scoped and attenuated authority so each agent or sub-agent can only use specific APIs within explicit budgets and time windows.
                </p>
              </div>
              <div>
                <h3 className="text-xl font-bold text-white mb-2">Can macaroon tokens enforce AI agent spending limits?</h3>
                <p className="text-gray-300 leading-relaxed mb-0">
                  Yes. Budget caveats can bind a macaroon to a spending limit, task, tool, route, or customer account. A gateway verifies those caveats before each request and blocks or downgrades calls that exceed policy.
                </p>
              </div>
              <div>
                <h3 className="text-xl font-bold text-white mb-2">How does macaroon authentication support revocation?</h3>
                <p className="text-gray-300 leading-relaxed mb-0">
                  Macaroon authentication supports revocation by combining short expirations, caveats, revocation identifiers, and gateway-side deny lists. A compromised agent token can be invalidated without rotating every account-level API key.
                </p>
              </div>
            </div>
          </div>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Side-by-Side Comparison: API Keys vs Macaroons</h2>

          <p className="text-gray-300 leading-relaxed">
            Let's look at how each authentication model handles real AI agent scenarios:
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Scenario 1: Budget-Aware Translation Agent</h3>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Requirement:</strong> Agent needs access to translation API, but spending must be capped at $25.
          </p>

          <div className="bg-red-900/20 border border-red-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">❌ API Key Approach</h4>
            <p className="text-gray-300 text-sm">
              Give agent the account-wide API key. API has no knowledge of $25 budget limit. Need to:
              <br />• Build proxy service that tracks spend per agent
              <br />• Implement budget enforcement outside the API
              <br />• Monitor spending dashboard manually
              <br />• Hope the agent doesn't exceed budget before checks run
            </p>
          </div>

          <div className="bg-green-900/20 border border-green-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">✅ Macaroon Approach</h4>
            <p className="text-gray-300 text-sm">
              Mint macaroon with <code className="text-green-300 bg-black/50 px-1 rounded">budget_max = 25.00</code> caveat. Gateway enforces budget on every request. When $25 is spent, further requests return <code className="text-green-300 bg-black/50 px-1 rounded">402 Payment Required</code>. No proxy needed. No dashboard monitoring. Hard enforcement at the protocol level.
            </p>
          </div>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Scenario 2: Multi-Agent Content Pipeline</h3>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Requirement:</strong> Main agent delegates to research agent ($10 budget), writing agent ($15 budget), and fact-check agent ($5 budget). Each sub-agent should only access relevant endpoints.
          </p>

          <div className="bg-red-900/20 border border-red-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">❌ API Key Approach</h4>
            <p className="text-gray-300 text-sm">
              Either share one API key (all agents see all endpoints, no budget controls) or request three separate API keys from provider (manual process, doesn't scale). No way to ensure research agent can't access writing endpoints.
            </p>
          </div>

          <div className="bg-green-900/20 border border-green-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">✅ Macaroon Approach</h4>
            <p className="text-gray-300 text-sm">
              Main agent attenuates its macaroon three ways:
              <br />• Research agent: <code className="text-green-300 bg-black/50 px-1 rounded">budget_max = 10.00, endpoints = /search/*</code>
              <br />• Writing agent: <code className="text-green-300 bg-black/50 px-1 rounded">budget_max = 15.00, endpoints = /generate/*</code>
              <br />• Fact-check agent: <code className="text-green-300 bg-black/50 px-1 rounded">budget_max = 5.00, endpoints = /verify/*</code>
              <br />Each agent gets exactly the access it needs. Main agent never shares full credentials.
            </p>
          </div>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Scenario 3: Time-Limited Demo Access</h3>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Requirement:</strong> Give prospect agent 24-hour access to test API with $5 demo budget.
          </p>

          <div className="bg-red-900/20 border border-red-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">❌ API Key Approach</h4>
            <p className="text-gray-300 text-sm">
              Create demo account, issue API key, set calendar reminder to revoke access in 24 hours, hope you don't forget. Budget enforcement requires custom billing logic. If demo user exceeds $5, bill arrives 30 days later.
            </p>
          </div>

          <div className="bg-green-900/20 border border-green-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">✅ Macaroon Approach</h4>
            <p className="text-gray-300 text-sm">
              Mint macaroon with <code className="text-green-300 bg-black/50 px-1 rounded">expires = 2026-04-01T12:00:00Z, budget_max = 5.00</code>. Access automatically expires. Budget automatically enforced. No manual cleanup needed. Demo user cannot exceed $5 by design.
            </p>
          </div>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Security Implications: Why Capability-Based Auth is Safer</h2>

          <p className="text-gray-300 leading-relaxed">
            Beyond delegation and budgets, macaroons provide better security properties than API keys for autonomous systems:
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">1. Principle of Least Privilege by Default</h3>

          <p className="text-gray-300 leading-relaxed">
            API keys typically grant broad access that gets refined through application-level authorization. Macaroons start with specific capabilities and can only become more restricted. This inverts the security model: instead of "grant access, then restrict," it's "define minimum necessary access from the start."
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">2. Temporal Authority</h3>

          <p className="text-gray-300 leading-relaxed">
            Every macaroon can include expiration caveats that are cryptographically enforced. API keys typically live forever unless manually revoked. For AI agents that might operate for minutes or hours, automatic expiration eliminates the risk of zombie credentials.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">3. Context-Aware Authorization</h3>

          <p className="text-gray-300 leading-relaxed">
            Macaroons can embed context into the authorization decision. A caveat like <code className="text-green-300 bg-black/50 px-1 rounded">time_of_day = business_hours</code> or <code className="text-green-300 bg-black/50 px-1 rounded">request_rate &lt; 10/min</code> lets you enforce policies that API keys can't express.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">4. Cryptographic Audit Trail</h3>

          <p className="text-gray-300 leading-relaxed">
            Every macaroon embeds its delegation history. You can cryptographically verify not just that a request is authorized, but how many delegation steps led to that authorization and what constraints were added at each step.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">When to Use Each Approach</h2>

          <p className="text-gray-300 leading-relaxed">
            API keys aren't bad — they're just optimized for a different use case. Here's when each approach makes sense:
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Stick with API Keys When:</h3>

          <ul className="text-gray-300 space-y-2 ml-6 list-disc">
            <li>You're building for human developers who manage credentials manually</li>
            <li>Your API has simple binary permissions (access or no access)</li>
            <li>You don't need delegation (one developer, one key, one application)</li>
            <li>Your existing auth infrastructure is working well</li>
            <li>Budget enforcement happens at the account level, not the request level</li>
          </ul>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Switch to Macaroons When:</h3>

          <ul className="text-gray-300 space-y-2 ml-6 list-disc">
            <li>You're building for AI agents that need bounded authority</li>
            <li>You need fine-grained permissions (different budgets, different endpoints, different rates)</li>
            <li>You want to enable safe delegation without manual key management</li>
            <li>You need real-time budget enforcement at the request level</li>
            <li>You want time-limited access tokens that expire automatically</li>
            <li>You're implementing micropayments or pay-per-use billing</li>
          </ul>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Implementation: Adding Macaroon Support to Your API</h2>

          <p className="text-gray-300 leading-relaxed">
            You don't have to choose between API keys and macaroons. Most implementations support both — API keys for human developers, macaroons for AI agents. Here's how to add macaroon support incrementally:
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Step 1: Deploy an Economic Gateway</h3>

          <p className="text-gray-300 leading-relaxed">
            Add a macaroon-aware gateway in front of your existing API. The gateway handles macaroon validation, caveat enforcement, and budget tracking. Your API continues to work exactly as it does today for API key users.
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`┌─────────────┐     ┌─────────────────────┐     ┌──────────┐
│ API Key     │────▶│                      │────▶│          │
│ Request     │     │  Economic Gateway    │     │ Your API │
└─────────────┘     │                      │     │          │
┌─────────────┐     │ • Macaroon validation│     │          │
│ Macaroon    │────▶│ • Budget enforcement │────▶│          │
│ Request     │     │ • Caveat checking    │     │          │
└─────────────┘     └─────────────────────┘     └──────────┘

# Two auth paths, same API backend`}</code>
          </pre>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Step 2: Define Capability Mappings</h3>

          <p className="text-gray-300 leading-relaxed">
            Map your API endpoints to capabilities and define which caveats you'll honor. Start simple:
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`# Example capability mapping
capabilities:
  translate-api:
    endpoints: 
      - /translate/*
      - /languages
    supported_caveats:
      - budget_max
      - rate_limit  
      - expires
      - source_lang
      - target_lang

  premium-translate:
    endpoints:
      - /translate/premium/*
    supported_caveats:
      - budget_max
      - expires
      - customer_tier >= premium`}</code>
          </pre>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Step 3: Mint Your First Macaroons</h3>

          <p className="text-gray-300 leading-relaxed">
            Create restricted tokens for your first AI agent users:
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`import { Macaroon } from 'macaroons.js';

// Mint a macaroon for agent demo access
const demo_macaroon = new Macaroon({
  rootKey: process.env.MACAROON_ROOT_KEY,
  identifier: 'agent-demo-march-2026',
  location: 'https://api.yourservice.com/'
});

demo_macaroon.addFirstPartyCaveat('budget_max = 10.00');
demo_macaroon.addFirstPartyCaveat('expires = 2026-04-07T00:00:00Z');
demo_macaroon.addFirstPartyCaveat('endpoints = /translate/*');

// Agent gets this serialized macaroon as bearer token
const token = demo_macaroon.serialize();`}</code>
          </pre>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Step 4: Enable Agent Self-Attenuation</h3>

          <p className="text-gray-300 leading-relaxed">
            Agents can create more restricted tokens for delegation. Provide simple SDK methods:
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`# Agent A has demo macaroon, delegates to Agent B
attenuated_token = agent_a_token.add_caveats([
  "budget_max = 2.00",           # Subset of parent budget
  "endpoints = /translate/en-es", # Subset of parent endpoints  
  "expires = 2026-04-01T12:00:00Z" # Shorter expiration
])

# Agent B can't spend more than $2 or access other endpoints
# even if Agent A's original token allowed more`}</code>
          </pre>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Real-World Impact: The Numbers</h2>

          <p className="text-gray-300 leading-relaxed">
            Moving from API keys to macaroons isn't just a security improvement — it's an economic enabler. Consider the operational impact:
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Before: API Key Management Overhead</h3>

          <ul className="text-gray-300 space-y-2 ml-6 list-disc">
            <li><strong className="text-white">Manual provisioning:</strong> 30 minutes per new agent integration</li>
            <li><strong className="text-white">Budget monitoring:</strong> Daily dashboard checks, quarterly budget reconciliation</li>
            <li><strong className="text-white">Access revocation:</strong> Manual cleanup when projects end</li>
            <li><strong className="text-white">Incident response:</strong> Hours to identify which agent caused usage spikes</li>
          </ul>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">After: Capability-Based Automation</h3>

          <ul className="text-gray-300 space-y-2 ml-6 list-disc">
            <li><strong className="text-white">Self-service provisioning:</strong> Agents mint restricted tokens instantly</li>
            <li><strong className="text-white">Automatic budget enforcement:</strong> Hard limits prevent overages</li>
            <li><strong className="text-white">Cryptographic expiration:</strong> No manual cleanup needed</li>
            <li><strong className="text-white">Built-in attribution:</strong> Every macaroon carries delegation history</li>
          </ul>

          <p className="text-gray-300 leading-relaxed">
            The productivity gain is significant, but the risk reduction is transformative. With API keys, a runaway agent can blow through a monthly budget in hours. With macaroons, an agent can spend exactly what it's authorized to spend, no more.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Strategic Advantage</h2>

          <p className="text-gray-300 leading-relaxed">
            API providers who adopt capability-based authentication early gain a significant competitive advantage in the agent economy. Here's why:
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">1. Safe Agent Integration</h3>

          <p className="text-gray-300 leading-relaxed">
            Enterprises hesitate to integrate AI agents with critical APIs because of cost and security risks. Macaroons eliminate both concerns. Agents can't exceed their budgets or access unauthorized endpoints by design.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">2. Zero-Friction Micropayments</h3>

          <p className="text-gray-300 leading-relaxed">
            Macaroons enable the holy grail of API monetization: agents that discover your API, pay for access, and start consuming — all without human intervention. Combined with Lightning Network payments, you get sub-second payment settlement for sub-cent API calls.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">3. Delegation-Native Architecture</h3>

          <p className="text-gray-300 leading-relaxed">
            As agent systems become more sophisticated, the ability to delegate safely becomes a core requirement. APIs that support capability-based delegation will integrate seamlessly with multi-agent workflows. APIs that don't will require complex proxy layers.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Getting Started Today</h2>

          <p className="text-gray-300 leading-relaxed">
            You don't need to rebuild your API to add macaroon support. Start with these three steps:
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">1. Deploy SatGate</strong> (or another macaroon-aware gateway) in front of your API. Configure it to pass through API key requests unchanged while adding macaroon validation for new requests.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">2. Define your first capability:</strong> Pick one API endpoint and create a macaroon that grants access with a budget limit. Test it with a simple agent script.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">3. Enable agent self-service:</strong> Publish a tool manifest that lets agents request macaroons with specific constraints. Watch how agents interact with budget-aware APIs differently than unlimited-access APIs.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Bottom Line</h2>

          <p className="text-gray-300 leading-relaxed">
            API keys were designed for a world where humans authenticate to access machine resources. Macaroons were designed for a world where machines authenticate to access machine resources <em>on behalf of humans with specific constraints</em>.
          </p>

          <p className="text-gray-300 leading-relaxed">
            The difference isn't academic. In the agent economy, the authentication model determines which integration patterns are possible, which business models work, and which APIs can safely operate at machine speed and scale.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Your API's next million customers are autonomous agents. The question is whether your authentication system can handle them safely, or whether you'll need proxies, workarounds, and manual oversight to bridge the gap between identity-based auth and capability-based workloads.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Choose capability-based authentication. Choose macaroons. Choose an auth model that scales with autonomous systems instead of fighting them.
          </p>

          <div className="my-10 rounded-2xl border border-blue-900/60 bg-blue-950/20 p-6">
            <h3 className="mb-3 text-xl font-bold text-white">Score your API key risk before agents inherit it</h3>
            <p className="mb-4 text-gray-300">Static API keys become dangerous when autonomous agents can spend, delegate, and retry. Start with a risk assessment, then move to revocable capabilities.</p>
            <div className="flex flex-wrap gap-3 text-sm font-semibold">
              <Link href="/agent-api-key-risk-assessment" className="text-cyan-300 hover:text-cyan-200">API key risk assessment →</Link>
              <Link href="/revocable-capability-token-policy-template" className="text-cyan-300 hover:text-cyan-200">Generate capability-token policy →</Link>
              <Link href="/revocable-agent-credentials" className="text-cyan-300 hover:text-cyan-200">Revocable credentials →</Link>
              <Link href="/agent-capability-tokens" className="text-cyan-300 hover:text-cyan-200">Capability tokens →</Link>
              <Link href="/agent-control-plane" className="text-cyan-300 hover:text-cyan-200">Agent control plane →</Link>
              <Link href="/economic-firewall-readiness-grader" className="text-cyan-300 hover:text-cyan-200">Readiness grader →</Link>
            </div>
          </div>


          <div className="my-10 rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6">
            <h3 className="mb-3 text-xl font-bold text-white">SatGate growth path: Observe → Control → Charge</h3>
            <p className="mb-4 text-gray-300">
              Start by using SatGate to Observe agent, API, and MCP usage. Move to Control when budgets, scopes, and revocation need to stop bad calls before they run. Add Charge when usage should become billable access, chargeback, or robot-customer revenue.
            </p>
            <div className="flex flex-wrap gap-3 text-sm font-semibold">
              <Link href="/mcp-gateway" className="text-cyan-300 hover:text-cyan-200">MCP gateway →</Link>
              <Link href="/capability-auth" className="text-cyan-300 hover:text-cyan-200">Capability auth →</Link>
              <Link href="/govern" className="text-cyan-300 hover:text-cyan-200">See SatGate governance →</Link>
            </div>
          </div>

          {/* CTA Section */}
          <div className="mt-16 bg-gradient-to-r from-blue-900/20 to-cyan-900/20 border border-blue-800/30 rounded-xl p-8">
            <h3 className="text-xl font-bold text-white mb-3">Ready to Add Capability-Based Auth to Your API?</h3>
            <p className="text-gray-300 mb-4">
              SatGate implements macaroon authentication with budget enforcement and delegation support. Deploy in minutes, test with your first AI agent, and scale to autonomous workloads.
            </p>
            <div className="flex flex-wrap gap-3">
              <Link href="/revocable-capability-token-policy-template" className="inline-flex items-center gap-2 bg-white text-black px-6 py-3 rounded-lg font-bold hover:bg-gray-200 transition text-sm">
                Generate Token Policy
              </Link>
              <a href="https://github.com/SatGate-io/satgate" className="inline-flex items-center gap-2 border border-blue-500 text-blue-300 px-6 py-3 rounded-lg font-bold hover:bg-blue-900/30 transition text-sm">
                View on GitHub
              </a>
              <Link href="/design-partners" className="inline-flex items-center gap-2 border border-blue-500 text-blue-300 px-6 py-3 rounded-lg font-bold hover:bg-blue-900/30 transition text-sm">
                Become a Design Partner
              </Link>
            </div>
          </div>

        </article>
      </div>
    </div>
  );
}
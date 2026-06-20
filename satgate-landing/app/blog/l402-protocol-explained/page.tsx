import Link from 'next/link';
import { ArrowLeft, Calendar, Clock } from 'lucide-react';

export const metadata = {
  title: "L402 Protocol Explained: HTTP 402 for Machine API Payments",
  description: "L402 combines HTTP 402, paid-rail context, and macaroon tokens so delegated agents can present payment proof for API access in real time.",
  alternates: { canonical: 'https://satgate.io/blog/l402-protocol-explained' },
  keywords: ['L402 protocol explained', 'L402 protocol', 'HTTP 402 Payment Required', 'Lightning API payments', 'machine-to-machine payments', 'L402 macaroons', 'API micropayments'],
  openGraph: {
    title: 'L402 Protocol Explained: HTTP 402 for Machine API Payments',
    description: 'L402 combines HTTP 402, Lightning invoices, and macaroon tokens so delegated agents can present payment proof for API access in real time.',
    url: 'https://satgate.io/blog/l402-protocol-explained',
    type: 'article',
    publishedTime: '2026-04-02T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'L402 Protocol Explained: HTTP 402 for Machine API Payments',
    description: 'Learn how L402 turns HTTP 402, Lightning invoices, and macaroons into machine-native paid API access.',
  },
};

export default function L402ProtocolExplainedBlogPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'L402 Protocol Explained: How HTTP 402 Enables Machine-Native API Payments',
    description: 'L402 combines HTTP 402, paid-rail context, and macaroon tokens so delegated agents can present payment proof for API access in real time.',
    url: 'https://satgate.io/blog/l402-protocol-explained',
    datePublished: '2026-04-02',
    dateModified: '2026-05-04',
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'L402 protocol' },
      { '@type': 'Thing', name: 'HTTP 402 Payment Required' },
      { '@type': 'Thing', name: 'Lightning API payments' },
      { '@type': 'Thing', name: 'machine-native API monetization' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is the L402 protocol?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'L402 is a machine-native API payment protocol that combines HTTP 402 Payment Required, Lightning invoices, and macaroon tokens so delegated software can present payment proof for API access without a per-request human checkout flow.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does an AI agent use L402 to pay for an API?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'The agent requests a protected API, receives a 402 response with a Lightning invoice and macaroon, pays the invoice, then retries with Authorization: L402 <macaroon>:<preimage> as proof of payment.',
        },
      },
      {
        '@type': 'Question',
        name: 'Is L402 the same as enterprise budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. L402 is per-request Lightning payment for external API monetization. Enterprise budget enforcement controls internal agent spend with policies, caps, revocation, and audit before upstream requests execute.',
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
            <span className="px-2 py-1 rounded-full bg-yellow-900/30 border border-yellow-500/30 text-yellow-300 text-xs font-mono">L402</span>
            <span className="px-2 py-1 rounded-full bg-green-900/30 border border-green-500/30 text-green-300 text-xs font-mono">Payments</span>
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">API Economics</span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">AI Agents</span>
          </div>
          
          <h1 className="text-4xl font-bold mb-4">L402 Protocol Explained: How HTTP 402 Enables Machine-Native API Payments</h1>
          
          <p className="text-xl text-gray-400 mb-6 italic">
            In 1997, the HTTP spec reserved status code 402 &ldquo;for future use.&rdquo; Nearly three decades later, L402 turns that placeholder into a fully functional payment protocol — letting AI agents discover, pay for, and consume APIs without a human in the loop.
          </p>
          
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> April 2, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 11 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">
          
          <p className="text-gray-300 text-lg leading-relaxed">
            Every API you use today requires a human at some point in the payment flow. Someone signs up for a Stripe account. Someone enters a credit card. Someone approves a subscription renewal. The entire API economy is built on the assumption that a person will eventually show up to handle the money.
          </p>

          <p className="text-gray-300 leading-relaxed">
            AI agents break that assumption. An agent making 1,500 API calls per prompt doesn&apos;t have a credit card. A sub-agent spawned at 3 AM to translate a document can&apos;t wait for human approval. An autonomous research pipeline that discovers a new data source mid-task can&apos;t pause to sign up for a developer account.
          </p>

          <p className="text-gray-300 leading-relaxed">
            The <strong className="text-white">L402 protocol</strong> solves this by making API payments machine-native. It combines HTTP&apos;s long-dormant <code className="text-green-300 bg-black/50 px-1 rounded">402 Payment Required</code> status code with Lightning Network micropayments and macaroon proof-of-payment tokens to create a payment flow that requires zero human involvement — from discovery through payment through access.
          </p>

          <p className="text-gray-300 leading-relaxed">
            This post explains exactly how the L402 protocol works, why it matters for the agent economy, and how to implement it today.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">What Is the L402 Protocol?</h2>

          <p className="text-gray-300 leading-relaxed">
            L402 is a protocol that enables machine-to-machine API payments using three components that already exist:
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">1. HTTP 402 Payment Required</strong> — The status code the HTTP spec reserved in 1997 for &ldquo;future use.&rdquo; L402 puts it to work as a standard signal that says: &ldquo;This resource costs money. Here&apos;s how to pay.&rdquo;
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">2. Lightning Network invoices</strong> — Bitcoin&apos;s Layer 2 payment network enables instant, sub-cent micropayments that settle in milliseconds. No credit cards, no bank accounts, no minimum transaction sizes.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">3. Macaroon tokens</strong> — Capability-based bearer tokens that carry cryptographically enforced constraints (budgets, expiration, scopes). Once delegated payment proof is presented, the client receives a macaroon that proves payment and grants access.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Together, these three components create a payment flow where an API can say &ldquo;pay me 50 satoshis to access this endpoint,&rdquo; an agent can pay instantly, and the proof-of-payment doubles as the access credential. No signup. No API keys. No human approval.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">How L402 Works: Step by Step</h2>

          <p className="text-gray-300 leading-relaxed">
            The L402 protocol flow is elegant in its simplicity. Here&apos;s what happens when an AI agent encounters an L402-protected API:
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`Step 1: Agent makes an API request
─────────────────────────────────
GET /api/v1/translate?text=hello&target=es
Host: api.example.com

Step 2: Server returns 402 with payment details
─────────────────────────────────────────────────
HTTP/1.1 402 Payment Required
WWW-Authenticate: L402 macaroon="<base64-macaroon>", invoice="<lightning-invoice>"
Content-Type: application/json

{
  "message": "Payment required",
  "amount": 50,
  "unit": "satoshis",
  "description": "Translation API - 1 request"
}

Step 3: Delegated client presents Lightning payment proof (~200ms)
─────────────────────────────────────────────────
Agent's Lightning wallet pays the invoice automatically.
Payment settles in milliseconds. Agent receives preimage
(cryptographic proof-of-payment).

Step 4: Agent retries with proof-of-payment
────────────────────────────────────────────
GET /api/v1/translate?text=hello&target=es
Host: api.example.com
Authorization: L402 <macaroon>:<preimage>

Step 5: Server validates and responds
──────────────────────────────────────
HTTP/1.1 200 OK
Content-Type: application/json

{
  "translation": "hola",
  "source": "en",
  "target": "es"
}`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The entire flow — from initial request through payment through access — happens in under a second. No account creation. No OAuth flow. No API key provisioning. The payment <em>is</em> the authentication.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">The Macaroon: More Than a Receipt</h3>

          <p className="text-gray-300 leading-relaxed">
            The macaroon returned in the <code className="text-green-300 bg-black/50 px-1 rounded">WWW-Authenticate</code> header isn&apos;t just a receipt — it&apos;s a capability token with embedded constraints called caveats. These caveats define exactly what the payment grants access to:
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`# Decoded macaroon caveats
token_id = "txn-a1b2c3d4"
scope = "/api/v1/translate"
expires = "2026-04-02T15:00:00Z"
budget_id = "budget-xyz-789"

# The macaroon is cryptographically bound:
# - Can't remove caveats without invalidating the token
# - Can add MORE caveats to create restricted sub-tokens
# - Preimage proves the Lightning payment was made`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            This is what makes L402 fundamentally different from simple pay-per-request schemes. The macaroon carries <em>capabilities</em>, not just payment proof. And because macaroons support attenuation, an agent can create more restricted tokens for sub-agents — enabling the kind of delegated authority that multi-agent systems require.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Why Existing Payment Models Fail for AI Agents</h2>

          <p className="text-gray-300 leading-relaxed">
            Before diving into comparisons, let&apos;s understand why existing API payment models weren&apos;t built for autonomous consumers:
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Subscriptions Require Signup</h3>

          <p className="text-gray-300 leading-relaxed">
            Monthly subscriptions assume a human will fill out a form, enter billing details, and agree to terms. An AI agent that discovers a useful API at 2 AM can&apos;t complete a signup flow. Subscriptions also force commitment before usage — agents don&apos;t know if they&apos;ll need 10 requests or 10,000.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">OAuth Requires Human Consent</h3>

          <p className="text-gray-300 leading-relaxed">
            OAuth flows are designed around a human clicking &ldquo;Authorize&rdquo; in a browser. They work beautifully for human-in-the-loop integrations. They fail completely for autonomous agents that need to access new services dynamically.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">API Keys Lack Payment</h3>

          <p className="text-gray-300 leading-relaxed">
            API keys handle authentication but have no concept of payment. They&apos;re issued by a provider to a known customer. There&apos;s no mechanism for an unknown agent to obtain an API key, pay for access, and start consuming — all in a single automated flow.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">Credit Card Processing Is Too Slow and Expensive</h3>

          <p className="text-gray-300 leading-relaxed">
            Credit card transactions have minimum fees (typically $0.30 + 2.9%), take days to settle, and require merchant accounts. When an agent needs to make a $0.001 API call, credit card infrastructure is three orders of magnitude too expensive and too slow.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">L402 vs Traditional API Payment Models</h2>

          <p className="text-gray-300 leading-relaxed">
            Here&apos;s how L402 stacks up against the payment models APIs use today:
          </p>

          <div className="bg-red-900/20 border border-red-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">❌ Traditional: Subscription + API Key</h4>
            <p className="text-gray-300 text-sm">
              <br />• Requires human signup and credit card on file
              <br />• Monthly commitment regardless of actual usage
              <br />• No delegation — keys are tied to one account
              <br />• Overages billed after the fact (30-day invoice cycles)
              <br />• Minimum viable customer: a developer with a Stripe account
            </p>
          </div>

          <div className="bg-green-900/20 border border-green-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">✅ L402: Pay-Per-Request with Instant Settlement</h4>
            <p className="text-gray-300 text-sm">
              <br />• Zero signup — payment <em>is</em> the authentication
              <br />• Pay only for what you use, per request
              <br />• Macaroon tokens support safe delegation to sub-agents
              <br />• Payment settles before the response is sent — no invoicing
              <br />• Minimum viable customer: any agent with a Lightning wallet
            </p>
          </div>

          <div className="bg-red-900/20 border border-red-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">❌ Traditional: Usage-Based Billing (Stripe Metered)</h4>
            <p className="text-gray-300 text-sm">
              <br />• Records usage, bills monthly — risk of non-payment
              <br />• Minimum transaction costs make sub-cent pricing impossible
              <br />• Provider takes on credit risk for the billing period
              <br />• Agents can accumulate unbounded charges before cutoff
              <br />• Reconciliation happens days or weeks after consumption
            </p>
          </div>

          <div className="bg-green-900/20 border border-green-800/30 rounded-lg p-6 my-6">
            <h4 className="text-white font-bold mb-2">✅ L402: Prepaid Micropayments</h4>
            <p className="text-gray-300 text-sm">
              <br />• Payment before access — zero credit risk for the provider
              <br />• Lightning enables sub-cent transactions (1 satoshi ≈ $0.001)
              <br />• No minimum transaction size — charge exactly what the call costs
              <br />• Real-time settlement — revenue hits your wallet instantly
              <br />• No billing infrastructure needed — the protocol <em>is</em> the billing
            </p>
          </div>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Real-World Use Cases</h2>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">1. API Monetization Without Billing Infrastructure</h3>

          <p className="text-gray-300 leading-relaxed">
            You built a useful API. You want to charge for it. With traditional billing, you need Stripe integration, customer management, invoicing, dunning emails, and a pricing page. With L402, you add a gateway in front of your API that returns <code className="text-green-300 bg-black/50 px-1 rounded">402 Payment Required</code> with a Lightning invoice. Your first paying customer can arrive in the time it takes to deploy the gateway.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">2. Agent-to-Agent Commerce</h3>

          <p className="text-gray-300 leading-relaxed">
            In multi-agent systems, agents regularly need services from other tools or APIs. A research workflow may need paid data; a coding workflow may need paid test execution. L402 can make the proof flow request-native, while SatGate keeps human or platform authority, scope, and budget explicit.
          </p>

          <h3 className="text-xl font-bold text-white mt-8 mb-3">3. MCP Tool Access and Monetization</h3>

          <p className="text-gray-300 leading-relaxed">
            The Model Context Protocol (MCP) lets AI agents discover and use tools dynamically. L402 adds the missing economic layer — tool providers can charge per invocation, and delegated clients can present payment proof instantly. Note that in SatGate&apos;s architecture, MCP tool access uses enterprise budget enforcement via Redis and PostgreSQL for fiat-based billing, while L402 operates as a per-payment protocol on the HTTP gateway layer. The two complement each other: L402 for permissionless micropayments, enterprise budgets for organizational cost control.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Implementing L402 with SatGate</h2>

          <p className="text-gray-300 leading-relaxed">
            SatGate implements L402 as a policy mode in its HTTP gateway. When you configure a route with the <code className="text-green-300 bg-black/50 px-1 rounded">l402</code> policy (also available as the <code className="text-green-300 bg-black/50 px-1 rounded">charge</code> or <code className="text-green-300 bg-black/50 px-1 rounded">monetize</code> aliases), every request to that route must include a valid L402 token. No token? The gateway returns <code className="text-green-300 bg-black/50 px-1 rounded">402 Payment Required</code> with a Lightning invoice.
          </p>

          <pre className="bg-gray-900/70 border border-gray-800 rounded-lg p-4 overflow-x-auto text-sm my-6">
            <code className="text-green-300">{`# satgate.yaml — L402 route configuration
routes:
  - path: /api/v1/translate
    target: http://localhost:8080
    policy: l402          # Also accepts "charge" or "monetize"
    price_sats: 50        # Cost per request in satoshis
    description: "Translation API — per-request pricing"

  - path: /api/v1/premium
    target: http://localhost:8080
    policy: l402
    price_sats: 200
    description: "Premium endpoint — higher quality, higher price"

# How it works in the gateway:
# 1. Request arrives at /api/v1/translate
# 2. Gateway checks for valid L402 token in Authorization header
# 3. No token → return 402 with Lightning invoice for 50 sats
# 4. Valid token → proxy request to target backend
# 5. L402 is per-payment: no budget tracking needed`}</code>
          </pre>

          <p className="text-gray-300 leading-relaxed">
            The key difference between L402 and SatGate&apos;s other policy modes: <strong className="text-white">L402 is per-payment with no budget enforcement.</strong> Each request is a discrete payment. The gateway validates the L402 token format — it doesn&apos;t track cumulative spend or enforce budget ceilings. That&apos;s by design: L402 is a micropayment protocol, not a cost management framework.
          </p>

          <p className="text-gray-300 leading-relaxed">
            For use cases that need budget enforcement on top of payments — say, an enterprise that wants agents to pay per-call but also cap total department spend — SatGate offers the <code className="text-green-300 bg-black/50 px-1 rounded">fiat402</code> policy, which combines payment requirements with Redis-backed budget tracking. Different tool for different requirements.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">The Bigger Picture: L402 as Agent Economy Infrastructure</h2>

          <p className="text-gray-300 leading-relaxed">
            L402 isn&apos;t just a payment protocol. It&apos;s infrastructure for a new kind of economy where machines are the primary consumers of digital services.
          </p>

          <p className="text-gray-300 leading-relaxed">
            Consider what happens when API payments become machine-native:
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Discovery becomes economic.</strong> An agent searching for a translation service doesn&apos;t just evaluate quality and latency — it evaluates price. APIs compete on cost per request, and agents route to the cheapest provider that meets their quality threshold. Markets emerge for every API category.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Access becomes permissionless.</strong> Any agent with a Lightning wallet can use any L402-protected API. No partnerships. No sales calls. No enterprise agreements. The barrier to entry for both providers and consumers drops to near zero.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Monetization becomes instant.</strong> You deploy an API, set a price in satoshis, and delegated clients can present proof immediately. Revenue does not wait for monthly billing cycles — it arrives with approved requests. A solo developer with a useful model can earn money from governed agent consumption within minutes of deployment.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">Delegation becomes safe.</strong> Because L402 uses macaroons, agents can create attenuated tokens for sub-agents — granting limited access that can never exceed the parent&apos;s authority. Multi-agent workflows can operate with proper economic boundaries at every level.
          </p>

          <p className="text-gray-300 leading-relaxed">
            This is why HTTP 402 was worth waiting for. The original spec authors couldn&apos;t have predicted AI agents, but they correctly intuited that HTTP would eventually need a native way to say &ldquo;this costs money.&rdquo; L402 fulfills that vision with a protocol stack — Lightning for payment, macaroons for proof, HTTP 402 for signaling — that&apos;s purpose-built for machine-speed commerce.
          </p>

          <h2 className="text-2xl font-bold text-white mt-12 mb-4">Getting Started with L402</h2>

          <p className="text-gray-300 leading-relaxed">
            If you&apos;re building APIs that agents will consume, L402 is worth understanding now — even if you don&apos;t implement it today. The protocol is open, the tooling is maturing, and the demand from agent developers is growing.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">For API providers:</strong> Start with one endpoint. Set a price. Deploy SatGate with the <code className="text-green-300 bg-black/50 px-1 rounded">l402</code> policy and see what happens when delegated clients can present payment proof without a per-request signup flow.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">For agent developers:</strong> Add Lightning wallet support to your agent and teach it to handle <code className="text-green-300 bg-black/50 px-1 rounded">402 Payment Required</code> responses. Your agent gains access to every L402-protected API on the internet — no integration work per service.
          </p>

          <p className="text-gray-300 leading-relaxed">
            <strong className="text-white">For enterprises:</strong> L402 and budget enforcement aren&apos;t mutually exclusive. Use L402 for external API monetization and SatGate&apos;s enterprise budget policies for internal cost control. The gateway handles both.
          </p>

          <section className="not-prose mt-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-yellow-300">FAQ</p>
            <h2 className="mb-6 text-2xl font-bold text-white">L402 protocol questions</h2>
            <div className="space-y-5">
              {[
                ['What is the L402 protocol?', 'L402 is a machine-native API payment protocol that combines HTTP 402 Payment Required, Lightning invoices, and macaroon tokens so delegated software can present payment proof for API access without a per-request human checkout flow.'],
                ['How does a delegated agent use L402 for paid API access?', 'The delegated client requests a protected API, receives a 402 response with a Lightning invoice and macaroon, presents proof, then retries with Authorization: L402 <macaroon>:<preimage> as proof of payment.'],
                ['Is L402 the same as enterprise budget enforcement?', 'No. L402 is per-request Lightning payment for external API monetization. Enterprise budget enforcement controls internal agent spend with policies, caps, revocation, and audit before upstream requests execute.'],
              ].map(([question, answer]) => (
                <div key={question} className="border-t border-gray-800 pt-5 first:border-t-0 first:pt-0">
                  <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                  <p className="leading-relaxed text-gray-400">{answer}</p>
                </div>
              ))}
            </div>
          </section>

          {/* CTA Section */}
          <div className="mt-16 bg-gradient-to-r from-yellow-900/20 to-green-900/20 border border-yellow-800/30 rounded-xl p-8">
            <h3 className="text-xl font-bold text-white mb-3">Ready to Add L402 Payments to Your API?</h3>
            <p className="text-gray-300 mb-4">
              SatGate implements L402 as a gateway policy — deploy in front of any API to enable machine-native micropayments with zero changes to your backend. Open source. Use the OSS gateway with your own operational gates, policies, and receipts.
            </p>
            <div className="flex flex-wrap gap-3">
              <a href="https://github.com/SatGate-io/satgate" className="inline-flex items-center gap-2 bg-white text-black px-6 py-3 rounded-lg font-bold hover:bg-gray-200 transition text-sm">
                View on GitHub
              </a>
              <Link href="/design-partners" className="inline-flex items-center gap-2 border border-yellow-500 text-yellow-300 px-6 py-3 rounded-lg font-bold hover:bg-yellow-900/30 transition text-sm">
                Become a Design Partner
              </Link>
            </div>
          </div>

        </article>
      </div>
    </div>
  );
}

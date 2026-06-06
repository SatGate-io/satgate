import Link from 'next/link';
import { ArrowLeft, Calendar, Clock, Shield, Lock, DollarSign, ArrowRight, CheckCircle, Zap, GitBranch, Eye, Activity, BookOpen } from 'lucide-react';

export const metadata = {
  title: 'Intelligent AI Delegation: Macaroons, Capability Tokens, and SatGate',
  description: 'DeepMind\'s Intelligent AI Delegation points to macaroon capability tokens for safe AI agent delegation. SatGate implements request-path delegation controls.',
  alternates: { canonical: 'https://satgate.io/blog/deepmind-intelligent-delegation-satgate' },
  keywords: ['AI agent delegation', 'macaroon tokens', 'capability-based security', 'Google DeepMind', 'delegation capability tokens', 'agent economy', 'economic access control', 'SatGate', 'privilege attenuation', 'agentic web'],
  openGraph: {
    title: 'Intelligent AI Delegation: Macaroons and Capability Tokens',
    description: 'DeepMind-style AI delegation needs macaroon capability tokens, scoped budgets, revocation, and request-path controls.',
    url: 'https://satgate.io/blog/deepmind-intelligent-delegation-satgate',
    type: 'article',
    publishedTime: '2026-03-11T00:00:00Z',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Intelligent AI Delegation: Macaroons and Capability Tokens',
    description: 'AI agent delegation needs attenuated authority, budget caveats, revocation, and request-path enforcement.',
  },
};

export default function DeepMindDelegationPage() {
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: 'Intelligent AI Delegation: Macaroons, Capability Tokens, and SatGate',
    description: metadata.description,
    author: { '@type': 'Organization', name: 'SatGate' },
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    datePublished: '2026-03-11',
    dateModified: '2026-06-04',
    mainEntityOfPage: 'https://satgate.io/blog/deepmind-intelligent-delegation-satgate',
    about: [
      { '@type': 'Thing', name: 'intelligent AI delegation' },
      { '@type': 'Thing', name: 'delegation capability tokens' },
      { '@type': 'Thing', name: 'macaroons for AI agents' },
      { '@type': 'Thing', name: 'privilege attenuation for agent chains' },
      { '@type': 'Thing', name: 'request-path delegation controls' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What are Delegation Capability Tokens for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Delegation Capability Tokens are scoped credentials that let an agent pass limited authority to another agent. The useful form is attenuated: each delegation can only narrow permissions, budgets, routes, tools, or time windows.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why are macaroons a strong primitive for AI agent delegation?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Macaroons support cryptographic caveats, local verification, and privilege attenuation. A parent token can mint child tokens with stricter constraints, but a child token cannot expand authority beyond its parent.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate implement intelligent agent delegation?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate uses macaroon-based capability tokens with caveats for route scope, MCP tool scope, budgets, expiry, and delegation chains, then enforces those constraints in the request path before upstream APIs execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'What does Intelligent AI Delegation require beyond task routing?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Intelligent AI delegation needs explicit authority transfer, attenuated permissions, resource budgets, accountability across delegation chains, revocation, and enforcement before delegated agents can call APIs, MCP tools, or paid services.',
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
            <span className="px-2 py-1 rounded-full bg-blue-900/30 border border-blue-500/30 text-blue-300 text-xs font-mono">Research</span>
            <span className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono">Delegation</span>
            <span className="px-2 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono">Macaroons</span>
          </div>
          <h1 className="text-4xl md:text-5xl font-extrabold tracking-tight mb-4">
            What Google DeepMind Gets Right About Agent Delegation
          </h1>
          <p className="text-xl text-gray-400 mb-4">And what SatGate already built</p>
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1"><Calendar size={14} /> March 11, 2026</span>
            <span className="flex items-center gap-1"><Clock size={14} /> 12 min read</span>
          </div>
        </header>

        <article className="prose prose-invert prose-lg max-w-none">

          {/* Intro */}
          <p className="text-xl text-gray-300 leading-relaxed">
            In February 2026, a team at Google DeepMind published{' '}
            <a href="https://arxiv.org/abs/2602.11865" target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:text-blue-300">
              &ldquo;Intelligent AI Delegation&rdquo;
            </a>
            {' '}— a framework for how autonomous agents should safely decompose tasks, transfer authority,
            and maintain accountability across delegation chains. The paper is dense, thorough, and arrives at
            a conclusion we find familiar: <strong>agents need attenuated capability tokens — specifically
            macaroons — to delegate safely.</strong>
          </p>

          <p className="text-gray-300 leading-relaxed">
            We didn&apos;t build SatGate because of this paper. We built it because macaroons are the only
            credential primitive that actually works for machine-to-machine delegation: they attenuate,
            they carry caveats, and they&apos;re cryptographically verifiable without phoning home.
            But when Google DeepMind independently arrives at the same architecture, it&apos;s worth
            walking through the overlap.
          </p>

          {/* The Core Thesis */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">The Paper&apos;s Core Thesis</h2>
          <p className="text-gray-300 leading-relaxed">
            The DeepMind team argues that current delegation approaches — simple task decomposition,
            round-robin assignment, prompt chaining — fail when agents need to operate across trust
            boundaries. Their framework defines delegation as more than task routing: it requires
            explicit <strong>transfer of authority</strong>, <strong>clear boundaries</strong>,{' '}
            <strong>accountability mechanisms</strong>, and <strong>resource constraints</strong>.
          </p>
          <p className="text-gray-300 leading-relaxed">
            In Section 6.1, they propose <strong>Delegation Capability Tokens (DCTs)</strong> based on
            macaroons as the cryptographic primitive to make this work. Their example:
          </p>

          <div className="bg-gray-900 border border-gray-800 rounded-lg p-6 my-8">
            <div className="flex items-center gap-2 mb-3">
              <BookOpen className="text-blue-400" size={18} />
              <span className="text-blue-300 text-sm font-mono">DeepMind, Section 6.1</span>
            </div>
            <p className="text-gray-300 italic text-base m-0">
              &ldquo;A delegator would mint a DCT that wraps the target resource credentials with
              cryptographic caveats. The attenuation could be defined as &lsquo;This token can access
              the designated Google Drive MCP server, BUT ONLY for folder Project_X AND ONLY for
              READ operations.&rsquo;&rdquo;
            </p>
          </div>

          <p className="text-gray-300 leading-relaxed">
            If you&apos;ve used SatGate, this should sound familiar. That&apos;s a macaroon with two
            first-party caveats — exactly what SatGate tokens are.
          </p>

          {/* The Mapping */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Framework ↔ Implementation</h2>
          <p className="text-gray-300 leading-relaxed">
            Here&apos;s how the paper&apos;s requirements map to what SatGate ships today:
          </p>

          <div className="space-y-4 my-8">
            {/* Row 1 */}
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
              <div className="flex items-start gap-4">
                <div className="bg-blue-900/30 rounded-lg p-2.5 shrink-0 mt-0.5">
                  <Lock className="text-blue-400" size={20} />
                </div>
                <div>
                  <p className="text-white font-semibold mb-1">Delegation Capability Tokens (DCT)</p>
                  <p className="text-gray-500 text-sm mb-2">Paper: Attenuated tokens with cryptographic caveats for scoped authority</p>
                  <p className="text-gray-300 text-sm">
                    <span className="text-cyan-400 font-mono text-xs">SatGate →</span> Every token is a macaroon.
                    Caveats enforce route restrictions, budget limits, time windows, and MCP tool scopes.
                    Tokens are minted with <code className="text-purple-300 bg-gray-800 px-1.5 py-0.5 rounded text-xs">satgate token mint</code> and
                    carry their constraints cryptographically — no database lookup required for verification.
                  </p>
                </div>
              </div>
            </div>

            {/* Row 2 */}
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
              <div className="flex items-start gap-4">
                <div className="bg-purple-900/30 rounded-lg p-2.5 shrink-0 mt-0.5">
                  <GitBranch className="text-purple-400" size={20} />
                </div>
                <div>
                  <p className="text-white font-semibold mb-1">Privilege Attenuation</p>
                  <p className="text-gray-500 text-sm mb-2">Paper: Sub-agents receive strictly fewer permissions than their delegator</p>
                  <p className="text-gray-300 text-sm">
                    <span className="text-cyan-400 font-mono text-xs">SatGate →</span> Delegation trees. A parent token can
                    mint child tokens with additional caveats, but can never grant more authority than it holds.
                    Budget allocation flows downward — a $100 parent can create ten $10 children, each scoped to
                    specific routes or tools.
                  </p>
                </div>
              </div>
            </div>

            {/* Row 3 */}
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
              <div className="flex items-start gap-4">
                <div className="bg-green-900/30 rounded-lg p-2.5 shrink-0 mt-0.5">
                  <DollarSign className="text-green-400" size={20} />
                </div>
                <div>
                  <p className="text-white font-semibold mb-1">Resource Constraints</p>
                  <p className="text-gray-500 text-sm mb-2">Paper: Explicit boundaries on what resources a delegated agent can consume</p>
                  <p className="text-gray-300 text-sm">
                    <span className="text-cyan-400 font-mono text-xs">SatGate →</span> Per-agent budget ceilings enforced at the
                    request layer. When a token hits its spend limit, the gateway returns HTTP 402 — the request
                    never reaches the upstream. Budget enforcement is pre-execution, not post-billing.
                  </p>
                </div>
              </div>
            </div>

            {/* Row 4 */}
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
              <div className="flex items-start gap-4">
                <div className="bg-yellow-900/30 rounded-lg p-2.5 shrink-0 mt-0.5">
                  <Eye className="text-yellow-400" size={20} />
                </div>
                <div>
                  <p className="text-white font-semibold mb-1">Accountability &amp; Audit</p>
                  <p className="text-gray-500 text-sm mb-2">Paper: Clear chain of responsibility with oversight mechanisms</p>
                  <p className="text-gray-300 text-sm">
                    <span className="text-cyan-400 font-mono text-xs">SatGate →</span> Every request is logged with full token
                    lineage — which parent minted it, what caveats it carries, what it spent. Token revocation
                    cascades: revoking a parent instantly invalidates all children. The Evidence Pack is the
                    accountability mechanism.
                  </p>
                </div>
              </div>
            </div>

            {/* Row 5 */}
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
              <div className="flex items-start gap-4">
                <div className="bg-red-900/30 rounded-lg p-2.5 shrink-0 mt-0.5">
                  <Zap className="text-red-400" size={20} />
                </div>
                <div>
                  <p className="text-white font-semibold mb-1">Authority Thresholds &amp; Kill Switches</p>
                  <p className="text-gray-500 text-sm mb-2">Paper: Human-in-the-loop intervention when risk exceeds tolerance (pp. 18–19)</p>
                  <p className="text-gray-300 text-sm">
                    <span className="text-cyan-400 font-mono text-xs">SatGate →</span> Enforcement modes (Observe → Control → Prove)
                    let operators graduate trust incrementally. Budget alerts trigger before limits hit.
                    Token revocation is immediate — one API call kills an agent&apos;s access across the
                    entire delegation tree.
                  </p>
                </div>
              </div>
            </div>

            {/* Row 6 */}
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
              <div className="flex items-start gap-4">
                <div className="bg-orange-900/30 rounded-lg p-2.5 shrink-0 mt-0.5">
                  <Zap className="text-orange-400" size={20} />
                </div>
                <div>
                  <p className="text-white font-semibold mb-1">Smart Contracts for Payment Settlement</p>
                  <p className="text-gray-500 text-sm mb-2">Paper: Blockchain escrow, staking, and automated payment release (pp. 20–22, 26–28)</p>
                  <p className="text-gray-300 text-sm">
                    <span className="text-cyan-400 font-mono text-xs">SatGate →</span> L402 — Lightning Network micropayment settlement.
                    Instant, autonomous, no intermediary, no gas fees. Where the paper proposes heavyweight smart contracts,
                    SatGate uses Lightning for sub-cent machine-to-machine payments that settle in milliseconds.
                    For enterprises not ready for crypto, Fiat402 provides the same budget controls over fiat rails.
                  </p>
                </div>
              </div>
            </div>

            {/* Row 7 */}
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
              <div className="flex items-start gap-4">
                <div className="bg-teal-900/30 rounded-lg p-2.5 shrink-0 mt-0.5">
                  <DollarSign className="text-teal-400" size={20} />
                </div>
                <div>
                  <p className="text-white font-semibold mb-1">Escrow + Staking + Slashing</p>
                  <p className="text-gray-500 text-sm mb-2">Paper: Economic penalties and bonds to incentivize correct behavior (pp. 19–21)</p>
                  <p className="text-gray-300 text-sm">
                    <span className="text-cyan-400 font-mono text-xs">SatGate →</span> Budget enforcement with auto-revocation.
                    When a token exhausts its budget, it&apos;s immediately invalid — no request reaches the upstream.
                    On revoke, unspent child credits atomically return to the parent via a Lua-scripted Redis cascade.
                    The economic incentive is structural: agents that overspend lose access.
                  </p>
                </div>
              </div>
            </div>

            {/* Row 8 */}
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
              <div className="flex items-start gap-4">
                <div className="bg-indigo-900/30 rounded-lg p-2.5 shrink-0 mt-0.5">
                  <Activity className="text-indigo-400" size={20} />
                </div>
                <div>
                  <p className="text-white font-semibold mb-1">Continuous Monitoring &amp; Signed Attestations</p>
                  <p className="text-gray-500 text-sm mb-2">Paper: Real-time observability and tamper-proof audit records (pp. 13, 16–17)</p>
                  <p className="text-gray-300 text-sm">
                    <span className="text-cyan-400 font-mono text-xs">SatGate →</span> Event streaming — every tool call, budget spend,
                    session lifecycle, and task correlation is published as a structured event in real-time.
                    The token spend ledger provides a complete Evidence Pack: which agent called what tool, at what cost,
                    under which delegation chain.
                  </p>
                </div>
              </div>
            </div>

            {/* Row 9 */}
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
              <div className="flex items-start gap-4">
                <div className="bg-pink-900/30 rounded-lg p-2.5 shrink-0 mt-0.5">
                  <Shield className="text-pink-400" size={20} />
                </div>
                <div>
                  <p className="text-white font-semibold mb-1">Circuit Breakers &amp; Automated Revocation</p>
                  <p className="text-gray-500 text-sm mb-2">Paper: Algorithmic circuit breakers that invalidate tokens across delegation chains (pp. 18–19)</p>
                  <p className="text-gray-300 text-sm">
                    <span className="text-cyan-400 font-mono text-xs">SatGate →</span> Governance service with cascade revocation.
                    Banning a parent token instantly invalidates every child and grandchild in the delegation tree.
                    Rate limiting acts as an additional circuit breaker — runaway agents hit throttling before they can
                    cause economic damage.
                  </p>
                </div>
              </div>
            </div>

            {/* Row 10 */}
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
              <div className="flex items-start gap-4">
                <div className="bg-cyan-900/30 rounded-lg p-2.5 shrink-0 mt-0.5">
                  <Activity className="text-cyan-400" size={20} />
                </div>
                <div>
                  <p className="text-white font-semibold mb-1">Task-Level Cost Tracking</p>
                  <p className="text-gray-500 text-sm mb-2">Paper: Multi-objective optimization across cost, quality, and latency (pp. 8–9, 12)</p>
                  <p className="text-gray-300 text-sm">
                    <span className="text-cyan-400 font-mono text-xs">SatGate →</span> Task-aware budget tracking correlates spend
                    by task ID across multi-step agent workflows. When an MCP task spawns retries or subtasks,
                    SatGate tracks cumulative cost per task — not just per request. This enables the cost dimension
                    of the multi-objective optimization the paper describes.
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Why Macaroons */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Why Macaroons — Not JWTs, Not API Keys</h2>
          <p className="text-gray-300 leading-relaxed">
            The paper&apos;s choice of macaroons isn&apos;t incidental. They explicitly cite{' '}
            <a href="https://research.google/pubs/pub41892/" target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:text-blue-300">
              Birgisson et al. (2014)
            </a>
            {' '}— the original Google Research macaroons paper. Here&apos;s why this matters for agent infrastructure:
          </p>

          <div className="space-y-3 my-8">
            <div className="flex items-start gap-3">
              <CheckCircle className="text-green-400 shrink-0 mt-1" size={16} />
              <p className="text-gray-300 text-sm m-0">
                <strong className="text-white">Attenuation without coordination.</strong> A token holder can add
                restrictions (caveats) without contacting the issuer. Agent A can give Agent B a more restricted
                version of its own token — no round-trip to an auth server.
              </p>
            </div>
            <div className="flex items-start gap-3">
              <CheckCircle className="text-green-400 shrink-0 mt-1" size={16} />
              <p className="text-gray-300 text-sm m-0">
                <strong className="text-white">Offline verification.</strong> Macaroons are HMAC chains.
                The gateway verifies them cryptographically — no database lookup, no token introspection endpoint,
                no latency penalty.
              </p>
            </div>
            <div className="flex items-start gap-3">
              <CheckCircle className="text-green-400 shrink-0 mt-1" size={16} />
              <p className="text-gray-300 text-sm m-0">
                <strong className="text-white">Composable constraints.</strong> Stack caveats: route + budget +
                time + MCP tool scope. Each caveat narrows the token&apos;s authority. They compose naturally —
                unlike JWT claims, which are fixed at signing time.
              </p>
            </div>
            <div className="flex items-start gap-3">
              <CheckCircle className="text-green-400 shrink-0 mt-1" size={16} />
              <p className="text-gray-300 text-sm m-0">
                <strong className="text-white">Delegation is a first-class operation.</strong> Minting a child
                token isn&apos;t a special API — it&apos;s appending a caveat. The delegation hierarchy is
                encoded in the token itself.
              </p>
            </div>
          </div>

          <p className="text-gray-300 leading-relaxed">
            JWTs can&apos;t do this. API keys can&apos;t do this. OAuth scopes are static at grant time.
            Macaroons are the only credential primitive where the <em>holder</em> can reduce their own authority
            and pass it downstream — which is exactly what agent delegation requires.
          </p>

          {/* What the paper doesn't cover */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">What the Paper Doesn&apos;t Cover</h2>
          <p className="text-gray-300 leading-relaxed">
            The DeepMind framework is strong on the <em>what</em> — the requirements for safe delegation.
            It&apos;s intentionally abstract on the <em>how</em>. A few things we&apos;ve learned from building this in production:
          </p>

          <div className="space-y-3 my-8">
            <div className="flex items-start gap-3">
              <ArrowRight className="text-gray-500 shrink-0 mt-1" size={16} />
              <p className="text-gray-300 text-sm m-0">
                <strong className="text-white">Economics is the missing enforcement layer.</strong> The paper
                discusses authority and accountability but doesn&apos;t address the economic dimension. In practice,
                the most common agent failure mode isn&apos;t unauthorized access — it&apos;s uncontrolled spend.
                Budget enforcement is where theory meets the real world.
              </p>
            </div>
            <div className="flex items-start gap-3">
              <ArrowRight className="text-gray-500 shrink-0 mt-1" size={16} />
              <p className="text-gray-300 text-sm m-0">
                <strong className="text-white">MCP changes the surface area.</strong> The paper references MCP
                (Model Context Protocol) but doesn&apos;t dig into per-tool cost attribution. When an agent calls
                10 MCP tools in a session, you need cost tracking at the tool level, not just the session level.
                SatGate parses MCP JSON-RPC to attribute spend per tool per agent.
              </p>
            </div>
            <div className="flex items-start gap-3">
              <ArrowRight className="text-gray-500 shrink-0 mt-1" size={16} />
              <p className="text-gray-300 text-sm m-0">
                <strong className="text-white">Enterprises need a trust gradient, not a binary switch.</strong> The
                paper discusses trust establishment. In practice, operators want to observe first, then control, then
                prove. SatGate&apos;s three-mode progression (Observe → Control → Prove) lets teams build confidence
                incrementally without rearchitecting.
              </p>
            </div>
            <div className="flex items-start gap-3">
              <ArrowRight className="text-gray-500 shrink-0 mt-1" size={16} />
              <p className="text-gray-300 text-sm m-0">
                <strong className="text-white">Blockchain is the wrong settlement layer for agents.</strong> The paper
                proposes smart contracts and escrow bonds (pp. 20–22). In practice, on-chain settlement adds latency, gas
                costs, and complexity that machine-to-machine micropayments can&apos;t tolerate. SatGate uses L402 —
                Lightning Network payments that settle in milliseconds with sub-cent fees. For enterprises, Fiat402
                provides the same architecture over fiat budget rails. No blockchain required.
              </p>
            </div>
            <div className="flex items-start gap-3">
              <ArrowRight className="text-gray-500 shrink-0 mt-1" size={16} />
              <p className="text-gray-300 text-sm m-0">
                <strong className="text-white">Multi-tenant isolation is a deployment reality.</strong> The paper
                discusses market coordination between agents, but doesn&apos;t address the infrastructure question:
                how do you run this for multiple organizations simultaneously? SatGate&apos;s enterprise tier provides
                per-tenant isolation with independent budgets, delegation trees, and cost profiles — all on shared infrastructure.
              </p>
            </div>
          </div>

          {/* CTA */}
          <h2 className="text-2xl font-bold mt-12 mb-4 text-white">Try It</h2>
          <p className="text-gray-300 leading-relaxed">
            SatGate is open source (Apache 2.0). The gateway runs as a sidecar or standalone proxy with sub-ms overhead.
            If the DeepMind delegation framework describes the architecture you want, SatGate is the implementation.
          </p>

          <div className="flex flex-wrap gap-3 mt-6 mb-8">
            <Link href="https://github.com/SatGate-io/satgate" target="_blank" className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-white text-black font-semibold text-sm hover:bg-gray-200 transition no-underline">
              View on GitHub <ArrowRight size={16} />
            </Link>
            <Link href="https://cloud.satgate.io/cloud/signup" className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-purple-600 text-white font-semibold text-sm hover:bg-purple-500 transition no-underline">
              Start Free on Cloud <ArrowRight size={16} />
            </Link>
            <Link href="/agent-authority-layer" className="inline-flex items-center gap-2 px-4 py-2 rounded-lg border border-purple-700/50 bg-purple-900/20 text-purple-300 font-semibold text-sm hover:bg-purple-900/40 transition no-underline">
              Agent authority layer <ArrowRight size={16} />
            </Link>
          </div>

          <section className="not-prose mt-16 rounded-2xl border border-gray-800 bg-gray-950 p-8">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-blue-300">FAQ</p>
            <h2 className="mb-6 text-2xl font-bold text-white">AI agent delegation questions</h2>
            <div className="space-y-5">
              {[
                ['What are Delegation Capability Tokens for AI agents?', 'Delegation Capability Tokens are scoped credentials that let an agent pass limited authority to another agent. The useful form is attenuated: each delegation can only narrow permissions, budgets, routes, tools, or time windows.'],
                ['Why are macaroons a strong primitive for AI agent delegation?', 'Macaroons support cryptographic caveats, local verification, and privilege attenuation. A parent token can mint child tokens with stricter constraints, but a child token cannot expand authority beyond its parent.'],
                ['How does SatGate implement intelligent agent delegation?', 'SatGate uses macaroon-based capability tokens with caveats for route scope, MCP tool scope, budgets, expiry, and delegation chains, then enforces those constraints in the request path before upstream APIs execute.'],
                ['What does Intelligent AI Delegation require beyond task routing?', 'Intelligent AI delegation needs explicit authority transfer, attenuated permissions, resource budgets, accountability across delegation chains, revocation, and enforcement before delegated agents can call APIs, MCP tools, or paid services.'],
              ].map(([question, answer]) => (
                <div key={question} className="border-t border-gray-800 pt-5 first:border-t-0 first:pt-0">
                  <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                  <p className="leading-relaxed text-gray-400">{answer}</p>
                </div>
              ))}
            </div>
          </section>

          {/* Reference */}
          <div className="border-t border-gray-800 pt-6 mt-12">
            <p className="text-gray-500 text-sm">
              <strong className="text-gray-400">Paper reference:</strong> Tomasev, N. et al. (2026).{' '}
              <a href="https://arxiv.org/abs/2602.11865" target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:text-blue-300">
                &ldquo;Intelligent AI Delegation&rdquo;
              </a>. arXiv:2602.11865 [cs.AI]. Google DeepMind.
            </p>
            <p className="text-gray-500 text-sm">
              <strong className="text-gray-400">Macaroons paper:</strong> Birgisson, A. et al. (2014).{' '}
              <a href="https://research.google/pubs/pub41892/" target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:text-blue-300">
                &ldquo;Macaroons: Cookies with Contextual Caveats for Decentralized Authorization in the Cloud&rdquo;
              </a>. NDSS 2014.
            </p>
          </div>

        </article>
      </div>
    </div>
  );
}

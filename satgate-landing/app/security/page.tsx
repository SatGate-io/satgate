import Link from 'next/link';
import { ArrowLeft, Shield, Lock, Eye, Key, Zap, CheckCircle, Code, GitBranch, Bot, Activity, Server, FileText } from 'lucide-react';

export const metadata = {
  title: 'SatGate Security | Capability Tokens for AI Agent Governance',
  description: 'SatGate security model for AI agent API governance: capability tokens, scoped budgets, delegation limits, revocation, audit, and request-path policy.',
  alternates: { canonical: 'https://satgate.io/security' },
  keywords: [
    'SatGate security',
    'AI agent security',
    'agent capability tokens',
    'macaroons for AI agents',
    'revocable agent credentials',
    'AI agent API governance',
    'economic access control',
    'request-path policy enforcement',
  ],
  openGraph: {
    title: 'SatGate Security | Capability Tokens for AI Agent Governance',
    description: 'Capability tokens, macaroons, scoped budgets, delegation limits, revocation, audit, and request-path enforcement for AI agent APIs.',
    url: 'https://satgate.io/security',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate Security | Capability Tokens for AI Agent Governance',
    description: 'Security model for scoped, budget-aware, revocable AI agent API access.',
  },
};

export default function SecurityPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'SatGate Security | Capability Tokens for AI Agent Governance',
    url: 'https://satgate.io/security',
    description: metadata.description,
    datePublished: '2026-04-12',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI agent security' },
      { '@type': 'Thing', name: 'capability tokens for AI agents' },
      { '@type': 'Thing', name: 'macaroons for AI agents' },
      { '@type': 'Thing', name: 'revocable agent credentials' },
      { '@type': 'Thing', name: 'request-path policy enforcement' },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Security', item: 'https://satgate.io/security' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'How does SatGate secure AI agent API access?',
        acceptedAnswer: { '@type': 'Answer', text: 'SatGate secures AI agent API access with request-path policy enforcement, capability tokens, macaroon caveats, scoped budgets, expiry, delegation limits, revocation, and audit evidence at the gateway before forwarding.' },
      },
      {
        '@type': 'Question',
        name: 'Why use capability tokens instead of static API keys for agents?',
        acceptedAnswer: { '@type': 'Answer', text: 'Capability tokens can constrain route, budget, calls, expiry, delegation, and revocation for one agent task. Static API keys are usually broad, long-lived, copyable, and disconnected from economic policy.' },
      },
      {
        '@type': 'Question',
        name: 'What role do macaroons play in SatGate?',
        acceptedAnswer: { '@type': 'Answer', text: 'Macaroons let SatGate attach cryptographic caveats to agent authority so delegated credentials can only become narrower, shorter-lived, or lower-budget as they move through agent workflows.' },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Home
        </Link>

        <h1 className="text-4xl font-bold mb-2">Security</h1>
        <p className="text-gray-500 mb-12">Economic Access Control — built by security practitioners, not bolted on after.</p>

        <div className="prose prose-invert prose-gray max-w-none space-y-8">

          {/* Security Model */}
          <section className="bg-gradient-to-r from-purple-900/20 to-cyan-900/20 border border-purple-800/30 rounded-xl p-6">
            <h2 className="text-xl font-bold text-white mb-4">The Security Model</h2>
            <p className="text-gray-400 leading-relaxed mb-4">
              SatGate implements <strong className="text-white">Economic Access Control</strong> — a capability-based security model
              where every API request must present a cryptographically verified token with embedded access constraints.
            </p>
            <p className="text-gray-400 leading-relaxed mb-4">
              Traditional APIs ask <em>&ldquo;who are you?&rdquo;</em> and often tie enforcement to broad user credentials.
              SatGate asks <em>&ldquo;what is this workload allowed to do and spend?&rdquo;</em> — enforcing budgets, scopes, and delegation limits
              at the request layer. The enforcement path does not require an upstream end-user identity database; SatGate Cloud may still store account emails for login, billing, and tenant administration.
            </p>
            <p className="text-gray-400 leading-relaxed mb-0">
              The gateway operates as a <strong className="text-white">Policy Enforcement Point (PEP)</strong>.
              No request reaches your upstream without passing cryptographic verification.
            </p>
          </section>

          {/* Mint — Identity Exchange */}
          <section>
            <div className="flex items-center gap-3 mb-4">
              <Key className="text-purple-400" size={22} />
              <h2 className="text-xl font-bold text-white mb-0">SatGate Mint — Trust Broker</h2>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              Agents authenticate through your existing identity infrastructure. SatGate Mint exchanges workload identity tokens
              (OIDC JWTs from Kubernetes, AWS IAM, or any OIDC provider) for capability-bearing macaroons with policy-defined constraints.
            </p>
            <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-5 mb-4">
              <div className="flex items-center justify-center gap-3 text-sm flex-wrap">
                <div className="px-3 py-2 rounded bg-gray-800 text-gray-400 text-xs text-center">
                  <Bot size={16} className="mx-auto mb-1 text-blue-400" />
                  Agent Identity<br/><span className="text-gray-600">K8s / AWS / OIDC</span>
                </div>
                <span className="text-gray-600">→</span>
                <div className="px-3 py-2 rounded bg-purple-900/40 border border-purple-500/50 text-purple-300 text-xs text-center font-bold">
                  <Key size={16} className="mx-auto mb-1" />
                  SatGate Mint<br/><span className="text-gray-500 font-normal">Verify + Policy Match</span>
                </div>
                <span className="text-gray-600">→</span>
                <div className="px-3 py-2 rounded bg-cyan-900/30 border border-cyan-500/50 text-cyan-400 text-xs text-center">
                  <Lock size={16} className="mx-auto mb-1" />
                  Macaroon<br/><span className="text-gray-500">Budget + Scope + TTL</span>
                </div>
              </div>
            </div>
            <ul className="text-gray-400 text-sm space-y-2 list-none pl-0">
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">Zero secrets to manage.</strong> Agents present their existing workload identity — no API keys to rotate, no credentials to store.</span></li>
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">Policy-driven issuance.</strong> Each agent policy defines budget, scope, TTL, and delegation permissions. Different agents get different constraints.</span></li>
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">Multi-provider support.</strong> Same issuer with different audiences routes to different policies — enforce different budgets per agent class.</span></li>
            </ul>
          </section>

          {/* Macaroons */}
          <section>
            <div className="flex items-center gap-3 mb-4">
              <Lock className="text-cyan-400" size={22} />
              <h2 className="text-xl font-bold text-white mb-0">Why Macaroons</h2>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              API keys are passwords. JWTs are ID cards. Macaroons are <strong className="text-white">signed checks with spending limits</strong> —
              and every person who touches the check can only reduce the limit, never increase it.
            </p>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
              <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-4">
                <h4 className="text-white text-sm font-bold mb-2">Delegation Without Calling Home</h4>
                <p className="text-gray-500 text-xs">
                  A manager mints a token, carves off a weaker one for their team, who delegates further to an agent.
                  Each step is cryptographically chained — no API call, no DB write. Permissions only shrink.
                </p>
              </div>
              <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-4">
                <h4 className="text-white text-sm font-bold mb-2">Runtime Constraints Built In</h4>
                <p className="text-gray-500 text-xs">
                  Caveats encode &ldquo;only until 5pm,&rdquo; &ldquo;only for /openai/*,&rdquo; or &ldquo;max 200 credits.&rdquo;
                  Verified at the gateway on every request. The token carries its own budget.
                </p>
              </div>
              <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-4">
                <h4 className="text-white text-sm font-bold mb-2">The Right Primitive for Agents</h4>
                <p className="text-gray-500 text-xs">
                  Agents can&apos;t do OAuth flows or sign up for accounts. But they can hold a bearer token
                  that encodes exactly what they&apos;re allowed to do and spend. That&apos;s a macaroon.
                </p>
              </div>
            </div>

            <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-4">
              <h4 className="text-white text-sm font-bold mb-2">Cascade Revocation</h4>
              <p className="text-gray-500 text-xs mb-0">
                Revoking a parent token invalidates governed child requests at the next policy check.
                The gateway checks revocation status before each governed request.
              </p>
            </div>
          </section>

          {/* Enforcement Modes */}
          <section>
            <div className="flex items-center gap-3 mb-4">
              <Shield className="text-purple-400" size={22} />
              <h2 className="text-xl font-bold text-white mb-0">Enforcement Modes</h2>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              Three modes, one adoption path. Start safe, add enforcement when ready.
            </p>
            <div className="space-y-3">
              <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-4 flex items-start gap-4">
                <Eye size={20} className="text-green-400 mt-1 flex-shrink-0" />
                <div>
                  <h4 className="text-white text-sm font-bold">Observe</h4>
                  <p className="text-gray-500 text-xs mb-0">Verify → Allow → Log. Full visibility into agent traffic, cost attribution by team, zero enforcement. Free forever.</p>
                </div>
              </div>
              <div className="bg-gray-900/50 border border-purple-800/30 rounded-lg p-4 flex items-start gap-4">
                <Shield size={20} className="text-purple-400 mt-1 flex-shrink-0" />
                <div>
                  <h4 className="text-white text-sm font-bold">Control</h4>
                  <p className="text-gray-500 text-xs mb-0">Verify → Enforce Budget → Allow. Per-agent spending caps, real-time budget enforcement. Works with Stripe, ERP — no crypto required. HTTP 402 when budget exhausted.</p>
                </div>
              </div>
              <div className="bg-gray-900/50 border border-yellow-800/30 rounded-lg p-4 flex items-start gap-4">
                <Zap size={20} className="text-yellow-400 mt-1 flex-shrink-0" />
                <div>
                  <h4 className="text-white text-sm font-bold">Charge</h4>
                  <p className="text-gray-500 text-xs mb-0">Verify → Payment Proof → Allow. paid-rail context micropayments. Per-request pricing. Instant settlement, no invoices, no chargebacks.</p>
                </div>
              </div>
            </div>
          </section>

          {/* Non-Custodial Architecture */}
          <section>
            <div className="flex items-center gap-3 mb-4">
              <Server className="text-blue-400" size={22} />
              <h2 className="text-xl font-bold text-white mb-0">Non-Custodial Architecture</h2>
            </div>
            <ul className="text-gray-400 text-sm space-y-2 list-none pl-0">
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">Never stores upstream credentials.</strong> SatGate verifies tokens at the gateway. Your API keys, secrets, and upstream credentials stay in your infrastructure.</span></li>
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">No request body inspection.</strong> Gateway inspects only token and routing metadata. Request payloads pass through opaque (MCP proxy reads method/tool name for cost attribution only).</span></li>
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">Tenant-isolated data.</strong> All data scoped by tenant_id. Cross-tenant access is architecturally impossible.</span></li>
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">paid-rail context settle directly.</strong> In Charge mode, payments go from payer to your Lightning node. SatGate never touches funds.</span></li>
            </ul>
          </section>

          {/* Zero Trust */}
          <section>
            <div className="flex items-center gap-3 mb-4">
              <Activity className="text-cyan-400" size={22} />
              <h2 className="text-xl font-bold text-white mb-0">Zero Trust Access Control</h2>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              SatGate is a Zero Trust Policy Enforcement Point for API access. It verifies every request and
              enforces scoped access — no network trust assumptions.
            </p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
              {[
                { label: 'Policy Enforcement Point', desc: 'Deny-by-default. Every protected request verified before proxying.' },
                { label: 'Continuous Authorization', desc: 'Token validated on each request — no long-lived sessions or network zone trust.' },
                { label: 'Least Privilege', desc: 'Macaroon caveats restrict scope, method, TTL, and budget per token.' },
                { label: 'No Trust Dependencies', desc: 'Cryptographic verification — no centralized user database required.' },
              ].map((item, i) => (
                <div key={i} className="bg-gray-900/50 border border-gray-800 rounded-lg p-3">
                  <h4 className="text-white text-xs font-bold mb-1">{item.label}</h4>
                  <p className="text-gray-500 text-xs mb-0">{item.desc}</p>
                </div>
              ))}
            </div>
            <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-4">
              <h4 className="text-white text-sm font-bold mb-2">What SatGate is (and isn&apos;t)</h4>
              <ul className="text-gray-400 text-xs space-y-1 list-none pl-0 mb-0">
                <li><span className="text-green-400">✓</span> Zero Trust PEP for API access (request-level verification + least privilege)</li>
                <li><span className="text-green-400">✓</span> Economic abuse friction — high-volume abuse becomes self-limiting</li>
                <li><span className="text-green-400">✓</span> Complements existing security stack (WAF/CDN, rate limiting, SIEM)</li>
                <li><span className="text-yellow-400">⚠</span> Not a full Zero Trust program (identity governance, device posture, microsegmentation)</li>
              </ul>
            </div>
          </section>

          {/* Audit & Compliance */}
          <section>
            <div className="flex items-center gap-3 mb-4">
              <FileText className="text-orange-400" size={22} />
              <h2 className="text-xl font-bold text-white mb-0">Audit & Compliance</h2>
            </div>
            <ul className="text-gray-400 text-sm space-y-2 list-none pl-0">
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">55 event types</strong> across 12 categories in immutable audit log. Every mint, revocation, delegation, budget change, and config update recorded.</span></li>
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">Real-time alerts</strong> via Slack, Discord, email, or custom webhooks for critical events.</span></li>
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">Shadow Report</strong> — real-time dashboard showing all agent traffic, spend, and policy violations.</span></li>
              <li className="flex items-start gap-2"><CheckCircle size={14} className="text-green-500 mt-1 flex-shrink-0" /> <span><strong className="text-white">Export</strong> — CSV/JSON export for audit log and billing data via dashboard and API.</span></li>
            </ul>
          </section>

          {/* Security Practices */}
          <section>
            <div className="flex items-center gap-3 mb-4">
              <Code className="text-green-400" size={22} />
              <h2 className="text-xl font-bold text-white mb-0">Security Practices</h2>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {[
                { title: 'Open Source', items: ['Core gateway is Apache 2.0 licensed', 'Full source audit at any time', 'No black boxes'] },
                { title: 'Dependencies', items: ['Automated Dependabot updates', 'Go vulnerability scanning', 'Checksum verification'] },
                { title: 'Infrastructure', items: ['TLS 1.2+ enforced', 'Stateless gateway design', 'Minimal attack surface'] },
                { title: 'Code Quality', items: ['Required code review', 'Automated tests on every PR', 'Static analysis enforced'] },
              ].map((group, i) => (
                <div key={i} className="bg-gray-900/50 border border-gray-800 rounded-lg p-4">
                  <h4 className="text-white text-sm font-bold mb-2">{group.title}</h4>
                  <ul className="text-gray-500 text-xs space-y-1 list-none pl-0 mb-0">
                    {group.items.map((item, j) => (
                      <li key={j} className="flex items-center gap-1.5"><CheckCircle size={10} className="text-green-500 flex-shrink-0" /> {item}</li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          </section>

          {/* FAQ */}
          <section>
            <h2 className="text-xl font-bold text-white mb-4">SatGate security FAQ</h2>
            <div className="space-y-3">
              <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-4">
                <h3 className="text-white text-sm font-bold mb-2">How does SatGate secure AI agent API access?</h3>
                <p className="text-gray-500 text-xs mb-0">
                  SatGate secures AI agent API access with request-path policy enforcement, capability tokens, macaroon caveats, scoped budgets, expiry, delegation limits, revocation, and audit evidence at the gateway before forwarding.
                </p>
              </div>
              <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-4">
                <h3 className="text-white text-sm font-bold mb-2">Why use capability tokens instead of static API keys for agents?</h3>
                <p className="text-gray-500 text-xs mb-0">
                  Capability tokens can constrain route, budget, calls, expiry, delegation, and revocation for one agent task. Static API keys are usually broad, long-lived, copyable, and disconnected from economic policy.
                </p>
              </div>
              <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-4">
                <h3 className="text-white text-sm font-bold mb-2">What role do macaroons play in SatGate?</h3>
                <p className="text-gray-500 text-xs mb-0">
                  Macaroons let SatGate attach cryptographic caveats to agent authority so delegated credentials can only become narrower, shorter-lived, or lower-budget as they move through agent workflows.
                </p>
              </div>
            </div>
          </section>

          {/* Security Posture Pack CTA */}
          <section className="bg-purple-900/20 border border-purple-800/30 rounded-xl p-6 text-center">
            <h2 className="text-xl font-bold text-white mb-3">Enterprise Security Review</h2>
            <p className="text-gray-400 text-sm mb-4">
              Need to evaluate SatGate for your security team? Download the full Security Posture Pack —
              covers architecture, encryption, data handling, audit, and compliance.
            </p>
            <a
              href="https://cloud.satgate.io/docs/security-pack"
              className="inline-flex items-center gap-2 px-6 py-3 bg-purple-600 hover:bg-purple-500 rounded-lg text-sm font-bold transition"
            >
              <FileText size={16} /> View Security Posture Pack →
            </a>
          </section>

          {/* Leadership */}
          <section>
            <h2 className="text-xl font-bold text-white mb-3">Leadership</h2>
            <p className="text-gray-400 leading-relaxed">
              SatGate was founded by{' '}
              <a href="https://linkedin.com/in/waynemattadeen" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">Wayne Mattadeen</a>,
              a cybersecurity executive with 27+ years of experience across Deloitte, Accenture, and EY — including
              partner-level roles focused on enterprise security architecture and Zero Trust.
            </p>
            <p className="text-gray-400 leading-relaxed">
              Security isn&apos;t something we&apos;re learning. It&apos;s where we come from.
            </p>
          </section>

          {/* Compliance Roadmap */}
          <section>
            <h2 className="text-xl font-bold text-white mb-3">Compliance Roadmap</h2>
            <div className="bg-gray-900/50 border border-gray-800 rounded-lg overflow-hidden">
              {[
                { item: 'Responsible disclosure program', status: 'Active', color: 'text-green-400' },
                { item: 'Automated vulnerability scanning', status: 'Active', color: 'text-green-400' },
                { item: 'Open source core (Apache 2.0)', status: 'Active', color: 'text-green-400' },
                { item: 'Third-party penetration test', status: 'Q2 2026', color: 'text-yellow-400' },
                { item: 'SOC 2 Type I', status: 'Post-revenue', color: 'text-gray-500' },
              ].map((item, i) => (
                <div key={i} className={`flex items-center justify-between px-4 py-2.5 text-sm ${i > 0 ? 'border-t border-gray-800' : ''}`}>
                  <span className="text-gray-400">{item.item}</span>
                  <span className={`text-xs font-bold ${item.color}`}>{item.status}</span>
                </div>
              ))}
            </div>
          </section>

          {/* Responsible Disclosure */}
          <section>
            <h2 className="text-xl font-bold text-white mb-3">Responsible Disclosure</h2>
            <p className="text-gray-400 text-sm leading-relaxed mb-3">
              Found a security vulnerability? Please report issues privately before public disclosure.
            </p>
            <p className="text-gray-400 text-sm mb-3">
              Email: <a href="mailto:security@satgate.io" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">security@satgate.io</a>
            </p>
            <ul className="text-gray-500 text-sm space-y-1 list-none pl-0">
              <li>• Acknowledge receipt within 48 hours</li>
              <li>• Initial assessment within 7 days</li>
              <li>• Coordinated disclosure</li>
              <li>• Public credit (unless you prefer anonymity)</li>
            </ul>
          </section>

          {/* Open Source */}
          <section className="text-center pb-8">
            <div className="flex items-center justify-center gap-2 mb-3">
              <GitBranch className="text-gray-400" size={18} />
              <h2 className="text-xl font-bold text-white mb-0">Open Source Transparency</h2>
            </div>
            <p className="text-gray-400 text-sm mb-4">
              All gateway code is open source under the Apache 2.0 License. Audit every line, fork it, run your own infrastructure.
            </p>
            <a
              href="https://github.com/SatGate-io/satgate"
              className="inline-flex items-center gap-2 px-6 py-3 bg-gray-800 hover:bg-gray-700 border border-gray-700 rounded-lg text-sm font-medium text-gray-300 transition"
            >
              <Code size={16} /> View Source on GitHub →
            </a>
          </section>

        </div>
      </div>
    </div>
  );
}

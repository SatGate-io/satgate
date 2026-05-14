'use client';

import React, { useState } from 'react';
import { Check, ChevronDown, Menu, X } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

const faqs = [
  {
    q: 'What counts as a request?',
    a: 'Every API call proxied through the gateway in Control or Charge mode counts as a metered request. Observe-mode requests are always free and unlimited — no catches.',
  },
  {
    q: 'How does the Free → Pro upgrade work?',
    a: 'Start with Free to see what your agents are spending. When you\'re ready to enforce budgets and set hard stops, upgrade to Pro in one click. No data migration — your dashboards carry over.',
  },
  {
    q: 'Can I switch plans?',
    a: 'Yes, upgrade or downgrade anytime. Changes take effect immediately. No lock-in.',
  },
  {
    q: 'Do you offer annual billing?',
    a: 'Yes, save 20% with annual billing. Contact us for details.',
  },
  {
    q: 'What happens if I exceed my request limit?',
    a: 'Overage is billed at $0.10 per 1,000 requests. No surprise charges — you\'ll get alerts at 80% and 90% so you can adjust budgets before you hit the limit.',
  },
  {
    q: 'Is there a free trial of Pro?',
    a: 'Every account starts with a 14-day Pro trial. No credit card required. After the trial, you drop to Free (Observe) — you never lose visibility.',
  },
  {
    q: 'Do you support hybrid/on-prem deployment?',
    a: 'Yes, on the Enterprise plan. The gateway runs in your infrastructure with our managed control plane. Or fully air-gapped — your choice.',
  },
];

const PricingPage = () => {
  const [openFaq, setOpenFaq] = useState<number | null>(null);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'SatGate Pricing',
    url: 'https://satgate.io/pricing',
    description: 'Pricing for bounded agent authority: Observe audits, request-path budget enforcement, MCP tool controls, receipts, and Evidence Pack proof.',
    datePublished: '2026-04-27',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'economic firewall for AI agents' },
      { '@type': 'Thing', name: 'AI agent budget enforcement' },
      { '@type': 'Thing', name: 'SatGate Economic Firewall' },
      { '@type': 'Thing', name: 'request-path spend governance' },
      { '@type': 'Thing', name: 'rail-neutral paid-rail governance' },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Pricing', item: 'https://satgate.io/pricing' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: faqs.map((faq) => ({
      '@type': 'Question',
      name: faq.q,
      acceptedAnswer: { '@type': 'Answer', text: faq.a },
    })),
  };

  const offerCatalogJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'OfferCatalog',
    name: 'SatGate Pricing',
    url: 'https://satgate.io/pricing',
    description: 'Pricing for bounded agent authority: Observe audits, request-path budget enforcement, MCP tool controls, receipts, and Evidence Pack proof.',
    dateModified: '2026-05-03',
    itemListElement: [
      {
        '@type': 'Offer',
        name: 'Builder / Observe',
        price: '0',
        priceCurrency: 'USD',
        description: 'Free Observe-mode receipt capture for AI agent API traffic, cost attribution, and Evidence Pack-ready decision proof.',
        availability: 'https://schema.org/InStock',
        itemOffered: { '@type': 'SoftwareApplication', name: 'SatGate Observe', applicationCategory: 'DeveloperApplication' },
      },
      {
        '@type': 'Offer',
        name: 'Pro / Control',
        priceCurrency: 'USD',
        description: 'Request-path budget enforcement, per-agent caps, MCP tool controls, alerts, revocation, and signed decision receipts for AI agent spend.',
        availability: 'https://schema.org/InStock',
        itemOffered: { '@type': 'SoftwareApplication', name: 'SatGate Control', applicationCategory: 'DeveloperApplication' },
      },
      {
        '@type': 'Offer',
        name: 'Enterprise / Charge',
        priceCurrency: 'USD',
        description: 'Enterprise deployment, hybrid/on-prem options, advanced governance, and paid-rail governance where paid calls return receipts and feed Evidence Packs.',
        availability: 'https://schema.org/InStock',
        itemOffered: { '@type': 'SoftwareApplication', name: 'SatGate paid-rail governance', applicationCategory: 'DeveloperApplication' },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans selection:bg-purple-500 selection:text-white">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(offerCatalogJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      {/* Navigation */}
      <nav className="border-b border-gray-800 backdrop-blur-md fixed w-full z-50 bg-black/50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2 shrink-0">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="w-7 h-7 sm:w-8 sm:h-8" />
            <span className="text-lg sm:text-xl font-bold text-white whitespace-nowrap">SatGate<sup className="text-xs font-normal">™</sup></span>
          </Link>

          {/* Desktop menu */}
          <div className="hidden md:flex gap-6 text-sm font-medium text-gray-400">
            <Link href="/govern" className="hover:text-white transition">Enterprise</Link>
            <Link href="/pricing" className="text-white transition">Pricing</Link>
            <Link href="/roi-calculator" className="hover:text-white transition">ROI Calculator</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud</a>
          </div>

          {/* Mobile menu button */}
          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="flex md:hidden items-center justify-center w-10 h-10 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 text-gray-400 hover:text-white transition"
            aria-label="Toggle menu"
            aria-expanded={mobileMenuOpen}
          >
            {mobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
        </div>

        {/* Mobile menu dropdown */}
        <div
          className={`md:hidden overflow-hidden transition-all duration-300 ease-in-out ${
            mobileMenuOpen ? 'max-h-80 opacity-100' : 'max-h-0 opacity-0'
          }`}
        >
          <div className="bg-black/95 backdrop-blur-xl border-t border-gray-800 px-4 py-4 space-y-1">
            <Link href="/govern" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Enterprise</Link>
            <Link href="/pricing" onClick={() => setMobileMenuOpen(false)} className="block text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Pricing</Link>
            <Link href="/roi-calculator" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">ROI Calculator</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Docs</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Cloud</a>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <header className="pt-32 pb-10 px-6 text-center">
        <div className="max-w-4xl mx-auto">
          <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
            Give Agents{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
              Bounded Authority.
            </span>
          </h1>
          <p className="text-xl text-gray-400 max-w-2xl mx-auto leading-relaxed">
            SatGate prices the controls humans and platforms need before agents reach protected APIs: visibility, hard budget stops, receipts, and Evidence Pack proof.
          </p>
        </div>
      </header>

      {/* Observe → Control → Charge Journey */}
      <section className="pb-10 px-6">
        <div className="max-w-3xl mx-auto">
          <div className="flex flex-col md:flex-row items-center justify-center gap-3 md:gap-4">
            <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-cyan-900/20 border border-cyan-800/30">
              <span className="text-cyan-400 font-bold text-sm">👁 Observe</span>
              <span className="text-gray-500 text-xs">See the spend</span>
            </div>
            <span className="text-gray-600 text-xl hidden md:block">→</span>
            <span className="text-gray-600 md:hidden">↓</span>
            <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-purple-900/20 border border-purple-800/30">
              <span className="text-purple-400 font-bold text-sm">🛡 Control</span>
              <span className="text-gray-500 text-xs">Stop the bleed</span>
            </div>
            <span className="text-gray-600 text-xl hidden md:block">→</span>
            <span className="text-gray-600 md:hidden">↓</span>
            <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-yellow-900/20 border border-yellow-800/30">
              <span className="text-yellow-400 font-bold text-sm">⚡ Prove</span>
              <span className="text-gray-500 text-xs">Export receipts</span>
            </div>
          </div>
        </div>
      </section>

      {/* Pricing Cards */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-6 items-stretch">
          {/* Free (Observe) */}
          <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-600 transition flex flex-col">
            <div className="mb-6">
              <h3 className="text-lg font-bold text-cyan-400 mb-1">Builder</h3>
              <p className="text-gray-500 text-sm">Get out of the dark. See every cent in real-time.</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">$0</span>
              <span className="text-gray-500 text-sm">/month forever</span>
            </div>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" /><span><b className="text-gray-200">Unlimited</b> observe-mode requests</span></li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Real-time receipt dashboard</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Cost attribution by agent &amp; team</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Up to 3 routes</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Community support</li>
            </ul>
            <a
              href="https://cloud.satgate.io/cloud/login"
              target="_blank"
              rel="noopener noreferrer"
              className="block text-center py-3 rounded-lg border border-gray-700 font-bold hover:border-gray-500 hover:bg-gray-800 transition"
            >
              Start Observing — Free →
            </a>
          </div>

          {/* Pro (Control + Charge) — highlighted */}
          <div className="p-6 rounded-xl bg-gray-900 border-2 border-purple-500/60 hover:border-purple-400 transition flex flex-col relative">
            <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-0.5 rounded-full bg-purple-600 text-white text-xs font-bold">
              MOST POPULAR
            </div>
            <div className="mb-6">
              <h3 className="text-lg font-bold text-purple-400 mb-1">Professional</h3>
              <p className="text-gray-500 text-sm">The only security tool that pays for itself.</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">$99</span>
              <span className="text-gray-500 text-sm">/month</span>
            </div>
            <p className="text-xs text-gray-500 mb-4 -mt-4">Everything in Free, plus:</p>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" /><span><b className="text-gray-200">Budget enforcement</b> — hard stops per agent, team, or API</span></li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Real-time alerts at 80%, 90%, and limit</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />1M control/charge requests included</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Then $0.10 per 1K overage</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Unlimited routes</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />CFO-ready receipt and chargeback exports</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Evidence Pack compliance exports</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Email support</li>
            </ul>
            <a
              href="https://cloud.satgate.io/cloud/login"
              target="_blank"
              rel="noopener noreferrer"
              className="block text-center py-3 rounded-lg bg-gradient-to-r from-purple-600 to-cyan-600 text-white font-bold hover:opacity-90 transition shadow-lg shadow-purple-500/20"
            >
              Start Pro — 14 Days Free →
            </a>
          </div>

          {/* Enterprise */}
          <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-600 transition flex flex-col">
            <div className="mb-6">
              <h3 className="text-lg font-bold text-green-400 mb-1">Enterprise</h3>
              <p className="text-gray-500 text-sm">Turn your infrastructure into a marketplace.</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">Custom</span>
            </div>
            <p className="text-xs text-gray-500 mb-4 -mt-4">Everything in Pro, plus:</p>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" /><span><b className="text-gray-200">Unlimited</b> requests — no metering caps</span></li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Hybrid / on-prem / air-gapped deployment</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />SSO/SCIM + RBAC</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Full Evidence Pack &amp; retention policies</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />SOC 2 compliance package</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Dedicated CSM + SLA</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Custom integrations &amp; onboarding</li>
            </ul>
            <a
              href="mailto:contact@satgate.io"
              className="block text-center py-3 rounded-lg border border-gray-700 font-bold hover:border-gray-500 hover:bg-gray-800 transition"
            >
              Talk to Us →
            </a>
          </div>
        </div>
      </section>

      {/* Why SatGate Wins */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">The Governance Gap</h2>
          <p className="text-gray-400 text-center mb-12 max-w-2xl mx-auto">Standard MCP is an open tap for your API credits. SatGate adds the meter — and the shutoff valve.</p>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800">
                  <th className="text-left py-3 px-4 text-gray-500 font-medium">Capability</th>
                  <th className="text-left py-3 px-4 text-gray-500 font-medium">Standard MCP</th>
                  <th className="text-left py-3 px-4 text-purple-400 font-medium">SatGate-Enabled</th>
                </tr>
              </thead>
              <tbody className="text-gray-400">
                <tr className="border-b border-gray-800/50">
                  <td className="py-3 px-4 text-gray-300">Budget Enforcement</td>
                  <td className="py-3 px-4">&ldquo;Faith-based&rdquo; — wait for the bill</td>
                  <td className="py-3 px-4 text-white">Real-time hard caps at protocol level</td>
                </tr>
                <tr className="border-b border-gray-800/50">
                  <td className="py-3 px-4 text-gray-300">Cost Attribution</td>
                  <td className="py-3 px-4">Aggregate — one big API bill</td>
                  <td className="py-3 px-4 text-white">Per-tool / per-agent granularity</td>
                </tr>
                <tr className="border-b border-gray-800/50">
                  <td className="py-3 px-4 text-gray-300">Access Control</td>
                  <td className="py-3 px-4">Static API keys — all or nothing</td>
                  <td className="py-3 px-4 text-white">Attenuated macaroons — time/tool limited</td>
                </tr>
                <tr className="border-b border-gray-800/50">
                  <td className="py-3 px-4 text-gray-300">Visibility</td>
                  <td className="py-3 px-4">Post-mortem — look back at logs</td>
                  <td className="py-3 px-4 text-white">Signed receipts + Evidence Pack export</td>
                </tr>
                <tr className="border-b border-gray-800/50">
                  <td className="py-3 px-4 text-gray-300">Agent Loops</td>
                  <td className="py-3 px-4">Potentially infinite spend</td>
                  <td className="py-3 px-4 text-white">Automated kill-switch at threshold</td>
                </tr>
                <tr>
                  <td className="py-3 px-4 text-gray-300">Authentication</td>
                  <td className="py-3 px-4">Basic / bearer tokens</td>
                  <td className="py-3 px-4 text-white">L402 — cryptographic proof-of-budget</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Value Props */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-4xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="text-3xl mb-3">🛡</div>
              <h3 className="font-bold text-white mb-2">Cost Avoidance</h3>
              <p className="text-gray-400 text-sm">Stop the $500 &ldquo;hallucination loop&rdquo; at $2. Automatically revoke access for idle or runaway agents.</p>
            </div>
            <div className="text-center">
              <div className="text-3xl mb-3">⚡</div>
              <h3 className="font-bold text-white mb-2">Operational Efficiency</h3>
              <p className="text-gray-400 text-sm">Engineers stop acting as manual billing auditors. One proxy to govern all MCP servers.</p>
            </div>
            <div className="text-center">
              <div className="text-3xl mb-3">💰</div>
              <h3 className="font-bold text-white mb-2">Revenue Enablement</h3>
              <p className="text-gray-400 text-sm">Charge other companies&apos; agents micropayments via L402. Trust-as-a-Service for enterprise deals.</p>
            </div>
          </div>
        </div>
      </section>

      {/* FAQ Section */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-3xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">Frequently Asked Questions</h2>
          <div className="space-y-2">
            {faqs.map((faq, i) => (
              <div key={i} className="border border-gray-800 rounded-xl overflow-hidden">
                <button
                  onClick={() => setOpenFaq(openFaq === i ? null : i)}
                  className="w-full flex items-center justify-between px-6 py-4 text-left hover:bg-gray-900/50 transition"
                >
                  <span className="font-medium text-white">{faq.q}</span>
                  <ChevronDown
                    size={18}
                    className={`text-gray-500 transition-transform duration-200 shrink-0 ml-4 ${
                      openFaq === i ? 'rotate-180' : ''
                    }`}
                  />
                </button>
                <div
                  className={`overflow-hidden transition-all duration-300 ease-in-out ${
                    openFaq === i ? 'max-h-40 opacity-100' : 'max-h-0 opacity-0'
                  }`}
                >
                  <p className="px-6 pb-4 text-gray-400 text-sm leading-relaxed">{faq.a}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Bottom CTA */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-2xl font-bold mb-3">If your agents aren&apos;t governed, they&apos;re a liability.</h2>
          <p className="text-lg text-gray-400 mb-8 max-w-xl mx-auto">
            With SatGate, they&apos;re an asset. <Link href="/roi-calculator" className="text-purple-400 hover:text-purple-300 underline underline-offset-2">Calculate your savings →</Link>
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <a
              href="https://cloud.satgate.io/cloud/login"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-block bg-white text-black px-10 py-4 rounded-full font-bold text-lg hover:bg-gray-200 transition"
            >
              Start Free →
            </a>
            <a
              href="mailto:contact@satgate.io"
              className="inline-block border border-gray-700 text-gray-300 px-10 py-4 rounded-full font-bold text-lg hover:border-gray-500 hover:bg-gray-800 transition"
            >
              Talk to Sales
            </a>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 border-t border-gray-800">
        <div className="max-w-6xl mx-auto px-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8 py-8">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <Image src="/logo_white_transparent.png" alt="SatGate" width={24} height={24} className="w-6 h-6" />
                <h4 className="font-bold text-white">SatGate</h4>
              </div>
              <p className="text-gray-500 text-sm">The Economic Firewall for AI agent requests.</p>
              <p className="text-gray-600 text-xs mt-3">Non-custodial. We never hold your keys.</p>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Resources</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a></li>
                <li><a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Documentation</a></li>
                <li><Link href="/govern" className="hover:text-white transition">Enterprise</Link></li>
                <li><Link href="/design-partners" className="hover:text-white transition">Design Partners</Link></li>
                <li><Link href="/pricing" className="hover:text-white transition">Pricing</Link></li>
                <li><a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud Dashboard</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Legal</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/terms" className="hover:text-white transition">Terms of Service</Link></li>
                <li><Link href="/privacy" className="hover:text-white transition">Privacy Policy</Link></li>
                <li><Link href="/security" className="hover:text-white transition">Security</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Contact</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="mailto:contact@satgate.io" className="hover:text-white transition">contact@satgate.io</a></li>
              </ul>
            </div>
          </div>
          <div className="pt-8 border-t border-gray-800 text-center text-gray-600 text-sm">
            © 2026 SatGate Inc. All rights reserved. SatGate™ is a trademark of SatGate Inc. Patent Pending.
          </div>
        </div>
      </footer>
    </div>
  );
};

export default PricingPage;

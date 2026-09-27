'use client';

import React, { useState } from 'react';
import { Check, ChevronDown, Menu, X } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

const faqs = [
  {
    q: 'How do I start?',
    a: 'Go to https://cloud.satgate.io/cloud/signup. The trial is 14 days of Pro features, with no credit card. Email verification creates the account.',
  },
  {
    q: 'What is included in Starter?',
    a: 'Starter is free. It includes Observe (metering) with a fair-use cap, 3 routes, default protection + Observe, and community support.',
  },
  {
    q: 'What is included in Pro?',
    a: 'Pro is $99/month. It includes Observe with no finite monthly cap, 1M Control/Admit requests per month, 25 routes, unlimited team members, and all policies (Observe, Control, and Admit — Admit is the Charge policy in the dashboard). It also includes budget enforcement, token governance, and priority support. Upgrade is self-serve by card in the dashboard (Stripe).',
  },
  {
    q: 'Is hosted access available without a sales call?',
    a: 'Yes. Sign up for a 14-day free trial of Pro features. No credit card. Email verification creates the account. Upgrade to Pro by card in the dashboard.',
  },
  {
    q: 'What does Enterprise include?',
    a: 'Enterprise is custom pricing. It is unlimited. Self-host means installing the same SatGate binary that runs SatGate Cloud, in your own environment. It includes SCIM, GitOps signed configs, WORM audit export, and dedicated support with an SLA. Contact contact@satgate.io.',
  },
  {
    q: 'Does a pilot require code changes?',
    a: 'Most pilots start with a DNS, proxy, or MCP configuration change around one bounded endpoint or tool. We verify that path before expanding it.',
  },
  {
    q: 'What does the design-partner pilot cost?',
    a: 'The initial 90-day design-partner pilot has no charge or credit-card requirement. Any later commercial contract depends on the accepted scope, measured usage, and operating requirements. The pilot is an option for larger teams that want help. It is not required to start.',
  },
  {
    q: 'What happens after the pilot?',
    a: 'We review the governed workload, policy outcomes, Evidence Pack use, support burden, and operating evidence together. Both sides then decide whether to expand, contract, or stop.',
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
    description: 'SatGate plans: free Starter, Pro at $99/month, and custom Enterprise self-host. 14-day free trial of Pro features, no credit card.',
    datePublished: '2026-04-27',
    dateModified: '2026-09-27',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'Agent Authority & Accountability Layer' },
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
    description: 'SatGate plans: free Starter, Pro at $99/month, and custom Enterprise self-host. 14-day free trial of Pro features, no credit card.',
    dateModified: '2026-09-27',
    itemListElement: [
      {
        '@type': 'Offer',
        name: 'SatGate Starter',
        price: '0',
        priceCurrency: 'USD',
        description: 'Free. Observe (metering) with a fair-use cap, 3 routes, default protection + Observe, community support.',
        availability: 'https://schema.org/InStock',
        url: 'https://cloud.satgate.io/cloud/signup',
        itemOffered: { '@type': 'Service', name: 'SatGate Starter', serviceType: 'AI agent governance' },
      },
      {
        '@type': 'Offer',
        name: 'SatGate Pro',
        price: '99',
        priceCurrency: 'USD',
        description: '14-day free trial of Pro features, no credit card. $99/month. Observe with no finite monthly cap, 1M Control/Admit requests per month, 25 routes, unlimited team members, all policies, budget enforcement, token governance, priority support. Upgrade by card in the dashboard.',
        availability: 'https://schema.org/InStock',
        url: 'https://cloud.satgate.io/cloud/signup',
        itemOffered: { '@type': 'Service', name: 'SatGate Pro', serviceType: 'AI agent governance' },
      },
      {
        '@type': 'Offer',
        name: 'SatGate Enterprise',
        description: 'Custom pricing. Unlimited. Self-host the same SatGate binary that runs SatGate Cloud. SCIM, GitOps signed configs, WORM audit export, dedicated support and SLA.',
        availability: 'https://schema.org/InStock',
        url: 'mailto:contact@satgate.io',
        itemOffered: { '@type': 'Service', name: 'SatGate Enterprise', serviceType: 'Self-hosted AI agent governance' },
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
            <a href="https://cloud.satgate.io/cloud/signup" target="_blank" rel="noopener noreferrer" className="rounded-full bg-white px-3 py-1.5 font-bold text-black hover:bg-gray-200 transition">Start free trial</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud login</a>
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
            <a href="https://cloud.satgate.io/cloud/signup" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block bg-white text-black font-bold transition py-3 px-4 rounded-lg">Start free trial</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Cloud login</a>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <header className="pt-32 pb-10 px-6 text-center">
        <div className="max-w-4xl mx-auto">
          <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
            Start free.{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
              Three plans.
            </span>
          </h1>
          <p className="text-xl text-gray-400 max-w-3xl mx-auto leading-relaxed">
            A 14-day trial of Pro features. No credit card. Email verification creates the account.
          </p>
        </div>
      </header>

      {/* Observe, Control, Admit, Prove */}
      <section className="pb-10 px-6">
        <div className="max-w-5xl mx-auto">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-cyan-900/20 border border-cyan-800/30">
              <span className="text-cyan-400 font-bold text-sm">Observe</span>
              <span className="text-gray-500 text-xs">Project impact</span>
            </div>
            <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-purple-900/20 border border-purple-800/30">
              <span className="text-purple-400 font-bold text-sm">Control</span>
              <span className="text-gray-500 text-xs">Enforce owned-agent authority</span>
            </div>
            <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-yellow-900/20 border border-yellow-800/30">
              <span className="text-yellow-400 font-bold text-sm">Admit</span>
              <span className="text-gray-500 text-xs">Govern external-agent access</span>
            </div>
            <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-green-900/20 border border-green-800/30">
              <span className="text-green-400 font-bold text-sm">Prove</span>
              <span className="text-gray-500 text-xs">Verify the evidence</span>
            </div>
          </div>
        </div>
      </section>

      {/* Plans */}
      <section className="pb-10 px-6">
        <div className="max-w-5xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-6 items-stretch">
          <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-600 transition flex flex-col">
            <div className="mb-6">
              <h3 className="text-lg font-bold text-cyan-400 mb-1">Starter</h3>
              <p className="text-gray-500 text-sm">Free. Signup starts a 14-day trial of Pro features.</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">Free</span>
            </div>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Observe (metering) with a fair-use cap</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />3 routes</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Default protection + Observe</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Community support</li>
            </ul>
            <a
              href="https://cloud.satgate.io/cloud/signup"
              target="_blank"
              rel="noopener noreferrer"
              className="block text-center py-3 rounded-lg border border-gray-700 font-bold hover:border-gray-500 hover:bg-gray-800 transition"
            >
              Start free trial
            </a>
          </div>

          <div className="p-6 rounded-xl bg-gray-900 border-2 border-purple-500/60 hover:border-purple-400 transition flex flex-col relative">
            <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-0.5 rounded-full bg-purple-600 text-white text-xs font-bold">
              14-DAY TRIAL
            </div>
            <div className="mb-6">
              <h3 className="text-lg font-bold text-purple-400 mb-1">Pro</h3>
              <p className="text-gray-500 text-sm">14 days free. No credit card.</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">$99</span>
              <span className="text-gray-500 text-sm"> / month</span>
            </div>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Observe with no finite monthly cap</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />1M Control/Admit requests per month</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />25 routes</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Unlimited team members</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />All policies: Observe, Control, and Admit (Admit is the Charge policy in the dashboard)</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Budget enforcement and token governance</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Priority support</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Upgrade by card in the dashboard (Stripe)</li>
            </ul>
            <a
              href="https://cloud.satgate.io/cloud/signup"
              target="_blank"
              rel="noopener noreferrer"
              className="block text-center py-3 rounded-lg bg-gradient-to-r from-purple-600 to-cyan-600 text-white font-bold hover:opacity-90 transition shadow-lg shadow-purple-500/20"
            >
              Start free trial
            </a>
          </div>

          <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-600 transition flex flex-col">
            <div className="mb-6">
              <h3 className="text-lg font-bold text-green-400 mb-1">Enterprise</h3>
              <p className="text-gray-500 text-sm">Self-host the same binary. Custom pricing.</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">Custom</span>
            </div>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Unlimited</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Self-host: the same SatGate binary that runs SatGate Cloud, installed in your own environment</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />SCIM</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />GitOps signed configs</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />WORM audit export</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Dedicated support + SLA</li>
            </ul>
            <a
              href="mailto:contact@satgate.io"
              className="block text-center py-3 rounded-lg border border-gray-700 font-bold hover:border-gray-500 hover:bg-gray-800 transition"
            >
              Contact us
            </a>
          </div>
        </div>
      </section>

      <section className="pb-16 px-6">
        <div className="max-w-3xl mx-auto text-center">
          <p className="text-gray-400">
            Want help rolling out?{' '}
            <Link href="/design-partners" className="text-purple-300 hover:text-purple-200 underline underline-offset-4">
              Ask about a 90-day design-partner pilot
            </Link>
          </p>
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
                  <td className="py-3 px-4 text-white">Paid-rail context — L402/x402-aware governance</td>
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
              <p className="text-gray-400 text-sm">Govern paid-rail context such as L402/x402 while SatGate proves agent authority, policy, and budget decisions.</p>
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
                    openFaq === i ? 'max-h-96 opacity-100' : 'max-h-0 opacity-0'
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
          <h2 className="text-2xl font-bold mb-3">Start a 14-day trial of Pro features.</h2>
          <p className="text-lg text-gray-400 mb-8 max-w-xl mx-auto">
            No credit card. Email verification creates the account.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <a
              href="https://cloud.satgate.io/cloud/signup"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-block bg-white text-black px-10 py-4 rounded-full font-bold text-lg hover:bg-gray-200 transition"
            >
              Start free trial
            </a>
            <a
              href="mailto:contact@satgate.io"
              className="inline-block border border-gray-700 text-gray-300 px-10 py-4 rounded-full font-bold text-lg hover:border-gray-500 hover:bg-gray-800 transition"
            >
              Contact us
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
              <p className="text-gray-500 text-sm">The Agent Authority & Accountability Layer for governed agent requests.</p>
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
                <li><a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud login</a></li>
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

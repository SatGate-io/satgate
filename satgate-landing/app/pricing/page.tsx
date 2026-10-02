'use client';

import React, { useState } from 'react';
import { Check, ChevronDown, Menu, X } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

const faqs = [
  {
    q: 'How do I start?',
    a: 'Sign up at https://cloud.satgate.io/cloud/signup and confirm your email. You get 14 days of Pro features free, with no credit card.',
  },
  {
    q: 'What is included in Starter?',
    a: 'Starter is free. You get Observe (usage tracking) up to a fair-use cap, 3 routes, default protection and community support.',
  },
  {
    q: 'What is included in Pro?',
    a: 'Pro is $99 a month. You get unlimited Observe, 1 million Control and Admit requests a month, 25 routes, unlimited team members and all three controls: Observe, Control and Admit (the dashboard calls Admit “Charge”). It also includes budgets, token management and priority support. Upgrade yourself by card in the dashboard (Stripe).',
  },
  {
    q: 'Can I start without talking to sales?',
    a: 'Yes. Sign up, confirm your email and you have 14 days of Pro features free, with no credit card. Upgrade to Pro by card in the dashboard.',
  },
  {
    q: 'What does Enterprise include?',
    a: 'Enterprise has custom pricing and no usage limits. You run the same SatGate software that powers SatGate Cloud on your own servers. It includes SCIM user provisioning, signed configs managed in Git, write-once (WORM) audit export and dedicated support with an SLA. Email contact@satgate.io.',
  },
  {
    q: 'Does a pilot require code changes?',
    a: 'Usually not. Most pilots start with a DNS, proxy or MCP config change in front of one API endpoint or tool. We make sure that works before adding more.',
  },
  {
    q: 'What does the design-partner pilot cost?',
    a: 'The 90-day design-partner pilot is free, with no credit card. If you continue afterward, the price depends on what we agree to cover, your usage and your support needs. The pilot is for larger teams that want hands-on help. You don’t need it to get started.',
  },
  {
    q: 'What happens after the pilot?',
    a: 'We look at the results together: what SatGate allowed and blocked, how you used the receipts and how much support you needed. Then we both decide whether to expand, scale back or stop.',
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
      { '@type': 'Thing', name: 'Economic firewall for AI agents' },
      { '@type': 'Thing', name: 'AI agent budget enforcement' },
      { '@type': 'Thing', name: 'SatGate Economic Firewall' },
      { '@type': 'Thing', name: 'AI agent spending limits' },
      { '@type': 'Thing', name: 'payment rules that work with any payment method' },
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
        description: 'Free. Observe (usage tracking) up to a fair-use cap, 3 routes, default protection, community support.',
        availability: 'https://schema.org/InStock',
        url: 'https://cloud.satgate.io/cloud/signup',
        itemOffered: { '@type': 'Service', name: 'SatGate Starter', serviceType: 'AI agent governance' },
      },
      {
        '@type': 'Offer',
        name: 'SatGate Pro',
        price: '99',
        priceCurrency: 'USD',
        description: '14-day free trial of Pro features, no credit card. $99/month. Unlimited Observe, 1M Control and Admit requests per month, 25 routes, unlimited team members, all three controls, budgets, token management, priority support. Upgrade by card in the dashboard.',
        availability: 'https://schema.org/InStock',
        url: 'https://cloud.satgate.io/cloud/signup',
        itemOffered: { '@type': 'Service', name: 'SatGate Pro', serviceType: 'AI agent governance' },
      },
      {
        '@type': 'Offer',
        name: 'SatGate Enterprise',
        description: 'Custom pricing, no usage limits. Run the same SatGate software that powers SatGate Cloud on your own servers. SCIM, signed configs in Git, WORM audit export, dedicated support and SLA.',
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
            Try Pro free for 14 days. No credit card; just confirm your email.
          </p>
        </div>
      </header>

      {/* Observe, Control, Admit. Receipts span all three. */}
      <section className="pb-10 px-6">
        <div className="max-w-5xl mx-auto">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-cyan-900/20 border border-cyan-800/30">
              <span className="text-cyan-400 font-bold text-sm">Observe</span>
              <span className="text-gray-500 text-xs">See usage and cost</span>
            </div>
            <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-purple-900/20 border border-purple-800/30">
              <span className="text-purple-400 font-bold text-sm">Control</span>
              <span className="text-gray-500 text-xs">Enforce your agents&apos; limits</span>
            </div>
            <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-yellow-900/20 border border-yellow-800/30">
              <span className="text-yellow-400 font-bold text-sm">Admit</span>
              <span className="text-gray-500 text-xs">Let external agents in, and charge them</span>
            </div>
          </div>
          <p className="mt-4 text-center text-sm text-gray-400">Receipts for all three. Anyone can check a signed receipt with the open-source verifier.</p>
        </div>
      </section>

      {/* Plans */}
      <section className="pb-10 px-6">
        <div className="max-w-5xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-6 items-stretch">
          <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-600 transition flex flex-col">
            <div className="mb-6">
              <h3 className="text-lg font-bold text-cyan-400 mb-1">Starter</h3>
              <p className="text-gray-500 text-sm">Free. New accounts start with 14 days of Pro.</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">Free</span>
            </div>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Observe (usage tracking) up to a fair-use cap</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />3 routes</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Default protection</li>
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
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Unlimited Observe</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />1M Control and Admit requests per month</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />25 routes</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Unlimited team members</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />All three controls: Observe, Control and Admit (called &ldquo;Charge&rdquo; in the dashboard)</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Budgets and token management</li>
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
              <p className="text-gray-500 text-sm">Run it on your own servers. Custom pricing.</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">Custom</span>
            </div>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />No usage limits</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />The same software that runs SatGate Cloud, on your own servers</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />SCIM user provisioning</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Signed configs managed in Git</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Write-once (WORM) audit export</li>
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
          <h2 className="text-3xl font-bold text-center mb-4">What plain MCP doesn&apos;t do</h2>
          <p className="text-gray-400 text-center mb-12 max-w-2xl mx-auto">Out of the box, an MCP server lets agents spend your API credits with nothing to stop them. SatGate adds a meter and a shutoff valve.</p>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800">
                  <th className="text-left py-3 px-4 text-gray-500 font-medium">&nbsp;</th>
                  <th className="text-left py-3 px-4 text-gray-500 font-medium">Plain MCP</th>
                  <th className="text-left py-3 px-4 text-purple-400 font-medium">With SatGate</th>
                </tr>
              </thead>
              <tbody className="text-gray-400">
                <tr className="border-b border-gray-800/50">
                  <td className="py-3 px-4 text-gray-300">Budgets</td>
                  <td className="py-3 px-4">None. You find out when the bill arrives</td>
                  <td className="py-3 px-4 text-white">Hard caps, checked on every call</td>
                </tr>
                <tr className="border-b border-gray-800/50">
                  <td className="py-3 px-4 text-gray-300">Who spent what</td>
                  <td className="py-3 px-4">One big API bill</td>
                  <td className="py-3 px-4 text-white">Broken down by tool and by agent</td>
                </tr>
                <tr className="border-b border-gray-800/50">
                  <td className="py-3 px-4 text-gray-300">Access</td>
                  <td className="py-3 px-4">Static API keys: all or nothing</td>
                  <td className="py-3 px-4 text-white">Tokens limited by tool and by time</td>
                </tr>
                <tr className="border-b border-gray-800/50">
                  <td className="py-3 px-4 text-gray-300">Visibility</td>
                  <td className="py-3 px-4">Dig through logs after the fact</td>
                  <td className="py-3 px-4 text-white">Signed receipts + receipt export</td>
                </tr>
                <tr className="border-b border-gray-800/50">
                  <td className="py-3 px-4 text-gray-300">Runaway agents</td>
                  <td className="py-3 px-4">No limit on spend</td>
                  <td className="py-3 px-4 text-white">Cut off automatically at the cap</td>
                </tr>
                <tr>
                  <td className="py-3 px-4 text-gray-300">Payments</td>
                  <td className="py-3 px-4">None built in</td>
                  <td className="py-3 px-4 text-white">Charge external agents on the routes you choose. Each route takes Lightning, USDC on Base (x402), or both. You set how many requests one Lightning payment buys. One USDC payment buys one request. A signed receipt either way.</td>
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
              <h3 className="font-bold text-white mb-2">Save money</h3>
              <p className="text-gray-400 text-sm">Stop a $500 runaway loop at $2. When an agent hits its cap, SatGate cuts it off.</p>
            </div>
            <div className="text-center">
              <div className="text-3xl mb-3">⚡</div>
              <h3 className="font-bold text-white mb-2">Save time</h3>
              <p className="text-gray-400 text-sm">Engineers stop combing through bills by hand. One gateway covers all your MCP servers.</p>
            </div>
            <div className="text-center">
              <div className="text-3xl mb-3">💰</div>
              <h3 className="font-bold text-white mb-2">Earn money</h3>
              <p className="text-gray-400 text-sm">Charge external agents on the routes you choose. Each route takes Lightning, USDC on Base, or both. One USDC payment on Base buys one request. On Lightning routes you choose how many requests one payment buys. A signed receipt either way.</p>
            </div>
          </div>
        </div>
      </section>

      {/* FAQ Section */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-3xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">Questions</h2>
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
          <h2 className="text-2xl font-bold mb-3">Try Pro free for 14 days.</h2>
          <p className="text-lg text-gray-400 mb-8 max-w-xl mx-auto">
            No credit card. Just confirm your email.
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
              <p className="text-gray-500 text-sm">An economic firewall for AI agents.</p>
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

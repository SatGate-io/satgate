'use client';

import React, { useState } from 'react';
import { Check, Zap, ArrowRight, ChevronDown, Menu, X } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

const faqs = [
  {
    q: 'What counts as a request?',
    a: 'Every API call proxied through the gateway in Control or Charge mode. Observe mode is always free and unlimited.',
  },
  {
    q: 'Can I switch plans?',
    a: 'Yes, upgrade or downgrade anytime. Changes take effect immediately.',
  },
  {
    q: 'Do you offer annual billing?',
    a: 'Yes, save 20% with annual billing. Contact us for details.',
  },
  {
    q: 'What happens if I exceed my request limit?',
    a: 'Overage is billed at $0.10 per 1,000 requests. No surprise charges — you\'ll get alerts at 80% and 90%.',
  },
  {
    q: 'Is there a free trial of Pro?',
    a: 'Every account starts with a 14-day Pro trial. No credit card required.',
  },
  {
    q: 'Do you support hybrid/on-prem deployment?',
    a: 'Yes, on the Enterprise plan. The gateway runs in your infrastructure with our managed control plane. Or fully air-gapped.',
  },
];

const PricingPage = () => {
  const [openFaq, setOpenFaq] = useState<number | null>(null);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans selection:bg-purple-500 selection:text-white">
      {/* Navigation */}
      <nav className="border-b border-gray-800 backdrop-blur-md fixed w-full z-50 bg-black/50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2 shrink-0">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="w-7 h-7 sm:w-8 sm:h-8" />
            <span className="text-lg sm:text-xl font-bold text-white whitespace-nowrap">SatGate<sup className="text-xs font-normal">™</sup></span>
          </Link>

          {/* Desktop menu */}
          <div className="hidden md:flex gap-6 text-sm font-medium text-gray-400">
            <Link href="/protect" className="hover:text-white transition">Protect Demo</Link>
            <Link href="/pay" className="hover:text-white transition">Pay Demo</Link>
            <Link href="/govern" className="hover:text-white transition">Govern</Link>
            <Link href="/pricing" className="text-white transition">Pricing</Link>
            <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://cloud.satgate.io" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud</a>
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
            <Link href="/protect" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Protect Demo</Link>
            <Link href="/pay" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Pay Demo</Link>
            <Link href="/govern" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Govern</Link>
            <Link href="/pricing" onClick={() => setMobileMenuOpen(false)} className="block text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Pricing</Link>
            <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Docs</a>
            <a href="https://cloud.satgate.io" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Cloud</a>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <header className="pt-32 pb-16 px-6 text-center">
        <div className="max-w-4xl mx-auto">
          <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
            Simple, Transparent{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
              Pricing
            </span>
          </h1>
          <p className="text-xl text-gray-400 max-w-lg mx-auto leading-relaxed">
            Start free. Scale when you&apos;re ready.
          </p>
        </div>
      </header>

      {/* Pricing Cards */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-6 items-stretch">
          {/* Free (Observe) */}
          <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-600 transition flex flex-col">
            <div className="mb-6">
              <h3 className="text-lg font-bold text-cyan-400 mb-1">Free</h3>
              <p className="text-gray-500 text-sm">Observe</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">$0</span>
              <span className="text-gray-500 text-sm">/month</span>
            </div>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Unlimited observe-mode requests</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Usage dashboard + analytics</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Up to 3 routes</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-cyan-400 mt-0.5 shrink-0" />Community support</li>
            </ul>
            <a
              href="https://cloud.satgate.io"
              target="_blank"
              rel="noopener noreferrer"
              className="block text-center py-3 rounded-lg border border-gray-700 font-bold hover:border-gray-500 hover:bg-gray-800 transition"
            >
              Start Free →
            </a>
          </div>

          {/* Pro (Control + Charge) — highlighted */}
          <div className="p-6 rounded-xl bg-gray-900 border-2 border-purple-500/60 hover:border-purple-400 transition flex flex-col relative">
            <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-0.5 rounded-full bg-purple-600 text-white text-xs font-bold">
              MOST POPULAR
            </div>
            <div className="mb-6">
              <h3 className="text-lg font-bold text-purple-400 mb-1">Pro</h3>
              <p className="text-gray-500 text-sm">Control + Charge</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">$99</span>
              <span className="text-gray-500 text-sm">/month</span>
            </div>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />1M control/charge requests included</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Then $0.10 per 1K requests</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Unlimited routes</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Budget enforcement + alerts</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />CFO chargeback reports</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Security dashboard + compliance exports</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-purple-400 mt-0.5 shrink-0" />Email support</li>
            </ul>
            <a
              href="https://cloud.satgate.io"
              target="_blank"
              rel="noopener noreferrer"
              className="block text-center py-3 rounded-lg bg-gradient-to-r from-purple-600 to-cyan-600 text-white font-bold hover:opacity-90 transition shadow-lg shadow-purple-500/20"
            >
              Start Pro →
            </a>
          </div>

          {/* Enterprise */}
          <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-600 transition flex flex-col">
            <div className="mb-6">
              <h3 className="text-lg font-bold text-green-400 mb-1">Enterprise</h3>
              <p className="text-gray-500 text-sm">Custom</p>
            </div>
            <div className="mb-6">
              <span className="text-4xl font-extrabold text-white">Custom</span>
            </div>
            <ul className="space-y-3 text-sm text-gray-400 mb-8 flex-1">
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Unlimited requests</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Hybrid deployment (your infrastructure)</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />SSO/SCIM integration</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Dedicated support + SLA</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />SOC2 compliance package</li>
              <li className="flex items-start gap-2"><Check size={16} className="text-green-400 mt-0.5 shrink-0" />Custom integrations</li>
            </ul>
            <a
              href="mailto:contact@satgate.io"
              className="block text-center py-3 rounded-lg border border-gray-700 font-bold hover:border-gray-500 hover:bg-gray-800 transition"
            >
              Contact Sales
            </a>
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
          <p className="text-xl text-gray-400 mb-6">
            Not sure which plan? Start with Free — upgrade when your CFO sees the report.
          </p>
          <a
            href="https://cloud.satgate.io"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-block bg-white text-black px-10 py-4 rounded-full font-bold text-lg hover:bg-gray-200 transition"
          >
            Start Free →
          </a>
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
              <p className="text-gray-500 text-sm">EZ-Pass for the API Economy.</p>
              <p className="text-gray-600 text-xs mt-3">Non-custodial. We never hold your keys.</p>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Resources</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a></li>
                <li><a href="https://github.com/SatGate-io/satgate#-quick-start" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Documentation</a></li>
                <li><Link href="/govern" className="hover:text-white transition">Govern</Link></li>
                <li><Link href="/design-partners" className="hover:text-white transition">Design Partners</Link></li>
                <li><Link href="/pricing" className="hover:text-white transition">Pricing</Link></li>
                <li><a href="https://cloud.satgate.io" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud Dashboard</a></li>
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
                <li><a href="https://x.com/SatGateIO" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">@SatGateIO</a></li>
              </ul>
            </div>
          </div>
          <div className="pt-8 border-t border-gray-800 text-center text-gray-600 text-sm">
            © 2025 SatGate Inc. All rights reserved. SatGate™ is a trademark of SatGate Inc. Patent Pending.
          </div>
        </div>
      </footer>
    </div>
  );
};

export default PricingPage;

'use client';

import React, { useState } from 'react';
import {
  ArrowRight, CheckCircle, Clock, Eye, MessageSquare, Headphones,
  Server, Calendar, BarChart3, Shield, ChevronDown, ChevronUp,
  Zap, Users, Code, Menu, X, Send, Building2, UserCheck
} from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

export default function DesignPartnersPage() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [openFaq, setOpenFaq] = useState<number | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    company: '',
    role: '',
    agentCount: '',
    apis: [] as string[],
    challenge: '',
  });

  const toggleFaq = (idx: number) => {
    setOpenFaq(openFaq === idx ? null : idx);
  };

  const handleApiToggle = (api: string) => {
    setFormData(prev => ({
      ...prev,
      apis: prev.apis.includes(api) ? prev.apis.filter(a => a !== api) : [...prev.apis, api],
    }));
  };

  const [submitStatus, setSubmitStatus] = useState<'idle' | 'sending' | 'sent' | 'error'>('idle');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitStatus('sending');
    try {
      const res = await fetch('/api/design-partner', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });
      if (res.ok) {
        setSubmitStatus('sent');
      } else {
        // Fallback to mailto
        const subject = encodeURIComponent('Design Partner Application');
        const body = encodeURIComponent(
          `Name: ${formData.name}\nEmail: ${formData.email}\nCompany: ${formData.company}\nRole: ${formData.role}\nAI Agents: ${formData.agentCount}\nAPIs: ${formData.apis.join(', ')}\n\nBiggest Challenge:\n${formData.challenge}`
        );
        window.location.href = `mailto:contact@satgate.io?subject=${subject}&body=${body}`;
        setSubmitStatus('sent');
      }
    } catch {
      // Fallback to mailto
      const subject = encodeURIComponent('Design Partner Application');
      const body = encodeURIComponent(
        `Name: ${formData.name}\nEmail: ${formData.email}\nCompany: ${formData.company}\nRole: ${formData.role}\nAI Agents: ${formData.agentCount}\nAPIs: ${formData.apis.join(', ')}\n\nBiggest Challenge:\n${formData.challenge}`
      );
      window.location.href = `mailto:contact@satgate.io?subject=${subject}&body=${body}`;
      setSubmitStatus('sent');
    }
  };

  const faqs = [
    { q: 'How long is the program?', a: '90 days. After that, you keep everything you built and get priority access to GA pricing.' },
    { q: 'Is it really free?', a: 'Yes. Observe mode is completely free—no credit card, no commitment. You get full visibility into your AI agent traffic at zero cost.' },
    { q: 'Do I need to change my code?', a: 'No. One DNS change points your AI agent traffic through SatGate. Zero code modifications. Takes about 5 minutes.' },
    { q: 'Where does my data go?', a: 'Your infrastructure in hybrid mode. The SatGate gateway runs in your VPC. We never see your API payloads or sensitive data.' },
    { q: 'Is this production-ready?', a: 'Yes. 60+ dashboard pages, full deployment options (Docker, K8s, Terraform, SaaS), and a battle-tested Go binary with zero dependencies.' },
  ];

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'SatGate Design Partners Program',
    url: 'https://satgate.io/design-partners',
    description: 'Early access for teams shaping SatGate economic firewall capabilities for AI agent budget enforcement, MCP governance, API controls, and L402 Charge.',
    datePublished: '2026-04-27',
    dateModified: '2026-05-02',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI agent economic governance' },
      { '@type': 'Thing', name: 'economic firewall design partners' },
      { '@type': 'Thing', name: 'MCP governance' },
      { '@type': 'Thing', name: 'L402 Charge' },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Design Partners', item: 'https://satgate.io/design-partners' },
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

  const programJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'Service',
    name: 'SatGate Design Partners Program',
    serviceType: 'AI agent economic governance design partner program',
    url: 'https://satgate.io/design-partners',
    provider: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    description: 'Early access for teams shaping SatGate economic firewall capabilities across AI agent budget enforcement, MCP governance, agent API controls, and L402 Charge.',
    areaServed: 'Global',
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans selection:bg-purple-500 selection:text-white">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(programJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      {/* Navigation */}
      <nav className="border-b border-gray-800 backdrop-blur-md fixed w-full z-50 bg-black/50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2 shrink-0">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="w-7 h-7 sm:w-8 sm:h-8" />
            <span className="text-lg sm:text-xl font-bold text-white whitespace-nowrap">SatGate<sup className="text-xs font-normal">™</sup></span>
          </Link>

          <div className="hidden md:flex gap-6 text-sm font-medium text-gray-400">
            <Link href="/mint-demo" className="hover:text-white transition">Mint Demo</Link>
            <Link href="/protect" className="hover:text-white transition">Control Demo</Link>
            <Link href="/pay" className="hover:text-white transition">Charge Demo</Link>
            <Link href="/govern" className="hover:text-white transition">Enterprise</Link>
            <Link href="/pricing" className="hover:text-white transition">Pricing</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud</a>
          </div>

          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="flex md:hidden items-center justify-center w-10 h-10 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 text-gray-400 hover:text-white transition"
            aria-label="Toggle menu"
            aria-expanded={mobileMenuOpen}
          >
            {mobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
        </div>

        <div className={`md:hidden overflow-hidden transition-all duration-300 ease-in-out ${mobileMenuOpen ? 'max-h-96 opacity-100' : 'max-h-0 opacity-0'}`}>
          <div className="bg-black/95 backdrop-blur-xl border-t border-gray-800 px-4 py-4 space-y-1">
            {[
              { href: '/mint-demo', label: 'Mint Demo' },
              { href: '/protect', label: 'Control Demo' },
              { href: '/pay', label: 'Charge Demo' },
              { href: '/govern', label: 'Enterprise' },
              { href: '/pricing', label: 'Pricing' },
            ].map(item => (
              <Link key={item.href} href={item.href} onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">
                {item.label}
              </Link>
            ))}
            <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">
              Docs
            </a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">
              Cloud
            </a>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <header className="pt-32 pb-20 px-6">
        <div className="max-w-4xl mx-auto text-center">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono mb-6">
            <Users size={12} /> Limited to 10 Companies
          </div>
          <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
            Shape the Future of{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 via-purple-400 to-pink-400">
              AI Agent Governance
            </span>
          </h1>
          <p className="text-xl text-gray-400 mb-10 max-w-2xl mx-auto leading-relaxed">
            We're working with <strong className="text-white">10 enterprises</strong> to deploy the Economic Firewall
            that enforces Economic Access Control for AI agent requests. Gate your MCP tool servers, REST APIs, and LLM endpoints — see what each agent spends per call. Get free access, direct engineering support, and a product shaped by your needs.
          </p>
          <a href="#apply" className="inline-flex items-center gap-2 bg-white text-black px-8 py-3 rounded-lg font-bold hover:bg-gray-200 transition">
            Apply Now <ArrowRight size={18} />
          </a>
        </div>
      </header>

      {/* What You Get */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">What You Get</h2>
          <p className="text-gray-500 text-center mb-12">Everything you need to evaluate SatGate. No strings attached.</p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="p-6 rounded-xl bg-gradient-to-br from-purple-950/30 to-purple-900/10 border border-purple-800/30 hover:border-purple-600/50 transition group">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-purple-900/50 rounded-lg group-hover:bg-purple-900/70 transition">
                  <Eye className="text-purple-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Free Observe Mode</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed">
                Full visibility into your AI agent traffic on your staging environment. See every API call, every token, every cost—across MCP servers, REST APIs, LLM endpoints—completely free.
              </p>
            </div>

            <div className="p-6 rounded-xl bg-gradient-to-br from-cyan-950/30 to-cyan-900/10 border border-cyan-800/30 hover:border-cyan-600/50 transition group">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-cyan-900/50 rounded-lg group-hover:bg-cyan-900/70 transition">
                  <Headphones className="text-cyan-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Direct Engineering Access</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed">
                Shared Slack channel with our engineering team. Priority bug fixes, feature requests, and architecture guidance tailored to your stack.
              </p>
            </div>

            <div className="p-6 rounded-xl bg-gradient-to-br from-green-950/30 to-green-900/10 border border-green-800/30 hover:border-green-600/50 transition group">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-green-900/50 rounded-lg group-hover:bg-green-900/70 transition">
                  <MessageSquare className="text-green-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Your Feedback Shapes Product</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed">
                Design partners get direct influence on the roadmap. The dashboards, alerts, and governance features you need—built with your input.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* What We Need */}
      <section className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-gray-900/30 to-black">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">What We Need</h2>
          <p className="text-gray-500 text-center mb-12">A small commitment. A big impact.</p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-600 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-gray-800 rounded-lg">
                  <Server className="text-gray-300" size={22} />
                </div>
                <h3 className="font-bold text-lg">One Endpoint or Tool</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed">
                Point one staging API endpoint or MCP tool through SatGate. One DNS change or config update, no code modifications.
              </p>
            </div>

            <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-600 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-gray-800 rounded-lg">
                  <Clock className="text-gray-300" size={22} />
                </div>
                <h3 className="font-bold text-lg">15 Minutes / Week</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed">
                A quick sync call or async feedback. Tell us what works, what doesn't, and what you need next.
              </p>
            </div>

            <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-gray-600 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-gray-800 rounded-lg">
                  <UserCheck className="text-gray-300" size={22} />
                </div>
                <h3 className="font-bold text-lg">Honest Feedback</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed">
                We want the truth. If something's broken, tell us. If a feature is missing, we need to know. Your candor makes the product better.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works - Timeline */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">How It Works</h2>
          <p className="text-gray-500 text-center mb-12">From application to insights in under 3 weeks.</p>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            {[
              {
                step: '1',
                title: 'Sign Up',
                time: '5 minutes',
                desc: 'Fill out the form below. We review and respond within 24 hours.',
                color: 'purple',
                icon: <Send size={20} />,
              },
              {
                step: '2',
                title: 'Setup Call',
                time: '30 minutes',
                desc: 'We walk through your architecture and configure SatGate for your environment.',
                color: 'cyan',
                icon: <Calendar size={20} />,
              },
              {
                step: '3',
                title: 'Traffic Flows',
                time: 'Week 1–2',
                desc: 'Route staging traffic through SatGate. Observe mode starts collecting data immediately.',
                color: 'green',
                icon: <Zap size={20} />,
              },
              {
                step: '4',
                title: 'Your Reports',
                time: 'Week 3+',
                desc: 'Your data, your dashboards. Cost attribution, security insights, governance reports—all yours.',
                color: 'yellow',
                icon: <BarChart3 size={20} />,
              },
            ].map((item) => (
              <div key={item.step} className="relative p-5 rounded-xl bg-gray-900 border border-gray-800 text-center">
                <div className={`inline-flex items-center justify-center w-10 h-10 rounded-full bg-${item.color}-900/50 border-2 border-${item.color}-500/60 mb-4`}>
                  <span className={`text-${item.color}-400`}>{item.icon}</span>
                </div>
                <h4 className="font-semibold text-white mb-1">{item.title}</h4>
                <p className={`text-${item.color}-400 text-xs font-mono mb-2`}>{item.time}</p>
                <p className="text-gray-500 text-sm">{item.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Application Form */}
      <section id="apply" className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-purple-950/10 to-black">
        <div className="max-w-2xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">Apply to the Design Partner Program</h2>
          <p className="text-gray-500 text-center mb-10">Takes about 2 minutes. We'll respond within 24 hours.</p>

          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Name */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Name *</label>
              <input
                type="text"
                required
                value={formData.name}
                onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                className="w-full px-4 py-3 bg-gray-900 border border-gray-800 rounded-lg text-white placeholder-gray-500 focus:border-purple-500 focus:outline-none focus:ring-1 focus:ring-purple-500/50 transition"
                placeholder="Jane Smith"
              />
            </div>

            {/* Email */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Work Email *</label>
              <input
                type="email"
                required
                value={formData.email}
                onChange={(e) => setFormData(prev => ({ ...prev, email: e.target.value }))}
                className="w-full px-4 py-3 bg-gray-900 border border-gray-800 rounded-lg text-white placeholder-gray-500 focus:border-purple-500 focus:outline-none focus:ring-1 focus:ring-purple-500/50 transition"
                placeholder="jane@company.com"
              />
            </div>

            {/* Company */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Company *</label>
              <input
                type="text"
                required
                value={formData.company}
                onChange={(e) => setFormData(prev => ({ ...prev, company: e.target.value }))}
                className="w-full px-4 py-3 bg-gray-900 border border-gray-800 rounded-lg text-white placeholder-gray-500 focus:border-purple-500 focus:outline-none focus:ring-1 focus:ring-purple-500/50 transition"
                placeholder="Acme Corp"
              />
            </div>

            {/* Role */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Role / Title *</label>
              <input
                type="text"
                required
                value={formData.role}
                onChange={(e) => setFormData(prev => ({ ...prev, role: e.target.value }))}
                className="w-full px-4 py-3 bg-gray-900 border border-gray-800 rounded-lg text-white placeholder-gray-500 focus:border-purple-500 focus:outline-none focus:ring-1 focus:ring-purple-500/50 transition"
                placeholder="VP Engineering"
              />
            </div>

            {/* Number of AI Agents */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Number of AI Agents *</label>
              <select
                required
                value={formData.agentCount}
                onChange={(e) => setFormData(prev => ({ ...prev, agentCount: e.target.value }))}
                className="w-full px-4 py-3 bg-gray-900 border border-gray-800 rounded-lg text-white focus:border-purple-500 focus:outline-none focus:ring-1 focus:ring-purple-500/50 transition appearance-none"
              >
                <option value="" className="text-gray-500">Select range...</option>
                <option value="1-10">1–10</option>
                <option value="10-50">10–50</option>
                <option value="50-100">50–100</option>
                <option value="100+">100+</option>
              </select>
            </div>

            {/* Primary APIs */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-3">Primary APIs Used</label>
              <div className="grid grid-cols-2 gap-3">
                {['OpenAI', 'Anthropic', 'Internal APIs', 'Other'].map(api => (
                  <label key={api} className="flex items-center gap-3 p-3 bg-gray-900 border border-gray-800 rounded-lg cursor-pointer hover:border-gray-600 transition has-[:checked]:border-purple-500/60 has-[:checked]:bg-purple-950/20">
                    <input
                      type="checkbox"
                      checked={formData.apis.includes(api)}
                      onChange={() => handleApiToggle(api)}
                      className="w-4 h-4 rounded border-gray-700 bg-gray-800 text-purple-500 focus:ring-purple-500/50 focus:ring-offset-0"
                    />
                    <span className="text-sm text-gray-300">{api}</span>
                  </label>
                ))}
              </div>
            </div>

            {/* Challenge */}
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                What's your biggest challenge with AI agent management?
              </label>
              <textarea
                value={formData.challenge}
                onChange={(e) => setFormData(prev => ({ ...prev, challenge: e.target.value }))}
                rows={4}
                className="w-full px-4 py-3 bg-gray-900 border border-gray-800 rounded-lg text-white placeholder-gray-500 focus:border-purple-500 focus:outline-none focus:ring-1 focus:ring-purple-500/50 transition resize-none"
                placeholder="e.g., We can't track which team is responsible for our $50k/month OpenAI bill..."
              />
            </div>

            {/* Submit */}
            {submitStatus === 'sent' ? (
              <div className="w-full py-4 bg-green-900/30 border border-green-700/50 text-green-400 rounded-xl font-bold text-lg flex items-center justify-center gap-2">
                ✓ Application received! We&apos;ll be in touch within 24 hours.
              </div>
            ) : (
              <button
                type="submit"
                disabled={submitStatus === 'sending'}
                className="w-full py-4 bg-gradient-to-r from-purple-600 to-cyan-600 text-white rounded-xl font-bold text-lg hover:opacity-90 transition shadow-lg shadow-purple-500/20 flex items-center justify-center gap-2 disabled:opacity-50"
              >
                {submitStatus === 'sending' ? 'Submitting...' : 'Submit Application'} <Send size={18} />
              </button>
            )}

            <p className="text-center text-gray-600 text-xs">
              By submitting, you agree to our{' '}
              <Link href="/terms" className="text-gray-500 underline hover:text-white transition">Terms</Link>{' '}
              and{' '}
              <Link href="/privacy" className="text-gray-500 underline hover:text-white transition">Privacy Policy</Link>.
            </p>
          </form>
        </div>
      </section>

      {/* FAQ */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-2xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-4">Frequently Asked Questions</h2>
          <p className="text-gray-500 text-center mb-12">Quick answers to common questions.</p>

          <div className="space-y-3">
            {faqs.map((faq, idx) => (
              <div key={idx} className="rounded-xl border border-gray-800 overflow-hidden">
                <button
                  onClick={() => toggleFaq(idx)}
                  className="w-full flex items-center justify-between p-5 text-left hover:bg-gray-900/50 transition"
                >
                  <span className="font-medium text-white pr-4">{faq.q}</span>
                  {openFaq === idx ? (
                    <ChevronUp size={18} className="text-gray-400 flex-shrink-0" />
                  ) : (
                    <ChevronDown size={18} className="text-gray-400 flex-shrink-0" />
                  )}
                </button>
                <div className={`overflow-hidden transition-all duration-300 ease-in-out ${openFaq === idx ? 'max-h-40 opacity-100' : 'max-h-0 opacity-0'}`}>
                  <div className="px-5 pb-5 text-gray-400 text-sm leading-relaxed">
                    {faq.a}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Final CTA */}
      <section className="py-20 px-6 border-t border-gray-800 bg-gradient-to-b from-cyan-950/10 to-black">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="text-3xl font-bold mb-4">Still have questions?</h2>
          <p className="text-gray-400 mb-8">
            Reach out directly. We'd love to chat about your use case.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <a href="mailto:contact@satgate.io" className="border border-gray-700 px-8 py-3 rounded-lg font-bold hover:border-gray-500 transition flex items-center justify-center gap-2">
              Email Us <ArrowRight size={18} />
            </a>
            <a href="#apply" className="bg-white text-black px-8 py-3 rounded-lg font-bold hover:bg-gray-200 transition flex items-center justify-center gap-2">
              Apply Now <ArrowRight size={18} />
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
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Product</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><Link href="/protect" className="hover:text-white transition">Protect</Link></li>
                <li><Link href="/pay" className="hover:text-white transition">Pay</Link></li>
                <li><Link href="/govern" className="hover:text-white transition">Enterprise</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Resources</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a></li>
                <li><a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud Dashboard</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Company</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="mailto:contact@satgate.io" className="hover:text-white transition">contact@satgate.io</a></li>
                <li><Link href="/terms" className="hover:text-white transition">Terms</Link></li>
                <li><Link href="/privacy" className="hover:text-white transition">Privacy</Link></li>
              </ul>
            </div>
          </div>
          <div className="pt-8 border-t border-gray-800 text-center text-gray-600 text-sm">
            © 2025–2026 SatGate Inc. All rights reserved. SatGate™ is a trademark of SatGate Inc. Patent Pending.
          </div>
        </div>
      </footer>
    </div>
  );
}

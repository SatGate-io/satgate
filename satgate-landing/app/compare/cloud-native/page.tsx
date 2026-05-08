'use client';

import React, { useState } from 'react';
import { ArrowRight, Menu, X, Shield, DollarSign, Globe, Zap, Eye, Lock, Cloud, Bot } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';
import type { Metadata } from 'next';

const comparisonRows = [
  {
    feature: 'Vendor Lock-in',
    icon: <Lock size={16} className="text-gray-400" />,
    cloud: 'High — policies only work inside their cloud',
    satgate: 'None — unified governance for Anthropic, OpenAI, local LLMs, any MCP tool',
  },
  {
    feature: 'Budget Enforcement',
    icon: <DollarSign size={16} className="text-gray-400" />,
    cloud: 'Reactive — throttling/alerts after spend occurs',
    satgate: 'Proactive — real-time hard-caps at tool-call level via request-path policy',
  },
  {
    feature: 'Governance Scope',
    icon: <Shield size={16} className="text-gray-400" />,
    cloud: 'Blunt — general "least privilege" IAM roles',
    satgate: 'Granular — per-tool, per-session, per-task constraints',
  },
  {
    feature: 'Cost Attribution',
    icon: <Eye size={16} className="text-gray-400" />,
    cloud: 'Aggregate — impossible to see which tool caused a spike',
    satgate: 'Surgical — real-time per-tool visibility',
  },
  {
    feature: 'Authentication',
    icon: <Lock size={16} className="text-gray-400" />,
    cloud: 'Centralized — dependent on EntraID/IAM',
    satgate: 'Decentralized — macaroons for delegation, L402 for proof-of-budget',
  },
  {
    feature: 'Visibility',
    icon: <Zap size={16} className="text-gray-400" />,
    cloud: 'Delayed — logs take minutes/hours to reach billing',
    satgate: 'Instant — millisecond-level economic telemetry',
  },
  {
    feature: 'Multi-Cloud',
    icon: <Globe size={16} className="text-gray-400" />,
    cloud: 'No — each cloud is its own silo',
    satgate: 'Yes — one governance layer across all providers',
  },
  {
    feature: 'Agent-to-Agent Commerce',
    icon: <Bot size={16} className="text-gray-400" />,
    cloud: 'Not supported',
    satgate: 'L402 micropayments built-in',
  },
];

const CloudNativeComparisonPage = () => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'SatGate vs Cloud-Native AI Governance',
    url: 'https://satgate.io/compare/cloud-native',
    description: 'Compare SatGate with cloud-native AI governance for provider-neutral agent spend control, MCP tool governance, and request-path enforcement.',
    datePublished: '2026-04-30',
    dateModified: '2026-05-06',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'cloud-native AI governance alternative' },
      { '@type': 'Thing', name: 'provider-neutral agent spend control' },
      { '@type': 'Thing', name: 'request-path budget enforcement' },
      { '@type': 'Thing', name: 'MCP tool governance across clouds' },
    ],
  };

  const comparisonJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'SatGate vs cloud-native AI governance comparison',
    description: 'Provider-neutral request-path controls SatGate adds beyond cloud-native AI governance for autonomous agents, MCP tools, budgets, and robot-customer commerce.',
    itemListElement: comparisonRows.map((row, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: row.feature,
      description: `Cloud-native: ${row.cloud}. SatGate: ${row.satgate}.`,
    })),
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'Why are cloud-native AI governance tools not enough for autonomous agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Cloud-native tools are usually siloed to one provider and built around IAM, logs, and after-the-fact billing. Autonomous agents need request-path controls that follow tool calls across clouds, APIs, and MCP servers.',
        },
      },
      {
        '@type': 'Question',
        name: 'How is SatGate different from AWS, Azure, or GCP AI governance?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate is provider-neutral and enforces economic policy before upstream calls execute. It can cap per-agent, per-tool, and per-task spend across clouds instead of relying on one cloud billing system.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can SatGate work alongside cloud-native AI platforms?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. SatGate can sit between agents and cloud-hosted models or tools, adding budget enforcement, cost attribution, and policy controls while the cloud platform continues to provide compute and model hosting.',
        },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans selection:bg-purple-500 selection:text-white">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(comparisonJsonLd) }} />
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
            <Link href="/roi-calculator" className="hover:text-white transition">ROI Calculator</Link>
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

        <div className={`md:hidden overflow-hidden transition-all duration-300 ease-in-out ${mobileMenuOpen ? 'max-h-80 opacity-100' : 'max-h-0 opacity-0'}`}>
          <div className="bg-black/95 backdrop-blur-xl border-t border-gray-800 px-4 py-4 space-y-1">
            <Link href="/mint-demo" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Mint Demo</Link>
            <Link href="/protect" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Control Demo</Link>
            <Link href="/pay" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Charge Demo</Link>
            <Link href="/govern" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Enterprise</Link>
            <Link href="/pricing" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Pricing</Link>
            <Link href="/roi-calculator" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">ROI Calculator</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Docs</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Cloud</a>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <header className="pt-32 pb-16 px-6 text-center">
        <div className="max-w-4xl mx-auto">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-purple-900/30 border border-purple-700/40 text-purple-300 text-sm font-medium mb-8">
            <Cloud size={14} />
            Comparison Guide
          </div>
          <h1 className="text-4xl sm:text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
            SatGate vs{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
              Cloud-Native AI Governance
            </span>
          </h1>
          <p className="text-lg sm:text-xl text-gray-400 max-w-2xl mx-auto leading-relaxed">
            Why your cloud provider&apos;s built-in tools aren&apos;t enough for the Agentic Web
          </p>
          <p className="text-base text-gray-500 max-w-xl mx-auto mt-4 leading-relaxed">
            Enterprise AI teams are told to &ldquo;just use what the cloud gives you.&rdquo; But cloud-native governance has blind spots that grow as agents become more autonomous.
          </p>
        </div>
      </header>

      {/* Comparison Table */}
      <section className="pb-20 px-4 sm:px-6">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-2xl sm:text-3xl font-bold text-center mb-10">
            Feature-by-Feature{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 to-cyan-400">Comparison</span>
          </h2>

          <div className="overflow-x-auto rounded-xl border border-gray-800">
            <table className="w-full min-w-[700px]">
              <thead>
                <tr className="border-b border-gray-800 bg-gray-900/50">
                  <th className="text-left py-4 px-5 text-sm font-semibold text-gray-400 w-[200px]">Feature</th>
                  <th className="text-left py-4 px-5 text-sm font-semibold text-gray-400">
                    <span className="flex items-center gap-2">
                      <Cloud size={16} />
                      Cloud-Native (AWS / Azure)
                    </span>
                  </th>
                  <th className="text-left py-4 px-5 text-sm font-semibold text-purple-300">
                    <span className="flex items-center gap-2">
                      <Image src="/logo_white_transparent.png" alt="SatGate" width={16} height={16} />
                      SatGate MCP Proxy
                    </span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {comparisonRows.map((row, i) => (
                  <tr key={row.feature} className={`border-b border-gray-800/60 ${i % 2 === 0 ? 'bg-black' : 'bg-gray-900/20'} hover:bg-gray-900/40 transition`}>
                    <td className="py-4 px-5">
                      <span className="flex items-center gap-2 text-sm font-semibold text-white">
                        {row.icon}
                        {row.feature}
                      </span>
                    </td>
                    <td className="py-4 px-5 text-sm text-gray-400">{row.cloud}</td>
                    <td className="py-4 px-5 text-sm text-green-300 font-medium">{row.satgate}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Three Reasons */}
      <section className="pb-20 px-6">
        <div className="max-w-4xl mx-auto space-y-12">
          <h2 className="text-2xl sm:text-3xl font-bold text-center mb-4">
            Three Reasons{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 to-cyan-400">Cloud-Native Falls Short</span>
          </h2>

          {/* Reason 1 */}
          <div className="p-6 sm:p-8 rounded-xl bg-gray-900/50 border border-gray-800 hover:border-purple-800/50 transition">
            <div className="flex items-center gap-3 mb-4">
              <span className="flex items-center justify-center w-8 h-8 rounded-lg bg-purple-900/50 text-purple-400 font-bold text-sm">1</span>
              <h3 className="text-xl font-bold">The Multi-Cloud Blindspot</h3>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              Enterprises are never single-cloud. Claude on AWS Bedrock, GPT on Azure OpenAI, custom models on-prem. Your governance can&apos;t be siloed to one provider.
            </p>
            <blockquote className="border-l-2 border-purple-500 pl-4 text-purple-300 italic">
              &ldquo;If you rely on AWS for security, what happens when your agent calls a tool on Azure?&rdquo;
            </blockquote>
          </div>

          {/* Reason 2 */}
          <div className="p-6 sm:p-8 rounded-xl bg-gray-900/50 border border-gray-800 hover:border-cyan-800/50 transition">
            <div className="flex items-center gap-3 mb-4">
              <span className="flex items-center justify-center w-8 h-8 rounded-lg bg-cyan-900/50 text-cyan-400 font-bold text-sm">2</span>
              <h3 className="text-xl font-bold">The &ldquo;Financial Hallucination&rdquo; Guardrail</h3>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              Cloud providers are built to help you <em>spend</em> money — scale, availability, more compute. Their incentives don&apos;t align with cost containment.
            </p>
            <blockquote className="border-l-2 border-cyan-500 pl-4 text-cyan-300 italic">
              &ldquo;AWS Bedrock wants your agents to be successful. SatGate wants your agents to be profitable.&rdquo;
            </blockquote>
          </div>

          {/* Reason 3 */}
          <div className="p-6 sm:p-8 rounded-xl bg-gray-900/50 border border-gray-800 hover:border-yellow-800/50 transition">
            <div className="flex items-center gap-3 mb-4">
              <span className="flex items-center justify-center w-8 h-8 rounded-lg bg-yellow-900/50 text-yellow-400 font-bold text-sm">3</span>
              <h3 className="text-xl font-bold">L402: The Future of Agent-to-Agent Commerce</h3>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              Cloud providers think in terms of &ldquo;users.&rdquo; SatGate thinks in terms of &ldquo;economies.&rdquo; As agents become autonomous economic actors, they need native payment primitives.
            </p>
            <blockquote className="border-l-2 border-yellow-500 pl-4 text-yellow-300 italic">
              &ldquo;In 24 months, your agents will be paying other companies&apos; agents for data.&rdquo;
            </blockquote>
          </div>
        </div>
      </section>

      {/* IT Director Objection Handler */}
      <section className="pb-20 px-6">
        <div className="max-w-3xl mx-auto">
          <div className="p-6 sm:p-8 rounded-xl bg-gradient-to-br from-gray-900 to-gray-900/50 border border-gray-700">
            <h2 className="text-xl sm:text-2xl font-bold mb-2 flex items-center gap-2">
              🤔 &ldquo;Can&apos;t we just use Azure AI Foundry for this?&rdquo;
            </h2>
            <p className="text-sm text-gray-500 mb-6">The IT Director Objection</p>
            <blockquote className="border-l-2 border-purple-500 pl-5 py-2">
              <p className="text-gray-300 leading-relaxed">
                &ldquo;Azure is great for hosting the <strong className="text-white">&lsquo;brain.&rsquo;</strong> But as soon as that brain starts using <strong className="text-white">&lsquo;hands&rsquo;</strong> (tools), you have a governance gap.
              </p>
              <p className="text-gray-300 leading-relaxed mt-3">
                Azure doesn&apos;t know if a search tool costs $0.01 or $1.00 until the bill arrives. SatGate lets you set a budget on the &lsquo;hands&rsquo; directly.&rdquo;
              </p>
            </blockquote>
          </div>
        </div>
      </section>

      <section className="pb-20 px-6">
        <div className="max-w-3xl mx-auto rounded-xl border border-gray-800 bg-gray-950 p-6 sm:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">FAQ</p>
          <h2 className="mb-6 text-2xl font-bold text-white">Cloud-native AI governance questions</h2>
          <div className="space-y-5">
            {[
              ['Why are cloud-native AI governance tools not enough for autonomous agents?', 'Cloud-native tools are usually siloed to one provider and built around IAM, logs, and after-the-fact billing. Autonomous agents need request-path controls that follow tool calls across clouds, APIs, and MCP servers.'],
              ['How is SatGate different from AWS, Azure, or GCP AI governance?', 'SatGate is provider-neutral and enforces economic policy before upstream calls execute. It can cap per-agent, per-tool, and per-task spend across clouds instead of relying on one cloud billing system.'],
              ['Can SatGate work alongside cloud-native AI platforms?', 'Yes. SatGate can sit between agents and cloud-hosted models or tools, adding budget enforcement, cost attribution, and policy controls while the cloud platform continues to provide compute and model hosting.'],
            ].map(([question, answer]) => (
              <div key={question} className="border-t border-gray-800 pt-5 first:border-t-0 first:pt-0">
                <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                <p className="leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="pb-24 px-6">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Ready to{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 to-cyan-400">
              govern your agents?
            </span>
          </h2>
          <p className="text-gray-400 mb-10 max-w-lg mx-auto">
            Stop relying on cloud-native tools that weren&apos;t built for agentic AI. Start with SatGate in minutes.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="/roi-calculator"
              className="inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-gradient-to-r from-purple-600 to-pink-600 hover:from-purple-500 hover:to-pink-500 text-white font-semibold transition shadow-lg shadow-purple-900/30"
            >
              See the Savings <ArrowRight size={16} />
            </Link>
            <Link
              href="/economic-firewall-readiness-grader"
              className="inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-gray-800 hover:bg-gray-700 text-white font-semibold transition border border-gray-700"
            >
              Grade readiness <Shield size={16} />
            </Link>
            <a
              href="https://cloud.satgate.io/cloud/login"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-gray-800 hover:bg-gray-700 text-white font-semibold transition border border-gray-700"
            >
              Start Free <Zap size={16} />
            </a>
            <a
              href="mailto:contact@satgate.io"
              className="inline-flex items-center gap-2 px-6 py-3 rounded-lg text-gray-400 hover:text-white font-semibold transition"
            >
              Talk to Sales →
            </a>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-10 px-6 text-center text-gray-500 text-sm">
        <p>© {new Date().getFullYear()} SatGate. All rights reserved.</p>
      </footer>
    </div>
  );
};

export default CloudNativeComparisonPage;

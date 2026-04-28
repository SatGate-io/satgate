'use client';

import React, { useState, useMemo } from 'react';
import { Menu, X } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

const fmt = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });
const fmtPct = new Intl.NumberFormat('en-US', { style: 'percent', maximumFractionDigits: 0 });

interface SliderInputProps {
  label: string;
  value: number;
  min: number;
  max: number;
  step: number;
  format?: (v: number) => string;
  onChange: (v: number) => void;
}

function SliderInput({ label, value, min, max, step, format, onChange }: SliderInputProps) {
  const pct = ((value - min) / (max - min)) * 100;
  return (
    <div className="mb-6">
      <div className="flex items-center justify-between mb-2">
        <label className="text-sm font-medium text-gray-300">{label}</label>
        <span className="text-sm font-bold text-white tabular-nums">{format ? format(value) : value}</span>
      </div>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(parseFloat(e.target.value))}
        className="w-full h-2 rounded-full appearance-none cursor-pointer"
        style={{
          background: `linear-gradient(to right, #a855f7 0%, #06b6d4 ${pct}%, #374151 ${pct}%, #374151 100%)`,
        }}
      />
      <div className="flex justify-between text-xs text-gray-600 mt-1">
        <span>{format ? format(min) : min}</span>
        <span>{format ? format(max) : max}</span>
      </div>
    </div>
  );
}

export default function ROICalculatorPage() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [agents, setAgents] = useState(50);
  const [costPerCall, setCostPerCall] = useState(0.05);
  const [callsPerDay, setCallsPerDay] = useState(200);
  const [loopFreq, setLoopFreq] = useState(2);
  const [loopDuration, setLoopDuration] = useState(150);

  const calc = useMemo(() => {
    const monthlyToolSpend = agents * callsPerDay * costPerCall * 30;
    const monthlyLoopWaste = agents * callsPerDay * 30 * (loopFreq / 100) * loopDuration * costPerCall;
    const annualRisk = monthlyLoopWaste * 12;
    const satgateSavings = monthlyLoopWaste * 0.98;
    const paybackDays = satgateSavings > 0 ? (99 / satgateSavings) * 30 : Infinity;
    const annualROI = satgateSavings > 0 ? ((satgateSavings * 12 - 99 * 12) / (99 * 12)) : 0;
    return { monthlyToolSpend, monthlyLoopWaste, annualRisk, satgateSavings, paybackDays, annualROI };
  }, [agents, costPerCall, callsPerDay, loopFreq, loopDuration]);

  const maxBar = calc.monthlyToolSpend + calc.monthlyLoopWaste;
  const withoutSatgate = maxBar;
  const withSatgate = calc.monthlyToolSpend + (calc.monthlyLoopWaste * 0.02);
  const barMax = Math.max(withoutSatgate, 1);

  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'AI Agent ROI Calculator',
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/roi-calculator',
    description: 'Estimate ghost spend, runaway agent loop exposure, payback period, and ROI from request-path AI agent budget enforcement.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'How do you calculate AI agent ghost spend?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Ghost spend is estimated from active agents, tool calls per day, average cost per tool call, loop frequency, and the number of calls wasted before a loop is detected.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why do agent loops create API cost risk?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Autonomous agents can retry, delegate, and call paid tools faster than humans can notice. Without inline budget enforcement, dashboards and alerts usually detect the cost after it has already happened.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate reduce runaway AI agent spend?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate enforces per-agent, per-tool, per-route, and per-request budget policy before upstream API calls execute, blocking or routing requests that exceed economic policy.',
        },
      },
      {
        '@type': 'Question',
        name: 'What inputs do I need for the AI agent ROI calculator?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'You need the number of active agents, average cost per tool call, calls per agent per day, expected loop or error frequency, and average loop duration before discovery.',
        },
      },
      {
        '@type': 'Question',
        name: 'What should I do after estimating runaway agent spend?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Turn the exposure model into enforceable policy: generate budget limits, MCP tool caps, scoped capability-token policy, and request-path controls that block over-budget calls before cost is created.',
        },
      },
    ],
  };

  const howToJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'HowTo',
    name: 'How to estimate AI agent budget enforcement ROI',
    description: 'Use the SatGate ROI calculator to estimate unmanaged AI agent spend, runaway loop exposure, monthly savings, payback period, and annual ROI.',
    totalTime: 'PT3M',
    tool: [{ '@type': 'HowToTool', name: 'SatGate AI Agent ROI Calculator' }],
    step: [
      {
        '@type': 'HowToStep',
        name: 'Enter active agents',
        text: 'Set the number of autonomous agents, workflows, or delegated sub-agents that can call paid tools or APIs.',
      },
      {
        '@type': 'HowToStep',
        name: 'Set request economics',
        text: 'Enter average tool-call cost and daily calls per agent to estimate normal monthly tool spend.',
      },
      {
        '@type': 'HowToStep',
        name: 'Model loop exposure',
        text: 'Estimate loop frequency and average calls wasted before discovery to calculate ghost spend and annual risk exposure.',
      },
      {
        '@type': 'HowToStep',
        name: 'Review enforcement savings',
        text: 'Compare unmanaged cost exposure with SatGate request-path budget enforcement to estimate savings, payback period, and annual ROI.',
      },
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'AI Agent Cost Control', item: 'https://satgate.io/ai-agent-cost-control' },
      { '@type': 'ListItem', position: 3, name: 'AI Agent ROI Calculator', item: 'https://satgate.io/roi-calculator' },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans selection:bg-purple-500 selection:text-white">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(howToJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
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
            <Link href="/roi-calculator" className="text-white transition">ROI Calculator</Link>
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Cloud</a>
          </div>
          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="flex md:hidden items-center justify-center w-10 h-10 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 text-gray-400 hover:text-white transition"
            aria-label="Toggle menu"
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
              { href: '/roi-calculator', label: 'ROI Calculator' },
            ].map((l) => (
              <Link key={l.href} href={l.href} onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">{l.label}</Link>
            ))}
            <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Docs</a>
            <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" onClick={() => setMobileMenuOpen(false)} className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg">Cloud</a>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <header className="pt-32 pb-10 px-6 text-center">
        <div className="max-w-4xl mx-auto">
          <h1 className="text-4xl md:text-6xl font-extrabold tracking-tight mb-6">
            How Much Are{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-red-400 via-pink-400 to-purple-400">
              Agent Loops
            </span>{' '}
            Costing You?
          </h1>
          <p className="text-xl text-gray-400 max-w-2xl mx-auto leading-relaxed">
            Adjust the sliders below to see your hidden &ldquo;ghost spend&rdquo; — and how fast SatGate pays for itself.
          </p>
        </div>
      </header>

      {/* Calculator */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto grid grid-cols-1 lg:grid-cols-2 gap-10">
          {/* Inputs */}
          <div className="p-6 md:p-8 rounded-2xl bg-gray-900 border border-gray-800">
            <h2 className="text-lg font-bold text-white mb-6">Your Infrastructure</h2>
            <SliderInput label="Number of Active Agents" value={agents} min={1} max={500} step={1} onChange={setAgents} />
            <SliderInput label="Average Cost per Tool Call" value={costPerCall} min={0.01} max={1} step={0.01} format={(v) => `$${v.toFixed(2)}`} onChange={setCostPerCall} />
            <SliderInput label="Tool Calls per Agent per Day" value={callsPerDay} min={10} max={1000} step={10} onChange={setCallsPerDay} />
            <SliderInput label="Loop / Error Frequency" value={loopFreq} min={0.1} max={10} step={0.1} format={(v) => `${v.toFixed(1)}%`} onChange={setLoopFreq} />
            <SliderInput label="Avg Loop Duration (calls before discovery)" value={loopDuration} min={10} max={500} step={5} onChange={setLoopDuration} />
          </div>

          {/* Results */}
          <div className="space-y-6">
            {/* Metrics */}
            <div className="p-6 md:p-8 rounded-2xl bg-gray-900 border border-gray-800">
              <h2 className="text-lg font-bold text-white mb-6">Unmanaged Cost Exposure</h2>
              <div className="space-y-4">
                <div className="flex justify-between items-baseline">
                  <span className="text-gray-400 text-sm">Monthly Tool Spend</span>
                  <span className="text-white font-bold text-lg tabular-nums transition-all duration-300">{fmt.format(calc.monthlyToolSpend)}</span>
                </div>
                <div className="flex justify-between items-baseline">
                  <span className="text-gray-400 text-sm">Monthly &ldquo;Ghost Spend&rdquo;</span>
                  <span className="text-red-400 font-extrabold text-2xl tabular-nums transition-all duration-300">{fmt.format(calc.monthlyLoopWaste)}</span>
                </div>
                <div className="flex justify-between items-baseline">
                  <span className="text-gray-400 text-sm">Annual Risk Exposure</span>
                  <span className="text-red-500 font-extrabold text-3xl tabular-nums transition-all duration-300">{fmt.format(calc.annualRisk)}</span>
                </div>
                <hr className="border-gray-800" />
                <div className="flex justify-between items-baseline">
                  <span className="text-gray-400 text-sm">SatGate Monthly Savings</span>
                  <span className="text-green-400 font-extrabold text-2xl tabular-nums transition-all duration-300">{fmt.format(calc.satgateSavings)}</span>
                </div>
                <div className="flex justify-between items-baseline">
                  <span className="text-gray-400 text-sm">Payback Period</span>
                  <span className="text-cyan-400 font-bold text-lg tabular-nums transition-all duration-300">
                    {calc.paybackDays === Infinity ? '—' : calc.paybackDays < 1 ? '< 1 day' : `${Math.ceil(calc.paybackDays)} days`}
                  </span>
                </div>
                <div className="flex justify-between items-baseline">
                  <span className="text-gray-400 text-sm">Annual ROI</span>
                  <span className="text-green-400 font-extrabold text-2xl tabular-nums transition-all duration-300">
                    {calc.annualROI > 100 ? `${Math.round(calc.annualROI * 100).toLocaleString()}%` : fmtPct.format(calc.annualROI)}
                  </span>
                </div>
              </div>
            </div>

            {/* Bar Chart */}
            <div className="p-6 md:p-8 rounded-2xl bg-gray-900 border border-gray-800">
              <h2 className="text-lg font-bold text-white mb-6">Monthly Cost Comparison</h2>
              <div className="flex items-end gap-6 h-48">
                {/* Without SatGate */}
                <div className="flex-1 flex flex-col items-center gap-2 h-full justify-end">
                  <span className="text-xs text-gray-400 font-medium tabular-nums transition-all duration-300">{fmt.format(withoutSatgate)}</span>
                  <div
                    className="w-full rounded-t-lg transition-all duration-500 ease-out"
                    style={{
                      height: `${Math.max((withoutSatgate / barMax) * 100, 2)}%`,
                      background: 'linear-gradient(to top, #991b1b, #dc2626, #6b7280)',
                    }}
                  />
                  <span className="text-xs text-gray-500 text-center">Without SatGate</span>
                </div>
                {/* With SatGate */}
                <div className="flex-1 flex flex-col items-center gap-2 h-full justify-end">
                  <span className="text-xs text-gray-400 font-medium tabular-nums transition-all duration-300">{fmt.format(withSatgate)}</span>
                  <div
                    className="w-full rounded-t-lg transition-all duration-500 ease-out"
                    style={{
                      height: `${Math.max((withSatgate / barMax) * 100, 2)}%`,
                      background: 'linear-gradient(to top, #065f46, #10b981)',
                    }}
                  />
                  <span className="text-xs text-gray-500 text-center">With SatGate</span>
                </div>
              </div>
              {/* Savings label */}
              <div className="mt-4 text-center">
                <span className="inline-block px-4 py-2 rounded-full bg-green-900/30 border border-green-800/40 text-green-400 text-sm font-bold tabular-nums transition-all duration-300">
                  Your Savings: {fmt.format(withoutSatgate - withSatgate)}/mo
                </span>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Buyer Intent Links */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto rounded-2xl border border-gray-800 bg-gray-900/60 p-6 md:p-8">
          <div className="mb-6 max-w-3xl">
            <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Go deeper</p>
            <h2 className="mb-3 text-2xl md:text-3xl font-bold text-white">Turn the ROI model into enforceable policy</h2>
            <p className="text-gray-400">
              The calculator shows the exposure. These guides show how to enforce budget limits before agents spend the money.
            </p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {[
              { href: '/ai-agent-cost-control', title: 'AI agent cost control', body: 'Control API spend before autonomous agents create runaway bills.' },
              { href: '/economic-firewall', title: 'Economic firewall', body: 'The request-path control layer for Observe, Control, and Charge.' },
              { href: '/mcp-governance', title: 'MCP budget enforcement', body: 'Assign per-tool costs and cap MCP tool spend in real time.' },
              { href: '/agent-api-governance', title: 'Agent API governance', body: 'Replace unlimited API keys with scoped, revocable capabilities.' },
              { href: '/revocable-capability-token-policy-template', title: 'Capability-token policy template', body: 'Generate scoped, expiring, revocable agent authority with budgets and audit fields.' },
              { href: '/agent-spend-policy-template', title: 'Agent spend policy template', body: 'Convert ROI exposure into copyable YAML and JSON budget enforcement policy.' },
              { href: '/runaway-agent-cost-calculator', title: 'Runaway agent cost calculator', body: 'Model loop, retry, fanout, and MCP tool-call exposure.' },
            ].map((item) => (
              <Link key={item.href} href={item.href} className="block rounded-xl border border-gray-800 bg-black/40 p-5 transition hover:border-cyan-500/40 hover:bg-cyan-950/20">
                <h3 className="mb-2 font-bold text-white">{item.title}</h3>
                <p className="text-sm text-gray-400">{item.body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* ROI assumptions */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">ROI assumptions</p>
          <h2 className="mb-4 text-2xl md:text-3xl font-bold text-white">How the calculator turns agent activity into budget-enforcement ROI</h2>
          <div className="grid gap-5 md:grid-cols-3">
            <div className="rounded-xl border border-gray-800 bg-black/40 p-5">
              <h3 className="mb-2 font-bold text-white">Normal monthly spend</h3>
              <p className="text-sm leading-relaxed text-gray-400">Agents × calls per day × average cost per call × 30 days.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-black/40 p-5">
              <h3 className="mb-2 font-bold text-white">Ghost spend exposure</h3>
              <p className="text-sm leading-relaxed text-gray-400">Normal call volume × loop/error frequency × wasted calls before discovery.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-black/40 p-5">
              <h3 className="mb-2 font-bold text-white">SatGate savings model</h3>
              <p className="text-sm leading-relaxed text-gray-400">Request-path budget enforcement blocks most loop waste before upstream APIs or MCP tools execute.</p>
            </div>
          </div>
        </div>
      </section>

      {/* SEO explainer */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="rounded-2xl border border-gray-800 bg-gray-900/60 p-6 md:p-8">
            <h2 className="text-2xl font-bold text-white mb-4">What this AI agent cost calculator measures</h2>
            <p className="text-gray-400 leading-relaxed mb-4">
              Most LLM cost dashboards measure known spend: tokens, requests, and invoices after the fact. This calculator focuses on avoidable agent-loop exposure: the API and tool spend created when autonomous agents retry, delegate, call MCP tools, or continue a task after the economics no longer make sense.
            </p>
            <p className="text-gray-400 leading-relaxed">
              The model is intentionally simple: agent count × daily tool calls × cost per call × loop frequency × loop duration. It gives finance, platform, and security teams a shared number for the cost of missing inline budget enforcement.
            </p>
          </div>
          <div className="rounded-2xl border border-gray-800 bg-gray-900/60 p-6 md:p-8">
            <h2 className="text-2xl font-bold text-white mb-4">How SatGate changes the result</h2>
            <p className="text-gray-400 leading-relaxed mb-4">
              SatGate does not wait for a billing export. It checks agent identity, route, tool, request cost, remaining budget, and policy before forwarding the call. That is why the savings estimate assumes most loop waste is prevented rather than merely reported.
            </p>
            <p className="text-gray-400 leading-relaxed">
              Start in Observe mode to measure real spend, then move high-risk routes to Control mode when you are ready to enforce hard budget limits.
            </p>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-2xl md:text-3xl font-bold mb-3">Start your free Shadow Audit</h2>
          <p className="text-lg text-gray-400 mb-8 max-w-xl mx-auto">
            See real numbers from your own infrastructure — not estimates.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="/design-partners"
              className="inline-block bg-gradient-to-r from-purple-600 to-cyan-600 text-white px-10 py-4 rounded-full font-bold text-lg hover:opacity-90 transition shadow-lg shadow-purple-500/20"
            >
              Start Free Shadow Audit →
            </Link>
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
                <li><Link href="/roi-calculator" className="hover:text-white transition">ROI Calculator</Link></li>
                <li><Link href="/llm-cost-dashboard" className="hover:text-white transition">LLM Cost Dashboard</Link></li>
                <li><Link href="/llm-cost-monitoring" className="hover:text-white transition">LLM Cost Monitoring</Link></li>
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

      <style jsx>{`
        input[type='range']::-webkit-slider-thumb {
          -webkit-appearance: none;
          appearance: none;
          width: 20px;
          height: 20px;
          border-radius: 50%;
          background: white;
          cursor: pointer;
          box-shadow: 0 0 6px rgba(168, 85, 247, 0.5);
        }
        input[type='range']::-moz-range-thumb {
          width: 20px;
          height: 20px;
          border-radius: 50%;
          background: white;
          cursor: pointer;
          border: none;
          box-shadow: 0 0 6px rgba(168, 85, 247, 0.5);
        }
      `}</style>
    </div>
  );
}

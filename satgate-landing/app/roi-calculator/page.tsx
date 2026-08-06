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
  const roiExamples = [
    {
      title: 'MCP support agent team',
      agents: 25,
      callsPerDay: 180,
      costPerCall: '$0.04',
      loopFrequency: '1.5%',
      wastedCalls: 120,
      monthlyAvoidableSpend: '$3,888',
      payback: '< 1 day',
      control: 'Per-agent MCP tool caps and deny receipts',
    },
    {
      title: 'AI voice agent support desk',
      agents: 80,
      callsPerDay: 260,
      costPerCall: '$0.06',
      loopFrequency: '1.2%',
      wastedCalls: 140,
      monthlyAvoidableSpend: '$12,579',
      payback: '< 1 day',
      control: 'Voice-agent tool budgets, route caps, and chargeback receipts',
    },
    {
      title: 'Enterprise coding-agent rollout',
      agents: 150,
      callsPerDay: 320,
      costPerCall: '$0.08',
      loopFrequency: '0.8%',
      wastedCalls: 180,
      monthlyAvoidableSpend: '$16,589',
      payback: '< 1 day',
      control: 'Route budgets, scoped capability tokens, and Evidence Pack exports',
    },
    {
      title: 'External robot-customer API',
      agents: 60,
      callsPerDay: 500,
      costPerCall: '$0.03',
      loopFrequency: '2.0%',
      wastedCalls: 90,
      monthlyAvoidableSpend: '$4,860',
      payback: '< 1 day',
      control: 'Request-path budget enforcement before paid upstream access',
    },
  ];

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'AI Agent ROI Calculator',
    description: 'Calculate AI agent ROI, voice agent payback, chargeback exposure, loop waste, budget-control priorities, and request-path enforcement savings with Policy-to-Proof evidence.',
    url: 'https://satgate.io/roi-calculator',
    dateModified: '2026-08-06',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'AI agent ROI calculator' },
      { '@type': 'Thing', name: 'AI voice agent ROI calculator' },
      { '@type': 'Thing', name: 'AI agent ROI formula' },
      { '@type': 'Thing', name: 'agent ROI calculator' },
      { '@type': 'Thing', name: 'voice agent payback calculator' },
      { '@type': 'Thing', name: 'AI agent payback period' },
      { '@type': 'Thing', name: 'AI chargeback ROI calculator' },
      { '@type': 'Thing', name: 'runaway AI agent loop exposure' },
      { '@type': 'Thing', name: 'request-path budget enforcement ROI' },
      { '@type': 'Thing', name: 'Policy-to-Proof receipts' },
      { '@type': 'Thing', name: 'MCP tool spend risk' },
      { '@type': 'Thing', name: 'economic firewall payback period' },
    ],
  };

  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'AI Agent ROI Calculator',
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/roi-calculator',
    description: webPageJsonLd.description,
    featureList: [
      'AI voice agent ROI calculator',
      'AI agent payback period model',
      'Chargeback exposure estimate',
      'Monthly AI agent tool spend estimate',
      'Runaway loop ghost spend exposure',
      'Annual AI agent cost-risk model',
      'ROI-to-enforcement priority model',
      'Request-path budget enforcement savings estimate',
      'Payback period and annual ROI estimate',
      'Links from ROI exposure to Policy-to-Proof controls and Evidence Pack receipts',
    ],
    audience: [
      { '@type': 'Audience', audienceType: 'Platform engineering teams' },
      { '@type': 'Audience', audienceType: 'Security teams governing autonomous agents' },
      { '@type': 'Audience', audienceType: 'Finance teams managing AI spend risk' },
      { '@type': 'Audience', audienceType: 'AI engineering teams deploying paid MCP tools and APIs' },
    ],
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-08-06',
    about: webPageJsonLd.about,
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'How do you calculate AI voice agent ROI?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'AI voice agent ROI can be estimated by comparing avoided support labor and automated resolution value against voice-agent model, telephony, tool, and failure-loop costs, then subtracting the cost of request-path budget enforcement.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does chargeback affect AI agent ROI?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Chargeback improves AI agent ROI analysis by attributing model, API, MCP tool, and workflow spend to the agent, tenant, route, policy, and business owner that caused the cost.',
        },
      },
      {
        '@type': 'Question',
        name: 'What is the payback period for AI agent budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Payback period is the enforcement cost divided by monthly avoided ghost spend. For high-volume agents or paid-tool workflows, the payback period can be measured in days when one prevented loop offsets the control cost.',
        },
      },
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
        name: 'What is the AI agent ROI formula?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'AI agent ROI can be estimated as annual avoided ghost spend minus annual enforcement cost, divided by annual enforcement cost. Ghost spend is active agents multiplied by daily tool calls, average cost per call, loop frequency, wasted calls before discovery, and 30 days.',
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
          text: 'SatGate enforces per-agent, per-tool, per-route, and per-request budget policy at the gateway before forwarding to upstream APIs, blocking or routing requests that exceed economic policy.',
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
          text: 'Turn the exposure model into Policy-to-Proof controls: define authority, budget limits, MCP tool caps, scoped capability-token policy, receipts, and Evidence Pack exports.',
        },
      },
      {
        '@type': 'Question',
        name: 'Is this different from an LLM token cost calculator?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. A token cost calculator estimates model usage. This ROI calculator estimates autonomous agent spend risk across paid tools, APIs, MCP calls, retries, delegation, and loops that may happen outside a single LLM invoice.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do ROI results become evidence?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Use Policy-to-Proof to connect budget policy, request decisions, receipts, and Evidence Pack exports for audit-ready agent governance.',
        },
      },
      {
        '@type': 'Question',
        name: 'What is a good payback period for AI agent budget enforcement?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'For agentic systems with paid tool access, payback can be measured in days when a small number of runaway loops or expensive MCP calls would exceed the monthly cost of request-path budget enforcement.',
        },
      },
      {
        '@type': 'Question',
        name: 'Who should use an AI agent ROI calculator?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Platform, security, finance, and AI engineering teams should use it before giving autonomous agents access to paid APIs, model providers, MCP tools, data services, or external agent marketplaces.',
        },
      },
      {
        '@type': 'Question',
        name: 'What controls should you fund first after calculating AI agent ROI?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Fund the controls closest to cost creation first: per-agent and per-tool budgets, pre-request allow or deny decisions, revocation before the next request, and Evidence Pack receipts for denied, downgraded, paid, and delegated calls.',
        },
      },
    ],
  };

  const howToJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'HowTo',
    name: 'How to estimate AI agent budget enforcement ROI',
    description: 'Use the SatGate ROI calculator to estimate unmanaged AI agent loop exposure, budget-control ROI, and Policy-to-Proof receipt coverage.',
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
        text: 'Estimate loop frequency and average calls wasted before enforcement to calculate avoidable spend and annual risk exposure.',
      },
      {
        '@type': 'HowToStep',
        name: 'Review enforcement savings',
        text: 'Compare unmanaged cost exposure with SatGate request-path budget enforcement to estimate savings, payback period, and annual ROI.',
      },
    ],
  };

  const scenariosJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent budget enforcement ROI scenarios',
    description: 'Common autonomous agent spend-risk scenarios modeled by the SatGate ROI calculator.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Runaway MCP tool loop',
        description: 'An agent repeatedly calls a paid MCP tool until request-path budget enforcement blocks the loop.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Delegated sub-agent fanout',
        description: 'A parent agent delegates work to sub-agents that multiply API calls and spend without per-agent caps.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'AI voice agent support desk',
        description: 'A voice agent can create ROI when successful automations exceed model, telephony, tool, escalation, and loop waste costs.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Paid API retry storm',
        description: 'A workflow retries failed or low-confidence calls against billable APIs before humans notice the cost.',
      },
    ],
  };

  const breakEvenJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent budget enforcement break-even examples',
    description: 'Examples that show when request-path budget enforcement can pay for itself by enforcing authority, budget, and receipt policy before agent execution.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Small team with one paid tool loop',
        description: 'A few dozen agents using a paid API can justify enforcement when one retry loop would exceed a monthly gateway subscription.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Enterprise MCP deployment',
        description: 'Hundreds of agents calling MCP tools need per-tool caps because provider dashboards miss spend created outside LLM token invoices.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'External agent access',
        description: 'APIs exposed to external agents need authority, budget, and receipt policy before access is granted.',
      },
    ],
  };

  const enforcementDecisionJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent ROI to budget enforcement decision model',
    description: 'How to turn AI agent ROI calculator output into the first request-path enforcement controls to fund.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'High ghost spend',
        description: 'Prioritize hard per-agent, per-tool, and per-route budget caps before paid API or MCP requests execute.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'High loop frequency',
        description: 'Prioritize loop detection, structured denials, downgrade paths, and revocation triggers before retry storms create cost.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'High delegation fanout',
        description: 'Prioritize scoped capability tokens, child-agent budget attenuation, delegation-chain attribution, and receipt export.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Short payback period',
        description: 'Move the highest-risk routes from Observe to Control mode and preserve Evidence Packs for finance, security, and audit review.',
      },
    ],
  };

  const formulaJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: 'AI agent ROI formula',
    description: 'Formula steps for converting autonomous agent activity into ghost spend, avoided spend, payback period, and ROI for request-path budget enforcement.',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Normal monthly tool spend',
        description: 'Active agents x daily tool calls x average cost per call x 30 days.',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Monthly ghost spend',
        description: 'Normal monthly call volume x loop frequency x wasted calls before discovery x average cost per call.',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: 'Annual avoided spend',
        description: 'Monthly ghost spend prevented by request-path budget enforcement x 12 months.',
      },
      {
        '@type': 'ListItem',
        position: 4,
        name: 'Annual ROI',
        description: '(Annual avoided spend - annual enforcement cost) / annual enforcement cost.',
      },
      {
        '@type': 'ListItem',
        position: 5,
        name: 'AI voice agent ROI',
        description: '(Avoided support labor + automated resolution value - voice-agent operating cost - enforcement cost) / enforcement cost.',
      },
      {
        '@type': 'ListItem',
        position: 6,
        name: 'Chargeback ROI',
        description: 'Attributed agent spend recovered or prevented through tenant, workflow, route, model, tool, policy, and receipt-level accountability.',
      },
    ],
  };

  const exampleResultsJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'Dataset',
    name: 'AI agent budget enforcement ROI example results',
    description: 'Representative examples for estimating runaway AI agent spend, avoided monthly waste, payback period, and the SatGate controls that map ROI to proof.',
    url: 'https://satgate.io/roi-calculator',
    creator: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-08-06',
    variableMeasured: [
      'active agents',
      'tool calls per agent per day',
      'average cost per tool call',
      'loop frequency',
      'wasted calls before discovery',
      'monthly avoidable spend',
      'payback period',
      'AI voice agent ROI',
      'chargeback exposure',
      'request-path enforcement control',
    ],
    distribution: {
      '@type': 'DataDownload',
      encodingFormat: 'text/html',
      contentUrl: 'https://satgate.io/roi-calculator',
    },
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
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(howToJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(scenariosJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breakEvenJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(enforcementDecisionJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(formulaJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(exampleResultsJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      {/* Navigation */}
      <nav className="border-b border-gray-800 backdrop-blur-md fixed w-full z-50 bg-black/50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2 shrink-0">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="w-7 h-7 sm:w-8 sm:h-8" />
            <span className="text-lg sm:text-xl font-bold text-white whitespace-nowrap">SatGate<sup className="text-xs font-normal">™</sup></span>
          </Link>
          <div className="hidden md:flex gap-6 text-sm font-medium text-gray-400">
            <Link href="/govern" className="hover:text-white transition">Govern</Link>
            <Link href="/policy-to-proof" className="hover:text-white transition">Policy-to-Proof</Link>
            <Link href="/tools" className="hover:text-white transition">Tools</Link>
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
              { href: '/govern', label: 'Govern' },
              { href: '/policy-to-proof', label: 'Policy-to-Proof' },
              { href: '/tools', label: 'Tools' },
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
            AI Agent ROI Calculator for{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-red-400 via-pink-400 to-purple-400">
              Payback and Ghost Spend
            </span>{' '}
          </h1>
          <p className="text-xl text-gray-400 max-w-2xl mx-auto leading-relaxed">
            Model AI agent ROI, voice agent payback, chargeback exposure, loop waste, and how fast request-path budget enforcement pays for itself.
          </p>
        </div>
      </header>

      <section className="pb-16 px-6">
        <div className="max-w-5xl mx-auto rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">Direct answer</p>
          <h2 className="mb-4 text-2xl md:text-3xl font-bold text-white">AI agent ROI is avoided waste plus automation value minus control cost</h2>
          <p className="mb-6 max-w-3xl text-gray-400 leading-relaxed">
            For platform and finance teams, the useful AI agent ROI formula is: annual avoided ghost spend plus automation value minus annual enforcement cost, divided by annual enforcement cost. Voice-agent ROI uses the same structure, but adds telephony, escalation, call-resolution value, and support-labor displacement.
          </p>
          <div className="grid gap-4 md:grid-cols-3">
            {[
              ['AI agent ROI formula', '(Annual avoided waste + automation value - annual control cost) / annual control cost.'],
              ['AI voice agent ROI', '(Resolved-call value + avoided support labor - model, voice, tool, escalation, and control costs) / control cost.'],
              ['Chargeback ROI', 'Attributed spend recovered or prevented by tenant, workflow, route, model, MCP tool, policy version, and receipt.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black/40 p-5">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

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
            <h2 className="mb-3 text-2xl md:text-3xl font-bold text-white">Turn the ROI model into Policy-to-Proof</h2>
            <p className="text-gray-400">
              The calculator shows the exposure. SatGate maps it to enforceable authority checks, receipts, and an Evidence Pack before agents execute.
            </p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {[
              { href: '/policy-to-proof', title: 'Map ROI to Policy-to-Proof', body: 'Connect exposure to enforceable checks, receipts, and an Evidence Pack.' },
              { href: '/govern', title: 'Govern agent execution', body: 'Put authority, budget, and audit policy in the request path before execution.' },
              { href: '/agent-spend-policy-template', title: 'Generate budget policy', body: 'Convert ROI exposure into YAML and JSON budget policy with receipt fields.' },
            ].map((item) => (
              <Link key={item.href} href={item.href} className="block rounded-xl border border-gray-800 bg-black/40 p-5 transition hover:border-cyan-500/40 hover:bg-cyan-950/20">
                <h3 className="mb-2 font-bold text-white">{item.title}</h3>
                <p className="text-sm text-gray-400">{item.body}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* Enforcement priority */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Enforcement priority</p>
          <h2 className="mb-4 text-2xl md:text-3xl font-bold text-white">Use ROI to decide which agent budget controls to fund first</h2>
          <p className="mb-8 max-w-3xl text-gray-400 leading-relaxed">
            The useful output is not just a payback number. It is a ranked enforcement plan: where spend is created, which request is still stoppable, and what proof finance and security will need afterward.
          </p>
          <div className="grid gap-5 md:grid-cols-2">
            {[
              ['High ghost spend', 'Start with hard per-agent, per-tool, and per-route budget caps before paid API or MCP requests execute.'],
              ['High loop frequency', 'Add loop detection, structured denials, downgrade paths, and revocation triggers before retries create cost.'],
              ['High delegation fanout', 'Use scoped capability tokens, child-agent budget attenuation, delegation-chain attribution, and receipt export.'],
              ['Short payback period', 'Move the highest-risk routes from Observe to Control mode and preserve Evidence Packs for review.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black/40 p-5">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
          <div className="mt-6 flex flex-wrap gap-3 text-sm font-semibold">
            <Link href="/ai-agent-cost-control" className="text-cyan-300 hover:text-cyan-200">AI agent cost control →</Link>
            <Link href="/mcp-tool-cost-policy-generator" className="text-cyan-300 hover:text-cyan-200">MCP tool policy generator →</Link>
            <Link href="/policy-to-proof" className="text-cyan-300 hover:text-cyan-200">See Policy-to-Proof →</Link>
          </div>
        </div>
      </section>

      {/* ROI assumptions */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto rounded-2xl border border-purple-900/50 bg-purple-950/10 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-purple-300">ROI assumptions</p>
          <h2 className="mb-4 text-2xl md:text-3xl font-bold text-white">AI agent ROI formula and assumptions</h2>
          <p className="mb-8 max-w-3xl text-gray-400 leading-relaxed">
            The formula translates autonomous agent activity into a business case: estimate normal spend, isolate ghost spend from loops or retries, then compare avoided annual waste with request-path enforcement cost.
          </p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
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
              <p className="text-sm leading-relaxed text-gray-400">Request-path budget enforcement denies over-budget requests at the gateway before forwarding to upstream APIs or MCP tools.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-black/40 p-5">
              <h3 className="mb-2 font-bold text-white">Annual ROI</h3>
              <p className="text-sm leading-relaxed text-gray-400">(Annual avoided ghost spend - annual enforcement cost) / annual enforcement cost.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-black/40 p-5">
              <h3 className="mb-2 font-bold text-white">Voice agent ROI</h3>
              <p className="text-sm leading-relaxed text-gray-400">(Resolved-call value + avoided support labor - model, voice, tool, escalation, and control costs) / control cost.</p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-black/40 p-5">
              <h3 className="mb-2 font-bold text-white">Chargeback ROI</h3>
              <p className="text-sm leading-relaxed text-gray-400">Attributed spend recovered or prevented by tenant, workflow, route, model, tool, policy version, and receipt.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Risk scenarios */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto rounded-2xl border border-red-900/50 bg-red-950/10 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-red-300">Risk scenarios</p>
          <h2 className="mb-4 text-2xl md:text-3xl font-bold text-white">Where runaway agent ROI usually comes from</h2>
          <p className="mb-8 max-w-3xl text-gray-400 leading-relaxed">
            The model is most useful when teams connect it to a concrete failure mode. These are the three agent-spend patterns that usually make request-path budget enforcement pay back fastest.
          </p>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              ['Runaway MCP tool loop', 'An agent repeatedly calls a paid tool, data source, browser action, cloud task, or SaaS operation until a budget check stops it.'],
              ['Delegated sub-agent fanout', 'A parent agent creates sub-agents that multiply model, API, and tool calls faster than team-level budgets can explain.'],
              ['Paid API retry storm', 'A workflow retries failed or low-confidence calls against billable APIs, turning an exception path into a hidden invoice.'],
              ['AI voice agent escalation loop', 'A voice agent repeats tool calls, escalations, summaries, or retries across calls until per-workflow budgets stop the waste.'],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl border border-gray-800 bg-black/40 p-5">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Break-even examples */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto rounded-2xl border border-green-900/50 bg-green-950/10 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-green-300">Break-even examples</p>
          <h2 className="mb-4 text-2xl md:text-3xl font-bold text-white">When request-path budget enforcement pays for itself</h2>
          <p className="mb-8 max-w-3xl text-gray-400 leading-relaxed">
            The calculator is most persuasive when it ties avoided waste to a specific operating model: internal agents, MCP tools, or externally exposed agent access.
          </p>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              ['Small team with one paid tool loop', 'A few dozen agents using a paid API can justify enforcement when one retry loop would exceed a monthly gateway subscription.', '/policy-to-proof'],
              ['Enterprise MCP deployment', 'Hundreds of agents calling MCP tools need per-tool caps because provider dashboards miss spend outside LLM token invoices.', '/govern'],
              ['External agent access', 'Externally exposed agent access needs scoped authority, budget checks, and receipts before execution.', '/policy-to-proof'],
            ].map(([title, body, href]) => (
              <Link key={title} href={href} className="rounded-xl border border-gray-800 bg-black/40 p-5 transition hover:border-green-500/50 hover:bg-green-950/20">
                <h3 className="mb-2 font-bold text-white">{title}</h3>
                <p className="mb-4 text-sm leading-relaxed text-gray-400">{body}</p>
                <span className="text-sm font-semibold text-green-300">Map this to proof →</span>
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* Example scenarios */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto rounded-2xl border border-gray-800 bg-gray-900/60 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">Example results</p>
          <h2 className="mb-4 text-2xl md:text-3xl font-bold text-white">AI agent budget enforcement ROI benchmarks</h2>
          <p className="mb-8 max-w-3xl text-gray-400 leading-relaxed">
            Use these examples as starting points when finance, platform, and security teams need a shared estimate before real traffic is available in Observe mode.
          </p>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[760px] text-left text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-xs uppercase tracking-wide text-gray-500">
                  <th className="py-3 pr-4 font-semibold">Scenario</th>
                  <th className="py-3 pr-4 font-semibold">Inputs</th>
                  <th className="py-3 pr-4 font-semibold">Avoidable spend</th>
                  <th className="py-3 pr-4 font-semibold">SatGate control</th>
                </tr>
              </thead>
              <tbody>
                {roiExamples.map((example) => (
                  <tr key={example.title} className="border-b border-gray-800/70 align-top">
                    <td className="py-4 pr-4 font-bold text-white">{example.title}</td>
                    <td className="py-4 pr-4 text-gray-400">
                      {example.agents} agents, {example.callsPerDay} calls/day, {example.costPerCall}/call, {example.loopFrequency} loop frequency, {example.wastedCalls} wasted calls
                    </td>
                    <td className="py-4 pr-4">
                      <span className="block font-bold text-green-300">{example.monthlyAvoidableSpend}/mo</span>
                      <span className="text-gray-500">Payback: {example.payback}</span>
                    </td>
                    <td className="py-4 pr-4 text-gray-400">{example.control}</td>
                  </tr>
                ))}
              </tbody>
            </table>
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

      {/* FAQ */}
      <section className="pb-20 px-6">
        <div className="max-w-5xl mx-auto rounded-2xl border border-cyan-900/50 bg-cyan-950/10 p-6 md:p-8">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="mb-6 text-2xl md:text-3xl font-bold text-white">AI agent ROI calculator questions</h2>
          <div className="space-y-6">
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">How do you calculate AI voice agent ROI?</h3>
              <p className="text-gray-400 leading-relaxed">
                AI voice agent ROI can be estimated by comparing avoided support labor and automated resolution value against voice-agent model, telephony, tool, and failure-loop costs, then subtracting the cost of request-path budget enforcement.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">How does chargeback affect AI agent ROI?</h3>
              <p className="text-gray-400 leading-relaxed">
                Chargeback improves AI agent ROI analysis by attributing model, API, MCP tool, and workflow spend to the agent, tenant, route, policy, and business owner that caused the cost.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">What is the payback period for AI agent budget enforcement?</h3>
              <p className="text-gray-400 leading-relaxed">
                Payback period is the enforcement cost divided by monthly avoided ghost spend. For high-volume agents or paid-tool workflows, the payback period can be measured in days when one prevented loop offsets the control cost.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">How do you calculate AI agent ghost spend?</h3>
              <p className="text-gray-400 leading-relaxed">
                Ghost spend is estimated from active agents, tool calls per day, average cost per tool call, loop frequency, and the number of calls wasted before a loop is detected.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">What is the AI agent ROI formula?</h3>
              <p className="text-gray-400 leading-relaxed">
                AI agent ROI can be estimated as annual avoided ghost spend minus annual enforcement cost, divided by annual enforcement cost. Ghost spend is active agents multiplied by daily tool calls, average cost per call, loop frequency, wasted calls before discovery, and 30 days.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">Why do agent loops create API cost risk?</h3>
              <p className="text-gray-400 leading-relaxed">
                Autonomous agents can retry, delegate, and call paid tools faster than humans can notice. Without inline budget enforcement, dashboards and alerts usually detect the cost after it has already happened.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">How does SatGate reduce runaway AI agent spend?</h3>
              <p className="text-gray-400 leading-relaxed">
                SatGate enforces per-agent, per-tool, per-route, and per-request budget policy at the gateway before forwarding to upstream APIs, blocking or routing requests that exceed economic policy.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">What inputs do I need for the AI agent ROI calculator?</h3>
              <p className="text-gray-400 leading-relaxed">
                You need the number of active agents, average cost per tool call, calls per agent per day, expected loop or error frequency, and average loop duration before discovery.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">What should I do after estimating runaway agent spend?</h3>
              <p className="text-gray-400 leading-relaxed">
                Turn the exposure model into Policy-to-Proof controls: define authority, budget limits, MCP tool caps, scoped capability-token policy, receipts, and Evidence Pack exports.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">Is this different from an LLM token cost calculator?</h3>
              <p className="text-gray-400 leading-relaxed">
                Yes. A token cost calculator estimates model usage. This ROI calculator estimates autonomous agent spend risk across paid tools, APIs, MCP calls, retries, delegation, and loops that may happen outside a single LLM invoice.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">What is a good payback period for AI agent budget enforcement?</h3>
              <p className="text-gray-400 leading-relaxed">
                For agentic systems with paid tool access, payback can be measured in days when a small number of runaway loops or expensive MCP calls would exceed the monthly cost of request-path budget enforcement.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">Who should use an AI agent ROI calculator?</h3>
              <p className="text-gray-400 leading-relaxed">
                Platform, security, finance, and AI engineering teams should use it before giving autonomous agents access to paid APIs, model providers, MCP tools, data services, or external agent marketplaces.
              </p>
            </div>
            <div>
              <h3 className="mb-2 text-xl font-bold text-white">What controls should you fund first after calculating AI agent ROI?</h3>
              <p className="text-gray-400 leading-relaxed">
                Fund the controls closest to cost creation first: per-agent and per-tool budgets, pre-request allow or deny decisions, revocation before the next request, and Evidence Pack receipts for denied, downgraded, paid, and delegated calls.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-20 px-6 border-t border-gray-800">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-2xl md:text-3xl font-bold mb-3">Turn this ROI model into proof</h2>
          <p className="text-lg text-gray-400 mb-8 max-w-xl mx-auto">
            SatGate checks authority before execution, records every policy decision as a receipt, and packages the evidence for review.
          </p>
          <p className="text-gray-500 mb-8 max-w-xl mx-auto">
            Every agent action leaves a receipt, so a 90-second Evidence Pack cut can show the budget decision behind the ROI number.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="/govern"
              className="inline-block bg-gradient-to-r from-purple-600 to-cyan-600 text-white px-10 py-4 rounded-full font-bold text-lg hover:opacity-90 transition shadow-lg shadow-purple-500/20"
            >
              Govern agent actions →
            </Link>
            <Link
              href="/policy-to-proof"
              className="inline-block border border-gray-700 text-gray-300 px-10 py-4 rounded-full font-bold text-lg hover:border-gray-500 hover:bg-gray-800 transition"
            >
              See Policy-to-Proof
            </Link>
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

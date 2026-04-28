'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { ArrowRight, Bot, Gauge, ReceiptText, ShieldCheck, Zap } from 'lucide-react';

const money = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });
const number = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 });

function Slider({ label, value, min, max, step, suffix = '', prefix = '', onChange }: {
  label: string;
  value: number;
  min: number;
  max: number;
  step: number;
  suffix?: string;
  prefix?: string;
  onChange: (value: number) => void;
}) {
  const pct = ((value - min) / (max - min)) * 100;
  return (
    <label className="block rounded-xl border border-gray-800 bg-black/50 p-5">
      <div className="mb-3 flex items-center justify-between gap-4">
        <span className="font-medium text-gray-300">{label}</span>
        <span className="font-mono font-bold text-white">{prefix}{value.toLocaleString()}{suffix}</span>
      </div>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
        className="h-2 w-full appearance-none rounded-full"
        style={{ background: `linear-gradient(to right, #f97316 0%, #facc15 ${pct}%, #374151 ${pct}%, #374151 100%)` }}
      />
    </label>
  );
}

export default function RunawayAgentCostCalculatorPage() {
  const [agents, setAgents] = useState(25);
  const [callsPerMinute, setCallsPerMinute] = useState(12);
  const [costPerCallCents, setCostPerCallCents] = useState(8);
  const [loopMinutes, setLoopMinutes] = useState(45);
  const [fanout, setFanout] = useState(3);
  const [incidentsPerMonth, setIncidentsPerMonth] = useState(4);

  const calc = useMemo(() => {
    const costPerCall = costPerCallCents / 100;
    const callsPerIncident = agents * callsPerMinute * loopMinutes * fanout;
    const incidentCost = callsPerIncident * costPerCall;
    const monthlyExposure = incidentCost * incidentsPerMonth;
    const annualExposure = monthlyExposure * 12;
    const blockedAtFiveMinutes = agents * callsPerMinute * 5 * fanout * costPerCall;
    const savingsPerIncident = Math.max(incidentCost - blockedAtFiveMinutes, 0);
    return { callsPerIncident, incidentCost, monthlyExposure, annualExposure, blockedAtFiveMinutes, savingsPerIncident };
  }, [agents, callsPerMinute, costPerCallCents, fanout, incidentsPerMonth, loopMinutes]);

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'Runaway Agent Cost Calculator',
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/runaway-agent-cost-calculator',
    description: 'Estimate runaway AI agent loop costs from agent count, calls per minute, tool-call cost, loop duration, delegation fanout, and incident frequency.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is runaway agent spend?',
        acceptedAnswer: { '@type': 'Answer', text: 'Runaway agent spend is API, model, or tool cost created when autonomous agents loop, retry, delegate, or keep calling paid resources after the work is no longer economically justified.' },
      },
      {
        '@type': 'Question',
        name: 'Why can AI agent loops get expensive so quickly?',
        acceptedAnswer: { '@type': 'Answer', text: 'Agents can call tools at machine speed, fan out work to sub-agents, and retry failed steps without waiting for a human to approve each paid request.' },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate reduce runaway agent costs?',
        acceptedAnswer: { '@type': 'Answer', text: 'SatGate enforces request-path budgets, per-tool spend caps, revocation, and route policy before upstream API or MCP tool calls execute.' },
      },
    ],
  };


  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'AI Agent Cost Control', item: 'https://satgate.io/ai-agent-cost-control' },
      { '@type': 'ListItem', position: 3, name: 'Runaway Agent Cost Calculator', item: 'https://satgate.io/runaway-agent-cost-calculator' },
    ],
  };
  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_0%,rgba(249,115,22,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(250,204,21,0.15),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-orange-500/30 bg-orange-950/20 px-4 py-2 text-sm text-orange-200">
            <Zap size={16} /> AI agent loop cost calculator
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            Runaway Agent Cost Calculator
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Estimate how fast autonomous agents can burn through API, model, and MCP tool budgets when loops, retries, or delegated sub-agents run without request-path budget enforcement.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/ai-agent-cost-control" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              See budget enforcement <ArrowRight size={18} />
            </Link>
            <Link href="/roi-calculator" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-orange-500">
              Compare ROI calculator
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[0.95fr_1.05fr]">
        <div className="space-y-4">
          <Slider label="Active agents in loop" value={agents} min={1} max={500} step={1} onChange={setAgents} />
          <Slider label="Paid calls per agent per minute" value={callsPerMinute} min={1} max={120} step={1} onChange={setCallsPerMinute} />
          <Slider label="Average cost per API/tool call" value={costPerCallCents} min={1} max={500} step={1} prefix="$" suffix="¢" onChange={setCostPerCallCents} />
          <Slider label="Minutes before discovery" value={loopMinutes} min={1} max={240} step={1} suffix=" min" onChange={setLoopMinutes} />
          <Slider label="Delegation fanout multiplier" value={fanout} min={1} max={20} step={1} suffix="×" onChange={setFanout} />
          <Slider label="Incidents per month" value={incidentsPerMonth} min={1} max={30} step={1} onChange={setIncidentsPerMonth} />
        </div>

        <div className="space-y-6">
          <div className="rounded-2xl border border-orange-900/50 bg-orange-950/10 p-6 md:p-8">
            <h2 className="mb-6 text-2xl font-bold text-white">Estimated runaway exposure</h2>
            <div className="grid gap-4 sm:grid-cols-2">
              {[
                ['Calls per incident', number.format(calc.callsPerIncident)],
                ['Cost per incident', money.format(calc.incidentCost)],
                ['Monthly exposure', money.format(calc.monthlyExposure)],
                ['Annual exposure', money.format(calc.annualExposure)],
              ].map(([label, value]) => (
                <div key={label} className="rounded-xl border border-gray-800 bg-black p-5">
                  <div className="mb-2 text-sm text-gray-400">{label}</div>
                  <div className="text-3xl font-extrabold text-white">{value}</div>
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-2xl border border-yellow-900/50 bg-yellow-950/10 p-6 md:p-8">
            <h2 className="mb-4 text-2xl font-bold text-white">If enforcement stops the loop at 5 minutes</h2>
            <p className="mb-6 leading-relaxed text-gray-300">
              A request-path economic firewall can block new calls once a budget, per-tool cap, route policy, or revocation rule is hit. In this model, stopping the loop at five minutes reduces each incident from <strong className="text-white">{money.format(calc.incidentCost)}</strong> to <strong className="text-white">{money.format(calc.blockedAtFiveMinutes)}</strong>.
            </p>
            <div className="rounded-xl border border-gray-800 bg-black p-5">
              <div className="mb-2 text-sm text-gray-400">Avoidable cost per incident</div>
              <div className="text-4xl font-extrabold text-green-400">{money.format(calc.savingsPerIncident)}</div>
            </div>
          </div>
        </div>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-8 text-3xl font-bold text-white">What drives runaway agent cost?</h2>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
            {[
              [Bot, 'Agent count', 'More autonomous workers means more simultaneous paid decisions.'],
              [Gauge, 'Call velocity', 'Retries and tool loops create cost at machine speed.'],
              [ReceiptText, 'Tool pricing', 'MCP tools can hide search, data, cloud, or premium API costs.'],
              [ShieldCheck, 'Late detection', 'Dashboards report spend after the bill. Enforcement blocks before execution.'],
            ].map(([Icon, title, body]) => {
              const TypedIcon = Icon as typeof Bot;
              return (
                <div key={title as string} className="rounded-xl border border-gray-800 bg-black p-6">
                  <TypedIcon className="mb-4 text-orange-300" size={28} />
                  <h3 className="mb-2 text-lg font-bold text-white">{title as string}</h3>
                  <p className="leading-relaxed text-gray-400">{body as string}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-orange-900/60 bg-gradient-to-br from-orange-950/30 to-yellow-950/20 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Stop runaway spend in the request path</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            SatGate observes agent/API spend, controls budgets before paid calls execute, and can charge robot customers with L402 when APIs become agent-native products.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/economic-firewall" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Learn economic firewalls <ArrowRight size={18} />
            </Link>
            <Link href="/mcp-governance" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-orange-500">
              Govern MCP tool spend
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

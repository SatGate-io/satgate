'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { ArrowRight, Bot, Calculator, Coins, Gauge, ReceiptText, Zap } from 'lucide-react';

const usd = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 2 });
const usd0 = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });
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
        style={{ background: `linear-gradient(to right, #facc15 0%, #22d3ee ${pct}%, #374151 ${pct}%, #374151 100%)` }}
      />
    </label>
  );
}

export default function L402ApiPricingCalculatorPage() {
  const [requestsPerDay, setRequestsPerDay] = useState(10000);
  const [costPerRequestCents, setCostPerRequestCents] = useState(2);
  const [marginPct, setMarginPct] = useState(70);
  const [conversionPct, setConversionPct] = useState(12);
  const [agentGrowthPct, setAgentGrowthPct] = useState(25);
  const [freeAllowancePct, setFreeAllowancePct] = useState(20);
  const [btcReferenceUsd, setBtcReferenceUsd] = useState(100000);

  const calc = useMemo(() => {
    const costPerRequest = costPerRequestCents / 100;
    const paidRequestsPerDay = requestsPerDay * (conversionPct / 100) * (1 - freeAllowancePct / 100);
    const pricePerRequest = costPerRequest / Math.max(1 - marginPct / 100, 0.01);
    const dailyRevenue = paidRequestsPerDay * pricePerRequest;
    const monthlyRevenue = dailyRevenue * 30;
    const monthlyCost = paidRequestsPerDay * costPerRequest * 30;
    const monthlyGrossProfit = monthlyRevenue - monthlyCost;
    const nextQuarterRequests = requestsPerDay * Math.pow(1 + agentGrowthPct / 100, 3);
    const nextQuarterMonthlyRevenue = nextQuarterRequests * (conversionPct / 100) * (1 - freeAllowancePct / 100) * pricePerRequest * 30;
    const satsPerUsd = 100000000 / btcReferenceUsd;
    const satsPerRequest = pricePerRequest * satsPerUsd;
    return { paidRequestsPerDay, pricePerRequest, dailyRevenue, monthlyRevenue, monthlyCost, monthlyGrossProfit, nextQuarterMonthlyRevenue, satsPerRequest };
  }, [agentGrowthPct, btcReferenceUsd, conversionPct, costPerRequestCents, freeAllowancePct, marginPct, requestsPerDay]);

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'L402 API Pricing Calculator',
    url: 'https://satgate.io/l402-api-pricing-calculator',
    description: 'Estimate per-request L402 API pricing, paid-agent access revenue, gross margin, free allowance, and Lightning sats per request for governed AI agent API access.',
    datePublished: '2026-05-01',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'L402 API pricing' },
      { '@type': 'Thing', name: 'paid-agent access revenue' },
      { '@type': 'Thing', name: 'Lightning sats per request' },
      { '@type': 'Thing', name: 'governed AI agent paid access' },
      { '@type': 'Thing', name: 'request-path paid access' },
    ],
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'L402 API Pricing Calculator',
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/l402-api-pricing-calculator',
    description: 'Estimate per-request L402 API pricing, paid-agent access revenue, gross margin, free allowance, and Lightning sats per request for governed AI agent API access.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-03',
    featureList: ['Per-request L402 pricing', 'Paid-agent access revenue estimate', 'Gross margin modeling', 'Free allowance planning', 'Adjustable sats per request conversion'],
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'L402 Agent Payments', item: 'https://satgate.io/l402-agent-payments' },
      { '@type': 'ListItem', position: 3, name: 'L402 API Pricing Calculator', item: 'https://satgate.io/l402-api-pricing-calculator' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is L402 API pricing?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'L402 API pricing is per-request API pricing where access is unlocked through an HTTP 402 challenge and Lightning payment proof. SatGate keeps that access tied to delegated authority, budget, scope, and receipts at request time.',
        },
      },
      {
        '@type': 'Question',
        name: 'How should APIs price paid agent access?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Start from marginal cost per request, add target gross margin, account for free allowances or trial traffic, and enforce payment or budget policy before upstream access. L402 is one paid rail; SatGate preserves the authority and receipt context around access.',
        },
      },
      {
        '@type': 'Question',
        name: 'Is L402 the same as Fiat402?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. L402 is Lightning-based payment for API access. Fiat402 is separate. x402, AgentCore Payments, and Pay.sh are also separate paid rails. SatGate can preserve paid-rail context without conflating the rails.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do you calculate sats per API request?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Start with the target USD price per request, then convert that value into sats using the BTC/USD reference assumption in the calculator. Teams should refresh that exchange-rate assumption before publishing production prices.'
        },
      },
      {
        '@type': 'Question',
        name: 'Should paid agent access pricing include free allowances?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Usually yes. Free allowances help agents test value before paying, but paid access should still be enforced with request-path pricing, budget checks, scoped access, and audit records.',
        },
      },
    ],
  };

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />

      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(250,204,21,0.18),transparent_32%),radial-gradient(circle_at_82%_10%,rgba(34,211,238,0.15),transparent_34%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-yellow-500/30 bg-yellow-950/30 px-4 py-2 text-sm text-yellow-200">
            <Zap size={16} /> Free L402 pricing calculator
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">
            L402 API Pricing Calculator
          </h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Estimate per-request pricing, gross margin, paid agent-access demand, and Lightning sats per request before you expose paid API access to autonomous agents.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <a href="#calculator" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Calculate pricing <ArrowRight size={18} />
            </a>
            <Link href="/l402-agent-payments" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-yellow-500">
              L402 agent payments
            </Link>
          </div>
        </div>
      </section>

      <section id="calculator" className="mx-auto grid max-w-6xl gap-10 px-6 py-20 lg:grid-cols-[1fr_0.85fr]">
        <div>
          <h2 className="mb-6 text-3xl font-bold text-white">Model paid agent/API demand</h2>
          <div className="grid gap-4">
            <Slider label="Agent/API requests per day" value={requestsPerDay} min={100} max={100000} step={100} onChange={setRequestsPerDay} />
            <Slider label="Marginal cost per request" value={costPerRequestCents} min={1} max={250} step={1} prefix="$0." onChange={setCostPerRequestCents} />
            <Slider label="Target gross margin" value={marginPct} min={20} max={95} step={1} suffix="%" onChange={setMarginPct} />
            <Slider label="Paying agent conversion" value={conversionPct} min={1} max={80} step={1} suffix="%" onChange={setConversionPct} />
            <Slider label="Free/trial allowance" value={freeAllowancePct} min={0} max={90} step={1} suffix="%" onChange={setFreeAllowancePct} />
            <Slider label="Monthly agent demand growth" value={agentGrowthPct} min={0} max={100} step={1} suffix="%" onChange={setAgentGrowthPct} />
            <Slider label="BTC/USD reference assumption" value={btcReferenceUsd} min={50000} max={200000} step={5000} prefix="$" onChange={setBtcReferenceUsd} />
          </div>
        </div>

        <aside className="sticky top-6 h-fit rounded-3xl border border-yellow-900/50 bg-yellow-950/10 p-8">
          <div className="mb-3 flex items-center gap-2 text-yellow-300"><Calculator size={22} /> Suggested pricing</div>
          <div className="mb-2 text-5xl font-extrabold text-white">{usd.format(calc.pricePerRequest)}</div>
          <div className="mb-6 text-sm text-gray-400">~{number.format(calc.satsPerRequest)} sats/request using your {usd0.format(btcReferenceUsd)} BTC/USD reference assumption</div>
          <div className="grid gap-4">
            <div className="rounded-xl border border-gray-800 bg-black p-4"><div className="text-sm text-gray-500">Paid requests/day</div><div className="text-2xl font-bold text-white">{number.format(calc.paidRequestsPerDay)}</div></div>
            <div className="rounded-xl border border-gray-800 bg-black p-4"><div className="text-sm text-gray-500">Monthly revenue</div><div className="text-2xl font-bold text-white">{usd0.format(calc.monthlyRevenue)}</div></div>
            <div className="rounded-xl border border-gray-800 bg-black p-4"><div className="text-sm text-gray-500">Monthly gross profit</div><div className="text-2xl font-bold text-white">{usd0.format(calc.monthlyGrossProfit)}</div></div>
            <div className="rounded-xl border border-gray-800 bg-black p-4"><div className="text-sm text-gray-500">Projected monthly revenue after 3 months</div><div className="text-2xl font-bold text-white">{usd0.format(calc.nextQuarterMonthlyRevenue)}</div></div>
          </div>
        </aside>
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-4 text-3xl font-bold text-white">Pricing is only safe with request-path control</h2>
          <p className="mb-10 max-w-3xl text-lg leading-relaxed text-gray-400">
            Paid agent-access pricing cannot be just a billing table. Autonomous agents need a challenge, proof, budget, scope, and audit decision before each protected resource unlocks.
          </p>
          <div className="grid gap-5 md:grid-cols-3">
            {[
              [Bot, 'Identify the agent', 'Know which agent, tenant, route, resource, and workflow is asking to spend.'],
              [Coins, 'Price the resource', 'Attach a per-request price or pricing tier to the protected API route.'],
              [ReceiptText, 'Verify proof', 'Accept paid-rail context payment proof before forwarding the paid API request.'],
              [Gauge, 'Respect budget', 'Check remaining budget and policy so an agent cannot spend beyond its allowed task.'],
              [Zap, 'Unlock instantly', 'Let software customers pay and proceed without account setup or invoice friction.'],
              [Calculator, 'Audit economics', 'Record route, price, payment proof, budget, and outcome for analysis.'],
            ].map(([Icon, title, body]) => {
              const CardIcon = Icon as typeof Bot;
              return (
                <div key={String(title)} className="rounded-2xl border border-gray-800 bg-black p-6">
                  <CardIcon className="mb-4 text-yellow-300" size={30} />
                  <h3 className="mb-2 text-xl font-bold text-white">{String(title)}</h3>
                  <p className="leading-relaxed text-gray-400">{String(body)}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      <section className="border-t border-gray-900 bg-black">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-yellow-300">FAQ</p>
          <h2 className="mb-8 text-3xl font-bold text-white">L402 API pricing questions</h2>
          <div className="grid gap-5 md:grid-cols-2">
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">What is L402 API pricing?</h3>
              <p className="text-gray-400 leading-relaxed">
                L402 API pricing is per-request API pricing where access is unlocked through an HTTP 402 challenge and Lightning payment proof. SatGate keeps that access tied to delegated authority, budget, scope, and receipts at request time.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How should APIs price paid agent access?</h3>
              <p className="text-gray-400 leading-relaxed">
                Start from marginal cost per request, add target gross margin, account for free allowances or trial traffic, and enforce payment or budget policy before upstream access. L402 is one paid rail; SatGate preserves the authority and receipt context around access.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Is L402 the same as Fiat402?</h3>
              <p className="text-gray-400 leading-relaxed">
                No. L402 is Lightning-based payment for API access. Fiat402 is separate. x402, AgentCore Payments, and Pay.sh are also separate paid rails. SatGate can preserve paid-rail context without conflating the rails.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">How do you calculate sats per API request?</h3>
              <p className="text-gray-400 leading-relaxed">
                Start with the target USD price per request, then convert that value into sats using the BTC/USD reference assumption in the calculator. Teams should refresh that exchange-rate assumption before publishing production prices.
              </p>
            </div>
            <div className="rounded-xl border border-gray-800 bg-gray-950 p-6">
              <h3 className="mb-2 text-xl font-bold text-white">Should paid agent access pricing include free allowances?</h3>
              <p className="text-gray-400 leading-relaxed">
                Usually yes. Free allowances help agents test value before paying, but paid access should still be enforced with request-path pricing, budget checks, scoped access, and audit records.
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="rounded-3xl border border-yellow-900/60 bg-gradient-to-br from-yellow-950/25 to-cyan-950/25 p-8 md:p-12">
          <h2 className="mb-4 text-3xl font-bold text-white">Model L402 paid access with policy proof.</h2>
          <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">
            L402 can challenge and verify payment for protected API access. SatGate keeps authority, budget, revocation, paid-rail context, and Evidence Pack receipts in the request path.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row">
            <Link href="/http-402-for-ai-agents" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
              Govern HTTP 402 access <ArrowRight size={18} />
            </Link>
            <Link href="/blog/l402-protocol-explained" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-yellow-500">
              L402 protocol explained
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}

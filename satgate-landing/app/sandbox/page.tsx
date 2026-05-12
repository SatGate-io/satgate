'use client';

import { useState, useRef, useEffect, useCallback } from 'react';
import Link from 'next/link';
import {
  ArrowLeft, Play, Square, Bot, Shield, Clock,
  AlertTriangle, CheckCircle, DollarSign, Activity,
  Key, Ban, ChevronDown, Server, ArrowRight, Zap,
} from 'lucide-react';

// ── Types ──────────────────────────────────────────────────────────────────────

type StepStatus = 'pending' | 'running' | 'success' | 'blocked' | 'error' | 'revoked';
type DemoMode = 'kill-switch' | 'budget' | 'both';

interface FeedEvent {
  id: string;
  agent: 'alpha' | 'bravo';
  type: 'auth' | 'mint' | 'call' | 'revoke' | 'verify' | 'summary';
  status: StepStatus;
  label: string;
  detail?: string;
  latencyMs?: number;
  credits?: number;
}

// ── Simulated latencies ────────────────────────────────────────────────────────

const jitter = (base: number) => base + Math.floor(Math.random() * 40) - 20;

const BRAVO_PROMPTS = [
  'List the top 3 API gateway products and their key features.',
  'Compare Kong vs Apigee pricing for startups.',
  'What security features does AWS API Gateway offer?',
  'How does Cloudflare compare for AI agent traffic?',
  'What gap exists for AI agent cost control?',
  'Summarize the competitive landscape.',
];

const BRAVO_RESPONSES = [
  '→ Kong, Apigee, AWS API Gateway — key features: rate limiting, auth, analytics.',
  '→ Kong starts free (OSS), Apigee ~$10K/yr, AWS pay-per-request.',
  '→ WAF, IAM policies, mutual TLS, request validation, usage plans.',
  '→ Strong DDoS + edge caching, but no per-agent cost controls.',
  '→ No gateway today enforces per-agent budgets with cryptographic tokens.',
  '→ Market gap: identity-aware economic controls for autonomous agents.',
];

const sandboxDemos = [
  {
    title: 'Mint Demo',
    href: '/mint-demo',
    icon: Key,
    eyebrow: '1 · Issue authority',
    body: 'Create scoped capability for an agent before it touches a tool, API, or paid resource.',
  },
  {
    title: 'Capability Control Demo',
    href: '/protect',
    icon: Shield,
    eyebrow: '2 · Control authority',
    body: 'See scope, delegation, and revocation in the request path without making spend the center of the demo.',
  },
  {
    title: 'Spend Control Demo',
    href: '#spend-control-demo',
    icon: DollarSign,
    eyebrow: '3 · Constrain spend',
    body: 'Run the budget-enforcement simulation and watch SatGate deny runaway agent spend before value moves.',
  },
  {
    title: 'Paid-Rails Demo',
    href: '/pay',
    icon: Zap,
    eyebrow: '4 · Govern rails',
    body: 'Put paid-rail context behind policy and receipts without making L402/x402 the product center.',
  },
];

function BudgetBar({ label, spent, limit, color }: { label: string; spent: number; limit: number; color: string }) {
  if (limit === 0) return null;
  const pct = Math.min((spent / limit) * 100, 100);
  const exhausted = spent >= limit;
  return (
    <div className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-3">
      <div className="flex justify-between text-xs mb-1.5">
        <span className="text-gray-400">{label}</span>
        <span className={exhausted ? 'text-red-400 font-bold' : 'text-gray-300'}>
          {spent}/{limit} credits {exhausted && '⛔'}
        </span>
      </div>
      <div className="h-2 bg-gray-900 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-500 ${exhausted ? 'bg-red-500' : color}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

// ── Main Page ──────────────────────────────────────────────────────────────────

export default function SandboxPage() {
  const [events, setEvents] = useState<FeedEvent[]>([]);
  const [running, setRunning] = useState(false);
  const [phase, setPhase] = useState<'idle' | 'alpha' | 'bravo' | 'done'>('idle');
  const [selectedMode, setSelectedMode] = useState<DemoMode>('both');
  const [alphaBudget, setAlphaBudget] = useState({ spent: 0, limit: 0 });
  const [bravoBudget, setBravoBudget] = useState({ spent: 0, limit: 0 });
  const [showArchitecture, setShowArchitecture] = useState(false);
  const [hasRun, setHasRun] = useState(false);
  const abortRef = useRef(false);
  const feedRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (feedRef.current) {
      feedRef.current.scrollTop = feedRef.current.scrollHeight;
    }
  }, [events]);

  const addEvent = (e: FeedEvent) => setEvents(prev => [...prev, e]);
  const updateEvent = (id: string, updates: Partial<FeedEvent>) => {
    setEvents(prev => prev.map(e => e.id === id ? { ...e, ...updates } : e));
  };

  const sleep = (ms: number) => new Promise(r => setTimeout(r, ms));

  const resetAll = useCallback(() => {
    setEvents([]);
    setPhase('idle');
    setAlphaBudget({ spent: 0, limit: 0 });
    setBravoBudget({ spent: 0, limit: 0 });
    abortRef.current = false;
    setHasRun(false);
  }, []);

  // ── ALPHA: Kill Switch ─────────────────────────────────────────

  const runAlpha = useCallback(async () => {
    setPhase('alpha');

    addEvent({ id: 'alpha-header', agent: 'alpha', type: 'summary', status: 'running',
      label: '🔴 Agent Alpha — Admin Kill Switch',
      detail: 'Can an admin instantly cut off a rogue agent? Watch.' });

    // Step 1: Auth
    const authId = 'alpha-auth';
    addEvent({ id: authId, agent: 'alpha', type: 'auth', status: 'running',
      label: 'Step 1 → Agent authenticates with Identity Provider',
      detail: 'Subject: alpha-research-001' });
    await sleep(jitter(400));
    if (abortRef.current) return;
    updateEvent(authId, { status: 'success', latencyMs: jitter(42),
      detail: '✓ JWT issued by Identity Provider for "alpha-research-001"' });

    await sleep(600);
    if (abortRef.current) return;

    // Step 2: Mint
    const mintId = 'alpha-mint';
    addEvent({ id: mintId, agent: 'alpha', type: 'mint', status: 'running',
      label: 'Step 2 → Exchanging identity for budget-scoped macaroon' });
    await sleep(jitter(500));
    if (abortRef.current) return;
    setAlphaBudget({ spent: 0, limit: 50 });
    updateEvent(mintId, { status: 'success', latencyMs: jitter(38),
      detail: '✓ Macaroon issued — Policy: "agent-standard" • Budget: 50 credits. Agent can now call APIs.' });

    await sleep(600);
    if (abortRef.current) return;

    // Step 3: One successful call
    const callId = 'alpha-call-1';
    addEvent({ id: callId, agent: 'alpha', type: 'call', status: 'running',
      label: 'Step 3 → API Call #1 through SatGate',
      detail: 'What is an API gateway? One sentence.' });
    await sleep(jitter(600));
    if (abortRef.current) return;
    setAlphaBudget({ spent: 10, limit: 50 });
    updateEvent(callId, { status: 'success', latencyMs: jitter(127), credits: 10,
      detail: '→ An API gateway is a single entry point that routes, secures, and manages API traffic.' });

    await sleep(800);
    if (abortRef.current) return;

    // Step 4: Admin revokes
    const revokeId = 'alpha-revoke';
    addEvent({ id: revokeId, agent: 'alpha', type: 'revoke', status: 'running',
      label: 'Step 4 → Admin revokes Agent Alpha\'s credential' });
    await sleep(jitter(300));
    if (abortRef.current) return;
    updateEvent(revokeId, { status: 'revoked', latencyMs: jitter(12),
      detail: '✓ Token revoked instantly — agent has no idea yet.' });

    await sleep(600);
    if (abortRef.current) return;

    // Step 5: Agent tries again — blocked
    const verifyId = 'alpha-verify';
    addEvent({ id: verifyId, agent: 'alpha', type: 'verify', status: 'running',
      label: 'Step 5 → Agent Alpha tries to call API again...' });
    await sleep(jitter(400));
    if (abortRef.current) return;
    updateEvent(verifyId, { status: 'blocked', latencyMs: jitter(8),
      detail: 'HTTP 401 — Blocked. Agent is permanently locked out. Zero human latency.' });

    addEvent({ id: 'alpha-done', agent: 'alpha', type: 'summary', status: 'success',
      label: '✓ Kill switch works: Authenticate → Call API → Admin Revoke → Instant Block' });

    return true;
  }, []);

  // ── BRAVO: Economic Firewall ───────────────────────────────────

  const runBravo = useCallback(async () => {
    setPhase('bravo');

    addEvent({ id: 'bravo-header', agent: 'bravo', type: 'summary', status: 'running',
      label: '🟡 Agent Bravo — Economic Firewall',
      detail: 'Agent gets a 50-credit budget. Each call costs 10. What happens on call #6?' });

    // Step 1: Auth
    const authId = 'bravo-auth';
    addEvent({ id: authId, agent: 'bravo', type: 'auth', status: 'running',
      label: 'Step 1 → Agent authenticates with Identity Provider',
      detail: 'Subject: bravo-analyst-001' });
    await sleep(jitter(400));
    if (abortRef.current) return;
    updateEvent(authId, { status: 'success', latencyMs: jitter(45),
      detail: '✓ JWT issued by Identity Provider for "bravo-analyst-001"' });

    await sleep(600);
    if (abortRef.current) return;

    // Step 2: Mint
    const mintId = 'bravo-mint';
    addEvent({ id: mintId, agent: 'bravo', type: 'mint', status: 'running',
      label: 'Step 2 → Exchanging identity for budget-scoped macaroon' });
    await sleep(jitter(500));
    if (abortRef.current) return;
    setBravoBudget({ spent: 0, limit: 50 });
    updateEvent(mintId, { status: 'success', latencyMs: jitter(41),
      detail: '✓ Macaroon issued — Policy: "agent-standard" • Budget: 50 credits. Agent can now call APIs.' });

    await sleep(600);

    // Steps 3-7: Make calls, last one blocked
    let spent = 0;
    for (let i = 0; i < BRAVO_PROMPTS.length; i++) {
      if (abortRef.current) break;

      const callId = `bravo-call-${i + 1}`;
      const isBlocked = i === 5; // 6th call (index 5) hits budget

      addEvent({ id: callId, agent: 'bravo', type: 'call', status: 'running',
        label: `Step 3 → API Call #${i + 1} through SatGate`,
        detail: BRAVO_PROMPTS[i] });

      await sleep(jitter(isBlocked ? 300 : 500));
      if (abortRef.current) break;

      if (isBlocked) {
        updateEvent(callId, { status: 'blocked', latencyMs: jitter(6),
          detail: `🚫 HTTP 402 Payment Required — Budget exhausted (${spent}/${50} credits spent)` });
        addEvent({ id: 'bravo-blocked', agent: 'bravo', type: 'summary', status: 'blocked',
          label: `💰 Budget Exhausted — HTTP 402`,
          detail: `Spent ${spent}/50 credits. Call #${i + 1} blocked automatically.` });
      } else {
        spent += 10;
        setBravoBudget({ spent, limit: 50 });
        updateEvent(callId, { status: 'success', latencyMs: jitter(130), credits: 10,
          detail: BRAVO_RESPONSES[i] });
      }

      await sleep(300);
    }

    addEvent({ id: 'bravo-done', agent: 'bravo', type: 'summary', status: 'success',
      label: '✓ Economic firewall works: Budget enforced automatically — no human needed' });

    return true;
  }, []);

  // ── Run orchestration ──────────────────────────────────────────

  const runDemo = useCallback(async () => {
    setRunning(true);
    setHasRun(true);
    abortRef.current = false;
    setEvents([]);
    setAlphaBudget({ spent: 0, limit: 0 });
    setBravoBudget({ spent: 0, limit: 0 });

    if (selectedMode === 'kill-switch' || selectedMode === 'both') {
      await runAlpha();
      if (abortRef.current) { setRunning(false); return; }
      if (selectedMode === 'both') await sleep(1200);
    }

    if (selectedMode === 'budget' || selectedMode === 'both') {
      await runBravo();
    }

    setPhase('done');
    setRunning(false);
  }, [selectedMode, runAlpha, runBravo]);

  const stopDemo = () => {
    abortRef.current = true;
    setRunning(false);
  };

  // ── Status colors ─────────────────────────────────────────────

  const statusColor = (s: StepStatus) => {
    switch (s) {
      case 'running': return 'text-blue-400 animate-pulse';
      case 'success': return 'text-green-400';
      case 'blocked': return 'text-red-400';
      case 'error': return 'text-red-500';
      case 'revoked': return 'text-orange-400';
      default: return 'text-gray-500';
    }
  };

  const statusIcon = (s: StepStatus) => {
    switch (s) {
      case 'running': return <Activity size={16} className="animate-spin" />;
      case 'success': return <CheckCircle size={16} />;
      case 'blocked': return <Ban size={16} />;
      case 'error': return <AlertTriangle size={16} />;
      case 'revoked': return <Ban size={16} />;
      default: return <Clock size={16} />;
    }
  };

  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'SatGate Demo',
    url: 'https://satgate.io/sandbox',
    description: 'Interactive SatGate demo for Mint, Capability Control, Spend Control, and Paid-Rails.',
    datePublished: '2026-04-12',
    dateModified: '2026-05-03',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'SatGate demo' },
      { '@type': 'Thing', name: 'capability control demo' },
      { '@type': 'Thing', name: 'AI agent spend control demo' },
      { '@type': 'Thing', name: 'paid-rail governance demo' },
      { '@type': 'Thing', name: 'macaroon capability verification' },
      { '@type': 'Thing', name: 'agent kill-switch revocation' },
      { '@type': 'Thing', name: 'runaway spend blocking simulation' },
    ],
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate Demo',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/sandbox',
    description: 'Interactive SatGate demo for Mint, Capability Control, Spend Control, and Paid-Rails.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
    dateModified: '2026-05-03',
    featureList: [
      'Mint Demo',
      'Capability Control Demo',
      'Spend Control Demo',
      'Paid-Rails Demo',
      'Request-path policy decisions',
    ],
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'AI Agent Cost Control', item: 'https://satgate.io/ai-agent-cost-control' },
      { '@type': 'ListItem', position: 3, name: 'SatGate Demo', item: 'https://satgate.io/sandbox' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What does the SatGate demo demonstrate?',
        acceptedAnswer: { '@type': 'Answer', text: 'The demo page collects the SatGate demo path: Mint for scoped authority, Capability Control for scope/delegation/revocation, Spend Control for budget enforcement, and Paid-Rails for governed payment context.' },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate stop unauthorized agent spend?',
        acceptedAnswer: { '@type': 'Answer', text: 'SatGate checks each agent request against identity, capability-token caveats, budget, policy, and revocation state before forwarding the request upstream.' },
      },
      {
        '@type': 'Question',
        name: 'Is the demo for AI agent cost control or security?',
        acceptedAnswer: { '@type': 'Answer', text: 'Both. SatGate treats spend as an enforceable security boundary, combining scoped authority, revocation, audit, and budget limits into an economic firewall for AI agents.' },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      {/* Header */}
      <div className="border-b border-gray-800 bg-black/80 backdrop-blur-sm sticky top-0 z-40">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition">
            <ArrowLeft size={18} /> <span className="hidden sm:inline">Back to Home</span>
          </Link>
          <h1 className="text-lg sm:text-xl font-bold flex items-center gap-2">
            <Shield className="text-purple-400" size={24} />
            <span className="bg-clip-text text-transparent bg-gradient-to-r from-purple-400 to-cyan-400">
              SatGate Demo
            </span>
          </h1>
          <Link
            href="https://cloud.satgate.io/cloud/login"
            className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-purple-600 hover:bg-purple-500 rounded-lg text-sm font-medium transition"
          >
            Try It Free <ArrowRight size={14} />
          </Link>
        </div>
      </div>

      {/* Demo Hub */}
      <div className="bg-gradient-to-b from-purple-950/20 to-transparent border-b border-gray-800/50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 py-12 text-center">
          <p className="mb-3 text-sm font-mono uppercase tracking-[0.22em] text-purple-300">Interactive demo</p>
          <h2 className="text-3xl sm:text-5xl font-extrabold tracking-tight text-white mb-4">
            Mint. Control. Constrain. Govern.
          </h2>
          <p className="mx-auto max-w-3xl text-lg leading-relaxed text-gray-400">
            Start with scoped authority, then move through capability control, spend control, and paid-rail context. Each demo is a separate proof path.
          </p>
          <div className="mt-10 grid gap-4 md:grid-cols-2 lg:grid-cols-4 text-left">
            {sandboxDemos.map(({ title, href, icon: Icon, eyebrow, body }) => (
              <Link
                key={title}
                href={href}
                className="group rounded-2xl border border-gray-800 bg-gray-950/80 p-5 transition hover:border-purple-500/60 hover:bg-purple-950/20"
              >
                <Icon className="mb-4 text-purple-300 transition group-hover:text-cyan-300" size={28} />
                <p className="mb-2 text-xs font-mono uppercase tracking-wide text-gray-500">{eyebrow}</p>
                <h3 className="mb-3 text-lg font-bold text-white">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{body}</p>
              </Link>
            ))}
          </div>
        </div>
      </div>

      {/* Spend Control Demo */}
      <div id="spend-control-demo" className="bg-gradient-to-b from-purple-950/20 to-transparent border-b border-gray-800/50">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8 text-center">
          <h2 className="text-2xl sm:text-3xl font-bold mb-4">
            <span className="bg-clip-text text-transparent bg-gradient-to-r from-purple-400 via-cyan-400 to-purple-400">
              Watch SatGate Stop Unauthorized Spend
            </span>
          </h2>
          <p className="text-gray-400 text-lg leading-relaxed max-w-2xl mx-auto">
            Two scenarios CFOs care about: a rogue agent gets cut off instantly, and an agent
            hits its budget ceiling and stops — before the bill arrives.
          </p>
        </div>
      </div>

      <div className="max-w-5xl mx-auto px-4 sm:px-6 py-8">
        {/* Controls */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 mb-6">
          <div className="flex items-center gap-3">
            <select
              value={selectedMode}
              onChange={e => setSelectedMode(e.target.value as DemoMode)}
              disabled={running}
              className="bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-purple-500"
            >
              <option value="both">Both Scenarios</option>
              <option value="kill-switch">🔴 Kill Switch Only</option>
              <option value="budget">🟡 Economic Firewall Only</option>
            </select>
            <span className="text-gray-600 text-xs hidden sm:inline">
              {selectedMode === 'both' ? '~30 seconds' : '~15 seconds'}
            </span>
          </div>
          <div className="flex items-center gap-3">
            {hasRun && !running && (
              <button onClick={resetAll} className="flex items-center gap-2 px-4 py-2 bg-gray-800 border border-gray-700 text-gray-400 rounded-lg text-sm font-medium hover:bg-gray-700 transition">
                Reset
              </button>
            )}
            <button
              onClick={running ? stopDemo : runDemo}
              className={`flex items-center gap-2 px-6 py-2.5 rounded-lg text-sm font-bold transition-all ${
                running
                  ? 'bg-red-900/50 border border-red-700 text-red-400 hover:bg-red-900'
                  : 'bg-purple-600 hover:bg-purple-500 text-white shadow-lg shadow-purple-500/20'
              }`}
            >
              {running ? <><Square size={16} /> Stop</> : <><Play size={16} /> Run Simulation</>}
            </button>
          </div>
        </div>

        {/* Budget Bars */}
        {(alphaBudget.limit > 0 || bravoBudget.limit > 0) && (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-6">
            <BudgetBar label="🔴 Alpha — Kill Switch" spent={alphaBudget.spent} limit={alphaBudget.limit} color="bg-red-500" />
            <BudgetBar label="🟡 Bravo — Economic Firewall" spent={bravoBudget.spent} limit={bravoBudget.limit} color="bg-yellow-500" />
          </div>
        )}

        {/* Event Feed */}
        <div
          ref={feedRef}
          className="bg-gray-900/50 border border-gray-800 rounded-xl overflow-y-auto"
          style={{ minHeight: '400px', maxHeight: '600px' }}
        >
          {events.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-[400px] text-gray-600">
              <Shield size={48} className="mb-4 opacity-30" />
              <p className="text-lg font-medium mb-2">Ready to simulate</p>
              <p className="text-sm">Click &quot;Run Simulation&quot; to watch SatGate enforce access controls in real-time.</p>
            </div>
          ) : (
            <div className="divide-y divide-gray-800/50">
              {events.map((event) => (
                <div
                  key={event.id}
                  className={`px-4 py-3 flex items-start gap-3 transition-all duration-300 ${
                    event.type === 'summary' ? 'bg-gray-800/30' : ''
                  } ${event.status === 'blocked' ? 'bg-red-950/20' : ''} ${event.status === 'revoked' ? 'bg-orange-950/20' : ''}`}
                >
                  <div className={`mt-0.5 flex-shrink-0 ${statusColor(event.status)}`}>
                    {statusIcon(event.status)}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className={`text-sm font-medium ${event.type === 'summary' ? 'text-white' : 'text-gray-300'}`}>
                        {event.label}
                      </span>
                      {event.latencyMs !== undefined && (
                        <span className="text-xs text-gray-600">{event.latencyMs}ms</span>
                      )}
                      {event.credits !== undefined && (
                        <span className="text-xs text-yellow-500/70 flex items-center gap-0.5">
                          <DollarSign size={10} />{event.credits} credits
                        </span>
                      )}
                    </div>
                    {event.detail && (
                      <p className={`text-sm mt-0.5 ${
                        event.status === 'blocked' ? 'text-red-400' :
                        event.status === 'revoked' ? 'text-orange-400' :
                        event.status === 'error' ? 'text-red-500' :
                        'text-gray-500'
                      }`}>
                        {event.detail}
                      </p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Architecture Expandable */}
        <div className="mt-8 border border-gray-800 rounded-xl overflow-hidden">
          <button
            onClick={() => setShowArchitecture(!showArchitecture)}
            className="w-full flex items-center justify-between px-6 py-4 bg-gray-900/50 hover:bg-gray-800/50 transition"
          >
            <div className="flex items-center gap-3">
              <Server size={18} className="text-purple-400" />
              <span className="text-sm font-semibold text-white">Under the Hood — How It Works</span>
            </div>
            <ChevronDown size={18} className={`text-gray-500 transition-transform ${showArchitecture ? 'rotate-180' : ''}`} />
          </button>
          {showArchitecture && (
            <div className="px-6 py-6 border-t border-gray-800 bg-gray-950/50">
              <div className="flex flex-col md:flex-row items-center justify-center gap-4 text-center">
                {[
                  { icon: <Bot size={24} />, label: 'AI Agent', sub: 'Presents workload identity', color: 'text-blue-400' },
                  { icon: <ArrowRight size={20} />, label: '', sub: '', color: 'text-gray-600' },
                  { icon: <Key size={24} />, label: 'SatGate Mint', sub: 'Identity → Macaroon exchange', color: 'text-purple-400' },
                  { icon: <ArrowRight size={20} />, label: '', sub: '', color: 'text-gray-600' },
                  { icon: <Shield size={24} />, label: 'SatGate Gateway', sub: 'Enforce budget + revocation', color: 'text-cyan-400' },
                  { icon: <ArrowRight size={20} />, label: '', sub: '', color: 'text-gray-600' },
                  { icon: <Zap size={24} />, label: 'Upstream API', sub: 'Only reached if allowed', color: 'text-green-400' },
                ].map((item, i) => (
                  <div key={i} className={`flex flex-col items-center gap-1 ${item.color}`}>
                    {item.icon}
                    {item.label && <span className="text-xs font-bold text-white">{item.label}</span>}
                    {item.sub && <span className="text-xs text-gray-500">{item.sub}</span>}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        <section className="mt-12 border-t border-gray-800 pt-10">
          <p className="mb-2 text-center text-xs font-mono uppercase tracking-wide text-purple-300">FAQ</p>
          <h2 className="mb-8 text-center text-2xl font-bold text-white">SatGate demo questions</h2>
          <div className="grid gap-4 md:grid-cols-3">
            {[
              ['What does the SatGate demo demonstrate?', 'The demo page collects the SatGate demo path: Mint for scoped authority, Capability Control for scope/delegation/revocation, Spend Control for budget enforcement, and Paid-Rails for governed payment context.'],
              ['How does SatGate stop unauthorized agent spend?', 'SatGate checks each agent request against identity, capability-token caveats, budget, policy, and revocation state before forwarding the request upstream.'],
              ['Is the demo for AI agent cost control or security?', 'Both. SatGate treats spend as an enforceable security boundary, combining scoped authority, revocation, audit, and budget limits into an economic firewall for AI agents.'],
            ].map(([question, answer]) => (
              <div key={question} className="rounded-xl border border-gray-800 bg-gray-900 p-5">
                <h3 className="mb-2 font-bold text-white">{question}</h3>
                <p className="text-sm leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </section>

        {/* CTA */}
        <div className="mt-12 text-center">
          <p className="text-gray-500 text-sm mb-4">
            This was a simulation. The real thing runs on your infrastructure in minutes.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="https://cloud.satgate.io/cloud/login"
              className="inline-flex items-center gap-2 px-6 py-3 bg-purple-600 hover:bg-purple-500 rounded-lg text-sm font-bold transition shadow-lg shadow-purple-500/20"
            >
              Start Free — No Credit Card <ArrowRight size={16} />
            </Link>
            <Link
              href="https://github.com/SatGate-io/satgate"
              className="inline-flex items-center gap-2 px-6 py-3 bg-gray-800 hover:bg-gray-700 border border-gray-700 rounded-lg text-sm font-medium text-gray-300 transition"
            >
              View on GitHub
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}

'use client';

import { useState } from 'react';
import Link from 'next/link';
import { ArrowLeft, ArrowRight, Key, Shield, CheckCircle, Copy, Check, Loader2, ChevronDown, ExternalLink } from 'lucide-react';

const PRESETS = [
  { id: 'research-agent', label: 'Research Agent', desc: 'K8s namespace: ai-agents, 1000 credits/day', icon: '🔬' },
  { id: 'data-pipeline', label: 'Data Pipeline', desc: 'K8s namespace: data, data engineering team', icon: '📊' },
  { id: 'ci-runner', label: 'CI Runner', desc: 'GitHub Actions, main branch only', icon: '⚡' },
  { id: 'intern-bot', label: 'Intern Bot', desc: 'K8s namespace: ai-agents, restricted access', icon: '🤖' },
];

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      onClick={() => { navigator.clipboard.writeText(text); setCopied(true); setTimeout(() => setCopied(false), 2000); }}
      className="inline-flex items-center gap-1 px-2 py-1 text-xs bg-gray-800 hover:bg-gray-700 rounded transition"
    >
      {copied ? <Check size={12} className="text-green-400" /> : <Copy size={12} className="text-gray-400" />}
      <span className="text-gray-300">{copied ? 'Copied' : 'Copy'}</span>
    </button>
  );
}

interface StepResult {
  token?: string;
  claims?: Record<string, unknown>;
  macaroon?: string;
  macaroonCaveats?: string[];
  policy?: string;
  budget?: Record<string, unknown>;
  verified?: boolean;
  error?: string;
}

export default function MintDemoPage() {
  const [preset, setPreset] = useState('research-agent');
  const [step, setStep] = useState(0); // 0=ready, 1=getting token, 2=got token, 3=exchanging, 4=got macaroon, 5=verifying, 6=verified
  const [result, setResult] = useState<StepResult>({});
  const [showRaw, setShowRaw] = useState(false);

  const runDemo = async () => {
    setStep(1);
    setResult({});

    try {
      // Step 1: Get identity token from Mock IdP
      const idpResp = await fetch('https://satgate-mock-idp.fly.dev/token', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ preset }),
      });
      const idpData = await idpResp.json();

      if (!idpData.token) throw new Error('Failed to get identity token');

      setResult({ token: idpData.token, claims: idpData.claims });
      setStep(2);

      // Brief pause for visual effect
      await new Promise(r => setTimeout(r, 800));
      setStep(3);

      // Step 2: Exchange for SatGate macaroon
      const mintResp = await fetch('https://cloud.satgate.io/api/mint/exchange', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ credentials: idpData.token }),
      });
      const mintData = await mintResp.json();

      if (mintData.error) throw new Error(mintData.error);

      setResult(prev => ({
        ...prev,
        macaroon: mintData.token,
        macaroonCaveats: mintData.caveats,
        policy: mintData.policy,
        budget: mintData.budget,
      }));
      setStep(4);

      await new Promise(r => setTimeout(r, 800));
      setStep(5);

      // Step 3: Verify the minted macaroon
      const verifyResp = await fetch('https://cloud.satgate.io/api/mint/verify', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${mintData.token}`,
        },
      });
      const verifyData = await verifyResp.json();

      setResult(prev => ({ ...prev, verified: verifyData.valid }));
      setStep(6);

    } catch (err) {
      setResult(prev => ({ ...prev, error: (err as Error).message }));
      setStep(0);
    }
  };

  const reset = () => {
    setStep(0);
    setResult({});
    setShowRaw(false);
  };

  const softwareJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'SatGate Mint Demo',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Web',
    url: 'https://satgate.io/mint-demo',
    description: 'Interactive demo for minting budget-aware capability tokens and macaroons for AI agents with scopes, expiry, delegation limits, and revocation policy.',
    publisher: { '@type': 'Organization', name: 'SatGate', url: 'https://satgate.io' },
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://satgate.io' },
      { '@type': 'ListItem', position: 2, name: 'Security', item: 'https://satgate.io/security' },
      { '@type': 'ListItem', position: 3, name: 'SatGate Mint Demo', item: 'https://satgate.io/mint-demo' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What does the SatGate Mint demo show?',
        acceptedAnswer: { '@type': 'Answer', text: 'The demo shows an AI agent exchanging workload identity for a scoped macaroon capability token with budget, expiry, policy, and verification data.' },
      },
      {
        '@type': 'Question',
        name: 'Why mint capability tokens for AI agents?',
        acceptedAnswer: { '@type': 'Answer', text: 'Capability tokens let teams give agents narrow, revocable, budget-aware API authority instead of broad static API keys.' },
      },
      {
        '@type': 'Question',
        name: 'How do macaroons help with agent delegation?',
        acceptedAnswer: { '@type': 'Answer', text: 'Macaroon caveats let delegated agent credentials become more constrained by route, budget, expiry, call count, and delegation policy while preserving cryptographic verification.' },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
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
            <Key className="text-purple-400" size={24} />
            <span className="bg-clip-text text-transparent bg-gradient-to-r from-purple-400 to-cyan-400">
              Mint Demo
            </span>
          </h1>
          <Link
            href="https://cloud.satgate.io/cloud/mint"
            className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-purple-600 hover:bg-purple-500 rounded-lg text-sm font-medium transition"
          >
            Configure Mint <ExternalLink size={14} />
          </Link>
        </div>
      </div>

      {/* Hero Description */}
      <div className="bg-gradient-to-b from-purple-950/20 to-transparent border-b border-gray-800/50">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8 text-center">
          <h2 className="text-2xl sm:text-3xl font-bold mb-4">
            <span className="bg-clip-text text-transparent bg-gradient-to-r from-purple-400 via-cyan-400 to-purple-400">
              Identity → Macaroon in One API Call
            </span>
          </h2>
          <p className="text-gray-400 text-lg leading-relaxed max-w-2xl mx-auto">
            Watch SatGate Mint exchange a workload identity token for a capability-bearing macaroon. 
            No secrets to manage. No tokens to rotate. The identity <em>is</em> the credential.
          </p>
        </div>
      </div>

      <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8">

        {/* Preset Selector */}
        <div className="mb-8">
          <label className="block text-sm text-gray-400 mb-3">Choose an agent identity:</label>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {PRESETS.map(p => (
              <button
                key={p.id}
                onClick={() => { setPreset(p.id); if (step === 6) reset(); }}
                className={`p-4 rounded-xl border text-left transition ${
                  preset === p.id
                    ? 'border-purple-500/50 bg-purple-500/10'
                    : 'border-gray-800 bg-[#12121a] hover:border-gray-700'
                }`}
              >
                <div className="text-2xl mb-2">{p.icon}</div>
                <div className="text-white text-sm font-medium">{p.label}</div>
                <div className="text-gray-500 text-xs mt-1">{p.desc}</div>
              </button>
            ))}
          </div>
        </div>

        {/* Run Button */}
        {step === 0 && (
          <div className="text-center mb-12">
            <button
              onClick={runDemo}
              className="inline-flex items-center gap-2 px-8 py-4 bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-500 hover:to-cyan-500 rounded-xl text-white font-semibold text-lg transition shadow-lg shadow-purple-500/20"
            >
              Run the Exchange <ArrowRight size={20} />
            </button>
          </div>
        )}

        {step === 6 && (
          <div className="text-center mb-12">
            <button
              onClick={reset}
              className="inline-flex items-center gap-2 px-6 py-3 bg-gray-800 hover:bg-gray-700 rounded-xl text-white font-medium transition"
            >
              Try Another Agent
            </button>
          </div>
        )}

        {/* Steps */}
        <div className="space-y-6">
          {/* Step 1: Identity Token */}
          <div className={`rounded-xl border transition-all duration-500 ${
            step >= 1 ? 'border-purple-500/30 bg-[#12121a]' : 'border-gray-800/50 bg-[#0d0d14]'
          }`}>
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold ${
                  step >= 2 ? 'bg-green-500/20 text-green-400' :
                  step >= 1 ? 'bg-purple-500/20 text-purple-400 animate-pulse' :
                  'bg-gray-800 text-gray-500'
                }`}>
                  {step >= 2 ? <CheckCircle size={16} /> : '1'}
                </div>
                <div>
                  <h3 className="text-white font-medium">Agent presents identity</h3>
                  <p className="text-gray-500 text-sm">Mock IdP issues an RS256-signed JWT (like K8s, AWS, or Okta would)</p>
                </div>
                {step === 1 && <Loader2 size={16} className="text-purple-400 animate-spin ml-auto" />}
              </div>

              {result.claims && (
                <div className="bg-[#0a0a0a] rounded-lg p-4 text-sm space-y-2">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-gray-500 text-xs font-mono">JWT Claims</span>
                    <CopyButton text={result.token || ''} />
                  </div>
                  <div className="grid grid-cols-2 gap-x-6 gap-y-1">
                    {Object.entries(result.claims).filter(([k]) => !['iat', 'nbf'].includes(k)).map(([k, v]) => (
                      <div key={k} className="flex gap-2">
                        <span className="text-purple-400 font-mono text-xs">{k}:</span>
                        <span className="text-gray-300 text-xs truncate">
                          {Array.isArray(v) ? (v as string[]).join(', ') : String(v)}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Step 2: Mint Exchange */}
          <div className={`rounded-xl border transition-all duration-500 ${
            step >= 3 ? 'border-cyan-500/30 bg-[#12121a]' :
            step >= 2 ? 'border-gray-700 bg-[#12121a]' :
            'border-gray-800/50 bg-[#0d0d14]'
          }`}>
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold ${
                  step >= 4 ? 'bg-green-500/20 text-green-400' :
                  step >= 3 ? 'bg-cyan-500/20 text-cyan-400 animate-pulse' :
                  'bg-gray-800 text-gray-500'
                }`}>
                  {step >= 4 ? <CheckCircle size={16} /> : '2'}
                </div>
                <div>
                  <h3 className="text-white font-medium">SatGate Mint exchanges identity for macaroon</h3>
                  <p className="text-gray-500 text-sm">Verifies JWT via JWKS → matches policy → mints capability token</p>
                </div>
                {step === 3 && <Loader2 size={16} className="text-cyan-400 animate-spin ml-auto" />}
              </div>

              {result.macaroon && (
                <div className="bg-[#0a0a0a] rounded-lg p-4 text-sm space-y-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="px-2 py-0.5 bg-green-500/10 text-green-400 text-xs rounded font-mono">
                        policy: {result.policy}
                      </span>
                      {result.budget && (
                        <span className="px-2 py-0.5 bg-yellow-500/10 text-yellow-400 text-xs rounded font-mono">
                          budget: {String((result.budget as Record<string, unknown>).limit)} {String((result.budget as Record<string, unknown>).currency)}
                        </span>
                      )}
                    </div>
                    <CopyButton text={result.macaroon} />
                  </div>

                  <div>
                    <span className="text-gray-500 text-xs font-mono block mb-1">Macaroon Caveats (immutable constraints):</span>
                    <div className="space-y-1">
                      {result.macaroonCaveats?.map((c, i) => (
                        <div key={i} className="flex items-center gap-2">
                          <span className="text-cyan-400">→</span>
                          <code className="text-gray-300 text-xs">{c}</code>
                        </div>
                      ))}
                    </div>
                  </div>

                  <button
                    onClick={() => setShowRaw(!showRaw)}
                    className="flex items-center gap-1 text-gray-500 text-xs hover:text-gray-300 transition"
                  >
                    <ChevronDown size={12} className={`transition ${showRaw ? 'rotate-180' : ''}`} />
                    {showRaw ? 'Hide' : 'Show'} raw macaroon
                  </button>
                  {showRaw && (
                    <pre className="text-xs text-gray-500 break-all bg-black/50 p-2 rounded">{result.macaroon}</pre>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Step 3: Verify */}
          <div className={`rounded-xl border transition-all duration-500 ${
            step >= 5 ? 'border-green-500/30 bg-[#12121a]' :
            'border-gray-800/50 bg-[#0d0d14]'
          }`}>
            <div className="p-6">
              <div className="flex items-center gap-3">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold ${
                  step >= 6 ? 'bg-green-500/20 text-green-400' :
                  step >= 5 ? 'bg-green-500/20 text-green-400 animate-pulse' :
                  'bg-gray-800 text-gray-500'
                }`}>
                  {step >= 6 ? <CheckCircle size={16} /> : '3'}
                </div>
                <div>
                  <h3 className="text-white font-medium">Token verified — agent is ready</h3>
                  <p className="text-gray-500 text-sm">The macaroon passes gateway verification. Agent can now call tools.</p>
                </div>
                {step === 5 && <Loader2 size={16} className="text-green-400 animate-spin ml-auto" />}
                {result.verified && (
                  <span className="ml-auto px-3 py-1 bg-green-500/10 border border-green-500/30 text-green-400 text-sm rounded-full font-medium">
                    ✓ Valid
                  </span>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Error */}
        {result.error && (
          <div className="mt-6 p-4 bg-red-500/10 border border-red-500/30 rounded-xl text-red-300 text-sm">
            {result.error}
          </div>
        )}

        {/* What just happened */}
        {step === 6 && (
          <div className="mt-12 bg-gradient-to-br from-purple-500/5 to-cyan-500/5 border border-purple-500/10 rounded-xl p-8">
            <h2 className="text-xl font-semibold text-white mb-4">What just happened?</h2>
            <ol className="space-y-3 text-gray-300 text-sm">
              <li className="flex gap-3">
                <span className="text-purple-400 font-mono shrink-0">1.</span>
                <span>The <strong className="text-white">Mock Identity Provider</strong> issued an RS256-signed JWT — simulating what Kubernetes, AWS IAM, or your corporate IdP would issue to a workload.</span>
              </li>
              <li className="flex gap-3">
                <span className="text-cyan-400 font-mono shrink-0">2.</span>
                <span><strong className="text-white">SatGate Mint</strong> verified the JWT signature via the IdP&apos;s JWKS endpoint, matched it against an agent policy, and minted a <strong className="text-white">macaroon</strong> with embedded caveats (budget, scope, identity binding).</span>
              </li>
              <li className="flex gap-3">
                <span className="text-green-400 font-mono shrink-0">3.</span>
                <span>The macaroon is a <strong className="text-white">gateway-compatible capability token</strong>. The agent uses it to connect through SatGate — every tool call is authenticated, cost-tracked, and budget-enforced.</span>
              </li>
            </ol>
            <div className="mt-6 flex flex-wrap gap-3">
              <Link
                href="https://cloud.satgate.io/cloud/mint"
                className="inline-flex items-center gap-2 px-4 py-2 bg-purple-600 hover:bg-purple-500 rounded-lg text-sm font-medium transition"
              >
                Configure Your IdP <ArrowRight size={14} />
              </Link>
              <Link
                href="https://cloud.satgate.io/docs/guides/mint-identity"
                className="inline-flex items-center gap-2 px-4 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-sm font-medium transition"
              >
                Read the Guide <ArrowRight size={14} />
              </Link>
              <Link
                href="https://cloud.satgate.io/docs/guides/sdk"
                className="inline-flex items-center gap-2 px-4 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-sm font-medium transition"
              >
                SDK Docs <ArrowRight size={14} />
              </Link>
            </div>
          </div>
        )}

        {/* API Reference */}
        <div className="mt-16 border-t border-gray-800 pt-12">
          <h2 className="text-lg font-semibold text-white mb-6">Try it yourself</h2>
          <div className="bg-[#0a0a0a] border border-gray-800 rounded-xl p-6 space-y-4">
            <div>
              <p className="text-gray-400 text-sm mb-2">1. Get an identity token:</p>
              <pre className="text-xs text-gray-300 bg-black/50 p-3 rounded overflow-x-auto">
{`curl -X POST https://satgate-mock-idp.fly.dev/token \\
  -H "Content-Type: application/json" \\
  -d '{"preset":"research-agent"}'`}
              </pre>
            </div>
            <div>
              <p className="text-gray-400 text-sm mb-2">2. Exchange for a macaroon:</p>
              <pre className="text-xs text-gray-300 bg-black/50 p-3 rounded overflow-x-auto">
{`curl -X POST https://cloud.satgate.io/api/mint/exchange \\
  -H "Content-Type: application/json" \\
  -d '{"credentials":"<paste-jwt-here>"}'`}
              </pre>
            </div>
            <div>
              <p className="text-gray-400 text-sm mb-2">3. Verify the macaroon:</p>
              <pre className="text-xs text-gray-300 bg-black/50 p-3 rounded overflow-x-auto">
{`curl -X POST https://cloud.satgate.io/api/mint/verify \\
  -H "Authorization: Bearer <paste-macaroon-here>"`}
              </pre>
            </div>
          </div>
          <p className="text-gray-600 text-xs mt-4">
            Mock IdP: <a href="https://satgate-mock-idp.fly.dev" className="text-purple-400 hover:text-purple-300">satgate-mock-idp.fly.dev</a> · 
            Presets: <a href="https://satgate-mock-idp.fly.dev/presets" className="text-purple-400 hover:text-purple-300">/presets</a> · 
            OIDC: <a href="https://satgate-mock-idp.fly.dev/.well-known/openid-configuration" className="text-purple-400 hover:text-purple-300">/.well-known/openid-configuration</a>
          </p>
        </div>
      </div>
    </div>
  );
}

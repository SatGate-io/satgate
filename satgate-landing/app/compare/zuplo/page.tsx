import Link from 'next/link';
import { ArrowLeft, Check, X, Minus, Shield, DollarSign, Zap, Eye, Lock, BarChart3, Code, Server } from 'lucide-react';

export const metadata = {
  title: 'SatGate vs Zuplo - Comparison',
  description: 'Compare SatGate and Zuplo API gateways. See why economic controls matter for the agent economy.',
  alternates: { canonical: 'https://satgate.io/compare/zuplo' },
  openGraph: {
    title: 'SatGate vs Zuplo: API Gateway vs Economic Governance',
    description: 'Compare Zuplo API gateway controls with SatGate economic governance for AI agent budgets, MCP tool costs, and L402 monetization.',
    url: 'https://satgate.io/compare/zuplo',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'SatGate vs Zuplo: API Gateway vs Economic Governance',
    description: 'Zuplo exposes APIs to agents. SatGate enforces budgets, MCP tool costs, scoped delegation, and L402 payments.',
  },
};

export default function CompareZuploPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'SatGate vs Zuplo',
    url: 'https://satgate.io/compare/zuplo',
    description: 'Compare SatGate and Zuplo API gateways for AI agent traffic, MCP exposure, budget enforcement, and L402 API monetization.',
    datePublished: '2026-04-30',
    dateModified: '2026-05-04',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'Zuplo alternative for AI agents' },
      { '@type': 'Thing', name: 'AI agent API gateway comparison' },
      { '@type': 'Thing', name: 'economic controls for API gateways' },
      { '@type': 'Thing', name: 'MCP budget enforcement' },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is the main difference between SatGate and Zuplo?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Zuplo is a traditional API gateway with MCP support for exposing APIs to agents. SatGate adds economic governance: per-tool cost attribution, hard budget enforcement, scoped delegation, and L402 monetization.',
        },
      },
      {
        '@type': 'Question',
        name: 'Does rate limiting replace economic controls for AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'No. Rate limits control request volume, not business cost. AI agents need dollar-denominated, per-agent, and per-tool controls so expensive calls can be capped before spend occurs.',
        },
      },
      {
        '@type': 'Question',
        name: 'Can SatGate work with an existing API gateway like Zuplo?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Yes. SatGate can be deployed as the economic policy layer in front of, behind, or alongside an existing API gateway so teams can keep API management while adding agent spend governance.',
        },
      },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <div className="max-w-4xl mx-auto px-6 py-16">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Home
        </Link>
        
        <div className="mb-12">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono mb-6">
            Comparison
          </div>
          <h1 className="text-4xl font-bold mb-4">SatGate vs Zuplo</h1>
          <p className="text-xl text-gray-400">
            Both protect APIs. One adds MCP. The other adds economics.
          </p>
        </div>

        {/* TL;DR */}
        <section className="bg-gradient-to-r from-cyan-900/20 to-purple-900/20 border border-cyan-800/30 rounded-xl p-6 mb-12">
          <h2 className="text-lg font-bold text-white mb-3">TL;DR</h2>
          <p className="text-gray-400 leading-relaxed">
            <strong className="text-white">Zuplo</strong> is a mature API gateway that added MCP support — great for exposing your APIs to AI agents securely. 
            <strong className="text-white"> SatGate</strong> is purpose-built for the agent economy — with hard budget enforcement, 
            per-tool cost attribution, and API monetization. Zuplo helps you expose APIs. SatGate helps you survive after you do.
          </p>
        </section>

        {/* Feature Comparison Table */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold mb-6">Feature Comparison</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800">
                  <th className="text-left py-4 text-gray-400 font-medium">Feature</th>
                  <th className="text-center py-4 text-cyan-400 font-medium">SatGate</th>
                  <th className="text-center py-4 text-gray-400 font-medium">Zuplo</th>
                </tr>
              </thead>
              <tbody className="text-gray-400">
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Hard budget enforcement</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><X className="inline text-red-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Per-tool cost attribution</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><X className="inline text-red-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">API monetization (L402)</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><X className="inline text-red-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Fiat-denominated budget control</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><X className="inline text-red-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">MCP Server Handler</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Traditional API gateway</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Rate limiting</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">GraphQL-to-MCP</td>
                  <td className="py-4 text-center"><Minus className="inline text-gray-500" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">SOC 2 certified</td>
                  <td className="py-4 text-center"><Minus className="inline text-gray-500" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Open source</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><X className="inline text-red-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Self-hosted option</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><Minus className="inline text-gray-500" size={18} /></td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        {/* Key Differences */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold mb-6">Key Differences</h2>
          
          <div className="grid gap-6">
            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-red-900/30 rounded-lg">
                  <DollarSign className="text-red-400" size={20} />
                </div>
                <h3 className="font-bold text-lg">Economic Controls</h3>
              </div>
              <div className="grid md:grid-cols-2 gap-4">
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-cyan-400 font-medium text-sm mb-2">SatGate</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-white">Purpose-built for agent economics.</strong> Hard budget caps, 
                    per-tool cost profiles, spend governance, fiat-denominated budget governance, and API monetization with L402.
                  </p>
                </div>
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-gray-500 font-medium text-sm mb-2">Zuplo</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-gray-300">Traditional API management.</strong> Rate limiting, authentication, 
                    request validation. No native economic controls or API monetization.
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-purple-900/30 rounded-lg">
                  <Code className="text-purple-400" size={20} />
                </div>
                <h3 className="font-bold text-lg">MCP Approach</h3>
              </div>
              <div className="grid md:grid-cols-2 gap-4">
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-cyan-400 font-medium text-sm mb-2">SatGate</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-white">MCP + economics.</strong> Parse MCP tool calls, 
                    apply per-tool cost profiles, enforce budgets at the tool level. Cost attribution by tool name.
                  </p>
                </div>
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-gray-500 font-medium text-sm mb-2">Zuplo</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-gray-300">MCP exposure.</strong> Turn your existing APIs into MCP tools 
                    with their Server Handler. Great for making APIs agent-accessible.
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-green-900/30 rounded-lg">
                  <Server className="text-green-400" size={20} />
                </div>
                <h3 className="font-bold text-lg">Deployment Model</h3>
              </div>
              <div className="grid md:grid-cols-2 gap-4">
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-cyan-400 font-medium text-sm mb-2">SatGate</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-white">Open-core, self-hostable.</strong> Run in your VPC, 
                    your Kubernetes cluster, or use SatGate Cloud. Full control over your data.
                  </p>
                </div>
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-gray-500 font-medium text-sm mb-2">Zuplo</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-gray-300">Managed SaaS.</strong> Edge-deployed, globally distributed. 
                    Less infrastructure management, but data flows through their network.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* When to Use Each */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold mb-6">When to Use Each</h2>
          
          <div className="grid md:grid-cols-2 gap-6">
            <div className="bg-gradient-to-br from-cyan-950/30 to-cyan-900/10 border border-cyan-800/30 rounded-xl p-6">
              <h3 className="font-bold text-lg text-cyan-400 mb-4">Use SatGate when...</h3>
              <ul className="space-y-3 text-gray-400 text-sm">
                <li className="flex items-start gap-2">
                  <Check className="text-cyan-400 mt-0.5 flex-shrink-0" size={16} />
                  <span>You need to <strong className="text-white">control what agents spend</strong> on your APIs</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-cyan-400 mt-0.5 flex-shrink-0" size={16} />
                  <span>You want <strong className="text-white">per-tool cost attribution</strong> for MCP calls</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-cyan-400 mt-0.5 flex-shrink-0" size={16} />
                  <span>You're <strong className="text-white">monetizing API access</strong> with micropayments</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-cyan-400 mt-0.5 flex-shrink-0" size={16} />
                  <span>You need <strong className="text-white">self-hosted deployment</strong> in your infrastructure</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-cyan-400 mt-0.5 flex-shrink-0" size={16} />
                  <span>You want <strong className="text-white">open-source transparency</strong> and auditability</span>
                </li>
              </ul>
            </div>

            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
              <h3 className="font-bold text-lg text-gray-400 mb-4">Use Zuplo when...</h3>
              <ul className="space-y-3 text-gray-400 text-sm">
                <li className="flex items-start gap-2">
                  <Check className="text-gray-500 mt-0.5 flex-shrink-0" size={16} />
                  <span>You need a <strong className="text-gray-300">mature, SOC 2 certified</strong> API gateway</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-gray-500 mt-0.5 flex-shrink-0" size={16} />
                  <span>You want to <strong className="text-gray-300">expose existing APIs as MCP tools</strong> quickly</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-gray-500 mt-0.5 flex-shrink-0" size={16} />
                  <span>You have <strong className="text-gray-300">GraphQL APIs</strong> to convert to MCP</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-gray-500 mt-0.5 flex-shrink-0" size={16} />
                  <span>You prefer <strong className="text-gray-300">fully-managed SaaS</strong> with edge deployment</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-gray-500 mt-0.5 flex-shrink-0" size={16} />
                  <span>Your primary need is <strong className="text-gray-300">traditional API management</strong></span>
                </li>
              </ul>
            </div>
          </div>
        </section>

        {/* The Economic Layer */}
        <section className="bg-gradient-to-r from-yellow-900/20 to-red-900/20 border border-yellow-800/30 rounded-xl p-6 mb-12">
          <h2 className="text-xl font-bold mb-4">The Missing Layer</h2>
          <p className="text-gray-400 leading-relaxed mb-4">
            Zuplo is excellent at what it does — secure API management with MCP support. 
            But when AI agents start calling your APIs autonomously, <strong className="text-white">who controls the spend?</strong>
          </p>
          <p className="text-gray-400 leading-relaxed">
            Rate limiting tells you "how many requests per minute." 
            Economic controls tell you "how many dollars per agent." 
            That's the layer SatGate adds.
          </p>
        </section>

        <section className="bg-gray-950 border border-gray-800 rounded-xl p-6 mb-12">
          <p className="mb-2 text-sm font-mono uppercase tracking-wide text-cyan-300">FAQ</p>
          <h2 className="text-xl font-bold mb-6">SatGate vs Zuplo questions</h2>
          <div className="space-y-5">
            {[
              ['What is the main difference between SatGate and Zuplo?', 'Zuplo is a traditional API gateway with MCP support for exposing APIs to agents. SatGate adds economic governance: per-tool cost attribution, hard budget enforcement, scoped delegation, and L402 monetization.'],
              ['Does rate limiting replace economic controls for AI agents?', 'No. Rate limits control request volume, not business cost. AI agents need dollar-denominated, per-agent, and per-tool controls so expensive calls can be capped before spend occurs.'],
              ['Can SatGate work with an existing API gateway like Zuplo?', 'Yes. SatGate can be deployed as the economic policy layer in front of, behind, or alongside an existing API gateway so teams can keep API management while adding agent spend governance.'],
            ].map(([question, answer]) => (
              <div key={question} className="border-t border-gray-800 pt-5 first:border-t-0 first:pt-0">
                <h3 className="mb-2 text-lg font-bold text-white">{question}</h3>
                <p className="leading-relaxed text-gray-400">{answer}</p>
              </div>
            ))}
          </div>
        </section>

        {/* CTA */}
        <section className="text-center">
          <h2 className="text-2xl font-bold mb-4">Ready to add economic controls?</h2>
          <p className="text-gray-400 mb-8">
            Start with free Observe mode. See what your agents are actually spending.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link 
              href="/design-partners" 
              className="inline-flex items-center justify-center gap-2 bg-white text-black px-8 py-3 rounded-lg font-bold hover:bg-gray-200 transition"
            >
              Apply for Design Partner Program
            </Link>
            <a 
              href="https://github.com/SatGate-io/satgate" 
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center gap-2 bg-gray-800 text-white px-8 py-3 rounded-lg font-bold hover:bg-gray-700 transition"
            >
              View on GitHub
            </a>
          </div>
        </section>

      </div>
    </div>
  );
}

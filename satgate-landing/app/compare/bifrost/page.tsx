import Link from 'next/link';
import { ArrowLeft, Check, X, Minus, Shield, DollarSign, Zap, Eye, Lock, BarChart3, Code, Server } from 'lucide-react';

export const metadata = {
  title: 'SatGate vs Bifrost - Comparison',
  description: 'Compare SatGate and Bifrost AI gateways. See why economic controls matter for the agent economy.',
};

export default function CompareBifrostPage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <div className="max-w-4xl mx-auto px-6 py-16">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Home
        </Link>
        
        <div className="mb-12">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono mb-6">
            Comparison
          </div>
          <h1 className="text-4xl font-bold mb-4">SatGate vs Bifrost</h1>
          <p className="text-xl text-gray-400">
            Both are AI gateways. One routes calls. The other governs spend.
          </p>
        </div>

        {/* TL;DR */}
        <section className="bg-gradient-to-r from-cyan-900/20 to-purple-900/20 border border-cyan-800/30 rounded-xl p-6 mb-12">
          <h2 className="text-lg font-bold text-white mb-3">TL;DR</h2>
          <p className="text-gray-400 leading-relaxed">
            <strong className="text-white">Bifrost</strong> is excellent at routing LLM calls efficiently across providers with minimal latency. 
            <strong className="text-white"> SatGate</strong> is built for <em>economic governance</em> — hard budget enforcement, 
            per-tool cost attribution, and monetization. If your agents need to call LLMs fast, use Bifrost. 
            If you need to control what they spend, use SatGate.
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
                  <th className="text-center py-4 text-gray-400 font-medium">Bifrost</th>
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
                  <td className="py-4 font-medium text-gray-300">Fiat billing (Fiat402)</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><X className="inline text-red-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">MCP-aware</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Multi-provider LLM routing</td>
                  <td className="py-4 text-center"><Minus className="inline text-gray-500" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Automatic LLM failover</td>
                  <td className="py-4 text-center"><Minus className="inline text-gray-500" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Semantic caching</td>
                  <td className="py-4 text-center"><Minus className="inline text-gray-500" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">General API protection</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><Minus className="inline text-gray-500" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Open source</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                </tr>
                <tr className="border-b border-gray-800">
                  <td className="py-4 font-medium text-gray-300">Enterprise dashboard</td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
                  <td className="py-4 text-center"><Check className="inline text-green-400" size={18} /></td>
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
                <h3 className="font-bold text-lg">Budget Enforcement</h3>
              </div>
              <div className="grid md:grid-cols-2 gap-4">
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-cyan-400 font-medium text-sm mb-2">SatGate</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-white">Hard enforcement.</strong> When budget is exceeded, requests are blocked. 
                    No "oops" moments. The CFO knows exactly what will be spent.
                  </p>
                </div>
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-gray-500 font-medium text-sm mb-2">Bifrost</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-gray-300">Soft limits.</strong> Budget tracking with alerts, but requests still go through. 
                    You find out what you spent after the fact.
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-purple-900/30 rounded-lg">
                  <BarChart3 className="text-purple-400" size={20} />
                </div>
                <h3 className="font-bold text-lg">Cost Attribution</h3>
              </div>
              <div className="grid md:grid-cols-2 gap-4">
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-cyan-400 font-medium text-sm mb-2">SatGate</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-white">Per-tool granularity.</strong> See exactly which MCP tool, which agent, 
                    which team spent what. Chargeback reports by cost center.
                  </p>
                </div>
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-gray-500 font-medium text-sm mb-2">Bifrost</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-gray-300">Request-level tracking.</strong> You see which requests were made, 
                    but no native cost attribution per tool or team.
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-green-900/30 rounded-lg">
                  <Zap className="text-green-400" size={20} />
                </div>
                <h3 className="font-bold text-lg">Primary Use Case</h3>
              </div>
              <div className="grid md:grid-cols-2 gap-4">
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-cyan-400 font-medium text-sm mb-2">SatGate</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-white">Economic governance.</strong> Protect APIs from runaway agent spend. 
                    Monetize API access. Control budgets at the request level.
                  </p>
                </div>
                <div className="bg-gray-800/50 rounded-lg p-4">
                  <p className="text-gray-500 font-medium text-sm mb-2">Bifrost</p>
                  <p className="text-gray-400 text-sm">
                    <strong className="text-gray-300">LLM routing.</strong> Efficiently route calls across multiple LLM providers. 
                    Automatic failover and load balancing. Minimize latency.
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
                  <span>You need to <strong className="text-white">enforce hard budget caps</strong> on agent spend</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-cyan-400 mt-0.5 flex-shrink-0" size={16} />
                  <span>You want <strong className="text-white">per-tool cost attribution</strong> for chargeback reporting</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-cyan-400 mt-0.5 flex-shrink-0" size={16} />
                  <span>You're <strong className="text-white">monetizing API access</strong> (charging per call)</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-cyan-400 mt-0.5 flex-shrink-0" size={16} />
                  <span>You're protecting <strong className="text-white">REST APIs, MCP tools, or any HTTP endpoint</strong></span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-cyan-400 mt-0.5 flex-shrink-0" size={16} />
                  <span>The CFO asks <strong className="text-white">"how do we control AI spend?"</strong></span>
                </li>
              </ul>
            </div>

            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
              <h3 className="font-bold text-lg text-gray-400 mb-4">Use Bifrost when...</h3>
              <ul className="space-y-3 text-gray-400 text-sm">
                <li className="flex items-start gap-2">
                  <Check className="text-gray-500 mt-0.5 flex-shrink-0" size={16} />
                  <span>You need to <strong className="text-gray-300">route LLM calls</strong> across multiple providers</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-gray-500 mt-0.5 flex-shrink-0" size={16} />
                  <span>You want <strong className="text-gray-300">automatic failover</strong> if one provider goes down</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-gray-500 mt-0.5 flex-shrink-0" size={16} />
                  <span>You're optimizing for <strong className="text-gray-300">minimum latency</strong> on LLM calls</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-gray-500 mt-0.5 flex-shrink-0" size={16} />
                  <span>You need <strong className="text-gray-300">semantic caching</strong> to reduce costs</span>
                </li>
                <li className="flex items-start gap-2">
                  <Check className="text-gray-500 mt-0.5 flex-shrink-0" size={16} />
                  <span>Your primary concern is <strong className="text-gray-300">LLM infrastructure reliability</strong></span>
                </li>
              </ul>
            </div>
          </div>
        </section>

        {/* Can You Use Both? */}
        <section className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-12">
          <h2 className="text-xl font-bold mb-4">Can You Use Both?</h2>
          <p className="text-gray-400 leading-relaxed mb-4">
            Yes. They solve different problems and can be deployed together:
          </p>
          <div className="bg-gray-800/50 rounded-lg p-4 font-mono text-sm text-gray-300">
            Agent → <span className="text-cyan-400">SatGate</span> (budget enforcement) → <span className="text-purple-400">Bifrost</span> (LLM routing) → OpenAI/Anthropic/etc.
          </div>
          <p className="text-gray-500 text-sm mt-4">
            SatGate enforces economic controls. Bifrost optimizes the LLM layer. Different layers, complementary value.
          </p>
        </section>

        {/* CTA */}
        <section className="text-center">
          <h2 className="text-2xl font-bold mb-4">Ready to control agent spend?</h2>
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

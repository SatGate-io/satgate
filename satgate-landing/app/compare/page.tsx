import Link from 'next/link';
import { ArrowLeft, ArrowRight, Route, Server, Shield, Zap } from 'lucide-react';

export const metadata = {
  title: 'Compare SatGate - AI Gateway and Economic Firewall Comparisons',
  description: 'Compare SatGate with LiteLLM, Bifrost, Zuplo, cloud-native API gateways, and other AI infrastructure. Routing is not economic governance.',
  alternates: { canonical: 'https://satgate.io/compare' },
};

export default function ComparePage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <div className="max-w-4xl mx-auto px-6 py-16">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Home
        </Link>
        
        <div className="mb-12">
          <h1 className="text-4xl font-bold mb-4">Compare SatGate</h1>
          <p className="text-xl text-gray-400">
            See how the Economic Firewall stacks up against other solutions.
          </p>
        </div>

        <div className="grid gap-6">

          <Link
            href="/compare/litellm"
            className="block bg-gray-900 border border-gray-800 rounded-xl p-6 hover:border-green-600/50 transition group"
          >
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-3 mb-2">
                  <div className="p-2 bg-green-900/30 rounded-lg">
                    <Route className="text-green-400" size={20} />
                  </div>
                  <h2 className="text-xl font-bold">SatGate vs LiteLLM</h2>
                </div>
                <p className="text-gray-400">
                  LLM gateway vs economic firewall. LiteLLM routes model access; SatGate governs agent/API spend, MCP tools, and payments.
                </p>
              </div>
              <ArrowRight className="text-gray-600 group-hover:text-green-400 transition" size={24} />
            </div>
          </Link>

          {/* Bifrost */}
          <Link 
            href="/compare/bifrost"
            className="block bg-gray-900 border border-gray-800 rounded-xl p-6 hover:border-purple-600/50 transition group"
          >
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-3 mb-2">
                  <div className="p-2 bg-purple-900/30 rounded-lg">
                    <Zap className="text-purple-400" size={20} />
                  </div>
                  <h2 className="text-xl font-bold">SatGate vs Bifrost</h2>
                </div>
                <p className="text-gray-400">
                  LLM routing vs economic governance. Both have MCP — only one enforces per-tool budgets on agent tool calls.
                </p>
              </div>
              <ArrowRight className="text-gray-600 group-hover:text-purple-400 transition" size={24} />
            </div>
          </Link>

          {/* Zuplo */}
          <Link 
            href="/compare/zuplo"
            className="block bg-gray-900 border border-gray-800 rounded-xl p-6 hover:border-cyan-600/50 transition group"
          >
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-3 mb-2">
                  <div className="p-2 bg-cyan-900/30 rounded-lg">
                    <Shield className="text-cyan-400" size={20} />
                  </div>
                  <h2 className="text-xl font-bold">SatGate vs Zuplo</h2>
                </div>
                <p className="text-gray-400">
                  API gateway vs economic firewall. Zuplo exposes APIs — SatGate governs spend.
                </p>
              </div>
              <ArrowRight className="text-gray-600 group-hover:text-cyan-400 transition" size={24} />
            </div>
          </Link>

          {/* Coming Soon */}
          <div className="bg-gray-900/50 border border-gray-800/50 rounded-xl p-6 opacity-60">
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-3 mb-2">
                  <div className="p-2 bg-gray-800 rounded-lg">
                    <Server className="text-gray-500" size={20} />
                  </div>
                  <h2 className="text-xl font-bold text-gray-500">More comparisons coming</h2>
                </div>
                <p className="text-gray-600">
                  Portkey, Helicone, Cloudflare AI Gateway, Kong, AWS API Gateway, Aperture, and more.
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Summary */}
        <section className="mt-12 bg-gradient-to-r from-cyan-900/20 to-purple-900/20 border border-cyan-800/30 rounded-xl p-6">
          <h2 className="text-lg font-bold text-white mb-3">The Short Version</h2>
          <p className="text-gray-400 leading-relaxed">
            Most AI gateways focus on <strong className="text-white">routing/model access</strong> (LiteLLM, Bifrost) or{' '}
            <strong className="text-white">API management</strong> (Zuplo, Kong). SatGate focuses on{' '}
            <strong className="text-cyan-400">economic governance</strong> — hard budget enforcement, 
            per-tool cost attribution via MCP proxy, delegation hierarchies for sub-agents, and API monetization. 
            Different problem, different solution.
          </p>
        </section>

      </div>
    </div>
  );
}

'use client';

import React, { useState } from 'react';
import { Terminal, Code, Cpu, Zap, ArrowRight, CheckCircle, Copy, Check } from 'lucide-react';
import Link from 'next/link';

const LandingPage = () => {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = async () => {
    await navigator.clipboard.writeText('pip install satgate');
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans selection:bg-purple-500 selection:text-white">
      
      {/* Navigation */}
      <nav className="border-b border-gray-800 backdrop-blur-md fixed w-full z-50 bg-black/50">
        <div className="max-w-6xl mx-auto px-6 h-16 flex items-center justify-between">
          <div className="text-xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-purple-400 to-cyan-400">
            ⚡ SatGate
          </div>
          <div className="flex gap-6 text-sm font-medium text-gray-400">
            <Link href="#features" className="hover:text-white transition">Features</Link>
            <Link href="/playground" className="hover:text-white transition">Playground</Link>
            <a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <header className="pt-32 pb-20 px-6">
        <div className="max-w-6xl mx-auto grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
          
          {/* Left: Copy */}
          <div>
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono mb-6">
              <Zap size={12} /> v1.0 Now Live: Python SDK Support
            </div>
            <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight mb-6">
              The Economic Layer for <br/>
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-cyan-400">
                Autonomous Agents
              </span>
            </h1>
            <p className="text-xl text-gray-400 mb-8 max-w-lg leading-relaxed">
              Agents can't pass KYC. They can't hold bank accounts. 
              SatGate gives them a <b>Lightning Wallet</b> and an <b>L402 Gateway</b> to buy data, API calls, and compute instantly.
            </p>
            <div className="flex gap-4">
              <Link href="/playground" className="bg-white text-black px-8 py-3 rounded-lg font-bold hover:bg-gray-200 transition flex items-center gap-2">
                Try Demo <ArrowRight size={18} />
              </Link>
              <button 
                onClick={copyToClipboard}
                className="border border-gray-700 px-8 py-3 rounded-lg font-bold hover:border-gray-500 transition font-mono flex items-center gap-2 group"
              >
                pip install satgate
                {copied ? (
                  <Check size={16} className="text-green-400" />
                ) : (
                  <Copy size={16} className="text-gray-500 group-hover:text-gray-300 transition" />
                )}
              </button>
            </div>
          </div>

          {/* Right: The Terminal Demo (Social Proof) */}
          <div className="relative group">
            <div className="absolute -inset-1 bg-gradient-to-r from-purple-600 to-cyan-600 rounded-xl blur opacity-25 group-hover:opacity-50 transition duration-1000"></div>
            <div className="relative bg-gray-900 rounded-xl border border-gray-800 p-4 shadow-2xl">
              <div className="flex items-center gap-2 mb-4 border-b border-gray-800 pb-4">
                <div className="w-3 h-3 rounded-full bg-red-500"></div>
                <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
                <div className="w-3 h-3 rounded-full bg-green-500"></div>
                <div className="text-xs text-gray-500 ml-2 font-mono">agent_demo.py — Python</div>
              </div>
              <div className="font-mono text-sm space-y-2 h-64 overflow-hidden text-gray-300">
                <div className="text-gray-500"># The agent hits a 402 Paywall automatically</div>
                <div className="flex gap-2">
                  <span className="text-green-400">❯</span>
                  <span>agent.run("Fetch premium market data")</span>
                </div>
                <div className="text-yellow-500 opacity-80 mt-2">
                  ⚡ 402 Detected. Invoice: lnbc10u...
                </div>
                <div className="text-blue-400 opacity-80">
                  💰 WALLET: Payment Sent! (Preimage: a1b2...)
                </div>
                <div className="text-purple-400 opacity-80">
                  🔄 Retrying with L402 Token...
                </div>
                <div className="mt-4 text-white">
                  Response: &#123; "market_sentiment": "bullish" &#125;
                </div>
                <div className="text-green-500 mt-2">
                  ✅ Mission Complete.
                </div>
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Feature Grid */}
      <section id="features" className="py-20 bg-gray-900/50 border-y border-gray-800">
        <div className="max-w-6xl mx-auto px-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            
            <FeatureCard 
              icon={<Cpu className="text-cyan-400" />}
              title="LangChain Native"
              desc="Plug-and-play 'Tool' for LangChain, AutoGPT, and CrewAI. Give your agent a wallet in 2 lines of code."
            />
            <FeatureCard 
              icon={<Terminal className="text-purple-400" />}
              title="L402 Standard"
              desc="Built on the open L402 protocol. No proprietary lock-in. Compatible with any Lightning Node."
            />
            <FeatureCard 
              icon={<Code className="text-pink-400" />}
              title="Micropayments"
              desc="Monetize per-request. Charge $0.001 per API call. Impossible with Stripe, trivial with SatGate."
            />
          </div>
        </div>
      </section>

      {/* Code Integration Section */}
      <section className="py-20 px-6">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">Universal Integration</h2>
          
          <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
            <div className="flex border-b border-gray-800">
              <button className="px-6 py-3 bg-gray-800 text-white font-mono text-sm border-r border-gray-700">Python (Agents)</button>
              <button className="px-6 py-3 text-gray-500 hover:text-white font-mono text-sm transition">Node.js</button>
              <button className="px-6 py-3 text-gray-500 hover:text-white font-mono text-sm transition">cURL</button>
            </div>
            <div className="p-8 overflow-x-auto">
<pre className="text-sm font-mono text-gray-300 leading-relaxed">
<span className="text-purple-400">from</span> satgate.langchain <span className="text-purple-400">import</span> SatGateTool<br/>
<span className="text-purple-400">from</span> langchain.agents <span className="text-purple-400">import</span> initialize_agent<br/>
<br/>
<span className="text-gray-500"># 1. Give your agent a wallet</span><br/>
tools = [SatGateTool(wallet=my_lnd_node)]<br/>
<br/>
<span className="text-gray-500"># 2. Let it roam the economy</span><br/>
agent = initialize_agent(tools, llm, agent=<span className="text-green-400">"openai-functions"</span>)<br/>
agent.run(<span className="text-green-400">"Buy the latest stock report from AlphaVantage"</span>)
</pre>
            </div>
          </div>
        </div>
      </section>

      {/* CTA / Footer */}
      <footer className="py-20 border-t border-gray-800">
        <div className="max-w-6xl mx-auto px-6">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-6">Ready to monetize the robots?</h2>
            <a href="mailto:contact@satgate.io" className="inline-block bg-gradient-to-r from-purple-600 to-cyan-600 text-white px-10 py-4 rounded-full font-bold text-lg hover:opacity-90 transition shadow-lg shadow-purple-500/20">
              Get in Touch
            </a>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8 py-12 border-t border-gray-800">
            <div>
              <h4 className="font-bold text-white mb-4">⚡ SatGate</h4>
              <p className="text-gray-500 text-sm">Stripe for AI Agents.<br/>EZ-Pass for the API Economy.</p>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Resources</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="https://github.com/SatGate-io/satgate" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">GitHub</a></li>
                <li><Link href="/playground" className="hover:text-white transition">Playground</Link></li>
                <li><a href="https://github.com/SatGate-io/satgate#-quick-start" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Documentation</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold text-white mb-4">Contact</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="mailto:contact@satgate.io" className="hover:text-white transition">contact@satgate.io</a></li>
                <li><a href="https://twitter.com/SatGate_io" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Twitter/X</a></li>
              </ul>
            </div>
          </div>
          
          <div className="pt-8 border-t border-gray-800 text-center text-gray-600 text-sm">
            © 2025 SatGate. Patent Pending. • Built for the machine economy.
          </div>
        </div>
      </footer>
    </div>
  );
};

// Simple helper component for features
const FeatureCard = ({ icon, title, desc }: any) => (
  <div className="p-6 rounded-xl bg-black border border-gray-800 hover:border-gray-600 transition group">
    <div className="mb-4 p-3 bg-gray-900 rounded-lg w-fit group-hover:bg-gray-800 transition">{icon}</div>
    <h3 className="text-xl font-bold mb-2">{title}</h3>
    <p className="text-gray-400 leading-relaxed">{desc}</p>
  </div>
);

export default LandingPage;

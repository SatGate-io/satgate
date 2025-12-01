'use client';

import React, { useState } from 'react';
import { Terminal, Code, Cpu, Zap, ArrowRight, CheckCircle, Copy, Check, Shield, Key, Lock, Clock } from 'lucide-react';
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

          {/* Right: Hero Demo Video */}
          <div className="relative group">
            <div className="absolute -inset-1 bg-gradient-to-r from-purple-600 to-cyan-600 rounded-xl blur opacity-25 group-hover:opacity-50 transition duration-1000"></div>
            <div className="relative bg-gray-900 rounded-xl border border-gray-800 overflow-hidden shadow-2xl">
              <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-800 bg-gray-900/80">
                <div className="w-3 h-3 rounded-full bg-red-500"></div>
                <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
                <div className="w-3 h-3 rounded-full bg-green-500"></div>
                <div className="text-xs text-gray-500 ml-2 font-mono">hero_demo.py — Live Demo</div>
              </div>
              <video 
                autoPlay 
                loop 
                muted 
                playsInline
                className="w-full"
                poster="/demo-poster.png"
              >
                <source src="/satgate-hero-demo.mp4" type="video/mp4" />
                {/* Fallback for browsers that don't support video */}
                <img src="/satgate-hero-demo.gif" alt="SatGate Demo" className="w-full" />
              </video>
              <div className="absolute bottom-4 right-4 bg-black/70 backdrop-blur-sm px-3 py-1 rounded-full text-xs text-gray-300 font-mono">
                ⚡ Metered in sats
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Feature Grid */}
      <section id="features" className="py-20 bg-gray-900/50 border-y border-gray-800">
        <div className="max-w-6xl mx-auto px-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
            
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
              desc="Meter in sats per call. Sub-cent pricing that's uneconomical on card rails, trivial with SatGate."
            />
            <FeatureCard 
              icon={<Shield className="text-green-400" />}
              title="Economic Friction for L7 Abuse"
              desc="High-volume scraping becomes expensive. Use alongside your WAF/CDN for volumetric protection."
            />
          </div>
        </div>
      </section>

      {/* Capability-Based Security Section */}
      <section className="py-20 px-6 bg-gradient-to-b from-black to-gray-900/50">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-4">Internet-Native Access Control</h2>
            <p className="text-gray-400 max-w-2xl mx-auto">
              SatGate implements L402 to turn payment into a <span className="text-purple-400 font-semibold">capability token</span> for APIs—ideal for stateless AI agents and zero-PII integrations.
            </p>
          </div>
          
          {/* Identity vs Capability comparison */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-12">
            <div className="p-6 rounded-xl bg-red-950/20 border border-red-900/30">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-red-900/30 rounded-lg">
                  <Lock className="text-red-400" size={20} />
                </div>
                <h3 className="font-bold text-red-300">Identity-Based (Traditional)</h3>
              </div>
              <p className="text-gray-400 text-sm mb-4">"Prove <em>who you are</em>, then we decide what you can do."</p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li>• Requires user databases & PII</li>
                <li>• OAuth flows, API keys, sessions</li>
                <li>• Credential stuffing risks</li>
                <li>• Agents can't sign up</li>
              </ul>
            </div>
            <div className="p-6 rounded-xl bg-green-950/20 border border-green-900/30">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-green-900/30 rounded-lg">
                  <Key className="text-green-400" size={20} />
                </div>
                <h3 className="font-bold text-green-300">Capability-Based (L402)</h3>
              </div>
              <p className="text-gray-400 text-sm mb-4">"Present a token that <em>already encodes</em> what you can do."</p>
              <ul className="space-y-2 text-sm text-gray-500">
                <li className="text-green-400">✓ No accounts required</li>
                <li className="text-green-400">✓ Payment-gated authorization</li>
                <li className="text-green-400">✓ Zero PII, no credentials to steal</li>
                <li className="text-green-400">✓ Perfect for autonomous agents</li>
              </ul>
            </div>
          </div>

          {/* Security Features */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <div className="flex items-center gap-3 mb-3">
                <Shield className="text-cyan-400" size={18} />
                <h4 className="font-semibold">Edge Verification</h4>
              </div>
              <p className="text-gray-500 text-sm">Tokens verified cryptographically at the gateway. No centralized identity store needed. Usage accounting/quotas can be tracked without storing PII.</p>
            </div>
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <div className="flex items-center gap-3 mb-3">
                <Clock className="text-purple-400" size={18} />
                <h4 className="font-semibold">Least Privilege</h4>
              </div>
              <p className="text-gray-500 text-sm">Add caveats to constrain scope, time, and budget (e.g., "valid 5 mins", "max 10 calls").</p>
            </div>
            <div className="p-5 rounded-xl bg-gray-900 border border-gray-800">
              <div className="flex items-center gap-3 mb-3">
                <Zap className="text-yellow-400" size={18} />
                <h4 className="font-semibold">Economic Friction</h4>
              </div>
              <p className="text-gray-500 text-sm">Spam becomes expensive and self-limiting. High-volume callers must continuously pay.</p>
            </div>
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
          
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8 py-12 border-t border-gray-800">
            <div>
              <h4 className="font-bold text-white mb-4">⚡ SatGate</h4>
              <p className="text-gray-500 text-sm">Stripe for AI Agents.<br/>EZ-Pass for the API Economy.</p>
              <p className="text-gray-600 text-xs mt-3">Non-custodial. We never hold your keys.</p>
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
                <li><a href="https://twitter.com/SatGate_io" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Twitter/X</a></li>
                <li><a href="mailto:security@satgate.io" className="hover:text-white transition">security@satgate.io</a></li>
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

'use client';

import React, { useState } from 'react';
import { Terminal, Code, Cpu, Zap, ArrowRight, CheckCircle, Copy, Check, Shield, Key, Lock, Clock, DollarSign, Bot, GitBranch, Activity, RefreshCw } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

const LandingPage = () => {
  const [copied, setCopied] = useState(false);
  const [activeTab, setActiveTab] = useState<'python' | 'nodejs' | 'curl'>('python');

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
          <Link href="/" className="flex items-center gap-2">
            <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="w-8 h-8" />
            <span className="text-xl font-bold text-white">SatGate</span>
          </Link>
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
              SatGate gives them a <b>Lightning Wallet</b> and an <b>L402 Gateway</b> to buy data, API calls, and compute instantly—and route to the best provider based on price, latency, and availability.
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
            <p className="text-center text-sm text-gray-500 mt-4">
              Watch an AI agent autonomously pay for API access in real-time.
            </p>
          </div>
        </div>
      </header>

      {/* Two Products Section */}
      <section className="py-16 px-6 border-b border-gray-800">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-2xl font-bold text-center mb-3">Two Products in One</h2>
          <p className="text-gray-500 text-center mb-10">Use SatGate for either—or both.</p>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="p-6 rounded-xl bg-gradient-to-br from-purple-950/30 to-purple-900/10 border border-purple-800/30 hover:border-purple-600/50 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-purple-900/50 rounded-lg">
                  <DollarSign className="text-purple-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Monetize APIs per request</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed">
                Charge 1 sat per call—sub-cent pricing that's impossible on card rails. No minimums, no chargebacks, instant settlement.
              </p>
            </div>
            <div className="p-6 rounded-xl bg-gradient-to-br from-cyan-950/30 to-cyan-900/10 border border-cyan-800/30 hover:border-cyan-600/50 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-cyan-900/50 rounded-lg">
                  <Bot className="text-cyan-400" size={22} />
                </div>
                <h3 className="font-bold text-lg">Secure agent traffic</h3>
              </div>
              <p className="text-gray-400 text-sm leading-relaxed">
                L402 tokens replace accounts and API keys. No PII, no credential stuffing. Paid capabilities for autonomous agents.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Agent Routing Section */}
      <section className="py-20 px-6 bg-gradient-to-b from-gray-900/30 to-black border-b border-gray-800">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-cyan-900/30 border border-cyan-500/30 text-cyan-300 text-xs font-mono mb-6">
              <GitBranch size={12} /> The Agent Era
            </div>
            <h2 className="text-3xl font-bold mb-4">Agents Route in Real Time. Humans Can't.</h2>
            <p className="text-gray-400 max-w-2xl mx-auto mb-4">
              In the agent era, switching APIs isn't a two-week integration project. With <span className="text-white font-medium">MCP</span> <span className="text-gray-500">(Model Context Protocol)</span> and agent tooling, providers publish price and health signals—agents route by policy, pay instantly, and get an L402 capability token.
            </p>
            <p className="text-cyan-400 font-medium">
              This enables an API marketplace where providers compete per request.
            </p>
          </div>

          {/* 3 Key Benefits */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-12">
            <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-cyan-800/50 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-cyan-900/30 rounded-lg">
                  <DollarSign className="text-cyan-400" size={20} />
                </div>
                <h3 className="font-bold">Dynamic Pricing</h3>
              </div>
              <p className="text-gray-500 text-sm leading-relaxed">
                Providers compete per call. Agents optimize spend automatically based on budget policies.
              </p>
            </div>
            <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-cyan-800/50 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-purple-900/30 rounded-lg">
                  <RefreshCw className="text-purple-400" size={20} />
                </div>
                <h3 className="font-bold">Instant Failover</h3>
              </div>
              <p className="text-gray-500 text-sm leading-relaxed">
                If one API is degraded, the agent switches on the next call. No human intervention, no downtime.
              </p>
            </div>
            <div className="p-6 rounded-xl bg-gray-900 border border-gray-800 hover:border-cyan-800/50 transition">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2.5 bg-green-900/30 rounded-lg">
                  <Zap className="text-green-400" size={20} />
                </div>
                <h3 className="font-bold">No Onboarding Drag</h3>
              </div>
              <p className="text-gray-500 text-sm leading-relaxed">
                Pay-to-authorize replaces accounts and API keys for machine clients. Instant access, any provider.
              </p>
            </div>
          </div>

          {/* Agent Decision Flow Diagram */}
          <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-8">
            <h4 className="text-center text-sm font-semibold text-gray-400 mb-6">AGENT DECISION LOOP</h4>
            <div className="flex flex-col md:flex-row items-center justify-center gap-3 md:gap-4">
              <div className="flex items-center gap-2 px-4 py-2 bg-gray-800 rounded-lg text-sm">
                <Activity size={16} className="text-cyan-400" />
                <span className="text-gray-300">Get quotes <span className="text-gray-500">(price, latency, uptime)</span></span>
              </div>
              <span className="text-gray-600 text-xl hidden md:block">→</span>
              <span className="text-gray-600 md:hidden">↓</span>
              <div className="flex items-center gap-2 px-4 py-2 bg-gray-800 rounded-lg text-sm">
                <GitBranch size={16} className="text-purple-400" />
                <span className="text-gray-300">Select best provider</span>
              </div>
              <span className="text-gray-600 text-xl hidden md:block">→</span>
              <span className="text-gray-600 md:hidden">↓</span>
              <div className="flex items-center gap-2 px-4 py-2 bg-purple-900/30 border border-purple-700/50 rounded-lg text-sm">
                <Zap size={16} className="text-yellow-400" />
                <span className="text-gray-300">Pay → receive L402 token</span>
              </div>
              <span className="text-gray-600 text-xl hidden md:block">→</span>
              <span className="text-gray-600 md:hidden">↓</span>
              <div className="flex items-center gap-2 px-4 py-2 bg-green-900/30 border border-green-700/50 rounded-lg text-sm">
                <CheckCircle size={16} className="text-green-400" />
                <span className="text-gray-300">Call API</span>
              </div>
            </div>
            <p className="text-center text-gray-600 text-xs mt-6">
              If timeout/error → agent automatically retries with next provider. SatGate is the primitive that makes this possible.
            </p>
          </div>
          
          <div className="text-center mt-8">
            <a 
              href="https://github.com/SatGate-io/satgate#-agent-routing-failover-example" 
              target="_blank" 
              rel="noopener noreferrer"
              className="text-sm text-cyan-400 hover:text-cyan-300 transition underline underline-offset-4"
            >
              See the Agent Routing Example →
            </a>
          </div>
        </div>
      </section>

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
            <p className="text-gray-400 max-w-2xl mx-auto mb-4">
              SatGate implements L402 to turn payment into a <span className="text-purple-400 font-semibold">capability token</span> for APIs—ideal for stateless AI agents and zero-PII integrations.
            </p>
            <Link href="/security" className="text-sm text-purple-400 hover:text-purple-300 transition underline underline-offset-4">
              Deep dive: Security architecture →
            </Link>
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

          {/* Zero Trust Blurb */}
          <div className="mt-8 p-5 rounded-xl bg-gradient-to-r from-cyan-900/20 to-purple-900/20 border border-cyan-800/30">
            <div className="flex items-start gap-4">
              <Shield className="text-cyan-400 flex-shrink-0 mt-1" size={20} />
              <div>
                <h4 className="font-semibold text-white mb-2">Zero Trust Access Control (PEP)</h4>
                <p className="text-gray-400 text-sm leading-relaxed">
                  Enforces per-request authorization at the edge using cryptographically verifiable, least-privilege capability tokens—no network trust assumptions. 
                  Use alongside your existing WAF/CDN for volumetric protection.
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* How L402 Works Section */}
      <section className="py-16 px-6 border-b border-gray-800">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-2xl font-bold text-center mb-3">How L402 Works</h2>
          <p className="text-gray-500 text-center mb-10">Three steps. No accounts. Instant access.</p>
          
          {/* 3-Step Flow */}
          <div className="flex flex-col md:flex-row items-center justify-center gap-4 md:gap-2 mb-10">
            <div className="flex-1 max-w-[200px] p-5 rounded-xl bg-gray-900 border border-gray-800 text-center">
              <div className="text-2xl font-bold text-purple-400 mb-2">1</div>
              <div className="font-semibold text-white mb-1">Request API</div>
              <div className="text-xs text-gray-500">Client hits protected endpoint</div>
            </div>
            <div className="text-gray-600 text-2xl hidden md:block">→</div>
            <div className="text-gray-600 text-xl md:hidden">↓</div>
            <div className="flex-1 max-w-[200px] p-5 rounded-xl bg-gray-900 border border-gray-800 text-center">
              <div className="text-2xl font-bold text-yellow-400 mb-2">2</div>
              <div className="font-semibold text-white mb-1">402 + Invoice</div>
              <div className="text-xs text-gray-500">Gateway returns Lightning invoice</div>
            </div>
            <div className="text-gray-600 text-2xl hidden md:block">→</div>
            <div className="text-gray-600 text-xl md:hidden">↓</div>
            <div className="flex-1 max-w-[200px] p-5 rounded-xl bg-gray-900 border border-gray-800 text-center">
              <div className="text-2xl font-bold text-cyan-400 mb-2">3</div>
              <div className="font-semibold text-white mb-1">Pay → Access</div>
              <div className="text-xs text-gray-500">Pay invoice, get L402 token</div>
            </div>
          </div>

          {/* Token explanation */}
          <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-6 text-center">
            <p className="text-gray-400 text-sm mb-2">
              <span className="text-white font-semibold">L402 Token</span> = Macaroon + Preimage
            </p>
            <p className="text-gray-500 text-xs">
              A bearer credential with embedded permissions (caveats) that proves payment. No centralized identity lookup required.
            </p>
          </div>
        </div>
      </section>

      {/* Code Integration Section */}
      <section className="py-20 px-6">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">Universal Integration</h2>
          
          <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
            <div className="flex border-b border-gray-800">
              <button 
                onClick={() => setActiveTab('python')}
                className={`px-6 py-3 font-mono text-sm border-r border-gray-700 transition ${activeTab === 'python' ? 'bg-gray-800 text-white' : 'text-gray-500 hover:text-white'}`}
              >
                Python (Agents)
              </button>
              <button 
                onClick={() => setActiveTab('nodejs')}
                className={`px-6 py-3 font-mono text-sm border-r border-gray-700 transition ${activeTab === 'nodejs' ? 'bg-gray-800 text-white' : 'text-gray-500 hover:text-white'}`}
              >
                Node.js
              </button>
              <button 
                onClick={() => setActiveTab('curl')}
                className={`px-6 py-3 font-mono text-sm transition ${activeTab === 'curl' ? 'bg-gray-800 text-white' : 'text-gray-500 hover:text-white'}`}
              >
                cURL
              </button>
            </div>
            <div className="p-8 overflow-x-auto">
              {activeTab === 'python' && (
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
              )}
              {activeTab === 'nodejs' && (
<pre className="text-sm font-mono text-gray-300 leading-relaxed">
<span className="text-purple-400">import</span> {'{ SatGateClient }'} <span className="text-purple-400">from</span> <span className="text-green-400">'satgate-sdk'</span>;<br/>
<br/>
<span className="text-gray-500">// Initialize client (uses WebLN in browser)</span><br/>
<span className="text-purple-400">const</span> client = <span className="text-purple-400">new</span> SatGateClient();<br/>
<br/>
<span className="text-gray-500">// Automatic: 402 → Pay → Retry → Response</span><br/>
<span className="text-purple-400">const</span> data = <span className="text-purple-400">await</span> client.get(<span className="text-green-400">'https://api.example.com/premium'</span>);<br/>
console.log(data);
</pre>
              )}
              {activeTab === 'curl' && (
<pre className="text-sm font-mono text-gray-300 leading-relaxed">
<span className="text-gray-500"># 1. Request protected endpoint → get 402 + invoice</span><br/>
curl -i https://api.example.com/api/premium<br/>
<br/>
<span className="text-gray-500"># 2. Pay the Lightning invoice (via your wallet)</span><br/>
<span className="text-gray-500"># Returns preimage as proof of payment</span><br/>
<br/>
<span className="text-gray-500"># 3. Retry with L402 token</span><br/>
curl -H <span className="text-green-400">"Authorization: L402 &lt;macaroon&gt;:&lt;preimage&gt;"</span> \<br/>
&nbsp;&nbsp;https://api.example.com/api/premium
</pre>
              )}
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
              <div className="flex items-center gap-2 mb-4">
                <Image src="/logo_white_transparent.png" alt="SatGate" width={24} height={24} className="w-6 h-6" />
                <h4 className="font-bold text-white">SatGate</h4>
              </div>
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
                <li><a href="https://x.com/SatGateIO" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">@SatGateIO</a></li>
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

import Link from 'next/link';
import { ArrowLeft, Shield, Lock, Eye, AlertTriangle, Key, Zap } from 'lucide-react';

export const metadata = {
  title: 'Security - SatGate',
};

export default function SecurityPage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Home
        </Link>
        
        <h1 className="text-4xl font-bold mb-2">Security</h1>
        <p className="text-gray-500 mb-12">Capability-based access control for the AI era</p>

        <div className="prose prose-invert prose-gray max-w-none space-y-8">

          {/* Capability Security Intro */}
          <section className="bg-gradient-to-r from-purple-900/20 to-cyan-900/20 border border-purple-800/30 rounded-xl p-6">
            <h2 className="text-xl font-bold text-white mb-4">From Identity to Capabilities</h2>
            <p className="text-gray-400 leading-relaxed mb-4">
              Traditional APIs use <strong>identity-based</strong> access: "Prove <em>who you are</em>, then we decide what you can do."
            </p>
            <p className="text-gray-400 leading-relaxed mb-0">
              SatGate uses <strong>capability-based</strong> access: "Present a cryptographic token that <em>already encodes</em> what you can do."
            </p>
          </section>

          <section className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="bg-red-950/20 border border-red-900/30 rounded-xl p-5">
              <div className="flex items-center gap-2 mb-3">
                <Lock className="text-red-400" size={18} />
                <h3 className="font-semibold text-red-300 text-sm">Identity-Based (OAuth/API Keys)</h3>
              </div>
              <ul className="text-gray-500 text-sm space-y-1">
                <li>• Requires user databases & PII</li>
                <li>• Credential stuffing risks</li>
                <li>• Agents can't sign up</li>
              </ul>
            </div>
            <div className="bg-green-950/20 border border-green-900/30 rounded-xl p-5">
              <div className="flex items-center gap-2 mb-3">
                <Key className="text-green-400" size={18} />
                <h3 className="font-semibold text-green-300 text-sm">Capability-Based (L402)</h3>
              </div>
              <ul className="text-gray-500 text-sm space-y-1">
                <li className="text-green-400">✓ No accounts required</li>
                <li className="text-green-400">✓ Payment = Authorization</li>
                <li className="text-green-400">✓ Perfect for agents</li>
              </ul>
            </div>
          </section>
          
          <section className="bg-gray-900 border border-gray-800 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-4">
              <Shield className="text-green-400" size={24} />
              <h2 className="text-xl font-bold text-white m-0">Non-Custodial Architecture</h2>
            </div>
            <p className="text-gray-400 leading-relaxed m-0">
              SatGate never holds, controls, or has access to your funds. Payments settle directly 
              from payers to your Lightning node. We never touch your private keys.
            </p>
          </section>

          <section className="bg-gray-900 border border-gray-800 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-4">
              <Key className="text-purple-400" size={24} />
              <h2 className="text-xl font-bold text-white m-0">Paid Capabilities (L402)</h2>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              L402 creates a new security primitive: <strong>paid capabilities</strong>. Payment gates token issuance, 
              and the token itself encodes permissions via macaroon caveats.
            </p>
            <ul className="text-gray-500 text-sm space-y-2">
              <li><strong className="text-gray-300">Edge Verification:</strong> Tokens verified cryptographically at the gateway—no user database lookup required</li>
              <li><strong className="text-gray-300">Least Privilege:</strong> Add caveats to constrain scope, time, and budget (e.g., "valid 5 mins", "max 10 calls")</li>
              <li><strong className="text-gray-300">Delegatable:</strong> Attenuate tokens before passing to sub-agents—permissions only shrink, never grow</li>
            </ul>
          </section>

          <section className="bg-gray-900 border border-gray-800 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-4">
              <Zap className="text-yellow-400" size={24} />
              <h2 className="text-xl font-bold text-white m-0">Economic Abuse Friction</h2>
            </div>
            <p className="text-gray-400 leading-relaxed m-0">
              Spam becomes expensive and self-limiting. High-volume callers must continuously pay to continue. 
              This complements (not replaces) your existing WAF/CDN for network-layer protection.
            </p>
          </section>

          <section className="bg-gray-900 border border-gray-800 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-4">
              <Eye className="text-cyan-400" size={24} />
              <h2 className="text-xl font-bold text-white m-0">Open Source Transparency</h2>
            </div>
            <p className="text-gray-400 leading-relaxed m-0">
              All SatGate code is open source under the MIT License. You can audit every line, 
              fork it, and run your own infrastructure. No black boxes.
            </p>
            <a 
              href="https://github.com/SatGate-io/satgate" 
              target="_blank" 
              rel="noopener noreferrer"
              className="inline-block mt-4 text-purple-400 hover:text-purple-300"
            >
              View Source on GitHub →
            </a>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Best Practices</h2>
            <ul className="list-disc list-inside text-gray-400 space-y-2">
              <li>Keep your LNC pairing phrase secure—treat it like a password</li>
              <li>Use environment variables for sensitive configuration</li>
              <li>Deploy behind a CDN/WAF (Cloudflare, AWS Shield) for network-layer protection</li>
              <li>Set appropriate macaroon expiration times in your aperture.yaml</li>
              <li>Monitor your Lightning node for unusual activity</li>
            </ul>
          </section>

          <section className="bg-yellow-900/20 border border-yellow-800/50 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-4">
              <AlertTriangle className="text-yellow-400" size={24} />
              <h2 className="text-xl font-bold text-white m-0">Responsible Disclosure</h2>
            </div>
            <p className="text-gray-400 leading-relaxed mb-4">
              Found a security vulnerability? We appreciate responsible disclosure. Please report 
              security issues privately before public disclosure.
            </p>
            <p className="text-gray-400 leading-relaxed m-0">
              Email:{' '}
              <a href="mailto:security@satgate.io" className="text-yellow-400 hover:text-yellow-300 font-mono">
                security@satgate.io
              </a>
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">What SatGate Protects Against</h2>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-800">
                    <th className="text-left py-3 text-gray-400 font-medium">Threat</th>
                    <th className="text-left py-3 text-gray-400 font-medium">Protection</th>
                  </tr>
                </thead>
                <tbody className="text-gray-400">
                  <tr className="border-b border-gray-800">
                    <td className="py-3">API scraping</td>
                    <td className="py-3 text-green-400">✓ Economic cost per request</td>
                  </tr>
                  <tr className="border-b border-gray-800">
                    <td className="py-3">Bot abuse</td>
                    <td className="py-3 text-green-400">✓ Payment required = economic friction</td>
                  </tr>
                  <tr className="border-b border-gray-800">
                    <td className="py-3">Credential theft</td>
                    <td className="py-3 text-green-400">✓ Tokens can have short expiration</td>
                  </tr>
                  <tr className="border-b border-gray-800">
                    <td className="py-3">Chargebacks</td>
                    <td className="py-3 text-green-400">✓ Lightning payments are final</td>
                  </tr>
                  <tr className="border-b border-gray-800">
                    <td className="py-3">Network DDoS (volumetric)</td>
                    <td className="py-3 text-yellow-400">⚠ Use CDN/WAF (Cloudflare, etc.)</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

        </div>
      </div>
    </div>
  );
}


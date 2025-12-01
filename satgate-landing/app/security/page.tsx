import Link from 'next/link';
import { ArrowLeft, Shield, Lock, Eye, AlertTriangle } from 'lucide-react';

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
        <p className="text-gray-500 mb-12">How SatGate protects you and your users</p>

        <div className="prose prose-invert prose-gray max-w-none space-y-8">
          
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
              <Lock className="text-purple-400" size={24} />
              <h2 className="text-xl font-bold text-white m-0">L402 Token Security</h2>
            </div>
            <p className="text-gray-400 leading-relaxed m-0">
              L402 tokens are cryptographic bearer credentials. They combine a macaroon (capability token) 
              with a Lightning payment preimage. Tokens can include caveats (restrictions) like expiration 
              times, usage limits, and scope constraints.
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


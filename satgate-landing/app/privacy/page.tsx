import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

export const metadata = {
  title: 'Privacy Policy - SatGate',
};

export default function PrivacyPage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Home
        </Link>
        
        <h1 className="text-4xl font-bold mb-2">Privacy Policy</h1>
        <p className="text-gray-500 mb-12">Last updated: December 2025</p>

        <div className="prose prose-invert prose-gray max-w-none space-y-8">
          
          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Overview</h2>
            <p className="text-gray-400 leading-relaxed">
              SatGate is designed with privacy as a core principle. We collect minimal data 
              and never require personal identification to use our software.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Data We Don't Collect</h2>
            <ul className="list-disc list-inside text-gray-400 space-y-2">
              <li>Personal Identifiable Information (PII)</li>
              <li>Email addresses (unless you contact us)</li>
              <li>Payment details or Lightning invoices (processed client-side)</li>
              <li>API request contents</li>
              <li>IP addresses (when using self-hosted deployments)</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Website Analytics</h2>
            <p className="text-gray-400 leading-relaxed">
              The satgate.io website may use privacy-respecting analytics (e.g., Plausible, Vercel Analytics) 
              to understand aggregate traffic patterns. This data is anonymized and does not track individual users.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Self-Hosted Deployments</h2>
            <p className="text-gray-400 leading-relaxed">
              When you self-host SatGate, all data stays on your infrastructure. We have no access to your 
              server logs, Lightning node data, or API traffic. You control everything.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Playground Demo</h2>
            <p className="text-gray-400 leading-relaxed">
              The Playground on satgate.io runs in simulation mode by default. Real Network mode connects 
              to your local server—no data is sent to our servers.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Third-Party Services</h2>
            <p className="text-gray-400 leading-relaxed">
              If you use third-party services (e.g., Voltage for Lightning nodes, Alby for wallets), 
              their privacy policies apply to your use of those services.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Contact</h2>
            <p className="text-gray-400 leading-relaxed">
              For privacy questions, contact us at{' '}
              <a href="mailto:contact@satgate.io" className="text-purple-400 hover:text-purple-300">
                contact@satgate.io
              </a>.
            </p>
          </section>

        </div>
      </div>
    </div>
  );
}



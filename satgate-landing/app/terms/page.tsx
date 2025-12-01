import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

export const metadata = {
  title: 'Terms of Service - SatGate',
};

export default function TermsPage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Home
        </Link>
        
        <h1 className="text-4xl font-bold mb-2">Terms of Service</h1>
        <p className="text-gray-500 mb-12">Last updated: December 2025</p>

        <div className="prose prose-invert prose-gray max-w-none space-y-8">
          
          <section>
            <h2 className="text-2xl font-bold text-white mb-4">1. Acceptance of Terms</h2>
            <p className="text-gray-400 leading-relaxed">
              By accessing or using SatGate ("Service"), you agree to be bound by these Terms of Service. 
              If you do not agree to these terms, do not use the Service.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">2. Description of Service</h2>
            <p className="text-gray-400 leading-relaxed">
              SatGate provides an open-source L402 gateway that enables API monetization via Lightning Network micropayments. 
              The Service includes SDKs, documentation, and related tools.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">3. Non-Custodial Nature</h2>
            <p className="text-gray-400 leading-relaxed">
              SatGate is non-custodial. We do not hold, control, or have access to your Bitcoin, Lightning funds, 
              or private keys. All payments settle directly between payers and your Lightning node. 
              You are solely responsible for the security of your node and credentials.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">4. Open Source License</h2>
            <p className="text-gray-400 leading-relaxed">
              The SatGate software is released under the MIT License. You may use, modify, and distribute 
              the software in accordance with that license. The MIT License is provided "as is" without warranty.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">5. No Warranty</h2>
            <p className="text-gray-400 leading-relaxed">
              THE SERVICE IS PROVIDED "AS IS" WITHOUT WARRANTIES OF ANY KIND, EXPRESS OR IMPLIED. 
              We do not guarantee uptime, accuracy, or fitness for any particular purpose. 
              Use at your own risk.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">6. Limitation of Liability</h2>
            <p className="text-gray-400 leading-relaxed">
              IN NO EVENT SHALL SATGATE BE LIABLE FOR ANY INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, 
              OR PUNITIVE DAMAGES, INCLUDING LOSS OF PROFITS, DATA, OR FUNDS, ARISING FROM YOUR USE OF THE SERVICE.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">7. Prohibited Uses</h2>
            <p className="text-gray-400 leading-relaxed">
              You agree not to use the Service for any unlawful purpose, to violate any laws in your jurisdiction, 
              or to infringe upon the rights of others.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">8. Changes to Terms</h2>
            <p className="text-gray-400 leading-relaxed">
              We reserve the right to modify these Terms at any time. Continued use of the Service 
              after changes constitutes acceptance of the new Terms.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">9. Contact</h2>
            <p className="text-gray-400 leading-relaxed">
              For questions about these Terms, contact us at{' '}
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


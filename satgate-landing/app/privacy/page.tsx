import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

export const metadata = {
  title: 'Privacy Policy - SatGate',
  description: 'SatGate privacy policy for cloud, self-hosted, and hybrid deployments, including request metadata, telemetry, cookies, retention, and third-party services.',
  alternates: { canonical: 'https://satgate.io/privacy' },
  openGraph: {
    title: 'Privacy Policy - SatGate',
    description: 'How SatGate handles privacy across cloud, self-hosted, and hybrid economic-control-plane deployments.',
    url: 'https://satgate.io/privacy',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Privacy Policy - SatGate',
    description: 'How SatGate handles privacy across cloud, self-hosted, and hybrid economic-control-plane deployments.',
  },
};

export default function PrivacyPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'Privacy Policy - SatGate',
    description: metadata.description,
    url: 'https://satgate.io/privacy',
    dateModified: '2026-05-05',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'SatGate Cloud privacy' },
      { '@type': 'Thing', name: 'self-hosted economic control plane privacy' },
      { '@type': 'Thing', name: 'API request metadata privacy' },
      { '@type': 'Thing', name: 'hybrid gateway telemetry' },
    ],
  };

  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <div className="max-w-3xl mx-auto px-6 py-16">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Home
        </Link>
        
        <h1 className="text-4xl font-bold mb-2">Privacy Policy</h1>
        <p className="text-gray-500 mb-12">Last updated: February 2026</p>

        <div className="prose prose-invert prose-gray max-w-none space-y-8">
          
          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Overview</h2>
            <p className="text-gray-400 leading-relaxed">
              SatGate is designed with privacy as a core principle. We collect minimal data 
              and are transparent about what we do and don&apos;t collect across our open-source software 
              and cloud service.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">SatGate Cloud (cloud.satgate.io)</h2>
            <p className="text-gray-400 leading-relaxed mb-4">
              When you sign up for SatGate Cloud, we collect:
            </p>
            <ul className="list-disc list-inside text-gray-400 space-y-2">
              <li>Email address — for authentication (magic link, OAuth, or SSO) and service communications</li>
              <li>OAuth profile data — if you sign in via Google or GitHub, we receive your name and email from the provider</li>
              <li>Usage telemetry — request counts, budget consumption, and policy events for your dashboard</li>
            </ul>
            <p className="text-gray-400 leading-relaxed mt-4">
              We do not inspect or store the contents of API requests proxied through SatGate. 
              The gateway processes request metadata (headers, path, method) for routing and policy enforcement only.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Data We Don&apos;t Collect</h2>
            <ul className="list-disc list-inside text-gray-400 space-y-2">
              <li>API request or response bodies</li>
              <li>Payment details or Lightning invoices (processed between payer and your node)</li>
              <li>Passwords (we use passwordless authentication)</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Self-Hosted &amp; Hybrid Deployments</h2>
            <p className="text-gray-400 leading-relaxed">
              When you self-host SatGate or deploy the gateway in hybrid mode (gateway in your VPC, 
              dashboard in our cloud), all proxied traffic stays in your infrastructure. We have no access 
              to your server logs, API traffic, or Lightning node data. In hybrid mode, the cloud dashboard 
              receives only aggregated telemetry (request counts, budget status) — never request contents.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Website Analytics</h2>
            <p className="text-gray-400 leading-relaxed">
              The satgate.io website may use privacy-respecting analytics (e.g., Vercel Analytics) 
              to understand aggregate traffic patterns. This data is anonymized and does not track individual users.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Cookies</h2>
            <p className="text-gray-400 leading-relaxed">
              SatGate Cloud uses HttpOnly session cookies for authentication. We do not use tracking cookies 
              or third-party advertising cookies.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Third-Party Services</h2>
            <p className="text-gray-400 leading-relaxed">
              SatGate Cloud uses Resend for transactional email delivery and WorkOS for enterprise SSO. 
              If you authenticate via Google or GitHub OAuth, their respective privacy policies apply. 
              For Lightning (L402) features, your Lightning node provider&apos;s privacy policy applies.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Data Retention</h2>
            <p className="text-gray-400 leading-relaxed">
              Usage telemetry and audit logs are retained for the duration of your account. 
              You may request deletion of your account and associated data by contacting us.
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

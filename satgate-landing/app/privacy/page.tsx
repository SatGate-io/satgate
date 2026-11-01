import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

export const metadata = {
  title: { absolute: 'Privacy Policy - SatGate' },
  description: 'SatGate privacy policy for SaaS and self-host, including request metadata, telemetry, cookies, retention, and third-party services.',
  alternates: { canonical: 'https://satgate.io/privacy' },
  openGraph: {
    title: 'Privacy Policy - SatGate',
    description: 'How SatGate handles privacy across SaaS and self-host.',
    url: 'https://satgate.io/privacy',
    type: 'website',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Privacy Policy - SatGate',
    description: 'How SatGate handles privacy across SaaS and self-host.',
  },
};

export default function PrivacyPage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: 'Privacy Policy - SatGate',
    description: metadata.description,
    url: 'https://satgate.io/privacy',
    dateModified: '2026-11-01',
    isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
    about: [
      { '@type': 'Thing', name: 'SatGate Cloud privacy' },
      { '@type': 'Thing', name: 'self-hosted Economic Firewall privacy' },
      { '@type': 'Thing', name: 'API request metadata privacy' },
      { '@type': 'Thing', name: 'dedicated gateway telemetry' },
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
        <p className="text-gray-500 mb-12">Last updated: November 2026</p>

        <div className="prose prose-invert prose-gray max-w-none space-y-8">
          
          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Overview</h2>
            <p className="text-gray-400 leading-relaxed">
              This page says what SatGate collects when you use SatGate Cloud or the satgate.io website,
              how long it keeps each kind of record, and which other companies receive your data. It also
              covers the open-source software you run yourself.
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
              <li>Name: the name you enter when you sign up</li>
              <li>Usage records: for requests through SatGate, the route, the request path and method, the result, the size and the time, and for MCP tools the tool name. Budget spending is recorded too.</li>
              <li>Refused requests: the route, the reason, the request path, and the IP address and the name of the browser or software that sent the request</li>
              <li>Audit log: what was done in your account and when, with the email address, IP address and browser of the person who did it</li>
              <li>Signed receipts: a signed record of what an agent did or was refused, such as the agent, the route or tool, the decision, the amounts and the time</li>
              <li>Held calls: when a tool needs your approval, the values of the call, so you can review them</li>
            </ul>
            <p className="text-gray-400 leading-relaxed mt-4">
              SatGate reads the token and where each request is going. For MCP tools it also reads the tool name and the
              arguments your rules check, for example an order&apos;s amount. If a tool needs your approval, the call&apos;s
              details are shown to you, and the approval email carries the same values. On SatGate Cloud, the credentials
              SatGate uses to reach your services are stored encrypted.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Data We Don&apos;t Collect</h2>
            <ul className="list-disc list-inside text-gray-400 space-y-2">
              <li>Request or response bodies, other than the values of a call held for your approval</li>
              <li>Card numbers (Stripe takes card payments for SatGate plans)</li>
              <li>Passwords (we use passwordless authentication)</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Payments</h2>
            <p className="text-gray-400 leading-relaxed">
              Card payments for SatGate plans go through Stripe. SatGate keeps your Stripe customer and subscription IDs,
              your plan and its status. For payments that your routes accept, Lightning invoices come from your own
              Lightning address and USDC goes to the wallet address you set on the route. SatGate keeps only what it
              needs to check a payment and refuse reuse: for Lightning the payment hash, and for USDC the payer&apos;s
              wallet address, the amount and the transaction.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Self-host</h2>
            <p className="text-gray-400 leading-relaxed">
              Enterprise self-host runs SatGate in your own environment.
              Custody boundaries are agreed by contract. When the
              gateway runs in your environment, all proxied traffic stays in your infrastructure. We have no access
              to your server logs, API traffic, or Lightning node data.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Website Analytics</h2>
            <p className="text-gray-400 leading-relaxed">
              The satgate.io website does not load an analytics script, and it sets no cookies. If you fill in the design
              partner form, your name, company, email address and answers are emailed to the SatGate team through Resend,
              and your name, company and email address are also written to the website&apos;s server logs on Vercel.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Cookies</h2>
            <p className="text-gray-400 leading-relaxed">
              SatGate Cloud uses HttpOnly session cookies for authentication. A session lasts up to 30 days. We do not use tracking cookies 
              or third-party advertising cookies.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Third-Party Services</h2>
            <p className="text-gray-400 leading-relaxed mb-4">
              SatGate Cloud uses these outside services. This is what each one receives.
            </p>
            <ul className="list-disc list-inside text-gray-400 space-y-2">
              <li>Fly.io runs the SatGate Cloud servers and database, in the United States (Virginia). It holds what SatGate Cloud stores.</li>
              <li>Upstash hosts the Redis store that SatGate Cloud uses for budget and spend tracking.</li>
              <li>Vercel hosts the satgate.io website. It receives the requests for the website&apos;s pages and the design partner form entries.</li>
              <li>Resend sends SatGate&apos;s emails, such as sign-in links and approval emails. It receives the recipient&apos;s email address and the contents of the email, including the call&apos;s values in an approval email.</li>
              <li>Stripe takes card payments for SatGate plans. It receives your name, email address and card details.</li>
              <li>WorkOS handles enterprise SSO. It receives your sign-in from your company&apos;s identity provider and passes your name and email address to SatGate.</li>
              <li>Google and GitHub, if you sign in with them. They know that you are signing in to SatGate, and SatGate receives your name, email address and account ID from them. Their respective privacy policies apply.</li>
              <li>Robinhood, if you connect it. You sign in on Robinhood&apos;s own page and SatGate never sees your Robinhood password. SatGate stores the access Robinhood gives it, encrypted, and Robinhood receives the calls your agent makes through SatGate.</li>
            </ul>
            <p className="text-gray-400 leading-relaxed mt-4">
              For Lightning (L402) features, your Lightning node provider&apos;s privacy policy applies.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold text-white mb-4">Data Retention</h2>
            <p className="text-gray-400 leading-relaxed mb-4">
              How long SatGate Cloud keeps a record depends on the kind of record and, for some, on your plan. This is
              what SatGate Cloud does today.
            </p>
            <ul className="list-disc list-inside text-gray-400 space-y-2">
              <li>Usage records: Starter 3 days. Trial, Growth, Personal and Pro 365 days. Enterprise 7 years.</li>
              <li>Refused requests: 7 days.</li>
              <li>Held calls: the call&apos;s values are cleared 24 hours after the approval is used, denied or expires. The approval record is deleted 90 days after that moment. The approval email stays where it was sent.</li>
              <li>Audit log: not deleted on a schedule today.</li>
              <li>Signed receipts and budget spending records: not deleted on a schedule today.</li>
              <li>Account details: not deleted on a schedule today.</li>
            </ul>
            <p className="text-gray-400 leading-relaxed mt-4">
              To ask for your account and its data to be deleted, email{' '}
              <a href="mailto:contact@satgate.io" className="text-purple-400 hover:text-purple-300">contact@satgate.io</a>.
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

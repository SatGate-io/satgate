import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'L402 API Pricing Calculator: Paid Agent Access Template',
  description:
    'Estimate L402 API pricing, HTTP 402 paid-agent access tiers, per-request margins, free allowances, and Lightning sats per API call.',
  alternates: { canonical: 'https://satgate.io/l402-api-pricing-calculator' },
  keywords: [
    'L402 API pricing calculator',
    'L402 pricing template',
    'HTTP 402 API pricing',
    'sats per API request calculator',
    'AI agent payment pricing',
    'paid API access calculator',
    'Lightning API pricing',
    'agent payment controls',
    'paid-rail context',
  ],
  openGraph: {
    title: 'L402 API Pricing Calculator',
    description:
      'Model L402 pricing tiers, paid-agent API access, margin, and Lightning sats per request while preserving paid-rail context.',
    url: 'https://satgate.io/l402-api-pricing-calculator',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'L402 API Pricing Calculator',
    description:
      'Estimate per-request L402 pricing, HTTP 402 paid access tiers, margins, free allowances, and sats per API call.',
  },
};

export default function L402ApiPricingCalculatorLayout({ children }: { children: React.ReactNode }) {
  return children;
}

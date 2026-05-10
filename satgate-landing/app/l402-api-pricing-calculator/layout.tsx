import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'L402 API Pricing Calculator',
  description:
    'Estimate L402 API pricing, per-request margins, free allowances, and paid-rail context for governed AI agent access.',
  alternates: { canonical: 'https://satgate.io/l402-api-pricing-calculator' },
  keywords: [
    'L402 API pricing calculator',
    'AI agent payment pricing',
    'paid API access calculator',
    'Lightning API pricing',
    'agent payment controls',
    'paid-rail context',
  ],
  openGraph: {
    title: 'L402 API Pricing Calculator',
    description:
      'Model L402 API pricing and margin while preserving paid-rail context for governed AI agent access.',
    url: 'https://satgate.io/l402-api-pricing-calculator',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'L402 API Pricing Calculator',
    description:
      'Estimate per-request L402 API pricing, margins, free allowances, and paid-rail context for AI agent access.',
  },
};

export default function L402ApiPricingCalculatorLayout({ children }: { children: React.ReactNode }) {
  return children;
}

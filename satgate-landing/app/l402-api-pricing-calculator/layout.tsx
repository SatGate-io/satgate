import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'L402 API Pricing Calculator',
  description:
    'Estimate per-request L402 API pricing, robot-customer revenue, gross margin, free allowance, and Lightning sats per request for AI agent API monetization.',
  alternates: { canonical: 'https://satgate.io/l402-api-pricing-calculator' },
};

export default function L402ApiPricingCalculatorLayout({ children }: { children: React.ReactNode }) {
  return children;
}

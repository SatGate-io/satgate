import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Revocable Capability Token Policy Template for AI Agents',
  description:
    'Generate revocable capability token policy for AI agents with caveats, attenuation, budgets, expiry, delegation limits, receipts, and Evidence Pack fields.',
  alternates: { canonical: 'https://satgate.io/revocable-capability-token-policy-template' },
  keywords: [
    'revocable capability token policy template',
    'AI agent capability token',
    'capability token caveats',
    'delegation attenuation for agents',
    'revocable agent credentials',
    'budget-aware capability tokens',
    'macaroon caveats for AI agents',
  ],
  openGraph: {
    title: 'Revocable Capability Token Policy Template',
    description: 'Generate AI agent capability-token policy with caveats, attenuation, budgets, expiry, revocation, and Evidence Pack fields.',
    url: 'https://satgate.io/revocable-capability-token-policy-template',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Revocable Capability Token Policy Template',
    description: 'Create scoped, expiring, revocable AI agent capability-token policy with budget and delegation caveats.',
  },
};

export default function RevocableCapabilityTokenPolicyTemplateLayout({ children }: { children: React.ReactNode }) {
  return children;
}

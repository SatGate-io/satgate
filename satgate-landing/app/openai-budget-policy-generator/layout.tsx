import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'OpenAI API Budget Limit Generator',
  description: 'Generate an AI agent budget policy for OpenAI API calls with per-request caps, daily budgets, model routing, revocation, and Evidence Pack receipts.',
  alternates: { canonical: 'https://satgate.io/openai-budget-policy-generator' },
  keywords: [
    'OpenAI API budget limits',
    'OpenAI budget policy generator',
    'OpenAI API cost control',
    'AI agent budget enforcement',
    'OpenAI spend limits',
    'LLM budget policy',
  ],
  openGraph: {
    title: 'OpenAI API Budget Limit Generator',
    description: 'Create request-path budget policy for OpenAI API calls, agent workflows, model routing, and Evidence Pack receipts.',
    url: 'https://satgate.io/openai-budget-policy-generator',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'OpenAI API Budget Limit Generator',
    description: 'Generate OpenAI API spend policy for agents: daily caps, per-request limits, model routing, revocation, and Evidence Pack receipts.',
  },
};

export default function OpenAiBudgetPolicyGeneratorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

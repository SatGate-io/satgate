import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'OpenAI API Budget Limit Generator for AI Agents',
  description: 'Generate request-path OpenAI API budget policy for AI agents with per-request caps, daily budgets, model routing, revocation, and Evidence Pack receipts.',
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
    title: 'OpenAI API Budget Limit Generator for AI Agents',
    description: 'Create request-path OpenAI API budget policy for agent workflows, model routing, revocation, and Evidence Pack receipts.',
    url: 'https://satgate.io/openai-budget-policy-generator',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'OpenAI API Budget Limit Generator for AI Agents',
    description: 'Generate OpenAI API budget policy for agents: daily caps, per-request limits, model routing, revocation, and Evidence Pack receipts.',
  },
};

export default function OpenAiBudgetPolicyGeneratorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

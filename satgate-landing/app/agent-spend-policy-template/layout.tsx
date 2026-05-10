import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Agent Budget Policy Template: Policy-to-Proof Controls',
  description:
    'Generate YAML and JSON agent budget policy with authority, MCP tool caps, revocation, receipts, and Evidence Pack fields.',
  alternates: { canonical: 'https://satgate.io/agent-spend-policy-template' },
  keywords: [
    'agent budget policy template',
    'AI agent budget policy',
    'MCP tool budget policy',
    'agent spend policy template',
    'Policy-to-Proof',
    'Evidence Pack',
  ],
  openGraph: {
    title: 'Agent Budget Policy Template: Policy-to-Proof Controls',
    description:
      'Generate copyable YAML and JSON policies for AI agent authority, budgets, MCP tool caps, revocation, receipts, and Evidence Pack proof.',
    url: 'https://satgate.io/agent-spend-policy-template',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Agent Budget Policy Template: Policy-to-Proof Controls',
    description:
      'Generate AI agent budget policy templates with scoped authority, revocation, receipts, and Evidence Pack fields.',
  },
};

export default function AgentSpendPolicyTemplateLayout({ children }: { children: React.ReactNode }) {
  return children;
}

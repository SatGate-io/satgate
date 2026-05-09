import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Agent Budget Policy Template: Policy-to-Proof Controls',
  description:
    'Generate YAML and JSON agent budget policy with authority, MCP tool caps, revocation, receipts, and Evidence Pack fields.',
  alternates: { canonical: 'https://satgate.io/agent-spend-policy-template' },
};

export default function AgentSpendPolicyTemplateLayout({ children }: { children: React.ReactNode }) {
  return children;
}

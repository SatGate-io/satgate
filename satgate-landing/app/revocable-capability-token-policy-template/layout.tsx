import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Revocable Capability Token Policy Template',
  description:
    'Generate scoped, expiring, revocable capability-token policy for AI agents, sub-agents, MCP tools, budgets, and audit trails.',
  alternates: { canonical: 'https://satgate.io/revocable-capability-token-policy-template' },
};

export default function RevocableCapabilityTokenPolicyTemplateLayout({ children }: { children: React.ReactNode }) {
  return children;
}

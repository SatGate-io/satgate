import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Agent Spend Policy Template',
  description:
    'Generate copyable YAML and JSON policy templates for AI agent budgets, MCP tool costs, delegation, revocation, and audit fields.',
  alternates: { canonical: 'https://satgate.io/agent-spend-policy-template' },
};

export default function AgentSpendPolicyTemplateLayout({ children }: { children: React.ReactNode }) {
  return children;
}

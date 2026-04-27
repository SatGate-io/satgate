import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Agent API Key Risk Assessment',
  description:
    'Free assessment for API key risk in autonomous AI agent workflows, including scope, budget, revocation, delegation, and audit gaps.',
  alternates: { canonical: 'https://satgate.io/agent-api-key-risk-assessment' },
};

export default function AgentApiKeyRiskAssessmentLayout({ children }: { children: React.ReactNode }) {
  return children;
}

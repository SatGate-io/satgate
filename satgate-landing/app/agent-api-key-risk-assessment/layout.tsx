import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Agent API Key Risk Assessment: AI Agent API Key Security',
  description:
    'Assess AI agent API key security risks across shared keys, scope, budgets, delegation, API key rotation, revocation, and audit gaps.',
  alternates: { canonical: 'https://satgate.io/agent-api-key-risk-assessment' },
  keywords: [
    'AI agent API key security',
    'agent API key risk assessment',
    'agent API key management',
    'API key rotation for AI agents',
    'static API key blast radius',
    'revocable agent credentials',
    'agent-scoped API keys',
  ],
  openGraph: {
    title: 'Agent API Key Risk Assessment',
    description: 'Score AI agent API key risk across shared keys, scope, budgets, delegation, API key rotation, revocation, and audit gaps.',
    url: 'https://satgate.io/agent-api-key-risk-assessment',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Agent API Key Risk Assessment',
    description: 'Assess AI agent API key security and static-key blast radius before autonomous agents inherit production access.',
  },
};

export default function AgentApiKeyRiskAssessmentLayout({ children }: { children: React.ReactNode }) {
  return children;
}

import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'SatGate vs AWS Bedrock vs Azure AI Foundry — Cloud-Native AI Governance Comparison',
  description:
    'Compare SatGate with AWS Bedrock and Azure AI Foundry for AI agent governance, budget enforcement, multi-cloud visibility, and L402 commerce.',
  keywords: [
    'SatGate vs AWS Bedrock',
    'SatGate vs Azure AI Foundry',
    'MCP governance comparison',
    'AI agent governance',
    'cloud-native AI governance',
    'L402 micropayments',
    'MCP proxy',
    'agent budget enforcement',
  ],
  alternates: { canonical: 'https://satgate.io/compare/cloud-native' },
  openGraph: {
    title: 'SatGate vs Cloud-Native AI Governance',
    description: "Why your cloud provider's built-in tools aren't enough for the Agentic Web",
    type: 'website',
    url: 'https://satgate.io/compare/cloud-native',
  },
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'SatGate vs AWS Bedrock vs Azure AI Foundry: Cloud AI Spending Controls Compared',
  description:
    'Compare SatGate with AWS Bedrock and Azure AI Foundry for AI agent controls, budget enforcement, visibility across clouds, and L402 payments.',
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
    title: 'SatGate vs Cloud Providers\' Built-In AI Controls',
    description: "Why your cloud provider's built-in tools aren't enough for AI agents that work across clouds",
    type: 'website',
    url: 'https://satgate.io/compare/cloud-native',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

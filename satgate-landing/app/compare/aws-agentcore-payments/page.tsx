import { BrutalComparisonPage } from '../_components/BrutalComparisonPage';
import { brutalComparisons } from '../_components/comparisons';

const config = brutalComparisons['aws-agentcore-payments'];

export const metadata = {
  title: 'SatGate vs AWS AgentCore Payments: Agent Spending Rules Beyond AWS',
  description: 'Compare SatGate and AWS AgentCore Payments: budgets across AI providers and sub-agents, MCP tool limits, payments, cloud or self-hosted, and signed receipts.',
  alternates: { canonical: 'https://satgate.io/compare/aws-agentcore-payments' },
  keywords: ['SatGate vs AWS AgentCore Payments', 'AWS AgentCore Payments alternative', 'agent payments governance', 'x402 agent payments', 'MCP agent policy', 'agent permissions'],
  openGraph: { title: config.title, description: 'Compare SatGate and AWS AgentCore Payments: budgets across AI providers and sub-agents, MCP tool limits, payments, cloud or self-hosted, and signed receipts.', url: 'https://satgate.io/compare/aws-agentcore-payments', type: 'article', images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }] },
  twitter: { card: 'summary_large_image', title: config.title, description: config.verdict },
};

export default function AwsAgentCorePaymentsComparisonPage() {
  return <BrutalComparisonPage config={config} />;
}

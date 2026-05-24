import { BrutalComparisonPage } from '../_components/BrutalComparisonPage';
import { brutalComparisons } from '../_components/comparisons';

const config = brutalComparisons['aws-agentcore-payments'];

export const metadata = {
  title: 'SatGate vs AWS AgentCore Payments: Governance Above Paid Rails',
  description: 'Compare SatGate and AWS AgentCore Payments across cross-provider control, paid rails, MCP proxying, delegated budgets, hybrid deployment, and Evidence Packs.',
  alternates: { canonical: 'https://satgate.io/compare/aws-agentcore-payments' },
  keywords: ['SatGate vs AWS AgentCore Payments', 'AWS AgentCore Payments alternative', 'agent payments governance', 'x402 agent payments', 'MCP agent policy', 'agent authority governance'],
  openGraph: { title: config.title, description: 'Compare SatGate and AWS AgentCore Payments across cross-provider control, paid rails, MCP proxying, delegated budgets, hybrid deployment, and Evidence Packs.', url: 'https://satgate.io/compare/aws-agentcore-payments', type: 'article' },
  twitter: { card: 'summary_large_image', title: config.title, description: config.verdict },
};

export default function AwsAgentCorePaymentsComparisonPage() {
  return <BrutalComparisonPage config={config} />;
}

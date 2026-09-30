import { BrutalComparisonPage } from '../_components/BrutalComparisonPage';
import { brutalComparisons } from '../_components/comparisons';

const config = brutalComparisons['api-gateway-rate-limits'];

export const metadata = {
  title: 'SatGate vs API Gateway Rate Limits: Budgets, Not Just Request Counts',
  description: 'Compare API gateway rate limits with SatGate per-agent rules: budgets for agents and sub-agents, MCP tool rules, payments, cloud or self-hosted, and signed receipts.',
  alternates: { canonical: 'https://satgate.io/compare/api-gateway-rate-limits' },
  keywords: ['SatGate vs API Gateway rate limits', 'agent API rate limits', 'API gateway budget enforcement', 'MCP rate limits', 'agent spend policy'],
  openGraph: { title: config.title, description: 'Compare API gateway rate limits with SatGate per-agent rules: budgets for agents and sub-agents, MCP tool rules, payments, cloud or self-hosted, and signed receipts.', url: 'https://satgate.io/compare/api-gateway-rate-limits', type: 'article', images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }] },
  twitter: { card: 'summary_large_image', title: config.title, description: config.verdict },
};

export default function ApiGatewayRateLimitsComparisonPage() {
  return <BrutalComparisonPage config={config} />;
}

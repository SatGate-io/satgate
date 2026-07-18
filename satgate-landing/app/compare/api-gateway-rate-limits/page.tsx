import { BrutalComparisonPage } from '../_components/BrutalComparisonPage';
import { brutalComparisons } from '../_components/comparisons';

const config = brutalComparisons['api-gateway-rate-limits'];

export const metadata = {
  title: 'SatGate vs API Gateway Rate Limits: Authority Beats Quotas',
  description: 'Compare API Gateway rate limits with SatGate agent authority: delegated budgets, MCP tool policy, paid rails, flexible deployment, and Evidence Packs.',
  alternates: { canonical: 'https://satgate.io/compare/api-gateway-rate-limits' },
  keywords: ['SatGate vs API Gateway rate limits', 'agent API rate limits', 'API gateway budget enforcement', 'MCP rate limits', 'agent spend policy'],
  openGraph: { title: config.title, description: 'Compare API Gateway rate limits with SatGate agent authority: delegated budgets, MCP tool policy, paid rails, flexible deployment, and Evidence Packs.', url: 'https://satgate.io/compare/api-gateway-rate-limits', type: 'article' },
  twitter: { card: 'summary_large_image', title: config.title, description: config.verdict },
};

export default function ApiGatewayRateLimitsComparisonPage() {
  return <BrutalComparisonPage config={config} />;
}

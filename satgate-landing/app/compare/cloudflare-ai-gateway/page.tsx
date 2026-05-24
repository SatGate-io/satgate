import { BrutalComparisonPage } from '../_components/BrutalComparisonPage';
import { brutalComparisons } from '../_components/comparisons';

const config = brutalComparisons['cloudflare-ai-gateway'];

export const metadata = {
  title: 'SatGate vs Cloudflare AI Gateway: Policy-to-Proof for Agent Actions',
  description: 'Compare Cloudflare AI Gateway and SatGate across AI routing, rate limits, pre-execution control, MCP proxying, delegated budgets, and Evidence Packs.',
  alternates: { canonical: 'https://satgate.io/compare/cloudflare-ai-gateway' },
  keywords: ['SatGate vs Cloudflare AI Gateway', 'Cloudflare AI Gateway alternative', 'AI gateway budget enforcement', 'MCP tool governance', 'agent authority governance'],
  openGraph: { title: config.title, description: 'Compare Cloudflare AI Gateway and SatGate across AI routing, rate limits, pre-execution control, MCP proxying, delegated budgets, and Evidence Packs.', url: 'https://satgate.io/compare/cloudflare-ai-gateway', type: 'article' },
  twitter: { card: 'summary_large_image', title: config.title, description: config.verdict },
};

export default function CloudflareAiGatewayComparisonPage() {
  return <BrutalComparisonPage config={config} />;
}

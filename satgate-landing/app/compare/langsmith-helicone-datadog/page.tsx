import { BrutalComparisonPage } from '../_components/BrutalComparisonPage';
import { brutalComparisons } from '../_components/comparisons';

const config = brutalComparisons['langsmith-helicone-datadog'];

export const metadata = {
  title: 'SatGate vs LangSmith, Helicone, Datadog: Stop Overspend Before It Happens',
  description: 'Compare LLM monitoring tools with SatGate, which checks agents before they spend: budgets for agents and sub-agents, MCP tool rules, payments, cloud or self-hosted, and signed receipts.',
  alternates: { canonical: 'https://satgate.io/compare/langsmith-helicone-datadog' },
  keywords: ['SatGate vs LangSmith', 'SatGate vs Helicone', 'SatGate vs Datadog LLM Observability', 'LLM observability vs control', 'agent Evidence Packs'],
  openGraph: { title: config.title, description: 'Compare LLM monitoring tools with SatGate, which checks agents before they spend: budgets for agents and sub-agents, MCP tool rules, payments, cloud or self-hosted, and signed receipts.', url: 'https://satgate.io/compare/langsmith-helicone-datadog', type: 'article', images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }] },
  twitter: { card: 'summary_large_image', title: config.title, description: config.verdict },
};

export default function LangSmithHeliconeDatadogComparisonPage() {
  return <BrutalComparisonPage config={config} />;
}

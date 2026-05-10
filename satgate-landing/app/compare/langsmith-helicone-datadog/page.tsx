import { BrutalComparisonPage } from '../_components/BrutalComparisonPage';
import { brutalComparisons } from '../_components/comparisons';

const config = brutalComparisons['langsmith-helicone-datadog'];

export const metadata = {
  title: 'SatGate vs LangSmith, Helicone, Datadog',
  description: 'Compare LLM observability tools with SatGate pre-execution control: delegated budgets, MCP tool policy, paid rails, hybrid enforcement, and Evidence Packs.',
  alternates: { canonical: 'https://satgate.io/compare/langsmith-helicone-datadog' },
  keywords: ['SatGate vs LangSmith', 'SatGate vs Helicone', 'SatGate vs Datadog LLM Observability', 'LLM observability vs control', 'agent Evidence Packs'],
  openGraph: { title: config.title, description: 'Compare LLM observability tools with SatGate pre-execution control: delegated budgets, MCP tool policy, paid rails, hybrid enforcement, and Evidence Packs.', url: 'https://satgate.io/compare/langsmith-helicone-datadog', type: 'article' },
  twitter: { card: 'summary_large_image', title: config.title, description: config.verdict },
};

export default function LangSmithHeliconeDatadogComparisonPage() {
  return <BrutalComparisonPage config={config} />;
}

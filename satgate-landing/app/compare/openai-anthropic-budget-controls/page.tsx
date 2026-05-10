import { BrutalComparisonPage } from '../_components/BrutalComparisonPage';
import { brutalComparisons } from '../_components/comparisons';

const config = brutalComparisons['openai-anthropic-budget-controls'];

export const metadata = {
  title: 'SatGate vs OpenAI / Anthropic Budgets',
  description: 'Compare native OpenAI and Anthropic budget controls with SatGate cross-provider agent budgets, MCP policy, paid rails, hybrid control, and Evidence Packs.',
  alternates: { canonical: 'https://satgate.io/compare/openai-anthropic-budget-controls' },
  keywords: ['OpenAI budget controls alternative', 'Anthropic budget controls alternative', 'cross-provider AI budgets', 'agent budget enforcement', 'SatGate vs OpenAI budgets'],
  openGraph: { title: config.title, description: 'Compare native OpenAI and Anthropic budget controls with SatGate cross-provider agent budgets, MCP policy, paid rails, hybrid control, and Evidence Packs.', url: 'https://satgate.io/compare/openai-anthropic-budget-controls', type: 'article' },
  twitter: { card: 'summary_large_image', title: config.title, description: config.verdict },
};

export default function OpenAiAnthropicBudgetControlsComparisonPage() {
  return <BrutalComparisonPage config={config} />;
}

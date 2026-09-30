import { BrutalComparisonPage } from '../_components/BrutalComparisonPage';
import { brutalComparisons } from '../_components/comparisons';

const config = brutalComparisons['openai-anthropic-budget-controls'];

export const metadata = {
  title: 'SatGate vs OpenAI and Anthropic Budgets: One Budget Across Providers',
  description: 'Compare OpenAI and Anthropic built-in budget controls with SatGate: one agent budget across providers, MCP tool rules, payments, cloud or self-hosted, and signed receipts.',
  alternates: { canonical: 'https://satgate.io/compare/openai-anthropic-budget-controls' },
  keywords: ['OpenAI budget controls alternative', 'Anthropic budget controls alternative', 'cross-provider AI budgets', 'agent budget enforcement', 'SatGate vs OpenAI budgets'],
  openGraph: { title: config.title, description: 'Compare OpenAI and Anthropic built-in budget controls with SatGate: one agent budget across providers, MCP tool rules, payments, cloud or self-hosted, and signed receipts.', url: 'https://satgate.io/compare/openai-anthropic-budget-controls', type: 'article', images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }] },
  twitter: { card: 'summary_large_image', title: config.title, description: config.verdict },
};

export default function OpenAiAnthropicBudgetControlsComparisonPage() {
  return <BrutalComparisonPage config={config} />;
}

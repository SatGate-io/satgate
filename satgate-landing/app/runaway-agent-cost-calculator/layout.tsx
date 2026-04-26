import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Runaway Agent Cost Calculator',
  description: 'Estimate the cost of AI agent loops, retries, delegated sub-agents, and paid tool calls before budget enforcement stops runaway spend.',
  alternates: { canonical: 'https://satgate.io/runaway-agent-cost-calculator' },
  keywords: [
    'runaway agent cost calculator',
    'AI agent loop cost calculator',
    'AI agent spend calculator',
    'agent budget enforcement calculator',
    'LLM runaway cost calculator',
    'MCP tool cost calculator',
  ],
  openGraph: {
    title: 'Runaway Agent Cost Calculator',
    description: 'Model how fast autonomous agents can burn API, model, and MCP tool budgets when loops or retries go unchecked.',
    url: 'https://satgate.io/runaway-agent-cost-calculator',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Runaway Agent Cost Calculator',
    description: 'Estimate AI agent loop, retry, and tool-call cost exposure before request-path budget enforcement.',
  },
};

export default function RunawayAgentCostCalculatorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

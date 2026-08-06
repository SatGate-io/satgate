import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Autonomous Retry Loop Cost Calculator | Runaway AI Agent Costs',
  description: 'Calculate runaway AI agent costs from autonomous retry loops, LLM agent loops, tool calls, delegation fanout, detection lag, and budget enforcement.',
  alternates: { canonical: 'https://satgate.io/runaway-agent-cost-calculator' },
  keywords: [
    'runaway agent cost calculator',
    'autonomous retry loops cost',
    'runaway cost',
    'runaway costs',
    'runaway LLM agent costs',
    'LLM agent runaway costs',
    'AI agent loop cost calculator',
    'AI agent spend calculator',
    'agent budget enforcement calculator',
    'LLM runaway cost calculator',
    'MCP tool cost calculator',
  ],
  openGraph: {
    title: 'Autonomous Retry Loop Cost Calculator | Runaway AI Agent Costs',
    description: 'Model runaway AI agent costs from retry loops, LLM calls, MCP tools, delegation fanout, detection lag, and budget enforcement.',
    url: 'https://satgate.io/runaway-agent-cost-calculator',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Autonomous Retry Loop Cost Calculator',
    description: 'Estimate AI agent loop, retry, and tool-call exposure before request-path budget enforcement.',
  },
};

export default function RunawayAgentCostCalculatorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

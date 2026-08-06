import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'MCP Tool Cost Policy Generator: Pricing Template for AI Tools',
  description: 'Generate MCP tool cost policy, pricing templates, spend limits, risk tiers, revocation rules, and Evidence Pack receipts for AI agents.',
  alternates: { canonical: 'https://satgate.io/mcp-tool-cost-policy-generator' },
  keywords: [
    'MCP tool cost policy generator',
    'MCP tool pricing template',
    'MCP tool cost calculator',
    'MCP budget enforcement',
    'MCP tool spend limits',
    'MCP governance policy',
    'Cursor MCP budget control',
    'Claude MCP governance',
  ],
  openGraph: {
    title: 'MCP Tool Cost Policy Generator',
    description: 'Create MCP tool pricing templates, spend limits, and Evidence Pack policy before agents trigger paid APIs, searches, browser sessions, or cloud tasks.',
    url: 'https://satgate.io/mcp-tool-cost-policy-generator',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Tool Cost Policy Generator',
    description: 'Generate MCP per-tool pricing templates, budgets, risk tiers, revocation, and Evidence Pack policy for AI agents.',
  },
};

export default function McpToolCostPolicyGeneratorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

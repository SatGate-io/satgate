import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'MCP Proxy Config Generator',
  description:
    'Generate MCP proxy configuration for Cursor, Claude Desktop, Claude Code, OpenClaw, and custom MCP clients with scoped authority, budgets, audit receipts, and revocation.',
  alternates: { canonical: 'https://satgate.io/mcp-proxy-config-generator' },
  keywords: [
    'MCP proxy config generator',
    'MCP server proxy configuration',
    'Claude Desktop MCP proxy',
    'Cursor MCP proxy',
    'MCP budget policy',
    'agent tool governance',
  ],
  openGraph: {
    title: 'MCP Proxy Config Generator',
    description:
      'Generate MCP proxy configs with scoped authority, budgets, audit receipts, revocation, and Evidence Pack-ready policy fields.',
    url: 'https://satgate.io/mcp-proxy-config-generator',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Proxy Config Generator',
    description:
      'Create MCP proxy configuration for agent tools with scoped authority, budgets, audit receipts, and revocation.',
  },
};

export default function McpProxyConfigGeneratorLayout({ children }: { children: React.ReactNode }) {
  return children;
}

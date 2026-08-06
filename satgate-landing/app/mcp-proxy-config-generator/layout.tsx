import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'MCP Proxy Config Generator for Cursor, Claude, and OpenClaw',
  description:
    'Generate MCP proxy configuration and setup steps for Cursor, Claude Desktop, Claude Code, OpenClaw, and custom clients with budgets, scoped authority, Evidence Pack receipts, and revocation.',
  alternates: { canonical: 'https://satgate.io/mcp-proxy-config-generator' },
  keywords: [
    'MCP proxy config generator',
    'MCP server proxy configuration',
    'Cursor MCP proxy config',
    'Claude Desktop MCP proxy config',
    'Claude Code MCP proxy config',
    'OpenClaw MCP proxy config',
    'Claude Desktop MCP proxy',
    'Cursor MCP proxy',
    'MCP budget policy',
    'agent tool governance',
  ],
  openGraph: {
    title: 'MCP Proxy Config Generator',
    description:
      'Generate MCP proxy configs and setup steps with scoped authority, budgets, Evidence Pack receipts, revocation, and Evidence Pack-ready policy fields.',
    url: 'https://satgate.io/mcp-proxy-config-generator',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Proxy Config Generator',
    description:
      'Create Cursor, Claude, OpenClaw, and custom MCP proxy configuration with scoped authority, budgets, Evidence Pack receipts, and revocation.',
  },
};

export default function McpProxyConfigGeneratorLayout({ children }: { children: React.ReactNode }) {
  return children;
}

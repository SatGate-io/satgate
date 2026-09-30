import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'MCP Proxy Config Generator',
  description:
    'Copy the Cloud MCP connect snippet for Cursor and Claude Code. A new token starts at 1,000 credits.',
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
      'Generate MCP proxy configs with scoped authority, budgets, revocation, and Evidence Pack fields.',
    url: 'https://satgate.io/mcp-proxy-config-generator',
    type: 'website',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Proxy Config Generator',
    description:
      'Create MCP proxy configuration for agent tools with scoped authority, budgets, signed receipts, and revocation.',
  },
};

export default function McpProxyConfigGeneratorLayout({ children }: { children: React.ReactNode }) {
  return children;
}

import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'MCP Proxy Config Generator',
  description:
    'Generate MCP proxy configuration for Cursor, Claude Desktop, Claude Code, OpenClaw, and custom MCP clients with budgets, audit, revocation, and L402 charge options.',
  alternates: { canonical: 'https://satgate.io/mcp-proxy-config-generator' },
};

export default function McpProxyConfigGeneratorLayout({ children }: { children: React.ReactNode }) {
  return children;
}

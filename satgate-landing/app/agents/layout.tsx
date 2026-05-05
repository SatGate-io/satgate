import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent Management | Budgets, Capabilities, and MCP Governance",
  alternates: { canonical: "https://satgate.io/agents" },
  description:
    "Manage AI agents with request-path budgets, scoped capability tokens, delegation trees, MCP tool governance, revocation, and real-time spend tracking.",
  keywords: [
    "AI agent management",
    "AI agent budgets",
    "agent capability tokens",
    "MCP tool governance",
    "AI agent spend tracking",
    "agent delegation controls",
    "economic firewall for agents",
  ],
  openGraph: {
    title: "AI Agent Management | Budgets, Capabilities, and MCP Governance",
    description:
      "Manage AI agents with request-path budgets, scoped capabilities, delegation controls, MCP tool governance, revocation, and spend tracking.",
    url: "https://satgate.io/agents",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent Management | Budgets, Capabilities, and MCP Governance",
    description:
      "Request-path budgets, scoped capabilities, MCP tool governance, revocation, and spend tracking for autonomous agents.",
  },
};

export default function AgentsLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

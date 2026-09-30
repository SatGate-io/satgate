import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent Management | Budgets, Capabilities, and MCP Governance",
  alternates: { canonical: "https://satgate.io/agents" },
  description:
    "Manage AI agents with budgets checked before the request goes through, scoped capability tokens, delegation trees, MCP tool governance, revocation, and real-time spend tracking.",
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
      "Manage AI agents with budgets checked before the request goes through, scoped capabilities, delegation controls, MCP tool governance, revocation, and spend tracking.",
    url: "https://satgate.io/agents",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent Management | Budgets, Capabilities, and MCP Governance",
    description:
      "Budgets checked before the request goes through, scoped capabilities, MCP tool governance, revocation, and spend tracking for autonomous agents.",
  },
};

export default function AgentsLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

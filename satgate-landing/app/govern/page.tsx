import type { Metadata } from "next";
import GovernClient from "../components/GovernClient";

export const metadata: Metadata = {
  title: "AI Agent Governance Platform: Budgets, MCP Controls, and L402 Charge",
  description:
    "Govern autonomous AI agents with request-path budget enforcement, MCP tool controls, delegation, revocation, cost attribution, and L402 Charge.",
  alternates: {
    canonical: "https://satgate.io/govern",
  },
  keywords: [
    "enterprise AI agent governance",
    "AI agent cost governance",
    "economic control plane for AI agents",
    "MCP governance for enterprises",
    "AI agent budget enforcement",
    "agent delegation controls",
    "L402 robot customer payments",
    "Observe Control Charge",
  ],
  openGraph: {
    title: "AI Agent Governance Platform: Budgets, MCP Controls, and L402 Charge",
    description:
      "Request-path budget enforcement, MCP governance, delegation controls, revocation, cost attribution, and L402 Charge for AI agents.",
    url: "https://satgate.io/govern",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent Governance Platform: Budgets, MCP Controls, and L402 Charge",
    description:
      "Enterprise AI agent governance for spend, MCP tools, delegation, revocation, attribution, and robot-customer payments.",
  },
};

export default function GovernPage() {
  return <GovernClient />;
}

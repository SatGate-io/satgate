import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent Budget Enforcement Demo | SatGate Control",
  alternates: { canonical: "https://satgate.io/protect" },
  description:
    "See request-path AI agent budget enforcement in action. Control API and MCP tool spend with per-agent caps, delegation limits, policy decisions, and instant revocation.",
  keywords: [
    "AI agent budget enforcement demo",
    "AI agent cost control demo",
    "MCP tool spend control",
    "request-path budget enforcement",
    "agent spend revocation",
    "economic firewall demo",
    "SatGate Control",
  ],
  openGraph: {
    title: "AI Agent Budget Enforcement Demo | SatGate Control",
    description:
      "Watch SatGate enforce per-agent budgets, MCP tool limits, delegation controls, policy decisions, and revocation before upstream spend happens.",
    url: "https://satgate.io/protect",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent Budget Enforcement Demo | SatGate Control",
    description:
      "Request-path budget enforcement for AI agent API and MCP tool spend.",
  },
};

export default function ProtectLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

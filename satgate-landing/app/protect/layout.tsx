import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent Budget Enforcement Demo | SatGate Control",
  alternates: { canonical: "https://satgate.io/protect" },
  description:
    "See AI agent budget enforcement before the request goes through. Control API and MCP tool spend with per-agent caps, delegation limits, policy decisions, and next-request revocation.",
  keywords: [
    "AI agent budget enforcement demo",
    "AI agent cost control demo",
    "MCP tool spend control",
    "budget enforcement before the request goes through",
    "agent spend revocation",
    "economic firewall demo",
    "SatGate Control",
  ],
  openGraph: {
    title: "AI Agent Budget Enforcement Demo | SatGate Control",
    description:
      "Watch SatGate enforce per-agent budgets, MCP tool limits, delegation controls, policy decisions, and revocation at the gateway policy check.",
    url: "https://satgate.io/protect",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent Budget Enforcement Demo | SatGate Control",
    description:
      "Budget enforcement before the request goes through, for AI agent API and MCP tool spend.",
  },
};

export default function ProtectLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

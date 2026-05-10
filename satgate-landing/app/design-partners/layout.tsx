import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "SatGate Design Partners | Economic Firewall for AI Agents",
  alternates: { canonical: "https://satgate.io/design-partners" },
  description:
    "Join SatGate as a design partner for Policy-to-Proof agent governance. Shape request-path budget enforcement, MCP governance, agent API controls, paid-rail context, and Evidence Pack proof before launch.",
  keywords: [
    "SatGate design partners",
    "economic firewall design partner",
    "AI agent governance design partner",
    "AI agent cost control early access",
    "MCP governance early access",
    "agent API governance",
    "Policy-to-Proof early access",
  ],
  openGraph: {
    title: "SatGate Design Partners | Economic Firewall for AI Agents",
    description:
      "Early access for teams shaping AI agent budget enforcement, MCP governance, agent API controls, paid-rail context, and Evidence Pack proof.",
    url: "https://satgate.io/design-partners",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate Design Partners | Economic Firewall for AI Agents",
    description:
      "Shape Policy-to-Proof governance for enterprise AI agents with SatGate.",
  },
};

export default function DesignPartnersLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

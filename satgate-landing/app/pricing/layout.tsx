import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "SatGate Pricing | Economic Firewall for AI Agents",
  alternates: { canonical: "https://satgate.io/pricing" },
  description:
    "SatGate pricing for bounded agent authority: Observe audits, request-path budget enforcement, per-agent spend caps, MCP tool controls, receipts, and Evidence Pack proof.",
  keywords: [
    "SatGate pricing",
    "AI agent cost control pricing",
    "AI agent budget enforcement pricing",
    "MCP governance pricing",
    "economic firewall pricing",
    "rail-neutral paid-rail governance pricing",
    "Observe Control Charge pricing",
  ],
  openGraph: {
    title: "SatGate Pricing | Economic Firewall for AI Agents",
    description:
      "Pricing for bounded agent authority: Observe audits, budget enforcement, MCP tool controls, per-agent caps, receipts, and Evidence Pack proof.",
    url: "https://satgate.io/pricing",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate Pricing | Economic Firewall for AI Agents",
    description:
      "AI agent cost control, MCP governance, receipt, and Evidence Pack pricing.",
  },
};

export default function PricingLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

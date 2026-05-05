import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "SatGate Pricing | Observe, Control, and Charge for AI Agents",
  alternates: { canonical: "https://satgate.io/pricing" },
  description:
    "SatGate pricing for AI agent economic governance: free Observe audits, Pro request-path budget enforcement, per-agent spend caps, MCP tool controls, and L402 Lightning Charge.",
  keywords: [
    "SatGate pricing",
    "AI agent cost control pricing",
    "AI agent budget enforcement pricing",
    "MCP governance pricing",
    "economic firewall pricing",
    "L402 API monetization pricing",
    "Observe Control Charge pricing",
  ],
  openGraph: {
    title: "SatGate Pricing | Observe, Control, and Charge for AI Agents",
    description:
      "Pricing for AI agent economic governance: Observe audits, Control budget enforcement, MCP tool controls, per-agent caps, and L402 Charge.",
    url: "https://satgate.io/pricing",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate Pricing | Observe, Control, and Charge for AI Agents",
    description:
      "AI agent cost control, MCP governance, and L402 API monetization pricing.",
  },
};

export default function PricingLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "SatGate Pricing | Economic Firewall for AI Agents",
  alternates: { canonical: "https://satgate.io/pricing" },
  description:
    "SatGate plans: free Starter, Pro at $99 a month, and Enterprise. Per-agent budgets, spending caps, MCP tool controls and signed receipts. 14-day free trial, no credit card.",
  keywords: [
    "SatGate pricing",
    "AI agent cost control pricing",
    "AI agent budget enforcement pricing",
    "MCP governance pricing",
    "economic firewall pricing",
    "agent payment controls pricing",
    "Observe Control Admit pricing",
  ],
  openGraph: {
    title: "SatGate Pricing | Economic Firewall for AI Agents",
    description:
      "Free Starter, Pro at $99 a month, and Enterprise. Per-agent budgets, MCP tool controls and signed receipts. 14-day free trial.",
    url: "https://satgate.io/pricing",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate Pricing | Economic Firewall for AI Agents",
    description:
      "Plans for AI agent budgets, MCP tool controls and signed receipts. 14-day free trial, no credit card.",
  },
};

export default function PricingLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent ROI Calculator: Budget Enforcement Savings",
  alternates: { canonical: "https://satgate.io/roi-calculator" },
  description:
    "Estimate AI agent loop exposure and map budget controls to signed receipts (Evidence Pack) with SatGate.",
  keywords: [
    "AI agent ROI calculator",
    "AI agent cost calculator",
    "AI agent budget enforcement",
    "runaway agent spend calculator",
    "LLM cost management calculator",
    "agent loop cost calculator",
  ],
  openGraph: {
    title: "AI Agent ROI Calculator",
    description:
      "Estimate runaway agent loop exposure, budget-control ROI, and signed receipt coverage.",
    url: "https://satgate.io/roi-calculator",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent ROI Calculator",
    description:
      "Estimate AI agent loop exposure, budget-control ROI, and SatGate signed receipt coverage.",
  },
};

export default function RoiCalculatorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

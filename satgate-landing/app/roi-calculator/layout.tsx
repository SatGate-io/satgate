import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent ROI Calculator: Budget Enforcement Savings",
  alternates: { canonical: "https://satgate.io/roi-calculator" },
  description:
    "Estimate AI agent loop exposure, payback period, and request-path budget enforcement savings with Policy-to-Proof receipts.",
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
      "Estimate runaway agent loop exposure, budget-control ROI, payback period, and Policy-to-Proof receipt coverage.",
    url: "https://satgate.io/roi-calculator",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent ROI Calculator",
    description:
      "Estimate AI agent loop exposure, budget-control ROI, payback period, and SatGate Policy-to-Proof receipt coverage.",
  },
};

export default function RoiCalculatorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent ROI Calculator: Budget Enforcement Savings",
  alternates: { canonical: "https://satgate.io/roi-calculator" },
  description:
    "Estimate AI agent ghost spend, loop waste, budget enforcement savings, payback period, and annual ROI with SatGate.",
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
      "Estimate ghost spend, runaway agent loop exposure, payback period, and savings from request-path budget enforcement.",
    url: "https://satgate.io/roi-calculator",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent ROI Calculator",
    description:
      "Estimate AI agent ghost spend, runaway loop exposure, payback period, and SatGate budget-enforcement savings.",
  },
};

export default function RoiCalculatorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

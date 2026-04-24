import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent ROI Calculator: Estimate Savings from Budget Enforcement",
  alternates: { canonical: "https://satgate.io/roi-calculator" },
  description:
    "Estimate savings from AI agent budget enforcement. Model ghost spend, runaway tool calls, loop waste, payback period, and annual ROI with SatGate.",
};

export default function RoiCalculatorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

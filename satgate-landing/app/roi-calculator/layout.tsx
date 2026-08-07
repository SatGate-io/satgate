import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent ROI Calculator: Voice Agent Payback and Budget Savings",
  alternates: { canonical: "https://satgate.io/roi-calculator" },
  description:
    "Calculate AI agent ROI, voice agent payback, chargeback exposure, loop waste, budget-control priorities, and request-path enforcement savings.",
  keywords: [
    "AI agent ROI calculator",
    "AI voice agent ROI calculator",
    "AI agent ROI formula",
    "agent ROI calculator",
    "voice agent payback calculator",
    "AI agent payback period",
    "AI chargeback ROI calculator",
    "AI agent cost calculator",
    "AI agent budget enforcement",
    "economic control plane for AI agents",
    "runaway agent spend calculator",
    "LLM cost management calculator",
    "agent loop cost calculator",
  ],
  openGraph: {
    title: "AI Agent ROI Calculator: Voice Agent Payback and Budget Savings",
    description:
      "Estimate AI agent ROI, voice-agent payback, loop exposure, chargeback risk, enforcement priorities, and Policy-to-Proof receipt coverage.",
    url: "https://satgate.io/roi-calculator",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent ROI Calculator",
    description:
      "Estimate AI agent ROI, voice-agent payback, chargeback risk, enforcement priorities, and SatGate Policy-to-Proof receipts.",
  },
};

export default function RoiCalculatorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

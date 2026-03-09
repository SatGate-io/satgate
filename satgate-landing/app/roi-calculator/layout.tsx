import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "ROI Calculator",
  alternates: { canonical: "https://satgate.io/roi-calculator" },
  description:
    "Calculate potential savings from AI agent spend governance. See projected ROI from budget enforcement and cost attribution with SatGate.",
};

export default function RoiCalculatorLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

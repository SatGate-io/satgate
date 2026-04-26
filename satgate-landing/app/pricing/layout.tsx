import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Pricing",
  alternates: { canonical: "https://satgate.io/pricing" },
  description:
    "SatGate pricing: free Observe audits, Pro budget enforcement, per-agent spend caps, and L402 Lightning monetization.",
};

export default function PricingLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

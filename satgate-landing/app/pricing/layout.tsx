import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Pricing",
  description:
    "SatGate pricing — free Observe tier for audit and attribution. Pro tier for real-time budget enforcement, per-agent spending caps, and L402 Lightning monetization.",
};

export default function PricingLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "L402 Paid-Rail Governance Demo | SatGate",
  alternates: { canonical: "https://satgate.io/pay" },
  description:
    "See SatGate govern delegated paid API access with paid-rail context, payment proof, scoped authority, receipts, and Evidence Packs at the gateway before forwarding.",
  keywords: [
    "L402 paid-rail governance demo",
    "SatGate paid-rail governance",
    "delegated paid API access",
    "AI agent API monetization",
    "paid-rail context",
    "HTTP 402 API payments",
    "per-request API pricing",
    "economic firewall proof",
  ],
  openGraph: {
    title: "L402 Paid-Rail Governance Demo | SatGate",
    description:
      "Watch delegated paid API access pass through policy, budget, payment proof, and receipt checks before protected requests are forwarded upstream.",
    url: "https://satgate.io/pay",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "L402 Paid-Rail Governance Demo | SatGate",
    description:
      "Per-request paid API access with scoped authority, paid-rail context, and proof.",
  },
};

export default function PayLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

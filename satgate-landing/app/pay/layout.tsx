import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "L402 Payment Rules Demo | SatGate",
  alternates: { canonical: "https://satgate.io/pay" },
  description:
    "See SatGate govern delegated paid API access with payment details, payment proof, scoped permissions, and signed receipts at the gateway before forwarding.",
  keywords: [
    "L402 payment rules demo",
    "SatGate payment rules",
    "delegated paid API access",
    "AI agent API monetization",
    "payment details",
    "HTTP 402 API payments",
    "per-request API pricing",
    "economic firewall proof",
  ],
  openGraph: {
    title: "L402 Payment Rules Demo | SatGate",
    description:
      "Watch delegated paid API access pass through policy, budget, payment proof, and receipt checks before protected requests are forwarded upstream.",
    url: "https://satgate.io/pay",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "L402 Payment Rules Demo | SatGate",
    description:
      "Per-request paid API access with scoped authority, payment details, and proof.",
  },
};

export default function PayLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

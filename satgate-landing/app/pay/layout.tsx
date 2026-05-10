import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "L402 Agent Payment Demo | SatGate paid-rail governance",
  alternates: { canonical: "https://satgate.io/pay" },
  description:
    "See SatGate paid-rail governance unlock API access with paid-rail context. Price paid-agent requests, verify payment proof, and settle API revenue before upstream access.",
  keywords: [
    "L402 agent payment demo",
    "SatGate paid-rail governance",
    "paid agent payments",
    "AI agent API monetization",
    "paid-rail context",
    "HTTP 402 API payments",
    "per-request API pricing",
    "economic firewall charge",
  ],
  openGraph: {
    title: "L402 Agent Payment Demo | SatGate paid-rail governance",
    description:
      "Watch AI agents pay for API access with paid-rail context before protected requests are forwarded upstream.",
    url: "https://satgate.io/pay",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "L402 Agent Payment Demo | SatGate paid-rail governance",
    description:
      "Per-request paid-agent payments for APIs using paid-rail context.",
  },
};

export default function PayLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

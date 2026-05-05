import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "L402 Agent Payment Demo | SatGate Charge",
  alternates: { canonical: "https://satgate.io/pay" },
  description:
    "See SatGate Charge unlock API access with L402 Lightning payments. Price robot-customer requests, verify payment proof, and settle API revenue before upstream access.",
  keywords: [
    "L402 agent payment demo",
    "SatGate Charge",
    "robot customer payments",
    "AI agent API monetization",
    "L402 Lightning payments",
    "HTTP 402 API payments",
    "per-request API pricing",
    "economic firewall charge",
  ],
  openGraph: {
    title: "L402 Agent Payment Demo | SatGate Charge",
    description:
      "Watch AI agents pay for API access with L402 Lightning payments before protected requests are forwarded upstream.",
    url: "https://satgate.io/pay",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "L402 Agent Payment Demo | SatGate Charge",
    description:
      "Per-request robot-customer payments for APIs using L402 Lightning.",
  },
};

export default function PayLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Agent Capability Token Minting Demo | SatGate Mint",
  alternates: { canonical: "https://satgate.io/mint-demo" },
  description:
    "Mint budget-aware capability tokens and macaroons for AI agents. Set request-path budgets, scopes, expiry, delegation limits, and revocation policy in seconds.",
  keywords: [
    "agent capability token demo",
    "AI agent macaroons",
    "SatGate Mint",
    "revocable agent credentials",
    "AI agent budget tokens",
    "scoped API credentials for agents",
    "agent delegation limits",
    "economic access control",
  ],
  openGraph: {
    title: "Agent Capability Token Minting Demo | SatGate Mint",
    description:
      "Mint scoped, budget-aware, revocable capability tokens for AI agents with budgets, expiry, scopes, and delegation limits.",
    url: "https://satgate.io/mint-demo",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Agent Capability Token Minting Demo | SatGate Mint",
    description:
      "Budget-aware capability tokens and macaroons for AI agent API governance.",
  },
};

export default function MintDemoLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

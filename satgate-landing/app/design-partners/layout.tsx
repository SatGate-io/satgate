import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "SatGate Design Partners | Economic Firewall for AI Agents",
  alternates: { canonical: "https://satgate.io/design-partners" },
  description:
    "Optional 90-day design-partner pilot for larger teams that want help with Policy-to-Proof governance. A 14-day self-serve trial is also open.",
  keywords: [
    "SatGate design partners",
    "economic firewall design partner",
    "AI agent governance design partner",
    "AI agent cost control pilot",
    "MCP governance pilot",
    "agent API governance",
    "Policy-to-Proof pilot",
  ],
  openGraph: {
    title: "SatGate Design Partners | Economic Firewall for AI Agents",
    description:
      "Optional 90-day design-partner pilot for larger teams that want help. A 14-day self-serve trial is also open.",
    url: "https://satgate.io/design-partners",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate Design Partners | Economic Firewall for AI Agents",
    description:
      "Shape Policy-to-Proof governance for enterprise AI agents with SatGate.",
  },
};

export default function DesignPartnersLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

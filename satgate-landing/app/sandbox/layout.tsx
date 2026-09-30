import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "SatGate Demo | Try AI Agent Budget Enforcement Free",
  alternates: { canonical: "https://satgate.io/sandbox" },
  description:
    "Try SatGate without signing up. Watch it give an agent a budget, block calls the agent is not allowed to make, revoke its token and charge external agents.",
  keywords: [
    "SatGate demo",
    "AI agent budget enforcement demo",
    "try economic firewall",
    "AI agent API governance demo",
    "MCP governance demo",
    "capability token demo",
    "request-path policy enforcement",
  ],
  openGraph: {
    title: "SatGate Demo | Try AI Agent Budget Enforcement Free",
    description:
      "Watch SatGate give an agent a budget, block calls it is not allowed to make and revoke its token.",
    url: "https://satgate.io/sandbox",
    type: "website",
    images: [{ url: "/logo.png", width: 512, height: 512, alt: "SatGate" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate Demo | Try AI Agent Budget Enforcement Free",
    description:
      "Watch SatGate cap an agent's budget, block calls and revoke its token.",
  },
};

export default function SandboxLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

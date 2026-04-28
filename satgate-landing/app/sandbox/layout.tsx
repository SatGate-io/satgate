import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "SatGate Sandbox | Try AI Agent Budget Enforcement Free",
  alternates: { canonical: "https://satgate.io/sandbox" },
  description:
    "Try SatGate without signup. Test economic access control, AI agent budget enforcement, MCP tool governance, capability tokens, and request-path policy decisions live.",
  keywords: [
    "SatGate sandbox",
    "AI agent budget enforcement sandbox",
    "try economic firewall",
    "AI agent API governance demo",
    "MCP governance sandbox",
    "capability token demo",
    "request-path policy enforcement",
  ],
  openGraph: {
    title: "SatGate Sandbox | Try AI Agent Budget Enforcement Free",
    description:
      "Interactive sandbox for economic access control, AI agent budgets, MCP governance, capability tokens, and request-path policy decisions.",
    url: "https://satgate.io/sandbox",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate Sandbox | Try AI Agent Budget Enforcement Free",
    description:
      "Test AI agent budget enforcement, MCP governance, and capability-token controls live.",
  },
};

export default function SandboxLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

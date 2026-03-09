import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent Management",
  alternates: { canonical: "https://satgate.io/agents" },
  description:
    "Manage AI agents with cryptographic capability tokens. Per-agent budgets, delegation trees, scope enforcement, and real-time spend tracking.",
};

export default function AgentsLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

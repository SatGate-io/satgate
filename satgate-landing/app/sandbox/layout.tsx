import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Sandbox — Try SatGate Free",
  alternates: { canonical: "https://satgate.io/sandbox" },
  description:
    "Try SatGate without signup. Interactive sandbox to test economic access control, budget enforcement, and AI agent API governance live.",
};

export default function SandboxLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

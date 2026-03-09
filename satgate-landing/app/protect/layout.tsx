import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Budget Enforcement Demo",
  alternates: { canonical: "https://satgate.io/protect" },
  description:
    "See real-time AI agent budget enforcement in action. Control API spend with per-agent caps, delegation chains, and instant revocation.",
};

export default function ProtectLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "L402 Lightning Payment Demo",
  alternates: { canonical: "https://satgate.io/pay" },
  description:
    "Monetize your API with L402 Lightning payments. Per-request pricing for AI agents with instant Bitcoin settlement and automatic payouts.",
};

export default function PayLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

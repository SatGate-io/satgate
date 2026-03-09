import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Design Partners Program",
  alternates: { canonical: "https://satgate.io/design-partners" },
  description:
    "Join SatGate as a design partner. Early access to the economic firewall for AI agent API governance. Shape the product with us.",
};

export default function DesignPartnersLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

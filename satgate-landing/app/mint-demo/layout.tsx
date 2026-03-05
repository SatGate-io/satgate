import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Token Minting Demo",
  description:
    "Mint capability tokens (macaroons) for AI agents. Set budgets, scopes, expiry, and delegation chains in seconds. Try the live demo.",
};

export default function MintDemoLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

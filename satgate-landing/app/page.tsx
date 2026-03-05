import type { Metadata } from "next";
import HomeClient from "./components/HomeClient";

export const metadata: Metadata = {
  title: "SatGate — The Economic Firewall for AI Agent Requests",
  description:
    "Control AI agent API spend at the request layer. Per-agent budgets, per-tool cost attribution, delegation hierarchies. Open source API gateway with sub-ms overhead.",
  alternates: {
    canonical: "https://satgate.io",
  },
};

export default function HomePage() {
  return <HomeClient />;
}

import type { Metadata } from "next";
import HomeClient from "./components/HomeClient";

export const metadata: Metadata = {
  title: "SatGate — The Economic Firewall for AI Agent Requests",
  description:
    "Control AI agent API spend before each request. SatGate adds per-agent budgets, per-tool attribution, delegation, and API monetization to agent traffic.",
  alternates: {
    canonical: "https://satgate.io",
  },
};

export default function HomePage() {
  return <HomeClient />;
}

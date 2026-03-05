import type { Metadata } from "next";
import GovernClient from "../components/GovernClient";

export const metadata: Metadata = {
  title: "Enterprise AI Agent Governance",
  description:
    "Govern AI agent spend across your organization. Budget enforcement, delegation hierarchies, and cost attribution for CISOs, CFOs, and engineering leaders.",
  alternates: {
    canonical: "https://satgate.io/govern",
  },
};

export default function GovernPage() {
  return <GovernClient />;
}

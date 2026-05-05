import type { Metadata } from "next";
import GovernClient from "../components/GovernClient";

export const metadata: Metadata = {
  title: "AI Agent Governance Platform: Budgets, MCP Controls, and L402 Charge",
  description:
    "Govern autonomous AI agents with request-path budget enforcement, MCP tool controls, delegation, revocation, cost attribution, and L402 Charge.",
  alternates: {
    canonical: "https://satgate.io/govern",
  },
  keywords: [
    "enterprise AI agent governance",
    "AI agent cost governance",
    "economic control plane for AI agents",
    "MCP governance for enterprises",
    "AI agent budget enforcement",
    "agent delegation controls",
    "L402 robot customer payments",
    "Observe Control Charge",
  ],
  openGraph: {
    title: "AI Agent Governance Platform: Budgets, MCP Controls, and L402 Charge",
    description:
      "Request-path budget enforcement, MCP governance, delegation controls, revocation, cost attribution, and L402 Charge for AI agents.",
    url: "https://satgate.io/govern",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent Governance Platform: Budgets, MCP Controls, and L402 Charge",
    description:
      "Enterprise AI agent governance for spend, MCP tools, delegation, revocation, attribution, and robot-customer payments.",
  },
};

const webPageSchema = {
  "@context": "https://schema.org",
  "@type": "WebPage",
  name: "AI Agent Governance Platform",
  description: metadata.description,
  url: "https://satgate.io/govern",
  dateModified: "2026-05-05",
  isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
  about: [
    { "@type": "Thing", name: "AI agent governance" },
    { "@type": "Thing", name: "economic control plane for AI agents" },
    { "@type": "Thing", name: "MCP governance for enterprises" },
    { "@type": "Thing", name: "agent delegation controls" },
    { "@type": "Thing", name: "L402 robot customer payments" },
  ],
};

const faqSchema = {
  "@context": "https://schema.org",
  "@type": "FAQPage",
  mainEntity: [
    {
      "@type": "Question",
      name: "What is AI agent governance?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "AI agent governance is the set of controls that determines which agents can call which APIs, tools, and models; how much they can spend; what authority they can delegate; and when access must be revoked. For autonomous agents, governance needs request-path enforcement, not just logs and dashboards.",
      },
    },
    {
      "@type": "Question",
      name: "What is an economic control plane for AI agents?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "An economic control plane for AI agents sits in the request path and applies budgets, prices, delegation rules, revocation, and audit before an agent reaches an upstream API, model, or MCP tool. It turns agent activity into observable, controllable, and chargeable economic events.",
      },
    },
    {
      "@type": "Question",
      name: "How should enterprises govern MCP tool usage?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "Enterprises should govern MCP tools with per-tool budgets, scoped capability tokens, task and tenant attribution, audit trails, revocation, and hard request-path policy decisions. Rate limits and dashboards are useful, but they do not replace enforcement before tool calls execute.",
      },
    },
    {
      "@type": "Question",
      name: "What is the difference between AI governance and AI agent governance?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "AI governance usually covers model risk, data policy, compliance, and human review. AI agent governance adds request-path controls for autonomous actions: budgets, tool scopes, delegated authority, revocation, cost attribution, and payment before APIs or MCP tools execute.",
      },
    },
  ],
};

export default function GovernPage() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageSchema) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(faqSchema) }}
      />
      <GovernClient />
    </>
  );
}

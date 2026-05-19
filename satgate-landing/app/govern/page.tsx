import type { Metadata } from "next";
import GovernClient from "../components/GovernClient";

export const metadata: Metadata = {
  title: "AI Agent Governance: Policy-to-Proof",
  description:
    "AI agent governance for the Economic Firewall category: scope authority, enforce request-path policy, and export Evidence Packs for every agent decision.",
  alternates: {
    canonical: "https://satgate.io/govern",
  },
  keywords: [
    "enterprise AI agent governance",
    "AI agent authority governance",
    "Economic Firewall for AI agents",
    "Policy-to-Proof governance for AI agents",
    "MCP governance for enterprises",
    "AI agent budget enforcement",
    "agent delegation controls",
    "Policy-to-Proof for AI agents",
    "Evidence Packs for AI agents",
  ],
  openGraph: {
    title: "AI Agent Governance: Policy-to-Proof",
    description:
      "Give enterprise agents bounded economic authority with request-path enforcement and Evidence Pack proof.",
    url: "https://satgate.io/govern",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent Governance: Policy-to-Proof",
    description:
      "Delegate bounded authority, enforce policy, and export Policy-to-Proof evidence for enterprise AI agents with SatGate.",
  },
};

const webPageSchema = {
  "@context": "https://schema.org",
  "@type": "WebPage",
  name: "AI Agent Governance Platform",
  description: metadata.description,
  url: "https://satgate.io/govern",
  dateModified: "2026-05-18",
  isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
  about: [
    { "@type": "Thing", name: "AI agent governance" },
    { "@type": "Thing", name: "Economic Firewall for AI agents" },
    { "@type": "Thing", name: "Policy-to-Proof governance for AI agents" },
    { "@type": "Thing", name: "MCP governance for enterprises" },
    { "@type": "Thing", name: "agent delegation controls" },
    { "@type": "Thing", name: "Policy-to-Proof for AI agents" },
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
      name: "What is Policy-to-Proof governance for AI agents?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "Policy-to-Proof governance for AI agents sits in the request path, applies scopes, budgets, delegation rules, and revocation before an agent reaches an upstream API, model, or MCP tool, then preserves an Evidence Pack so the decision can be verified later.",
      },
    },
    {
      "@type": "Question",
      name: "How should enterprises govern MCP tool usage?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "Enterprises should govern MCP tools with per-tool budgets, scoped capability tokens, task and tenant attribution, Evidence Packs, revocation, and hard request-path policy decisions. Rate limits and dashboards are useful, but they do not replace enforcement before tool calls execute.",
      },
    },
    {
      "@type": "Question",
      name: "What is the difference between AI governance and AI agent governance?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "AI governance usually covers model risk, data policy, compliance, and human review. AI agent governance adds request-path controls for autonomous actions: scopes, budgets, delegated authority, revocation, denial reasons, spend attribution, and proof before APIs or MCP tools execute.",
      },
    },
    {
      "@type": "Question",
      name: "Is SatGate tied to x402, L402, AgentCore Payments, or Pay.sh?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "No. x402, L402, AgentCore Payments, Pay.sh, and related rails make it easier for agents to call paid services. SatGate is protocol-independent: it records the requesting agent, allowed action, policy basis, spend context, and evidence needed for audit, review, and control — payment or not.",
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

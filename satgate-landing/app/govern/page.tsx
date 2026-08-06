import type { Metadata } from "next";
import GovernClient from "../components/GovernClient";

export const metadata: Metadata = {
  title: "AI Agent Governance: Enterprise Policy Enforcement Tools",
  description:
    "Govern AI agents with enterprise API policy enforcement, MCP governance policy, budget controls, governed data access, and Evidence Packs.",
  alternates: {
    canonical: "https://satgate.io/govern",
  },
  keywords: [
    "enterprise AI agent governance",
    "enterprise API policy enforcement tools",
    "governed AI delivery",
    "governed data access for AI agents",
    "govern AI agents with MCP",
    "MCP governance policy for enterprise teams",
    "AI agent authority governance",
    "Agent Authority & Accountability Layer",
    "Policy-to-Proof governance for AI agents",
    "MCP governance for enterprises",
    "AI agent budget enforcement",
    "agent delegation controls",
    "Policy-to-Proof for AI agents",
    "Evidence Packs for AI agents",
  ],
  openGraph: {
    title: "AI Agent Governance: Enterprise Policy Enforcement Tools",
    description:
      "Govern AI agents before execution with API policy enforcement, MCP governance policy, budget controls, governed data access, and Evidence Packs.",
    url: "https://satgate.io/govern",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "AI Agent Governance for Enterprise Teams",
    description:
      "Request-path policy enforcement for AI agents, MCP tools, data access, budgets, delegation, and proof.",
  },
};

const webPageSchema = {
  "@context": "https://schema.org",
  "@type": "WebPage",
  name: "AI Agent Governance: Enterprise Policy Enforcement Tools",
  description: metadata.description,
  url: "https://satgate.io/govern",
  dateModified: "2026-08-06",
  isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
  about: [
    { "@type": "Thing", name: "AI agent governance" },
    { "@type": "Thing", name: "enterprise API policy enforcement tools" },
    { "@type": "Thing", name: "governed AI delivery" },
    { "@type": "Thing", name: "governed data access for AI agents" },
    { "@type": "Thing", name: "govern AI agents with MCP" },
    { "@type": "Thing", name: "MCP governance policy for enterprise teams" },
    { "@type": "Thing", name: "Agent Authority & Accountability Layer" },
    { "@type": "Thing", name: "Policy-to-Proof governance for AI agents" },
    { "@type": "Thing", name: "MCP governance for enterprises" },
    { "@type": "Thing", name: "agent delegation controls" },
    { "@type": "Thing", name: "Policy-to-Proof for AI agents" },
  ],
};

const governanceControlsSchema = {
  "@context": "https://schema.org",
  "@type": "ItemList",
  name: "AI agent governance control checklist",
  description:
    "Request-path controls for governing AI agent access, budget, delegation, MCP tool usage, revocation, paid-rail context, and Evidence Pack proof before agents execute actions.",
  itemListElement: [
    {
      "@type": "ListItem",
      position: 1,
      name: "Authority before execution",
      description:
        "Check agent identity, tenant, task, route scope, tool scope, and capability caveats before forwarding the request.",
    },
    {
      "@type": "ListItem",
      position: 2,
      name: "Budget and MCP tool enforcement",
      description:
        "Enforce per-agent, per-route, per-session, and per-tool budget controls in the request path rather than relying on dashboards after spend occurs.",
    },
    {
      "@type": "ListItem",
      position: 3,
      name: "Delegation and revocation proof",
      description:
        "Record delegated authority, attenuated child credentials, revocation events, and the first denied request after revoke.",
    },
    {
      "@type": "ListItem",
      position: 4,
      name: "Evidence Pack export",
      description:
        "Preserve receipts for policy basis, denial reasons, spend context, paid-rail context, and audit-ready verification.",
    },
  ],
};

const enterprisePolicySchema = {
  "@context": "https://schema.org",
  "@type": "ItemList",
  name: "Enterprise AI agent governance policy model",
  description:
    "Policy areas enterprise teams need when governing AI agents across APIs, MCP tools, data access, spend, delegation, and Evidence Pack proof.",
  itemListElement: [
    {
      "@type": "ListItem",
      position: 1,
      name: "Enterprise API policy enforcement",
      description:
        "Apply route, method, tenant, identity, scope, and rate policy before the agent reaches internal or external APIs.",
    },
    {
      "@type": "ListItem",
      position: 2,
      name: "MCP governance policy",
      description:
        "Govern MCP tool calls with allowlists, per-tool budgets, tenant boundaries, delegation depth, and audit receipts.",
    },
    {
      "@type": "ListItem",
      position: 3,
      name: "Governed data access",
      description:
        "Limit which datasets, records, exports, and enrichment services an AI agent can touch for the current task.",
    },
    {
      "@type": "ListItem",
      position: 4,
      name: "Governed AI delivery",
      description:
        "Move from policy documents to request-path enforcement, denial, revocation, and proof that security, finance, and platform teams can review.",
    },
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
        text: "AI agent governance is the set of controls that determines which agents can call which APIs, tools, and models; how much they can spend; what authority they can delegate; and when access must be revoked. For autonomous agents, governance needs request-path enforcement, not just logs, dashboards, and postmortems.",
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
      name: "Where can enterprises enforce AI agent API policy?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "Enterprises should enforce AI agent API policy in the request path, before calls reach APIs, models, MCP tools, or paid services. SatGate checks identity, tenant, route, scope, budget, data-access caveats, delegation, and revocation before forwarding the request.",
      },
    },
    {
      "@type": "Question",
      name: "How do you govern AI agents with MCP at scale?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "To govern AI agents with MCP at scale, put a gateway or proxy in front of MCP tools, define per-tool allowlists and budgets, attach tenant and task identity, constrain delegation, enforce revocation before execution, and export Evidence Pack receipts for every allow or deny decision.",
      },
    },
    {
      "@type": "Question",
      name: "Is SatGate tied to x402, L402, AgentCore Payments, or Pay.sh?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "No. x402, L402, AgentCore Payments, Pay.sh, and related rails make it easier for agents to call paid services. SatGate is protocol-independent: it records the requesting agent, allowed action, policy basis, spend context, and Evidence Pack receipts needed for accountability and control — payment or not.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(governanceControlsSchema) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(enterprisePolicySchema) }}
      />
      <GovernClient />
    </>
  );
}

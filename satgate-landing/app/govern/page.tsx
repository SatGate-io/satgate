import type { Metadata } from "next";
import GovernClient from "../components/GovernClient";

export const metadata: Metadata = {
  title: "Enterprise AI Agent Governance: Rules and Receipts",
  description:
    "Govern AI agents before they act: Observe usage, Control budgets and access, and keep signed receipts across APIs, MCP, and payment methods.",
  alternates: {
    canonical: "https://satgate.io/govern",
  },
  keywords: [
    "enterprise AI agent governance",
    "AI agent permissions",
    "Agent Permissions and Receipts",
    "rules and receipts for AI agents",
    "MCP governance for enterprises",
    "AI agent budget enforcement",
    "agent delegation controls",
    "rules and receipts",
    "signed receipts for AI agents",
  ],
  openGraph: {
    title: "Enterprise AI Agent Governance: Rules and Receipts",
    description:
      "Govern AI agents before they act: Observe usage, Control budgets and access, and keep signed receipts across APIs, MCP, and payment methods.",
    url: "https://satgate.io/govern",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "Enterprise AI Agent Governance: Rules and Receipts",
    description:
      "Observe agent usage, Control access and budgets before the agent acts, and keep a signed receipt for every decision.",
  },
};

const webPageSchema = {
  "@context": "https://schema.org",
  "@type": "WebPage",
  name: "Enterprise AI Agent Governance Platform",
  description: metadata.description,
  url: "https://satgate.io/govern",
  dateModified: "2026-06-01",
  isPartOf: { "@type": "WebSite", name: "SatGate", url: "https://satgate.io" },
  about: [
    { "@type": "Thing", name: "AI agent governance" },
    { "@type": "Thing", name: "Agent Permissions and Receipts" },
    { "@type": "Thing", name: "rules and receipts for AI agents" },
    { "@type": "Thing", name: "MCP governance for enterprises" },
    { "@type": "Thing", name: "agent delegation controls" },
    { "@type": "Thing", name: "rules and receipts" },
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
        text: "AI agent governance is the set of controls that determines which agents can call which APIs, tools, and models; how much they can spend; what permissions they can pass down; and when access must be revoked. For autonomous agents, governance needs a check before the request goes through, not just logs, dashboards, and postmortems.",
      },
    },
    {
      "@type": "Question",
      name: "What are rules and receipts for AI agents?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "Rules and receipts for AI agents apply before the request goes through. They apply scopes, budgets, delegation rules, and revocation before an agent reaches an upstream API, model, or MCP tool, then keep a signed receipt (Evidence Pack) so the decision can be verified later.",
      },
    },
    {
      "@type": "Question",
      name: "How should enterprises govern MCP tool usage?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "Enterprises should govern MCP tools with per-tool budgets, scoped capability tokens, task and tenant attribution, signed receipts, revocation, and hard policy decisions before the request goes through. Rate limits and dashboards are useful, but they do not replace enforcement before tool calls execute.",
      },
    },
    {
      "@type": "Question",
      name: "What is the difference between AI governance and AI agent governance?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "AI governance usually covers model risk, data policy, compliance, and human review. AI agent governance adds controls before the request goes through for autonomous actions: scopes, budgets, delegated permissions, revocation, denial reasons, spend attribution, and proof before APIs or MCP tools execute.",
      },
    },
    {
      "@type": "Question",
      name: "Is SatGate tied to x402, L402, AgentCore Payments, or Pay.sh?",
      acceptedAnswer: {
        "@type": "Answer",
        text: "Lightning (L402) and USDC on Base (x402) are live, and each route opts in. AgentCore Payments and Pay.sh are planned. Payment never overrides permissions.",
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

import type { Metadata } from "next";
import HomeClient from "./components/HomeClient";

export const metadata: Metadata = {
  title: "SatGate — Economic Firewall for AI Agents",
  description:
    "Authority before execution. Evidence Packs after every decision across MCP, APIs, API keys, L402, x402, AgentCore Payments, Pay.sh, and enterprise billing.",
  alternates: {
    canonical: "https://satgate.io",
  },
  openGraph: {
    title: "SatGate — Economic Firewall for AI Agents",
    description:
      "Policy-to-Proof governance for enterprise agents: scoped authority before execution and Evidence Packs across APIs, MCP tools, and paid rails.",
    url: "https://satgate.io",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate — Economic Firewall for AI Agents",
    description:
      "Policy-to-Proof governance for enterprise agents: scoped authority before execution and Evidence Packs across APIs, MCP tools, and paid rails.",
  },
};

export default function HomePage() {
  const webPageJsonLd = {
    '@context': 'https://schema.org',
    '@graph': [
      {
        '@type': 'Organization',
        name: 'SatGate',
        url: 'https://satgate.io',
        logo: 'https://satgate.io/logo_white_transparent.png',
        description: 'SatGate is Policy-to-Proof governance for AI agent authority: decisions before execution and Evidence Packs after every action.',
      },
      {
        '@type': 'WebSite',
        name: 'SatGate',
        url: 'https://satgate.io',
        publisher: { '@type': 'Organization', name: 'SatGate' },
      },
      {
        '@type': 'WebPage',
        name: 'SatGate — Economic Firewall for AI Agents',
        url: 'https://satgate.io',
        description: 'Authority before execution. Evidence Packs after every decision across MCP, APIs, API keys, L402, x402, AgentCore Payments, Pay.sh, and enterprise billing.',
        datePublished: '2026-04-30',
        dateModified: '2026-05-05',
        isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
        about: [
          { '@type': 'Thing', name: 'Policy-to-Proof governance' },
          { '@type': 'Thing', name: 'Evidence Packs' },
          { '@type': 'Thing', name: 'authority before execution' },
          { '@type': 'Thing', name: 'MCP governance' },
          { '@type': 'Thing', name: 'rail-neutral paid-rail governance' },
        ],
      },
    ],
  };

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'What is SatGate?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate is Policy-to-Proof governance for enterprise agents. It sits in the request path to check authority, enforce policy and budgets, and preserve Evidence Packs across APIs, MCP tools, and paid rails.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate govern AI agents?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate applies scoped authority, per-agent policy, revocation, and budgets before each request reaches an API or MCP tool, so unauthorized actions and expensive calls can be blocked before they happen.',
        },
      },
      {
        '@type': 'Question',
        name: 'What are Observe, Control, and Charge?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Observe tracks agent traffic and cost without blocking. Control enforces budgets and scoped policy for internal agents. Charge preserves authorization evidence around external paid access across L402, x402, API-key, or enterprise billing rails.',
        },
      },
    ],
  };

  return (
    <>
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(webPageJsonLd) }} />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }} />
      <HomeClient />
    </>
  );
}

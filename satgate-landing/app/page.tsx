import type { Metadata } from "next";
import HomeClient from "./components/HomeClient";

export const metadata: Metadata = {
  title: { absolute: "SatGate: an economic firewall for AI agents" },
  description:
    "Keep your AI agents within budget, charge external agents for access to your API, and get a signed receipt for every decision.",
  alternates: {
    canonical: "https://satgate.io",
  },
  openGraph: {
    title: "SatGate: an economic firewall for AI agents",
    description:
      "Keep your agents within budget. Make external agents pay before they use your API. Get a signed receipt for every decision.",
    url: "https://satgate.io",
    type: "website",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate: an economic firewall for AI agents",
    description:
      "Keep your agents within budget. Make external agents pay before they use your API. Get a signed receipt for every decision.",
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
        description: 'SatGate is an economic firewall for agent workloads. It checks budgets and permissions on each request and signs a receipt for every decision.',
      },
      {
        '@type': 'WebSite',
        name: 'SatGate',
        url: 'https://satgate.io',
        publisher: { '@type': 'Organization', name: 'SatGate' },
      },
      {
        '@type': 'WebPage',
        name: 'SatGate | Economic Firewall for AI Agents',
        url: 'https://satgate.io',
        description: 'SatGate checks what each AI agent is allowed to do and spend before its request reaches your API, and signs a receipt for every decision.',
        datePublished: '2026-04-30',
        dateModified: '2026-09-26',
        isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
        about: [
          { '@type': 'Thing', name: 'rules and receipts for AI agents' },
          { '@type': 'Thing', name: 'signed receipts (Evidence Packs)' },
          { '@type': 'Thing', name: 'permission before the agent acts' },
          { '@type': 'Thing', name: 'MCP governance' },
          { '@type': 'Thing', name: 'economic resource admission' },
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
          text: 'SatGate is a gateway that sits in front of your APIs and MCP tools. It keeps your own agents within the budgets and permissions you set, charges external agents on the routes you choose, and signs a receipt for each decision.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate control what agents do?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Before a request reaches your API or MCP tool, SatGate checks the agent’s permissions, its remaining budget and whether its token was revoked. If a check fails, the request stops at SatGate.',
        },
      },
      {
        '@type': 'Question',
        name: 'How do I give an agent a budget?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'You set the budget, the permissions and how many times it can hand work to a sub-agent. The agent gets a token with those limits built in. Every allow, refusal, payment, handoff and revoke gets a receipt.',
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

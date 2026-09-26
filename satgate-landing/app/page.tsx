import type { Metadata } from "next";
import HomeClient from "./components/HomeClient";

export const metadata: Metadata = {
  title: "SatGate | Economic Firewall for AI Agents",
  description:
    "Protect internal budgets and external access with request-path economic controls. Give agents scoped authority for useful work and verify the recorded decisions.",
  alternates: {
    canonical: "https://satgate.io",
  },
  openGraph: {
    title: "SatGate | Economic Firewall for AI Agents",
    description:
      "Economic defense for APIs and tools: enforce internal budgets, scope external access and require payment where policy calls for it. Evidence Pack proof includes decision receipts and paid-call receipts on supported governed paths.",
    url: "https://satgate.io",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate | Economic Firewall for AI Agents",
    description:
      "Economic defense for APIs and tools: enforce internal budgets, scope external access and require payment where policy calls for it. Evidence Pack proof includes decision receipts and paid-call receipts on supported governed paths.",
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
        description: 'SatGate is an economic firewall for agent workloads. It combines request-path budget and scope controls with signed evidence of governed decisions.',
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
        description: 'SatGate governs agent authority before execution so humans, platforms, and upstream APIs can trust what agents access, spend, and prove.',
        datePublished: '2026-04-30',
        dateModified: '2026-09-26',
        isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
        about: [
          { '@type': 'Thing', name: 'Policy-to-Proof governance' },
          { '@type': 'Thing', name: 'Evidence Packs' },
          { '@type': 'Thing', name: 'authority before execution' },
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
          text: 'SatGate is the Agent Authority & Accountability Layer for governed agent execution. It sits in the request path so humans and platforms can delegate bounded authority to agents, enforce policy and budgets, and preserve Evidence Packs for governed API, MCP tool, and paid-rail decisions.',
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
        name: 'How does SatGate give agents bounded economic authority?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Humans and platforms define policy, budgets, scope, and delegation depth. Agents consume approved API and MCP primitives through SatGate, and allowed, denied, delegated, revoked, and paid-rail decisions leave receipt-backed proof.',
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

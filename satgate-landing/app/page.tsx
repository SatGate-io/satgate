import type { Metadata } from "next";
import HomeClient from "./components/HomeClient";

export const metadata: Metadata = {
  title: "SatGate | Economic Firewall for AI Agents",
  description:
    "Protect internal budgets and external access: keep agents within budget, charge external callers for authorized access and retain verifiable evidence for audit.",
  alternates: {
    canonical: "https://satgate.io",
  },
  openGraph: {
    title: "SatGate | Economic Firewall for AI Agents",
    description:
      "Keep your agents within budget. Make external callers pay before they use your resources. Keep verifiable evidence of gateway decisions.",
    url: "https://satgate.io",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate | Economic Firewall for AI Agents",
    description:
      "Keep your agents within budget. Make external callers pay before they use your resources. Keep verifiable evidence of gateway decisions.",
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
          text: 'SatGate is an economic firewall for AI agents. Economic control enforces internal budgets and permissions. Economic admission charges external callers for authorized access on paid routes. Signed decision receipts, paid-call receipts and Evidence Pack proof support independent audit review.',
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

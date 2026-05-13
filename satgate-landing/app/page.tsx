import type { Metadata } from "next";
import HomeClient from "./components/HomeClient";

export const metadata: Metadata = {
  title: "SatGate — Economic Firewall for Agentic API Access",
  description:
    "SatGate gives agents bounded economic authority so humans, platforms, and upstream APIs can trust what they consume, spend, and prove.",
  alternates: {
    canonical: "https://satgate.io",
  },
  openGraph: {
    title: "SatGate — Economic Firewall for Agentic API Access",
    description:
      "Policy-to-Proof governance for enterprise agents: bounded economic authority before execution and Evidence Packs after every decision.",
    url: "https://satgate.io",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate — Economic Firewall for Agentic API Access",
    description:
      "Policy-to-Proof governance for enterprise agents: bounded economic authority before execution and Evidence Packs after every decision.",
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
        description: 'SatGate is the Economic Firewall for agentic API access: bounded economic authority before execution and Evidence Packs after every decision.',
      },
      {
        '@type': 'WebSite',
        name: 'SatGate',
        url: 'https://satgate.io',
        publisher: { '@type': 'Organization', name: 'SatGate' },
      },
      {
        '@type': 'WebPage',
        name: 'SatGate — Economic Firewall for Agentic API Access',
        url: 'https://satgate.io',
        description: 'SatGate gives agents bounded economic authority so humans, platforms, and upstream APIs can trust what they consume, spend, and prove.',
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
          text: 'SatGate is the Economic Firewall for agentic API access. It sits in the request path so humans and platforms can delegate bounded economic authority to agents, enforce policy and budgets, and preserve Evidence Packs across APIs, MCP tools, and paid rails.',
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
          text: 'Humans and platforms define policy, budgets, scope, and delegation depth. Agents consume approved API and MCP primitives through SatGate, and every approval, denial, spend event, delegation, and revocation leaves receipt-backed proof.',
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

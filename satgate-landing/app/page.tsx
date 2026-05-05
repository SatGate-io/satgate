import type { Metadata } from "next";
import HomeClient from "./components/HomeClient";

export const metadata: Metadata = {
  title: "SatGate — The Economic Firewall for AI Agent Requests",
  description:
    "Control AI agent API spend before each request. SatGate adds per-agent budgets, per-tool attribution, delegation, and API monetization to agent traffic.",
  alternates: {
    canonical: "https://satgate.io",
  },
  openGraph: {
    title: "SatGate — The Economic Firewall for AI Agent Requests",
    description:
      "Control AI agent API spend before each request with budgets, per-tool attribution, delegation controls, and L402 API monetization.",
    url: "https://satgate.io",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate — Economic Firewall for AI Agent Requests",
    description:
      "Control AI agent API spend before each request with budgets, per-tool attribution, delegation controls, and L402 API monetization.",
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
        description: 'SatGate is the economic firewall and economic control plane for AI agent requests.',
      },
      {
        '@type': 'WebSite',
        name: 'SatGate',
        url: 'https://satgate.io',
        publisher: { '@type': 'Organization', name: 'SatGate' },
      },
      {
        '@type': 'WebPage',
        name: 'SatGate — The Economic Firewall for AI Agent Requests',
        url: 'https://satgate.io',
        description: 'Control AI agent API spend before each request with per-agent budgets, per-tool attribution, delegation, and L402 API monetization.',
        datePublished: '2026-04-30',
        dateModified: '2026-05-05',
        isPartOf: { '@type': 'WebSite', name: 'SatGate', url: 'https://satgate.io' },
        about: [
          { '@type': 'Thing', name: 'economic control plane for AI agents' },
          { '@type': 'Thing', name: 'economic firewall for AI agents' },
          { '@type': 'Thing', name: 'request-path budget enforcement' },
          { '@type': 'Thing', name: 'L402 API monetization' },
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
          text: 'SatGate is an economic firewall and economic control plane for AI agents. It sits in the request path to observe usage, enforce budgets, attribute spend, and charge external agents before upstream APIs execute.',
        },
      },
      {
        '@type': 'Question',
        name: 'How does SatGate control AI agent spend?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'SatGate applies per-agent, per-tool, per-team, and per-task budgets before each request reaches an API or MCP tool, so runaway loops and expensive calls can be blocked before spend occurs.',
        },
      },
      {
        '@type': 'Question',
        name: 'What are Observe, Control, and Charge?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'Observe tracks agent traffic and cost without blocking. Control enforces budgets and scoped policy for internal agents. Charge uses L402 Lightning payments to monetize external agent and robot-customer API access.',
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

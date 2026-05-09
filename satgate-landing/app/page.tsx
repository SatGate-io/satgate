import type { Metadata } from "next";
import HomeClient from "./components/HomeClient";

export const metadata: Metadata = {
  title: "SatGate — The Economic Firewall for AI Agent Requests",
  description:
    "Govern internal enterprise agents before each request. SatGate adds scoped authority, per-agent budgets, delegation, revocation, and evidence across MCP, API keys, L402, x402, and enterprise billing.",
  alternates: {
    canonical: "https://satgate.io",
  },
  openGraph: {
    title: "SatGate — The Economic Firewall for AI Agent Requests",
    description:
      "Govern what internal enterprise agents can do with scoped authority, budgets, runtime enforcement, and evidence across internal APIs and paid external rails.",
    url: "https://satgate.io",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate — Economic Firewall for AI Agent Requests",
    description:
      "Govern what internal enterprise agents can do with scoped authority, budgets, runtime enforcement, and evidence across internal APIs and paid external rails.",
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
        description: 'Govern internal enterprise agents before each request with scoped authority, per-agent budgets, delegation, revocation, and evidence across MCP, API keys, L402, x402, and enterprise billing.',
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
          text: 'SatGate is an economic control plane for internal enterprise agents. It sits in the request path to scope authority, enforce policy and budgets, prove revocation, and preserve evidence across internal APIs and paid external calls.',
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

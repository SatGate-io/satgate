import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  metadataBase: new URL("https://satgate.io"),
  title: {
    default: "SatGate: an economic firewall for AI agents",
    template: "%s | SatGate",
  },
  description:
    "Budgets and permissions for your AI agents, paid access for external agents where you choose, and a signed receipt for every decision.",
  keywords: [
    "rules and receipts",
    "signed receipts",
    "permission before the agent acts",
    "payment rules that work with any payment method",
    "MCP governance",
    "AI agent gateway",
    "API cost control",
    "AI agent budget enforcement",
    "MCP proxy",
    "economic firewall",
    "macaroon tokens",
    "charging external agents",
    "agent spend management",
    "API governance",
    "AI agent API gateway",
    "economic access control",
    "capability tokens",
    "agent delegation",
    "API monetization",
    "charging external agents API",
    "AI agent cost control",
    "MCP budget enforcement",
    "agent economy",
    "API security gateway",
    "Fiat402",
  ],
  openGraph: {
    title: "SatGate: an economic firewall for AI agents",
    description:
      "Budgets and permissions for your AI agents, paid access for external agents where you choose, and a signed receipt for every decision.",
    url: "https://satgate.io",
    siteName: "SatGate",
    images: [
      {
        url: "/og.png",
        width: 1200,
        height: 630,
        alt: "SatGate: an economic firewall for AI agents",
      },
    ],
    locale: "en_US",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate: an economic firewall for AI agents",
    description:
      "Budgets and permissions for your AI agents, paid access for external agents where you choose, and a signed receipt for every decision.",
    images: ["/og.png"],
  },
  robots: {
    index: true,
    follow: true,
  },
  icons: {
    icon: "/logo.png",
    apple: "/logo.png",
  },
};

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "Organization",
      name: "SatGate",
      url: "https://satgate.io",
      logo: "https://satgate.io/logo.png",
      description:
        "SatGate is an economic firewall for AI agents. Before a request reaches your API or MCP tool, it checks the agent's budget and permissions, and payment where you charge for access. Each decision gets a signed receipt.",
      sameAs: ["https://github.com/SatGate-io/satgate"],
      contactPoint: {
        "@type": "ContactPoint",
        email: "contact@satgate.io",
        contactType: "sales",
      },
    },
    {
      "@type": "SoftwareApplication",
      name: "SatGate",
      applicationCategory: "DeveloperApplication",
      operatingSystem: "Any",
      description:
        "Economic firewall for AI agent API requests. Per-agent budgets, per-tool cost attribution, delegation hierarchies, and MCP proxy support.",
      offers: {
        "@type": "Offer",
        price: "0",
        priceCurrency: "USD",
      },
      url: "https://satgate.io",
    },
  ],
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
        />
        {children}
      </body>
    </html>
  );
}

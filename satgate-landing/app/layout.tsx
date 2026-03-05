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
    default: "SatGate — The Economic Firewall for AI Agent Requests",
    template: "%s | SatGate",
  },
  description:
    "SatGate is an open-source API gateway that adds economic governance to AI agent traffic. Per-agent budgets, per-tool cost attribution, delegation hierarchies, and MCP proxy support. Sub-ms overhead.",
  keywords: [
    "AI agent gateway",
    "API cost control",
    "AI agent budget enforcement",
    "MCP proxy",
    "economic firewall",
    "macaroon tokens",
    "L402 payments",
    "agent spend management",
    "API governance",
    "AI agent API gateway",
    "economic access control",
    "capability tokens",
    "agent delegation",
    "API monetization",
    "Lightning payments API",
    "AI agent cost control",
    "MCP budget enforcement",
    "agent economy",
    "API security gateway",
    "Fiat402",
  ],
  openGraph: {
    title: "SatGate — The Economic Firewall for AI Agent Requests",
    description:
      "Control AI agent API spend at the request layer. Per-agent budgets, per-tool cost attribution, delegation hierarchies. Open source with sub-ms overhead.",
    url: "https://satgate.io",
    siteName: "SatGate",
    images: [
      {
        url: "/logo.png",
        width: 512,
        height: 512,
        alt: "SatGate — Economic Firewall for AI Agents",
      },
    ],
    locale: "en_US",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "SatGate — The Economic Firewall for AI Agent Requests",
    description:
      "Control AI agent API spend at the request layer. Per-agent budgets, cost attribution, delegation hierarchies. Open source.",
    images: ["/logo.png"],
  },
  robots: {
    index: true,
    follow: true,
  },
  alternates: {
    canonical: "https://satgate.io",
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
        "Open-source API gateway that adds economic governance to AI agent traffic. Budget enforcement, cost attribution, and monetization for the agent economy.",
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

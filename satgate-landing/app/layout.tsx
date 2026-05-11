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
    default: "SatGate — Economic Firewall for AI Agents",
    template: "%s | SatGate",
  },
  description:
    "SatGate governs AI agent authority before execution across MCP, APIs, API keys, and paid rails, then exports Evidence Packs for every decision.",
  keywords: [
    "Policy-to-Proof",
    "Evidence Packs",
    "authority before execution",
    "rail-neutral paid-rail governance",
    "MCP governance",
    "AI agent gateway",
    "API cost control",
    "AI agent budget enforcement",
    "MCP proxy",
    "economic firewall",
    "macaroon tokens",
    "paid-rail context",
    "agent spend management",
    "API governance",
    "AI agent API gateway",
    "economic access control",
    "capability tokens",
    "agent delegation",
    "API monetization",
    "paid-rail context API",
    "AI agent cost control",
    "MCP budget enforcement",
    "agent economy",
    "API security gateway",
    "Fiat402",
  ],
  openGraph: {
    title: "SatGate — Economic Firewall for AI Agents",
    description:
      "Authority before execution. Evidence Packs after every decision across MCP, APIs, API keys, and rail-neutral paid-rail governance.",
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
    title: "SatGate — Economic Firewall for AI Agents",
    description:
      "Policy-to-Proof governance for enterprise agents across MCP, APIs, API keys, and paid rails.",
    images: ["/logo.png"],
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
        "Policy-to-Proof governance for AI agent authority. SatGate checks scope, budgets, delegation, MCP access, and paid-rail policy before execution, then exports Evidence Packs.",
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

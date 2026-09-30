import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Economic Firewall Readiness Grader',
  description: 'Grade what an AI agent is allowed to do: identity, budgets, MCP tools, revocation, signed receipt capture, routing, and payment details.',
  alternates: { canonical: 'https://satgate.io/economic-firewall-readiness-grader' },
  keywords: [
    'economic firewall readiness grader',
    'AI agent governance assessment',
    'AI agent spend control checklist',
    'agent API governance readiness',
    'MCP governance readiness',
    'AI agent budget enforcement assessment',
  ],
  openGraph: {
    title: 'Economic Firewall Readiness Grader',
    description: 'Assess whether your agent/API stack is ready for autonomous authority, delegated tools, budget enforcement, revocation, signed receipt capture, and payment details.',
    url: 'https://satgate.io/economic-firewall-readiness-grader',
    type: 'website',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Economic Firewall Readiness Grader',
    description: 'Grade what an AI agent is allowed to do: identity, budgets, MCP tools, revocation, signed receipt capture, routing, and payment details.',
  },
};

export default function EconomicFirewallReadinessGraderLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

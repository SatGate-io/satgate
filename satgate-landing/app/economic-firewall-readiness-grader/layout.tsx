import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Economic Firewall Readiness Grader',
  description: 'Grade your AI agent economic governance readiness across identity, budgets, MCP tools, revocation, audit, routing, and payments.',
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
    description: 'Assess whether your agent/API stack is ready for autonomous spend, delegated tools, budget enforcement, revocation, audit, and L402 payments.',
    url: 'https://satgate.io/economic-firewall-readiness-grader',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Economic Firewall Readiness Grader',
    description: 'Grade your readiness for AI agent economic governance: identity, budgets, MCP tools, revocation, audit, routing, and payments.',
  },
};

export default function EconomicFirewallReadinessGraderLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

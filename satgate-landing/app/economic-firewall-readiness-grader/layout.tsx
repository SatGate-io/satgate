import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Economic Firewall Readiness Grader | AI Agent Governance Score',
  description: 'Build an AI agent governance readiness score and decision package across identity, budgets, MCP tools, revocation, Evidence Pack capture, routing, and paid rails.',
  alternates: { canonical: 'https://satgate.io/economic-firewall-readiness-grader' },
  keywords: [
    'economic firewall readiness grader',
    'economic firewall readiness decision package',
    'AI agent governance assessment',
    'AI agent governance readiness score',
    'AI agent spend control checklist',
    'agent API governance readiness',
    'MCP governance readiness',
    'AI agent budget enforcement assessment',
  ],
  openGraph: {
    title: 'Economic Firewall Readiness Grader | AI Agent Governance Score',
    description: 'Assess whether your agent/API stack is ready for autonomous authority, delegated tools, budget enforcement, revocation, Evidence Pack capture, and paid-rail context.',
    url: 'https://satgate.io/economic-firewall-readiness-grader',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Economic Firewall Readiness Grader | AI Agent Governance Score',
    description: 'Grade AI agent authority readiness and build a decision package: identity, budgets, MCP tools, revocation, Evidence Pack capture, routing, and paid-rail context.',
  },
};

export default function EconomicFirewallReadinessGraderLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

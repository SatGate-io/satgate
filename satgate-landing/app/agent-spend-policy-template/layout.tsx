import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Agent Budget Policy Template: Rules and Receipts',
  description:
    'Generate YAML and JSON agent budget policy with authority, MCP tool caps, revocation, receipts, and signed receipt (Evidence Pack) fields.',
  alternates: { canonical: 'https://satgate.io/agent-spend-policy-template' },
  keywords: [
    'agent budget policy template',
    'AI agent budget policy',
    'MCP tool budget policy',
    'agent spend policy template',
    'rules and receipts',
    'signed receipt',
  ],
  openGraph: {
    title: 'Agent Budget Policy Template: Rules and Receipts',
    description:
      'Generate copyable YAML and JSON policies for AI agent permissions, budgets, MCP tool caps, revocation, receipts, and signed receipts.',
    url: 'https://satgate.io/agent-spend-policy-template',
    type: 'website',
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "SatGate: an economic firewall for AI agents" }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Agent Budget Policy Template: Rules and Receipts',
    description:
      'Generate AI agent budget policy templates with scoped authority, revocation, receipts, and receipt fields.',
  },
};

export default function AgentSpendPolicyTemplateLayout({ children }: { children: React.ReactNode }) {
  return children;
}

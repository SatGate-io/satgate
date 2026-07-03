import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'SatGate Docs',
  description: 'SatGate public documentation entry point for Policy-to-Proof, Evidence Packs, MCP gateway, SDKs, and governance concepts.',
  alternates: { canonical: 'https://satgate.io/docs' },
};

const docs = [
  ['Verify Evidence Packs', '/verify-evidence-pack', 'Independently verify signed receipts, JWKS, pack mirrors, and tamper evidence.'],
  ['Policy-to-Proof', '/policy-to-proof', 'How SatGate turns authority decisions into portable evidence.'],
  ['Evidence Pack Demo', '/evidence-pack-demo', 'Explore a buyer-readable sample artifact and schema.'],
  ['MCP Gateway', '/mcp-gateway', 'Govern MCP tool calls with scoped authority and budget evidence.'],
  ['Build with SatGate', '/build', 'SDK and developer starting point.'],
];

export default function DocsPage() {
  return (
    <main className="min-h-screen bg-black px-6 py-20 text-white">
      <div className="mx-auto max-w-4xl">
        <Link href="/" className="text-sm text-gray-400 hover:text-white">← Back to Home</Link>
        <h1 className="mt-10 text-5xl font-black tracking-tight">SatGate Docs</h1>
        <p className="mt-5 text-xl leading-8 text-gray-300">Public entry points for SatGate’s Economic Firewall, Policy-to-Proof artifacts, Evidence Pack verification, MCP gateway, and SDK surfaces.</p>
        <div className="mt-10 grid gap-4">
          {docs.map(([title, href, body]) => (
            <Link key={href} href={href} className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 transition hover:border-cyan-300/40">
              <h2 className="text-2xl font-black">{title}</h2>
              <p className="mt-2 text-gray-400">{body}</p>
            </Link>
          ))}
        </div>
      </div>
    </main>
  );
}

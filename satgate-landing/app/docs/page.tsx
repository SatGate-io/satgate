import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: { absolute: 'SatGate Docs' },
  description: 'SatGate public documentation entry point for rules and receipts, signed receipts, MCP gateway, SDKs, and governance concepts.',
  alternates: { canonical: 'https://satgate.io/docs' },
};

const docs = [
  ['Verify a signed receipt', '/verify-evidence-pack', 'Check a signed receipt yourself with SatGate’s published keys.'],
  ['Rules and receipts', '/policy-to-proof', 'How SatGate’s rules and receipts work.'],
  ['Signed receipt demo', '/evidence-pack-demo', 'See what a receipt looks like.'],
  ['MCP gateway', '/mcp', 'Budgets and allowed tools for MCP tool calls.'],
  ['Build with SatGate', '/build', 'SDK and developer starting point.'],
];

export default function DocsPage() {
  return (
    <main className="min-h-screen bg-black px-6 py-20 text-white">
      <div className="mx-auto max-w-4xl">
        <Link href="/" className="text-sm text-gray-400 hover:text-white">← Back to Home</Link>
        <h1 className="mt-10 text-5xl font-black tracking-tight">SatGate Docs</h1>
        <p className="mt-5 text-xl leading-8 text-gray-300">Start here: how SatGate works, how to check a receipt, MCP setup and the SDKs.</p>
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

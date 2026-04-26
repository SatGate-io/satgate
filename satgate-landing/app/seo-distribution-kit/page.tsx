import Link from 'next/link';
import { ArrowRight, Megaphone, Radio, Send, Sparkles } from 'lucide-react';

export const metadata = {
  title: 'SatGate SEO Distribution Kit: AI Agent Cost Control Launch Copy',
  description: 'Distribution copy, social snippets, backlink targets, and launch positioning for SatGate AI agent cost control tools and runaway spend data assets.',
  alternates: { canonical: 'https://satgate.io/seo-distribution-kit' },
  keywords: ['AI agent cost control distribution', 'SatGate launch kit', 'AI agent spend control social copy', 'economic firewall distribution'],
};

const assets = [
  ['Tools hub', '/tools'],
  ['Runaway spend index', '/ai-agent-runaway-spend-index'],
  ['Agent spend policy template', '/agent-spend-policy-template'],
  ['MCP proxy config generator', '/mcp-proxy-config-generator'],
  ['Runaway spend benchmark', '/ai-agent-runaway-spend-benchmark'],
];

export default function SeoDistributionKitPage() {
  return (
    <main className="min-h-screen bg-black text-gray-100">
      <section className="relative overflow-hidden border-b border-gray-900">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(168,85,247,0.18),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(34,211,238,0.14),transparent_32%)]" />
        <div className="relative mx-auto max-w-6xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-purple-500/30 bg-purple-950/30 px-4 py-2 text-sm text-purple-200">
            <Megaphone size={16} /> Distribution package
          </div>
          <h1 className="mb-8 max-w-5xl text-5xl font-extrabold tracking-tight md:text-7xl">SatGate SEO Distribution Kit</h1>
          <p className="mb-10 max-w-4xl text-xl leading-relaxed text-gray-300 md:text-2xl">
            Copy, launch angles, and promotion targets for SatGate&apos;s AI agent cost control tools, MCP policy generators, L402 pricing calculator, and recurring runaway spend data assets.
          </p>
          <a href="/distribution/satgate-seo-machine-launch-kit.md" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
            Download Markdown kit <ArrowRight size={18} />
          </a>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-6 px-6 py-20 lg:grid-cols-3">
        {[
          ['LinkedIn angle', 'AI agents create economic risk, not just security risk. SatGate controls spend before the request executes.', Radio],
          ['Short social angle', 'Dashboards report agent spend. Economic firewalls decide whether the next request is allowed to spend.', Send],
          ['Backlink angle', 'Original benchmark data and policy templates for FinOps, MCP, API monetization, and agent security communities.', Sparkles],
        ].map(([title, body, Icon]) => (
          <div key={title as string} className="rounded-2xl border border-gray-800 bg-gray-950 p-6">
            <Icon className="mb-4 text-purple-300" size={28} />
            <h2 className="mb-3 text-2xl font-bold text-white">{title as string}</h2>
            <p className="leading-relaxed text-gray-400">{body as string}</p>
          </div>
        ))}
      </section>

      <section className="border-y border-gray-900 bg-gray-950/60">
        <div className="mx-auto max-w-6xl px-6 py-20">
          <h2 className="mb-8 text-3xl font-bold text-white">Promote these pages first</h2>
          <div className="grid gap-4 md:grid-cols-2">
            {assets.map(([label, href]) => (
              <Link key={href} href={href} className="rounded-2xl border border-gray-800 bg-black p-6 transition hover:border-purple-500/50">
                <h3 className="mb-2 text-xl font-bold text-white">{label}</h3>
                <p className="text-cyan-300">satgate.io{href}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>
    </main>
  );
}

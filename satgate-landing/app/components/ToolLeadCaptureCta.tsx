import Link from 'next/link';
import { ArrowRight, ClipboardCheck, Download, Mail } from 'lucide-react';

type ToolLeadCaptureCtaProps = {
  title?: string;
  body?: string;
};

export default function ToolLeadCaptureCta({
  title = 'Turn this into an enforceable SatGate policy',
  body = 'Export the calculator result or generated policy, then have SatGate review the budget model, MCP tool costs, credential scope, and enforcement path with your team.',
}: ToolLeadCaptureCtaProps) {
  return (
    <section className="rounded-3xl border border-cyan-900/70 bg-gradient-to-br from-cyan-950/30 via-gray-950 to-purple-950/20 p-8 md:p-10">
      <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/40 px-3 py-1 text-sm font-semibold text-cyan-200">
        <ClipboardCheck size={16} /> Free policy review
      </div>
      <h2 className="mb-4 text-3xl font-bold text-white">{title}</h2>
      <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">{body}</p>
      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-2xl border border-gray-800 bg-black/50 p-5">
          <Download className="mb-3 text-cyan-300" size={24} />
          <h3 className="mb-2 font-bold text-white">Export</h3>
          <p className="text-sm leading-relaxed text-gray-400">Save the YAML, JSON, CSV, benchmark data, or calculator output your team already generated.</p>
        </div>
        <div className="rounded-2xl border border-gray-800 bg-black/50 p-5">
          <Mail className="mb-3 text-purple-300" size={24} />
          <h3 className="mb-2 font-bold text-white">Send for review</h3>
          <p className="text-sm leading-relaxed text-gray-400">Share your policy with SatGate and get feedback on budgets, revocation, audit fields, and rollout risk.</p>
        </div>
        <div className="rounded-2xl border border-gray-800 bg-black/50 p-5">
          <ClipboardCheck className="mb-3 text-green-300" size={24} />
          <h3 className="mb-2 font-bold text-white">Enforce</h3>
          <p className="text-sm leading-relaxed text-gray-400">Convert the model into request-path controls for agents, MCP tools, model routes, and paid API access.</p>
        </div>
      </div>
      <div className="mt-8 flex flex-col gap-4 sm:flex-row">
        <Link href="/design-partners" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
          Get a SatGate policy review <ArrowRight size={18} />
        </Link>
        <Link href="/agent-spend-policy-template" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
          Generate policy template
        </Link>
      </div>
    </section>
  );
}

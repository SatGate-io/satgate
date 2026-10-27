import Link from 'next/link';
import { ArrowRight, ClipboardCheck, Download, Mail } from 'lucide-react';

type ToolLeadCaptureCtaProps = {
  title?: string;
  body?: string;
};

export default function ToolLeadCaptureCta({
  title = 'Try it on one workflow',
  body = 'Start a free 14-day trial and put one agent’s calls behind a budget.',
}: ToolLeadCaptureCtaProps) {
  return (
    <section className="rounded-3xl border border-cyan-900/70 bg-gradient-to-br from-cyan-950/30 via-gray-950 to-purple-950/20 p-8 md:p-10">
      <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/40 px-3 py-1 text-sm font-semibold text-cyan-200">
        <ClipboardCheck size={16} /> Free 14-day trial
      </div>
      <h2 className="mb-4 text-3xl font-bold text-white">{title}</h2>
      <p className="mb-8 max-w-3xl text-lg leading-relaxed text-gray-300">{body}</p>
      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-2xl border border-gray-800 bg-black/50 p-5">
          <Download className="mb-3 text-cyan-300" size={24} />
          <h3 className="mb-2 font-bold text-white">Export</h3>
          <p className="text-sm leading-relaxed text-gray-400">Save the calculator output or benchmark data your team already has.</p>
        </div>
        <div className="rounded-2xl border border-gray-800 bg-black/50 p-5">
          <Mail className="mb-3 text-purple-300" size={24} />
          <h3 className="mb-2 font-bold text-white">Try it</h3>
          <p className="text-sm leading-relaxed text-gray-400">Put one agent&apos;s calls behind a budget in the free 14-day trial.</p>
        </div>
        <div className="rounded-2xl border border-gray-800 bg-black/50 p-5">
          <ClipboardCheck className="mb-3 text-green-300" size={24} />
          <h3 className="mb-2 font-bold text-white">Enforce</h3>
          <p className="text-sm leading-relaxed text-gray-400">Convert the model into controls that run before the request goes through, for agents, MCP tools and paid APIs.</p>
        </div>
      </div>
      <div className="mt-8 flex flex-col gap-4 sm:flex-row">
        <a href="https://cloud.satgate.io/cloud/signup" target="_blank" rel="noopener noreferrer" className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
          Start free trial <ArrowRight size={18} />
        </a>
        <Link href="/build" className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-700 px-6 py-3 font-bold text-white transition hover:border-cyan-500">
          See how to build with SatGate
        </Link>
      </div>
    </section>
  );
}

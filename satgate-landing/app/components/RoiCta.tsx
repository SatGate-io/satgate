import Link from 'next/link';
import { ArrowRight, Calculator } from 'lucide-react';

type RoiCtaProps = {
  eyebrow?: string;
  title?: string;
  body?: string;
};

export default function RoiCta({
  eyebrow = 'Calculate the business case',
  title = 'See what unmanaged agent spend is costing you',
  body = 'Use the SatGate ROI calculator to model ghost spend from loops, runaway tool calls, and delayed budget enforcement — then turn that estimate into a free Shadow Audit.',
}: RoiCtaProps) {
  return (
    <aside className="not-prose my-12 rounded-2xl border border-purple-500/30 bg-gradient-to-br from-purple-950/40 via-gray-950 to-cyan-950/30 p-6 md:p-8">
      <div className="flex flex-col gap-5 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-900/20 px-3 py-1 text-xs font-mono text-cyan-300">
            <Calculator size={14} /> {eyebrow}
          </div>
          <h2 className="mb-3 text-2xl font-bold text-white">{title}</h2>
          <p className="max-w-2xl text-gray-300">{body}</p>
        </div>
        <Link
          href="/roi-calculator"
          className="inline-flex shrink-0 items-center justify-center gap-2 rounded-lg bg-gradient-to-r from-purple-600 to-cyan-600 px-6 py-3 font-bold text-white shadow-lg shadow-purple-500/20 transition hover:from-purple-500 hover:to-cyan-500"
        >
          Run the ROI Calculator <ArrowRight size={16} />
        </Link>
      </div>
    </aside>
  );
}

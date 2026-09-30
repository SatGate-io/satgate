'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { ArrowRight, Cable, Copy, Check } from 'lucide-react';

const cursorSnippet = `{
  "mcpServers": {
    "satgate": {
      "command": "npx",
      "args": ["-y", "satgate-mcp-bridge"],
      "env": {
        "SATGATE_URL": "https://satgate-mcp-saas.fly.dev",
        "SATGATE_TOKEN": "paste-the-token-from-cloud.satgate.io"
      }
    }
  }
}`;

const desktopNote = `Claude Desktop and OpenClaw do not use this npx command.
Open MCP Setup in the dashboard and copy the tenant /sse URL.
Send it with this header:

Authorization: Bearer paste-the-token-from-cloud.satgate.io`;

type ClientKey = 'cursor' | 'claudeCode' | 'claudeDesktop' | 'openclaw';

const clients: Record<ClientKey, { label: string; kind: 'npx' | 'sse' }> = {
  cursor: { label: 'Cursor', kind: 'npx' },
  claudeCode: { label: 'Claude Code', kind: 'npx' },
  claudeDesktop: { label: 'Claude Desktop', kind: 'sse' },
  openclaw: { label: 'OpenClaw', kind: 'sse' },
};

export default function McpProxyConfigGeneratorPage() {
  const [client, setClient] = useState<ClientKey>('cursor');
  const [copied, setCopied] = useState(false);
  const output = useMemo(
    () => (clients[client].kind === 'npx' ? cursorSnippet : desktopNote),
    [client],
  );

  async function copyOutput() {
    await navigator.clipboard.writeText(output);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <main className="min-h-screen bg-black text-gray-100 font-sans">
      <section className="border-b border-gray-900">
        <div className="mx-auto max-w-3xl px-6 py-24">
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-950/30 px-4 py-2 text-sm text-cyan-200">
            <Cable size={16} /> Cloud MCP connect snippet
          </div>
          <h1 className="mb-6 text-4xl font-extrabold tracking-tight md:text-6xl">MCP connect snippet</h1>
          <p className="mb-8 text-xl leading-relaxed text-gray-300">
            This is the snippet Cloud MCP gives Cursor and Claude Code. It does not set a dollar budget, and it does not add flags SatGate does not read.
          </p>
          <a href="https://cloud.satgate.io/cloud/signup" className="inline-flex items-center gap-2 rounded-lg bg-white px-6 py-3 font-bold text-black transition hover:bg-gray-200">
            Start free trial <ArrowRight size={18} />
          </a>
        </div>
      </section>

      <section className="mx-auto max-w-3xl px-6 py-16">
        <label className="mb-6 block">
          <span className="mb-2 block font-semibold text-white">Client</span>
          <select
            value={client}
            onChange={(event) => setClient(event.target.value as ClientKey)}
            className="w-full rounded-lg border border-gray-700 bg-black px-4 py-3 text-white outline-none focus:border-cyan-500"
          >
            {Object.entries(clients).map(([key, value]) => (
              <option key={key} value={key}>{value.label}</option>
            ))}
          </select>
        </label>

        <div className="rounded-xl border border-gray-800 bg-gray-950 p-5">
          <div className="mb-3 flex items-center justify-between">
            <p className="text-sm font-semibold text-white">
              {clients[client].kind === 'npx' ? 'mcp.json' : 'From MCP Setup'}
            </p>
            <button type="button" onClick={copyOutput} className="inline-flex items-center gap-2 text-sm text-cyan-300 hover:text-cyan-200">
              {copied ? <Check size={16} /> : <Copy size={16} />}
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
          <pre className="overflow-x-auto text-sm leading-6 text-gray-300">{output}</pre>
        </div>

        <p className="mt-6 text-sm leading-relaxed text-gray-400">
          A new token starts at 1,000 credits. A tool call costs 1 credit unless you set a per-tool cost. This snippet does not set a dollar budget. Copy the live URL and token from{' '}
          <a href="https://cloud.satgate.io/cloud/mcp/onboarding" className="text-cyan-300 hover:text-cyan-200">MCP Setup</a>.
        </p>
        <p className="mt-4 text-sm text-gray-500">
          <Link href="/mcp" className="text-cyan-300 hover:text-cyan-200">Back to the MCP quickstart</Link>
        </p>
        {/* The Cloud snippet does not read SATGATE_REQUIRE_RECEIPT_ID, SATGATE_REQUIRE_EVIDENCE_PACK_ID, SATGATE_REQUIRE_DECISION_REASON, or SATGATE_REQUIRE_POLICY_VERSION. Those names are not flags. Receipts are signed by the gateway. */}
      </section>
    </main>
  );
}

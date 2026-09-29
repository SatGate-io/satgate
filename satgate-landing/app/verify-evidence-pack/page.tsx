import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Verify a SatGate Evidence Pack',
  description: 'Check a SatGate receipt yourself, with no SatGate account: download a real Evidence Pack and verify its Ed25519 signature against the published keys.',
  alternates: { canonical: 'https://satgate.io/verify-evidence-pack' },
};

const livePackUrl = 'https://satgate-mcp-saas.fly.dev/v1/evidence/evid_MD98srRaolXE1L3rOjWQuFi2M3wgXdXG';
const samplePackFile = '/evidence/sample-mcp-budget-refusal-20260928.json';
const samplePackJwks = 'https://satgate-mcp-saas.fly.dev/.well-known/jwks.json';

export default function VerifyEvidencePackPage() {
  return (
    <main className="min-h-screen bg-black text-white">
      <section className="border-b border-white/10 px-6 py-20">
        <div className="mx-auto max-w-4xl">
          <Link href="/" className="text-sm text-gray-400 hover:text-white">← Back to Home</Link>
          <p className="mt-10 text-sm font-bold uppercase tracking-[0.24em] text-cyan-300">Check it yourself</p>
          <h1 className="mt-4 text-5xl font-black tracking-tight sm:text-6xl">Don&apos;t trust us. Check it yourself.</h1>
          <p className="mt-6 text-xl leading-8 text-gray-300">
            You don&apos;t need a SatGate account to check an Evidence Pack. Download the pack and SatGate&apos;s published public keys, then run the open-source verifier. It confirms SatGate signed the receipt and that nothing in it was changed afterward.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <a href="https://github.com/SatGate-io/satgate/tree/main/tools" className="rounded-lg bg-white px-5 py-3 text-center font-bold text-black hover:bg-gray-200">Get verifier tool</a>
            <a href={samplePackFile} className="rounded-lg border border-cyan-300/40 px-5 py-3 text-center font-bold text-cyan-100 hover:border-cyan-200">Download a real receipt</a>
          </div>
          <p className="mt-4 text-sm leading-6 text-gray-400">
            This is the signed refusal from the demo video on the home page: an MCP tool call refused with <code>budget_exhausted</code> after a 3¢ budget ran out. It&apos;s signed with the hosted MCP service&apos;s key. The <a href={livePackUrl} className="text-cyan-200 underline underline-offset-4">live copy</a> stays up until the plan&apos;s retention period ends; the downloaded file verifies the same way.
          </p>
        </div>
      </section>

      <section className="border-b border-white/10 px-6 py-16">
        <div className="mx-auto max-w-5xl">
          <h2 className="text-3xl font-black">What the result means.</h2>
          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <div className="rounded-2xl border border-amber-400/20 bg-amber-400/10 p-5 text-amber-50">
              <h3 className="font-bold"><code>valid=true</code></h3>
              <p className="mt-2 text-sm leading-6">Checked with the key stored inside the pack, this only shows the pack is consistent with itself. It doesn&apos;t show who signed it.</p>
            </div>
            <div className="rounded-2xl border border-emerald-400/20 bg-emerald-400/10 p-5 text-emerald-50">
              <h3 className="font-bold"><code>trusted_issuer_valid=true</code></h3>
              <p className="mt-2 text-sm leading-6">This is the one that matters: the signature checks out against SatGate&apos;s public keys, downloaded separately from the pack.</p>
            </div>
          </div>
        </div>
      </section>

      <section className="px-6 py-16">
        <div className="mx-auto grid max-w-5xl gap-6 lg:grid-cols-2">
          <article className="min-w-0 rounded-2xl border border-white/10 bg-white/[0.03] p-6">
            <h2 className="text-2xl font-black">Run the verifier</h2>
            <pre className="mt-5 overflow-x-auto rounded-xl bg-black p-4 text-sm text-gray-300"><code>{`python3 -m venv .venv-verify
. .venv-verify/bin/activate
pip install cryptography rfc8785
curl -fsS https://satgate.io${samplePackFile} -o pack.json
curl -fsS ${samplePackJwks} -o jwks.json
python tools/verify_evidence_pack.py pack.json \
  --jwks-file jwks.json \
  --require-trusted-issuer`}</code></pre>
          </article>

          <article className="min-w-0 rounded-2xl border border-white/10 bg-white/[0.03] p-6">
            <h2 className="text-2xl font-black">What the verifier checks</h2>
            <ul className="mt-5 space-y-3 text-gray-300">
              <li>• The receipt format version, and whether it&apos;s marked as real or a test.</li>
              <li>• It rebuilds the signed content in a standard form (RFC 8785), leaving out <code>receipt_hash</code> and <code>signature</code>.</li>
              <li>• The SHA-256 <code>receipt_hash</code> and the Ed25519 signature.</li>
              <li>• The signature against SatGate&apos;s public keys at <code>/.well-known/jwks.json</code>. A key stored in the pack is only a fallback and doesn&apos;t prove who signed.</li>
              <li>• The copies of the result and budget at the top of the pack match the signed receipt.</li>
              <li>• The optional <code>evidence_pack_hash</code>, and that tokens and secrets are blanked out.</li>
            </ul>
          </article>
        </div>
      </section>

      <section className="border-t border-white/10 px-6 py-16">
        <div className="mx-auto max-w-4xl">
          <h2 className="text-3xl font-black">Current limits</h2>
          <ul className="mt-6 space-y-3 text-gray-300">
            <li>• Signing keys are stored as platform secrets. We don&apos;t claim they sit in a hardware security module, and no outside firm has audited them.</li>
            <li>• Receipts are kept in durable storage. They aren&apos;t anchored anywhere outside SatGate, and the storage isn&apos;t write-once.</li>
            <li>• Anyone with an Evidence Pack link can open it, until the retention period ends or the pack is deleted. Share links with care.</li>
            <li>• A pass means the receipt is intact and SatGate signed it. It says nothing about what your API did next, whether a payment settled, or regulatory compliance.</li>
          </ul>
        </div>
      </section>

      <section className="border-t border-white/10 px-6 py-16">
        <div className="mx-auto max-w-4xl">
          <h2 className="text-3xl font-black">What a pass proves, and what it doesn&apos;t.</h2>
          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <div className="rounded-2xl border border-emerald-400/20 bg-emerald-400/10 p-5 text-emerald-50">
              <h3 className="font-bold">Proves</h3>
              <p className="mt-2 text-sm leading-6">The key named in <code>issuer_kid</code> signed the receipt. Any change to a signed field makes the check fail. The rest of the pack agrees with the signed receipt, and tokens and secrets are blanked out.</p>
            </div>
            <div className="rounded-2xl border border-amber-400/20 bg-amber-400/10 p-5 text-amber-50">
              <h3 className="font-bold">Does not prove by itself</h3>
              <p className="mt-2 text-sm leading-6">That a payment settled, that your API&apos;s own logs agree, that a revoke took effect everywhere instantly, or anything else beyond this one receipt.</p>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}

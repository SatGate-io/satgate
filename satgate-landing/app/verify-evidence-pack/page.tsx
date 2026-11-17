import type { Metadata } from 'next';
import Link from 'next/link';
import ReceiptChecker from './ReceiptChecker';

export const metadata: Metadata = {
  title: { absolute: 'Check a SatGate receipt' },
  description: 'Check a SatGate receipt yourself, with no SatGate account: download a real signed receipt (Evidence Pack) and verify its Ed25519 signature against the published keys.',
  alternates: { canonical: 'https://satgate.io/verify-evidence-pack' },
};

const samplePackFile = '/evidence/mcp-budget-refusal-20261005.json';
const samplePackJwks = 'https://satgate-mcp-saas.fly.dev/.well-known/jwks.json';

export default function VerifyEvidencePackPage() {
  return (
    <main className="min-h-screen bg-black text-white">
      <section className="border-b border-white/10 px-6 py-20">
        <div className="mx-auto max-w-4xl">
          <Link href="/" prefetch={false} className="text-sm text-gray-400 hover:text-white">← Back to Home</Link>
          <h1 className="mt-10 text-5xl font-black tracking-tight sm:text-6xl">Check a SatGate receipt</h1>
          <p className="mt-6 text-xl leading-8 text-gray-300">
            Choose a receipt file you downloaded from SatGate. This page checks SatGate&apos;s signature against our published keys, right in your browser. Your file is not uploaded.
          </p>
          <ReceiptChecker />
        </div>
      </section>


      <section className="px-6 py-16">
        <div className="mx-auto grid max-w-5xl gap-6 lg:grid-cols-2">
          <article className="min-w-0 rounded-2xl border border-white/10 bg-white/[0.03] p-6">
            <h2 className="text-2xl font-black">Run the verifier</h2>
            <pre className="mt-5 overflow-x-auto rounded-xl bg-black p-4 text-sm text-gray-300"><code>{`git clone https://github.com/SatGate-io/satgate && cd satgate
python3 -m venv .venv-verify
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
      <section className="border-t border-white/10 px-6 py-16">
        <div className="mx-auto max-w-4xl">
          <h2 className="text-3xl font-black">Current limits</h2>
          <ul className="mt-6 space-y-3 text-gray-300">
            <li>• Signing keys are stored as platform secrets.</li>
            <li>• Receipts are kept on SatGate&apos;s servers.</li>
            <li>• Anyone with a receipt link can open it. Share links with care.</li>
          </ul>
        </div>
      </section>
    </main>
  );
}

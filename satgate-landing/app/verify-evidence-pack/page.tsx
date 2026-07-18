import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Verify a SatGate Evidence Pack',
  description: 'Independently verify SatGate Evidence Packs with RFC 8785 canonicalization, SHA-256 receipt hashes, Ed25519 signatures, and issuer JWKS.',
  alternates: { canonical: 'https://satgate.io/verify-evidence-pack' },
};

const livePackUrl = 'https://api.satgate.io/v1/evidence/evid_QBBiz-GEI-stsaP6KS01-RL414Csuidv';

export default function VerifyEvidencePackPage() {
  return (
    <main className="min-h-screen bg-black text-white">
      <section className="border-b border-white/10 px-6 py-20">
        <div className="mx-auto max-w-4xl">
          <Link href="/" className="text-sm text-gray-400 hover:text-white">← Back to Home</Link>
          <p className="mt-10 text-sm font-bold uppercase tracking-[0.24em] text-cyan-300">Independent verification</p>
          <h1 className="mt-4 text-5xl font-black tracking-tight sm:text-6xl">Don&apos;t trust us—verify it yourself.</h1>
          <p className="mt-6 text-xl leading-8 text-gray-300">
            SatGate Evidence Packs are designed to be checked without SatGate credentials. Fetch the pack and issuer JWKS, canonicalize the signed receipt with RFC 8785 JCS, recompute its SHA-256 hash, verify the Ed25519 signature, and compare unsigned pack mirrors against the signed receipt.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <a href="https://github.com/SatGate-io/satgate/tree/main/tools" className="rounded-lg bg-white px-5 py-3 text-center font-bold text-black hover:bg-gray-200">Get verifier tool</a>
            <a href={livePackUrl} className="rounded-lg border border-cyan-300/40 px-5 py-3 text-center font-bold text-cyan-100 hover:border-cyan-200">Open sample production pack</a>
            <a href="/evidence/policy-to-proof-closure-20260718.json" className="rounded-lg border border-white/20 px-5 py-3 text-center font-bold text-gray-100 hover:border-white/40">Download sanitized closure</a>
          </div>
        </div>
      </section>

      <section className="border-b border-white/10 px-6 py-16">
        <div className="mx-auto max-w-5xl">
          <h2 className="text-3xl font-black">Read the result correctly.</h2>
          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <div className="rounded-2xl border border-amber-400/20 bg-amber-400/10 p-5 text-amber-50">
              <h3 className="font-bold"><code>valid=true</code></h3>
              <p className="mt-2 text-sm leading-6">With an embedded key, this proves only that the artifact is internally self-consistent. It does not establish who controls the issuer.</p>
            </div>
            <div className="rounded-2xl border border-emerald-400/20 bg-emerald-400/10 p-5 text-emerald-50">
              <h3 className="font-bold"><code>trusted_issuer_valid=true</code></h3>
              <p className="mt-2 text-sm leading-6">Buyer-verifiable proof requires the signature to validate against a separately fetched or pinned issuer JWKS, with trusted-issuer verification required.</p>
            </div>
          </div>
          <div className="mt-6 rounded-2xl border border-cyan-300/20 bg-cyan-300/[0.06] p-5 text-gray-200">
            <h3 className="font-bold text-white">Latest bounded closure record</h3>
            <p className="mt-2 text-sm leading-6">The July 18 record is sanitized, staging-only evidence. It records strict verifier and trusted-issuer success, verifier-copy parity, restart parity, and containment of historical staging bearer links. It does not authorize production promotion.</p>
            <p className="mt-3 break-all font-mono text-xs text-cyan-100">Source manifest SHA-256: 62d00ac4bff91e56fea8f5e8e42ceb0bb46461c46ba5d5a8c9645047baba4f5a</p>
            <p className="mt-2 break-all font-mono text-xs text-cyan-100">Public record SHA-256: 162f523d054feb99c2d65fadad7ecb3aa2d5127f1748160ca97424b73215eb7c</p>
          </div>
        </div>
      </section>

      <section className="px-6 py-16">
        <div className="mx-auto grid max-w-5xl gap-6 lg:grid-cols-2">
          <article className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
            <h2 className="text-2xl font-black">Clean-room verification</h2>
            <pre className="mt-5 overflow-x-auto rounded-xl bg-black p-4 text-sm text-gray-300"><code>{`python3 -m venv .venv-verify
. .venv-verify/bin/activate
pip install cryptography rfc8785
curl -fsS ${livePackUrl} -o pack.json
curl -fsS https://api.satgate.io/.well-known/jwks.json -o jwks.json
python tools/verify_evidence_pack.py pack.json \
  --jwks-file jwks.json \
  --require-trusted-issuer`}</code></pre>
          </article>

          <article className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
            <h2 className="text-2xl font-black">What the verifier checks</h2>
            <ul className="mt-5 space-y-3 text-gray-300">
              <li>• Receipt schema version and production/mock markers.</li>
              <li>• RFC 8785 canonical payload excluding <code>receipt_hash</code> and <code>signature</code>.</li>
              <li>• SHA-256 <code>receipt_hash</code> and Ed25519 signature.</li>
              <li>• Issuer JWKS at <code>/.well-known/jwks.json</code>; embedded public keys are fallback evidence, not issuer trust.</li>
              <li>• Top-level pack mirrors and budget-state mirrors match the signed receipt.</li>
              <li>• Optional <code>evidence_pack_hash</code> and secret redaction markers.</li>
            </ul>
          </article>
        </div>
      </section>

      <section className="border-t border-white/10 px-6 py-16">
        <div className="mx-auto max-w-4xl">
          <h2 className="text-3xl font-black">Current limits</h2>
          <ul className="mt-6 space-y-3 text-gray-300">
            <li>• Receipt signing keys are platform-secret-managed; this page does not claim KMS/HSM custody or external assessment.</li>
            <li>• The evidence archive is durable platform storage, not externally anchored or WORM-assured.</li>
            <li>• Evidence URLs are bearer-by-ID links. Anyone holding a URL can fetch that Pack until retention or targeted deletion removes it.</li>
            <li>• Verifier success establishes artifact integrity and issuer anchoring—not upstream behavior, settlement, or regulatory compliance.</li>
          </ul>
        </div>
      </section>

      <section className="border-t border-white/10 px-6 py-16">
        <div className="mx-auto max-w-4xl">
          <h2 className="text-3xl font-black">What this proves — and what it does not.</h2>
          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <div className="rounded-2xl border border-emerald-400/20 bg-emerald-400/10 p-5 text-emerald-50">
              <h3 className="font-bold">Proves</h3>
              <p className="mt-2 text-sm leading-6">The signed receipt was emitted by the issuer key identified by <code>issuer_kid</code>; signed-field tampering fails; the pack mirrors agree with the signed receipt; bearer/capability secrets are redacted.</p>
            </div>
            <div className="rounded-2xl border border-amber-400/20 bg-amber-400/10 p-5 text-amber-50">
              <h3 className="font-bold">Does not prove by itself</h3>
              <p className="mt-2 text-sm leading-6">Billing settlement, upstream counter reconciliation, Hybrid/MCP parity, instant revocation propagation, or broad production readiness unless those claims are separately evidenced.</p>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}

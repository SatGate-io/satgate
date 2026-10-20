import type { Metadata } from 'next';
import {readFileSync} from 'node:fs';
import path from 'node:path';
import ReceiptChecker from './ReceiptChecker';
import {HELD_RECEIPT_COPY} from './held-receipt-copy';

const title = 'Check a SatGate receipt';
const description = HELD_RECEIPT_COPY.lead;
export const metadata: Metadata = {
  title: {absolute:title}, description,
  alternates: {canonical:'https://satgate.io/verify-evidence-pack'},
  openGraph: {title, description, url:'https://satgate.io/verify-evidence-pack', images:[{url:'/og.png',width:1200,height:630}]},
  twitter: {card:'summary_large_image',title,description,images:['/og.png']},
};
const samplePackFile = '/evidence/mcp-budget-refusal-20261005.json';
const commands = `git clone https://github.com/SatGate-io/satgate.git
cd satgate
python3 -m venv .venv-verify
. .venv-verify/bin/activate
pip install cryptography rfc8785
curl -fsS https://satgate.io${samplePackFile} -o pack.json
curl -fsS https://satgate-mcp-saas.fly.dev/.well-known/jwks.json -o jwks.json
python tools/verify_evidence_pack.py pack.json \\
  --jwks-file jwks.json \\
  --require-trusted-issuer`;
export default function VerifyEvidencePackPage() {
  const build = JSON.parse(readFileSync(path.join(process.cwd(),'public/receipt-checker/build.json'),'utf8'));
  return <main className="min-h-screen bg-black px-6 py-16 text-white sm:py-24">
    <div className="mx-auto max-w-3xl space-y-12 text-lg leading-8 text-gray-300">
      <header>
        <h1 className="text-4xl font-black leading-tight tracking-tight text-white sm:text-6xl">Check a SatGate receipt</h1>
        <p className="mt-6 text-xl">{HELD_RECEIPT_COPY.lead}</p>
      </header>
      <section>
        <h2 className="mb-4 text-2xl font-bold text-white">Why it matters</h2>
        <p>A receipt records what your agent asked for and what SatGate decided. Your agent never writes or edits its own record. Because anyone can check a receipt, you can hand one to an accountant, an auditor or a customer, and they can confirm it for themselves.</p>
      </section>
      <section>
        <h2 className="mb-4 text-2xl font-bold text-white">Check one in three steps</h2>
        <ol className="list-decimal space-y-3 pl-6">
          <li>{HELD_RECEIPT_COPY.stepOne}</li>
          <li>Drop the file on the box below.</li>
          <li>Read the result. Valid means SatGate signed this receipt and nothing in it has changed since. Not valid means don&apos;t rely on it.</li>
        </ol>
        <ReceiptChecker />
        <p className="mt-3 text-sm text-gray-400">One-time checker download: {(build.bytes / 1000000).toFixed(2)} MB ({build.bytes.toLocaleString('en-US')} bytes, before transfer compression).</p>
      </section>
      <section>
        <h2 className="mb-4 text-2xl font-bold text-white">{HELD_RECEIPT_COPY.coverageHeading}</h2>
        {HELD_RECEIPT_COPY.coverage.map((text,index)=><p key={text} className={index ? 'mt-4' : ''}>{text}</p>)}
      </section>
      <section>
        <h2 className="mb-4 text-2xl font-bold text-white">What a valid receipt doesn&apos;t show</h2>
        <p>It shows what SatGate decided. It doesn&apos;t show what happened after SatGate let a request through, for example whether a broker filled an order or whether a payment settled.</p>
      </section>
      <section id="engineers" className="min-w-0 border-t border-white/10 pt-12 text-base leading-7">
        <h2 className="mb-4 text-2xl font-bold text-white">For engineers: check it on your own computer</h2>
        <p>The checker on this page runs the open-source verifier, tools/verify_evidence_pack.py from github.com/SatGate-io/satgate, inside your browser. The file it runs has the SHA-256 shown here, the same as that file on GitHub: <code className="break-all text-cyan-200">{build.verifiers['verify_evidence_pack.py'].sha256}</code>.</p>
        <p className="mt-4 break-all">tools/verify_receipt_record.py SHA-256: <code className="text-cyan-200">{build.verifiers['verify_receipt_record.py'].sha256}</code></p>
        <p className="mt-6">To check without using anything from this site, run the verifier yourself:</p>
        <pre className="mt-4 max-w-full overflow-x-auto rounded-xl border border-white/10 bg-white/5 p-5 text-sm leading-6"><code>{commands}</code></pre>
        <p className="mt-4 break-words">Cloud gateway keys: <a href="https://api.satgate.io/.well-known/jwks.json" className="break-all text-cyan-200 underline">https://api.satgate.io/.well-known/jwks.json</a></p>
        <p className="mt-2 break-words">Cloud MCP keys: <a href="https://satgate-mcp-saas.fly.dev/.well-known/jwks.json" className="break-all text-cyan-200 underline">https://satgate-mcp-saas.fly.dev/.well-known/jwks.json</a></p>
        <p className="mt-6">Two results matter. valid=true means the receipt matches the key embedded in it, which alone doesn&apos;t say who signed it. trusted_issuer_valid=true means it matches SatGate&apos;s published key, fetched separately. The second one is the check that counts, and it&apos;s the only one this page calls Valid.</p>
        <p className="mt-4">A self-hosted SatGate gateway signs with its own key. Check its receipts against that gateway&apos;s published keys.</p>
        <h3 className="mb-4 mt-8 text-xl font-bold text-white">Limits</h3>
        <ul className="list-disc space-y-3 pl-6">
          <li>Anyone with a pack&apos;s link can open that pack until it&apos;s deleted, so share links with care.</li>
          <li>Our signing keys are kept in our hosting provider&apos;s secret storage, not in a hardware security module, and they haven&apos;t had an outside security assessment.</li>
          <li>Receipts are stored in SatGate&apos;s own storage. Nothing outside SatGate anchors them, so keep your own copy of any receipt you rely on.</li>
          <li>A valid check shows the receipt is intact and was signed with SatGate&apos;s key. It doesn&apos;t show what an outside service did, whether money settled, or that you meet any regulation.</li>
        </ul>
      </section>
    </div>
  </main>;
}

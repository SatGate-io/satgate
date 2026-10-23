'use client';

import {useEffect, useRef, useState} from 'react';

// Sentence-only snapshot of dashboard/app/lib/evidence-verify-words.ts.
// The CLI's check names are mapped below; never render its untrusted values.
const words = {
  receipt: "The pack doesn't carry a signed SatGate receipt that SatGate can read.",
  issuer: 'The pack or its receipt points to an issuer, key set or pack address other than the trusted SatGate issuer.',
  receipt_hash: "The receipt hash doesn't match the receipt's contents.",
  signature: "The receipt's signature is missing or doesn't match the issuer's published key.",
  unknown_key: "The receipt was signed with a key that the issuer's published keys don't list.",
  pack_receipt: "The pack doesn't match its signed receipt.",
  archive_hash: "The pack's archive details or its own hash don't match the pack's contents.",
  fallback: "SatGate could not match this pack's signature to its issuer's published key, or the pack did not match its signed receipt.",
};
const couldntRun = "SatGate couldn't run the check, so this pack wasn't checked. Try again later.";
type Answer = {kind:string; checks?:Record<string,boolean>};
export function failureSentence(checks:Record<string,boolean>):string {
  const failed = Object.keys(checks).filter(k=>checks[k] === false);
  // The dashboard's age sentence is limited to its 90-day history view,
  // not the public verifier's clock-skew/expiry rules. New draft words.
  if (failed.some(k=>k.endsWith('time_valid'))) return "The receipt's dates do not pass the check.";
  if (failed.some(k=>k.endsWith('receipt_hash_match') || k === 'receipt_hash_match')) return words.receipt_hash;
  if (failed.some(k=>k.endsWith('signature_valid') || k === 'manifest_signature')) return words.signature;
  if (failed.some(k=>k === 'evidence_pack_hash_match')) return words.archive_hash;
  if (failed.some(k=>k.startsWith('pack_') || k === 'budget_state_matches_primary_receipt')) return words.pack_receipt;
  if (failed.includes('jwks_issuer_match')) return words.issuer;
  if (failed.some(k=>k.endsWith('trusted_issuer_valid'))) return words.unknown_key;
  if (failed.some(k=>k === 'embedded_receipts_present' || k === 'input_valid' || k.includes('schema_version'))) return words.receipt;
  if (failed.some(k=>['manifest_present','manifest_fields','count_matches','hashes_match','order_and_range','no_duplicates','one_key','windows','counts_by_reason'].includes(k))) return 'This receipt record does not pass all its checks.';
  return words.fallback;
}
function message(answer:Answer):string {
  switch (answer.kind) {
    case 'valid': return 'Valid';
    case 'invalid': return failureSentence(answer.checks || {});
    case 'keys-network': return "SatGate couldn't reach the issuer's published signing keys, so this pack wasn't checked. Try again later.";
    case 'keys-read': return "SatGate couldn't read the issuer's published signing keys, so this pack wasn't checked. Try again later.";
    case 'issuer': return "This checker can only check SatGate Cloud receipts. For a self-hosted gateway, see the engineer section below.";
    case 'empty': return 'Choose a receipt file that is not empty.';
    case 'large': return 'Choose a receipt file smaller than 10 MiB.';
    case 'json': return 'This file is not a receipt in JSON or a receipt record. Choose the file you downloaded from SatGate.';
    default: return couldntRun;
  }
}
export default function ReceiptChecker() {
  const [answer,setAnswer] = useState<Answer|null>(null);
  const [busy,setBusy] = useState(false);
  const worker = useRef<Worker|null>(null);
  const timer = useRef<ReturnType<typeof setTimeout>|null>(null);
  const input = useRef<HTMLInputElement|null>(null);
  function stop() {
    worker.current?.terminate(); worker.current=null;
    if (timer.current) clearTimeout(timer.current);
    timer.current=null;
  }
  useEffect(()=>()=>stop(),[]);
  function check(file?:File) {
    if (!file) return;
    stop(); setAnswer(null); setBusy(true);
    const finish = (value:Answer) => { stop(); setBusy(false); setAnswer(value); };
    try {
      const w = new Worker('/receipt-checker-worker.mjs',{type:'module'});
      worker.current=w;
      w.onmessage = e=>finish(e.data);
      w.onerror = ()=>finish({kind:'runtime'});
      timer.current=setTimeout(()=>finish({kind:'runtime'}),120000);
      w.postMessage(file);
    } catch { finish({kind:'runtime'}); }
  }
  return <div className="mt-8">
    <div className="rounded-2xl border-2 border-dashed border-cyan-300/50 bg-cyan-300/5 p-8 text-center focus-within:border-cyan-200" onDragOver={e=>e.preventDefault()} onDrop={e=>{e.preventDefault();check(e.dataTransfer.files[0]);}}>
      <label htmlFor="receipt-file" className="block cursor-pointer text-lg font-semibold text-cyan-100">Drop a receipt file here, or choose one</label>
      <input ref={input} id="receipt-file" type="file" accept=".json,.ndjson,.jsonl,application/json,application/x-ndjson" className="sr-only" onChange={e=>{check(e.target.files?.[0]);e.target.value='';}} />
    </div>
    <p className="mt-4 text-sm leading-6 text-gray-400">Your file stays on your computer. The check runs in your browser and nothing is uploaded.</p>
    <div role="status" aria-live="polite" aria-busy={busy} data-testid="verdict" className="mt-5 break-words rounded-xl border border-white/10 p-5 empty:hidden">
      {busy ? 'Checking your receipt…' : answer ? <>
        {answer.kind === 'invalid' && <p className="mb-2 font-bold text-amber-200">Not valid</p>}
        <p className={answer.kind==='valid' ? 'text-xl font-bold text-emerald-300' : 'text-gray-200'}>{message(answer)}</p>
        {answer.kind==='issuer' && <a href="#engineers" className="mt-2 inline-block text-cyan-200 underline">For engineers: check it on your own computer</a>}
      </> : null}
    </div>
  </div>;
}

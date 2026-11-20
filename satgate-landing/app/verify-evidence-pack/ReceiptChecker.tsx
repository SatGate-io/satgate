'use client';

import {useEffect, useRef, useState} from 'react';

// Ordered, exhaustive names from tools/verify_evidence_pack.py. Never show
// file values, Python diagnostics, summaries or untrusted strings in the UI.
export const checkGroups = [
  ['schema_version', 'input_valid', 'embedded_receipts_present', 'receipt_*_schema_version',
    'receipt_*_canonicalization', 'receipt_*_hash_algorithm', 'receipt_*_signature_algorithm'],
  ['jwks_issuer_match'],
  ['receipt_hash_match', 'receipt_*_receipt_hash_match', 'evidence_pack_hash_match',
    'pack_hash_matches_primary_receipt', 'pack_*_matches_primary_receipt',
    'pack_optional_*_matches_primary_receipt', 'budget_state_matches_primary_receipt'],
  ['signature_valid', 'trusted_issuer_valid', 'receipt_*_signature_valid', 'receipt_*_trusted_issuer_valid'],
  ['production_artifact', 'receipt_*_production_artifact', 'time_valid', 'receipt_*_time_valid',
    'receipt_*_decision', 'receipt_*_decision_reason', 'receipt_*_paid_rail',
    'receipt_*_unavailable_check_refusal', 'receipt_*_allowed_not_evaluated',
    'receipt_*_rate_limit_window', 'receipt_*_observe_unrecorded', 'receipt_*_new_profile_provenance',
    'pack_paid_rail', 'pack_no_forbidden_claims', 'no_secret_leakage'],
] as const;
const reasons = [
  'this file is not a SatGate receipt',
  'it was not signed by a SatGate address',
  'it was changed after it was signed',
  'the signature does not match',
  'this file is not a SatGate receipt',
];
type Answer = {kind:string; checks?:Record<string,boolean>};
function matches(name:string, pattern:string) {
  return new RegExp('^' + pattern.split('*').join('[a-zA-Z0-9_]+') + '$').test(name);
}
export function failureSentence(checks:Record<string,boolean>):string {
  const failed = Object.keys(checks).filter(k=>checks[k] === false);
  const group = checkGroups.findIndex(names=>failed.some(k=>names.some(n=>matches(k,n))));
  return `This receipt did not pass: ${reasons[group < 0 ? 0 : group]}.`;
}
export function message(answer:Answer):string {
  switch (answer.kind) {
    case 'valid': return 'Signed by SatGate, and nothing in it has changed since.';
    case 'invalid': return failureSentence(answer.checks || {});
    case 'keys-network':
    case 'keys-read': return "We couldn't load SatGate's published keys. Check your connection and try again.";
    case 'issuer': return 'This receipt did not pass: it was not signed by a SatGate address.';
    case 'empty':
    case 'large':
    case 'json': return 'This receipt did not pass: this file is not a SatGate receipt.';
    default: return 'Check your connection and try again.';
  }
}
export default function ReceiptChecker() {
  const [answer,setAnswer] = useState<Answer|null>(null);
  const [busy,setBusy] = useState(false);
  const worker = useRef<Worker|null>(null);
  const timer = useRef<ReturnType<typeof setTimeout>|null>(null);
  function stop() {
    worker.current?.terminate(); worker.current=null;
    if (timer.current) clearTimeout(timer.current);
    timer.current=null;
  }
  useEffect(()=>()=>stop(),[]);
  function check(file:File|{sample:true}) {
    stop(); setAnswer(null); setBusy(true);
    const finish = (value:Answer) => { stop(); setBusy(false); setAnswer(value); };
    try {
      const w = new Worker('/receipt-checker-worker.mjs',{type:'module',credentials:'omit'});
      worker.current=w;
      w.onmessage = e=>finish(e.data);
      w.onerror = e=>{e.preventDefault();finish({kind:'runtime'});};
      timer.current=setTimeout(()=>finish({kind:'runtime'}),120000);
      w.postMessage(file);
    } catch { finish({kind:'runtime'}); }
  }
  return <div className="mt-8">
    <label htmlFor="receipt-file" className="inline-block cursor-pointer rounded-lg bg-white px-5 py-3 font-bold text-black focus-within:outline-2">Choose a receipt file
      <input id="receipt-file" type="file" accept=".json,.ndjson,.jsonl,application/json,application/x-ndjson" className="sr-only" onChange={e=>{if(e.target.files?.[0]) check(e.target.files[0]);e.target.value='';}} />
    </label>
    <p className="mt-4 text-gray-300">No receipt yet? <button type="button" className="cursor-pointer text-cyan-200 underline underline-offset-4" onClick={()=>check({sample:true})}>Try a sample.</button></p>
    <div role="status" aria-live="polite" aria-busy={busy} data-testid="verdict" className="mt-5 break-words rounded-xl border border-white/10 p-5 empty:hidden">
      {busy ? <span aria-hidden="true" className="inline-block h-6 w-6 animate-spin rounded-full border-2 border-gray-500 border-t-cyan-200"/> : answer ?
        answer.kind === 'record' ? <p>This page checks one receipt at a time. To check a full record, use <a href="https://github.com/SatGate-io/satgate/tree/main/tools" className="text-cyan-200 underline underline-offset-4">our open-source checker</a>.</p> :
        <p className={answer.kind==='valid' ? 'text-xl font-bold text-emerald-300' : 'text-gray-200'}>{message(answer)}</p> : null}
    </div>
    <p className="mt-4 text-sm leading-6 text-gray-400">A pass shows that SatGate signed this receipt and that nothing in it has changed since. It does not show what your broker filled. Your broker&apos;s statement shows that.</p>
    <p className="mt-4 text-sm leading-6 text-gray-400">Prefer not to rely on this page? Run the same check on your own computer with <a href="https://github.com/SatGate-io/satgate/tree/main/tools" className="text-cyan-200 underline underline-offset-4">our open-source checker</a>.</p>
  </div>;
}

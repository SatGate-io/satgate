// No verifier logic lives here. The public Python CLIs run unchanged in MEMFS.
// Issuer routing is deliberately exact: never follow a URL supplied by a file.
const KEY_URLS = Object.freeze({
  'https://api.satgate.io': 'https://api.satgate.io/.well-known/jwks.json',
  'https://satgate-mcp-saas.fly.dev': 'https://satgate-mcp-saas.fly.dev/.well-known/jwks.json',
});
const MAX_BYTES = 10 * 1024 * 1024;
const base = '/receipt-checker/';
self.onmessage = async ({data: file}) => {
  let py;
  try {
    if (!file.size) return postMessage({kind:'empty'});
    if (file.size > MAX_BYTES) return postMessage({kind:'large'});
    const inputBytes = new Uint8Array(await file.arrayBuffer());
    let text;
    try { text = new TextDecoder('utf-8', {fatal:true}).decode(inputBytes); }
    catch { return postMessage({kind:'json'}); }
    let objects, record = false;
    try { objects = [JSON.parse(text)]; }
    catch {
      try { objects = text.split(/\r?\n/).filter(s=>s.trim()).map(s=>JSON.parse(s)); record = true; }
      catch { return postMessage({kind:'json'}); }
    }
    if (!objects.length || objects.some(o=>!o || Array.isArray(o) || typeof o !== 'object')) return postMessage({kind:'json'});
    if (objects[0].schema_version === 'satgate.receipt_record_manifest.v1') record = true;
    const issuers = objects.flatMap(o=>[o.issuer, ...(Array.isArray(o.receipts) ? o.receipts.map(r=>r?.issuer) : [])]);
    if (!issuers.length || issuers.some(i=>!Object.hasOwn(KEY_URLS,i)) || new Set(issuers).size !== 1) return postMessage({kind:'issuer'});
    let keys;
    try {
      const response = await fetch(KEY_URLS[issuers[0]], {credentials:'omit', redirect:'error', cache:'no-store', referrerPolicy:'no-referrer', signal:AbortSignal.timeout(10000)});
      if (!response.ok) return postMessage({kind:'keys-network'});
      // Bound the key response before parsing or giving it to Python.
      const reader = response.body.getReader();
      const chunks = []; let size = 0;
      while (true) {
        const {done,value} = await reader.read(); if (done) break;
        size += value.length; if (size > 65536) { await reader.cancel(); return postMessage({kind:'keys-read'}); }
        chunks.push(value);
      }
      keys = await new Blob(chunks).text();
      let parsed;
      try { parsed = JSON.parse(keys); }
      catch { return postMessage({kind:'keys-read'}); }
      if (!parsed || !Array.isArray(parsed.keys) || !parsed.keys.length) return postMessage({kind:'keys-read'});
    } catch { return postMessage({kind:'keys-network'}); }
    const {loadPyodide} = await import('/receipt-checker/pyodide.mjs');
    py = await loadPyodide({indexURL: base, stdout:()=>{}, stderr:()=>{}});
    await py.loadPackage('cryptography');
    py.unpackArchive(await (await fetch(base+'rfc8785-0.1.4-py3-none-any.whl')).arrayBuffer(), 'zip', {extractDir:'/lib/python3.14/site-packages'});
    for (const name of ['verify_evidence_pack.py','verify_receipt_record.py']) {
      const response = await fetch(base+name);
      if (!response.ok) throw new Error('Verifier unavailable');
      py.FS.writeFile('/'+name, new Uint8Array(await response.arrayBuffer()));
    }
    py.FS.writeFile('/input', inputBytes);
    py.FS.writeFile('/keys', keys);
    py.globals.set('entry', record ? 'verify_receipt_record' : 'verify_evidence_pack');
    const output = JSON.parse(await py.runPythonAsync(`
import contextlib, importlib, io, json, sys
sys.path.insert(0, '/')
module = importlib.import_module(entry)
stdout, stderr = io.StringIO(), io.StringIO()
with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
    exit_code = module.main(['/input', '--jwks-file', '/keys', '--require-trusted-issuer'])
result = json.loads(stdout.getvalue())
if entry == 'verify_receipt_record' and not result.get('valid'):
    # The record CLI reports receipt_invalid, not each pack's check names.
    # Ask the same public pack verifier for those names to choose the live
    # dashboard sentence. This never changes the record CLI's verdict.
    from pathlib import Path
    keys = json.loads(Path('/keys').read_text())
    for raw in Path('/input').read_text().splitlines():
        if not raw.strip(): continue
        try: candidate = json.loads(raw)
        except ValueError: continue
        if isinstance(candidate, dict) and candidate.get('schema_version') == module.PACK_SCHEMA:
            detail = module.vep.verify_pack(candidate, jwks=keys, require_trusted_issuer=True)
            for name, ok in detail.get('checks', {}).items():
                if not ok: result.setdefault('checks', {})[name] = False
json.dumps({'exit': exit_code, 'result': result})
`));
    const result = output.result;
    const trusted = record ? result.checks?.manifest_trusted_issuer === true : result.trusted_issuer_valid === true;
    // Send only check names and booleans, never file values or verifier summaries.
    postMessage({kind: output.exit === 0 && result.valid === true && trusted ? 'valid' : 'invalid', checks:result.checks || {}});
  } catch { postMessage({kind:'runtime'}); }
  finally {
    if (py) { for (const name of ['/input','/keys']) { try { py.FS.unlink(name); } catch { /* not created */ } } }
    // Main thread also terminates the worker. No IDBFS, storage, or service worker.
    self.close();
  }
};

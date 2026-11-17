// No verifier logic lives here. The public pack CLI runs unchanged in MEMFS.
// Pyodide's own static downloads must also omit credentials and referrers.
const nativeFetch = self.fetch.bind(self);
self.fetch = (url, options = {}) => nativeFetch(url, {...options, method:'GET',
  body:undefined, credentials:'omit', redirect:'error', cache:'no-store', referrerPolicy:'no-referrer'});
// Issuer routing is deliberately exact: never follow a URL supplied by a file.
const KEY_URLS = Object.freeze({
  'https://api.satgate.io': 'https://api.satgate.io/.well-known/jwks.json',
  'https://satgate-mcp-saas.fly.dev': 'https://satgate-mcp-saas.fly.dev/.well-known/jwks.json',
  'https://mcp.satgate.io': 'https://mcp.satgate.io/.well-known/jwks.json',
});
const MAX_BYTES = 10 * 1024 * 1024;
const base = '/receipt-checker/';
// JSON.parse accepts duplicate object members. A key publication with two
// meanings is an availability failure, even if its last value looks usable.
function uniqueMembers(text) {
  const stack = [];
  const tokens = /"(?:\\.|[^"\\])*"|[{}[\]]/g;
  for (const match of text.matchAll(tokens)) {
    const token = match[0];
    if (token === '{') stack.push(new Set());
    else if (token === '[') stack.push(null);
    else if (token === '}' || token === ']') stack.pop();
    else if (/^\s*:/.test(text.slice(match.index + token.length))) {
      const names = stack[stack.length - 1], name = JSON.parse(token);
      if (!names || names.has(name)) return false;
      names.add(name);
    }
  }
  return true;
}
// This is a key-publication preflight, not receipt verification. A broken or
// ambiguous publication means could-not-check, not a rejected signature.
async function usableKeys(jwks) {
  if (!jwks || !Array.isArray(jwks.keys) || !jwks.keys.length) return false;
  const ids = new Set();
  for (const key of jwks.keys) {
    if (!key || typeof key !== 'object' || Array.isArray(key) ||
        typeof key.kid !== 'string' || !key.kid || ids.has(key.kid) ||
        key.kty !== 'OKP' || key.crv !== 'Ed25519' ||
        (key.alg != null && key.alg !== 'EdDSA') ||
        (key.use != null && key.use !== 'sig') ||
        typeof key.x !== 'string' || !/^[A-Za-z0-9_-]{43}$/.test(key.x)) return false;
    ids.add(key.kid);
    try {
      const raw = Uint8Array.from(atob(key.x.replace(/-/g,'+').replace(/_/g,'/')+'='), c=>c.charCodeAt(0));
      // Reject noncanonical base64url (including nonzero unused bits).
      if (raw.length !== 32 || btoa(String.fromCharCode(...raw)).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,'') !== key.x) return false;
      await crypto.subtle.importKey('raw', raw, {name:'Ed25519'}, false, ['verify']);
    } catch { return false; }
  }
  return true;
}
self.onmessage = async ({data}) => {
  let py;
  try {
    let file = data;
    if (data?.sample === true) {
      const response = await fetch('/evidence/mcp-budget-refusal-20261005.json');
      if (response.status !== 200) throw new Error('Sample unavailable');
      file = await response.blob();
    }
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
    if (record && objects.length) return postMessage({kind:'record'});
    if (!objects.length || objects.some(o=>!o || Array.isArray(o) || typeof o !== 'object')) return postMessage({kind:'json'});
    if (['satgate.receipt_record_manifest.v1', 'satgate.receipt_record_truncated.v1'].includes(objects[0].schema_version)) record = true;
    if (record) return postMessage({kind:'record'});
    const pack = objects[0];
    if (pack.schema_version !== 'satgate.evidence_pack.v1' ||
        !Array.isArray(pack.receipts) || !pack.receipts.length ||
        pack.receipts.some(r=>!r || typeof r !== 'object' || Array.isArray(r) ||
          r.schema_version !== 'satgate.receipt.v1' || r.canonicalization !== 'jcs-rfc8785' ||
          r.hash_algorithm !== 'sha256' || r.signature_algorithm !== 'ed25519')) return postMessage({kind:'json'});
    const issuers = objects.flatMap(o=>[o.issuer, ...(Array.isArray(o.receipts) ? o.receipts.map(r=>r?.issuer) : [])]);
    if (!issuers.length || issuers.some(i=>typeof i !== 'string' || !Object.hasOwn(KEY_URLS,i)) || new Set(issuers).size !== 1) return postMessage({kind:'issuer'});
    let now = Date.now();
    try {
      const clock = await fetch(base+'build.json', {signal:AbortSignal.timeout(10000)});
      const server = Date.parse(clock.headers.get('date'));
      if (clock.status === 200 && Number.isFinite(server)) now = Math.max(now, server);
    } catch { /* Offline clock request: use the device clock alone. */ }
    let keys;
    try {
      const response = await fetch(KEY_URLS[issuers[0]], {credentials:'omit', redirect:'error', cache:'no-store', referrerPolicy:'no-referrer', signal:AbortSignal.timeout(10000)});
      if (response.status !== 200 || response.redirected) return postMessage({kind:'keys-network'});
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
      if (!uniqueMembers(keys) || !await usableKeys(parsed)) return postMessage({kind:'keys-read'});
    } catch { return postMessage({kind:'keys-network'}); }
    const {loadPyodide} = await import('/receipt-checker/pyodide.mjs');
    py = await loadPyodide({indexURL: base, stdout:()=>{}, stderr:()=>{}});
    await py.loadPackage('cryptography');
    py.unpackArchive(await (await fetch(base+'rfc8785-0.1.4-py3-none-any.whl')).arrayBuffer(), 'zip', {extractDir:'/lib/python3.14/site-packages'});
    for (const name of ['verify_evidence_pack.py']) {
      const response = await fetch(base+name);
      if (!response.ok) throw new Error('Verifier unavailable');
      py.FS.writeFile('/'+name, new Uint8Array(await response.arrayBuffer()));
    }
    py.FS.writeFile('/input', inputBytes);
    py.FS.writeFile('/keys', keys);
    py.globals.set('check_now', new Date(now).toISOString());
    const output = JSON.parse(await py.runPythonAsync(`
import contextlib, importlib, io, json, sys
sys.path.insert(0, '/')
module = importlib.import_module('verify_evidence_pack')
stdout, stderr = io.StringIO(), io.StringIO()
with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
    exit_code = module.main(['/input', '--jwks-file', '/keys', '--require-trusted-issuer', '--now', check_now])
result = json.loads(stdout.getvalue())
json.dumps({'exit': exit_code, 'result': result})
`));
    const result = output.result;
    const trusted = result.trusted_issuer_valid === true;
    // Send only check names and booleans, never file values or verifier summaries.
    postMessage({kind: output.exit === 0 && result.valid === true && trusted ? 'valid' : 'invalid', checks:result.checks || {}});
  } catch { postMessage({kind:'runtime'}); }
  finally {
    if (py) { for (const name of ['/input','/keys']) { try { py.FS.unlink(name); } catch { /* not created */ } } }
    // Main thread also terminates the worker. No IDBFS, storage, or service worker.
    self.close();
  }
};

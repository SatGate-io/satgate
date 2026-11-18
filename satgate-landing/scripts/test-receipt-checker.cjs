/* eslint-disable @typescript-eslint/no-require-imports -- Test-only CJS. */
// Local production browser tests. Intercept every external request before it can
// leave this machine. Fixtures, corpus replay and CSP mutations are test-side only.
const {chromium} = require('playwright');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const lane=process.argv[2], origin=process.argv[3] || 'http://127.0.0.1:3928';
assert.equal(new URL(origin).hostname,'127.0.0.1');
const read=name=>fs.readFileSync(path.join(lane,name));
const root=path.resolve(__dirname,'..');
const copy=require('./receipt-checker-copy.json');
const pass='Signed by SatGate, and nothing in it has changed since.';
const fail=reason=>'This receipt did not pass: '+reason+'.';
const malformed=fail('this file is not a SatGate receipt');
const issuer=fail('it was not signed by a SatGate address');
const changed=fail('it was changed after it was signed');
const signature=fail('the signature does not match');
const keys="We couldn't load SatGate's published keys. Check your connection and try again.";
const runtime='Check your connection and try again.';
const record='This page checks one receipt at a time. To check a full record, use our open-source checker.';
const keyURLs=['https://api.satgate.io/.well-known/jwks.json','https://satgate-mcp-saas.fly.dev/.well-known/jwks.json','https://mcp.satgate.io/.well-known/jwks.json'];
const siteIcons=['/icon-32.png','/icon-192.png','/apple-touch-icon.png','/favicon.ico'];
const normal=s=>s.replace(/\s+/g,' ').trim();
const cases=JSON.parse(read('corpus/ui-cases.json'));
cases.push({name:'invalid-utf8',bytes:[255],expected:'malformed'});
const expectedByName={
  'pack-tampered':changed,'pack-other-key':signature,'pack-embedded-only':signature,
  'pack-expired':malformed,'self-hosted':issuer,'multiple-issuers':issuer,
  'file-key-url-ignored':changed,'signature-only':signature,'mock-signed':malformed,
  'unknown-decision':malformed,'future-signed':malformed,'secret':malformed,
  'unknown-reason':malformed,'test-signed':malformed,'keys-duplicate-member':keys,
  'pack-copy':changed,'pack-budget':changed,'pack-hash':changed,
  'keys-blocked':keys,'keys-malformed':keys,'keys-status':keys,'keys-redirect':keys,'keys-oversize':keys,
  'runtime-load':runtime,'runtime-crash':runtime,'runtime-timeout':runtime,
  'slow-clock':pass,'clock-offline':pass,
};
function expected(item) {
  if(item.name.startsWith('record-') || item.expected==='record') return record;
  if(item.name.includes('-keys-')) return keys;
  if(item.expected==='malformed' || ['not-json','empty','large','invalid-utf8'].includes(item.name)) return malformed;
  return expectedByName[item.name] || (item.expected==='Valid' ? pass : assert.fail('Unmapped case: '+item.name));
}
function allowedProductRequest(r) {
  assert.equal(r.bytes,0,'upload');assert.equal(r.method,'GET');
  const u=new URL(r.url);
  return keyURLs.includes(r.url) || (u.origin===origin && !u.search &&
    (u.pathname==='/verify-evidence-pack' || u.pathname.startsWith('/_next/static/') ||
     u.pathname.startsWith('/receipt-checker/') || u.pathname==='/receipt-checker-worker.mjs' ||
     u.pathname==='/evidence/mcp-budget-refusal-20261005.json' || siteIcons.includes(u.pathname)));
}
const results=[];
const layoutOnly=process.argv.includes('--layout-only');
const command=copy.after[copy.after.indexOf('Run the verifier')+1];
const layout=[];
async function measureLayout(page,width,state) {
  await page.evaluate(()=>document.fonts.ready);
  const bounds=await page.evaluate(()=>({
    innerWidth:window.innerWidth,
    scrollWidth:document.documentElement.scrollWidth,
    outside:[...document.querySelectorAll('*')].flatMap(el=>{
      const {left,right}=el.getBoundingClientRect();
      return left<0 || right>window.innerWidth ? [{tag:el.localName,left,right}] : [];
    }),
  }));
  const block=page.locator('main pre code');
  const {text,selected,clientWidth,scrollWidth,right}=await block.evaluate(code=>{
    const range=document.createRange();
    range.selectNodeContents(code);
    const selection=window.getSelection();
    selection.removeAllRanges();
    selection.addRange(range);
    const selected=selection.toString();
    selection.removeAllRanges();
    const pre=code.parentElement;
    return {text:code.textContent,selected,clientWidth:pre.clientWidth,scrollWidth:pre.scrollWidth,right:code.getBoundingClientRect().right};
  });
  assert.equal(text,command,`${width} ${state}: exact command bytes`);
  assert.equal(selected,command,`${width} ${state}: exact selection text`);
  layout.push({width,state,...bounds,command:{right,clientWidth,scrollWidth,sha256:crypto.createHash('sha256').update(text).digest('hex')}});
}
let browser;
(async()=>{
  browser=await chromium.launch();
  // The icon exception must not permit uploads, off-origin icons or file queries.
  for(const icon of siteIcons) assert.ok(allowedProductRequest({url:origin+icon,method:'GET',bytes:0}));
  for(const url of [origin+'/icon-32.png?receipt=private',origin+'/private-receipt.png','https://attacker.example/icon-32.png']) assert.equal(allowedProductRequest({url,method:'GET',bytes:0}),false);
  assert.throws(()=>allowedProductRequest({url:origin+'/icon-32.png',method:'POST',bytes:0}));
  assert.throws(()=>allowedProductRequest({url:origin+'/icon-32.png',method:'GET',bytes:1}));
  results.push({name:'static-icon-allowlist-controls',exit:0});
  async function scenario(item,{sample=false,shot=null,width=1440,cspMutation=null,layoutState=null}={}) {
    const ctx=await browser.newContext({viewport:{width,height:1000},serviceWorkers:'block'});
    const requests=[],responses=[],errors=[],requestRows=new Map();
    ctx.on('request',r=>{
      const row={url:r.url(),method:r.method(),bytes:r.postDataBuffer()?.length||0,headers:r.headers()};
      requests.push(row);requestRows.set(r,row);
    });
    await ctx.addInitScript(({slow,timeout})=>{
      if(slow) { const RealDate=Date; window.Date=class extends RealDate {constructor(...args){super(...(args.length?args:[RealDate.now()-3600000]));}static now(){return RealDate.now()-3600000;}}; }
      if(timeout) { const timer=window.setTimeout.bind(window); window.setTimeout=(fn,ms,...args)=>timer(fn,ms===120000?100:ms,...args); }
      Storage.prototype.setItem=()=>{throw Error('Storage is forbidden');};
      window.indexedDB.open=()=>{throw Error('Storage is forbidden');};
      navigator.serviceWorker.register=()=>{throw Error('Service workers are forbidden');};
    },{slow:item.name.includes('slow-clock'),timeout:item.name==='runtime-timeout'});
    await ctx.route('**/*',async route=>{
      const req=route.request(),url=req.url();
      // Read complete headers while the request is paused and its worker is
      // alive. Asking a terminated worker for headers can hang Playwright.
      const row=requestRows.get(req);
      assert.ok(row,'request event missing');
      row.headers=cspMutation ? req.headers() : await req.allHeaders();row.intercepted=true;
      if(keyURLs.includes(url)) {
        if(item.name==='keys-blocked') return route.abort('failed');

        const status=item.name==='keys-status'?503:item.name==='keys-redirect'?302:200;
        const body=item.name==='keys-malformed'?'not JSON':item.name==='keys-oversize'?' '.repeat(65537):item.keyBytes||JSON.stringify(item.keys);
        responses.push({url,bytes:Buffer.byteLength(body)});
        return route.fulfill({status,headers:{'content-type':'application/jwk-set+json','access-control-allow-origin':'*',...(status===302?{location:url+'/redirect'}:{})},body});
      }
      if(!url.startsWith(origin+'/')) {await route.abort();assert.fail('External request: '+url);}
      if(item.name==='clock-offline' && url===origin+'/receipt-checker/build.json') return route.abort();
      if(item.name==='runtime-load' && url===origin+'/receipt-checker/pyodide.mjs') return route.abort();
      if(item.name.includes('slow-clock') && url===origin+'/receipt-checker-worker.mjs') {
        const response=await route.fetch();
        let body=await response.text();
        if(item.name==='recent-slow-clock-without-server') body=body.replace('now = Math.max(now, server);','now = now;');
        return route.fulfill({response,body:'const RealDate=Date;self.Date=class extends RealDate {constructor(...args){super(...(args.length?args:[RealDate.now()-3600000]));}static now(){return RealDate.now()-3600000;}};\n'+body});
      }
      if(['runtime-crash','runtime-timeout'].includes(item.name) && url===origin+'/receipt-checker-worker.mjs') {
        const response=await route.fetch();return route.fulfill({response,body:item.name==='runtime-crash'?'throw Error("test crash")':'self.onmessage=()=>{}'});
      }
      if(cspMutation && [origin+'/verify-evidence-pack',origin+'/receipt-checker-worker.mjs'].includes(url)) {
        const response=await route.fetch();const headers=response.headers();
        headers['content-security-policy']=cspMutation(url,headers['content-security-policy']);
        return route.fulfill({response,headers});
      }
      const response=await route.fetch({maxRedirects:0});
      const body=await response.body();
      responses.push({url,bytes:body.length});
      return route.fulfill({response,body});
    });
    const page=await ctx.newPage();page.on('pageerror',e=>errors.push(String(e)));
    const consoleErrors=[];page.on('console',m=>{if(/Content Security Policy|violates.*directive/i.test(m.text())) consoleErrors.push(m.text());});
    await page.goto(origin+'/verify-evidence-pack',{waitUntil:'networkidle'});
    assert.equal(await page.title(),'Check a SatGate receipt');
    assert.equal(await page.locator('link[rel=canonical]').getAttribute('href'),'https://satgate.io/verify-evidence-pack');
    assert.equal(await page.locator('meta[name=description]').getAttribute('content'),'Check a SatGate receipt yourself, with no SatGate account: download a real signed receipt (Evidence Pack) and verify its Ed25519 signature against the published keys.');
    const words=async result=>assert.equal(normal(await page.locator('main').innerText()),normal([...copy.before,result,...copy.after].filter(Boolean).join(' ')),item.name+' exact visible words');
    await words('');
    assert.equal(normal(await page.locator('body').innerText()),normal(await page.locator('main').innerText()));
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth),width);
    if(layoutState==='pass') await measureLayout(page,width,'empty');
    if(shot==='empty' || (layoutState==='pass' && shot==='pass')) await page.screenshot({path:path.join(lane,'shots',`${width}-empty.png`),fullPage:true});
    // Seed only after page assets load: checker requests must omit even cookies
    // which exist on this origin, not merely happen to run in a clean context.
    await ctx.addCookies([{name:'dummy',value:'dummy',url:origin}]);
    const initial=requests.length;
    const initialDownloads=responses.length;
    const start=performance.now();
    if(sample) await page.getByRole('button',{name:'Try a sample.',exact:true}).click();
    else await page.locator('#receipt-file').setInputFiles({name:'dummy-receipt.json',mimeType:'application/json',buffer:Buffer.from(item.bytes||item.text)});
    // Spinner has no visible or spoken words; aria-busy covers the result only.
    if(await page.getByTestId('verdict').getAttribute('aria-busy')==='true') {
      assert.equal(await page.getByTestId('verdict').innerText(),'');
      assert.equal(await page.getByTestId('verdict').locator('[aria-hidden=true]').count(),1);
      await words('');
    }
    await page.waitForFunction(()=>document.querySelector('[data-testid=verdict]').getAttribute('aria-busy')==='false',null,{timeout:130000});
    const text=await page.getByTestId('verdict').innerText();
    assert.equal(text,item.wanted||expected(item),item.name);
    await words(text);
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth),width);
    if(layoutState) await measureLayout(page,width,layoutState);
    if(shot && shot!=='empty') await page.screenshot({path:path.join(lane,'shots',`${width}-${shot}.png`),fullPage:true});
    for(const r of requests) assert.ok(allowedProductRequest(r),item.name+': '+r.url);
    for(const r of requests.slice(initial)) {
      assert.ok(r.intercepted || cspMutation,item.name+' request escaped audit');
      assert.ok(!r.headers.cookie,item.name+' cookie '+r.url);
      assert.ok(!new URL(r.url).search);
      assert.ok(!r.url.includes('dummy-receipt'),item.name+' leaked filename');
    }
    const keyRequests=requests.filter(r=>keyURLs.includes(r.url));
    assert.ok(keyRequests.length<=1,'more than one key request');
    if(keyRequests.length) {
      const artifact=sample?JSON.parse(fs.readFileSync(path.join(root,'public/evidence/mcp-budget-refusal-20261005.json'))):JSON.parse(item.text);
      assert.equal(keyRequests[0].url,artifact.issuer+'/.well-known/jwks.json');
    }
    if([record,issuer].includes(text) || item.expected==='malformed' || ['not-json','empty','large','invalid-utf8'].includes(item.name)) {
      assert.equal(keyRequests.length,0,item.name+' preflight keys');
      assert.ok(!requests.some(r=>r.url.startsWith(origin+'/receipt-checker/')),item.name+' preflight Python');
    }
    if(text===keys) assert.ok(!requests.some(r=>r.url===origin+'/receipt-checker/pyodide.mjs'),item.name+' keys failure loaded Python');
    assert.ok(!requests.some(r=>r.url.includes('verify_receipt_record.py')));
    assert.equal(await page.evaluate(()=>Object.keys(localStorage).length+Object.keys(sessionStorage).length),0);
    assert.deepEqual(await page.evaluate(()=>indexedDB.databases()),[]);
    assert.deepEqual(ctx.serviceWorkers(),[]);
    if(!cspMutation && !['runtime-crash'].includes(item.name)) { assert.deepEqual(errors,[]);assert.deepEqual(consoleErrors,[]); }
    await page.waitForFunction(()=>true); // allow worker close events to drain
    await new Promise(resolve=>setTimeout(resolve,50));
    assert.equal(page.workers().length,0,item.name+' worker not terminated');
    const downloads=responses.slice(initialDownloads);
    const result={name:item.name,width,exit:0,text,ms:Math.round(performance.now()-start),requests:requests.length,checkDownloads:downloads.reduce((sum,r)=>sum+r.bytes,0),consoleErrors};
    results.push(result);
    console.log('PASS',item.name,width,text);
    fs.mkdirSync(path.join(lane,'requests'),{recursive:true});
    fs.writeFileSync(path.join(lane,'requests',item.name+'-'+width+'.json'),JSON.stringify({requests,downloads},null,2));
    await ctx.close();
    return result;
  }
  const real={name:'sample',keyBytes:read('fixtures/jwks-mcp.json'),expected:'Valid'};
  for(const width of [320,390,480,1280]) {
    await scenario({...real,name:'layout-pass'},{sample:true,width,layoutState:'pass',shot:width<=390?'pass':null});
    await scenario({name:'layout-fail',text:'not json',expected:'malformed'},{width,layoutState:'fail'});
  }
  fs.writeFileSync(path.join(lane,'layout-results.json'),JSON.stringify(layout,null,2));
  const overflows=layout.filter(row=>row.outside.length || row.scrollWidth!==row.innerWidth || row.command.scrollWidth!==row.command.clientWidth);
  assert.deepEqual(overflows,[],'horizontal overflow: '+JSON.stringify(overflows));
  console.log(`PASS ${layout.length} viewport/state bounds and command byte checks`);
  if(layoutOnly) {await browser.close();return;}
  await scenario(real,{sample:true,shot:'empty'});
  await scenario({...real,name:'real-chosen-file',text:fs.readFileSync(path.join(root,'public/evidence/mcp-budget-refusal-20261005.json'),'utf8')});
  for(const width of [1440,390]) {
    await scenario({...real,name:'sample-'+width},{sample:true,shot:'pass',width});
    if(width===390) await scenario(real,{sample:true,shot:'empty',width});
    await scenario({name:'sample-wrong-key',keyBytes:read('fixtures/jwks-api.json'),wanted:signature},{sample:true,shot:'fail',width});
    await scenario({name:'keys-blocked',text:cases[0].text,wanted:keys},{shot:'keys',width});
  }
  for(const item of cases) await scenario(item);
  await scenario({...cases.find(c=>c.name==='recent-slow-clock'),name:'recent-slow-clock-without-server',wanted:malformed});
  // All additions to CSP have a browser-negative proof using the exact served
  // policy with just the token under test removed. No product bypass is added.
  const good=cases.find(c=>c.name==='pack-valid');
  const negatives=[

    ['worker-script-self',(url,csp)=>url.endsWith('.mjs')?csp.replace("script-src 'self' 'wasm-unsafe-eval'","script-src 'wasm-unsafe-eval'"):csp,runtime],
    ['worker-wasm',(url,csp)=>url.endsWith('.mjs')?csp.replace(" 'wasm-unsafe-eval'",''):csp,runtime],
    ['worker-connect-self',(url,csp)=>url.endsWith('.mjs')?csp.replace("connect-src 'self'","connect-src"):csp,runtime],
    ...keyURLs.map(url=>['worker-key-'+new URL(url).hostname,(target,csp)=>target.endsWith('.mjs')?csp.replace(' '+url,''):csp,keys]),
  ];
  for(const [name,mutate,sentence] of negatives) {
    const keyURL=keyURLs.find(u=>name==='worker-key-'+new URL(u).hostname);
    const candidate=keyURL?cases.find(c=>c.name==='issuer-'+new URL(keyURL).hostname)||good:good;
    await scenario({...candidate,name,wanted:sentence},{cspMutation:mutate});
  }
  // Native corpus verdicts and exact original options, through unchanged pack
  // Python in Pyodide. No record CLI is served or loaded, even by this harness.
  const ctx=await browser.newContext({serviceWorkers:'block'});
  const corpus=JSON.parse(read('corpus/parity-corpus.json'));
  // Keep Python's exact non-finite corpus values; JS JSON.parse rejects them.
  assert.ok(corpus.every(c=>c.startsWith('{"entry": "verify_pack",')));
  const worker=`import {loadPyodide} from '/receipt-checker/pyodide.mjs';
self.onmessage=async({data})=>{try{
const py=await loadPyodide({indexURL:'/receipt-checker/',stdout:()=>{},stderr:()=>{}});
await py.loadPackage('cryptography');
py.unpackArchive(await(await fetch('/receipt-checker/rfc8785-0.1.4-py3-none-any.whl')).arrayBuffer(),'zip',{extractDir:'/lib/python3.14/site-packages'});
py.FS.writeFile('/verify_evidence_pack.py',await(await fetch('/receipt-checker/verify_evidence_pack.py')).text());
py.globals.set('cases_json',JSON.stringify(data));
postMessage(JSON.parse(await py.runPythonAsync(\`import json,sys
sys.path.insert(0,'/')
import verify_evidence_pack as pack
cases=json.loads(cases_json)
mismatches=[]
for index,raw in enumerate(cases):
    c=json.loads(raw)
    result=pack.verify_pack(*c['args'],**c['kwargs'])
    if result['valid']!=c['valid']: mismatches.append(index)
json.dumps({'count':len(cases),'mismatches':mismatches})\`)));
}catch(e){postMessage({error:String(e)})}finally{self.close()}};`;
  await ctx.route('**/*',async route=>{
    const url=route.request().url();assert.ok(url.startsWith(origin+'/'),'parity external traffic');
    if(url===origin+'/receipt-checker-worker.mjs') {const response=await route.fetch();return route.fulfill({response,body:worker});}
    return route.continue();
  });
  const p=await ctx.newPage();await p.goto(origin+'/verify-evidence-pack');
  await p.evaluate(data=>{window.parity=null;const w=new Worker('/receipt-checker-worker.mjs',{type:'module',credentials:'omit'});w.onmessage=e=>{window.parity=e.data;w.terminate();};w.onerror=e=>{window.parity={error:e.message};w.terminate();};w.postMessage(data);},corpus);
  await p.waitForFunction(()=>window.parity!==null,null,{timeout:180000});
  const parity=await p.evaluate(()=>window.parity);assert.ok(!parity.error,JSON.stringify(parity));assert.deepEqual(parity.mismatches,[]);assert.equal(parity.count,corpus.length);
  const source=fs.readFileSync(path.join(root,'../tools/verify_evidence_pack.py'));
  const served=await(await ctx.request.get(origin+'/receipt-checker/verify_evidence_pack.py')).body();assert.deepEqual(source,served);
  const manifest=require('./receipt-checker-manifest.json');
  for(const file of manifest.files.filter(f=>f.notice)) {
    const response=await ctx.request.get(origin+'/receipt-checker/'+file.file);assert.equal(response.status(),200);
    const bytes=await response.body();assert.equal(bytes.length,file.bytes);assert.equal(crypto.createHash('sha256').update(bytes).digest('hex'),file.sha256);
    results.push({name:'notice-'+file.file,exit:0,bytes:bytes.length});
  }
  assert.equal((await ctx.request.get(origin+'/receipt-checker/verify_receipt_record.py')).status(),404);
  const redirect=await ctx.request.get(origin+'/verify',{maxRedirects:0});assert.equal(redirect.status(),307);assert.equal(redirect.headers().location,'/verify-evidence-pack');
  results.push({name:'public-corpus-cli-browser-parity',exit:0,...parity});
  fs.writeFileSync(path.join(lane,'browser-results.json'),JSON.stringify(results,null,2));
  console.log(JSON.stringify(results,null,2));
  await ctx.close();await browser.close();
})().catch(async e=>{console.error(e);fs.writeFileSync(path.join(lane,'browser-partial-results.json'),JSON.stringify(results,null,2));await browser?.close();process.exit(1)});

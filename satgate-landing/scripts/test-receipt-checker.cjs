/* eslint-disable @typescript-eslint/no-require-imports -- Test-only CJS supports the coordinator's NODE_PATH Playwright installation. */
// Test only: no fixtures, interception, or alternate verifier mode enter the build.
// Usage: NODE_PATH=<playwright node_modules> node scripts/test-receipt-checker.cjs LANE [BASE_URL]
const {chromium} = require('playwright');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const lane=process.argv[2], origin=process.argv[3] || 'http://127.0.0.1:3928';
const root=path.resolve(__dirname,'..');
const read=name=>fs.readFileSync(path.join(lane,name));
const cases=JSON.parse(read('corpus/ui-cases.json'));
cases.push({name:'invalid-utf8',bytes:[255],expected:'This file is not a receipt in JSON or a receipt record.'});
const keyURLs=['https://api.satgate.io/.well-known/jwks.json','https://satgate-mcp-saas.fly.dev/.well-known/jwks.json'];
const results=[];
let browser;
(async()=>{
  browser=await chromium.launch(process.env.CHROMIUM_EXECUTABLE ? {executablePath:process.env.CHROMIUM_EXECUTABLE} : {});
  const ctx=await browser.newContext({viewport:{width:1440,height:1000}});
  const page=await ctx.newPage();
  const errors=[];page.on('pageerror',e=>errors.push(String(e)));
  let requests=[];
  page.on('request',r=>requests.push({url:r.url(),method:r.method(),bytes:r.postDataBuffer()?.length || 0}));
  async function scenario(item, screenshots=false) {
    requests=[];
    await ctx.unrouteAll();
    for (const url of keyURLs) await ctx.route(url,route=>item.name==='keys-blocked' ? route.abort('failed') : route.fulfill({status:200,headers:{'content-type':'application/jwk-set+json','access-control-allow-origin':'*'},body:item.name==='keys-malformed' ? 'not JSON' : item.keyBytes || JSON.stringify(item.keys)}));
    await page.goto(origin+'/verify-evidence-pack');
    if (screenshots) await page.screenshot({path:path.join(lane,'desktop-before.png'),fullPage:true});
    const start=performance.now();
    const upload={name:item.name+'.json',mimeType:'application/json',buffer:Buffer.from(item.bytes || item.text)};
    if (item.name==='pack-valid') {
      const chooser=page.waitForEvent('filechooser');
      await page.locator('label[for=receipt-file]').click();
      await (await chooser).setFiles(upload);
    } else await page.locator('#receipt-file').setInputFiles(upload);
    const status=page.getByTestId('verdict');
    await page.waitForFunction(()=>document.querySelector('[data-testid=verdict]').getAttribute('aria-busy')==='false');
    const text=await status.innerText();
    if (item.expected==='Valid') assert.equal(text,'Valid',item.name);
    else { assert.ok(text.includes(item.expected),item.name+': '+text);assert.ok(!/^Valid$/m.test(text),item.name); }
    if (item.sentence) assert.ok(text.includes(item.sentence),item.name+': '+text);
    for (const r of requests) {
      assert.equal(r.bytes,0,item.name+' upload');assert.equal(r.method,'GET');
      const local = r.url.startsWith(origin+'/') && new URL(r.url).search === '' && (r.url === origin+'/verify-evidence-pack' || r.url.startsWith(origin+'/_next/static/') || r.url.startsWith(origin+'/receipt-checker/') || r.url === origin+'/receipt-checker-worker.mjs');
      assert.ok(local || keyURLs.includes(r.url),r.url);
    }
    assert.deepEqual(errors,[]);
    assert.equal(await page.evaluate(()=>Object.keys(localStorage).length+Object.keys(sessionStorage).length),0);
    fs.mkdirSync(path.join(lane,'requests'),{recursive:true});
    fs.writeFileSync(path.join(lane,'requests',item.name+'.json'),JSON.stringify(requests,null,2));
    results.push({name:item.name,exit:0,ms:Math.round(performance.now()-start),requests:requests.length,requestBodies:0,text});
    if (screenshots) {
      await page.screenshot({path:path.join(lane,'desktop-valid.png'),fullPage:true});
      await page.setViewportSize({width:390,height:844});
      assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth),390);
      await page.screenshot({path:path.join(lane,'phone-valid.png'),fullPage:true});
      await page.reload();
      await page.screenshot({path:path.join(lane,'phone-before.png'),fullPage:true});
      const rendered=await page.locator('main').innerText();
      fs.writeFileSync(path.join(lane,'rendered-text.txt'),rendered);
      const copy=JSON.parse(fs.readFileSync(path.join(__dirname,'receipt-checker-copy.json'),'utf8'));
      let position=0;
      for (const sentence of copy) {const next=rendered.indexOf(sentence,position);assert.ok(next>=position,'Draft copy missing or out of order: '+sentence);position=next+sentence.length;}
      assert.ok(!rendered.includes('July 18'));
      assert.ok(!rendered.includes('—'));
      for (const name of ['verify_evidence_pack.py','verify_receipt_record.py']) {
        const source=fs.readFileSync(path.join(root,'../tools',name));
        const served=Buffer.from(await (await page.request.get(origin+'/receipt-checker/'+name)).body());
        assert.deepEqual(source,served);
        assert.ok(rendered.includes(crypto.createHash('sha256').update(source).digest('hex')));
      }
      const jargon=rendered.match(/governed|governance|primitive|authority|bounded|admission|rail|deterministic|caveat|canonical|attenuat|substrate|posture|comprehension|prospect|buyer|promotion|request-path|proof path|—/gi)||[];
      fs.writeFileSync(path.join(lane,'rendered-jargon.json'),JSON.stringify(jargon));
      for (const route of ['/verify-evidence-pack','/docs','/']) {
        await page.goto(origin+route);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth),390,route);
      }
      await page.setViewportSize({width:1440,height:1000});
    }
  }
  await scenario({name:'real-cloud',text:read('real-pack.json').toString(),keyBytes:read('mcp-jwks.json'),expected:'Valid'},true);
  for (const item of cases) await scenario(item);
  // Real drag-and-drop path, independent of the file picker.
  requests=[];
  await page.goto(origin+'/verify-evidence-pack');
  const good=cases.find(c=>c.name==='pack-valid');
  await ctx.unrouteAll();
  await ctx.route(keyURLs[0],r=>r.fulfill({headers:{'access-control-allow-origin':'*'},body:JSON.stringify(good.keys)}));
  const transfer=await page.evaluateHandle(({text})=>{const dt=new DataTransfer();dt.items.add(new File([text],'receipt.json',{type:'application/json'}));return dt;},good);
  await page.locator('label[for=receipt-file]').dispatchEvent('drop',{dataTransfer:transfer});
  await page.waitForFunction(()=>document.querySelector('[data-testid=verdict]').innerText==='Valid',null,{timeout:120000});
  results.push({name:'drop-box',exit:0});
  assert.ok(requests.every(r=>r.bytes===0 && r.method==='GET' && (r.url.startsWith(origin+'/') || keyURLs.includes(r.url))));
  fs.writeFileSync(path.join(lane,'requests/drop-box.json'),JSON.stringify(requests,null,2));
  requests=[];
  // Browser API parity uses test-side code served by Playwright only. The exact
  // native CLI verdicts and every captured original option are replayed, including
  // old dates, nonfinite values and source-origin mismatch cases.
  const corpus=JSON.parse(read('corpus/parity-corpus.json'));
  const parityWorker=`import {loadPyodide} from '/receipt-checker/pyodide.mjs';
self.onmessage=async({data})=>{try{
const py=await loadPyodide({indexURL:'/receipt-checker/',stdout:()=>{},stderr:()=>{}});
await py.loadPackage('cryptography');
py.unpackArchive(await(await fetch('/receipt-checker/rfc8785-0.1.4-py3-none-any.whl')).arrayBuffer(),'zip',{extractDir:'/lib/python3.14/site-packages'});
for(const n of ['verify_evidence_pack.py','verify_receipt_record.py'])py.FS.writeFile('/'+n,await(await fetch('/receipt-checker/'+n)).text());
py.globals.set('cases_json',JSON.stringify(data));
postMessage(JSON.parse(await py.runPythonAsync(\`import json,sys
sys.path.insert(0,'/')
import verify_evidence_pack as pack,verify_receipt_record as record
cases=json.loads(cases_json)
mismatches=[]
for index,raw in enumerate(cases):
    c=json.loads(raw)
    fn=pack.verify_pack if c['entry']=='verify_pack' else record.verify_record
    result=fn(*c['args'],**c['kwargs'])
    if result['valid']!=c['valid']: mismatches.append(index)
json.dumps({'count':len(cases),'mismatches':mismatches})\`)));
}catch(e){postMessage({error:String(e)})}};`;
  await ctx.route(origin+'/__test/parity.mjs',r=>r.fulfill({contentType:'text/javascript',body:parityWorker}));
  await page.evaluate(corpus=>{window.parity=null;const w=new Worker('/__test/parity.mjs',{type:'module'});w.onmessage=e=>{window.parity=e.data;w.terminate();};w.onerror=e=>window.parity={error:e.message};w.postMessage(corpus);},corpus);
  await page.waitForFunction(()=>window.parity!==null,null,{timeout:120000});
  const parity=await page.evaluate(()=>window.parity);
  assert.ok(!parity.error,JSON.stringify(parity));assert.deepEqual(parity.mismatches,[]);assert.equal(parity.count,corpus.length);
  results.push({name:'public-corpus-cli-browser-parity',exit:0,...parity});
  assert.ok(requests.every(r=>r.bytes===0 && r.method==='GET' && r.url.startsWith(origin+'/')));
  fs.writeFileSync(path.join(lane,'requests/parity.json'),JSON.stringify(requests,null,2));
  fs.writeFileSync(path.join(lane,'browser-results.json'),JSON.stringify(results,null,2));
  console.log(JSON.stringify(results,null,2));
  await browser.close();
})().catch(async e=>{console.error(e);fs.writeFileSync(path.join(lane,'browser-partial-results.json'),JSON.stringify(results,null,2));await browser?.close();process.exit(1)});

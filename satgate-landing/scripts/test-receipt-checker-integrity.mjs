// Exercise the actual builder in an isolated directory; never change product pins.
import assert from 'node:assert/strict';
import {mkdtemp, mkdir, readFile, writeFile, rm, copyFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {spawnSync} from 'node:child_process';
const root=fileURLToPath(new URL('../',import.meta.url));
const manifest=JSON.parse(await readFile(path.join(root,'scripts/receipt-checker-manifest.json'),'utf8'));
const notices=[...manifest.files.filter(f=>f.notice),manifest.files.find(f=>f.file==='pyodide.mjs')];
const temp=await mkdtemp(path.join(tmpdir(),'receipt-integrity-'));
try {
  const site=path.join(temp,'site');
  await mkdir(path.join(site,'scripts'),{recursive:true});
  await mkdir(path.join(temp,'tools'));
  await mkdir(path.join(site,'public/receipt-checker'),{recursive:true});
  await copyFile(path.join(root,'scripts/prepare-receipt-checker.mjs'),path.join(site,'scripts/prepare-receipt-checker.mjs'));
  await copyFile(path.join(root,'public/receipt-checker-worker.mjs'),path.join(site,'public/receipt-checker-worker.mjs'));
  for (const name of ['verify_evidence_pack.py','verify_receipt_record.py']) await copyFile(path.join(root,'../tools',name),path.join(temp,'tools',name));
  for (const file of notices) {
    if (file.source) await copyFile(path.join(root,file.source),path.join(site,file.source));
  }
  // Download interception returns the real hash-checked served bytes, not mocks.
  const downloads=Object.fromEntries(await Promise.all(notices.filter(f=>!f.source).map(async f=>[f.url,(await readFile(path.join(root,'public/receipt-checker',f.file))).toString('base64')])));
  await writeFile(path.join(site,'fetch.mjs'),`const bytes=${JSON.stringify(downloads)};globalThis.fetch=async url=>new Response(Buffer.from(bytes[url],'base64'));`);
  function run() { return spawnSync(process.execPath,['--import',path.join(site,'fetch.mjs'),path.join(site,'scripts/prepare-receipt-checker.mjs')],{encoding:'utf8'}); }
  async function pins(files) { await writeFile(path.join(site,'scripts/receipt-checker-manifest.json'),JSON.stringify({...manifest,files})); }
  await pins(notices);
  const good=run(); assert.equal(good.status,0,good.stderr);
  const results=[{case:'valid-notices',exit:good.status}];
  for (const file of notices) {
    for (const field of ['sha256','bytes']) {
      const bad={...file,[field]:field==='sha256'?'0'.repeat(64):file.bytes+1};
      await pins([bad]);
      const result=run();
      assert.notEqual(result.status,0,file.file+' '+field);
      assert.ok(result.stderr.includes('Runtime integrity failed: '+file.file),result.stderr);
      results.push({case:file.file+'-'+field,exit:result.status,expectedFailure:true});
    }
  }
  console.log(JSON.stringify(results,null,2));
} finally { await rm(temp,{recursive:true,force:true}); }

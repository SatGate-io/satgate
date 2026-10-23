// Build-time downloads only. Runtime requests always use this site's static files.
import { createHash } from 'node:crypto';
import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const root = fileURLToPath(new URL('../', import.meta.url));
const manifest = JSON.parse(await readFile(path.join(root, 'scripts/receipt-checker-manifest.json'), 'utf8'));
const dest = path.join(root, 'public/receipt-checker');
await mkdir(dest, { recursive: true });
const hash = data => createHash('sha256').update(data).digest('hex');
for (const file of manifest.files) {
  const target = path.join(dest, file.file);
  let data;
  if (file.source) {
    if (file.url !== `/receipt-checker/${file.file}`) throw new Error(`Invalid local notice URL: ${file.file}`);
    data = await readFile(path.join(root, file.source));
  } else {
    try { data = await readFile(target); } catch { /* cold build */ }
  }
  if (!file.source && (!data || hash(data) !== file.sha256)) {
    const response = await fetch(file.url, { signal: AbortSignal.timeout(120000) });
    if (!response.ok) throw new Error(`Runtime download failed: ${file.file}: ${response.status}`);
    data = Buffer.from(await response.arrayBuffer());
  }
  if (data.length !== file.bytes || hash(data) !== file.sha256) throw new Error(`Runtime integrity failed: ${file.file}`);
  await writeFile(target, data);
}
const verifiers = {};
for (const name of ['verify_evidence_pack.py', 'verify_receipt_record.py']) {
  const data = await readFile(path.join(root, '../tools', name));
  await writeFile(path.join(dest, name), data);
  verifiers[name] = { sha256: hash(data), bytes: data.length };
}
const workerBytes = (await readFile(path.join(root, 'public/receipt-checker-worker.mjs'))).length;
const bytes = manifest.files.reduce((n, f) => n + f.bytes, 0) + Object.values(verifiers).reduce((n, f) => n + f.bytes, 0) + workerBytes;
await writeFile(path.join(dest, 'build.json'), JSON.stringify({ pyodide: manifest.version, bytes, verifiers }, null, 2) + '\n');
console.log(`Receipt checker: ${bytes} bytes, all pinned hashes verified, verifier copies unchanged`);

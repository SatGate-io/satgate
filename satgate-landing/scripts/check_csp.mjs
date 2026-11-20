import { chromium } from 'playwright';
import { readdir } from 'node:fs/promises';
import assert from 'node:assert/strict';

// Run against `npm start -- --hostname 127.0.0.1 --port 3114` after build.
// No external traffic, form submissions or real email is permitted.
const origin = process.env.CSP_TEST_ORIGIN || 'http://127.0.0.1:3114';
assert.equal(new URL(origin).hostname, '127.0.0.1', 'local verification only');
const paths = [];
async function discover(dir, parts = []) {
  for (const item of await readdir(dir, { withFileTypes: true })) {
    if (item.isDirectory() && !item.name.startsWith('_') && !item.name.startsWith('.')) {
      await discover(`${dir}/${item.name}`, [...parts, item.name]);
    } else if (item.isFile() && item.name === 'page.tsx') {
      assert.ok(!parts.some(p => p.includes('[')), 'dynamic routes need explicit test cases');
      paths.push('/' + parts.filter(p => !p.startsWith('(')).join('/'));
    }
  }
}
await discover('app');
// These legacy URLs are now config redirects, not executable demo pages.
paths.push('/crawl', '/dashboard', '/monetize');
const browser = await chromium.launch({ headless: true });
const originalPolicy = "default-src 'self'; script-src 'self' 'nonce-NONCE' 'strict-dynamic'; script-src-attr 'none'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' https://api.satgate.io https://cloud.satgate.io https://satgate-production-9354.up.railway.app; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'";
const workerPolicy = "default-src 'none'; script-src 'self' 'wasm-unsafe-eval'; connect-src 'self' https://api.satgate.io/.well-known/jwks.json https://satgate-mcp-saas.fly.dev/.well-known/jwks.json https://mcp.satgate.io/.well-known/jwks.json";
let count = 0;
try {
  const context = await browser.newContext();
  const workerResponse = await context.request.get(origin + '/receipt-checker-worker.mjs');
  assert.equal(workerResponse.status(), 200);
  assert.equal(workerResponse.headers()['content-security-policy'], workerPolicy);
  // The narrowly scoped WebAssembly permission must not become JavaScript eval.
  assert.ok(!/(?:^|[\s;])'unsafe-eval'/.test(workerPolicy));
  assert.ok(!/script-src[^;]*unsafe-inline/.test(workerPolicy));
  await context.route('**/*', route => {
    const url = new URL(route.request().url());
    return url.origin === origin && route.request().method() === 'GET'
      ? route.continue() : route.abort();
  });
  await context.addInitScript(() => {
    window.__cspViolations = [];
    document.addEventListener('securitypolicyviolation', event => {
      window.__cspViolations.push(`${event.violatedDirective}: ${event.blockedURI}`);
    });
  });
  for (const path of [...new Set(paths)].sort()) {
    const page = await context.newPage();
    const errors = [];
    page.on('console', message => {
      if (/content.security.policy|violates.*directive|refused to (execute|load|connect|apply)/i.test(message.text())) errors.push(message.text());
    });
    const response = await page.goto(origin + path, { waitUntil: 'networkidle' });
    assert.equal(response.status(), 200, `${path}: page did not load`);
    const csp = response.headers()['content-security-policy'];
    assert.equal(csp.replace(/'nonce-[^']+'/g, "'nonce-NONCE'"),
      originalPolicy,
      `${path}: exact policy; all other routes unchanged`);
    assert.ok(csp && !csp.includes('unsafe-eval'), `${path}: missing or unsafe policy`);
    assert.match(csp, /script-src [^;]*'nonce-[A-Za-z0-9+/=]+'/);
    assert.ok(!/script-src[^;]*unsafe-inline/.test(csp));
    assert.equal(response.headers()['cache-control'], 'private, no-store');
    assert.ok((await page.locator('body').innerText()).trim().length > 0, `${path}: empty body`);
    const violations = await page.evaluate(() => window.__cspViolations);
    assert.deepEqual(violations, [], `${path}: CSP violation events`);
    assert.deepEqual(errors, [], `${path}: CSP console errors`);
    const nonce = csp.match(/'nonce-([^']+)'/)[1];
    const scriptNonces = await page.locator('script:not([type="application/ld+json"])').evaluateAll(scripts => scripts.map(s => s.nonce));
    assert.ok(scriptNonces.length > 0, `${path}: missing hydration scripts`);
    assert.ok(scriptNonces.every(n => n === nonce), `${path}: script nonce mismatch`);
    const repeated = await context.request.get(page.url());
    assert.notEqual(repeated.headers()['content-security-policy'], csp, `${path}: nonce reused`);
    console.log(`PASS ${path}`);
    count++;
    await page.close();
  }
  console.log(`PASS ${count} page routes: no CSP console errors or violation events; nonce freshness and hydration verified`);
} finally { await browser.close(); }

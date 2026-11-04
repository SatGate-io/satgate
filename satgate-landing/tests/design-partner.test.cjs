/* eslint-disable @typescript-eslint/no-require-imports -- CommonJS loader exercises the TS route without a separate build artifact. */
const { test, beforeEach, afterEach } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const ts = require('typescript');

// Exercise the actual Next route, not a second implementation. No network is
// permitted: both Redis and mail HTTP calls are replaced before every request.
require.extensions['.ts'] = (module, filename) => {
  module._compile(ts.transpileModule(fs.readFileSync(filename, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText, filename);
};
const { POST } = require('../app/api/design-partner/route.ts');
const { escapeHtml, MAX_BODY_BYTES } = require('../lib/design-partner-intake.ts');
const originalFetch = global.fetch;
const originalEnv = { ...process.env };
let mails, counters, now, failMail, failStore;
const application = () => ({ name: 'Dummy Person', email: 'dummy@example.invalid', company: 'Dummy Co', role: 'Engineer', agentCount: '2', apis: ['Example'], challenge: 'Dummy challenge', website: '' });
const request = (data = application(), headers = {}) => new Request('http://localhost/api/design-partner', {
  method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(data),
});
beforeEach(() => {
  mails = []; counters = new Map(); now = 0; failMail = false; failStore = false;
  process.env.RESEND_API_KEY = 'dummy-mail-credential';
  process.env.RESEND_FROM_NAME = 'Dummy Sender';
  process.env.RESEND_FROM_EMAIL = 'sender@example.invalid';
  process.env.UPSTASH_REDIS_REST_URL = 'https://redis.example.invalid';
  process.env.UPSTASH_REDIS_REST_TOKEN = 'dummy-store-credential';
  global.fetch = async (url, options) => {
    if (url === 'https://redis.example.invalid') {
      assert.equal(options.cache, 'no-store');
      const [command, script, count, key] = JSON.parse(options.body);
      assert.equal(command, 'EVAL'); assert.equal(count, '1');
      assert.match(script, /EXPIRE/); assert.match(script, /3600/);
      assert.match(key, /^design-partner:[a-f0-9]{64}$/);
      assert.ok(!key.includes('@'));
      if (failStore) return Response.json({ error: 'dummy private store detail' }, { status: 500 });
      let entry = counters.get(key);
      if (!entry || now >= entry.expires) entry = { count: 0, expires: now + 3600 };
      counters.set(key, entry);
      return Response.json({ result: ++entry.count });
    }
    assert.equal(url, 'https://api.resend.com/emails', 'unexpected network call');
    mails.push(JSON.parse(options.body));
    return failMail ? new Response('dummy private provider detail', { status: 422 }) : Response.json({ id: 'dummy-message' });
  };
});
afterEach(() => { global.fetch = originalFetch; process.env = { ...originalEnv }; });

test('escapes every HTML value and removes the obsolete cc', async () => {
  const data = application();
  const payload = `<img src=x onerror="dummy">&'`;
  for (const field of ['name', 'company', 'role', 'agentCount', 'challenge']) data[field] = payload;
  data.email = "dummy&'@example.invalid";
  data.apis = [payload];
  assert.equal((await POST(request(data))).status, 200);
  assert.equal(mails.length, 2);
  const notification = mails[0];
  assert.equal(notification.cc, undefined);
  assert.equal(notification.html.includes(payload), false);
  assert.ok(notification.html.split(escapeHtml(payload)).length >= 7);
  assert.ok(notification.html.includes(escapeHtml(data.email)));
  assert.ok(!notification.html.includes(data.email));
  assert.ok(notification.subject.includes(escapeHtml(payload)));
});

test('confirmation is fixed and contains no submitted text', async () => {
  const first = { ...application(), agentCount: 'Dummy agent count' };
  await POST(request(first));
  const second = { ...application(), name: 'Other Dummy', company: 'Other Company', email: 'other@example.invalid', challenge: 'Other challenge' };
  await POST(request(second));
  assert.equal(mails[1].html, mails[3].html);
  assert.equal(mails[1].subject, mails[3].subject);
  for (const text of Object.values(first).filter(v => typeof v === 'string' && v)) assert.ok(!mails[1].html.includes(text));
  assert.match(mails[1].html, /Thanks for applying!/);
  assert.match(mails[1].html, /We received your Design Partner application\./);
  assert.match(mails[1].html, /We review applications within 24 hours/);
  assert.deepEqual(mails[1].to, [first.email]);
});

test('refuses announced and actual bodies above 16 KB before mail or rate storage', async () => {
  assert.equal((await POST(request(application(), { 'Content-Length': String(MAX_BODY_BYTES + 1) }))).status, 413);
  assert.equal((await POST(request({ ...application(), challenge: 'x'.repeat(MAX_BODY_BYTES) }))).status, 413);
  // Actual streaming bytes, not JS characters or a trusted Content-Length.
  assert.equal((await POST(request({ ...application(), challenge: '😀'.repeat(5000) }, { 'Content-Length': '1' }))).status, 413);
  assert.equal(mails.length, 0); assert.equal(counters.size, 0);
});

test('honeypot silently accepts without any side effects', async () => {
  const response = await POST(request({ website: 'https://bot.example.invalid' }));
  assert.equal(response.status, 200); assert.equal((await response.json()).ok, true);
  assert.equal(mails.length, 0); assert.equal(counters.size, 0);
});

test('three per address per hour, case normalized, distinct recipient independent, resets after TTL', async () => {
  const results = await Promise.all(Array.from({ length: 8 }, (_, i) => POST(request({ ...application(), email: i % 2 ? 'DUMMY@example.invalid' : 'dummy@example.invalid' }))));
  assert.equal(results.filter(r => r.status === 200).length, 3);
  assert.equal(results.filter(r => r.status === 429).length, 5);
  assert.equal(results.find(r => r.status === 429).headers.get('retry-after'), '3600');
  assert.equal(mails.length, 6);
  assert.equal((await POST(request({ ...application(), email: 'distinct@example.invalid' }))).status, 200);
  now = 3600;
  assert.equal((await POST(request())).status, 200);
});

test('validates email, types, required fields and all bounded fields', async () => {
  const bad = [null, [], {}, { email: 'invalid' }, { email: '.dummy@example.invalid' }, { email: 'dummy..test@example.invalid' }, { email: 'dummy.@example.invalid' }, { email: `dummy@${'a'.repeat(64)}.invalid` }, { email: 'a\r\n@example.invalid' }, { name: 12 }, { company: [] }, { company: 'x'.repeat(201) }, { name: 'x'.repeat(101) }, { role: 'x'.repeat(101) }, { agentCount: 'x'.repeat(51) }, { challenge: 'x'.repeat(4001) }, { email: `${'a'.repeat(65)}@example.invalid` }, { apis: 'not-an-array' }, { apis: Array(21).fill('x') }, { apis: [12] }, { apis: ['x'.repeat(101)] }, { company: 'Injected\r\nheader' }];
  for (const patch of bad) {
    const data = patch === null || Array.isArray(patch) ? patch : { ...application(), ...patch };
    if (patch && Object.keys(patch).length === 0) data.name = '';
    assert.equal((await POST(request(data))).status, 400);
  }
  assert.equal(mails.length, 0); assert.equal(counters.size, 0);
  assert.equal((await POST(new Request('http://localhost', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{' }))).status, 400);
  assert.equal((await POST(request(application(), { 'Content-Type': 'text/plain' }))).status, 415);
});

test('provider errors and transport exceptions never reach response or logs', async () => {
  const originalError = console.error;
  const logs = [];
  console.error = (...args) => logs.push(args.join(' '));
  try {
    failMail = true;
    let response = await POST(request());
    assert.equal(response.status, 500); assert.equal(await response.text(), '{"error":"Internal error"}');
    assert.equal(mails.length, 1);
    global.fetch = async () => { throw new Error('dummy private transport detail'); };
    response = await POST(request());
    assert.equal(response.status, 500); assert.equal(await response.text(), '{"error":"Internal error"}');
    assert.ok(logs.every(l => l === '[Design Partner] Delivery unavailable'));
  } finally { console.error = originalError; }
});

test('missing or failed shared store fails closed without email', async () => {
  delete process.env.UPSTASH_REDIS_REST_TOKEN;
  assert.equal((await POST(request())).status, 503);
  process.env.UPSTASH_REDIS_REST_TOKEN = 'dummy-store-credential'; failStore = true;
  const response = await POST(request());
  assert.equal(response.status, 503); assert.equal(await response.text(), '{"error":"Service unavailable"}');
  assert.equal(mails.length, 0);
});

test('missing mail configuration is not reported as successful delivery', async () => {
  delete process.env.RESEND_API_KEY;
  assert.equal((await POST(request())).status, 503); assert.equal(mails.length, 0);
});

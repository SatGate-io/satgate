/* eslint-disable @typescript-eslint/no-require-imports -- CommonJS loader exercises the TS route without a separate build artifact. */
const { test, beforeEach, afterEach } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const ts = require('typescript');

// Exercise the actual Next route. Refuse all network except the two stubbed
// services; afterEach also detects unexpected calls swallowed by route catches.
require.extensions['.ts'] = (module, filename) => {
  module._compile(ts.transpileModule(fs.readFileSync(filename, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText, filename);
};
const { POST } = require('../app/api/design-partner/route.ts');
const { escapeHtml, MAX_BODY_BYTES, success } = require('../lib/design-partner-intake.ts');
const originalFetch = global.fetch;
const originalEnv = { ...process.env };
let mails, counters, now, failMail, failStore, unexpected, storeCalls, confirmationFailure;
const application = () => ({ name: 'Dummy Person', email: 'dummy@example.invalid', company: 'Dummy Co', role: 'Engineer', agentCount: '2', apis: ['Example'], challenge: 'Dummy challenge', website: '' });
const request = (data = application(), headers = {}) => new Request('http://localhost/api/design-partner', {
  method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(data),
});
const snapshot = async response => ({ status: response.status, headers: [...response.headers], body: await response.text() });
const accepted = { status: 200, headers: [['content-type', 'application/json']], body: JSON.stringify(success) };
const notifications = () => mails.filter(mail => mail.subject.startsWith('🚀'));
const confirmations = () => mails.filter(mail => mail.subject === 'SatGate Design Partner Application Received');
const clearStorage = () => {
  for (const key of ['UPSTASH_REDIS_REST_URL', 'UPSTASH_REDIS_REST_TOKEN', 'KV_REST_API_URL', 'KV_REST_API_TOKEN']) delete process.env[key];
};
beforeEach(() => {
  mails = []; counters = new Map(); now = 0; failMail = false; failStore = false;
  unexpected = []; storeCalls = []; confirmationFailure = false;
  clearStorage();
  process.env.RESEND_API_KEY = 'dummy-mail-credential';
  process.env.RESEND_FROM_NAME = 'Dummy Sender';
  process.env.RESEND_FROM_EMAIL = 'sender@example.invalid';
  process.env.UPSTASH_REDIS_REST_URL = 'https://redis.example.invalid';
  process.env.UPSTASH_REDIS_REST_TOKEN = 'dummy-store-credential';
  global.fetch = async (url, options) => {
    if (url === 'https://redis.example.invalid' || url === 'https://kv.example.invalid') {
      storeCalls.push(url);
      assert.equal(options.cache, 'no-store');
      assert.equal(options.method, 'POST');
      assert.ok(options.signal instanceof AbortSignal);
      assert.equal(options.headers.Authorization, url.includes('kv.') ? 'Bearer dummy-kv-credential' : 'Bearer dummy-store-credential');
      const [command, script, count, key] = JSON.parse(options.body);
      assert.equal(command, 'EVAL'); assert.equal(count, '1');
      assert.match(script, /INCR/); assert.match(script, /EXPIRE/); assert.match(script, /3600/);
      assert.match(key, /^design-partner:[a-f0-9]{64}$/);
      assert.ok(!key.includes('@'));
      if (failStore === 'http') return Response.json({ error: 'dummy private store detail' }, { status: 500 });
      if (failStore === 'read') throw new Error('dummy private read detail');
      if (failStore === 'write') return Response.json({ error: 'dummy private write detail' });
      if (failStore === 'partial-write') return Response.json({ result: 1, error: 'dummy private write detail' });
      if (failStore === 'json') return new Response('not json');
      if (failStore) return Response.json({ result: failStore === 'null' ? null : failStore });
      let entry = counters.get(key);
      if (!entry || now >= entry.expires) entry = { count: 0, expires: now + 3600 };
      counters.set(key, entry);
      return Response.json({ result: ++entry.count });
    }
    if (url !== 'https://api.resend.com/emails') {
      unexpected.push('unexpected network call');
      throw new Error('unexpected network call');
    }
    assert.equal(options.method, 'POST');
    assert.ok(options.signal instanceof AbortSignal);
    const mail = JSON.parse(options.body);
    mails.push(mail);
    if (confirmationFailure && mail.subject === 'SatGate Design Partner Application Received') {
      if (confirmationFailure === 'throw') throw new Error('dummy private confirmation transport detail');
      return new Response('dummy private confirmation detail', { status: 422 });
    }
    return failMail ? new Response('dummy private provider detail', { status: 422 }) : Response.json({ id: 'dummy-message' });
  };
});
afterEach(() => {
  global.fetch = originalFetch; process.env = { ...originalEnv };
  assert.deepEqual(unexpected, []);
});

test('escapes every HTML value and removes the obsolete cc', async () => {
  const data = application();
  const payload = `<img src=x onerror="dummy">&'`;
  for (const field of ['name', 'company', 'role', 'agentCount', 'challenge']) data[field] = payload;
  data.email = "dummy&'@example.invalid";
  data.apis = [payload];
  assert.deepEqual(await snapshot(await POST(request(data))), accepted);
  assert.equal(mails.length, 2);
  const notification = mails[0];
  assert.equal(notification.cc, undefined);
  assert.equal(notification.html.includes(payload), false);
  assert.ok(notification.html.split(escapeHtml(payload)).length >= 7);
  assert.ok(notification.html.includes(escapeHtml(data.email)));
  assert.ok(!notification.html.includes(data.email));
  assert.ok(notification.subject.includes(escapeHtml(payload)));
  assert.ok(mails.every(mail => mail.cc === undefined));
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
  assert.equal((await POST(request({ ...application(), challenge: '😀'.repeat(5000) }, { 'Content-Length': '1' }))).status, 413);
  assert.equal(mails.length, 0); assert.equal(storeCalls.length, 0);
});

test('honeypot silently accepts without any side effects', async () => {
  assert.deepEqual(await snapshot(await POST(request({ website: 'https://bot.example.invalid' }))), accepted);
  assert.equal(mails.length, 0); assert.equal(storeCalls.length, 0);
});

test('limit suppresses only confirmation: concurrent normalized addresses, independent recipients and TTL', async () => {
  const results = await Promise.all(Array.from({ length: 8 }, (_, i) => POST(request({ ...application(), email: i % 2 ? 'DUMMY@example.invalid' : 'dummy@example.invalid' }))));
  for (const response of results) assert.deepEqual(await snapshot(response), accepted);
  assert.equal(notifications().length, 8); assert.equal(confirmations().length, 3);
  const fixedTo = notifications()[0].to;
  assert.ok(fixedTo.length === 1 && !fixedTo.includes(application().email));
  assert.ok(notifications().every(mail => JSON.stringify(mail.to) === JSON.stringify(fixedTo)));
  assert.deepEqual(await snapshot(await POST(request({ ...application(), email: 'distinct@example.invalid' }))), accepted);
  assert.equal(confirmations().length, 4);
  now = 3600;
  assert.deepEqual(await snapshot(await POST(request())), accepted);
  assert.equal(confirmations().length, 5); assert.equal(notifications().length, 10);
});

for (const pair of ['upstash', 'kv', 'both', 'incomplete-upstash-with-kv']) {
  test(`healthy storage allows confirmation using ${pair} configuration`, async () => {
    if (pair !== 'upstash') {
      process.env.KV_REST_API_URL = 'https://kv.example.invalid';
      process.env.KV_REST_API_TOKEN = 'dummy-kv-credential';
    }
    if (pair === 'kv') { delete process.env.UPSTASH_REDIS_REST_URL; delete process.env.UPSTASH_REDIS_REST_TOKEN; }
    if (pair === 'incomplete-upstash-with-kv') delete process.env.UPSTASH_REDIS_REST_TOKEN;
    assert.deepEqual(await snapshot(await POST(request())), accepted);
    assert.equal(notifications().length, 1); assert.equal(confirmations().length, 1);
    assert.deepEqual(storeCalls, [pair === 'upstash' || pair === 'both' ? 'https://redis.example.invalid' : 'https://kv.example.invalid']);
  });
}

for (const config of ['absent', 'upstash-url-only', 'upstash-token-only', 'kv-url-only', 'kv-token-only', 'mixed-pairs', 'non-https']) {
  test(`${config} storage still accepts and notifies internally without confirmation`, async () => {
    clearStorage();
    if (config === 'upstash-url-only' || config === 'mixed-pairs') process.env.UPSTASH_REDIS_REST_URL = 'https://redis.example.invalid';
    if (config === 'upstash-token-only') process.env.UPSTASH_REDIS_REST_TOKEN = 'dummy-store-credential';
    if (config === 'kv-url-only') process.env.KV_REST_API_URL = 'https://kv.example.invalid';
    if (config === 'kv-token-only' || config === 'mixed-pairs') process.env.KV_REST_API_TOKEN = 'dummy-kv-credential';
    if (config === 'non-https') {
      process.env.UPSTASH_REDIS_REST_URL = 'http://redis.example.invalid';
      process.env.UPSTASH_REDIS_REST_TOKEN = 'dummy-store-credential';
    }
    assert.deepEqual(await snapshot(await POST(request())), accepted);
    assert.equal(notifications().length, 1); assert.equal(confirmations().length, 0); assert.equal(storeCalls.length, 0);
  });
}

for (const failure of ['http', 'read', 'write', 'partial-write', 'json', 'null', 'not-a-number', -1, 1.5]) {
  test(`storage ${failure} failure still accepts and notifies internally without confirmation`, async () => {
    failStore = failure;
    const originalError = console.error;
    const logs = [];
    console.error = (...args) => logs.push(args.join(' '));
    try {
      assert.deepEqual(await snapshot(await POST(request())), accepted);
      assert.equal(notifications().length, 1); assert.equal(confirmations().length, 0); assert.equal(storeCalls.length, 1);
      assert.deepEqual(logs, []);
    } finally { console.error = originalError; }
  });
}

test('validates email, types, required fields and all bounded fields', async () => {
  const bad = [null, [], {}, { email: 'invalid' }, { email: '.dummy@example.invalid' }, { email: 'dummy..test@example.invalid' }, { email: 'dummy.@example.invalid' }, { email: `dummy@${'a'.repeat(64)}.invalid` }, { email: 'a\r\n@example.invalid' }, { name: 12 }, { company: [] }, { company: 'x'.repeat(201) }, { name: 'x'.repeat(101) }, { role: 'x'.repeat(101) }, { agentCount: 'x'.repeat(51) }, { challenge: 'x'.repeat(4001) }, { email: `${'a'.repeat(65)}@example.invalid` }, { apis: 'not-an-array' }, { apis: Array(21).fill('x') }, { apis: [12] }, { apis: ['x'.repeat(101)] }, { company: 'Injected\r\nheader' }];
  for (const patch of bad) {
    const data = patch === null || Array.isArray(patch) ? patch : { ...application(), ...patch };
    if (patch && Object.keys(patch).length === 0) data.name = '';
    assert.equal((await POST(request(data))).status, 400);
  }
  assert.equal(mails.length, 0); assert.equal(storeCalls.length, 0);
  assert.equal((await POST(new Request('http://localhost', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{' }))).status, 400);
  assert.equal((await POST(request(application(), { 'Content-Type': 'text/plain' }))).status, 415);
});

test('internal provider errors and transport exceptions never reach response or logs', async () => {
  const originalError = console.error;
  const logs = [];
  console.error = (...args) => logs.push(args.join(' '));
  try {
    failMail = true;
    let response = await POST(request());
    assert.equal(response.status, 500); assert.equal(await response.text(), '{"error":"Internal error"}');
    assert.equal(mails.length, 1); assert.equal(storeCalls.length, 0);
    const stub = global.fetch;
    global.fetch = async (url, options) => {
      if (url === 'https://api.resend.com/emails') throw new Error('dummy private transport detail');
      return stub(url, options);
    };
    response = await POST(request());
    assert.equal(response.status, 500); assert.equal(await response.text(), '{"error":"Internal error"}');
    assert.ok(logs.every(l => l === '[Design Partner] Delivery unavailable'));
  } finally { console.error = originalError; }
});

for (const failure of ['http', 'throw']) {
  test(`failed confirmation ${failure} consumes reservation but never changes browser success`, async () => {
    confirmationFailure = failure;
    for (let i = 0; i < 4; i++) assert.deepEqual(await snapshot(await POST(request())), accepted);
    assert.equal(notifications().length, 4); assert.equal(confirmations().length, 3);
  });
}

test('missing mail configuration fails closed even without storage', async () => {
  clearStorage();
  delete process.env.RESEND_API_KEY;
  const response = await POST(request());
  assert.equal(response.status, 503); assert.equal(await response.text(), '{"error":"Service unavailable"}');
  assert.equal(mails.length, 0); assert.equal(storeCalls.length, 0);
});

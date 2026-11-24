import assert from 'node:assert/strict';
import { chromium } from 'playwright';

// Run against the locally built site, like check_csp.mjs.
const origin = process.env.CSP_TEST_ORIGIN || 'http://127.0.0.1:3114';
assert.equal(new URL(origin).origin, origin, 'expected an origin with no path or credentials');
assert.equal(new URL(origin).hostname, '127.0.0.1', 'local verification only');
const signup = 'https://cloud.satgate.io/cloud/signup';
const title = 'Dollar limits and signed receipts for AI trading agents | SatGate';
const description = 'Dollar limits for your AI trading agent. Approve large orders, give each agent its own key, get signed receipts, and stop any agent with one button.';
const browser = await chromium.launch({ headless: true });
try {
  const context = await browser.newContext({ serviceWorkers: 'block' });
  await context.route('**/*', route => {
    const request = route.request();
    return new URL(request.url()).origin === origin && request.method() === 'GET'
      ? route.continue() : route.abort();
  });
  const page = await context.newPage();
  assert.equal((await page.goto(`${origin}/`)).status(), 200);

  // K1–K3 and the revised Robinhood introduction.
  assert.equal(await page.getByRole('heading', { level: 1 }).innerText(),
    'One budget for your AI agent, across Robinhood and Kalshi.');
  for (const copy of [
    'Set a dollar limit once. SatGate stops the agent at that limit wherever it trades, and signs a receipt for every order.',
    "Early access. SatGate runs today on our own Robinhood and Kalshi accounts. We're waiting on approval from Robinhood and Kalshi before we open connections to customers. Sign up and we'll email you when they open.",
    'SatGate is built for Robinhood first. Everything in this section is about Robinhood.',
  ]) {
    assert.ok(await page.getByText(copy, { exact: true }).isVisible(), copy);
  }
  assert.doesNotMatch(await page.locator('body').innerText(), /works with today|set up my agent's limits/i);

  // K4 plus both relabeled links; use accessible names and inspect hrefs without following them.
  const links = page.getByRole('link', { name: 'Join the early-access list', exact: true });
  assert.equal(await links.count(), 3);
  assert.deepEqual(await links.evaluateAll(anchors => anchors.map(link => link.getAttribute('href'))),
    [signup, `${signup}?intent=trading`, `${signup}?intent=trading`]);
  for (const link of await links.all()) {
    assert.ok(await link.isVisible());
    assert.equal((await link.innerText()).trim(), 'Join the early-access list');
  }
  assert.equal(await page.getByText('Free for 30 days, no card needed. Continue with Personal for $9.99 a month.', { exact: true }).count(), 2);

  assert.equal(await page.title(), title);
  assert.equal(await page.locator('meta[name="description"]').getAttribute('content'), description);
  assert.equal(await page.locator('meta[property="og:title"]').getAttribute('content'), title);
  assert.equal(await page.locator('meta[property="og:description"]').getAttribute('content'), description);

  // K5 and the platform's unchanged links.
  assert.equal((await page.goto(`${origin}/platform`)).status(), 200);
  const platformCopy = page.locator('p').filter({
    hasText: 'Letting an AI agent trade? Set one limit across Robinhood and Kalshi. Early access, pending approval.',
  });
  assert.ok(await platformCopy.isVisible());
  assert.equal(await platformCopy.getByRole('link', { name: 'See how', exact: true }).getAttribute('href'), '/trading');
  const trialLinks = page.getByRole('link', { name: 'Start free trial', exact: true });
  assert.ok(await trialLinks.count());
  for (const link of await trialLinks.all()) assert.equal(await link.getAttribute('href'), signup);

  const redirect = await context.request.get(`${origin}/trading`, { maxRedirects: 0 });
  assert.equal(redirect.status(), 308);
  assert.equal(redirect.headers().location, '/');
  console.log('PASS K1–K5, Robinhood introduction, both relabeled links, unchanged metadata and trial copy, and /trading redirect');
} finally {
  await browser.close();
}

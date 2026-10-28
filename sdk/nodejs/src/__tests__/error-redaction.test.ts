import * as http from 'http';
import * as net from 'net';
import { AddressInfo } from 'net';
import { inspect } from 'util';

import { AlbyWallet, LNDWallet, SatGateAgentClient } from '../agent-client';
import { SatGateClient } from '../client';
import {
  AuthenticationError,
  PaymentFailedError,
  SatGateError,
  ValidationError,
} from '../errors';
import { SatGate } from '../private-beta';
import * as transport from '../transport';

// The private-beta client sends through node-fetch (ESM-only, so jest
// cannot load it); here node-fetch is the global fetch, so its requests
// reach the same local server.
jest.mock('node-fetch', () => ({
  __esModule: true,
  default: (url: string, init: RequestInit) => globalThis.fetch(url, init),
}));

/*
 * SDK error messages carry no credential and no request path or query.
 *
 * Every message the SDK builds or throws about a request (an HTTP status
 * error quoting the response, a JSON parse error quoting the body, a body
 * read error, an invalid header, a wrapped error) is passed through
 * transport.redact: the request URL is named by its host and port, its path
 * and query are removed, and every credential the request carried
 * (Authorization, X-Admin-Token, the LND macaroon header, tokens in the body
 * or the query) is replaced, also where the response quotes it back. Error
 * types and status codes are unchanged.
 *
 * A local server (127.0.0.1 only) echoes the request's path, query,
 * credential headers and body into its answer. Each check reads
 * util.inspect of the error: its message, stack, own properties and causes.
 */

// Synthetic values. Every credential and URL part contains "sekret" or
// "hidden", so one search finds any of them.
const ADMIN = 'adm-sekret-0123456789';
const CAP = 'cap-sekret-0123456789';
const API_KEY = 'key-sekret-0123456789';
const ALBY = 'alby-sekret-0123456789';
const MACAROON = '0201036c6e64sekret0123';
const QUERY_VALUE = 'qv-sekret-0123456789';
const PARENT = 'parent-sekret-0123456789';
const BASE_PATH = '/hidden-base';
const QUERY = `hidden-query=${QUERY_VALUE}`;
const MARKERS = ['sekret', 'hidden', '/api/', '/v1/'];

type Shape =
  | 'status-text' | 'status-json' | 'bad-request' | 'bad-request-text'
  | 'not-json' | 'payment-error' | 'cut-short';

interface Echo {
  base: string;
  hits: number;
  shape: Shape;
  /** Answer the first request with an L402 challenge. */
  l402First: boolean;
  close(): Promise<void>;
}

async function startEcho(shape: Shape): Promise<Echo> {
  const echo: Echo = { base: '', hits: 0, shape, l402First: false, close: async () => undefined };
  const server = http.createServer((req, res) => {
    const chunks: Buffer[] = [];
    req.on('data', (c: Buffer) => chunks.push(c));
    req.on('end', () => {
      echo.hits += 1;
      if (echo.l402First && echo.hits === 1) {
        res.writeHead(402, {
          'Content-Type': 'application/json',
          'WWW-Authenticate': 'L402 macaroon="m", invoice="lnbc1"',
        });
        res.end(JSON.stringify({ invoice: 'lnbc1', macaroon: 'm' }));
        return;
      }
      // The text starts with the request path, so that the start of the
      // body a JSON parse error quotes holds it.
      const text =
        `${req.url} cannot be served; authorization=${req.headers.authorization} ` +
        `admin=${req.headers['x-admin-token']} macaroon=${req.headers['grpc-metadata-macaroon']} ` +
        `body=${Buffer.concat(chunks).toString('utf8')}`;
      switch (echo.shape) {
        case 'status-text':
          res.writeHead(500, { 'Content-Type': 'text/plain' });
          res.end(text);
          return;
        case 'status-json':
          res.writeHead(500, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ error: text, message: text }));
          return;
        case 'bad-request':
          res.writeHead(400, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ error: text }));
          return;
        case 'bad-request-text':
          res.writeHead(400, { 'Content-Type': 'text/plain' });
          res.end(text);
          return;
        case 'payment-error':
          res.writeHead(200, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ payment_error: text }));
          return;
        case 'cut-short':
          res.writeHead(200, { 'Content-Type': 'application/json', 'Content-Length': String(text.length + 1000) });
          res.write(`{"x": "${text}`, () => res.destroy());
          return;
        default:
          res.writeHead(200, { 'Content-Type': 'application/json' });
          res.end(text);
      }
    });
  });
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
  echo.base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
  echo.close = async () => {
    server.closeAllConnections();
    await new Promise<void>((resolve) => server.close(() => resolve()));
  };
  return echo;
}

async function closedPort(): Promise<number> {
  const server = net.createServer();
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
  const { port } = server.address() as AddressInfo;
  await new Promise<void>((resolve) => server.close(() => resolve()));
  return port;
}

async function caught(p: Promise<unknown>): Promise<unknown> {
  try {
    await p;
  } catch (err) {
    return err;
  }
  throw new Error('the call did not reject');
}

function expectRedacted(err: unknown): void {
  const text = inspect(err, { depth: 6 });
  for (const marker of MARKERS) {
    expect(text).not.toContain(marker);
  }
  expect(String(err)).not.toContain('@');
}

type ErrorClass = abstract new (...args: never[]) => Error;

/**
 * err is of class expected. Errors from fetch's body parsing come from
 * Node's own realm, so a built-in class (SyntaxError, TypeError) is matched
 * by constructor name; an SDK class must match with instanceof.
 */
function expectClass(err: unknown, expected: ErrorClass): void {
  expect((err as Error)?.constructor?.name).toBe(expected.name);
  if (expected !== SyntaxError && expected !== TypeError) expect(err).toBeInstanceOf(expected);
}

function alby(base: string): AlbyWallet {
  const wallet = new AlbyWallet({ accessToken: ALBY });
  (wallet as unknown as { apiBase: string }).apiBase = base;
  return wallet;
}

const STATUS_QUOTED: Partial<Record<Shape, ErrorClass>> = { 'status-json': SatGateError };
const AGENT_SHAPES: Partial<Record<Shape, ErrorClass>> = {
  'status-text': SatGateError, 'status-json': SatGateError, 'not-json': SatGateError, 'cut-short': SatGateError,
};

// name -> (call taking the server's base URL, {shape: the error class expected}).
const CALLS: Record<string, [(base: string) => Promise<unknown>, Partial<Record<Shape, ErrorClass>>]> = {
  'SatGateClient tokens.mint': [
    (b) => new SatGateClient({ url: `${b}${BASE_PATH}?${QUERY}`, token: ADMIN }).tokens.mint(),
    { ...STATUS_QUOTED, 'status-text': SatGateError, 'bad-request': ValidationError,
      'bad-request-text': SatGateError, 'not-json': SyntaxError, 'cut-short': TypeError }],
  'SatGateClient tokens.validate': [
    (b) => new SatGateClient({ url: `${b}${BASE_PATH}?${QUERY}`, token: ADMIN }).tokens.validate(CAP),
    { ...STATUS_QUOTED, 'bad-request': ValidationError, 'not-json': SyntaxError }],
  'SatGateClient tokens.delegate': [
    (b) => new SatGateClient({ url: `${b}${BASE_PATH}?${QUERY}`, token: ADMIN }).tokens.delegate({ parentToken: PARENT } as never),
    { ...STATUS_QUOTED, 'bad-request': ValidationError, 'not-json': SyntaxError }],
  'SatGateClient governance.ban': [
    (b) => new SatGateClient({ url: `${b}${BASE_PATH}?${QUERY}`, token: ADMIN }).governance.ban('sig-sekret-0123'),
    { ...STATUS_QUOTED, 'bad-request': ValidationError, 'not-json': SyntaxError }],
  'SatGateClient governance.getGraph': [
    (b) => new SatGateClient({ url: `${b}${BASE_PATH}?${QUERY}`, token: ADMIN }).governance.getGraph(),
    { ...STATUS_QUOTED, 'not-json': SyntaxError }],
  'SatGateClient governance.reset': [
    (b) => new SatGateClient({ url: `${b}${BASE_PATH}?${QUERY}`, token: ADMIN }).governance.reset(),
    { ...STATUS_QUOTED, 'not-json': SyntaxError }],
  'SatGateClient request (path and query)': [
    (b) => new SatGateClient({ url: b, token: ADMIN }).request('GET', `/hidden-route?${QUERY}`),
    { ...STATUS_QUOTED, 'bad-request': ValidationError, 'bad-request-text': SatGateError, 'not-json': SyntaxError }],
  'SatGateClient ping': [
    (b) => new SatGateClient({ url: `${b}${BASE_PATH}?${QUERY}`, token: ADMIN }).ping(CAP),
    { 'status-text': SyntaxError, 'not-json': SyntaxError, 'cut-short': TypeError }],
  'SatGate issue': [
    (b) => new SatGate({ apiKey: API_KEY, baseUrl: `${b}${BASE_PATH}?${QUERY}` }).issue({ task: 't', agent: 'a', allow: [] }),
    { ...STATUS_QUOTED, 'status-text': SatGateError, 'not-json': SyntaxError, 'cut-short': TypeError }],
  'SatGate pay': [
    (b) => new SatGate({ apiKey: API_KEY, baseUrl: `${b}${BASE_PATH}` }).pay({ upstream: 'u', capability: { token: CAP } }),
    { ...STATUS_QUOTED, 'not-json': SyntaxError }],
  'SatGate verify': [
    (b) => new SatGate({ apiKey: API_KEY, baseUrl: `${b}${BASE_PATH}` }).verify({ signature: 'sig-sekret-0123' }),
    { ...STATUS_QUOTED, 'not-json': SyntaxError }],
  'SatGateAgentClient get (capability token)': [
    (b) => new SatGateAgentClient({ gatewayUrl: `${b}${BASE_PATH}`, token: CAP }).get(`/hidden-path?${QUERY}`),
    AGENT_SHAPES],
  'SatGateAgentClient post': [
    (b) => new SatGateAgentClient({ gatewayUrl: `${b}${BASE_PATH}`, token: CAP }).post(`/hidden-path?${QUERY}`, { token: PARENT }),
    AGENT_SHAPES],
  'SatGateAgentClient validateToken': [
    (b) => new SatGateAgentClient({ gatewayUrl: `${b}${BASE_PATH}`, token: CAP }).validateToken(),
    AGENT_SHAPES],
  'SatGateAgentClient ping': [
    (b) => new SatGateAgentClient({ gatewayUrl: `${b}${BASE_PATH}`, token: CAP }).ping(),
    AGENT_SHAPES],
  'SatGateAgentClient delegate': [
    (b) => new SatGateAgentClient({ gatewayUrl: `${b}${BASE_PATH}`, token: CAP }).delegate(),
    { 'status-text': SatGateError, 'not-json': SatGateError }],
  'SatGateAgentClient get (admin token mint)': [
    (b) => new SatGateAgentClient({ gatewayUrl: `${b}${BASE_PATH}`, adminToken: ADMIN }).get(`/x?${QUERY}`),
    { 'status-text': AuthenticationError, 'not-json': AuthenticationError, 'cut-short': AuthenticationError }],
  'AlbyWallet getBalance': [
    (b) => alby(`${b}${BASE_PATH}?${QUERY}`).getBalance(),
    { 'status-text': SatGateError, 'not-json': SyntaxError, 'cut-short': TypeError }],
  'AlbyWallet payInvoice': [
    (b) => alby(`${b}${BASE_PATH}?${QUERY}`).payInvoice('lnbc1'),
    { 'status-text': PaymentFailedError, 'not-json': SyntaxError }],
};

function cases<T>(table: Record<string, [T, Partial<Record<Shape, ErrorClass>>]>): [string, Shape][] {
  const out: [string, Shape][] = [];
  for (const [name, [, shapes]] of Object.entries(table)) {
    for (const shape of Object.keys(shapes) as Shape[]) out.push([name, shape]);
  }
  return out;
}

describe('an error quoting the response names no credential, path or query', () => {
  it.each(cases(CALLS))('%s: %s', async (name, shape) => {
    const [call, expected] = CALLS[name];
    const echo = await startEcho(shape);
    try {
      const err = await caught(call(echo.base));
      expect(echo.hits).toBeGreaterThanOrEqual(1);
      expectClass(err, expected[shape] as ErrorClass);
      expectRedacted(err);
    } finally {
      await echo.close();
    }
  });
});

describe('an LND wallet error names no macaroon, path or query', () => {
  // LND is reached over https; here the global fetch is replaced by one that
  // answers like the echo server, so no TLS server is needed.
  const realFetch = globalThis.fetch;
  afterEach(() => {
    globalThis.fetch = realFetch;
  });

  const LND: Record<string, [(w: LNDWallet) => Promise<unknown>, Partial<Record<Shape, ErrorClass>>]> = {
    'LNDWallet getBalance': [(w) => w.getBalance(), { 'status-text': SatGateError, 'not-json': SyntaxError }],
    'LNDWallet payInvoice': [(w) => w.payInvoice('lnbc1'), {
      'status-text': PaymentFailedError, 'payment-error': PaymentFailedError, 'not-json': SyntaxError,
    }],
  };

  it.each(cases(LND))('%s: %s', async (name, shape) => {
    const [call, expected] = LND[name];
    let hits = 0;
    globalThis.fetch = (async (url: string, init: RequestInit) => {
      hits += 1;
      const headers = new Headers(init.headers);
      const u = new URL(url);
      const text = `${u.pathname}${u.search} cannot be served; macaroon=${headers.get('grpc-metadata-macaroon')} body=${init.body}`;
      if (shape === 'status-text') return new Response(text, { status: 500 });
      if (shape === 'payment-error') return new Response(JSON.stringify({ payment_error: text }), { status: 200 });
      return new Response(text, { status: 200 });
    }) as typeof fetch;
    const wallet = new LNDWallet({ host: `127.0.0.1:8080${BASE_PATH}?${QUERY}`, macaroonHex: MACAROON });
    const err = await caught(call(wallet));
    expect(hits).toBe(1);
    expectClass(err, expected[shape] as ErrorClass);
    expectRedacted(err);
  });
});

describe('the L402 flow names no credential, path or query', () => {
  it.each(['status-text', 'status-json'] as Shape[])('a wallet error quoted as Lightning payment failed: %s', async (shape) => {
    const gateway = await startEcho('status-text');
    gateway.l402First = true;
    const wallet = await startEcho(shape);
    try {
      const agent = new SatGateAgentClient({ gatewayUrl: `${gateway.base}${BASE_PATH}`, token: CAP, wallet: alby(`${wallet.base}${BASE_PATH}?${QUERY}`) });
      const err = await caught(agent.get(`/hidden-path?${QUERY}`));
      expectClass(err, PaymentFailedError);
      expect((err as Error).message.startsWith('Lightning payment failed: PaymentFailedError: Alby payment failed: ')).toBe(true);
      expectRedacted(err);
    } finally {
      await gateway.close();
      await wallet.close();
    }
  });

  it('a retry answer that is not JSON gives a SyntaxError naming no L402 token, path or query', async () => {
    const gateway = await startEcho('not-json');
    gateway.l402First = true;
    try {
      const wallet = { payInvoice: async () => 'pre-sekret-0123', getBalance: async () => 0 };
      const agent = new SatGateAgentClient({ gatewayUrl: `${gateway.base}${BASE_PATH}`, token: CAP, wallet });
      const err = await caught(agent.get(`/hidden-path?${QUERY}`));
      expectClass(err, SyntaxError);
      expect(gateway.hits).toBe(2);
      expectRedacted(err);
    } finally {
      await gateway.close();
    }
  });
});

describe('an invalid header value is named in no error', () => {
  const CALLS_WITH_BAD_HEADER: Record<string, [(base: string) => Promise<unknown>, ErrorClass]> = {
    'SatGateAgentClient get (capability token)': [
      (b) => new SatGateAgentClient({ gatewayUrl: b, token: `${CAP}\rx` }).get('/x'), TypeError],
    'SatGateAgentClient L402 retry': [
      (b) => new SatGateAgentClient({
        gatewayUrl: b, token: CAP, wallet: { payInvoice: async () => 'pre-sekret\rx', getBalance: async () => 0 },
      }).get('/x'), TypeError],
    'SatGateClient tokens.mint': [
      (b) => new SatGateClient({ url: b, token: `${ADMIN}\rx` }).tokens.mint(), SatGateError],
    'SatGateClient ping': [(b) => new SatGateClient({ url: b, token: ADMIN }).ping(`${CAP}\rx`), SatGateError],
    'SatGate pay': [(b) => new SatGate({ apiKey: `${API_KEY}\rx`, baseUrl: b }).pay({ upstream: 'u', capability: {} }), SatGateError],
    'AlbyWallet getBalance': [(b) => {
      const wallet = new AlbyWallet({ accessToken: `${ALBY}\rx` });
      (wallet as unknown as { apiBase: string }).apiBase = b;
      return wallet.getBalance();
    }, TypeError],
  };

  it.each(Object.keys(CALLS_WITH_BAD_HEADER))('%s', async (name) => {
    const [call, expected] = CALLS_WITH_BAD_HEADER[name];
    const echo = await startEcho('status-text');
    echo.l402First = name.includes('L402');
    try {
      const err = await caught(call(echo.base));
      expectClass(err, expected);
      expectRedacted(err);
      expect(echo.hits).toBe(name.includes('L402') ? 1 : 0);
    } finally {
      await echo.close();
    }
  });
});

describe('reviewer cases (round 4)', () => {
  it('a JSON parse error for a body echoing the request path and query names neither', async () => {
    const server = http.createServer((req, res) => {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(req.url);
    });
    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
    try {
      const c = new SatGateClient({ url: `http://127.0.0.1:${(server.address() as AddressInfo).port}`, token: 'fixture' });
      const err = await caught(c.request('GET', `/private-route?credential=${QUERY_VALUE}`));
      expectClass(err, SyntaxError);
      expect(String(err)).not.toContain('private');
      expect(String(err)).not.toContain(QUERY_VALUE);
    } finally {
      server.closeAllConnections();
      await new Promise<void>((resolve) => server.close(() => resolve()));
    }
  });

  for (const [secret, path, query] of [
    [QUERY_VALUE, '/private-route', 'access='],
    ['0123456789ab', '/p', 'q='],
  ]) {
    it(`status, JSON and header errors echoing path ${path} and the credential name none of them`, async () => {
      let status = 200;
      const server = http.createServer((req, res) => {
        req.resume();
        req.on('end', () => {
          res.writeHead(status, { 'content-type': 'application/json' });
          res.end(`invalid response for ${req.url} ${req.headers.authorization || req.headers['x-admin-token'] || ''}`);
        });
      });
      await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
      const base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
      const pq = `${path}?${query}${secret}`;
      try {
        const results: [string, unknown, ErrorClass][] = [];
        results.push(['agent-json', await caught(new SatGateAgentClient({ gatewayUrl: base, token: secret }).get(pq)), SatGateError]);
        results.push(['admin-json', await caught(new SatGateClient({ url: base + pq, token: secret }).tokens.mint()), SyntaxError]);
        status = 500;
        results.push(['agent-status', await caught(new SatGateAgentClient({ gatewayUrl: base, token: secret }).get(pq)), SatGateError]);
        results.push(['agent-header', await caught(new SatGateAgentClient({ gatewayUrl: base, token: `${secret}\n` }).get('/x')), SatGateError]);
        for (const [, err, expected] of results) {
          expectClass(err, expected);
          expect(String(err)).not.toContain(secret);
          expect(String(err)).not.toContain(path === '/p' ? '/p' : 'private-route');
          expect(String(err)).not.toContain(query);
        }
      } finally {
        server.closeAllConnections();
        await new Promise<void>((resolve) => server.close(() => resolve()));
      }
    });
  }

  it('a refused connection for a path with a query names no query name or value', async () => {
    const port = await closedPort();
    const err = await caught(new SatGateAgentClient({ gatewayUrl: `http://127.0.0.1:${port}`, token: CAP }).get(`/hidden-path?${QUERY}`));
    expectClass(err, SatGateError);
    expectRedacted(err);
  });
});

describe('the redaction function', () => {
  it('replaces the URL, path, query and credentials and keeps other text', () => {
    const url = `https://gw.example:8443/hidden-base/x?${QUERY}`;
    const redaction = transport.requestRedaction(url, {
      headers: { Authorization: `Bearer ${CAP}`, 'X-Admin-Token': ADMIN, 'X-Api-Key': 'k+sekret/0123=' },
      body: JSON.stringify({ parentToken: PARENT, caveats: ['scope = api:read'], nested: { preimage: 'pre-sekret' } }),
    });
    const out = transport.redact(
      `GET ${url} failed; /hidden-base/x and /x; ${CAP} ${ADMIN} ${PARENT} pre-sekret; ` +
      `quoted "Bearer ${CAP}" encoded ${encodeURIComponent('k+sekret/0123=')}; status 500; scope = api:read; the hidden-query name`,
      redaction,
    );
    for (const marker of ['sekret', 'hidden']) expect(out).not.toContain(marker);
    expect(out.startsWith('GET gw.example:8443 failed;')).toBe(true);
    expect(out).toContain('status 500');
    expect(out).toContain('scope = api:read');
  });

  it('redactError keeps the object, its type, status code and cause', () => {
    const cause = new Error(`inner ${CAP}`);
    const err = new SatGateError(`outer /hidden-path ${CAP}`, 502);
    (err as Error & { cause?: unknown }).cause = cause;
    const redaction = transport.requestRedaction('http://127.0.0.1:1/hidden-path', { headers: { Authorization: `Bearer ${CAP}` } });
    expect(transport.redactError(err, redaction)).toBe(err);
    expect(err).toBeInstanceOf(SatGateError);
    expect(err.statusCode).toBe(502);
    expect((err as Error & { cause?: unknown }).cause).toBe(cause);
    expect(err.message).toBe('outer [path and query removed] [redacted]');
    expect(cause.message).toBe('inner [redacted]');
  });

  it('drops the response text a JSON parse error quotes', () => {
    let err: unknown;
    try {
      JSON.parse('cannot serve /anything at all, long enough to be cut');
    } catch (e) {
      err = e;
    }
    transport.redactError(err, transport.requestRedaction('http://127.0.0.1:1/x', {}));
    expectClass(err, SyntaxError);
    expect((err as Error).message).toBe(`Unexpected token 'c', ${transport.RESPONSE_TEXT_REMOVED} is not valid JSON`);
  });

  it('short values are removed only as part of the query', () => {
    const out = transport.redact('HTTP 500 for /p?q=50 and q=50', transport.requestRedaction('http://h:1/p?q=50'));
    expect(out).toBe('HTTP 500 for [path and query removed] and [redacted]');
  });
});

import * as net from 'net';
import { AddressInfo } from 'net';
import { inspect } from 'util';

import { AlbyWallet, LNDWallet, SatGateAgentClient } from '../agent-client';
import { SatGateClient } from '../client';
import { RequestUrlRefusedError, SatGateError } from '../errors';
import { SatGate } from '../private-beta';
import { fetchWithoutRedirects } from '../transport';

// The private-beta client sends through node-fetch; here it is a mock that
// records calls and, per test, answers or fails the way node-fetch does.
jest.mock('node-fetch', () => ({ __esModule: true, default: jest.fn() }));
const nodeFetch = (jest.requireMock('node-fetch') as { default: jest.Mock }).default;

/*
 * No SDK error names the user information, path or query of a request URL.
 *
 * A request URL with a user name or password, or one that cannot be parsed,
 * is refused before anything is sent, with RequestUrlRefusedError naming
 * only the host and port. Errors from the fetch itself (network errors,
 * body-read and JSON errors) name no part of the URL but its host and port.
 *
 * Each check looks at util.inspect of the error: its message, stack, own
 * properties and causes. Requests go to a local TCP listener that counts
 * connections, so "nothing was sent" means it saw none.
 */

const USER = 'synthetic-user';
const PASSWORD = 'synthetic-password';
const CFG_PATH = '/cfgpath';
const CFG_QUERY = 'cfgquery=hidden-value';
const URL_PARTS = [USER, PASSWORD, 'cfgpath', 'cfgquery', 'hidden-value'];
const SDK_PATHS = ['/api/capability', '/v1/'];
const MACAROON_HEX = '0201036c6e64';

function expectNoUrlParts(err: unknown, extra: string[] = []): void {
  const text = inspect(err, { depth: 6 });
  for (const part of [...URL_PARTS, ...extra]) {
    expect(text).not.toContain(part);
  }
  expect((err as Error).message).not.toContain('@');
}

function expectRefused(err: unknown, host: string, label: string): void {
  expect(err).toBeInstanceOf(RequestUrlRefusedError);
  const e = err as RequestUrlRefusedError;
  expect(e.host).toBe(host);
  expect(e.message).toBe(
    `${label} URL for ${host} has a user name or password in it; the SDK does not ` +
    'send a request to a URL with user information, and no request was sent',
  );
  expectNoUrlParts(e, SDK_PATHS);
}

function expectUnparseable(err: unknown, label: string): void {
  expect(err).toBeInstanceOf(RequestUrlRefusedError);
  const e = err as RequestUrlRefusedError;
  expect(e.host).toBe('an unreadable host');
  expect(e.message).toBe(`${label} URL could not be parsed; no request was sent`);
  expectNoUrlParts(e, SDK_PATHS);
}

/** Errors from fetch and Response come from Node's realm, not jest's, so check by shape. */
function expectError(err: unknown, name?: string): void {
  expect(typeof (err as Error)?.message).toBe('string');
  if (name) expect((err as Error).name).toBe(name);
}

async function caught(p: Promise<unknown>): Promise<unknown> {
  try {
    await p;
  } catch (e) {
    return e;
  }
  throw new Error('the call did not reject');
}

interface Listener {
  port: number;
  connections: number;
  close(): Promise<void>;
}

async function listener(): Promise<Listener> {
  const sockets = new Set<net.Socket>();
  const state = { connections: 0 };
  const srv = net.createServer((socket) => {
    state.connections += 1;
    sockets.add(socket);
    socket.destroy();
  });
  await new Promise<void>((resolve) => srv.listen(0, '127.0.0.1', resolve));
  return {
    port: (srv.address() as AddressInfo).port,
    get connections() { return state.connections; },
    close: () => new Promise<void>((resolve) => {
      for (const s of sockets) s.destroy();
      srv.close(() => resolve());
    }),
  };
}

async function closedPort(): Promise<number> {
  const srv = net.createServer();
  await new Promise<void>((resolve) => srv.listen(0, '127.0.0.1', resolve));
  const port = (srv.address() as AddressInfo).port;
  await new Promise<void>((resolve) => srv.close(() => resolve()));
  return port;
}

const USER_INFO: Record<string, string> = {
  'a user name and password': `${USER}:${PASSWORD}@`,
  'a user name only': `${USER}@`,
  'a password only': `:${PASSWORD}@`,
};

type Call = (base: string, lndHost: string) => Promise<unknown>;

const GLOBAL_FETCH_CALLS: Record<string, { label: string; call: Call }> = {
  'SatGateClient tokens.mint': { label: 'SatGate gateway', call: (b) => new SatGateClient({ url: b, token: 'adm-0123' }).tokens.mint() },
  'SatGateClient tokens.validate': { label: 'SatGate gateway', call: (b) => new SatGateClient({ url: b, token: 'adm-0123' }).tokens.validate('cap-0123') },
  'SatGateClient governance.getGraph': { label: 'SatGate gateway', call: (b) => new SatGateClient({ url: b, token: 'adm-0123' }).governance.getGraph() },
  'SatGateClient ping': { label: 'SatGate gateway', call: (b) => new SatGateClient({ url: b, token: 'adm-0123' }).ping('cap-0123') },
  'SatGateAgentClient get (capability token)': { label: 'SatGate gateway', call: (b) => new SatGateAgentClient({ gatewayUrl: b, token: 'cap-0123' }).get('/x') },
  'SatGateAgentClient post (capability token)': { label: 'SatGate gateway', call: (b) => new SatGateAgentClient({ gatewayUrl: b, token: 'cap-0123' }).post('/x', { a: 1 }) },
  'SatGateAgentClient get (admin token mint)': { label: 'SatGate gateway', call: (b) => new SatGateAgentClient({ gatewayUrl: b, adminToken: 'adm-0123' }).get('/x') },
  'SatGateAgentClient delegate': { label: 'SatGate gateway', call: (b) => new SatGateAgentClient({ gatewayUrl: b, token: 'cap-0123' }).delegate() },
  'LNDWallet getBalance': { label: 'LND host', call: (_b, h) => new LNDWallet({ host: h, macaroonHex: MACAROON_HEX }).getBalance() },
  'LNDWallet payInvoice': { label: 'LND host', call: (_b, h) => new LNDWallet({ host: h, macaroonHex: MACAROON_HEX }).payInvoice('lnbc1') },
  'AlbyWallet getBalance': { label: 'Alby API host', call: (b) => albyAt(b).getBalance() },
  'AlbyWallet payInvoice': { label: 'Alby API host', call: (b) => albyAt(b).payInvoice('lnbc1') },
};

function albyAt(base: string): AlbyWallet {
  const wallet = new AlbyWallet({ accessToken: 'alby-0123' });
  (wallet as unknown as { apiBase: string }).apiBase = base;
  return wallet;
}

const NODE_FETCH_CALLS: Record<string, (base: string) => Promise<unknown>> = {
  'SatGate issue': (b) => new SatGate({ apiKey: 'key-0123', baseUrl: b }).issue({ task: 't', agent: 'a', allow: ['x'] }),
  'SatGate pay': (b) => new SatGate({ apiKey: 'key-0123', baseUrl: b }).pay({ upstream: 'u', capability: 'c' }),
  'SatGate verify': (b) => new SatGate({ apiKey: 'key-0123', baseUrl: b }).verify({ id: 'r' }),
};

describe('a request URL with user information is refused before anything is sent', () => {
  let srv: Listener;

  beforeEach(async () => {
    srv = await listener();
    nodeFetch.mockReset();
  });

  afterEach(async () => {
    await srv.close();
  });

  for (const [shape, info] of Object.entries(USER_INFO)) {
    for (const [name, { label, call }] of Object.entries(GLOBAL_FETCH_CALLS)) {
      it(`${name}: a URL with ${shape} is refused naming only host and port, and nothing is sent`, async () => {
        const host = `127.0.0.1:${srv.port}`;
        const err = await caught(call(`http://${info}${host}${CFG_PATH}?${CFG_QUERY}`, `${info}${host}${CFG_PATH}?${CFG_QUERY}`));
        expectRefused(err, host, label);
        expect(srv.connections).toBe(0);
      });
    }

    for (const [name, call] of Object.entries(NODE_FETCH_CALLS)) {
      it(`${name} (node-fetch): a URL with ${shape} is refused naming only host and port, and nothing is sent`, async () => {
        const err = await caught(call(`https://${info}api.example.test:8443${CFG_PATH}`));
        expectRefused(err, 'api.example.test:8443', 'SatGate API host');
        expect(nodeFetch).not.toHaveBeenCalled();
      });
    }

    it(`fetchWithoutRedirects: a URL with ${shape} is refused before the fetch is called`, async () => {
      const fetchImpl = jest.fn(async () => ({ status: 200 }));
      const err = await caught(fetchWithoutRedirects(`https://${info}gw.example:8443${CFG_PATH}?${CFG_QUERY}`, {}, { label: 'test host', fetchImpl }));
      expectRefused(err, 'gw.example:8443', 'test host');
      expect(fetchImpl).not.toHaveBeenCalled();
    });
  }

  it('SatGateClient health: a URL with user information gives false, and nothing is sent', async () => {
    const client = new SatGateClient({ url: `http://${USER}:${PASSWORD}@127.0.0.1:${srv.port}`, token: 'adm-0123' });
    await expect(client.health()).resolves.toBe(false);
    expect(srv.connections).toBe(0);
  });
});

describe('a request URL that cannot be parsed is refused, naming no part of it', () => {
  const BAD = `${USER}:${PASSWORD}@127.0.0.1:99999${CFG_PATH}?${CFG_QUERY}`;

  beforeEach(() => nodeFetch.mockReset());

  for (const [name, { label, call }] of Object.entries(GLOBAL_FETCH_CALLS)) {
    it(`${name}: an unparseable URL gives an error naming no part of it`, async () => {
      expectUnparseable(await caught(call(`http://${BAD}`, BAD)), label);
    });
  }

  for (const [name, call] of Object.entries(NODE_FETCH_CALLS)) {
    it(`${name} (node-fetch): an unparseable URL gives an error naming no part of it`, async () => {
      expectUnparseable(await caught(call(`https://${BAD}`)), 'SatGate API host');
      expect(nodeFetch).not.toHaveBeenCalled();
    });
  }
});

describe('network errors name no path or query of the request URL', () => {
  for (const [name, { call }] of Object.entries(GLOBAL_FETCH_CALLS)) {
    it(`${name}: a refused connection gives an error with no path or query`, async () => {
      const host = `127.0.0.1:${await closedPort()}`;
      const err = await caught(call(`http://${host}${CFG_PATH}?${CFG_QUERY}`, `${host}${CFG_PATH}?${CFG_QUERY}`));
      expectError(err);
      expectNoUrlParts(err, SDK_PATHS);
    });
  }
});

/** An error shaped like node-fetch's FetchError (node-fetch is ESM-only, so jest cannot load it). */
class FetchError extends Error {
  constructor(message: string, public type: string, public code?: string) {
    super(message);
    this.name = 'FetchError';
  }
}

describe('node-fetch errors (private-beta client) name no path or query of the request URL', () => {
  const BASE = `https://api.example.test:8443${CFG_PATH}`;

  beforeEach(() => nodeFetch.mockReset());

  for (const [name, call] of Object.entries(NODE_FETCH_CALLS)) {
    it(`${name}: a network error gives an error with no path or query`, async () => {
      nodeFetch.mockImplementation(async (url: string) => {
        throw new FetchError(`request to ${new URL(url).href} failed, reason: connect ECONNREFUSED 127.0.0.1:8443`, 'system', 'ECONNREFUSED');
      });
      const err = await caught(call(BASE));
      expect(err).toBeInstanceOf(SatGateError);
      expect((err as Error).message).toBe(
        'Request failed: FetchError: request to api.example.test:8443 failed, reason: connect ECONNREFUSED 127.0.0.1:8443',
      );
      expectNoUrlParts(err, SDK_PATHS);
    });

    it(`${name}: a body-read error gives an error with no path or query`, async () => {
      nodeFetch.mockImplementation(async (url: string) => ({
        ok: true,
        status: 200,
        json: async () => {
          throw new FetchError(`Invalid response body while trying to fetch ${new URL(url).href}: Premature close`, 'system', 'ERR_STREAM_PREMATURE_CLOSE');
        },
      }));
      const err = await caught(call(BASE));
      expect(err).toBeInstanceOf(FetchError);
      expect((err as Error).message).toBe('Invalid response body while trying to fetch api.example.test:8443: Premature close');
      expectNoUrlParts(err, SDK_PATHS);
    });

    it(`${name}: a JSON parse error gives an error with no part of the URL`, async () => {
      nodeFetch.mockImplementation(async () => ({ ok: true, status: 200, json: async () => JSON.parse('not json{') }));
      const err = await caught(call(BASE));
      expect(err).toBeInstanceOf(SyntaxError);
      expectNoUrlParts(err, [...SDK_PATHS, 'api.example.test']);
    });
  }
});

describe('JSON parse errors (global fetch) name no part of the request URL', () => {
  const realFetch = globalThis.fetch;

  beforeEach(() => {
    globalThis.fetch = (async () => new Response('not json{', {
      status: 200,
      headers: { 'Content-Type': 'application/json', 'WWW-Authenticate': 'L402 macaroon="m", invoice="lnbc1"' },
    })) as typeof fetch;
  });

  afterEach(() => {
    globalThis.fetch = realFetch;
  });

  for (const [name, { call }] of Object.entries(GLOBAL_FETCH_CALLS)) {
    it(`${name}: an unreadable JSON answer gives an error with no part of the URL`, async () => {
      const host = 'gw.example:8443';
      const err = await caught(call(`https://${host}${CFG_PATH}?${CFG_QUERY}`, `${host}${CFG_PATH}?${CFG_QUERY}`));
      expectError(err);
      expectNoUrlParts(err, [...SDK_PATHS, 'gw.example']);
    });
  }

  it('SatGateAgentClient L402 retry: an unreadable JSON answer gives an error with no part of the URL', async () => {
    let n = 0;
    globalThis.fetch = (async () => {
      n += 1;
      return n === 1
        ? new Response('{}', { status: 402, headers: { 'Content-Type': 'application/json', 'WWW-Authenticate': 'L402 macaroon="m", invoice="lnbc1"' } })
        : new Response('not json{', { status: 200 });
    }) as typeof fetch;
    const wallet = { payInvoice: async () => 'pre', getBalance: async () => 0 };
    const agent = new SatGateAgentClient({ gatewayUrl: `https://gw.example:8443${CFG_PATH}`, token: 'cap-0123', wallet });
    const err = await caught(agent.get(`/x?${CFG_QUERY}`));
    expectError(err, 'SyntaxError');
    expect(n).toBe(2);
    expectNoUrlParts(err, [...SDK_PATHS, 'gw.example']);
  });
});

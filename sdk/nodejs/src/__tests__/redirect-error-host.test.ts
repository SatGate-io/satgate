import { LNDWallet, SatGateAgentClient } from '../agent-client';
import { SatGateClient } from '../client';
import { RedirectRefusedError, RequestUrlRefusedError } from '../errors';
import { fetchWithoutRedirects, requestHost } from '../transport';

/*
 * Redirect errors name only the host and port of the request URL.
 *
 * RedirectRefusedError.host and its message come from the URL the request
 * was sent to, with user information, path and query removed. LNDWallet's
 * host string is configured by the caller and may contain any of these.
 *
 * These tests answer every request with a 302 from a stub fetch and record
 * the URL each request was sent to. A URL with user information is refused
 * before sending (RequestUrlRefusedError, also naming only host and port),
 * so the stub gets nothing.
 */

const USER = 'synthetic-user';
const PASSWORD = 'synthetic-password';
const CFG_PATH = '/cfgpath';
const CFG_QUERY = 'cfgquery=hidden-value';
const FORBIDDEN = [USER, PASSWORD, 'cfgpath', 'cfgquery', 'hidden-value', '@', '?'];
const MACAROON_HEX = '0201036c6e64';

const LND_HOSTS: Record<string, string> = {
  'user information': `${USER}:${PASSWORD}@lnd.example:8080`,
  'a user name only': `${USER}@lnd.example:8080`,
  'a path and query': `lnd.example:8080${CFG_PATH}?${CFG_QUERY}`,
  'user information, a path and a query': `${USER}:${PASSWORD}@lnd.example:8080${CFG_PATH}?${CFG_QUERY}`,
};

function expectHostOnly(err: unknown, host: string, label: string, status = 302): void {
  expect(err).toBeInstanceOf(RedirectRefusedError);
  const e = err as RedirectRefusedError;
  expect(e.host).toBe(host);
  expect(e.statusCode).toBe(status);
  expect(e.message.startsWith(`${label} ${host} answered HTTP ${status};`)).toBe(true);
  expectNoUrlParts(e);
}

function expectRefusedHostOnly(err: unknown, host: string, label: string): void {
  expect(err).toBeInstanceOf(RequestUrlRefusedError);
  const e = err as RequestUrlRefusedError;
  expect(e.host).toBe(host);
  expect(e.message.startsWith(`${label} URL for ${host} has a user name or password in it;`)).toBe(true);
  expectNoUrlParts(e);
}

function expectNoUrlParts(e: RedirectRefusedError | RequestUrlRefusedError): void {
  for (const part of FORBIDDEN) {
    expect(e.message).not.toContain(part);
    expect(e.host).not.toContain(part);
  }
}

async function caught(p: Promise<unknown>): Promise<unknown> {
  try {
    await p;
  } catch (e) {
    return e;
  }
  throw new Error('the call did not reject');
}

describe('redirect errors name only the host and port of the request URL', () => {
  const realFetch = globalThis.fetch;
  let sent: string[];

  beforeEach(() => {
    sent = [];
    globalThis.fetch = (async (url: string | URL | Request) => {
      sent.push(String(url));
      return new Response(null, { status: 302, headers: { Location: '/sink' } });
    }) as typeof fetch;
  });

  afterEach(() => {
    globalThis.fetch = realFetch;
  });

  for (const [shape, host] of Object.entries(LND_HOSTS)) {
    for (const operation of ['getBalance', 'payInvoice'] as const) {
      it(`LNDWallet ${operation}: a configured host with ${shape} gives an error naming only host and port`, async () => {
        const wallet = new LNDWallet({ host, macaroonHex: MACAROON_HEX });
        const err = await caught(operation === 'payInvoice' ? wallet.payInvoice('lnbc1') : wallet.getBalance());
        if (host.includes('@')) {
          expectRefusedHostOnly(err, 'lnd.example:8080', 'LND host');
          expect(sent).toHaveLength(0);
        } else {
          expectHostOnly(err, 'lnd.example:8080', 'LND host');
          expect(sent).toHaveLength(1);
        }
      });
    }
  }

  for (const status of [301, 302, 303, 307, 308]) {
    it(`fetchWithoutRedirects: ${status} names only host and port of a URL with a path and query`, async () => {
      const url = `https://gw.example:8443${CFG_PATH}?${CFG_QUERY}`;
      const err = await caught(fetchWithoutRedirects(url, {}, {
        label: 'test host',
        fetchImpl: async () => ({ status, body: null }),
      }));
      expectHostOnly(err, 'gw.example:8443', 'test host', status);
    });
  }

  it('fetchWithoutRedirects ignores a host option and uses the request URL', async () => {
    const options = { label: 'test host', host: `${USER}:${PASSWORD}@configured`, fetchImpl: async () => ({ status: 302, body: null }) };
    const err = await caught(fetchWithoutRedirects(`https://gw.example:8443${CFG_PATH}?${CFG_QUERY}`, {}, options));
    expectHostOnly(err, 'gw.example:8443', 'test host');
  });

  it('requestHost drops user information, path, query and fragment', () => {
    expect(requestHost(`https://${USER}:${PASSWORD}@gw.example:8443${CFG_PATH}?${CFG_QUERY}#frag`)).toBe('gw.example:8443');
    expect(requestHost('https://[::1]:8080/x?y=1')).toBe('[::1]:8080');
    expect(requestHost(`https://${USER}:${PASSWORD}@LND.Example:443/x?y=1`)).toBe('lnd.example:443');
    expect(requestHost('https://lnd.example/x')).toBe('lnd.example');
    expect(requestHost('not a url')).toBe('an unreadable host');
  });

  it('SatGateAgentClient: a gateway URL with a path and query gives an error naming only host and port', async () => {
    const agent = new SatGateAgentClient({ gatewayUrl: `https://gw.example:8443${CFG_PATH}`, token: 'cap-0123' });
    const err = await caught(agent.get(`/x?${CFG_QUERY}`));
    expectHostOnly(err, 'gw.example:8443', 'SatGate gateway');
  });

  it('SatGateClient: a URL with a path gives an error naming only host and port', async () => {
    const client = new SatGateClient({ url: `https://gw.example:8443${CFG_PATH}`, token: 'adm-0123' });
    const err = await caught(client.tokens.mint());
    expectHostOnly(err, 'gw.example:8443', 'SatGate gateway');
  });
});

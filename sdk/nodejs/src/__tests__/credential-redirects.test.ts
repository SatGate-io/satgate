import { execFileSync } from 'child_process';
import * as fs from 'fs';
import * as http from 'http';
import * as https from 'https';
import { AddressInfo } from 'net';
import * as os from 'os';
import * as path from 'path';
import tlsModule from 'tls';

import { AlbyWallet, LightningWallet, SatGateAgentClient } from '../agent-client';
import { SatGateClient } from '../client';
import { RedirectRefusedError } from '../errors';
import { SatGate } from '../private-beta';

// The private-beta client sends through node-fetch. Here node-fetch is the
// global fetch, so its redirect option is honoured the same way.
jest.mock('node-fetch', () => ({
  __esModule: true,
  default: (url: string, init: RequestInit) => globalThis.fetch(url, init),
}));

/*
 * No SDK request that carries a credential follows a redirect.
 *
 * Each credential-bearing request is sent to a local HTTPS origin that
 * answers 301, 302, 303, 307 or 308 with a Location on another host (HTTPS on
 * "localhost", trusted by the same certificate, so only the redirect rule
 * keeps the request from it) or on plain HTTP. Every call must reject with
 * RedirectRefusedError naming the status and the origin host only; the
 * origin gets the request once and the target gets nothing.
 */

const STATUSES = [301, 302, 303, 307, 308];
const TARGETS = ['another host', 'plain HTTP'] as const;
type Target = typeof TARGETS[number];

const ADMIN = 'adm-0123456789abcdef';
const CAP = 'cap-0123456789abcdef';
const API_KEY = 'key-0123456789abcdef';
const ALBY = 'alby-0123456789abcdef';
const SECRETS = [ADMIN, CAP, API_KEY, ALBY];

type Seen = { method: string; path: string; headers: http.IncomingHttpHeaders; body: string };
type Answer = (seen: Seen) => { status: number; headers?: Record<string, string>; body?: string } | undefined;

interface Server {
  seen: Seen[];
  port: number;
  url(host?: string): string;
  close(): Promise<void>;
}

let tls: { cert: Buffer; key: Buffer } | undefined;

function haveOpenssl(): boolean {
  try {
    execFileSync('openssl', ['version'], { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

function makeCert(): { cert: Buffer; key: Buffer } {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'cred-redirect-'));
  const cert = path.join(dir, 'cert.pem');
  const key = path.join(dir, 'key.pem');
  execFileSync('openssl', [
    'req', '-x509', '-newkey', 'ec', '-pkeyopt', 'ec_paramgen_curve:prime256v1',
    '-nodes', '-days', '1', '-subj', '/CN=127.0.0.1',
    '-addext', 'subjectAltName=IP:127.0.0.1,DNS:localhost',
    '-keyout', key, '-out', cert,
  ], { stdio: 'ignore' });
  const out = { cert: fs.readFileSync(cert), key: fs.readFileSync(key) };
  fs.rmSync(dir, { recursive: true, force: true });
  return out;
}

async function server(secure: boolean, answer?: Answer): Promise<Server> {
  const seen: Seen[] = [];
  const handler = (req: http.IncomingMessage, res: http.ServerResponse) => {
    const chunks: Buffer[] = [];
    req.on('data', (c: Buffer) => chunks.push(c));
    req.on('end', () => {
      const s: Seen = { method: req.method || '', path: req.url || '', headers: req.headers, body: Buffer.concat(chunks).toString() };
      seen.push(s);
      const out = answer?.(s) ?? { status: 200, headers: { 'Content-Type': 'application/json' }, body: '{}' };
      const body = out.body ?? '';
      res.writeHead(out.status, { ...out.headers, 'Content-Length': String(Buffer.byteLength(body)) });
      res.end(body);
    });
  };
  const srv = secure ? https.createServer({ cert: tls!.cert, key: tls!.key }, handler) : http.createServer(handler);
  await new Promise<void>((resolve) => srv.listen(0, '127.0.0.1', resolve));
  const port = (srv.address() as AddressInfo).port;
  const scheme = secure ? 'https' : 'http';
  return {
    seen,
    port,
    url: (host = '127.0.0.1') => `${scheme}://${host}:${port}`,
    close: () => new Promise<void>((resolve) => {
      srv.closeAllConnections?.();
      srv.close(() => resolve());
    }),
  };
}

function carries(s: Seen, secret: string): boolean {
  return Object.values(s.headers).some((v) => String(v).includes(secret)) || s.body.includes(secret);
}

const redirectAll = (status: number, location: string): Answer => () => ({ status, headers: { Location: location } });

const maybe = haveOpenssl() ? describe : describe.skip;

maybe('no credential-bearing SDK request follows a redirect', () => {
  const realConnect = tlsModule.connect;
  const open: Server[] = [];

  beforeAll(() => {
    tls = makeCert();
    // Node's fetch takes no CA option. For this file only, TLS connections
    // trust the test certificate (and nothing else); verification stays on.
    const ca = tls.cert;
    (tlsModule as { connect: unknown }).connect = (...args: unknown[]) => {
      const options = args.find((a) => typeof a === 'object' && a !== null) as tlsModule.ConnectionOptions | undefined;
      if (options && options.ca === undefined) {
        options.ca = [ca];
      }
      return (realConnect as (...a: unknown[]) => tlsModule.TLSSocket)(...args);
    };
  });

  afterAll(() => {
    (tlsModule as { connect: unknown }).connect = realConnect;
  });

  afterEach(async () => {
    await Promise.all(open.splice(0).map((s) => s.close()));
  });

  async function serve(secure: boolean, answer?: Answer): Promise<Server> {
    const s = await server(secure, answer);
    open.push(s);
    return s;
  }

  async function sinkFor(target: Target): Promise<[Server, string]> {
    if (target === 'another host') {
      const s = await serve(true);
      return [s, `${s.url('localhost')}/sink`];
    }
    const s = await serve(false);
    return [s, `${s.url()}/sink`];
  }

  async function expectRefused(call: () => Promise<unknown>, origin: Server, sink: Server, status: number, originRequests = 1): Promise<Seen[]> {
    let caught: unknown;
    try {
      await call();
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(RedirectRefusedError);
    const err = caught as RedirectRefusedError;
    expect(err.statusCode).toBe(status);
    expect(err.host).toBe(`127.0.0.1:${origin.port}`);
    expect(err.message).toContain(`127.0.0.1:${origin.port} answered HTTP ${status};`);
    // Only the status and the origin host are named.
    expect(err.message).not.toContain(String(sink.port));
    expect(err.message).not.toContain('localhost');
    expect(err.message).not.toContain('/sink');
    for (const secret of SECRETS) {
      expect(err.message).not.toContain(secret);
    }
    expect(sink.seen).toEqual([]);
    expect(origin.seen).toHaveLength(originRequests);
    return origin.seen;
  }

  const agentCalls: Record<string, [Record<string, unknown>, (c: SatGateAgentClient) => Promise<unknown>, string]> = {
    'agent mint (admin token)': [{ adminToken: ADMIN }, (c) => c.get('/api/data'), ADMIN],
    'agent get (capability token)': [{ token: CAP }, (c) => c.get('/api/data'), CAP],
    'agent post (capability token)': [{ token: CAP }, (c) => c.post('/api/data', { a: 1 }), CAP],
    'agent put (capability token)': [{ token: CAP }, (c) => c.put('/api/data', { a: 1 }), CAP],
    'agent delete (capability token)': [{ token: CAP }, (c) => c.delete('/api/data'), CAP],
    'agent ping (capability token)': [{ token: CAP }, (c) => c.ping(), CAP],
    'agent validateToken (token in header and body)': [{ token: CAP }, (c) => c.validateToken(), CAP],
    'agent delegate (token in body)': [{ token: CAP }, (c) => c.delegate({ caveats: ['scope = api:read'] }), CAP],
  };

  const adminCalls: Record<string, [(c: SatGateClient) => Promise<unknown>, string | undefined]> = {
    'admin tokens.mint (admin token)': [(c) => c.tokens.mint({ scope: 'api:read' }), ADMIN],
    'admin tokens.validate (token in body)': [(c) => c.tokens.validate(CAP), CAP],
    'admin tokens.delegate (token in body)': [(c) => c.tokens.delegate({ parentToken: CAP, caveats: [] }), CAP],
    'admin governance.ban (admin token)': [(c) => c.governance.ban('sig', 'r'), ADMIN],
    'admin governance.reset (admin token)': [(c) => c.governance.reset(), ADMIN],
    'admin ping (capability token)': [(c) => c.ping(CAP), CAP],
    'admin governance.getGraph (same transport)': [(c) => c.governance.getGraph(), undefined],
  };

  const betaCalls: Record<string, (c: SatGate) => Promise<unknown>> = {
    'private-beta issue (API key)': (c) => c.issue({ task: 't' } as never),
    'private-beta pay (API key)': (c) => c.pay({ upstream: 'u', capability: 'c' }),
    'private-beta verify (API key)': (c) => c.verify({ receipt_id: 'r' }),
  };

  const albyCalls: Record<string, (w: AlbyWallet) => Promise<unknown>> = {
    'Alby payInvoice (access token)': (w) => w.payInvoice('lnbc1'),
    'Alby getBalance (access token)': (w) => w.getBalance(),
  };

  for (const status of STATUSES) {
    for (const target of TARGETS) {
      for (const [name, [opts, call, secret]] of Object.entries(agentCalls)) {
        it(`${name}: ${status} to ${target} is refused`, async () => {
          const [sink, location] = await sinkFor(target);
          const origin = await serve(true, redirectAll(status, location));
          const client = new SatGateAgentClient({ gatewayUrl: origin.url(), ...opts });
          const seen = await expectRefused(() => call(client), origin, sink, status);
          expect(carries(seen[0], secret)).toBe(true);
        });
      }

      it(`agent L402 retry (L402 token): ${status} to ${target} is refused`, async () => {
        const [sink, location] = await sinkFor(target);
        const origin = await serve(true, (s): ReturnType<Answer> => {
          const auth = String(s.headers.authorization || '');
          if (auth.startsWith('L402 ')) {
            return { status, headers: { Location: location } };
          }
          return { status: 402, headers: { 'WWW-Authenticate': 'L402 macaroon="mac-0123456789abcdef", invoice="lnbc1"' } };
        });
        const wallet: LightningWallet = { payInvoice: async () => 'pre-0123456789abcdef', getBalance: async () => 0 };
        const client = new SatGateAgentClient({ gatewayUrl: origin.url(), token: CAP, wallet });
        const seen = await expectRefused(() => client.get('/paid'), origin, sink, status, 2);
        expect(seen[1].headers.authorization).toBe('L402 mac-0123456789abcdef:pre-0123456789abcdef');
      });

      it(`agent refresh retry after 401 (fresh token): ${status} to ${target} is refused`, async () => {
        const [sink, location] = await sinkFor(target);
        let data = 0;
        const origin = await serve(true, (s): ReturnType<Answer> => {
          if (s.path === '/api/capability/mint') {
            return { status: 200, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ token: `${CAP}-new` }) };
          }
          data += 1;
          return data === 1 ? { status: 401 } : { status, headers: { Location: location } };
        });
        const client = new SatGateAgentClient({ gatewayUrl: origin.url(), token: CAP, adminToken: ADMIN });
        const seen = await expectRefused(() => client.get('/api/data'), origin, sink, status, 3);
        expect(seen[1].path).toBe('/api/capability/mint');
        expect(seen[2].headers.authorization).toBe(`Bearer ${CAP}-new`);
      });

      for (const [name, [call, secret]] of Object.entries(adminCalls)) {
        it(`${name}: ${status} to ${target} is refused`, async () => {
          const [sink, location] = await sinkFor(target);
          const origin = await serve(true, redirectAll(status, location));
          const client = new SatGateClient({ url: origin.url(), token: ADMIN });
          const seen = await expectRefused(() => call(client), origin, sink, status);
          if (secret) {
            expect(carries(seen[0], secret)).toBe(true);
          }
        });
      }

      it(`admin health: ${status} to ${target} reports unhealthy and sends nothing on`, async () => {
        const [sink, location] = await sinkFor(target);
        const origin = await serve(true, redirectAll(status, location));
        const client = new SatGateClient({ url: origin.url(), token: ADMIN });
        await expect(client.health()).resolves.toBe(false);
        expect(sink.seen).toEqual([]);
        expect(origin.seen).toHaveLength(1);
      });

      for (const [name, call] of Object.entries(betaCalls)) {
        it(`${name}: ${status} to ${target} is refused`, async () => {
          const [sink, location] = await sinkFor(target);
          const origin = await serve(true, redirectAll(status, location));
          const client = new SatGate({ apiKey: API_KEY, baseUrl: origin.url() });
          const seen = await expectRefused(() => call(client), origin, sink, status);
          expect(seen[0].headers.authorization).toBe(`Bearer ${API_KEY}`);
        });
      }

      for (const [name, call] of Object.entries(albyCalls)) {
        it(`${name}: ${status} to ${target} is refused`, async () => {
          const [sink, location] = await sinkFor(target);
          const origin = await serve(true, redirectAll(status, location));
          const wallet = new AlbyWallet({ accessToken: ALBY });
          (wallet as unknown as { apiBase: string }).apiBase = origin.url();
          const seen = await expectRefused(() => call(wallet), origin, sink, status);
          expect(seen[0].headers.authorization).toBe(`Bearer ${ALBY}`);
        });
      }
    }
  }

  it('a redirect option from the caller is overridden', async () => {
    const [sink, location] = await sinkFor('another host');
    const origin = await serve(true, redirectAll(307, location));
    const client = new SatGateAgentClient({ gatewayUrl: origin.url(), token: CAP });
    await expectRefused(() => client.get('/api/data', { redirect: 'follow' }), origin, sink, 307);
  });

  // Reviewer case: the admin token and capability tokens (in a header or in
  // the JSON body) never reach another host or plain HTTP after a 302 or a
  // 307, from the agent client or the admin client.
  for (const status of [302, 307]) {
    it(`reviewer case: admin and capability tokens never reach the target of a ${status}`, async () => {
      for (const target of TARGETS) {
        const [sink, location] = await sinkFor(target);
        const origin = await serve(true, redirectAll(status, location));
        const calls = [
          () => new SatGateAgentClient({ gatewayUrl: origin.url(), adminToken: ADMIN }).get('/api/data'),
          () => new SatGateAgentClient({ gatewayUrl: origin.url(), token: CAP }).get('/api/data'),
          () => new SatGateAgentClient({ gatewayUrl: origin.url(), token: CAP }).delegate(),
          () => new SatGateClient({ url: origin.url(), token: ADMIN }).tokens.mint(),
          () => new SatGateClient({ url: origin.url(), token: ADMIN }).tokens.validate(CAP),
        ];
        for (const call of calls) {
          await expect(call()).rejects.toBeInstanceOf(RedirectRefusedError);
        }
        expect(sink.seen.filter((s) => SECRETS.some((x) => carries(s, x)))).toEqual([]);
        expect(sink.seen).toEqual([]);
      }
    });
  }

  // Reviewer case: every 3xx answer is refused, not only the five common
  // redirect codes, and the error does not name the Location.
  for (const status of [300, 304, 305, 306, 399]) {
    it(`reviewer case: HTTP ${status} is refused naming status and host only`, async () => {
      const origin = await serve(true, () => ({ status, headers: { Location: 'http://127.0.0.1:9/x' } }));
      const client = new SatGateAgentClient({ gatewayUrl: origin.url(), token: CAP });
      let caught: unknown;
      try {
        await client.get('/api/data');
      } catch (e) {
        caught = e;
      }
      expect(caught).toBeInstanceOf(RedirectRefusedError);
      expect((caught as RedirectRefusedError).statusCode).toBe(status);
      expect((caught as Error).message).not.toContain('127.0.0.1:9');
    });
  }
});

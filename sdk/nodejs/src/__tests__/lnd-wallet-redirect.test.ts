import { execFileSync } from 'child_process';
import * as fs from 'fs';
import * as http from 'http';
import * as https from 'https';
import { AddressInfo } from 'net';
import * as os from 'os';
import * as path from 'path';
import tlsModule from 'tls';

import { LNDWallet } from '../agent-client';
import { SatGateError } from '../errors';

const MACAROON_HEX = '0201036c6e64';
const STATUSES = [301, 302, 303, 307, 308];
const OPERATIONS = ['getBalance', 'payInvoice'] as const;

type Seen = { path: string; macaroon: string | undefined };

interface Recorder {
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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'lnd-redirect-'));
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

async function recorder(opts: { secure: boolean; status?: number; location?: string }): Promise<Recorder> {
  const seen: Seen[] = [];
  const handler = (req: http.IncomingMessage, res: http.ServerResponse) => {
    req.resume();
    req.on('end', () => {
      const header = req.headers['grpc-metadata-macaroon'];
      seen.push({ path: req.url || '', macaroon: Array.isArray(header) ? header[0] : header });
      if (opts.location !== undefined) {
        res.writeHead(opts.status || 302, { Location: opts.location, 'Content-Length': '0' });
        res.end();
        return;
      }
      const body = JSON.stringify({ balance: '42', payment_preimage: 'ab' });
      res.writeHead(200, { 'Content-Type': 'application/json', 'Content-Length': String(body.length) });
      res.end(body);
    });
  };
  const server = opts.secure ? https.createServer({ cert: tls!.cert, key: tls!.key }, handler) : http.createServer(handler);
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
  const port = (server.address() as AddressInfo).port;
  const scheme = opts.secure ? 'https' : 'http';
  return {
    seen,
    port,
    url: (host = '127.0.0.1') => `${scheme}://${host}:${port}/sink`,
    close: () => new Promise<void>((resolve) => {
      server.closeAllConnections?.();
      server.close(() => resolve());
    }),
  };
}

const maybe = haveOpenssl() ? describe : describe.skip;

maybe('LNDWallet never follows a redirect with the macaroon header', () => {
  const realConnect = tlsModule.connect;
  const open: Recorder[] = [];

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
    await Promise.all(open.splice(0).map((r) => r.close()));
  });

  async function serve(opts: { secure: boolean; status?: number; location?: string }): Promise<Recorder> {
    const r = await recorder(opts);
    open.push(r);
    return r;
  }

  function call(wallet: LNDWallet, operation: typeof OPERATIONS[number]): Promise<unknown> {
    return operation === 'payInvoice' ? wallet.payInvoice('lnbc1') : wallet.getBalance();
  }

  async function expectRefused(operation: typeof OPERATIONS[number], status: number, target: string, sink: Recorder) {
    const origin = await serve({ secure: true, status, location: target });
    const host = `127.0.0.1:${origin.port}`;
    const wallet = new LNDWallet({ host, macaroonHex: MACAROON_HEX });
    let caught: unknown;
    try {
      await call(wallet, operation);
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(SatGateError);
    const message = (caught as Error).message;
    expect(message).toContain(`HTTP ${status}`);
    expect(message).toContain(host);
    expect(origin.seen.map((s) => s.macaroon)).toEqual([MACAROON_HEX]);
    expect(sink.seen).toEqual([]);
    return message;
  }

  for (const operation of OPERATIONS) {
    for (const status of STATUSES) {
      it(`${operation}: ${status} to a plain-HTTP URL is refused`, async () => {
        const sink = await serve({ secure: false });
        const message = await expectRefused(operation, status, sink.url(), sink);
        expect(message).not.toContain(String(sink.port));
      });

      it(`${operation}: ${status} to another host is refused`, async () => {
        const sink = await serve({ secure: true });
        const message = await expectRefused(operation, status, sink.url('localhost'), sink);
        expect(message).not.toContain('localhost');
        expect(message).not.toContain(String(sink.port));
      });
    }
  }

  for (const operation of OPERATIONS) {
    it(`${operation}: a configured host with a path and query gives an error naming only host and port`, async () => {
      const origin = await serve({ secure: true, status: 302, location: '/sink' });
      const host = `127.0.0.1:${origin.port}`;
      const wallet = new LNDWallet({ host: `${host}/cfgpath?cfgquery=hidden-value`, macaroonHex: MACAROON_HEX });
      let caught: unknown;
      try {
        await call(wallet, operation);
      } catch (e) {
        caught = e;
      }
      expect(caught).toBeInstanceOf(SatGateError);
      const err = caught as SatGateError & { host?: string };
      expect(err.host).toBe(host);
      expect(err.message.startsWith(`LND host ${host} answered HTTP 302;`)).toBe(true);
      for (const part of ['cfgpath', 'cfgquery', 'hidden-value', '?']) {
        expect(err.message).not.toContain(part);
      }
      expect(origin.seen).toHaveLength(1);
      expect(origin.seen[0].path.startsWith('/cfgpath?cfgquery=hidden-value')).toBe(true);
    });
  }

  it('the error names only the status and the LND host', async () => {
    const origin = await serve({ secure: true, status: 302, location: 'http://user:pw@127.0.0.1:9/path?q=hidden' });
    const host = `127.0.0.1:${origin.port}`;
    const wallet = new LNDWallet({ host, macaroonHex: MACAROON_HEX });
    await expect(wallet.getBalance()).rejects.toThrow(`LND host ${host} answered HTTP 302;`);
    try {
      await wallet.getBalance();
    } catch (e) {
      for (const part of ['127.0.0.1:9', 'user', 'pw', '/path', 'hidden']) {
        expect((e as Error).message).not.toContain(part);
      }
    }
  });
});

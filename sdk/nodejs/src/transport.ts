/**
 * The one transport for every HTTP request the SDK sends.
 *
 * Every request (including those with a credential: an Authorization header,
 * X-Admin-Token, an LND macaroon header, or a token in the JSON body) goes
 * through fetchWithoutRedirects. It never follows a redirect: the request is
 * sent once, with redirect: 'manual', to the URL the caller configured, and
 * any 3xx answer throws RedirectRefusedError naming only the status and the
 * host and port of the request URL (never its user information, path or
 * query).
 */

import { RedirectRefusedError } from './errors';

/**
 * host[:port] of url, without user info, path or query. A port written in
 * the URL is kept even when it is the scheme's default (URL drops it).
 */
export function requestHost(url: string): string {
  try {
    const parsed = new URL(url);
    if (!parsed.hostname) return 'an unreadable host';
    let port = parsed.port;
    if (!port) {
      const authority = /^[a-z][a-z0-9+.-]*:\/\/([^/?#]*)/i.exec(url)?.[1] ?? '';
      const hostPart = authority.slice(authority.lastIndexOf('@') + 1);
      const written = /:(\d+)$/.exec(hostPart);
      if (written) port = String(Number(written[1]));
    }
    return port ? `${parsed.hostname}:${port}` : parsed.hostname;
  } catch {
    return 'an unreadable host';
  }
}

/** The parts of a fetch Response this module reads (global fetch or node-fetch). */
interface MinimalResponse {
  status: number;
  type?: string;
  body?: unknown;
}

export interface NoRedirectOptions<R> {
  /** Names the kind of host in the error, e.g. "SatGate gateway". */
  label?: string;
  /** The fetch to use; by default the global fetch. */
  fetchImpl?: (url: string, init: RequestInit) => Promise<R>;
}

async function discard(body: unknown): Promise<void> {
  const b = body as { cancel?: () => Promise<void>; destroy?: () => void } | null | undefined;
  if (!b) return;
  if (typeof b.cancel === 'function') {
    await b.cancel().catch(() => undefined);
  } else if (typeof b.destroy === 'function') {
    b.destroy();
  }
}

/**
 * Send one request and never follow a redirect. Any redirect mode the caller
 * passed is replaced by 'manual'. A 3xx answer (or an opaque redirect)
 * throws RedirectRefusedError naming the status and requestHost(url) (the
 * host and port of the request URL, without user information, path or
 * query), so the request is never sent to the redirect's target. Network
 * errors are thrown unchanged.
 */
export async function fetchWithoutRedirects<R extends MinimalResponse = Response>(
  url: string,
  init: RequestInit = {},
  options: NoRedirectOptions<R> = {},
): Promise<R> {
  const send = options.fetchImpl ?? (fetch as unknown as (url: string, init: RequestInit) => Promise<R>);
  const response = await send(url, { ...init, redirect: 'manual' });
  if (response.type === 'opaqueredirect' || (response.status >= 300 && response.status < 400)) {
    await discard(response.body);
    throw new RedirectRefusedError(response.status, requestHost(url), options.label);
  }
  return response;
}

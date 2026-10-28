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
 *
 * The URL is checked before anything is sent: a URL with user information
 * (a user name or password), or one that cannot be parsed, throws
 * RequestUrlRefusedError and nothing is sent. An error from the fetch itself
 * is thrown as the same error object, with the request URL in its message,
 * stack and causes replaced by the host and port.
 */

import { RedirectRefusedError, RequestUrlRefusedError } from './errors';

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

/**
 * Refuse url before anything is sent: throws RequestUrlRefusedError when it
 * cannot be parsed, or when it has a user name or password. The error names
 * only the host and port.
 */
export function checkRequestUrl(url: string, label?: string): void {
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch {
    throw new RequestUrlRefusedError('an unreadable host', label, 'unparseable');
  }
  if (parsed.username || parsed.password) {
    throw new RequestUrlRefusedError(requestHost(url), label, 'user-info');
  }
}

/** The ways a fetch implementation writes url in an error message, longest first. */
function urlForms(url: string): string[] {
  const forms = new Set<string>([url]);
  try {
    const parsed = new URL(url);
    forms.add(parsed.href);
    forms.add(parsed.href.replace(/#.*$/, ''));
    forms.add(`${parsed.origin}${parsed.pathname}${parsed.search}`);
  } catch {
    // Only the URL as written.
  }
  return [...forms].filter((form) => form.length > 0).sort((a, b) => b.length - a.length);
}

/**
 * Replace url (as written, or as a fetch implementation writes it) with its
 * host and port in the message and stack of err and of its causes. The same
 * error object is returned, so its type and other properties are unchanged.
 */
export function withoutUrl<E>(err: E, url: string): E {
  const forms = urlForms(url);
  const host = requestHost(url);
  const scrub = (text: string): string => forms.reduce((out, form) => out.split(form).join(host), text);
  let current: unknown = err;
  for (let depth = 0; depth < 5 && current !== null && typeof current === 'object'; depth++) {
    const record = current as Record<string, unknown>;
    for (const key of ['message', 'stack']) {
      const value = record[key];
      if (typeof value !== 'string') continue;
      const clean = scrub(value);
      if (clean === value) continue;
      try {
        Object.defineProperty(record, key, { value: clean, writable: true, configurable: true, enumerable: false });
      } catch {
        // A frozen error keeps its text.
      }
    }
    current = record.cause;
  }
  return err;
}

/**
 * Run work (for example reading a response body) and throw any error it
 * raises through withoutUrl, so the error names no part of url but its host
 * and port.
 */
export async function withoutUrlInErrors<T>(url: string, work: () => Promise<T>): Promise<T> {
  try {
    return await work();
  } catch (err) {
    throw withoutUrl(err, url);
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
 * Send one request and never follow a redirect. The URL is checked first
 * (checkRequestUrl): a URL with user information, or one that cannot be
 * parsed, throws RequestUrlRefusedError and nothing is sent. Any redirect
 * mode the caller passed is replaced by 'manual'. A 3xx answer (or an opaque
 * redirect) throws RedirectRefusedError naming the status and
 * requestHost(url) (the host and port of the request URL, without user
 * information, path or query), so the request is never sent to the
 * redirect's target. A network error is thrown as the same error object,
 * with the request URL in its text replaced by the host and port
 * (withoutUrl).
 */
export async function fetchWithoutRedirects<R extends MinimalResponse = Response>(
  url: string,
  init: RequestInit = {},
  options: NoRedirectOptions<R> = {},
): Promise<R> {
  checkRequestUrl(url, options.label);
  const send = options.fetchImpl ?? (fetch as unknown as (url: string, init: RequestInit) => Promise<R>);
  const response = await withoutUrlInErrors(url, () => send(url, { ...init, redirect: 'manual' }));
  if (response.type === 'opaqueredirect' || (response.status >= 300 && response.status < 400)) {
    await discard(response.body);
    throw new RedirectRefusedError(response.status, requestHost(url), options.label);
  }
  return response;
}

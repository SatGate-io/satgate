/**
 * The one transport for every HTTP request the SDK sends, and the redaction
 * of every error message the SDK builds or throws about one.
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
 * RequestUrlRefusedError and nothing is sent.
 *
 * Messages are redacted by redact(), the one redaction function of the SDK.
 * A Redaction records what one request carried (requestRedaction); the
 * transport attaches it to the response and to any error it throws, so that
 * every SDK site that builds a message, quotes a response body or quotes
 * another error passes the text through redact(text, responseOrError).
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

/** Replacements written into a redacted message. */
export const PATH_AND_QUERY_REMOVED = '[path and query removed]';
export const PATH_REMOVED = '[path removed]';
export const REDACTED = '[redacted]';
/** What replaces the response text a JSON parse error quotes. */
export const RESPONSE_TEXT_REMOVED = '(response text removed)';

/**
 * A credential, query name or query value shorter than this is not replaced
 * on its own (it would match ordinary text such as a status code); it is
 * still removed as part of the URL, the query string or a name=value pair.
 */
export const MIN_REDACTED_LENGTH = 3;

/** Request headers whose values are credentials. */
const CREDENTIAL_HEADER = /auth|token|key|secret|macaroon|cookie|password/i;
/**
 * JSON body fields whose string values are credentials: the capability
 * token, a parent token, a macaroon, a preimage, a receipt or capability, an
 * API key.
 */
const CREDENTIAL_FIELD =
  /token|macaroon|preimage|secret|password|passwd|key|credential|capability|auth|signature|receipt/i;
/** Characters that continue a query name; a name is replaced only as a whole word. */
const NAME_CHARS = 'A-Za-z0-9_.~%-';

function safeDecode(text: string): string {
  try {
    return decodeURIComponent(text);
  } catch {
    return text;
  }
}

/** value as written and as it is commonly escaped in a message. */
function forms(value: string): Set<string> {
  const out = new Set<string>([value, value.trim()]);
  for (const form of [...out]) {
    out.add(encodeURIComponent(form));
    out.add(encodeURIComponent(form).replace(/%20/g, '+'));
    out.add(JSON.stringify(form).slice(1, -1));
    out.add(safeDecode(form));
    out.add(safeDecode(form.replace(/\+/g, ' ')));
  }
  out.delete('');
  return out;
}

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\/-]/g, '\\$&');
}

function headerCredentials(value: string): string[] {
  const out = [value];
  const match = /^\s*([A-Za-z0-9_-]+)\s+([\s\S]+?)\s*$/.exec(value);
  if (match) {
    const scheme = match[1].toLowerCase();
    const rest = match[2];
    out.push(rest);
    if (scheme === 'basic') {
      try {
        const decoded = typeof atob === 'function' ? atob(rest) : '';
        const colon = decoded.indexOf(':');
        if (colon >= 0) out.push(decoded.slice(0, colon), decoded.slice(colon + 1));
      } catch {
        // Not base64: the value and the token above are enough.
      }
    }
    if (scheme === 'l402' || scheme === 'lsat') out.push(...rest.split(':'));
  }
  for (const part of value.split(';')) {
    const eq = part.indexOf('=');
    if (eq >= 0) out.push(part.slice(eq + 1));
  }
  return out;
}

function bodyCredentials(value: unknown, out: string[]): void {
  if (Array.isArray(value)) {
    for (const item of value) bodyCredentials(item, out);
    return;
  }
  if (value && typeof value === 'object') {
    for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
      if (CREDENTIAL_FIELD.test(key)) {
        if (typeof item === 'string') {
          out.push(item);
          continue;
        }
        if (Array.isArray(item) && item.every((i) => typeof i === 'string')) {
          out.push(...(item as string[]));
          continue;
        }
      }
      bodyCredentials(item, out);
    }
  }
}

/**
 * What one request carried that its error messages must not show: the URL,
 * its user information, path and query, and the credentials in its headers,
 * query and JSON body.
 */
export class Redaction {
  private readonly exact = new Map<string, string>();
  private readonly words = new Map<string, string>();
  private pattern?: RegExp;

  private add(text: string, replacement: string, minimum = 1): void {
    if (text && text.length >= minimum && !this.exact.has(text)) {
      this.exact.set(text, replacement);
      this.pattern = undefined;
    }
  }

  /** The URL, its user information, path and query. */
  addUrl(url: string): void {
    if (typeof url !== 'string' || !url) return;
    const host = requestHost(url);
    this.add(url, host);
    this.add(url.replace(/#.*$/, ''), host);
    let parsed: URL;
    try {
      parsed = new URL(url);
    } catch {
      return;
    }
    this.add(parsed.href, host);
    this.add(parsed.href.replace(/#.*$/, ''), host);
    this.add(`${parsed.origin}${parsed.pathname}${parsed.search}`, host);
    if (parsed.username || parsed.password) {
      const userinfo = parsed.password ? `${parsed.username}:${parsed.password}` : parsed.username;
      this.add(`${userinfo}@`, '');
      this.addCredential(safeDecode(parsed.username));
      this.addCredential(safeDecode(parsed.password));
    }
    const requestUri = `${parsed.pathname}${parsed.search}`;
    if (requestUri !== '' && requestUri !== '/') {
      for (const form of [requestUri, safeDecode(requestUri)]) this.add(form, PATH_AND_QUERY_REMOVED);
    }
    for (const path of new Set([parsed.pathname, safeDecode(parsed.pathname)])) {
      // The path and every tail of it from a "/" (the server may name the
      // path without a prefix it was routed under).
      for (let i = 0; i < path.length; i++) {
        if (path[i] === '/' && path.length - i > 1) this.add(path.slice(i), PATH_REMOVED);
      }
    }
    const query = parsed.search.replace(/^\?/, '');
    if (query) {
      for (const form of [query, safeDecode(query), safeDecode(query.replace(/\+/g, ' '))]) this.add(form, REDACTED);
      for (const raw of query.split('&')) {
        this.add(raw, REDACTED);
        this.add(safeDecode(raw.replace(/\+/g, ' ')), REDACTED);
      }
      for (const [name, value] of parsed.searchParams) {
        this.add(`${name}=${value}`, REDACTED);
        this.addCredential(value);
        if (name.length >= MIN_REDACTED_LENGTH) {
          for (const form of forms(name)) {
            if (form.length >= MIN_REDACTED_LENGTH && !this.words.has(form)) {
              this.words.set(form, REDACTED);
              this.pattern = undefined;
            }
          }
        }
      }
    }
  }

  /** A credential, as written and as commonly escaped. */
  addCredential(value: unknown): void {
    if (typeof value !== 'string') return;
    for (const form of forms(value)) this.add(form, REDACTED, MIN_REDACTED_LENGTH);
  }

  /** The values of the credential headers among headers (Headers, pairs or a record). */
  addHeaders(headers: unknown): void {
    if (!headers) return;
    let pairs: [string, unknown][] = [];
    try {
      if (typeof (headers as Headers).entries === 'function' && !Array.isArray(headers)) {
        pairs = [...(headers as Headers).entries()];
      } else if (Array.isArray(headers)) {
        pairs = headers.filter((p) => Array.isArray(p) && p.length >= 2).map((p) => [String(p[0]), p[1]]);
      } else if (typeof headers === 'object') {
        pairs = Object.entries(headers as Record<string, unknown>);
      }
    } catch {
      return;
    }
    for (const [name, value] of pairs) {
      if (!CREDENTIAL_HEADER.test(name)) continue;
      for (const v of Array.isArray(value) ? value : [value]) {
        if (typeof v === 'string') for (const credential of headerCredentials(v)) this.addCredential(credential);
      }
    }
  }

  /** The credential fields of a JSON body (a string is parsed first). */
  addBody(body: unknown): void {
    let value = body;
    if (typeof body === 'string') {
      try {
        value = JSON.parse(body);
      } catch {
        return;
      }
    }
    const out: string[] = [];
    bodyCredentials(value, out);
    for (const credential of out) this.addCredential(credential);
  }

  merge(other: Redaction): void {
    for (const [text, replacement] of other.exact) if (!this.exact.has(text)) this.exact.set(text, replacement);
    for (const [text, replacement] of other.words) if (!this.words.has(text)) this.words.set(text, replacement);
    this.pattern = undefined;
  }

  apply(text: string): string {
    if (typeof text !== 'string' || (this.exact.size === 0 && this.words.size === 0)) return text;
    if (!this.pattern) {
      const alternatives: [string, string][] = [...this.exact.keys()].map((t) => [t, escapeRegExp(t)]);
      for (const t of this.words.keys()) {
        if (!this.exact.has(t)) alternatives.push([t, `(?<![${NAME_CHARS}])${escapeRegExp(t)}(?![${NAME_CHARS}])`]);
      }
      alternatives.sort((a, b) => b[0].length - a[0].length);
      this.pattern = new RegExp(alternatives.map(([, regex]) => regex).join('|'), 'g');
    }
    return text.replace(this.pattern, (match) => this.exact.get(match) ?? this.words.get(match) ?? REDACTED);
  }
}

/** The redaction for a request sent with fetch(url, init). */
export function requestRedaction(url: string, init: RequestInit = {}): Redaction {
  const redaction = new Redaction();
  redaction.addUrl(url);
  redaction.addHeaders(init.headers);
  redaction.addBody(init.body);
  return redaction;
}

const attached = new WeakMap<object, Redaction>();

/** Record redaction on a response or error (merged with any it has). */
export function attach(target: unknown, redaction: Redaction): void {
  if (target === null || (typeof target !== 'object' && typeof target !== 'function')) return;
  const existing = attached.get(target as object);
  if (existing) {
    if (existing !== redaction) existing.merge(redaction);
  } else {
    attached.set(target as object, redaction);
  }
}

function redactionOf(sources: unknown[]): Redaction {
  const combined = new Redaction();
  for (const source of sources) {
    const found = source instanceof Redaction
      ? source
      : source !== null && typeof source === 'object' ? attached.get(source as object) : undefined;
    if (found) combined.merge(found);
  }
  return combined;
}

/**
 * The SDK's one redaction function.
 *
 * Returns text with what the request behind each source carried replaced
 * (each source is a Redaction, or a response or error the transport
 * attached one to):
 * - the request URL becomes its host and port;
 * - its path and query become "[path and query removed]", the path (or a
 *   tail of it from a "/") alone "[path removed]", and the query string,
 *   each name=value pair, each query value and each query name (as a whole
 *   word) "[redacted]"; its user information is dropped;
 * - every credential the request carried (the values of its Authorization,
 *   X-Admin-Token, macaroon, cookie and other headers whose names contain
 *   auth, token, key, secret, macaroon, cookie or password; the token after
 *   the scheme; Basic and L402 parts; the credential fields of the JSON
 *   body) becomes "[redacted]", also URL-encoded or JSON-escaped, and also
 *   in text quoted from a response body.
 * Credentials, query names and query values shorter than
 * MIN_REDACTED_LENGTH characters are replaced only as part of a longer form.
 */
export function redact(text: string, ...sources: unknown[]): string {
  return redactionOf(sources).apply(text);
}

/** A JSON parse error quotes the start of the response text; drop it. */
function withoutJsonExcerpt(text: string): string {
  return text.replace(/(?:\.\.\.)?"[\s\S]*?"(?:\.\.\.)? is not valid JSON/g, `${RESPONSE_TEXT_REMOVED} is not valid JSON`);
}

/**
 * Pass the message and stack of err, and of its causes, through redact; the
 * response text a JSON parse error quotes is dropped. The same error object
 * is returned, so its type, its other properties (a status code) and its
 * cause chain are unchanged; the redaction is attached to it so that an SDK
 * message quoting it can be redacted the same way.
 */
export function redactError<E>(err: E, ...sources: unknown[]): E {
  const redaction = redactionOf(sources);
  let current: unknown = err;
  for (let depth = 0; depth < 8 && current !== null && typeof current === 'object'; depth++) {
    const record = current as Record<string, unknown>;
    for (const key of ['message', 'stack']) {
      const value = record[key];
      if (typeof value !== 'string') continue;
      const clean = redaction.apply(withoutJsonExcerpt(value));
      if (clean === value) continue;
      try {
        Object.defineProperty(record, key, { value: clean, writable: true, configurable: true, enumerable: false });
      } catch {
        // A frozen error keeps its text.
      }
    }
    attach(record, redaction);
    current = record.cause;
  }
  return err;
}

/**
 * Run work (for example building headers or reading a response body) and
 * throw any error it raises through redactError for sources.
 */
export async function redactingErrors<T>(work: () => Promise<T> | T, ...sources: unknown[]): Promise<T> {
  try {
    return await work();
  } catch (err) {
    throw redactError(err, ...sources);
  }
}

/** The body-reading methods of a fetch Response (global fetch or node-fetch). */
interface ReadableResponse {
  json(): Promise<unknown>;
  text(): Promise<string>;
}

/** response.json(); a read or parse error is thrown redacted for its request. */
export function readJson<T>(response: ReadableResponse): Promise<T> {
  return redactingErrors(() => response.json() as Promise<T>, response);
}

/** response.text(); a read error is thrown redacted for its request. */
export function readText(response: ReadableResponse): Promise<string> {
  return redactingErrors(() => response.text(), response);
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
 * redirect's target. An error from the fetch (a network error, an invalid
 * header) is thrown as the same error object with its text redacted
 * (redactError). The response carries the request's Redaction, for the
 * messages the SDK builds from it (redact, readJson, readText).
 */
export async function fetchWithoutRedirects<R extends MinimalResponse = Response>(
  url: string,
  init: RequestInit = {},
  options: NoRedirectOptions<R> = {},
): Promise<R> {
  checkRequestUrl(url, options.label);
  const redaction = requestRedaction(url, init);
  const send = options.fetchImpl ?? (fetch as unknown as (url: string, init: RequestInit) => Promise<R>);
  const response = await redactingErrors(() => send(url, { ...init, redirect: 'manual' }), redaction);
  if (response.type === 'opaqueredirect' || (response.status >= 300 && response.status < 400)) {
    await discard(response.body);
    throw new RedirectRefusedError(response.status, requestHost(url), options.label);
  }
  attach(response, redaction);
  return response;
}

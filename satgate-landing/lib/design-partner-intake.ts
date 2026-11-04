import { createHash } from 'node:crypto';

export const MAX_BODY_BYTES = 16 * 1024;
export const RECEIPTS_PER_HOUR = 3;
export const success = { ok: true, message: "Application received! We'll be in touch within 24 hours." };

export class IntakeError extends Error {
  constructor(public status: number, message: string) { super(message); }
}

export function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]!));
}

export async function readApplication(req: Request) {
  const length = req.headers.get('content-length');
  if (length && (!/^\d+$/.test(length) || Number(length) > MAX_BODY_BYTES)) {
    throw new IntakeError(413, 'Request too large');
  }
  if (!/^application\/json(?:\s*;|$)/i.test(req.headers.get('content-type') || '')) {
    throw new IntakeError(415, 'Expected JSON');
  }
  const reader = req.body?.getReader();
  if (!reader) throw new IntakeError(400, 'Invalid application');
  const chunks: Uint8Array[] = [];
  let size = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > MAX_BODY_BYTES) {
        await reader.cancel();
        throw new IntakeError(413, 'Request too large');
      }
      chunks.push(value);
    }
  } finally { reader.releaseLock(); }
  let data;
  try { data = JSON.parse(Buffer.concat(chunks).toString('utf8')); }
  catch { throw new IntakeError(400, 'Invalid application'); }
  if (!data || typeof data !== 'object' || Array.isArray(data)) throw new IntakeError(400, 'Invalid application');
  // Quietly accept honeypot hits without sending either email.
  if (data.website !== undefined && data.website !== '') return null;
  const field = (name: string, max: number, required = false): string => {
    const value = data[name] ?? '';
    if (typeof value !== 'string' || value.length > max || /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/.test(value) || (required && !value.trim())) {
      throw new IntakeError(400, 'Invalid application');
    }
    return value.trim();
  };
  const name = field('name', 100, true);
  const email = field('email', 254, true).toLowerCase();
  const company = field('company', 200, true);
  const role = field('role', 100);
  const agentCount = field('agentCount', 50);
  const challenge = field('challenge', 4000);
  const local = email.split('@')[0];
  if (local.startsWith('.') || local.endsWith('.') || local.includes('..') || email.split('@')[1]?.split('.').some(label => label.length > 63)) {
    throw new IntakeError(400, 'Invalid application');
  }
  if (!/^[a-z0-9.!#$%&'*+\/=?^_`{|}~-]+@[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)+$/i.test(email) || email.split('@')[0].length > 64 || /[\r\n]/.test(company)) {
    throw new IntakeError(400, 'Invalid application');
  }
  const apis = data.apis ?? [];
  if (!Array.isArray(apis) || apis.length > 20 || apis.some(v => typeof v !== 'string' || v.length > 100 || /[\r\n\x00-\x1f\x7f]/.test(v))) {
    throw new IntakeError(400, 'Invalid application');
  }
  return { name, email, company, role, agentCount, challenge, apis: apis as string[] };
}

// Only the applicant confirmation needs a durable recipient limit. Reuse either
// existing Redis REST configuration pair; never mix credentials between pairs.
// Missing/unavailable storage suppresses confirmation, not the application.
export async function reserveRecipient(email: string): Promise<boolean> {
  const upstash = process.env.UPSTASH_REDIS_REST_URL && process.env.UPSTASH_REDIS_REST_TOKEN;
  const url = upstash ? process.env.UPSTASH_REDIS_REST_URL : process.env.KV_REST_API_URL;
  const token = upstash ? process.env.UPSTASH_REDIS_REST_TOKEN : process.env.KV_REST_API_TOKEN;
  if (!url || !token || !url.startsWith('https://')) return false;
  const key = `design-partner:${createHash('sha256').update(email).digest('hex')}`;
  const script = "local n = redis.call('INCR', KEYS[1]); if n == 1 then redis.call('EXPIRE', KEYS[1], 3600) end; return n";
  try {
    const res = await fetch(url, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify(['EVAL', script, '1', key]),
      signal: AbortSignal.timeout(5000),
      cache: 'no-store',
    });
    if (!res.ok) return false;
    const result = await res.json();
    return !result?.error && Number.isInteger(result?.result) && result.result >= 1 && result.result <= RECEIPTS_PER_HOUR;
  } catch {
    return false;
  }
}

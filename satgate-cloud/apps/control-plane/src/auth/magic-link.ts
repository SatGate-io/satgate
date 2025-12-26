/**
 * Magic Link Authentication
 * 
 * Flow:
 * 1. POST /auth/magic-link {email} → send email with code
 * 2. User clicks link → GET /auth/callback?code=xxx (web app)
 * 3. Web app calls POST /auth/verify {code} → session cookie
 */

import * as crypto from 'crypto';
import { query } from '../db';
import { sendMagicLinkEmail } from './email';
import { logger, ValidationError } from '@satgate/common';

// Code expires in 10 minutes
const CODE_TTL_MS = 10 * 60 * 1000;

// Rate limit: max 5 codes per email per hour
const RATE_LIMIT_WINDOW_MS = 60 * 60 * 1000;
const RATE_LIMIT_MAX = 5;

/**
 * Generate a secure random code
 */
function generateCode(): string {
  return crypto.randomBytes(32).toString('hex');
}

/**
 * Hash a code for storage
 */
function hashCode(code: string): string {
  return crypto.createHash('sha256').update(code).digest('hex');
}

/**
 * Validate email format
 */
function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email) && email.length <= 255;
}

/**
 * Request a magic link
 */
export async function requestMagicLink(email: string): Promise<{ sent: boolean }> {
  // Validate email
  const normalizedEmail = email.toLowerCase().trim();
  
  if (!isValidEmail(normalizedEmail)) {
    throw new ValidationError('Invalid email address');
  }
  
  // Check rate limit
  const rateCheck = await query<{ count: string }>(
    `SELECT COUNT(*) as count FROM auth_codes 
     WHERE email = $1 AND created_at > NOW() - INTERVAL '1 hour'`,
    [normalizedEmail]
  );
  
  if (parseInt(rateCheck.rows[0]?.count || '0', 10) >= RATE_LIMIT_MAX) {
    logger.warn('Magic link rate limit exceeded', { email: normalizedEmail });
    // Don't reveal rate limiting - just say "sent"
    return { sent: true };
  }
  
  // Generate code
  const code = generateCode();
  const codeHash = hashCode(code);
  const expiresAt = new Date(Date.now() + CODE_TTL_MS);
  
  // Store hashed code
  await query(
    `INSERT INTO auth_codes (email, code_hash, expires_at) VALUES ($1, $2, $3)`,
    [normalizedEmail, codeHash, expiresAt]
  );
  
  // Send email
  const appUrl = process.env.APP_URL || 'https://cloud.satgate.io';
  const magicLink = `${appUrl}/auth/callback?code=${code}`;
  
  try {
    await sendMagicLinkEmail(normalizedEmail, magicLink);
    logger.info('Magic link sent', { email: normalizedEmail });
  } catch (err) {
    logger.error('Failed to send magic link email', { 
      email: normalizedEmail, 
      error: (err as Error).message 
    });
    // Don't reveal email failures
  }
  
  return { sent: true };
}

/**
 * Verify a magic link code
 */
export async function verifyCode(code: string): Promise<{
  valid: boolean;
  email?: string;
  reason?: string;
}> {
  if (!code || code.length !== 64) {
    return { valid: false, reason: 'Invalid code format' };
  }
  
  const codeHash = hashCode(code);
  
  // Find and validate code (atomic: mark as used in same query)
  const result = await query<{ email: string }>(
    `UPDATE auth_codes 
     SET used = true 
     WHERE code_hash = $1 
       AND used = false 
       AND expires_at > NOW()
     RETURNING email`,
    [codeHash]
  );
  
  if (result.rowCount === 0) {
    return { valid: false, reason: 'Invalid or expired code' };
  }
  
  const email = result.rows[0].email;
  
  logger.info('Magic link verified', { email });
  
  return { valid: true, email };
}

/**
 * Clean up expired codes (call periodically)
 */
export async function cleanupExpiredCodes(): Promise<number> {
  const result = await query(
    `DELETE FROM auth_codes WHERE expires_at < NOW() - INTERVAL '1 day'`
  );
  return result.rowCount;
}


/**
 * Session management
 * 
 * Uses HttpOnly + SameSite=Strict cookies for security
 */

import * as crypto from 'crypto';
import { Request, Response, NextFunction } from 'express';
import { query } from '../db';
import { logger, AuthError } from '@satgate/common';

// Session expires in 7 days
const SESSION_TTL_MS = 7 * 24 * 60 * 60 * 1000;

// Cookie name
const SESSION_COOKIE = 'satgate_session';

/**
 * Session data stored in cookie
 */
export interface Session {
  tenantId: string;
  email: string;
  createdAt: number;
  expiresAt: number;
}

/**
 * Request with session
 */
export interface AuthenticatedRequest extends Request {
  session: Session;
  tenant: {
    id: string;
    slug: string;
    email: string;
  };
}

/**
 * Generate a session token
 */
function generateSessionToken(): string {
  return crypto.randomBytes(32).toString('hex');
}

/**
 * Hash a session token for storage
 */
function hashToken(token: string): string {
  return crypto.createHash('sha256').update(token).digest('hex');
}

/**
 * Create a session for a user
 */
export async function createSession(email: string): Promise<{
  token: string;
  expiresAt: Date;
  tenant: { id: string; slug: string };
}> {
  // Find or create tenant
  let tenant = await query<{ id: string; slug: string }>(
    `SELECT id, slug FROM tenants WHERE email = $1`,
    [email]
  );
  
  if (tenant.rowCount === 0) {
    // Create new tenant
    const slug = generateSlug(email);
    tenant = await query<{ id: string; slug: string }>(
      `INSERT INTO tenants (email, slug) VALUES ($1, $2) RETURNING id, slug`,
      [email, slug]
    );
    logger.info('New tenant created', { email, slug });
  }
  
  const tenantId = tenant.rows[0].id;
  const tenantSlug = tenant.rows[0].slug;
  
  // Generate session
  const token = generateSessionToken();
  const tokenHash = hashToken(token);
  const expiresAt = new Date(Date.now() + SESSION_TTL_MS);
  
  // Store session (delete old sessions for this tenant first)
  await query(
    `DELETE FROM sessions WHERE tenant_id = $1 AND expires_at < NOW()`,
    [tenantId]
  );
  
  await query(
    `INSERT INTO sessions (tenant_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
    [tenantId, tokenHash, expiresAt]
  );
  
  logger.info('Session created', { tenantId, expiresAt });
  
  return {
    token,
    expiresAt,
    tenant: { id: tenantId, slug: tenantSlug },
  };
}

/**
 * Generate a URL-safe slug from email
 */
function generateSlug(email: string): string {
  // Use prefix of email + random suffix
  const prefix = email.split('@')[0]
    .toLowerCase()
    .replace(/[^a-z0-9]/g, '')
    .substring(0, 12);
  const suffix = crypto.randomBytes(4).toString('hex');
  return `${prefix}-${suffix}`;
}

/**
 * Validate a session token
 */
export async function validateSession(token: string): Promise<Session | null> {
  if (!token || token.length !== 64) {
    return null;
  }
  
  const tokenHash = hashToken(token);
  
  const result = await query<{
    tenant_id: string;
    email: string;
    created_at: Date;
    expires_at: Date;
  }>(
    `SELECT s.tenant_id, t.email, s.created_at, s.expires_at
     FROM sessions s
     JOIN tenants t ON t.id = s.tenant_id
     WHERE s.token_hash = $1 AND s.expires_at > NOW()`,
    [tokenHash]
  );
  
  if (result.rowCount === 0) {
    return null;
  }
  
  const row = result.rows[0];
  
  return {
    tenantId: row.tenant_id,
    email: row.email,
    createdAt: row.created_at.getTime(),
    expiresAt: row.expires_at.getTime(),
  };
}

/**
 * Get cookie options based on environment
 * 
 * Same-origin (recommended):
 *   Dashboard & API on same domain → SameSite=Strict
 * 
 * Cross-origin (if needed):
 *   Set COOKIE_CROSS_SITE=true → SameSite=None; Secure
 *   Set COOKIE_DOMAIN=.satgate.io for shared cookies
 */
function getCookieOptions(expiresAt?: Date): {
  httpOnly: boolean;
  secure: boolean;
  sameSite: 'strict' | 'lax' | 'none';
  domain?: string;
  path: string;
  expires?: Date;
} {
  const isProduction = process.env.NODE_ENV === 'production';
  const isCrossSite = process.env.COOKIE_CROSS_SITE === 'true';
  const cookieDomain = process.env.COOKIE_DOMAIN; // e.g., ".satgate.io"
  
  const options: ReturnType<typeof getCookieOptions> = {
    httpOnly: true,
    secure: isProduction || isCrossSite, // Secure required for SameSite=None
    sameSite: isCrossSite ? 'none' : 'strict',
    path: '/',
  };
  
  if (cookieDomain) {
    options.domain = cookieDomain;
  }
  
  if (expiresAt) {
    options.expires = expiresAt;
  }
  
  return options;
}

/**
 * Set session cookie on response
 */
export function setSessionCookie(res: Response, token: string, expiresAt: Date): void {
  res.cookie(SESSION_COOKIE, token, getCookieOptions(expiresAt));
}

/**
 * Clear session cookie
 */
export function clearSessionCookie(res: Response): void {
  res.clearCookie(SESSION_COOKIE, getCookieOptions());
}

/**
 * Auth middleware - require session
 */
export function requireAuth(
  req: Request,
  res: Response,
  next: NextFunction
): void {
  const token = req.cookies?.[SESSION_COOKIE];
  
  if (!token) {
    res.status(401).json({ error: 'Authentication required' });
    return;
  }
  
  validateSession(token)
    .then(async (session) => {
      if (!session) {
        clearSessionCookie(res);
        res.status(401).json({ error: 'Invalid or expired session' });
        return;
      }
      
      // Get tenant info
      const tenant = await query<{ id: string; slug: string; email: string }>(
        `SELECT id, slug, email FROM tenants WHERE id = $1`,
        [session.tenantId]
      );
      
      if (tenant.rowCount === 0) {
        clearSessionCookie(res);
        res.status(401).json({ error: 'Tenant not found' });
        return;
      }
      
      (req as AuthenticatedRequest).session = session;
      (req as AuthenticatedRequest).tenant = tenant.rows[0];
      
      next();
    })
    .catch((err) => {
      logger.error('Session validation error', { error: (err as Error).message });
      res.status(500).json({ error: 'Internal server error' });
    });
}

/**
 * Optional auth middleware - sets session if valid but doesn't require it
 */
export function optionalAuth(
  req: Request,
  res: Response,
  next: NextFunction
): void {
  const token = req.cookies?.[SESSION_COOKIE];
  
  if (!token) {
    next();
    return;
  }
  
  validateSession(token)
    .then(async (session) => {
      if (session) {
        const tenant = await query<{ id: string; slug: string; email: string }>(
          `SELECT id, slug, email FROM tenants WHERE id = $1`,
          [session.tenantId]
        );
        
        if (tenant.rowCount > 0) {
          (req as AuthenticatedRequest).session = session;
          (req as AuthenticatedRequest).tenant = tenant.rows[0];
        }
      }
      next();
    })
    .catch(() => {
      next();
    });
}

/**
 * Logout - invalidate session
 */
export async function logout(token: string): Promise<void> {
  const tokenHash = hashToken(token);
  await query(`DELETE FROM sessions WHERE token_hash = $1`, [tokenHash]);
}


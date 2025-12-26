/**
 * Auth routes
 */

import { Router, Request, Response, IRouter } from 'express';
import {
  requestMagicLink,
  verifyCode,
  createSession,
  setSessionCookie,
  clearSessionCookie,
  requireAuth,
  logout,
  AuthenticatedRequest,
} from '../auth';
import { logger, ValidationError } from '@satgate/common';

const router: IRouter = Router();

/**
 * POST /auth/magic-link
 * Request a magic link for email
 */
router.post('/magic-link', async (req: Request, res: Response) => {
  try {
    const { email } = req.body;
    
    if (!email || typeof email !== 'string') {
      return res.status(400).json({ error: 'Email is required' });
    }
    
    const result = await requestMagicLink(email);
    
    // Always return success to prevent email enumeration
    return res.json({
      message: 'If this email is valid, a magic link has been sent.',
      sent: result.sent,
    });
  } catch (err) {
    if (err instanceof ValidationError) {
      return res.status(400).json({ error: err.message });
    }
    
    logger.error('Magic link error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * POST /auth/verify
 * Verify a magic link code and create session
 * 
 * NOTE: The web app should:
 * 1. Receive code via GET /auth/callback?code=xxx
 * 2. Immediately POST to /auth/verify { code }
 * 3. Use history.replaceState() to remove code from URL
 */
router.post('/verify', async (req: Request, res: Response) => {
  try {
    const { code } = req.body;
    
    if (!code || typeof code !== 'string') {
      return res.status(400).json({ error: 'Code is required' });
    }
    
    const verification = await verifyCode(code);
    
    if (!verification.valid || !verification.email) {
      return res.status(401).json({ 
        error: verification.reason || 'Invalid code' 
      });
    }
    
    // Create session
    const session = await createSession(verification.email);
    
    // Set cookie
    setSessionCookie(res, session.token, session.expiresAt);
    
    return res.json({
      success: true,
      tenant: {
        slug: session.tenant.slug,
      },
    });
  } catch (err) {
    logger.error('Verify error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * POST /auth/logout
 * Clear session
 */
router.post('/logout', requireAuth, async (req: Request, res: Response) => {
  try {
    const token = req.cookies?.satgate_session;
    
    if (token) {
      await logout(token);
    }
    
    clearSessionCookie(res);
    
    return res.json({ success: true });
  } catch (err) {
    logger.error('Logout error', { error: (err as Error).message });
    return res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * GET /auth/me
 * Get current user info
 */
router.get('/me', requireAuth, (req: Request, res: Response) => {
  const authReq = req as AuthenticatedRequest;
  
  return res.json({
    tenant: {
      id: authReq.tenant.id,
      slug: authReq.tenant.slug,
      email: authReq.tenant.email,
    },
  });
});

export default router;


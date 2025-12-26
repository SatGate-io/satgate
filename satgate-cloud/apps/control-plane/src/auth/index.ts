/**
 * Auth module exports
 */

export { requestMagicLink, verifyCode, cleanupExpiredCodes } from './magic-link';
export { sendMagicLinkEmail } from './email';
export {
  createSession,
  validateSession,
  setSessionCookie,
  clearSessionCookie,
  requireAuth,
  optionalAuth,
  logout,
  Session,
  AuthenticatedRequest,
} from './session';


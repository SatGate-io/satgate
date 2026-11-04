import { NextRequest, NextResponse } from 'next/server';

export function proxy(request: NextRequest) {
  const nonce = Buffer.from(crypto.getRandomValues(new Uint8Array(24))).toString('base64');
  const csp = [
    "default-src 'self'",
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'`,
    "script-src-attr 'none'",
    // React and the existing pages use style attributes; scripts never allow unsafe-inline/eval.
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: https:",
    "font-src 'self'",
    "connect-src 'self' https://api.satgate.io https://cloud.satgate.io https://satgate-production-9354.up.railway.app",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
  ].join('; ');
  const headers = new Headers(request.headers);
  // Replace, never trust, client-supplied CSP/nonce headers. Next extracts this
  // nonce to authorize its bootstrap and streamed hydration scripts.
  headers.set('Content-Security-Policy', csp);
  const response = NextResponse.next({ request: { headers } });
  response.headers.set('Content-Security-Policy', csp);
  response.headers.set('Cache-Control', 'private, no-store');
  return response;
}

export const config = {
  matcher: ['/((?!api(?:/|$)|_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt).*)'],
};

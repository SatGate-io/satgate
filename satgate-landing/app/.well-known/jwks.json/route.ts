// Receipts are signed by https://api.satgate.io. Do not publish an empty key set here.
const JWKS_URL = "https://api.satgate.io/.well-known/jwks.json";

export const dynamic = "force-static";

export function GET() {
  return new Response(null, {
    status: 308,
    headers: {
      Location: JWKS_URL,
      "Cache-Control": "public, max-age=300",
      "Access-Control-Allow-Origin": "*",
      "X-Content-Type-Options": "nosniff",
    },
  });
}

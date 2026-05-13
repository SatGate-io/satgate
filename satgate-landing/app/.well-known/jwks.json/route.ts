const jwks = {
  keys: [],
  note: "SatGate tenant or deployment issuers publish signing keys at their issuer-specific JWKS URI. The public satgate.io metadata issuer does not expose tenant receipt-signing keys here.",
} as const;

export const dynamic = "force-static";

export function GET() {
  return Response.json(jwks, {
    headers: {
      "Cache-Control": "public, max-age=3600",
      "Access-Control-Allow-Origin": "*",
      "X-Content-Type-Options": "nosniff",
    },
  });
}

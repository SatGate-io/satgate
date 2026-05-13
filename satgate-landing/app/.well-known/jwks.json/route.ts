const jwks = {
  keys: [],
} as const;

export const dynamic = "force-static";

export function GET() {
  return Response.json(jwks, {
    headers: {
      "Cache-Control": "public, max-age=3600",
      "Access-Control-Allow-Origin": "*",
      "X-Content-Type-Options": "nosniff",
      "Vary": "Accept-Encoding",
    },
  });
}

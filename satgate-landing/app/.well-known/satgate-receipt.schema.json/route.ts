import schema from "../../../schemas/satgate-receipt.schema.json";

export const dynamic = "force-static";

export function GET() {
  return Response.json(schema, {
    headers: {
      "Cache-Control": "public, max-age=86400",
      "Access-Control-Allow-Origin": "*",
      "X-Content-Type-Options": "nosniff",
    },
  });
}

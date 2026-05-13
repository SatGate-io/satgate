const schema = {
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://satgate.io/.well-known/satgate.schema.json",
  title: "SatGate Trust Metadata v1",
  type: "object",
  required: [
    "schema_version",
    "metadata_url",
    "schema_url",
    "roles",
    "issuer",
    "capability_acceptance",
    "receipt_verification",
    "rails_adapters",
    "signing_key_discovery",
    "docs",
  ],
  properties: {
    schema_version: { const: "satgate.trust_metadata.v1" },
    metadata_url: { type: "string", format: "uri" },
    schema_url: { type: "string", format: "uri" },
    roles: { type: "array", items: { enum: ["issuer", "acceptor"] }, minItems: 1 },
    issuer: {
      type: "object",
      required: ["name", "issuer_id", "product", "contact", "key_discovery"],
      properties: {
        name: { type: "string" },
        issuer_id: { type: "string", format: "uri" },
        product: { type: "string" },
        contact: { type: "string" },
        key_discovery: {
          type: "object",
          required: ["method", "key_id_field", "jwks_uri"],
          properties: {
            method: { const: "jwks_uri" },
            key_id_field: { const: "issuer_kid" },
            jwks_uri: { type: "string", format: "uri" },
          },
          additionalProperties: true,
        },
      },
      additionalProperties: true,
    },
    capability_acceptance: { type: "object", additionalProperties: true },
    receipt_verification: {
      type: "object",
      required: ["receipt_format", "evidence_pack_format", "required_receipt_fields", "decisions"],
      properties: {
        decisions: {
          type: "array",
          items: { enum: ["allowed", "denied", "delegated", "revoked", "paid"] },
        },
      },
      additionalProperties: true,
    },
    rails_adapters: {
      type: "object",
      required: ["supported"],
      properties: {
        supported: {
          type: "array",
          items: {
            type: "object",
            required: ["id", "type", "role"],
            properties: {
              id: { type: "string" },
              type: { type: "string" },
              role: { type: "string" },
            },
            additionalProperties: true,
          },
        },
      },
      additionalProperties: true,
    },
    signing_key_discovery: { type: "object", additionalProperties: true },
    docs: { type: "object", additionalProperties: true },
  },
  additionalProperties: true,
} as const;

export const dynamic = "force-static";

export function GET() {
  return Response.json(schema, {
    headers: {
      "Cache-Control": "public, max-age=3600",
      "Access-Control-Allow-Origin": "*",
      "X-Content-Type-Options": "nosniff",
    },
  });
}

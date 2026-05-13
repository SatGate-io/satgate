const schema = {
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://satgate.io/.well-known/satgate.schema.json",
  title: "SatGate Trust Metadata v1",
  description: "Machine-readable discovery metadata for SatGate capability issuance, receipt verification, rail-adapter status, and issuer-key discovery.",
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
    schema_version: {
      const: "satgate.trust_metadata.v1",
      description: "Additive-only v1 schema identifier. Breaking changes require a new schema_version.",
    },
    metadata_url: {
      type: "string",
      format: "uri",
      description: "Canonical URL for this metadata artifact.",
    },
    schema_url: {
      type: "string",
      format: "uri",
      description: "Canonical JSON Schema URL for this metadata version.",
    },
    roles: {
      type: "array",
      description: "Metadata roles this artifact asserts. The public SatGate artifact is issuer-side only.",
      items: { enum: ["issuer", "acceptor"] },
      minItems: 1,
      uniqueItems: true,
    },
    issuer: {
      type: "object",
      description: "Issuer identity and issuer JWKS discovery metadata for this artifact.",
      required: ["name", "issuer_id", "product", "contact", "key_discovery"],
      properties: {
        name: { type: "string", description: "Human-readable issuer name." },
        issuer_id: { type: "string", format: "uri", description: "Origin-style issuer identifier." },
        product: { type: "string", description: "Human-readable product/category descriptor." },
        contact: { type: "string", description: "Operational contact for metadata consumers." },
        key_discovery: {
          type: "object",
          description: "JWKS discovery location for this manifest publisher/issuer.",
          required: ["method", "key_id_field", "jwks_uri"],
          properties: {
            method: { const: "jwks_uri", description: "Key discovery uses a JSON Web Key Set endpoint." },
            key_id_field: { const: "issuer_kid", description: "Receipt field that selects the signing key." },
            jwks_uri: { type: "string", format: "uri", description: "JWKS URL for this issuer." },
          },
          additionalProperties: true,
        },
      },
      additionalProperties: true,
    },
    capability_acceptance: {
      type: "object",
      description: "Capability/token formats, authorization schemes, required claims, caveats, and delegation-depth semantics accepted by this issuer profile.",
      additionalProperties: true,
    },
    receipt_verification: {
      type: "object",
      description: "Receipt and Evidence Pack formats, required receipt fields, closed decision vocabulary, and verification endpoints.",
      required: ["receipt_format", "evidence_pack_format", "required_receipt_fields", "decisions"],
      properties: {
        required_receipt_fields: {
          type: "array",
          description: "Receipt fields needed to identify the issuer, select the signing key, verify the decision, and bind the receipt to an Evidence Pack.",
          items: {
            enum: [
              "receipt_id",
              "evidence_pack_id",
              "issuer",
              "issuer_kid",
              "decision",
              "decision_reason",
              "policy_version",
              "receipt_hash",
              "signature",
            ],
          },
          uniqueItems: true,
          allOf: [
            { contains: { const: "issuer" } },
            { contains: { const: "issuer_kid" } },
          ],
        },
        decisions: {
          type: "array",
          description: "Closed v1 decision vocabulary emitted in SatGate receipts.",
          items: { enum: ["allowed", "denied", "delegated", "revoked", "paid"] },
          uniqueItems: true,
        },
      },
      additionalProperties: true,
    },
    rails_adapters: {
      type: "object",
      description: "Rail/protocol/billing adapters governed beneath SatGate authority and receipt verification.",
      required: ["supported"],
      properties: {
        supported: {
          type: "array",
          description: "Known adapter catalog. Consumers should key on id and treat status=planned as not currently supported.",
          uniqueItems: true,
          items: {
            type: "object",
            required: ["id", "type", "role", "status"],
            properties: {
              id: { type: "string", description: "Stable adapter identifier." },
              type: { type: "string", description: "Adapter category, such as protocol, payment_rail, billing_adapter, or ledger_adapter." },
              role: { type: "string", description: "Adapter role under SatGate authority/proof." },
              status: { enum: ["supported", "planned"], description: "Current support status for this adapter id." },
            },
            additionalProperties: true,
          },
        },
      },
      additionalProperties: true,
    },
    signing_key_discovery: {
      type: "object",
      description: "General federation rule: verify receipts against the trusted issuer origin's JWKS selected by issuer_kid.",
      required: ["mode", "key_id_field", "jwks_uri_template"],
      properties: {
        mode: { const: "issuer_discovery", description: "Keys are discovered from the trusted receipt issuer origin." },
        key_id_field: { const: "issuer_kid", description: "Receipt field that selects the issuer signing key." },
        jwks_uri_template: { const: "{issuer_id}/.well-known/jwks.json", description: "JWKS discovery template after the issuer_id has matched an acceptor trust anchor." },
      },
      additionalProperties: true,
    },
    docs: {
      type: "object",
      description: "Human-readable documentation links for builders, verifiers, and auditors.",
      additionalProperties: true,
    },
  },
  additionalProperties: true,
} as const;

export const dynamic = "force-static";

export function GET() {
  return Response.json(schema, {
    headers: {
      "Cache-Control": "public, max-age=86400",
      "Access-Control-Allow-Origin": "*",
      "X-Content-Type-Options": "nosniff",
      "Vary": "Accept-Encoding",
    },
  });
}

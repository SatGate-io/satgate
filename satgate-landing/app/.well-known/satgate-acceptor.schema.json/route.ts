const schema = {
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://satgate.io/.well-known/satgate-acceptor.schema.json",
  title: "SatGate Acceptor Metadata v0",
  description:
    "Draft machine-readable metadata for upstream APIs that accept SatGate-scoped capabilities and emit SatGate-compatible receipts. v0 is intentionally permissive but closes core enums before third-party drift.",
  type: "object",
  required: [
    "schema_version",
    "metadata_url",
    "schema_url",
    "roles",
    "status",
    "acceptor",
    "accepted_capability_formats",
    "recognized_receipt_decisions",
    "emitted_receipt_decisions",
    "trust_anchors",
    "rails_adapters",
    "claims",
  ],
  properties: {
    schema_version: {
      const: "satgate.acceptor_metadata.v0",
      description: "Draft acceptor metadata schema. Breaking changes require a new schema_version.",
    },
    metadata_url: { type: "string", format: "uri", description: "Canonical URL for this acceptor metadata artifact." },
    schema_url: { const: "https://satgate.io/.well-known/satgate-acceptor.schema.json" },
    roles: {
      type: "array",
      items: { enum: ["acceptor", "issuer"] },
      minItems: 1,
      uniqueItems: true,
      contains: { const: "acceptor" },
      description: "Acceptor v0 artifacts must assert acceptor. Dual-role artifacts are reserved for same-path deployments and must still satisfy issuer schemas separately.",
    },
    status: {
      enum: ["internal_mock_only", "draft", "active", "deprecated", "revoked"],
      description: "Artifact lifecycle status. internal_mock_only is not a production acceptor claim.",
    },
    acceptor: {
      type: "object",
      required: ["name", "acceptor_id"],
      properties: {
        name: { type: "string" },
        acceptor_id: { type: "string", format: "uri", description: "Stable upstream acceptor identity bound into returned receipts." },
        verification_endpoint: { type: "string", format: "uri" },
      },
      additionalProperties: true,
    },
    accepted_capability_formats: {
      type: "array",
      items: { enum: ["satgate.capability.v1", "macaroon-bearer"] },
      minItems: 1,
      uniqueItems: true,
      description: "Capability formats this upstream accepts as request authority.",
    },
    recognized_receipt_decisions: {
      type: "array",
      items: { enum: ["allowed", "paid"] },
      minItems: 1,
      uniqueItems: true,
      description: "Prior receipt decisions this acceptor recognizes as acceptable evidence for entry. denied is intentionally excluded.",
    },
    emitted_receipt_decisions: {
      type: "array",
      items: { enum: ["allowed", "denied", "paid"] },
      minItems: 1,
      uniqueItems: true,
      description: "Receipt decisions this acceptor can emit after evaluating a capability. Acceptor v0 emits a subset of the issuer decision space.",
    },
    trust_anchors: {
      type: "array",
      minItems: 1,
      items: {
        type: "object",
        required: ["issuer_id", "status"],
        properties: {
          issuer_id: { type: "string", format: "uri" },
          jwks_uri: { type: "string", format: "uri", description: "Optional explicit JWKS pin. If omitted, verifiers use {issuer_id}/.well-known/jwks.json after trust-anchor validation." },
          status: {
            enum: ["accepted", "provisional", "revoked", "deprecated", "accepted_for_mock"],
            description: "Closed trust-anchor lifecycle state. accepted_for_mock is reserved for examples only.",
          },
        },
        additionalProperties: true,
      },
    },
    rails_adapters: {
      type: "object",
      required: ["accepted"],
      properties: {
        accepted: {
          type: "array",
          description: "Rails this upstream will settle via or route through. Acceptor metadata uses accepted; issuer metadata uses supported.",
          items: {
            type: "object",
            required: ["id", "type", "role", "status"],
            properties: {
              id: { type: "string" },
              type: { type: "string" },
              role: { type: "string" },
              status: { enum: ["supported", "planned", "deprecated"] },
            },
            additionalProperties: true,
          },
        },
      },
      additionalProperties: true,
    },
    claims: {
      type: "object",
      required: ["acceptance_means", "acceptance_does_not_mean"],
      properties: {
        acceptance_means: { const: "capability verification and receipt emission" },
        acceptance_does_not_mean: {
          type: "array",
          allOf: [
            { contains: { const: "marketplace listing" } },
            { contains: { const: "reputation score" } },
            { contains: { const: "SatGate endorsement" } },
            { contains: { const: "network-wide trust" } },
            { contains: { const: "ranking" } },
            { contains: { const: "certification" } },
          ],
          items: {
            enum: ["marketplace listing", "reputation score", "SatGate endorsement", "network-wide trust", "ranking", "certification"],
          },
          uniqueItems: true,
        },
      },
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
    },
  });
}

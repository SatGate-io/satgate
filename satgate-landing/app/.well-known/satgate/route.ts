const metadata = {
  schema_version: "satgate.trust_metadata.v1",
  metadata_url: "https://satgate.io/.well-known/satgate",
  issuer: {
    name: "SatGate",
    issuer_id: "https://satgate.io",
    product: "Economic Firewall for AI agents",
    contact: "contact@satgate.io",
  },
  capability_acceptance: {
    accepted_formats: ["satgate.capability.v1", "macaroon-bearer"],
    authorization_schemes: ["Bearer"],
    required_claims: [
      "issuer",
      "subject",
      "audience",
      "scope",
      "expires_at",
      "policy_version",
    ],
    supported_caveats: [
      "scope",
      "route",
      "tool",
      "budget",
      "tenant",
      "expires_at",
      "delegation_depth",
    ],
    delegation_depth: "policy_defined",
  },
  receipt_verification: {
    receipt_format: "satgate.receipt.v1",
    evidence_pack_format: "satgate.evidence_pack.v1",
    required_receipt_fields: [
      "receipt_id",
      "evidence_pack_id",
      "decision",
      "decision_reason",
      "policy_version",
      "receipt_hash",
      "signature",
    ],
    decisions: ["allowed", "denied", "delegated", "revoked", "paid"],
    verification_endpoint: "https://api.satgate.io/v1/verify",
    evidence_pack_schema_url: "https://satgate.io/evidence-packs/evidence-pack.schema.v1.json",
  },
  rails_adapters: {
    supported: ["mcp", "x402", "l402", "api_key_billing", "enterprise_ledger"],
    note: "Rails are adapters below SatGate authority and receipt verification; this metadata makes no marketplace or reputation claim.",
  },
  signing_key_discovery: {
    mode: "issuer_discovery",
    key_id_field: "issuer_kid",
    jwks_url_template: "{issuer_id}/.well-known/jwks.json",
    note: "Verify receipts against the issuer key referenced by issuer_kid. Tenant or deployment issuers may publish their own JWKS endpoint.",
  },
  docs: {
    build: "https://satgate.io/build",
    policy_to_proof: "https://satgate.io/policy-to-proof",
    evidence_pack_reference: "https://github.com/SatGate-io/satgate/blob/main/docs/reference/evidence-pack.md",
    api_docs: "https://cloud.satgate.io/docs",
  },
} as const;

export const dynamic = "force-static";

export function GET() {
  return Response.json(metadata, {
    headers: {
      "Cache-Control": "public, max-age=3600",
      "Access-Control-Allow-Origin": "*",
      "X-Content-Type-Options": "nosniff",
    },
  });
}

const metadata = {
  schema_version: "satgate.trust_metadata.v1",
  metadata_url: "https://satgate.io/.well-known/satgate",
  schema_url: "https://satgate.io/.well-known/satgate.schema.json",
  roles: ["issuer"],
  issuer: {
    name: "SatGate",
    issuer_id: "https://api.satgate.io",
    product: "Economic firewall for AI agents: budgets for your own agents, payment from external agents on the routes you choose, and a signed receipt for every decision.",
    contact: "contact@satgate.io",
    key_discovery: {
      method: "jwks_uri",
      key_id_field: "issuer_kid",
      jwks_uri: "https://api.satgate.io/.well-known/jwks.json",
    },
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
    receipt_schema_url: "https://satgate.io/.well-known/satgate-receipt.schema.json",
    evidence_pack_format: "satgate.evidence_pack.v1",
    required_receipt_fields: [
      "schema_version",
      "schema_url",
      "receipt_id",
      "evidence_pack_id",
      "issuer",
      "issuer_kid",
      "decision",
      "decision_reason",
      "policy_version",
      "timestamp",
      "canonicalization",
      "hash_algorithm",
      "signature_algorithm",
      "receipt_hash",
      "signature",
    ],
    decisions: ["allowed", "denied", "delegated", "revoked", "paid"],
    verifier: {
      mode: "offline",
      repository: "https://github.com/SatGate-io/satgate",
      path: "tools/verify_evidence_pack.py",
      url: "https://github.com/SatGate-io/satgate/blob/main/tools/verify_evidence_pack.py",
    },
    evidence_pack_schema_url: "https://satgate.io/evidence-packs/evidence-pack.schema.v1.json",
  },
  rails_adapters: {
    supported: [
      { id: "mcp", type: "protocol", role: "tool_transport", status: "supported" },
      {
        id: "x402",
        type: "payment_rail",
        role: "external_paid_access",
        status: "supported",
        x402_version: 2,
        schemes: ["exact"],
        networks: ["eip155:8453", "eip155:84532"],
        assets: {
          "eip155:8453": "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
          "eip155:84532": "0x036cbd53842c5426634e7929541ec2318f3dcf7e",
        },
        payment_scope: "one_request",
        receipt_authority: "settled_x402_authorization",
      },
      { id: "l402", type: "payment_rail", role: "external_paid_access", status: "supported" },
      { id: "api_key_billing", type: "billing_adapter", role: "existing_vendor_billing", status: "supported" },
      { id: "enterprise_ledger", type: "ledger_adapter", role: "internal_chargeback", status: "supported" },
      { id: "agentcore_payments", type: "payment_rail", role: "external_paid_access", status: "planned" },
      { id: "pay_sh", type: "payment_rail", role: "external_paid_access", status: "planned" },
    ],
  },
  signing_key_discovery: {
    mode: "issuer_discovery",
    key_id_field: "issuer_kid",
    jwks_uri_template: "{issuer_id}/.well-known/jwks.json",
  },
  docs: {
    build: "https://satgate.io/build",
    policy_to_proof: "https://satgate.io/policy-to-proof",
    evidence_pack_reference: "https://github.com/SatGate-io/satgate/blob/main/docs/reference/evidence-pack.md",
    trust_metadata_reference: "https://github.com/SatGate-io/satgate/blob/main/docs/reference/satgate-trust-metadata.md",
    receipt_schema_reference: "https://github.com/SatGate-io/satgate/blob/main/docs/reference/receipt-schema.md",
    acceptor_metadata_draft: "https://github.com/SatGate-io/satgate/blob/main/docs/reference/acceptor.md",
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

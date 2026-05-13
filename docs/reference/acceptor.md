# SatGate Acceptor Metadata Draft

Status: **v0.0 draft — not yet on the wire**.

This document reserves the mirror side of the SatGate trust metadata model before issuer-side `satgate.trust_metadata.v1` ossifies in downstream parsers. It is intentionally documentation-only until an upstream service actually accepts SatGate capabilities.

## Purpose

Issuer metadata answers: “What capabilities can this issuer mint, and how do I verify its receipts?”

Acceptor metadata should answer: “What capabilities will this upstream accept, from which issuers, and on which rails can it settle?”

## Candidate discovery path

Two options are still open:

- Same path, role-selected: `/.well-known/satgate` with `roles: ["acceptor"]` or `roles: ["issuer", "acceptor"]`.
- Companion path: `/.well-known/satgate-upstream` for upstream-only services.

The compatibility rule is the same: clients must ignore unknown fields, and breaking field changes require a new schema version.

## Candidate fields

```json
{
  "schema_version": "satgate.acceptor_metadata.v0",
  "metadata_url": "https://api.example.com/.well-known/satgate",
  "schema_url": "https://satgate.io/.well-known/satgate-acceptor.schema.json",
  "roles": ["acceptor"],
  "acceptor": {
    "name": "Example API",
    "acceptor_id": "https://api.example.com",
    "verification_endpoint": "https://api.example.com/satgate/verify"
  },
  "accepted_capability_formats": [
    "satgate.capability.v1",
    "macaroon-bearer"
  ],
  "accepted_receipt_decisions": [
    "allowed",
    "delegated",
    "paid"
  ],
  "trust_anchors": [
    {
      "issuer_id": "https://satgate.io",
      "jwks_uri": "https://satgate.io/.well-known/jwks.json",
      "status": "accepted"
    }
  ],
  "rails_adapters": {
    "accepted": [
      { "id": "mcp", "type": "protocol", "role": "tool_transport", "status": "supported" },
      { "id": "x402", "type": "payment_rail", "role": "external_paid_access", "status": "supported" }
    ]
  }
}
```

## Field notes

- `verification_endpoint`: optional upstream endpoint for online verification, replay checks, or policy-specific acceptance decisions. Offline signature checks should still use issuer JWKS discovery when possible.
- `accepted_capability_formats`: subset of capability formats the upstream honors. An acceptor does not need to accept every format an issuer can emit.
- `accepted_receipt_decisions`: subset of receipt decisions the upstream treats as acceptable evidence for entry. For example, a read-only upstream may accept `allowed` and reject `paid`.
- `trust_anchors`: issuer origins the upstream accepts, optionally pinned to JWKS URIs or future certificate transparency/audit roots.
- `rails_adapters.accepted`: rails the upstream can settle on or route through.

## Non-goals

- No marketplace, ranking, reputation, endorsement, or compliance claim.
- No fake acceptor role in the public SatGate issuer artifact.
- No requirement that SatGate stay in the request path after an upstream can verify receipts and capabilities directly.

## Open questions

- Whether acceptor metadata belongs under the same `satgate.trust_metadata.v1` schema or a sibling acceptor schema.
- Whether `trust_anchors` should support key pinning, issuer catalogs, or both.
- Whether online verification endpoints should be required for paid rails or only recommended for replay-sensitive routes.

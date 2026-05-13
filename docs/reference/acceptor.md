# SatGate Acceptor Metadata Draft

Status: **v0.0 draft — mock schema on the wire; no public acceptor claim yet**.

This document reserves the mirror side of the SatGate trust metadata model before issuer-side `satgate.trust_metadata.v1` ossifies in downstream parsers. The public badge and integration story live in [Accept SatGate Capabilities](accept-satgate-capabilities.md). The draft JSON Schema is published at `https://satgate.io/.well-known/satgate-acceptor.schema.json` so early implementers do not invent incompatible field names.

## Purpose

Issuer metadata answers: “What capabilities can this issuer mint, and how do I verify its receipts?”

Acceptor metadata answers: “What capabilities will this upstream accept, from which issuers, which prior receipt decisions will it recognize, which decisions can it emit, and on which rails can it settle?”

## Candidate discovery path

Two options are still open:

- Same path, role-selected: `/.well-known/satgate` with `roles: ["acceptor"]` or, for a future dual-role artifact, `roles: ["issuer", "acceptor"]`. The v0 acceptor schema requires `acceptor` and leaves issuer validation to the issuer schema.
- Companion path: `/.well-known/satgate-upstream` for upstream-only services.

The compatibility rule is the same: clients must ignore unknown fields, and breaking field changes require a new schema version. For v0, the schema keeps `additionalProperties: true` while closing the fields most likely to drift: role, claim boundary, trust-anchor status, and receipt decision sets.

## Candidate fields

```json
{
  "schema_version": "satgate.acceptor_metadata.v0",
  "metadata_url": "https://api.example.com/.well-known/satgate",
  "schema_url": "https://satgate.io/.well-known/satgate-acceptor.schema.json",
  "roles": ["acceptor"],
  "status": "active",
  "acceptor": {
    "name": "Example API",
    "acceptor_id": "https://api.example.com",
    "verification_endpoint": "https://api.example.com/satgate/verify"
  },
  "accepted_capability_formats": [
    "satgate.capability.v1",
    "macaroon-bearer"
  ],
  "recognized_receipt_decisions": [
    "allowed",
    "paid"
  ],
  "emitted_receipt_decisions": [
    "allowed",
    "denied",
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
  },
  "claims": {
    "acceptance_means": "capability verification and receipt emission",
    "acceptance_does_not_mean": [
      "marketplace listing",
      "reputation score",
      "SatGate endorsement",
      "network-wide trust",
      "ranking",
      "certification"
    ]
  }
}
```

## Field notes

- `schema_url`: canonical v0 JSON Schema. v0 is intentionally permissive, but it closes field names and enums that would otherwise fragment.
- `status`: artifact lifecycle. Closed v0 values are `internal_mock_only`, `draft`, `active`, `deprecated`, and `revoked`.
- `verification_endpoint`: optional upstream endpoint for online verification, replay checks, or policy-specific acceptance decisions. Offline signature checks should still use issuer JWKS discovery when possible; do not require an online verifier for simple read-only flows.
- `accepted_capability_formats`: subset of capability formats the upstream honors. An acceptor does not need to accept every format an issuer can emit.
- `recognized_receipt_decisions`: prior receipt decisions the upstream recognizes as acceptable evidence for entry. Do not include `denied` here; denied receipts are evidence of rejection, not entry.
- `emitted_receipt_decisions`: receipt decisions the upstream can return after evaluating a capability. Acceptor v0 emits a subset of the issuer decision space: `allowed`, `denied`, and `paid`. Emitted receipts must validate against `https://satgate.io/.well-known/satgate-receipt.schema.json`.
- `trust_anchors`: issuer origins the upstream accepts, optionally pinned to JWKS URIs or future certificate transparency/audit roots. The issuer must match a trust anchor before any JWKS fetch. If `jwks_uri` is omitted, verifiers use `{issuer_id}/.well-known/jwks.json`; if present, it must be HTTPS unless a local test explicitly allows otherwise. Closed v0 statuses are `accepted`, `provisional`, `revoked`, `deprecated`, and `accepted_for_mock`; `accepted_for_mock` is reserved for examples.
- `rails_adapters.accepted`: rails the upstream will settle via or route through. The verb is deliberately different from issuer-side `rails_adapters.supported`: issuers can **support** rails for minting/routing; acceptors **accept** rails for settlement or upstream access.
- `claims`: machine-readable claim boundary. Parsers should see that acceptance means capability verification and receipt emission, not marketplace listing, reputation score, endorsement, network-wide trust, ranking, or certification.

## Decision-space note

Issuer-side receipt metadata currently defines the closed decision vocabulary `allowed`, `denied`, `delegated`, `revoked`, and `paid`.

Acceptor v0 intentionally emits a subset: `allowed`, `denied`, and `paid`. Delegation and revocation remain issuer-side authority lifecycle primitives unless a future acceptor profile explicitly defines upstream delegation or upstream revocation semantics.

## Non-goals

- No marketplace, ranking, reputation, endorsement, certification, or compliance claim.
- No fake acceptor role in the public SatGate issuer artifact.
- No requirement that SatGate stay in the request path after an upstream can verify receipts and capabilities directly.

## Interop test plan draft

The first non-SatGate acceptor verifying a SatGate-issued receipt is the point where this becomes a candidate standard rather than a vendor-only document. Before putting acceptor metadata on the wire, run an interop test with:

1. **Issuer metadata fixture**: `satgate.trust_metadata.v1` with `roles: ["issuer"]`, required `issuer` / `issuer_kid` receipt fields, closed receipt decisions, and a JWKS URI under the issuer origin.
2. **JWKS fixture**: one public key with stable `kid`; no placeholder keys.
3. **Receipt fixture**: canonical signed `satgate.receipt.v1` validating against `https://satgate.io/.well-known/satgate-receipt.schema.json` and containing `schema_version`, `receipt_id`, `evidence_pack_id`, `issuer`, `issuer_kid`, `acceptor_id`, `decision`, `decision_reason`, `policy_version`, `timestamp`, `canonicalization`, `hash_algorithm`, `signature_algorithm`, `receipt_hash`, and `signature`.
4. **Acceptor metadata fixture**: `roles: ["acceptor"]`, `schema_url`, `status`, `accepted_capability_formats`, `recognized_receipt_decisions`, `emitted_receipt_decisions`, `trust_anchors`, `rails_adapters.accepted`, and `claims`.
5. **Verifier behavior**:
   - reject receipts whose `issuer` is not in `trust_anchors`;
   - fetch JWKS from the trusted issuer origin, not from SatGate's public manifest unless SatGate is the issuer;
   - select the key by `issuer_kid`;
   - verify the canonical receipt payload and fail closed on signature, expiry, replay, or decision-policy mismatch;
   - on signature failure for a known trusted issuer, force-revalidate JWKS once before final rejection.

Minimum passing behavior is offline signature verification plus issuer trust-anchor enforcement. Online `verification_endpoint` checks may be layered on for replay-sensitive or paid-rail flows.

## Cache and freshness notes

Protocol implementations should not assume HTTP caching is perfectly honest. Corporate proxies, hosted agent runtimes, and framework fetchers may serve stale bytes even when the issuer has deployed new metadata. Candidate v1.1 fields to evaluate:

- `metadata_version`: monotonic manifest version for stale-cache detection.
- `next_rotation_at`: advisory JWKS/key-rotation timestamp.
- `refresh_after`: issuer-requested revalidation time separate from CDN TTL.

These are not v0.0 requirements; they are reserved as implementation lessons from opaque fetcher caches.

## Open questions

- Whether acceptor metadata belongs under the same `satgate.trust_metadata.v1` schema or a sibling acceptor schema.
- Whether `trust_anchors` should support key pinning, issuer catalogs, or both.
- Whether online verification endpoints should be required for paid rails or only recommended for replay-sensitive routes.
- Whether stale-cache detection belongs in issuer metadata, acceptor metadata, JWKS metadata, or all three.

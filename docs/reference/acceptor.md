# SatGate Acceptor Metadata Draft

Status: **v0.0 draft — not yet on the wire**.

This document reserves the mirror side of the SatGate trust metadata model before issuer-side `satgate.trust_metadata.v1` ossifies in downstream parsers. It is intentionally documentation-only until an upstream service actually accepts SatGate capabilities. The public badge and integration story live in [Accept SatGate Capabilities](accept-satgate-capabilities.md).

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
  "emitted_receipt_decisions": [
    "allowed",
    "denied",
    "delegated",
    "revoked",
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
- `accepted_receipt_decisions`: subset of receipt decisions the upstream treats as acceptable evidence for entry. Do not include `denied` here; denied receipts are evidence of rejection, not entry.
- `emitted_receipt_decisions`: subset of receipt decisions the upstream can return after evaluating a capability, including `denied` when a request is rejected before execution.
- `trust_anchors`: issuer origins the upstream accepts, optionally pinned to JWKS URIs or future certificate transparency/audit roots.
- `rails_adapters.accepted`: rails the upstream can settle on or route through.

## Non-goals

- No marketplace, ranking, reputation, endorsement, or compliance claim.
- No fake acceptor role in the public SatGate issuer artifact.
- No requirement that SatGate stay in the request path after an upstream can verify receipts and capabilities directly.


## Interop test plan draft

The first non-SatGate acceptor verifying a SatGate-issued receipt is the point where this becomes a candidate standard rather than a vendor-only document. Before putting acceptor metadata on the wire, run an interop test with:

1. **Issuer metadata fixture**: `satgate.trust_metadata.v1` with `roles: ["issuer"]`, required `issuer` / `issuer_kid` receipt fields, closed receipt decisions, and a JWKS URI under the issuer origin.
2. **JWKS fixture**: one public key with stable `kid`; no placeholder keys.
3. **Receipt fixture**: canonical signed `satgate.receipt.v1` containing `receipt_id`, `evidence_pack_id`, `issuer`, `issuer_kid`, `decision`, `decision_reason`, `policy_version`, `receipt_hash`, and `signature`.
4. **Acceptor metadata fixture**: `roles: ["acceptor"]`, `accepted_capability_formats`, `accepted_receipt_decisions`, `trust_anchors`, and `rails_adapters.accepted`.
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

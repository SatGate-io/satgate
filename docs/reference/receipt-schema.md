# SatGate Receipt Schema

Status: **v1 canonical schema — on the wire**.

Canonical schema URL: `https://satgate.io/.well-known/satgate-receipt.schema.json`

SatGate receipts are signed decision artifacts. They are the proof spine for authority decisions: allowed, denied, delegated, revoked, and paid. Issuer metadata advertises `satgate.receipt.v1`; acceptor metadata can emit receipt subsets that still validate against this schema.

## Required fields

Every `satgate.receipt.v1` receipt must include:

- `schema_version`: `satgate.receipt.v1`
- `schema_url`: `https://satgate.io/.well-known/satgate-receipt.schema.json`
- `receipt_id`: stable receipt identifier
- `evidence_pack_id`: Evidence Pack this receipt rolls into
- `issuer`: trusted issuer origin used for JWKS discovery after trust-anchor validation
- `issuer_kid`: issuer signing-key id used to select from issuer JWKS
- `decision`: one of `allowed`, `denied`, `delegated`, `revoked`, `paid`
- `decision_reason`: stable machine-readable explanation
- `policy_version`: policy version evaluated for this decision
- `timestamp`: RFC 3339 / JSON Schema `date-time`
- `canonicalization`: `jcs-rfc8785`
- `hash_algorithm`: `sha256`
- `signature_algorithm`: `ed25519`
- `receipt_hash`: SHA-256 hash of the canonical payload, prefixed with `sha256:`
- `signature`: Ed25519 issuer signature, prefixed with `ed25519:`

A receipt must also include at least one capability binding:

- `capability_id`, or
- `capability_hash`

Acceptor-bound receipts should include both:

- `acceptor_id`
- `capability_hash`

## Decision vocabulary

The global receipt decision enum is closed for v1:

```json
["allowed", "denied", "delegated", "revoked", "paid"]
```

Specific profiles can emit a subset. For example, acceptor metadata v0 emits only `allowed`, `denied`, and `paid`; delegation and revocation remain issuer-side authority lifecycle decisions unless a future acceptor profile defines upstream delegation/revocation semantics.

## Verification sequence

1. Parse the receipt as JSON.
2. Validate against `https://satgate.io/.well-known/satgate-receipt.schema.json`.
3. Validate the `issuer` against the verifier's trust anchors before any JWKS fetch.
4. Fetch JWKS from `{issuer}/.well-known/jwks.json` or an explicitly pinned trusted JWKS URI.
5. Select the key by `issuer_kid`.
6. Rebuild the canonical payload using RFC 8785 JSON Canonicalization Scheme (JCS), excluding `receipt_hash` and `signature`.
7. Compute SHA-256 over those canonical payload bytes and compare it to `receipt_hash`.
8. Verify the Ed25519 `signature` over the canonical payload bytes with the issuer JWK selected by `issuer_kid`. The selected JWK must be Ed25519-compatible.
9. Apply verifier policy for expiry, replay, audience, route/tool scope, budget, mock fixtures, and accepted decision subset.
10. On signature failure for a known trusted issuer, force-refresh JWKS once before final rejection.


## Example acceptor-bound receipt

```json
{
  "schema_version": "satgate.receipt.v1",
  "schema_url": "https://satgate.io/.well-known/satgate-receipt.schema.json",
  "receipt_id": "rcpt_mock_accept_001",
  "evidence_pack_id": "ep_mock_accept_001",
  "issuer": "https://satgate.io",
  "issuer_kid": "satgate-mock-2026-05",
  "acceptor_id": "https://api.internal.example/satgate-mock",
  "decision": "allowed",
  "decision_reason": "capability_scope_audience_and_budget_ok",
  "policy_version": "policy_mock_acceptance_v0",
  "capability_hash": "sha256:mock_capability_hash",
  "timestamp": "2026-05-13T12:00:00Z",
  "canonicalization": "jcs-rfc8785",
  "hash_algorithm": "sha256",
  "signature_algorithm": "ed25519",
  "receipt_hash": "sha256:mock_receipt_hash",
  "signature": "ed25519:mock_signature_not_for_production",
  "mock_only": true
}
```

## Schema design notes

- The schema keeps `additionalProperties: true` so rails, billing systems, and Evidence Pack implementations can add contextual fields without breaking v1 consumers.
- The dangerous parts are closed: `schema_version`, `decision`, origin-style `issuer`, canonicalization/hash/signature algorithms, signature/hash prefixes, and required proof fields.
- `issuer` must be an HTTPS origin: no path, query, or fragment. Verifiers must reject non-origin issuers before JWKS lookup.
- `paid` receipts require `amount_usd`, `currency: "USD"`, and `rail` so paid-call evidence does not lose payment context.
- If `acceptor_id` is present, `capability_hash` is required so acceptor-bound receipts are tied to the capability material without leaking raw credentials.

## Related artifacts

- Issuer metadata: `https://satgate.io/.well-known/satgate`
- Issuer schema: `https://satgate.io/.well-known/satgate.schema.json`
- Issuer JWKS: `https://satgate.io/.well-known/jwks.json`
- Acceptor schema: `https://satgate.io/.well-known/satgate-acceptor.schema.json`
- Mock acceptor receipt: `https://satgate.io/examples/mock-accepted-satgate-receipt.v1.json`

## Production verifier notes

- Reject `mock_only: true` outside explicit fixture/test mode.
- Reject unknown critical extensions if a future profile introduces a critical-extension mechanism; otherwise ignore unknown non-critical fields.
- `schema_url` is required and must equal `https://satgate.io/.well-known/satgate-receipt.schema.json`.

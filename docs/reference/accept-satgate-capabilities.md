# Accept SatGate Capabilities

Status: **acceptance story v0 — internal/mock example available; no public marketplace claim**.

This page defines what an upstream API may honestly mean by **Accepts SatGate capabilities**.

Acceptance means the upstream verifies a SatGate-scoped capability before execution and returns a SatGate-compatible receipt after the decision. It does **not** mean SatGate operates a marketplace, assigns network-wide reputation, endorses the upstream, or ranks the upstream against others.

## Badge and copy

### Badge

```text
Accepts SatGate capabilities
```

### One-line explanation

```text
Verifies scoped SatGate capabilities and returns SatGate-compatible receipts.
```

### Required disclaimer

```text
Acceptance means capability verification and receipt emission. It is not a marketplace listing, reputation score, ranking, or SatGate endorsement.
```

### Forbidden copy

Do not use:

- `SatGate trusted marketplace member`
- `SatGate certified reputation score`
- `SatGate endorsed API`
- `Network-approved upstream`
- `SatGate verified vendor`

Those phrases overclaim. They imply reputation or endorsement, not capability acceptance.

## Minimal integration checklist

An upstream can claim **Accepts SatGate capabilities** only when it can demonstrate all of these:

1. **Acceptor identity**: the upstream has a stable `acceptor_id` and contact path.
2. **Capability input**: the upstream accepts `Authorization: Bearer <capability>` using `satgate.capability.v1` or `macaroon-bearer`.
3. **Trust anchors**: the upstream checks the receipt/capability issuer against configured `trust_anchors` before JWKS discovery.
4. **Issuer JWKS discovery**: the upstream fetches JWKS from `{issuer_id}/.well-known/jwks.json`, selected only after issuer trust-anchor validation.
5. **Capability checks**: the upstream verifies signature, expiry, audience, route/tool scope, caveats, budget, and delegation depth before execution.
6. **Bounded execution**: the upstream executes only the action authorized by the capability.
7. **Receipt emission**: every accepted or rejected call returns a SatGate-compatible receipt with at least `receipt_id`, `evidence_pack_id`, `issuer`, `issuer_kid`, `acceptor_id`, `decision`, `decision_reason`, `policy_version`, `receipt_hash`, and `signature`.
8. **Honest claim boundary**: public copy says acceptance means verification + receipts, not marketplace reputation.

## Verification criteria

A third-party reviewer should be able to reproduce these outcomes:

### Positive path

- Given a valid scoped capability for the upstream audience and route,
- when the caller invokes the upstream with that capability,
- then the upstream verifies issuer trust, verifies the capability, executes only the scoped action, and returns an `allowed` or `paid` receipt.

### Negative path

- Given an expired, wrong-audience, wrong-route, over-budget, or untrusted-issuer capability,
- when the caller invokes the upstream,
- then the upstream rejects before execution and returns a receipt-grade `denied` decision.

### Receipt path

Returned receipts must include:

```json
{
  "receipt_id": "rcpt_...",
  "evidence_pack_id": "ep_...",
  "issuer": "https://satgate.io",
  "issuer_kid": "kid_...",
  "acceptor_id": "https://api.example.com",
  "decision": "allowed",
  "decision_reason": "capability_scope_audience_and_budget_ok",
  "policy_version": "policy_...",
  "receipt_hash": "sha256:...",
  "signature": "ed25519:..."
}
```

### Claim boundary

The upstream must not claim marketplace membership, global trust, ranking, endorsement, certification, or network-wide reputation.

## Mock/internal example

Until a real upstream is ready, SatGate publishes a mock-only acceptor example:

- Mock acceptor metadata: `/examples/mock-acceptor-metadata.v0.json`
- Mock accepted receipt: `/examples/mock-accepted-satgate-receipt.v1.json`
- Public explainer: `https://satgate.io/accept-satgate-capabilities`
- Acceptor JSON Schema: `https://satgate.io/.well-known/satgate-acceptor.schema.json`

The mock proves the integration shape. It is not a live upstream, production acceptor, or public network adoption claim.

The mock metadata separates `recognized_receipt_decisions` from `emitted_receipt_decisions`: `denied` can be emitted as rejection evidence, but it is not recognized as evidence for entry.

## Example request shape

```http
POST /v1/research HTTP/1.1
Host: api.example.com
Authorization: Bearer cap_...
Content-Type: application/json

{
  "query": "current supplier pricing"
}
```

## Example response shape

```json
{
  "result": {
    "summary": "..."
  },
  "satgate_receipt": {
    "receipt_id": "rcpt_mock_accept_001",
    "evidence_pack_id": "ep_mock_accept_001",
    "issuer": "https://satgate.io",
    "issuer_kid": "satgate-mock-2026-05",
    "acceptor_id": "https://api.internal.example/satgate-mock",
    "decision": "allowed",
    "decision_reason": "capability_scope_audience_and_budget_ok",
    "policy_version": "policy_mock_acceptance_v0",
    "receipt_hash": "sha256:mock_receipt_hash",
    "signature": "ed25519:mock_signature_not_for_production"
  }
}
```

## Relationship to acceptor metadata

The acceptor-side metadata draft lives at [`acceptor.md`](acceptor.md). The v0 schema lives at `https://satgate.io/.well-known/satgate-acceptor.schema.json`. This page defines the public badge and minimum integration story; the acceptor metadata draft and schema define the machine-readable shape.

# Reputation Substrate

Status: **internal substrate only — no public reputation launch**.

This document inventories the receipt fields, event history, and verifier evidence needed to support future trust or reputation analysis without minting a public score, ranking, marketplace claim, certification, or endorsement.

The substrate inventory covers: stayed within budget, completed requested task, looped or retried excessively, was revoked or attenuated, was vouched by a principal, and was accepted by upstream.

Principle: **reputation falls out of signed receipts and verifier-observed history; SatGate should not mint reputation by assertion.**

A future reputation view is only credible if it is derived from real transaction history: signed `satgate.receipt.v1` receipts, issuer/acceptor trust metadata, Evidence Pack event history, and verifier policy. Until that history exists, SatGate should expose proof artifacts and verification criteria — not scores.

## Non-goals

SatGate must not publicly launch any of the following without real transaction history and explicit product approval:

- reputation score
- trust score
- ranking
- marketplace listing
- certification
- SatGate endorsement
- network-wide trust badge
- public directory ordered by quality or trust

`Accepts SatGate capabilities` remains the correct public phrase. It means scoped capability verification plus SatGate-compatible receipt emission. It does not mean the upstream is trusted, ranked, certified, or vouched for by SatGate.

## Questions the substrate must eventually answer

### Stayed within budget

Verifier question: did the subject remain within the budget envelope that applied at decision time?

Evidence needed:

- `receipt_id`
- `evidence_pack_id`
- `subject` or `agent_id`
- `budget_id`
- `budget_limit_usd`
- `amount_usd` or `attempted_amount_usd`
- `remaining_budget_usd`
- `decision`
- `decision_reason`
- `policy_version`
- `timestamp`
- Evidence Pack `budget_snapshot`
- event history entries: `budget_observed`, `budget_denied`

Interpretation boundary: “stayed within budget” is a verifier conclusion over receipts and budget snapshots, not a global claim about the subject.

### Completed requested task

Verifier question: did the subject complete the requested work, or merely receive authority to attempt it?

Evidence needed:

- `task_id`
- `task_status`
- `receipt_id`
- `decision`
- `decision_reason`
- `route_or_tool`
- `timestamp`
- optional upstream or principal completion receipt
- event history entries: `task_requested`, `task_started`, `task_completed`, `task_failed`

Interpretation boundary: an `allowed` receipt proves authority was granted; it does not by itself prove task completion.

### Looped or retried excessively

Verifier question: did the subject retry/loop beyond normal policy or task limits?

Evidence needed:

- `task_id`
- `attempt`
- `max_attempts`
- `retry_of_receipt_id`
- `parent_receipt_id`
- `decision_reason`
- `timestamp`
- ordered event history with event hashes or receipt chain references
- event history entries: `attempt`, `retry`

Interpretation boundary: loop detection requires event sequence history. A single receipt can carry attempt context, but cannot prove absence of looping alone.

### Was revoked or attenuated

Verifier question: was the capability revoked, delegated, or attenuated before/at the decision?

Evidence needed:

- `capability_id` or `capability_hash`
- `parent_receipt_id`
- `revoked_receipt_id`
- `attenuation_depth`
- `caveats_hash`
- `decision`: `delegated` or `revoked` where applicable
- `decision_reason`
- `policy_version`
- Evidence Pack `authority_chain`
- event history entries: `delegated`, `attenuated`, `revoked`

Interpretation boundary: revocation/attenuation is authority lifecycle evidence. It is not reputation by itself.

### Was vouched by a principal

Verifier question: did a human, service owner, or principal explicitly vouch for this capability/agent/upstream?

Evidence needed:

- `principal_id`
- `principal_authorization_id`
- `vouch_receipt_id`
- `subject` or `agent_id`
- `capability_id` or `capability_hash`
- `timestamp`
- signature/JWKS verification for the principal or issuer path
- event history entry: `principal_vouched`

Interpretation boundary: a vouch is scoped evidence from a named principal. It is not a SatGate endorsement unless SatGate is the principal and explicitly says so in a signed receipt.

### Was accepted by upstream

Verifier question: did an upstream actually accept a SatGate capability and emit a compatible receipt?

Evidence needed:

- `acceptor_id`
- `issuer`
- `issuer_kid`
- `capability_hash`
- `decision`: `allowed`, `denied`, or `paid` in acceptor v0
- `receipt_hash`
- `signature`
- acceptor metadata trust anchors
- event history entries: `upstream_accepted`, `upstream_denied`

Interpretation boundary: upstream acceptance proves that one integration path verified authority and emitted a receipt. It does not prove global upstream quality, safety, or ranking.

## Principal and vouch semantics

`principal_id` has privacy weight. It is a scoped evidence handle, not a portable behavioral identity. Verifiers MAY correlate `principal_id` across receipts inside the same tenant, acceptor, Evidence Pack, audit export, or explicit verifier trust domain. Verifiers MUST NOT treat `principal_id` as permission for cross-acceptor behavioral tracking unless the receipts were collected under a disclosed common controller, shared audit mandate, or explicit principal consent.

Preferred emitters should use pairwise or tenant-scoped principal identifiers where possible. A public acceptor should not expose stable human identifiers in receipts unless the principal intentionally chose that identifier for audit use.

`vouch_receipt_id` and `principal_vouched` are reputation-adjacent evidence primitives. A vouch means: a named principal made a scoped, signed assertion that a subject/capability/upstream is acceptable for a specific purpose, policy, tenant, and time window. It does not mean global trust.

Vouching rules for future profiles:

- A vouch must point to a signed receipt or authorization event; unsigned labels are not vouches.
- A vouch must declare scope: subject, capability or upstream, policy/purpose, issuer, and expiry or review window.
- Self-vouch is not valid for external verifier conclusions unless a profile explicitly labels it as self-attestation.
- Vouches must be revocable; revocation should reference the original `vouch_receipt_id` and emit a `revoked` receipt or `revoked` event.
- Counting vouches is a third-party reputation derivation. SatGate should not publish counts, tiers, badges, rankings, or scores from vouches without a later explicit reputation product decision.

## Additive schema optionality

`/.well-known/satgate-receipt.schema.json` remains backward-compatible. The substrate adds optional field names so future emitters do not invent incompatible names:

- `task_id`
- `task_status`
- `attempt`
- `max_attempts`
- `retry_of_receipt_id`
- `parent_receipt_id`
- `budget_id`
- `budget_limit_usd`
- `principal_id`
- `principal_authorization_id`
- `vouch_receipt_id`
- `attenuation_depth`
- `caveats_hash`
- `revoked_receipt_id`
- `event_history_ref`

None of these fields are required in v1. They preserve future optionality without breaking existing receipts. Acceptor metadata can advertise `emitted_receipt_fields` so optional substrate fields remain per-acceptor expectations instead of drifting into implicit global requirements.

Evidence Pack v1 likewise keeps an optional `event_history` array for verifier-observed history. Event history can answer sequence questions that a single receipt cannot: looping, retry counts, task lifecycle, post-revocation denials, and upstream acceptance over time.

## Verifier posture

A verifier can compute local conclusions from receipts and events, such as:

- “within budget for policy X during window Y”
- “completed task Z under principal P”
- “retried N times and stayed under max_attempts M”
- “denied after revocation receipt R”
- “accepted by upstream A under trust-anchor set T”

Those are evidence-derived conclusions. They should stay local or auditor-facing until there is real transaction history, clear scope, and an explicit product decision to expose any aggregate view.

## Guardrail language

Allowed:

- “receipt-derived evidence”
- “verifier-observed history”
- “Evidence Pack event history”
- “acceptance proves verification behavior”
- “reputation falls out of signed receipts”

Forbidden in public launch copy:

- “SatGate reputation score”
- “trust score”
- “ranked upstreams”
- “certified acceptor”
- “trusted marketplace”
- “SatGate endorsed”
- “network-wide reputation”
- “rating”
- “agent rating”
- “trust rating”
- “reputation rating”
- “tier”
- “trust tier”
- “reputation tier”
- “agent score”
- “credit score”
- “verified agent rating” when it implies endorsement rather than protocol verification
- “preferred upstream”

## Implementation rule

If a future UI wants to show any reputation-like summary, it must first point to the receipts, event history window, verifier policy, and trust anchors used to compute it. No receipts, no score. No observed history, no ranking. No signed principal vouch, no vouch claim.

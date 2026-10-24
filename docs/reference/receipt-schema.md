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

Gateway receipts use a closed `decision_reason` vocabulary. The public verifier accepts these values and rejects any other reason:

- `auth_missing`
- `budget_authorized`
- `budget_exhausted`
- `budget_backend_error`
- `capability_expired`
- `capability_invalid`
- `insufficient_budget`
- `observe_projected`
- `observe_unrecorded`
- `pending_capacity_exhausted`
- `payment_required`
- `payment_verified`
- `policy_allowed`
- `policy_denied`
- `sandbox_no_spend`
- `token_revoked`
- `tool_price_unavailable`
- `tool_controls_unavailable`

The four unavailable-check reasons require checker bundle `unavailable-check-v1`
(released at `40b22c58e6c62d4c4def1aa487035fcd52da73e5`) or a matching newer
reader. Older checkers reject them even when their signatures are valid.
The public MCP proxy produces them only when both `SetSignEveryAction(true)`
and the separate `SetSignUnavailableCheckRefusals(true)` are set before serving.
The latter defaults to false, including after a module upgrade: existing
sign-every-action deployments do not implicitly opt in to new reason vocabulary.
Cloud enablement is a separate rollout step, not enabled by this code release.

### Unavailable-check refusal (explicit producer opt-in)

Each reason requires `decision: denied`, a verified key, and no evaluated budget:

| reason | meaning |
| --- | --- |
| `tool_price_unavailable` | A required checked price lookup failed; the call was not sent. No fallback price is claimed. |
| `tool_controls_unavailable` | Required spending/approval controls could not be checked, were absent, or returned an invalid admission; not a limit or approval denial. |
| `budget_backend_error` | A required fail-closed budget/hold backend operation failed; not budget exhaustion or a known failed-store return value. |
| `pending_capacity_exhausted` | The local Control pending reservation could not be made before a hold; not evidence the customer's funds were exhausted. |

`budget` and the Pack's `budget_state` are exactly
`{"spend_mode":"not_evaluated","cost_credits":0}` (integer zero, no extra keys).
Authority is `verified_macaroon_caveats`, with a nonempty `capability_hash` bound
to the receipt's, only the verified-capability authority fields, and all four raw
capability/token/budget/parent fields present as `redacted`. Metadata and policy
are objects. The Pack must carry matching `authority`, `budget` and `policy`
mirrors, recursively equal in concrete JSON type and value. The receipt and
wrapper cannot make balance, attempted, remaining, limit, projected, spend or
payment claims at any depth outside the exact budget mirrors. Secret scanning,
signature, time, hash and issuer trust checks still apply. These are per-call
refusals, never rate-window or Observe gap pairs; window/gap fields are rejected.
Existing profiles retain their existing behavior. Signer, identity or archive
failure cannot be proven by signing a new receipt and remains unsigned.

Price refusals precede controls, reservation and spend. Invalid control admissions
are settled before signing; real spending-limit and approval denials keep their
policy meaning. Budget backend receipts are produced only for the fail-closed
blocking branch, never Observe or configured fail-open Control. Pending-capacity
receipts are Control-only, after proof preflight and before a hold, without
reserving an additional pending slot. Missing hold support and preparation failure
after a hold remain unsigned. A possibly persisted allowed receipt or
`outcome_unknown` is never reinterpreted as a refusal. With both switches on, a
transport failure after admission returns the existing admission evidence handles
on its existing error, not a second or outage receipt. With the new switch off,
all these responses and receipts retain their previous bytes and meaning.
Disabling the setting stops future emission; it never deletes or rewrites archived
receipts. Keep the matching checker bundle available after rollback.

## Receipts when sign-every-action is on

A gateway can be run with a switch (`Proxy.SetSignEveryAction` in the open-source MCP proxy; the hosted gateways wire their own setting; off by default) that makes it sign a receipt for each call made with a key SatGate issued, whether the call is let through or refused. While the switch is off, only the calls that had a receipt before get one, and nothing below is promised. While it is on, a call whose receipt cannot be made is not passed on, and nothing is charged for it: the caller gets `proof_unavailable`. A recorder that answers with nothing counts as a receipt that could not be made. A refusal that cannot be recorded is answered `proof_unavailable` as well, in place of the usual refusal. Observe is the one exception: the call goes through and the miss is counted (see the gap receipt below). The switch needs a recorder that keeps refusal windows (`WindowRecorder`); turning it on without one is a setup error. With the switch on, the `request_id` of a receipt is made by the server for that one call (random, never built from the caller's JSON-RPC id, session or request id), so two callers that number their requests alike get receipts that can be told apart.

What counts as a call: a request that arrives with a key that verifies. What does not: the connection handshake (`initialize`, `ping`, notifications), listing tools, a request with no key or a key SatGate did not issue. A request stopped because a required check was unavailable stays unsigned unless the separate unavailable-check setting above is also on. Signer, identity and archive failures remain unsigned in either case.

Three sign-every-action shapes carry this, plus the separately opted-in unavailable-check refusal above. None changes how an older receipt is read: each check below applies only to a receipt that has the shape.

### A call that was allowed and read no balance

`decision` is `allowed`, `decision_reason` is `policy_allowed`, and `budget` is exactly `{"spend_mode": "not_evaluated", "cost_credits": 0}`. A free tool, or a key with no budget, never reads a balance, so the receipt claims none. It names the tool or method, the key (hashed), the time and the reason, and never the call's arguments.

The verifier refuses such a receipt if a limit, remaining, attempted, used, spent or projected amount, a balance, or any payment or spend field appears anywhere in it, at any depth (in `metadata`, `policy`, `authority` or any extension; outside that one `budget` object), or if `budget` is anything but the object above. It also refuses one whose authority does not name a verified key, and a Pack wrapper that carries such a field, a `budget_state` other than the object above, or that leaves out its `authority`, `budget` or `policy` copy of the signed receipt (both verifier copies require all three for these receipts). The two refusal windows and the gap receipt below are held to the same rule. An `allowed` / `policy_allowed` receipt whose budget was evaluated (it states balances, as the HTTP gateway signs today) is still read as before.

### A refusal that repeats: the opener and the closer

A key that keeps being refused (the rate limit says no, or a key that verified is revoked or has expired) does not get a receipt for every refused call. Per key, tool and UTC minute SatGate signs two receipts:

- the opener, when the first call of that minute is refused;
- the closer, once the minute is over (plus a 15 second grace), carrying how many calls were refused in it.

Both are denials (`denied` / `policy_denied`, or `denied` / `token_revoked` or `capability_expired` for a revoked or expired key) with a `denial_code` (for example `RATE_LIMITED`) and these signed top-level fields:

| field | meaning |
| --- | --- |
| `rate_limit_phase` | `opened` or `closed` |
| `rate_limit_window_start` | the minute, UTC, written `YYYY-MM-DDTHH:MM:00Z` |
| `rate_limit_window_seconds` | `60` |
| `rate_limit_refused_calls` | the closer only: refused calls in the minute, the opener included, a whole number of at least 1 |

The verifier refuses a window receipt with another phase, a start that is not a UTC minute, a length other than 60, no denial code, a count on an opener, no count (or a count below 1, or not a whole number) on a closer, a closer issued before its minute ended, or any balance claim. A burst of any size costs two signatures a minute.

### A gap: calls that went through in Observe with no receipt

Observe never stops a call. If SatGate cannot make the receipt for an Observe call, the call still goes through and SatGate counts it. When signing works again it signs one gap receipt per key, tool and UTC minute: `decision` `allowed`, `decision_reason` `observe_unrecorded`, with these signed top-level fields:

| field | meaning |
| --- | --- |
| `unrecorded_window_start` | the minute, UTC, written `YYYY-MM-DDTHH:MM:00Z` |
| `unrecorded_window_seconds` | `60` |
| `unrecorded_calls` | how many calls went through with no receipt, a whole number of at least 1 |

The gap receipt makes no balance claim (same budget object as above). The verifier refuses one with no window, no count, a count below 1 or not a whole number, a window that is not a UTC minute, one issued before its minute ended, or an `observe_unrecorded` reason on a denial. A Pack of a gap receipt must repeat the receipt's `issued_at` and `evidence_url`, equal to the signed ones; both verifier copies refuse one that leaves either out (for the other new receipts and for every older one, those two stay optional in `tools/verify_evidence_pack.py`, as before). A record that holds a gap receipt is complete, and it says in writing that the calls it counts have no receipt of their own.

### Checking a whole record

Download the record checker together with its exact sibling pack checker,
requirements, contract and hashes as bundle `unavailable-check-v1`; see
[`tools/README.md`](../../tools/README.md#versioned-checker-bundle).
Updating only the record script does not teach it new reasons. A saved older
checker still rejects these reasons; a bundle/version field in an untrusted
record cannot negotiate support or authorize downloading/executing a checker.

`tools/verify_receipt_record.py` checks a downloaded record: one Evidence Pack per line, then a signed complete manifest or cut-off terminal. It runs the per-pack check on every line, checks the terminal's signature and closed field set (unknown fields fail), that the count and the receipt hashes in order match (so nothing was dropped, added, changed or moved), that every receipt is in the terminal's time range, in order, not repeated, from the same issuer, and belongs to the one agent key the terminal names, and that the refusal windows and counts by reason match. Each receipt and the terminal select their own signing key by `issuer_kid`; with `--require-trusted-issuer`, all keys must be in the issuer JWKS supplied by `--jwks-file` or `--jwks-url`. Embedded-key mode, including mixed keys, proves self-consistency only and says so in output. A record line holds exactly one receipt. In a complete record, an overdue window closer makes the record invalid; a closer not yet due, or a window running past the range, is pending or boundary. A signed `satgate.receipt_record_truncated.v1` terminal is checked against its prefix, reports `status: truncated`, its reason and `verified_prefix`, and exits 4 with `valid` and `complete` false; an unclosed prefix window is `prefix_open`. Exit codes are 0 complete, 1 invalid record, 2 input/JWKS loading or usage error, 3 missing cryptography, 4 verified truncation. See `tools/VERIFIER-CONTRACT.md` for the field sets and limits of these claims.

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
  "timestamp": "2026-05-13T00:00:00Z",
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
- Additive context is not negotiation of new reasons or critical semantics. Gateway reasons remain closed; unknown reasons, schemas and record lines fail, not silently skip or become unsupported warnings. The no-budget profile's recursive claim restrictions override contextual optionality.
- The dangerous parts are closed: `schema_version`, `decision`, origin-style `issuer`, canonicalization/hash/signature algorithms, signature/hash prefixes, and required proof fields.
- `issuer` must be an HTTPS origin: no path, query, or fragment. Verifiers must reject non-origin issuers before JWKS lookup.
- `paid` receipts require `amount_usd`, `currency: "USD"`, and `rail` so paid-call evidence does not lose payment context.
- If `acceptor_id` is present, `capability_hash` is required so acceptor-bound receipts are tied to the capability material without leaking raw credentials.

## Optional verifier-context fields

Receipt v1 is intentionally additive. The following optional fields preserve future verifier/audit optionality without launching public reputation, scores, rankings, or marketplace semantics:

- `task_id`, `task_status` — correlate receipts to a requested task and observed completion state.
- `attempt`, `max_attempts`, `retry_of_receipt_id`, `parent_receipt_id` — support loop/retry analysis across event history.
- `budget_id`, `budget_limit_usd` — bind receipts to the budget envelope used for stayed-within-budget conclusions.
- `principal_id`, `principal_authorization_id`, `vouch_receipt_id` — preserve evidence that a named principal vouched for a subject or capability.
- `attenuation_depth`, `caveats_hash`, `revoked_receipt_id` — preserve delegation, attenuation, caveat, and revocation lineage without leaking raw credentials.
- `event_history_ref` — link to Evidence Pack event history when sequence evidence is required.

These fields are evidence handles. They do not create a reputation claim by themselves. Verifier conclusions must be derived from signed receipts, event history, trust anchors, and verifier policy. See [Reputation Substrate](reputation-substrate.md).

## Related artifacts

- Issuer metadata: `https://satgate.io/.well-known/satgate`
- Issuer schema: `https://satgate.io/.well-known/satgate.schema.json`
- Issuer JWKS: `https://api.satgate.io/.well-known/jwks.json` (`https://satgate.io/.well-known/jwks.json` redirects there)
- Acceptor schema: `https://satgate.io/.well-known/satgate-acceptor.schema.json`
- Mock acceptor receipt: `https://satgate.io/examples/mock-accepted-satgate-receipt.v1.json`

## Production verifier notes

- Verifiers MUST reject `mock_only: true` receipts when operating in a production context.
- Reject unknown critical extensions if a future profile introduces a critical-extension mechanism; otherwise ignore unknown non-critical fields.
- `schema_url` is required and must equal `https://satgate.io/.well-known/satgate-receipt.schema.json`.

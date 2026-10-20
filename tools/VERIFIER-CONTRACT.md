# Public verifier validation contract

`tools/verify_evidence_pack.py` and the standalone distributed
`demo/verify_evidence_pack.py` share the following paid-rail contract. The shared
synthetic Ed25519-signed corpus in `tools/test_paid_rail.py` exercises both APIs
and actual CLIs. No real payment, private Pack, or bearer locator is a fixture.

## Paid rail

- `decision_reason=payment_verified` permits `decision=allowed` or `paid` only.
- Rail is `l402`. Payment, invoice, and macaroon hashes are nonblank strings.
- Lightning sats are integral: `amount_sats` must be a positive JSON integer
  within RFC8785's safe integer domain, `1..9007199254740991`. Boolean, string,
  null, zero, negative, fractional, and floating-point representations are
  rejected. This deliberately rejects `1.0`, `100.0`, and exponent-form `1e0`
  as well as `1.5`; it tightens the former tools float acceptance to match demo.
- `paid_rail_context` contains exactly `rail`, `payment_hash`, `invoice_hash`,
  `macaroon_hash`, and `amount_sats`, matching the signed root fields in value
  and concrete JSON type. Budget is exactly `spend_mode=paid_rail`, `rail=l402`,
  and the same integer `amount_sats`.
- Credit/USD field names are forbidden recursively, even with null values.
  If any receipt is paid-rail, the Pack cannot carry those names or a top-level
  `budget_state` (even null).

## Signed mirrors and untrusted input

Mirror comparisons are recursive and type-strict, including objects and arrays:
`1`, `1.0`, and `true` are not interchangeable. A recomputed unsigned Pack hash
cannot bypass a signed mirror mismatch. Authority, policy, budget, core identity
mirrors, and supported budget-state views retain their profile-specific presence
rules, but not Python's coercing equality.

Optional top-level `rail`, `amount_sats`, `payment_hash`, `invoice_hash`,
`macaroon_hash`, and `paid_rail_context` are supported display mirrors, not new
claims. When present they must have an exactly matching signed counterpart.
The same binding applies to optional `issued_at`, `evidence_url`, `timestamp`,
`policy_version`, `jwks_url`, `schema_url`, and `verify_url`. Demo retains its
existing required `issued_at`/`evidence_url` mirror checks; tools does not newly
require their presence. Mirror mismatches emit `pack_primary_mismatch` with the
field name; budget-state mismatches emit `budget_state_mismatch`.

The entire input, including unsigned extensions, must be RFC8785-serializable.
Nonfinite numbers (including overflow from valid JSON `1e309`), unsafe integers,
invalid Unicode, invalid shapes, and expected parser/canonicalization failures
produce structured `valid=false` verdicts. CLI failures exit nonzero, without a
traceback. Input errors do not echo attacker-controlled values. This is input
validation, not a general memory/CPU resource sandbox for arbitrarily large
artifacts.

## Temporal validation

Every receipt, including secondary receipts, must pass temporal validation.
Absent or JSON-null `expires_at` deliberately means no expiration constraint.
A supplied non-null `expires_at` must parse successfully; malformed values
(including empty strings and non-string values) fail that receipt's time check,
the aggregate `time_valid` check, and the Pack verdict even when signatures pass.
Parsed expiration earlier than the evaluation time remains `receipt_expired`;
equality is not expired. Existing issued-at skew and timestamp rules are retained.

Omitted API `now` (or `None`) and omitted CLI `--now` use the current UTC clock.
An explicitly supplied non-null API `now` or CLI `--now` must parse successfully,
including when supplied as an empty string. Parse failure returns structured
`valid=false`, `checks.time_valid=false` before evaluating receipts, with a
nonzero CLI exit. Existing parser reason codes (`missing_now`/`malformed_now`,
`missing_expires_at`/`malformed_expires_at`) are retained; diagnostics alone are
never a substitute for a failed temporal verdict. Timestamp parser grammar is
unchanged by this correction.

The shared synthetic signed corpus checks both APIs and actual CLIs with absent,
null, future, boundary, expired, and malformed expiration on primary and secondary
receipts, plus malformed explicit clocks and default-clock compatibility.

## Compatibility boundaries retained

Tools defaults to embedded-key artifact integrity; issuer anchoring is explicit
with `--require-trusted-issuer` and supplied/discovered JWKS. Demo CLI defaults
to issuer anchoring and automatic discovery, with `--allow-embedded-key` as its
explicit opt-out and warning. Existing flag precedence is unchanged. The API
`verify_pack` default remains artifact-integrity mode in both copies.

Demo has richer paired decision/reason profiles and provenance validation. Tools
validates non-paid enum membership rather than adopting demo's paired-profile
matrix (for example, the historical tools acceptance and demo rejection of
paid / budget_authorized are covered). Both surfaces accept every
`decision_reason` the gateway writes. Unknown reasons still fail. These
differences are legacy behavior, not a claim that the two surfaces are fully
equivalent outside paid rail.

If the `cryptography` package cannot be imported, both CLIs print that
signatures could not be checked and how to install the package, then exit 3.
That is not a receipt verdict: exit 0 is a valid pack, exit 1 is an invalid
pack, and exit 2 is input that could not be loaded.

| Non-paid pair | Tools | Demo, with proper provenance |
| --- | --- | --- |
| allowed / budget_authorized | accept | accept |
| allowed / policy_allowed | accept | accept |
| allowed / observe_unrecorded | accept with its window and count | accept with its window and count |
| allowed / sandbox_no_spend | accept | accept |
| allowed / observe_projected | accept | accept |
| denied / budget_exhausted | accept | accept |
| denied / insufficient_budget | accept | accept |
| denied / policy_denied | accept | accept |
| denied / capability_invalid | accept | accept |
| denied / capability_expired | accept | accept |
| denied / auth_missing | accept | accept |
| denied / token_revoked | accept | accept |
| denied / payment_required | accept | accept |

`denied / policy_denied` has a second accepted demo shape: a verified capability
refused by policy before any budget lookup (for example a scope miss). Authority
is `verified_macaroon_caveats` with the capability hash bound and raw token,
budget, and parent fields redacted. Budget is exactly
`{"spend_mode": "not_evaluated", "cost_credits": 0}`, and `budget_state` mirrors
it exactly. Any limit, remaining, attempted, used, spent, or projected amount
or balance key anywhere in the receipt (outside the strict budget object) or
anywhere on the Pack wrapper fails, at any depth and whether matched by exact
name or by marker. A nonzero cost, unredacted raw fields, and
`no_verified_capability` authority also fail. The `not_evaluated` budget stays illegal on
every other evaluated pair. Tools already accepts `policy_denied` by enum and
rejects the remaining and attempted balance fields on this shape.

Tools still validates non-paid enum membership rather than adopting demo's
paired-profile matrix (for example, the historical tools acceptance and demo
rejection of paid / budget_authorized are covered). Unknown enums and unsigned
tampering remain rejected. These differences are legacy behavior, not a claim
that the two surfaces are fully equivalent outside paid rail.

## Receipts when sign-every-action is on

A gateway with the sign-every-action switch on signs a receipt for each call a verified key makes. Three receipt shapes carry it. Both public copies (`tools/` and `demo/`) hold them to the same cases (`tools/test_every_action_receipts.py`); the copies are patched separately, not copied over each other. A check runs only for a receipt that has the shape, so every older receipt is read as it was, and unknown reasons and unknown pairs still fail.

**Allowed, no balance claim.** `allowed` / `policy_allowed` with `budget.spend_mode` `not_evaluated` (a free tool, a key with no budget). Budget is exactly `{"spend_mode": "not_evaluated", "cost_credits": 0}`. No limit, remaining, attempted, used, spent, projected or payment key at any depth outside `budget`, and none on the Pack wrapper. Demo reads it as the same verified not-evaluated profile as a scope refusal (verified capability authority, redacted raw fields); tools applies the budget and no-claim rules and, as before, does not check provenance. An `allowed` / `policy_allowed` receipt with an evaluated budget is unchanged. This replaces the earlier rule that the not-evaluated budget was legal only on `denied` / `policy_denied`.

**Refusal windows.** A receipt with any `rate_limit_*` field is a window receipt and must be a denial (`denied` with `policy_denied`, `token_revoked` or `capability_expired`) with a non-empty `denial_code`, `rate_limit_phase` `opened` or `closed`, `rate_limit_window_start` a UTC minute written `YYYY-MM-DDTHH:MM:00Z`, `rate_limit_window_seconds` the integer `60`, no `unrecorded_*` field and no balance claim. An opener has no `rate_limit_refused_calls` and is issued at or after its window start. A closer has `rate_limit_refused_calls` as a JSON integer of at least 1 (boolean, string, float, zero and negative are rejected) and is issued at or after the end of its window.

**Observe gap.** `allowed` / `observe_unrecorded`: the signed count of Observe calls that went through in one UTC minute with no receipt. `unrecorded_window_start` (UTC minute), `unrecorded_window_seconds` (the integer `60`), `unrecorded_calls` (integer, at least 1), no `rate_limit_*` field, no balance claim, and issued at or after the end of its window. Without a window or a count it is refused. Demo also refuses the reason on any budget but `not_evaluated`.

## Receipt record

`tools/verify_receipt_record.py` checks a downloaded record: NDJSON, one `satgate.evidence_pack.v1` pack per line in `issued_at` order, then one last line, the manifest. Tests: `tools/test_receipt_record.py`.

The manifest (`schema_version` `satgate.receipt_record_manifest.v1`) is signed like a receipt: JCS (RFC 8785) canonical bytes of the manifest without `manifest_hash` and `signature`, SHA-256 into `manifest_hash` (`sha256:` plus base64url), Ed25519 into `signature` (`ed25519:` plus base64url), with the key `issuer_kid` names in the issuer's JWKS. Without `--require-trusted-issuer` the embedded key in `metadata.public_key_ed25519_b64` is used and the output says the issuer was not anchored. Fields:

| field | meaning |
| --- | --- |
| `issuer`, `issuer_kid` | who signed; must equal each receipt's issuer |
| `key_hash` | the one agent key the record is about; every receipt's authority carries it (`token_id_hash`, `budget_id_hash` or `capability_hash`) |
| `from`, `to`, `generated_at` | the range (`from` inclusive, `to` exclusive) and when the record was made |
| `record_starts_at` | optional; when sign-every-action was switched on, so calls before it are known not to be covered |
| `count` | the number of packs |
| `first_receipt_hash`, `last_receipt_hash` | the first and last pack's `receipt_hash` (absent for an empty record) |
| `receipt_hashes_sha256` | `sha256:` plus base64url of SHA-256 over every `receipt_hash`, in order, each followed by a newline |
| `counts_by_decision_reason` | receipts per `decision_reason` |
| `windows` | one entry per refusal window: `route_or_tool`, `window_start`, `denial_code`, `opened_receipt_hash`, `closed_receipt_hash` (null if not closed), `refused_calls` (null if not closed) |

Problems (exit 1): a pack that does not verify, a missing or misplaced manifest, a bad manifest hash or signature, a count that differs, receipt hashes that do not match (a receipt dropped, added, changed or moved), a receipt outside the range, out of order or from another issuer, a receipt id, pack id or receipt hash twice, a receipt that does not belong to `key_hash`, a window with two openers or two closers, a closer with no opener, windows the manifest does not list, counts by reason that differ. Warnings (exit 0): an opener never closed once its window has been over for 5 minutes; a window that began before the record's range; a request id on more than one receipt (an agent can legitimately reuse a JSON-RPC id, so each is a separate decision); calls that went through in Observe with no receipt (counted from the gap receipts); a range that starts before `record_starts_at`.

Exit codes: 0 the record checks out, 1 a problem, 2 the input could not be loaded, 3 the `cryptography` package is missing. The JSON output ends with a one-line `summary`, `N receipts, M refusals, K windows, P problems`, also printed on stderr.

What it cannot prove. The manifest is made at download time from what SatGate holds. It shows that nothing was dropped from that list, not that no call ever failed to be stored; the gateway's own counters of calls seen against receipts made cover that.

## Provenance and verification

The demo verifier is now an `adapted` file in `demo/SOURCE-MANIFEST.json`; the
original source commit/tree/blob remain unchanged as provenance, and its SHA-256
identifies the actual distributed bytes. Static tests verify every manifest file
digest, not just digest syntax. The narrow duplicated paid-rail validator keeps
the demo standalone; the shared signed behavior corpus is the parity guard.

CI runs both Python commands as mandatory steps in the Go job without path
filters or failure suppression:

```sh
python -B -m unittest discover -s tools -p 'test_*.py' -v
python -B -m unittest discover -s demo/tests -p 'test_*.py' -v
go test -race -count=1 -v ./pkg/proxy -run '^TestAgentLoop'
```

Local Python requires cryptography and RFC8785; the latter may be provided via
`PYTHONPATH="$PWD/demo/_vendor"`. These self-tests are implementation evidence,
not an independent security verdict, live payment test, or deployment approval.

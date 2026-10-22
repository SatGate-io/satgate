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
require their presence, except for the Observe gap profile (below). Mirror mismatches emit `pack_primary_mismatch` with the
field name; budget-state mismatches emit `budget_state_mismatch`.

### The Pack wrapper's mirrors for the new profiles

A Pack whose primary receipt is one of the sign-every-action shapes (an
`allowed` / `policy_allowed` receipt with the no-balance budget, a window opener
or closer, or an Observe gap record) must repeat the receipt's `authority`,
`budget` and `policy` in the Pack, each exactly equal (recursive, type-strict)
to the signed receipt's. A Pack that leaves one out is refused with
`pack_primary_mismatch` naming the field, by both copies. `tools` used to treat
these three as optional when absent and `demo` as required; for these shapes
only, `tools` now requires them, so the two copies agree on every Pack of a new
shape (a signed Pack with each of the three removed, for each of the shapes, is a
parity test, `tools/test_every_action_receipts.py`). The rule for any other Pack
is unchanged: `tools` checks the three when they are present, `demo` always
compares them. The legacy evaluated and paid profiles are not touched.

An Observe gap record has two more mirrors that must be present. A Pack whose
primary receipt is `allowed` / `observe_unrecorded` (or carries any of the
`unrecorded_*` fields) must repeat the receipt's `issued_at` and `evidence_url`
in the Pack, each exactly equal (type-strict) to the signed receipt's. A gap Pack
that leaves either out is refused with `pack_primary_mismatch` naming the
field, by both copies, as `main` refuses it in both. `demo` has always required
these two for every Pack; `tools` now requires them for the gap profile only.
For the other new profiles (`allowed` / `policy_allowed`, window openers and
closers) and for every legacy Pack, `tools` keeps `issued_at` and
`evidence_url` optional (checked when present), exactly as `main` has it.

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

**Allowed, no balance claim.** `allowed` / `policy_allowed` with `budget.spend_mode` `not_evaluated` (a free tool, a key with no budget). Budget is exactly `{"spend_mode": "not_evaluated", "cost_credits": 0}`. No limit, remaining, attempted, used, spent, projected, spend or payment key at any depth outside `budget`, and none on the Pack wrapper (the wrapper's `budget_state` must be the exact budget). The same rule, with the same key list, applies to the other two new shapes below. It covers keys inside `metadata`, `policy`, `authority` and any extension, matched by exact name or by marker whatever the case or punctuation (`payment_hash`, `Payment-Hash`, `rail`, `amount_sats`, `remaining_credits`, `balance`, `spend_mode`, `cost_credits`). Both copies read provenance the same way: the authority names a verified key (`verified_macaroon_caveats`, capability hash bound to the receipt's, raw token, budget and parent fields redacted), and a missing authority, the `no_verified_capability` form or an unredacted field fails. An `allowed` / `policy_allowed` receipt with an evaluated budget is unchanged. This replaces the earlier rule that the not-evaluated budget was legal only on `denied` / `policy_denied`. The old evaluated and paid profiles are not judged by any of this.

**Refusal windows.** A receipt with any `rate_limit_*` field is a window receipt and must be a denial (`denied` with `policy_denied`, `token_revoked` or `capability_expired`) with a non-empty `denial_code`, `rate_limit_phase` `opened` or `closed`, `rate_limit_window_start` a UTC minute written `YYYY-MM-DDTHH:MM:00Z`, `rate_limit_window_seconds` the integer `60`, no `unrecorded_*` field and no balance claim. A window receipt for a `token_revoked` or `capability_expired` refusal names no capability (authority exactly `{"provenance_level": "no_verified_capability"}`, no `capability_hash`); one for `policy_denied` names the verified key, as above. An opener has no `rate_limit_refused_calls` and is issued at or after its window start. A closer has `rate_limit_refused_calls` as a JSON integer of at least 1 (boolean, string, float, zero and negative are rejected) and is issued at or after the end of its window.

**Observe gap.** `allowed` / `observe_unrecorded`: the signed count of Observe calls that went through in one UTC minute with no receipt. `unrecorded_window_start` (UTC minute), `unrecorded_window_seconds` (the integer `60`), `unrecorded_calls` (integer, at least 1), no `rate_limit_*` field, no balance claim, verified-key authority, and issued at or after the end of its window. Without a window or a count it is refused. Demo also refuses the reason on any budget but `not_evaluated`.

## Receipt record

`tools/verify_receipt_record.py` checks a downloaded record: NDJSON, one `satgate.evidence_pack.v1` pack per line in `issued_at` order, then exactly one last signed line: a complete manifest or a truncation terminal. Missing, repeated or misplaced terminals fail closed.

**The rule: exactly one receipt per record line.** A pack in a record holds one receipt (`receipts` has length 1), and the manifest's hashes cover that receipt. The verifier refuses a line with none, two or more, whether a receipt was added, dropped, duplicated or moved inside a pack. The alternative, hashing and counting every embedded receipt, was not chosen: one receipt per line is what the gateway writes, it keeps `count` equal to the number of receipts, and a pack's signature does not commit to its `receipts` array. Tests: `tools/test_receipt_record.py`.

The manifest (`schema_version` `satgate.receipt_record_manifest.v1`) is signed like a receipt: JCS (RFC 8785) canonical bytes of the manifest without `manifest_hash` and `signature`, SHA-256 into `manifest_hash` (`sha256:` plus unpadded base64url), Ed25519 into `signature` (`ed25519:` plus unpadded base64url), with its own `issuer_kid` key. The truncation terminal uses exactly the same signing and hash procedure, including the field name `manifest_hash`.

**Issuer keys and rotation.** Supply the issuer's trusted set with `--jwks-file issuer-jwks.json` or `--jwks-url https://issuer.example/.well-known/jwks.json`, and use `--require-trusted-issuer` to require anchoring for every receipt and the terminal. Each signature selects its own `issuer_kid`; different key ids and different public keys are allowed, including a terminal signed by a newer key. Every receipt's signed `issuer` must still equal the terminal's. A missing key id fails (per-pack `receipt_invalid` with `jwks_kid_not_found` / `trusted_issuer_required`, or terminal `trusted_issuer_required`); embedding that missing key cannot bypass trusted mode. The caller must obtain the trusted set from the intended issuer; a local JWKS is an explicit trust input, not proof of who supplied it. The record checker does not automatically discover/fetch keys from record contents.

Without a trusted set and `--require-trusted-issuer`, embedded keys in each object's `metadata.public_key_ed25519_b64` can check a mixed-key record for self-consistency only. They do not authenticate the claimed issuer, even though issuer strings must agree. Successful output includes `trust_mode: embedded_key_self_consistency`, `trusted_issuer_valid: false` and warning `issuer_not_authenticated`. If all signatures actually anchor to the supplied JWKS, output instead says `trust_mode: trusted_issuer` and `trusted_issuer_valid: true`. Failed records say `trust_mode: unverified`. Fields:

| field | meaning |
| --- | --- |
| `issuer`, `issuer_kid` | who signed the terminal; every receipt has the same issuer but selects its own signing key by its own `issuer_kid` |
| `key_hash` | the one agent key the record is about; every receipt's authority carries it (`token_id_hash`, `budget_id_hash` or `capability_hash`) |
| `from`, `to`, `generated_at` | the range (`from` inclusive, `to` exclusive) and when the record was made |
| `record_starts_at` | optional; when sign-every-action was switched on, so calls before it are known not to be covered |
| `count` | the number of packs |
| `first_receipt_hash`, `last_receipt_hash` | the first and last pack's `receipt_hash`; absent or null for an empty record, and nothing else is accepted there (not `false`, not an empty string) |
| `receipt_hashes_sha256` | `sha256:` plus base64url of SHA-256 over every `receipt_hash`, in order, each followed by a newline |
| `counts_by_decision_reason` | receipts per `decision_reason`; every value is a whole number (a JSON `true` or `false` is refused, as are floats) |
| `windows` | one entry per refusal window: `route_or_tool`, `window_start`, `denial_code`, `opened_receipt_hash`, `closed_receipt_hash` (null if not closed), `refused_calls` (null if not closed) |
| `filter` | optional signed producer selection information (E3 uses `token_id_hash` and `surfaces`); does not replace the receipt-level `key_hash` check |
| `coverage`, `legacy_identity_policy` | optional signed producer statements, not independently proven by the checker (E3 states stored receipts only and its legacy identity limitation) |
| `truncated` | optional on a complete manifest; if present, must be the JSON boolean `false`, not `0`, `null` or a string |

**Closed v1 field sets.** The complete manifest permits exactly the table's fields plus `schema_version`, `canonicalization`, `hash_algorithm`, `signature_algorithm`, `metadata`, `manifest_hash` and `signature`. Algorithm values must be `jcs-rfc8785`, `sha256`, `ed25519`. Unknown top-level fields fail with `manifest_unknown_field`, even when properly signed. New top-level semantics need a new supported schema version, not an ignored flag. `metadata` and `filter` are signed producer maps, not extensions to the checker's completeness rules. `truncation_reason` is not a complete-manifest field. An ordinary manifest with `truncated: true` fails with `manifest_truncated_invalid` instead of passing as complete.

**Authenticated cut-off prefix.** A controllable producer failure may end a record with `schema_version: satgate.receipt_record_truncated.v1`. This terminal permits the same fields plus `truncation_reason`, and requires `truncated` exactly `true` and a nonempty string reason (`truncation_fields_invalid` otherwise). It uses the normal `count`, endpoint hashes, newline hash-of-hashes, `counts_by_decision_reason` and `windows` to describe only the prefix before it; there are no renamed `prefix_*` wire fields. E3's existing output is supported without a wire-format change. Reasons such as `time_limit`, `manifest_limit`, `archive_or_index_unavailable` and `signing_identity_changed` are authenticated producer statements, not diagnoses independently established by this checker; other nonempty reasons are preserved.

The checker verifies the terminal signature, all prefix packs, counts, endpoint hashes, ordered hash digest, decision counts, range, order, duplicates, issuer, agent identity and window entries. Any mismatch is invalid (exit 1), not a verified truncation. Only after all checks pass does it return `status: truncated`, `valid: false`, `complete: false`, `truncation_reason`, and `verified_prefix` containing `count`, `first_receipt_hash`, `last_receipt_hash`, `receipt_hashes_sha256`, `counts_by_decision_reason` and `windows`. Its summary begins `truncated (reason): verified prefix, ...`, and its exit code is **4**, never 0. An empty verified prefix is allowed. A lost connection with no signed terminal still fails verification; it is not authenticated truncation.

A truncation terminal does not prove that a range is complete: an opener without a closer is `prefix_open` in `window_states`, not an overdue-closer error, because the closer may lie in the undownloaded suffix. Other prefix/window checks still apply. Complete records retain the overdue/pending/boundary rules below. Invalid terminals have no `verified_prefix` or authenticated `truncation_reason` in output.

Problems (exit 1): a pack that does not verify, a pack that does not hold exactly one receipt, a missing or misplaced terminal, a bad terminal hash or signature, unknown terminal fields or invalid truncation markers, a count that differs, receipt hashes that do not match (a receipt dropped, added, changed or moved), a receipt outside the range, out of order or from another issuer, a signing key absent from the required trusted set, a receipt id, pack id or receipt hash twice, a receipt that does not belong to `key_hash`, a window with two openers or two closers, a closer with no opener, **a complete record's window whose closer is overdue**, windows the terminal does not list, counts by reason that differ or are not whole numbers, first or last hashes on an empty record.

A window's closer is overdue when the window's end plus the 15 second grace the gateway waits is at or before the earlier of the range's end (`to`) and the time the record was made (`generated_at`), the window was opened inside the record, and no closer is in it: a complete record has it. That is a problem, not a warning. A window whose closer is not due yet at that point is reported as `pending` (warning `window_closer_pending`), and one whose window runs past the end of the range as `boundary` (warning `window_runs_past_range`): neither fails the record. The output lists every window with its state (`closed`, `pending`, `boundary` or `overdue`) under `window_states`. Warnings (exit 0): a pending or boundary window; a window that began before the record's range; a request id on more than one receipt (an agent can legitimately reuse a JSON-RPC id, so each is a separate decision); calls that went through in Observe with no receipt (counted from the gap receipts); a range that starts before `record_starts_at`.

Exit codes: **0** a verified complete record (`status: complete`, `valid: true`, `complete: true`); **1** an invalid record, including malformed NDJSON (`status: invalid`); **2** file/JWKS loading or CLI usage errors; **3** the `cryptography` package is missing; **4** an authenticated truncated prefix, never a complete record. Warnings alone do not change a complete record's exit 0. JSON includes a one-line `summary`, `N receipts, M refusals, K windows, P problems` (prefixed with truncation information for exit 4), also printed on stderr. `problems: []` alone is not a completeness test: check the status/exit code.

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

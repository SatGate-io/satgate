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

Demo has richer paired decision/reason profiles and provenance validation. The
following pre-existing differences are intentionally not erased by copying one
implementation over the other:

| Non-paid pair | Tools | Demo, with proper provenance |
| --- | --- | --- |
| allowed / budget_authorized | accept | accept |
| allowed / policy_allowed | accept | accept |
| allowed / sandbox_no_spend | reject | accept |
| allowed / observe_projected | reject | accept |
| denied / budget_exhausted | accept | accept |
| denied / policy_denied | accept | accept |
| denied / capability_invalid | accept | accept |
| denied / capability_expired | accept | accept |
| denied / auth_missing | reject | accept |
| denied / token_revoked | reject | accept |
| denied / payment_required | accept | accept |

Tools still validates non-paid enum membership rather than adopting demo's
paired-profile matrix (for example, the historical tools acceptance and demo
rejection of paid / budget_authorized are covered). Unknown enums and unsigned
tampering remain rejected. These differences are legacy behavior, not a claim
that the two surfaces are fully equivalent outside paid rail.

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

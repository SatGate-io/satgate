# V2 decision reader contract (development profile)

Reader-only, binding-only development support. No producer, financial enrollment,
real broker adapter, authority validation, execution observation, capsule, bundle,
replay permission or completeness claim is added here. Existing v1 verification
and its deliberate tools/demo differences are unchanged.

## Entry points and trust

Both maintained `verify_evidence_pack.py` scripts accept a raw decision object,
not a v2 object inserted into a v1 Evidence Pack or receipt-record NDJSON.
`verify_pack(..., opening=...)` checks an optional private opening. CLI input uses
`--opening-file` with local files/stdin and `--jwks-file`; it refuses URL input,
JWKS URLs and discovery when an opening file is provided. V2 never discovers
keys or falls back to an embedded key, even when a v1 embedded-key option is set.
A supplied local JWKS must be independently obtained for the intended issuer.

The offline SDK `SatGateAgentClient.verify_pack` accepts the optional opening and
passes it only to the local verifier subprocess. `satgate-cli agent verify` can
forward `--opening-file`. The separate remote beta SDK verifier is unchanged and
is not an opening-verification API. Historical pinned/released scripts and the
immutable unavailable-check-v1 bundle remain v1 artifacts. A v2 downloadable
bundle is separate future work; these artifacts are not silently upgraded.

## Closed decision object

Every field below is required, and no additional field is allowed:

- `schema_version`: `satgate.receipt.v2`
- `record_type`: `decision`; `phase`: `pre_dispatch`
- `binding_profile`: `satgate.binding_only.v1`
- `canonicalization`: `jcs-rfc8785`; `hash_algorithm`: `sha256`;
  `signature_algorithm`: `ed25519`
- `receipt_id`, `issuer_kid`, `admission_revision`: nonempty opaque strings
- `issuer`: HTTPS origin only; `issued_at`: UTC RFC3339 with uppercase T/Z,
  optional fractional seconds; future skew bounded to 300 seconds
- `enforcement_mode`: `control`
- `authority_evidence`: `not_evidenced`
- `authorization_record_hash`, `effective_limits_record_hash`,
  `approval_record_hash`: explicit null in this development profile
- `approval_disposition`: `not_required` or `required_pending`
- `admission`: `allowed` or `denied`; pending approval requires denied
- `dispatch`: `not_sent` (this is a pre-dispatch decision, not execution evidence)
- `call_commitment`: exactly `{profile, algorithm, digest}` with
  `satgate.exact_call.v1`, `sha256`, and `sha256:` + unpadded canonical base64url
  of 32 bytes
- `receipt_hash`: the same digest encoding
- `signature`: `ed25519:` + unpadded canonical base64url of 64 bytes

Receipt payload is RFC 8785 JCS of the entire object excluding only its own
TOP-LEVEL `receipt_hash` and `signature`. Hash SHA-256 of those bytes; verify
Ed25519 over those bytes, not the digest. Nested fields remain authenticated.
Unknown versions, record types, profiles, fields and algorithms are rejected;
missing required fields never select legacy verification. JWKS key selection is
unique by `issuer_kid`, OKP/Ed25519, EdDSA algorithm/signing use if present, with a
canonical unpadded 32-byte public key. No embedded-key fallback is permitted.

Only a binding-only profile is defined. Enrollment profiles and authenticated
reference records must be specified and implemented separately; null references
cannot be used to claim that authority/limits/approval were verified.

## Opening and commitment envelope

The opening is a closed object with all these required fields:

- `schema_version`: `satgate.exact_call.opening.v1`
- `receipt_id`, `attempt_id`, `operation_id`, `authority_binding`,
  `account_binding`, `route_binding`, `adapter_version`: nonempty private strings
- `opening`: canonical unpadded base64url of exactly 32 private random bytes
- `transport`, `normalized_intent`, `policy_projection`: the objects below

Rebuild the envelope by excluding only the opening's `schema_version` and adding
`domain: satgate.exact_call.v1`. SHA-256 of the envelope's JCS bytes must equal the
signed `call_commitment.digest`; its `receipt_id` must equal the decision's id.
No other member is omitted. The salt/attempt are independent of client request
ids. The deterministic generator uses fake entropy/keys for repeatable tests,
not production entropy generation. Fresh server entropy and operation/attempt
lifecycle belong to the producer lanes, not this reader.

`transport` is closed and contains all of:

- `profile`: `satgate.binding_only.transport.v1`
- `service`, `tool`, `method`, `path`: nonempty private strings
- `query_b64`, `headers_b64`, `body_b64`, `params_b64`: exact bytes, encoded as
  canonical unpadded base64url; order/whitespace are not normalized

This deliberately limited development transport supports UTF-8 query/headers
and UTF-8 JSON bodies/params, not arbitrary broker protocols. It excludes
credentials: authorization/cookie/token/secret/password/API-key names or markers
are rejected in decoded headers/query and recursively in decoded JSON params
and body. Full params, including meaningful metadata, are committed, not merely
arguments. Duplicate names, case-fold collisions, invalid Unicode, nonfinite or
out-of-JCS-range numbers in those JSON bytes are rejected. No real adapter
exclusion list, broker boundary or helper handshake is inferred from this
profile. Comparing a signed exact byte commitment does not independently prove
that the producer captured the final request or projected policy correctly.

`normalized_intent` is closed, with all of:

- `operation`: `create`, `amend` or `cancel`
- `quantity`, `price`: nonnegative decimal strings, no signs/exponents/leading
  zero ambiguity; `quantity_unit`, `price_unit`, `time_in_force`: nonempty strings
- `cancel_target`, `replacement_link`: explicit null or nonempty private strings

`policy_projection` is exactly `{credit_charge, default_time_in_force}`.
`default_time_in_force` is explicit and nonempty. `credit_charge` is exactly
`{evaluation_state, amount, unit}`: evaluated has a decimal-string amount and
nonempty unit; not_evaluated requires explicit null amount/unit. Unknown economic
fields in these closed projections are rejected. Real adapter economic-field,
default and policy-derivation validation before admission remains producer work;
this generic development profile cannot authorize a real financial dispatch.

## Bounded verdicts and privacy

`valid` describes the requested signature/schema/hash and, only when supplied,
opening checks. `call_binding` is `not_checked`, `matched` or `mismatched`.
Without an opening it is never matched. `authority_state` and `execution_state`
are always `not_evidenced`. Pending approval reports `admission_state: held` and
`v2_hold_not_executed`, never executed or a broker rejection. A valid allow is
not execution evidence. An opening mismatch can retain signature trust while
`valid` is false. No private opening values or input-derived errors are echoed.

New runtime output is machine codes only (`reasons` equals `reason_codes`):

- `v2_schema_invalid`
- `v2_hash_mismatch`
- `v2_key_required`
- `v2_signature_invalid`
- `v2_opening_invalid`
- `v2_call_mismatch`
- `v2_call_matched`
- `v2_call_not_checked`
- `v2_hold_not_executed`
- `v2_authority_not_evidenced`
- `v2_unsupported_container`
- `v2_input_invalid`

The reference test generator is `tools/tests/generate_decision_v2.py`.
`tools/fixtures/reader-v2-decisions.json` pins its dummy corpus. The extended
compatibility matrix preserves whole v1 verdict bytes and checks v2 parity in
the maintained copies. New tests exercise both CLIs, the actual offline SDK/Go
wrappers, and explicit historical/linked/record unsupported-format boundaries.

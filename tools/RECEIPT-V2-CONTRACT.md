# V2 receipt reader contract (development profile)

Reader-only, binding-only development support. No producer, financial enrollment,
real broker adapter, authority validation, capsule decryption,
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
forward `--opening-file`. `SATGATE_VERIFIER` selects the pack script from
`tools/checker-bundles/receipt-bundle-v2/` for either wrapper; the bundle's
`SHA256SUMS` digest is pinned in `IMMUTABLE_BUNDLE_MANIFESTS` in
`tools/test_reader_compatibility.py`. The Python and Node beta `SatGate.verify`
methods refuse private bundle/opening markers locally with
`private_evidence_not_posted`, before `POST /v1/verify`. Historical pinned/released
scripts and the immutable `unavailable-check-v1` bundle remain v1 artifacts.

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
Without an opening it is never matched. For decision records, `authority_state`
and `execution_state` are always `not_evidenced`. Pending approval reports `admission_state: held` and
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

## Closed execution observation (reader-only development choice)

Both maintained readers also accept a separate raw `satgate.receipt.v2` object
with `record_type: execution_observation`. The API adds `parent` (exactly one raw
decision) and `result_opening`; `opening` remains the parent's exact-call opening.
The CLI adds local `--parent-file` and `--result-opening-file`. All three input
options prohibit source URLs, JWKS URLs and discovery. The offline SDK and Go
wrapper forward these inputs only to the local subprocess, not the beta API.
No v1 pack or NDJSON format is repurposed. Historical immutable artifacts stay
v1-only; the matrix pins rejection alongside the separate v2 distribution.

Every observation field below is required; extra fields are rejected:

- `schema_version`, `record_type` as above; `binding_profile: satgate.binding_only.v1`
- `canonicalization: jcs-rfc8785`, `hash_algorithm: sha256`,
  `signature_algorithm: ed25519`
- `record_id`, `issuer`, `issuer_kid`, `observed_at`: same bounded string,
  HTTPS-origin and UTC/future-skew rules as the decision identity
- `decision_receipt_hash`: the parent's payload SHA-256 digest, not a pack hash
- `call_commitment`: the same closed profile/algorithm/digest object as the parent
- `operation_id`, `attempt_id`: opaque references; these must equal the refs in
  the authenticated parent opening, not client request ids
- `status`: exactly `accepted`, `rejected`, `transport_failed_before_send`,
  `sent_outcome_unknown`, `partially_filled`, `filled`, `cancelled`, `later_reconciled`
- `result_commitment`: exactly `{profile, algorithm, digest}`, with
  `satgate.execution_result.v1`, `sha256`, and the usual digest encoding
- `capsule_digest`, `record_hash`: the usual SHA-256 digest encoding
- `signature`: the usual Ed25519 signature encoding

Hash/sign RFC 8785 bytes excluding ONLY this object's top-level `record_hash`
and `signature`. In particular, `decision_receipt_hash`, both nested commitment
digests and `capsule_digest` remain signed. The same independently obtained local
JWKS selection rules apply; embedded keys never rescue verification. Capsule
syntax and signed digest are checked here; ciphertext/disclosure selection and
manifest completeness belong to the separate bundle lane, not this verifier.

The parent is independently schema/hash/signature verified. Require the same
issuer (key rotation is allowed), exact payload digest and call commitment,
allowed admission with no pending approval, and observation time no earlier than
decision issuance. A denied/held decision cannot become broker execution evidence.
No candidate-parent search or accepting a list of parents is allowed.

Important row-03 boundary: decisions do not publish operation/attempt refs outside
their private commitment. This lane does not change those signed decision bytes.
Without the call opening, authentic public observation and parent signatures and
the parent digest can be checked, but the attempt join is not established:
`valid: false`, `v2_attempt_not_checked`, `attempt_binding: not_checked`. This is
explicit partial verification, not an unsupported observation or a false join.
With the parent's valid opening, wrong attempt/operation/commitment fails. A result
opening is optional: without it `result_binding: not_checked` and no private
broker-id, response or amount match is claimed, even if the joined record is valid.

## Private result opening and bounded observation verdict

The result opening is closed, with all of these required members:

- `schema_version: satgate.execution_result.opening.v1`
- `decision_receipt_hash`, `operation_id`, `attempt_id`, `status`: match the record
- `opening`: canonical unpadded base64url of 32 bytes, distinct from the call opening
- `account_binding`: matches the authenticated call opening
- `broker_order_id`, `client_order_id`, `venue_status`: private nonempty strings
  or explicit null; no public mirror of these fields
- `target_order_id`, `replacement_link`: match the call intent's `cancel_target`
  and `replacement_link`, including explicit nulls
- `venue_times`: exactly `{received_at, updated_at}`, UTC timestamps or null
- `parser_version`: nonempty private string
- `financial_values`: exactly `{quantity, quantity_unit, price, price_unit}`;
  each amount/unit pair is null/null or nonnegative unambiguous decimal string
  plus nonempty unit; unknown economic fields fail
- `response`: null or exactly `{profile, headers_b64, body_b64}`, with
  `profile: satgate.binding_only.response.v1`. Header and body encodings preserve
  exact credential-free UTF-8 bytes. This development profile accepts JSON body
  bytes only; reject duplicate/case-colliding names, invalid Unicode, unsafe JCS
  numbers and credentials. Credential exclusion rules match the call development
  transport. Reordering equivalent response JSON still changes its commitment.

Rebuild the envelope by excluding only opening `schema_version` and adding
`domain: satgate.execution_result.v1`. SHA-256 its JCS bytes; compare the signed
result commitment. Nothing else is omitted. This commits the response bytes AND
the private parser projection, rather than interpreting a broker protocol here.
`accepted`, `partially_filled`, `filled` and `cancelled` openings require response,
broker id and venue status. `transport_failed_before_send` requires no response,
broker id or venue status. Missing reliable broker id/status cannot be upgraded
to one of those reliable statuses in an opened record. `sent_outcome_unknown`
remains unknown, never rejection; no resend permission is produced.

These are development-profile field/format choices for offline readers, not a
settled real adapter/producer contract. The reader does not derive a broker
status, client id, financial projection or broker id from proprietary response
semantics, corroborate order history, or prove that capture/parser behavior was
honest. Reconciliation/fill statuses attest only the signed observation. Each
record links to its immutable decision; no aggregate transition/previous-result
chain, finality or stream completeness is inferred by this raw-record API.

`execution_state` is exactly `observed_` plus the authenticated status, never an
upgrade from acceptance to fill/settlement. `observation_status` preserves that
status. `broker_witness` and `settlement_state` stay `not_evidenced`. No private
values appear in verdicts/errors. All new runtime reasons are machine codes:

- `v2_parent_required`
- `v2_parent_invalid`
- `v2_parent_mismatch`
- `v2_attempt_not_checked`
- `v2_observation_verified`
- `v2_result_not_checked`
- `v2_result_matched`
- `v2_result_mismatch`
- `v2_result_opening_invalid`
- `v2_result_link_mismatch`
- `v2_outcome_unknown`

Shared v2 schema/hash/key/signature/input/authority/container codes retain their
existing meaning. `reasons` equals `reason_codes`; no new runtime sentence ships.
The dummy-only generator is `tools/tests/generate_observation_v2.py`, and its
checked-in corpus is `tools/fixtures/reader-v2-observations.json`. Both maintained
APIs/CLIs, real offline SDK/Go wrappers and the compatibility matrix consume every
vector; immutable pack/record/linked artifacts retain explicit rejection tests.

## Closed private bundle (reader-only development choice)

The maintained tools pack, demo pack and receipt-record readers accept a single
private JSON object from a local file or stdin. The Python CLIs accept
`--private-bundle` to force local-only parsing even for malformed or mislabelled
input; pin the keys with `--jwks-file`. The existing SDK and Go verification
wrappers also delegate local bundle files to the maintained pack reader. This
does not repurpose v1 NDJSON or change the remote beta API or v1 downloads.

The top level is exactly `schema_version: satgate.receipt_bundle.v2`,
`record_type: private_bundle`, `manifest`, `artifacts`, `capsules`, and `openings`.
The last three are maps from exact byte SHA-256 digests to canonical unpadded
base64url of those bytes. Signed records and already-decrypted openings are
UTF-8 JSON; capsule ciphertext is opaque. Base64url is not encryption. Do not
publish the bundle, the openings, or their plaintext digests. No AEAD format,
decryption, export authorization, owner/recipient binding or recovery protocol
is defined by this development reader.

The manifest has exactly these fields:

| Field | Definition |
| --- | --- |
| `schema_version`, `record_type` | `satgate.receipt_bundle_manifest.v2`, `bundle_manifest` |
| `binding_profile`, `canonicalization`, `hash_algorithm`, `signature_algorithm` | `satgate.binding_only.v1`, `jcs-rfc8785`, `sha256`, `ed25519` |
| `issuer`, `issuer_kid`, `generated_at` | same origin, key and UTC timestamp syntax/skew as v2 records |
| `range` | exactly `{from,to}`, UTC timestamps, inclusive start and exclusive end; ordered and no later than `generated_at` |
| `artifacts` | list of distinct exact-byte artifact digests; order is signed, not an execution or coverage claim |
| `disclosure` | exactly one entry for each artifact digest; each entry is exactly `{state,capsule_digest,opening_digest}` |
| `outcomes` | exactly one state for each decision **payload** digest |
| `retention_state` | `retained`, `expired`, or `deleted` for the export selection |
| `historical_keys` | reference-only list of public Ed25519 JWKs with exactly `kty`, `crv`, `alg`, `use`, `kid`, `x`; distinct `kid`s |
| `manifest_hash`, `signature` | SHA-256 and Ed25519 of the manifest JCS bytes excluding only these two top-level fields |

All digest and signature values use the same `sha256:` / `ed25519:` encodings
as the individual records. Historical keys do not supply trust or override
pins. The manifest, all decisions and all observations must independently
verify under the caller's pins for the same issuer; rotation by `issuer_kid`
is allowed. No receipt-directed network or historical-key fallback is used.

The byte maps must have exactly the three signed digest sets. A disclosure
state is `disclosed` (non-null `capsule_digest` and `opening_digest`) or
`withheld`, `missing`, `expired`, `deleted` (null opening digest; the ciphertext
may still be listed and supplied). Each non-null digest in a map is unique.
The reader checks each disclosed opening against its signed record commitment.
For an observation, its own signed `capsule_digest` also matches the exact
capsule ciphertext bytes when present. The reader checks both commitments but
does not prove that a supplied opening was decrypted from that ciphertext.
All artifacts must be in range; an observation
must join exactly one included decision by payload digest via the row-04 link
checks. Without a parent call opening, the private attempt/account join remains
unchecked, even if the result opening independently matches the observation.

An allowed decision has `available` when at least one observation is present
and none has `sent_outcome_unknown`; any signed unknown observation requires
`unknown`. Without observations only `pending` or `unknown` is valid. A denied
decision has `not_expected` and cannot have an observation. A `later_reconciled`
record does not override a separate signed unknown in this profile because
row 04 does not bind replacements or reconciliation between observations.

`status: verified` has `valid: true`, `complete: true`, and CLI exit 0 for the
listed selection only. Explicit missing openings/outcomes or a retention cut
give authenticated `status: partial`, `valid: false`, `complete: false` and
exit 1. Structural, signature, byte-hash and link failures are `invalid` and
exit 1. No verdict includes the underlying orders, amounts, outcomes or other
private values. No completeness for an account, independent clock or broker
witness, or fill/settlement claim follows from the manifest. Verdict text is
limited to machine codes.

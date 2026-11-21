# Receipt reader compatibility inventory

Scope: public main baseline `b7e6790d3eebb43e1b5804d7e595f2db9c035985`,
public module pin `f852c2918fb9a0640bf40f8f46919f7379d28c35`, downloadable
`unavailable-check-v1` and `receipt-bundle-v2`, and archived October readers.
No producer is changed. Baseline line numbers refer to the v1 inventory; the
new v2 distribution is listed separately. All tests use dummy signed data and
local JWKS files.

## Reader inventory

| Reader / entry | Versions and record types | Decisions / reasons | Issuer acceptance |
| --- | --- | --- | --- |
| `tools/verify_evidence_pack.py:54-78,718,858` | `satgate.evidence_pack.v1` containing `satgate.receipt.v1`; decision, refusal-window opener/closer, Observe gap | allowed, denied, paid; non-paid reasons enumerated below, payment_verified via rail validator | embedded keys by default; explicit local JWKS + require-trusted pins each kid/key; optional explicit discovery |
| `demo/verify_evidence_pack.py:56-83,99-104` | same pack/receipt versions and shapes | paired decision/reason profiles, richer provenance; same gateway reasons | API defaults embedded; CLI defaults trusted/discovery; explicit local JWKS keeps matrix offline |
| `tools/verify_receipt_record.py:16-29,73-78,85-94,355-359` | NDJSON v1 packs, exactly one receipt each; last signed `satgate.receipt_record_manifest.v1` or `satgate.receipt_record_truncated.v1`; closed terminal fields | delegates every pack to sibling tools reader; validates counts, ranges, windows, key binding | every receipt and terminal must share issuer; key rotation permitted; local JWKS checks every kid; no automatic discovery |
| `tools/checker-bundles/unavailable-check-v1/verify_evidence_pack.py:54-78,858` and `verify_receipt_record.py:73-78` | same as tools, frozen download | same as tools | same as tools |
| `tools/checker-bundles/unavailable-check-v1/SHA256SUMS:1-4` | manifest for both scripts, requirements, frozen contract | not a verifier itself | trusted release bytes must be obtained independently |
| `tools/checker-bundles/receipt-bundle-v2/{verify_evidence_pack,verify_receipt_record}.py` | byte-equal tools readers; v1 inputs plus v2 decisions, observations and private JSON bundles as specified by the v2 contract | same as the corresponding tools reader | independently pinned local JWKS for v2; v1 trust rules unchanged |
| `tools/checker-bundles/receipt-bundle-v2/SHA256SUMS` | manifest for five scripts, requirements and two contracts | not a verifier itself | SHA-256 pinned in the matrix: `e66c2b85c6aa2d351953f5f6262078788865ca171fada4dd273587181ed3b3b7` |
| public pin: `tools/fixtures/pinned-checkers-f852c2918fb9/{tools,demo}/verify_evidence_pack.py` and `tools/verify_receipt_record.py` | exact public git objects at the named module pin, not enterprise source | baseline executable bytes match main | same as corresponding tools/demo reader |
| October archive: `tools/fixtures/released-checkers-20261022/{tools,demo}/verify_evidence_pack.py`, `tools/verify_receipt_record.py`; `PROVENANCE.json` | older public module and dashboard pin recorded in provenance | rejects four unavailable-check reasons; historical tools/demo differences remain | same tools/demo trust distinction |
| `sdk/python/satgate/agent_client.py:765-785` | subprocess wrapper, requires SATGATE_VERIFIER; no independent format parser | delegated to chosen public pack script; invalid verdict raises SatGateError | supplied JWKS; trusted mode defaults true |
| `sdk/python/satgate/private_beta.py` and `sdk/nodejs/src/private-beta.ts` | remote POST `/v1/verify` for receipts, not an offline reader | private bundle/opening markers fail locally with `private_evidence_not_posted` | configured API, not offline issuer verification; mocked transport only |
| `cmd/satgate-cli/agent.go:110-127` | `agent verify` invokes python3; default tools script, SATGATE_VERIFIER override | delegated verdict; nonzero collapsed to exit 1 | trusted only when SATGATE_JWKS is set |
| `satgate-landing/app/verify-evidence-pack/page.tsx:1-108` | static instructions/download links, no browser or server verification implementation | delegates instructions to tools script at lines 55-63 | instructions require independently downloaded local JWKS |

Non-paid reasons: auth_missing, budget_authorized, budget_exhausted,
capability_expired, capability_invalid, insufficient_budget, observe_projected,
observe_unrecorded, payment_required, policy_allowed, policy_denied,
sandbox_no_spend, token_revoked, tool_price_unavailable,
tool_controls_unavailable, budget_backend_error, pending_capacity_exhausted.
Paid reason: payment_verified (allowed or paid, L402 or x402).
The complete accepted pair/profile list is in demo's table and the existing
`tools/VERIFIER-CONTRACT.md:120-153,155-196`; tools retains enum-only legacy
non-paid checking. Unknown pack/receipt versions and reasons fail.
There is no fixed issuer hostname allowlist when a caller supplies a local
trusted JWKS: that caller is responsible for obtaining the intended issuer's
keys. The matrix tests another dummy issuer, missing kid, embedded trust,
wrong keys, and explicit trusted mode without network access.

The separately linked `SatGate-io/evidence-pack-verifier` is also frozen and
exercised, at public commit `d62beb34ffa413f71f62ff56125ed8a3972b075d`, in
`tools/fixtures/linked-verifier-d62beb34/` (Apache-2.0 license retained).
Its `src/evidence_pack_verifier/verifier.py:96-103,106-213` checks only the first
receipt hash/signature, with no schema-version, decision/reason, mirror,
secondary-receipt, or record validation. It therefore accepts signed unknown
versions/reasons that full checkers reject; those exact cells are pinned in the
single deliberate-difference table, not silently treated as semantic parity.
Its issuer-name allowlist (`verifier.py:15,164-166`) is distinct from tools'
caller-supplied local JWKS trust model; optional key discovery is at lines
149-156. The URL-only API and CLI (`cli.py:36-48`) receive the same golden bytes
through a blocked-network test transport, never an actual HTTP call. The
upstream package's own tests are run separately with mocked HTTP.

## Deliberate v1 differences

The executable table `DELIBERATE_V1_DIFFERENCES` in
`tools/test_reader_compatibility.py` is the single allowlist. Entries identify
exact reader families, vector names, verdicts, and reasons; a broad reader-wide
exception is not permitted. Differences not exercised by a vector are not
permission to drift. CLI trust defaults are tested separately with explicit
embedded opt-out, not by permitting network discovery.

Record verdicts are compared with the reference record reader, including
status, complete, trusted issuer and exit 4 for authenticated truncation.
Pack verdicts are compared with the reference tools pack reader. A record has
extra agent-identity/range/window requirements, so wrapping a valid pack does
not necessarily produce a complete valid record; all readers receive the same
wrapped bytes and must agree with the record reference.

## CI

`.github/workflows/ci.yml`, workflow `CI`, job `go` (`Go`), discovers
`tools/test_*.py` and runs demo provenance tests without path filters. The
matrix compiles and exercises the actual Go CLI, calls the actual Python SDK
wrapper, invokes every pack CLI, and isolates each record script with its own
sibling pack reader. Python commands run in virtual environments.

No new signed fields, protocol versions, producer switches, product sentences,
or released checker bytes are changed by this work.

## V2 reader extensions

The baseline readers do not acquire new formats. The maintained tools/demo
pack readers and the `receipt-bundle-v2` pack reader read raw v2 decisions and
separate execution observations under the binding-only development profile in
`RECEIPT-V2-CONTRACT.md`. The APIs/CLIs and offline SDK/Go wrappers consume
the decision and observation vectors. The matrix compares complete v2 API
verdicts across those pack readers, checks sibling-module discovery and byte
parity, and retains whole-v1-verdict comparisons. The historical v1
pack/record/linked artifacts retain rejection coverage; the immutable
`unavailable-check-v1` download is not rewritten. Public signatures alone cannot
establish the private attempt join of a row-03 decision: without its call opening
the observation verdict remains partial (`v2_attempt_not_checked`).

The tools and demo pack readers, tools receipt-record reader, and both
`receipt-bundle-v2` readers accept the private v2 JSON bundle under the same
offline issuer pins and return the same verdict. The SDK and Go CLI continue to
delegate to the pack reader; `SATGATE_VERIFIER` can select the distributed copy.
The matrix exercises the generated private bundle corpus across all of them.
The frozen v1 download, archived pin/release copies and signature-only linked
verifier retain their previous formats and reject the v2 bundle. The beta SDK
guards do not verify private bundles.

# SatGate public verifier tools

## Verify an Evidence Pack

Install the verifier dependencies in a temporary environment:

```bash
python3 -m venv .venv-verify
. .venv-verify/bin/activate
pip install -r tools/requirements-verify-evidence-pack.txt
```

Verify a public production Evidence Pack against issuer JWKS:

```bash
curl -fsS https://satgate.io/evidence/sample-mcp-budget-refusal-20260928.json -o pack.json
python tools/verify_evidence_pack.py pack.json \
  --discover-jwks \
  --require-trusted-issuer
```

The sample is a signed hosted-MCP refusal (`budget_exhausted`) from the homepage demo. `--discover-jwks` fetches the issuer key from the pack's `jwks_url` (`https://satgate-mcp-saas.fly.dev/.well-known/jwks.json`).

The verifier checks RFC 8785/JCS receipt canonicalization, `receipt_hash`, Ed25519 signature, issuer JWKS anchoring, top-level mirror fields, budget-state mirrors, optional pack hash, and obvious secret leakage.

## Verify a downloaded receipt record

### Versioned checker bundle

Minimum bundle for unavailable-check refusal receipts: `unavailable-check-v1`.
It supports `denied` / `tool_price_unavailable`, `tool_controls_unavailable`,
`budget_backend_error`, and `pending_capacity_exhausted`, with the exact verified-key,
no-evaluated-budget profile in `VERIFIER-CONTRACT.md`. SatGate does not yet sign
these blocks: this is a reader-only release. Checkers older than this release
reject them even with valid signatures. New readers still check old receipts.

The record script imports its sibling `verify_evidence_pack.py`: download both,
requirements and contract as one version, not an isolated record script. The
versioned directory is `tools/checker-bundles/unavailable-check-v1/`; obtain its
reviewed full commit SHA independently from the repository/release coordinator.
No tag or GitHub release is created by this change. Once reviewed, the coordinator
may tag that exact commit `checker-unavailable-check-v1`. Do not use mutable
`main` as an audit pin or trust a version/link supplied by the record itself.

Download from that immutable commit, then verify the hash list obtained from the
same trusted source before installing dependencies or executing code:

```bash
REF=<reviewed-full-commit-sha>
mkdir -p unavailable-check-v1
cd unavailable-check-v1
BASE="https://raw.githubusercontent.com/SatGate-io/satgate/$REF/tools/checker-bundles/unavailable-check-v1"
for FILE in verify_receipt_record.py verify_evidence_pack.py requirements-verify-evidence-pack.txt VERIFIER-CONTRACT.md SHA256SUMS; do
  curl -fsS "$BASE/$FILE" -o "$FILE" || exit 1
done
shasum -a 256 -c SHA256SUMS || exit 1
python3 -m venv .venv
. .venv/bin/activate
pip install -r requirements-verify-evidence-pack.txt
python verify_receipt_record.py /path/to/record.ndjson \
  --jwks-file /path/to/issuer-jwks.json --require-trusted-issuer
```

Keep these files and the reviewed ref with the exported record. Hashes show
matching bytes, not who authorized them; the repository/ref must be trusted
separately. Never auto-download/execute code from an untrusted record. The
separate linked `SatGate-io/evidence-pack-verifier` checks the first receipt's
signature, not this full semantic/record contract; it is not a substitute.

### Whole-record checks

A record is one agent key's receipts for a stretch of time: one Evidence Pack per line, then a signed complete manifest or cut-off terminal. Check all of it at once:

```bash
python tools/verify_receipt_record.py record.ndjson \
  --jwks-url https://api.satgate.io/.well-known/jwks.json \
  --require-trusted-issuer
```

For an offline check, replace `--jwks-url ...` with `--jwks-file issuer-jwks.json`, obtained from the intended issuer. A record can span signing-key rotations: each receipt and the terminal verify with their own `issuer_kid` from that issuer's trusted set. A missing key fails; every receipt's issuer must match the terminal's. The agent `key_hash` is distinct from these issuer signing keys.

It verifies every pack, the terminal's signature and fixed field set (unknown fields fail), the count and the hash over the receipt hashes in order (so a dropped or moved receipt is caught), the time range, duplicates, the one agent key, one receipt per line, and the refusal windows. A complete record's overdue closer makes it invalid. A signed cut-off terminal is checked against only the downloaded prefix; it cannot pass as complete. Output is JSON with a `summary` also printed on stderr.

Exit codes:

- `0`: verified complete record; `status: complete`, `valid: true`, `complete: true`.
- `1`: invalid record, including malformed NDJSON, bad signatures, unknown fields or prefix mismatches.
- `2`: input/JWKS loading or CLI usage error.
- `3`: missing `cryptography` dependency.
- `4`: authenticated cut-off record; `status: truncated`, `valid: false`, `complete: false`, with `truncation_reason` and `verified_prefix` (count, endpoint hashes, ordered hash digest, decision counts and windows). An open window in a cut-off prefix is `prefix_open`, not proof its closer is missing.

Without a trusted JWKS and `--require-trusted-issuer`, each embedded key proves self-consistency only, including for mixed-key records. Output says `trust_mode: embedded_key_self_consistency`, `trusted_issuer_valid: false` and warning `issuer_not_authenticated`. All signatures anchored to the supplied issuer set instead produce `trust_mode: trusted_issuer`. Do not treat `problems: []` alone as complete; authenticated truncation also has no verification problems but exits 4.

See `VERIFIER-CONTRACT.md` for the two terminal schemas, fields and what the checker cannot prove. No login is needed, and the checker does not automatically fetch keys from record contents.

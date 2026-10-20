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

A record is one agent key's receipts for a stretch of time: one Evidence Pack per line, then a signed manifest. Check all of it at once:

```bash
python tools/verify_receipt_record.py record.ndjson \
  --jwks-url https://api.satgate.io/.well-known/jwks.json \
  --require-trusted-issuer
```

It verifies every pack, the manifest's signature, the count and the hash over the receipt hashes in order (so a dropped or moved receipt is caught), the time range, duplicates, the one key, one receipt per line, the signing key, and the refusal windows (a window whose closer is overdue makes the record invalid). The last line of output is a summary such as `4 receipts, 1 refusals, 0 windows, 0 problems`. See `VERIFIER-CONTRACT.md` for the manifest and what the checker cannot prove.

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

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
python tools/verify_evidence_pack.py \
  https://api.satgate.io/v1/evidence/evid_QBBiz-GEI-stsaP6KS01-RL414Csuidv \
  --discover-jwks \
  --require-trusted-issuer
```

The verifier checks RFC 8785/JCS receipt canonicalization, `receipt_hash`, Ed25519 signature, issuer JWKS anchoring, top-level mirror fields, budget-state mirrors, optional pack hash, and obvious secret leakage.

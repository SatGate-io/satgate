# SatGate Trust Metadata

SatGate publishes a small trust metadata artifact at:

```text
https://satgate.io/.well-known/satgate
```

This is the first stable machine-readable discovery surface for agent runtimes, API gateways, and upstream services that want to understand what a SatGate issuer accepts and how receipts can be verified.

It is intentionally small. It does **not** make marketplace, reputation, endorsement, or compliance claims.

## Schema version

```json
{
  "schema_version": "satgate.trust_metadata.v1"
}
```

Clients should treat unknown fields as optional and preserve forward compatibility. Breaking changes require a new `schema_version`.

## Minimal fields

- `metadata_url`: canonical URL for this metadata document.
- `issuer`: SatGate issuer identity for the public artifact.
- `capability_acceptance`: accepted capability/token formats, bearer schemes, required claims, caveats, and delegation-depth semantics.
- `receipt_verification`: receipt and Evidence Pack formats, required receipt fields, supported decision labels, verification endpoint, and Evidence Pack schema URL.
- `rails_adapters`: supported rails/adapters beneath SatGate authority and receipts.
- `signing_key_discovery`: how to discover issuer signing keys for `issuer_kid` verification.
- `docs`: human-readable links for builders and auditors.

## Capability acceptance

The public artifact currently declares acceptance for:

- `satgate.capability.v1`
- `macaroon-bearer`

Required claims are deliberately minimal:

- `issuer`
- `subject`
- `audience`
- `scope`
- `expires_at`
- `policy_version`

Supported caveats include scope, route, tool, budget, tenant, expiry, and delegation depth. Delegation depth is `policy_defined`; agents should not assume standing authority or unlimited subdelegation.

## Receipt verification

Receipts are the proof spine. A verifier should expect receipt-grade decisions to carry at least:

- `receipt_id`
- `evidence_pack_id`
- `decision`
- `decision_reason`
- `policy_version`
- `receipt_hash`
- `signature`

Decision labels are limited to:

- `allowed`
- `denied`
- `delegated`
- `revoked`
- `paid`

Evidence Packs use the public schema at:

```text
https://satgate.io/evidence-packs/evidence-pack.schema.v1.json
```

## Rails/adapters

The artifact lists supported rails/adapters, not marketplaces:

- `mcp`
- `x402`
- `l402`
- `api_key_billing`
- `enterprise_ledger`

Rails sit below authority and receipt verification. SatGate proof is the receipt/Evidence Pack layer around the rail.

## Signing key discovery

`signing_key_discovery.mode` is `issuer_discovery`.

Receipts reference an `issuer_kid`. Verifiers should resolve the issuer from the receipt or Evidence Pack, then discover issuer keys using the declared `jwks_url_template`:

```text
{issuer_id}/.well-known/jwks.json
```

Tenant or deployment issuers may publish their own JWKS endpoint. The marketing-site trust metadata artifact does not embed production tenant keys.

## HTTP requirements

The live path must return:

- `200 OK`
- `Content-Type: application/json`
- `Cache-Control: public, max-age=3600`
- `Access-Control-Allow-Origin: *`
- valid JSON matching `satgate.trust_metadata.v1`

## Verification

```bash
curl -i https://satgate.io/.well-known/satgate
```

Minimal JSON check:

```bash
python3 - <<'PY'
import json, urllib.request
url = 'https://satgate.io/.well-known/satgate'
with urllib.request.urlopen(url, timeout=20) as r:
    assert r.status == 200
    assert 'application/json' in r.headers['content-type']
    data = json.load(r)
assert data['schema_version'] == 'satgate.trust_metadata.v1'
assert 'satgate.capability.v1' in data['capability_acceptance']['accepted_formats']
assert 'receipt_id' in data['receipt_verification']['required_receipt_fields']
assert data['signing_key_discovery']['mode'] == 'issuer_discovery'
print('satgate trust metadata ok')
PY
```

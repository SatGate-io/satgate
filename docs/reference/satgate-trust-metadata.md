# SatGate Trust Metadata

SatGate publishes a small trust metadata artifact at:

```text
https://satgate.io/.well-known/satgate
```

This is the first stable machine-readable discovery surface for agent runtimes, API gateways, and upstream services that want to understand what a SatGate issuer accepts and how receipts can be verified.

It is intentionally small. It does **not** make marketplace, reputation, endorsement, ranking, or compliance claims.

## Compatibility rule

Treat this like a versioned API:

- Clients should ignore unknown fields.
- SatGate may add fields within `satgate.trust_metadata.v1`.
- SatGate must not rename or remove existing v1 fields without a new `schema_version`.

## Schema version and schema URL

```json
{
  "schema_version": "satgate.trust_metadata.v1",
  "schema_url": "https://satgate.io/.well-known/satgate.schema.json",
  "roles": ["issuer"]
}
```

`roles` is present now so the issuer-side artifact does not paint us into a corner when upstream acceptor metadata exists later. The first public artifact is issuer-side only.

## Minimal fields

- `metadata_url`: canonical URL for this metadata document.
- `schema_url`: JSON Schema URL for this version.
- `roles`: metadata role list; currently `issuer` only.
- `issuer`: SatGate issuer identity and key-discovery metadata for the public artifact.
- `capability_acceptance`: accepted capability/token formats, bearer schemes, required claims, caveats, and delegation-depth semantics.
- `receipt_verification`: receipt and Evidence Pack formats, required receipt fields, supported decision labels, verification endpoint, and Evidence Pack schema URL.
- `rails_adapters`: supported rails/adapters beneath SatGate authority and receipts, as structured objects.
- `signing_key_discovery`: issuer-key discovery rule for `issuer_kid` verification.
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

Decision labels are a closed enum in v1:

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

The artifact lists supported rails/adapters as an array of objects, not free strings:

```json
{
  "id": "x402",
  "type": "payment_rail",
  "role": "external_paid_access"
}
```

Current adapter IDs:

- `mcp`
- `x402`
- `l402`
- `api_key_billing`
- `enterprise_ledger`

Rails sit below authority and receipt verification. SatGate proof is the receipt/Evidence Pack layer around the rail.

## Signing key discovery

The `issuer` block exposes key discovery as:

```json
{
  "method": "jwks_uri",
  "key_id_field": "issuer_kid",
  "jwks_uri": "https://satgate.io/.well-known/jwks.json"
}
```

`signing_key_discovery.mode` is `issuer_discovery` and keeps the general rule:

```text
{issuer_id}/.well-known/jwks.json
```

Receipts reference an `issuer_kid`. Verifiers should resolve the issuer from the receipt or Evidence Pack, then discover issuer keys using the issuer's JWKS URI. Tenant or deployment issuers may publish their own JWKS endpoint. The public `satgate.io` JWKS route currently publishes an empty `keys` array instead of placeholder production tenant keys.

## Future acceptor-side metadata

The mirror artifact is intentionally not shipped yet. The likely shape is either a future `roles: ["acceptor"]` profile in this schema or a companion `/.well-known/satgate-upstream` document advertising:

- accepted capability formats
- verification endpoint
- accepted receipt decisions
- rails/adapters the upstream settles on
- issuer allowlist or key-discovery policy

Do not add acceptor claims to the issuer artifact until an upstream actually accepts SatGate capabilities.

## HTTP requirements

The live path must return:

- `200 OK`
- `Content-Type: application/json`
- `Cache-Control: public, max-age=3600`
- `Access-Control-Allow-Origin: *`
- `X-Content-Type-Options: nosniff`
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
    assert r.headers['cache-control'] == 'public, max-age=3600'
    assert r.headers['access-control-allow-origin'] == '*'
    data = json.load(r)
assert data['schema_version'] == 'satgate.trust_metadata.v1'
assert data['schema_url'] == 'https://satgate.io/.well-known/satgate.schema.json'
assert data['roles'] == ['issuer']
assert 'satgate.capability.v1' in data['capability_acceptance']['accepted_formats']
assert 'receipt_id' in data['receipt_verification']['required_receipt_fields']
assert set(data['receipt_verification']['decisions']) == {'allowed','denied','delegated','revoked','paid'}
assert isinstance(data['rails_adapters']['supported'][0], dict)
assert data['issuer']['key_discovery']['method'] == 'jwks_uri'
print('satgate trust metadata ok')
PY
```

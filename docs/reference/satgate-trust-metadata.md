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
- `rails_adapters`: rail/protocol/billing adapter catalog beneath SatGate authority and receipts, as structured objects with explicit `status`.
- `signing_key_discovery`: issuer-key discovery rule for `issuer_kid` verification.
- `docs`: human-readable links for builders, verifiers, and auditors.

Human explanations belong in this document, not in prose-only `note` fields inside the machine artifact.

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
- `issuer`
- `issuer_kid`
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

The artifact lists rail/protocol/billing adapters as an array of objects, not free strings:

```json
{
  "id": "x402",
  "type": "payment_rail",
  "role": "external_paid_access",
  "status": "supported"
}
```

Current adapter catalog:

- `mcp`: `protocol` / `tool_transport` / `supported`
- `x402`: `payment_rail` / `external_paid_access` / `supported`
- `l402`: `payment_rail` / `external_paid_access` / `supported`
- `api_key_billing`: `billing_adapter` / `existing_vendor_billing` / `supported`
- `enterprise_ledger`: `ledger_adapter` / `internal_chargeback` / `supported`
- `agentcore_payments`: `payment_rail` / `external_paid_access` / `planned`
- `pay_sh`: `payment_rail` / `external_paid_access` / `planned`

Rails sit below authority and receipt verification. SatGate proof is the receipt/Evidence Pack layer around the rail. Marketing copy may mention planned rails, but the well-known artifact is canonical for whether a rail is currently supported or planned.

## Signing key discovery

The `issuer` block exposes key discovery for the manifest publisher as:

```json
{
  "method": "jwks_uri",
  "key_id_field": "issuer_kid",
  "jwks_uri": "https://satgate.io/.well-known/jwks.json"
}
```

The federation rule is:

```text
{issuer_id}/.well-known/jwks.json
```

Receipts reference both an issuer and an `issuer_kid`. Verifiers must first authorize the issuer origin against configured trust anchors, then fetch that issuer's JWKS, select the key by `issuer_kid`, and verify the receipt signature. Tenant or deployment issuers may publish their own JWKS endpoint. The public `satgate.io` JWKS route currently publishes an empty `keys` array instead of placeholder production tenant keys.

Do **not** treat the top-level `https://satgate.io/.well-known/jwks.json` as the universal key source for every receipt. It only represents the `https://satgate.io` issuer. Also do **not** trust any issuer URL found in a receipt until it matches an acceptor-configured trust anchor; otherwise an attacker can publish their own JWKS and sign their own fake receipt.

## Reference verifier snippets

These snippets show the discovery sequence only. Production verifiers must also canonicalize the signed payload, enforce expiry/replay checks, and verify the signature with the algorithm declared by the selected JWK.

### Python

```python
import json
from urllib.request import urlopen


TRUSTED_ISSUERS = {"https://satgate.io"}


def load_jwks_for_receipt(receipt: dict) -> dict:
    issuer_id = receipt["issuer"].rstrip("/")
    issuer_kid = receipt["issuer_kid"]
    if issuer_id not in TRUSTED_ISSUERS:
        raise PermissionError(f"untrusted receipt issuer: {issuer_id}")
    jwks_url = issuer_id + "/.well-known/jwks.json"

    with urlopen(jwks_url, timeout=10) as response:
        jwks = json.load(response)

    for key in jwks.get("keys", []):
        if key.get("kid") == issuer_kid:
            return key
    raise LookupError(f"issuer key not found: {issuer_id} kid={issuer_kid}")


def verify_receipt(receipt: dict) -> bool:
    key = load_jwks_for_receipt(receipt)
    # Verify receipt["signature"] over the canonical receipt payload with `key`.
    # Keep this line explicit so implementers do not accidentally fetch satgate.io's empty JWKS for tenant receipts.
    return bool(key)
```

### Node.js

```ts
type Receipt = {
  issuer: string;
  issuer_kid: string;
  signature: string;
};

const trustedIssuers = new Set(["https://satgate.io"]);

export async function loadJwksKeyForReceipt(receipt: Receipt) {
  const issuer = receipt.issuer.replace(/\/$/, "");
  if (!trustedIssuers.has(issuer)) throw new Error(`untrusted receipt issuer: ${issuer}`);
  const jwksUrl = `${issuer}/.well-known/jwks.json`;
  const response = await fetch(jwksUrl, { headers: { accept: "application/json" } });
  if (!response.ok) throw new Error(`JWKS fetch failed: ${response.status}`);

  const jwks = await response.json() as { keys?: Array<{ kid?: string }> };
  const key = jwks.keys?.find((candidate) => candidate.kid === receipt.issuer_kid);
  if (!key) throw new Error(`issuer key not found: ${receipt.issuer} kid=${receipt.issuer_kid}`);
  return key;
}

export async function verifyReceipt(receipt: Receipt) {
  const key = await loadJwksKeyForReceipt(receipt);
  // Verify receipt.signature over the canonical receipt payload with `key`.
  // Do not fetch satgate.io's empty JWKS unless receipt.issuer === "https://satgate.io".
  return Boolean(key);
}
```

## Future acceptor-side metadata

The mirror artifact is intentionally not on the wire yet. A draft is tracked in [Acceptor Metadata Draft](acceptor.md). An acceptor needs to advertise at least:

- `verification_endpoint`
- `accepted_capability_formats`
- `accepted_receipt_decisions`
- `trust_anchors` / issuer allowlist
- rails/adapters it can settle on

Do not add acceptor claims to the issuer artifact until an upstream actually accepts SatGate capabilities.

## HTTP requirements

The live manifest path must return:

- `200 OK`
- `Content-Type: application/json`
- `Cache-Control: public, max-age=3600`
- `Access-Control-Allow-Origin: *`
- `X-Content-Type-Options: nosniff`
- valid JSON matching `satgate.trust_metadata.v1`

The schema path uses a longer cache TTL because it changes less often:

- `https://satgate.io/.well-known/satgate.schema.json`: `Cache-Control: public, max-age=86400`

`Vary` is platform-managed by the CDN/framework and should not be treated as part of the SatGate metadata contract.

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
    assert r.headers['x-content-type-options'] == 'nosniff'
    data = json.load(r)
assert data['schema_version'] == 'satgate.trust_metadata.v1'
assert data['schema_url'] == 'https://satgate.io/.well-known/satgate.schema.json'
assert data['roles'] == ['issuer']
assert 'satgate.capability.v1' in data['capability_acceptance']['accepted_formats']
assert {'receipt_id', 'issuer', 'issuer_kid'} <= set(data['receipt_verification']['required_receipt_fields'])
assert set(data['receipt_verification']['decisions']) == {'allowed','denied','delegated','revoked','paid'}
assert isinstance(data['rails_adapters']['supported'][0], dict)
adapters = {adapter['id']: adapter for adapter in data['rails_adapters']['supported']}
assert adapters['agentcore_payments']['status'] == 'planned'
assert adapters['pay_sh']['status'] == 'planned'
assert {adapter['status'] for adapter in adapters.values()} <= {'supported','planned'}
assert data['issuer']['key_discovery']['method'] == 'jwks_uri'
print('satgate trust metadata ok')
PY
```

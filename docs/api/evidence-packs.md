# Evidence Pack API

Evidence Packs are SatGate's Policy-to-Proof exports. They are designed for auditors, CISOs, FinOps, incident review, and buyer demos.

## Export latest Evidence Pack

```http
GET /api/governance/evidence-pack
X-Tenant-ID: tenant_acme_finance
X-API-Key: <tenant api key>
```

Response: `application/json`

```json
{
  "schema_version": "satgate.evidence_pack.v1",
  "evidence_pack_id": "ep_177...",
  "tenant_id": "tenant_acme_finance",
  "policy_snapshot": {},
  "budget_snapshot": {},
  "authority_chain": [],
  "receipts": [],
  "payment_context": {},
  "receipt_chain": {},
  "chain_root": "sha256:...",
  "signature": "ed25519:..."
}
```

The runtime OSS export is a usage-derived snapshot. Enterprise exports should source receipts from the persistent audit log.

## Authentication

- `X-Tenant-ID` selects the tenant export boundary.
- `X-API-Key` authenticates tenant API callers in Cloud/client flows.
- Admin-only deployment paths may use the configured admin bearer token.

Do not expose raw macaroons, bearer tokens, prompts, customer payloads, or unredacted settlement secrets in exported packs.

## Canonical schema

The canonical schema is `satgate.evidence_pack.v1`.

Public references:

- Viewer: <https://satgate.io/evidence-pack-demo>
- Schema: <https://satgate.io/evidence-packs/evidence-pack.schema.v1.json>
- Sample fixture: <https://satgate.io/evidence-packs/sample-evidence-pack.v1.json>

## Export semantics

Each export declares:

- export type: manual, scheduled, auditor request, or public demo
- redaction profile
- included receipt range
- formats
- completeness
- export timestamp

The export event is itself a receipt, so the act of producing proof is auditable.

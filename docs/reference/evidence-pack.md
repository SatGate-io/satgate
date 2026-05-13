# SatGate Evidence Pack v1

An Evidence Pack is SatGate's canonical Policy-to-Proof artifact. It is the buyer-readable and machine-readable export that answers:

- Who authorized the agent?
- Which agent acted?
- What was it allowed to do?
- Which policy and budget constrained it?
- Was authority delegated or attenuated?
- Which paid rail or ledger was involved?
- What was allowed, denied, revoked, and exported?
- Can the receipt chain be verified?

The public demo fixture is available at:

- Viewer: <https://satgate.io/evidence-pack-demo>
- JSON fixture: <https://satgate.io/evidence-packs/sample-evidence-pack.v1.json>
- JSON Schema: <https://satgate.io/evidence-packs/evidence-pack.schema.v1.json>

## Required top-level fields

```json
{
  "schema_version": "satgate.evidence_pack.v1",
  "evidence_pack_id": "ep_demo_2026_05_10_001",
  "environment": "demo",
  "pack_type": "public_demo_redacted",
  "issued_at": "2026-05-10T14:26:31Z",
  "tenant": { "id": "tenant_acme_finance", "name": "Acme Finance" },
  "subject": { "id": "agent:invoice-reconciler-worker", "kind": "agent" },
  "policy_snapshot": {},
  "budget_snapshot": {},
  "authority_chain": [],
  "receipts": [],
  "payment_context": {},
  "receipt_chain": {},
  "redaction": {},
  "export": {},
  "verification": {},
  "chain_root": "sha256:...",
  "signature": "ed25519:..."
}
```

## Identity model

Evidence Packs use structured identities instead of raw strings. Actors can be humans, agents, service accounts, tools, gateways, or policy engines.

Required identity fields:

- `id`: canonical identifier, for example `agent:invoice-reconciler-worker`
- `kind`: `agent`, `human`, `service_account`, `policy_engine`, `gateway`, or `tool`
- `tenant_id`: tenant boundary for the actor

Tokens and macaroons must not be exported raw. Use token fingerprints, caveat digests, issuer key IDs, and receipt hashes.

## Policy snapshot

`policy_snapshot` identifies the exact rule set that governed the workflow:

- `policy_id`
- `policy_version`
- `policy_digest`
- `mode`: Observe, Control, Prove, or Charge
- `decision_model`: `authority_before_execution`
- `matched_rules`
- `obligations`

The point is not just “a policy allowed this.” The pack must name which policy version decided it.

## Budget snapshot

`budget_snapshot` captures the relevant budget state:

- `budget_id`
- `owner`
- `cost_center`
- `currency`
- `root_limit`
- `delegated_limit`
- `spent`
- `remaining`
- `reset_window`
- `exhausted`
- `ledger_root`

Amounts should be decimal strings, not floats.

## Authority chain

`authority_chain` is ordered from root grant to delegated child capability. A delegated worker capability should prove attenuation:

- child scope is narrower than parent scope
- child budget is less than or equal to parent remaining budget
- child delegation depth is within the configured maximum
- child expiry is no later than parent expiry
- child caveats are equal or stricter

## Receipts

Receipts are ordered, hash-linked events. Common receipt types:

- `mint`
- `delegation`
- `spend`
- `denial`
- `revocation`
- `post_revoke_denial`
- `export`

Every receipt should include:

- `receipt_id`
- `seq`
- `type`
- `ts`
- `actor`
- `action`
- `resource`
- `result`
- `policy_decision_ref`
- `budget_ledger_ref`
- `prev_receipt_hash`
- `receipt_hash`

## Paid-rail context

SatGate is rail-neutral. `payment_context` records the rail or ledger involved without making the rail the product.

Supported examples:

- `enterprise_ledger`
- `x402`
- `l402`
- `api_key_billing`
- `enterprise_contract`

Payment proves value moved. SatGate proves the agent had authority to move it.

## Redaction

Production Evidence Packs should support redacted exports. The redaction block must state:

- redaction profile
- redacted fields
- method, such as masked values with hash commitments
- whether the pack remains directly verifiable
- original and redacted chain roots when applicable

Never leak raw bearer tokens, macaroons, prompts, customer payloads, or full external transaction secrets in buyer-safe exports.

## Verification

`receipt_chain` defines canonicalization and hash semantics. SatGate v1 uses:

- canonicalization: `RFC8785-JCS`
- hash algorithm: `sha256`
- chain type: `linear_hash_chain`

The top-level signature signs the Evidence Pack envelope and receipt-chain root. Demo fixtures are explicitly marked as non-verifiable because they use deterministic placeholder hashes and a redacted demo signature.

## Event history and future verifier conclusions

Evidence Pack v1 may include optional `event_history` entries. Event history is the substrate for verifier-observed conclusions such as stayed within budget, completed task, looped/retried, revoked/attenuated, vouched by principal, or accepted by upstream. It is not a public score surface. Reputation, if ever exposed, must fall out of signed receipts and verifier-observed history rather than SatGate assertion.

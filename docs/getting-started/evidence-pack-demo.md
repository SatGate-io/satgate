# Evidence Pack demo walkthrough

This guide shows the buyer moment SatGate should end on: one artifact that proves authority, policy, budget, delegation, paid-rail context, denial, revocation, and export.

Open the public demo:

- Viewer: <https://satgate.io/evidence-pack-demo>
- JSON: <https://satgate.io/evidence-packs/sample-evidence-pack.v1.json>
- Schema: <https://satgate.io/evidence-packs/evidence-pack.schema.v1.json>

## Demo lifecycle

1. **Mint** — a security admin mints a scoped capability for a parent finance agent.
2. **Delegate** — the parent agent delegates narrower authority to an invoice-reconciler worker.
3. **Spend** — the worker searches and compares invoices inside its delegated budget.
4. **Paid rail** — the worker calls a document-AI OCR tool with x402 context preserved in the pack.
5. **Deny scope** — a customer-data export attempt is blocked before data leaves.
6. **Deny budget** — a reconciliation call is blocked after the delegated budget is exhausted.
7. **Revoke** — security revokes the worker capability.
8. **Post-revoke deny** — the worker retries and is blocked.
9. **Export** — SatGate emits the Evidence Pack and records the export as its own receipt.

## Buyer comprehension checklist

A prospect should be able to answer these in under two minutes:

- Who authorized the worker?
- Which agent actually acted?
- What was the worker allowed to do?
- Which policy version made the decisions?
- Which budget constrained the worker?
- What was delegated, and how was it attenuated?
- Why was customer-data export blocked?
- Why did the over-budget call fail?
- Which paid rail was involved?
- What does payment prove, and what does SatGate prove beyond payment?
- Is this a full, redacted, demo, or auditor export?

## CLI export

Runtime exports use the governance API:

```bash
satgate-cli proof export --tenant tenant_acme_finance > evidence-pack.json
```

The CLI sends `X-Tenant-ID` and `X-API-Key` to:

```text
GET /api/governance/evidence-pack
```

## API export

```bash
curl -s https://gateway.example.com/api/governance/evidence-pack \
  -H "X-Tenant-ID: tenant_acme_finance" \
  -H "X-API-Key: $SATGATE_API_KEY" \
  -o evidence-pack.json
```

The response follows `satgate.evidence_pack.v1` and includes the legacy `decisions` field only as a compatibility alias.

## Talk track

Use this phrasing:

> Payment proves value moved. SatGate proves the agent was allowed to move it.

Then show:

- the authority chain
- the policy snapshot
- the budget snapshot
- the x402 payment-context event
- the scope denial
- the budget denial
- the post-revoke denial
- the receipt-chain verification block

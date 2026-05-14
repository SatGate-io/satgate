# SatGate Capability Schema

Status: **v1 design contract — capability material can be transported as macaroon bearer tokens, SDK objects, or signed envelopes depending on deployment profile.**

A SatGate capability answers one machine question: *what is this agent allowed to do before the action happens?*

## Required semantic fields

A capability profile must bind at least these fields, whether the wire encoding is JSON, a macaroon caveat set, or an SDK object:

- `schema_version`: `satgate.capability.v1`
- `capability_id`: stable identifier or derived hash handle
- `issuer`: trusted issuer origin
- `subject`: agent, workload, or delegated actor receiving authority
- `principal`: human, tenant, service, or account that granted authority
- `allow`: route, tool, method, or audience allowlist
- `budget`: max spend or credit envelope
- `expires_at`: absolute expiry timestamp
- `delegation_depth`: maximum remaining attenuation depth
- `caveats`: additional constraints such as tenant, route, method, model, amount, region, or time window

## Verification rules

1. Validate the capability's issuer against verifier trust anchors.
2. Confirm the capability has not expired.
3. Confirm the requested route/tool/action is included in `allow`.
4. Confirm the request stays within budget and any caller-supplied max spend.
5. Confirm delegation depth and caveat attenuation never widen authority.
6. Bind the decision receipt to either `capability_id` or `capability_hash`.

## Minimal JSON profile

```json
{
  "schema_version": "satgate.capability.v1",
  "capability_id": "cap_2Xn83k",
  "issuer": "https://api.satgate.io",
  "subject": "research-agent",
  "principal": "tenant_54bb5860",
  "allow": ["mcp:web.search", "api:prices.read"],
  "budget": { "amount": "25.00", "currency": "USD" },
  "expires_at": "2026-05-14T12:00:00Z",
  "delegation_depth": 1,
  "caveats": [
    { "type": "tenant", "value": "tenant_54bb5860" },
    { "type": "max_request_amount", "amount": "4.20", "currency": "USD" }
  ]
}
```

## Related machine artifacts

- [Receipt schema](receipt-schema.md)
- [Evidence Pack profile](evidence-pack.md)
- [Trust metadata](satgate-trust-metadata.md)
- [Accept SatGate capabilities upstream](accept-satgate-capabilities.md)

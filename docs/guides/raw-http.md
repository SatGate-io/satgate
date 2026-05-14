# Raw HTTP issue/pay/verify

Use this when you do not want an SDK. The voice is developer-first: issue, pay, verify.

## 1. Issue a scoped capability

```bash
curl https://api.satgate.io/v1/issue \
  -H "Authorization: Bearer $SATGATE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "research-agent",
    "task": "research market prices",
    "allow": ["mcp:web.search", "api:prices.read"],
    "budget_usd": 25,
    "expires_in": "1h",
    "delegation_depth": 1
  }'
```

## 2. Pay or invoke upstream with a max budget

```bash
curl https://api.satgate.io/v1/pay \
  -H "Authorization: Bearer $SATGATE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "capability": "cap_...",
    "upstream": "https://api.example.com/search",
    "max_usd": 4.20
  }'
```

## 3. Verify the returned receipt

```bash
curl https://api.satgate.io/v1/verify \
  -H "Authorization: Bearer $SATGATE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"receipt":"rcpt_..."}'
```

## Machine contracts

- [Capability schema](../reference/capability-schema.md)
- [Receipt schema](../reference/receipt-schema.md)
- [Evidence Pack profile](../reference/evidence-pack.md)

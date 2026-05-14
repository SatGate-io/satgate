# SatGate Documentation

SatGate docs are organized around three voices. Use the voice that matches the reader, then cross-link only when the next layer is needed.

## 1. Marketing voice: govern / enforce / prove

For buyers, operators, and security leaders. The story is not a payment rail. It is Policy-to-Proof governance for agents.

- **Govern**: define scoped authority before an agent can act.
- **Enforce**: apply policy, budget, route/tool scope, expiry, and delegation limits before upstream access.
- **Prove**: return signed receipts and Evidence Packs for allowed, denied, delegated, revoked, and settlement-aware decisions.

Start here:

- [Core concepts](getting-started/concepts.md)
- [Policy-to-Proof Evidence Packs](reference/evidence-pack.md)
- [Trust metadata](reference/satgate-trust-metadata.md)
- [Production checklist](operations/production-checklist.md)

## 2. Developer voice: issue / pay / verify

For app teams and agent builders. The developer primitive is three calls:

1. `issue` a scoped capability for a task.
2. `pay` or invoke an upstream path with a caller-supplied max budget.
3. `verify` the receipt and attach Evidence Pack proof.

Build paths:

- [Quickstart](getting-started/quickstart.md)
- [Raw HTTP issue/pay/verify](guides/raw-http.md)
- [MCP integration](guides/mcp-gateway.md)
- [OpenAI tools example](guides/openai-tools.md)
- [Anthropic tools example](guides/anthropic-tools.md)
- [LangChain integration](guides/langchain-integration.md)
- [CrewAI example](guides/crewai.md)
- [Node.js SDK](sdks/nodejs.md)
- [Python SDK](sdks/python.md)

## 3. Machine voice: schemas / signatures / receipts

For verifiers, gateways, acceptors, and auditors. These docs define the machine-readable contract.

- [Capability schema](reference/capability-schema.md)
- [Receipt schema](reference/receipt-schema.md)
- [Evidence Pack schema and verifier profile](reference/evidence-pack.md)
- [Acceptor metadata](reference/acceptor.md)
- [Accept SatGate capabilities upstream](reference/accept-satgate-capabilities.md)
- [Metadata cache protocol notes](reference/cache-protocol.md)

## Documentation information architecture

```text
SatGate docs
├── Marketing: govern / enforce / prove
│   ├── Core concepts
│   ├── Policy-to-Proof Evidence Packs
│   └── Production governance checklist
├── Developer: issue / pay / verify
│   ├── Quickstart
│   ├── Raw HTTP
│   ├── MCP
│   ├── OpenAI tools
│   ├── Anthropic tools
│   ├── LangChain
│   └── CrewAI
└── Machine: schemas / signatures / receipts
    ├── Capability schema
    ├── Receipt schema
    ├── Evidence Pack schema
    ├── JWKS / issuer metadata
    └── Acceptor metadata
```

## Vocabulary guardrails

Use these guardrails when adding docs:

- Prefer **govern / enforce / prove** for the buyer story.
- Prefer **issue / pay / verify** for developer examples.
- Prefer **schemas / signatures / receipts** for protocol docs.
- Treat L402, x402, Stripe, credits, ledgers, and future rails as settlement context below the authority layer.
- Do not lead with rail-first language. SatGate governs authority and evidence across rails; it is not a wallet-first marketplace.
- Use **Evidence Pack** for proof artifacts. Use "audit" only as the buyer outcome, not as the protocol noun.

## Deployment and operations

- [Route configuration](configuration/routes.md)
- [Policy scope](configuration/policy-scope.md)
- [Lightning providers](configuration/lightning-providers.md)
- [Kubernetes deployment](guides/kubernetes.md)
- [Backup and restore](operations/backup-restore.md)
- [SDK adoption dashboard](operations/sdk-adoption-dashboard.md)

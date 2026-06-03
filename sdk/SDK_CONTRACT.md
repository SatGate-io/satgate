# SatGate SDK Contract and Package Policy

## Canonical client contract

The OSS SDK is the canonical client contract. Both OSS (`@satgate/sdk`, `satgate`) and Enterprise (`@satgate/client`, `satgate`) must expose the same top-level developer primitive:

- `SatGate.issue(payload)` → `POST /v1/issue`, with compatibility fallback to `POST /v1/capabilities` only when `/v1/issue` returns `404`.
- `SatGate.pay(payload)` → `POST /v1/pay`.
- `SatGate.verify(receipt)` → `POST /v1/verify` with `{ receipt }` wrapping.

The canonical payload shape uses developer-friendly camelCase in Node examples and snake_case in Python examples. SDKs serialize wire payloads to snake_case (`budget_usd`, `expires_in`, `max_usd`, `receipt_id`). Missing credentials must fail locally with `SatGateAuthError` and the `cloud.satgate.io/docs` access CTA; it must not perform a network request or return mocked receipts.

## Enterprise overlays

Enterprise is maintained as a thin overlay over the OSS contract, not as a forked client surface. Enterprise-only APIs must remain explicit lower-level namespaces:

- Node: `SatGateClient.config`, `SatGateClient.stats`, and enterprise identity providers exported from `SatGateAgentClient`.
- Python: `SatGateClient.config`, `SatGateClient.stats`, and enterprise identity providers exported from `SatGateAgentClient`.

Do not add Enterprise-only methods to the top-level `SatGate` issue/pay/verify facade. If an Enterprise capability graduates to the developer primitive, add it to OSS first and update the shared contract tests in both repos.

## Drift comparison snapshot

Current SDK comparison before this refactor:

- Node identical/near-identical: `delegation.ts`, `tsconfig.json`, `errors.ts` mostly aligned.
- Node divergent: `client.ts`, `agent-client.ts`, `types.ts`, README/package metadata. Enterprise adds config/stats/admin v1 APIs and identity providers.
- Node OSS-only: `private-beta.ts`, issue/pay/verify tests, Jest/ESLint config.
- Python near-identical: `delegation.py`, `exceptions.py`, `langchain.py`, `setup.py` metadata.
- Python divergent: `client.py`, `agent_client.py`, `models.py`, README/package metadata. Enterprise adds config/stats/admin v1 APIs and identity providers.
- Python OSS-only: `private_beta.py`, issue/pay/verify tests.

This refactor intentionally keeps the divergent Enterprise admin/agent APIs as overlays while copying the canonical issue/pay/verify facade and tests into Enterprise.

## Version/package policy

- OSS Node package: `@satgate/sdk`; OSS Python package: `satgate`.
- Enterprise Node package: `@satgate/client`; Enterprise Python source package currently remains `satgate` for compatibility, but must document that it is the Enterprise distribution and must not claim to be `@satgate/sdk`.
- OSS owns semantic changes to `SatGate.issue/pay/verify`. Enterprise may lag only by patch-level implementation details, never by contract behavior.
- Contract tests are release gates for both packages. A package version bump that changes issue/pay/verify must update this file and both language contract tests.

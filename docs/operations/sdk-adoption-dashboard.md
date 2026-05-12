# SatGate SDK adoption dashboard

This is the lightweight post-`/build` adoption loop. Use public/package-manager signals first; do **not** add quiet SDK phone-home telemetry.

## Runbook

```bash
python3 scripts/sdk_adoption_snapshot.py
```

Optional GitHub traffic requires an authenticated `gh` session or `GITHUB_TOKEN` with repo access:

```bash
GITHUB_TOKEN=... python3 scripts/sdk_adoption_snapshot.py
```

## Signals to watch

- npm package version and downloads for `@satgate/sdk`
- PyPI package version and recent downloads for `satgate`
- GitHub repo views/clones when authenticated
- `/build → GitHub/docs/package` click paths from Vercel analytics, if available
- `cloud.satgate.io/request-access` submissions after developers hit `SatGateAuthError`

## Interpretation

Download counts are noisy: CI, mirrors, bots, and retries all inflate them. Watch the curve:

- Spike then flatline: the page is interesting, but the SDK path is not sticky yet.
- Week-over-week growth: the `issue/pay/verify` primitive is landing.
- Downloads without request-access submissions: the docs funnel is losing committed developers.

## Telemetry rule

No hidden SDK telemetry. If SDK instrumentation is ever added, it must be obvious in README/package docs, payload-free, and opt-out via an environment variable such as:

```bash
SATGATE_TELEMETRY=0
```

Allowed anonymous fields, if explicitly enabled later:

- SDK package/version
- runtime/language
- method name (`issue`, `pay`, `verify`)
- success/error class

Forbidden fields:

- prompts, payloads, receipts, URLs, tenant IDs, emails, API keys, payment details, or upstream request content.

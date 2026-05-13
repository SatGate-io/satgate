# Economic Firewall Quick Start

Set up SatGate as an economic firewall — observe, control, then charge — in three steps.

## Step 1: Observe (Shadow Mode)

Start by watching what agents are doing without blocking anything:

```yaml
version: 1
server:
  listen: ":8080"
admin:
  token: "${ADMIN_TOKEN}"
lightning:
  provider: mock

upstreams:
  my-api:
    url: "https://your-api.example.com"

routes:
  - name: observe-all
    match:
      pathPrefix: /
    upstream: my-api
    policy:
      kind: observe    # Alias for chargeback — verify, allow, meter
```

This logs every request with cost attribution. Check the [SatGate Cloud dashboard](https://cloud.satgate.io) to see usage patterns.

## Step 2: Control (Budget Enforcement)

Once you understand usage, switch to budget enforcement:

```yaml
routes:
  - name: controlled-api
    match:
      pathPrefix: /api/
    upstream: my-api
    policy:
      kind: control     # Alias for fiat402 — verify, enforce budget, allow/deny
      costCredits: 1
      pay:
        mode: fiat402
        price: 0.01
        unit: USD
        enforceBudget: true
        costCenterHeader: X-Cost-Center
```

When an agent's budget hits zero, requests are **blocked**. Not merely logged or alerted. Blocked, with a denial receipt for the Evidence Pack.

## Step 3: Charge (Lightning Payments)

Monetize your API with sub-cent Lightning payments:

```yaml
routes:
  - name: paid-api
    match:
      pathPrefix: /api/premium
    upstream: my-api
    policy:
      kind: charge      # Alias for l402 — require payment proof
      priceSats: 5
```

Agents pay per request via the L402 protocol (HTTP 402 + Lightning invoice).

## Mix and Match

A real deployment uses all three:

```yaml
routes:
  - name: paid-premium
    match:
      pathPrefix: /api/premium
    upstream: my-api
    policy:
      kind: l402
      priceSats: 25

  - name: controlled-standard
    match:
      pathPrefix: /api/v1
    upstream: my-api
    policy:
      kind: fiat402
      costCredits: 1
      pay:
        mode: fiat402
        price: 0.01
        unit: USD
        enforceBudget: true

  - name: observed-internal
    match:
      pathPrefix: /internal
    upstream: my-api
    policy:
      kind: chargeback

  - name: public-docs
    match:
      pathPrefix: /docs
    upstream: my-api
    policy:
      kind: public
```

**Default protection, choose your economic policy.**

<p align="center">
  <img src="logo_blue_transparent.png" alt="SatGate Logo" width="200">
</p>

# SatGate™ — Competitive Analysis

> **Last updated:** 2026-02-04
>
> Enterprise-grade API protection and payments for the Agent Economy
>
> **SatGate™ Gateway Documentation**

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [SatGate vs Aperture (Lightning Labs)](#satgate-vs-aperture-lightning-labs)
3. [Broader Competitive Landscape](#broader-competitive-landscape)
4. [Head-to-Head Comparisons](#head-to-head-comparisons)
5. [Where Competitors Are Stronger (Honest Assessment)](#where-competitors-are-stronger)
6. [Why "Economic Firewall" Is a Different Category](#why-economic-firewall-is-a-different-category)
7. [Migration from Aperture](#migration-from-aperture)
8. [When to Use What](#when-to-use-what)
9. [Resources](#resources)

---

## Executive Summary

SatGate originated from L402 payments on Aperture (Lightning Labs). It extends the L402 protocol into a full **Economic Firewall** — combining API protection, micropayments, enterprise budget control, and agent governance into a single gateway.

No single competitor covers the same surface area. The competitive landscape splits into:

| Category | Competitors | SatGate Overlap |
|----------|------------|-----------------|
| **L402 / Lightning-native** | Aperture | Direct — SatGate is the enterprise evolution |
| **Traditional API Gateway** | Kong, Envoy, Traefik | Proxy/routing layer — but no payments or economic controls |
| **API Key Management** | Unkey | Auth tokens — but no macaroons, no payments, no delegation |
| **AI/MCP Gateway** | Zuplo | Agent governance — but no economic layer |
| **API Monetization** | Stripe Billing, Moesif | Usage metering — but bolt-on, not gateway-native |

**SatGate's thesis:** In the agent economy, authentication, authorization, and economic controls must be fused at the gateway layer. Bolting payments onto an API gateway (or governance onto a payment proxy) creates gaps that autonomous agents will exploit.

---

## SatGate vs Aperture (Lightning Labs)

### Aperture: Current State (as of Feb 2026)

**Aperture is NOT stale.** This is important to get right. Contrary to what you might assume:

- **Active commits** through Jan 2026 (most recent: Jan 27, 2026)
- **Key recent work:** Configurable per-endpoint rate limiting with L402 token awareness (PR #195, merged Jan 2026)
- **Maintained by:** Primarily `hieblmi` with occasional contributions from `guggero`, `Roasbeef`, and `starius`
- **Production use:** Lightning Loop (Lightning Labs' own service)
- **Latest tagged release:** v0.3-beta (Nov 2023) — no new tagged releases despite ongoing commits
- **LND bump:** Updated to LND v0.20.0-beta (Jan 2026)

However, the scope remains narrow: Aperture is a **reverse proxy for L402 payments**. It does that one thing and does it fine for its intended use case.

### Feature Comparison

| Feature | Aperture | SatGate Gateway |
|---------|:--------:|:---------------:|
| **L402 Lightning Payments** | ✅ | ✅ |
| **Rate Limiting** | ✅ Per-endpoint, token-aware (new) | ✅ Per-route, Redis HA, multi-strategy |
| **gRPC + REST proxying** | ✅ | ✅ |
| **Sqlite/Postgres backends** | ✅ | ✅ |
| **LNC (Lightning Node Connect)** | ✅ | ❌ (not yet) |
| **Capability Tokens (no payment)** | ❌ | ✅ |
| **Fiat402 (JWT receipts)** | ❌ | ✅ |
| **Chargeback/Showback Mode** | ❌ | ✅ |
| **Multi-tenant Isolation** | ❌ | ✅ |
| **Token Governance (ban/lineage)** | ❌ | ✅ |
| **Delegation Trees** | ❌ | ✅ |
| **Admin Dashboard UI** | ❌ | ✅ (60+ pages) |
| **RBAC** | ❌ | ✅ |
| **SCIM 2.0 Provisioning** | ❌ | ✅ |
| **GitOps Signed Configs** | ❌ | ✅ |
| **Audit Logging (WORM)** | ❌ | ✅ |
| **Budget Management** | ❌ | ✅ |
| **mTLS to Upstreams** | ❌ | ✅ |
| **Circuit Breaker** | ❌ | ✅ |
| **Prometheus Metrics** | Basic | ✅ Full observability |
| **Helm Chart** | Community | ✅ Official |
| **Client SDKs** | Go only | ✅ Go, Node.js, Python |
| **Supply Chain Security** | ❌ | ✅ Cosign, SBOM, attestations |
| **Lightning Providers** | LND only | Phoenixd, LND, CLN, Alby, LNbits, Strike |

### Where Aperture Wins

Be honest:

- **Simplicity.** Aperture is ~5K lines of Go. Deploy it in minutes. SatGate is a full platform with a learning curve.
- **Lightning Node Connect.** LNC support for connecting to remote LND nodes through the Lightning Terminal — SatGate doesn't have this yet.
- **Battle-tested in production.** Running Lightning Loop at scale. SatGate is newer and proving itself.
- **Lightning Labs backing.** Part of the Lightning Labs ecosystem. If you're already in their stack (LND + Loop + Pool + Faraday), Aperture is the natural fit.
- **Zero dependencies.** No Redis, no Postgres required for basic operation. Just the binary and an LND node.

### Where SatGate Wins

- **Enterprise readiness.** Multi-tenant, RBAC, audit, compliance — none of which Aperture has.
- **Payment flexibility.** Three modes (Capability, L402, Fiat402) vs. one (L402 only). Start free, add payments later.
- **Token governance.** Delegation trees, ban/revoke, lineage tracking — critical for autonomous agents.
- **Provider choice.** Not locked to LND. Phoenixd, CLN, Alby, etc.
- **Dashboard & ops tooling.** 60+ page admin UI, config generators, preflight checks, support bundles.
- **Budget controls.** Enterprise spending limits, chargeback/showback — things enterprises actually need.

### The Real Difference

Aperture answers: *"How do I charge for an API with Lightning?"*

SatGate answers: *"How do I protect, govern, and monetize APIs when the clients are autonomous agents with delegated economic authority?"*

Different questions. Different products.

---

## Broader Competitive Landscape

### Kong Gateway

**What it is:** The 800-pound gorilla of API gateways. 35K+ GitHub stars, 312M+ downloads, 400B+ API calls/day processed. NGINX-based, plugin-driven architecture.

**Enterprise features (Kong Enterprise/Konnect):**
- RBAC, workspaces, audit logging
- Developer portal, API analytics
- Advanced rate limiting, request validation
- FIPS 140-2 compliance, SBOM
- OpenTelemetry, consumer groups
- Kubernetes-native ingress controller

**Relevance to SatGate:** Kong is the gold standard for traditional API gateways. Any enterprise buyer considering SatGate will ask "why not just use Kong?"

| Dimension | Kong | SatGate |
|-----------|------|---------|
| API proxying & routing | ✅ Mature, battle-tested | ✅ Functional |
| Plugin ecosystem | ✅ 100+ plugins | ⚠️ Small but growing |
| L402 / Lightning payments | ❌ | ✅ Native |
| Macaroon-based auth | ❌ | ✅ Native |
| Fiat402 enterprise payments | ❌ | ✅ |
| Token delegation/governance | ❌ | ✅ |
| Economic controls (budgets, spend limits) | ❌ | ✅ |
| Multi-tenant isolation | ✅ Workspaces/CP | ✅ |
| RBAC | ✅ | ✅ |
| Audit logging | ✅ | ✅ |
| Scale / throughput | ✅ 50K+ TPS/node | ⚠️ Proving |
| Community / ecosystem | ✅ Massive | ⚠️ Early |
| Enterprise customers | ✅ Mercedes-Benz, UnitedHealth | ⚠️ Building |

### Zuplo

**What it is:** Programmable API gateway + developer portal, edge-deployed. Recently launched **AI Gateway** and **MCP Gateway** products.

**Why Zuplo matters:** They're the closest to SatGate's *positioning* in the agent economy, even though the *mechanism* is completely different.

- **AI Gateway:** Model routing, prompt injection detection, semantic caching, cost controls, team governance for LLM access
- **MCP Gateway:** Enterprise governance for MCP servers — virtual MCP servers, RBAC for tool access, centralized management
- **API Management:** Rate limiting, auth, dev portals, monetization

| Dimension | Zuplo | SatGate |
|-----------|-------|---------|
| API gateway | ✅ Edge-deployed | ✅ Self-hosted |
| AI/LLM governance | ✅ Model routing, caching | ⚠️ Via proxy, not native |
| MCP server governance | ✅ Native | ❌ Not yet |
| L402 / Lightning payments | ❌ | ✅ Native |
| Macaroon tokens | ❌ | ✅ Native |
| Economic controls | ⚠️ Cost controls (LLM budgets) | ✅ Full economic layer |
| Token delegation | ❌ | ✅ |
| Developer portal | ✅ | ❌ |
| Edge deployment | ✅ Global edge | ❌ Self-hosted |
| Enterprise customers | ✅ AccuWeather, Yext, Blockdaemon | ⚠️ Building |

**⚠️ Zuplo's MCP Gateway is a real competitive threat.** If the agent economy converges on MCP as the standard, Zuplo is already there with enterprise governance. SatGate should monitor this closely and consider MCP-native support.

### Unkey

**What it is:** Open-source API key management platform. 5.1K GitHub stars, very active development (4,599 commits). SaaS-hosted with self-host option.

**Features:**
- API key creation, verification, revocation
- Rate limiting per key
- Usage tracking and analytics
- Temporary keys, key metadata
- SDKs for major languages

| Dimension | Unkey | SatGate |
|-----------|-------|---------|
| API key management | ✅ Purpose-built | ✅ Via macaroons |
| Key verification speed | ✅ Optimized (edge) | ✅ Local verification (macaroons) |
| Key delegation/attenuation | ❌ Static keys | ✅ Macaroon caveats + delegation trees |
| Payment integration | ❌ | ✅ Native L402/Fiat402 |
| Rate limiting | ✅ | ✅ |
| Reverse proxy | ❌ (API only) | ✅ Full gateway |
| Self-hosted | ✅ (complex) | ✅ |
| Developer experience | ✅ Excellent | ⚠️ More complex |
| Budget/economic controls | ❌ | ✅ |

**Where Unkey wins:** If you just need API key management and nothing else, Unkey is simpler and more focused. Beautiful dashboard, great DX.

**Where SatGate wins:** Macaroons are strictly more powerful than static API keys — they support attenuation, delegation, and cryptographic verification without a database call. Plus SatGate bundles the gateway.

---

## Head-to-Head Comparisons

### Full Feature Matrix

| Feature | Aperture | Kong Ent. | Zuplo | Unkey | SatGate |
|---------|:--------:|:---------:|:-----:|:-----:|:-------:|
| **Reverse proxy** | ✅ | ✅ | ✅ | ❌ | ✅ |
| **L402 payments** | ✅ | ❌ | ❌ | ❌ | ✅ |
| **Fiat402 payments** | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Macaroon tokens** | ✅ | ❌ | ❌ | ❌ | ✅ |
| **Token delegation** | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Token governance (ban/revoke)** | ❌ | ❌ | ❌ | ✅ (keys) | ✅ |
| **Multi-tenant** | ❌ | ✅ | ✅ | ✅ | ✅ |
| **RBAC** | ❌ | ✅ | ✅ | ⚠️ | ✅ |
| **Audit logging** | ❌ | ✅ | ✅ | ⚠️ | ✅ |
| **Budget controls** | ❌ | ❌ | ✅ (AI) | ❌ | ✅ |
| **Rate limiting** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **AI/LLM governance** | ❌ | ⚠️ | ✅ | ❌ | ⚠️ |
| **MCP support** | ❌ | ❌ | ✅ | ❌ | ❌ |
| **Developer portal** | ❌ | ✅ | ✅ | ✅ | ❌ |
| **Admin dashboard** | ❌ | ✅ | ✅ | ✅ | ✅ |
| **Plugin ecosystem** | ❌ | ✅ | ✅ | ❌ | ⚠️ |
| **Edge deployment** | ❌ | ✅ | ✅ | ✅ | ❌ |
| **Self-hosted / OSS** | ✅ | ✅ (core) | ❌ | ✅ | ✅ |
| **Supply chain security** | ❌ | ✅ | ⚠️ | ⚠️ | ✅ |

---

## Where Competitors Are Stronger

This section exists because Wayne would call bullshit on a doc that pretends SatGate wins everywhere. Here's where we're genuinely behind:

### vs Kong

- **Maturity & trust.** Kong has been in production at Fortune 500s for a decade. We're new. Enterprise procurement teams care about this.
- **Throughput at scale.** NGINX-based, proven at 50K+ TPS per node. SatGate needs to publish benchmarks.
- **Plugin ecosystem.** 100+ plugins vs. our small set. Want OAuth2, LDAP, GraphQL, Kafka, OPA? Kong has it off the shelf.
- **Developer portal.** Kong has one. We don't. For API-as-a-product companies, this matters.
- **Analyst coverage.** Kong shows up in Gartner and Forrester. We don't exist in analyst reports yet.

### vs Zuplo

- **MCP Gateway.** Zuplo has a dedicated MCP gateway product for managing agent tool access. We have nothing here yet. This is a gap in our "agent economy" story.
- **AI-native features.** Prompt injection detection, semantic caching, model routing — Zuplo treats AI workloads as first-class. We proxy them but don't add AI-specific value.
- **Edge deployment.** Zuplo deploys to the edge globally in seconds. SatGate is self-hosted only. For latency-sensitive use cases, this matters.
- **Developer portal.** Again, we don't have one.

### vs Unkey

- **Developer experience.** Unkey's DX is polished — `npm install @unkey/api`, verify a key in 3 lines. SatGate is more complex to integrate.
- **Time to first API call.** Unkey: minutes. SatGate: need to deploy a gateway, configure routes, understand macaroons.
- **SaaS simplicity.** Unkey is hosted. SatGate requires infrastructure.

### vs Aperture

- **Simplicity.** Aperture is a single Go binary with zero dependencies beyond LND. SatGate requires Postgres, Redis, and understanding of its configuration model.
- **LNC support.** Aperture has Lightning Node Connect. We don't.
- **Proven at Lightning Labs.** Runs Lightning Loop in production. Known quantity in the Lightning ecosystem.

---

## Why "Economic Firewall" Is a Different Category

### The Problem with "API Gateway"

Calling SatGate an "API gateway" puts us in a comparison with Kong, Envoy, Traefik, NGINX, etc. We lose that comparison on maturity, scale, ecosystem, and market presence. Every time.

### The Problem with "L402 Proxy"

Calling SatGate an "L402 proxy" puts us in a comparison with Aperture. We win that comparison, but it limits our market to people who already understand Lightning payments — a tiny audience.

### The "Economic Firewall" Category

An **Economic Firewall** is infrastructure that enforces economic policy at the network boundary. It answers questions that neither API gateways nor payment proxies address:

| Question | API Gateway | L402 Proxy | Economic Firewall |
|----------|:-----------:|:----------:|:-----------------:|
| Who is this client? | ✅ | ✅ | ✅ |
| Are they rate-limited? | ✅ | ⚠️ | ✅ |
| Can they pay for this request? | ❌ | ✅ | ✅ |
| How much budget do they have left? | ❌ | ❌ | ✅ |
| Who delegated their access? | ❌ | ❌ | ✅ |
| Can I revoke an entire delegation tree? | ❌ | ❌ | ✅ |
| What's the total spend across all agents? | ❌ | ❌ | ✅ |
| Can I set per-department spending limits? | ❌ | ❌ | ✅ |
| Can I audit who spent what, when, where? | ⚠️ | ❌ | ✅ |

### Why This Matters for the Agent Economy

When an AI agent uses your API:
1. **It was delegated access** by a human or another agent — you need to track that chain
2. **It has a budget** — you need to enforce it in real-time
3. **It might create sub-agents** — you need to govern the delegation tree
4. **It might go rogue** — you need to ban it and everything it spawned
5. **Someone is paying** — you need to know who, how much, and for what

Traditional API gateways handle #1 (poorly). Aperture handles parts of #1. Only an Economic Firewall handles all five.

### Category Creation Rationale

- **Existing categories are saturated.** "API Gateway" has Kong, AWS API Gateway, Envoy, Traefik, NGINX, Azure APIM, etc. We can't outspend them on marketing.
- **The need is real.** Autonomous agents with economic authority are coming. No existing product governs them.
- **First-mover advantage.** If we define the category, we set the buying criteria. Competitors have to play on our field.
- **The "and" positioning.** SatGate is an API gateway AND an L402 proxy AND a budget controller AND a governance engine. But saying all that is confusing. "Economic Firewall" captures the essence: **it controls the flow of economic value across system boundaries.**

---

## Architectural Differences

### Aperture

```
┌─────────────────────────────────┐
│          APERTURE               │
├─────────────────────────────────┤
│  ┌──────────────┐               │
│  │   L402 Proxy │               │
│  │  (pay only)  │               │
│  └──────────────┘               │
│         │                       │
│  ┌──────┴──────┐                │
│  │  LND only   │                │
│  │  (coupled)  │                │
│  └─────────────┘                │
└─────────────────────────────────┘
```

### SatGate Gateway

```
┌─────────────────────────────────────────┐
│          SATGATE GATEWAY                │
│         "Economic Firewall"             │
├─────────────────────────────────────────┤
│  Protection │ Payments  │ Governance    │
│  (Capability)│(L402/Fiat)│(Audit/RBAC)  │
├─────────────────────────────────────────┤
│  Token Layer: Macaroons + Delegation    │
│  Budget Engine: Limits, Spend Tracking  │
├─────────────────────────────────────────┤
│  Pluggable Lightning Providers          │
│  Phoenixd│LND│CLN│Alby│LNbits│Strike   │
├─────────────────────────────────────────┤
│  Enterprise Infrastructure              │
│  Postgres│Redis│Prometheus│SCIM│Helm    │
└─────────────────────────────────────────┘
```

### Traditional API Gateway (Kong et al.)

```
┌─────────────────────────────────┐
│       TRADITIONAL GATEWAY       │
├─────────────────────────────────┤
│  Routing │ Auth │ Rate Limit    │
│  (proxy)  │(keys)│(throttle)    │
├─────────────────────────────────┤
│  Plugin Ecosystem               │
│  OAuth│LDAP│OPA│GraphQL│etc.    │
├─────────────────────────────────┤
│  NO economic layer              │
│  NO token delegation            │
│  NO payment integration         │
└─────────────────────────────────┘
```

---

## Payment Modes Explained

### Capability (Protection Only)

- No payment required
- Token-based authentication
- Scope-based authorization
- Use case: Internal APIs, free tier, agent auth

### L402 (Lightning Payments)

- Pay-per-request with Lightning
- Macaroon + preimage verification
- Use case: Public API monetization

### Chargeback (Enterprise FinOps)

- Metered usage, no real payment
- Internal cost allocation
- Budget enforcement
- Use case: Enterprise showback/chargeback

### Fiat402 (JWT Receipts)

- Prepaid receipts via external billing
- JWT token verification
- Use case: Enterprise invoicing, credits

---

## Migration from Aperture

SatGate maintains L402 wire compatibility — existing macaroon clients work unchanged.

**Aperture config:**
```yaml
services:
  - name: myapi
    price: 100
```

**SatGate equivalent:**
```yaml
routes:
  - name: myapi
    match:
      pathPrefix: /
    upstream: http://myapi:8080
    policy:
      kind: l402
      priceSats: 100
```

> **Wire Compatibility:** SatGate maintains L402 wire compatibility — existing macaroon clients work unchanged. You can migrate incrementally, route by route.

---

## When to Use What

| Use Case | Recommended | Why |
|----------|-------------|-----|
| Simple L402 demo / hobby project | **Aperture** | Simpler, fewer moving parts |
| Already deep in LND/Lightning Labs stack | **Aperture** | Natural fit, LNC support |
| Just need API key management | **Unkey** | Purpose-built, great DX |
| Traditional API gateway at scale | **Kong** | Mature, proven, ecosystem |
| AI/LLM governance + MCP | **Zuplo** | AI-native, MCP gateway |
| Enterprise API monetization | **SatGate** | Multi-mode payments, budget controls |
| Multi-tenant SaaS platform | **SatGate** | Tenant isolation, delegation |
| AI Agent authentication & governance | **SatGate** | Economic Firewall model |
| FinOps / chargeback tracking | **SatGate** | Native budget engine |
| Compliance requirements (SOC2, audit) | **SatGate** or **Kong** | Both have enterprise audit |
| Production deployment with SLAs | **Kong** (today) | Until SatGate proves scale |

---

## Strategic Gaps & Roadmap Implications

Things we should consider based on this analysis:

1. **MCP Gateway support.** Zuplo is already there. If MCP becomes the agent standard, we need native support.
2. **AI-specific features.** Prompt injection detection, semantic caching, model routing — these matter for AI API providers.
3. **Developer portal.** Kong, Zuplo, and Unkey all have one. We don't. For API-as-a-product use cases, this is table stakes.
4. **Published benchmarks.** We need throughput numbers. "Trust us" doesn't work against Kong's proven 50K+ TPS claims.
5. **LNC support.** Aperture has it. We should too, for Lightning Labs ecosystem compatibility.
6. **Edge deployment option.** Self-hosted is fine for enterprise, but a managed/edge option would expand TAM.

---

## Resources

- [SatGate documentation index](index.md)
- [SatGate GitHub Repository](https://github.com/SatGate-io/satgate)
- [Aperture (Lightning Labs)](https://github.com/lightninglabs/aperture)
- [Kong Gateway](https://konghq.com/products/kong-gateway)
- [Zuplo](https://zuplo.com)
- [Unkey](https://github.com/unkeyed/unkey)

---

<p align="center">
  <img src="logo_blue_transparent.png" alt="SatGate Logo" width="120">
</p>

<p align="center">
  <strong>SatGate™ Gateway</strong> — Economic Firewall for the Agent Economy ⚡
</p>

<p align="center">
  <em>"Protection by default. Payments optional."</em>
</p>

<p align="center">
  <sub>SatGate™ is a trademark of SatGate, Inc. All rights reserved.</sub>
</p>

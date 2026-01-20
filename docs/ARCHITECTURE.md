<p align="center">
  <img src="logo_white_transparent.png" alt="SatGate Logo" width="200">
</p>

# SatGate Gateway Architecture

> Technical architecture, deployment patterns, and system design
>
> **SatGate™ Gateway Documentation**

---

📚 **Documentation Navigation**
| [Overview](./OVERVIEW.md) | **Architecture** | [Capability Tokens](./CAPABILITY_TOKENS.md) | [Policy Modes](./POLICY_MODES.md) |
|:-------------------------:|:----------------:|:-------------------------------------------:|:--------------------------------:|
| *What & Why* | *How It's Built* | *Credential System* | *Protection Levels* |

---

## Executive Summary

SatGate Gateway is an **API Gateway** purpose-built for the agent economy. It sits between clients (humans, services, AI agents) and your upstream APIs, providing:

- **Authentication** via capability tokens (macaroons)
- **Authorization** via scope matching
- **Metering** for usage tracking and observability
- **Budget Enforcement** for cost control (Control mode)
- **Payment Integration** via Lightning Network L402 or Fiat402 (Charge mode)
- **Governance** for revocation, audit, and compliance

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              SatGate Gateway                                │
│                                                                             │
│   "EZ-Pass for the Agent Economy"                                           │
│   "Protection by default. Payments optional."                               │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## System Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Clients Layer                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐                │
│   │  Human   │   │  Service │   │ AI Agent │   │   CLI    │                │
│   │  User    │   │  (M2M)   │   │  (LLM)   │   │  Tool    │                │
│   └────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬─────┘                │
│        │              │              │              │                       │
│        └──────────────┴──────┬───────┴──────────────┘                       │
│                              │                                              │
│              Authorization: Bearer <capability_token>                       │
│                         (or L402 mac:preimage)                              │
│                              │                                              │
└──────────────────────────────┼──────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SatGate Gateway                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                         Request Pipeline                              │  │
│  ├───────────────────────────────────────────────────────────────────────┤  │
│  │                                                                       │  │
│  │   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐│  │
│  │   │  Rate   │──▶│ Request │──▶│  Auth   │──▶│ Policy  │──▶│ Metrics ││  │
│  │   │ Limiter │   │  Parse  │   │ Verify  │   │ Engine  │   │ Record  ││  │
│  │   └─────────┘   └─────────┘   └─────────┘   └─────────┘   └─────────┘│  │
│  │                                                                       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐   │
│  │   Macaroon    │ │   Budget      │ │   Payment     │ │  Governance   │   │
│  │   Service     │ │   Service     │ │   Service     │ │   Service     │   │
│  │               │ │               │ │               │ │               │   │
│  │ • Mint        │ │ • Allocate    │ │ • Lightning   │ │ • Ban/Unban   │   │
│  │ • Verify      │ │ • Check       │ │ • Stripe      │ │ • Audit Log   │   │
│  │ • Delegate    │ │ • Enforce     │ │ • Invoices    │ │ • Lineage     │   │
│  │ • Decode      │ │ • Alert       │ │ • Settlement  │ │ • Export      │   │
│  └───────────────┘ └───────────────┘ └───────────────┘ └───────────────┘   │
│                                                                             │
│  ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐   │
│  │   Metering    │ │   Config      │ │   Cloud       │ │   Mint        │   │
│  │   Service     │ │   Service     │ │   Service     │ │   Service     │   │
│  │               │ │               │ │               │ │               │   │
│  │ • Count       │ │ • Routes      │ │ • Tenants     │ │ • Identity    │   │
│  │ • Aggregate   │ │ • Upstreams   │ │ • Sessions    │ │ • Policy      │   │
│  │ • Export      │ │ • Reload      │ │ • Lifecycle   │ │ • Exchange    │   │
│  │ • Prometheus  │ │ • Validate    │ │ • WebSocket   │ │ • Rotate      │   │
│  └───────────────┘ └───────────────┘ └───────────────┘ └───────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Enterprise Services (Cloud Only)                  │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  ┌───────────────┐ ┌───────────────┐ ┌───────────────┐             │   │
│  │  │ Delegation v2 │ │  Settlement   │ │    Team       │             │   │
│  │  │   Service     │ │   Service     │ │   Service     │             │   │
│  │  │               │ │               │ │               │             │   │
│  │  │ • Token Tree  │ │ • Ledger      │ │ • Members     │             │   │
│  │  │ • Scope Check │ │ • Credits     │ │ • Invites     │             │   │
│  │  │ • Cascade Rev │ │ • L402 Bridge │ │ • Roles       │             │   │
│  │  │ • Budget Alloc│ │ • Idempotent  │ │ • RBAC        │             │   │
│  │  └───────────────┘ └───────────────┘ └───────────────┘             │   │
│  │                                                                     │   │
│  │  ┌───────────────┐ ┌───────────────┐ ┌───────────────┐             │   │
│  │  │   Support     │ │    Audit      │ │ Impersonation │             │   │
│  │  │   Service     │ │   Service     │ │   Service     │             │   │
│  │  │               │ │               │ │               │             │   │
│  │  │ • Tickets     │ │ • Events      │ │ • Session     │             │   │
│  │  │ • Messages    │ │ • Hash Chain  │ │ • Banner      │             │   │
│  │  │ • Categories  │ │ • Retention   │ │ • Cookie      │             │   │
│  │  │ • Priorities  │ │ • Filters     │ │ • Auto-Expire │             │   │
│  │  └───────────────┘ └───────────────┘ └───────────────┘             │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Upstreams Layer                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐                   │
│   │  REST API    │   │  GraphQL     │   │  gRPC        │                   │
│   │  Service     │   │  Service     │   │  Service     │                   │
│   └──────────────┘   └──────────────┘   └──────────────┘                   │
│                                                                             │
│   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐                   │
│   │  LLM API     │   │  Database    │   │  Internal    │                   │
│   │  (OpenAI)    │   │  Service     │   │  Microservice│                   │
│   └──────────────┘   └──────────────┘   └──────────────┘                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Request Flow

### Complete Request Lifecycle

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                         SatGate Request Pipeline                              │
├───────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  ┌──────────────────────────────────────────────────────────────────────────┐│
│  │ 1. REQUEST RECEIVED                                                      ││
│  │    ─────────────────                                                     ││
│  │    • TLS termination (if configured)                                     ││
│  │    • Request ID assigned (X-Request-ID)                                  ││
│  │    • Rate limit check (fail-closed if store unavailable)                 ││
│  └──────────────────────────────────────────────────────────────────────────┘│
│                                    │                                          │
│                                    ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────┐│
│  │ 2. ROUTE MATCHING                                                        ││
│  │    ──────────────                                                        ││
│  │    • Match request path against configured routes                        ││
│  │    • Select upstream and policy                                          ││
│  │    • If no route matches: 404 Not Found                                  ││
│  └──────────────────────────────────────────────────────────────────────────┘│
│                                    │                                          │
│                                    ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────┐│
│  │ 3. POLICY CHECK                                                          ││
│  │    ────────────                                                          ││
│  │                                                                          ││
│  │    ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐  ││
│  │    │   PUBLIC     │ │   OBSERVE    │ │   CONTROL    │ │   CHARGE     │  ││
│  │    │              │ │              │ │              │ │              │  ││
│  │    │ • No auth    │ │ • Token req  │ │ • Token req  │ │ • Token req  │  ││
│  │    │ • No meter   │ │ • Metering   │ │ • Metering   │ │ • Metering   │  ││
│  │    │ • Pass-thru  │ │ • No enforce │ │ • Budget chk │ │ • Payment chk│  ││
│  │    │              │ │              │ │ • 429 on fail│ │ • 402 on fail│  ││
│  │    └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘  ││
│  │                                                                          ││
│  └──────────────────────────────────────────────────────────────────────────┘│
│                                    │                                          │
│                                    ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────┐│
│  │ 4. TOKEN VALIDATION (if policy requires)                                 ││
│  │    ────────────────────────────────────                                  ││
│  │    a. Extract token from Authorization header                            ││
│  │       └── Bearer <token> | L402 <mac>:<preimage> | Receipt <jwt>         ││
│  │    b. Base64 decode → JSON parse                                         ││
│  │    c. Verify HMAC signature (HMAC-SHA256)                                ││
│  │    d. Check caveats (expires, scope, tenant_id, custom)                  ││
│  │    e. Check governance ban list                                          ││
│  │    f. If L402: verify preimage against payment_hash                      ││
│  │                                                                          ││
│  │    ❌ Invalid? → 401 Unauthorized                                         ││
│  └──────────────────────────────────────────────────────────────────────────┘│
│                                    │                                          │
│                                    ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────┐│
│  │ 5. BUDGET CHECK (Control mode only)                                      ││
│  │    ──────────────────────────────                                        ││
│  │    • Look up budget allocation for token/tenant                          ││
│  │    • Check: current_usage < budget_limit                                 ││
│  │    • Add X-Budget-Remaining header                                       ││
│  │                                                                          ││
│  │    ❌ Budget exceeded? → 429 Too Many Requests                            ││
│  │       └── Retry-After header included                                    ││
│  └──────────────────────────────────────────────────────────────────────────┘│
│                                    │                                          │
│                                    ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────┐│
│  │ 6. PAYMENT CHECK (Charge mode only)                                      ││
│  │    ────────────────────────────────                                      ││
│  │                                                                          ││
│  │    L402 (Lightning):                                                     ││
│  │    • Check if payment preimage provided                                  ││
│  │    • Verify SHA256(preimage) == payment_hash in macaroon                 ││
│  │    • Mark invoice as settled                                             ││
│  │                                                                          ││
│  │    Fiat402 (Stripe):                                                     ││
│  │    • Verify JWT receipt from Stripe                                      ││
│  │    • Check payment intent status                                         ││
│  │                                                                          ││
│  │    ❌ No payment? → 402 Payment Required                                  ││
│  │       └── WWW-Authenticate: L402 macaroon="...", invoice="..."           ││
│  └──────────────────────────────────────────────────────────────────────────┘│
│                                    │                                          │
│                                    ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────┐│
│  │ 7. METERING                                                              ││
│  │    ────────                                                              ││
│  │    • Record request metadata (timestamp, path, method, token)            ││
│  │    • Increment usage counters (tenant, token, route)                     ││
│  │    • Export to Prometheus metrics                                        ││
│  │    • Check for usage alerts/thresholds                                   ││
│  └──────────────────────────────────────────────────────────────────────────┘│
│                                    │                                          │
│                                    ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────┐│
│  │ 8. PROXY TO UPSTREAM                                                     ││
│  │    ─────────────────                                                     ││
│  │    • Forward request with original headers                               ││
│  │    • Add X-Forwarded-For, X-Request-ID                                   ││
│  │    • Add X-SatGate-Tenant-ID, X-SatGate-Token-Signature                  ││
│  │    • Configurable timeout, retries                                       ││
│  └──────────────────────────────────────────────────────────────────────────┘│
│                                    │                                          │
│                                    ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────┐│
│  │ 9. RESPONSE                                                              ││
│  │    ────────                                                              ││
│  │    • Pass upstream response to client                                    ││
│  │    • Record response status for metrics                                  ││
│  │    • Update latency histograms                                           ││
│  │    • Audit log (if enabled)                                              ││
│  └──────────────────────────────────────────────────────────────────────────┘│
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
```

---

## Component Architecture

### Core Services

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Gateway Core Services                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Macaroon Service                             │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Responsibilities:                                                  │   │
│  │  • Mint capability tokens with configurable scope, TTL              │   │
│  │  • Verify token signatures (HMAC-SHA256)                            │   │
│  │  • Decode and parse token structure                                 │   │
│  │  • Validate caveats (expires, scope, custom)                        │   │
│  │  • Support delegation (child token creation)                        │   │
│  │                                                                     │   │
│  │  Dependencies:                                                      │   │
│  │  • CAPABILITY_ROOT_KEY (environment variable)                       │   │
│  │  • Governance Service (ban list check)                              │   │
│  │                                                                     │   │
│  │  Key Functions:                                                     │   │
│  │  • Mint(scope, ttl) → Token                                         │   │
│  │  • Verify(token) → Claims, error                                    │   │
│  │  • Delegate(parentToken, caveats) → ChildToken                      │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Budget Service                               │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Responsibilities:                                                  │   │
│  │  • Manage budget allocations per tenant/token                       │   │
│  │  • Track usage against budgets                                      │   │
│  │  • Enforce limits (429 when exceeded)                               │   │
│  │  • Support budget delegation (child budgets)                        │   │
│  │  • Alert on threshold warnings (80%, 90%, 100%)                     │   │
│  │                                                                     │   │
│  │  Storage:                                                           │   │
│  │  • PostgreSQL (persistent allocation records)                       │   │
│  │  • Redis (real-time usage counters, optional)                       │   │
│  │                                                                     │   │
│  │  Key Functions:                                                     │   │
│  │  • AllocateBudget(tokenSig, limit, period) → Allocation             │   │
│  │  • CheckBudget(tokenSig) → Remaining, error                         │   │
│  │  • IncrementUsage(tokenSig, amount) → error                         │   │
│  │  • DelegateBudget(parentSig, childSig, amount) → error              │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Payment Service                               │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Responsibilities:                                                  │   │
│  │  • Generate Lightning invoices (L402)                               │   │
│  │  • Verify payment preimages                                         │   │
│  │  • Integrate with Stripe (Fiat402)                                  │   │
│  │  • Track payment status                                             │   │
│  │  • Support settlement and reconciliation                            │   │
│  │                                                                     │   │
│  │  Providers:                                                         │   │
│  │  • LND (Lightning Network Daemon)                                   │   │
│  │  • CLN (Core Lightning)                                             │   │
│  │  • LNBits (Lightning backend)                                       │   │
│  │  • Stripe (fiat payments)                                           │   │
│  │                                                                     │   │
│  │  Key Functions:                                                     │   │
│  │  • CreateInvoice(amountSats, memo) → Invoice                        │   │
│  │  • VerifyPreimage(paymentHash, preimage) → bool                     │   │
│  │  • CreateStripeIntent(amount, currency) → PaymentIntent             │   │
│  │  • CheckPaymentStatus(invoiceID) → Status                           │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                       Governance Service                             │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Responsibilities:                                                  │   │
│  │  • Maintain token ban list (instant revocation)                     │   │
│  │  • Track token lineage (parent → child relationships)               │   │
│  │  • Record all minted tokens                                         │   │
│  │  • Audit logging (immutable)                                        │   │
│  │  • Compliance exports (SOC2, audit)                                 │   │
│  │                                                                     │   │
│  │  Storage:                                                           │   │
│  │  • PostgreSQL (audit_log, token_governance tables)                  │   │
│  │  • Immutable tables (RLS + triggers prevent modification)           │   │
│  │                                                                     │   │
│  │  Key Functions:                                                     │   │
│  │  • Ban(signature, reason) → error                                   │   │
│  │  • Unban(signature) → error                                         │   │
│  │  • IsBanned(signature) → bool                                       │   │
│  │  • RecordMint(token, metadata) → error                              │   │
│  │  • GetLineage(signature) → []Token                                  │   │
│  │  • ExportAuditLog(startTime, endTime) → []Event                     │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Metering Service                              │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Responsibilities:                                                  │   │
│  │  • Record all request metadata                                      │   │
│  │  • Aggregate usage by tenant, token, route                          │   │
│  │  • Export metrics to Prometheus                                     │   │
│  │  • Support custom metric labels                                     │   │
│  │  • Real-time usage dashboards                                       │   │
│  │                                                                     │   │
│  │  Metrics Exposed:                                                   │   │
│  │  • satgate_requests_total (counter)                                 │   │
│  │  • satgate_request_duration_seconds (histogram)                     │   │
│  │  • satgate_active_connections (gauge)                               │   │
│  │  • satgate_budget_usage_ratio (gauge)                               │   │
│  │  • satgate_payment_revenue_sats (counter)                           │   │
│  │                                                                     │   │
│  │  Key Functions:                                                     │   │
│  │  • RecordRequest(req, resp, latency) → error                        │   │
│  │  • GetUsage(tenantID, period) → Usage                               │   │
│  │  • ExportPrometheus() → string                                      │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                          Mint Service                                │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Responsibilities:                                                  │   │
│  │  • Exchange platform credentials for capability tokens              │   │
│  │  • Verify identity providers (K8s, AWS IAM, OIDC)                   │   │
│  │  • Apply policy-as-code rules                                       │   │
│  │  • Support automatic token rotation                                 │   │
│  │  • Track issued tokens for governance                               │   │
│  │                                                                     │   │
│  │  Identity Providers:                                                │   │
│  │  • Kubernetes (ServiceAccount JWT)                                  │   │
│  │  • AWS IAM (instance/role identity)                                 │   │
│  │  • Azure AD (OIDC tokens)                                           │   │
│  │  • Okta (OIDC tokens)                                               │   │
│  │  • Custom OIDC providers                                            │   │
│  │                                                                     │   │
│  │  Key Functions:                                                     │   │
│  │  • Exchange(provider, credentials, scopes) → Token                  │   │
│  │  • Delegate(parentToken, caveats) → ChildToken                      │   │
│  │  • ListPolicies() → []Policy                                        │   │
│  │  • ListProviders() → []Provider                                     │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │              Delegation v2 Service (Enterprise)                     │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Responsibilities:                                                  │   │
│  │  • Manage hierarchical delegation token tree                        │   │
│  │  • Enforce scope subset validation (child ≤ parent)                 │   │
│  │  • Allocate budgets from parent to child (atomic transfer)          │   │
│  │  • Cascade revocation (revoke parent → revoke all descendants)      │   │
│  │  • Gateway-side budget enforcement (HTTP 402 hard stop)             │   │
│  │  • Alert generation at thresholds (80%, 90%, 100%)                  │   │
│  │                                                                     │   │
│  │  Storage:                                                           │   │
│  │  • PostgreSQL (delegation_tokens, spend_ledger tables)              │   │
│  │  • Redis (atomic budget counters via Lua scripts)                   │   │
│  │                                                                     │   │
│  │  Key Functions:                                                     │   │
│  │  • CreateToken(parent, scope, budget, expiry) → Token               │   │
│  │  • Spend(tokenID, requestID, cost) → SpendResult                    │   │
│  │  • Revoke(tokenID, cascade) → RevokeResult                          │   │
│  │  • GetTree(tenantID) → DelegationTree                               │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                Settlement Service (Enterprise)                      │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Responsibilities:                                                  │   │
│  │  • Add credits to delegation tokens (admin-only)                    │   │
│  │  • Maintain immutable settlement ledger                             │   │
│  │  • Ensure idempotency (no double-credit)                            │   │
│  │  • Bridge L402 payments to credits (X-Credit-Token)                 │   │
│  │  • Bridge Stripe payments to credits                                │   │
│  │                                                                     │   │
│  │  Storage:                                                           │   │
│  │  • PostgreSQL (token_settlement_ledger table)                       │   │
│  │  • Redis (budget increment via Lua scripts)                         │   │
│  │                                                                     │   │
│  │  Key Functions:                                                     │   │
│  │  • AddCredits(tokenID, amount, source, reference) → Settlement      │   │
│  │  • GetHistory(tokenID) → []Settlement                               │   │
│  │  • ProcessL402Payment(paymentHash, tokenID) → SettleResult          │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                   Team Service (Enterprise)                         │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Responsibilities:                                                  │   │
│  │  • Manage multi-user accounts (organizations)                       │   │
│  │  • Role-based access control (Owner/Admin/Member/Viewer)            │   │
│  │  • Team invitations via email                                       │   │
│  │  • Role changes and member removal                                  │   │
│  │                                                                     │   │
│  │  Storage:                                                           │   │
│  │  • PostgreSQL (cloud_team_members, cloud_team_invites tables)       │   │
│  │                                                                     │   │
│  │  Key Functions:                                                     │   │
│  │  • ListMembers(tenantID) → []Member                                 │   │
│  │  • CreateInvite(tenantID, email, role) → Invite                     │   │
│  │  • AcceptInvite(token) → Member                                     │   │
│  │  • ChangeRole(tenantID, userID, role) → Member                      │   │
│  │  • RemoveMember(tenantID, userID) → error                           │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Policy Modes

SatGate supports four policy modes that determine how requests are processed:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Policy Mode Spectrum                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                                                                     │   │
│   │   PUBLIC        OBSERVE         CONTROL         CHARGE              │   │
│   │   (Open)     (Authentication) (Budget Limit)  (Payment Gate)        │   │
│   │                                                                     │   │
│   │      ●────────────●────────────●────────────●                       │   │
│   │                                                                     │   │
│   │   No Auth     Token Required  Token + Budget  Token + Payment       │   │
│   │   No Meter    Metering Only   Enforce Limits  Pay-per-Use           │   │
│   │                                                                     │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│   "Protection by default. Payments optional."                               │
│                                                                             │
│   Start with OBSERVE for visibility, graduate to CONTROL for budgets,       │
│   add CHARGE when you're ready to monetize.                                 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Mode Comparison

| Aspect | Public | Observe | Control | Charge |
|--------|--------|---------|---------|--------|
| **Token Required** | ❌ | ✅ | ✅ | ✅ |
| **Metering** | ❌ | ✅ | ✅ | ✅ |
| **Budget Enforcement** | ❌ | ❌ | ✅ | ❌ |
| **Payment Required** | ❌ | ❌ | ❌ | ✅ |
| **Fail Response** | N/A | 401 | 401 / 429 | 401 / 402 |
| **Use Case** | Public APIs | Visibility | Cost Control | Monetization |

### Configuration Example

```yaml
routes:
  - name: public-health
    match:
      pathPrefix: /health
    upstream: http://api:8080
    policy:
      kind: public           # No authentication

  - name: api-observe
    match:
      pathPrefix: /api/v1/analytics
    upstream: http://analytics:8080
    policy:
      kind: observe          # Token required, metering only
      scope: api:analytics:read

  - name: api-control
    match:
      pathPrefix: /api/v1/compute
    upstream: http://compute:8080
    policy:
      kind: control          # Token + budget enforcement
      scope: api:compute
      budget:
        default: 10000       # Default budget per token
        period: daily

  - name: api-charge
    match:
      pathPrefix: /api/v1/premium
    upstream: http://premium:8080
    policy:
      kind: charge           # Payment required
      scope: api:premium
      price:
        sats: 100            # Satoshis per request
        currency: BTC
```

---

## Deployment Patterns

### Single Instance (Development/Testing)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      Single Instance Deployment                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                        ┌───────────────────┐                                │
│                        │   Load Balancer   │                                │
│                        │   (optional)      │                                │
│                        └─────────┬─────────┘                                │
│                                  │                                          │
│                                  ▼                                          │
│                        ┌───────────────────┐                                │
│                        │  SatGate Gateway  │                                │
│                        │    (single)       │                                │
│                        │                   │                                │
│                        │  • All services   │                                │
│                        │  • In-memory state│                                │
│                        │  • SQLite/Postgres│                                │
│                        └─────────┬─────────┘                                │
│                                  │                                          │
│                                  ▼                                          │
│                        ┌───────────────────┐                                │
│                        │   Upstream APIs   │                                │
│                        └───────────────────┘                                │
│                                                                             │
│   Suitable for:                                                             │
│   • Development environments                                                │
│   • Testing and staging                                                     │
│   • Low-traffic production (<1000 RPS)                                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### High Availability (Production)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                   High Availability Deployment                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                        ┌───────────────────┐                                │
│                        │   Load Balancer   │                                │
│                        │   (L7, health)    │                                │
│                        └─────────┬─────────┘                                │
│                                  │                                          │
│              ┌───────────────────┼───────────────────┐                      │
│              │                   │                   │                      │
│              ▼                   ▼                   ▼                      │
│     ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐            │
│     │ SatGate Gateway │ │ SatGate Gateway │ │ SatGate Gateway │            │
│     │   Instance 1    │ │   Instance 2    │ │   Instance 3    │            │
│     │                 │ │                 │ │    (N+1)        │            │
│     └────────┬────────┘ └────────┬────────┘ └────────┬────────┘            │
│              │                   │                   │                      │
│              └───────────────────┼───────────────────┘                      │
│                                  │                                          │
│              ┌───────────────────┼───────────────────┐                      │
│              │                   │                   │                      │
│              ▼                   ▼                   ▼                      │
│     ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐            │
│     │   PostgreSQL    │ │     Redis       │ │  Lightning Node │            │
│     │   (Primary/     │ │   (Cluster)     │ │  (LND/CLN)      │            │
│     │    Replica)     │ │                 │ │                 │            │
│     └─────────────────┘ └─────────────────┘ └─────────────────┘            │
│                                                                             │
│   Features:                                                                 │
│   • Horizontal scaling (add instances as needed)                            │
│   • Session affinity not required (stateless auth)                          │
│   • Shared state via PostgreSQL                                             │
│   • Real-time counters via Redis                                            │
│   • Leader election for singleton jobs                                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### SatGate Cloud (SaaS) + Hybrid Gateway

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SatGate Cloud + Hybrid Deployment                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                        SatGate Cloud (SaaS)                           │  │
│  │                    https://cloud.satgate.io                           │  │
│  │                                                                       │  │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐     │  │
│  │  │   Control   │ │   Config    │ │  Governance │ │   Billing   │     │  │
│  │  │   Plane     │ │   Store     │ │    Store    │ │   Service   │     │  │
│  │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘     │  │
│  │                                                                       │  │
│  │                         WebSocket Control                             │  │
│  │                    wss://cloud.satgate.io/gateway/v1                  │  │
│  │                                                                       │  │
│  └───────────────────────────────────┬───────────────────────────────────┘  │
│                                      │                                      │
│                                      │ Config push, Telemetry               │
│                                      │                                      │
│  ┌───────────────────────────────────┼───────────────────────────────────┐  │
│  │                     Customer VPC / Data Center                        │  │
│  │                                                                       │  │
│  │  ┌───────────────────────────────────────────────────────────────┐   │  │
│  │  │                    Hybrid Gateway                              │   │  │
│  │  │                                                                │   │  │
│  │  │   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐        │   │  │
│  │  │   │   Request   │   │   Policy    │   │   Local     │        │   │  │
│  │  │   │   Pipeline  │   │   Engine    │   │   Cache     │        │   │  │
│  │  │   └─────────────┘   └─────────────┘   └─────────────┘        │   │  │
│  │  │                                                                │   │  │
│  │  │   • Processes all API traffic locally (data never leaves VPC) │   │  │
│  │  │   • Syncs config from control plane                            │   │  │
│  │  │   • Reports telemetry (aggregated, no PII)                     │   │  │
│  │  │   • Operates independently during network partitions           │   │  │
│  │  │                                                                │   │  │
│  │  └───────────────────────────────────────────────────────────────┘   │  │
│  │                                      │                                │  │
│  │                                      ▼                                │  │
│  │  ┌───────────────────────────────────────────────────────────────┐   │  │
│  │  │              Customer Internal APIs (Upstreams)                │   │  │
│  │  │                                                                │   │  │
│  │  │   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   │   │  │
│  │  │   │  API 1   │   │  API 2   │   │  API 3   │   │  API N   │   │   │  │
│  │  │   └──────────┘   └──────────┘   └──────────┘   └──────────┘   │   │  │
│  │  │                                                                │   │  │
│  │  └───────────────────────────────────────────────────────────────┘   │  │
│  │                                                                       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   Benefits:                                                                 │
│   • Data sovereignty: API traffic stays in customer VPC                     │
│   • Low latency: Local processing, no round-trip to SaaS                    │
│   • Control plane benefits: Central config, governance, billing             │
│   • Offline resilience: Gateway operates with cached config                 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Data Flow

### Authentication Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Authentication Data Flow                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Client                    Gateway                         Upstream         │
│    │                          │                               │             │
│    │  POST /api/resource      │                               │             │
│    │  Authorization: Bearer   │                               │             │
│    │  <capability_token>      │                               │             │
│    │─────────────────────────▶│                               │             │
│    │                          │                               │             │
│    │                  ┌───────┴───────┐                       │             │
│    │                  │ Extract token │                       │             │
│    │                  │ from header   │                       │             │
│    │                  └───────┬───────┘                       │             │
│    │                          │                               │             │
│    │                  ┌───────┴───────┐                       │             │
│    │                  │ Base64 decode │                       │             │
│    │                  │ JSON parse    │                       │             │
│    │                  └───────┬───────┘                       │             │
│    │                          │                               │             │
│    │                  ┌───────┴───────┐                       │             │
│    │                  │ Verify HMAC   │                       │             │
│    │                  │ signature     │                       │             │
│    │                  └───────┬───────┘                       │             │
│    │                          │                               │             │
│    │                  ┌───────┴───────┐                       │             │
│    │                  │ Check caveats │                       │             │
│    │                  │ • expires     │                       │             │
│    │                  │ • scope       │                       │             │
│    │                  │ • custom      │                       │             │
│    │                  └───────┬───────┘                       │             │
│    │                          │                               │             │
│    │                  ┌───────┴───────┐                       │             │
│    │                  │ Check ban list│                       │             │
│    │                  └───────┬───────┘                       │             │
│    │                          │                               │             │
│    │                          │ If valid:                     │             │
│    │                          │ POST /api/resource            │             │
│    │                          │ X-SatGate-Tenant-ID: xxx      │             │
│    │                          │ X-SatGate-Token-Signature:xxx │             │
│    │                          │──────────────────────────────▶│             │
│    │                          │                               │             │
│    │                          │◀──────────────────────────────│             │
│    │                          │         200 OK                │             │
│    │                          │                               │             │
│    │◀─────────────────────────│                               │             │
│    │         200 OK           │                               │             │
│    │                          │                               │             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### L402 Payment Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          L402 Payment Data Flow                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Client              Gateway              LN Node           Upstream        │
│    │                    │                    │                 │            │
│    │ GET /api/premium   │                    │                 │            │
│    │ (no payment)       │                    │                 │            │
│    │───────────────────▶│                    │                 │            │
│    │                    │                    │                 │            │
│    │                    │ Create invoice     │                 │            │
│    │                    │───────────────────▶│                 │            │
│    │                    │                    │                 │            │
│    │                    │◀───────────────────│                 │            │
│    │                    │ invoice, hash      │                 │            │
│    │                    │                    │                 │            │
│    │◀───────────────────│                    │                 │            │
│    │ 402 Payment Req    │                    │                 │            │
│    │ WWW-Authenticate:  │                    │                 │            │
│    │   L402 macaroon=...,│                   │                 │            │
│    │   invoice=...       │                   │                 │            │
│    │                    │                    │                 │            │
│    │                    │                    │                 │            │
│    │ (Client pays invoice via Lightning wallet)                │            │
│    │                    │                    │                 │            │
│    │                    │                    │                 │            │
│    │ GET /api/premium   │                    │                 │            │
│    │ Authorization:     │                    │                 │            │
│    │   L402 <mac>:<preimage>                 │                 │            │
│    │───────────────────▶│                    │                 │            │
│    │                    │                    │                 │            │
│    │                    │ Verify:            │                 │            │
│    │                    │ • Macaroon sig     │                 │            │
│    │                    │ • SHA256(preimage) │                 │            │
│    │                    │   == payment_hash  │                 │            │
│    │                    │                    │                 │            │
│    │                    │ Proxy request      │                 │            │
│    │                    │─────────────────────────────────────▶│            │
│    │                    │                    │                 │            │
│    │                    │◀─────────────────────────────────────│            │
│    │                    │                    │       200 OK    │            │
│    │                    │                    │                 │            │
│    │◀───────────────────│                    │                 │            │
│    │        200 OK      │                    │                 │            │
│    │                    │                    │                 │            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Storage Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Storage Architecture                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                         PostgreSQL                                    │  │
│  ├───────────────────────────────────────────────────────────────────────┤  │
│  │                                                                       │  │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐       │  │
│  │  │  audit_log      │  │ token_governance│  │  budget_alloc   │       │  │
│  │  │  (immutable)    │  │  (ban list)     │  │  (allocations)  │       │  │
│  │  │                 │  │                 │  │                 │       │  │
│  │  │ • event_type    │  │ • signature     │  │ • token_sig     │       │  │
│  │  │ • tenant_id     │  │ • banned_at     │  │ • limit         │       │  │
│  │  │ • actor         │  │ • reason        │  │ • period        │       │  │
│  │  │ • resource      │  │ • parent_sig    │  │ • used          │       │  │
│  │  │ • metadata      │  │ • lineage       │  │ • reset_at      │       │  │
│  │  │ • timestamp     │  │                 │  │                 │       │  │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘       │  │
│  │                                                                       │  │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐       │  │
│  │  │ cloud_tenants   │  │ cloud_sessions  │  │ metering_events │       │  │
│  │  │ (multi-tenant)  │  │  (auth)         │  │ (usage data)    │       │  │
│  │  │                 │  │                 │  │                 │       │  │
│  │  │ • id            │  │ • session_id    │  │ • tenant_id     │       │  │
│  │  │ • email         │  │ • tenant_id     │  │ • token_sig     │       │  │
│  │  │ • plan          │  │ • expires_at    │  │ • route         │       │  │
│  │  │ • status        │  │ • created_at    │  │ • request_count │       │  │
│  │  │ • config        │  │                 │  │ • timestamp     │       │  │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘       │  │
│  │                                                                       │  │
│  │  Security Features:                                                   │  │
│  │  • Row Level Security (RLS) for tenant isolation                      │  │
│  │  • Immutable audit_log (INSERT only, triggers prevent UPDATE/DELETE)  │  │
│  │  • Encrypted connections (TLS required)                               │  │
│  │  • Prepared statements (SQL injection prevention)                     │  │
│  │                                                                       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                           Redis (Optional)                            │  │
│  ├───────────────────────────────────────────────────────────────────────┤  │
│  │                                                                       │  │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐       │  │
│  │  │  rate_limits    │  │  budget_counters│  │  session_cache  │       │  │
│  │  │                 │  │                 │  │                 │       │  │
│  │  │ Key:            │  │ Key:            │  │ Key:            │       │  │
│  │  │ rate:{ip}:{min} │  │ budget:{sig}    │  │ session:{id}    │       │  │
│  │  │                 │  │                 │  │                 │       │  │
│  │  │ Value: count    │  │ Value: used     │  │ Value: claims   │       │  │
│  │  │ TTL: 60s        │  │ TTL: period     │  │ TTL: 1h         │       │  │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘       │  │
│  │                                                                       │  │
│  │  Use Cases:                                                           │  │
│  │  • Rate limiting (high-frequency counters)                            │  │
│  │  • Budget enforcement (real-time usage)                               │  │
│  │  • Session caching (reduce DB load)                                   │  │
│  │  • Distributed locking (leader election)                              │  │
│  │                                                                       │  │
│  │  Fail Behavior:                                                       │  │
│  │  • Rate limiting: Fail CLOSED (503 if Redis unavailable)              │  │
│  │  • Session cache: Fall back to PostgreSQL                             │  │
│  │                                                                       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Security Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Security Architecture                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                       Network Security                                │  │
│  │                                                                       │  │
│  │  • TLS 1.3 for all external connections                               │  │
│  │  • mTLS optional for hybrid gateways                                  │  │
│  │  • IP allowlist for admin endpoints                                   │  │
│  │  • Rate limiting at edge (fail-closed)                                │  │
│  │                                                                       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                      Authentication Security                          │  │
│  │                                                                       │  │
│  │  • HMAC-SHA256 token signatures (256-bit key)                         │  │
│  │  • Time-based expiration (caveats)                                    │  │
│  │  • Instant revocation (governance ban list)                           │  │
│  │  • Scope-based authorization                                          │  │
│  │  • No token storage (stateless validation)                            │  │
│  │                                                                       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                       Data Security                                   │  │
│  │                                                                       │  │
│  │  • Secrets in environment variables (not config files)                │  │
│  │  • Database encryption at rest                                        │  │
│  │  • Audit log immutability (RLS + triggers)                            │  │
│  │  • PII minimization in logs                                           │  │
│  │  • Request ID correlation (not tokens)                                │  │
│  │                                                                       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    Response Headers                                   │  │
│  │                                                                       │  │
│  │  Content-Security-Policy: default-src 'self'; ...                     │  │
│  │  X-Content-Type-Options: nosniff                                      │  │
│  │  X-Frame-Options: DENY                                                │  │
│  │  Referrer-Policy: strict-origin-when-cross-origin                     │  │
│  │  Permissions-Policy: geolocation=(), microphone=(), camera=()         │  │
│  │  Strict-Transport-Security: max-age=31536000; includeSubDomains       │  │
│  │                                                                       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Observability

### Metrics (Prometheus)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Prometheus Metrics                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  # Request metrics                                                          │
│  satgate_requests_total{tenant, route, status, policy_mode}                 │
│  satgate_request_duration_seconds{tenant, route, quantile}                  │
│  satgate_request_size_bytes{tenant, route}                                  │
│  satgate_response_size_bytes{tenant, route}                                 │
│                                                                             │
│  # Auth metrics                                                             │
│  satgate_auth_attempts_total{tenant, result}  # success, invalid, banned    │
│  satgate_tokens_minted_total{tenant, scope}                                 │
│  satgate_tokens_revoked_total{tenant}                                       │
│                                                                             │
│  # Budget metrics                                                           │
│  satgate_budget_usage_ratio{tenant, token_sig}  # 0.0 - 1.0                 │
│  satgate_budget_exceeded_total{tenant}                                      │
│  satgate_budget_remaining{tenant, token_sig}                                │
│                                                                             │
│  # Payment metrics                                                          │
│  satgate_payment_revenue_sats_total{tenant, route}                          │
│  satgate_payment_attempts_total{tenant, status}  # success, failed          │
│  satgate_invoices_created_total{tenant}                                     │
│                                                                             │
│  # System metrics                                                           │
│  satgate_upstreams_health{upstream, status}                                 │
│  satgate_active_connections                                                 │
│  satgate_goroutines                                                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Logging

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Structured Logging                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  {                                                                          │
│    "level": "info",                                                         │
│    "time": "2024-01-15T10:30:00Z",                                          │
│    "request_id": "abc123",                                                  │
│    "tenant_id": "tenant-456",                                               │
│    "method": "GET",                                                         │
│    "path": "/api/v1/resource",                                              │
│    "status": 200,                                                           │
│    "latency_ms": 45,                                                        │
│    "policy_mode": "control",                                                │
│    "token_signature": "a1b2c3...",  // truncated                            │
│    "budget_remaining": 9500,                                                │
│    "upstream": "backend-api"                                                │
│  }                                                                          │
│                                                                             │
│  Log Levels:                                                                │
│  • debug: Detailed request tracing (dev only)                               │
│  • info: Normal operations, request logs                                    │
│  • warn: Recoverable issues, budget warnings                                │
│  • error: Failures, upstream errors                                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Configuration Reference

### Complete gateway.yaml Example

```yaml
# SatGate Gateway Configuration
# Version: 1.0

server:
  host: 0.0.0.0
  port: 8080
  metricsPort: 9090
  readTimeout: 30s
  writeTimeout: 30s
  shutdownTimeout: 10s

admin:
  token: ${ADMIN_TOKEN}
  capabilityRootKey: ${CAPABILITY_ROOT_KEY}
  ipAllowlist: "10.0.0.0/8,192.168.0.0/16"  # Optional

database:
  postgres:
    url: ${DATABASE_URL}
    maxConnections: 25
    autoMigrate: true

redis:
  url: ${REDIS_URL}  # Optional, enables real-time counters

lightning:
  backend: lnd  # lnd, cln, lnbits
  lnd:
    rpcHost: ${LND_HOST}
    macaroonPath: ${LND_MACAROON_PATH}
    tlsCertPath: ${LND_TLS_CERT_PATH}

stripe:  # Optional, for Fiat402
  secretKey: ${STRIPE_SECRET_KEY}
  webhookSecret: ${STRIPE_WEBHOOK_SECRET}

upstreams:
  - name: backend-api
    url: http://api:8080
    timeout: 30s
    retries: 3
    healthCheck:
      path: /health
      interval: 10s

routes:
  - name: public-health
    match:
      pathPrefix: /health
    upstream: backend-api
    policy:
      kind: public

  - name: api-observe
    match:
      pathPrefix: /api/v1/read
    upstream: backend-api
    policy:
      kind: observe
      scope: api:read

  - name: api-control
    match:
      pathPrefix: /api/v1/compute
    upstream: backend-api
    policy:
      kind: control
      scope: api:compute
      budget:
        default: 10000
        period: daily

  - name: api-charge
    match:
      pathPrefix: /api/v1/premium
    upstream: backend-api
    policy:
      kind: charge
      scope: api:premium
      price:
        sats: 100
        currency: BTC

logging:
  level: info
  format: json

metrics:
  enabled: true
  prefix: satgate
```

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `ADMIN_TOKEN` | Admin API authentication token | Yes |
| `CAPABILITY_ROOT_KEY` | Root key for token signing (32+ bytes hex) | Yes |
| `DATABASE_URL` | PostgreSQL connection string | Yes |
| `REDIS_URL` | Redis connection string | No |
| `LND_HOST` | Lightning node RPC host | For Charge mode |
| `LND_MACAROON_PATH` | Path to LND macaroon | For Charge mode |
| `STRIPE_SECRET_KEY` | Stripe API key | For Fiat402 |

---

## API Reference

### Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/tokens` | Mint a new capability token |
| `GET` | `/api/v1/tokens` | List minted tokens |
| `DELETE` | `/api/v1/tokens/{sig}` | Revoke a token |
| `POST` | `/api/v1/tokens/{sig}/delegate` | Delegate (attenuate) a token |
| `POST` | `/api/v1/governance/ban` | Ban a token |
| `POST` | `/api/v1/governance/unban` | Unban a token |
| `GET` | `/api/v1/governance/graph` | Get token lineage graph |
| `GET` | `/api/v1/metrics` | Prometheus metrics |
| `GET` | `/api/v1/health` | Health check |

### Mint Endpoints (when enabled)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/mint` | Exchange credentials for token |
| `POST` | `/v1/mint/delegate` | Delegate with additional caveats |
| `GET` | `/v1/mint/policies` | List configured policies |
| `GET` | `/v1/mint/providers` | List identity providers |

---

## Resources

- [SatGate Gateway Documentation](https://satgate.io/docs)
- [GitHub Repository](https://github.com/SatGate-io/satgate-gateway)
- [Capability Tokens](./CAPABILITY_TOKENS.md)
- [Policy Modes](./POLICY_MODES.md)
- [Self-Host Guide](./SELFHOST.md)

---

<p align="center">
  <img src="logo_white_transparent.png" alt="SatGate Logo" width="120">
</p>

<p align="center">
  <strong>SatGate™ Gateway</strong> — EZ-Pass for the Agent Economy ⚡
</p>

<p align="center">
  <em>"Protection by default. Payments optional."</em>
</p>

<p align="center">
  <sub>SatGate™ is a trademark of SatGate, Inc. All rights reserved.</sub>
</p>

# SatGate Quick Start Guide

**The Economic Firewall for your APIs** — Protect by default. Choose your economic policy.

---

## 🚀 5-Minute Setup

### 1. Sign Up & Get Your Gateway

```bash
# Sign up at https://satgate.io
# You'll get a gateway URL like: https://your-tenant.satgate.io
```

### 2. Add Your First Route

Create a file `satgate.yaml`:

```yaml
upstreams:
  - name: my-api
    url: https://api.example.com

routes:
  # Observe policy: Meter everything, allow all authenticated requests
  - name: api-observe
    match:
      pathPrefix: /api/
    upstream: my-api
    policy:
      kind: observe  # Aliases: chargeback, audit

  # Health check: Public (explicit opt-out from protection)
  - name: health
    match:
      pathPrefix: /healthz
    upstream: my-api
    policy:
      kind: public
```

### 3. Apply Configuration

```bash
# Via CLI
satgate apply -f satgate.yaml

# Or via Dashboard
# Go to https://dashboard.satgate.io → Gateway Configuration → Upload YAML
```

### 4. Mint a Token

```bash
# Via CLI
satgate token mint --scope /api/ --expires 30d

# Output:
# Token: sat_abc123...
# Scope: /api/*
# Expires: 2026-02-10
```

### 5. Make Your First Request

```bash
curl -H "Authorization: Bearer sat_abc123..." \
  https://your-tenant.satgate.io/api/users
```

🎉 **That's it!** Your API is now protected with Default Protection (cryptographic verification).

---

## 📊 Understanding the Economic Firewall

### Layer 0: Default Protection (Always On)

Every non-PUBLIC route requires a valid credential:
- **Macaroon-based capability tokens** with caveats
- **Delegation support** for agent swarms
- **Revocation** without redeploying keys

### Layer 1: Economic Policies (Your Choice)

| Policy | Description | Use Case |
|--------|-------------|----------|
| **Observe** | Verify → Allow → Meter | FinOps visibility, audit logging |
| **Control** | Verify → Check budget → Allow | Internal chargebacks, cost control |
| **Charge** | Verify → Require payment → Allow | API monetization |

---

## 🛠️ Common Patterns

### Pattern 1: Start with Observe (The Trojan Horse)

Most teams start here. Get full visibility with zero friction:

```yaml
routes:
  - name: all-traffic
    match:
      pathPrefix: /
    upstream: my-api
    policy:
      kind: observe
```

**What you get:**
- ✅ All requests cryptographically verified
- ✅ Full audit trail with tamper-evident logging
- ✅ Usage metrics per route
- ✅ No blocking, no payment required

### Pattern 2: Add Budget Controls (Control Policy)

When you need to limit LLM/AI agent costs:

```yaml
routes:
  - name: expensive-ai-endpoint
    match:
      pathPrefix: /api/ai/
    upstream: ai-backend
    policy:
      kind: control
      fiat402:
        budgetPerHour: 100  # $1.00 per hour
        budgetPerDay: 1000  # $10.00 per day
```

**What you get:**
- ✅ Default Protection + metering
- ✅ Budget enforcement before request proceeds
- ✅ Automatic 402 challenge when budget exceeded

### Pattern 3: Monetize with L402 (Charge Policy)

Turn your API into a revenue stream:

```yaml
routes:
  - name: premium-api
    match:
      pathPrefix: /api/premium/
    upstream: premium-backend
    policy:
      kind: charge
      l402:
        priceSats: 10  # 10 sats per request
```

**What you get:**
- ✅ Default Protection + metering
- ✅ L402 payment challenge for new users
- ✅ Instant settlement via Lightning Network
- ✅ Revenue tracking in dashboard

---

## 🔐 Token Management

### Minting Tokens

```bash
# Basic token
satgate token mint --scope /api/

# With time limit
satgate token mint --scope /api/ --expires 24h

# With request limit
satgate token mint --scope /api/ --max-uses 1000

# Delegatable (for agent swarms)
satgate token mint --scope /api/ --delegatable
```

### Token Caveats

Tokens can include caveats (restrictions):

| Caveat | Example | Description |
|--------|---------|-------------|
| `scope` | `/api/users/*` | Limits to specific paths |
| `expires` | `2026-12-31` | Expiration date |
| `maxUses` | `1000` | Request count limit |
| `rateLimit` | `100/min` | Rate limiting |
| `sourceIP` | `192.168.0.0/24` | IP restriction |

### Delegation

Let your AI agents create their own restricted tokens:

```bash
# Master token with delegation rights
satgate token mint --scope /api/ --delegatable --max-delegation-depth 2

# Agent creates child token (via SDK)
child_token = master_token.delegate(
    scope="/api/users/",
    expires="1h",
    max_uses=100
)
```

---

## 📈 Monitoring & Analytics

### Dashboard Views

1. **Usage Analytics** (`/cloud/usage`)
   - Observe / Control / Charge breakdown
   - Billable vs free request counts
   - Daily usage charts

2. **Route Statistics** (`/cloud/routes`)
   - Per-route request counts
   - Error rates and latency
   - Revenue by route

3. **Invoices** (`/cloud/invoices`)
   - Monthly billing with usage breakdown
   - Expandable details per invoice

### API Access

```bash
# Get usage summary
curl -H "Authorization: Bearer $TOKEN" \
  https://api.satgate.io/v1/cloud/billing/usage

# Get route statistics
curl -H "Authorization: Bearer $TOKEN" \
  https://api.satgate.io/v1/cloud/billing/usage/routes
```

---

## 🏗️ Deployment Models

### SaaS (Fastest Start)

- **Best for:** Public APIs, MVPs, small teams
- **Setup:** Point DNS to SatGate
- **Data flow:** Through SatGate cloud

```yaml
# No infrastructure needed - just configure routes
```

### Hybrid (Enterprise Default)

- **Best for:** Enterprise APIs, sensitive data
- **Setup:** Gateway in your VPC, control plane in SatGate Cloud
- **Data flow:** Stays in your network

```bash
# Deploy gateway in your VPC
helm install satgate-gateway satgate/gateway \
  --set controlPlane.url=https://api.satgate.io \
  --set controlPlane.apiKey=$SATGATE_API_KEY
```

### Self-Host (Air-Gapped)

- **Best for:** Regulated industries, maximum control
- **Setup:** Full stack in your infrastructure

```bash
# Deploy everything
helm install satgate satgate/full-stack \
  --set enterprise.license=$LICENSE_KEY
```

---

## 🔧 Troubleshooting

### "401 Unauthorized"

1. Check token is valid: `satgate token inspect $TOKEN`
2. Verify scope matches route: Token scope must be prefix of request path
3. Check expiration: Token may be expired

### "402 Payment Required"

1. For Control policy: Budget exceeded - wait for reset or increase budget
2. For Charge policy: Payment required - pay L402 invoice or use prepaid token

### "403 Forbidden"

1. Token revoked: Check `satgate token status $TOKEN`
2. Caveat failed: Check IP restrictions, time limits, etc.

---

## 📚 Next Steps

1. **Read the [Security Model](./SECURITY-MODEL.md)** — Understand Default Protection
2. **Explore [Multi-Backend Setup](./MULTI-BACKEND.md)** — Route to multiple upstreams
3. **Configure [WORM Audit Logs](./gateway/DEPLOY_SELF_HOSTED.md)** — Compliance storage
4. **Set up [KMS Integration](./ARCHITECTURE.md)** — Hardware-backed keys

---

## 💬 Support

- **Documentation:** https://docs.satgate.io
- **Dashboard:** https://dashboard.satgate.io/support
- **Email:** support@satgate.io

---

*SatGate™ — Protection and Payments for the Enterprise*

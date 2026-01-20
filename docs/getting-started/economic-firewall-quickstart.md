# Economic Firewall Quickstart

SatGate is the **Economic Firewall** for the Agentic Web. This guide shows how to configure the policy modes that control AI agent access to your APIs.

> ⚠️ **Enterprise Features**: Budget enforcement, Delegation v2, alerts, and settlement are Enterprise/Cloud features. OSS includes Protection Mode and L402 monetization only.

## The Four Modes

| Mode | Alias | Use Case | Friction | Tier |
|------|-------|----------|----------|------|
| **Protect** | `protect` / `capability` | Identity verification, no payment | Low | OSS |
| **Audit** | `audit` / `chargeback` | Internal metering, budget tracking | Low | Enterprise |
| **Budget** | `budget` / `fiat402` | Internal chargebacks with 402 challenge | Medium | Enterprise |
| **Monetize** | `monetize` / `l402` | External revenue via Lightning | High | OSS |

## OSS vs Enterprise

| Feature | OSS | Enterprise |
|---------|-----|------------|
| Protect Mode (capability tokens) | ✅ | ✅ |
| Monetize Mode (L402 Lightning) | ✅ | ✅ |
| Audit Mode (chargeback metering) | ❌ | ✅ |
| Budget Mode (Fiat402 enforcement) | ❌ | ✅ |
| Delegation v2 (token tree) | ❌ | ✅ |
| Budget alerts (80/90/100%) | ❌ | ✅ |
| Settlement APIs | ❌ | ✅ |
| L402 → Credits coupling | ❌ | ✅ |
| Spend export (JSON/CSV) | ❌ | ✅ |

## Mode 1: Protect (Capability Tokens)

**Use case**: Verify agent identity without payment. Ideal for internal services or trusted partners.

```yaml
routes:
  - name: "internal-api"
    match:
      pathPrefix: "/api/internal/"
    upstream: "backend"
    policy:
      kind: "protect"  # or "capability"
      scope: "api:internal"
```

**Agent access**:
```bash
# Get a capability token from admin API
TOKEN=$(curl -s -X POST http://localhost:9090/api/v1/tokens \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope": "api:internal", "expiresIn": 3600}' | jq -r .token)

# Access protected endpoint
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/internal/data
```

---

## Mode 2: Audit (Chargeback/Metering)

**Use case**: Track usage per department/team for internal billing. No 402 friction.

```yaml
routes:
  - name: "metered-api"
    match:
      pathPrefix: "/api/metered/"
    upstream: "llm-service"
    policy:
      kind: "audit"  # or "chargeback"
      scope: "api:metered"
      pay:
        mode: "chargeback"
        price: 0.001  # $0.001 per request (for metering only)
        unit: "USD"
        budgetEnforcement: "soft"  # Log overage, don't block
```

**How it works**:
- Requests are metered and tagged with department/cost-center from the macaroon
- Usage is exported to your billing/ERP system
- No 402 challenge - transparent to the agent

**Agent access**: Same as Protect mode (capability token required).

---

## Mode 3: Budget (Fiat402)

**Use case**: Internal APIs with real budget enforcement. Agent must acknowledge cost.

```yaml
routes:
  - name: "budget-controlled-api"
    match:
      pathPrefix: "/api/expensive/"
    upstream: "gpu-inference"
    policy:
      kind: "budget"  # or "fiat402"
      scope: "api:expensive"
      pay:
        mode: "fiat402"
        price: 0.10  # $0.10 per request
        unit: "USD"
        budgetEnforcement: "hard"  # Block when budget exhausted
```

**How it works**:
1. Agent calls endpoint without payment proof
2. Gateway returns `402 Payment Required` with Fiat402 invoice
3. Agent acknowledges cost (internal budget deduction)
4. Agent retries with receipt
5. Request proceeds, usage metered

**Agent access** (using SDK):
```python
from satgate import SatGateAgentClient

client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    budget_mode="fiat402",
    department_id="engineering"
)

# SDK automatically handles 402 challenge
response = client.get("/api/expensive/inference")
```

---

## Mode 4: Monetize (L402)

**Use case**: External API monetization. Real Lightning payments required.

```yaml
routes:
  - name: "premium-api"
    match:
      pathPrefix: "/api/premium/"
    upstream: "backend"
    policy:
      kind: "monetize"  # or "l402"
      priceSats: 100  # 100 satoshis per request
      scope: "api:premium"
      pay:
        mode: "l402"
        price: 100
        unit: "sats"

lightning:
  provider: "phoenixd"  # or "lnd", "cln"
  config:
    apiUrl: "http://phoenixd:9740"
    apiPassword: "${PHOENIXD_PASSWORD}"
```

**How it works**:
1. Agent calls endpoint without payment
2. Gateway returns `402 Payment Required` with Lightning invoice
3. Agent pays invoice via Lightning Network
4. Agent retries with L402 token (macaroon + preimage)
5. Gateway verifies payment, forwards request

**Agent access** (using SDK with NWC):
```python
from satgate import SatGateAgentClient

client = SatGateAgentClient(
    gateway_url="http://localhost:8080",
    nwc_connection_string="nostr+walletconnect://..."
)

# SDK automatically pays invoices and retries
response = client.get("/api/premium/data")
```

---

## Complete Example: Multi-Tier Configuration

```yaml
version: 1

server:
  listen: ":8080"

admin:
  token: "${ADMIN_TOKEN}"
  separateListener: "127.0.0.1:9090"

# Mint (Trust Broker) for agent identity
mint:
  enabled: true
  rootKey: "${MINT_ROOT_KEY}"
  useGatewayMacaroons: true
  identityProviders:
    - name: kubernetes
      type: kubernetes
      config:
        apiServer: "https://kubernetes.default.svc"
        audiences: ["satgate"]
        failClosed: true
    - name: aws
      type: aws
      config:
        allowedAccounts: ["123456789012"]
        irsaEnabled: true

lightning:
  provider: "phoenixd"
  config:
    apiUrl: "${PHOENIXD_URL}"
    apiPassword: "${PHOENIXD_PASSWORD}"

upstreams:
  internal:
    url: "http://internal-service:8080"
  llm:
    url: "http://llm-gateway:8080"
  public:
    url: "https://api.example.com"

routes:
  # Public health check
  - name: "health"
    match:
      pathExact: "/health"
    upstream: "internal"
    policy:
      kind: "public"

  # Internal agents - protect only
  - name: "internal-api"
    match:
      pathPrefix: "/api/internal/"
    upstream: "internal"
    policy:
      kind: "protect"
      scope: "api:internal"

  # Department-metered LLM access
  - name: "llm-metered"
    match:
      pathPrefix: "/api/llm/"
    upstream: "llm"
    policy:
      kind: "audit"
      scope: "api:llm"
      pay:
        mode: "chargeback"
        price: 0.01
        unit: "USD"

  # Budget-controlled expensive operations
  - name: "llm-premium"
    match:
      pathPrefix: "/api/llm-premium/"
    upstream: "llm"
    policy:
      kind: "budget"
      scope: "api:llm-premium"
      pay:
        mode: "fiat402"
        price: 0.50
        unit: "USD"
        budgetEnforcement: "hard"

  # External monetization
  - name: "public-api"
    match:
      pathPrefix: "/api/v1/"
    upstream: "public"
    policy:
      kind: "monetize"
      priceSats: 50
      scope: "api:public"

  # Default deny
  - name: "default"
    match:
      pathPrefix: "/"
    policy:
      kind: "deny"
```

---

## Agent SDK Quick Reference

### Python

```python
from satgate import SatGateAgentClient

# For Protect/Audit modes (capability token)
client = SatGateAgentClient(
    gateway_url="http://gateway:8080",
    mint_url="http://gateway:9090",  # If separate listener
    token="<capability-token>"
)

# For Budget mode (Fiat402)
client = SatGateAgentClient(
    gateway_url="http://gateway:8080",
    budget_mode="fiat402"
)

# For Monetize mode (L402 with NWC)
client = SatGateAgentClient(
    gateway_url="http://gateway:8080",
    nwc_connection_string="nostr+walletconnect://..."
)

# All modes support automatic retry on 402
response = client.get("/api/endpoint")
```

### Node.js

```typescript
import { SatGateAgentClient } from '@satgate/sdk';

// For Protect/Audit modes
const client = new SatGateAgentClient({
  gatewayUrl: 'http://gateway:8080',
  mintUrl: 'http://gateway:9090',
  token: '<capability-token>'
});

// For Monetize mode (L402)
const client = new SatGateAgentClient({
  gatewayUrl: 'http://gateway:8080',
  nwcConnectionString: 'nostr+walletconnect://...'
});

const response = await client.get('/api/endpoint');
```

---

## Mint (Trust Broker) Integration

For Kubernetes workloads to automatically obtain tokens:

```python
from satgate import SatGateAgentClient
from satgate.identity import KubernetesIdentity

# Automatically uses mounted ServiceAccount token
identity = KubernetesIdentity()

client = SatGateAgentClient(
    gateway_url="http://gateway:8080",
    mint_url="http://gateway:9090",
    identity=identity
)

# Token is automatically obtained from Mint
response = client.get("/api/internal/data")
```

For AWS workloads (EC2/Lambda/EKS):

```python
from satgate import SatGateAgentClient
from satgate.identity import AWSIdentity

# Uses presigned GetCallerIdentity URL
identity = AWSIdentity()

# Or for EKS with IRSA
identity = AWSIdentity(
    irsa_role_arn="arn:aws:iam::123456789012:role/MyAgentRole"
)

client = SatGateAgentClient(
    gateway_url="http://gateway:8080",
    mint_url="http://gateway:9090",
    identity=identity
)
```

---

---

## Enterprise: Delegation v2 (Token Tree)

> ⚠️ **Enterprise/Cloud Only**

Delegation v2 enables hierarchical capability tokens with budgets, scope restrictions, and cascade revocation.

### Key Concepts

| Concept | Description |
|---------|-------------|
| **Root Token** | Top-level token with full budget and scope |
| **Child Token** | Derived from parent with restricted scope/budget |
| **Cascade Revoke** | Revoking parent invalidates all descendants |
| **Budget Allocation** | Parent budget is transferred (not shared) to child |
| **Scope Subset** | Child can only have equal or fewer permissions |

### Create Root Token (Admin API)

```bash
curl -X POST "${GATEWAY_URL}/cloud/delegation-v2/tokens" \
  -H "X-Admin-Token: ${ADMIN_TOKEN}" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "engineering-root",
    "scope": {
      "routes": ["/api/*", "/control/*"],
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "policies": ["observe", "control"]
    },
    "budget_limit_credits": 10000,
    "expires_at": "2026-12-31T23:59:59Z",
    "cost_center": "engineering",
    "department": "platform"
  }'
```

**Response:**
```json
{
  "token_id": "tok_abc123...",
  "token_secret": "stks_secret123...",
  "budget": {
    "limit_credits": 10000,
    "remaining_credits": 10000
  }
}
```

### Create Child Token (Subset Scope)

```bash
curl -X POST "${GATEWAY_URL}/cloud/delegation-v2/tokens" \
  -H "X-Admin-Token: ${ADMIN_TOKEN}" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "parent_id": "tok_abc123...",
    "name": "frontend-team",
    "scope": {
      "routes": ["/api/*"],
      "methods": ["GET", "POST"],
      "policies": ["observe"]
    },
    "budget_limit_credits": 2000,
    "cost_center": "frontend"
  }'
```

**Validation Rules:**
- Child routes must be subset of parent routes
- Child methods must be subset of parent methods
- Child policies must be subset of parent policies
- Child budget must be ≤ parent remaining budget
- Child expiry must be ≤ parent expiry

### Use Delegation Token

```bash
curl "${GATEWAY_URL}/control/api/endpoint" \
  -H "X-SatGate-Tenant: ${TENANT_SLUG}" \
  -H "Authorization: Bearer stks_secret123..."
```

**Response Headers:**
```
X-Budget-Remaining: 9999
```

### Budget Enforcement (HTTP 402)

When budget is exhausted:

```bash
curl "${GATEWAY_URL}/control/api/endpoint" \
  -H "X-SatGate-Tenant: ${TENANT_SLUG}" \
  -H "Authorization: Bearer stks_exhausted..."
```

**Response:**
```
HTTP/1.1 402 Payment Required

{
  "error": "budget_exhausted",
  "remaining_credits": 0,
  "token_id": "tok_abc123..."
}
```

### L402 → Credits Coupling

When an L402 payment is made, credits can be added to a delegation token:

```bash
curl "${GATEWAY_URL}/pay/api/endpoint" \
  -H "X-SatGate-Tenant: ${TENANT_SLUG}" \
  -H "Authorization: L402 ${MACAROON}:${PREIMAGE}" \
  -H "X-Credit-Token: stks_delegation_token..."
```

**Response Headers:**
```
X-Credit-Settlement: ok
X-Credits-Added: 100
X-Budget-Remaining: 9100
```

**Settlement Status Values:**
| Status | Description |
|--------|-------------|
| `ok` | Credits added successfully |
| `duplicate` | Same payment already settled |
| `failed:invalid_token` | Credit token not found |
| `failed:settlement_error` | Settlement service error |

---

## Next Steps

- [Agent Policy Examples](../../config-templates/agent-policy-examples/) - YAML templates for common agent types
- [SDK Reference - Python](../sdks/python.md)
- [SDK Reference - Node.js](../sdks/nodejs.md)
- [Mint Configuration](../configuration/mint.md)
- [Enterprise Deployment](../enterprise/DEPLOYMENT_GUIDE.md)
- [Enterprise Feature Matrix](../enterprise/FEATURE_MATRIX.md)
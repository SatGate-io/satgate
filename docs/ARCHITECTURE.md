# ⚡ SatGate

## Stripe for AI Agents • EZ-Pass for the API Economy

**Meter in sats per request. No accounts. No bank required.**

---

## Executive Summary

**SatGate** is a production-ready API monetization gateway that uses Bitcoin's Lightning Network to charge for API access. No accounts, no API keys, no credit cards — just instant micropayments that work globally, 24/7.

> **Think of SatGate as Stripe for AI Agents** — but with one critical difference. Card rails can't economically process sub-cent transactions. Our agents pay 1 sat per request, instantly, with no bank account required. **We unlock the economy that card rails are too expensive to serve.**

Built on the **L402 protocol** (formerly LSAT), SatGate leverages HTTP status code `402 Payment Required` — reserved since 1999 for "future use" — to create a native internet payment layer for APIs.

### Two Products in One

| Use Case | What SatGate Does |
|----------|-------------------|
| **Monetize APIs per request** | Sub-cent pricing that's impossible on card rails. Charge 1 sat per call. |
| **Secure agent traffic with paid capabilities** | L402 tokens replace accounts/API keys. No PII, no credential stuffing. |

### The EZ-Pass Analogy

| | Stripe (Toll Booth) | SatGate (EZ-Pass) |
|---|---|---|
| **Flow** | Stop → Talk to attendant → Hand over card → Wait | Drive through at full speed |
| **Speed** | Slow, human-centric | Instant, machine-native |
| **Auth** | Payment separate from authorization | Payment-gated authorization (L402) |
| **Minimum** | ~$0.30 per transaction | 1 sat per transaction |

### The Stripe Parallel

Just as Stripe abstracts away the complexity of payments, SatGate abstracts away Lightning:

| Layer | Stripe Abstracts... | SatGate Abstracts... |
|-------|---------------------|----------------------|
| **Infrastructure** | Banking, card networks, PCI compliance | Lightning nodes, channel management, invoice generation |
| **Developer Experience** | 3 lines of code to accept payments | 3 lines of code to monetize any API |
| **Core Promise** | "Drop this in to start getting paid" | "Drop this in to start getting paid" |

**Side-by-side comparison:**

```javascript
// STRIPE (3 lines)
const stripe = require('stripe')(key);
const charge = await stripe.charges.create({
  amount: 1000, currency: 'usd'
});

// SATGATE (3 lines)
const client = new SatGateClient();
const data = await client.get('/api/premium');
// Payment handled automatically ⚡
```

---

## The Problem

### For API Providers

| Challenge | Impact |
|-----------|--------|
| **Bot abuse & scraping** | Inflated costs, degraded service |
| **Payment friction** | Lost customers during signup |
| **Chargebacks & fraud** | Revenue loss, operational overhead |
| **Global access barriers** | Users without credit cards excluded |
| **Micropayment economics** | Can't charge <$1 profitably |

### For API Consumers

| Challenge | Impact |
|-----------|--------|
| **Account fatigue** | Yet another login to manage |
| **Minimum commitments** | Forced to prepay for unused capacity |
| **Privacy concerns** | PII required for simple API access |
| **Geographic restrictions** | Banking limitations block access |

---

## The Solution

### SatGate: Lightning-Native API Access Control

```
┌──────────────┐                      ┌──────────────┐
│              │   1. Request API     │              │
│    Client    │─────────────────────▶│   SatGate    │
│              │                      │  (Aperture)  │
│              │◀─────────────────────│              │
│              │   2. 402 + Invoice   │              │
│              │                      │              │
│              │   3. Pay Lightning   │              │
│              │─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─▶│              │
│              │        ⚡             │              │
│              │   4. Retry + Token   │              │
│              │─────────────────────▶│              │
│              │                      │              │
│              │◀─────────────────────│              │
│              │   5. API Response    │              │
└──────────────┘                      └──────────────┘
```

### How It Works

| Step | Action | Technical Detail |
|------|--------|------------------|
| **1** | Client requests protected resource | `GET /api/analytics` |
| **2** | Server returns payment challenge | `402` + `WWW-Authenticate: L402 macaroon="...", invoice="..."` |
| **3** | Client pays Lightning invoice | Receives cryptographic preimage as proof |
| **4** | Client retries with token | `Authorization: L402 <macaroon>:<preimage>` |
| **5** | Server validates & responds | Token valid for 1 hour, reusable |

---

## Why Lightning? (Not Stablecoins or Stripe)

A common question: *"Why Bitcoin/Lightning instead of USDC or traditional payment rails?"*

### The Short Answer

Agents can't KYC. Stripe requires identity. Stablecoins require wallets that can be frozen. Lightning is the only payment rail where **code can pay code** without permission.

### The Long Answer

| Requirement | Stripe | Stablecoins (USDC) | Lightning |
|-------------|--------|-------------------|-----------|
| Sub-cent transactions ($0.001) | ❌ $0.30 minimum | ⚠️ Gas fees vary | ✅ <$0.01 fees |
| No KYC/Identity | ❌ Required | ⚠️ Wallet can be frozen | ✅ Permissionless |
| Instant finality | ❌ Chargebacks | ⚠️ Block confirmations | ✅ Milliseconds |
| Censorship-resistant | ❌ Account freezes | ❌ OFAC blacklists | ✅ No central authority |
| Agent-native | ❌ Needs human | ⚠️ Needs signing keys | ✅ Bearer tokens |

### Why Not Stablecoins?

1. **Freezable**: Circle (USDC) and Tether can blacklist addresses. An agent's wallet could be frozen without recourse.
2. **Gas volatility**: Ethereum L1 fees spike unpredictably. Even L2s have variable costs.
3. **Not truly permissionless**: Most stablecoin on-ramps require KYC.

### Why Not Stripe?

1. **Minimum fees**: $0.30 + 2.9% makes micropayments impossible.
2. **Identity required**: Agents can't pass KYC or hold bank accounts.
3. **Chargebacks**: Payments can be reversed, breaking the trust model.

### Lightning's Unique Properties

- **Bearer instrument**: The L402 token IS the proof of payment. No database lookup needed.
- **Atomic**: Payment either completes or fails. No partial states.
- **Instant**: Settlement in milliseconds, not days.
- **Programmable**: Macaroons enable caveats (expiry, usage limits) baked into the token.

This isn't ideological—it's practical. Lightning is currently the **only** payment rail that meets all requirements for autonomous agent commerce.

---

## Architecture

### Standard Deployment (Full Gateway)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              INTERNET                                    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         NGINX (Production)                               │
│                    TLS termination • Rate limiting • Caching             │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                          │
│                        APERTURE (L402 Gateway)                           │
│                                                                          │
│   ┌────────────────┐   ┌────────────────┐   ┌────────────────────────┐  │
│   │ Request Router │   │    Invoice     │   │    Token Validator     │  │
│   │                │   │   Generator    │   │  (Macaroon + Preimage) │  │
│   │ • Path matching│   │  (LNC → LND)   │   │                        │  │
│   │ • Price lookup │   │                │   │  • Signature check     │  │
│   │ • Free vs paid │   │  • Amount      │   │  • Expiry validation   │  │
│   └────────────────┘   │  • Expiry      │   │  • Caveat verification │  │
│                        └────────────────┘   └────────────────────────┘  │
│                                                                          │
│                        ┌────────────────┐                                │
│                        │ SQLite Database│                                │
│                        │ (Token Store)  │                                │
│                        └────────────────┘                                │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
           │                                              │
           │ LNC (Lightning Node Connect)                 │ HTTP Proxy
           ▼                                              ▼
┌─────────────────────────┐                ┌──────────────────────────────┐
│                         │                │                              │
│      VOLTAGE LND        │                │     BACKEND (Node.js)        │
│    (Lightning Node)     │                │                              │
│                         │                │   ┌────────────────────────┐ │
│  • Invoice generation   │                │   │  /api/analytics (paid) │ │
│  • Payment receipt      │                │   │  /api/free/ping (free) │ │
│  • Preimage validation  │                │   │  /health, /ready       │ │
│  • Channel management   │                │   └────────────────────────┘ │
│                         │                │                              │
└─────────────────────────┘                └──────────────────────────────┘
                                                          │
                                                          │ Static Files
                                                          ▼
                                           ┌──────────────────────────────┐
                                           │                              │
                                           │    FRONTEND (HTML/JS)        │
                                           │                              │
                                           │  • WebLN integration (Alby)  │
                                           │  • QR code generation        │
                                           │  • Token persistence         │
                                           │  • Payment flow UI           │
                                           │                              │
                                           └──────────────────────────────┘
```

### Enterprise "Sidecar" Deployment

**For enterprises with existing API Gateways (Kong, Apigee, AWS API Gateway):**

SatGate can be deployed as a **Sidecar** rather than a full gateway replacement. This lowers adoption risk by keeping existing infrastructure intact.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              INTERNET                                    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     EXISTING GATEWAY (Kong/Apigee/AWS)                   │
│                           Legacy Auth • Routing • Logging                │
└─────────────────────────────────────────────────────────────────────────┘
                    │                                    │
                    │ /api/legacy/*                      │ /api/agents/*
                    │ (Existing Traffic)                 │ (AI Agent Traffic)
                    ▼                                    ▼
    ┌───────────────────────────┐         ┌───────────────────────────────┐
    │                           │         │                               │
    │    LEGACY API BACKEND     │         │   SATGATE (Sidecar Mode)      │
    │                           │         │                               │
    │  • Existing auth          │         │   ┌─────────────────────────┐ │
    │  • Existing business logic│         │   │ Capability Validation   │ │
    │                           │         │   │ (Crawl/Walk)            │ │
    │                           │         │   ├─────────────────────────┤ │
    │                           │         │   │ Economic Firewall       │ │
    │                           │         │   │ (Run - L402 Payments)   │ │
    │                           │         │   └─────────────────────────┘ │
    │                           │         │             │                 │
    └───────────────────────────┘         │             ▼                 │
                                          │   ┌─────────────────────────┐ │
                                          │   │ Agent API Backend       │ │
                                          │   └─────────────────────────┘ │
                                          │                               │
                                          └───────────────────────────────┘
```

**Why Sidecar?**

| Objection | Gateway Replacement | Sidecar Deployment |
|-----------|--------------------|--------------------|
| "We've invested millions in Kong" | ❌ Rip & replace | ✅ Keep existing gateway |
| "Our team doesn't know SatGate" | ❌ Full migration | ✅ Only route agent traffic |
| "Too risky for production" | ❌ All-or-nothing | ✅ Incremental adoption |
| "Need approval from 5 teams" | ❌ Enterprise blocker | ✅ Just route `/agents/*` |

**The Pitch:**

> "Keep your existing Gateway for routing and legacy auth. Drop SatGate in specifically to handle the Agent Economy traffic. We're an addon, not a replacement."

---

## Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Gateway** | Aperture (Go) | L402 protocol, invoice generation, token validation |
| **Backend** | Node.js + Express | Business logic, API endpoints, health checks |
| **Lightning** | Voltage LND + LNC | Payment infrastructure, hosted node |
| **Frontend** | Next.js + Tailwind | Landing page, interactive playground |
| **SDKs** | JavaScript, Python | Client libraries for browsers and agents |
| **Database** | SQLite | Token storage, payment tracking |
| **Reverse Proxy** | Nginx | TLS, rate limiting, caching |
| **Container** | Docker + Compose | Deployment, orchestration |

---

## Client SDKs

SatGate provides production-ready SDKs that handle the complete L402 payment flow automatically.

### JavaScript SDK (Browser / Node.js)

For web applications with WebLN wallet support (Alby, etc.):

```javascript
import { SatGateClient } from 'satgate-sdk';

const client = new SatGateClient({
  baseUrl: 'https://api.yoursite.com',
  onPayment: (payment) => console.log(`Paid ${payment.amount} sats`)
});

// Automatic: 402 → Pay Invoice → Retry with Token → Response
const data = await client.get('/api/premium/analytics');
```

**Features:**
- WebLN integration for browser wallets
- Auth Discovery (tries LSAT/L402, raw/base64 formats)
- Payment callbacks for UI integration
- Token caching and reuse

**Installation:**
```bash
npm install satgate-sdk
```

### Python SDK (Agents / Backend)

For AI agents and server-side applications:

```python
from satgate import SatGateSession, LightningWallet

class MyWallet(LightningWallet):
    def pay_invoice(self, invoice: str) -> str:
        # Connect to your LND node, LNBits, or wallet API
        return lnd.pay(invoice).preimage

session = SatGateSession(wallet=MyWallet())

# Automatic: 402 → Pay Invoice → Retry with Token → Response
response = session.get('https://api.yoursite.com/api/premium/data')
data = response.json()
```

**Installation:**
```bash
pip install satgate
```

### LangChain Integration (AI Agents)

Give your AI agents a "credit card" to access paid APIs:

```python
from satgate.langchain import SatGateTool
from langchain.agents import initialize_agent

# The agent can now autonomously pay for API access
tools = [SatGateTool(wallet=my_wallet)]
agent = initialize_agent(tools, llm, agent="openai-functions")

# Agent will: discover API → hit 402 → pay invoice → get data
agent.run("Fetch the premium market analysis from SatGate")
```

**Why This Matters:**
- AI agents can't have credit cards or bank accounts
- L402 + Lightning = programmatic payments without human intervention
- Agents can browse the "economic web" autonomously

---

## Key Features

### Security & Protection

- **Capability-Based Access** — L402 tokens encode permissions; no identity database required
- **Economic Friction for L7 Abuse** — Every request costs real money, making scraping/bots expensive and self-limiting (use alongside WAF/CDN for volumetric protection)
- **Zero Trust PEP** — Per-request verification at the edge; no network trust assumptions
- **Rate Limiting** — Configurable limits at nginx and application layers
- **Security Headers** — Helmet.js, CORS, CSP policies
- **TLS Encryption** — Let's Encrypt or custom certificates
- **No Stored Credentials** — Tokens are self-validating macaroons with caveats (scope, time, budget)

### Payment Flexibility

- **Micropayments** — Charge as little as 1 satoshi (sats-first pricing)
- **Instant Settlement** — Payments confirm in milliseconds
- **Global Access** — Works anywhere Lightning reaches
- **No Chargebacks** — Payments are cryptographically final
- **Configurable Pricing** — Per-endpoint, per-tier pricing

### Developer Experience

- **Zero Account Friction** — No signup, no API keys
- **WebLN Support** — One-click payment with browser wallets
- **QR Codes** — Mobile wallet compatibility
- **Token Caching** — Pay once, reuse for 1 hour
- **Health Endpoints** — Production monitoring ready

---

## Capability-Based Security

Traditional APIs use **identity-based** access: "Prove *who you are*, then we decide what you can do."

SatGate uses **capability-based** access: "Present a token that *already encodes* what you can do."

### Why This Matters

| | Identity-Based (OAuth/API Keys) | Capability-Based (L402) |
|---|---|---|
| **Model** | Who you are | What you hold |
| **Requires** | User databases, PII | Cryptographic tokens |
| **Risk** | Credential stuffing, breaches | Token theft (mitigated by short-lived caveats) |
| **For Agents** | ❌ Can't sign up | ✅ Just present token |

### Security Features

- **No Accounts Required** — Access via L402 bearer tokens (macaroons + proof-of-payment), not usernames or API keys
- **Edge Verification** — Tokens verified cryptographically at the gateway; no centralized identity store needed (usage accounting/quotas can be tracked without storing PII)
- **Least Privilege** — Add caveats to constrain scope, time, audience, and budget (e.g., `"valid_until": 5min`, `"max_calls": 10`)
- **Delegatable** — Attenuate tokens before passing to sub-agents; permissions only shrink, never grow

> **The security primitive:** L402 creates *paid capabilities* — cryptographic tokens where payment gates issuance and the token itself encodes permissions.

### Zero Trust Access Control (PEP)

SatGate is a **Zero Trust Policy Enforcement Point** — the gateway that verifies every protected request and enforces scoped access via L402/macaroons.

- **Per-request verification** — Every protected call requires a valid L402 token; no network trust assumptions
- **Continuous authorization** — Token validated on each request, not just at session start
- **Least privilege by design** — Macaroon caveats constrain scope, time, and budget
- **Reduced trust dependencies** — Cryptographic verification without centralized user databases

> ✅ Zero Trust PEP for API access  
> ✅ Complements existing security stack (WAF/CDN, rate limiting, SIEM)  
> ⚠️ Not a full Zero Trust program (identity governance, device posture, microsegmentation)

---

## Configuration

### Pricing & Access Control

```yaml
# aperture.yaml
services:
  # Premium endpoint — 1000 sats ($1)
  - name: paid-analytics
    pathregexp: '^/api/analytics($|/.*)$'
    price: 1000
    timeout: 3600  # 1 hour token validity

  # Free endpoints
  - name: public
    pathregexp: '^/.*$'
    price: 0
    authwhitelistpaths:
      - '^/api/free/.*$'
      - '^/health$'
```

### Current Settings

| Parameter | Value |
|-----------|-------|
| **Price per request** | 1,000 sats (~$1.00) |
| **Token validity** | 1 hour |
| **Free endpoints** | `/api/free/*`, `/health`, `/ready` |
| **Rate limit (API)** | 100 requests / 15 minutes |
| **Rate limit (Nginx)** | 10 requests / second |

---

## Value Proposition

### Comparison: Traditional vs SatGate

| Aspect | Traditional API Monetization | SatGate (L402) |
|--------|------------------------------|----------------|
| **Onboarding** | Create account, verify email, add payment | Just pay |
| **Payment method** | Credit card required | Lightning (Bitcoin) |
| **Time to access** | Minutes to days | < 1 second |
| **Minimum charge** | $5-50 (processing fees) | 1 sat (~$0.001) |
| **Chargebacks** | Possible, costly | Impossible |
| **Global access** | Banking required | Permissionless |
| **Privacy** | PII required | Pseudonymous |
| **Bot protection** | CAPTCHAs, heuristics | Economic barrier |

### ROI for API Providers

| Benefit | Impact |
|---------|--------|
| **Eliminate bot abuse** | Reduce infrastructure costs 50-90% |
| **Zero payment fraud** | No chargebacks, no disputes |
| **Global market access** | Reach unbanked users (2B+ people) |
| **Micropayment revenue** | Monetize previously free endpoints |
| **Reduced ops overhead** | No account management, no billing disputes |

### Benefits for API Consumers

| Benefit | Impact |
|---------|--------|
| **Instant access** | No waiting for approval |
| **Pay-per-use** | Only pay for what you consume |
| **Privacy preserved** | No personal information required |
| **No commitments** | No subscriptions, no minimums |
| **Works everywhere** | No geographic restrictions |

---

## Use Cases

### Ideal For

- **AI/ML APIs** — Per-inference pricing for models
- **Data APIs** — Pay-per-query for datasets
- **Premium Content** — Article unlocks, media access
- **Rate-Limited Services** — Tiered access levels
- **Bot-Sensitive Endpoints** — Search, scraping protection
- **Global Services** — Users without traditional banking

### Example Pricing Models

| Use Case | Price | Token Duration |
|----------|-------|----------------|
| AI inference | 100-10,000 sats | Per request |
| Data query | 50-500 sats | Per query |
| Article unlock | 1,000 sats | 24 hours |
| API tier upgrade | 10,000 sats | 1 hour |
| Premium feature | 5,000 sats | Session |

---

## Getting Started

### Quick Start

```bash
# 1. Clone and install
git clone https://github.com/SatGate-io/satgate.git && cd satgate
npm install

# 2. Configure Aperture (edit proxy/aperture.yaml)
#    Add your LNC credentials to the authenticator section:
#    authenticator:
#      passphrase: "your-10-word-lnc-phrase"
#      mailboxaddress: "mailbox.terminal.lightning.today:443"
#      network: "mainnet"

# 3. Start backend
node proxy/server.js

# 4. Start Aperture (in another terminal)
aperture --configfile=proxy/aperture.yaml

# 5. Test it
curl http://localhost:8081/api/free/ping     # ✅ Free
curl http://localhost:8081/api/micro/ping    # ⚡ 402 → Pay 1 sat
```

> ⚠️ **Security:** Keep secrets in config files (with proper permissions), not CLI flags. CLI args leak via process listings (`ps aux`). Ensure `aperture.yaml` is not committed to version control.

### Production Deployment

#### Option 1: One-Command Installer

```bash
# Interactive setup - prompts for LNC credentials
./install.sh
```

#### Option 2: Docker Compose (Full Stack)

```bash
# 1. Configure environment
cp env.example .env
# Edit .env with your LNC passphrase

# 2. Start all services (Backend + Aperture + Docs)
docker compose -f docker-compose.full.yml up -d

# Services:
#   - Aperture (L402 Gateway): http://localhost:8081
#   - Backend API: http://localhost:8083 (internal)
#   - Playground/Docs: http://localhost:8080
```

#### Option 3: Makefile Commands

```bash
make install   # Interactive setup
make start     # Start all services
make stop      # Stop all services
make logs      # View logs
make clean     # Remove containers and volumes
```

---

## Dynamic Pricing API

Adjust prices at runtime without restarting Aperture:

```bash
# Get current pricing
curl http://localhost:8083/api/free/pricing

# Update a tier
curl -X PUT http://localhost:8083/api/free/pricing/premium \
  -H "Content-Type: application/json" \
  -d '{"price": 2000}'

# Bulk update
curl -X POST http://localhost:8083/api/free/pricing/bulk \
  -H "Content-Type: application/json" \
  -d '{"basic": 10, "standard": 100, "premium": 1000}'
```

---

## Interactive Playground

Two playground options are available:

### 1. Original Playground (`docs/index.html`)
- Vanilla HTML/JS implementation
- WebLN integration
- Verbose logging with truncated hashes
- "View Client Code" snippets

### 2. Next.js Playground (`satgate-landing/`)
- Modern React implementation
- **Simulation Mode**: Demo without real payments (for investors/mobile)
- **Real Network Mode**: Actual Lightning payments via Alby
- Endpoint selector (10/100/1000 sats)
- Visual step indicators

```bash
# Run the Next.js landing page + playground
cd satgate-landing && npm run dev
# Visit http://localhost:3000/playground
```

---

## Resources

- **L402 Protocol Specification**: [lightning.engineering/posts/2020-03-30-lsat](https://lightning.engineering/posts/2020-03-30-lsat/)
- **Aperture Documentation**: [github.com/lightninglabs/aperture](https://github.com/lightninglabs/aperture)
- **Lightning Network**: [lightning.network](https://lightning.network)
- **Voltage (Hosted LND)**: [voltage.cloud](https://voltage.cloud)
- **WebLN Specification**: [webln.guide](https://www.webln.guide/)

---

## Summary

**SatGate** transforms API monetization by replacing traditional account-based access with instant Lightning payments. 

**For providers**: Eliminate bots, fraud, and payment friction while opening global markets.

**For consumers**: Get instant, private, pay-per-use access to any API.

> *The internet finally has a native payment layer. SatGate brings it to your API.*

---

<p align="center">
  <strong>⚡ SatGate</strong><br>
  <em>Protect & monetize your API with Lightning. Instant. Permissionless.</em>
</p>

---

*Document Version: 2.0 | Last Updated: November 2025*

## Changelog

### v2.2 (December 2025)
- Added **Governance Dashboard** with real-time WebSocket updates
- Added **Cytoscape.js** force-directed graph visualization
- Added **Kill Switch** for instant token revocation
- Added **Redis persistence** for banned tokens and stats
- Published **Python SDK v0.3.0** with LangChain integration
- Published **JavaScript SDK** to npm (`@satgate/sdk`)
- Added **Go SDK** with LNBits, Alby, LND wallet support

### v2.1 (December 2025)
- Added "Sidecar" deployment model for enterprise adoption
- Added Economic Firewall positioning (security-first framing)
- Added Governance Inspector CLI (`cli/inspect.js`)
- Added Telemetry module for dashboard data stream
- Enhanced Demo Playbook with Mission Control summary

### v2.0 (November 2025)
- Added JavaScript SDK with WebLN support
- Added Python SDK with LangChain integration
- Added Docker one-command installer
- Added Dynamic Pricing API
- Added dual-mode Playground (Simulation + Real Network)
- Added Next.js landing page

### v1.0 (November 2025)
- Initial architecture documentation
- Core L402 flow implementation
- Aperture + Backend + Frontend stack



<p align="center">
  <img src="https://img.shields.io/badge/⚡-SatGate-orange?style=for-the-badge&labelColor=black" alt="SatGate" />
</p>

<h1 align="center">⚡ SatGate</h1>

<p align="center">
  <strong>Stripe for AI Agents • EZ-Pass for the API Economy</strong><br>
  <em>~$0.001 per request. No accounts. No bank required.</em>
</p>

<p align="center">
  <a href="#-quick-start"><img src="https://img.shields.io/badge/Quick_Start-5_min-green?style=flat-square" alt="Quick Start" /></a>
  <a href="https://github.com/SatGate-io/satgate/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" alt="MIT License" /></a>
  <img src="https://img.shields.io/badge/Status-Patent_Pending-orange?style=flat-square" alt="Patent Pending" />
  <img src="https://img.shields.io/badge/Protocol-L402-purple?style=flat-square" alt="L402" />
  <img src="https://img.shields.io/badge/Payment-Lightning-yellow?style=flat-square" alt="Lightning" />
</p>

<p align="center">
  <a href="#-python-sdk">Python SDK</a> •
  <a href="#-javascript-sdk">JS SDK</a> •
  <a href="#-docker">Docker</a> •
  <a href="https://satgate.io/playground">Live Demo</a>
</p>

---

<p align="center">
  <img src="docs/screenshots/satgate-hero-demo.gif" alt="AI Agent paying 1 satoshi for API access" width="700" />
  <br>
  <em>An AI agent pays ~$0.001 for API access — in real-time</em>
</p>

---

## The Problem

```
An AI agent needs 50 API calls to research a topic.
Each call costs $0.01.

On Card Rails:  $15.00 in fees (50 × $0.30 minimum)  ← 3,000% overhead
On SatGate:     $0.50 total                          ← Just the value
```

**Captures the high-frequency agent traffic that is economically unviable on card rails.**

**Bonus:** Every request costs money, so bots and scrapers can't abuse your API. Don't block them—monetize them. *(Economic Rate Limiting — complements your WAF/CDN)*

---

## 🐍 Python SDK

```bash
pip install satgate
```

```python
from satgate import SatGateSession

# Create a session with your Lightning wallet
session = SatGateSession(wallet=my_wallet)

# That's it. 402s are handled automatically.
response = session.get("https://api.example.com/premium/data")
print(response.json())
```

### LangChain Integration

```python
from satgate import SatGateTool
from langchain.agents import initialize_agent

# Give your AI agent a wallet
tools = [SatGateTool(wallet=my_wallet)]
agent = initialize_agent(tools, llm, agent="openai-functions")

# Let it roam the paid API economy
agent.run("Fetch the premium market report from AlphaVantage")
```

---

## 📦 JavaScript SDK

```bash
npm install satgate-sdk
```

```javascript
import { SatGateClient } from 'satgate-sdk';

const client = new SatGateClient();

// Automatic payment handling via WebLN (Alby)
const data = await client.get('https://api.example.com/premium');
console.log(data);
```

---

## 🚀 Quick Start

### Option 1: Docker (Recommended)

```bash
git clone https://github.com/SatGate-io/satgate.git
cd satgate/docker
cp env.example .env
# Edit .env with your LNC credentials
docker compose -f docker-compose.full.yml up -d
```

### Option 2: Local Development

```bash
# 1. Clone
git clone https://github.com/SatGate-io/satgate.git
cd satgate

# 2. Install
npm install

# 3. Start the backend
node proxy/server.js

# 4. Start Aperture (in another terminal)
aperture --configfile=proxy/aperture.yaml \
  --authenticator.network=mainnet \
  --authenticator.passphrase="your-10-word-lnc-phrase" \
  --authenticator.mailboxaddress="mailbox.terminal.lightning.today:443"

# 5. Test it
curl http://localhost:8081/api/free/ping     # ✅ Free
curl http://localhost:8081/api/micro/ping    # ⚡ 402 → Pay 1 sat
```

---

## 💰 Pricing Tiers

| Endpoint | Price | ~USD* | Use Case |
|----------|-------|-------|----------|
| `/api/micro/*` | 1 sat | ~$0.001 | True micropayments |
| `/api/basic/*` | 10 sats | ~$0.01 | High-volume |
| `/api/standard/*` | 100 sats | ~$0.10 | Analytics |
| `/api/premium/*` | 1000 sats | ~$1.00 | AI inference |
| `/api/free/*` | Free | — | Health checks |

*USD values approximate; varies with BTC price.

Configure in `proxy/aperture.yaml`:

```yaml
services:
  - name: micro
    pathregexp: '^/api/micro($|/.*)$'
    price: 1      # 1 satoshi ≈ $0.001
    timeout: 86400
```

---

## 🏗️ Architecture

```
┌─────────────┐     402 + Invoice    ┌──────────────┐     Forward     ┌─────────────┐
│   Client    │◄────────────────────►│   SatGate    │◄───────────────►│  Your API   │
│ (Human/AI)  │     L402 Token       │  (Aperture)  │    Validate     │  (Backend)  │
└─────────────┘                      └──────────────┘                 └─────────────┘
       │                                    │
       │         ⚡ Pay Invoice             │
       └────────────────────────────────────┘
                 Lightning Network
```

### How L402 Works (3 Steps)

1. **402 Response** — Client requests protected endpoint, gateway returns `HTTP 402` with a Lightning invoice
2. **Pay Invoice** — Client pays invoice via Lightning, receives cryptographic preimage
3. **L402 Token** — Client combines macaroon + preimage into an `Authorization: L402` header for access

**L402 Token = Macaroon + Preimage** — A bearer credential with embedded permissions (caveats) that proves payment.

### Non-Custodial by Design

> **SatGate never holds your funds.** We help generate invoices, but payments settle directly to your Lightning node (or your partner custodian). We never hold your keys.

---

## 🔐 Capability-Based Security

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
- **Edge Verification** — Tokens verified cryptographically at the gateway; no centralized identity store needed
- **Least Privilege** — Add caveats to constrain scope, time, audience, and budget (e.g., `"valid_until": 5min`, `"max_calls": 10`)
- **Economic Abuse Friction** — Spam becomes expensive and self-limiting; high-volume callers must continuously pay
- **Privacy-Forward** — Zero PII collection; reduced credential-stuffing exposure with short-lived scoped tokens

> **The security primitive:** L402 creates *paid capabilities* — cryptographic tokens where payment gates issuance and the token itself encodes permissions.

---

## 📁 Repository Structure

```
satgate/
├── README.md           # You are here
├── LICENSE             # MIT
├── proxy/              # Gateway (Aperture config + Node.js backend)
│   ├── aperture.yaml   # Pricing configuration
│   ├── server.js       # API endpoints
│   └── nginx/          # Production configs
├── sdk/
│   ├── python/         # Python SDK + LangChain Tool
│   └── js/             # JavaScript SDK + TypeScript
├── docker/             # One-click deployment
│   ├── docker-compose.full.yml
│   └── env.example
├── examples/           # Demo scripts
│   └── hero_demo.py    # The "money shot" demo
└── docs/               # Documentation & pitch decks
```

---

## 🔑 Prerequisites

1. **Lightning Node** with LNC enabled:
   - [Voltage](https://voltage.cloud) (Managed)
   - [Umbrel](https://umbrel.com) (Self-hosted)
   - Any LND node

2. **LNC Pairing Phrase**:
   - Go to [Terminal Web](https://terminal.lightning.engineering)
   - Create new LNC session
   - Copy your 10-word phrase

---

## 🎬 Demo

<p align="center">
  <a href="docs/screenshots/satgate-hero-demo.mp4">
    <img src="docs/screenshots/satgate-hero-demo.gif" alt="SatGate Demo" width="600" />
  </a>
</p>

```bash
python examples/hero_demo.py
```

Watch an AI agent autonomously pay for API access in real-time.

---

## 🔍 Troubleshooting

| Error | Solution |
|-------|----------|
| "Self-payment not allowed" | Use a different wallet than your node |
| "Cannot find payment route" | Need inbound liquidity (Voltage Flow, LN+) |
| "L402 has expired" | Restart Aperture to refresh macaroons |
| CORS errors | Ensure Aperture is running on port 8081 |

---

## 📚 Resources

- [L402 Protocol Spec](https://lightning.engineering/posts/2020-03-30-lsat/)
- [Aperture Docs](https://github.com/lightninglabs/aperture)
- [WebLN Guide](https://www.webln.guide/)
- [Voltage Cloud](https://voltage.cloud)

---

## 🤝 Contributing

PRs welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## 📄 License

MIT License — See [LICENSE](LICENSE)

---

<p align="center">
  <strong>© 2025 SatGate. Patent Pending.</strong><br>
  <em>Stripe for AI Agents • EZ-Pass for the API Economy</em>
</p>

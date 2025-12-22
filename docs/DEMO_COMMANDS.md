# SatGate Demo Commands

Quick reference for demonstrating SatGate capabilities.

---

## 🎯 Live Infrastructure

| Component | URL |
|-----------|-----|
| **Landing Page** | https://satgate.io |
| **Playground** | https://satgate.io/playground |
| **Dashboard (Live)** | https://satgate-production.up.railway.app/dashboard |
| **Gateway** | https://satgate-production.up.railway.app |
| **Health Check** | https://satgate-production.up.railway.app/health |
| **Governance API** | https://satgate-production.up.railway.app/api/governance/graph |

---

## 🔐 Phase 1: Capability Tokens (No Crypto Required)

### Health Check
```bash
curl https://satgate-production.up.railway.app/health
```

### Mint a Token
```bash
curl -X POST https://satgate-production.up.railway.app/api/capability/mint
```

### Use a Token
```bash
# Store token in variable
TOKEN=$(curl -s -X POST https://satgate-production.up.railway.app/api/capability/mint | jq -r '.token')

# Use it
curl -H "Authorization: Bearer $TOKEN" \
  https://satgate-production.up.railway.app/api/capability/ping
```

### Offline Delegation Demo
```bash
# Option A: Run locally (requires Node.js)
node cli/delegation-demo.js

# Option B: Server-side (works from any device)
curl -X POST https://satgate-production.up.railway.app/api/capability/demo/delegate
```

### Token Inspection (Governance)
```bash
# Inspect any token's constraints and lineage
node cli/inspect.js <TOKEN>

# JSON output for automation
node cli/inspect.js <TOKEN> --json
```

### Scope Enforcement Test
```bash
# Get a delegated child token (scope: ping only)
CHILD=$(curl -s -X POST https://satgate-production.up.railway.app/api/capability/demo/delegate | jq -r '.childToken')

# This works (allowed scope)
curl -H "Authorization: Bearer $CHILD" \
  https://satgate-production.up.railway.app/api/capability/ping

# This fails (scope violation)
curl -X POST -H "Authorization: Bearer $CHILD" \
  https://satgate-production.up.railway.app/api/capability/mint
```

---

## 💰 Phase 3: L402 Payments (Economic Firewall)

### Trigger 402 Challenge
```bash
curl -i https://satgate-production.up.railway.app/api/micro/ping
```

### Interactive Playground
Open https://satgate.io/playground with an Alby wallet to complete the full payment flow.

---

## 🛡️ Governance & Kill Switch

### View Live Dashboard
Open https://satgate-production.up.railway.app/dashboard in a browser.

The dashboard shows:
- **Active Tokens** — Observed agents in last 10 minutes
- **Economic Firewall** — Blocked unpaid requests (scrapers, bots)
- **Kill Switch Hits** — Revoked tokens that attempted access

### Get Governance Data (API)
```bash
curl https://satgate-production.up.railway.app/api/governance/graph
```

### Get Stats Summary
```bash
curl https://satgate-production.up.railway.app/api/governance/stats
```

### Kill Switch: Ban a Token
```bash
# Get a token signature first (from inspect tool or dashboard)
curl -X POST -H "Content-Type: application/json" \
  -d '{"tokenSignature":"<TOKEN_SIGNATURE>","reason":"Compromised"}' \
  https://satgate-production.up.railway.app/api/governance/ban
```

### Kill Switch: Unban a Token
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"tokenSignature":"<TOKEN_SIGNATURE>"}' \
  https://satgate-production.up.railway.app/api/governance/unban
```

### List Banned Tokens
```bash
curl https://satgate-production.up.railway.app/api/governance/banned
```

---

## 📂 CLI Tools

| Tool | Purpose |
|------|---------|
| `cli/mint-token.js` | Mint capability tokens |
| `cli/delegation-demo.js` | Demonstrate offline delegation |
| `cli/inspect.js` | Governance inspector (audit tokens) |

---

## 🔧 Local Development

```bash
# Start backend
node proxy/server.js

# Test capability endpoint
curl -X POST http://localhost:8083/api/capability/mint
```

---

## ✅ Any-Device Ready

All commands work from any terminal with `curl` — iPad, borrowed laptop, phone SSH.

Base URL: `https://satgate-production.up.railway.app`


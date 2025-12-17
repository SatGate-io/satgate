# SatGate Demo Playbook

**The complete demo script for presenting Zero Trust access control and API monetization.**

This document provides commands, talk tracks, and objection handlers for demonstrating SatGate's full capability—from governance (Phase 1) to monetization (Phase 3).

---

## Pre-Demo Checklist

- [ ] Terminal open with clean screen
- [ ] Railway deployment healthy (`curl https://satgate-production.up.railway.app/health`)
- [ ] Test delegation works: `node cli/delegation-demo.js` (local) OR `curl -X POST .../api/capability/demo/delegate` (server)
- [ ] Backup tokens pre-minted (in case of network issues)
- [ ] (Phase 3 only) Alby wallet funded with a few sats

> ✅ **Any-Device Ready:** All demo commands work via curl. No laptop required.

---

# 🔐 Phase 1: Capability-Only Access (No Crypto)

**Goal:** Prove we solve IAM scale problems *today* without Bitcoin.

---

### **Scene 1: The "CISO" Persona (Token Minting)**

*Narrative:* "I am the CISO. I don't create a user. I issue a capability."

**Command:**
```bash
# Mint a master token for the "Data Agent"
curl -X POST https://satgate-production.up.railway.app/api/capability/mint
```

**Talk Track:**

> "See that? No database write. No service account. Just a token that says 'You can read data for 1 hour'. It's stateless."

---

### **Scene 2: The "Agent" Persona (Usage)**

*Narrative:* "The agent uses the token. Watch the speed."

**Command:**
```bash
# Use the token (Copy/Paste from previous step output)
curl -H "Authorization: Bearer <PASTE_TOKEN_HERE>" \
  https://satgate-production.up.railway.app/api/capability/ping
```

> 💡 **Pro-Tip: Smoother Demo Flow**
> Store the token in a variable to avoid copy-paste friction:
> ```bash
> # After minting, store the token:
> TOKEN=$(curl -s -X POST https://satgate-production.up.railway.app/api/capability/mint | jq -r '.token')
> 
> # Now use it cleanly:
> curl -H "Authorization: Bearer $TOKEN" https://satgate-production.up.railway.app/api/capability/ping
> ```

**Talk Track:**

> "Authenticated instantly. The Gateway validated the signature mathematically. No LDAP lookup. No bottleneck."

---

### **Scene 3: The "Superpower" (Offline Delegation)**

*Narrative:* "Now the Agent needs to delegate a task to a sub-worker. In the old world, this is a ticket. In our world, it's math."

**Command (Option A - Local):**
```bash
# Run the delegation simulation locally
node cli/delegation-demo.js
```

**Command (Option B - Server Backup):**
```bash
# If laptop unavailable, run from ANY terminal:
curl -X POST https://satgate-production.up.railway.app/api/capability/demo/delegate
```

> 💡 **"Two is one, one is none"** — Option B works from any device with curl (iPad, borrowed laptop, phone terminal app).

**Talk Track (Option A - Local):**

> "Watch the '[NETWORK]' line. Zero requests. The agent is minting a restricted credential offline. It just cut a spare key for the janitor that only opens the basement and expires in 5 minutes. This is the Google-grade capability competitors can't touch."

**Talk Track (Option B - Server Backup):**

> "I'm triggering this remotely to show you the logic. Even though this ran on my server just now for the demo, in production this entire logic happens inside the Agent's own code—client-side. That means zero latency and zero dependency on our uptime to create these sub-credentials. The agent doesn't phone home to delegate."

---

### **Scene 3b: The "Security Proof" (Least Privilege)**

*Narrative:* "But here's the key question: Can the child token escalate privileges? Let's test it."

> 💡 **Note:** Both Option A and Option B from Scene 3 output a CHILD TOKEN at the end. Copy that token for the tests below.

> 💡 **Pro-Tip: Store Child Token in Variable**
> ```bash
> # After running the delegation demo, store the child token:
> CHILD_TOKEN="<paste the child token here>"
> 
> # Now use it for both tests:
> curl -X POST -H "Authorization: Bearer $CHILD_TOKEN" .../api/capability/mint  # BLOCKED
> curl -H "Authorization: Bearer $CHILD_TOKEN" .../api/capability/ping          # ALLOWED
> ```

**Command 1: The BLOCKED Action (Negative Test)**
```bash
# Copy the CHILD token from Scene 3 output
# Try to mint a NEW token with it (should FAIL)
curl -X POST -H "Authorization: Bearer <PASTE_CHILD_TOKEN>" \
  https://satgate-production.up.railway.app/api/capability/mint
```

**Expected Output:**
```json
{
  "error": "Access Denied",
  "reason": "caveat check failed (scope = api:capability:ping): Scope violation: token has 'api:capability:ping', need 'api:capability:admin'",
  "hint": "Token scope does not permit this action"
}
```

**Talk Track:**

> "403 Forbidden. The child token tried to escalate privileges—to mint a new token. The Gateway rejected it. Not because we looked it up in a database, but because the **token itself** said 'I can only access /ping'. The math enforced least privilege."

---

**Command 2: The ALLOWED Action (Positive Test)**
```bash
# Same child token, but for its intended purpose
curl -H "Authorization: Bearer <PASTE_CHILD_TOKEN>" \
  https://satgate-production.up.railway.app/api/capability/ping
```

**Expected Output:**
```json
{
  "ok": true,
  "tier": "capability",
  "mode": "Phase 1: Capability-Only",
  "message": "✓ Authenticated with capability token..."
}
```

**Talk Track:**

> "200 OK. Same token, correct endpoint. The janitor's key opens the basement—nothing more, nothing less. This is Zero Trust at the token level."

---

### **Scene 4: The Phase 1 Close**

*Narrative:* "Everything you just saw happened without a single satoshi of Bitcoin."

**Closing Line:**

> "Phase 1 is live. Phase 3 (Payments) is just a config change away."

---

# 💰 Phase 3: The "Revenue Switch" (Optional Closer)

**Goal:** Prove that monetization is a config change, not a re-platform.

> 💡 **When to show this:** Only if the partner is interested in the monetization story. Phase 1 alone is often enough for governance-focused clients.

---

### **Scene 5: The Config Diff**

*Narrative:* "When you're ready to charge for this API, we don't deploy new code. We just update the policy."

**Show the conceptual config change:**

```yaml
services:
  - name: "my-service"
    # Phase 1: Governance Only (No payment)
    # price: 0
    
    # Phase 3: Monetized (Uncomment to enable)
    price: 1  # sats per request
```

**Talk Track:**

> "One line. That's the difference between 'access control' and 'revenue generation'. Same Gateway. Same tokens. New capability."

---

### **Scene 6: The 402 Response (CLI)**

*Narrative:* "Let me show you what happens when payment is required."

**Command:**
```bash
# Hit the paid endpoint without payment
curl -i https://satgate-production.up.railway.app/api/micro/ping
```

**Expected Output:**
```
HTTP/1.1 402 Payment Required
WWW-Authenticate: L402 macaroon="...", invoice="lnbc..."
```

**Talk Track:**

> "402 Payment Required. This is a standard HTTP status code—not a proprietary hack. The response includes a Lightning invoice. The agent pays it instantly, gets a cryptographic token, and proceeds. No credit cards. No net-30 invoicing. No chargebacks."

---

### **Scene 7: The Payment Flow (Playground UI - Optional)**

*Narrative:* "Let me show you the full experience."

**Option A: Live Payment (if Alby wallet is funded)**
1. Open https://satgate.io/playground
2. Click "Request Protected Resource"
3. Pay the invoice via Alby
4. Watch the access granted

**Option B: Just Show the UI**
1. Open https://satgate.io/playground
2. Show the interface without paying
3. Explain: "The agent would pay this 1-sat invoice and immediately receive access."

**Talk Track:**

> "This is what an AI agent sees. Request, pay, access. Sub-second. Sub-cent. No accounts. No friction."

---

### **Scene 8: The Phase 3 Close**

**Closing Line:**

> "Phase 1 gives you governance. Phase 3 gives you revenue. Same architecture. Same day deployment. Different config file."

---

# 📚 Reference Materials

## Quick Recovery Commands

```bash
# Health check
curl https://satgate-production.up.railway.app/health

# Test L402 (Phase 3) - should return 402
curl -i https://satgate-production.up.railway.app/api/micro/ping

# Test Capability (Phase 1) - should return 401
curl https://satgate-production.up.railway.app/api/capability/ping

# Mint a fresh token
curl -X POST https://satgate-production.up.railway.app/api/capability/mint
```

---

## The "Maturity Model" Reference

| Phase | What You Showed | Crypto Required | Value Prop |
|-------|-----------------|-----------------|------------|
| **Phase 1** | Scenes 1-4 (Capability tokens) | ❌ NO | Governance & Speed |
| **Phase 2** | Quota tracking (future) | ❌ NO | Cost Control |
| **Phase 3** | Scenes 5-8 (L402 payments) | ✅ YES | Revenue Generation |

---

## Objection Handlers

**"Is this just OAuth with extra steps?"**
> "OAuth tokens are opaque references to server-side state. These are self-describing, cryptographically verifiable capabilities. The token IS the permission—no database lookup required."

**"Why not just use JWTs?"**
> "JWTs can't be attenuated after issuance. With macaroons, an agent can mint a more restricted sub-token without calling us. That's the delegation superpower."

**"What about revocation?"**
> "Time-based expiry handles 90% of cases. For hard revocation, we maintain a small blocklist—but that's the exception, not the rule."

**"Why Lightning? Why not Stripe?"**
> "Stripe charges 2.9% + 30¢. That makes a 1-cent API call cost 31 cents. Lightning lets us charge 1 sat ($0.001) with near-zero fees. It's the only rail that makes micropayments viable."

**"Our clients won't use Bitcoin."**
> "That's exactly why we built Phase 1. Start with governance—no crypto required. When they're ready for monetization, the payment rail is already in place."

---

## Backup: Pre-Minted Tokens

```bash
# Generate backup tokens before the meeting:
curl -X POST https://satgate-production.up.railway.app/api/capability/mint > backup-token.json
```


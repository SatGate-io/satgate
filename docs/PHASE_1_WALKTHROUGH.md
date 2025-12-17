# SatGate Phase 1: Capability-Only Access Walkthrough

**A step-by-step demo script for presenting Zero Trust access control without cryptocurrency.**

This document provides the commands, talk tracks, and objection handlers for demonstrating SatGate's Phase 1 (Capability-Only) mode to enterprise clients and partners.

---

## Pre-Demo Checklist

- [ ] Terminal open with clean screen
- [ ] Railway deployment healthy (`curl https://satgate-production.up.railway.app/health`)
- [ ] `cli/delegation-demo.js` tested locally
- [ ] Backup tokens pre-minted (in case of network issues)

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

**Talk Track:**

> "Authenticated instantly. The Gateway validated the signature mathematically. No LDAP lookup. No bottleneck."

---

### **Scene 3: The "Superpower" (Offline Delegation)**

*Narrative:* "Now the Agent needs to delegate a task to a sub-worker. In the old world, this is a ticket. In our world, it's math."

**Command:**
```bash
# Run the delegation simulation
node cli/delegation-demo.js
```

**Talk Track (While script runs):**

> "Watch the '[NETWORK]' line. Zero requests. The agent is minting a restricted credential offline. It just cut a spare key for the janitor that only opens the basement and expires in 5 minutes. This is the Google-grade capability competitors can't touch."

---

### **Scene 3b: The "Security Proof" (Least Privilege)**

*Narrative:* "But here's the key question: Can the child token escalate privileges? Let's test it."

**Command 1: The BLOCKED Action (Negative Test)**
```bash
# Copy the CHILD token from delegation-demo.js output
# Try to mint a NEW token with it (should FAIL)
curl -X POST -H "Authorization: Bearer <PASTE_CHILD_TOKEN>" \
  https://satgate-production.up.railway.app/api/capability/mint
```

**Expected Output:**
```json
{
  "error": "Access Denied",
  "reason": "Scope violation: token has 'api:capability:ping', need 'api:capability:admin'",
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

### **Scene 4: The "Safe Landing" (Closing)**

*Narrative:* "Everything you just saw happened without a single satoshi of Bitcoin."

**Closing Line:**

> "Phase 1 is live. Phase 3 (Payments) is just a config change away."

---

## Backup: Pre-Minted Tokens

If the network is slow during the demo, use these pre-generated tokens:

```bash
# Generate backup tokens before the meeting:
curl -X POST https://satgate-production.up.railway.app/api/capability/mint > backup-token.json
```

---

## Quick Recovery Commands

```bash
# Health check
curl https://satgate-production.up.railway.app/health

# Test L402 (Phase 3) still works
curl -i https://satgate-production.up.railway.app/api/micro/ping
# Expected: 402 Payment Required

# Test Capability (Phase 1)
curl https://satgate-production.up.railway.app/api/capability/ping
# Expected: 401 Missing Token (proves auth is required)
```

---

## The "Maturity Model" Slide Reference

| Phase | What You Showed | Crypto Required |
|-------|-----------------|-----------------|
| **Phase 1** | Scenes 1-3 (Capability tokens) | ❌ NO |
| **Phase 2** | Quota tracking (future) | ❌ NO |
| **Phase 3** | L402 payments (`/api/micro/*`) | ✅ YES |

---

## Objection Handlers

**"Is this just OAuth with extra steps?"**
> "OAuth tokens are opaque references to server-side state. These are self-describing, cryptographically verifiable capabilities. The token IS the permission—no database lookup required."

**"Why not just use JWTs?"**
> "JWTs can't be attenuated after issuance. With macaroons, an agent can mint a more restricted sub-token without calling us. That's the delegation superpower."

**"What about revocation?"**
> "Time-based expiry handles 90% of cases. For hard revocation, we maintain a small blocklist—but that's the exception, not the rule."


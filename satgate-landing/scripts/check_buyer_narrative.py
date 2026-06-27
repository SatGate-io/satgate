#!/usr/bin/env python3
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]

TARGETS = [
    "app/page.tsx",
    "app/components/HomeClient.tsx",
    "app/policy-to-proof/page.tsx",
    "app/govern/page.tsx",
    "app/components/GovernClient.tsx",
    "app/build/page.tsx",
    "app/mcp-gateway/page.tsx",
    "app/pricing/page.tsx",
    "app/pricing/layout.tsx",
    "app/monetize/page.tsx",
    "app/pay/layout.tsx",
    "app/pay/page.tsx",
    "app/http-402-for-ai-agents/page.tsx",
    "app/l402-agent-payments/page.tsx",
    "app/paid-agent-payments/page.tsx",
    "app/robot-customer-payments/page.tsx",
    "app/agent-payment-controls/page.tsx",
    "app/agents/page.tsx",
    "app/agent-api-governance/page.tsx",
    "app/capability-auth/page.tsx",
    "app/economic-firewall-readiness-grader/page.tsx",
    "app/integrations/page.tsx",
    "app/blog/l402-protocol-explained/page.tsx",
    "app/blog/http-402-payment-required-use-cases/page.tsx",
    "app/l402-api-pricing-calculator/page.tsx",
    "app/stripe-link-agents-vs-satgate/page.tsx",
]

REQUIRED = {
    "app/page.tsx": [
        "Economic Firewall for Agentic API Access",
        "bounded economic authority",
        "humans, platforms, and upstream APIs can trust",
    ],
    "app/components/HomeClient.tsx": [
        "Humans and platforms set policy. Agents consume approved primitives.",
        "Humans and platforms buy. Agents consume bounded primitives.",
        "Economic Firewall for agentic API access",
    ],
    "app/policy-to-proof/page.tsx": [
        "From policy to proof for agentic API access.",
        "Define what an agent is allowed to do, enforce it at the gateway",
        "Bind agent actions to human or platform authority",
    ],
    "app/govern/page.tsx": [
        "bounded economic authority",
        "Humans and platforms set authority; agents consume bounded primitives; upstreams receive evidence.",
    ],
    "app/pricing/layout.tsx": [
        "Economic Firewall for AI Agents",
        "bounded agent authority",
    ],
    "app/build/page.tsx": [
        "Build Agents With Bounded Economic Authority",
        "Build agents with bounded economic authority",
        "Consume upstream with max budget",
        "enforce max budgets at the gateway before forwarding",
    ],
    "app/paid-agent-payments/page.tsx": [
        "Agents consume. Humans and platforms buy.",
        "Governed Paid API Access for Agents",
        "A human or platform delegates the authority",
    ],
    "app/robot-customer-payments/page.tsx": [
        "redirect('/paid-agent-payments')",
    ],
    "app/pay/layout.tsx": [
        "L402 Paid-Rail Governance Demo",
        "delegated paid API access",
    ],
    "app/l402-agent-payments/page.tsx": [
        "L402 Paid-Rail Governance",
        "delegated authority, budget, scope, and payment proof before access",
    ],
}

FORBIDDEN_PATTERNS = [
    r"agent as customer",
    r"agent customer",
    r"robot[- ]customer",
    r"agents? are the next API buyers",
    r"turn autonomous agents into paying API customers",
    r"wallet[- ]first",
    r"wallet-native",
    r"marketplace[- ]first",
    r"agent payment marketplace",
    r"payment marketplace",
    r"rail[- ]first",
    r"SatGate Charge",
    r"L402 Charge",
    r"Observe, Control, and Charge",
    r"Observe, Control, Charge",
    r"autonomous spend platform",
    r"agents buy",
    r"agents pay",
    r"AI agents pay",
    r"paid agents pay",
    r"agents can pay",
    r"agent pays",
    r"pay a Lightning invoice",
    r"API customers",
    r"new customer type",
    r"robot payments",
    r"What is a paid agent",
    r"worth paying for right now",
    r"payment rail for AI agents",
    r"wallet layer",
    r"wallets? for agents",
]

# Phrases we intentionally preserve only as high-intent SEO terms are not banned by
# themselves. The ban is on buyer/rail/wallet-first overclaims and agent-as-customer framing.

errors: list[str] = []
for rel in TARGETS:
    path = ROOT / rel
    if not path.exists():
        errors.append(f"missing target: {rel}")
        continue
    text = path.read_text()
    for needle in REQUIRED.get(rel, []):
        if needle not in text:
            errors.append(f"{rel} missing required thesis string: {needle}")
    for pattern in FORBIDDEN_PATTERNS:
        if re.search(pattern, text, flags=re.IGNORECASE):
            errors.append(f"{rel} matched stale framing: {pattern}")

if errors:
    print("buyer narrative regression check failed:")
    for err in errors:
        print(f"- {err}")
    sys.exit(1)

print("buyer narrative regression check passed")

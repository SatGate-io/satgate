#!/usr/bin/env python3
"""Copy contract only: no runtime, settlement or adoption certification."""
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = {
    "app/components/HomeClient.tsx": [
        "Economic firewall for agents", "An economic firewall",
        "for AI agents.", "Economic admission", "Economic control", "Admit",
        "Verifiable evidence for audit", "funded attackers",
        "Payment never overrides scope", "Proof across all three controls",
    ],
    "app/page.tsx": ["Economic Firewall for AI Agents", "internal budgets", "external access"],
    "app/economic-firewall/page.tsx": [
        "Economic defense is the foundation", "Fiat402", "internal budget control",
        "funded attackers", "upstream credentials", "admission costs",
        ">Admit</h2>", "Proof across Observe, Control and Admit",
    ],
    "app/build/page.tsx": [
        "Complete a useful task within an explicit budget", "Discover before installing",
        "Delegation, refusal and recovery", "same operation", "Request a bounded evaluation",
        "https://github.com/SatGate-io/satgate/tree/main/demo",
        "Synthetic HTTP demo", "issue/pay/verify API namespace is in private beta",
        "Private-beta examples", "pip install satgate", "npm install @satgate/sdk",
        "Test same-operation recovery and duplicate-spend handling in the selected deployment",
    ],
    "app/agent-authority-layer/page.tsx": [
        "Authorization and settlement are separate", "planned", "answer quality",
        "trusted issuer", "Payment cannot override",
    ],
    "public/llms.txt": [
        "### Observe", "### Control", "### Admit", "## Proof across all three controls",
        "Fiat402 is internal budget control, not a payment or settlement rail",
        "owner-enforced", "private beta", "same operation", "answer quality",
    ],
}
FORBIDDEN = [
    "legacy SEO/category term", "Rail-neutral is the moat",
    "Observe/Control/Prove", "Observe, Control, and Prove",
    "Fiat402 are paid rails", "whatever comes next", "SatGate fills it once",
    "without double spending",
]
# Keep implementation names out of the buyer-facing hero and control cards.
errors = []
home = (ROOT / "app/components/HomeClient.tsx").read_text()
hero = home.split("{/* Hero Section */}", 1)[1].split("</header>", 1)[0]
cards = home.split("{/* Your Agents */}", 1)[1].split("{/* Token Delegation Video */}", 1)[0]
for name, block in [("hero", hero), ("control cards", cards)]:
    for jargon in ["Fiat402", "L402", "above rails"]:
        if jargon in block:
            errors.append(f"homepage {name}: protocol-first wording {jargon!r}")

checks = 0
for rel, required in REQUIRED.items():
    text = (ROOT / rel).read_text()
    for needle in required:
        checks += 1
        if needle not in text:
            errors.append(f"{rel}: missing {needle!r}")
    for needle in FORBIDDEN:
        checks += 1
        if needle in text:
            errors.append(f"{rel}: stale claim {needle!r}")
if errors:
    print("Copy contract FAIL")
    print("\n".join(errors))
    sys.exit(1)
print(f"Copy contract PASS: {checks} checks across {len(REQUIRED)} source surfaces")

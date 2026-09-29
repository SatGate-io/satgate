#!/usr/bin/env python3
"""Copy contract only: no runtime, settlement or adoption certification."""
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = {
    "app/components/HomeClient.tsx": [
        "Economic firewall for agents", "An economic firewall",
        "for AI agents.", "External agents: access and payment", "Your agents: budgets", "Admit",
        "Receipts anyone can check", "makes abuse more expensive",
        "Paying never gets an agent past your access rules", "From Observe, Control and Admit",
    ],
    "app/page.tsx": ["Economic Firewall for AI Agents", "within budget", "external agents"],
    "app/economic-firewall/page.tsx": [
        "SatGate has three controls", "Fiat402", "internal budget control",
        "raises the cost of an attack", "keep your API keys behind it", "only way to reach your API",
        ">Admit</h2>", "Receipts from all three",
    ],
    "app/build/page.tsx": [
        "Give your agent a task and a budget", "Look before you install",
        "Handoffs, refusals and retries", "same call", "Talk to us about a pilot",
        "https://github.com/SatGate-io/satgate/tree/main/demo",
        "simulated HTTP demo", "issue, pay and verify API is in private beta",
        "Private-beta examples", "pip install satgate", "npm install @satgate/sdk",
        "Check that a retried call isn’t charged twice",
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

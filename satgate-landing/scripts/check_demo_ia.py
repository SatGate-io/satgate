#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
HOME = ROOT / "app" / "components" / "HomeClient.tsx"
PRICING = ROOT / "app" / "pricing" / "page.tsx"
SANDBOX = ROOT / "app" / "sandbox" / "page.tsx"
BACK_LINK_PAGES = [
    ROOT / "app" / "mcp-gateway" / "page.tsx",
    ROOT / "app" / "build" / "page.tsx",
    ROOT / "app" / "capability-auth" / "page.tsx",
    ROOT / "app" / "tools" / "page.tsx",
]

errors: list[str] = []

home = HOME.read_text()
if "Live Demo" in home:
    errors.append("homepage top nav still contains Live Demo")
if 'href="/sandbox"' not in home or "Open Sandbox" not in home:
    errors.append("homepage does not route demo CTA/nav to /sandbox")

pricing = PRICING.read_text()
for stale in ["Mint Demo", "Control Demo", "Charge Demo", 'href="/mint-demo"', 'href="/protect"', 'href="/pay"']:
    if stale in pricing:
        errors.append(f"pricing page still contains demo link/copy: {stale}")

sandbox = SANDBOX.read_text()
ordered = ["Mint Demo", "Capability Control Demo", "Spend Control Demo", "Paid-Rails Demo"]
positions = []
for label in ordered:
    pos = sandbox.find(label)
    if pos == -1:
        errors.append(f"sandbox missing demo card: {label}")
    positions.append(pos)
if all(pos != -1 for pos in positions) and positions != sorted(positions):
    errors.append("sandbox demo cards are not ordered Mint → Capability Control → Spend Control → Paid-Rails")
for href in ["/mint-demo", "/protect", "#spend-control-demo", "/pay"]:
    if href not in sandbox:
        errors.append(f"sandbox missing demo href: {href}")

for page in BACK_LINK_PAGES:
    text = page.read_text()
    if "Back to Home" not in text or 'href="/"' not in text:
        errors.append(f"{page.relative_to(ROOT)} missing Back to Home link")

if errors:
    print("demo IA regression check failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

print("demo IA regression check passed")

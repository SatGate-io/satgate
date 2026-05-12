#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
HOME = ROOT / "app" / "components" / "HomeClient.tsx"
PRICING = ROOT / "app" / "pricing" / "page.tsx"
SANDBOX = ROOT / "app" / "sandbox" / "page.tsx"
SANDBOX_LAYOUT = ROOT / "app" / "sandbox" / "layout.tsx"
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
if "Open Sandbox" in home or ">Sandbox<" in home:
    errors.append("homepage still exposes Sandbox as public label")
if 'href="/sandbox"' not in home or "See Demo" not in home or ">Demo</Link>" not in home:
    errors.append("homepage does not route Demo nav/CTA to /sandbox")
for crowded in ["Capability Auth", "Control Plane", "Tools", "Integrations", "Blog", "GitHub"]:
    desktop_marker = f'>{crowded}</Link>' if crowded != "GitHub" else f'>{crowded}</a>'
    if desktop_marker in home.split("{/* Mobile menu button */}", 1)[0]:
        errors.append(f"desktop nav still includes crowded link: {crowded}")

pricing = PRICING.read_text()
for stale in ["Mint Demo", "Control Demo", "Charge Demo", 'href="/mint-demo"', 'href="/protect"', 'href="/pay"']:
    if stale in pricing:
        errors.append(f"pricing page still contains demo link/copy: {stale}")

sandbox = SANDBOX.read_text()
layout = SANDBOX_LAYOUT.read_text()
for required in ["SatGate Demo", "Interactive demo"]:
    if required not in sandbox and required not in layout:
        errors.append(f"demo page missing required public label: {required}")
for stale in ["SatGate Sandbox", "Demo path", "SatGate sandbox questions", "What does the SatGate sandbox", "Is the sandbox"]:
    if stale in sandbox or stale in layout:
        errors.append(f"demo page still contains stale sandbox label: {stale}")
ordered = ["Mint Demo", "Capability Control Demo", "Spend Control Demo", "Paid-Rails Demo"]
positions = []
for label in ordered:
    pos = sandbox.find(label)
    if pos == -1:
        errors.append(f"sandbox route missing demo card: {label}")
    positions.append(pos)
if all(pos != -1 for pos in positions) and positions != sorted(positions):
    errors.append("demo cards are not ordered Mint → Capability Control → Spend Control → Paid-Rails")
for href in ["/mint-demo", "/protect", "#spend-control-demo", "/pay"]:
    if href not in sandbox:
        errors.append(f"sandbox route missing demo href: {href}")

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

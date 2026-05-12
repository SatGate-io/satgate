#!/usr/bin/env python3
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
PAGE = ROOT / "app" / "build" / "page.tsx"
SITEMAP = ROOT / "app" / "sitemap.ts"
HOME = ROOT / "app" / "components" / "HomeClient.tsx"

required_page_strings = [
    "Build agents that can spend safely",
    "Capabilities in. Receipts out. Rails abstracted.",
    "Issue scoped capabilities",
    "Route paid calls",
    "Verify receipts",
    "satgate.issue",
    "satgate.pay",
    "satgate.verify",
    "import os",
    "os.getenv(\"SATGATE_API_KEY\")",
    "Economic Firewall for AI agents",
    "authority and evidence layer",
    "https://cloud.satgate.io/docs",
    "pip install satgate",
    "npm install @satgate/sdk",
    "issue/pay/verify API namespace is in private beta",
    "# Install today (public packages):",
    "Request access →",
    "Works with:",
    "OpenAI tools",
    "Anthropic tools",
    "Raw HTTP",
    "decision: \"denied\"",
    "budget_exhausted",
    "Node example",
    "HTTP example",
    "YOUR_API_KEY",
]

forbidden_page_patterns = [
    r"robot[- ]customer",
    r"SatGate Charge",
    r"L402 Charge",
    r"agent payment marketplace",
    r"wallet[- ]first",
    r"wallet-native",
    r"when APIs become products",
    r"autonomous spend platform",
    r"See proof model",
    r"ISO 8601 duration support belongs",
    r"Open developer docs",
    r"Bearer \*\*\*",
]

errors: list[str] = []
if not PAGE.exists():
    errors.append("missing app/build/page.tsx")
else:
    text = PAGE.read_text()
    for needle in required_page_strings:
        if needle not in text:
            errors.append(f"missing required /build string: {needle}")
    for pattern in forbidden_page_patterns:
        if re.search(pattern, text, flags=re.IGNORECASE):
            errors.append(f"forbidden /build framing matched: {pattern}")

sitemap = SITEMAP.read_text() if SITEMAP.exists() else ""
if "path: '/build'" not in sitemap:
    errors.append("/build missing from app/sitemap.ts")

home = HOME.read_text() if HOME.exists() else ""
if 'href="/build"' not in home:
    errors.append("homepage navigation/CTA does not link to /build")

if errors:
    print("/build regression check failed:")
    for err in errors:
        print(f"- {err}")
    sys.exit(1)

print("/build regression check passed")

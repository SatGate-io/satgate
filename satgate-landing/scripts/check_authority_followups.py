#!/usr/bin/env python3
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
APP = ROOT / "app"
HOME = APP / "components" / "HomeClient.tsx"
SITEMAP = APP / "sitemap.ts"
RAILS = APP / "partners" / "rails" / "page.tsx"
AUTHORITY = APP / "agent-authority-layer" / "page.tsx"
PDF = ROOT / "public" / "briefs" / "satgate-agent-authority-rails-brief.pdf"

errors: list[str] = []

required_files = [RAILS, PDF]
for path in required_files:
    if not path.exists():
        errors.append(f"missing required artifact: {path.relative_to(ROOT)}")

if RAILS.exists():
    rails = RAILS.read_text()
    for needle in [
        "Agent authority for every payment rail.",
        "Economic Firewall category stays durable while rails change",
        "Agent Authority & Accountability Layer",
        "Policy-to-Proof",
        "Evidence Pack",
        "https://github.com/SatGate-io/evidence-pack-verifier",
        "/briefs/satgate-agent-authority-rails-brief.pdf",
    ]:
        if needle not in rails:
            errors.append(f"/partners/rails missing anchor: {needle}")

sitemap = SITEMAP.read_text() if SITEMAP.exists() else ""
if "path: '/partners/rails'" not in sitemap:
    errors.append("/partners/rails missing from sitemap")

home = HOME.read_text() if HOME.exists() else ""
if 'href="/partners/rails"' not in home:
    errors.append("homepage footer does not link to /partners/rails")
if "Authority Layer" in home:
    errors.append("homepage still uses short stale label: Authority Layer")

if AUTHORITY.exists():
    authority = AUTHORITY.read_text()
    if "/partners/rails" not in authority:
        errors.append("/agent-authority-layer does not link to /partners/rails")

public_tsx = "\n".join(path.read_text(errors="ignore") for path in APP.rglob("*.tsx"))
for pattern in [
    r"economic control plane",
    r"Authority Layer",
    r"\bAgent Authority Layer\b",
    r"robot[- ]customer",
    r"SatGate Charge",
    r"L402 Charge",
    r"production-ready",
    r"Gemma4",
    r"buyers need to see",
    r"credible demo",
]:
    if re.search(pattern, public_tsx, flags=re.IGNORECASE):
        errors.append(f"forbidden public framing matched: {pattern}")

if PDF.exists():
    head = PDF.read_bytes()[:5]
    if head != b"%PDF-":
        errors.append("partner brief is not a PDF")

if errors:
    print("authority follow-up check failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

print("authority follow-up check passed")

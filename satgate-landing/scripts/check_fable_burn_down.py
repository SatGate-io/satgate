#!/usr/bin/env python3
from __future__ import annotations

from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]

failures: list[str] = []

def read(rel: str) -> str:
    return (ROOT / rel).read_text()

home = read('app/page.tsx')
pricing = read('app/pricing/page.tsx')
robots = read('app/robots.ts')

canonical_org_nodes = home.count("logo: 'https://satgate.io/logo_white_transparent.png'")
if canonical_org_nodes != 1:
    failures.append('homepage JSON-LD must contain exactly one canonical Organization node with logo in app/page.tsx')
if "description: 'SatGate is the Agent Authority & Accountability Layer" not in home:
    failures.append('homepage Organization JSON-LD must use the current Agent Authority description')
for stale in ['Evidence Packs after every decision', 'Evidence Pack receipts after every decision']:
    if stale in home:
        failures.append(f'homepage contains overbroad proof claim: {stale}')

for stale in ['robot customer', 'Trust-as-a-Service for enterprise deals', 'L402 — cryptographic proof-of-budget']:
    if stale.lower() in pricing.lower():
        failures.append(f'pricing still contains stale paid-rail/product-center phrase: {stale}')

if robots.count("userAgent: '*'") != 1 or robots.count("sitemap: 'https://satgate.io/sitemap.xml'") != 1:
    failures.append('robots.ts must define one canonical allow/disallow block and one sitemap')

if failures:
    print('FABLE_LANDING_GUARD_FAIL')
    for failure in failures:
        print(f'- {failure}')
    raise SystemExit(1)

print('FABLE_LANDING_GUARD_PASS')

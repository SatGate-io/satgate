#!/usr/bin/env python3
"""Regression guard for SatGate /.well-known/satgate trust metadata."""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ROUTE = ROOT / "app" / ".well-known" / "satgate" / "route.ts"
BUILD = ROOT / "app" / "build" / "page.tsx"
DOC = ROOT.parent / "docs" / "reference" / "satgate-trust-metadata.md"
DOC_INDEX = ROOT.parent / "docs" / "index.md"
PACKAGE = ROOT / "package.json"

REQUIRED_ROUTE_STRINGS = [
    "schema_version: \"satgate.trust_metadata.v1\"",
    "metadata_url: \"https://satgate.io/.well-known/satgate\"",
    "capability_acceptance",
    "accepted_formats",
    "satgate.capability.v1",
    "macaroon-bearer",
    "receipt_verification",
    "satgate.receipt.v1",
    "satgate.evidence_pack.v1",
    "receipt_id",
    "evidence_pack_id",
    "decision_reason",
    "policy_version",
    "rails_adapters",
    "x402",
    "l402",
    "api_key_billing",
    "enterprise_ledger",
    "signing_key_discovery",
    "jwks_url_template",
    "Response.json",
    "Cache-Control",
    "public, max-age=3600",
    "Access-Control-Allow-Origin",
    "X-Content-Type-Options",
]

REQUIRED_BUILD_STRINGS = [
    "Trust metadata",
    "/.well-known/satgate",
    "capability acceptance",
    "receipt verification fields",
]

REQUIRED_DOC_STRINGS = [
    "https://satgate.io/.well-known/satgate",
    "satgate.trust_metadata.v1",
    "Capability acceptance",
    "Receipt verification",
    "Signing key discovery",
    "200 OK",
    "Content-Type: application/json",
]

FORBIDDEN_ROUTE_PATTERNS = [
    r"marketplace",
    r"reputation",
    r"endorsement",
    r"rank(ing)?",
    r"trust score",
]

errors: list[str] = []

if not ROUTE.exists():
    errors.append("missing /.well-known/satgate route")
else:
    route_text = ROUTE.read_text()
    for needle in REQUIRED_ROUTE_STRINGS:
        if needle not in route_text:
            errors.append(f"route missing required string: {needle}")
    for pattern in FORBIDDEN_ROUTE_PATTERNS:
        # Permit the explicit disclaimer phrase "no marketplace or reputation claim".
        text_without_disclaimer = route_text.replace("no marketplace or reputation claim", "")
        if re.search(pattern, text_without_disclaimer, flags=re.IGNORECASE):
            errors.append(f"route has forbidden claim language: {pattern}")

build_text = BUILD.read_text() if BUILD.exists() else ""
for needle in REQUIRED_BUILD_STRINGS:
    if needle not in build_text:
        errors.append(f"/build missing trust metadata string: {needle}")

doc_text = DOC.read_text() if DOC.exists() else ""
for needle in REQUIRED_DOC_STRINGS:
    if needle not in doc_text:
        errors.append(f"docs missing trust metadata string: {needle}")

index_text = DOC_INDEX.read_text() if DOC_INDEX.exists() else ""
if "reference/satgate-trust-metadata.md" not in index_text:
    errors.append("docs index does not link satgate-trust-metadata reference")

package_text = PACKAGE.read_text() if PACKAGE.exists() else ""
if "test:well-known" not in package_text:
    errors.append("package.json missing test:well-known script")

if errors:
    print("/.well-known/satgate regression check failed:")
    for err in errors:
        print(f"- {err}")
    sys.exit(1)

print("/.well-known/satgate regression check passed")

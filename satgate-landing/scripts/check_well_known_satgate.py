#!/usr/bin/env python3
"""Regression guard for SatGate /.well-known/satgate trust metadata."""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ROUTE = ROOT / "app" / ".well-known" / "satgate" / "route.ts"
SCHEMA_ROUTE = ROOT / "app" / ".well-known" / "satgate.schema.json" / "route.ts"
JWKS_ROUTE = ROOT / "app" / ".well-known" / "jwks.json" / "route.ts"
BUILD = ROOT / "app" / "build" / "page.tsx"
DOC = ROOT.parent / "docs" / "reference" / "satgate-trust-metadata.md"
ACCEPTOR_DOC = ROOT.parent / "docs" / "reference" / "acceptor.md"
DOC_INDEX = ROOT.parent / "docs" / "index.md"
PACKAGE = ROOT / "package.json"

REQUIRED_ROUTE_STRINGS = [
    "schema_version: \"satgate.trust_metadata.v1\"",
    "metadata_url: \"https://satgate.io/.well-known/satgate\"",
    "schema_url: \"https://satgate.io/.well-known/satgate.schema.json\"",
    "roles: [\"issuer\"]",
    "capability_acceptance",
    "accepted_formats",
    "satgate.capability.v1",
    "macaroon-bearer",
    "receipt_verification",
    "satgate.receipt.v1",
    "satgate.evidence_pack.v1",
    "receipt_id",
    "evidence_pack_id",
    "issuer",
    "issuer_kid",
    "decision_reason",
    "policy_version",
    "rails_adapters",
    "x402",
    "l402",
    "api_key_billing",
    "enterprise_ledger",
    "agentcore_payments",
    "pay_sh",
    "status: \"supported\"",
    "status: \"planned\"",
    "type: \"payment_rail\"",
    "role: \"external_paid_access\"",
    "key_discovery",
    "jwks_uri: \"https://satgate.io/.well-known/jwks.json\"",
    "signing_key_discovery",
    "jwks_uri_template",
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
    "canonical for rail adapter status",
]

REQUIRED_DOC_STRINGS = [
    "https://satgate.io/.well-known/satgate",
    "satgate.trust_metadata.v1",
    "Capability acceptance",
    "Receipt verification",
    "Signing key discovery",
    "200 OK",
    "Content-Type: application/json",
    "schema_url",
    "jwks_uri",
    "array of objects",
    "Reference verifier snippets",
    "Acceptor Metadata Draft",
    "agentcore_payments",
    "pay_sh",
    "TRUSTED_ISSUERS",
    "untrusted receipt issuer",
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
        if re.search(pattern, route_text, flags=re.IGNORECASE):
            errors.append(f"route has forbidden claim language: {pattern}")
    if re.search(r"\bnote\s*:", route_text):
        errors.append("route should not contain prose-only note fields; keep human explanations in docs")
    if '"issuer_kid"' not in route_text or '"issuer"' not in route_text:
        errors.append("route required_receipt_fields must include issuer and issuer_kid")
    for planned_id in ["agentcore_payments", "pay_sh"]:
        planned_pattern = rf'{{ id: "{planned_id}"[^}}]+status: "planned"'
        if not re.search(planned_pattern, route_text):
            errors.append(f"route must mark {planned_id} as planned, not supported")


for path, label in [(SCHEMA_ROUTE, "schema route"), (JWKS_ROUTE, "JWKS route")]:
    if not path.exists():
        errors.append(f"missing {label}: {path.relative_to(ROOT)}")
    else:
        text = path.read_text()
        for needle in ["Response.json", "Cache-Control", "Access-Control-Allow-Origin", "X-Content-Type-Options"]:
            if needle not in text:
                errors.append(f"{label} missing required HTTP/header string: {needle}")

if SCHEMA_ROUTE.exists():
    schema_text = SCHEMA_ROUTE.read_text()
    for needle in ["SatGate Trust Metadata v1", "description", "schema_version", "roles", "uniqueItems", "required_receipt_fields", "issuer_kid", "allOf", "contains", "rails_adapters", "supported", "id", "type", "role", "status", "planned", "public, max-age=86400"]:
        if needle not in schema_text:
            errors.append(f"schema route missing required schema string: {needle}")

if JWKS_ROUTE.exists():
    jwks_text = JWKS_ROUTE.read_text()
    if "keys: []" not in jwks_text:
        errors.append("JWKS route should currently expose an empty keys array rather than placeholder signing keys")
    if re.search(r"\bnote\s*:", jwks_text):
        errors.append("JWKS route should not contain prose-only note fields")

build_text = BUILD.read_text() if BUILD.exists() else ""
for needle in REQUIRED_BUILD_STRINGS:
    if needle not in build_text:
        errors.append(f"/build missing trust metadata string: {needle}")
if "governs MCP tools, REST APIs, API-key billing, x402, L402, and enterprise ledgers today" not in build_text:
    errors.append("/build should separate supported rails from planned rails")
if "planned rails such as AgentCore Payments and Pay.sh" not in build_text:
    errors.append("/build should call AgentCore Payments and Pay.sh planned rails")

doc_text = DOC.read_text() if DOC.exists() else ""
for needle in REQUIRED_DOC_STRINGS:
    if needle not in doc_text:
        errors.append(f"docs missing trust metadata string: {needle}")

acceptor_text = ACCEPTOR_DOC.read_text() if ACCEPTOR_DOC.exists() else ""
for needle in ["v0.0 draft", "verification_endpoint", "accepted_capability_formats", "recognized_receipt_decisions", "emitted_receipt_decisions", "trust_anchors", "rails_adapters", "satgate-acceptor.schema.json"]:
    if needle not in acceptor_text:
        errors.append(f"acceptor draft missing required string: {needle}")

index_text = DOC_INDEX.read_text() if DOC_INDEX.exists() else ""
if "reference/satgate-trust-metadata.md" not in index_text:
    errors.append("docs index does not link satgate-trust-metadata reference")
if "reference/acceptor.md" not in index_text:
    errors.append("docs index does not link acceptor metadata draft")

package_text = PACKAGE.read_text() if PACKAGE.exists() else ""
if "test:well-known" not in package_text:
    errors.append("package.json missing test:well-known script")

if errors:
    print("/.well-known/satgate regression check failed:")
    for err in errors:
        print(f"- {err}")
    sys.exit(1)

print("/.well-known/satgate regression check passed")

#!/usr/bin/env python3
"""Regression guard for the SatGate upstream acceptance story."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parent
PAGE = ROOT / "app" / "accept-satgate-capabilities" / "page.tsx"
DOC = REPO / "docs" / "reference" / "accept-satgate-capabilities.md"
ACCEPTOR_DOC = REPO / "docs" / "reference" / "acceptor.md"
DOC_INDEX = REPO / "docs" / "index.md"
SITEMAP = ROOT / "app" / "sitemap.ts"
BUILD = ROOT / "app" / "build" / "page.tsx"
SCHEMA_ROUTE = ROOT / "app" / ".well-known" / "satgate-acceptor.schema.json" / "route.ts"
MOCK_METADATA = ROOT / "public" / "examples" / "mock-acceptor-metadata.v0.json"
MOCK_RECEIPT = ROOT / "public" / "examples" / "mock-accepted-satgate-receipt.v1.json"

errors: list[str] = []

for path, label in [
    (PAGE, "public acceptance page"),
    (DOC, "acceptance docs page"),
    (ACCEPTOR_DOC, "acceptor metadata draft"),
    (SCHEMA_ROUTE, "acceptor schema route"),
    (MOCK_METADATA, "mock acceptor metadata"),
    (MOCK_RECEIPT, "mock accepted receipt"),
]:
    if not path.exists():
        errors.append(f"missing {label}: {path}")

page_text = PAGE.read_text() if PAGE.exists() else ""
doc_text = DOC.read_text() if DOC.exists() else ""
acceptor_doc_text = ACCEPTOR_DOC.read_text() if ACCEPTOR_DOC.exists() else ""
schema_text = SCHEMA_ROUTE.read_text() if SCHEMA_ROUTE.exists() else ""
combined = page_text + "\n" + doc_text + "\n" + acceptor_doc_text + "\n" + schema_text

required_strings = [
    "Accepts SatGate capabilities",
    "capability verification and receipt emission",
    "not a marketplace",
    "No marketplace",
    "not mean SatGate endorses",
    "network-wide reputation",
    "Minimal integration checklist",
    "Verification criteria",
    "mock acceptor",
    "internal/mock",
    "Authorization: Bearer <capability>",
    "trust_anchors",
    "issuer_kid",
    "receipt_id",
    "evidence_pack_id",
    "acceptor_id",
    "recognized_receipt_decisions",
    "emitted_receipt_decisions",
    "emitted_receipt_fields",
    "satgate-acceptor.schema.json",
    "accepted_for_mock",
    "provisional",
    "revoked",
    "deprecated",
    "issuer-side authority lifecycle primitives",
    "issuers can **support** rails",
    "acceptors **accept** rails",
    "decision_reason",
    "policy_version",
]
for needle in required_strings:
    if needle not in combined:
        errors.append(f"acceptance story missing required string: {needle}")

for stale in ["accepted_receipt_decisions", "allowed, denied, paid, delegated, or revoked"]:
    if stale in combined:
        errors.append(f"stale ambiguous acceptor field remains: {stale}")

# These are permitted only inside the explicit forbidden-copy section.
for overclaim in [
    "SatGate trusted marketplace member",
    "SatGate certified reputation score",
    "SatGate endorsed API",
    "Network-approved upstream",
]:
    if overclaim not in combined:
        errors.append(f"forbidden-copy example missing from guard/docs: {overclaim}")

if "global trust" not in combined or "ranking" not in combined:
    errors.append("acceptance story must explicitly reject global trust/ranking claims")

if SITEMAP.exists() and "/accept-satgate-capabilities" not in SITEMAP.read_text():
    errors.append("sitemap missing /accept-satgate-capabilities")
if DOC_INDEX.exists() and "reference/accept-satgate-capabilities.md" not in DOC_INDEX.read_text():
    errors.append("docs index missing acceptance docs link")
if ACCEPTOR_DOC.exists() and "accept-satgate-capabilities.md" not in ACCEPTOR_DOC.read_text():
    errors.append("acceptor draft does not link acceptance story")
if BUILD.exists() and "/accept-satgate-capabilities" not in BUILD.read_text():
    errors.append("/build does not link upstream acceptance story")

schema_match = re.search(r"const schema = (\{.*?\}) as const;", schema_text, re.S) if schema_text else None
if not schema_match:
    errors.append("acceptor schema route does not define const schema")
else:
    # Avoid TypeScript parsing; assert the closed enum strings we care about are present.
    for enum_value in [
        "satgate.acceptor_metadata.v0",
        "internal_mock_only",
        "active",
        "accepted",
        "provisional",
        "revoked",
        "deprecated",
        "accepted_for_mock",
        "recognized_receipt_decisions",
        "emitted_receipt_decisions",
        "emitted_receipt_fields",
        "capability verification and receipt emission",
    ]:
        if enum_value not in schema_text:
            errors.append(f"acceptor schema missing enum/field: {enum_value}")

if MOCK_METADATA.exists():
    metadata = json.loads(MOCK_METADATA.read_text())
    if metadata.get("schema_url") != "https://satgate.io/.well-known/satgate-acceptor.schema.json":
        errors.append("mock acceptor metadata must point schema_url at v0 acceptor schema")
    if metadata.get("status") != "internal_mock_only":
        errors.append("mock acceptor metadata must be labeled internal_mock_only")
    if metadata.get("roles") != ["acceptor"]:
        errors.append("mock acceptor metadata must use roles: ['acceptor']")
    if "trust_anchors" not in metadata:
        errors.append("mock acceptor metadata missing trust_anchors")
    for anchor in metadata.get("trust_anchors", []):
        if anchor.get("status") not in {"accepted", "provisional", "revoked", "deprecated", "accepted_for_mock"}:
            errors.append(f"trust anchor has undefined status enum: {anchor.get('status')}")
    claims = metadata.get("claims", {})
    if claims.get("acceptance_means") != "capability verification and receipt emission":
        errors.append("mock acceptor metadata must define honest acceptance_means")
    for forbidden in ["marketplace listing", "reputation score", "SatGate endorsement", "network-wide trust", "ranking", "certification"]:
        if forbidden not in claims.get("acceptance_does_not_mean", []):
            errors.append(f"mock acceptor metadata missing negative claim: {forbidden}")
    if "accepted_receipt_decisions" in metadata:
        errors.append("mock acceptor metadata must not use ambiguous accepted_receipt_decisions")
    if "denied" in metadata.get("recognized_receipt_decisions", []):
        errors.append("mock acceptor metadata must not treat denied as recognized evidence for entry")
    if "denied" not in metadata.get("emitted_receipt_decisions", []):
        errors.append("mock acceptor metadata should list denied under emitted_receipt_decisions")
    emitted_fields = metadata.get("emitted_receipt_fields", [])
    for field in ["schema_version", "receipt_id", "acceptor_id", "decision", "receipt_hash", "signature", "task_id", "budget_id", "event_history_ref"]:
        if field not in emitted_fields:
            errors.append(f"mock acceptor metadata emitted_receipt_fields missing: {field}")

if MOCK_RECEIPT.exists():
    receipt = json.loads(MOCK_RECEIPT.read_text())
    for field in ["receipt_id", "evidence_pack_id", "issuer", "issuer_kid", "acceptor_id", "decision", "decision_reason", "policy_version", "receipt_hash", "signature", "mock_only"]:
        if field not in receipt:
            errors.append(f"mock receipt missing field: {field}")
    if receipt.get("mock_only") is not True:
        errors.append("mock receipt must be labeled mock_only: true")

if errors:
    print("acceptance story regression check failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

print("acceptance story regression check passed")

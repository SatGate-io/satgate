#!/usr/bin/env python3
"""Regression guard for the SatGate upstream acceptance story."""
from __future__ import annotations

import json
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
MOCK_METADATA = ROOT / "public" / "examples" / "mock-acceptor-metadata.v0.json"
MOCK_RECEIPT = ROOT / "public" / "examples" / "mock-accepted-satgate-receipt.v1.json"

errors: list[str] = []

for path, label in [
    (PAGE, "public acceptance page"),
    (DOC, "acceptance docs page"),
    (MOCK_METADATA, "mock acceptor metadata"),
    (MOCK_RECEIPT, "mock accepted receipt"),
]:
    if not path.exists():
        errors.append(f"missing {label}: {path}")

page_text = PAGE.read_text() if PAGE.exists() else ""
doc_text = DOC.read_text() if DOC.exists() else ""
combined = page_text + "\n" + doc_text

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
    "emitted_receipt_decisions",
    "decision_reason",
    "policy_version",
]
for needle in required_strings:
    if needle not in combined:
        errors.append(f"acceptance story missing required string: {needle}")

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

if MOCK_METADATA.exists():
    metadata = json.loads(MOCK_METADATA.read_text())
    if metadata.get("status") != "internal_mock_only":
        errors.append("mock acceptor metadata must be labeled internal_mock_only")
    if metadata.get("roles") != ["acceptor"]:
        errors.append("mock acceptor metadata must use roles: ['acceptor']")
    if "trust_anchors" not in metadata:
        errors.append("mock acceptor metadata missing trust_anchors")
    claims = metadata.get("claims", {})
    if claims.get("acceptance_means") != "capability verification and receipt emission":
        errors.append("mock acceptor metadata must define honest acceptance_means")
    for forbidden in ["marketplace listing", "reputation score", "SatGate endorsement", "network-wide trust"]:
        if forbidden not in claims.get("acceptance_does_not_mean", []):
            errors.append(f"mock acceptor metadata missing negative claim: {forbidden}")
    if "denied" in metadata.get("accepted_receipt_decisions", []):
        errors.append("mock acceptor metadata must not treat denied as accepted evidence for entry")
    if "denied" not in metadata.get("emitted_receipt_decisions", []):
        errors.append("mock acceptor metadata should list denied under emitted_receipt_decisions")

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

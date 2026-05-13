#!/usr/bin/env python3
"""Regression guard for reputation substrate without public reputation launch."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parent
RECEIPT_SCHEMA = ROOT / "schemas" / "satgate-receipt.schema.json"
EVIDENCE_SCHEMA = ROOT / "public" / "evidence-packs" / "evidence-pack.schema.v1.json"
SUBSTRATE_DOC = REPO / "docs" / "reference" / "reputation-substrate.md"
RECEIPT_DOC = REPO / "docs" / "reference" / "receipt-schema.md"
EVIDENCE_DOC = REPO / "docs" / "reference" / "evidence-pack.md"
DOC_INDEX = REPO / "docs" / "index.md"
PUBLIC_ROOTS = [ROOT / "app", ROOT / "public"]

errors: list[str] = []

for path, label in [
    (RECEIPT_SCHEMA, "receipt schema"),
    (EVIDENCE_SCHEMA, "evidence pack schema"),
    (SUBSTRATE_DOC, "reputation substrate doc"),
    (RECEIPT_DOC, "receipt schema doc"),
    (DOC_INDEX, "docs index"),
]:
    if not path.exists():
        errors.append(f"missing {label}: {path}")

if errors:
    print("reputation substrate check failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

receipt_schema = json.loads(RECEIPT_SCHEMA.read_text())
evidence_schema = json.loads(EVIDENCE_SCHEMA.read_text())
substrate_text = SUBSTRATE_DOC.read_text()
receipt_doc_text = RECEIPT_DOC.read_text()
evidence_doc_text = EVIDENCE_DOC.read_text() if EVIDENCE_DOC.exists() else ""
index_text = DOC_INDEX.read_text()
combined_reference = "\n".join([substrate_text, receipt_doc_text, evidence_doc_text, index_text])

# Required inventory questions and principle.
for needle in [
    "stayed within budget",
    "completed requested task",
    "looped or retried excessively",
    "was revoked or attenuated",
    "was vouched by a principal",
    "was accepted by upstream",
    "reputation falls out of signed receipts",
    "SatGate should not mint reputation by assertion",
    "No receipts, no score",
    "No observed history, no ranking",
]:
    if needle not in combined_reference:
        errors.append(f"missing reputation-substrate principle/inventory phrase: {needle}")

# Optional receipt field names must be stable and non-required.
optional_receipt_fields = [
    "task_id",
    "task_status",
    "attempt",
    "max_attempts",
    "retry_of_receipt_id",
    "parent_receipt_id",
    "budget_id",
    "budget_limit_usd",
    "principal_id",
    "principal_authorization_id",
    "vouch_receipt_id",
    "attenuation_depth",
    "caveats_hash",
    "revoked_receipt_id",
    "event_history_ref",
]
receipt_required = set(receipt_schema.get("required", []))
receipt_properties = receipt_schema.get("properties", {})
for field in optional_receipt_fields:
    if field not in receipt_properties:
        errors.append(f"receipt schema missing optional substrate field: {field}")
    if field in receipt_required:
        errors.append(f"substrate field must stay optional in receipt v1: {field}")
    if field not in receipt_doc_text or field not in substrate_text:
        errors.append(f"docs missing optional substrate field: {field}")

# Evidence Pack event history must be optional and expressive enough for verifier history.
evidence_required = set(evidence_schema.get("required", []))
if "event_history" not in evidence_schema.get("properties", {}):
    errors.append("evidence pack schema missing optional event_history")
if "event_history" in evidence_required:
    errors.append("event_history must remain optional in Evidence Pack v1")
event_def = evidence_schema.get("$defs", {}).get("event", {})
event_types = event_def.get("properties", {}).get("event_type", {}).get("enum", [])
for event_type in [
    "task_requested",
    "task_started",
    "task_completed",
    "task_failed",
    "attempt",
    "retry",
    "budget_observed",
    "budget_denied",
    "delegated",
    "attenuated",
    "revoked",
    "principal_vouched",
    "upstream_accepted",
    "upstream_denied",
]:
    if event_type not in event_types:
        errors.append(f"event history enum missing: {event_type}")

def _has_boundary_context(text: str, start: int, end: int) -> bool:
    window = text[max(0, start - 500): min(len(text), end + 500)]
    return any(
        marker in window
        for marker in [
            "acceptance_does_not_mean",
            "Forbidden badge copy",
            "What the badge does not mean",
            "does **not** mean",
            "does not mean",
            "must not claim",
            "Forbidden in public launch copy",
            "Non-goals",
            "overclaim",
        ]
    )

# Keep public launch language out of app/public surfaces except explicit anti-claim examples.
public_forbidden = [
    r"SatGate reputation score",
    r"trust score",
    r"ranked upstreams",
    r"certified acceptor",
    r"trusted marketplace",
    r"SatGate endorsed",
    r"network-wide reputation",
]
for root in PUBLIC_ROOTS:
    for path in root.rglob("*"):
        if not path.is_file() or path.suffix.lower() not in {".ts", ".tsx", ".json", ".md", ".txt"}:
            continue
        text = path.read_text(errors="ignore")
        for pattern in public_forbidden:
            for match in re.finditer(pattern, text, flags=re.IGNORECASE):
                if not _has_boundary_context(text, match.start(), match.end()):
                    errors.append(f"public surface has forbidden reputation launch phrase {pattern}: {path.relative_to(ROOT)}")

for pattern in public_forbidden:
    # Explicit docs may mention forbidden phrases only as boundaries.
    if not re.search(pattern, substrate_text, flags=re.IGNORECASE):
        errors.append(f"substrate doc missing forbidden-boundary phrase: {pattern}")

if "reference/reputation-substrate.md" not in index_text:
    errors.append("docs index missing reputation substrate reference")

if errors:
    print("reputation substrate check failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

print("reputation substrate check passed")

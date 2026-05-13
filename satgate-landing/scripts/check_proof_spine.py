#!/usr/bin/env python3
"""Guard the SatGate receipts/Evidence Packs proof spine.

This is intentionally targeted at the surfaces that tend to drift back to
"logs/analytics/telemetry" framing. It does not ban operational logs; it checks
that the buyer/developer proof artifacts contain the canonical fields and that
highest-impact pages say receipts/Evidence Packs where decisions are described.
"""
from __future__ import annotations

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parent

REQUIRED_PHRASES = {
    "app/page.tsx": [
        "decision receipts",
        "paid-call receipts",
        "Evidence Pack proof",
    ],
    "app/components/HomeClient.tsx": [
        "Receipts for allowed, denied, paid, delegated, and revoked decisions",
        "Allowed receipts",
        "Paid receipts",
        "Export Evidence Pack",
    ],
    "app/pricing/page.tsx": [
        "receipt capture",
        "Real-time receipt dashboard",
        "Evidence Pack compliance exports",
        "Signed receipts + Evidence Pack export",
    ],
    "app/pay/page.tsx": [
        "paid-call receipts",
        "Receipt Returned",
        "Paid-call receipt queued for Evidence Pack",
        "receipt_id",
    ],
    "app/dashboard/page.tsx": [
        "Receipt-backed Governance Dashboard",
        "receipt-backed decisions",
        "Evidence Packs are the exportable proof artifact",
        "Denied-call receipts",
    ],
    "app/policy-to-proof/page.tsx": [
        "paid call, denial, delegation, and revocation produces receipts",
        "receipt_id",
        "evidence_pack_id",
        "policy_version",
        "decision_reason",
    ],
    "app/openai-budget-policy-generator/page.tsx": [
        "receipt_id",
        "evidence_pack_id",
        "decision_reason",
        "policy_version",
    ],
    "app/mcp-tool-cost-policy-generator/page.tsx": [
        "receipt_id",
        "evidence_pack_id",
        "decision_reason",
        "policy_version",
    ],
    "app/mcp-proxy-config-generator/page.tsx": [
        "SATGATE_REQUIRE_RECEIPT_ID",
        "SATGATE_REQUIRE_EVIDENCE_PACK_ID",
        "SATGATE_REQUIRE_DECISION_REASON",
        "SATGATE_REQUIRE_POLICY_VERSION",
    ],
    "../pkg/mcpserver/events.go": [
        "receipt-ready proxy event used to build Evidence Packs",
        "receipt-ready MCP proxy events",
    ],
}

FORBIDDEN_PUBLIC_PHRASES = {
    "app/components/HomeClient.tsx": ["Metered:    $847 usage", "Audit: who, when, diff"],
    "app/pricing/page.tsx": ["Real-time usage dashboard", "Live economic telemetry"],
    "app/pay/page.tsx": ["Zero Invoices. Zero Contracts. Zero Wait."],
    "app/dashboard/page.tsx": ["Dashboard telemetry explains decisions", "live governance telemetry"],
    "app/policy-to-proof/page.tsx": ["revocation event", "spend event", "matching receipts preserve the event log"],
}

REQUIRED_RECEIPT_FIELDS = {"receipt_id", "evidence_pack_id", "policy_version", "decision_reason"}


def fail(msg: str) -> None:
    print(f"FAIL: {msg}")
    raise SystemExit(1)


def read(rel: str) -> str:
    path = (ROOT / rel) if not rel.startswith("../") else (ROOT / rel).resolve()
    if not path.exists():
        fail(f"missing expected file: {rel}")
    return path.read_text()


def check_phrases() -> list[str]:
    errors: list[str] = []
    for rel, phrases in REQUIRED_PHRASES.items():
        text = read(rel)
        for phrase in phrases:
            if phrase not in text:
                errors.append(f"{rel}: missing required phrase {phrase!r}")
    for rel, phrases in FORBIDDEN_PUBLIC_PHRASES.items():
        text = read(rel)
        for phrase in phrases:
            if phrase in text:
                errors.append(f"{rel}: forbidden stale proof framing {phrase!r}")
    return errors


def check_evidence_pack_schema() -> list[str]:
    errors: list[str] = []
    schema_path = ROOT / "public/evidence-packs/evidence-pack.schema.v1.json"
    schema = json.loads(schema_path.read_text())
    receipt_schema = schema["properties"]["receipts"]["items"]
    required = set(receipt_schema.get("required", []))
    missing = REQUIRED_RECEIPT_FIELDS - required
    if missing:
        errors.append(f"schema receipt.required missing {sorted(missing)}")
    properties = set(receipt_schema.get("properties", {}).keys())
    missing_props = REQUIRED_RECEIPT_FIELDS - properties
    if missing_props:
        errors.append(f"schema receipt.properties missing {sorted(missing_props)}")
    return errors


def check_evidence_pack_samples() -> list[str]:
    errors: list[str] = []
    for name in ["sample-evidence-pack.json", "sample-evidence-pack.v1.json"]:
        path = ROOT / "public/evidence-packs" / name
        data = json.loads(path.read_text())
        if "policy_version" not in data:
            errors.append(f"{name}: missing top-level policy_version")
        for idx, receipt in enumerate(data.get("receipts", []), start=1):
            missing = REQUIRED_RECEIPT_FIELDS - set(receipt.keys())
            if missing:
                errors.append(f"{name}: receipt {idx} missing {sorted(missing)}")
    return errors


def check_mcp_templates() -> list[str]:
    errors: list[str] = []
    for path in (ROOT / "public/policy-templates/mcp-governance").glob("*.yaml"):
        text = path.read_text()
        if "required_receipt_fields:" in text and "- evidence_pack_id" not in text:
            errors.append(f"{path.name}: required_receipt_fields missing evidence_pack_id")
    bundle = json.loads((ROOT / "public/policy-templates/mcp-governance/mcp-governance-policy-bundle.v1.json").read_text())
    defaults = bundle.get("global_security_defaults", {})
    for key in ["require_receipt_id", "require_evidence_pack_id", "require_decision_reason", "require_policy_version_on_decision"]:
        if defaults.get(key) is not True:
            errors.append(f"policy bundle global_security_defaults.{key} is not true")
    return errors


def main() -> int:
    errors = []
    errors += check_phrases()
    errors += check_evidence_pack_schema()
    errors += check_evidence_pack_samples()
    errors += check_mcp_templates()
    if errors:
        print("Proof-spine guard failed:")
        for error in errors:
            print(f"- {error}")
        return 1
    print("proof-spine guard ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())

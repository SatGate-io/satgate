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
        "SatGate signs a receipt when it refuses an agent over a budget, a missing payment or a tool permission, and when an agent spends from a budget in Control.",
        "signed receipts for budget spends and for refusals over a budget, a missing payment or a tool permission",
    ],
    "app/components/HomeClient.tsx": [
        "Signed receipts for budget spends and for refusals over a budget, a missing payment or a tool permission, ready to export",
        "Allowed receipts",
        "Paid receipts",
        "Export Evidence Pack",
    ],
    "app/pricing/page.tsx": [
        "Signed receipts + receipt export",
        "Start free trial",
        "Ask about a 90-day design-partner pilot",
    ],
    "app/pay/page.tsx": [
        "SatGate signs a receipt when it refuses an agent for a missing payment",
        "Receipt Returned",
        "Paid-call receipt queued for export",
        "receipt_id",
    ],
    "app/dashboard/page.tsx": [
        "Receipt-backed Governance Dashboard",
        "receipt-backed decisions",
        "Evidence Pack) is the exportable proof artifact",
        "Denied-call receipts",
    ],
    "app/policy-to-proof/page.tsx": [
        "Signed receipts for budget spends and for refusals over a budget, a missing payment or a tool permission, which your CISO",
        "receipt_id",
        "evidence_pack_id",
        "policy_version",
        "decision_reason",
    ],
    "app/verify-evidence-pack/page.tsx": [
        "Check a SatGate receipt",
        "valid=true",
        "trusted_issuer_valid=true",
        "--jwks-file jwks.json",
        "--require-trusted-issuer",
        "<h3 className=\"mb-4 mt-8 text-xl font-bold text-white\">Limits</h3>",
        "/evidence/mcp-budget-refusal-20261005.json",
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
    "app/pricing/page.tsx": [
        "Real-time usage dashboard",
        "Live economic telemetry",
        "Start Free",
        "Start Pro",
        "Unlimited observe-mode requests",
        "$0.10 per 1K overage",
        "SOC 2 compliance package",
        "Dedicated CSM + SLA",
        "14 Days Free",
    ],
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


REMOVED_MCP_TEMPLATES = [
    "spend-caps.v1.yaml",
    "tool-allowlist.v1.yaml",
    "tenant-isolation.v1.yaml",
    "delegation-depth.v1.yaml",
    "mcp-governance-policy-bundle.v1.json",
]


def check_mcp_templates() -> list[str]:
    # SatGate does not read the satgate.io/policy/v1 format, so these files must stay deleted.
    errors: list[str] = []
    folder = ROOT / "public/policy-templates/mcp-governance"
    for name in REMOVED_MCP_TEMPLATES:
        if (folder / name).exists():
            errors.append(f"{name}: removed MCP policy template is back in public/policy-templates/mcp-governance")
    if list(folder.glob("*.yaml")):
        errors.append("public/policy-templates/mcp-governance has a .yaml policy template again")
    llms = (ROOT / "public/llms.txt").read_text()
    for name in REMOVED_MCP_TEMPLATES:
        if name in llms:
            errors.append(f"public/llms.txt still links removed MCP policy template {name}")
    return errors


def check_sanitized_closure() -> list[str]:
    errors: list[str] = []
    path = ROOT / "public/evidence/policy-to-proof-closure-20260718.json"
    if not path.exists():
        return ["missing sanitized Policy-to-Proof closure record"]
    record = json.loads(path.read_text())
    if record.get("scope") != "staging_only":
        errors.append("sanitized closure scope must remain staging_only")
    verification = record.get("verification", {})
    for key in [
        "manifest_entries_verified",
        "sensitive_data_scan_passed",
        "verifier_copies_byte_identical",
        "strict_verifier_valid",
        "trusted_issuer_valid",
        "restart_parity_verified",
    ]:
        if verification.get(key) is not True:
            errors.append(f"sanitized closure verification.{key} is not true")
    boundaries = record.get("boundaries", {})
    for key in [
        "production_touched",
        "production_promotion_authorized",
        "contains_raw_pack_identifiers",
        "contains_raw_receipt_identifiers",
        "contains_evidence_access_identifiers",
        "contains_bearer_urls",
        "contains_payment_credentials",
        "proves_runtime_truth_independently",
        "proves_billing_settlement",
        "proves_external_archive_anchoring",
        "proves_hardware_key_custody",
    ]:
        if boundaries.get(key) is not False:
            errors.append(f"sanitized closure boundaries.{key} must remain false")
    containment = record.get("historical_bearer_containment", {})
    if containment.get("exposed_staging_evidence_urls_found") != 2:
        errors.append("sanitized closure containment count drifted")
    if containment.get("targeted_archive_entries_removed") != 2:
        errors.append("sanitized closure removed count drifted")
    if containment.get("all_urls_http_404_after_restart") is not True:
        errors.append("sanitized closure must preserve post-restart 404 containment")
    if containment.get("git_history_rewritten") is not False:
        errors.append("sanitized closure must not imply Git history was rewritten")
    return errors


def main() -> int:
    errors = []
    errors += check_phrases()
    errors += check_evidence_pack_schema()
    errors += check_evidence_pack_samples()
    errors += check_mcp_templates()
    errors += check_sanitized_closure()
    if errors:
        print("Proof-spine guard failed:")
        for error in errors:
            print(f"- {error}")
        return 1
    print("proof-spine guard ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())

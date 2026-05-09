#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PAGE = ROOT / "app" / "policy-to-proof" / "page.tsx"

AGENT_CONTROL_PLANE = ROOT / "app" / "agent-control-plane" / "page.tsx"

required_phrases = [
    "Every agent action leaves a receipt",
    "without permanent credentials, unlimited spend, or unobservable authority",
    "Who authorized this?",
    "Which agent got access?",
    "What exactly could it do?",
    "What did it spend?",
    "What was denied?",
    "Can we prove revocation worked?",
    "Evidence Pack",
    "Mint",
    "Delegate",
    "Spend",
    "Deny",
    "Revoke",
    "Export",
    "Download sample Evidence Pack",
    "chain_root",
    "signature",
    "receipt_hash",
    "Other gateways",
    "SOC 2 CC6.1",
    "NIST AC-3",
    "Auditor",
    "CISO",
    "FinOps lead",
    "/acp-demo/satgate-acp-walkthrough.mp4",
    "/evidence-packs/sample-evidence-pack.json",
    "/evidence-packs/sample-evidence-pack.pdf",
    "REDACTED_DEMO_SAMPLE_DO_NOT_VERIFY",
    "Even producing the Evidence Pack is itself an auditable event",
    "Authority-chain entries preserve lineage",
]


def main() -> int:
    if not PAGE.exists():
        raise SystemExit(f"missing page: {PAGE.relative_to(ROOT)}")
    text = PAGE.read_text()
    missing = [phrase for phrase in required_phrases if phrase not in text]
    if missing:
        raise SystemExit("missing policy-to-proof phrases:\n" + "\n".join(missing))
    acp_text = AGENT_CONTROL_PLANE.read_text()
    if "/policy-to-proof" not in acp_text:
        raise SystemExit("agent-control-plane page must link to /policy-to-proof")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

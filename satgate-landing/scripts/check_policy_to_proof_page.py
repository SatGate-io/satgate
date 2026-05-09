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
    "/evidence-packs/sample-evidence-pack.json",
    "/evidence-packs/sample-evidence-pack.pdf",
    "REDACTED_DEMO_SAMPLE_DO_NOT_VERIFY",
    "evidence-pack-export-poster.svg",
    "satgate-evidence-pack-walkthrough.mp4",
    "90-second Evidence Pack cut",
    "What the {title} gets",
    "Full hashes, payment rail context, ed25519 signature, and verification block",
    "payment_context",
    "x402-style",
    "internal_enterprise_agents",
    "Internal first, rail-aware when needed",
    "Payment proves value moved. SatGate proves the agent was allowed to move it.",
    "ISO 27001 A.9.2.1",
    "Mint receipt — US SOC 2",
    "Mint receipt — ISO 27001",
    "Demo path",
    "Read the six-step lifecycle below, or watch the 90-second cut",
    "Want this for your stack?",
    "Book a 15-minute walkthrough",
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

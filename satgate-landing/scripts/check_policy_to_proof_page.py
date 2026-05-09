#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PAGE = ROOT / "app" / "policy-to-proof" / "page.tsx"

AGENT_CONTROL_PLANE = ROOT / "app" / "agent-control-plane" / "page.tsx"
GOVERN_PAGE = ROOT / "app" / "components" / "GovernClient.tsx"

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
    "x402",
    "internal_enterprise_agents",
    "Internal first, rail-aware when needed",
    "Why now",
    "Agents are moving from demos to workflows. Spend rails are arriving with them.",
    "Enterprises need proof of authority before those rails scale",
    "Payment proves value moved. SatGate proves the agent was allowed to move it.",
    "AgentCore Payments",
    "Pay.sh",
    "Authority before scale",
    "Sits above the rails",
    "Every authority decision is recorded — payment or not",
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

govern_required_phrases = [
    "Govern what agents can do",
    "Prove every decision",
    "Govern internal agents. Preserve proof across external rails.",
    "Payment proves value moved. SatGate proves the agent was allowed to move it",
    "Govern, enforce, prove.",
    "Evidence Pack exports",
    "first denied call after revoke",
    "Stop giving AI agents standing authority",
    "Export proof after",
    "Policy-to-Proof",
    "Policy enforced in the request path",
    "invoice-reconciler worker",
    "Worker Attempts Unauthorized Tool",
    "SatGate Denies Before Execution",
    "Evidence Pack Exported",
    "Authority Dashboard",
    "Revocation & Denial",
    "Root Policy",
    "Invoice Reconciler",
    "Agent governance readiness grader",
    "Is SatGate tied to x402, L402, AgentCore Payments, or Pay.sh?",
]

govern_forbidden_phrases = [
    "Not &ldquo;Who Are You?&rdquo; — &ldquo;What Can You Afford?&rdquo;",
    "Two Problems. One Token.",
    "Built for the C-Suite",
    "Three Modes. One Gateway.",
    "Monetize your APIs with L402",
    "Charge policy with Lightning settlement",
    "The Rogue Intern Story",
    "Your Internal APIs Are an Untapped Market",
    "Stop giving AI agents an all-you-can-eat buffet pass",
    "Govern → enforce → prove policy ratchet",
    "Agent Burns Budget",
    "CFO Chargeback Report",
    "Usage Dashboard",
    "Root Token",
    "Engineering VP",
    "Economic firewall readiness grader",
    "How does SatGate relate to x402, AgentCore Payments, and Pay.sh?",
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
    govern_text = GOVERN_PAGE.read_text()
    govern_missing = [phrase for phrase in govern_required_phrases if phrase not in govern_text]
    if govern_missing:
        raise SystemExit("missing govern policy-to-proof phrases:\n" + "\n".join(govern_missing))
    govern_stale = [phrase for phrase in govern_forbidden_phrases if phrase in govern_text]
    if govern_stale:
        raise SystemExit("stale govern spend/rail-first phrases remain:\n" + "\n".join(govern_stale))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

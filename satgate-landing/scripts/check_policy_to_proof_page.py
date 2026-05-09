#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PAGE = ROOT / "app" / "policy-to-proof" / "page.tsx"

AGENT_CONTROL_PLANE = ROOT / "app" / "agent-control-plane" / "page.tsx"
GOVERN_PAGE = ROOT / "app" / "components" / "GovernClient.tsx"
ECONOMIC_FIREWALL_PAGE = ROOT / "app" / "economic-firewall" / "page.tsx"

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
    "Policy enforced before execution",
    "Evidence preserved across rails",
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
    "By Monday",
    "How does SatGate relate to x402, AgentCore Payments, and Pay.sh?",
]

acp_required_phrases = [
    "Govern enterprise AI agents before they touch expensive models",
    "Internal first, rail-aware when needed",
    "One control plane for internal agents and governed paid calls.",
    "Evidence preserved across rails",
    "Finance parent agent",
    "Invoice-reconciler worker",
    "Customer-data export blocked by policy",
    "Platform team",
    "Security team",
    "FinOps team",
    "A one-page proof card. The full lifecycle exports as an Evidence Pack.",
    "See the full Evidence Pack lifecycle",
    "Why do enterprise AI agents need no standing authority?",
    "Govern enterprise agents end to end.",
]

acp_forbidden_phrases = [
    "Govern local AI agents before they touch expensive models",
    "Govern Local AI Agents",
    "Why do local AI agents need no standing authority?",
    "The same request path can charge robot customers",
    "A one-page view of the control model.",
    "Turn local agents into governed enterprise actors.",
]

economic_required_phrases = [
    "governs AI agent authority, spend, paid rails, and revocation before execution, then exports Evidence Pack proof",
    "Budget and authority limits",
    "Evidence capture",
    "Paid-rail context",
    "agents exercise authority at machine speed",
    "is this agent allowed to take this action right now?",
    "Evidence Pack capture",
    "Payment rails authorize value movement. Economic firewalls authorize behavior — and preserve the proof.",
    "Payment proves value moved; SatGate proves the agent was allowed to move it.",
    "Map agent authority",
    "Enforce scoped authority",
    "Preserve proof across paid rails",
    "Policy-to-Proof",
    "SatGate governs agent authority before value moves",
]

economic_forbidden_phrases = [
    "Charge/L402",
    "L402 Charge",
    "SatGate Charge uses L402 Lightning",
    "Fiat402 is a separate path",
    "Charge robot customers",
    "external robot customers",
    "agents spend money at machine speed",
    "is this agent allowed to spend this money right now?",
    "Govern local agent authority",
    "access, spend, delegate, and prove",
    "paid-rail proof",
    "Agent wallets validate the category",
    "Wallets authorize payment",
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
    acp_missing = [phrase for phrase in acp_required_phrases if phrase not in acp_text]
    if acp_missing:
        raise SystemExit("missing agent-control-plane policy-to-proof phrases:\n" + "\n".join(acp_missing))
    acp_stale = [phrase for phrase in acp_forbidden_phrases if phrase in acp_text]
    if acp_stale:
        raise SystemExit("stale agent-control-plane local/spend-first phrases remain:\n" + "\n".join(acp_stale))
    govern_text = GOVERN_PAGE.read_text()
    govern_missing = [phrase for phrase in govern_required_phrases if phrase not in govern_text]
    if govern_missing:
        raise SystemExit("missing govern policy-to-proof phrases:\n" + "\n".join(govern_missing))
    govern_stale = [phrase for phrase in govern_forbidden_phrases if phrase in govern_text]
    if govern_stale:
        raise SystemExit("stale govern spend/rail-first phrases remain:\n" + "\n".join(govern_stale))
    economic_text = ECONOMIC_FIREWALL_PAGE.read_text()
    economic_missing = [phrase for phrase in economic_required_phrases if phrase not in economic_text]
    if economic_missing:
        raise SystemExit("missing economic-firewall authority/proof phrases:\n" + "\n".join(economic_missing))
    economic_stale = [phrase for phrase in economic_forbidden_phrases if phrase in economic_text]
    if economic_stale:
        raise SystemExit("stale economic-firewall Charge/spend-first phrases remain:\n" + "\n".join(economic_stale))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

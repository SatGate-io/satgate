#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PAGE = ROOT / "app" / "policy-to-proof" / "page.tsx"

AGENT_CONTROL_PLANE = ROOT / "app" / "agent-control-plane" / "page.tsx"
GOVERN_PAGE = ROOT / "app" / "components" / "GovernClient.tsx"
ECONOMIC_FIREWALL_PAGE = ROOT / "app" / "economic-firewall" / "page.tsx"
READINESS_GRADER_PAGE = ROOT / "app" / "economic-firewall-readiness-grader" / "page.tsx"
READINESS_GRADER_LAYOUT = ROOT / "app" / "economic-firewall-readiness-grader" / "layout.tsx"
ROI_CALCULATOR_PAGE = ROOT / "app" / "roi-calculator" / "page.tsx"
ROI_CALCULATOR_LAYOUT = ROOT / "app" / "roi-calculator" / "layout.tsx"
AGENT_BUDGET_POLICY_PAGE = ROOT / "app" / "agent-spend-policy-template" / "page.tsx"
AGENT_BUDGET_POLICY_LAYOUT = ROOT / "app" / "agent-spend-policy-template" / "layout.tsx"
CAPABILITY_TOKEN_TEMPLATE_PAGE = ROOT / "app" / "revocable-capability-token-policy-template" / "page.tsx"
CAPABILITY_TOKEN_TEMPLATE_LAYOUT = ROOT / "app" / "revocable-capability-token-policy-template" / "layout.tsx"

SPEND_LONGTAIL_PAGES = {
    "ai-agent-cost-control": ROOT / "app" / "ai-agent-cost-control" / "page.tsx",
    "ai-api-budget-enforcement": ROOT / "app" / "ai-api-budget-enforcement" / "page.tsx",
    "agent-spending-limits": ROOT / "app" / "agent-spending-limits" / "page.tsx",
    "mcp-cost-control": ROOT / "app" / "mcp-cost-control" / "page.tsx",
    "agent-payment-controls": ROOT / "app" / "agent-payment-controls" / "page.tsx",
}

L402_RAIL_PAGES = {
    "http-402-for-ai-agents": ROOT / "app" / "http-402-for-ai-agents" / "page.tsx",
    "l402-agent-payments": ROOT / "app" / "l402-agent-payments" / "page.tsx",
}

MCP_AUTHORITY_PAGES = {
    "mcp-governance": ROOT / "app" / "mcp-governance" / "page.tsx",
    "mcp-gateway": ROOT / "app" / "mcp-gateway" / "page.tsx",
    "agent-api-governance": ROOT / "app" / "agent-api-governance" / "page.tsx",
}

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
    "Payment rails authorize value movement. SatGate authorizes behavior and preserves proof.",
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
    "What scoped capability is it using?",
    "Is the requested action allowed under policy?",
    "Does authority remain — scope, budget, expiry, revocation?",
    "Should the request be allowed, denied, delegated, paid, or recorded in the Evidence Pack?",
    "Payment rails authorize value movement. Economic firewalls authorize behavior — and preserve the proof.",
    "Every authority decision — allowed, denied, delegated, revoked, or paid — feeds the Evidence Pack.",
    "Payment proves value moved; SatGate proves the agent was allowed to move it.",
    "Map agent authority",
    "Enforce scoped authority",
    "Preserve proof across paid rails",
    "Policy-to-Proof",
    "SatGate governs agent authority before value moves",
    "govern paid rails when value moves",
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
    "What capability/token is it using?",
    "Is this route allowed?",
    "What will this request cost?",
    "Does budget remain?",
    "routed, charged, or recorded as evidence",
    "When an API becomes a product for external agents",
    "what evidence is captured",
    "evidence capture, and optional payment context",
    "when APIs become products for external agents",
]

readiness_required_phrases = [
    "Grade whether your agent/API stack can handle autonomous authority",
    "Evidence Pack capture",
    "Paid-rail context",
    "govern paid rails when value moves",
    "SatGate governs agent authority before execution",
    "Prove allowed, denied, delegated, revoked, or paid decisions with an Evidence Pack",
    "Assess whether your agent/API stack is ready for autonomous authority",
    "See SatGate governance",
    "See the Evidence Pack",
    "spend context",
]

readiness_forbidden_phrases = [
    "autonomous spend",
    "L402 robot payments",
    "L402 Charge",
    "Robot payments",
    "robot customers",
    "when APIs become products",
    "when agents become customers",
    "Observe what agents spend",
    "Charge robot customers",
    "See agent authority control",
    "/ai-agent-cost-control",
    "Generate MCP tool policy",
    "Generate OpenAI budget policy",
    "authority, cost, policy decision",
]

tool_required_phrases = {
    "roi-calculator": [
        "Turn the ROI model into Policy-to-Proof",
        "Evidence Pack",
        "See Policy-to-Proof",
        "Govern agent actions",
        "Policy-to-Proof receipt coverage",
        "Map ROI to Policy-to-Proof",
    ],
    "agent-budget-policy-template": [
        "Agent Budget Policy Template",
        "control_with_receipts",
        "receipt_id",
        "evidence_pack_id",
        "Evidence Pack fields",
        "See Policy-to-Proof",
        "Govern agent execution",
        "Create Evidence Pack trail",
    ],
    "revocable-capability-token-policy-template": [
        "External agent access",
        "policy:decision:read",
        "receipt_id",
        "evidence_pack_id",
        "Evidence Pack evidence",
        "See Policy-to-Proof",
        "Govern agent authority",
        "Create Evidence Pack trail",
    ],
}

tool_forbidden_phrases = {
    "roi-calculator": [
        "robot customers",
        "Charge Demo",
        "L402 before access is granted",
        "/pay",
        "/l402-api-pricing-calculator",
        "/runaway-agent-cost-calculator",
        "Observe, Control, and Charge",
    ],
    "agent-budget-policy-template": [
        "control_and_charge",
        "Charge when robot customers pay",
        "L402 payments",
        "/ai-api-budget-enforcement",
        "/agent-spending-limits",
        "/mcp-cost-control",
        "Agent Spend Policy Template",
    ],
    "revocable-capability-token-policy-template": [
        "agent:robot-customer",
        "paid-api-access",
        "charge:l402:pay",
        "Robot customer / L402",
        "L402 Charge",
        "Charge/L402",
        "/agent-capability-tokens",
        "/revocable-agent-credentials",
        "/agent-api-key-risk-assessment",
        "/economic-firewall-readiness-grader",
    ],
}

spend_longtail_required_phrases = {
    "ai-agent-cost-control": [
        "authority before execution",
        "audit receipts",
        "Policy-to-Proof",
        "Evidence Pack capture",
        "/govern",
        "/policy-to-proof",
        "paid-rail evidence",
    ],
    "ai-api-budget-enforcement": [
        "authority before execution",
        "audit receipts",
        "Policy-to-Proof",
        "Govern AI API budgets",
        "/govern",
        "/policy-to-proof",
    ],
    "agent-spending-limits": [
        "authority before execution",
        "receipt for every budget decision",
        "Policy-to-Proof",
        "Govern agent spending limits",
        "/govern",
        "/policy-to-proof",
    ],
    "mcp-cost-control": [
        "MCP cost control belongs before tool execution",
        "authority, budget, revocation, and receipt policy",
        "Policy-to-Proof",
        "Govern MCP tool spend",
        "/govern",
        "/policy-to-proof",
    ],
    "agent-payment-controls": [
        "Policy Before Agent Payments",
        "paid-rail context",
        "Evidence Pack",
        "Policy-to-Proof evidence",
        "Govern agent payments",
        "/govern",
        "/policy-to-proof",
    ],
}

spend_longtail_forbidden_phrases = {
    "ai-agent-cost-control": [
        "Charge robot customers",
        "robot customers",
        "external robot customers",
        "Charge/L402",
        "L402 Charge",
        "L402 API monetization",
        "when APIs become products",
        "when agents become customers",
        "03 / CHARGE",
        "/robot-customer-payments",
    ],
    "ai-api-budget-enforcement": [
        "Charge robot customers",
        "robot customers",
        "external robot customers",
        "Charge/L402",
        "L402 Charge",
        "L402 API monetization",
        "when APIs become products",
        "Open free tools",
    ],
    "agent-spending-limits": [
        "Charge robot customers",
        "robot customers",
        "external robot customers",
        "Charge/L402",
        "L402 Charge",
        "L402 API monetization",
        "when APIs become products",
        "Open free tools",
    ],
    "mcp-cost-control": [
        "Charge robot customers",
        "robot customers",
        "external robot customers",
        "Charge/L402",
        "L402 Charge",
        "L402 API monetization",
        "when APIs become products",
        "Open free tools",
    ],
    "agent-payment-controls": [
        "Charge robot customers",
        "robot customers",
        "external robot customers",
        "L402 Charge",
        "L402 API monetization",
        "SatGate Charge",
        "when APIs become agent-native products",
        "Compare Link and SatGate",
        "Learn the economic firewall",
        "Read HTTP 402 for agents",
    ],
}

l402_rail_required_phrases = {
    "http-402-for-ai-agents": [
        "Payment Required, governed before execution",
        "authority to spend",
        "Evidence Pack receipt",
        "Govern paid agent actions",
        "See Policy-to-Proof",
        "L402 Lightning is one paid rail",
        "/govern",
        "/policy-to-proof",
        "/l402-agent-payments",
    ],
    "l402-agent-payments": [
        "L402 Agent Payments, Governed Before Access",
        "SatGate applies Policy-to-Proof before execution",
        "preserves proof for every paid action",
        "Evidence Pack receipt",
        "Govern L402 access",
        "Govern paid agent actions",
        "View Policy-to-Proof",
        "/govern",
        "/policy-to-proof",
        "/http-402-for-ai-agents",
    ],
}

l402_rail_forbidden_phrases = [
    "robot customers",
    "robot-customer",
    "SatGate Charge",
    "Charge/L402",
    "L402 Charge",
    "Charge is L402",
    "L402 is SatGate Charge",
    "when APIs become products",
    "when APIs become robot-customer products",
    "Turn protected APIs into robot-customer products",
    "Try the payment demo",
    "Monetize APIs",
    "Price L402 API access",
    "Grade your readiness",
]

mcp_authority_required_phrases = {
    "mcp-governance": [
        "MCP Governance for Agents That Need Authority Before Execution",
        "before execution",
        "audit receipt",
        "Evidence Pack",
        "finance-automation",
        "invoice-reconciler",
        "Govern MCP tool calls",
        "See Policy-to-Proof",
        "/govern",
        "/policy-to-proof",
    ],
    "mcp-gateway": [
        "MCP Gateway for Governed Agent Tool Access",
        "authority before execution",
        "audit receipt",
        "Evidence Pack",
        "SaaS MCP is Fly-hosted",
        "Hybrid MCP is Hetzner-hosted",
        "Govern MCP tool access",
        "See Policy-to-Proof",
        "/govern",
        "/policy-to-proof",
    ],
    "agent-api-governance": [
        "Agent API Governance",
        "static API keys",
        "scoped capabilities",
        "before every API request",
        "audit receipt",
        "Evidence Pack",
        "finance-automation",
        "invoice-reconciler",
        "See SatGate governance",
        "See Policy-to-Proof",
        "/govern",
        "/policy-to-proof",
    ],
}

mcp_authority_forbidden_phrases = {
    "mcp-governance": [
        "Agents That Can Actually Spend Money",
        "charge when tools become products",
        "robot-customer",
        "Charge-mode",
        "Economic firewall category",
        "Generate MCP tool policy",
        "control spend and access",
        "economic control plane",
    ],
    "mcp-gateway": [
        "observe, control, and charge",
        "Observe, Control, Charge",
        "Charge for tool access",
        "robot-customer",
        "Charge-mode",
        "chargeable events",
        "economic activity",
        "Generate MCP tool policy",
        "/blog/http-402-payment-required-use-cases",
    ],
    "agent-api-governance": [
        "research-bot",
        "local agent authority",
        "Charge readiness",
        "Economic firewall overview",
        "Macaroons vs API keys <ArrowRight",
    ],
}


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
    readiness_text = READINESS_GRADER_PAGE.read_text() + "\n" + READINESS_GRADER_LAYOUT.read_text()
    readiness_missing = [phrase for phrase in readiness_required_phrases if phrase not in readiness_text]
    if readiness_missing:
        raise SystemExit("missing readiness-grader authority/proof phrases:\n" + "\n".join(readiness_missing))
    readiness_stale = [phrase for phrase in readiness_forbidden_phrases if phrase in readiness_text]
    if readiness_stale:
        raise SystemExit("stale readiness-grader spend/Charge phrases remain:\n" + "\n".join(readiness_stale))
    tool_texts = {
        "roi-calculator": ROI_CALCULATOR_PAGE.read_text() + "\n" + ROI_CALCULATOR_LAYOUT.read_text(),
        "agent-budget-policy-template": AGENT_BUDGET_POLICY_PAGE.read_text() + "\n" + AGENT_BUDGET_POLICY_LAYOUT.read_text(),
        "revocable-capability-token-policy-template": CAPABILITY_TOKEN_TEMPLATE_PAGE.read_text()
        + "\n"
        + CAPABILITY_TOKEN_TEMPLATE_LAYOUT.read_text(),
    }
    for name, tool_text in tool_texts.items():
        tool_missing = [phrase for phrase in tool_required_phrases[name] if phrase not in tool_text]
        if tool_missing:
            raise SystemExit(f"missing {name} authority/proof phrases:\n" + "\n".join(tool_missing))
        tool_stale = [phrase for phrase in tool_forbidden_phrases[name] if phrase in tool_text]
        if tool_stale:
            raise SystemExit(f"stale {name} spend/Charge/tool-exit phrases remain:\n" + "\n".join(tool_stale))

    for name, path in SPEND_LONGTAIL_PAGES.items():
        page_text = path.read_text()
        missing_longtail = [phrase for phrase in spend_longtail_required_phrases[name] if phrase not in page_text]
        if missing_longtail:
            raise SystemExit(f"missing {name} spend-longtail authority/proof phrases:\n" + "\n".join(missing_longtail))
        stale_longtail = [phrase for phrase in spend_longtail_forbidden_phrases[name] if phrase in page_text]
        if stale_longtail:
            raise SystemExit(f"stale {name} spend/Charge-first phrases remain:\n" + "\n".join(stale_longtail))

    for name, path in L402_RAIL_PAGES.items():
        page_text = path.read_text()
        missing_l402 = [phrase for phrase in l402_rail_required_phrases[name] if phrase not in page_text]
        if missing_l402:
            raise SystemExit(f"missing {name} paid-rail authority/proof phrases:\n" + "\n".join(missing_l402))
        stale_l402 = [phrase for phrase in l402_rail_forbidden_phrases if phrase in page_text]
        if stale_l402:
            raise SystemExit(f"stale {name} Charge/robot-customer phrases remain:\n" + "\n".join(stale_l402))


    for name, path in MCP_AUTHORITY_PAGES.items():
        page_text = path.read_text()
        missing_mcp = [phrase for phrase in mcp_authority_required_phrases[name] if phrase not in page_text]
        if missing_mcp:
            raise SystemExit(f"missing {name} MCP authority/proof phrases:\n" + "\n".join(missing_mcp))
        stale_mcp = [phrase for phrase in mcp_authority_forbidden_phrases[name] if phrase in page_text]
        if stale_mcp:
            raise SystemExit(f"stale {name} MCP spend/Charge/local-first phrases remain:\n" + "\n".join(stale_mcp))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

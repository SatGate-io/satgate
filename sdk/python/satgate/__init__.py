"""
SatGate Python SDK

Developer primitive: issue/pay/verify.

    import os
    from satgate import SatGate

    satgate = SatGate(api_key=os.getenv("SATGATE_API_KEY"))
    capability = satgate.issue(
        task="summarize vendor invoice",
        agent="invoice-agent",
        allow=["POST /v1/invoices/*"],
        budget_usd=0.25,
        expires_in="10m",
    )
    receipt = satgate.pay(
        upstream="https://api.vendor.test/v1/invoices/42",
        capability=capability,
        max_usd=0.10,
    )
    verified = satgate.verify(receipt)
    print(verified.decision, getattr(verified, "evidence_pack_id", None))

Compatibility: SatGateClient and SatGateAgentClient preserve existing OSS Gateway
token, delegation, and paid-rail APIs. New Cloud/private-beta app examples should
lead with issue/pay/verify.
"""

from .private_beta import SatGate
from .client import SatGateClient
from .agent_client import (
    SatGateAgentClient,
    AgentResponse,
    LightningWallet,
    LNDWallet,
    AlbyWallet,
)

# LangChain integration (optional - requires langchain)
try:
    from .langchain import (
        SatGateTool,
        SatGateRESTTool,
        SatGateToolkit,
        create_satgate_tool,
    )
    LANGCHAIN_AVAILABLE = True
except ImportError:
    LANGCHAIN_AVAILABLE = False
    SatGateTool = None
    SatGateRESTTool = None
    SatGateToolkit = None
    create_satgate_tool = None

from .models import Token, TokenInfo, BanRecord, Stats, GraphData
from .delegation import (
    Caveats,
    DelegationBuilder,
    DelegationPatterns,
    delegate,
    parse_caveats,
    is_more_restrictive,
    DelegationNode,
)
from .exceptions import (
    SatGateError,
    AuthenticationError,
    SatGateAuthError,
    NotFoundError,
    PaymentRequiredError,
    PaymentFailedError,
    BudgetExceededError,
    TokenExpiredError,
    DelegationError,
)

__version__ = "0.3.2"
__all__ = [
    # SatGate Cloud private-beta facade
    "SatGate",

    # Admin client
    "SatGateClient",
    
    # Agent client
    "SatGateAgentClient",
    "AgentResponse",
    
    # Lightning wallets
    "LightningWallet",
    "LNDWallet",
    "AlbyWallet",
    
    # LangChain integration
    "SatGateTool",
    "SatGateRESTTool",
    "SatGateToolkit",
    "create_satgate_tool",
    "LANGCHAIN_AVAILABLE",
    
    # Models
    "Token",
    "TokenInfo",
    "BanRecord",
    "Stats",
    "GraphData",
    
    # Delegation helpers
    "Caveats",
    "DelegationBuilder",
    "DelegationPatterns",
    "delegate",
    "parse_caveats",
    "is_more_restrictive",
    "DelegationNode",
    
    # Exceptions
    "SatGateError",
    "AuthenticationError",
    "SatGateAuthError",
    "NotFoundError",
    "PaymentRequiredError",
    "PaymentFailedError",
    "BudgetExceededError",
    "TokenExpiredError",
    "DelegationError",
]

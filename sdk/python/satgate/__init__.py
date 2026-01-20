"""
SatGate Gateway Python SDK

A Python SDK for interacting with the SatGate Enterprise Gateway.

## Admin Client (for operators)

    from satgate import SatGateClient
    
    client = SatGateClient(
        base_url="http://localhost:9090",
        admin_token="your-admin-token"
    )
    
    # Mint a new token
    token = client.tokens.mint(scope="api:*", expires_in=3600)
    print(f"Token: {token.token}")

## Agent Client (for AI agents)

    from satgate import SatGateAgentClient
    
    # Auto-detect environment (K8s/AWS)
    client = SatGateAgentClient(
        gateway_url="https://gateway.internal",
        identity="auto"
    )
    
    # Make requests - 402 handling is automatic
    response = client.get("/api/v1/data")
    print(f"Cost: ${response.cost:.2f}")

## With Lightning wallet for external APIs

    from satgate import SatGateAgentClient, LNDWallet
    
    client = SatGateAgentClient(
        gateway_url="https://api.external.com",
        wallet=LNDWallet(macaroon_path="~/.lnd/admin.macaroon")
    )
"""

from .client import SatGateClient
from .agent_client import (
    SatGateAgentClient,
    AgentResponse,
    IdentityProvider,
    KubernetesIdentity,
    AWSIdentity,
    OIDCIdentity,
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
from .models import Token, TokenInfo, BanRecord, Stats
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
    NotFoundError,
    PaymentRequiredError,
    PaymentFailedError,
    BudgetExceededError,
    TokenExpiredError,
    DelegationError,
)

__version__ = "2.0.0"
__all__ = [
    # Admin client
    "SatGateClient",
    
    # Agent client
    "SatGateAgentClient",
    "AgentResponse",
    
    # Identity providers
    "IdentityProvider",
    "KubernetesIdentity",
    "AWSIdentity",
    "OIDCIdentity",
    
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
    "NotFoundError",
    "PaymentRequiredError",
    "PaymentFailedError",
    "BudgetExceededError",
    "TokenExpiredError",
    "DelegationError",
]




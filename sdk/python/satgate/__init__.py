"""
SatGate Python SDK - Automatic L402 Payment Handling

Give your AI agents a Lightning wallet in 2 lines of code.

Quick Start:
    from satgate import SatGateClient, LNBitsWallet
    
    wallet = LNBitsWallet(url="https://legend.lnbits.com", admin_key="...")
    client = SatGateClient(wallet)
    
    # 402 → Pay → Retry happens automatically
    response = client.get("https://api.example.com/premium")

LangChain Integration:
    from satgate import SatGateTool, LNBitsWallet
    from langchain.agents import initialize_agent
    
    wallet = LNBitsWallet(url="...", admin_key="...")
    tool = SatGateTool(wallet=wallet)
    
    agent = initialize_agent(
        tools=[tool],
        llm=your_llm,
        agent="zero-shot-react-description"
    )

"""

from .client import (
    # Main client
    SatGateSession,
    SatGateClient,  # Alias
    
    # Wallet interfaces
    LightningWallet,
    LNBitsWallet,
    AlbyWallet,
    
    # Data classes
    PaymentInfo,
    TokenCache,
    
    # Exceptions
    SatGateError,
    PaymentError,
    L402ParseError,
)

__version__ = "0.3.1"

# Optional LangChain integration (only if langchain is installed)
try:
    from .langchain_integrations import SatGateTool, SatGateToolInput
    _LANGCHAIN_AVAILABLE = True
except ImportError:
    _LANGCHAIN_AVAILABLE = False
    SatGateTool = None
    SatGateToolInput = None

__all__ = [
    # Main client
    "SatGateSession",
    "SatGateClient",
    
    # Wallets
    "LightningWallet",
    "LNBitsWallet", 
    "AlbyWallet",
    
    # Data
    "PaymentInfo",
    "TokenCache",
    
    # Exceptions
    "SatGateError",
    "PaymentError",
    "L402ParseError",
    
    # LangChain (optional)
    "SatGateTool",
    "SatGateToolInput",
]

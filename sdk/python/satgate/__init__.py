"""
SatGate Python SDK - Automatic L402 Payment Handling

Give your AI agents a Lightning wallet in 2 lines of code.

Quick Start:
    from satgate import SatGateClient, LNBitsWallet
    
    wallet = LNBitsWallet(url="https://legend.lnbits.com", admin_key="...")
    client = SatGateClient(wallet)
    
    # 402 → Pay → Retry happens automatically
    response = client.get("https://api.example.com/premium")

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

__version__ = "0.2.0"

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
]

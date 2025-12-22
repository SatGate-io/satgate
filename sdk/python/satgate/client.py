"""
SatGate Python SDK - Automatic L402 Payment Handling

The "Stripe Moment": 2 lines to access any L402-protected API.

    from satgate import SatGateClient
    
    client = SatGateClient(wallet=my_wallet)
    response = client.get("https://api.example.com/premium")
    # That's it. 402 → Pay → Retry happens automatically.

"""

import requests
import re
import hashlib
import time
from abc import ABC, abstractmethod
from typing import Optional, Dict, Any, Callable
from dataclasses import dataclass


# =============================================================================
# WALLET INTERFACES
# =============================================================================

class LightningWallet(ABC):
    """Abstract Base Class for any Lightning Wallet.
    
    Implement pay_invoice() to connect your wallet:
    - LND (via REST or gRPC)
    - LNBits
    - Alby API
    - Any other Lightning wallet
    """
    
    @abstractmethod
    def pay_invoice(self, invoice: str) -> str:
        """
        Pay a Lightning invoice and return the preimage.
        
        Args:
            invoice: BOLT11 invoice string (starts with lnbc...)
            
        Returns:
            Preimage as hex string (64 characters)
            
        Raises:
            PaymentError: If payment fails
        """
        pass


class LNBitsWallet(LightningWallet):
    """Pre-built wallet for LNBits instances."""
    
    def __init__(self, url: str, admin_key: str):
        """
        Args:
            url: LNBits instance URL (e.g., https://legend.lnbits.com)
            admin_key: Admin key for the wallet
        """
        self.url = url.rstrip('/')
        self.admin_key = admin_key
    
    def pay_invoice(self, invoice: str) -> str:
        response = requests.post(
            f"{self.url}/api/v1/payments",
            headers={"X-Api-Key": self.admin_key},
            json={"out": True, "bolt11": invoice}
        )
        response.raise_for_status()
        data = response.json()
        
        # LNBits returns payment_hash, we need to get preimage
        payment_hash = data.get("payment_hash")
        if not payment_hash:
            raise PaymentError("No payment_hash in response")
        
        # Check payment status to get preimage
        check = requests.get(
            f"{self.url}/api/v1/payments/{payment_hash}",
            headers={"X-Api-Key": self.admin_key}
        )
        check.raise_for_status()
        check_data = check.json()
        
        preimage = check_data.get("preimage")
        if not preimage:
            raise PaymentError("Payment succeeded but no preimage returned")
        
        return preimage


class AlbyWallet(LightningWallet):
    """Pre-built wallet for Alby API (getalby.com)."""
    
    def __init__(self, access_token: str):
        """
        Args:
            access_token: Alby API access token
        """
        self.access_token = access_token
        self.base_url = "https://api.getalby.com"
    
    def pay_invoice(self, invoice: str) -> str:
        response = requests.post(
            f"{self.base_url}/payments/bolt11",
            headers={"Authorization": f"Bearer {self.access_token}"},
            json={"invoice": invoice}
        )
        response.raise_for_status()
        data = response.json()
        
        preimage = data.get("payment_preimage")
        if not preimage:
            raise PaymentError("No preimage in Alby response")
        
        return preimage


# =============================================================================
# EXCEPTIONS
# =============================================================================

class SatGateError(Exception):
    """Base exception for SatGate SDK."""
    pass


class PaymentError(SatGateError):
    """Payment failed."""
    pass


class L402ParseError(SatGateError):
    """Could not parse L402 challenge."""
    pass


# =============================================================================
# PAYMENT INFO
# =============================================================================

@dataclass
class PaymentInfo:
    """Information about a completed payment."""
    invoice: str
    preimage: str
    macaroon: str
    amount_sats: Optional[int] = None
    endpoint: Optional[str] = None
    timestamp: float = 0.0
    
    def __post_init__(self):
        if self.timestamp == 0.0:
            self.timestamp = time.time()


# =============================================================================
# TOKEN CACHE
# =============================================================================

class TokenCache:
    """Cache L402 tokens to avoid paying twice for the same endpoint."""
    
    def __init__(self, ttl_seconds: int = 3600):
        self._cache: Dict[str, PaymentInfo] = {}
        self._ttl = ttl_seconds
    
    def _key(self, url: str) -> str:
        """Generate cache key from URL (domain + path)."""
        # Hash the URL to create a consistent key
        return hashlib.sha256(url.encode()).hexdigest()[:16]
    
    def get(self, url: str) -> Optional[str]:
        """Get cached L402 token for URL, if still valid."""
        key = self._key(url)
        if key not in self._cache:
            return None
        
        info = self._cache[key]
        # Check if expired
        if time.time() - info.timestamp > self._ttl:
            del self._cache[key]
            return None
        
        return f"L402 {info.macaroon}:{info.preimage}"
    
    def set(self, url: str, info: PaymentInfo):
        """Cache a payment for an endpoint."""
        key = self._key(url)
        self._cache[key] = info
    
    def clear(self):
        """Clear all cached tokens."""
        self._cache.clear()


# =============================================================================
# MAIN CLIENT
# =============================================================================

class SatGateSession(requests.Session):
    """
    Drop-in replacement for requests.Session with automatic L402 handling.
    
    The "Stripe Moment" - automatic payment flow:
    1. Request hits 402 Payment Required
    2. Parse L402 challenge (macaroon + invoice)
    3. Pay invoice via your wallet
    4. Retry with L402 token
    5. Return successful response
    
    Example:
        from satgate import SatGateSession, LNBitsWallet
        
        wallet = LNBitsWallet(url="https://legend.lnbits.com", admin_key="...")
        session = SatGateSession(wallet)
        
        # This just works - payment is automatic
        response = session.get("https://api.example.com/premium/data")
        print(response.json())
    """
    
    def __init__(
        self, 
        wallet: LightningWallet,
        cache_tokens: bool = True,
        cache_ttl: int = 3600,
        on_payment: Optional[Callable[[PaymentInfo], None]] = None,
        verbose: bool = True
    ):
        """
        Args:
            wallet: Lightning wallet for paying invoices
            cache_tokens: Cache L402 tokens to avoid re-paying (default: True)
            cache_ttl: Token cache TTL in seconds (default: 1 hour)
            on_payment: Callback when payment is made (for logging/UI)
            verbose: Print payment progress to console (default: True)
        """
        super().__init__()
        self.wallet = wallet
        self.cache = TokenCache(cache_ttl) if cache_tokens else None
        self.on_payment = on_payment
        self.verbose = verbose
        self._total_paid_sats = 0

    def request(self, method: str, url: str, *args, **kwargs) -> requests.Response:
        """Override request to handle L402 automatically."""
        
        # Check cache first
        if self.cache:
            cached_token = self.cache.get(url)
            if cached_token:
                if self.verbose:
                    print(f"🎫 Using cached L402 token for {url[:50]}...")
                headers = kwargs.get("headers", {}) or {}
                headers = headers.copy()
                headers["Authorization"] = cached_token
                kwargs["headers"] = headers
        
        # Make the request
        try:
            response = super().request(method, url, *args, **kwargs)
        except requests.exceptions.RequestException as e:
            raise e

        # Handle 402
        if response.status_code == 402:
            return self._handle_l402(response, method, url, *args, **kwargs)
        
        return response

    def _handle_l402(
        self, 
        response: requests.Response, 
        method: str, 
        url: str, 
        *args, 
        **kwargs
    ) -> requests.Response:
        """Parse L402 challenge, pay, and retry."""
        
        # Parse the challenge
        auth_header = response.headers.get("WWW-Authenticate", "")
        
        if not auth_header:
            if self.verbose:
                print("⚠️ 402 received but no WWW-Authenticate header")
            return response

        # Support both L402 and LSAT schemes
        if "L402" not in auth_header and "LSAT" not in auth_header:
            if self.verbose:
                print("⚠️ 402 received but not L402/LSAT scheme")
            return response

        # Extract macaroon and invoice
        macaroon_match = re.search(r'macaroon="([^"]+)"', auth_header)
        invoice_match = re.search(r'invoice="([^"]+)"', auth_header)

        if not macaroon_match or not invoice_match:
            raise L402ParseError("Could not parse macaroon/invoice from L402 header")

        macaroon = macaroon_match.group(1)
        invoice = invoice_match.group(1)
        
        # Try to extract amount from invoice (basic parsing)
        amount_sats = self._parse_invoice_amount(invoice)

        if self.verbose:
            amount_str = f"{amount_sats} sats" if amount_sats else "unknown amount"
            print(f"⚡ L402 Challenge: {amount_str}")
            print(f"   Invoice: {invoice[:25]}...{invoice[-10:]}")

        # Pay the invoice
        try:
            preimage = self.wallet.pay_invoice(invoice)
            if not preimage:
                raise PaymentError("Wallet returned empty preimage")
            
            if self.verbose:
                print(f"✅ Paid! Preimage: {preimage[:16]}...")
            
            # Track payment
            if amount_sats:
                self._total_paid_sats += amount_sats
            
            # Create payment info
            payment_info = PaymentInfo(
                invoice=invoice,
                preimage=preimage,
                macaroon=macaroon,
                amount_sats=amount_sats,
                endpoint=url
            )
            
            # Cache the token
            if self.cache:
                self.cache.set(url, payment_info)
            
            # Callback
            if self.on_payment:
                self.on_payment(payment_info)
                
        except Exception as e:
            if self.verbose:
                print(f"❌ Payment failed: {e}")
            raise PaymentError(f"Payment failed: {e}") from e

        # Retry with L402 token
        l402_token = f"L402 {macaroon}:{preimage}"
        
        headers = kwargs.get("headers", {}) or {}
        headers = headers.copy()
        headers["Authorization"] = l402_token
        kwargs["headers"] = headers

        if self.verbose:
            print("🔄 Retrying with L402 token...")
        
        return super().request(method, url, *args, **kwargs)

    def _parse_invoice_amount(self, invoice: str) -> Optional[int]:
        """Extract amount in sats from BOLT11 invoice (basic parsing)."""
        try:
            # BOLT11 format: lnbc<amount><multiplier>...
            # Multipliers: m=milli, u=micro, n=nano, p=pico
            match = re.match(r'ln(?:bc|tb)(\d+)([munp])?', invoice.lower())
            if not match:
                return None
            
            amount = int(match.group(1))
            multiplier = match.group(2)
            
            # Convert to satoshis (1 BTC = 100,000,000 sats)
            if multiplier == 'm':  # milli-BTC
                return amount * 100000
            elif multiplier == 'u':  # micro-BTC
                return amount * 100
            elif multiplier == 'n':  # nano-BTC
                return amount // 10
            elif multiplier == 'p':  # pico-BTC
                return amount // 10000
            else:
                # No multiplier = BTC
                return amount * 100000000
        except:
            return None
    
    @property
    def total_paid_sats(self) -> int:
        """Total satoshis paid in this session."""
        return self._total_paid_sats


# Alias for simpler import
SatGateClient = SatGateSession

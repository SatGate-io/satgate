"""SatGate Gateway Exceptions"""

from typing import Any, Dict, Optional


class SatGateError(Exception):
    """Base exception for SatGate errors"""
    pass


class AuthenticationError(SatGateError):
    """Raised when authentication fails"""
    pass


class SatGateAuthError(AuthenticationError):
    """Raised when SatGate Cloud private-beta API access is missing."""

    def __init__(self, message: str, status_code: int = 401, docs_url: str = "https://cloud.satgate.io/docs"):
        super().__init__(message)
        self.status_code = status_code
        self.docs_url = docs_url


class NotFoundError(SatGateError):
    """Raised when a resource is not found"""
    pass


class ValidationError(SatGateError):
    """Raised when validation fails"""
    pass


class RateLimitError(SatGateError):
    """Raised when rate limit is exceeded"""
    pass


class PaymentRequiredError(SatGateError):
    """402 Payment Required - economic gate triggered"""
    
    def __init__(self, message: str, challenge: Optional[Dict[str, Any]] = None):
        super().__init__(message)
        self.challenge = challenge or {}
    
    @property
    def amount(self) -> float:
        """The amount required for payment"""
        return self.challenge.get('amount', 0)
    
    @property
    def unit(self) -> str:
        """The currency unit (USD, sats, credits)"""
        return self.challenge.get('unit', 'unknown')
    
    @property
    def invoice_id(self) -> Optional[str]:
        """The invoice ID for Fiat402 challenges"""
        return self.challenge.get('invoice_id')
    
    @property
    def invoice(self) -> Optional[str]:
        """The Lightning invoice for L402 challenges"""
        return self.challenge.get('invoice')


class PaymentFailedError(SatGateError):
    """Payment attempt failed"""
    pass


class BudgetExceededError(SatGateError):
    """Budget limit exceeded"""
    
    def __init__(self, message: str, used: float = 0, limit: float = 0):
        super().__init__(message)
        self.used = used
        self.limit = limit
    
    @property
    def remaining(self) -> float:
        """Remaining budget (may be negative)"""
        return self.limit - self.used


class TokenExpiredError(SatGateError):
    """Token has expired"""
    pass


class DelegationError(SatGateError):
    """Token delegation failed"""
    pass


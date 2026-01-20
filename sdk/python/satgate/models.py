"""SatGate Gateway Data Models"""

from dataclasses import dataclass
from datetime import datetime
from typing import Optional, Dict, Any


@dataclass
class Token:
    """Represents a capability token"""
    token: str
    signature: str
    scope: Optional[str] = None
    expires_at: Optional[datetime] = None
    
    def __str__(self) -> str:
        return f"Token(signature={self.signature[:16]}..., scope={self.scope})"


@dataclass
class TokenInfo:
    """Token information with usage statistics"""
    signature: str
    status: str
    total_requests: int = 0
    last_used: Optional[datetime] = None
    routes: Optional[Dict[str, int]] = None
    banned_at: Optional[datetime] = None
    ban_reason: Optional[str] = None
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "TokenInfo":
        return cls(
            signature=data.get("signature", ""),
            status=data.get("status", "unknown"),
            total_requests=data.get("totalRequests", 0),
            last_used=_parse_datetime(data.get("lastUsed")),
            routes=data.get("routes"),
            banned_at=_parse_datetime(data.get("bannedAt")),
            ban_reason=data.get("banReason")
        )
    
    @property
    def is_active(self) -> bool:
        return self.status == "active"
    
    @property
    def is_banned(self) -> bool:
        return self.status == "banned"


@dataclass
class BanRecord:
    """Represents a banned token"""
    signature: str
    reason: str
    banned_at: datetime
    banned_by: str
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "BanRecord":
        return cls(
            signature=data.get("signature", ""),
            reason=data.get("reason", ""),
            banned_at=_parse_datetime(data.get("bannedAt")) or datetime.now(),
            banned_by=data.get("bannedBy", "unknown")
        )


@dataclass
class Stats:
    """Gateway statistics"""
    total_requests: int
    total_l402: int
    total_capability: int
    total_denied: int
    total_errors: int
    active_tokens: int
    banned_tokens: int
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Stats":
        governance = data.get("governance", {})
        return cls(
            total_requests=data.get("totalRequests", 0),
            total_l402=data.get("totalL402", 0),
            total_capability=data.get("totalCapability", 0),
            total_denied=data.get("totalDenied", 0),
            total_errors=data.get("totalErrors", 0),
            active_tokens=governance.get("active", 0),
            banned_tokens=governance.get("banned", 0)
        )


def _parse_datetime(value: Optional[str]) -> Optional[datetime]:
    """Parse ISO datetime string"""
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00"))
    except:
        return None


# Economic Policy types
# SatGate uses a Layer 0/Layer 1 model:
# - Layer 0 (Default Protection): Cryptographic verification on all non-PUBLIC routes
# - Layer 1 (Economic Policy): Observe, Control, or Charge

from enum import Enum
from typing import Literal

PolicyKind = Literal[
    'public',      # Exception: No protection
    'deny',        # Exception: Block all
    'protected',   # Layer 0: Verify only (alias: capability, protect)
    'observe',     # Layer 1: Verify → meter/log (alias: chargeback, audit)
    'control',     # Layer 1: Verify → budget enforcement (alias: fiat402, budget)
    'charge',      # Layer 1: Verify → payment required (alias: l402, monetize, pay)
]


@dataclass
class RoutePolicy:
    """Route economic policy configuration"""
    kind: str  # PolicyKind
    l402_price_sats: Optional[int] = None
    budget_amount_cents: Optional[int] = None
    budget_period: Optional[str] = None  # hourly, daily, monthly


@dataclass
class RouteConfig:
    """Route configuration"""
    name: str
    path_prefix: Optional[str] = None
    path_exact: Optional[str] = None
    methods: Optional[list] = None
    policy: Optional[RoutePolicy] = None
    upstream: Optional[str] = None
    rate_limit_rps: Optional[float] = None


@dataclass
class UsageSummary:
    """Usage summary with separate observe/billable counts"""
    tenant_id: str
    period_start: datetime
    period_end: datetime
    total_requests: int
    billable_requests: int  # Control/Charge only (counts against quota)
    observe_requests: int   # Free/unlimited
    total_bytes: int
    l402_revenue_sats: int
    platform_fee_sats: int  # 2% platform fee
    net_revenue_sats: int
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "UsageSummary":
        return cls(
            tenant_id=data.get("tenantId", ""),
            period_start=_parse_datetime(data.get("periodStart")) or datetime.now(),
            period_end=_parse_datetime(data.get("periodEnd")) or datetime.now(),
            total_requests=data.get("totalRequests", 0),
            billable_requests=data.get("billableRequests", 0),
            observe_requests=data.get("observeRequests", 0),
            total_bytes=data.get("totalBytes", 0),
            l402_revenue_sats=data.get("l402RevenueSats", 0),
            platform_fee_sats=data.get("platformFeeSats", 0),
            net_revenue_sats=data.get("netRevenueSats", 0)
        )




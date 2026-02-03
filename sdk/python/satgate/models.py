"""SatGate Gateway Data Models"""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional, Dict, Any, List


@dataclass
class Token:
    """Represents a capability token"""
    token: str
    signature: str
    scope: Optional[str] = None
    expires_at: Optional[datetime] = None
    caveats: Optional[List[str]] = None
    
    def __str__(self) -> str:
        sig = self.signature[:16] if self.signature else "unknown"
        return f"Token(signature={sig}..., scope={self.scope})"


@dataclass
class TokenInfo:
    """Token information with usage statistics (from governance graph)"""
    signature: str
    status: str
    scope: Optional[str] = None
    total_requests: int = 0
    last_used: Optional[datetime] = None
    routes: Optional[Dict[str, int]] = None
    banned_at: Optional[datetime] = None
    ban_reason: Optional[str] = None
    parent_signature: Optional[str] = None
    label: Optional[str] = None
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "TokenInfo":
        return cls(
            signature=data.get("signature", data.get("id", "")),
            status=data.get("status", "unknown"),
            scope=data.get("scope"),
            total_requests=data.get("totalRequests", data.get("requests", 0)),
            last_used=_parse_datetime(data.get("lastUsed")),
            routes=data.get("routes"),
            banned_at=_parse_datetime(data.get("bannedAt")),
            ban_reason=data.get("banReason", data.get("reason")),
            parent_signature=data.get("parentSignature", data.get("parent")),
            label=data.get("label"),
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
    """Gateway statistics (from governance graph)"""
    active_tokens: int = 0
    banned_tokens: int = 0
    blocked_tokens: int = 0
    banned_hits: int = 0
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Stats":
        stats = data.get("stats", data)
        return cls(
            active_tokens=stats.get("active", 0),
            banned_tokens=stats.get("banned", 0),
            blocked_tokens=stats.get("blocked", 0),
            banned_hits=stats.get("bannedHits", 0),
        )


@dataclass
class GraphData:
    """Full governance graph data from GET /api/governance/graph"""
    nodes: List[Dict[str, Any]] = field(default_factory=list)
    edges: List[Dict[str, Any]] = field(default_factory=list)
    stats: Optional[Stats] = None
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "GraphData":
        return cls(
            nodes=data.get("nodes", []),
            edges=data.get("edges", []),
            stats=Stats.from_dict(data.get("stats", {})) if "stats" in data else None,
        )


def _parse_datetime(value: Optional[str]) -> Optional[datetime]:
    """Parse ISO datetime string"""
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00"))
    except Exception:
        return None

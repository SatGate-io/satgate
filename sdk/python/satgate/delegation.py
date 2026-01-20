"""
SatGate Delegation Helpers

Utilities for creating and managing macaroon delegation hierarchies.
Supports common patterns like team budgets, time-limited access, and scope restrictions.
"""

from dataclasses import dataclass, field
from datetime import datetime, timedelta
from typing import List, Dict, Optional, Any, TYPE_CHECKING
import re

if TYPE_CHECKING:
    from .client import SatGateClient
    from .models import Token


class Caveats:
    """Caveat builders for common delegation patterns"""
    
    @staticmethod
    def scope(scope: str) -> str:
        """
        Restrict scope to specific actions
        
        Example:
            Caveats.scope('api:read')
        """
        return f"scope = {scope}"
    
    @staticmethod
    def expires(seconds: int = None, dt: datetime = None) -> str:
        """
        Set expiration time
        
        Args:
            seconds: Seconds from now
            dt: Specific datetime
        """
        if dt:
            expiry = int(dt.timestamp())
        elif seconds:
            expiry = int(datetime.now().timestamp()) + seconds
        else:
            raise ValueError("Either 'seconds' or 'dt' must be provided")
        return f"expires = {expiry}"
    
    @staticmethod
    def routes(patterns: List[str]) -> str:
        """
        Limit to specific routes
        
        Example:
            Caveats.routes(['/api/v1/*', '/health'])
        """
        return f"routes = {','.join(patterns)}"
    
    @staticmethod
    def rate_limit(count: int, period: int = 60) -> str:
        """
        Limit requests per period
        
        Args:
            count: Max requests
            period: Period in seconds (default: 60 = per minute)
        """
        return f"ratelimit = {count}/{period}"
    
    @staticmethod
    def budget(amount: float, currency: str = "USD") -> str:
        """
        Set a budget limit (for billing/metering)
        
        Args:
            amount: Budget amount
            currency: Currency code (default: 'USD')
        """
        return f"budget = {amount} {currency}"
    
    @staticmethod
    def source_ip(cidrs: List[str]) -> str:
        """
        Restrict to specific IP addresses/CIDRs
        
        Example:
            Caveats.source_ip(['10.0.0.0/8', '192.168.1.100'])
        """
        return f"source_ip = {','.join(cidrs)}"
    
    @staticmethod
    def methods(methods: List[str]) -> str:
        """
        Restrict to specific methods
        
        Example:
            Caveats.methods(['GET', 'POST'])
        """
        return f"methods = {','.join(methods)}"
    
    @staticmethod
    def team(team_id: str) -> str:
        """Add a team/cost-center identifier for attribution"""
        return f"team = {team_id}"
    
    @staticmethod
    def project(project_id: str) -> str:
        """Add a project identifier for attribution"""
        return f"project = {project_id}"
    
    @staticmethod
    def label(key: str, value: str) -> str:
        """Add a custom label for tracking"""
        return f"label:{key} = {value}"
    
    @staticmethod
    def custom(caveat: str) -> str:
        """Add an arbitrary caveat"""
        return caveat


class DelegationBuilder:
    """
    Fluent builder for creating delegated tokens
    
    Example:
        token = (DelegationBuilder(parent_token)
            .with_scope('api:read')
            .with_expiry(3600)
            .for_team('engineering')
            .delegate(client))
    """
    
    def __init__(self, parent_token: str):
        self._parent_token = parent_token
        self._caveats: List[str] = []
    
    def with_scope(self, scope: str) -> 'DelegationBuilder':
        """Add a scope restriction"""
        self._caveats.append(Caveats.scope(scope))
        return self
    
    def with_expiry(self, seconds: int = None, dt: datetime = None) -> 'DelegationBuilder':
        """Add an expiration"""
        self._caveats.append(Caveats.expires(seconds=seconds, dt=dt))
        return self
    
    def with_routes(self, patterns: List[str]) -> 'DelegationBuilder':
        """Add route restrictions"""
        self._caveats.append(Caveats.routes(patterns))
        return self
    
    def with_rate_limit(self, count: int, period_seconds: int = 60) -> 'DelegationBuilder':
        """Add rate limiting"""
        self._caveats.append(Caveats.rate_limit(count, period_seconds))
        return self
    
    def with_budget(self, amount: float, currency: str = "USD") -> 'DelegationBuilder':
        """Add a budget limit"""
        self._caveats.append(Caveats.budget(amount, currency))
        return self
    
    def with_source_ip(self, cidrs: List[str]) -> 'DelegationBuilder':
        """Restrict to source IPs"""
        self._caveats.append(Caveats.source_ip(cidrs))
        return self
    
    def with_methods(self, methods: List[str]) -> 'DelegationBuilder':
        """Restrict to HTTP methods"""
        self._caveats.append(Caveats.methods(methods))
        return self
    
    def for_team(self, team_id: str) -> 'DelegationBuilder':
        """Add team attribution"""
        self._caveats.append(Caveats.team(team_id))
        return self
    
    def for_project(self, project_id: str) -> 'DelegationBuilder':
        """Add project attribution"""
        self._caveats.append(Caveats.project(project_id))
        return self
    
    def with_label(self, key: str, value: str) -> 'DelegationBuilder':
        """Add a custom label"""
        self._caveats.append(Caveats.label(key, value))
        return self
    
    def with_caveat(self, caveat: str) -> 'DelegationBuilder':
        """Add a raw caveat"""
        self._caveats.append(caveat)
        return self
    
    def build(self) -> Dict[str, Any]:
        """Build the delegation request dictionary"""
        return {
            "parentToken": self._parent_token,
            "caveats": self._caveats,
        }
    
    def delegate(self, client: 'SatGateClient') -> 'Token':
        """Execute the delegation via client"""
        return client.tokens.delegate(
            parent_token=self._parent_token,
            caveats=self._caveats
        )


def delegate(parent_token: str) -> DelegationBuilder:
    """
    Create a new delegation builder
    
    Args:
        parent_token: The parent token to delegate from
    
    Example:
        token = delegate(parent_token).with_scope('api:read').delegate(client)
    """
    return DelegationBuilder(parent_token)


class DelegationPatterns:
    """Common delegation patterns"""
    
    @staticmethod
    def read_only(parent_token: str) -> DelegationBuilder:
        """Create a read-only token"""
        return (delegate(parent_token)
            .with_scope('api:read')
            .with_methods(['GET', 'HEAD', 'OPTIONS']))
    
    @staticmethod
    def temporary(parent_token: str, hours: int = 24) -> DelegationBuilder:
        """Create a time-limited token (24 hours by default)"""
        return delegate(parent_token).with_expiry(seconds=hours * 3600)
    
    @staticmethod
    def team_budget(
        parent_token: str,
        team_id: str,
        budget_amount: float,
        currency: str = "USD"
    ) -> DelegationBuilder:
        """Create a team token with budget"""
        return (delegate(parent_token)
            .for_team(team_id)
            .with_budget(budget_amount, currency))
    
    @staticmethod
    def api_client(
        parent_token: str,
        client_id: str,
        requests_per_minute: int = 100
    ) -> DelegationBuilder:
        """Create a rate-limited API client token"""
        return (delegate(parent_token)
            .with_label('client_id', client_id)
            .with_rate_limit(requests_per_minute, 60))
    
    @staticmethod
    def webhook(
        parent_token: str,
        callback_path: str,
        expiry_minutes: int = 30
    ) -> DelegationBuilder:
        """Create a webhook callback token (single route, short expiry)"""
        return (delegate(parent_token)
            .with_routes([callback_path])
            .with_methods(['POST'])
            .with_expiry(seconds=expiry_minutes * 60))
    
    @staticmethod
    def cicd(
        parent_token: str,
        pipeline_id: str,
        expiry_minutes: int = 60
    ) -> DelegationBuilder:
        """Create a CI/CD pipeline token (specific routes, short-lived)"""
        return (delegate(parent_token)
            .with_label('pipeline', pipeline_id)
            .with_expiry(seconds=expiry_minutes * 60))
    
    @staticmethod
    def agent_swarm(
        parent_token: str,
        swarm_id: str,
        budget: float = None,
        currency: str = "USD",
        requests_per_minute: int = None,
        max_agents: int = None
    ) -> DelegationBuilder:
        """Create an agent swarm token (budget + rate limit)"""
        builder = (delegate(parent_token)
            .with_label('swarm_id', swarm_id))
        
        if budget is not None:
            builder = builder.with_budget(budget, currency)
        if requests_per_minute is not None:
            builder = builder.with_rate_limit(requests_per_minute, 60)
        if max_agents is not None:
            builder = builder.with_label('max_agents', str(max_agents))
        
        return builder


@dataclass
class DelegationNode:
    """Delegation tree node for visualizing delegation hierarchies"""
    signature: str
    scope: Optional[str] = None
    team: Optional[str] = None
    project: Optional[str] = None
    budget: Optional[str] = None
    expires_at: Optional[datetime] = None
    children: List['DelegationNode'] = field(default_factory=list)
    metadata: Dict[str, str] = field(default_factory=dict)


def parse_caveats(caveats: List[str]) -> Dict[str, str]:
    """Parse caveats from a token into structured data"""
    result = {}
    for caveat in caveats:
        match = re.match(r'^([^=]+)\s*=\s*(.+)$', caveat)
        if match:
            result[match.group(1).strip()] = match.group(2).strip()
    return result


def is_more_restrictive(
    parent_caveats: List[str],
    child_caveat: str
) -> bool:
    """Check if a caveat would be more restrictive than the parent"""
    key = child_caveat.split('=')[0].strip()
    parent_parsed = parse_caveats(parent_caveats)
    child_parsed = parse_caveats([child_caveat])
    
    # If parent doesn't have this caveat, child is more restrictive
    if key not in parent_parsed:
        return True
    
    # For numeric caveats, child must be <= parent
    numeric_caveats = ['budget', 'ratelimit', 'expires']
    if any(nc in key for nc in numeric_caveats):
        try:
            parent_val = float(parent_parsed[key].split()[0])
        except (ValueError, IndexError):
            parent_val = float('inf')
        try:
            child_val = float(child_parsed[key].split()[0])
        except (ValueError, IndexError):
            child_val = 0
        return child_val <= parent_val
    
    # For scope caveats, child must be subset of parent
    if key == 'scope':
        parent_scopes = parent_parsed[key].split(',')
        child_scopes = child_parsed[key].split(',')
        return all(
            any(
                ps.strip() == cs.strip() or 
                ps.strip() == '*' or 
                (ps.strip().endswith(':*') and cs.strip().startswith(ps.strip()[:-1]))
                for ps in parent_scopes
            )
            for cs in child_scopes
        )
    
    return True  # Default to allowing

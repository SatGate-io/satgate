"""SatGate Gateway Python Client"""

import requests
from typing import Optional, List, Dict, Any
from dataclasses import dataclass
from datetime import datetime
from .models import Token, TokenInfo, BanRecord, Stats
from .exceptions import SatGateError, AuthenticationError, NotFoundError


class SatGateClient:
    """
    SatGate Gateway Admin API Client
    
    Args:
        base_url: Gateway admin API URL (e.g., "http://localhost:9090")
        admin_token: Admin API authentication token
        timeout: Request timeout in seconds (default: 30)
    """
    
    def __init__(
        self,
        base_url: str,
        admin_token: str,
        timeout: int = 30
    ):
        self.base_url = base_url.rstrip("/")
        self.admin_token = admin_token
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update({
            "Content-Type": "application/json",
            "X-Admin-Token": admin_token
        })
        
        # Initialize services
        self.tokens = TokensService(self)
        self.governance = GovernanceService(self)
        self.config = ConfigService(self)
        self.stats = StatsService(self)
    
    def _request(
        self,
        method: str,
        path: str,
        data: Optional[Dict] = None
    ) -> Dict[str, Any]:
        """Make an API request"""
        url = f"{self.base_url}{path}"
        
        try:
            response = self.session.request(
                method=method,
                url=url,
                json=data,
                timeout=self.timeout
            )
        except requests.RequestException as e:
            raise SatGateError(f"Request failed: {e}")
        
        if response.status_code == 401:
            raise AuthenticationError("Invalid admin token")
        elif response.status_code == 404:
            raise NotFoundError("Resource not found")
        elif response.status_code >= 400:
            error_msg = response.json().get("error", "Unknown error")
            raise SatGateError(f"API error {response.status_code}: {error_msg}")
        
        if response.content:
            return response.json()
        return {}
    
    def health(self) -> bool:
        """Check if gateway is healthy"""
        try:
            result = self._request("GET", "/healthz")
            return result.get("status") == "ok"
        except:
            return False


class TokensService:
    """Token management operations"""
    
    def __init__(self, client: SatGateClient):
        self._client = client
    
    def mint(
        self,
        scope: str = "api:*",
        expires_in: int = 3600
    ) -> Token:
        """
        Mint a new root token
        
        Args:
            scope: Token scope (e.g., "api:read", "api:*")
            expires_in: Token lifetime in seconds
            
        Returns:
            Token object with token string and signature
        """
        result = self._client._request("POST", "/api/v1/tokens", {
            "scope": scope,
            "expiresIn": expires_in
        })
        return Token(
            token=result["token"],
            signature=result["signature"],
            scope=result.get("scope", scope),
            expires_at=datetime.fromisoformat(result["expiresAt"].replace("Z", "+00:00"))
        )
    
    def list(self) -> List[TokenInfo]:
        """List all tokens"""
        result = self._client._request("GET", "/api/v1/tokens")
        return [TokenInfo.from_dict(t) for t in result]
    
    def get(self, signature: str) -> TokenInfo:
        """Get token details by signature"""
        result = self._client._request("GET", f"/api/v1/tokens/{signature}")
        return TokenInfo.from_dict(result)
    
    def revoke(self, signature: str) -> None:
        """Revoke a token"""
        self._client._request("DELETE", f"/api/v1/tokens/{signature}")
    
    def delegate(
        self,
        parent_token: str,
        caveats: Optional[List[str]] = None
    ) -> Token:
        """
        Create a child token via delegation
        
        Args:
            parent_token: Base64-encoded parent token
            caveats: Additional caveats to add
            
        Returns:
            Token object with child token
        """
        result = self._client._request("POST", "/api/v1/tokens/delegate", {
            "parentToken": parent_token,
            "caveats": caveats or []
        })
        return Token(
            token=result["token"],
            signature=result["signature"]
        )


class GovernanceService:
    """Token governance operations"""
    
    def __init__(self, client: SatGateClient):
        self._client = client
    
    def ban(self, signature: str, reason: str = "") -> None:
        """Ban a token"""
        self._client._request("POST", "/api/v1/governance/ban", {
            "tokenSignature": signature,
            "reason": reason
        })
    
    def unban(self, signature: str) -> None:
        """Remove a token from the ban list"""
        self._client._request("DELETE", f"/api/v1/governance/ban/{signature}")
    
    def get_banlist(self) -> List[BanRecord]:
        """Get all banned tokens"""
        result = self._client._request("GET", "/api/v1/governance/banlist")
        return [BanRecord.from_dict(b) for b in result]
    
    def reset(self) -> None:
        """Reset all governance data"""
        self._client._request("POST", "/api/v1/governance/reset")


class ConfigService:
    """Configuration operations"""
    
    def __init__(self, client: SatGateClient):
        self._client = client
    
    def get(self) -> Dict[str, Any]:
        """Get current configuration"""
        return self._client._request("GET", "/api/v1/config")
    
    def validate(self, config: Dict[str, Any]) -> bool:
        """Validate a configuration"""
        try:
            self._client._request("POST", "/api/v1/config/validate", config)
            return True
        except SatGateError:
            return False


class StatsService:
    """Statistics operations"""
    
    def __init__(self, client: SatGateClient):
        self._client = client
    
    def get(self) -> Stats:
        """Get gateway statistics"""
        result = self._client._request("GET", "/api/v1/stats")
        return Stats.from_dict(result)
    
    def get_token_stats(self) -> List[TokenInfo]:
        """Get per-token statistics"""
        result = self._client._request("GET", "/api/v1/stats/tokens")
        return [TokenInfo.from_dict(t) for t in result]




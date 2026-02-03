"""SatGate Gateway Python Client (OSS)

Admin client for the SatGate OSS Gateway. Provides access to:
- Token minting (POST /api/capability/mint)
- Token validation (POST /api/capability/validate)
- Token delegation (POST /api/capability/delegate)
- Governance: ban, graph, reset
- Health checks
"""

import requests
from typing import Optional, List, Dict, Any
from dataclasses import dataclass
from datetime import datetime
from .models import Token, TokenInfo, BanRecord, Stats
from .exceptions import SatGateError, AuthenticationError, NotFoundError


class SatGateClient:
    """
    SatGate Gateway Admin API Client (OSS)
    
    Args:
        base_url: Gateway URL (e.g., "http://localhost:8080")
        admin_token: Admin API authentication token (sent as X-Admin-Token header)
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
    
    def _request(
        self,
        method: str,
        path: str,
        data: Optional[Dict] = None,
        include_admin_token: bool = True
    ) -> Dict[str, Any]:
        """Make an API request"""
        url = f"{self.base_url}{path}"
        
        headers = {}
        if include_admin_token:
            headers["X-Admin-Token"] = self.admin_token
        
        try:
            response = self.session.request(
                method=method,
                url=url,
                json=data,
                headers=headers,
                timeout=self.timeout
            )
        except requests.RequestException as e:
            raise SatGateError(f"Request failed: {e}")
        
        if response.status_code == 401:
            raise AuthenticationError("Invalid admin token")
        elif response.status_code == 404:
            raise NotFoundError("Resource not found")
        elif response.status_code >= 400:
            try:
                error_msg = response.json().get("error", "Unknown error")
            except Exception:
                error_msg = response.text
            raise SatGateError(f"API error {response.status_code}: {error_msg}")
        
        if response.content:
            return response.json()
        return {}
    
    def health(self) -> bool:
        """Check if gateway is healthy"""
        try:
            result = self._request("GET", "/health", include_admin_token=False)
            return result.get("status") == "healthy"
        except Exception:
            return False
    
    def ping(self, token: str) -> Dict[str, Any]:
        """
        Ping the gateway with a capability token to verify it's valid.
        
        Args:
            token: Bearer token to validate
            
        Returns:
            dict with validation result
        """
        url = f"{self.base_url}/api/capability/ping"
        try:
            resp = self.session.get(
                url,
                headers={"Authorization": f"Bearer {token}"},
                timeout=self.timeout
            )
            if resp.status_code != 200:
                return {"status": "error", "code": resp.status_code}
            return resp.json()
        except requests.RequestException as e:
            raise SatGateError(f"Ping failed: {e}")


class TokensService:
    """Token management operations (OSS)"""
    
    def __init__(self, client: SatGateClient):
        self._client = client
    
    def mint(
        self,
        scope: str = "api:*",
        duration: str = "1h"
    ) -> Token:
        """
        Mint a new capability token.
        
        Uses POST /api/capability/mint with X-Admin-Token header.
        
        Args:
            scope: Token scope (e.g., "api:read", "api:*", "api:capability:admin")
            duration: Token lifetime as a Go duration string (e.g., "1h", "30m", "24h")
            
        Returns:
            Token object with token string, signature, scope, and expiry
        """
        result = self._client._request("POST", "/api/capability/mint", {
            "scope": scope,
            "duration": duration,
        })
        return Token(
            token=result["token"],
            signature=result["signature"],
            scope=result.get("scope", scope),
            expires_at=datetime.fromisoformat(
                result["expiresAt"].replace("Z", "+00:00")
            ) if "expiresAt" in result else None
        )
    
    def validate(self, token: str) -> Dict[str, Any]:
        """
        Validate a capability token.
        
        Uses POST /api/capability/validate.
        
        Args:
            token: The token string to validate
            
        Returns:
            dict with valid (bool), identifier, and caveats
        """
        return self._client._request(
            "POST", "/api/capability/validate",
            {"token": token},
            include_admin_token=False
        )
    
    def delegate(
        self,
        parent_token: str,
        caveats: Optional[List[str]] = None
    ) -> Token:
        """
        Create a child token via delegation.
        
        Uses POST /api/capability/delegate.
        
        Args:
            parent_token: The parent token to delegate from
            caveats: Additional caveats to add (e.g., ["scope = api:read"])
            
        Returns:
            Token object with the delegated child token
        """
        result = self._client._request(
            "POST", "/api/capability/delegate",
            {
                "parentToken": parent_token,
                "caveats": caveats or [],
            },
            include_admin_token=False
        )
        return Token(
            token=result["token"],
            signature=result.get("signature", ""),
            caveats=result.get("caveats"),
        )


class GovernanceService:
    """Token governance operations (OSS)"""
    
    def __init__(self, client: SatGateClient):
        self._client = client
    
    def ban(self, signature: str, reason: str = "") -> None:
        """
        Ban a token by its signature.
        
        Uses POST /api/governance/ban with X-Admin-Token header.
        
        Args:
            signature: The hex-encoded token signature to ban
            reason: Reason for banning
        """
        self._client._request("POST", "/api/governance/ban", {
            "tokenSignature": signature,
            "reason": reason
        })
    
    def get_graph(self) -> Dict[str, Any]:
        """
        Get the token governance graph (nodes, edges, stats).
        
        Uses GET /api/governance/graph. Returns data suitable
        for rendering a delegation tree / dashboard.
        
        Returns:
            dict with nodes, edges, and stats
        """
        return self._client._request(
            "GET", "/api/governance/graph",
            include_admin_token=False
        )
    
    def reset(self) -> None:
        """
        Reset all governance data (tokens, bans, usage).
        
        Uses POST /api/governance/reset with X-Admin-Token header.
        """
        self._client._request("POST", "/api/governance/reset")

"""
SatGate Agent Client - Economic Middleware for AI Agents

This module provides the SatGateAgentClient, which automatically handles:
- Admin token authentication for minting (OSS gateway)
- 402 Payment Required challenges (L402)
- Token caching and refresh
- Offline delegation for worker agents
- Budget tracking and alerts

Example usage (OSS Gateway):
    from satgate import SatGateAgentClient
    
    # With admin token (OSS mode)
    client = SatGateAgentClient(
        gateway_url="http://localhost:8080",
        admin_token="your-admin-token"
    )
    
    # Make requests through the gateway
    response = client.get("/api/data")
    
    # With Lightning wallet for L402 endpoints
    from satgate.agent_client import LNDWallet
    
    client = SatGateAgentClient(
        gateway_url="http://localhost:8080",
        admin_token="your-admin-token",
        wallet=LNDWallet(macaroon_path="~/.lnd/admin.macaroon")
    )
"""

import base64
import json
import os
import time
from dataclasses import dataclass, field
from typing import Any, Callable, Dict, List, Optional, Union
from urllib.parse import urljoin

import requests

from .exceptions import (
    SatGateError,
    AuthenticationError,
    BudgetExceededError,
    PaymentRequiredError,
    PaymentFailedError,
)


@dataclass
class AgentResponse:
    """Response from a SatGate-protected API call"""
    status_code: int
    headers: Dict[str, str]
    content: bytes
    cost: float = 0.0
    budget_remaining: Optional[float] = None
    
    def json(self) -> Any:
        """Parse response as JSON"""
        return json.loads(self.content)
    
    def text(self) -> str:
        """Return response as text"""
        return self.content.decode('utf-8')


@dataclass
class CachedToken:
    """A cached macaroon with metadata"""
    token: str
    signature: str
    expires_at: float
    scope: str = "api:*"
    budget_limit: Optional[float] = None
    budget_used: float = 0.0
    caveats: List[str] = field(default_factory=list)
    
    @property
    def is_expired(self) -> bool:
        return time.time() > self.expires_at
    
    @property
    def budget_remaining(self) -> Optional[float]:
        if self.budget_limit is None:
            return None
        return max(0, self.budget_limit - self.budget_used)


class LightningWallet:
    """Base class for Lightning wallets"""
    
    def pay_invoice(self, invoice: str) -> str:
        """Pay a Lightning invoice and return the preimage"""
        raise NotImplementedError
    
    def get_balance(self) -> int:
        """Return balance in satoshis"""
        raise NotImplementedError


class LNDWallet(LightningWallet):
    """LND Lightning wallet"""
    
    def __init__(
        self,
        host: str = "localhost:10009",
        macaroon_path: Optional[str] = None,
        macaroon_hex: Optional[str] = None,
        cert_path: Optional[str] = None,
        use_rest: bool = True
    ):
        self.host = host
        self.use_rest = use_rest
        
        # Load macaroon
        if macaroon_hex:
            self.macaroon = macaroon_hex
        elif macaroon_path:
            with open(macaroon_path, 'rb') as f:
                self.macaroon = f.read().hex()
        else:
            raise SatGateError("LND macaroon required")
        
        # Load cert
        self.cert_path = cert_path
    
    def pay_invoice(self, invoice: str) -> str:
        if self.use_rest:
            return self._pay_rest(invoice)
        else:
            return self._pay_grpc(invoice)
    
    def _pay_rest(self, invoice: str) -> str:
        url = f"https://{self.host}/v1/channels/transactions"
        headers = {"Grpc-Metadata-macaroon": self.macaroon}
        data = {"payment_request": invoice}
        
        resp = requests.post(
            url,
            json=data,
            headers=headers,
            verify=self.cert_path or False,
            timeout=60
        )
        
        if resp.status_code != 200:
            raise PaymentFailedError(f"LND payment failed: {resp.text}")
        
        result = resp.json()
        if result.get('payment_error'):
            raise PaymentFailedError(f"Payment error: {result['payment_error']}")
        
        return result.get('payment_preimage', '')
    
    def _pay_grpc(self, invoice: str) -> str:
        raise NotImplementedError("gRPC support requires lndgrpc package")
    
    def get_balance(self) -> int:
        url = f"https://{self.host}/v1/balance/channels"
        headers = {"Grpc-Metadata-macaroon": self.macaroon}
        
        resp = requests.get(
            url,
            headers=headers,
            verify=self.cert_path or False,
            timeout=10
        )
        
        if resp.status_code != 200:
            raise SatGateError(f"Failed to get balance: {resp.text}")
        
        return int(resp.json().get('balance', 0))


class AlbyWallet(LightningWallet):
    """Alby wallet using NWC or API"""
    
    def __init__(self, access_token: Optional[str] = None, nwc_url: Optional[str] = None):
        self.access_token = access_token or os.environ.get('ALBY_ACCESS_TOKEN')
        self.nwc_url = nwc_url or os.environ.get('ALBY_NWC_URL')
        
        if not self.access_token and not self.nwc_url:
            raise SatGateError("Alby access token or NWC URL required")
    
    def pay_invoice(self, invoice: str) -> str:
        if self.nwc_url:
            return self._pay_nwc(invoice)
        else:
            return self._pay_api(invoice)
    
    def _pay_api(self, invoice: str) -> str:
        url = "https://api.getalby.com/payments/bolt11"
        headers = {"Authorization": f"Bearer {self.access_token}"}
        data = {"invoice": invoice}
        
        resp = requests.post(url, json=data, headers=headers, timeout=60)
        
        if resp.status_code != 200:
            raise PaymentFailedError(f"Alby payment failed: {resp.text}")
        
        result = resp.json()
        return result.get('payment_preimage', '')
    
    def _pay_nwc(self, invoice: str) -> str:
        raise NotImplementedError("NWC support requires nostr package")
    
    def get_balance(self) -> int:
        url = "https://api.getalby.com/balance"
        headers = {"Authorization": f"Bearer {self.access_token}"}
        
        resp = requests.get(url, headers=headers, timeout=10)
        
        if resp.status_code != 200:
            raise SatGateError(f"Failed to get balance: {resp.text}")
        
        return int(resp.json().get('balance', 0))


class SatGateAgentClient:
    """
    SatGate Agent Client - Economic Middleware for AI Agents
    
    For the OSS gateway, uses admin token authentication to mint
    capability tokens, then uses those tokens for API access.
    
    Automatically handles:
    - Token minting via POST /api/capability/mint (X-Admin-Token)
    - Token delegation via POST /api/capability/delegate
    - L402 Payment Required challenges
    - Token caching and refresh
    - Budget tracking
    """
    
    def __init__(
        self,
        gateway_url: str,
        admin_token: Optional[str] = None,
        token: Optional[str] = None,
        wallet: Optional[LightningWallet] = None,
        budget_limit: Optional[float] = None,
        scope: str = "api:*",
        duration: str = "1h",
        timeout: int = 30,
        max_retries: int = 3,
        on_budget_alert: Optional[Callable[[float, float], None]] = None,
    ):
        """
        Initialize the agent client.
        
        Args:
            gateway_url: Base URL of the SatGate gateway
            admin_token: Admin token for minting capability tokens (X-Admin-Token header).
                         Can also be set via SATGATE_ADMIN_TOKEN environment variable.
            token: Pre-existing capability token to use (skips minting).
                   Can also be set via SATGATE_TOKEN environment variable.
            wallet: Lightning wallet for L402 payments (optional)
            budget_limit: Maximum budget to spend (optional)
            scope: Default scope for minted tokens (default: "api:*")
            duration: Default duration for minted tokens (default: "1h")
            timeout: Request timeout in seconds
            max_retries: Maximum retry attempts for 402 challenges
            on_budget_alert: Callback when budget is running low
        """
        self.gateway_url = gateway_url.rstrip('/')
        self.admin_token = admin_token or os.environ.get('SATGATE_ADMIN_TOKEN')
        self.wallet = wallet
        self.budget_limit = budget_limit
        self.default_scope = scope
        self.default_duration = duration
        self.timeout = timeout
        self.max_retries = max_retries
        self.on_budget_alert = on_budget_alert
        
        # Session for connection pooling
        self.session = requests.Session()
        
        # Token cache
        self._token_cache: Optional[CachedToken] = None
        
        # If a pre-existing token was provided, cache it
        pre_token = token or os.environ.get('SATGATE_TOKEN')
        if pre_token:
            self._token_cache = CachedToken(
                token=pre_token,
                signature="pre-provided",
                expires_at=time.time() + 3600,
                scope=scope,
            )
        
        # Total cost tracking
        self._total_cost: float = 0.0
    
    def _get_token(self) -> Optional[str]:
        """Get a valid token, minting if necessary"""
        # Check cache
        if self._token_cache and not self._token_cache.is_expired:
            return self._token_cache.token
        
        # No admin token - can't mint
        if not self.admin_token:
            return None
        
        # Mint new token via OSS endpoint
        try:
            resp = self.session.post(
                f"{self.gateway_url}/api/capability/mint",
                json={
                    "scope": self.default_scope,
                    "duration": self.default_duration,
                },
                headers={
                    "Content-Type": "application/json",
                    "X-Admin-Token": self.admin_token,
                },
                timeout=self.timeout
            )
            
            if resp.status_code != 200:
                raise AuthenticationError(f"Mint failed ({resp.status_code}): {resp.text}")
            
            result = resp.json()
            
            # Parse expiration
            expires_at = time.time() + 3600  # Default 1 hour
            if 'expiresAt' in result:
                try:
                    from datetime import datetime
                    dt = datetime.fromisoformat(result['expiresAt'].replace('Z', '+00:00'))
                    expires_at = dt.timestamp()
                except (ValueError, ImportError):
                    pass
            
            # Cache token
            self._token_cache = CachedToken(
                token=result['token'],
                signature=result.get('signature', ''),
                expires_at=expires_at,
                scope=result.get('scope', self.default_scope),
            )
            
            return self._token_cache.token
            
        except requests.RequestException as e:
            raise AuthenticationError(f"Failed to mint token: {e}")
    
    def _handle_l402(self, response: requests.Response, method: str, url: str, **kwargs) -> AgentResponse:
        """Handle an L402 Payment Required response"""
        www_auth = response.headers.get('WWW-Authenticate', '')
        
        challenge = {}
        content_type = response.headers.get('Content-Type', '')
        if 'application/json' in content_type:
            try:
                challenge = response.json()
            except Exception:
                pass
        
        if not self.wallet:
            raise PaymentRequiredError(
                "L402 payment required but no wallet configured",
                challenge=challenge
            )
        
        # Parse L402 header: L402 macaroon="...", invoice="..."
        macaroon = ""
        invoice = ""
        
        for part in www_auth.split(','):
            part = part.strip()
            if 'macaroon=' in part:
                macaroon = part.split('=', 1)[1].strip('"\'')
            elif 'invoice=' in part:
                invoice = part.split('=', 1)[1].strip('"\'')
        
        # Also check JSON challenge
        if not invoice:
            invoice = challenge.get('invoice', '')
        if not macaroon:
            macaroon = challenge.get('macaroon', '')
        
        if not invoice:
            raise PaymentRequiredError("No invoice in L402 challenge", challenge=challenge)
        
        # Pay the invoice
        try:
            preimage = self.wallet.pay_invoice(invoice)
        except Exception as e:
            raise PaymentFailedError(f"Lightning payment failed: {e}")
        
        # Build L402 token and retry
        l402_token = f"L402 {macaroon}:{preimage}"
        headers = kwargs.get('headers', {})
        headers['Authorization'] = l402_token
        kwargs['headers'] = headers
        
        retry_resp = self.session.request(method, url, timeout=self.timeout, **kwargs)
        
        cost_sats = challenge.get('amount_sats', challenge.get('amount', 0))
        
        return AgentResponse(
            status_code=retry_resp.status_code,
            headers=dict(retry_resp.headers),
            content=retry_resp.content,
            cost=cost_sats
        )
    
    def request(
        self,
        method: str,
        path: str,
        **kwargs
    ) -> AgentResponse:
        """
        Make an HTTP request with automatic token and L402 handling.
        
        Args:
            method: HTTP method (GET, POST, etc.)
            path: URL path (will be joined with gateway_url)
            **kwargs: Additional arguments passed to requests
            
        Returns:
            AgentResponse with status, content, and cost information
        """
        url = urljoin(self.gateway_url + "/", path.lstrip("/"))
        
        # Add capability token if available
        token = self._get_token()
        if token:
            headers = kwargs.get('headers', {})
            if 'Authorization' not in headers:
                headers['Authorization'] = f"Bearer {token}"
                kwargs['headers'] = headers
        
        # Make request
        for attempt in range(self.max_retries):
            resp = self.session.request(method, url, timeout=self.timeout, **kwargs)
            
            # Success
            if resp.status_code < 400:
                return AgentResponse(
                    status_code=resp.status_code,
                    headers=dict(resp.headers),
                    content=resp.content,
                    cost=0.0,
                    budget_remaining=self._token_cache.budget_remaining if self._token_cache else None
                )
            
            # Payment required (L402)
            if resp.status_code == 402:
                return self._handle_l402(resp, method, url, **kwargs)
            
            # Auth error - try refreshing token
            if resp.status_code == 401 and attempt < self.max_retries - 1:
                self._token_cache = None
                token = self._get_token()
                if token:
                    headers = kwargs.get('headers', {})
                    headers['Authorization'] = f"Bearer {token}"
                    kwargs['headers'] = headers
                continue
            
            # Other error
            raise SatGateError(f"Request failed: {resp.status_code} {resp.text}")
        
        raise SatGateError("Max retries exceeded")
    
    def get(self, path: str, **kwargs) -> AgentResponse:
        """Make a GET request"""
        return self.request("GET", path, **kwargs)
    
    def post(self, path: str, **kwargs) -> AgentResponse:
        """Make a POST request"""
        return self.request("POST", path, **kwargs)
    
    def put(self, path: str, **kwargs) -> AgentResponse:
        """Make a PUT request"""
        return self.request("PUT", path, **kwargs)
    
    def delete(self, path: str, **kwargs) -> AgentResponse:
        """Make a DELETE request"""
        return self.request("DELETE", path, **kwargs)
    
    def ping(self) -> dict:
        """
        Ping the gateway to verify the capability token is valid.
        
        Calls GET /api/capability/ping with the Bearer token.
        
        Returns:
            dict with validation result
        """
        response = self.get("/api/capability/ping")
        return response.json()
    
    def validate_token(self, token: Optional[str] = None) -> dict:
        """
        Validate a capability token.
        
        Args:
            token: Token to validate. If not provided, uses the cached token.
            
        Returns:
            dict with validation result including caveats
        """
        t = token or (self._token_cache.token if self._token_cache else None)
        if not t:
            raise SatGateError("No token available to validate")
        
        resp = self.session.post(
            f"{self.gateway_url}/api/capability/validate",
            json={"token": t},
            timeout=self.timeout
        )
        
        if resp.status_code != 200:
            raise SatGateError(f"Validation failed: {resp.text}")
        
        return resp.json()
    
    def delegate(
        self,
        caveats: Optional[List[str]] = None,
    ) -> str:
        """
        Create a delegated (child) token for a worker agent.
        
        Uses POST /api/capability/delegate with the current token as parent.
        
        Args:
            caveats: Additional caveats/restrictions to add to the child token
            
        Returns:
            The delegated child token string
        """
        if not self._token_cache:
            self._get_token()
        
        if not self._token_cache:
            raise SatGateError("No token available for delegation")
        
        resp = self.session.post(
            f"{self.gateway_url}/api/capability/delegate",
            json={
                "parentToken": self._token_cache.token,
                "caveats": caveats or [],
            },
            timeout=self.timeout
        )
        
        if resp.status_code != 200:
            raise SatGateError(f"Delegation failed: {resp.text}")
        
        return resp.json()['token']
    
    @property
    def total_cost(self) -> float:
        """Total cost incurred by this client"""
        return self._total_cost
    
    @property
    def budget_remaining(self) -> Optional[float]:
        """Remaining budget (if budget_limit was set)"""
        if self.budget_limit is None:
            return None
        return max(0, self.budget_limit - self._total_cost)
    
    @property
    def current_token(self) -> Optional[str]:
        """The currently cached capability token, if any"""
        if self._token_cache and not self._token_cache.is_expired:
            return self._token_cache.token
        return None
    
    @property
    def current_signature(self) -> Optional[str]:
        """The signature of the currently cached token, if any"""
        if self._token_cache:
            return self._token_cache.signature
        return None

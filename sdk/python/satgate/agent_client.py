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
import logging
import os
import sys
import time
from dataclasses import dataclass, field
from typing import Any, Callable, Dict, List, Optional, Union
from urllib.parse import urljoin

import requests

from ._http import send_without_redirects
from .exceptions import (
    SatGateError,
    AuthenticationError,
    BudgetExceededError,
    PaymentRequiredError,
    PaymentFailedError,
)

logger = logging.getLogger(__name__)


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


def _resolve_cert_path(
    tls_cert_path: Optional[str], cert_path: Optional[str]
) -> Optional[str]:
    """Return the one certificate path given, or None. cert_path is an alias."""
    if (
        tls_cert_path is not None
        and cert_path is not None
        and os.path.expanduser(tls_cert_path) != os.path.expanduser(cert_path)
    ):
        raise SatGateError("tls_cert_path and cert_path name different files; pass one")
    path = tls_cert_path if tls_cert_path is not None else cert_path
    if path is None:
        return None
    if not isinstance(path, str) or path == "":
        raise SatGateError("tls_cert_path must be a non-empty path")
    path = os.path.expanduser(path)
    if not os.path.isfile(path):
        raise SatGateError(f"tls_cert_path {path} is not a file")
    return path


def _tls_verify_setting(cert_path: Optional[str], insecure: bool) -> Union[bool, str]:
    """The value passed to requests as verify=. Only insecure=True gives False."""
    if insecure:
        return False
    if cert_path is not None:
        return cert_path
    return True


class LightningWallet:
    """Base class for Lightning wallets"""
    
    def pay_invoice(self, invoice: str) -> str:
        """Pay a Lightning invoice and return the preimage"""
        raise NotImplementedError
    
    def get_balance(self) -> int:
        """Return balance in satoshis"""
        raise NotImplementedError


class LNDWallet(LightningWallet):
    """LND Lightning wallet (REST).

    TLS is always verified unless ``insecure=True`` is passed:

    * with no certificate path, the LND REST host must present a certificate
      the system trusts; one it does not trust fails with ``SatGateError``;
    * with ``tls_cert_path`` (or its alias ``cert_path``), the host is verified
      against that file, for example LND's own ``tls.cert``;
    * ``insecure=True`` turns verification off and logs a warning naming the
      host. It cannot be combined with a certificate path.
    """
    
    def __init__(
        self,
        host: str = "localhost:10009",
        macaroon_path: Optional[str] = None,
        macaroon_hex: Optional[str] = None,
        cert_path: Optional[str] = None,
        use_rest: bool = True,
        *,
        tls_cert_path: Optional[str] = None,
        insecure: bool = False,
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
        
        self.cert_path = _resolve_cert_path(tls_cert_path, cert_path)
        self.tls_cert_path = self.cert_path
        if insecure is not True and insecure is not False:
            raise SatGateError("insecure must be True or False")
        if insecure and self.cert_path is not None:
            raise SatGateError("insecure=True cannot be combined with tls_cert_path")
        self.insecure = insecure
        self._verify = _tls_verify_setting(self.cert_path, insecure)
        if insecure:
            logger.warning(
                "LNDWallet: TLS certificate verification is disabled for %s "
                "(insecure=True); the macaroon is sent to an unverified server",
                self.host,
            )

    def _request(self, method: str, url: str, **kwargs) -> requests.Response:
        """Send one request carrying the macaroon header to the LND host.

        Redirects are never followed (send_without_redirects): the macaroon
        is sent only to the configured host, once. A 3xx answer raises
        RedirectRefusedError naming the status and the host.
        """
        try:
            return send_without_redirects(
                method, url, host=self.host, label="LND host",
                verify=self._verify, **kwargs
            )
        except requests.exceptions.SSLError as e:
            if self.cert_path is not None:
                hint = f"the certificate in {self.cert_path} does not verify it"
            else:
                hint = (
                    "its certificate is not trusted by the system; pass "
                    "tls_cert_path= (for example LND's tls.cert)"
                )
            raise SatGateError(
                f"TLS verification failed for LND host {self.host}: {hint}"
            ) from e
    
    def pay_invoice(self, invoice: str) -> str:
        if self.use_rest:
            return self._pay_rest(invoice)
        else:
            return self._pay_grpc(invoice)
    
    def _pay_rest(self, invoice: str) -> str:
        url = f"https://{self.host}/v1/channels/transactions"
        headers = {"Grpc-Metadata-macaroon": self.macaroon}
        data = {"payment_request": invoice}
        
        resp = self._request(
            "POST",
            url,
            json=data,
            headers=headers,
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
        
        resp = self._request(
            "GET",
            url,
            headers=headers,
            timeout=10
        )
        
        if resp.status_code != 200:
            raise SatGateError(f"Failed to get balance: {resp.text}")
        
        return int(resp.json().get('balance', 0))


class AlbyWallet(LightningWallet):
    """Alby wallet using NWC or API"""

    # The Alby API the access token is sent to. Requests to it never follow
    # a redirect.
    _api_base = "https://api.getalby.com"
    
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
        url = f"{self._api_base}/payments/bolt11"
        headers = {"Authorization": f"Bearer {self.access_token}"}
        data = {"invoice": invoice}
        
        resp = send_without_redirects(
            "POST", url, label="Alby API host", json=data, headers=headers, timeout=60
        )
        
        if resp.status_code != 200:
            raise PaymentFailedError(f"Alby payment failed: {resp.text}")
        
        result = resp.json()
        return result.get('payment_preimage', '')
    
    def _pay_nwc(self, invoice: str) -> str:
        raise NotImplementedError("NWC support requires nostr package")
    
    def get_balance(self) -> int:
        url = f"{self._api_base}/balance"
        headers = {"Authorization": f"Bearer {self.access_token}"}
        
        resp = send_without_redirects(
            "GET", url, label="Alby API host", headers=headers, timeout=10
        )
        
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
    
    def _send(self, method: str, url: str, **kwargs) -> requests.Response:
        """Send one gateway request through the session, never following a
        redirect. Every request this client makes with a credential (the
        capability token in a header or body, an L402 token, the admin token)
        goes through here; a 3xx answer raises RedirectRefusedError naming the
        status and the gateway host.
        """
        return send_without_redirects(
            method, url, session=self.session, label="SatGate gateway", **kwargs
        )

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
            resp = self._send(
                "POST",
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
        
        retry_resp = self._send(method, url, timeout=self.timeout, **kwargs)
        
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
            resp = self._send(method, url, timeout=self.timeout, **kwargs)
            
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
        
        resp = self._send(
            "POST",
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
        
        resp = self._send(
            "POST",
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

    def stage_policy(self, policy: dict) -> dict:
        """Stage a policy. It is not live until promote."""
        return self._admin_json("POST", "/api/agent/policy", policy)

    def simulate_policy(self, policy_id: str, **fields) -> dict:
        """Run a staged policy against its upstream and return the signed pack."""
        body = {"policy_id": policy_id}
        body.update({key: value for key, value in fields.items() if value is not None})
        return self._admin_json("POST", "/api/agent/simulate", body)

    def promote_policy(self, policy_id: str) -> dict:
        """Make a simulated policy live. Does not invent a price."""
        return self._admin_json("POST", "/api/agent/promote", {"policy_id": policy_id})

    def fetch_pack(self, pack_id: str) -> dict:
        """Fetch a pack the gateway already signed."""
        return self._admin_json("GET", "/api/agent/pack/" + pack_id)

    def call(self, method: str, path: str, body: Optional[dict] = None, token: Optional[str] = None) -> dict:
        """Call a promoted route. Returns the upstream body and the live pack id."""
        headers = {}
        if token:
            headers["Authorization"] = "Bearer " + token
        resp = self._send(
            method,
            self.gateway_url.rstrip("/") + path,
            json=body,
            headers=headers,
            timeout=self.timeout,
        )
        parsed = None
        if resp.content:
            try:
                parsed = resp.json()
            except ValueError:
                parsed = resp.text
        return {
            "status_code": resp.status_code,
            "body": parsed,
            "pack_id": resp.headers.get("SatGate-Evidence-Pack"),
        }

    def mint_capability(self, scope: str, duration: str = "1h") -> str:
        """Mint a real capability token. This does not call an identity provider."""
        minted = self._admin_json("POST", "/api/capability/mint", {"scope": scope, "duration": duration})
        token = minted.get("token")
        if not token:
            raise SatGateError("mint returned no token")
        return token

    def fetch_jwks(self) -> dict:
        """Fetch the gateway's published signing key."""
        resp = self.session.get(
            self.gateway_url.rstrip("/") + "/.well-known/satgate-agent-jwks.json",
            timeout=self.timeout,
        )
        if resp.status_code >= 400:
            raise SatGateError(f"jwks failed: {resp.status_code} {resp.text}")
        return resp.json()

    def verify_pack(self, pack: dict, jwks: dict, require_trusted_issuer: bool = True) -> dict:
        """Verify a pack with the public verifier. SATGATE_VERIFIER must point at that script."""
        import subprocess
        import tempfile
        script = os.environ.get("SATGATE_VERIFIER")
        if not script:
            raise SatGateError("SATGATE_VERIFIER is not set")
        with tempfile.TemporaryDirectory() as tmp:
            pack_path = os.path.join(tmp, "pack.json")
            jwks_path = os.path.join(tmp, "jwks.json")
            with open(pack_path, "w") as handle:
                json.dump(pack, handle)
            with open(jwks_path, "w") as handle:
                json.dump(jwks, handle)
            cmd = [sys.executable, script, pack_path, "--jwks-file", jwks_path]
            if require_trusted_issuer:
                cmd.append("--require-trusted-issuer")
            verified = subprocess.run(cmd, capture_output=True, text=True)
        if verified.returncode != 0:
            raise SatGateError(verified.stdout + verified.stderr)
        return json.loads(verified.stdout)

    def _admin_json(self, method: str, path: str, body: Optional[dict] = None) -> dict:
        if not self.admin_token:
            raise AuthenticationError("admin token required")
        resp = self._send(
            method,
            self.gateway_url.rstrip("/") + path,
            json=body,
            headers={"X-Admin-Token": self.admin_token, "Content-Type": "application/json"},
            timeout=self.timeout,
        )
        if resp.status_code >= 400:
            raise SatGateError(f"{method} {path} failed: {resp.status_code} {resp.text}")
        return resp.json()

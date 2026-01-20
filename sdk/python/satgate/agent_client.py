"""
SatGate Agent Client - Economic Middleware for AI Agents

This module provides the SatGateAgentClient, which automatically handles:
- Identity badge-in (K8s, AWS, OIDC)
- 402 Payment Required challenges (Fiat402 and L402)
- Token caching and refresh
- Offline delegation for worker agents
- Budget tracking and alerts

Example usage:
    from satgate import SatGateAgentClient
    
    # Auto-detect environment (K8s/AWS)
    client = SatGateAgentClient(
        gateway_url="https://gateway.internal",
        identity="auto"
    )
    
    # Make requests - 402 handling is automatic
    response = client.get("/api/v1/data")
    print(f"Cost: ${response.cost:.2f}")
    
    # With Lightning wallet for external APIs
    from satgate.wallets import LNDWallet
    
    client = SatGateAgentClient(
        gateway_url="https://api.external.com",
        wallet=LNDWallet(...)
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


class IdentityProvider:
    """Base class for identity providers"""
    
    def get_credentials(self) -> str:
        """Return credentials for the Mint API"""
        raise NotImplementedError
    
    def provider_type(self) -> str:
        """Return the provider type identifier"""
        raise NotImplementedError


class KubernetesIdentity(IdentityProvider):
    """Kubernetes ServiceAccount identity provider"""
    
    TOKEN_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/token"
    
    def __init__(self, token_path: Optional[str] = None):
        self.token_path = token_path or self.TOKEN_PATH
    
    def get_credentials(self) -> str:
        try:
            with open(self.token_path, 'r') as f:
                return f.read().strip()
        except FileNotFoundError:
            raise SatGateError(f"K8s token not found at {self.token_path}")
    
    def provider_type(self) -> str:
        return "kubernetes"
    
    @classmethod
    def is_available(cls) -> bool:
        return os.path.exists(cls.TOKEN_PATH)


class AWSIdentity(IdentityProvider):
    """AWS IAM identity provider using STS"""
    
    def __init__(self, role_arn: Optional[str] = None):
        self.role_arn = role_arn
    
    def get_credentials(self) -> str:
        # Check for EKS IRSA token
        irsa_token_path = os.environ.get('AWS_WEB_IDENTITY_TOKEN_FILE')
        if irsa_token_path and os.path.exists(irsa_token_path):
            with open(irsa_token_path, 'r') as f:
                return json.dumps({"irsaToken": f.read().strip()})
        
        # Use boto3 if available
        try:
            import boto3
            sts = boto3.client('sts')
            identity = sts.get_caller_identity()
            return json.dumps({
                "stsResponse": {
                    "Account": identity['Account'],
                    "Arn": identity['Arn'],
                    "UserId": identity['UserId']
                }
            })
        except ImportError:
            raise SatGateError("boto3 required for AWS identity")
        except Exception as e:
            raise SatGateError(f"AWS identity failed: {e}")
    
    def provider_type(self) -> str:
        return "aws"
    
    @classmethod
    def is_available(cls) -> bool:
        # Check for IRSA
        if os.environ.get('AWS_WEB_IDENTITY_TOKEN_FILE'):
            return True
        # Check for AWS credentials
        return any([
            os.environ.get('AWS_ACCESS_KEY_ID'),
            os.environ.get('AWS_ROLE_ARN'),
            os.path.exists(os.path.expanduser('~/.aws/credentials'))
        ])


class OIDCIdentity(IdentityProvider):
    """OIDC identity provider"""
    
    def __init__(self, token: Optional[str] = None, token_env: str = 'OIDC_TOKEN'):
        self.token = token
        self.token_env = token_env
    
    def get_credentials(self) -> str:
        token = self.token or os.environ.get(self.token_env)
        if not token:
            raise SatGateError(f"OIDC token not found in {self.token_env}")
        return token
    
    def provider_type(self) -> str:
        return "oidc"


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
    
    Automatically handles:
    - Identity badge-in (K8s, AWS, OIDC)
    - 402 Payment Required challenges
    - Token caching and refresh
    - Budget tracking
    """
    
    def __init__(
        self,
        gateway_url: str,
        identity: Union[str, IdentityProvider] = "auto",
        wallet: Optional[LightningWallet] = None,
        budget_limit: Optional[float] = None,
        timeout: int = 30,
        max_retries: int = 3,
        on_budget_alert: Optional[Callable[[float, float], None]] = None,
        mint_url: Optional[str] = None,
    ):
        """
        Initialize the agent client.
        
        Args:
            gateway_url: Base URL of the SatGate gateway (for API requests)
            identity: Identity provider ("auto", "k8s", "aws", "oidc") or IdentityProvider instance
            wallet: Lightning wallet for L402 payments (optional)
            budget_limit: Maximum budget to spend (optional)
            timeout: Request timeout in seconds
            max_retries: Maximum retry attempts for 402 challenges
            on_budget_alert: Callback when budget is running low
            mint_url: Separate URL for Mint service (e.g., "http://gateway-admin:9090")
                      In enterprise deployments, Mint may be on a separate internal listener.
                      If not specified, uses gateway_url + "/v1/mint".
                      Can also be set via SATGATE_MINT_URL environment variable.
        """
        self.gateway_url = gateway_url.rstrip('/')
        
        # Mint URL: explicit > env > gateway_url
        self.mint_url = mint_url or os.environ.get('SATGATE_MINT_URL') or self.gateway_url
        self.mint_url = self.mint_url.rstrip('/')
        
        self.wallet = wallet
        self.budget_limit = budget_limit
        self.timeout = timeout
        self.max_retries = max_retries
        self.on_budget_alert = on_budget_alert
        
        # Session for connection pooling
        self.session = requests.Session()
        
        # Token cache
        self._token_cache: Optional[CachedToken] = None
        
        # Total cost tracking
        self._total_cost: float = 0.0
        
        # Initialize identity provider
        if isinstance(identity, IdentityProvider):
            self._identity = identity
        elif identity == "auto":
            self._identity = self._auto_detect_identity()
        elif identity == "k8s" or identity == "kubernetes":
            self._identity = KubernetesIdentity()
        elif identity == "aws":
            self._identity = AWSIdentity()
        elif identity == "oidc":
            self._identity = OIDCIdentity()
        elif identity == "none":
            self._identity = None
        else:
            raise SatGateError(f"Unknown identity provider: {identity}")
    
    def _auto_detect_identity(self) -> Optional[IdentityProvider]:
        """Auto-detect the best available identity provider"""
        if KubernetesIdentity.is_available():
            return KubernetesIdentity()
        if AWSIdentity.is_available():
            return AWSIdentity()
        return None
    
    def _get_token(self) -> Optional[str]:
        """Get a valid token, minting if necessary"""
        # Check cache
        if self._token_cache and not self._token_cache.is_expired:
            return self._token_cache.token
        
        # No identity provider - can't mint
        if not self._identity:
            return None
        
        # Mint new token
        try:
            credentials = self._identity.get_credentials()
            
            # Use separate mint_url (for enterprise deployments with separate listeners)
            mint_endpoint = urljoin(self.mint_url, "/v1/mint")
            resp = self.session.post(
                mint_endpoint,
                json={
                    "provider": self._identity.provider_type(),
                    "credentials": credentials
                },
                timeout=self.timeout
            )
            
            if resp.status_code != 200:
                raise AuthenticationError(f"Mint failed: {resp.text}")
            
            result = resp.json()
            
            # Cache token
            self._token_cache = CachedToken(
                token=result['macaroon'],
                signature=result['signature'],
                expires_at=time.time() + 3600,  # Default 1 hour
                budget_limit=result.get('budget', {}).get('limit'),
                caveats=result.get('caveats', [])
            )
            
            return self._token_cache.token
            
        except Exception as e:
            raise AuthenticationError(f"Failed to get token: {e}")
    
    def _handle_402(self, response: requests.Response, method: str, url: str, **kwargs) -> AgentResponse:
        """Handle a 402 Payment Required response"""
        
        # Parse the challenge
        www_auth = response.headers.get('WWW-Authenticate', '')
        content_type = response.headers.get('Content-Type', '')
        
        challenge = {}
        if 'application/json' in content_type:
            try:
                challenge = response.json()
            except:
                pass
        
        challenge_type = challenge.get('type', '')
        
        # Fiat402 challenge (internal budget)
        if 'SatGate-Billing' in www_auth or challenge_type == 'fiat402':
            return self._handle_fiat402(challenge, method, url, **kwargs)
        
        # L402 challenge (Lightning payment)
        if 'L402' in www_auth or 'LSAT' in www_auth or challenge_type == 'l402':
            return self._handle_l402(www_auth, challenge, method, url, **kwargs)
        
        # Unknown challenge type
        raise PaymentRequiredError(
            f"Unknown 402 challenge type",
            challenge=challenge
        )
    
    def _handle_fiat402(self, challenge: dict, method: str, url: str, **kwargs) -> AgentResponse:
        """Handle Fiat402 (internal budget) challenge"""
        
        invoice_id = challenge.get('invoice_id')
        amount = challenge.get('amount', 0)
        
        if not invoice_id:
            raise PaymentRequiredError("Missing invoice_id in Fiat402 challenge", challenge=challenge)
        
        # Check budget
        if self.budget_limit and self._total_cost + amount > self.budget_limit:
            raise BudgetExceededError(
                f"Budget exceeded: ${self._total_cost:.2f} + ${amount:.2f} > ${self.budget_limit:.2f}",
                used=self._total_cost,
                limit=self.budget_limit
            )
        
        # Get receipt from gateway
        receipt_url = challenge.get('receipt_url', f"/api/v1/billing/receipts?invoice_id={invoice_id}")
        full_receipt_url = urljoin(self.gateway_url, receipt_url)
        
        receipt_resp = self.session.post(
            full_receipt_url,
            json={"invoiceId": invoice_id},
            timeout=self.timeout
        )
        
        if receipt_resp.status_code != 200:
            raise PaymentRequiredError(f"Failed to get receipt: {receipt_resp.text}", challenge=challenge)
        
        receipt_data = receipt_resp.json()
        receipt_token = receipt_data.get('receipt') or receipt_data.get('token')
        
        if not receipt_token:
            raise PaymentRequiredError("No receipt token in response", challenge=challenge)
        
        # Retry with receipt
        headers = kwargs.get('headers', {})
        headers['Authorization'] = f"Receipt {receipt_token}"
        kwargs['headers'] = headers
        
        retry_resp = self.session.request(method, url, timeout=self.timeout, **kwargs)
        
        # Track cost
        self._total_cost += amount
        if self._token_cache:
            self._token_cache.budget_used += amount
        
        # Check budget alert
        if self.on_budget_alert and self.budget_limit:
            remaining = self.budget_limit - self._total_cost
            if remaining < self.budget_limit * 0.2:  # 20% remaining
                self.on_budget_alert(remaining, self.budget_limit)
        
        return AgentResponse(
            status_code=retry_resp.status_code,
            headers=dict(retry_resp.headers),
            content=retry_resp.content,
            cost=amount,
            budget_remaining=self.budget_limit - self._total_cost if self.budget_limit else None
        )
    
    def _handle_l402(self, www_auth: str, challenge: dict, method: str, url: str, **kwargs) -> AgentResponse:
        """Handle L402 (Lightning payment) challenge"""
        
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
        
        # Build L402 token
        l402_token = f"L402 {macaroon}:{preimage}"
        
        # Retry with L402 token
        headers = kwargs.get('headers', {})
        headers['Authorization'] = l402_token
        kwargs['headers'] = headers
        
        retry_resp = self.session.request(method, url, timeout=self.timeout, **kwargs)
        
        # Extract cost from invoice (simplified - would need BOLT11 decoder)
        cost_sats = challenge.get('amount', 0)
        
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
        Make an HTTP request with automatic 402 handling.
        
        Args:
            method: HTTP method (GET, POST, etc.)
            path: URL path (will be joined with gateway_url)
            **kwargs: Additional arguments passed to requests
            
        Returns:
            AgentResponse with status, content, and cost information
        """
        url = urljoin(self.gateway_url, path)
        
        # Add token if available
        token = self._get_token()
        if token:
            headers = kwargs.get('headers', {})
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
            
            # Payment required
            if resp.status_code == 402:
                return self._handle_402(resp, method, url, **kwargs)
            
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
    
    def delegate(
        self,
        additional_caveats: Optional[List[str]] = None,
        ttl: Optional[str] = None,
        budget_limit: Optional[float] = None
    ) -> str:
        """
        Create a delegated (child) token for a worker agent.
        
        Args:
            additional_caveats: Additional restrictions to add
            ttl: Reduced lifetime (e.g., "5m", "1h")
            budget_limit: Maximum budget for the child token
            
        Returns:
            Base64-encoded child macaroon
        """
        if not self._token_cache:
            self._get_token()
        
        if not self._token_cache:
            raise SatGateError("No token available for delegation")
        
        caveats = additional_caveats or []
        
        if ttl:
            caveats.append(f"ttl = {ttl}")
        if budget_limit:
            caveats.append(f"budget_limit = {budget_limit}")
        
        # Use mint_url for delegation (may be on separate listener)
        delegate_endpoint = urljoin(self.mint_url, "/v1/mint/delegate")
        resp = self.session.post(
            delegate_endpoint,
            json={
                "parentMacaroon": self._token_cache.token,
                "additionalCaveats": caveats
            },
            timeout=self.timeout
        )
        
        if resp.status_code != 200:
            raise SatGateError(f"Delegation failed: {resp.text}")
        
        return resp.json()['macaroon']
    
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

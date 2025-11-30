import requests
import re
from abc import ABC, abstractmethod
from typing import Optional, Dict, Any

# --- 1. Wallet Interface (Plug & Play) ---

class LightningWallet(ABC):
    """Abstract Base Class for any Lightning Wallet"""
    
    @abstractmethod
    def pay_invoice(self, invoice: str) -> str:
        """Pays the invoice and returns the preimage (hex string)"""
        pass

# --- 2. The Intelligent Session ---

class SatGateSession(requests.Session):
    def __init__(self, wallet: LightningWallet):
        super().__init__()
        self.wallet = wallet

    def request(self, method: str, url: str, *args, **kwargs) -> requests.Response:
        # 1. Attempt the request normally
        try:
            response = super().request(method, url, *args, **kwargs)
        except requests.exceptions.RequestException as e:
            # If we can't even connect, re-raise
            raise e

        # 2. Intercept 402 Errors
        if response.status_code == 402:
            return self._handle_payment_flow(response, method, url, *args, **kwargs)
        
        return response

    def _handle_payment_flow(self, response: requests.Response, method: str, url: str, *args, **kwargs) -> requests.Response:
        """The magic loop: Parse -> Pay -> Retry"""
        
        # A. Parse the L402 Header
        auth_header = response.headers.get("WWW-Authenticate")
        if not auth_header:
            # Some servers might not send WWW-Authenticate or send it differently
            # Try to see if it's in the body or standard L402
            print("⚠️ 402 received but no WWW-Authenticate header found.")
            return response

        # Check for L402 or LSAT
        if "L402" not in auth_header and "LSAT" not in auth_header:
             print("⚠️ 402 received but header does not contain L402/LSAT scheme.")
             return response

        # Extract Invoice and Macaroon (Regex to handle standard L402 format)
        # Example: L402 macaroon="...", invoice="lnbc..."
        # Using non-greedy match for values in quotes
        macaroon_match = re.search(r'macaroon="([^"]+)"', auth_header)
        invoice_match = re.search(r'invoice="([^"]+)"', auth_header)

        if not macaroon_match or not invoice_match:
            print("❌ Invalid L402 header format: could not find macaroon or invoice.")
            return response

        macaroon = macaroon_match.group(1)
        invoice = invoice_match.group(1)

        print(f"⚡ 402 Detected. Price: Unknown (check invoice).")
        print(f"   Invoice: {invoice[:20]}...{invoice[-10:]}")

        # B. Pay the Invoice (User's Wallet Implementation)
        try:
            preimage = self.wallet.pay_invoice(invoice)
            if not preimage:
                 raise ValueError("Wallet returned empty preimage")
            print(f"✅ Payment Confirmed. Preimage: {preimage[:10]}...")
        except Exception as e:
            print(f"❌ Payment Failed: {e}")
            return response

        # C. Retry with Authorization
        # Format: Authorization: L402 <macaroon>:<preimage>
        # Note: Some implementations might use LSAT, but L402 is the standard.
        # We'll use L402 as the prefix.
        l402_token = f"L402 {macaroon}:{preimage}"
        
        # Merge headers if they exist, or create new dict
        # We need to be careful not to modify the original kwargs['headers'] in place 
        # if it's reused elsewhere, though usually fine here.
        req_headers = kwargs.get("headers", {})
        if req_headers is None:
            req_headers = {}
        
        # Copy to avoid side effects
        req_headers = req_headers.copy()
        req_headers["Authorization"] = l402_token
        kwargs["headers"] = req_headers

        print("🔄 Retrying request with L402 Token...")
        return super().request(method, url, *args, **kwargs)


import threading
import time
import requests
import unittest
from http.server import HTTPServer, BaseHTTPRequestHandler
from satgate.client import SatGateSession, LightningWallet

# --- Mock Server ---

class MockL402Handler(BaseHTTPRequestHandler):
    protocol_version = 'HTTP/1.1'

    def do_GET(self):
        # Check for Authorization header
        auth_header = self.headers.get("Authorization")
        
        if auth_header and auth_header.startswith("L402"):
            # Validate token (Mock validation)
            # Expected: L402 macaroon:preimage
            token = auth_header.split(" ")[1]
            if ":" in token:
                macaroon, preimage = token.split(":")
                if macaroon == "test_macaroon" and preimage == "test_preimage":
                    self.send_response(200)
                    self.send_header("Content-type", "application/json")
                    self.end_headers()
                    self.wfile.write(b'{"status": "success", "data": "premium_content"}')
                    return
        
        # Default: Return 402
        self.send_response(402)
        self.send_header("WWW-Authenticate", 'L402 macaroon="test_macaroon", invoice="lnbc_test_invoice"')
        self.send_header("Content-type", "text/plain")
        self.end_headers()
        self.wfile.write(b'Payment Required')

    def log_message(self, format, *args):
        # Silence logs
        pass

class MockServer:
    def __init__(self):
        self.server = HTTPServer(('localhost', 0), MockL402Handler)
        self.port = self.server.server_port
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.daemon = True

    def start(self):
        self.thread.start()
        # Give it a moment to start
        time.sleep(0.1)

    def stop(self):
        self.server.shutdown()
        self.server.server_close()

# --- Mock Wallet ---

class MockWallet(LightningWallet):
    def pay_invoice(self, invoice: str) -> str:
        if invoice == "lnbc_test_invoice":
            return "test_preimage"
        raise ValueError("Unknown invoice")

# --- Integration Test ---

class TestSatGateSDK(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = MockServer()
        cls.server.start()
        cls.base_url = f"http://localhost:{cls.server.port}"

    @classmethod
    def tearDownClass(cls):
        cls.server.stop()

    def test_l402_flow(self):
        # 1. Setup Wallet and Session
        wallet = MockWallet()
        session = SatGateSession(wallet=wallet)

        # 2. Make Request
        print(f"\nTesting request to {self.base_url}...")
        response = session.get(self.base_url)

        # 3. Verify Result
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), {"status": "success", "data": "premium_content"})
        print("✅ L402 Flow Verified: 402 -> Pay -> 200")

    def test_payment_failure(self):
        # Wallet that fails
        class BrokeWallet(LightningWallet):
            def pay_invoice(self, invoice: str) -> str:
                raise Exception("Insufficient funds")

        session = SatGateSession(wallet=BrokeWallet())
        
        # Should return the original 402 response
        response = session.get(self.base_url)
        self.assertEqual(response.status_code, 402)
        print("✅ Failure Handling Verified")

if __name__ == "__main__":
    unittest.main()


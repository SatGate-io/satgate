"""TLS verification of the LND REST wallet."""

import json
import logging
import shutil
import ssl
import subprocess
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

import pytest
import requests

from satgate.agent_client import LNDWallet
from satgate.exceptions import SatGateError

MACAROON_HEX = "0201036c6e64"


def _make_cert(tmp_path, name):
    if shutil.which("openssl") is None:
        pytest.skip("openssl is not installed")
    cert = tmp_path / f"{name}.cert"
    key = tmp_path / f"{name}.key"
    subprocess.run(
        [
            "openssl", "req", "-x509", "-newkey", "ec",
            "-pkeyopt", "ec_paramgen_curve:prime256v1", "-nodes", "-days", "1",
            "-subj", "/CN=127.0.0.1",
            "-addext", "subjectAltName=IP:127.0.0.1,DNS:localhost",
            "-addext", "basicConstraints=critical,CA:TRUE",
            "-addext", "keyUsage=critical,digitalSignature,keyCertSign",
            "-keyout", str(key), "-out", str(cert),
        ],
        check=True,
        capture_output=True,
    )
    return cert, key


class _LNDServer:
    """A local HTTPS server with a self-signed certificate, like LND's REST port."""

    def __init__(self, cert, key):
        self.seen_macaroons = []
        server = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def _reply(self, body):
                server.seen_macaroons.append(self.headers.get("Grpc-Metadata-macaroon"))
                data = json.dumps(body).encode()
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

            def do_GET(self):
                self._reply({"balance": "42"})

            def do_POST(self):
                length = int(self.headers.get("Content-Length", "0"))
                self.rfile.read(length)
                self._reply({"payment_preimage": "ab" * 32})

        self.httpd = HTTPServer(("127.0.0.1", 0), Handler)
        context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        context.load_cert_chain(str(cert), str(key))
        self.httpd.socket = context.wrap_socket(self.httpd.socket, server_side=True)
        self.host = f"127.0.0.1:{self.httpd.server_address[1]}"
        self.thread = threading.Thread(target=self.httpd.serve_forever, daemon=True)
        self.thread.start()

    def close(self):
        self.httpd.shutdown()
        self.httpd.server_close()


@pytest.fixture
def lnd(tmp_path):
    cert, key = _make_cert(tmp_path, "lnd")
    server = _LNDServer(cert, key)
    server.cert = cert
    yield server
    server.close()


@pytest.fixture
def record_verify(monkeypatch):
    """Record the verify= value of every request without sending it."""
    seen = []

    class FakeResponse:
        status_code = 200
        text = "{}"

        def json(self):
            return {"balance": "1", "payment_preimage": ""}

    def fake_request(self, method, url, **kwargs):
        seen.append(kwargs.get("verify"))
        return FakeResponse()

    monkeypatch.setattr(requests.sessions.Session, "request", fake_request)
    return seen


def test_lnd_wallet_verifies_tls_by_default(record_verify):
    wallet = LNDWallet(host="lnd.example.test:8080", macaroon_hex=MACAROON_HEX)
    wallet.get_balance()
    wallet.pay_invoice("lnbc1")
    assert record_verify == [True, True]


def test_lnd_wallet_untrusted_https_host_fails_loudly(lnd):
    wallet = LNDWallet(host=lnd.host, macaroon_hex=MACAROON_HEX)
    with pytest.raises(SatGateError) as exc:
        wallet.get_balance()
    assert lnd.host in str(exc.value)
    assert "tls_cert_path" in str(exc.value)
    with pytest.raises(SatGateError):
        wallet.pay_invoice("lnbc1")
    assert lnd.seen_macaroons == []


def test_lnd_wallet_tls_cert_path_verifies_against_that_file(lnd, tmp_path):
    wallet = LNDWallet(host=lnd.host, macaroon_hex=MACAROON_HEX, tls_cert_path=str(lnd.cert))
    assert wallet.get_balance() == 42
    assert wallet.pay_invoice("lnbc1") == "ab" * 32
    assert lnd.seen_macaroons == [MACAROON_HEX, MACAROON_HEX]


def test_lnd_wallet_cert_path_alias_still_accepted(lnd):
    wallet = LNDWallet(host=lnd.host, macaroon_hex=MACAROON_HEX, cert_path=str(lnd.cert))
    assert wallet.get_balance() == 42
    same = LNDWallet(
        host=lnd.host, macaroon_hex=MACAROON_HEX,
        cert_path=str(lnd.cert), tls_cert_path=str(lnd.cert),
    )
    assert same.get_balance() == 42


def test_lnd_wallet_other_certificate_file_fails_loudly(lnd, tmp_path):
    other, _ = _make_cert(tmp_path, "other")
    wallet = LNDWallet(host=lnd.host, macaroon_hex=MACAROON_HEX, tls_cert_path=str(other))
    with pytest.raises(SatGateError) as exc:
        wallet.get_balance()
    assert lnd.host in str(exc.value)
    assert lnd.seen_macaroons == []


def test_lnd_wallet_insecure_skips_verification_and_warns(lnd, caplog):
    with caplog.at_level(logging.WARNING, logger="satgate.agent_client"):
        wallet = LNDWallet(host=lnd.host, macaroon_hex=MACAROON_HEX, insecure=True)
    warnings = [r.getMessage() for r in caplog.records if r.levelno == logging.WARNING]
    assert any(lnd.host in message for message in warnings), warnings
    assert wallet.get_balance() == 42


def test_lnd_wallet_insecure_is_the_only_way_to_skip(record_verify, tmp_path):
    # An empty path is refused rather than read as "no verification".
    with pytest.raises(SatGateError):
        LNDWallet(macaroon_hex=MACAROON_HEX, cert_path="")
    with pytest.raises(SatGateError):
        LNDWallet(macaroon_hex=MACAROON_HEX, tls_cert_path=str(tmp_path / "missing.cert"))
    for value in ("true", 1, None):
        with pytest.raises(SatGateError):
            LNDWallet(macaroon_hex=MACAROON_HEX, insecure=value)
    cert = tmp_path / "a.cert"
    cert.write_text("x")
    with pytest.raises(SatGateError):
        LNDWallet(macaroon_hex=MACAROON_HEX, tls_cert_path=str(cert), insecure=True)
    other = tmp_path / "b.cert"
    other.write_text("x")
    with pytest.raises(SatGateError):
        LNDWallet(macaroon_hex=MACAROON_HEX, tls_cert_path=str(cert), cert_path=str(other))
    LNDWallet(macaroon_hex=MACAROON_HEX, insecure=False).get_balance()
    assert record_verify == [True]

"""LNDWallet never follows a redirect with the macaroon header."""

import logging
import ssl
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

import pytest
from test_lnd_wallet_tls import MACAROON_HEX, _LNDServer, _make_cert

from satgate.agent_client import LNDWallet
from satgate.exceptions import SatGateError

REDIRECT_STATUSES = [301, 302, 303, 307, 308]
OPERATIONS = ["get_balance", "pay_invoice"]


class _Recorder:
    """A local server that records each request's path and macaroon header."""

    def __init__(
        self, bind="127.0.0.1", cert=None, key=None, status=200, location=None
    ):
        self.requests = []
        recorder = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def _reply(self):
                length = int(self.headers.get("Content-Length", "0"))
                if length:
                    self.rfile.read(length)
                recorder.requests.append(
                    (self.path, self.headers.get("Grpc-Metadata-macaroon"))
                )
                if recorder.location is not None:
                    self.send_response(recorder.status)
                    self.send_header("Location", recorder.location)
                    self.send_header("Content-Length", "0")
                    self.end_headers()
                    return
                body = b'{"balance": "42", "payment_preimage": "ab"}'
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            do_GET = _reply
            do_POST = _reply

        self.status = status
        self.location = location
        self.httpd = HTTPServer((bind, 0), Handler)
        self.port = self.httpd.server_address[1]
        self.scheme = "http"
        if cert is not None:
            context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            context.load_cert_chain(str(cert), str(key))
            self.httpd.socket = context.wrap_socket(self.httpd.socket, server_side=True)
            self.scheme = "https"
        self.thread = threading.Thread(target=self.httpd.serve_forever, daemon=True)
        self.thread.start()

    def url(self, host="127.0.0.1"):
        return f"{self.scheme}://{host}:{self.port}/sink"

    def close(self):
        self.httpd.shutdown()
        self.httpd.server_close()


@pytest.fixture
def servers(tmp_path):
    cert, key = _make_cert(tmp_path, "lnd")
    made = []

    def make(**kwargs):
        server = _Recorder(**kwargs)
        made.append(server)
        return server

    make.cert, make.key = cert, key
    make.redirect = lambda status, location: make(
        cert=cert, key=key, status=status, location=location
    )
    yield make
    for server in made:
        server.close()


def _call(wallet, operation):
    if operation == "pay_invoice":
        return wallet.pay_invoice("lnbc1")
    return wallet.get_balance()


def _assert_refused(wallet, operation, origin, sink, status):
    with pytest.raises(SatGateError) as exc:
        _call(wallet, operation)
    message = str(exc.value)
    assert f"HTTP {status}" in message
    assert wallet.host in message
    # The macaroon went to the configured host once and nowhere else.
    assert [m for _, m in origin.requests] == [MACAROON_HEX]
    assert sink.requests == []
    return message


@pytest.mark.parametrize("status", REDIRECT_STATUSES)
@pytest.mark.parametrize("operation", OPERATIONS)
def test_lnd_wallet_never_follows_redirect_to_http(servers, operation, status):
    sink = servers()
    origin = servers.redirect(status, sink.url())
    wallet = LNDWallet(
        host=f"127.0.0.1:{origin.port}", macaroon_hex=MACAROON_HEX,
        tls_cert_path=str(servers.cert),
    )
    message = _assert_refused(wallet, operation, origin, sink, status)
    assert f"http://127.0.0.1:{sink.port}" in message


@pytest.mark.parametrize("status", REDIRECT_STATUSES)
@pytest.mark.parametrize("operation", OPERATIONS)
def test_lnd_wallet_never_follows_redirect_to_another_host(servers, operation, status):
    # The other host verifies with the same certificate, so only the redirect
    # rule keeps the macaroon from reaching it.
    sink = servers(cert=servers.cert, key=servers.key)
    origin = servers(
        cert=servers.cert, key=servers.key, status=status,
        location=sink.url(host="localhost"),
    )
    wallet = LNDWallet(
        host=f"127.0.0.1:{origin.port}", macaroon_hex=MACAROON_HEX,
        tls_cert_path=str(servers.cert),
    )
    message = _assert_refused(wallet, operation, origin, sink, status)
    assert f"https://localhost:{sink.port}" in message


@pytest.mark.parametrize("operation", OPERATIONS)
def test_lnd_wallet_never_follows_redirect_on_the_same_host(servers, operation):
    origin = servers.redirect(307, "/elsewhere")
    wallet = LNDWallet(
        host=f"127.0.0.1:{origin.port}", macaroon_hex=MACAROON_HEX,
        tls_cert_path=str(servers.cert),
    )
    with pytest.raises(SatGateError) as exc:
        _call(wallet, operation)
    assert "HTTP 307" in str(exc.value)
    assert [m for _, m in origin.requests] == [MACAROON_HEX]


@pytest.mark.parametrize("operation", OPERATIONS)
def test_lnd_wallet_insecure_still_never_follows_redirects(servers, operation):
    sink = servers()
    origin = servers.redirect(302, sink.url())
    wallet = LNDWallet(
        host=f"127.0.0.1:{origin.port}", macaroon_hex=MACAROON_HEX, insecure=True
    )
    _assert_refused(wallet, operation, origin, sink, 302)


def test_lnd_wallet_redirect_error_names_only_scheme_host_and_port(servers):
    origin = servers(
        cert=servers.cert, key=servers.key, status=302,
        location="http://user:pw@127.0.0.1:9/path?q=hidden",
    )
    wallet = LNDWallet(
        host=f"127.0.0.1:{origin.port}", macaroon_hex=MACAROON_HEX,
        tls_cert_path=str(servers.cert),
    )
    with pytest.raises(SatGateError) as exc:
        wallet.get_balance()
    message = str(exc.value)
    assert "http://127.0.0.1:9" in message
    for part in ("user", "pw", "/path", "hidden"):
        assert part not in message


# Reviewer case: a 302 from the verified HTTPS host to a plain-HTTP URL must
# not deliver the macaroon header to the HTTP recipient, for both operations.
@pytest.mark.parametrize("operation", OPERATIONS)
def test_lnd_wallet_macaroon_never_reaches_http_after_redirect(servers, operation):
    sink = servers()
    origin = servers.redirect(302, sink.url())
    wallet = LNDWallet(
        host=f"127.0.0.1:{origin.port}", macaroon_hex=MACAROON_HEX,
        tls_cert_path=str(servers.cert),
    )
    with pytest.raises(SatGateError):
        _call(wallet, operation)
    assert not any(m for _, m in sink.requests)


# Reviewer case: with each way of trusting the certificate (the system store,
# here REQUESTS_CA_BUNDLE; tls_cert_path; cert_path) and insecure=False, a 307
# to plain HTTP sends no request carrying the header, for both operations.
@pytest.mark.parametrize("trust", ["default", "tls_cert_path", "cert_path"])
def test_lnd_wallet_redirect_refused_for_every_certificate_option(
    servers, monkeypatch, trust
):
    sink = servers()
    origin = servers.redirect(307, sink.url())
    kwargs = {}
    if trust == "default":
        monkeypatch.setenv("REQUESTS_CA_BUNDLE", str(servers.cert))
    else:
        monkeypatch.delenv("REQUESTS_CA_BUNDLE", raising=False)
        kwargs[trust] = str(servers.cert)
    wallet = LNDWallet(
        host=f"127.0.0.1:{origin.port}", macaroon_hex=MACAROON_HEX,
        insecure=False, **kwargs,
    )
    for operation in OPERATIONS:
        with pytest.raises(SatGateError):
            _call(wallet, operation)
    assert sum(1 for _, m in sink.requests if m) == 0
    assert len(origin.requests) == 2


# Reviewer case: insecure=True logs one warning naming the host, however many
# requests the wallet then makes.
def test_lnd_wallet_insecure_warns_once_for_both_operations(tmp_path, caplog):
    cert, key = _make_cert(tmp_path, "warn")
    server = _LNDServer(cert, key)
    try:
        with caplog.at_level(logging.WARNING, logger="satgate.agent_client"):
            wallet = LNDWallet(
                host=server.host, macaroon_hex=MACAROON_HEX, insecure=True
            )
            wallet.get_balance()
            wallet.pay_invoice("lnbc1")
        entries = [r for r in caplog.records if r.name == "satgate.agent_client"]
        assert len(entries) == 1
        assert server.host in entries[0].getMessage()
    finally:
        server.close()

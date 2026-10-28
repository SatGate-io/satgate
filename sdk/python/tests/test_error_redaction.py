"""SDK error messages carry no credential and no request path or query.

Every message the SDK builds or raises about a request (a transport error,
an HTTP status error quoting the response, an invalid header, a JSON parse
error, a wrapped error) is passed through ``satgate._http.redact``: the
request URL is named by its host and port, its path and query are removed,
and every credential the request carried (``Authorization``,
``X-Admin-Token``, the LND macaroon header, tokens in the body or the query)
is replaced, also where the response body quotes it back. Exception types
and the objects callers use are unchanged.

A local server (127.0.0.1 only) echoes the request's path, query,
credential headers and body into its answer. Each check reads the messages
of the error and of every exception it wraps, and the formatted traceback.
"""

import json
import socket
import ssl
import threading
import traceback
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import quote

import pytest
import requests
from test_lnd_wallet_tls import _make_cert

from satgate import SatGate, SatGateAgentClient, SatGateClient, SatGateError, _http
from satgate.agent_client import AlbyWallet, LNDWallet
from satgate.exceptions import AuthenticationError, PaymentFailedError

# Synthetic values. Every credential and URL part contains "private" or
# "sekret", so one search finds any of them in a message.
ADMIN = "adm-sekret-0123456789"
CAP = "cap-sekret-0123456789"
API_KEY = "key-sekret-0123456789"
ALBY = "alby-sekret-0123456789"
MACAROON = "0201036c6e64sekret0123"
QUERY_VALUE = "qv-sekret-0123456789"
PARENT = "parent-sekret-0123456789"
BASE_PATH = "/private-base"
QUERY = f"private-query={QUERY_VALUE}"
MARKERS = ("sekret", "private", "/api/", "/v1/", "/.well-known/")


class _EchoServer:
    """Answers every request by echoing its path (with query), credential
    headers and body. ``shape`` picks the answer:

    * ``status-text``: 500, text/plain;
    * ``status-json``: 500, JSON with the echo in ``error`` and ``message``;
    * ``bad-request``: 400, JSON with the echo in ``error``;
    * ``not-json``: 200, the echo as text/plain (not JSON);
    * ``payment-error``: 200, JSON with the echo in ``payment_error``;
    * ``cut-short``: 200, a Content-Length longer than the echo sent.
    """

    def __init__(self, shape, cert=None, key=None):
        self.shape = shape
        self.hits = 0
        server = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def _reply(self):
                server.hits += 1
                length = int(self.headers.get("Content-Length", "0"))
                body = self.rfile.read(length).decode("utf-8", "replace") if length else ""
                echo = (
                    f"cannot serve {self.path}; authorization={self.headers.get('Authorization')}"
                    f" admin={self.headers.get('X-Admin-Token')}"
                    f" macaroon={self.headers.get('Grpc-Metadata-macaroon')} body={body}"
                )
                status, ctype, data = 200, "text/plain", echo.encode()
                if server.shape == "status-text":
                    status = 500
                elif server.shape == "status-json":
                    status, ctype = 500, "application/json"
                    data = json.dumps({"error": echo, "message": echo}).encode()
                elif server.shape == "bad-request":
                    status, ctype = 400, "application/json"
                    data = json.dumps({"error": echo}).encode()
                elif server.shape == "payment-error":
                    ctype = "application/json"
                    data = json.dumps({"payment_error": echo}).encode()
                self.send_response(status)
                self.send_header("Content-Type", ctype)
                if server.shape == "cut-short":
                    self.send_header("Content-Length", str(len(data) + 1000))
                    self.end_headers()
                    self.wfile.write(data)
                    self.wfile.flush()
                    self.close_connection = True
                    return
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

            do_GET = do_POST = do_PUT = do_DELETE = _reply

        self.httpd = HTTPServer(("127.0.0.1", 0), Handler)
        self.port = self.httpd.server_address[1]
        if cert is not None:
            context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            context.load_cert_chain(str(cert), str(key))
            self.httpd.socket = context.wrap_socket(self.httpd.socket, server_side=True)
        self.thread = threading.Thread(target=self.httpd.serve_forever, daemon=True)
        self.thread.start()

    def close(self):
        self.httpd.shutdown()
        self.httpd.server_close()
        self.thread.join(timeout=5)


def _closed_port() -> int:
    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    port = s.getsockname()[1]
    s.close()
    return port


def _chain(err):
    seen, out, pending = set(), [], [err]
    while pending:
        e = pending.pop()
        if not isinstance(e, BaseException) or id(e) in seen:
            continue
        seen.add(id(e))
        out.append(e)
        pending.extend([e.__cause__, e.__context__, getattr(e, "reason", None), *e.args])
    return out


def _formatted(err):
    """What a traceback prints about err and the exceptions chained to it
    (without the source lines of the frames, which are code, not data)."""
    return "".join(
        line
        for line in traceback.format_exception(type(err), err, err.__traceback__)
        if not line.startswith("  ")
    )


def _assert_redacted(err):
    formatted = _formatted(err)
    for e in _chain(err):
        for marker in MARKERS:
            assert marker not in str(e), (marker, str(e))
    for marker in ("sekret", "private"):
        assert marker not in formatted, (marker, formatted)
    assert "@" not in str(err)


def _raised(fn):
    try:
        fn()
    except Exception as e:  # noqa: BLE001 - the test inspects whatever is raised
        return e
    raise AssertionError("the call did not raise")


def _alby(base):
    wallet = AlbyWallet(access_token=ALBY)
    wallet._api_base = base
    return wallet


def _l402_agent(base, wallet):
    """An agent whose first answer is a 402 challenge (from the echo server
    at base), paying with wallet."""
    return SatGateAgentClient(base, token=CAP, wallet=wallet)


# name -> (call taking the server's base URL "http://127.0.0.1:port",
#          {shape: the exception type expected for that answer}).
STATUS = {"status-text": SatGateError, "status-json": SatGateError}
CALLS = {
    "SatGateClient tokens.mint": (
        lambda b: SatGateClient(f"{b}{BASE_PATH}?{QUERY}", ADMIN).tokens.mint(),
        {**STATUS, "bad-request": SatGateError, "not-json": requests.JSONDecodeError,
         "cut-short": SatGateError}),
    "SatGateClient tokens.validate": (
        lambda b: SatGateClient(f"{b}{BASE_PATH}?{QUERY}", ADMIN).tokens.validate(CAP),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGateClient tokens.delegate": (
        lambda b: SatGateClient(f"{b}{BASE_PATH}?{QUERY}", ADMIN).tokens.delegate(PARENT),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGateClient governance.ban": (
        lambda b: SatGateClient(f"{b}{BASE_PATH}?{QUERY}", ADMIN).governance.ban("sig-sekret-0123"),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGateClient governance.get_graph": (
        lambda b: SatGateClient(f"{b}{BASE_PATH}?{QUERY}", ADMIN).governance.get_graph(),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGateClient ping": (
        lambda b: SatGateClient(f"{b}{BASE_PATH}?{QUERY}", ADMIN).ping(CAP),
        {"not-json": SatGateError, "cut-short": SatGateError}),
    "SatGate issue": (
        lambda b: SatGate(api_key=API_KEY, base_url=f"{b}{BASE_PATH}?{QUERY}").issue(task="t"),
        {**STATUS, "not-json": requests.JSONDecodeError, "cut-short": SatGateError}),
    "SatGate pay": (
        lambda b: SatGate(api_key=API_KEY, base_url=f"{b}{BASE_PATH}").pay(
            capability={"token": CAP}, upstream="u"),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGate verify": (
        lambda b: SatGate(api_key=API_KEY, base_url=f"{b}{BASE_PATH}").verify(
            {"signature": "sig-sekret-0123"}),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGateAgentClient get (capability token, params)": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", token=CAP).get(
            "/private-path", params={"private-query": QUERY_VALUE}),
        {**STATUS, "cut-short": requests.exceptions.ChunkedEncodingError}),
    "SatGateAgentClient get (admin token mint)": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", admin_token=ADMIN).get(f"/x?{QUERY}"),
        {"status-text": AuthenticationError, "status-json": AuthenticationError,
         "not-json": AuthenticationError, "cut-short": AuthenticationError}),
    "SatGateAgentClient post": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", token=CAP).post(
            f"/private-path?{QUERY}", json={"token": PARENT}),
        STATUS),
    "SatGateAgentClient validate_token": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", token=CAP).validate_token(),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGateAgentClient delegate": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", token=CAP).delegate(),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGateAgentClient fetch_jwks": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}").fetch_jwks(),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGateAgentClient stage_policy": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", admin_token=ADMIN).stage_policy({"a": 1}),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGateAgentClient simulate_policy": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", admin_token=ADMIN).simulate_policy("p"),
        STATUS),
    "SatGateAgentClient promote_policy": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", admin_token=ADMIN).promote_policy("p"),
        STATUS),
    "SatGateAgentClient fetch_pack": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", admin_token=ADMIN).fetch_pack(
            f"private-pack?{QUERY}"),
        {**STATUS, "not-json": requests.JSONDecodeError}),
    "SatGateAgentClient mint_capability": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", admin_token=ADMIN).mint_capability("s"),
        STATUS),
    "AlbyWallet get_balance": (
        lambda b: _alby(f"{b}{BASE_PATH}?{QUERY}").get_balance(),
        {**STATUS, "not-json": requests.JSONDecodeError, "cut-short": requests.exceptions.ChunkedEncodingError}),
    "AlbyWallet pay_invoice": (
        lambda b: _alby(f"{b}{BASE_PATH}?{QUERY}").pay_invoice("lnbc1"),
        {"status-text": PaymentFailedError, "status-json": PaymentFailedError,
         "not-json": requests.JSONDecodeError}),
}


@pytest.mark.parametrize(
    "name,shape",
    [(name, shape) for name in sorted(CALLS) for shape in sorted(CALLS[name][1])],
)
def test_error_quoting_the_response_names_no_credential_path_or_query(name, shape):
    call, expected = CALLS[name]
    server = _EchoServer(shape)
    try:
        err = _raised(lambda: call(f"http://127.0.0.1:{server.port}"))
    finally:
        server.close()
    assert server.hits >= 1
    assert type(err) is expected[shape] or isinstance(err, expected[shape]), repr(err)
    _assert_redacted(err)


@pytest.fixture
def lnd_cert(tmp_path):
    return _make_cert(tmp_path, "lnd")


LND_CALLS = {
    "LNDWallet get_balance": (
        lambda w: w.get_balance(),
        {"status-text": SatGateError, "not-json": requests.JSONDecodeError}),
    "LNDWallet pay_invoice": (
        lambda w: w.pay_invoice("lnbc1"),
        {"status-text": PaymentFailedError, "payment-error": PaymentFailedError,
         "not-json": requests.JSONDecodeError}),
}


@pytest.mark.parametrize(
    "name,shape",
    [(name, shape) for name in sorted(LND_CALLS) for shape in sorted(LND_CALLS[name][1])],
)
def test_lnd_wallet_error_names_no_macaroon_path_or_query(name, shape, lnd_cert):
    cert, key = lnd_cert
    call, expected = LND_CALLS[name]
    server = _EchoServer(shape, cert=cert, key=key)
    try:
        wallet = LNDWallet(
            host=f"127.0.0.1:{server.port}{BASE_PATH}?{QUERY}",
            macaroon_hex=MACAROON,
            tls_cert_path=str(cert),
        )
        err = _raised(lambda: call(wallet))
    finally:
        server.close()
    assert isinstance(err, expected[shape]), repr(err)
    _assert_redacted(err)


@pytest.mark.parametrize("shape", ["status-text", "status-json"])
def test_lightning_payment_failure_quoting_the_wallet_names_no_credential(shape):
    """The L402 flow quotes the wallet's error (Lightning payment failed)."""
    alby = _EchoServer(shape)
    gateway = _GatewayL402()
    try:
        wallet = _alby(f"http://127.0.0.1:{alby.port}{BASE_PATH}?{QUERY}")
        agent = _l402_agent(f"http://127.0.0.1:{gateway.port}{BASE_PATH}", wallet)
        err = _raised(lambda: agent.get(f"/private-path?{QUERY}"))
    finally:
        gateway.close()
        alby.close()
    assert type(err) is PaymentFailedError
    assert str(err).startswith("Lightning payment failed: Alby payment failed: ")
    _assert_redacted(err)


class _GatewayL402:
    """Answers 402 with an L402 challenge, then echoes like status-text."""

    def __init__(self):
        server = self
        self.hits = 0

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def do_GET(self):
                server.hits += 1
                data = json.dumps({"invoice": "lnbc1", "macaroon": "m"}).encode()
                self.send_response(402)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

        self.httpd = HTTPServer(("127.0.0.1", 0), Handler)
        self.port = self.httpd.server_address[1]
        self.thread = threading.Thread(target=self.httpd.serve_forever, daemon=True)
        self.thread.start()

    def close(self):
        self.httpd.shutdown()
        self.httpd.server_close()
        self.thread.join(timeout=5)


TRANSPORT_CALLS = {
    "SatGateAgentClient get (params)": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}", token=CAP).get(
            "/private-path", params={"private-query": QUERY_VALUE}),
        requests.ConnectionError),
    "SatGateAgentClient get (session params)": (
        lambda b: _with_session_params(SatGateAgentClient(f"{b}{BASE_PATH}", token=CAP)).get("/x"),
        requests.ConnectionError),
    "SatGateAgentClient call": (
        lambda b: SatGateAgentClient(f"{b}{BASE_PATH}").call(
            "POST", f"/private-path?{QUERY}", body={"token": PARENT}, token=CAP),
        requests.ConnectionError),
    "SatGateClient tokens.validate": (
        lambda b: SatGateClient(f"{b}{BASE_PATH}?{QUERY}", ADMIN).tokens.validate(CAP),
        SatGateError),
    "SatGate pay": (
        lambda b: SatGate(api_key=API_KEY, base_url=f"{b}{BASE_PATH}?{QUERY}").pay(
            capability={"token": CAP}),
        SatGateError),
}


def _with_session_params(agent):
    agent.session.params = {"private-query": QUERY_VALUE}
    return agent


@pytest.mark.parametrize("name", sorted(TRANSPORT_CALLS))
def test_transport_error_names_no_query_parameter_or_credential(name):
    call, expected = TRANSPORT_CALLS[name]
    err = _raised(lambda: call(f"http://127.0.0.1:{_closed_port()}"))
    assert isinstance(err, expected), repr(err)
    _assert_redacted(err)
    assert "Max retries exceeded with url: [path and query removed]" in str(err)


# Reviewer cases (round 4), as tests named for what they check.


def test_transport_error_names_neither_query_name_nor_value_passed_as_params():
    """A refused connection for get(path, params=...) names no query name or value."""
    err = _raised(
        lambda: SatGateAgentClient(f"http://127.0.0.1:{_closed_port()}", token="fixture").get(
            "/private-path", params={"private-query": QUERY_VALUE}
        )
    )
    assert type(err) is requests.ConnectionError
    assert "private-query" not in str(err)
    assert QUERY_VALUE not in str(err)
    assert err.request is not None


def test_status_error_quoting_an_echoed_path_names_neither_path_nor_query_value():
    """A 500 whose JSON error echoes the request path and query."""
    server = _EchoServer("status-json")
    try:
        client = SatGateClient(f"http://127.0.0.1:{server.port}", "fixture")
        err = _raised(lambda: client._request("GET", f"/private-route?credential={QUERY_VALUE}"))
    finally:
        server.close()
    assert type(err) is SatGateError
    assert str(err).startswith("API error 500: ")
    assert "private-route" not in str(err)
    assert QUERY_VALUE not in str(err)


@pytest.mark.parametrize("pack_id", ["private-path?private-query", f"review-pack?access={QUERY_VALUE}"])
def test_admin_helper_error_names_no_request_path_or_query(pack_id):
    server = _EchoServer("status-text")
    try:
        agent = SatGateAgentClient(f"http://127.0.0.1:{server.port}", admin_token=ADMIN)
        err = _raised(lambda: agent.fetch_pack(pack_id))
    finally:
        server.close()
    assert type(err) is SatGateError
    assert str(err).startswith("fetch_pack failed: 500 ")
    for part in ("private-path", "private-query", "review-pack", "access=", QUERY_VALUE, "/api/", ADMIN):
        assert part not in str(err)


@pytest.mark.parametrize(
    "case",
    ["agent-status", "admin-status", "beta-status"],
)
def test_status_errors_quoting_path_query_and_credential_name_none_of_them(case):
    """The server answers 500 with the request path and the credential it got."""
    server = _EchoServer("status-text")
    base = f"http://127.0.0.1:{server.port}/private-route?access={QUERY_VALUE}"
    calls = {
        "agent-status": lambda: SatGateAgentClient(base, token=CAP).get(
            f"/private-route?access={QUERY_VALUE}"),
        "admin-status": lambda: SatGateClient(base, ADMIN).tokens.mint(),
        "beta-status": lambda: SatGate(api_key=API_KEY, base_url=base).pay(),
    }
    try:
        err = _raised(calls[case])
    finally:
        server.close()
    assert type(err) is SatGateError
    text = _formatted(err)
    for secret in (CAP, ADMIN, API_KEY, QUERY_VALUE):
        assert secret not in text
    assert "private-route" not in str(err)
    assert "access=" not in str(err)


@pytest.mark.parametrize("case", ["agent-header", "admin-header", "beta-header", "l402-retry-header"])
def test_invalid_header_error_names_no_credential(case):
    """A credential with a line break is refused by requests; the error names it not."""
    server = _EchoServer("status-text")
    base = f"http://127.0.0.1:{server.port}"

    class _Wallet:
        def pay_invoice(self, invoice):
            return "pre-sekret-0123\n"

    gateway = _GatewayL402() if case == "l402-retry-header" else None
    calls = {
        "agent-header": (lambda: SatGateAgentClient(base, token=CAP + "\n").get("/x"),
                         requests.exceptions.InvalidHeader),
        "admin-header": (lambda: SatGateClient(base, ADMIN + "\n").tokens.mint(), SatGateError),
        "beta-header": (lambda: SatGate(api_key=API_KEY + "\r\n", base_url=base).pay(),
                        SatGateError),
        "l402-retry-header": (
            lambda: SatGateAgentClient(
                f"http://127.0.0.1:{gateway.port}", token=CAP, wallet=_Wallet()
            ).get(f"/private-path?{QUERY}"),
            requests.exceptions.InvalidHeader),
    }
    call, expected = calls[case]
    try:
        err = _raised(call)
    finally:
        server.close()
        if gateway is not None:
            gateway.close()
    assert type(err) is expected
    text = _formatted(err)
    for secret in (CAP, ADMIN, API_KEY, "pre-sekret"):
        assert secret not in text
    assert server.hits == 0
    if gateway is not None:
        assert gateway.hits == 1


# The redaction function itself.


def test_redact_replaces_url_path_query_and_credentials_and_keeps_other_text():
    redaction = _http.Redaction.for_request(
        "POST",
        f"https://u-sekret:p-sekret@gw.example:8443/private-base/x?{QUERY}",
        headers={"Authorization": f"Bearer {CAP}", "X-Admin-Token": ADMIN, "X-Api-Key": "k+sekret/0123="},
        params={"page-token": "pt-sekret-0123"},
        json={"parentToken": PARENT, "caveats": ["scope = api:read"], "nested": {"preimage": "pre-sekret"}},
    )
    text = (
        f"GET https://gw.example:8443/private-base/x?{QUERY}&page-token=pt-sekret-0123 failed;"
        f" /private-base/x and /x; {CAP} {ADMIN} {PARENT} pre-sekret u-sekret p-sekret;"
        f" quoted \"Bearer {CAP}\" encoded {quote('k+sekret/0123=', safe='')}; status 500;"
        " scope = api:read; the private-query name"
    )
    out = _http.redact(text, redaction)
    for marker in ("sekret", "private-base", "private-query", "page-token"):
        assert marker not in out, out
    assert out.startswith("GET gw.example:8443 failed;"), out
    assert "status 500" in out
    assert "scope = api:read" in out


def test_redact_exception_keeps_the_object_type_attributes_and_chain():
    request = requests.Request("GET", "http://127.0.0.1:1/private-path").prepare()
    inner = ValueError(f"inner {CAP}")
    outer = requests.ConnectionError(f"outer /private-path {CAP}", request=request)
    outer.__cause__ = inner
    redaction = _http.Redaction.for_request(
        "GET", "http://127.0.0.1:1/private-path", headers={"Authorization": f"Bearer {CAP}"}
    )
    assert _http.redact_exception(outer, redaction) is outer
    assert type(outer) is requests.ConnectionError
    assert outer.request is request
    assert outer.__cause__ is inner
    assert str(outer) == "outer [path and query removed] [redacted]"
    assert str(inner) == "inner [redacted]"


def test_short_values_are_removed_only_as_part_of_the_query():
    redaction = _http.Redaction.for_request("GET", "http://h:1/p?q=50")
    out = _http.redact("HTTP 500 for /p?q=50 and q=50", redaction)
    assert out == "HTTP 500 for [path and query removed] and [redacted]"

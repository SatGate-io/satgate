"""No SDK request that carries a credential follows a redirect.

Each credential-bearing request in the SDK is sent to a local HTTPS origin
that answers 301, 302, 303, 307 or 308 with a Location on another host (HTTPS
on "localhost", trusted by the same certificate, so only the redirect rule
keeps the request from it) or on plain HTTP. Every call must raise
RedirectRefusedError naming the status and the origin host only, the origin
must have received the request exactly once, and the redirect target must
receive nothing.
"""

import json
import ssl
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

import pytest
from test_lnd_wallet_tls import _make_cert

from satgate import (
    RedirectRefusedError,
    SatGate,
    SatGateAgentClient,
    SatGateClient,
    SatGateError,
)
from satgate.agent_client import AlbyWallet

REDIRECT_STATUSES = [301, 302, 303, 307, 308]
TARGETS = ["another_host", "plain_http"]

ADMIN = "adm-0123456789abcdef"
CAP = "cap-0123456789abcdef"
API_KEY = "key-0123456789abcdef"
ALBY = "alby-0123456789abcdef"
SECRETS = (ADMIN, CAP, API_KEY, ALBY)


class _Server:
    """A local server. answer(method, path, headers, body) returns
    (status, headers, body) or None for a 200 with an empty JSON object."""

    def __init__(self, cert=None, key=None, answer=None):
        self.requests = []
        self.answer = answer
        server = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def _reply(self):
                length = int(self.headers.get("Content-Length", "0"))
                body = self.rfile.read(length) if length else b""
                headers = {k.lower(): v for k, v in self.headers.items()}
                server.requests.append((self.command, self.path, headers, body))
                out = (
                    server.answer(self.command, self.path, headers, body)
                    if server.answer
                    else None
                )
                status, extra, data = out if out else (200, {}, b"{}")
                self.send_response(status)
                for name, value in extra.items():
                    self.send_header(name, value)
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

            do_GET = do_POST = do_PUT = do_DELETE = _reply

        self.httpd = HTTPServer(("127.0.0.1", 0), Handler)
        self.port = self.httpd.server_address[1]
        self.scheme = "http"
        if cert is not None:
            context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            context.load_cert_chain(str(cert), str(key))
            self.httpd.socket = context.wrap_socket(self.httpd.socket, server_side=True)
            self.scheme = "https"
        self.thread = threading.Thread(
            target=self.httpd.serve_forever, kwargs={"poll_interval": 0.05}, daemon=True
        )
        self.thread.start()

    def url(self, host="127.0.0.1"):
        return f"{self.scheme}://{host}:{self.port}"

    def close(self):
        self.httpd.shutdown()
        self.httpd.server_close()


@pytest.fixture(scope="module")
def gw_cert(tmp_path_factory):
    return _make_cert(tmp_path_factory.mktemp("gw"), "gw")


@pytest.fixture
def net(gw_cert, monkeypatch):
    cert, key = gw_cert
    # requests (and sessions) trust the test certificate through the
    # environment, with verification on.
    monkeypatch.setenv("REQUESTS_CA_BUNDLE", str(cert))
    made = []

    class Net:
        def server(self, tls=True, answer=None):
            s = _Server(cert, key, answer) if tls else _Server(answer=answer)
            made.append(s)
            return s

        def sink(self, target):
            if target == "another_host":
                s = self.server(tls=True)
                return s, s.url("localhost") + "/sink"
            s = self.server(tls=False)
            return s, s.url() + "/sink"

    yield Net()
    for s in made:
        s.close()


def _redirect_all(status, location):
    return lambda method, path, headers, body: (status, {"Location": location}, b"")


def _assert_refused(call, origin, sink, status, expect_origin_requests=1):
    with pytest.raises(RedirectRefusedError) as exc:
        call()
    err = exc.value
    message = str(err)
    assert err.status_code == status
    assert err.host == f"127.0.0.1:{origin.port}"
    assert f"127.0.0.1:{origin.port} answered HTTP {status};" in message
    # Only the status and the origin host are named: not the target, and no
    # credential.
    assert str(sink.port) not in message
    assert "localhost" not in message and "/sink" not in message
    for secret in SECRETS:
        assert secret not in message
    assert sink.requests == []
    assert len(origin.requests) == expect_origin_requests
    return origin.requests


def _carries(request, secret):
    _, _, headers, body = request
    return any(secret in v for v in headers.values()) or secret.encode() in body


# --- Agent client (capability token, L402 token, admin token) --------------


def _agent(origin, **kwargs):
    return SatGateAgentClient(gateway_url=origin.url(), **kwargs)


AGENT_CALLS = {
    # name: (client kwargs, call, credential the request carries)
    "agent_mint": ({"admin_token": ADMIN}, lambda c: c.get("/api/data"), ADMIN),
    "agent_request_get": ({"token": CAP}, lambda c: c.get("/api/data"), CAP),
    "agent_request_post": (
        {"token": CAP},
        lambda c: c.post("/api/data", json={"a": 1}),
        CAP,
    ),
    "agent_request_put": (
        {"token": CAP},
        lambda c: c.put("/api/data", json={"a": 1}),
        CAP,
    ),
    "agent_request_delete": ({"token": CAP}, lambda c: c.delete("/api/data"), CAP),
    "agent_ping": ({"token": CAP}, lambda c: c.ping(), CAP),
    "agent_validate_token": ({"token": CAP}, lambda c: c.validate_token(), CAP),
    "agent_delegate": (
        {"token": CAP},
        lambda c: c.delegate(["scope = api:read"]),
        CAP,
    ),
    "agent_call": (
        {},
        lambda c: c.call("POST", "/route", {"a": 1}, token=CAP),
        CAP,
    ),
    "agent_mint_capability": (
        {"admin_token": ADMIN},
        lambda c: c.mint_capability("api:read"),
        ADMIN,
    ),
    "agent_stage_policy": (
        {"admin_token": ADMIN},
        lambda c: c.stage_policy({"p": 1}),
        ADMIN,
    ),
    "agent_simulate_policy": (
        {"admin_token": ADMIN},
        lambda c: c.simulate_policy("pol"),
        ADMIN,
    ),
    "agent_promote_policy": (
        {"admin_token": ADMIN},
        lambda c: c.promote_policy("pol"),
        ADMIN,
    ),
    "agent_fetch_pack": ({"admin_token": ADMIN}, lambda c: c.fetch_pack("pk"), ADMIN),
}


@pytest.mark.parametrize("target", TARGETS)
@pytest.mark.parametrize("status", REDIRECT_STATUSES)
@pytest.mark.parametrize("name", sorted(AGENT_CALLS))
def test_agent_client_credentialed_request_never_follows_redirect(
    net, name, status, target
):
    kwargs, call, secret = AGENT_CALLS[name]
    sink, location = net.sink(target)
    origin = net.server(answer=_redirect_all(status, location))
    client = _agent(origin, **kwargs)
    seen = _assert_refused(lambda: call(client), origin, sink, status)
    assert _carries(seen[0], secret)


@pytest.mark.parametrize("target", TARGETS)
@pytest.mark.parametrize("status", REDIRECT_STATUSES)
def test_agent_client_l402_retry_never_follows_redirect(net, status, target):
    """The paid retry carries the L402 token; it must not follow a redirect."""
    sink, location = net.sink(target)
    challenge = 'L402 macaroon="mac-0123456789abcdef", invoice="lnbc1"'

    def answer(method, path, headers, body):
        if "authorization" in headers and headers["authorization"].startswith("L402 "):
            return status, {"Location": location}, b""
        return 402, {"WWW-Authenticate": challenge}, b""

    origin = net.server(answer=answer)

    class Wallet:
        def pay_invoice(self, invoice):
            return "pre-0123456789abcdef"

    client = _agent(origin, token=CAP, wallet=Wallet())
    seen = _assert_refused(
        lambda: client.get("/paid"), origin, sink, status, expect_origin_requests=2
    )
    assert (
        seen[1][2]["authorization"] == "L402 mac-0123456789abcdef:pre-0123456789abcdef"
    )


@pytest.mark.parametrize("target", TARGETS)
@pytest.mark.parametrize("status", REDIRECT_STATUSES)
def test_agent_client_refresh_retry_never_follows_redirect(net, status, target):
    """After a 401 the client mints a fresh token (admin token) and retries
    with it; the retry must not follow a redirect."""
    sink, location = net.sink(target)
    calls = {"data": 0}

    def answer(method, path, headers, body):
        if path == "/api/capability/mint":
            return (
                200,
                {"Content-Type": "application/json"},
                json.dumps({"token": CAP + "-new"}).encode(),
            )
        calls["data"] += 1
        if calls["data"] == 1:
            return 401, {}, b""
        return status, {"Location": location}, b""

    origin = net.server(answer=answer)
    client = _agent(origin, token=CAP, admin_token=ADMIN)
    seen = _assert_refused(
        lambda: client.get("/api/data"), origin, sink, status, expect_origin_requests=3
    )
    assert seen[1][1] == "/api/capability/mint"
    assert seen[2][2]["authorization"] == f"Bearer {CAP}-new"


def test_agent_client_ignores_allow_redirects_from_the_caller(net):
    sink, location = net.sink("another_host")
    origin = net.server(answer=_redirect_all(307, location))
    client = _agent(origin, token=CAP)
    _assert_refused(
        lambda: client.get("/api/data", allow_redirects=True), origin, sink, 307
    )


# --- Admin client (X-Admin-Token on every request, capability tokens) ------

ADMIN_CLIENT_CALLS = {
    "admin_tokens_mint": (lambda c: c.tokens.mint("api:read"), ADMIN),
    "admin_tokens_validate": (lambda c: c.tokens.validate(CAP), CAP),
    "admin_tokens_delegate": (
        lambda c: c.tokens.delegate(CAP, ["scope = api:read"]),
        CAP,
    ),
    "admin_governance_ban": (lambda c: c.governance.ban("sig", "r"), ADMIN),
    "admin_governance_get_graph": (lambda c: c.governance.get_graph(), ADMIN),
    "admin_governance_reset": (lambda c: c.governance.reset(), ADMIN),
    "admin_ping": (lambda c: c.ping(CAP), CAP),
}


@pytest.mark.parametrize("target", TARGETS)
@pytest.mark.parametrize("status", REDIRECT_STATUSES)
@pytest.mark.parametrize("name", sorted(ADMIN_CLIENT_CALLS))
def test_admin_client_credentialed_request_never_follows_redirect(
    net, name, status, target
):
    call, secret = ADMIN_CLIENT_CALLS[name]
    sink, location = net.sink(target)
    origin = net.server(answer=_redirect_all(status, location))
    client = SatGateClient(origin.url(), ADMIN)
    seen = _assert_refused(lambda: call(client), origin, sink, status)
    assert _carries(seen[0], secret)
    # The session sends the admin token on every request.
    assert seen[0][2]["x-admin-token"] == ADMIN


@pytest.mark.parametrize("target", TARGETS)
@pytest.mark.parametrize("status", REDIRECT_STATUSES)
def test_admin_client_health_never_follows_redirect(net, status, target):
    """health() reports False rather than raising, and the session's admin
    token never reaches the redirect target."""
    sink, location = net.sink(target)
    origin = net.server(answer=_redirect_all(status, location))
    client = SatGateClient(origin.url(), ADMIN)
    assert client.health() is False
    assert sink.requests == []
    assert len(origin.requests) == 1


# --- Private-beta client (API key) -----------------------------------------

BETA_CALLS = {
    "beta_issue": lambda c: c.issue(task="t"),
    "beta_pay": lambda c: c.pay(capability="c", max_budget_usd=0.1),
    "beta_verify": lambda c: c.verify({"receipt_id": "r"}),
}


@pytest.mark.parametrize("target", TARGETS)
@pytest.mark.parametrize("status", REDIRECT_STATUSES)
@pytest.mark.parametrize("name", sorted(BETA_CALLS))
def test_private_beta_request_never_follows_redirect(net, name, status, target):
    sink, location = net.sink(target)
    origin = net.server(answer=_redirect_all(status, location))
    client = SatGate(api_key=API_KEY, base_url=origin.url())
    seen = _assert_refused(lambda: BETA_CALLS[name](client), origin, sink, status)
    assert seen[0][2]["authorization"] == f"Bearer {API_KEY}"


# --- Alby wallet (access token) --------------------------------------------

ALBY_CALLS = {
    "alby_pay_invoice": lambda w: w.pay_invoice("lnbc1"),
    "alby_get_balance": lambda w: w.get_balance(),
}


@pytest.mark.parametrize("target", TARGETS)
@pytest.mark.parametrize("status", REDIRECT_STATUSES)
@pytest.mark.parametrize("name", sorted(ALBY_CALLS))
def test_alby_wallet_request_never_follows_redirect(net, name, status, target):
    sink, location = net.sink(target)
    origin = net.server(answer=_redirect_all(status, location))
    wallet = AlbyWallet(access_token=ALBY)
    wallet._api_base = origin.url()
    seen = _assert_refused(lambda: ALBY_CALLS[name](wallet), origin, sink, status)
    assert seen[0][2]["authorization"] == f"Bearer {ALBY}"


# --- Reviewer cases --------------------------------------------------------


# Reviewer case: the admin token and capability tokens (in a header or in the
# JSON body) must not reach another host or plain HTTP after a 302 or a 307,
# from either the agent client or the admin client.
@pytest.mark.parametrize("status", [302, 307])
def test_admin_and_capability_tokens_never_reach_redirect_target(net, status):
    for target in TARGETS:
        sink, location = net.sink(target)
        origin = net.server(answer=_redirect_all(status, location))
        calls = [
            lambda o: _agent(o, admin_token=ADMIN).mint_capability("api:read"),
            lambda o: _agent(o, token=CAP).get("/api/data"),
            lambda o: _agent(o, token=CAP).validate_token(),
            lambda o: _agent(o, token=CAP).delegate(),
            lambda o: SatGateClient(o.url(), ADMIN).tokens.mint(),
            lambda o: SatGateClient(o.url(), ADMIN).tokens.validate(CAP),
        ]
        for call in calls:
            with pytest.raises(SatGateError):
                call(origin)
        assert not any(_carries(r, s) for r in sink.requests for s in SECRETS)
        assert sink.requests == []


# Reviewer case: the refusal is the same for every 3xx answer, not only the
# five common redirect codes.
@pytest.mark.parametrize("status", [300, 304, 305, 306, 399])
def test_any_3xx_is_refused_with_status_and_host_only(net, status):
    origin = net.server(
        answer=lambda m, p, h, b: (status, {"Location": "http://127.0.0.1:9/x"}, b"")
    )
    client = _agent(origin, token=CAP)
    with pytest.raises(RedirectRefusedError) as exc:
        client.get("/api/data")
    assert exc.value.status_code == status
    assert "127.0.0.1:9" not in str(exc.value)

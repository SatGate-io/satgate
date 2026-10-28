"""SatGateAgentClient.fetch_jwks does not use the client's session.

The agent client's session keeps the cookies a gateway sets and any header,
auth or cookie a caller adds to it. fetch_jwks sends its request on a new
session instead, with no cookie, no session header, no authentication (also
none from ~/.netrc) and no redirect following. Only the session's ``verify``
setting is carried over.
"""

import json
import os

import pytest
import requests
from test_credential_redirects import _Server
from test_lnd_wallet_tls import _make_cert

from satgate import RedirectRefusedError, SatGateAgentClient, SatGateError

JWKS_PATH = "/.well-known/satgate-agent-jwks.json"
JWKS = {"keys": [{"kty": "OKP", "crv": "Ed25519", "x": "AAAA", "kid": "k1"}]}
COOKIE = "ck-0123456789abcdef"
SESSION_AUTH = "sa-0123456789abcdef"
ADMIN = "adm-0123456789abcdef"
CAP = "cap-0123456789abcdef"
NETRC_PASSWORD = "nr-0123456789abcdef"
SECRETS = (COOKIE, SESSION_AUTH, ADMIN, CAP, NETRC_PASSWORD)


@pytest.fixture(scope="module")
def cert_pair(tmp_path_factory):
    return _make_cert(tmp_path_factory.mktemp("jwks"), "jwks")


@pytest.fixture
def servers(cert_pair, monkeypatch):
    # No CA bundle in the environment: the client trusts the test certificate
    # only through session.verify, so these tests also show that fetch_jwks
    # keeps the session's verify setting.
    monkeypatch.delenv("REQUESTS_CA_BUNDLE", raising=False)
    monkeypatch.delenv("CURL_CA_BUNDLE", raising=False)
    cert, key = cert_pair
    made = []

    def make(answer=None, tls=True):
        s = _Server(cert, key, answer) if tls else _Server(answer=answer)
        made.append(s)
        return s

    yield make
    for s in made:
        s.close()


def _jwks_answer(redirect_status=None, location=None):
    def answer(method, path, headers, body):
        if path == "/seed":
            return (
                200,
                {"Set-Cookie": f"session={COOKIE}; Path=/; Secure; HttpOnly"},
                b"{}",
            )
        if path == JWKS_PATH and redirect_status is not None:
            return redirect_status, {"Location": location}, b""
        if path == JWKS_PATH or path == "/sink":
            return (
                200,
                {"Content-Type": "application/json"},
                json.dumps(JWKS).encode(),
            )
        return None

    return answer


def _client(origin, cert, **kwargs):
    client = SatGateAgentClient(gateway_url=origin.url(), **kwargs)
    client.session.verify = str(cert)
    return client


def _jwks_requests(server):
    return [r for r in server.requests if r[1] == JWKS_PATH]


def _carries_any_secret(request):
    _, _, headers, body = request
    return any(s in v for v in headers.values() for s in SECRETS) or any(
        s.encode() in body for s in SECRETS
    )


def _seed_cookie(client):
    """The gateway sets a cookie; the session keeps it."""
    client.get("/seed")
    assert client.session.cookies.get("session") == COOKIE


def test_fetch_jwks_sends_no_cookie_the_session_holds(servers, cert_pair):
    cert, _ = cert_pair
    origin = servers(_jwks_answer())
    client = _client(origin, cert, token=CAP)
    _seed_cookie(client)
    assert client.fetch_jwks() == JWKS
    (request,) = _jwks_requests(origin)
    assert "cookie" not in request[2]
    assert not _carries_any_secret(request)
    # The session still holds its cookie; fetch_jwks did not change it.
    assert client.session.cookies.get("session") == COOKIE


def test_fetch_jwks_sends_no_session_header_auth_or_cookie_set_by_the_caller(
    servers, cert_pair
):
    cert, _ = cert_pair
    origin = servers(_jwks_answer())
    client = _client(origin, cert, token=CAP, admin_token=ADMIN)
    client.session.headers["Authorization"] = f"Bearer {CAP}"
    client.session.headers["X-Admin-Token"] = ADMIN
    client.session.auth = ("user", SESSION_AUTH)
    client.session.cookies.set("session", COOKIE)
    assert client.fetch_jwks() == JWKS
    (request,) = _jwks_requests(origin)
    headers = request[2]
    for name in ("cookie", "authorization", "x-admin-token", "proxy-authorization"):
        assert name not in headers
    assert not _carries_any_secret(request)


def test_fetch_jwks_sends_no_netrc_credentials(
    servers, cert_pair, tmp_path, monkeypatch
):
    cert, _ = cert_pair
    netrc = tmp_path / "netrc"
    netrc.write_text(f"machine 127.0.0.1 login user password {NETRC_PASSWORD}\n")
    os.chmod(netrc, 0o600)
    monkeypatch.setenv("NETRC", str(netrc))
    origin = servers(_jwks_answer())
    # Control: a plain requests call to this host would add the netrc entry.
    plain = requests.get(origin.url() + "/plain", verify=str(cert), timeout=10)
    assert plain.status_code == 200
    assert "authorization" in origin.requests[-1][2]
    client = _client(origin, cert)
    assert client.fetch_jwks() == JWKS
    (request,) = _jwks_requests(origin)
    assert "authorization" not in request[2]
    assert not _carries_any_secret(request)


@pytest.mark.parametrize("status", [300, 301, 302, 303, 304, 305, 306, 307, 308, 399])
@pytest.mark.parametrize("target", ["same_host", "another_host", "plain_http"])
def test_fetch_jwks_never_follows_a_redirect(servers, cert_pair, status, target):
    cert, _ = cert_pair
    if target == "plain_http":
        sink = servers(_jwks_answer(), tls=False)
        location = sink.url() + "/sink"
    elif target == "another_host":
        sink = servers(_jwks_answer())
        location = sink.url("localhost") + "/sink"
    else:
        sink = None
        location = "/sink"
    origin = servers(_jwks_answer(status, location))
    client = _client(origin, cert, token=CAP)
    _seed_cookie(client)
    with pytest.raises(RedirectRefusedError) as exc:
        client.fetch_jwks()
    err = exc.value
    assert err.status_code == status
    assert err.host == f"127.0.0.1:{origin.port}"
    assert str(err).startswith(
        f"SatGate gateway 127.0.0.1:{origin.port} answered HTTP {status};"
    )
    assert "/sink" not in str(err) and "localhost" not in str(err)
    assert len(_jwks_requests(origin)) == 1
    assert not _carries_any_secret(_jwks_requests(origin)[0])
    assert [r for r in origin.requests if r[1] == "/sink"] == []
    if sink is not None:
        assert sink.requests == []


def test_fetch_jwks_keeps_the_session_verify_setting(servers, cert_pair):
    cert, _ = cert_pair
    origin = servers(_jwks_answer())
    trusted = _client(origin, cert)
    assert trusted.fetch_jwks() == JWKS
    # With the session's default verify=True the test certificate is not
    # trusted, so the request fails verification instead of being sent
    # unverified.
    untrusted = SatGateAgentClient(gateway_url=origin.url())
    with pytest.raises(requests.exceptions.SSLError):
        untrusted.fetch_jwks()


def test_fetch_jwks_error_status_still_raises(servers, cert_pair):
    cert, _ = cert_pair
    origin = servers(lambda m, p, h, b: (503, {}, b"down"))
    client = _client(origin, cert)
    with pytest.raises(SatGateError) as exc:
        client.fetch_jwks()
    assert not isinstance(exc.value, RedirectRefusedError)
    assert "jwks failed: 503" in str(exc.value)


# --- Reviewer cases --------------------------------------------------------


# Reviewer case: the gateway sets a session cookie through an ordinary agent
# request; a JWKS answer of 302 to another path on the same host must not be
# followed, so that path never receives the cookie.
def test_reviewer_jwks_redirect_with_session_cookie_is_not_followed(servers, cert_pair):
    cert, _ = cert_pair
    seen_cookie_at_sink = []

    def answer(method, path, headers, body):
        if path == "/sink":
            seen_cookie_at_sink.append("cookie" in headers)
            return 200, {}, b'{"keys":[]}'
        if path == "/seed":
            cookie = f"session={COOKIE}; Path=/; Secure; HttpOnly"
            return 200, {"Set-Cookie": cookie}, b"{}"
        return 302, {"Location": "/sink"}, b""

    origin = servers(answer)
    client = _client(origin, cert, token=CAP)
    client.get("/seed")
    with pytest.raises(SatGateError):
        client.fetch_jwks()
    assert seen_cookie_at_sink == []
    assert not any(seen_cookie_at_sink)


# Reviewer case: the cookie is set in answer to call() with a capability
# token; the JWKS request carries neither the cookie nor the token, the
# redirect is not followed, and the refusal is a RedirectRefusedError.
def test_reviewer_jwks_request_carries_no_session_credential_and_is_refused(
    servers, cert_pair
):
    cert, _ = cert_pair

    def answer(method, path, headers, body):
        if path.startswith("/.well-known/"):
            return 302, {"Location": "/target"}, b""
        if path == "/seed":
            cookie = f"session={COOKIE}; Secure; HttpOnly; Path=/"
            return 200, {"Set-Cookie": cookie}, b"{}"
        return 200, {"Content-Type": "application/json"}, b'{"keys":[]}'

    origin = servers(answer)
    client = _client(origin, cert, token=CAP)
    client.call("GET", "/seed", token=CAP)
    assert client.session.cookies
    with pytest.raises(RedirectRefusedError):
        client.fetch_jwks()
    jwks = [r for r in origin.requests if r[1].startswith("/.well-known/")]
    target = [r for r in origin.requests if r[1] == "/target"]
    assert len(jwks) == 1
    assert COOKIE not in jwks[0][2].get("cookie", "")
    assert CAP not in jwks[0][2].get("authorization", "")
    assert target == []

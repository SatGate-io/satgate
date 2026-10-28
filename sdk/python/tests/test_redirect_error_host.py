"""Redirect errors name only the host and port of the request URL.

RedirectRefusedError.host and its message are derived from the URL the
request was sent to, with user information, path and query removed. This
holds for LNDWallet, whose host string is configured by the caller and may
contain any of these, for the gateway clients, and for the helper itself.
"""

import logging

import pytest
import requests
from test_credential_redirects import _Server
from test_lnd_wallet_tls import _make_cert

from satgate import (
    RedirectRefusedError,
    SatGateAgentClient,
    SatGateClient,
    SatGateError,
)
from satgate._http import request_host, send_without_redirects
from satgate.agent_client import LNDWallet

USER = "synthetic-user"
PASSWORD = "synthetic-password"
CFG_PATH = "/cfgpath"
CFG_QUERY = "cfgquery=hidden-value"
MACAROON_HEX = "0201036c6e64"
FORBIDDEN = (USER, PASSWORD, "cfgpath", "cfgquery", "hidden-value", "@", "?")


@pytest.fixture(scope="module")
def cert_pair(tmp_path_factory):
    return _make_cert(tmp_path_factory.mktemp("rhost"), "rhost")


@pytest.fixture
def redirecting(cert_pair, monkeypatch):
    cert, key = cert_pair
    monkeypatch.setenv("REQUESTS_CA_BUNDLE", str(cert))
    made = []

    def make(status=302):
        s = _Server(cert, key, lambda m, p, h, b: (status, {"Location": "/sink"}, b""))
        made.append(s)
        return s

    yield make
    for s in made:
        s.close()


def _assert_host_only(err, port, status=302, label="LND host"):
    assert isinstance(err, RedirectRefusedError)
    assert err.status_code == status
    assert err.host == f"127.0.0.1:{port}"
    message = str(err)
    assert message.startswith(f"{label} 127.0.0.1:{port} answered HTTP {status};")
    for part in FORBIDDEN:
        assert part not in message
        assert part not in err.host


LND_HOSTS = {
    "userinfo": lambda port: f"{USER}:{PASSWORD}@127.0.0.1:{port}",
    "user_only": lambda port: f"{USER}@127.0.0.1:{port}",
    "path_and_query": lambda port: f"127.0.0.1:{port}{CFG_PATH}?{CFG_QUERY}",
    "userinfo_path_and_query": (
        lambda port: f"{USER}:{PASSWORD}@127.0.0.1:{port}{CFG_PATH}?{CFG_QUERY}"
    ),
}


@pytest.mark.parametrize("operation", ["get_balance", "pay_invoice"])
@pytest.mark.parametrize("shape", sorted(LND_HOSTS))
def test_lnd_redirect_error_omits_userinfo_path_and_query_of_configured_host(
    redirecting, cert_pair, shape, operation
):
    cert, _ = cert_pair
    origin = redirecting()
    wallet = LNDWallet(
        host=LND_HOSTS[shape](origin.port),
        macaroon_hex=MACAROON_HEX,
        tls_cert_path=str(cert),
    )
    with pytest.raises(RedirectRefusedError) as exc:
        if operation == "pay_invoice":
            wallet.pay_invoice("lnbc1")
        else:
            wallet.get_balance()
    _assert_host_only(exc.value, origin.port)
    assert len(origin.requests) == 1


def test_lnd_tls_error_and_insecure_warning_omit_userinfo_path_and_query(
    redirecting, tmp_path, caplog
):
    origin = redirecting()
    host = LND_HOSTS["userinfo_path_and_query"](origin.port)
    other_cert, _ = _make_cert(tmp_path, "other")
    wallet = LNDWallet(host=host, macaroon_hex=MACAROON_HEX, tls_cert_path=str(other_cert))
    with pytest.raises(SatGateError) as exc:
        wallet.get_balance()
    assert f"LND host 127.0.0.1:{origin.port}:" in str(exc.value)
    for part in FORBIDDEN:
        assert part not in str(exc.value)
    with caplog.at_level(logging.WARNING, logger="satgate.agent_client"):
        LNDWallet(host=host, macaroon_hex=MACAROON_HEX, insecure=True)
    text = " ".join(r.getMessage() for r in caplog.records)
    assert f"127.0.0.1:{origin.port}" in text
    for part in FORBIDDEN:
        assert part not in text


@pytest.mark.parametrize("use_session", [False, True])
@pytest.mark.parametrize("status", [301, 302, 303, 307, 308])
def test_helper_redirect_error_host_comes_from_request_url_without_userinfo_path_or_query(
    redirecting, use_session, status
):
    origin = redirecting(status)
    url = f"https://{USER}:{PASSWORD}@127.0.0.1:{origin.port}{CFG_PATH}?{CFG_QUERY}"
    session = requests.Session() if use_session else None
    with pytest.raises(RedirectRefusedError) as exc:
        send_without_redirects("GET", url, session=session, label="test host", timeout=10)
    _assert_host_only(exc.value, origin.port, status=status, label="test host")


def test_helper_has_no_host_override():
    """The host in the error cannot be supplied separately from the URL."""
    with pytest.raises(TypeError):
        send_without_redirects(
            "GET", "https://127.0.0.1:9/", host="configured", allow_redirects=False
        )


def test_request_host_keeps_only_host_and_written_port():
    assert request_host(f"https://{USER}:{PASSWORD}@LND.Example:443{CFG_PATH}?{CFG_QUERY}#f") == (
        "lnd.example:443"
    )
    assert request_host("https://lnd.example/x") == "lnd.example"
    assert request_host("https://[::1]:8080/x?y=1") == "[::1]:8080"
    assert request_host("https://lnd.example:notaport/") == "an unreadable host"


@pytest.mark.parametrize("client", ["agent", "admin"])
def test_gateway_redirect_error_omits_userinfo_path_and_query_of_gateway_url(
    redirecting, client
):
    origin = redirecting()
    base = f"https://{USER}:{PASSWORD}@127.0.0.1:{origin.port}{CFG_PATH}"
    with pytest.raises(RedirectRefusedError) as exc:
        if client == "agent":
            SatGateAgentClient(gateway_url=base, token="cap-0123").get(f"/x?{CFG_QUERY}")
        else:
            SatGateClient(base, "adm-0123").tokens.mint()
    _assert_host_only(exc.value, origin.port, label="SatGate gateway")


# --- Reviewer cases --------------------------------------------------------


# Reviewer case: an LND host configured with user information, answered with
# a 302, gives an error with neither the user name nor the password in it.
def test_reviewer_lnd_redirect_error_has_no_url_user_information(redirecting, cert_pair):
    cert, _ = cert_pair
    origin = redirecting()
    wallet = LNDWallet(
        host=f"{USER}:{PASSWORD}@127.0.0.1:{origin.port}",
        macaroon_hex=MACAROON_HEX,
        tls_cert_path=str(cert),
    )
    try:
        wallet.get_balance()
    except SatGateError as err:
        assert USER not in str(err) and PASSWORD not in str(err)
    else:
        raise AssertionError("redirect was not refused")


# Reviewer case: with a long random password in the LND host, the refusal is
# a RedirectRefusedError whose host is exactly 127.0.0.1:<port> and whose
# message does not contain the password.
def test_reviewer_lnd_redirect_error_host_is_exactly_host_and_port(redirecting, cert_pair):
    cert, _ = cert_pair
    origin = redirecting()
    password = "f" * 64
    wallet = LNDWallet(
        host=f"user:{password}@127.0.0.1:{origin.port}",
        macaroon_hex=MACAROON_HEX,
        tls_cert_path=str(cert),
    )
    with pytest.raises(RedirectRefusedError) as exc:
        wallet.get_balance()
    assert exc.value.host == f"127.0.0.1:{origin.port}"
    assert password not in str(exc.value)

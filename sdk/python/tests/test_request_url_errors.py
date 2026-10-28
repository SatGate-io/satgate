"""No SDK error names the user information, path or query of a request URL.

An error from requests itself (an invalid URL, a refused connection) is
raised by the SDK as the same exception, and the SDK's own errors that
quote it (``Request failed: ...``, ``Failed to mint token: ...``) quote
the cleaned text. The message names the URL's host and port only; urllib3's
"with url: <path>" reads "with url: [path and query removed]".

Every request goes to 127.0.0.1: a port nothing listens on, or a local
server. Each check reads the messages of the error and of the exceptions it
wraps, and the formatted traceback.
"""

import socket
import traceback

import pytest
import requests
from test_credential_redirects import _Server

from satgate import SatGate, SatGateAgentClient, SatGateClient, SatGateError, _http
from satgate._http import send_without_redirects
from satgate.agent_client import AlbyWallet, LNDWallet

USER = "synthetic-user"
PASSWORD = "synthetic-password"
CFG_PATH = "/cfgpath"
CFG_QUERY = "cfgquery=hidden-value"
URL_PARTS = (USER, PASSWORD, "cfgpath", "cfgquery", "hidden-value")
SDK_PATHS = ("/api/", "/v1/", "/.well-known/")
MACAROON_HEX = "0201036c6e64"


def _closed_port() -> int:
    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    port = s.getsockname()[1]
    s.close()
    return port


# Each shape gives (host part, kind): the host part is what follows the scheme.
SHAPES = {
    "refused, user info, path and query": (
        lambda: f"{USER}:{PASSWORD}@127.0.0.1:{_closed_port()}{CFG_PATH}?{CFG_QUERY}",
        "refused",
    ),
    "refused, path and query": (
        lambda: f"127.0.0.1:{_closed_port()}{CFG_PATH}?{CFG_QUERY}",
        "refused",
    ),
    "unparseable port, user info, path and query": (
        lambda: f"{USER}:{PASSWORD}@127.0.0.1:99999{CFG_PATH}?{CFG_QUERY}",
        "invalid",
    ),
}


def _alby(base):
    wallet = AlbyWallet(access_token="alby-0123")
    wallet._api_base = base
    return wallet


# name -> (call taking the base URL and the bare host part,
#          whether the SDK raises its own error quoting the requests error)
CALLS = {
    "SatGateClient tokens.mint": (
        lambda b, h: SatGateClient(b, "adm-0123").tokens.mint(), True),
    "SatGateClient ping": (
        lambda b, h: SatGateClient(b, "adm-0123").ping("cap-0123"), True),
    "SatGate issue": (
        lambda b, h: SatGate(api_key="key-0123", base_url=b).issue(task="t"), True),
    "SatGateAgentClient get (admin token mint)": (
        lambda b, h: SatGateAgentClient(b, admin_token="adm-0123").get("/x"), True),
    "SatGateAgentClient get (capability token)": (
        lambda b, h: SatGateAgentClient(b, token="cap-0123").get("/x"), False),
    "SatGateAgentClient validate_token": (
        lambda b, h: SatGateAgentClient(b, token="cap-0123").validate_token(), False),
    "SatGateAgentClient delegate": (
        lambda b, h: SatGateAgentClient(b, token="cap-0123").delegate(), False),
    "SatGateAgentClient fetch_pack": (
        lambda b, h: SatGateAgentClient(b, admin_token="adm-0123").fetch_pack("p1"),
        False),
    "SatGateAgentClient fetch_jwks": (
        lambda b, h: SatGateAgentClient(b).fetch_jwks(), False),
    "LNDWallet get_balance": (
        lambda b, h: LNDWallet(host=h, macaroon_hex=MACAROON_HEX).get_balance(), False),
    "LNDWallet pay_invoice": (
        lambda b, h: LNDWallet(host=h, macaroon_hex=MACAROON_HEX).pay_invoice("lnbc1"),
        False),
    "AlbyWallet get_balance": (lambda b, h: _alby(b).get_balance(), False),
    "AlbyWallet pay_invoice": (lambda b, h: _alby(b).pay_invoice("lnbc1"), False),
}


def _chain(err):
    seen, out, pending = set(), [], [err]
    while pending:
        e = pending.pop()
        if not isinstance(e, BaseException) or id(e) in seen:
            continue
        seen.add(id(e))
        out.append(e)
        pending.extend([e.__cause__, e.__context__, *e.args])
    return out


def _assert_no_url_parts(err):
    messages = [str(e) for e in _chain(err)]
    formatted = "".join(traceback.format_exception(type(err), err, err.__traceback__))
    for part in URL_PARTS:
        for message in messages:
            assert part not in message
        assert part not in formatted
    for part in SDK_PATHS:
        for message in messages:
            assert part not in message
    assert "@" not in str(err)


def _raised(fn):
    try:
        fn()
    except Exception as e:  # noqa: BLE001 - the test inspects whatever is raised
        return e
    raise AssertionError("the call did not raise")


@pytest.mark.parametrize("shape", sorted(SHAPES))
@pytest.mark.parametrize("name", sorted(CALLS))
def test_request_error_names_no_user_info_path_or_query(name, shape):
    host_part, kind = SHAPES[shape]
    call, sdk_error = CALLS[name]
    hp = host_part()
    err = _raised(lambda: call(f"http://{hp}", hp))
    _assert_no_url_parts(err)
    if sdk_error:
        assert isinstance(err, SatGateError)
    else:
        assert isinstance(err, requests.RequestException)
    text = str(err)
    if kind == "refused":
        assert "Max retries exceeded with url: [path and query removed]" in text
        assert "127.0.0.1" in text
    else:
        assert "Failed to parse: an unreadable host" in text


@pytest.mark.parametrize("name", ["SatGateClient tokens.mint", "SatGate issue",
                                  "SatGateAgentClient get (capability token)"])
def test_url_without_scheme_names_no_user_info_path_or_query(name):
    call, _ = CALLS[name]
    hp = f"{USER}:{PASSWORD}@127.0.0.1{CFG_PATH}?{CFG_QUERY}"
    err = _raised(lambda: call(hp, hp))
    _assert_no_url_parts(err)
    # InvalidSchema ("No connection adapters were found for ...") or
    # MissingSchema ("Invalid URL ...: No scheme supplied"), naming no URL.
    assert "'an unreadable host'" in str(err)


def test_lightning_payment_failure_names_no_part_of_the_lnd_url():
    """SatGateAgentClient quotes the wallet's error (Lightning payment failed)."""
    challenge = b'{"invoice": "lnbc1", "macaroon": "m"}'
    gateway = _Server(
        answer=lambda m, p, h, b: (402, {"Content-Type": "application/json"}, challenge)
    )
    try:
        lnd_host = f"{USER}:{PASSWORD}@127.0.0.1:{_closed_port()}{CFG_PATH}?{CFG_QUERY}"
        wallet = LNDWallet(host=lnd_host, macaroon_hex=MACAROON_HEX)
        agent = SatGateAgentClient(
            f"http://127.0.0.1:{gateway.port}", token="cap-0123", wallet=wallet
        )
        err = _raised(lambda: agent.get("/x"))
    finally:
        gateway.close()
    assert type(err).__name__ == "PaymentFailedError"
    assert str(err).startswith("Lightning payment failed: ")
    _assert_no_url_parts(err)


def test_send_without_redirects_raises_the_same_exception_type_and_object():
    hp = f"{USER}:{PASSWORD}@127.0.0.1:{_closed_port()}{CFG_PATH}?{CFG_QUERY}"
    err = _raised(lambda: send_without_redirects("GET", f"http://{hp}", timeout=5))
    assert isinstance(err, requests.exceptions.ConnectionError)
    assert err.request is not None
    _assert_no_url_parts(err)


def test_hide_url_keeps_other_text_and_the_object():
    url = f"https://{USER}:{PASSWORD}@gw.example:8443{CFG_PATH}?{CFG_QUERY}"
    exc = ValueError(
        f"cannot load {url}; also {CFG_PATH}?{CFG_QUERY} and {USER}:{PASSWORD}@x", 7
    )
    assert _http.hide_url(exc, url) is exc
    assert exc.args == (
        "cannot load gw.example:8443; also [path and query removed] and x",
        7,
    )

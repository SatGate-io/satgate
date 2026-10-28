"""The one transport for SDK requests that carry a credential.

Every request the SDK sends with a credential (an ``Authorization`` header,
``X-Admin-Token``, an LND macaroon header, or a token in the JSON body) goes
through :func:`send_without_redirects`. It never follows a redirect: the
request is sent once, to the URL the caller configured, and any 3xx answer
raises :class:`~satgate.exceptions.RedirectRefusedError` naming only the
status and the host that answered.
"""

from typing import Optional
from urllib.parse import urlsplit

import requests

from .exceptions import RedirectRefusedError


def request_host(url: str) -> str:
    """host[:port] of url, without user info, path or query."""
    try:
        parts = urlsplit(url)
        host = parts.hostname or ""
        if ":" in host:
            host = f"[{host}]"
        port = f":{parts.port}" if parts.port is not None else ""
    except ValueError:
        return "an unreadable host"
    return f"{host}{port}" if host else "an unreadable host"


def send_without_redirects(
    method: str,
    url: str,
    *,
    session: Optional[requests.Session] = None,
    host: Optional[str] = None,
    label: str = "host",
    **kwargs,
) -> requests.Response:
    """Send one request and never follow a redirect.

    ``session`` is used when given (``Session.request``), otherwise
    ``requests.request``. Any ``allow_redirects`` the caller passed is
    dropped; the request is always sent with ``allow_redirects=False``.

    A 3xx answer closes the response and raises RedirectRefusedError naming
    the status and ``host`` (by default the host of ``url``), so the
    credential is never sent to the redirect's target. Errors from requests
    itself (connection, TLS, timeout) are raised unchanged.
    """
    kwargs.pop("allow_redirects", None)
    sender = session.request if session is not None else requests.request
    resp = sender(method, url, allow_redirects=False, **kwargs)
    if 300 <= resp.status_code < 400:
        status = resp.status_code
        resp.close()
        raise RedirectRefusedError(status, host or request_host(url), label=label)
    return resp

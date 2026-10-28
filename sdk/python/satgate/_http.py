"""The transport for every HTTP request the SDK sends.

Every request goes through :func:`send_without_redirects` (requests that
carry a credential: an ``Authorization`` header, ``X-Admin-Token``, an LND
macaroon header, or a token in the JSON body, plus anything else a caller's
session holds) or :func:`send_without_credentials` (requests that need no
credential, such as the gateway's JWKS). Neither follows a redirect: the
request is sent once, to the configured URL, and any 3xx answer raises
:class:`~satgate.exceptions.RedirectRefusedError` naming only the status and
the host and port of the request URL (never its user information, path or
query).
"""

from typing import Optional, Union
from urllib.parse import urlsplit

import requests
from requests.auth import AuthBase

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
    label: str = "host",
    **kwargs,
) -> requests.Response:
    """Send one request and never follow a redirect.

    ``session`` is used when given (``Session.request``), otherwise
    ``requests.request``. Any ``allow_redirects`` the caller passed is
    dropped; the request is always sent with ``allow_redirects=False``.

    A 3xx answer closes the response and raises RedirectRefusedError naming
    the status and the host and port of ``url``, with its user information,
    path and query removed. The request is not sent again, to that URL or to
    the redirect's target. Errors from requests itself (connection, TLS,
    timeout) are raised unchanged.
    """
    kwargs.pop("allow_redirects", None)
    sender = session.request if session is not None else requests.request
    resp = sender(method, url, allow_redirects=False, **kwargs)
    if 300 <= resp.status_code < 400:
        status = resp.status_code
        resp.close()
        raise RedirectRefusedError(status, request_host(url), label=label)
    return resp


class _NoAuth(AuthBase):
    """Adds nothing to the request.

    Passed as ``auth=`` so that requests does not add credentials it finds
    in ``~/.netrc`` (or ``$NETRC``) for the host.
    """

    def __call__(self, r):
        return r


def send_without_credentials(
    method: str,
    url: str,
    *,
    verify: Union[bool, str] = True,
    timeout: Optional[float] = None,
    label: str = "host",
) -> requests.Response:
    """Send one request that carries no credential, and never follow a redirect.

    The request is sent on a new ``requests.Session`` used for this request
    alone and closed afterwards, so it carries no cookie, no header and no
    authentication from any other session. Credentials from ``~/.netrc`` are
    not added, no client certificate is presented, and the caller cannot
    pass headers, cookies or a body. ``verify`` (which certificates to trust)
    is the only setting taken from the caller. Proxy and CA-bundle settings
    from the environment apply as they do for any requests call.

    A 3xx answer raises RedirectRefusedError as in send_without_redirects.
    """
    with requests.Session() as fresh:
        return send_without_redirects(
            method,
            url,
            session=fresh,
            label=label,
            verify=verify,
            timeout=timeout,
            auth=_NoAuth(),
        )

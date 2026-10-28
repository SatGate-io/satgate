"""The transport for every HTTP request the SDK sends.

Every request goes through :func:`send_without_redirects` (requests that
carry a credential: an ``Authorization`` header, ``X-Admin-Token``, an LND
macaroon header, or a token in the JSON body, plus anything else a caller's
session holds) or :func:`send_without_credentials` (requests that need no
credential, such as the gateway's JWKS). Neither follows a redirect: the
request is sent once, to the configured URL, and any 3xx answer raises
:class:`~satgate.exceptions.RedirectRefusedError` naming only the status and
the host and port of the request URL (never its user information, path or
query). An error from requests itself names no part of the request URL but
its host and port (:func:`hide_url`).
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


_PATH_REMOVED = "[path and query removed]"


def _url_replacements(url: str) -> list:
    """(text, replacement) pairs for the ways requests and urllib3 write url
    in an error message, longest first: the URL as written and as requests
    prepares it become its host and port; its path and query (as urllib3
    writes them) and its user information are removed."""
    host = request_host(url)
    written = {url}
    try:
        prepared = requests.models.PreparedRequest()
        prepared.prepare_url(url, None)
    except (requests.RequestException, ValueError):  # keep only the written form
        prepared = None
    if prepared is not None and prepared.url:
        written.add(prepared.url)
    pairs = {form: host for form in written}
    for form in written:
        try:
            parts = urlsplit(form)
        except ValueError:
            continue
        request_uri = parts.path + (f"?{parts.query}" if parts.query else "")
        if request_uri not in ("", "/"):
            pairs.setdefault(request_uri, _PATH_REMOVED)
        userinfo = parts.netloc.rpartition("@")[0]
        if userinfo:
            pairs.setdefault(f"{userinfo}@", "")
    return sorted(pairs.items(), key=lambda pair: len(pair[0]), reverse=True)


def hide_url(exc: BaseException, url: str) -> BaseException:
    """Remove url from the message of exc and of the exceptions it wraps.

    The URL as written (or as requests prepares it) is replaced by its host
    and port, its path and query by a placeholder, and its user information
    is dropped, in the string arguments of exc, of exceptions passed as its
    arguments, and of its ``__cause__`` and ``__context__``. The same
    exception object is returned, so its type and attributes are unchanged.
    """
    pairs = _url_replacements(url)

    def scrub(text: str) -> str:
        for form, replacement in pairs:
            text = text.replace(form, replacement)
        return text

    seen = set()
    pending = [exc]
    while pending and len(seen) < 16:
        current = pending.pop()
        if not isinstance(current, BaseException) or id(current) in seen:
            continue
        seen.add(id(current))
        args = []
        changed = False
        for arg in current.args:
            if isinstance(arg, str):
                clean = scrub(arg)
                changed = changed or clean != arg
                args.append(clean)
            else:
                pending.append(arg)
                args.append(arg)
        if changed:
            try:
                current.args = tuple(args)
            except (AttributeError, TypeError):
                pass
        reason = getattr(current, "reason", None)
        pending.extend([current.__cause__, current.__context__, reason])
    return exc


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
    the redirect's target. An error from requests itself (an invalid URL, a
    connection, TLS or timeout error) is raised as the same exception, with
    the URL removed from its message (hide_url): the message names the
    URL's host and port, never its user information, path or query.
    """
    kwargs.pop("allow_redirects", None)
    sender = session.request if session is not None else requests.request
    try:
        resp = sender(method, url, allow_redirects=False, **kwargs)
    except Exception as exc:
        hide_url(exc, url)
        raise
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

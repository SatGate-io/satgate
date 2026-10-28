"""The transport for every HTTP request the SDK sends, and the redaction of
every error message the SDK builds or raises about one.

Every request goes through :func:`send_without_redirects` (requests that
carry a credential: an ``Authorization`` header, ``X-Admin-Token``, an LND
macaroon header, or a token in the JSON body, plus anything else a caller's
session holds) or :func:`send_without_credentials` (requests that need no
credential, such as the gateway's JWKS). Neither follows a redirect: the
request is sent once, to the configured URL, and any 3xx answer raises
:class:`~satgate.exceptions.RedirectRefusedError` naming only the status and
the host and port of the request URL (never its user information, path or
query).

Messages are redacted by :func:`redact`, the one redaction function of the
SDK. :class:`Redaction` records what one request carried (see
:meth:`Redaction.for_request`); the transport attaches it to the response
and to any exception it raises, so that every SDK site that builds a
message, quotes a response body or quotes another exception passes the
text through ``redact(text, response_or_exception)``.
"""

import base64
import binascii
import json
import re
from collections.abc import Iterable
from typing import Any, Optional, Union
from urllib.parse import (
    parse_qsl,
    quote,
    quote_plus,
    unquote,
    unquote_plus,
    urlencode,
    urlsplit,
)

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


# Replacements written into a redacted message.
PATH_AND_QUERY_REMOVED = "[path and query removed]"
PATH_REMOVED = "[path removed]"
REDACTED = "[redacted]"

# A credential, query name or query value shorter than this is not replaced
# on its own (it would match ordinary text such as a status code); it is
# still removed as part of the URL, the query string or a name=value pair.
MIN_REDACTED_LENGTH = 3

# Request headers whose values are credentials.
_CREDENTIAL_HEADER = re.compile(
    r"auth|token|key|secret|macaroon|cookie|password", re.IGNORECASE
)
# JSON body fields whose string values (at any depth below them) are
# credentials: the capability token, a parent token, a macaroon, a
# preimage, a receipt or capability object, an API key.
_CREDENTIAL_FIELD = re.compile(
    r"token|macaroon|preimage|secret|password|passwd|key|credential|capability|auth|signature|receipt",
    re.IGNORECASE,
)
# Characters that continue a query name; a query name is replaced only where
# it is not part of a longer word.
_NAME_CHARS = r"A-Za-z0-9_.~%-"

_ATTRIBUTE = "_satgate_redaction"


def _forms(value: str) -> set:
    """value as written and as it is commonly escaped in a message."""
    forms = set()
    for form in {value, value.strip()}:
        forms.add(form)
        forms.update(
            {
                quote(form, safe=""),
                quote_plus(form, safe=""),
                json.dumps(form)[1:-1],
                repr(form)[1:-1],
                unquote(form),
                unquote_plus(form),
            }
        )
    return {form for form in forms if form}


def _query_values(value: Any) -> Iterable[str]:
    """The strings a query parameter value is written as."""
    if isinstance(value, bytes):
        yield value.decode("utf-8", "replace")
    elif isinstance(value, (list, tuple)):
        for item in value:
            yield from _query_values(item)
    elif value is not None:
        yield str(value)


def _body_credentials(value: Any) -> Iterable[str]:
    """String values of the JSON body fields whose names match
    _CREDENTIAL_FIELD, at any depth."""
    if isinstance(value, dict):
        for key, item in value.items():
            if isinstance(key, str) and _CREDENTIAL_FIELD.search(key):
                if isinstance(item, bytes):
                    item = item.decode("utf-8", "replace")
                if isinstance(item, str):
                    yield item
                    continue
                strings = isinstance(item, (list, tuple)) and all(
                    isinstance(i, str) for i in item
                )
                if strings:
                    yield from item
                    continue
            yield from _body_credentials(item)
    elif isinstance(value, (list, tuple)):
        for item in value:
            yield from _body_credentials(item)


def _header_credentials(value: Any) -> Iterable[str]:
    """A credential header's value, its token after the scheme, and its parts."""
    if isinstance(value, bytes):
        value = value.decode("latin-1", "replace")
    if not isinstance(value, str):
        return
    yield value
    match = re.match(r"^\s*([A-Za-z0-9_-]+)\s+(.+?)\s*$", value, re.DOTALL)
    if match:
        scheme, rest = match.group(1).lower(), match.group(2)
        yield rest
        if scheme == "basic":
            try:
                decoded = base64.b64decode(rest).decode("utf-8")
                user, _, password = decoded.partition(":")
            except (binascii.Error, UnicodeDecodeError, ValueError):
                user = password = ""
            yield user
            yield password
        if scheme in ("l402", "lsat"):
            yield from rest.split(":")
    for part in value.split(";"):  # cookies: name=value; name=value
        _, sep, cookie = part.partition("=")
        if sep:
            yield cookie


class Redaction:
    """What one request carried that its error messages must not show: the
    URL (with merged query parameters), its user information, path and
    query, and the credentials in its headers, query and JSON body."""

    def __init__(self) -> None:
        # text -> replacement, and the query names replaced only as words.
        self._exact: dict[str, str] = {}
        self._words: dict[str, str] = {}
        self._pattern: Optional[re.Pattern] = None

    def _add(self, text: str, replacement: str, minimum: int = 1) -> None:
        if text and len(text) >= minimum and text not in self._exact:
            self._exact[text] = replacement
            self._pattern = None

    def add_url(self, url: str) -> None:
        """The URL, its user information, path and query."""
        if not isinstance(url, str) or not url:
            return
        host = request_host(url)
        for form in (url, url.split("#", 1)[0]):
            self._add(form, host)
        try:
            parts = urlsplit(url)
        except ValueError:
            return
        userinfo = parts.netloc.rpartition("@")[0]
        if userinfo:
            # The URL as written without its user information.
            for form in (url, url.split("#", 1)[0]):
                self._add(form.replace(f"{userinfo}@", "", 1), host)
            self._add(f"{userinfo}@", "")
            for value in userinfo.split(":", 1):
                self.add_credential(value)
        request_uri = parts.path + (f"?{parts.query}" if parts.query else "")
        if request_uri not in ("", "/"):
            for form in (request_uri, unquote(request_uri), unquote_plus(request_uri)):
                self._add(form, PATH_AND_QUERY_REMOVED)
        path = parts.path
        for form in {path, unquote(path)}:
            # The path and every tail of it from a "/" (the server may name
            # the path without a prefix it was routed under).
            for index, char in enumerate(form):
                if char == "/" and len(form) - index > 1:
                    self._add(form[index:], PATH_REMOVED)
        if parts.query:
            for form in (parts.query, unquote(parts.query), unquote_plus(parts.query)):
                self._add(form, REDACTED)
            for raw in parts.query.split("&"):
                self._add(raw, REDACTED)
                self._add(unquote_plus(raw), REDACTED)
            for name, value in parse_qsl(parts.query, keep_blank_values=True):
                self._add(f"{name}={value}", REDACTED)
                self.add_credential(value)
                if len(name) >= MIN_REDACTED_LENGTH:
                    for form in _forms(name):
                        if len(form) >= MIN_REDACTED_LENGTH:
                            self._words.setdefault(form, REDACTED)
                            self._pattern = None

    def add_credential(self, value: Any) -> None:
        """A credential, as written and as commonly escaped."""
        if isinstance(value, bytes):
            value = value.decode("utf-8", "replace")
        if not isinstance(value, str):
            return
        for form in _forms(value):
            self._add(form, REDACTED, MIN_REDACTED_LENGTH)

    def add_headers(self, headers: Any) -> None:
        if not headers:
            return
        try:
            items = list(headers.items())
        except Exception:  # noqa: BLE001 - not a mapping: try a list of pairs
            try:
                items = list(headers)
            except Exception:  # noqa: BLE001 - nothing to read
                return
        for item in items:
            try:
                name, value = item
            except (TypeError, ValueError):
                continue
            if isinstance(name, bytes):
                name = name.decode("latin-1", "replace")
            if isinstance(name, str) and _CREDENTIAL_HEADER.search(name):
                for credential in _header_credentials(value):
                    self.add_credential(credential)

    def add_body(self, body: Any) -> None:
        if isinstance(body, (bytes, str)):
            try:
                body = json.loads(body)
            except (ValueError, TypeError):
                return
        for credential in _body_credentials(body):
            self.add_credential(credential)

    def merge(self, other: "Redaction") -> None:
        for text, replacement in other._exact.items():
            self._exact.setdefault(text, replacement)
        for text, replacement in other._words.items():
            self._words.setdefault(text, replacement)
        self._pattern = None

    @classmethod
    def for_request(
        cls,
        method: str,
        url: str,
        session: Optional[requests.Session] = None,
        **kwargs: Any,
    ) -> "Redaction":
        """The redaction for a request sent with ``session.request(method,
        url, **kwargs)`` (or ``requests.request``): the URL as written and as
        requests prepares it with the session's and the call's query
        parameters merged; the credential headers of the session and of the
        call, and those requests adds (authentication from the session, the
        URL or ``~/.netrc``, the session's cookies); and the credential
        fields of the JSON body."""
        redaction = cls()
        headers = kwargs.get("headers")
        params = kwargs.get("params")
        body = kwargs.get("json")
        if body is None:
            body = kwargs.get("data")
        redaction.add_url(url)
        if session is not None:
            redaction.add_headers(session.headers)
        redaction.add_headers(headers)
        redaction.add_body(body)
        # The URL with the session's and the call's query parameters merged,
        # written by hand (so that it is known even when requests cannot
        # prepare the request) and as requests prepares it.
        merged_params: list[tuple[Any, Any]] = []
        raw_queries: list[str] = []
        for source in (getattr(session, "params", None), params):
            if isinstance(source, dict):
                merged_params.extend(source.items())
            elif isinstance(source, (list, tuple)):
                merged_params.extend(source)
            elif isinstance(source, bytes) and source:
                raw_queries.append(source.decode("utf-8", "replace"))
            elif isinstance(source, str) and source:
                raw_queries.append(source)
        if merged_params:
            try:
                raw_queries.append(urlencode(merged_params, doseq=True))
            except (TypeError, ValueError):
                pass
            for _, value in merged_params:
                for written in _query_values(value):
                    redaction.add_credential(written)
        separator = "&" if "?" in url else "?"
        for query in raw_queries:
            redaction.add_url(f"{url}{separator}{query}")
        if raw_queries:
            redaction.add_url(f"{url}{separator}{'&'.join(raw_queries)}")
        # What requests itself sends, when the request can be prepared.
        request = requests.Request(
            method=method,
            url=url,
            headers=headers,
            params=params,
            json=kwargs.get("json"),
            data=kwargs.get("data"),
            cookies=kwargs.get("cookies"),
            auth=kwargs.get("auth"),
        )
        prepared = None
        try:
            if session is not None:
                prepared = session.prepare_request(request)
            else:
                with requests.Session() as fresh:
                    prepared = fresh.prepare_request(request)
        except Exception:  # noqa: BLE001 - an unpreparable request keeps the forms above
            prepared = None
        if prepared is not None:
            redaction.add_url(prepared.url or "")
            redaction.add_headers(prepared.headers)
        return redaction

    def apply(self, text: str) -> str:
        if not isinstance(text, str) or not (self._exact or self._words):
            return text
        if self._pattern is None:
            alternatives = [(t, re.escape(t)) for t in self._exact]
            alternatives += [
                (t, f"(?<![{_NAME_CHARS}]){re.escape(t)}(?![{_NAME_CHARS}])")
                for t in self._words
                if t not in self._exact
            ]
            alternatives.sort(key=lambda pair: len(pair[0]), reverse=True)
            self._pattern = re.compile("|".join(regex for _, regex in alternatives))
        table = {**self._words, **self._exact}
        return self._pattern.sub(lambda m: table.get(m.group(0), REDACTED), text)


def _redaction_of(sources: Iterable[Any]) -> Redaction:
    combined = Redaction()
    for source in sources:
        if isinstance(source, Redaction):
            found = source
        else:
            found = getattr(source, _ATTRIBUTE, None)
        if isinstance(found, Redaction):
            combined.merge(found)
    return combined


def redact(text: str, *sources: Any) -> str:
    """The SDK's one redaction function.

    Returns text with what the request behind each source carried replaced:
    each source is a :class:`Redaction`, or a response or exception the
    transport attached one to. In text:

    * the request URL (as written, or as requests prepares it with merged
      query parameters) becomes its host and port;
    * its path and query become ``[path and query removed]``, the path (or a
      tail of it from a ``/``) alone ``[path removed]``, and the query
      string, each ``name=value`` pair, each query value and each query name
      (as a whole word) ``[redacted]``; its user information is dropped;
    * every credential the request carried (the values of its
      ``Authorization``, ``X-Admin-Token``, macaroon, cookie and other
      headers whose names contain auth, token, key, secret, macaroon, cookie
      or password; the token after the scheme; Basic and L402 parts; the
      credential fields of the JSON body; the user name and password of the
      URL) becomes ``[redacted]``, also where it is quoted, URL-encoded or
      JSON-escaped, and also in text quoted from a response body.

    Credentials, query names and query values shorter than
    ``MIN_REDACTED_LENGTH`` characters are replaced only as part of a longer
    form above.
    """
    return _redaction_of(sources).apply(text)


def redact_exception(exc: BaseException, *sources: Any) -> BaseException:
    """Pass the message of exc, and of the exceptions it wraps, through redact.

    The string arguments of exc, of exceptions passed as its arguments, and
    of its ``__cause__``, ``__context__`` and urllib3 ``reason`` are
    replaced. The same exception object is returned, so its type, its
    attributes (``request``, ``response``, a status code) and its chain are
    unchanged; the redaction is attached to it so that an SDK message
    quoting it can be redacted the same way.
    """
    redaction = _redaction_of(sources)
    seen = set()
    pending: list[Any] = [exc]
    while pending and len(seen) < 32:
        current = pending.pop()
        if not isinstance(current, BaseException) or id(current) in seen:
            continue
        seen.add(id(current))
        args = []
        changed = False
        for arg in current.args:
            if isinstance(arg, str):
                clean = redaction.apply(arg)
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
        for name in ("msg", "strerror"):
            value = getattr(current, name, None)
            if isinstance(value, str):
                try:
                    setattr(current, name, redaction.apply(value))
                except (AttributeError, TypeError):
                    pass
        attach(current, redaction)
        reason = getattr(current, "reason", None)
        pending.extend([current.__cause__, current.__context__, reason])
    return exc


def attach(obj: Any, redaction: Redaction) -> None:
    """Record redaction on a response or exception (merged with any it has)."""
    existing = getattr(obj, _ATTRIBUTE, None)
    if isinstance(existing, Redaction):
        if existing is not redaction:
            existing.merge(redaction)
        return
    try:
        setattr(obj, _ATTRIBUTE, redaction)
    except (AttributeError, TypeError):
        pass


def read_json(resp: requests.Response) -> Any:
    """resp.json(); a parse error is raised as the same exception with its
    message redacted for the request behind resp."""
    try:
        return resp.json()
    except Exception as exc:
        redact_exception(exc, resp)
        raise


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
    the redirect's target. An error from requests itself (an invalid URL or
    header, a connection, TLS or timeout error, a body cut short) is raised
    as the same exception with its messages redacted (redact_exception).
    The returned response carries the request's Redaction, for the messages
    the SDK builds from it.
    """
    kwargs.pop("allow_redirects", None)
    redaction = Redaction.for_request(method, url, session=session, **kwargs)
    sender = session.request if session is not None else requests.request
    try:
        resp = sender(method, url, allow_redirects=False, **kwargs)
    except Exception as exc:
        redact_exception(exc, redaction)
        raise
    if 300 <= resp.status_code < 400:
        status = resp.status_code
        resp.close()
        raise RedirectRefusedError(status, request_host(url), label=label)
    attach(resp, redaction)
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

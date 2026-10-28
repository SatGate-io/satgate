# Changelog

All notable changes to the `satgate` Python SDK are listed here.

## Unreleased

### Changed

- `LNDWallet` now verifies the LND REST server's TLS certificate by default.
  Before, a wallet created without a certificate path sent its requests (and
  the macaroon header) without verifying the server. Now:
  - with no certificate path, the server must present a certificate the
    system trusts; an untrusted certificate raises `SatGateError` naming the
    host, and the macaroon is not sent;
  - `tls_cert_path=` verifies against that file (for example LND's
    `tls.cert`); `cert_path=` is still accepted as an alias. A path that is
    empty or not a file raises `SatGateError` when the wallet is created;
  - `insecure=True` (keyword-only) is the only way to skip verification and
    logs a warning naming the host. It cannot be combined with a certificate
    path.
- `LNDWallet` requests (`get_balance`, `pay_invoice`) no longer follow HTTP
  redirects. The macaroon header is sent only to the configured host, once.
  A 3xx answer raises `RedirectRefusedError` naming the status and the host.
- No HTTP request the SDK sends follows a redirect any more. Every request
  (`SatGateClient`, whose session sends `X-Admin-Token` on every request;
  `SatGateAgentClient` minting, gateway requests with the capability token
  including the 401 refresh and the L402 retry, `validate_token` and
  `delegate` with the token in the body, `call`, the admin helpers and
  `fetch_jwks`; `SatGate` `issue`/`pay`/`verify`; `AlbyWallet`;
  `LNDWallet`) is sent once with `allow_redirects=False`, whatever the
  caller passes. A 3xx answer raises the new `satgate.RedirectRefusedError`
  (a subclass of `SatGateError`, with `status_code` and `host`). Before,
  these requests followed redirects.
- Requests sent through a client's `requests.Session` (all of
  `SatGateClient` and `SatGate`, and every `SatGateAgentClient` request
  except `fetch_jwks`) still carry what that session holds: its headers,
  its authentication and the cookies the gateway set. They are sent only to
  the configured URL.
- `SatGateAgentClient.fetch_jwks()` no longer uses the client's session. It
  is sent on a new session used for that request alone, with no cookies, no
  session headers, no authentication (none from `~/.netrc` either) and no
  client certificate. Only the session's `verify` setting is used.

### Error messages

- `RedirectRefusedError.host` and its message name only the host and port
  of the URL the request was sent to: user information, path and query are
  removed, also when they are part of the `host` given to `LNDWallet`. The
  `LNDWallet` TLS error and the `insecure=True` warning name the host the
  same way.
- When `requests` raises while sending (for example for an invalid URL or a
  refused connection), the URL the SDK passed to it is replaced in the
  message by that URL's host and port, and the path and query urllib3 writes
  for it (`with url: ...`) by `[path and query removed]`; the same is done
  in the exceptions it wraps. The exception keeps its type and is the same
  object, and SDK errors that quote it (such as `Request failed: ...`) quote
  the changed text. Query parameters passed separately with `params=` (by
  the caller or on a session) are not removed.
- Apart from the errors described above, error messages are unchanged in
  this release. Status errors, admin-helper errors, JSON parse errors,
  header-validation errors and conversion errors read as before and can
  quote the request path or query, response text, or a header value.
  Redaction of these messages is planned as a separate change.

### Gateway (Go packages in this repository, not part of this package)

- A `budget_limit` or `delegation_budget` caveat is read exactly, without
  conversion to a floating-point number. It must be a whole number of
  credits above zero that fits in a 64-bit integer; a whole value written
  with zeros after the point (`5.00`) reads as 5. A token with one that is
  zero, negative, fractional, malformed or out of range is refused. When a
  token has several, the smallest applies.
- Delegation writes a child's `budget_limit` as the exact integer
  (`budget_limit = N`) instead of `N.00` formatted from a floating-point
  number, so every accepted allocation up to 9223372036854775807 verifies
  as that number. A budget parameter written as a JSON number that is a
  fraction, uses an exponent or is above 9223372036854775807 is refused
  with an `invalid delegate params` message that states the allowed range.
  A zero-credit child still gets no `budget_limit` of its own.
- A positive delegation allocation above the parent's effective cap (its
  smallest `budget_limit`, and its remaining balance when the budget store
  reports one) is refused before anything is minted or carved.

### Compatibility

- An LND REST endpoint reached through a redirect now fails with
  `SatGateError`; point `host` at the final address.
- A gateway, SatGate API or Alby URL that answers with a redirect (for
  example `http://` redirecting to `https://`, or a trailing-slash redirect)
  now raises `RedirectRefusedError` instead of being followed; configure the
  final URL. `SatGateClient.health()` returns `False` for such a gateway.
  `SatGateAgentClient.fetch_jwks()` raises `RedirectRefusedError` for such a
  gateway too.
- `fetch_jwks()` against a gateway that requires a cookie, a session header
  or a client certificate on the JWKS path now fails; publish the JWKS
  without them. A CA set with `agent.session.verify` still applies.
- Code that passed `cert_path=` keeps working unchanged.
- Code that created `LNDWallet` without a certificate path against LND's
  default self-signed certificate now fails with `SatGateError`; pass
  `tls_cert_path=` pointing at LND's `tls.cert`.
- On the gateway, a token whose `budget_limit` or `delegation_budget` caveat
  is zero, fractional or unreadable is now refused; before, such a caveat
  was ignored or truncated. Tokens with no such caveat, or with whole values
  (including `N.00`), verify as before.

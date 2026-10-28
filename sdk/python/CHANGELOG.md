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
  these requests followed redirects and, depending on the status, resent
  `X-Admin-Token`, the API key or a token in the body to the redirect
  target, and the session's cookies to a target on the same host.
- Requests sent through a client's `requests.Session` (all of
  `SatGateClient` and `SatGate`, and every `SatGateAgentClient` request
  except `fetch_jwks`) still carry what that session holds: its headers,
  its authentication and the cookies the gateway set. They are sent only to
  the configured URL.
- `SatGateAgentClient.fetch_jwks()` no longer uses the client's session. It
  is sent on a new session used for that request alone, with no cookies, no
  session headers, no authentication (none from `~/.netrc` either) and no
  client certificate. Only the session's `verify` setting is used.
- `RedirectRefusedError.host` and its message name only the host and port
  of the URL the request was sent to: user information, path and query are
  removed, also when they are part of the `host` given to `LNDWallet`. The
  `LNDWallet` TLS error and the `insecure=True` warning name the host the
  same way.
- An error from `requests` itself (an invalid URL, a refused connection, a
  timeout) is raised as the same exception, with the request URL removed
  from its message: the URL is named by its host and port, and the path and
  query urllib3 writes (`with url: ...`) read `[path and query removed]`.
  The SDK's own errors that quote such an error (`Request failed: ...`,
  `Ping failed: ...`, `Failed to mint token: ...`, `Lightning payment
  failed: ...`) quote the cleaned text, so no SDK error message carries the
  URL's user information, path or query.

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

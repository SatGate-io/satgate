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
- No SDK request that carries a credential follows an HTTP redirect any
  more. Every such request (`SatGateClient`, whose session sends
  `X-Admin-Token` on every request; `SatGateAgentClient` minting, gateway
  requests with the capability token including the 401 refresh and the L402
  retry, `validate_token` and `delegate` with the token in the body, `call`,
  and the admin helpers; `SatGate` `issue`/`pay`/`verify`; `AlbyWallet`;
  `LNDWallet`) is sent once with `allow_redirects=False`, whatever the
  caller passes. A 3xx answer raises the new `satgate.RedirectRefusedError`
  (a subclass of `SatGateError`, with `status_code` and `host`) naming only
  the status and the host that answered. Before, these requests followed
  redirects and, depending on the status, resent `X-Admin-Token`, the API key
  or a token in the body to the redirect target.

### Compatibility

- An LND REST endpoint reached through a redirect now fails with
  `SatGateError`; point `host` at the final address.
- A gateway, SatGate API or Alby URL that answers with a redirect (for
  example `http://` redirecting to `https://`, or a trailing-slash redirect)
  now raises `RedirectRefusedError` instead of being followed; configure the
  final URL. `SatGateClient.health()` returns `False` for such a gateway.
  `SatGateAgentClient.fetch_jwks()` sends no credential and still follows
  redirects.
- Code that passed `cert_path=` keeps working unchanged.
- Code that created `LNDWallet` without a certificate path against LND's
  default self-signed certificate now fails with `SatGateError`; pass
  `tls_cert_path=` pointing at LND's `tls.cert`.

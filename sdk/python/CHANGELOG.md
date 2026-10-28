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
  A 3xx answer raises `SatGateError` naming the status, the host and the
  redirect's scheme, host and port.

### Compatibility

- An LND REST endpoint reached through a redirect now fails with
  `SatGateError`; point `host` at the final address.
- Code that passed `cert_path=` keeps working unchanged.
- Code that created `LNDWallet` without a certificate path against LND's
  default self-signed certificate now fails with `SatGateError`; pass
  `tls_cert_path=` pointing at LND's `tls.cert`.

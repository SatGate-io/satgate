# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 0.4.x   | ✅ Current release  |
| < 0.4   | ❌ No longer supported |

## Reporting a Vulnerability

**Please do not open public GitHub issues for security vulnerabilities.**

Email **security@satgate.io** with:

- Description of the vulnerability
- Steps to reproduce
- Impact assessment
- Any suggested fixes

We will acknowledge receipt within 48 hours and aim to provide a fix or mitigation plan within 7 days for critical issues.

## Security Design

SatGate follows these security principles:

- **Default-deny**: All routes require valid credentials unless explicitly configured as public
- **Timing-safe authentication**: All credential comparisons use constant-time operations
- **No credential logging**: Tokens, keys, and payment data are never written to logs
- **Ephemeral key warnings**: Clear warnings when running without persistent root keys
- **CORS allowlisting**: Origin restrictions configurable per deployment

For the full security model, see [docs/operations/production-checklist.md](docs/operations/production-checklist.md).

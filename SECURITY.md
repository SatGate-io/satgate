# Security Policy

## Reporting a Vulnerability

We take the security of SatGate seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### How to Report

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please send an email to: **security@satgate.io**

Include the following information:
- Type of issue (e.g., token bypass, privilege escalation, injection, etc.)
- Full paths of source file(s) related to the issue
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

### What to Expect

| Timeframe | Action |
|-----------|--------|
| 24 hours | Initial acknowledgment of your report |
| 72 hours | Preliminary assessment and severity determination |
| 7 days | Detailed response with remediation plan |
| 30 days | Target fix timeline for confirmed vulnerabilities |

We will keep you informed of our progress toward resolving the issue.

### Safe Harbor

We consider security research conducted in accordance with this policy to be:
- Authorized concerning any applicable anti-hacking laws
- Authorized concerning any relevant anti-circumvention laws
- Exempt from restrictions in our Terms of Service that would interfere with conducting security research

We will not pursue legal action against researchers who:
- Make a good faith effort to avoid privacy violations
- Avoid destruction of data and interruption of our services
- Provide us reasonable time to resolve the issue before public disclosure

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.x.x   | ✅ Active |
| < 1.0   | ❌ Not supported |

## Security Updates

Security updates are released as soon as possible after a vulnerability is confirmed and fixed. Subscribe to releases on GitHub to be notified of updates.

## Security Best Practices

When deploying SatGate:

1. **Set strong admin credentials**: Use a cryptographically random `PRICING_ADMIN_TOKEN`
2. **Keep dashboard private**: Set `DASHBOARD_PUBLIC=false` in production
3. **Enable Redis**: For persistent ban list and audit logs
4. **Use TLS**: Terminate TLS at edge and protect origin
5. **Monitor audit logs**: Watch for suspicious admin activity
6. **Rotate credentials**: Periodically rotate admin tokens

See [docs/SECURITY-MODEL.md](docs/SECURITY-MODEL.md) for the full security architecture.

